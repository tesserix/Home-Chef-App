package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
		// Enough for an operator to recognise WHICH key is on file, without
		// ever rendering the key itself.
		"signingKeyFingerprint": client.SigningKeyFingerprint(),
		"clientSecretSet":       true,
		"webhookSecretSet":      client.HasWebhookSecret(),
		"slotWarning":           cashfreePayoutSlotWarning(slot, environment, clientID, client.SignatureConfigured()),
		"error":                 healthErr,
	})
}

// cashfreePayoutSlotWarning flags a slot whose configuration will bite later.
func cashfreePayoutSlotWarning(slot, environment, clientID string, signing bool) string {
	switch {
	case !models.IsTestMode(slot) && environment == "sandbox":
		// Now the expected state when the slot holds sandbox credentials: the
		// host follows the keys, so this works and disburses nothing real.
		return "The Live slot is holding sandbox credentials — payouts are routed to the Cashfree SANDBOX and no real money is disbursed. Enter live credentials before paying chefs for real."
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
// GET /admin/payouts/batches?state=pending_approval&payeeId=<uuid>
func (h *AdminPayoutRailHandler) ListPayoutBatches(c *gin.Context) {
	q := database.DB.Model(&payouts.Batch{})
	if state := strings.TrimSpace(c.Query("state")); state != "" {
		q = q.Where("state = ?", state)
	}
	if payee := strings.TrimSpace(c.Query("payeeId")); payee != "" {
		payeeID, err := uuid.Parse(payee)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payeeId"})
			return
		}
		q = q.Where("payee_id = ?", payeeID)
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
	batch, decision, pErr := services.PrepareStatementBatch(c.Request.Context(), database.DB, &stmt, mode)
	if pErr != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": pErr.Error()})
		return
	}
	services.LogAudit(c, "payout.batch.prepare", "payout_batch", batch.ID.String(), nil, map[string]any{
		"statementId": stmt.ID, "state": string(batch.State),
		"amount": services.FromPaise(int(batch.AmountMinor)), "holdReasons": decision.Reasons,
	})
	c.JSON(http.StatusOK, gin.H{
		"batchId": batch.ID, "state": batch.State,
		"amount":      services.FromPaise(int(batch.AmountMinor)),
		"holdReasons": decision.Reasons,
	})
}

// --- Per-chef payout profile ---

// Cashfree's documented sandbox test bank accounts that simulate SUCCESSFUL
// transfers (docs → Payouts → Test Data). Registering one against the sandbox
// rail yields a VERIFIED beneficiary, which is what lets the whole
// prepare → approve → execute flow run end to end with no real chef data.
//
// A table rather than one account because the sandbox enforces one beneficiary
// per account/IFSC across the whole merchant account — seeding every chef with
// the same account would 409 from the second chef on. Rotating by chef id keeps
// collisions away until the accounts are exhausted, and the 409 handler turns
// the eventual collision into a readable rejection rather than a mystery.
var cashfreeSandboxTestAccounts = []struct{ account, ifsc string }{
	{"00011020001772", "HDFC0000001"},
	{"026291800001191", "YESB0000262"},
	{"1233943142", "ICIC0000009"},
	{"388108022658", "ICIC0000009"},
	{"000890289871772", "SCBL0036078"},
	{"000100289877623", "SBIN0008752"},
}

func sandboxTestAccountFor(chefID uuid.UUID) (account, ifsc string) {
	sum := 0
	for _, b := range chefID[:] {
		sum += int(b)
	}
	pick := cashfreeSandboxTestAccounts[sum%len(cashfreeSandboxTestAccounts)]
	return pick.account, pick.ifsc
}

type chefPayoutMethodRow struct {
	ID                uuid.UUID  `json:"id"`
	Kind              string     `json:"kind"`
	Status            string     `json:"status"`
	Primary           bool       `json:"primary"`
	DisplayHint       string     `json:"displayHint"`
	BeneficiaryName   string     `json:"beneficiaryName"`
	Rail              string     `json:"rail"`
	RailBeneficiaryID string     `json:"railBeneficiaryId,omitempty"`
	RailStatusDetail  string     `json:"railStatusDetail,omitempty"`
	VerifiedAt        *time.Time `json:"verifiedAt,omitempty"`
	Payable           bool       `json:"payable"`
}

func chefPayoutMethodRows(chefID uuid.UUID) ([]chefPayoutMethodRow, error) {
	var methods []payouts.PayoutMethod
	if err := database.DB.
		Where("payee_type = ? AND payee_id = ?", payouts.PayeeChef, chefID).
		Order("created_at ASC").Find(&methods).Error; err != nil {
		return nil, err
	}
	rows := make([]chefPayoutMethodRow, 0, len(methods))
	for i := range methods {
		m := &methods[i]
		rows = append(rows, chefPayoutMethodRow{
			ID: m.ID, Kind: string(m.Kind), Status: string(m.Status),
			Primary: m.Primary, DisplayHint: m.DisplayHint,
			BeneficiaryName: m.BeneficiaryName, Rail: m.Rail,
			RailBeneficiaryID: m.RailBeneficiaryID, RailStatusDetail: m.RailStatusDetail,
			VerifiedAt: m.VerifiedAt, Payable: m.Payable(),
		})
	}
	return rows, nil
}

// GetChefPayoutProfile is everything the admin needs to judge "can this chef be
// paid, and where does the money go" — masked destination, the rail's opinion of
// the beneficiary, and whether the slot behind it is sandbox or live.
//
// GET /admin/chefs/:id/payout-profile
func (h *AdminPayoutRailHandler) GetChefPayoutProfile(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	ctx := c.Request.Context()
	vendorID := chef.ID.String()
	// Same masking rules as the chef's own settings screen: the account number
	// and VPA never render whole anywhere, including here.
	accountName, _ := services.GetVendorSecret(ctx, vendorID, "bank-account-name")
	accountNumber, _ := services.GetVendorSecret(ctx, vendorID, "bank-account-number")
	ifsc, _ := services.GetVendorSecret(ctx, vendorID, "bank-ifsc")
	upiID, _ := services.GetVendorSecret(ctx, vendorID, "upi-id")

	methods, mErr := chefPayoutMethodRows(chef.ID)
	if mErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load payout methods"})
		return
	}

	railConfigured, railSandbox := false, false
	if client := services.GetCashfreePayoutFor(chef.Mode); client != nil {
		railConfigured = true
		railSandbox = client.IsSandbox()
	}

	capMinor, capUnreadable := services.PayoutAutoDisburseCap(database.DB)

	c.JSON(http.StatusOK, gin.H{
		"chef": gin.H{
			"id": chef.ID, "businessName": chef.BusinessName, "mode": chef.Mode,
		},
		"payoutMethod":      chef.PayoutMethod,
		"bankAccountName":   accountName,
		"bankAccountNumber": maskBankAccount(accountNumber),
		"bankIFSC":          ifsc,
		"upiId":             maskEmail(upiID),
		"methods":           methods,
		"rail": gin.H{
			"configured": railConfigured, "sandbox": railSandbox, "mode": chef.Mode,
		},
		"automation": gin.H{
			"disburse":           chef.PayoutAutoDisburse,
			"globalAutoDisburse": services.PayoutAutoDisburseEnabled(database.DB),
			"effective":          services.ChefAutoDisburseEnabled(database.DB, &chef),
			"autoCapMinor":       capMinor,
			"autoCapUnreadable":  capUnreadable,
		},
		"easySplit": gin.H{
			"enabled":   services.EasySplitEnabled(database.DB),
			"mode":      chef.EasySplitMode,
			"effective": services.EasySplitEnabledForChef(database.DB, &chef),
			"vendorId":  chef.CashfreeVendorID,
			"status":    chef.CashfreeVendorStatus,
		},
	})
}

