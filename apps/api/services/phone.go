package services

// phone.go — server-side phone validation (defense in depth).
//
// The mobile client already hard-caps and validates phone input, but the API
// must never trust the client: a direct/replayed request could still carry a
// malformed number. This mirrors the client's country-aware rule
// (packages/mobile-shared/src/validation/phone.ts). Add a country here as the
// platform expands.

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// phoneRule is the national-number rule for one country.
type phoneRule struct {
	pattern *regexp.Regexp
	length  int
}

var phoneRules = map[string]phoneRule{
	"IN": {pattern: regexp.MustCompile(`^[6-9]\d{9}$`), length: 10},
	"AU": {pattern: regexp.MustCompile(`^4\d{8}$`), length: 9},
	"NZ": {pattern: regexp.MustCompile(`^2\d{7,9}$`), length: 10},
}

const defaultPhoneCountry = "IN"

func phoneRuleFor(country string) phoneRule {
	if r, ok := phoneRules[strings.ToUpper(strings.TrimSpace(country))]; ok {
		return r
	}
	return phoneRules[defaultPhoneCountry]
}

// IsValidPhone reports whether phone is a complete, valid national number for the
// country. An empty phone is NOT valid here — callers that treat phone as
// optional should guard on non-empty before calling (mirrors the existing
// `if req.Phone != ""` checks at the write sites).
func IsValidPhone(country, phone string) bool {
	return phoneRuleFor(country).pattern.MatchString(strings.TrimSpace(phone))
}

// CustomerPhoneCountry picks the country a customer's phone is validated in: the
// one they chose for the phone, else their address's, else India for old clients.
func CustomerPhoneCountry(phoneCountry, addressCountry string) (string, bool) {
	if strings.TrimSpace(phoneCountry) != "" {
		return NormalizeKitchenCountry(phoneCountry)
	}
	return NormalizeKitchenCountry(addressCountry)
}

// InvalidPhoneMessage is the validation error shown for a bad number in country.
func InvalidPhoneMessage(country string) string {
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "AU":
		return "Enter a valid 9-digit mobile number, without the leading 0"
	case "NZ":
		return "Enter a valid NZ mobile number, without the leading 0"
	default:
		return "Enter a valid 10-digit mobile number"
	}
}

// DefaultAddressPhoneCountry is the phone country implied by the customer's
// default address, for clients that don't send one; India when unknown.
func DefaultAddressPhoneCountry(db *gorm.DB, userID uuid.UUID) string {
	var country string
	db.Table("addresses").Select("country").Where("user_id = ?", userID).
		Order("is_default DESC, created_at DESC").Limit(1).Scan(&country)
	if code, ok := NormalizeKitchenCountry(country); ok {
		return code
	}
	return defaultPhoneCountry
}
