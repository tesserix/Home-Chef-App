package models

import (
	"reflect"
	"strings"
	"testing"
)

// #1125 — the column default is what a row inserted WITHOUT an explicit provider
// gets. Since #1101 refuses a razorpay order at verify, such a row is money that
// cannot be captured, and nothing about it fails loudly at insert time. Reflect
// over the tag rather than grep: a tag typo doesn't fail to compile either.
func TestPaymentProviderColumnDefaultsAreCashfree(t *testing.T) {
	cases := []struct {
		model any
		field string
	}{
		{Order{}, "PaymentProvider"},
		{ChefProfile{}, "PaymentProvider"},
		{DeliveryPartner{}, "PaymentProvider"},
		{MealPlan{}, "PaymentProvider"},
		{CateringRequest{}, "PaymentProvider"},
		{ChefPromotion{}, "PaymentProvider"},
		{GroupOrderParticipant{}, "PaymentProvider"},
		{MealSubscription{}, "PaymentGateway"},
	}

	for _, tc := range cases {
		typ := reflect.TypeOf(tc.model)
		field, ok := typ.FieldByName(tc.field)
		if !ok {
			t.Errorf("%s has no field %s", typ.Name(), tc.field)
			continue
		}
		got := gormDefault(field.Tag.Get("gorm"))
		if got != PreferredChefPaymentProvider {
			t.Errorf("%s.%s defaults to %q, want %q — a row inserted without a provider "+
				"would claim a gateway that can no longer take money",
				typ.Name(), tc.field, got, PreferredChefPaymentProvider)
		}
	}
}

// gormDefault pulls the default:'x' value out of a gorm struct tag.
func gormDefault(tag string) string {
	for _, part := range strings.Split(tag, ";") {
		if after, found := strings.CutPrefix(part, "default:"); found {
			return strings.Trim(after, "'")
		}
	}
	return ""
}
