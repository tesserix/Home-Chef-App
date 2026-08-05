package services

// fssai_pricing.go — what a chef pays us to obtain their FSSAI registration.
//
//	total = (government fee × years) + service fee + GST on both
//
// One computation, used by the quote the app shows, by the charge that is
// actually minted, and by the tests. A screen that priced this itself is how
// checkout came to show ₹264.58 for a receipt that said ₹264.57.
//
// The government fee is a PASS-THROUGH: we collect it and pay FoSCoS with it.
// It is stated separately on every surface so a chef can see that the ₹100 is
// FSSAI's and only the service fee is ours.
//
// See .planning/FSSAI-IN-APP-REQUEST-DESIGN.md.

import "github.com/homechef/api/models"

// FssaiQuote is the price of one filing request, broken out so every surface
// renders the same lines and none of them re-add the total.
type FssaiQuote struct {
	TermYears int `json:"termYears"`
	// GovernmentFee is FSSAI's own fee for the whole term, before tax.
	GovernmentFee float64 `json:"governmentFee"`
	// GovernmentTax is GST on that fee. FSSAI licensing lost its Entry 47
	// exemption on 18 July 2022, so this is not zero.
	GovernmentTax float64 `json:"governmentTax"`
	// ServiceFee is ours, charged once per application whatever the term — it is
	// one form whether the chef registers for one year or five.
	ServiceFee float64 `json:"serviceFee"`
	ServiceTax float64 `json:"serviceTax"`
	// Total is the EXACT sum of the four lines above, to the paise.
	Total      float64 `json:"total"`
	Currency   string  `json:"currency"`
	GstPercent float64 `json:"gstPercent"`
}

// FssaiTermYearsMin / Max — FSSAI issues a registration for one to five years.
const (
	FssaiTermYearsMin = 1
	FssaiTermYearsMax = 5
)

// QuoteFssaiFiling prices a filing request for a term, from the live policy.
//
// Years outside 1–5 are clamped rather than rejected: this is a price, and a
// caller asking for an impossible term gets the nearest real one. The handler
// validates the input separately, so a bad request is still a 400 — this
// function simply cannot return a nonsense price.
func QuoteFssaiFiling(years int) FssaiQuote {
	p := GetPlatformPolicy()
	if years < FssaiTermYearsMin {
		years = FssaiTermYearsMin
	}
	if years > FssaiTermYearsMax {
		years = FssaiTermYearsMax
	}

	gst := p.FssaiGstPercent / 100
	q := FssaiQuote{
		TermYears:     years,
		GovernmentFee: Round2(p.FssaiGovernmentFeePerYear * float64(years)),
		ServiceFee:    Round2(p.FssaiServiceFee),
		Currency:      "INR",
		GstPercent:    p.FssaiGstPercent,
	}
	q.GovernmentTax = Round2(q.GovernmentFee * gst)
	q.ServiceTax = Round2(q.ServiceFee * gst)
	// Total is the sum of the SNAPPED lines, never an independently rounded
	// figure — so the breakdown a chef reads always adds up to what they are
	// charged. Round2 here only removes IEEE-754 noise.
	q.Total = Round2(q.GovernmentFee + q.GovernmentTax + q.ServiceFee + q.ServiceTax)
	return q
}

// FssaiFilingEnabled reports whether the in-app filing request is open for
// business. Default ON, per the owner — switchable off at runtime from
// Settings → Platform without a deploy, because the flow takes money and
// identity documents and may need stopping at short notice.
func FssaiFilingEnabled() bool { return GetPlatformPolicy().FssaiFilingEnabled }

// ApplyQuote stamps a priced quote onto a request. The figures are FROZEN on
// the row at creation: a later price change must never restate what a chef was
// already charged, and the refund path reads these, not today's policy.
func ApplyQuote(r *models.FssaiRequest, q FssaiQuote) {
	r.TermYears = q.TermYears
	// FeeAmount is everything before tax — the government fee plus ours — because
	// that is the base the charge was computed on.
	r.FeeAmount = Round2(q.GovernmentFee + q.ServiceFee)
	r.FeeTax = Round2(q.GovernmentTax + q.ServiceTax)
	r.FeeTotal = q.Total
	r.Currency = q.Currency
}
