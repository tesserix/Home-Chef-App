package handlers

// admin_payout.go — the admin payout release queue endpoints (#388). Admin-only
// (mounted under the /admin group's bffAuth + RequireAdmin, so no per-route auth
// here). Every method parses/validates, calls the payout_release service, and
// returns a structured JSON envelope {"error": code, "message": ...}; no panics.
//
// Money movement stays behind the escrow flags in the service seam — with both
// OFF (launch), release/withhold/reverse are DB-only state advances. Every action
// writes an audit_logs row (actor + old→new + mandatory reason for withhold/reverse).

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type AdminPayoutHandler struct{}

func NewAdminPayoutHandler() *AdminPayoutHandler { return &AdminPayoutHandler{} }

// reasonReq is the mandatory-reason body for withhold/reverse.
type reasonReq struct {
	Reason string `json:"reason"`
}

// parseAggType accepts only the two payout aggregates; anything else → 400.
func parseAggType(c *gin.Context) (string, bool) {
	agg := c.Param("aggType")
	if agg != "order" && agg != "meal-plan-day" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_agg_type", "message": "Unknown payout type."})
		return "", false
	}
	return agg, true
}

// parsePayoutID reads the :id path param as a UUID.
func parsePayoutID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id", "message": "Invalid payout id."})
		return uuid.Nil, false
	}
	return id, true
}

// respondActionError maps a service error to the right status + code. A racing /
// duplicate action (ErrHoldNotEligible) is a 409, not a 500.
func respondActionError(c *gin.Context, err error) {
	if err == services.ErrHoldNotEligible {
		c.JSON(http.StatusConflict, gin.H{"error": "hold_not_eligible",
			"message": "This payout is no longer in a state that allows this action."})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "payout_action_failed",
		"message": "Could not complete the payout action."})
}

// requireReason binds and trims the mandatory reason; empty → 400.
func requireReason(c *gin.Context) (string, bool) {
	var req reasonReq
	_ = c.ShouldBindJSON(&req)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason_required", "message": "A reason is required."})
		return "", false
	}
	return reason, true
}

// GetPendingPayouts lists release-eligible holds (awaiting via ?include=awaiting),
// optionally scoped by ?chefId / ?before. GET /admin/payouts/pending.
func (h *AdminPayoutHandler) GetPendingPayouts(c *gin.Context) {
	f := services.PendingFilter{IncludeAwaiting: c.Query("include") == "awaiting"}
	if v := c.Query("chefId"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			f.ChefID = id
		}
	}
	if v := c.Query("before"); v != "" {
		if ts, err := time.Parse(time.RFC3339, v); err == nil {
			f.Before = &ts
		}
	}
	rows, err := services.ListPendingPayouts(database.DB, f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "payout_list_failed",
			"message": "Could not load pending payouts."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"payouts": rows, "count": len(rows)})
}

// ReleasePayout advances one hold release_eligible → released and drives the
// flag-gated release seam. POST /admin/payouts/:aggType/:id/release.
func (h *AdminPayoutHandler) ReleasePayout(c *gin.Context) {
	agg, ok := parseAggType(c)
	if !ok {
		return
	}
	id, ok := parsePayoutID(c)
	if !ok {
		return
	}
	if err := services.ReleaseHold(database.DB, agg, id); err != nil {
		respondActionError(c, err)
		return
	}
	services.LogAudit(c, "payout.released", agg, id.String(),
		gin.H{"status": string(models.PayoutHoldReleaseEligible)}, gin.H{"status": string(models.PayoutHoldReleased)})
	c.JSON(http.StatusOK, gin.H{"status": string(models.PayoutHoldReleased)})
}

// WithholdPayout blocks an eligible payout (→ withheld) with a mandatory reason.
// POST /admin/payouts/:aggType/:id/withhold.
func (h *AdminPayoutHandler) WithholdPayout(c *gin.Context) {
	agg, ok := parseAggType(c)
	if !ok {
		return
	}
	id, ok := parsePayoutID(c)
	if !ok {
		return
	}
	reason, ok := requireReason(c)
	if !ok {
		return
	}
	if err := services.WithholdHold(database.DB, agg, id, reason); err != nil {
		respondActionError(c, err)
		return
	}
	services.LogAudit(c, "payout.withheld", agg, id.String(), nil,
		gin.H{"status": string(models.PayoutHoldWithheld), "reason": reason})
	c.JSON(http.StatusOK, gin.H{"status": string(models.PayoutHoldWithheld)})
}

// ReversePayout claws a released/eligible payout back (→ reversed) with a
// mandatory reason. POST /admin/payouts/:aggType/:id/reverse.
func (h *AdminPayoutHandler) ReversePayout(c *gin.Context) {
	agg, ok := parseAggType(c)
	if !ok {
		return
	}
	id, ok := parsePayoutID(c)
	if !ok {
		return
	}
	reason, ok := requireReason(c)
	if !ok {
		return
	}
	if err := services.ReverseHold(database.DB, agg, id, reason); err != nil {
		respondActionError(c, err)
		return
	}
	services.LogAudit(c, "payout.reversed", agg, id.String(), nil,
		gin.H{"status": string(models.PayoutHoldReversed), "reason": reason})
	c.JSON(http.StatusOK, gin.H{"status": string(models.PayoutHoldReversed)})
}

// bulkItem is one target in a bulk release request.
type bulkItem struct {
	AggType string `json:"aggType"`
	ID      string `json:"id"`
}

type bulkReleaseReq struct {
	Items  []bulkItem `json:"items"`
	Before *time.Time `json:"before,omitempty"`
}

// BulkReleasePayouts releases many holds, skipping ineligible ones (reported, not
// fatal). If items is empty and before is set, the eligible set is resolved from
// the pending queue. POST /admin/payouts/release-bulk.
func (h *AdminPayoutHandler) BulkReleasePayouts(c *gin.Context) {
	var req bulkReleaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_body", "message": "Invalid request body."})
		return
	}
	items := h.resolveBulkItems(req)
	releasedIDs := make([]string, 0, len(items))
	skipped := make([]gin.H, 0)
	for _, it := range items {
		id, err := uuid.Parse(it.ID)
		if err != nil || (it.AggType != "order" && it.AggType != "meal-plan-day") {
			skipped = append(skipped, gin.H{"id": it.ID, "reason": "invalid"})
			continue
		}
		if err := services.ReleaseHold(database.DB, it.AggType, id); err != nil {
			skipped = append(skipped, gin.H{"id": it.ID, "reason": "not_eligible"})
			continue
		}
		releasedIDs = append(releasedIDs, it.ID)
	}
	services.LogAudit(c, "payout.bulk_released", "payout", "",
		nil, gin.H{"released": len(releasedIDs), "skipped": len(skipped)})
	c.JSON(http.StatusOK, gin.H{"released": len(releasedIDs), "releasedIds": releasedIDs, "skipped": skipped})
}

// resolveBulkItems returns the explicit items, or (when none are given but a
// `before` cutoff is) the currently eligible holds older than that cutoff.
func (h *AdminPayoutHandler) resolveBulkItems(req bulkReleaseReq) []bulkItem {
	if len(req.Items) > 0 || req.Before == nil {
		return req.Items
	}
	rows, err := services.ListPendingPayouts(database.DB, services.PendingFilter{Before: req.Before})
	if err != nil {
		return nil
	}
	items := make([]bulkItem, 0, len(rows))
	for _, r := range rows {
		if r.HoldStatus == models.PayoutHoldReleaseEligible {
			items = append(items, bulkItem{AggType: r.AggType, ID: r.ID.String()})
		}
	}
	return items
}