// RegisterChefEasySplitVendor registers the chef's bank details as an Easy
// Split vendor — the destination split-at-capture settles to.
//
// POST /admin/chefs/:id/easy-split/register
func (h *AdminPayoutRailHandler) RegisterChefEasySplitVendor(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	vendor, esErr := services.EnsureEasySplitVendor(c.Request.Context(), database.DB, &chef)
	if esErr != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": esErr.Error()})
		return
	}
	services.LogAudit(c, "chef.easysplit.register", "chef", chefID.String(), nil, map[string]any{
		"vendorId": vendor.VendorID, "status": vendor.Status,
	})
	c.JSON(http.StatusOK, gin.H{"vendorId": vendor.VendorID, "status": vendor.Status})
}

// RefreshChefEasySplitVendor re-reads the vendor's verification state.
//
// POST /admin/chefs/:id/easy-split/refresh
func (h *AdminPayoutRailHandler) RefreshChefEasySplitVendor(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	vendor, esErr := services.RefreshEasySplitVendor(c.Request.Context(), database.DB, &chef)
	if esErr != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": esErr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"vendorId": vendor.VendorID, "status": vendor.Status})
}

// Platform settlement account — where Cashfree pays the COMPANY's share.
//
// Stored in Secret Manager only, never the DB, and always rendered masked.
// Cashfree settles the merchant share to the bank account verified in the
// merchant dashboard KYC; this record is the platform's own copy for
// operators to check the two match. The reserved id keeps it in the same
// vault namespace as chef bank details without colliding with a chef UUID.
const platformSettlementSecretID = "platform-settlement"

