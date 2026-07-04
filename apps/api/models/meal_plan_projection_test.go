package models

// meal_plan_projection_test.go — security audit H1/M1: a chef must not receive
// the customer's email/phone, and a customer must not receive the home-chef's
// address/lat-long/FSSAI/GSTIN. These pin the role-scoped projections + prove the
// raw relations never serialize (json:"-").

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func sampleMealPlan() MealPlan {
	return MealPlan{
		Customer: &User{
			FirstName: "Asha", LastName: "R",
			Email: "asha@example.com", Phone: "+919876543210",
		},
		Chef: &ChefProfile{
			BusinessName: "Asha's Kitchen", ProfileImage: "img.jpg",
			AddressLine1: "42 Secret Home Lane", PostalCode: "560001",
			Latitude: 12.9716, Longitude: 77.5946,
			FSSAILicenseNumber: "FSSAI-123", GSTIN: "GST-123",
		},
	}
}

func TestProjectForChef_HidesCustomerContact(t *testing.T) {
	p := sampleMealPlan()
	p.ProjectForChef()
	b, err := json.Marshal(p)
	require.NoError(t, err)
	s := string(b)

	require.Contains(t, s, "Asha", "chef sees the customer's name")
	require.NotContains(t, s, "asha@example.com", "chef must NOT see the customer's email")
	require.NotContains(t, s, "+919876543210", "chef must NOT see the customer's phone")
	// The chef's own object isn't attached in the chef view.
	require.Nil(t, p.ChefView)
}

func TestProjectForCustomer_HidesChefLocationAndLicences(t *testing.T) {
	p := sampleMealPlan()
	p.ProjectForCustomer()
	b, err := json.Marshal(p)
	require.NoError(t, err)
	s := string(b)

	require.Contains(t, s, "Asha's Kitchen", "customer sees the chef's business name")
	for _, secret := range []string{"Secret Home Lane", "560001", "12.9716", "77.5946", "FSSAI-123", "GST-123"} {
		require.False(t, strings.Contains(s, secret), "customer must NOT see %q", secret)
	}
	require.Nil(t, p.CustomerView)
}

func TestProjectForAdmin_KeepsFullCustomerContact(t *testing.T) {
	p := sampleMealPlan()
	p.ProjectForAdmin()
	b, _ := json.Marshal(p)
	s := string(b)
	require.Contains(t, s, "asha@example.com", "admin support view keeps contact")
	require.Contains(t, s, "+919876543210")
}

// The raw relations must never auto-serialize even without a projection call.
func TestMealPlan_RawRelationsNeverSerialize(t *testing.T) {
	p := sampleMealPlan()
	b, _ := json.Marshal(p)
	s := string(b)
	require.NotContains(t, s, "asha@example.com")
	require.NotContains(t, s, "Secret Home Lane")
	require.NotContains(t, s, "FSSAI-123")
}
