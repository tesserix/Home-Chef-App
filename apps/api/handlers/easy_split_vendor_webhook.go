package handlers

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// easy_split_vendor_webhook.go — Cashfree tells us a chef's bank account
// finished verifying (#1083). Without this the only thing that noticed was a
// 30-minute sweep, so a chef verified at 09:01 stayed unpayable until 09:30.

// cashfreeVendorEvent is the payload of every vendor-status webhook.
type cashfreeVendorEvent struct {
	VendorID string `json:"vendor_id"`
	Status   string `json:"status"`
	// Cashfree has used both spellings; whichever arrives is the only thing
	// that says WHICH detail was wrong.
	Remarks string `json:"remarks"`
	Reason  string `json:"reason"`
}

func (e cashfreeVendorEvent) why() string {
	if r := strings.TrimSpace(e.Remarks); r != "" {
		return r
	}
	return strings.TrimSpace(e.Reason)
}

// isCashfreeVendorEvent recognises the event by shape rather than by one exact
// type string: Cashfree has renamed these before, and the cron is the backstop
// for anything this misses.
func isCashfreeVendorEvent(eventType string) bool {
	return strings.Contains(strings.ToUpper(eventType), "VENDOR")
}

// handleCashfreeVendorStatus records a verification outcome against the chef
// holding that vendor id. An event for a vendor we never registered, or one
// carrying no status, is ignored — never an error, since Cashfree would retry
// it forever and nothing about it is going to change.
func (h *PaymentHandler) handleCashfreeVendorStatus(raw json.RawMessage) error {
	var event cashfreeVendorEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		log.Printf("cashfree vendor webhook: unreadable payload: %v", err)
		return nil
	}
	vendorID := strings.TrimSpace(event.VendorID)
	if vendorID == "" || strings.TrimSpace(event.Status) == "" {
		return nil
	}

	var chef models.ChefProfile
	if err := database.DB.First(&chef, "cashfree_vendor_id = ?", vendorID).Error; err != nil {
		log.Printf("cashfree vendor webhook: no chef for vendor %s — ignoring", vendorID)
		return nil
	}

	// Conditional on the status being replaced, so a webhook and a cron tick
	// reporting the same transition notify the chef once.
	services.ApplyEasySplitVendorStatusWithReason(database.DB, &chef, vendorID, event.Status, event.why())
	return nil
}
