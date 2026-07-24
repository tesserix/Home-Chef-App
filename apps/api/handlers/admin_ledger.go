package handlers

// admin_ledger.go — operator endpoints for the wallet→ledger migration
// (docs/wallet-ledger-plan.md). The one-time opening-balance backfill and the shadow-phase
// reconcile status. Both are admin-gated + audited; the backfill is idempotent and the reconcile
// is read-only and NEVER corrects.

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/services"
)

// BackfillLedger seeds a double-entry OPENING_BALANCE entry per wallet equal to the gap between
// its legacy float balance and its current ledger balance, so the ledger projection starts equal
// to the legacy balance (the one-time migration step, §2). Idempotent — one opening per user, so
// re-running is safe and posts nothing the second time. The ledger schema must be deployed first
// (tesserix-k8s); run this around enabling LEDGER_SHADOW_ENABLED (the `legacy − ledger` delta is
// correct whether dual-write started before or after). POST /admin/ledger/backfill
func (h *AdminHandler) BackfillLedger(c *gin.Context) {
	n, err := services.BackfillLedgerOpeningBalances(database.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to backfill ledger opening balances"})
		return
	}
	services.LogAudit(c, "ledger.backfill", "ledger", "", nil, map[string]any{"openingsPosted": n})
	c.JSON(http.StatusOK, gin.H{"openingsPosted": n})
}

// LedgerReconcileStatus reports, per wallet, any mismatch between the legacy float balance and the
// double-entry ledger projection (paise) — the shadow-phase bake signal. Empty drift means the
// ledger tracks the legacy balance everywhere. It NEVER corrects (a mismatch is investigated).
// GET /admin/ledger/reconcile
func (h *AdminHandler) LedgerReconcileStatus(c *gin.Context) {
	drift, err := services.ReconcileLedgerVsWallet(database.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reconcile ledger"})
		return
	}
	debit, credit, err := services.LedgerTotals(database.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read ledger totals"})
		return
	}
	out := make([]gin.H, 0, len(drift))
	for _, d := range drift {
		out = append(out, gin.H{
			"userId":      d.UserID.String(),
			"legacyMinor": int64(d.LegacyMinor),
			"ledgerMinor": int64(d.LedgerMinor),
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"shadowEnabled":    services.LedgerShadowActive(),
		"balanced":         debit == credit,
		"totalDebitMinor":  int64(debit),
		"totalCreditMinor": int64(credit),
		"driftCount":       len(drift),
		"drift":            out,
	})
}
