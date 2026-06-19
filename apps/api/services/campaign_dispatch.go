package services

// campaign_dispatch.go — the send pipeline for marketing campaigns (#56).
//
// Flow: an admin schedules a campaign (or sends now). The campaign-dispatch cron
// scans for due scheduled campaigns and, in one transaction, transitions each to
// "queued" and stages a campaigns.dispatch event on the transactional outbox →
// CAMPAIGNS JetStream stream. A durable consumer runs DispatchCampaign, which
// resolves the reachable segment per channel, writes a per-recipient×channel
// CampaignDelivery row, and sends over the existing FCM + email infrastructure —
// honoring marketing consent + opt-out. The delivery ledger's unique key makes
// the whole thing idempotent and resumable: a redelivered event re-sends only to
// recipients not already marked sent. A Redis SETNX claim stops two instances
// dispatching the same campaign at once.

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const campaignDispatchInterval = 1 * time.Minute

// campaignStuckAfter is how long a campaign may sit in queued/sending before the
// scan re-enqueues a dispatch — a safety net for a lost event / DLQ exhaustion.
// Comfortably past the consumer's AckWait so it doesn't race a live attempt.
const campaignStuckAfter = 15 * time.Minute

// publicAPIBase is the externally reachable API origin used to build the email
// open-pixel + unsubscribe links. Overridable for non-prod via PUBLIC_API_URL.
func publicAPIBase() string {
	if v := os.Getenv("PUBLIC_API_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://api.fe3dr.com"
}

// ScheduleCampaign sets a future send time and moves the campaign to scheduled.
// The dispatch cron picks it up once scheduled_at passes.
func ScheduleCampaign(db *gorm.DB, id uuid.UUID, at time.Time) (*models.Campaign, error) {
	c, err := GetCampaign(db, id)
	if err != nil {
		return nil, err
	}
	if c.Status != models.CampaignStatusDraft && c.Status != models.CampaignStatusScheduled {
		return nil, ErrCampaignNotEditable
	}
	at = at.UTC()
	c.ScheduledAt = &at
	c.Status = models.CampaignStatusScheduled
	if err := db.Save(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

// SendCampaignNow queues a draft/scheduled campaign for immediate dispatch by
// transitioning it to queued and staging the dispatch event in the same tx.
func SendCampaignNow(db *gorm.DB, id uuid.UUID) (*models.Campaign, error) {
	c, err := GetCampaign(db, id)
	if err != nil {
		return nil, err
	}
	if c.Status != models.CampaignStatusDraft && c.Status != models.CampaignStatusScheduled {
		return nil, ErrCampaignNotEditable
	}
	if err := queueCampaign(db, id); err != nil {
		return nil, err
	}
	return GetCampaign(db, id)
}

// queueCampaign atomically transitions a campaign to queued (only from
// draft/scheduled) and stages the dispatch event — so the event is only ever
// enqueued once per campaign.
func queueCampaign(db *gorm.DB, id uuid.UUID) error {
	return db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.Campaign{}).
			Where("id = ? AND status IN ?", id, []string{models.CampaignStatusDraft, models.CampaignStatusScheduled}).
			Update("status", models.CampaignStatusQueued)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // already queued/sent/cancelled — no double-enqueue
		}
		return EnqueueEvent(tx, SubjectCampaignDispatch, "campaign_dispatch", uuid.Nil, map[string]any{
			"campaign_id": id.String(),
		})
	})
}

// DispatchCampaign fans a campaign out to its reachable segment over each enabled
// channel. Idempotent + resumable via the delivery ledger; guarded by a Redis
// claim so concurrent instances don't double-send.
func DispatchCampaign(ctx context.Context, db *gorm.DB, campaignID uuid.UUID) error {
	if !claimCampaignDispatch(ctx, campaignID) {
		return nil // another instance holds the claim
	}
	c, err := GetCampaign(db, campaignID)
	if err != nil {
		return err
	}
	if c.Status == models.CampaignStatusSent || c.Status == models.CampaignStatusCancelled {
		return nil // already finished or aborted
	}
	if err := db.Model(c).Update("status", models.CampaignStatusSending).Error; err != nil {
		return err
	}

	seg := SegmentOf(c)

	if c.SendPush {
		ids, err := SegmentReachable(db, seg, models.CampaignChannelPush)
		if err != nil {
			return err
		}
		dispatchPushChannel(db, c, ids)
	}
	if c.SendEmail {
		ids, err := SegmentReachable(db, seg, models.CampaignChannelEmail)
		if err != nil {
			return err
		}
		dispatchEmailChannel(db, c, ids)
	}

	// Recipient count is derived from the ledger (distinct users actually sent),
	// not the in-memory segment — so it's correct even on a crash-resume where
	// the live segment may have shifted (e.g. someone unsubscribed mid-send).
	var recipients int64
	db.Model(&models.CampaignDelivery{}).
		Where("campaign_id = ? AND status = ?", c.ID, models.CampaignDeliverySent).
		Distinct("user_id").Count(&recipients)

	return db.Model(c).Updates(map[string]any{
		"status":     models.CampaignStatusSent,
		"sent_at":    nowPtr(),
		"recipients": int(recipients),
	}).Error
}

