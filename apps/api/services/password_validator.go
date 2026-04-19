package services

import (
	"errors"
	"fmt"
	"unicode"
)

// ValidatePasswordAgainstPolicy checks a plaintext password against the
// currently configured SecurityPolicy. The returned error message is
// safe to show to users — it describes what's missing without leaking
// any internal detail.
func ValidatePasswordAgainstPolicy(pw string) error {
	return validatePassword(pw, GetSecurityPolicy())
}

// ValidatePasswordWithPolicy is the pure form, exposed for tests.
func ValidatePasswordWithPolicy(pw string, p SecurityPolicy) error {
	return validatePassword(pw, p)
}

func validatePassword(pw string, p SecurityPolicy) error {
	if len(pw) < p.PasswordMinLength {
		return fmt.Errorf("password must be at least %d characters", p.PasswordMinLength)
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, r := range pw {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasNumber = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	var missing []string
	if p.PasswordRequireUpper && !hasUpper {
		missing = append(missing, "an uppercase letter")
	}
	if p.PasswordRequireLower && !hasLower {
		missing = append(missing, "a lowercase letter")
	}
	if p.PasswordRequireNumber && !hasNumber {
		missing = append(missing, "a number")
	}
	if p.PasswordRequireSpecial && !hasSpecial {
		missing = append(missing, "a special character")
	}

	if len(missing) == 0 {
		return nil
	}

	return errors.New("password must include " + joinMissing(missing))
}

func joinMissing(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	}
	// Oxford comma for 3+
	out := ""
	for i, p := range parts {
		if i == len(parts)-1 {
			out += "and " + p
			continue
		}
		out += p + ", "
	}
	return out
}
