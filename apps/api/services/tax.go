package services

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// TaxLookup resolves the applicable tax rate for a given country (and
// optionally region, for countries where rates vary below the country
// level — US states, Canadian provinces). Rules come from the tax_rates
// table, cached for 5 minutes so per-order lookups don't round-trip the DB.
//
// The resolution order is: (country, region) → (country, "") → zero. Zero
// means "tax not configured for this region" and returns a no-op rule with
// rate=0 and name="Tax" so the order still goes through (just with no tax
// line) and the admin can spot the gap in reporting.

const taxCacheTTL = 5 * time.Minute

type taxKey struct {
	Country string
	Region  string
}

var (
	taxCache     = map[taxKey]*models.TaxRate{}
	taxCacheMu   sync.RWMutex
	taxCachedAt  time.Time
	taxEmptyRate = &models.TaxRate{
		TaxName: "Tax",
		Rate:    0,
	}
)

func refreshTaxCache() {
	var rules []models.TaxRate
	if err := database.DB.Where("is_active = ?", true).Find(&rules).Error; err != nil {
		log.Printf("tax: failed to load rules: %v", err)
		return
	}
	next := make(map[taxKey]*models.TaxRate, len(rules))
	for i := range rules {
		r := rules[i]
		next[taxKey{Country: strings.ToUpper(r.CountryCode), Region: strings.ToUpper(r.Region)}] = &r
	}
	taxCacheMu.Lock()
	taxCache = next
	taxCachedAt = time.Now()
	taxCacheMu.Unlock()
}

// ResolveTaxRate picks the tax rule for a delivery address. Region wins
// over country-wide if present; otherwise falls back to country-wide.
// Returns a zero-rate default if no rule exists so callers don't have to
// nil-check.
func ResolveTaxRate(country, region string) *models.TaxRate {
	country = strings.ToUpper(country)
	region = strings.ToUpper(region)

	taxCacheMu.RLock()
	stale := time.Since(taxCachedAt) > taxCacheTTL
	taxCacheMu.RUnlock()
	if stale {
		refreshTaxCache()
	}

	taxCacheMu.RLock()
	defer taxCacheMu.RUnlock()

	if region != "" {
		if r, ok := taxCache[taxKey{Country: country, Region: region}]; ok {
			return r
		}
	}
	if r, ok := taxCache[taxKey{Country: country, Region: ""}]; ok {
		return r
	}
	return taxEmptyRate
}

// InvalidateTaxCache forces the next ResolveTaxRate call to re-read the DB.
// Call after admin tax-rate edits.
func InvalidateTaxCache() {
	taxCacheMu.Lock()
	taxCachedAt = time.Time{}
	taxCacheMu.Unlock()
}

