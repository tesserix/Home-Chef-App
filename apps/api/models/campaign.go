package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// campaign.go — admin marketing campaigns (#56). An admin builds a recipient
// segment, composes a push and/or email message, schedules it, and the dispatch
// pipeline fans it out over the existing FCM + email infrastructure honoring
// marketing consent + opt-out. CampaignDelivery is the per-recipient×channel
// ledger that powers idempotent resumable sends and delivery/open metrics.

// Campaign lifecycle. draft → scheduled (has ScheduledAt) → queued (the cron
// claimed it) → sending → sent. draft/scheduled → cancelled is allowed.
const (
	CampaignStatusDraft     = "draft"
	CampaignStatusScheduled = "scheduled"
	CampaignStatusQueued    = "queued"
	CampaignStatusSending   = "sending"
	CampaignStatusSent      = "sent"
	CampaignStatusCancelled = "cancelled"
)

// Delivery channels + per-delivery status.
const (
	CampaignChannelPush  = "push"
	CampaignChannelEmail = "email"

	CampaignDeliveryPending = "pending"
	CampaignDeliverySending = "sending" // atomically claimed by one worker, in-flight
	CampaignDeliverySent    = "sent"
	CampaignDeliveryFailed  = "failed"
)

// Campaign is one marketing send. Segment is the JSON-encoded recipient criteria
// (services.SegmentCriteria); channel toggles say which of push/email to send.
// The cached counts are filled when the campaign is dispatched so the admin list
// + metrics read fast without re-aggregating the delivery ledger every time.
type Campaign struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name         string     `gorm:"type:varchar(160);not null" json:"name"`
	Status       string     `gorm:"type:varchar(16);not null;default:'draft';index" json:"status"`
	SendPush     bool       `gorm:"not null;default:false" json:"sendPush"`
	SendEmail    bool       `gorm:"not null;default:false" json:"sendEmail"`
	PushTitle    string     `gorm:"type:varchar(120)" json:"pushTitle"`
	PushBody     string     `gorm:"type:text" json:"pushBody"`
	EmailSubject string     `gorm:"type:varchar(200)" json:"emailSubject"`
	EmailHTML    string     `gorm:"type:text" json:"emailHtml"`
	Segment      string     `gorm:"type:text" json:"segment"` // JSON SegmentCriteria
	ScheduledAt  *time.Time `gorm:"index" json:"scheduledAt,omitempty"`
	SentAt       *time.Time `json:"sentAt,omitempty"`
	CreatedBy    *uuid.UUID `gorm:"type:uuid" json:"createdBy,omitempty"`
	// Cached recipient counts, set at dispatch.
	Recipients int       `gorm:"not null;default:0" json:"recipients"`
	CreatedAt  time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (c *Campaign) BeforeCreate(*gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// CampaignDelivery is one recipient×channel send attempt — an append-only ledger
// row created before the send and updated to sent/failed after. The unique index
// on (campaign_id, user_id, channel) makes dispatch idempotent and resumable: a
// retried dispatch skips rows already marked sent. OpenedAt is set by the email
// open pixel (or a push-open callback) for the open-rate metric.
type CampaignDelivery struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CampaignID    uuid.UUID  `gorm:"type:uuid;not null;index:idx_campaign_delivery_cell,unique;index" json:"campaignId"`
	UserID        uuid.UUID  `gorm:"type:uuid;not null;index:idx_campaign_delivery_cell,unique;index" json:"userId"`
	Channel       string     `gorm:"type:varchar(10);not null;index:idx_campaign_delivery_cell,unique" json:"channel"`
	Status        string     `gorm:"type:varchar(10);not null;default:'pending';index" json:"status"`
	FailureReason string     `gorm:"type:text" json:"failureReason,omitempty"`
	SentAt        *time.Time `json:"sentAt,omitempty"`
	OpenedAt      *time.Time `json:"openedAt,omitempty"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (d *CampaignDelivery) BeforeCreate(*gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
