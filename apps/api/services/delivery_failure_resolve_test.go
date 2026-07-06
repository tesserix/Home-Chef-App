package services

// delivery_failure_resolve_test.go — #393 slice 3. The admin-confirm resolver executes
// the owner's money policy for a frozen `delivery_failed` issue: customer-fault → no
// refund + release the vendor hold (fee retained); platform/chef-fault → full customer
// refund + reverse the vendor hold; ambiguous → the admin must confirm a concrete fault.
// This closes the slice-1 → slice-3 loop end to end.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func seedDeliveryFailedIssue(t *testing.T, db *gorm.DB, orderID, customerID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO order_issues (id, order_id, chef_id, customer_id, reason, description, status)
		VALUES (?,?,?,?,?,?,?)`,
		id.String(), orderID.String(), uuid.NewString(), customerID.String(),
		string(models.IssueDeliveryFailed),
		"delivery failed: reason=customer_unavailable suggested_fault=customer", string(models.IssuePending)).Error)
	return id
}

func loadIssueRow(t *testing.T, db *gorm.DB, id uuid.UUID) *models.OrderIssue {
	t.Helper()
	var i models.OrderIssue
	require.NoError(t, db.Raw(`SELECT id, order_id, customer_id, chef_id, reason, description, status
		FROM order_issues WHERE id = ?`, id.String()).Scan(&i).Error)
	return &i
}

func loadIssueStatus(t *testing.T, db *gorm.DB, id uuid.UUID) models.IssueStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM order_issues WHERE id = ?`, id.String()).Scan(&s).Error)
	return models.IssueStatus(s)
}

func TestResolveDeliveryFailure_CustomerFault_ReleasesNoRefund(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, customerID := seedCrossOrder(t, db, models.PayoutHoldDisputed, "delivering", nil)
	issueID := seedDeliveryFailedIssue(t, db, orderID, customerID)

	require.NoError(t, ResolveDeliveryFailure(db, loadIssueRow(t, db, issueID), models.FaultCustomer, uuid.New()))

	require.Equal(t, models.IssueRejected, loadIssueStatus(t, db, issueID))
	require.Equal(t, models.PayoutHoldReleaseEligible, loadOrderHold(t, db, orderID), "customer-fault → vendor paid (hold released)")
	require.Equal(t, 0.0, walletBalance(t, db, customerID), "customer-fault → NO refund")
	require.Equal(t, 0.0, loadOrderRefundAmount(t, db, orderID))
}

func TestResolveDeliveryFailure_PlatformFault_FullRefundReversesHold(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, customerID := seedCrossOrder(t, db, models.PayoutHoldDisputed, "delivering", nil)
	issueID := seedDeliveryFailedIssue(t, db, orderID, customerID)

	require.NoError(t, ResolveDeliveryFailure(db, loadIssueRow(t, db, issueID), models.FaultPlatform, uuid.New()))

	require.Equal(t, models.IssueResolved, loadIssueStatus(t, db, issueID))
	// The hold was frozen from `none` (chef never paid), so the cross-guard WITHHOLDS the
	// payout (blocks it) rather than reversing a settled transfer — the correct
	// never-released outcome. Escrow: captured = full refund + withheld chef portion.
	require.Equal(t, models.PayoutHoldWithheld, loadOrderHold(t, db, orderID), "platform-fault → chef payout blocked")
	require.Equal(t, 250.0, walletBalance(t, db, customerID), "full order total refunded to customer")
	require.Equal(t, 250.0, loadOrderRefundAmount(t, db, orderID))
}

func TestResolveDeliveryFailure_ChefFault_FullRefundReversesHold(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, customerID := seedCrossOrder(t, db, models.PayoutHoldDisputed, "delivering", nil)
	issueID := seedDeliveryFailedIssue(t, db, orderID, customerID)

	require.NoError(t, ResolveDeliveryFailure(db, loadIssueRow(t, db, issueID), models.FaultChef, uuid.New()))

	require.Equal(t, models.IssueResolved, loadIssueStatus(t, db, issueID))
	require.Equal(t, models.PayoutHoldWithheld, loadOrderHold(t, db, orderID), "chef-fault → chef payout blocked")
	require.Equal(t, 250.0, walletBalance(t, db, customerID))
}