// SeedTaxRates inserts a baseline row per country on first boot. Safe to
// call every startup — uses INSERT ... WHERE NOT EXISTS semantics via
// GORM's FirstOrCreate so existing rules are never overwritten.
//
// These are conservative, country-wide defaults meant to get the invoice
// pipeline working; operators should refine per-region rules in the admin
// UI once deployed. Picked from public tax-rate references as of mid-2025;
// values may drift and should be reviewed quarterly.
func SeedTaxRates() {
	type seed struct {
		Country string
		Region  string
		Name    string
		Rate    float64
		Incl    bool
		Notes   string
		// Per-supply overrides. Left at 0 the supply takes Rate, which is what
		// every non-India row below does — they are single-rate jurisdictions.
		Food             float64
		Service          float64
		ServiceIncl      bool
		DeliverySelf     float64
		DeliveryPlatform float64
		Subscription     float64
		IDLabel          string
	}
	seeds := []seed{
		// India. One bill, three rates — see docs/india-gst-model.md:
		//   food                5% exclusive  — restaurant service, no ITC (Notif. 11/2017-CTR)
		//   platform fee       18% INCLUSIVE  — the platform's own service. Inclusive is the
		//                                      pricing decision: the customer's all-in fee does
		//                                      not move and the GST is backed out of it.
		//   chef self-delivery  5% exclusive  — door delivery by the cook, arguable as part of
		//                                      the composite restaurant supply (open: §7 Q2)
		//   platform delivery  18% exclusive  — local delivery through an ECO, a notified §9(5)
		//                                      supply since 22 Sep 2025 (56th Council, 3 Sep 2025)
		//   subscription       18% exclusive  — recurring plan fee, a standard-rated service
		{
			Country: "IN", Name: "GST", Rate: 5.0, Incl: false,
			Food: 5, Service: 18, ServiceIncl: true, DeliverySelf: 5, DeliveryPlatform: 18, Subscription: 18,
			IDLabel: "GSTIN",
			Notes:   "India GST. Food 5% (no ITC); platform fee 18% inclusive; delivery 5% chef / 18% platform.",
		},

		// European Union — reduced VAT on prepared food varies (5%–10%);
		// picking a mid-point per country. Admins should confirm.
		{Country: "GB", Name: "VAT", Rate: 20.0, Incl: true, Notes: "UK VAT on hot takeaway food."},
		{Country: "IE", Name: "VAT", Rate: 13.5, Incl: true, Notes: "Irish VAT reduced rate on restaurant/food services."},
		{Country: "DE", Name: "VAT", Rate: 19.0, Incl: true, Notes: "Germany standard VAT."},
		{Country: "FR", Name: "TVA", Rate: 10.0, Incl: true, Notes: "France reduced TVA on prepared food."},
		{Country: "IT", Name: "IVA", Rate: 10.0, Incl: true, Notes: "Italy reduced IVA on restaurant services."},
		{Country: "ES", Name: "IVA", Rate: 10.0, Incl: true, Notes: "Spain reduced IVA on restaurant services."},
		{Country: "NL", Name: "BTW", Rate: 9.0, Incl: true, Notes: "Netherlands reduced BTW on food."},
		{Country: "BE", Name: "VAT", Rate: 12.0, Incl: true, Notes: "Belgium reduced VAT on restaurant services."},
		{Country: "PT", Name: "IVA", Rate: 13.0, Incl: true, Notes: "Portugal reduced IVA on restaurant services."},
		{Country: "AT", Name: "MwSt", Rate: 10.0, Incl: true, Notes: "Austria reduced MwSt on food."},

		// Nordics
		{Country: "SE", Name: "VAT", Rate: 12.0, Incl: true, Notes: "Sweden reduced VAT on food."},
		{Country: "NO", Name: "VAT", Rate: 15.0, Incl: true, Notes: "Norway reduced VAT on food."},
		{Country: "DK", Name: "VAT", Rate: 25.0, Incl: true, Notes: "Denmark standard VAT (no reduced rate on food)."},
		{Country: "FI", Name: "VAT", Rate: 14.0, Incl: true, Notes: "Finland reduced VAT on food."},

		// North America — tax is sub-national; country-wide rate is the
		// federal portion. Admins should add state/province rows to
		// cover the combined rate for each jurisdiction.
		{Country: "US", Name: "Sales Tax", Rate: 0.0, Incl: false, Notes: "US has no federal sales tax — add state-level TaxRate rows per region."},
		{Country: "CA", Name: "GST", Rate: 5.0, Incl: false, Notes: "Canadian federal GST; provinces stack HST/PST (AB=5, ON=13, QC=14.975, BC=12)."},

		// Canada — common provincial rules so GST+provincial stacks.
		{Country: "CA", Region: "ON", Name: "HST", Rate: 13.0, Incl: false, Notes: "Ontario HST (supersedes GST-only)."},
		{Country: "CA", Region: "QC", Name: "GST+QST", Rate: 14.975, Incl: false, Notes: "Quebec GST+QST combined."},
		{Country: "CA", Region: "BC", Name: "GST+PST", Rate: 12.0, Incl: false, Notes: "BC GST+PST combined."},
		{Country: "CA", Region: "AB", Name: "GST", Rate: 5.0, Incl: false, Notes: "Alberta — GST only."},
		{Country: "CA", Region: "NS", Name: "HST", Rate: 15.0, Incl: false, Notes: "Nova Scotia HST."},

		// Asia-Pacific
		{Country: "AU", Name: "GST", Rate: 10.0, Incl: true, Notes: "Australian GST."},
		{Country: "NZ", Name: "GST", Rate: 15.0, Incl: true, Notes: "New Zealand GST."},
		{Country: "SG", Name: "GST", Rate: 9.0, Incl: false, Notes: "Singapore GST."},
		{Country: "HK", Name: "Tax", Rate: 0.0, Incl: false, Notes: "Hong Kong has no sales tax / GST on food."},
		{Country: "MY", Name: "SST", Rate: 6.0, Incl: false, Notes: "Malaysia Sales & Service Tax (service portion)."},
		{Country: "TH", Name: "VAT", Rate: 7.0, Incl: true, Notes: "Thailand VAT."},
		{Country: "JP", Name: "Consumption Tax", Rate: 10.0, Incl: true, Notes: "Japan Consumption Tax (8% reduced for takeaway, 10% dine-in — defaulting to 10%)."},
		{Country: "KR", Name: "VAT", Rate: 10.0, Incl: true, Notes: "South Korea VAT."},
		{Country: "AE", Name: "VAT", Rate: 5.0, Incl: false, Notes: "UAE VAT."},

		// South Asia (neighbors of India)
		{Country: "PK", Name: "GST", Rate: 17.0, Incl: false, Notes: "Pakistan GST on services (varies by province)."},
		{Country: "BD", Name: "VAT", Rate: 15.0, Incl: false, Notes: "Bangladesh VAT."},
		{Country: "LK", Name: "VAT", Rate: 18.0, Incl: false, Notes: "Sri Lanka VAT."},
		{Country: "NP", Name: "VAT", Rate: 13.0, Incl: false, Notes: "Nepal VAT."},

		// Middle East / Gulf
		{Country: "SA", Name: "VAT", Rate: 15.0, Incl: false, Notes: "Saudi Arabia VAT."},
		{Country: "QA", Name: "Tax", Rate: 0.0, Incl: false, Notes: "Qatar has no VAT as of 2025."},
		{Country: "KW", Name: "Tax", Rate: 0.0, Incl: false, Notes: "Kuwait has no VAT as of 2025."},
		{Country: "BH", Name: "VAT", Rate: 10.0, Incl: false, Notes: "Bahrain VAT."},
		{Country: "OM", Name: "VAT", Rate: 5.0, Incl: false, Notes: "Oman VAT."},
	}

	for _, s := range seeds {
		// FirstOrCreate on (country, region) so reruns don't duplicate
		// rows but admin edits survive future restarts.
		var row models.TaxRate
		result := database.DB.Where(
			"country_code = ? AND region = ?",
			strings.ToUpper(s.Country), strings.ToUpper(s.Region),
		).Attrs(models.TaxRate{
			TaxName:                 s.Name,
			Rate:                    s.Rate,
			Inclusive:               s.Incl,
			FoodPercent:             s.Food,
			ServicePercent:          s.Service,
			ServiceInclusive:        s.ServiceIncl,
			DeliverySelfPercent:     s.DeliverySelf,
			DeliveryPlatformPercent: s.DeliveryPlatform,
			SubscriptionPercent:     s.Subscription,
			RegistrationIDLabel:     s.IDLabel,
			Notes:                   s.Notes,
			IsActive:                true,
		}).FirstOrCreate(&row, models.TaxRate{
			CountryCode: strings.ToUpper(s.Country),
			Region:      strings.ToUpper(s.Region),
		})
		if result.Error != nil {
			log.Printf("tax: seed failed for %s/%s: %v", s.Country, s.Region, result.Error)
			continue
		}
		fillUnsetSupplyRates(row, s.Food, s.Service, s.ServiceIncl, s.DeliverySelf, s.DeliveryPlatform, s.Subscription, s.IDLabel)
	}
	InvalidateTaxCache()
	log.Printf("tax: seeded %d baseline rules (existing admin edits preserved)", len(seeds))
}

