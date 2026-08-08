package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
)

// payout_disbursement.go — turning a settled WeeklyStatement into money in a
// chef's bank account.
//
// This is the path models/statement.go describes as manual ("disbursement is
// currently MANUAL … Admin marks each statement paid after sending the
// transfer"). It stays compatible with that: a statement disbursed by hand is
// still marked paid by hand, and this only automates the ones that go through a
// batch.
//
// ── Ordering is the whole design ─────────────────────────────────────────────
//
// The batch is moved to `executing` and COMMITTED BEFORE the rail is called.
// That ordering is not stylistic:
//
//	commit executing → call rail → crash   ⇒ batch is `executing`; the poll
//	                                          resolves it against the rail. Safe.
//	call rail → crash → commit never runs  ⇒ batch is `approved`; the next run
//	                                          sees an un-executed batch and pays
//	                                          a second time. Money lost.
//
// So the window between the write and the call is deliberately empty of anything
// that can fail, and `executing` is a state the engine can always resolve —
// which is exactly why BatchState refuses executing → cancelled.

// PayoutTenantID scopes every payout row. HomeChef is single-tenant; the engine
// carries a tenant because it was written to outlive that, and hardcoding the
// value in one place is better than defaulting it in twenty.
const PayoutTenantID = "homechef"

// SettingPayoutAutoDisburse gates automatic disbursement.
//
// Default OFF. With it off every batch stops at pending_approval and waits for
// an admin; with it on the sweep disburses batches that clear the governor and
// an admin intervenes only to withhold. Off is the right default for a rail that
// moves real money out of the platform with no refund endpoint.
const SettingPayoutAutoDisburse = "payout_auto_disburse_enabled"

