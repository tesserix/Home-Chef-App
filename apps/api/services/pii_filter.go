package services

import (
	"regexp"
	"strings"
)

// piiPattern holds a compiled regex and metadata for a single PII type.
type piiPattern struct {
	regex     *regexp.Regexp
	label     string // human-readable violation type
	redaction string // replacement text
	block     bool   // if true, the entire message should be blocked (not just redacted)
}

var piiPatterns []piiPattern

func init() {
	piiPatterns = []piiPattern{
		// --- Phone numbers ---
		// Indian mobile: +91, 0091, 91 prefix followed by 10-digit number starting with 6-9
		{
			regex:     regexp.MustCompile(`(?i)(\+91|0091|91)[\s\-.]?[6-9]\d{9}\b`),
			label:     "phone_number",
			redaction: "[PHONE REDACTED]",
		},
		// 10-digit Indian mobile number (standalone, starts with 6-9)
		{
			regex:     regexp.MustCompile(`\b[6-9]\d{9}\b`),
			label:     "phone_number",
			redaction: "[PHONE REDACTED]",
		},
		// International phone numbers: +<country code> followed by digits, spaces, dashes
		{
			regex:     regexp.MustCompile(`\+\d{1,3}[\s\-.]?\(?\d{1,4}\)?[\s\-.]?\d{3,4}[\s\-.]?\d{3,4}`),
			label:     "phone_number",
			redaction: "[PHONE REDACTED]",
		},
		// Phone with parenthesized area code: (0XX) XXXX-XXXX
		{
			regex:     regexp.MustCompile(`\(\d{2,4}\)\s?\d{3,4}[\s\-.]?\d{3,4}`),
			label:     "phone_number",
			redaction: "[PHONE REDACTED]",
		},

		// --- Email addresses ---
		{
			regex:     regexp.MustCompile(`(?i)[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
			label:     "email_address",
			redaction: "[EMAIL REDACTED]",
		},

		// --- Aadhaar number (12 digits, often written with spaces: XXXX XXXX XXXX) ---
		{
			regex:     regexp.MustCompile(`\b\d{4}[\s\-]?\d{4}[\s\-]?\d{4}\b`),
			label:     "aadhaar_number",
			redaction: "[AADHAAR REDACTED]",
		},

		// --- PAN number (ABCDE1234F pattern) ---
		{
			regex:     regexp.MustCompile(`\b[A-Z]{5}\d{4}[A-Z]\b`),
			label:     "pan_number",
			redaction: "[PAN REDACTED]",
		},

		// --- URLs ---
		{
			regex:     regexp.MustCompile(`(?i)https?://[^\s]+`),
			label:     "url",
			redaction: "[URL BLOCKED]",
			block:     true,
		},
		{
			regex:     regexp.MustCompile(`(?i)www\.[^\s]+`),
			label:     "url",
			redaction: "[URL BLOCKED]",
			block:     true,
		},

		// --- Social media handles & platform mentions ---
		// @username pattern (min 2 chars after @)
		{
			regex:     regexp.MustCompile(`@[a-zA-Z0-9_]{2,30}\b`),
			label:     "social_media_handle",
			redaction: "[HANDLE BLOCKED]",
			block:     true,
		},
		// Platform name mentions with contact intent
		{
			regex:     regexp.MustCompile(`(?i)\b(telegram|whatsapp|whats\s*app|signal|instagram|insta|snapchat|snap|facebook|fb|twitter|discord|viber|wechat|line)\b`),
			label:     "social_media_platform",
			redaction: "[PLATFORM BLOCKED]",
			block:     true,
		},
		// Instagram/telegram/etc. direct links
		{
			regex:     regexp.MustCompile(`(?i)(instagram\.com|t\.me|wa\.me|chat\.whatsapp\.com|signal\.group|discord\.gg|fb\.com|twitter\.com|x\.com)/[^\s]*`),
			label:     "social_media_link",
			redaction: "[SOCIAL LINK BLOCKED]",
			block:     true,
		},

		// --- Physical addresses (house/flat/building number patterns) ---
		// Flat/House/Apt/Building number patterns
		{
			regex:     regexp.MustCompile(`(?i)\b(flat|house|apt|apartment|bldg|building|plot|door|unit)\s*(no\.?|number|#)?\s*[\-:]?\s*\d+[a-zA-Z]?(/\d+)?\b`),
			label:     "physical_address",
			redaction: "[ADDRESS REDACTED]",
		},
		// Indian PIN code in address context (6-digit, starts with 1-9)
		{
			regex:     regexp.MustCompile(`(?i)(pin\s*(code)?|postal\s*code|zip)\s*[\-:]?\s*\d{6}\b`),
			label:     "physical_address",
			redaction: "[PIN REDACTED]",
		},
		// Standalone 6-digit PIN preceded by comma/dash/space and city-like context
		{
			regex:     regexp.MustCompile(`(?i),\s*\d{6}\b`),
			label:     "physical_address",
			redaction: "[PIN REDACTED]",
		},
	}
}

// FilterChatMessage sanitizes a chat message by detecting and redacting PII.
// Returns the sanitized message, whether any PII was detected, and a list of
// violation types that were found.
func FilterChatMessage(content string) (sanitized string, hasPII bool, violations []string) {
	sanitized = content
	seen := make(map[string]bool)

	for _, p := range piiPatterns {
		if p.regex.MatchString(sanitized) {
			sanitized = p.regex.ReplaceAllString(sanitized, p.redaction)
			if !seen[p.label] {
				violations = append(violations, p.label)
				seen[p.label] = true
			}
			hasPII = true
		}
	}

	// Normalize excessive whitespace that may result from redaction
	spaceNorm := regexp.MustCompile(`\s{2,}`)
	sanitized = strings.TrimSpace(spaceNorm.ReplaceAllString(sanitized, " "))

	return sanitized, hasPII, violations
}
