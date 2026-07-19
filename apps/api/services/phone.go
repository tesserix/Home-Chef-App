package services

// phone.go — server-side phone validation (defense in depth).
//
// The mobile client already hard-caps and validates phone input, but the API
// must never trust the client: a direct/replayed request could still carry a
// malformed number. This mirrors the client's country-aware rule
// (packages/mobile-shared/src/validation/phone.ts). India-only today; add a
// country here as the platform expands.

import (
	"regexp"
	"strings"
)

// phoneRule is the national-number rule for one country.
type phoneRule struct {
	pattern *regexp.Regexp
	length  int
}

var phoneRules = map[string]phoneRule{
	"IN": {pattern: regexp.MustCompile(`^[6-9]\d{9}$`), length: 10},
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
