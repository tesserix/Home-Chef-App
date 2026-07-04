package models

// staff_role_gate_test.go — security audit #4: the invitation gate and the
// accept-time User.Role mapping must share ONE predicate so a lower-trust
// inviter (a delivery FleetManager) can never mint a platform admin, and the two
// can never drift apart.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStaffRoleGrantsPlatformAdmin(t *testing.T) {
	// Everything except the two delivery roles maps to RoleAdmin on accept.
	adminTier := []StaffRole{StaffRoleSuperAdmin, StaffRoleAdmin, StaffRoleSupport}
	deliveryTier := []StaffRole{StaffRoleFleetManager, StaffRoleDeliveryOps}

	for _, r := range adminTier {
		require.True(t, StaffRoleGrantsPlatformAdmin(r), "%s must be admin-tier", r)
	}
	for _, r := range deliveryTier {
		require.False(t, StaffRoleGrantsPlatformAdmin(r), "%s must be delivery-tier", r)
	}
}

func TestStaffRoleIsPlatformAdmin_OnlyAdmins(t *testing.T) {
	require.True(t, StaffRoleIsPlatformAdmin(StaffRoleSuperAdmin))
	require.True(t, StaffRoleIsPlatformAdmin(StaffRoleAdmin))
	// A FleetManager holds SPInviteStaff but is NOT a platform admin — this is the
	// gate that blocks the escalation (a FleetManager inviting staffRole=admin).
	require.False(t, StaffRoleIsPlatformAdmin(StaffRoleFleetManager))
	require.False(t, StaffRoleIsPlatformAdmin(StaffRoleDeliveryOps))
	require.False(t, StaffRoleIsPlatformAdmin(StaffRoleSupport))
}

// The exact escalation the gate must reject: a non-admin inviter inviting an
// admin-tier role.
func TestEscalationRejected_FleetManagerInvitingAdmin(t *testing.T) {
	inviter := StaffRoleFleetManager
	invited := StaffRoleAdmin
	// The gate rejects when the invited role grants admin AND the inviter isn't one.
	rejected := StaffRoleGrantsPlatformAdmin(invited) && !StaffRoleIsPlatformAdmin(inviter)
	require.True(t, rejected, "a FleetManager inviting an admin must be rejected")

	// A FleetManager inviting a delivery role is allowed.
	allowedInvited := StaffRoleDeliveryOps
	rejected2 := StaffRoleGrantsPlatformAdmin(allowedInvited) && !StaffRoleIsPlatformAdmin(inviter)
	require.False(t, rejected2, "a FleetManager inviting delivery_ops is allowed")

	// An admin inviting an admin is allowed.
	rejected3 := StaffRoleGrantsPlatformAdmin(StaffRoleAdmin) && !StaffRoleIsPlatformAdmin(StaffRoleAdmin)
	require.False(t, rejected3, "an admin inviting an admin is allowed")
}
