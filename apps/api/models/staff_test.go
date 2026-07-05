package models

import "testing"

func staffTestContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// TestSPManagePayouts_GrantedToSuperAdminOnly locks in the least-privilege
// policy for the money-release surface (#515 / #461 step 2). Releasing,
// withholding, or reversing an escrow payout hold is a finance-sensitive
// action, so only the super_admin role carries SPManagePayouts by default.
// Regular admin/support staff must NOT be able to move payout money even
// though they can reach the rest of the admin panel.
func TestSPManagePayouts_GrantedToSuperAdminOnly(t *testing.T) {
	su := &StaffMember{StaffRole: StaffRoleSuperAdmin}
	if !su.HasPermission(SPManagePayouts) {
		t.Fatalf("super_admin must have SPManagePayouts")
	}
	for _, role := range []StaffRole{
		StaffRoleAdmin, StaffRoleFleetManager, StaffRoleDeliveryOps, StaffRoleSupport,
	} {
		s := &StaffMember{StaffRole: role}
		if s.HasPermission(SPManagePayouts) {
			t.Fatalf("role %q must NOT have SPManagePayouts (least privilege for money release)", role)
		}
	}
}

func TestLoadSuperAdminEmails_DefaultWhenUnset(t *testing.T) {
	t.Setenv("SUPER_ADMIN_EMAILS", "")
	got := loadSuperAdminEmails()
	if len(got) != len(defaultSuperAdminEmails) {
		t.Fatalf("expected %d default emails, got %d: %v", len(defaultSuperAdminEmails), len(got), got)
	}
	if !staffTestContains(got, "samyak.rout@gmail.com") {
		t.Fatalf("default list missing built-in email: %v", got)
	}
}

func TestLoadSuperAdminEmails_FromEnvOverridesDefault(t *testing.T) {
	t.Setenv("SUPER_ADMIN_EMAILS", " Alice@Example.com , bob@example.com ,")
	got := loadSuperAdminEmails()
	want := []string{"alice@example.com", "bob@example.com"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
	// Env override replaces the hardcoded defaults entirely.
	if staffTestContains(got, "samyak.rout@gmail.com") {
		t.Fatalf("env override should replace defaults, got %v", got)
	}
}

func TestIsSuperAdminEmail_CaseInsensitiveAndTrimmed(t *testing.T) {
	orig := SuperAdminEmails
	t.Cleanup(func() { SuperAdminEmails = orig })
	SuperAdminEmails = normalizeEmails([]string{"Admin@Example.com"})

	if !IsSuperAdminEmail("  ADMIN@example.COM ") {
		t.Fatal("expected case-insensitive + trimmed match against the allowlist")
	}
	if IsSuperAdminEmail("nobody@example.com") {
		t.Fatal("unexpected match for a non-super-admin email")
	}
}