// PayoutAutoDisburseEnabled reads the flag. Fails CLOSED: any read error leaves
// payouts requiring approval, because the failure mode of "wrongly manual" is a
// delayed payout and the failure mode of "wrongly automatic" is unreviewed money
// leaving the platform.
func PayoutAutoDisburseEnabled(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	var value string
	if err := db.Model(&models.PlatformSettings{}).
		Where("key = ?", SettingPayoutAutoDisburse).
		Select("value").Scan(&value).Error; err != nil {
		log.Printf("payout-disburse: auto-disburse setting unreadable (%v) — requiring approval", err)
		return false
	}
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

// SettingPayoutAutoDisburseMax caps auto-approval, in minor units (paise).
// Absent or zero disables the cap; a present-but-unparseable value fails
// closed — every batch then queues for manual approval.
const SettingPayoutAutoDisburseMax = "payout_auto_disburse_max_minor"

// PayoutAutoDisburseCap reads the cap, reporting unreadable separately so the
// decision can fail closed instead of treating a typo as "no cap".
func PayoutAutoDisburseCap(db *gorm.DB) (capMinor int64, unreadable bool) {
	var value string
	if err := db.Model(&models.PlatformSettings{}).
		Where("key = ?", SettingPayoutAutoDisburseMax).
		Select("value").Scan(&value).Error; err != nil {
		log.Printf("payout-disburse: auto cap unreadable (%v) — requiring approval", err)
		return 0, true
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n < 0 {
		log.Printf("payout-disburse: auto cap %q unparseable — requiring approval", value)
		return 0, true
	}
	return n, false
}

// ChefAutoDisburseEnabled resolves whether this chef's batches are candidates
// for auto-approval: the chef's tri-state wins, "" follows the global flag.
// The guardrails in payouts.DecideAutoApprove still run on candidates.
func ChefAutoDisburseEnabled(db *gorm.DB, chef *models.ChefProfile) bool {
	if chef == nil {
		return false
	}
	switch chef.PayoutAutoDisburse {
	case PayoutAutoOn:
		return true
	case PayoutAutoOff:
		return false
	default:
		return PayoutAutoDisburseEnabled(db)
	}
}

// statementBusinessDate is the batch's business date for a weekly statement.
//
// WeekEnd (exclusive) minus a day — the last day the statement actually covers.
// Deterministic, so a re-run of the same statement collides with the existing
// batch on the (tenant, payee, business date) unique index rather than creating
// a second. That index is the last line of defence against a double payout, and
// it only works if this function is stable.
func statementBusinessDate(stmt *models.WeeklyStatement) string {
	return payouts.FormatBusinessDate(stmt.WeekEnd.AddDate(0, 0, -1), time.UTC)
}

// EnsurePayoutMethod registers (or refreshes) a payee's destination with the rail
// and upserts the local PayoutMethod row.
//
// Safe to call repeatedly: the rail-side beneficiary id is deterministic
// (payouts.BeneficiaryIDFor), so re-registration resolves to the same
// beneficiary instead of minting a second whose verification state could diverge
// from the first.
func EnsurePayoutMethod(ctx context.Context, db *gorm.DB, ref payouts.PayeeRef, mode string) (*payouts.PayoutMethod, error) {
	instrument, name, err := ResolvePayeeInstrument(ctx, ref.ID, ref.Type)
	if err != nil {
		return nil, err
	}
	return EnsurePayoutMethodWith(ctx, db, ref, mode, instrument, name)
}

// EnsurePayoutMethodWith is EnsurePayoutMethod for a caller that ALREADY holds
// the instrument, and must not read it back from Secret Manager.
//
// That caller is the payout-details handler. It writes the chef's bank details to
// Secret Manager in a fire-and-forget goroutine, so a read-back here would race
// the write and register a beneficiary from stale — or entirely absent — details.
// It has the values in hand; taking them directly removes both the race and a
// pointless round-trip.
//
// The instrument is used and dropped. Nothing here persists it.
func EnsurePayoutMethodWith(
	ctx context.Context, db *gorm.DB, ref payouts.PayeeRef, mode string,
	instrument payouts.Instrument, name string,
) (*payouts.PayoutMethod, error) {
	if !instrument.Valid() {
		return nil, fmt.Errorf("payouts: payee %s has no usable payout destination", ref.ID)
	}
	if name == "" {
		// The rail matches the name against the account and rejects a mismatch,
		// so registering with a guessed name would produce an INVALID
		// beneficiary that looks like a bank problem. Fail here with something
		// an admin can act on.
		return nil, fmt.Errorf("payouts: payee %s has payout details but no account holder name on file", ref.ID)
	}

	rail := NewCashfreeRail(mode)
	// Instrument-scoped: changed bank details register a fresh beneficiary
	// instead of silently resolving to the old destination (Cashfree has no
	// beneficiary update). Same details still yield the same id on retry.
	beneficiaryID := payouts.BeneficiaryIDForInstrument(ref, instrument)

	res, regErr := rail.EnsureBeneficiary(ctx, payouts.BeneficiaryRequest{
		BeneficiaryID: beneficiaryID,
		Name:          name,
		Instrument:    instrument,
	})
	// A rejection still gets persisted — an INVALID method with the rail's own
	// wording is what tells an admin the IFSC is wrong, and losing it would
	// leave the chef silently unpayable with nothing on screen to explain why.
	if regErr != nil && !errors.Is(regErr, payouts.ErrBeneficiaryRejected) {
		return nil, regErr
	}
	// The computed id is a request. When the account was already registered the
	// rail answers with the id it holds, and a transfer addressed to anything else
	// 404s (#1151).
	if res.BeneficiaryID != "" {
		beneficiaryID = res.BeneficiaryID
	}

	method := payouts.PayoutMethod{
		TenantID:          PayoutTenantID,
		PayeeType:         ref.Type,
		PayeeID:           ref.ID,
		Kind:              instrument.Kind,
		Status:            res.Status,
		Primary:           true,
		DisplayHint:       DisplayHintFor(instrument),
		BeneficiaryName:   name,
		Rail:              rail.Name(),
		RailBeneficiaryID: beneficiaryID,
		RailStatusDetail:  res.Detail,
	}
	if res.Status == payouts.MethodVerified {
		now := time.Now()
		method.VerifiedAt = &now
	}

	// Upsert on (tenant, payee, rail): one registration per payee per rail.
	var existing payouts.PayoutMethod
	err := db.Where("tenant_id = ? AND payee_type = ? AND payee_id = ? AND rail = ?",
		PayoutTenantID, ref.Type, ref.ID, rail.Name()).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		if cErr := db.Create(&method).Error; cErr != nil {
			return nil, fmt.Errorf("payouts: persist method for %s: %w", ref, cErr)
		}
	case err != nil:
		return nil, fmt.Errorf("payouts: load method for %s: %w", ref, err)
	default:
		method.ID = existing.ID
		if uErr := db.Model(&existing).Updates(map[string]any{
			"kind":                method.Kind,
			"status":              method.Status,
			"display_hint":        method.DisplayHint,
			"beneficiary_name":    method.BeneficiaryName,
			"rail_beneficiary_id": method.RailBeneficiaryID,
			"rail_status_detail":  method.RailStatusDetail,
			"verified_at":         method.VerifiedAt,
		}).Error; uErr != nil {
			return nil, fmt.Errorf("payouts: update method for %s: %w", ref, uErr)
		}
	}

	if regErr != nil {
		return &method, regErr // rejection, now recorded
	}
	return &method, nil
}

// PrepareStatementBatch creates (or returns) the batch for a statement, without
// moving any money.
//
// The batch lands in pending_approval or approved per payouts.DecideAutoApprove:
// the chef's tri-state (falling back to the global auto-disburse flag) grants
// candidacy, then the guardrails run — first disbursement to a destination is
// always manual, and amounts above the cap setting queue for review. The
// returned decision carries the hold reasons for the caller's audit trail; it
// is zero-valued when an existing batch was reused. Splitting preparation from
// execution is what lets an admin see, and withhold, a payout before it goes —
// and what makes the approved → executing transition a single reviewable step.
func PrepareStatementBatch(ctx context.Context, db *gorm.DB, stmt *models.WeeklyStatement, mode string) (*payouts.Batch, payouts.AutoApproveDecision, error) {
	var none payouts.AutoApproveDecision
	if stmt.Status != models.PayoutPending {
		return nil, none, fmt.Errorf("payouts: statement %s is %s, not pending", stmt.ID, stmt.Status)
	}
	amount := payouts.FromMinor(int64(ToPaise(stmt.NetPayout)), payouts.CurrencyINR)
	if !amount.IsPositive() {
		return nil, none, fmt.Errorf("payouts: statement %s has nothing to pay (%s)", stmt.ID, amount)
	}

	ref := payouts.PayeeRef{Type: payouts.PayeeChef, ID: stmt.ChefID}
	businessDate := statementBusinessDate(stmt)
	key := payouts.IdempotencyKeyFor(PayoutTenantID, ref, businessDate)

	// Reuse an existing batch rather than creating a second. The unique index
	// would reject the insert anyway; returning the existing one means a retried
	// prepare is idempotent instead of an error an operator has to interpret.
	var existing payouts.Batch
	err := db.Where("idempotency_key = ?", key).First(&existing).Error
	if err == nil {
		return &existing, none, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, none, fmt.Errorf("payouts: load batch for statement %s: %w", stmt.ID, err)
	}

	method, mErr := EnsurePayoutMethod(ctx, db, ref, mode)
	if mErr != nil {
		return nil, none, mErr
	}
	if !method.Payable() {
		return nil, none, fmt.Errorf("payouts: chef %s has no verified payout destination (%s)",
			stmt.ChefID, method.Status)
	}

	// Candidacy: the chef's tri-state over the global flag. A missing chef row
	// fails closed to manual, like every other read on this path.
	autoEnabled := false
	var chef models.ChefProfile
	if cErr := db.First(&chef, "id = ?", stmt.ChefID).Error; cErr == nil {
		autoEnabled = ChefAutoDisburseEnabled(db, &chef)
	}

	// First disbursement to a destination always gets human eyes: only a batch
	// already PAID against this exact method counts as precedent. A count error
	// reads as "no precedent" — closed, not open.
	var paidBefore int64
	if autoEnabled {
		if cErr := db.Model(&payouts.Batch{}).
			Where("tenant_id = ? AND payee_type = ? AND payee_id = ? AND method_id = ? AND state = ?",
				PayoutTenantID, ref.Type, ref.ID, method.ID, payouts.BatchPaid).
			Count(&paidBefore).Error; cErr != nil {
			log.Printf("payout-disburse: precedent lookup failed (%v) — requiring approval", cErr)
			paidBefore = 0
		}
	}

	capMinor, capUnreadable := PayoutAutoDisburseCap(db)
	decision := payouts.DecideAutoApprove(payouts.AutoApproveInput{
		AutomationEnabled:     autoEnabled,
		DestinationPaidBefore: paidBefore > 0,
		AmountMinor:           amount.Minor,
		AutoCapMinor:          capMinor,
		CapUnreadable:         capUnreadable,
	})
	state := payouts.BatchPendingApproval
	if decision.Approve {
		state = payouts.BatchApproved
	}

	batch := payouts.Batch{
		TenantID:       PayoutTenantID,
		PayeeType:      ref.Type,
		PayeeID:        ref.ID,
		BusinessDate:   businessDate,
		State:          state,
		AmountMinor:    amount.Minor,
		Currency:       amount.Currency,
		MethodID:       &method.ID,
		Provider:       CashfreePayoutRailName,
		IdempotencyKey: key,
	}
	if err := db.Create(&batch).Error; err != nil {
		// A concurrent prepare won the unique index — load and return theirs.
		var raced payouts.Batch
		if db.Where("idempotency_key = ?", key).First(&raced).Error == nil {
			return &raced, none, nil
		}
		return nil, none, fmt.Errorf("payouts: create batch for statement %s: %w", stmt.ID, err)
	}
	return &batch, decision, nil
}

// ExecuteBatch sends an approved batch to the rail.
//
// Read the ordering note at the top of this file before changing anything here.
// The approved → executing write is committed FIRST, guarded so exactly one
// caller can perform it; only that caller then talks to the rail.
func ExecuteBatch(ctx context.Context, db *gorm.DB, batchID uuid.UUID, mode string) (*payouts.Batch, error) {
	var batch payouts.Batch
	if err := db.First(&batch, "id = ?", batchID).Error; err != nil {
		return nil, fmt.Errorf("payouts: load batch %s: %w", batchID, err)
	}
	if batch.State == payouts.BatchExecuting {
		// Already in flight. Resolve rather than re-send.
		return ResolveExecutingBatch(ctx, db, &batch, mode)
	}
	if _, err := batch.State.Transition(payouts.BatchExecuting); err != nil {
		return nil, err
	}

	// THE GUARD. Exactly one concurrent caller flips approved → executing, and
	// only that one proceeds to the rail. Without the WHERE state = 'approved'
	// two callers could both read `approved` and both disburse.
	res := db.Model(&payouts.Batch{}).
		Where("id = ? AND state = ?", batch.ID, payouts.BatchApproved).
		Updates(map[string]any{"state": payouts.BatchExecuting})
	if res.Error != nil {
		return nil, fmt.Errorf("payouts: claim batch %s: %w", batch.ID, res.Error)
	}
	if res.RowsAffected == 0 {
		// Someone else claimed it. Return current truth, never re-send.
		_ = db.First(&batch, "id = ?", batch.ID).Error
		return &batch, nil
	}
	batch.State = payouts.BatchExecuting

	var method payouts.PayoutMethod
	if batch.MethodID == nil {
		return nil, fmt.Errorf("payouts: batch %s has no payout method", batch.ID)
	}
	if err := db.First(&method, "id = ?", *batch.MethodID).Error; err != nil {
		return nil, fmt.Errorf("payouts: load method for batch %s: %w", batch.ID, err)
	}

	rail := NewCashfreeRail(mode)
	result, err := rail.Disburse(ctx, payouts.DisburseRequest{
		IdempotencyKey: batch.IdempotencyKey,
		BeneficiaryID:  method.RailBeneficiaryID,
		Amount:         batch.Amount(),
		Remarks:        "Fe3dr weekly payout",
		Kind:           method.Kind,
	})
	if err != nil {
		if errors.Is(err, payouts.ErrRailAmbiguous) {
			// Money may or may not have moved. The batch stays `executing` and
			// the poll resolves it. Explicitly NOT an error state — treating an
			// ambiguous send as failed is how a paid batch gets re-sent.
			log.Printf("payout-disburse: batch %s ambiguous (%v) — left executing for the poll to resolve", batch.ID, err)
			return &batch, nil
		}
		return nil, fmt.Errorf("payouts: disburse batch %s: %w", batch.ID, err)
	}
	return applyRailResult(db, &batch, result)
}

// ResolveExecutingBatch asks the rail what happened to a batch already in flight.
//
// This is the answer to an ambiguous send, and — with webhooks deliberately not
// in use — the ONLY way a batch ever reaches a terminal state. Cashfree transfers
// are asynchronous: a create returns RECEIVED or PENDING and nothing more, so
// without this sweep every batch would sit executing forever and no statement
// would ever be marked paid.
func ResolveExecutingBatch(ctx context.Context, db *gorm.DB, batch *payouts.Batch, mode string) (*payouts.Batch, error) {
	rail := NewCashfreeRail(mode)
	result, err := rail.GetPayoutByReference(ctx, batch.IdempotencyKey)
	if err != nil {
		if errors.Is(err, payouts.ErrRailNotFound) {
			// The rail has no record — the ONLY safe proof money did not move.
			// Fail the batch so it can be rebuilt and re-sent deliberately.
			log.Printf("payout-disburse: batch %s unknown to rail — marking failed (safe to re-attempt)", batch.ID)
			return applyRailResult(db, batch, payouts.DisburseResult{
				Status:        payouts.RailFailed,
				FailureCode:   "not_found_at_rail",
				FailureDetail: "the rail has no record of this transfer",
			})
		}
		return nil, err
	}
	return applyRailResult(db, batch, result)
}

// applyRailResult writes a rail outcome onto the batch and, when terminal,
// settles the statement behind it.
func applyRailResult(db *gorm.DB, batch *payouts.Batch, result payouts.DisburseResult) (*payouts.Batch, error) {
	next, ok := result.Status.BatchState()
	if !ok {
		return nil, fmt.Errorf("payouts: rail returned unmappable status %q for batch %s", result.Status, batch.ID)
	}

	updates := map[string]any{
		"provider_ref": result.Reference,
		"provider_utr": result.UTR,
	}
	if next != batch.State {
		if _, err := batch.State.Transition(next); err != nil {
			// An illegal transition means our view and the rail's have diverged.
			// Record it loudly rather than forcing the state — forcing it is how
			// a reversed payout silently becomes "paid".
			return nil, fmt.Errorf("payouts: batch %s: %w", batch.ID, err)
		}
		updates["state"] = next
	}
	if next == payouts.BatchFailed || next == payouts.BatchReversed {
		updates["failure_code"] = result.FailureCode
		updates["failure_detail"] = result.FailureDetail
	}

	if err := db.Model(&payouts.Batch{}).Where("id = ?", batch.ID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("payouts: persist batch %s outcome: %w", batch.ID, err)
	}
	batch.State = next
	batch.ProviderRef = result.Reference
	batch.ProviderUTR = result.UTR

	if next == payouts.BatchPaid {
		if err := markStatementPaidForBatch(db, batch); err != nil {
			// The money HAS moved. Never fail the caller on a bookkeeping write
			// — surface it and let reconciliation catch the drift.
			log.Printf("payout-disburse: batch %s paid but statement stamp failed: %v", batch.ID, err)
			CaptureBackgroundError(err)
		}
	}
	return batch, nil
}

// markStatementPaidForBatch stamps the WeeklyStatement behind a paid batch.
//
// The statement is identified EXACTLY, by reversing statementBusinessDate: the
// batch's business date is WeekEnd minus one day, so WeekEnd is the day after
// it. A date-range search would be wrong rather than merely loose — a chef with
// two pending statements (a missed week, or a backfill) would have whichever one
// the range happened to catch marked paid, and the other left owing with no
// signal. Paying one week and stamping another is not recoverable by inspection.
//
// The UTR is recorded as PayoutRef because that is the reference a chef can
// quote to their own bank — the rail's internal id means nothing to them, and
// this field is rendered straight into the chef's statement screen.
func markStatementPaidForBatch(db *gorm.DB, batch *payouts.Batch) error {
	businessDate, err := time.Parse("2006-01-02", batch.BusinessDate)
	if err != nil {
		return fmt.Errorf("payouts: batch %s has an unparseable business date %q: %w",
			batch.ID, batch.BusinessDate, err)
	}
	weekEnd := businessDate.AddDate(0, 0, 1)

	ref := batch.ProviderUTR
	if ref == "" {
		ref = batch.ProviderRef
	}
	now := time.Now()

	// Guarded on status so a replayed resolution cannot re-stamp PaidAt and move
	// a statement's recorded payment date. Matching on the DATE part of week_end
	// because it is stored as a UTC timestamp at midnight.
	res := db.Model(&models.WeeklyStatement{}).
		Where("chef_id = ? AND status = ? AND DATE(week_end) = DATE(?)",
			batch.PayeeID, models.PayoutPending, weekEnd).
		Updates(map[string]any{
			"status":     models.PayoutPaid,
			"paid_at":    &now,
			"payout_ref": ref,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// Not necessarily wrong — a re-resolved batch finds the statement
		// already paid. Worth a line either way: money moved and nothing was
		// stamped, which is exactly the drift reconciliation looks for.
		log.Printf("payout-disburse: batch %s paid but no pending statement matched chef=%s weekEnd=%s",
			batch.ID, batch.PayeeID, weekEnd.Format("2006-01-02"))
	}
	return nil
}

// PollExecutingPayouts resolves every in-flight batch against the rail.
//
// This REPLACES the webhook. It is not a backstop here — it is the only path to
// a terminal state, so it must run on a schedule for payouts to ever settle.
// Returns how many batches reached a terminal state.
func PollExecutingPayouts(ctx context.Context, db *gorm.DB, mode string) (int, error) {
	var batches []payouts.Batch
	if err := db.Where("state = ? AND provider = ?", payouts.BatchExecuting, CashfreePayoutRailName).
		Order("updated_at ASC").Limit(200).Find(&batches).Error; err != nil {
		return 0, fmt.Errorf("payouts: load executing batches: %w", err)
	}

	settled := 0
	for i := range batches {
		b := &batches[i]
		out, err := ResolveExecutingBatch(ctx, db, b, mode)
		if err != nil {
			// One batch's failure must not stop the sweep — the next batch is a
			// different chef's money.
			log.Printf("payout-poll: batch %s unresolved: %v", b.ID, err)
			continue
		}
		if out.State.IsTerminal() {
			settled++
		}
	}
	return settled, nil
}
