package services

// fy_statement_pdf.go — renders the annual income & expense statement PDF a
// chef downloads at tax time (GST + ITR working papers). Numbers come from
// ComputeFYStatement; this file is layout only, and borrows the invoice's
// palette and hairline so the two documents read as one family.

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// expenseCategoryLabels are the human names printed on the PDF and UIs.
var expenseCategoryLabels = map[models.ChefExpenseCategory]string{
	models.ExpenseIngredients: "Ingredients & groceries",
	models.ExpenseGas:         "Cooking gas / fuel",
	models.ExpenseUtensils:    "Utensils & cookware",
	models.ExpensePackaging:   "Packaging & disposables",
	models.ExpenseTransport:   "Transport & delivery",
	models.ExpenseEquipment:   "Kitchen equipment",
	models.ExpenseUtilities:   "Electricity & utilities",
	models.ExpenseOther:       "Other expenses",
}

// ExpenseCategoryLabel resolves the display label for a category.
func ExpenseCategoryLabel(c models.ChefExpenseCategory) string {
	if label, ok := expenseCategoryLabels[c]; ok {
		return label
	}
	return string(c)
}

// GenerateFYStatementPDF renders the FY income & expense statement.
func GenerateFYStatementPDF(chefID uuid.UUID, fyStartYear int) ([]byte, string, error) {
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		return nil, "", fmt.Errorf("chef not found: %w", err)
	}
	stmt, err := ComputeFYStatement(chefID, fyStartYear)
	if err != nil {
		return nil, "", err
	}

	cfg := config.NewBuilder().
		WithPageNumber().
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		Build()
	m := maroto.New(cfg)

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
		return nil, "", fmt.Errorf("generate fy statement pdf: %w", err)
	}
	var buf bytes.Buffer
	if _, err := buf.Write(doc.GetBytes()); err != nil {
		return nil, "", fmt.Errorf("buffer fy statement pdf: %w", err)
	}
	filename := fmt.Sprintf("fy-statement-FY%d-%02d.pdf", fyStartYear, (fyStartYear+1)%100)
	return buf.Bytes(), filename, nil
}

// formatStatementAmount groups thousands and brackets deductions, the way a
// ledger a tax professional reads is written. A leading minus in a right-aligned
// column is easy to miss and easy to mistake for a typo.
func formatStatementAmount(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := fmt.Sprintf("%.2f", v)
	whole, frac, _ := strings.Cut(s, ".")
	var out []byte
	for i, d := range []byte(whole) {
		if i > 0 && (len(whole)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, d)
	}
	s = string(out) + "." + frac
	if neg {
		return "(" + s + ")"
	}
	return s
}

// sectionHeading is the one heading style in the document: small, spaced, muted,
// sitting on a hairline instead of inside a box.
func sectionHeading(m core.Maroto, label string) {
	m.AddRow(6, col.New(12).Add(text.New(label, props.Text{
		Top: 1.5, Size: 8, Style: fontstyle.Bold, Color: docMutedColor(),
	})))
	m.AddRows(hairline())
	m.AddRow(1.5, col.New(12).Add(spacer()))
}

// sectionGap is the single spacing step between sections — one rhythm, so the
// document reads as blocks of air rather than a stack of boxes.
func sectionGap(m core.Maroto) { m.AddRow(5, col.New(12).Add(spacer())) }

func addFYHeader(m core.Maroto, stmt *FYStatement) {
	// Masthead: app mark and wordmark left, document type right, on one baseline.
	m.AddRow(12,
		// No Top offset — the mark is square and any nudge pushes it past the
		// cell, which clips the bottom of the bowl rather than moving it.
		col.New(1).Add(image.NewFromBytes(brandMarkPNG(), extension.Png, props.Rect{Percent: 72})),
		col.New(5).Add(
			text.New(BrandName, props.Text{Left: 2, Top: 1, Size: 18, Style: fontstyle.Bold}),
			text.New(BrandWebsite, props.Text{Left: 2, Top: 8.5, Size: 8, Color: docMutedColor()}),
		),
		col.New(6).Add(
			text.New("ANNUAL STATEMENT", props.Text{Top: 2, Size: 13, Style: fontstyle.Bold, Align: align.Right, Color: docMutedColor()}),
			text.New(fmt.Sprintf("%s · AY %d-%02d", stmt.FYLabel, stmt.FYStartYear+1, (stmt.FYStartYear+2)%100),
				props.Text{Top: 8.5, Size: 8, Align: align.Right, Color: docMutedColor()}),
		),
	)
	m.AddRows(hairline())
	m.AddRow(3.5, col.New(12).Add(spacer()))
}

