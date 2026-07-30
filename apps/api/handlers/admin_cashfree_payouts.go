package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
	"github.com/homechef/api/services"
)

// admin_cashfree_payouts.go — the admin surface for money going OUT.
//
// Two things live here: the Cashfree Payouts credential slots, and the queue an
// admin approves batches from. They are together because they are one screen —
// an operator looking at a stuck payout needs to see whether the rail is even
// authenticated, and that is the first question when a disbursement 403s.
//
// Everything that moves money requires an explicit action. There is no endpoint
// here that disburses as a side effect of reading.

type AdminPayoutRailHandler struct{}

func NewAdminPayoutRailHandler() *AdminPayoutRailHandler { return &AdminPayoutRailHandler{} }

// GetCashfreePayoutStatus reports one credential slot's health.
//
// GET /admin/payouts/cashfree/status?mode=live|test
func (h *AdminPayoutRailHandler) GetCashfreePayoutStatus(c *gin.Context) {
	slot := models.NormalizeMode(c.Query("mode"))
	client := services.GetCashfreePayoutFor(slot)

	if client == nil {
		c.JSON(http.StatusOK, gin.H{
			"configured":          false,
			"slot":                slot,
			"environment":         "unknown",
			"keyPrefix":           "",
			"signatureConfigured": false,
			"error":               fmt.Sprintf("Cashfree Payouts %s slot is not configured.", slot),
		})
		return
	}

	clientID := client.GetClientID()
	keyPrefix := clientID
	if len(clientID) > 12 {
		keyPrefix = clientID[:12] + "..."
	}
	environment := "production"
	if client.IsSandbox() {
		environment = "sandbox"
	}

	healthErr := ""
	configured := true
	if err := client.HealthCheck(c.Request.Context()); err != nil {
		healthErr = err.Error()
		configured = false
	}

	c.JSON(http.StatusOK, gin.H{
		"configured":  configured,
		"slot":        slot,
		"environment": environment,
		"keyPrefix":   keyPrefix,
		// The single most useful field on this screen. Cashfree Payouts
		// authenticates by whitelisted source IP unless a signing key is
		// configured, and this platform's egress is a Cloud NAT address that can
		// be reallocated — so "am I signing, or am I depending on an IP that
		// might change?" is the first thing an operator needs to know when
		// payouts start returning 403.
		"signatureConfigured": client.SignatureConfigured(),
		"webhookSecretSet":    client.HasWebhookSecret(),
		"slotWarning":         cashfreePayoutSlotWarning(slot, environment, clientID, client.SignatureConfigured()),
		"error":               healthErr,
	})
}

// cashfreePayoutSlotWarning flags a slot whose configuration will bite later.
func cashfreePayoutSlotWarning(slot, environment, clientID string, signing bool) string {
	switch {
	case !models.IsTestMode(slot) && environment == "sandbox":
		return "The Live slot is resolving to the Cashfree SANDBOX — no real money will be disbursed."
	case models.IsTestMode(slot) && environment == "production":
		return "The Test slot is resolving to Cashfree PRODUCTION — a test payout would send real money. Fix this before using test mode."
	case !signing:
		// Not an error today, but it is the failure waiting to happen: an IP
		// whitelist works until the NAT address is reallocated, and then every
		// payout 403s with valid credentials and nothing failing to explain it.
		return "No signing key configured — this slot authenticates by whitelisted IP. The platform's egress IP can change, which would silently stop all payouts. Ask Cashfree for a Payouts public key."
	default:
		return ""
	}
}

// UpdateCashfreePayoutKeys writes one credential slot to Secret Manager.
//
// PUT /admin/payouts/cashfree/keys
func (h *AdminPayoutRailHandler) UpdateCashfreePayoutKeys(c *gin.Context) {
	var req struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
		// PublicKey is the PEM Cashfree issues for signature auth. Optional, but
		// it is what removes the IP-whitelist dependency.
		PublicKey     string `json:"publicKey"`
		WebhookSecret string `json:"webhookSecret"`
		Mode          string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ClientID == "" && req.ClientSecret == "" && req.PublicKey == "" && req.WebhookSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one field is required"})
		return
	}
	// A mismatched pair is a guaranteed 401 — require both or neither.
	if (req.ClientID == "") != (req.ClientSecret == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clientId and clientSecret must be provided together"})
		return
	}

	ctx := c.Request.Context()
	slot := models.NormalizeMode(req.Mode)
	idName, secretName, webhookName := services.CashfreePayoutSecretNames(slot)

	for name, value := range map[string]string{
		idName:      req.ClientID,
		secretName:  req.ClientSecret,
		webhookName: req.WebhookSecret,
		services.CashfreePayoutPublicKeySecretName(slot): req.PublicKey,
	} {
		if value == "" {
			continue
		}
		if err := services.StorePlatformSecret(ctx, name, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to store %s: %v", name, err)})
			return
		}
	}

	services.InvalidateCashfreePayoutFor(slot)
	services.LogAudit(c, "payout.keys.update", "payout_rail", services.CashfreePayoutRailName, nil, map[string]any{
		"slot": slot,
		"updatedFields": []string{
			boolField("clientId", req.ClientID != ""),
			boolField("clientSecret", req.ClientSecret != ""),
			boolField("publicKey", req.PublicKey != ""),
			boolField("webhookSecret", req.WebhookSecret != ""),
		},
	})

	client := services.GetCashfreePayoutFor(slot)
	if client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Saved to Secret Manager, but client failed to initialize"})
		return
	}
	if err := client.HealthCheck(ctx); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message":   "Credentials saved, but validation failed",
			"testError": err.Error(),
			"slot":      slot,
			"verified":  false,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":  "Cashfree Payouts credentials saved and verified",
		"slot":     slot,
		"verified": true,
	})
}