// claimDeliveryForSend is the send-once anchor. It find-or-creates the
// (campaign,user,channel) ledger row (unique index), then ATOMICALLY transitions
// it pending → sending. Only the worker whose UPDATE affects a row may send —
// so even with two concurrent dispatchers (or Redis down, where the claim
// no-ops) a recipient is sent to at most once. A row already sent/sending/failed
// is skipped (we never re-send), which makes a redelivered dispatch a safe
// resume. Returns (row, true) only to the winning sender.
func claimDeliveryForSend(db *gorm.DB, campaignID, userID uuid.UUID, channel string) (*models.CampaignDelivery, bool) {
	d := models.CampaignDelivery{
		CampaignID: campaignID, UserID: userID, Channel: channel, Status: models.CampaignDeliveryPending,
	}
	if err := db.Where("campaign_id = ? AND user_id = ? AND channel = ?", campaignID, userID, channel).
		FirstOrCreate(&d).Error; err != nil {
		return nil, false
	}
	res := db.Model(&models.CampaignDelivery{}).
		Where("id = ? AND status = ?", d.ID, models.CampaignDeliveryPending).
		Update("status", models.CampaignDeliverySending)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil, false // lost the claim, or already sent/sending/failed
	}
	return &d, true
}

func markDelivery(db *gorm.DB, d *models.CampaignDelivery, sendErr error) {
	if sendErr != nil {
		db.Model(d).Updates(map[string]any{"status": models.CampaignDeliveryFailed, "failure_reason": sendErr.Error()})
		return
	}
	db.Model(d).Updates(map[string]any{"status": models.CampaignDeliverySent, "sent_at": nowPtr()})
}

func dispatchPushChannel(db *gorm.DB, c *models.Campaign, ids []uuid.UUID) {
	data := map[string]string{"type": "campaign", "campaignId": c.ID.String()}
	for _, uid := range ids {
		d, ok := claimDeliveryForSend(db, c.ID, uid, models.CampaignChannelPush)
		if !ok {
			continue
		}
		markDelivery(db, d, SendPushNotification(uid, c.PushTitle, c.PushBody, data))
	}
}

func dispatchEmailChannel(db *gorm.DB, c *models.Campaign, ids []uuid.UUID) {
	for _, uid := range ids {
		d, ok := claimDeliveryForSend(db, c.ID, uid, models.CampaignChannelEmail)
		if !ok {
			continue
		}
		var u models.User
		if err := db.Select("id", "email").First(&u, "id = ?", uid).Error; err != nil {
			markDelivery(db, d, err)
			continue
		}
		html := BuildCampaignEmailHTML(c.EmailHTML, d.ID)
		markDelivery(db, d, GetEmailService().Send(u.Email, c.EmailSubject, html))
	}
}

// BuildCampaignEmailHTML appends the one-click unsubscribe footer and the open
// tracking pixel (keyed on the delivery id) to the composed email body.
func BuildCampaignEmailHTML(body string, deliveryID uuid.UUID) string {
	base := publicAPIBase()
	unsub := fmt.Sprintf(
		`<p style="font-size:12px;color:#888;margin-top:24px">You're receiving this because you opted in to Fe3dr updates. <a href="%s/api/v1/campaigns/unsubscribe/%s">Unsubscribe</a>.</p>`,
		base, deliveryID)
	pixel := fmt.Sprintf(
		`<img src="%s/api/v1/campaigns/track/open/%s" width="1" height="1" alt="" style="display:none" />`,
		base, deliveryID)
	return body + unsub + pixel
}

// MarkCampaignDeliveryOpened records the first open for a delivery (open-pixel
// hit). Idempotent — only the first open sets the timestamp.
func MarkCampaignDeliveryOpened(db *gorm.DB, deliveryID uuid.UUID) {
	db.Model(&models.CampaignDelivery{}).
		Where("id = ? AND opened_at IS NULL", deliveryID).
		Update("opened_at", time.Now())
}

// UnsubscribeMarketingByDelivery withdraws the recipient's marketing consent
// (one-click email unsubscribe) — the strongest honored opt-out, since
// reachability gates on consent.
func UnsubscribeMarketingByDelivery(db *gorm.DB, deliveryID uuid.UUID) error {
	var d models.CampaignDelivery
	if err := db.First(&d, "id = ?", deliveryID).Error; err != nil {
		return err
	}
	return db.Model(&models.User{}).Where("id = ?", d.UserID).
		Updates(map[string]any{"marketing_consent": false}).Error
}