// addFYSummaryBand leads with what the document is FOR — the taxable net income
// — the same way the invoice leads with the amount paid.
func addFYSummaryBand(m core.Maroto, stmt *FYStatement) {
	curr := strOrDefault(stmt.Currency, "INR")
	m.AddRows(
		row.New(18).Add(
			col.New(6).Add(
				text.New("NET INCOME (BEFORE INCOME TAX)", props.Text{Left: 3, Top: 4, Size: 7, Style: fontstyle.Bold, Color: docMutedColor()}),
				text.New(fmt.Sprintf("%s %s", curr, formatStatementAmount(stmt.NetIncome)),
					props.Text{Left: 3, Top: 7.5, Size: 16, Style: fontstyle.Bold}),
			),
			col.New(6).Add(
				text.New("PERIOD", props.Text{Top: 4, Size: 7, Style: fontstyle.Bold, Align: align.Right, Color: docMutedColor()}),
				text.New(fmt.Sprintf("1 Apr %d – 31 Mar %d", stmt.FYStartYear, stmt.FYStartYear+1),
					props.Text{Right: 3, Top: 8, Size: 9, Align: align.Right}),
				text.New(fmt.Sprintf("%d delivered orders", stmt.OrdersCount),
					props.Text{Right: 3, Top: 12.5, Size: 8, Align: align.Right, Color: docMutedColor()}),
			),
		).WithStyle(&props.Cell{BackgroundColor: docTintColor()}),
	)
	m.AddRow(5, col.New(12).Add(spacer()))
}

func addFYParties(m core.Maroto, chef *models.ChefProfile) {
	right := []string{}
	if chef.PanNumber != "" {
		right = append(right, "PAN: "+chef.PanNumber)
	}
	if chef.GSTIN != "" {
		right = append(right, "GSTIN: "+chef.GSTIN)
	}
	if addr := joinNonEmpty([]string{chef.City, chef.State, chef.PostalCode}, ", "); addr != "" {
		right = append(right, addr)
	}

	left := []core.Component{
		text.New("PLATFORM", props.Text{Size: 7, Style: fontstyle.Bold, Color: docMutedColor()}),
		text.New(BrandLegalName, props.Text{Top: 4, Size: 10, Style: fontstyle.Bold}),
		// Names who withheld the TDS the income section deducts, which is the
		// first thing an accountant reading this asks.
		text.New("E-commerce operator u/s 194-O", props.Text{Top: 9.5, Size: 8.5, Color: docMutedColor()}),
	}
	sellerCol := []core.Component{
		text.New("SELLER (YOU)", props.Text{Size: 7, Style: fontstyle.Bold, Align: align.Right, Color: docMutedColor()}),
		text.New(strOrDefault(chef.BusinessName, "Chef"), props.Text{Top: 4, Size: 10, Style: fontstyle.Bold, Align: align.Right}),
	}
	// maroto positions text absolutely, so each optional line has to be offset
	// past the wrap of the one above it or they land on top of each other.
	top := 9.5
	for _, line := range right {
		sellerCol = append(sellerCol, text.New(line, props.Text{Top: top, Size: 8.5, Align: align.Right, Color: docMutedColor()}))
		top += partyLineMM * float64(max(1, textLines(line, 8.5, halfColumnMM)))
	}

	m.AddRow(top+1, col.New(6).Add(left...), col.New(6).Add(sellerCol...))
	m.AddRow(3, col.New(12).Add(spacer()))
}