// GetPlatformSettlementAccount returns the masked company account.
//
// GET /admin/platform/settlement-account
func (h *AdminPayoutRailHandler) GetPlatformSettlementAccount(c *gin.Context) {
	ctx := c.Request.Context()
	name, _ := services.GetVendorSecret(ctx, platformSettlementSecretID, "bank-account-name")
	account, _ := services.GetVendorSecret(ctx, platformSettlementSecretID, "bank-account-number")
	ifsc, _ := services.GetVendorSecret(ctx, platformSettlementSecretID, "bank-ifsc")
	c.JSON(http.StatusOK, gin.H{
		"bankAccountName":   name,
		"bankAccountNumber": maskBankAccount(account),
		"bankIFSC":          ifsc,
		"configured":        account != "",
	})
}

// SetPlatformSettlementAccount stores the company current account.
//
// PUT /admin/platform/settlement-account
func (h *AdminPayoutRailHandler) SetPlatformSettlementAccount(c *gin.Context) {
	var req struct {
		BankAccountName   string `json:"bankAccountName" binding:"required"`
		BankAccountNumber string `json:"bankAccountNumber" binding:"required"`
		BankIFSC          string `json:"bankIFSC" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bankAccountName, bankAccountNumber and bankIFSC are required"})
		return
	}
	ctx := c.Request.Context()
	fields := map[string]string{
		"bank-account-name":   strings.TrimSpace(req.BankAccountName),
		"bank-account-number": strings.TrimSpace(req.BankAccountNumber),
		"bank-ifsc":           strings.ToUpper(strings.TrimSpace(req.BankIFSC)),
	}
	for field, value := range fields {
		if err := services.StoreVendorSecret(ctx, platformSettlementSecretID, field, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store settlement account"})
			return
		}
	}
	services.LogAudit(c, "platform.settlement_account.update", "platform", platformSettlementSecretID,
		nil, map[string]any{"bankAccountNumber": maskBankAccount(req.BankAccountNumber)})
	c.JSON(http.StatusOK, gin.H{
		"bankAccountName":   fields["bank-account-name"],
		"bankAccountNumber": maskBankAccount(fields["bank-account-number"]),
		"bankIFSC":          fields["bank-ifsc"],
		"configured":        true,
	})
}

// validEasySplitMode reports whether the value is one of the three the
// tri-state recognises. Anything else must be refused rather than stored: an
// unrecognised string reads back as "follow the platform flag", which is how a
// typo would move a chef's money onto a rail nobody chose (#1084).
func validEasySplitMode(value string) bool {
	switch value {
	case services.PayoutAutoOn, services.PayoutAutoOff, "":
		return true
	}
	return false
}

// SetChefEasySplitMode flips one chef's Easy Split rollout override.
//
// PUT /admin/chefs/:id/easy-split-mode
func (h *AdminPayoutRailHandler) SetChefEasySplitMode(c *gin.Context) {
	var req struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !validEasySplitMode(req.Value) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "value must be on, off or empty"})
		return
	}
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	old := chef.EasySplitMode
	if err := database.DB.Model(&chef).Update("easy_split_mode", req.Value).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
		return
	}
	chef.EasySplitMode = req.Value

	services.LogAudit(c, "chef.payout.easy_split_mode", "chef", chefID.String(),
		gin.H{"easySplitMode": old}, gin.H{"easySplitMode": req.Value})
	c.JSON(http.StatusOK, gin.H{
		"easySplitMode": req.Value,
		"effective":     services.EasySplitEnabledForChef(database.DB, &chef),
	})
}

// SetEasySplitModeBulk flips a whole cohort in one call — how a rollout
// actually proceeds. One statement, so a closed browser tab cannot leave half
// the cohort on one rail and half on the other.
//
// PUT /admin/chefs/easy-split-mode
func (h *AdminPayoutRailHandler) SetEasySplitModeBulk(c *gin.Context) {
	var req struct {
		Value   string      `json:"value"`
		ChefIDs []uuid.UUID `json:"chefIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !validEasySplitMode(req.Value) || len(req.ChefIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chefIds and a value of on, off or empty are required"})
		return
	}
	ids := make([]string, 0, len(req.ChefIDs))
	for _, id := range req.ChefIDs {
		ids = append(ids, id.String())
	}
	res := database.DB.Model(&models.ChefProfile{}).Where("id IN ?", ids).
		Update("easy_split_mode", req.Value)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update"})
		return
	}

	services.LogAudit(c, "chef.payout.easy_split_mode_bulk", "chef", "",
		nil, gin.H{"easySplitMode": req.Value, "chefIds": ids})
	c.JSON(http.StatusOK, gin.H{"easySplitMode": req.Value, "updated": res.RowsAffected})
}