// TestSendCampaign sends the composed push + email to a single user (the admin
// previewing) — bypassing consent/segment since it's an explicit self-test. The
// email omits the pixel/unsubscribe footer so the preview is clean.
func TestSendCampaign(db *gorm.DB, campaignID, userID uuid.UUID) error {
	c, err := GetCampaign(db, campaignID)
	if err != nil {
		return err
	}
	var u models.User
	if err := db.Select("id", "email", "fcm_token").First(&u, "id = ?", userID).Error; err != nil {
		return err
	}
	if c.SendPush && u.FCMToken != "" {
		_ = SendPushNotification(userID, c.PushTitle, c.PushBody, map[string]string{"type": "campaign_test"})
	}
	if c.SendEmail && u.Email != "" {
		if err := GetEmailService().Send(u.Email, "[TEST] "+c.EmailSubject, c.EmailHTML); err != nil {
			return err
		}
	}
	return nil
}

// CampaignChannelMetrics is the sent/failed/opened breakdown for one channel.
type CampaignChannelMetrics struct {
	Sent   int64 `json:"sent"`
	Failed int64 `json:"failed"`
	Opened int64 `json:"opened"`
}

// CampaignMetricsResult is the delivery/open overview for a campaign.
type CampaignMetricsResult struct {
	Recipients int                    `json:"recipients"`
	Push       CampaignChannelMetrics `json:"push"`
	Email      CampaignChannelMetrics `json:"email"`
}

// GetCampaignMetrics aggregates the delivery ledger for a campaign.
func GetCampaignMetrics(db *gorm.DB, campaignID uuid.UUID) (CampaignMetricsResult, error) {
	var res CampaignMetricsResult
	c, err := GetCampaign(db, campaignID)
	if err != nil {
		return res, err
	}
	res.Recipients = c.Recipients
	res.Push = channelMetrics(db, campaignID, models.CampaignChannelPush)
	res.Email = channelMetrics(db, campaignID, models.CampaignChannelEmail)
	return res, nil
}

func channelMetrics(db *gorm.DB, campaignID uuid.UUID, channel string) CampaignChannelMetrics {
	var m CampaignChannelMetrics
	base := db.Model(&models.CampaignDelivery{}).Where("campaign_id = ? AND channel = ?", campaignID, channel)
	base.Session(&gorm.Session{}).Where("status = ?", models.CampaignDeliverySent).Count(&m.Sent)
	base.Session(&gorm.Session{}).Where("status = ?", models.CampaignDeliveryFailed).Count(&m.Failed)
	base.Session(&gorm.Session{}).Where("opened_at IS NOT NULL").Count(&m.Opened)
	return m
}

// claimCampaignDispatch is a per-campaign Redis SETNX guard so concurrent
// instances don't double-dispatch. No Redis → allow (the delivery-ledger unique
// key + status guard still prevent double-sends).
func claimCampaignDispatch(ctx context.Context, campaignID uuid.UUID) bool {
	r := GetRedisClient()
	if !r.IsConnected() {
		return true
	}
	dedupCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	acquired, err := r.SetNX(dedupCtx, fmt.Sprintf("campaign_dispatch:%s", campaignID), "1", 30*time.Minute)
	if err != nil {
		return true
	}
	return acquired
}

// StartCampaignCron runs the due-campaign scan as an in-process ticker (the
// Temporal-schedule path calls runCampaignDispatch directly).
func StartCampaignCron(ctx context.Context) {
	go func() {
		runCampaignDispatch(ctx)
		ticker := time.NewTicker(campaignDispatchInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCampaignDispatch(ctx)
			}
		}
	}()
}

// runCampaignDispatch queues every scheduled campaign whose time has come.
// Panic-safe; the actual fan-out happens in the durable consumer.
func runCampaignDispatch(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()
	var due []models.Campaign
	// Compare in UTC — scheduled_at is stored UTC (ScheduleCampaign), so a UTC
	// "now" keeps the ordering correct on every driver.
	database.DB.Where("status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?",
		models.CampaignStatusScheduled, time.Now().UTC()).Find(&due)
	for _, c := range due {
		_ = queueCampaign(database.DB, c.ID)
	}

	// Safety net: re-enqueue a dispatch for any campaign stuck in queued/sending
	// past the grace window (a lost/dead-lettered event). Re-dispatch is safe —
	// the atomic per-delivery claim + the "sent" status guard make it idempotent.
	var stuck []models.Campaign
	database.DB.Where("status IN ? AND updated_at < ?",
		[]string{models.CampaignStatusQueued, models.CampaignStatusSending},
		time.Now().Add(-campaignStuckAfter)).Find(&stuck)
	for _, c := range stuck {
		_ = EnqueueEvent(database.DB, SubjectCampaignDispatch, "campaign_dispatch", uuid.Nil, map[string]any{
			"campaign_id": c.ID.String(),
		})
	}
}