// moneyRow renders one "label / amount" line. The currency is stated once in the
// column header, so twelve repeated "INR" prefixes don't crowd the numbers.
func moneyRow(label string, amount float64, bold bool) core.Row {
	style := fontstyle.Normal
	color := (*props.Color)(nil)
	if bold {
		style = fontstyle.Bold
	}
	return row.New(5.4).Add(
		col.New(8).Add(text.New(label, props.Text{Top: 1, Size: 9, Style: style, Color: color})),
		col.New(4).Add(text.New(formatStatementAmount(amount), props.Text{Top: 1, Size: 9, Style: style, Align: align.Right})),
	)
}

// totalRow closes a section on a tint band rather than bold text alone.
func totalRow(label string, amount float64) core.Row {
	return row.New(7).Add(
		col.New(8).Add(text.New(label, props.Text{Left: 2, Top: 2, Size: 9, Style: fontstyle.Bold})),
		col.New(4).Add(text.New(formatStatementAmount(amount), props.Text{Right: 2, Top: 2, Size: 9, Style: fontstyle.Bold, Align: align.Right})),
	).WithStyle(&props.Cell{BackgroundColor: docTintColor()})
}

// amountColumnHeader names the currency once for the rows beneath it.
func amountColumnHeader(m core.Maroto, currency string) {
	m.AddRow(5,
		col.New(8).Add(spacer()),
		col.New(4).Add(text.New(fmt.Sprintf("Amount (%s)", strOrDefault(currency, "INR")),
			props.Text{Size: 7, Align: align.Right, Color: docFaintColor()})),
	)
}

func addFYIncome(m core.Maroto, stmt *FYStatement) {
	sectionHeading(m, "INCOME FROM PLATFORM SALES")
	amountColumnHeader(m, stmt.Currency)
	gstOnCommission := stmt.CommissionCGST + stmt.CommissionSGST + stmt.CommissionIGST
	m.AddRows(
		moneyRow(fmt.Sprintf("Food revenue (%d delivered orders, net of chef-funded discounts)", stmt.OrdersCount), stmt.FoodRevenue, false),
		moneyRow("GST collected from customers on food (outward supply)", stmt.GSTCollected, false),
		moneyRow("Customer tips", stmt.Tips, false),
	)
	m.AddRows(totalRow("Gross receipts", stmt.GrossReceipts))
	m.AddRow(2, col.New(12).Add(spacer()))
	m.AddRows(
		moneyRow("Less: platform commission", -stmt.PlatformCommission, false),
		moneyRow("GST charged on commission (input tax credit eligible)", gstOnCommission, false),
		moneyRow("Less: TDS withheld u/s 194-O (credit against income tax)", -stmt.TDSWithheld, false),
	)
	m.AddRows(totalRow("Net earnings from platform", stmt.NetEarnings))
	sectionGap(m)
}

func addFYQuarters(m core.Maroto, stmt *FYStatement) {
	sectionHeading(m, "QUARTERLY BREAKDOWN")
	m.AddRows(
		row.New(7).Add(
			col.New(4).Add(text.New("Quarter", props.Text{Left: 2, Top: 2, Size: 8, Style: fontstyle.Bold, Color: docMutedColor()})),
			col.New(2).Add(text.New("Orders", props.Text{Top: 2, Size: 8, Style: fontstyle.Bold, Align: align.Right, Color: docMutedColor()})),
			col.New(2).Add(text.New("Gross", props.Text{Top: 2, Size: 8, Style: fontstyle.Bold, Align: align.Right, Color: docMutedColor()})),
			col.New(2).Add(text.New("TDS", props.Text{Top: 2, Size: 8, Style: fontstyle.Bold, Align: align.Right, Color: docMutedColor()})),
			col.New(2).Add(text.New("Net", props.Text{Right: 2, Top: 2, Size: 8, Style: fontstyle.Bold, Align: align.Right, Color: docMutedColor()})),
		).WithStyle(&props.Cell{BackgroundColor: docTintColor()}),
	)
	rows := make([]core.Row, 0, len(stmt.Quarters))
	for _, q := range stmt.Quarters {
		// A quarter with no trade is context, not data — it recedes.
		color := (*props.Color)(nil)
		if q.OrdersCount == 0 {
			color = docFaintColor()
		}
		rows = append(rows, row.New(5.6).Add(
			col.New(4).Add(text.New(q.Label, props.Text{Left: 2, Top: 1.4, Size: 9, Color: color})),
			col.New(2).Add(text.New(fmt.Sprintf("%d", q.OrdersCount), props.Text{Top: 1.4, Size: 9, Align: align.Right, Color: color})),
			col.New(2).Add(text.New(formatStatementAmount(q.Gross), props.Text{Top: 1.4, Size: 9, Align: align.Right, Color: color})),
			col.New(2).Add(text.New(formatStatementAmount(q.TDS), props.Text{Top: 1.4, Size: 9, Align: align.Right, Color: color})),
			col.New(2).Add(text.New(formatStatementAmount(q.NetPayout), props.Text{Right: 2, Top: 1.4, Size: 9, Align: align.Right, Color: color})),
		), hairline())
	}
	m.AddRows(rows...)
	sectionGap(m)
}

