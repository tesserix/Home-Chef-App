package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/homechef/api/models"
)

func TestMaskPartnerDetailPII(t *testing.T) {
	full := models.DeliveryPartnerDetailResponse{
		Email:          "driver@example.com",
		Phone:          "+919876543210",
		LicenseNumber:  "DL0420110149646",
		VehicleNumber:  "KA01AB1234",
		EmergencyPhone: "+919812345678",
		Name:           "Ravi Kumar", // non-PII identity — must stay intact
	}

	masked := maskPartnerDetailPII(full)

	// Sensitive fields are redacted to the shared mask shape.
	assert.Equal(t, maskEmail("driver@example.com"), masked.Email)
	assert.Equal(t, "****3210", masked.Phone)
	assert.Equal(t, "****5678", masked.EmergencyPhone)
	assert.Equal(t, maskID("DL0420110149646"), masked.LicenseNumber)
	assert.Equal(t, maskID("KA01AB1234"), masked.VehicleNumber)

	// Name is not masked (already shown in the list view).
	assert.Equal(t, "Ravi Kumar", masked.Name)

	// The input is not mutated.
	assert.Equal(t, "driver@example.com", full.Email)
}

func TestMaskPartnerDetailPII_EmptyValues(t *testing.T) {
	masked := maskPartnerDetailPII(models.DeliveryPartnerDetailResponse{})
	assert.Equal(t, "", masked.Email)
	assert.Equal(t, "", masked.LicenseNumber)
	assert.Equal(t, "", masked.VehicleNumber)
	// A short/empty phone masks to the fixed placeholder, never leaking length.
	assert.Equal(t, "****", masked.Phone)
}
