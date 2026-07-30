package services

// account_archive.go — the financial record kept when an account is erased.
//
// Two obligations pull in opposite directions here. Indian tax law requires the
// financial trail behind every transaction to survive for years; DPDP requires
// the personal data to actually go. The resolution is to archive the money and
// erase the identity: the archive carries order numbers, totals, tax lines and
// payout references, and NO name, email, phone, address or geo.
//
// The user id is retained as an opaque key. On its own it identifies nobody once
// the users row is gone — there is nothing left to join it back to — but it lets
// an auditor tie the archived orders together as one historical customer.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// archivedOrder is the deliberately narrow projection that survives erasure.
// Adding a field here is a privacy decision, not a convenience one: anything
// that could re-identify the person must stay out.
type archivedOrder struct {
	OrderNumber string    `json:"orderNumber"`
	Status      string    `json:"status"`
	Subtotal    float64   `json:"subtotal"`
	DeliveryFee float64   `json:"deliveryFee"`
	PlatformFee float64   `json:"platformFee"`
	Tax         float64   `json:"tax"`
	TaxName     string    `json:"taxName"`
	Total       float64   `json:"total"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ArchiveAccountFinancials writes the PII-stripped financial record for a user
// to the private bucket before their rows are erased.
//
// A no-op when GCS is unconfigured (local/dev), so the purge sweeper behaves
// identically in environments without object storage.
func ArchiveAccountFinancials(ctx context.Context, user models.User) error {
	if config.AppConfig == nil || config.AppConfig.GCSPrivateBucket == "" {
		return nil
	}

	var orders []models.Order
	if err := database.DB.WithContext(ctx).Unscoped().
		Where("customer_id = ? OR chef_id IN (SELECT id FROM chef_profiles WHERE user_id = ?)",
			user.ID, user.ID).
		Find(&orders).Error; err != nil {
		return fmt.Errorf("archive: load orders: %w", err)
	}

	records := make([]archivedOrder, 0, len(orders))
	for _, o := range orders {
		records = append(records, archivedOrder{
			OrderNumber: o.OrderNumber,
			Status:      string(o.Status),
			Subtotal:    o.Subtotal,
			DeliveryFee: o.DeliveryFee,
			PlatformFee: o.PlatformFee,
			Tax:         o.Tax,
			TaxName:     o.TaxName,
			Total:       o.Total,
			CreatedAt:   o.CreatedAt,
		})
	}

	// No orders means no statutory record to keep — skip the write rather than
	// leave an object whose only content is an identifier.
	if len(records) == 0 {
		return nil
	}

	doc := map[string]any{
		"schema":     "homechef.account-financial-archive.v1",
		"archivedAt": time.Now().UTC().Format(time.RFC3339),
		"userId":     user.ID.String(),
		"role":       string(user.Role),
		"notice": "Financial records retained for statutory accounting purposes. " +
			"All personal data for this account was erased at archival time.",
		"orders": records,
	}

	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("archive: encode: %w", err)
	}

	objectPath := fmt.Sprintf("account-archives/%s/%s.json",
		time.Now().UTC().Format("2006/01"), user.ID)
	if _, err := UploadFile(ctx, config.AppConfig.GCSPrivateBucket, objectPath,
		bytes.NewReader(body), "application/json"); err != nil {
		return fmt.Errorf("archive: upload: %w", err)
	}
	return nil
}
