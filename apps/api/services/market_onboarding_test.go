package services

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestKitchenCountryDefaultsToIndiaAndRejectsUnservedMarkets(t *testing.T) {
	cases := map[string]struct {
		want string
		ok   bool
	}{
		"":    {"IN", true},
		"in":  {"IN", true},
		" AU": {"AU", true},
		"nz":  {"NZ", true},
		"US":  {"", false},
		"XX":  {"", false},
	}
	for raw, tc := range cases {
		got, ok := NormalizeKitchenCountry(raw)
		require.Equal(t, tc.ok, ok, raw)
		require.Equal(t, tc.want, got, raw)
	}
}

func TestPhoneRulesForAustraliaAndNewZealand(t *testing.T) {
	require.True(t, IsValidPhone("AU", "412345678"))
	require.False(t, IsValidPhone("AU", "0412345678"), "store the national number without the trunk 0")
	require.False(t, IsValidPhone("AU", "212345678"), "landlines are not mobiles")
	require.True(t, IsValidPhone("NZ", "21234567"))
	require.True(t, IsValidPhone("NZ", "2123456789"))
	require.False(t, IsValidPhone("NZ", "912345678"))
	require.True(t, IsValidPhone("IN", "9876543210"))
	require.False(t, IsValidPhone("AU", "9876543210"), "an Indian number is not an Australian one")
}

func TestPostcodeRulesPerCountry(t *testing.T) {
	require.True(t, IsValidPostcode("IN", "560001"))
	require.False(t, IsValidPostcode("IN", "3000"))
	require.True(t, IsValidPostcode("AU", "3000"))
	require.False(t, IsValidPostcode("AU", "30000"))
	require.True(t, IsValidPostcode("NZ", "1010"))
	require.False(t, IsValidPostcode("NZ", "10a0"))
}

func TestFoodRegistrationIsTheCouncilCertificateOutsideIndia(t *testing.T) {
	require.Equal(t, models.DocFSSAILicense, FoodRegistrationDocType("IN"))
	require.Equal(t, models.DocFSSAILicense, FoodRegistrationDocType(""), "legacy chefs have no country")
	require.Equal(t, models.DocFoodSafetyCert, FoodRegistrationDocType("AU"))
	require.Equal(t, models.DocFoodSafetyCert, FoodRegistrationDocType("NZ"))
}

func TestRequiredChefDocsFollowTheKitchenCountry(t *testing.T) {
	require.Equal(t,
		[]models.DocumentType{models.DocIDProof, models.DocAddressProof, models.DocFSSAILicense},
		RequiredChefDocTypes("IN"))
	require.Equal(t,
		[]models.DocumentType{models.DocIDProof, models.DocAddressProof, models.DocFoodSafetyCert},
		RequiredChefDocTypes("AU"))
}

func TestMissingChefDocsUsesTheCountrysLicence(t *testing.T) {
	docs := []models.ChefDocument{{Type: models.DocIDProof}, {Type: models.DocFSSAILicense}}
	require.Equal(t, []string{"address_proof"}, MissingChefDocs("IN", docs))
	require.Equal(t, []string{"address_proof", "food_safety_cert"}, MissingChefDocs("NZ", docs))
	require.Empty(t, MissingChefDocs("AU", []models.ChefDocument{
		{Type: models.DocIDProof}, {Type: models.DocAddressProof}, {Type: models.DocFoodSafetyCert}}))
}

func TestFoodRegistrationDocsAreExpiryChecked(t *testing.T) {
	require.True(t, IsFoodRegistrationDoc(models.DocFSSAILicense))
	require.True(t, IsFoodRegistrationDoc(models.DocFoodSafetyCert))
	require.False(t, IsFoodRegistrationDoc(models.DocIDProof))
}

func TestOnlyTheRupeeMarketHasCashfreeOnlyProducts(t *testing.T) {
	require.True(t, IsRupeeMarket("IN"))
	require.True(t, IsRupeeMarket(""), "legacy kitchens are Indian")
	require.False(t, IsRupeeMarket("AU"))
	require.False(t, IsRupeeMarket("nz"))
}

func TestValidateKitchenLocale(t *testing.T) {
	cases := []struct {
		name, country, phone, postcode string
		wantCountry, wantField         string
	}{
		{"legacy india", "", "9876543210", "560001", "IN", ""},
		{"melbourne", "AU", "412345678", "3000", "AU", ""},
		{"auckland", "nz", "211234567", "1010", "NZ", ""},
		{"phone optional", "AU", "", "2000", "AU", ""},
		{"unserved", "US", "", "10001", "", "country"},
		{"indian phone in AU", "AU", "9876543210", "3000", "AU", "phone"},
		{"indian pin in NZ", "NZ", "211234567", "560001", "NZ", "postalCode"},
	}
	for _, tc := range cases {
		country, field, msg := ValidateKitchenLocale(tc.country, tc.phone, tc.postcode)
		require.Equal(t, tc.wantField, field, tc.name)
		if tc.wantField == "" {
			require.Equal(t, tc.wantCountry, country, tc.name)
			require.Empty(t, msg, tc.name)
		} else {
			require.NotEmpty(t, msg, tc.name)
		}
	}
	_, _, msg := ValidateKitchenLocale("AU", "12", "3000")
	require.Contains(t, msg, "Australian")
}

func TestBusinessNumbers(t *testing.T) {
	require.True(t, IsValidBusinessNumber("AU", "51 824 753 556"), "ATO's published sample ABN")
	require.False(t, IsValidBusinessNumber("AU", "51824753557"), "bad ABN checksum")
	require.True(t, IsValidBusinessNumber("NZ", "9429041533864"))
	require.False(t, IsValidBusinessNumber("NZ", "9429041533869"), "bad GS1 check digit")
	require.False(t, IsValidBusinessNumber("NZ", "942904153386"))
	require.True(t, IsValidBusinessNumber("AU", ""), "optional for sole traders under the threshold")
}

func TestNewKitchenProviderFollowsMarket(t *testing.T) {
	require.Equal(t, models.PreferredChefPaymentProvider, ChefPaymentProviderForCountry("IN"))
	require.Equal(t, models.PaymentProviderStripe, ChefPaymentProviderForCountry("AU"))
	require.Equal(t, models.PaymentProviderStripe, ChefPaymentProviderForCountry("NZ"))
}
