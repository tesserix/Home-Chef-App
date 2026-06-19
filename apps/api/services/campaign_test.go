package services

// campaign_test.go — CRUD + lifecycle rules for marketing campaigns (#56):
// compose validation (channel + content), the segment round-trips through
// storage, and a sent/sending campaign is immutable.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func validInput() CampaignInput {
	return CampaignInput{
		Name:      "Diwali offer",
		SendPush:  true,
		PushTitle: "🪔 Diwali treats",
		PushBody:  "Order from home chefs near you",
		Segment:   SegmentCriteria{Recency: "lapsed", RecencyDays: 30},
	}
}

func TestCreateCampaign_Validation(t *testing.T) {
	db := setupCampaignDB(t)

	_, err := CreateCampaign(db, CampaignInput{Name: ""}, nil)
	require.ErrorIs(t, err, ErrCampaignNoName)

	_, err = CreateCampaign(db, CampaignInput{Name: "x"}, nil)
	require.ErrorIs(t, err, ErrCampaignNoChannel)

	_, err = CreateCampaign(db, CampaignInput{Name: "x", SendPush: true}, nil)
	require.ErrorIs(t, err, ErrCampaignPushContent)

	_, err = CreateCampaign(db, CampaignInput{Name: "x", SendEmail: true, EmailSubject: "Hi"}, nil)
	require.ErrorIs(t, err, ErrCampaignEmailContent)
}

func TestCreateCampaign_PersistsAndRoundTripsSegment(t *testing.T) {
	db := setupCampaignDB(t)
	c, err := CreateCampaign(db, validInput(), nil)
	require.NoError(t, err)
	require.Equal(t, models.CampaignStatusDraft, c.Status)

	got, err := GetCampaign(db, c.ID)
	require.NoError(t, err)
	seg := SegmentOf(got)
	require.Equal(t, "lapsed", seg.Recency)
	require.Equal(t, 30, seg.RecencyDays)
}

func TestUpdateCampaign_BlockedAfterSent(t *testing.T) {
	db := setupCampaignDB(t)
	c, err := CreateCampaign(db, validInput(), nil)
	require.NoError(t, err)

	// Simulate a sent campaign.
	require.NoError(t, db.Model(c).Update("status", models.CampaignStatusSent).Error)

	_, err = UpdateCampaign(db, c.ID, validInput())
	require.ErrorIs(t, err, ErrCampaignNotEditable)
}

func TestCancelCampaign(t *testing.T) {
	db := setupCampaignDB(t)
	c, err := CreateCampaign(db, validInput(), nil)
	require.NoError(t, err)

	cancelled, err := CancelCampaign(db, c.ID)
	require.NoError(t, err)
	require.Equal(t, models.CampaignStatusCancelled, cancelled.Status)
}

func TestGetCampaign_NotFound(t *testing.T) {
	db := setupCampaignDB(t)
	_, err := GetCampaign(db, uuid.New())
	require.ErrorIs(t, err, ErrCampaignNotFound)
}
