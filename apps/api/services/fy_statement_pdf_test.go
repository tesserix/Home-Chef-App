package services

import (
	"os"
	"testing"

	"github.com/homechef/api/models"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/config"
)

func fyStatementFixture() *FYStatement {
	return &FYStatement{
		FYStartYear:        2026,
		FYLabel:            "FY 2026-27",
		Currency:           "INR",
		OrdersCount:        20,
		FoodRevenue:        7780,
		GSTCollected:       452.46,
		Tips:               105,
		GrossReceipts:      8821.61,
		PlatformCommission: 466.80,
		CommissionCGST:     42.03,
		CommissionSGST:     42.03,
		TDSWithheld:        88.22,
		NetEarnings:        8266.59,
		Quarters: []FYQuarter{
			{Label: "Q1 (Apr–Jun)"},
			{Label: "Q2 (Jul–Sep)", OrdersCount: 20, Gross: 8821.61, TDS: 88.22, NetPayout: 8266.59},
			{Label: "Q3 (Oct–Dec)"},
			{Label: "Q4 (Jan–Mar)"},
		},
		Expenses: &ExpenseSummary{ByCategory: []ExpenseCategoryTotal{
			{Category: models.ExpenseIngredients, Count: 3, Amount: 265},
		}},
		TotalExpenses: 265,
		NetIncome:     8001.59,
	}
}

// The statement is assembled from six blocks; rendering all of them for a
// realistic year keeps a layout change from shipping as a document that fails
// to generate (the DB-backed entry point can't be exercised in a unit test).
func TestFYStatementDocumentRenders(t *testing.T) {
	chef := models.ChefProfile{
		BusinessName: "Saffron Home Kitchen",
		PanNumber:    "ABCPV1234D",
		GSTIN:        "29ABCDE1234F1Z5",
		City:         "Bengaluru",
		State:        "Karnataka",
		PostalCode:   "560038",
	}
	stmt := fyStatementFixture()

	m := maroto.New(config.NewBuilder().WithLeftMargin(15).WithTopMargin(15).WithRightMargin(15).Build())
	addFYHeader(m, stmt)
	addFYSummaryBand(m, stmt)
	addFYParties(m, &chef)
	addFYIncome(m, stmt)
	addFYQuarters(m, stmt)
	addFYExpenses(m, stmt)
	addFYNetIncome(m, stmt)
	addFYFooter(m)

	doc, err := m.Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(doc.GetBytes()) < 1000 {
		t.Fatalf("document looks empty: %d bytes", len(doc.GetBytes()))
	}
	if out := os.Getenv("FY_STATEMENT_PDF_OUT"); out != "" {
		_ = os.WriteFile(out, doc.GetBytes(), 0o600)
	}
}

// A brand new chef with no orders still downloads a statement; the empty-state
// blocks have to render rather than divide by a missing summary.
func TestFYStatementRendersWithNoActivity(t *testing.T) {
	stmt := &FYStatement{FYStartYear: 2026, FYLabel: "FY 2026-27", Currency: "INR", Quarters: []FYQuarter{{Label: "Q1 (Apr–Jun)"}}}

	m := maroto.New(config.NewBuilder().WithLeftMargin(15).WithTopMargin(15).WithRightMargin(15).Build())
	addFYHeader(m, stmt)
	addFYSummaryBand(m, stmt)
	addFYParties(m, &models.ChefProfile{})
	addFYIncome(m, stmt)
	addFYQuarters(m, stmt)
	addFYExpenses(m, stmt)
	addFYNetIncome(m, stmt)
	addFYFooter(m)

	if _, err := m.Generate(); err != nil {
		t.Fatalf("generate: %v", err)
	}
}

// The masthead carries the app's brand mark, so the embedded asset has to be
// a real PNG and not an empty file the linker happily accepted.
func TestBrandMarkAssetIsUsable(t *testing.T) {
	png := brandMarkPNG()
	if len(png) < 512 {
		t.Fatalf("brand mark looks empty: %d bytes", len(png))
	}
	if string(png[1:4]) != "PNG" {
		t.Fatalf("brand mark is not a PNG: % x", png[:8])
	}
}

// Negative movements read as deductions, not as arithmetic typos, and every
// amount lines up on the same decimal.
func TestFormatStatementAmount(t *testing.T) {
	cases := map[float64]string{
		0:        "0.00",
		8821.61:  "8,821.61",
		-466.8:   "(466.80)",
		-88.22:   "(88.22)",
		1234567:  "1,234,567.00",
		-1234.5:  "(1,234.50)",
		105:      "105.00",
		-1000000: "(1,000,000.00)",
	}
	for in, want := range cases {
		if got := formatStatementAmount(in); got != want {
			t.Errorf("formatStatementAmount(%v) = %q, want %q", in, got, want)
		}
	}
}