// --- The payout queue ---

type payoutBatchRow struct {
	ID            uuid.UUID `json:"id"`
	PayeeType     string    `json:"payeeType"`
	PayeeID       uuid.UUID `json:"payeeId"`
	PayeeName     string    `json:"payeeName"`
	BusinessDate  string    `json:"businessDate"`
	State         string    `json:"state"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Destination   string    `json:"destination"`
	Provider      string    `json:"provider"`
	ProviderRef   string    `json:"providerRef,omitempty"`
	ProviderUTR   string    `json:"providerUtr,omitempty"`
	FailureCode   string    `json:"failureCode,omitempty"`
	FailureDetail string    `json:"failureDetail,omitempty"`
	CreatedAt     string    `json:"createdAt"`
}

// ListPayoutBatches returns the queue, newest first.
//
// GET /admin/payouts/batches?state=pending_approval
func (h *AdminPayoutRailHandler) ListPayoutBatches(c *gin.Context) {
	q := database.DB.Model(&payouts.Batch{})
	if state := strings.TrimSpace(c.Query("state")); state != "" {
		q = q.Where("state = ?", state)
	}
	var batches []payouts.Batch
	if err := q.Order("created_at DESC").Limit(200).Find(&batches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load payout batches"})
		return
	}

	rows := make([]payoutBatchRow, 0, len(batches))
	for i := range batches {
		b := &batches[i]
		row := payoutBatchRow{
			ID:            b.ID,
			PayeeType:     string(b.PayeeType),
			PayeeID:       b.PayeeID,
			BusinessDate:  b.BusinessDate,
			State:         string(b.State),
			Amount:        services.FromPaise(int(b.AmountMinor)),
			Currency:      string(b.Currency),
			Provider:      b.Provider,
			ProviderRef:   b.ProviderRef,
			ProviderUTR:   b.ProviderUTR,
			FailureCode:   b.FailureCode,
			FailureDetail: b.FailureDetail,
			CreatedAt:     b.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		// Destination is the MASKED hint only. The admin queue must be able to
		// tell two destinations apart without this screen ever being a place
		// account numbers can be read.
		if b.MethodID != nil {
			var m payouts.PayoutMethod
			if database.DB.First(&m, "id = ?", *b.MethodID).Error == nil {
				row.Destination = m.DisplayHint
				row.PayeeName = m.BeneficiaryName
			}
		}
		rows = append(rows, row)
	}
	c.JSON(http.StatusOK, gin.H{"batches": rows, "count": len(rows)})
}

// ApprovePayoutBatch clears a batch to execute. Does NOT move money.
//
// Separating approval from execution is deliberate: it makes the decision
// auditable on its own, and it means the act that actually sends money is a
// single, explicit call rather than a side effect of approving.
//
// POST /admin/payouts/batches/:id/approve
func (h *AdminPayoutRailHandler) ApprovePayoutBatch(c *gin.Context) {
	batchID, err := uuid.Parse(c.Param("batchId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid batch id"})
		return
	}
	var batch payouts.Batch
	if err := database.DB.First(&batch, "id = ?", batchID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Batch not found"})
		return
	}
	if _, tErr := batch.State.Transition(payouts.BatchApproved); tErr != nil {
		c.JSON(http.StatusConflict, gin.H{"error": tErr.Error()})
		return
	}

	actor, _ := c.Get("userID")
	actorID, _ := actor.(uuid.UUID)
	now := database.DB.NowFunc()

	// Guarded so two admins clicking at once produce one approval.
	res := database.DB.Model(&payouts.Batch{}).
		Where("id = ? AND state = ?", batchID, payouts.BatchPendingApproval).
		Updates(map[string]any{
			"state": payouts.BatchApproved, "approved_by": actorID, "approved_at": now,
		})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve batch"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Batch is no longer awaiting approval"})
		return
	}

	services.LogAudit(c, "payout.batch.approve", "payout_batch", batchID.String(), nil, map[string]any{
		"amount": services.FromPaise(int(batch.AmountMinor)), "payee": batch.PayeeID.String(),
	})
	c.JSON(http.StatusOK, gin.H{"id": batchID, "state": payouts.BatchApproved})
}

// CancelPayoutBatch abandons a batch before execution.
//
// POST /admin/payouts/batches/:id/cancel
func (h *AdminPayoutRailHandler) CancelPayoutBatch(c *gin.Context) {
	batchID, err := uuid.Parse(c.Param("batchId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid batch id"})
		return
	}
	var batch payouts.Batch
	if err := database.DB.First(&batch, "id = ?", batchID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Batch not found"})
		return
	}
	// executing -> cancelled is refused by the state machine, and that refusal
	// is the point: once the rail has been called the outcome must be
	// established, never assumed away. Surface it as a 409 rather than a 500.
	if _, tErr := batch.State.Transition(payouts.BatchCancelled); tErr != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "This batch cannot be cancelled — if it is executing, its outcome must be resolved against the rail first.",
		})
		return
	}

	var reason struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&reason)

	res := database.DB.Model(&payouts.Batch{}).
		Where("id = ? AND state IN ?", batchID,
			[]payouts.BatchState{payouts.BatchBuilding, payouts.BatchPendingApproval, payouts.BatchApproved}).
		Updates(map[string]any{"state": payouts.BatchCancelled, "reason_codes": reason.Reason})
	if res.Error != nil || res.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Batch is no longer cancellable"})
		return
	}

	services.LogAudit(c, "payout.batch.cancel", "payout_batch", batchID.String(), nil, map[string]any{
		"reason": reason.Reason,
	})
	c.JSON(http.StatusOK, gin.H{"id": batchID, "state": payouts.BatchCancelled})
}

// ExecutePayoutBatch sends an approved batch to the rail. THIS MOVES MONEY.
//
// POST /admin/payouts/batches/:id/execute
func (h *AdminPayoutRailHandler) ExecutePayoutBatch(c *gin.Context) {
	batchID, err := uuid.Parse(c.Param("batchId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid batch id"})
		return
	}
	var batch payouts.Batch
	if err := database.DB.First(&batch, "id = ?", batchID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Batch not found"})
		return
	}

	mode := services.PaymentModeForChef(batch.PayeeID)
	out, execErr := services.ExecuteBatch(c.Request.Context(), database.DB, batchID, mode)
	if execErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": execErr.Error()})
		return
	}

	services.LogAudit(c, "payout.batch.execute", "payout_batch", batchID.String(), nil, map[string]any{
		"amount": services.FromPaise(int(out.AmountMinor)), "state": string(out.State),
	})
	c.JSON(http.StatusOK, gin.H{
		"id": out.ID, "state": out.State,
		"providerRef": out.ProviderRef, "providerUtr": out.ProviderUTR,
	})
}

// PrepareStatementPayout builds the batch for a pending weekly statement.
//
// POST /admin/payouts/statements/:id/prepare
func (h *AdminPayoutRailHandler) PrepareStatementPayout(c *gin.Context) {
	stmtID, err := uuid.Parse(c.Param("statementId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid statement id"})
		return
	}
	var stmt models.WeeklyStatement
	if err := database.DB.First(&stmt, "id = ?", stmtID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Statement not found"})
		return
	}

	mode := services.PaymentModeForChef(stmt.ChefID)
	batch, pErr := services.PrepareStatementBatch(c.Request.Context(), database.DB, &stmt, mode)
	if pErr != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": pErr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"batchId": batch.ID, "state": batch.State,
		"amount": services.FromPaise(int(batch.AmountMinor)),
	})
}

// GetPayoutSettings / UpdatePayoutSettings expose the auto-disburse flag.
//
// GET/PUT /admin/payouts/settings
func (h *AdminPayoutRailHandler) GetPayoutSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"autoDisburseEnabled": services.PayoutAutoDisburseEnabled(database.DB),
	})
}

func (h *AdminPayoutRailHandler) UpdatePayoutSettings(c *gin.Context) {
	var req struct {
		AutoDisburseEnabled *bool `json:"autoDisburseEnabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.AutoDisburseEnabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "autoDisburseEnabled is required"})
		return
	}

	value := "false"
	if *req.AutoDisburseEnabled {
		value = "true"
	}
	actor, _ := c.Get("userID")
	actorID, _ := actor.(uuid.UUID)

	res := database.DB.Model(&models.PlatformSettings{}).
		Where("key = ?", services.SettingPayoutAutoDisburse).
		Updates(map[string]any{"value": value, "updated_by": actorID})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting"})
		return
	}
	if res.RowsAffected == 0 {
		if err := database.DB.Create(&models.PlatformSettings{
			Key: services.SettingPayoutAutoDisburse, Value: value, Type: "bool", UpdatedBy: &actorID,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create setting"})
			return
		}
	}

	// Turning this ON means money can leave with no human in the loop. It is the
	// single most consequential toggle on the payouts screen, so it is audited
	// with the old value alongside the new.
	services.LogAudit(c, "payout.settings.update", "payout_settings", services.SettingPayoutAutoDisburse,
		map[string]any{"autoDisburseEnabled": !*req.AutoDisburseEnabled},
		map[string]any{"autoDisburseEnabled": *req.AutoDisburseEnabled})

	c.JSON(http.StatusOK, gin.H{"autoDisburseEnabled": *req.AutoDisburseEnabled})
}
