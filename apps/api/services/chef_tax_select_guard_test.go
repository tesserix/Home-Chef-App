package services

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ChefTaxOf falls back to the WHOLE order tax when tax_food and tax_service both
// scan as zero — correct for orders priced before tax was split per supply, and
// silently wrong for every order since. A raw SELECT that reads `tax` but omits
// the two per-supply columns therefore credits the chef the platform's own GST
// on the fee and delivery, with nothing to indicate it.
//
// That is exactly what shipped: #983 added the fields to the scan structs and
// routed them through ChefTaxOf, but two raw queries never selected them. The
// regression test at the time exercised the helper directly and stayed green.
//
// This guard is static because the defect is static: it lives in a SELECT list,
// not in behaviour any unit test of the helper can reach.
func TestRawOrderSelectsCarryPerSupplyTax(t *testing.T) {
	// Matches a raw SQL SELECT ... FROM orders block in Go source.
	selectBlock := regexp.MustCompile(`(?is)SELECT\s+(.*?)\s+FROM\s+orders\b`)
	// `tax` as a selected column — o.tax or a bare tax, not tax_food/tax_rate/etc.
	readsTax := regexp.MustCompile(`(?i)(^|[\s,(])(o\.)?tax\s*(,|$|\s)`)

	var offenders []string
	roots := []string{"../services", "../handlers"}
	for _, root := range roots {
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") ||
				strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			// Strip // comments: a doc block quoting a query is not a query.
			code := regexp.MustCompile(`(?m)^\s*//.*$`).ReplaceAllString(string(src), "")
			for _, m := range selectBlock.FindAllStringSubmatch(code, -1) {
				cols := m[1]
				if !readsTax.MatchString(cols) {
					continue // does not read tax at all — nothing to get wrong
				}
				if strings.Contains(cols, "tax_food") && strings.Contains(cols, "tax_service") {
					continue
				}
				// Aggregates (SUM/COUNT over revenue) never feed ChefTaxOf.
				if strings.Contains(strings.ToUpper(cols), "SUM(") ||
					strings.Contains(strings.ToUpper(cols), "COUNT(") {
					continue
				}
				offenders = append(offenders, filepath.Base(path))
			}
			return nil
		})
	}

	if len(offenders) > 0 {
		t.Fatalf("raw SELECT reads orders.tax without tax_food/tax_service in %v — "+
			"ChefTaxOf will fall back to the whole order tax and credit the chef the "+
			"platform's GST on the fee and delivery", offenders)
	}
}
