package handlers

import "strings"

// maskPhone masks a phone number, showing only last 4 digits.
// e.g., "+919876543210" → "****3210"
func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return "****" + phone[len(phone)-4:]
}

// maskPAN masks a PAN number, showing only last 3 characters.
// e.g., "ABCDE1234F" → "*******34F"
func maskPAN(pan string) string {
	if pan == "" {
		return ""
	}
	if len(pan) <= 3 {
		return strings.Repeat("*", len(pan))
	}
	return strings.Repeat("*", len(pan)-3) + pan[len(pan)-3:]
}

// maskID masks a generic ID/number, showing only last 4 characters.
// e.g., "12345678901234" → "**********1234"
func maskID(id string) string {
	if id == "" {
		return ""
	}
	if len(id) <= 4 {
		return strings.Repeat("*", len(id))
	}
	return strings.Repeat("*", len(id)-4) + id[len(id)-4:]
}

// maskEmail masks an email, showing first char and domain.
// e.g., "john@example.com" → "j***@example.com"
func maskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return "****"
	}
	name := parts[0]
	if len(name) <= 1 {
		return name + "***@" + parts[1]
	}
	return string(name[0]) + strings.Repeat("*", len(name)-1) + "@" + parts[1]
}

// maskBankAccount masks a bank account number, showing only last 4 digits.
// e.g., "1234567890" → "******7890"
func maskBankAccount(account string) string {
	return maskID(account)
}
