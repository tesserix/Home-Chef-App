package services

// campaign.go — CRUD + lifecycle for admin marketing campaigns (#56). The send
// pipeline (dispatch, scheduling, metrics) lives in campaign_dispatch.go; this
// file owns composition, validation, and the draft→scheduled→cancelled
// transitions an admin drives directly.

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

var (
	ErrCampaignNoChannel   = errors.New("select at least one channel (push or email)")
	ErrCampaignNoName      = errors.New("campaign name is required")
	ErrCampaignPushContent = errors.New("push campaigns need a title and body")
	ErrCampaignEmailContent = errors.New("email campaigns need a subject and body")
	ErrCampaignNotEditable = errors.New("only draft or scheduled campaigns can be edited")
	ErrCampaignNotFound    = errors.New("campaign not found")
)

// CampaignInput is the composed campaign an admin creates or edits.
type CampaignInput struct {
	Name         string          `json:"name"`
	SendPush     bool            `json:"sendPush"`
	SendEmail    bool            `json:"sendEmail"`
	PushTitle    string          `json:"pushTitle"`
	PushBody     string          `json:"pushBody"`
	EmailSubject string          `json:"emailSubject"`
	EmailHTML    string          `json:"emailHtml"`
	Segment      SegmentCriteria `json:"segment"`
}

// validateCampaignInput enforces the compose rules so a half-built campaign can
// never be created/scheduled.
func validateCampaignInput(in CampaignInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return ErrCampaignNoName
	}
	if !in.SendPush && !in.SendEmail {
		return ErrCampaignNoChannel
	}
	if in.SendPush && (strings.TrimSpace(in.PushTitle) == "" || strings.TrimSpace(in.PushBody) == "") {
		return ErrCampaignPushContent
	}
	if in.SendEmail && (strings.TrimSpace(in.EmailSubject) == "" || strings.TrimSpace(in.EmailHTML) == "") {
		return ErrCampaignEmailContent
	}
	return nil
}

func applyInput(c *models.Campaign, in CampaignInput) error {
	seg, err := json.Marshal(in.Segment)
	if err != nil {
		return err
	}
	c.Name = strings.TrimSpace(in.Name)
	c.SendPush = in.SendPush
	c.SendEmail = in.SendEmail
	c.PushTitle = in.PushTitle
	c.PushBody = in.PushBody
	c.EmailSubject = in.EmailSubject
	c.EmailHTML = in.EmailHTML
	c.Segment = string(seg)
	return nil
}

// CreateCampaign validates and persists a new draft campaign.
func CreateCampaign(db *gorm.DB, in CampaignInput, createdBy *uuid.UUID) (*models.Campaign, error) {
	if err := validateCampaignInput(in); err != nil {
		return nil, err
	}
	c := &models.Campaign{Status: models.CampaignStatusDraft, CreatedBy: createdBy}
	if err := applyInput(c, in); err != nil {
		return nil, err
	}
	if err := db.Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

// UpdateCampaign edits a draft or scheduled campaign (a sent/sending/cancelled
// one is immutable).
func UpdateCampaign(db *gorm.DB, id uuid.UUID, in CampaignInput) (*models.Campaign, error) {
	if err := validateCampaignInput(in); err != nil {
		return nil, err
	}
	c, err := GetCampaign(db, id)
	if err != nil {
		return nil, err
	}
	if c.Status != models.CampaignStatusDraft && c.Status != models.CampaignStatusScheduled {
		return nil, ErrCampaignNotEditable
	}
	if err := applyInput(c, in); err != nil {
		return nil, err
	}
	if err := db.Save(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

// GetCampaign loads one campaign by id.
func GetCampaign(db *gorm.DB, id uuid.UUID) (*models.Campaign, error) {
	var c models.Campaign
	if err := db.First(&c, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCampaignNotFound
		}
		return nil, err
	}
	return &c, nil
}

// ListCampaigns returns a page of campaigns, newest first.
func ListCampaigns(db *gorm.DB, limit, offset int) ([]models.Campaign, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var campaigns []models.Campaign
	var total int64
	if err := db.Model(&models.Campaign{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&campaigns).Error; err != nil {
		return nil, 0, err
	}
	return campaigns, total, nil
}

// DeleteCampaign soft-deletes a draft or cancelled campaign. A campaign that has
// been (or is being) sent is kept for its metrics.
func DeleteCampaign(db *gorm.DB, id uuid.UUID) error {
	c, err := GetCampaign(db, id)
	if err != nil {
		return err
	}
	if c.Status != models.CampaignStatusDraft && c.Status != models.CampaignStatusCancelled {
		return ErrCampaignNotEditable
	}
	return db.Delete(c).Error
}

// CancelCampaign marks a draft/scheduled/queued campaign cancelled so the
// dispatch cron skips it.
func CancelCampaign(db *gorm.DB, id uuid.UUID) (*models.Campaign, error) {
	c, err := GetCampaign(db, id)
	if err != nil {
		return nil, err
	}
	if c.Status == models.CampaignStatusSent || c.Status == models.CampaignStatusSending {
		return nil, ErrCampaignNotEditable
	}
	c.Status = models.CampaignStatusCancelled
	if err := db.Save(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

// SegmentOf decodes a campaign's stored segment JSON.
func SegmentOf(c *models.Campaign) SegmentCriteria {
	var s SegmentCriteria
	_ = json.Unmarshal([]byte(c.Segment), &s)
	return s
}

// nowPtr is a tiny helper for setting *time.Time fields.
func nowPtr() *time.Time { t := time.Now(); return &t }
