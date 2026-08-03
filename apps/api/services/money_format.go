package services

// money_format.go — one way to write a rupee amount into text a customer or chef
// reads (#934/#935).
//
// Notification copy formatted money with `%.0f`, which rounds to whole rupees:
// "₹306 paid" for an order that charged ₹303.48, "₹327 refunded" against a
// ₹326.64 breakdown. A customer checking a push against their bank app sees two
// different numbers for the same movement.
//
// This mirrors apps/mobile-customer/lib/format.ts exactly — whole amounts show no
// decimals, fractional amounts show paise, grouped Indian-style — so the same
// value reads identically in a push, in the app, and on a receipt.

import (
	"strconv"
	"strings"

	"github.com/homechef/api/models"
)

// FormatMoney renders a rupee amount for display: "₹780", "₹780.50", "₹1,23,456".
//
// A negative amount reads "-₹41.07" rather than "₹-41.07". This is the one place
// the Go formatter deliberately differs from lib/format.ts, whose toLocaleString
// produces the latter — no caller here passes a negative today, and the sign
// belongs outside the symbol if one ever does.
func FormatMoney(amount float64) string {
	if amount < 0 {
		return "-₹" + FormatAmount(-amount)
	}
	return "₹" + FormatAmount(amount)
}

// FormatAmount is FormatMoney without the symbol, for copy that supplies its own.
func FormatAmount(amount float64) string {
	rounded := models.RoundAmount(amount)
	neg := rounded < 0
	if neg {
		rounded = -rounded
	}

	// Work in paise so the whole-rupee test and the decimal both come from one
	// rounding rather than two that can disagree at the boundary.
	paise := int64(rounded*100 + 0.5)
	rupees := paise / 100
	frac := paise % 100

	out := groupIndian(rupees)
	if frac != 0 {
		out += "." + strconv.FormatInt(frac, 10)
		if frac < 10 {
			// 5 paise is ".05", not ".5".
			out = out[:len(out)-1] + "0" + out[len(out)-1:]
		}
	}
	if neg {
		return "-" + out
	}
	return out
}

// groupIndian applies Indian digit grouping: the last three digits, then pairs
// (1,23,456 rather than 123,456).
func groupIndian(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	head, tail := s[:len(s)-3], s[len(s)-3:]

	var parts []string
	for len(head) > 2 {
		parts = append([]string{head[len(head)-2:]}, parts...)
		head = head[:len(head)-2]
	}
	if head != "" {
		parts = append([]string{head}, parts...)
	}
	return strings.Join(parts, ",") + "," + tail
}

// OrderCapturePaise is what the gateway actually charged: the order total less any
// wallet credit and loyalty value applied at checkout.
//
// Reporting order.Total as though it were the charge is what made a push read
// "₹306 paid" when ₹303.48 left the card (#934) — the error scales with the credit
// applied, so it is largest exactly when the customer is most likely to check.
func OrderCapturePaise(order *models.Order) int {
	if order == nil {
		return 0
	}
	capture := ToPaise(order.Total) - ToPaise(order.WalletApplied) - ToPaise(order.LoyaltyApplied)
	if capture < 0 {
		return 0
	}
	return capture
}

// OrderCaptureAmount is OrderCapturePaise in rupees.
func OrderCaptureAmount(order *models.Order) float64 {
	return FromPaise(OrderCapturePaise(order))
}
