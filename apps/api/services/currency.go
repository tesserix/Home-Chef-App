package services

import "strings"

// currencyDecimals returns how many decimal places the given ISO-4217 currency
// uses in its minor unit. Most currencies are 2 (USD cents, INR paise, EUR
// cents). Zero-decimal ones (JPY, KRW, VND, IDR, HUF) are quoted as whole
// units — the "minor" amount equals the major amount. Three-decimal ones
// (KWD, BHD, OMR, TND, JOD, LYD) have 1000 filler units per major unit.
//
// Stripe expects `amount` in minor units for the PaymentIntent API, so the
// right multiplier here is what separates "charged ₹500" from "charged ¥50000
// when the customer owes ¥500".
func currencyDecimals(code string) int {
	switch strings.ToUpper(code) {
	case "JPY", "KRW", "VND", "IDR", "HUF", "CLP", "ISK", "PYG", "RWF", "UGX", "XAF", "XOF", "XPF":
		return 0
	case "KWD", "BHD", "OMR", "TND", "JOD", "LYD":
		return 3
	default:
		return 2
	}
}

// ToMinor converts a major-unit amount (e.g. 499.50 USD) to its minor-unit
// integer representation (49950 cents) using the currency's decimal
// convention. Use this when handing amounts to Stripe/Razorpay.
func ToMinor(amount float64, currency string) int {
	switch currencyDecimals(currency) {
	case 0:
		return int(amount + 0.5)
	case 3:
		return int(amount*1000 + 0.5)
	default:
		return int(amount*100 + 0.5)
	}
}

// FromMinor is the inverse of ToMinor — useful for rendering gateway-sourced
// amounts (e.g. from a webhook payload) as floats for display and arithmetic.
func FromMinor(amount int, currency string) float64 {
	switch currencyDecimals(currency) {
	case 0:
		return float64(amount)
	case 3:
		return float64(amount) / 1000.0
	default:
		return float64(amount) / 100.0
	}
}

// CurrencyForCountry maps an ISO-3166 alpha-2 country to the lowercase
// currency code Stripe expects for charges against a Connect account
// registered in that country. Covers the set of countries Stripe Connect
// Express supports — unknown countries fall back to INR so Stripe rejects
// cleanly rather than silently transacting against the wrong currency.
func CurrencyForCountry(country string) string {
	switch strings.ToUpper(country) {
	case "US":
		return "usd"
	case "GB":
		return "gbp"
	case "CA":
		return "cad"
	case "AU":
		return "aud"
	case "NZ":
		return "nzd"
	case "SG":
		return "sgd"
	case "HK":
		return "hkd"
	case "AE":
		return "aed"
	case "JP":
		return "jpy"
	case "KR":
		return "krw"
	case "MY":
		return "myr"
	case "TH":
		return "thb"
	case "CH":
		return "chf"
	case "SE":
		return "sek"
	case "NO":
		return "nok"
	case "DK":
		return "dkk"
	case "DE", "FR", "IT", "ES", "NL", "BE", "AT", "PT", "IE", "FI", "GR", "LU", "SK", "SI", "EE", "LV", "LT", "CY", "MT":
		return "eur"
	case "IN", "":
		return "inr"
	default:
		return "inr"
	}
}
