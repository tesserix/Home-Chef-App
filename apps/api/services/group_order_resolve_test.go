package services

// group_order_resolve_test.go — #594 slice B. The admin confirms fault on a delivery-
// FAILED group (frozen by slice A) and the hybrid money policy executes: customer-fault →
// chef paid + group→delivered; platform/chef-fault → refund all participants + reverse the
// chef's held direct transfer + group→cancelled. State transitions tested deterministically
// (flags off = money-safe; participant wallet refunds are NOT flag-gated).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func seedGroupParticipant(t *testing.T, db *gorm.DB, groupID uuid.UUID, share float64) models.GroupOrderParticipant {
	t.Helper()
	pid, uid := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO group_order_participants (id, group_order_id, user_id, role, share_amount, payment_status)
		VALUES (?,?,?,?,?,?)`, pid.String(), groupID.String(), uid.String(), "guest", share, string(models.GroupPayCompleted)).Error)
	return models.GroupOrderParticipant{ID: pid, GroupOrderID: groupID, UserID: uid, ShareAmount: share, PaymentStatus: models.GroupPayCompleted}
}

func loadGroupOrderStatus(t *testing.T, db *gorm.DB, orderID uuid.UUID) models.OrderStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM orders WHERE id = ?`, orderID.String()).Scan(&s).Error)
	return models.OrderStatus(s)
}

func partRefunded(t *testing.T, db *gorm.DB, pid uuid.UUID) bool {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM group_order_participants WHERE id = ? AND refund_txn_id IS NOT NULL`, pid.String()).Scan(&n).Error)
	return n == 1
}

// loadGroup returns the group struct with its participants, as the handler would pass it.
func loadGroup(t *testing.T, db *gorm.DB, gid uuid.UUID, parts []models.GroupOrderParticipant) *models.GroupOrder {
	t.Helper()
	var g models.GroupOrder
	require.NoError(t, db.Raw(`SELECT id, order_id, status, payout_hold_status FROM group_orders WHERE id = ?`, gid.String()).Scan(&g).Error)
	g.Participants = parts
	return &g
}

func TestResolveGroupOrderFailure_CustomerFault_PaysChef(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	orderID := uuid.New()
	gid := seedCrossGroup(t, db, models.PayoutHoldDisputed, &orderID)
	setGroupStatus(t, db, gid, models.GroupOrderFailed)
	p := seedGroupParticipant(t, db, gid, 100)
	g := loadGroup(t, db, gid, []models.GroupOrderParticipant{p})

	require.NoError(t, ResolveGroupOrderFailure(db, g, models.FaultCustomer, uuid.New()))

	require.Equal(t, models.GroupOrderDelivered, loadGroupStatus(t, db, gid), "chef paid → paid-terminal")
	require.Equal(t, models.PayoutHoldReleaseEligible, loadGroupHold(t, db, gid), "hold enters pay queue")
	require.False(t, partRefunded(t, db, p.ID), "no participant refund on customer fault")
	require.Equal(t, 0.0, walletBalance(t, db, p.UserID))
}

func TestResolveGroupOrderFailure_PlatformFault_RefundsAndReverses(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivering", nil)
	gid := seedCrossGroup(t, db, models.PayoutHoldDisputed, &orderID)
	setGroupStatus(t, db, gid, models.GroupOrderFailed)
	p1 := seedGroupParticipant(t, db, gid, 120)
	p2 := seedGroupParticipant(t, db, gid, 80)
	g := loadGroup(t, db, gid, []models.GroupOrderParticipant{p1, p2})

	require.NoError(t, ResolveGroupOrderFailure(db, g, models.FaultChef, uuid.New()))

	require.Equal(t, models.GroupOrderCancelled, loadGroupStatus(t, db, gid))
	require.True(t, partRefunded(t, db, p1.ID))
	require.True(t, partRefunded(t, db, p2.ID))
	require.Equal(t, 120.0, walletBalance(t, db, p1.UserID), "participant 1 refunded their share")
	require.Equal(t, 80.0, walletBalance(t, db, p2.UserID))
	// The chef's hold is driven terminal-non-payable (disputed → reversed via
	// ReverseGroupHoldForCancel; flags-off = no transfer moved, hold state still flips).
	require.Equal(t, models.PayoutHoldReversed, loadGroupHold(t, db, gid))
	require.Equal(t, models.OrderStatusCancelled, loadGroupOrderStatus(t, db, orderID), "consolidated order cancelled")
}

func TestResolveGroupOrderFailure_Idempotent(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivering", nil)
	gid := seedCrossGroup(t, db, models.PayoutHoldDisputed, &orderID)
	setGroupStatus(t, db, gid, models.GroupOrderFailed)
	p := seedGroupParticipant(t, db, gid, 100)
	g := loadGroup(t, db, gid, []models.GroupOrderParticipant{p})

	require.NoError(t, ResolveGroupOrderFailure(db, g, models.FaultPlatform, uuid.New()))
	err := ResolveGroupOrderFailure(db, g, models.FaultCustomer, uuid.New())
	require.ErrorIs(t, err, ErrIssueAlreadyHandled)
	require.Equal(t, models.GroupOrderCancelled, loadGroupStatus(t, db, gid), "first resolution wins")
	require.Equal(t, 100.0, walletBalance(t, db, p.UserID), "no double refund")
}

func TestResolveGroupOrderFailure_NotFailed(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	gid := seedCrossGroup(t, db, models.PayoutHoldAwaitingConfirmation, nil) // status='delivered'
	g := loadGroup(t, db, gid, nil)
	require.ErrorIs(t, ResolveGroupOrderFailure(db, g, models.FaultPlatform, uuid.New()), ErrNotDeliveryFailure)
}

func TestResolveGroupOrderFailure_Ambiguous(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	gid := seedCrossGroup(t, db, models.PayoutHoldDisputed, nil)
	setGroupStatus(t, db, gid, models.GroupOrderFailed)
	g := loadGroup(t, db, gid, nil)
	require.ErrorIs(t, ResolveGroupOrderFailure(db, g, models.FaultAmbiguous, uuid.New()), ErrAmbiguousFault)
	require.Equal(t, models.GroupOrderFailed, loadGroupStatus(t, db, gid), "still frozen")
	require.Equal(t, models.PayoutHoldDisputed, loadGroupHold(t, db, gid))
}