// SetChefDisburseAutomation flips the chef's disbursement auto-approval
// tri-state. Same closed value set as SetPayoutAutomation (#747): an
// unrecognised string must never read back as "follow the default".
//
// PUT /admin/chefs/:id/disburse-automation
func (h *AdminPayoutRailHandler) SetChefDisburseAutomation(c *gin.Context) {
	var req struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	switch req.Value {
	case services.PayoutAutoOn, services.PayoutAutoOff, "":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "value must be on, off or empty"})
		return
	}

	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	old := chef.PayoutAutoDisburse
	if err := database.DB.Model(&chef).Update("payout_auto_disburse", req.Value).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update"})
		return
	}

	services.LogAudit(c, "chef.payout.disburse_automation", "chef", chefID.String(),
		gin.H{"payoutAutoDisburse": old}, gin.H{"payoutAutoDisburse": req.Value})
	c.JSON(http.StatusOK, gin.H{
		"payoutAutoDisburse": req.Value,
		"effective":          services.ChefAutoDisburseEnabled(database.DB, &chef),
	})
}

// RefreshChefPayoutMethod re-registers the chef's destination with the rail and
// re-reads its verdict — the answer to "it says pending/invalid, is that still
// true?". Idempotent: the deterministic beneficiary id makes re-registration
// resolve to the existing beneficiary, never a second one.
//
// POST /admin/chefs/:id/payout-methods/refresh
func (h *AdminPayoutRailHandler) RefreshChefPayoutMethod(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	if services.GetCashfreePayoutFor(chef.Mode) == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf(
			"Cashfree Payouts is not configured for the chef's %q mode — set it up on the payout rail panel first.", chef.Mode)})
		return
	}

	method, mErr := services.EnsurePayoutMethod(c.Request.Context(), database.DB,
		payouts.PayeeRef{Type: payouts.PayeeChef, ID: chef.ID}, chef.Mode)
	if mErr != nil && method == nil {
		// No destination on file, or the rail was unreachable. Nothing was
		// recorded, so this is the admin's answer rather than an internal error.
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": mErr.Error()})
		return
	}

	services.LogAudit(c, "payout.method.refresh", "chef", chefID.String(), nil, map[string]any{
		"status": string(method.Status), "rail": method.Rail,
	})

	resp := gin.H{"method": chefPayoutMethodRow{
		ID: method.ID, Kind: string(method.Kind), Status: string(method.Status),
		Primary: method.Primary, DisplayHint: method.DisplayHint,
		BeneficiaryName: method.BeneficiaryName, Rail: method.Rail,
		RailBeneficiaryID: method.RailBeneficiaryID, RailStatusDetail: method.RailStatusDetail,
		VerifiedAt: method.VerifiedAt, Payable: method.Payable(),
	}}
	if mErr != nil {
		// Rejected by the rail — recorded as invalid, and the detail is on the row.
		resp["warning"] = mErr.Error()
	}
	c.JSON(http.StatusOK, resp)
}