func addFYExpenses(m core.Maroto, stmt *FYStatement) {
	sectionHeading(m, "BUSINESS EXPENSES (SELF-DECLARED)")
	if stmt.Expenses == nil || len(stmt.Expenses.ByCategory) == 0 {
		m.AddRow(6, col.New(12).Add(text.New("No expenses recorded for this financial year.",
			props.Text{Size: 9, Style: fontstyle.Italic, Color: docMutedColor()})))
		sectionGap(m)
		return
	}
	amountColumnHeader(m, stmt.Currency)
	rows := make([]core.Row, 0, len(stmt.Expenses.ByCategory))
	for _, ct := range stmt.Expenses.ByCategory {
		label := fmt.Sprintf("%s (%d entr%s)", ExpenseCategoryLabel(ct.Category), ct.Count, pluralYIes(ct.Count))
		rows = append(rows, moneyRow(label, ct.Amount, false))
	}
	m.AddRows(rows...)
	m.AddRows(totalRow("Total expenses", stmt.TotalExpenses))
	sectionGap(m)
}

func pluralYIes(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

// addFYNetIncome restates the headline as the arithmetic that produced it, so
// the number in the band above can be checked without re-adding the document.
func addFYNetIncome(m core.Maroto, stmt *FYStatement) {
	m.AddRows(hairline())
	m.AddRow(1.5, col.New(12).Add(spacer()))
	m.AddRows(
		moneyRow("Net earnings from platform", stmt.NetEarnings, false),
		moneyRow("Less: business expenses", -stmt.TotalExpenses, false),
	)
	m.AddRows(
		row.New(10).Add(
			col.New(8).Add(text.New("NET INCOME (before income tax)",
				props.Text{Left: 2, Top: 3, Size: 11, Style: fontstyle.Bold, Color: docAccentColor()})),
			col.New(4).Add(text.New(fmt.Sprintf("%s %s", strOrDefault(stmt.Currency, "INR"), formatStatementAmount(stmt.NetIncome)),
				props.Text{Right: 2, Top: 3, Size: 11, Style: fontstyle.Bold, Align: align.Right, Color: docAccentColor()})),
		).WithStyle(&props.Cell{BackgroundColor: docTintColor()}),
	)
}

func addFYFooter(m core.Maroto) {
	m.AddRow(6, col.New(12).Add(spacer()))
	m.AddRows(hairline())
	m.AddRow(3, col.New(12).Add(spacer()))
	m.AddRow(9, col.New(12).Add(text.New(
		"Expenses in this statement are self-declared by the seller and are not verified by "+BrandName+". "+
			"This document is a working summary for GST and income-tax preparation — it is not a GST return, "+
			"Form 16A, or audited financial statement. Please consult a tax professional before filing.",
		props.Text{Size: 7, Align: align.Center, Color: docFaintColor(), Style: fontstyle.Italic},
	)))
	m.AddRow(4, col.New(12).Add(text.New(
		fmt.Sprintf("Generated %s · "+BrandLegalName, time.Now().In(istLoc).Format("02 Jan 2006 15:04 IST")),
		props.Text{Size: 7, Align: align.Center, Color: docFaintColor()},
	)))
}