func TestResolveDeliveryFailure_Ambiguous_NoMoneyMoved(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, customerID := seedCrossOrder(t, db, models.PayoutHoldDisputed, "delivering", nil)
	issueID := seedDeliveryFailedIssue(t, db, orderID, customerID)

	err := ResolveDeliveryFailure(db, loadIssueRow(t, db, issueID), models.FaultAmbiguous, uuid.New())
	require.ErrorIs(t, err, ErrAmbiguousFault)
	// Untouched — admin must confirm a concrete fault first.
	require.Equal(t, models.IssuePending, loadIssueStatus(t, db, issueID))
	require.Equal(t, models.PayoutHoldDisputed, loadOrderHold(t, db, orderID))
	require.Equal(t, 0.0, walletBalance(t, db, customerID))
}

// Cross-path race: two admins load the same pending delivery_failed issue; A resolves
// customer-fault (issue→rejected, no refund), B races platform-fault on the stale issue.
// With #581 (claim-before-credit in RefundIssueToWallet), B's refund claims WHERE pending
// → 0 rows → NO wallet credit. The customer must NOT be double-refunded.
func TestResolveDeliveryFailure_CrossPathRace_NoDoubleRefund(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, customerID := seedCrossOrder(t, db, models.PayoutHoldDisputed, "delivering", nil)
	issueID := seedDeliveryFailedIssue(t, db, orderID, customerID)
	stale := loadIssueRow(t, db, issueID) // admin B's snapshot, taken while pending

	// Admin A: customer-fault → issue rejected, hold released, NO refund.
	require.NoError(t, ResolveDeliveryFailure(db, loadIssueRow(t, db, issueID), models.FaultCustomer, uuid.New()))
	require.Equal(t, models.IssueRejected, loadIssueStatus(t, db, issueID))

	// Admin B: platform-fault on the now-rejected issue → must be a no-op refund-wise.
	require.NoError(t, ResolveDeliveryFailure(db, stale, models.FaultPlatform, uuid.New()))
	require.Equal(t, 0.0, walletBalance(t, db, customerID), "cross-path race must NOT double-refund (#581)")
	require.Equal(t, 0.0, loadOrderRefundAmount(t, db, orderID))
	require.Equal(t, models.IssueRejected, loadIssueStatus(t, db, issueID), "the customer-fault resolution stands")
	// The losing platform-fault call must NOT touch the hold either — A's legitimate
	// release must stand, not be flipped to a terminal `withheld` with no refund behind it.
	require.Equal(t, models.PayoutHoldReleaseEligible, loadOrderHold(t, db, orderID),
		"a lost-claim refund must not withhold the chef payout the winner released")

	// The losing platform-fault call must not clobber the winner's audit trail.
	var desc string
	db.Raw(`SELECT description FROM order_issues WHERE id = ?`, issueID.String()).Scan(&desc)
	require.Contains(t, desc, "admin_confirmed_fault=customer")
	require.NotContains(t, desc, "admin_confirmed_fault=platform")
}

func TestResolveDeliveryFailure_RejectsNonDeliveryIssue(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, customerID := seedCrossOrder(t, db, models.PayoutHoldDisputed, "delivering", nil)
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO order_issues (id, order_id, chef_id, customer_id, reason, status)
		VALUES (?,?,?,?,?,?)`, id.String(), orderID.String(), uuid.NewString(), customerID.String(),
		string(models.IssueQualityIssue), string(models.IssuePending)).Error)

	err := ResolveDeliveryFailure(db, loadIssueRow(t, db, id), models.FaultPlatform, uuid.New())
	require.ErrorIs(t, err, ErrNotDeliveryFailure, "the fault resolver is only for delivery_failed issues")
}