// fillUnsetSupplyRates back-fills the per-supply columns on a row that predates
// them — one UPDATE per column, each guarded on the column still being unset, so
// a rate an admin has deliberately chosen is never overwritten and re-running is
// a no-op.
//
// ⚠️ On an existing deployment this is the step that MOVES MONEY: an India row
// created before per-supply rates existed carries 0 in every column, so this is
// what puts the platform fee on 18% and platform-arranged delivery on 18%. That
// is the intended go-live (docs/gst-refunds-and-gateway-costs.md §9) — but it
// happens on the first boot after deploy, not on an admin's say-so, so the log
// line below is deliberately loud.
func fillUnsetSupplyRates(row models.TaxRate, food, service float64, serviceIncl bool, deliverySelf, deliveryPlatform, subscription float64, idLabel string) {
	set := func(column string, value any, guard string, guardArgs ...any) {
		args := append([]any{row.ID}, guardArgs...)
		if err := database.DB.Model(&models.TaxRate{}).
			Where("id = ? AND "+guard, args...).
			Update(column, value).Error; err != nil {
			log.Printf("tax: adopt %s failed for %s: %v", column, row.CountryCode, err)
		}
	}
	for _, c := range []struct {
		column string
		value  float64
	}{
		{"food_percent", food},
		{"service_percent", service},
		{"delivery_self_percent", deliverySelf},
		{"delivery_platform_percent", deliveryPlatform},
		{"subscription_percent", subscription},
	} {
		if c.value > 0 {
			set(c.column, c.value, c.column+" = 0")
		}
	}
	// Inclusivity rides with the service rate: it is only meaningful once that
	// rate is set, and it is set in the same "was still unset" window.
	if serviceIncl {
		set("service_inclusive", true, "service_percent = ? AND service_inclusive = false", service)
	}
	if idLabel != "" {
		set("registration_id_label", idLabel, "(registration_id_label = '' OR registration_id_label IS NULL)")
	}
}
