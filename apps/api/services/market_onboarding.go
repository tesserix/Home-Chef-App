package services

import (
	"regexp"
	"strings"

	"github.com/homechef/api/internal/markets"
	"github.com/homechef/api/models"
)

var postcodeRules = map[string]*regexp.Regexp{
	"IN": regexp.MustCompile(`^\d{6}$`),
	"AU": regexp.MustCompile(`^\d{4}$`),
	"NZ": regexp.MustCompile(`^\d{4}$`),
}

// NormalizeKitchenCountry maps a kitchen's country onto a served market; blank
// is India so clients that predate the field keep working.
func NormalizeKitchenCountry(raw string) (string, bool) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if code == "" {
		return "IN", true
	}
	if _, ok := markets.Lookup(code); !ok {
		return "", false
	}
	return code, true
}

// IsValidPostcode reports whether postcode is well-formed for the country.
func IsValidPostcode(country, postcode string) bool {
	rule, ok := postcodeRules[strings.ToUpper(strings.TrimSpace(country))]
	return ok && rule.MatchString(strings.TrimSpace(postcode))
}

// FoodRegistrationDocType is the food-business licence a kitchen must hold:
// FSSAI in India, the local council's food business registration in AU/NZ.
func FoodRegistrationDocType(country string) models.DocumentType {
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "AU", "NZ":
		return models.DocFoodSafetyCert
	default:
		return models.DocFSSAILicense
	}
}

// RequiredChefDocTypes is the document set a kitchen in country must upload
// before it can be approved.
func RequiredChefDocTypes(country string) []models.DocumentType {
	return []models.DocumentType{models.DocIDProof, models.DocAddressProof, FoodRegistrationDocType(country)}
}

// IsFoodRegistrationDoc reports whether docType is a food-business licence,
// whose lapsed copies are refused at upload.
func IsFoodRegistrationDoc(docType models.DocumentType) bool {
	return docType == models.DocFSSAILicense || docType == models.DocFoodSafetyCert
}

// MissingChefDocs lists the required document types the kitchen has not
// uploaded, in checklist order.
func MissingChefDocs(country string, docs []models.ChefDocument) []string {
	present := map[models.DocumentType]bool{}
	for _, d := range docs {
		present[d.Type] = true
	}
	missing := []string{}
	for _, req := range RequiredChefDocTypes(country) {
		if !present[req] {
			missing = append(missing, string(req))
		}
	}
	return missing
}

// IsRupeeMarket reports whether a kitchen in country charges in INR. Meal plans
// and group orders pay only through Cashfree, so they are offered only here.
func IsRupeeMarket(country string) bool {
	return CurrencyForCountry(strings.TrimSpace(country)) == "inr"
}

var phoneHints = map[string]string{
	"IN": "Enter a valid 10-digit Indian mobile number",
	"AU": "Enter a valid Australian mobile number, without the leading 0 (e.g. 412 345 678)",
	"NZ": "Enter a valid New Zealand mobile number, without the leading 0 (e.g. 21 123 4567)",
}

var postcodeHints = map[string]string{
	"IN": "Enter a valid 6-digit PIN code",
	"AU": "Enter a valid 4-digit Australian postcode",
	"NZ": "Enter a valid 4-digit New Zealand postcode",
}

// ValidateKitchenLocale checks the onboarding country, phone and postcode
// together. On failure it returns the offending field and a user-facing message.
func ValidateKitchenLocale(rawCountry, phone, postcode string) (country, field, message string) {
	country, ok := NormalizeKitchenCountry(rawCountry)
	if !ok {
		return "", "country", "Fe3dr kitchens can only be registered in India, Australia or New Zealand"
	}
	if phone != "" && !IsValidPhone(country, phone) {
		return country, "phone", phoneHints[country]
	}
	if !IsValidPostcode(country, postcode) {
		return country, "postalCode", postcodeHints[country]
	}
	return country, "", ""
}

var abnWeights = [11]int{10, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19}

// IsValidBusinessNumber checks an optional AU ABN (11-digit mod-89 checksum)
// or NZ NZBN (13-digit GS1 number); other countries have no such number here.
func IsValidBusinessNumber(country, raw string) bool {
	digits := strings.ReplaceAll(strings.TrimSpace(raw), " ", "")
	if digits == "" {
		return true
	}
	if strings.Trim(digits, "0123456789") != "" {
		return false
	}
	switch strings.ToUpper(country) {
	case "AU":
		if len(digits) != 11 {
			return false
		}
		sum := 0
		for i, w := range abnWeights {
			d := int(digits[i] - '0')
			if i == 0 {
				d--
			}
			sum += d * w
		}
		return sum%89 == 0
	case "NZ":
		if len(digits) != 13 || !strings.HasPrefix(digits, "94") {
			return false
		}
		sum := 0
		for i := 0; i < 12; i++ {
			w := 1
			if i%2 == 1 {
				w = 3
			}
			sum += int(digits[i]-'0') * w
		}
		return (10-sum%10)%10 == int(digits[12]-'0')
	default:
		return false
	}
}

// ChefPaymentProviderForCountry is the gateway stamped on a new kitchen: the
// market's own provider outside India, the preferred Indian gateway otherwise.
func ChefPaymentProviderForCountry(country string) string {
	if market, ok := markets.Lookup(country); ok && market.CountryCode != "IN" {
		return market.PaymentProvider
	}
	return DefaultChefPaymentProvider(models.ChefModeLive)
}
