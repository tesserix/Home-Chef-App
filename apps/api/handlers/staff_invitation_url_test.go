package handlers

import (
	"testing"

	"github.com/homechef/api/models"
	"github.com/stretchr/testify/assert"
)

func TestStaffInvitationBaseURL_UsesCanonicalPortalForRole(t *testing.T) {
	t.Setenv("ADMIN_PORTAL_BASE_URL", "https://admin.staging.fe3dr.com/")
	t.Setenv("DELIVERY_PORTAL_BASE_URL", "https://delivery.staging.fe3dr.com/")

	assert.Equal(t, "https://delivery.staging.fe3dr.com", staffInvitationBaseURL(models.StaffRoleFleetManager))
	assert.Equal(t, "https://delivery.staging.fe3dr.com", staffInvitationBaseURL(models.StaffRoleDeliveryOps))
	assert.Equal(t, "https://admin.staging.fe3dr.com", staffInvitationBaseURL(models.StaffRoleSuperAdmin))
}