// SeedChefTestBankAccount puts Cashfree's documented sandbox test account on
// file for a chef — secrets, payout method selector and rail beneficiary — so
// the payout flow can be exercised without inventing bank details by hand.
//
// Hard-refused unless the rail the chef's mode resolves to is the SANDBOX: on a
// live slot this would register a fake destination real money could be sent to.
//
// POST /admin/chefs/:id/payout-methods/test-bank
func (h *AdminPayoutRailHandler) SeedChefTestBankAccount(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	client := services.GetCashfreePayoutFor(chef.Mode)
	if client == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf(
			"Cashfree Payouts is not configured for the chef's %q mode.", chef.Mode)})
		return
	}
	if !client.IsSandbox() {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf(
			"Refusing: the chef's %q mode resolves to the LIVE Cashfree rail — a test bank account may only be registered against the sandbox.", chef.Mode)})
		return
	}

	accountName := strings.TrimSpace(chef.BusinessName)
	if accountName == "" {
		accountName = "Test Chef"
	}

	testAccount, testIFSC := sandboxTestAccountFor(chef.ID)

	// Synchronous, unlike the chef's own save: the entire point of this action is
	// that a prepare can immediately resolve the instrument from Secret Manager,
	// so a failed write must fail the request.
	ctx := c.Request.Context()
	for field, value := range map[string]string{
		"bank-account-name":   accountName,
		"bank-account-number": testAccount,
		"bank-ifsc":           testIFSC,
	} {
		if sErr := services.StoreVendorSecret(ctx, chef.ID.String(), field, value); sErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to store %s: %v", field, sErr)})
			return
		}
	}
	if uErr := database.DB.Model(&models.ChefProfile{}).Where("id = ?", chef.ID).
		Update("payout_method", "bank_transfer").Error; uErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update chef payout method"})
		return
	}

	method, mErr := services.EnsurePayoutMethodWith(ctx, database.DB,
		payouts.PayeeRef{Type: payouts.PayeeChef, ID: chef.ID}, chef.Mode,
		payouts.Instrument{Kind: payouts.MethodBankAccount,
			AccountNumber: testAccount, IFSC: testIFSC},
		accountName)
	if mErr != nil && method == nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": mErr.Error()})
		return
	}

	services.LogAudit(c, "payout.method.seed_test", "chef", chefID.String(), nil, map[string]any{
		"status": string(method.Status), "displayHint": method.DisplayHint,
	})

	resp := gin.H{
		"message": "Sandbox test bank account registered",
		"method": chefPayoutMethodRow{
			ID: method.ID, Kind: string(method.Kind), Status: string(method.Status),
			Primary: method.Primary, DisplayHint: method.DisplayHint,
			BeneficiaryName: method.BeneficiaryName, Rail: method.Rail,
			RailBeneficiaryID: method.RailBeneficiaryID, RailStatusDetail: method.RailStatusDetail,
			VerifiedAt: method.VerifiedAt, Payable: method.Payable(),
		},
	}
	if mErr != nil {
		resp["warning"] = mErr.Error()
	}
	c.JSON(http.StatusOK, resp)
}

// GetPayoutSettings / UpdatePayoutSettings expose the auto-disburse flag.
//
// GET/PUT /admin/payouts/settings
func (h *AdminPayoutRailHandler) GetPayoutSettings(c *gin.Context) {
	capMinor, capUnreadable := services.PayoutAutoDisburseCap(database.DB)
	feeMinor, feeOK := services.PlatformFeeFlatMinor(database.DB)
	c.JSON(http.StatusOK, gin.H{
		"autoDisburseEnabled":   services.PayoutAutoDisburseEnabled(database.DB),
		"autoCapMinor":          capMinor,
		"autoCapUnreadable":     capUnreadable,
		"easySplitEnabled":      services.EasySplitEnabled(database.DB),
		"platformFeeFlatMinor":  feeMinor,
		"platformFeeUnreadable": !feeOK,
	})
}

