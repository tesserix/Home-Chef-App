package services

// campaign_dispatch_test.go — the send pipeline (#56). Pins: dispatch sends only
// to the reachable (consented) segment and records per-recipient×channel
// deliveries; a re-dispatch is idempotent (no duplicate deliveries, stable
// recipient count); send-now and the due-scan both queue exactly one dispatch
// event; the open pixel and unsubscribe move the right state. Push/email "sends"
// are no-ops in tests (no provider configured) so they resolve to sent, which is
// exactly what lets us assert the ledger + metrics.

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func makeCampaign(t *testing.T, db *gorm.DB, push, email bool) *models.Campaign {
	t.Helper()
	in := CampaignInput{
		Name: "Test", SendPush: push, SendEmail: email,
		PushTitle: "T", PushBody: "B", EmailSubject: "S", EmailHTML: "<p>Hi</p>",
		Segment: SegmentCriteria{}, // all active customers
	}
	c, err := CreateCampaign(db, in, nil)
	require.NoError(t, err)
	return c
}

func TestDispatchCampaign_SendsToReachableOnly(t *testing.T) {
	db := setupCampaignDB(t)
	reachable := seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, consent: true, fcm: "tok", lastOrd: -1})
	seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, consent: false, fcm: "tok", lastOrd: -1}) // no consent

	c := makeCampaign(t, db, true, true)
	require.NoError(t, DispatchCampaign(context.Background(), db, c.ID))

	got, _ := GetCampaign(db, c.ID)
	require.Equal(t, models.CampaignStatusSent, got.Status)
	require.NotNil(t, got.SentAt)
	require.Equal(t, 1, got.Recipients) // only the consented user

	// One push + one email delivery, both for the reachable user, both sent.
	var deliveries []models.CampaignDelivery
	db.Where("campaign_id = ?", c.ID).Find(&deliveries)
	require.Len(t, deliveries, 2)
	for _, d := range deliveries {
		require.Equal(t, reachable, d.UserID)
		require.Equal(t, models.CampaignDeliverySent, d.Status)
	}

	m, _ := GetCampaignMetrics(db, c.ID)
	require.Equal(t, int64(1), m.Push.Sent)
	require.Equal(t, int64(1), m.Email.Sent)
}

func TestDispatchCampaign_Idempotent(t *testing.T) {
	db := setupCampaignDB(t)
	seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, consent: true, fcm: "tok", lastOrd: -1})
	c := makeCampaign(t, db, true, true)

	require.NoError(t, DispatchCampaign(context.Background(), db, c.ID))
	// Force a re-dispatch (simulating a redelivered event): reset status as the
	// consumer would see "sending" on a crash-resume.
	require.NoError(t, db.Model(c).Update("status", models.CampaignStatusSending).Error)
	require.NoError(t, DispatchCampaign(context.Background(), db, c.ID))

	var n int64
	db.Model(&models.CampaignDelivery{}).Where("campaign_id = ?", c.ID).Count(&n)
	require.Equal(t, int64(2), n) // still just push + email, no duplicates
}

func TestDispatchCampaign_SkipsAlreadySentDelivery(t *testing.T) {
	db := setupCampaignDB(t)
	uid := seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, consent: true, email: "x@ex.com", lastOrd: -1})
	c := makeCampaign(t, db, false, true)

	// A delivery already marked sent (e.g. from a prior partial dispatch) must
	// NOT be re-claimed/re-sent — the atomic pending→sending claim is the
	// send-once anchor, independent of Redis.
	pre := models.CampaignDelivery{
		CampaignID: c.ID, UserID: uid, Channel: models.CampaignChannelEmail,
		Status: models.CampaignDeliverySent, SentAt: nowPtr(),
	}
	require.NoError(t, db.Create(&pre).Error)

	require.NoError(t, DispatchCampaign(context.Background(), db, c.ID))

	// Still exactly one delivery, still sent (not re-sent / duplicated).
	var rows []models.CampaignDelivery
	db.Where("campaign_id = ? AND user_id = ?", c.ID, uid).Find(&rows)
	require.Len(t, rows, 1)
	require.Equal(t, models.CampaignDeliverySent, rows[0].Status)
}

func TestSendCampaignNow_QueuesAndEnqueuesOnce(t *testing.T) {
	db := setupCampaignDB(t)
	c := makeCampaign(t, db, true, false)

	_, err := SendCampaignNow(db, c.ID)
	require.NoError(t, err)

	got, _ := GetCampaign(db, c.ID)
	require.Equal(t, models.CampaignStatusQueued, got.Status)

	var n int64
	db.Raw(`SELECT count(*) FROM outbox_events WHERE subject = ?`, SubjectCampaignDispatch).Scan(&n)
	require.Equal(t, int64(1), n)

	// A second send-now is a no-op (already queued) — no second event.
	_, _ = SendCampaignNow(db, c.ID)
	db.Raw(`SELECT count(*) FROM outbox_events WHERE subject = ?`, SubjectCampaignDispatch).Scan(&n)
	require.Equal(t, int64(1), n)
}

func TestRunCampaignDispatch_QueuesDueScheduledOnly(t *testing.T) {
	db := setupCampaignDB(t)
	due := makeCampaign(t, db, true, false)
	_, err := ScheduleCampaign(db, due.ID, time.Now().Add(-time.Minute)) // already due
	require.NoError(t, err)
	future := makeCampaign(t, db, true, false)
	_, err = ScheduleCampaign(db, future.ID, time.Now().Add(time.Hour)) // not yet
	require.NoError(t, err)

	runCampaignDispatch(context.Background())

	dueGot, _ := GetCampaign(db, due.ID)
	require.Equal(t, models.CampaignStatusQueued, dueGot.Status)
	futureGot, _ := GetCampaign(db, future.ID)
	require.Equal(t, models.CampaignStatusScheduled, futureGot.Status)
}

func TestMarkOpenedAndUnsubscribe(t *testing.T) {
	db := setupCampaignDB(t)
	uid := seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, consent: true, email: "x@ex.com", lastOrd: -1})
	c := makeCampaign(t, db, false, true)
	require.NoError(t, DispatchCampaign(context.Background(), db, c.ID))

	var d models.CampaignDelivery
	require.NoError(t, db.Where("campaign_id = ? AND channel = ?", c.ID, models.CampaignChannelEmail).First(&d).Error)

	// Open pixel sets opened_at once.
	MarkCampaignDeliveryOpened(db, d.ID)
	var reload models.CampaignDelivery
	db.First(&reload, "id = ?", d.ID)
	require.NotNil(t, reload.OpenedAt)

	// Unsubscribe withdraws marketing consent.
	require.NoError(t, UnsubscribeMarketingByDelivery(db, d.ID))
	var u models.User
	db.First(&u, "id = ?", uid)
	require.False(t, u.MarketingConsent)
}