func upsertPayoutSetting(key, value, kind string, actorID uuid.UUID) error {
	res := database.DB.Model(&models.PlatformSettings{}).
		Where("key = ?", key).
		Updates(map[string]any{"value": value, "updated_by": actorID})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return database.DB.Create(&models.PlatformSettings{
			Key: key, Value: value, Type: kind, UpdatedBy: &actorID,
		}).Error
	}
	return nil
}

func (h *AdminPayoutRailHandler) UpdatePayoutSettings(c *gin.Context) {
	var req struct {
		AutoDisburseEnabled  *bool  `json:"autoDisburseEnabled"`
		AutoCapMinor         *int64 `json:"autoCapMinor"`
		EasySplitEnabled     *bool  `json:"easySplitEnabled"`
		PlatformFeeFlatMinor *int64 `json:"platformFeeFlatMinor"`
	}
	if err := c.ShouldBindJSON(&req); err != nil ||
		(req.AutoDisburseEnabled == nil && req.AutoCapMinor == nil &&
			req.EasySplitEnabled == nil && req.PlatformFeeFlatMinor == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one setting is required"})
		return
	}
	if req.AutoCapMinor != nil && *req.AutoCapMinor < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "autoCapMinor must be >= 0"})
		return
	}
	if req.PlatformFeeFlatMinor != nil && *req.PlatformFeeFlatMinor < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platformFeeFlatMinor must be >= 0"})
		return
	}

	actor, _ := c.Get("userID")
	actorID, _ := actor.(uuid.UUID)

	if req.AutoDisburseEnabled != nil {
		value := "false"
		if *req.AutoDisburseEnabled {
			value = "true"
		}
		if err := upsertPayoutSetting(services.SettingPayoutAutoDisburse, value, "bool", actorID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting"})
			return
		}
		// Turning this ON means money can leave with no human in the loop. It is
		// the single most consequential toggle on the payouts screen, so it is
		// audited with the old value alongside the new.
		services.LogAudit(c, "payout.settings.update", "payout_settings", services.SettingPayoutAutoDisburse,
			map[string]any{"autoDisburseEnabled": !*req.AutoDisburseEnabled},
			map[string]any{"autoDisburseEnabled": *req.AutoDisburseEnabled})
	}

	if req.AutoCapMinor != nil {
		if err := upsertPayoutSetting(services.SettingPayoutAutoDisburseMax,
			strconv.FormatInt(*req.AutoCapMinor, 10), "number", actorID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting"})
			return
		}
		services.LogAudit(c, "payout.settings.update", "payout_settings", services.SettingPayoutAutoDisburseMax,
			nil, map[string]any{"autoCapMinor": *req.AutoCapMinor})
	}

	if req.EasySplitEnabled != nil {
		value := "false"
		if *req.EasySplitEnabled {
			value = "true"
		}
		if err := upsertPayoutSetting(services.SettingEasySplitEnabled, value, "bool", actorID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting"})
			return
		}
		// Split-at-capture reroutes real money at the gateway — audited like
		// the auto-disburse flag, with both sides of the flip.
		services.LogAudit(c, "payout.settings.update", "payout_settings", services.SettingEasySplitEnabled,
			map[string]any{"easySplitEnabled": !*req.EasySplitEnabled},
			map[string]any{"easySplitEnabled": *req.EasySplitEnabled})
	}

	if req.PlatformFeeFlatMinor != nil {
		if err := upsertPayoutSetting(services.SettingPlatformFeeFlatMinor,
			strconv.FormatInt(*req.PlatformFeeFlatMinor, 10), "number", actorID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting"})
			return
		}
		services.LogAudit(c, "payout.settings.update", "payout_settings", services.SettingPlatformFeeFlatMinor,
			nil, map[string]any{"platformFeeFlatMinor": *req.PlatformFeeFlatMinor})
	}

	capMinor, capUnreadable := services.PayoutAutoDisburseCap(database.DB)
	feeMinor, feeOK := services.PlatformFeeFlatMinor(database.DB)
	c.JSON(http.StatusOK, gin.H{
		"autoDisburseEnabled":   services.PayoutAutoDisburseEnabled(database.DB),
		"autoCapMinor":          capMinor,
		"autoCapUnreadable":     capUnreadable,
		"easySplitEnabled":      services.EasySplitEnabled(database.DB),
		"platformFeeFlatMinor":  feeMinor,
		"platformFeeUnreadable": !feeOK,
	})
}
