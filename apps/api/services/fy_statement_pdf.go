package services

// fy_statement_pdf.go — renders the annual income & expense statement PDF a
// chef downloads at tax time (GST + ITR working papers). Numbers come from
// ComputeFYStatement; this file is layout only.

import (
	"bytes"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
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

func addFYHeader(m core.Maroto, stmt *FYStatement) {
	m.AddRow(12,
		col.New(8).Add(text.New("ANNUAL STATEMENT", props.Text{Top: 2, Size: 15, Style: fontstyle.Bold})),
		col.New(4).Add(text.New("Home Chef", props.Text{Top: 2, Size: 14, Style: fontstyle.Bold, Align: align.Right})),
	)
	m.AddRow(6,
		col.New(12).Add(text.New(
			fmt.Sprintf("Income & expense statement · %s (AY %d-%02d) · 1 Apr – 31 Mar",
				stmt.FYLabel, stmt.FYStartYear+1, (stmt.FYStartYear+2)%100),
			props.Text{Size: 9})),
	)
	m.AddRow(4, col.New(12).Add(spacer()))
}

func addFYParties(m core.Maroto, chef *models.ChefProfile) {
	left := []core.Component{
		text.New("PLATFORM", props.Text{Size: 8, Style: fontstyle.Bold, Color: &props.Color{Red: 90, Green: 90, Blue: 90}}),
		text.New("Home Chef Marketplace", props.Text{Top: 4, Size: 11, Style: fontstyle.Bold}),
	}
	right := []core.Component{
		text.New("SELLER (You)", props.Text{Size: 8, Style: fontstyle.Bold, Color: &props.Color{Red: 90, Green: 90, Blue: 90}}),
		text.New(strOrDefault(chef.BusinessName, "Chef"), props.Text{Top: 4, Size: 11, Style: fontstyle.Bold}),
	}
	line := 9.0
	if chef.PanNumber != "" {
		right = append(right, text.New("PAN: "+chef.PanNumber, props.Text{Top: line, Size: 9}))
		line += 5
	}
	if chef.GSTIN != "" {
		right = append(right, text.New("GSTIN: "+chef.GSTIN, props.Text{Top: line, Size: 9}))
		line += 5
	}
	if addr := joinNonEmpty([]string{chef.City, chef.State, chef.PostalCode}, ", "); addr != "" {
		right = append(right, text.New(addr, props.Text{Top: line, Size: 9}))
	}
	m.AddRow(30,
		col.New(6).Add(left...),
		col.New(6).Add(right...),
	)
	m.AddRow(4, col.New(12).Add(spacer()))
}

// moneyRow renders one "label / amount" line.
func moneyRow(label string, amount float64, bold bool) core.Row {
	style := fontstyle.Normal
	if bold {
		style = fontstyle.Bold
	}
	return row.New(6).Add(
		col.New(8).Add(text.New(label, props.Text{Size: 9, Style: style})),
		col.New(4).Add(text.New(fmt.Sprintf("INR %.2f", amount), props.Text{Size: 9, Style: style, Align: align.Right})),
	)
}

func addFYIncome(m core.Maroto, stmt *FYStatement) {
	m.AddRow(8, col.New(12).Add(text.New("INCOME FROM PLATFORM SALES", props.Text{Size: 10, Style: fontstyle.Bold})))
	rows := []core.Row{
		moneyRow(fmt.Sprintf("Food revenue (%d delivered orders, net of chef-funded discounts)", stmt.OrdersCount), stmt.FoodRevenue, false),
		moneyRow("GST collected from customers on food (outward supply)", stmt.GSTCollected, false),
		moneyRow("Customer tips", stmt.Tips, false),
		moneyRow("Gross receipts", stmt.GrossReceipts, true),
		moneyRow("Less: platform commission", -stmt.PlatformCommission, false),
	}
	gstOnCommission := stmt.CommissionCGST + stmt.CommissionSGST + stmt.CommissionIGST
	rows = append(rows,
		moneyRow("GST charged on commission (input tax credit eligible)", gstOnCommission, false),
		moneyRow("Less: TDS withheld u/s 194-O (credit against income tax)", -stmt.TDSWithheld, false),
		moneyRow("Net earnings from platform", stmt.NetEarnings, true),
	)
	m.AddRows(rows...)
	m.AddRow(4, col.New(12).Add(spacer()))
}

func addFYQuarters(m core.Maroto, stmt *FYStatement) {
	m.AddRow(8, col.New(12).Add(text.New("QUARTERLY BREAKDOWN", props.Text{Size: 10, Style: fontstyle.Bold})))
	m.AddRow(7,
		col.New(4).Add(text.New("Quarter", props.Text{Size: 9, Style: fontstyle.Bold})),
		col.New(2).Add(text.New("Orders", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
		col.New(2).Add(text.New("Gross", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
		col.New(2).Add(text.New("TDS", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
		col.New(2).Add(text.New("Net", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
	)
	rows := make([]core.Row, 0, len(stmt.Quarters))
	for _, q := range stmt.Quarters {
		rows = append(rows, row.New(6).Add(
			col.New(4).Add(text.New(q.Label, props.Text{Size: 9})),
			col.New(2).Add(text.New(fmt.Sprintf("%d", q.OrdersCount), props.Text{Size: 9, Align: align.Right})),
			col.New(2).Add(text.New(fmt.Sprintf("%.2f", q.Gross), props.Text{Size: 9, Align: align.Right})),
			col.New(2).Add(text.New(fmt.Sprintf("%.2f", q.TDS), props.Text{Size: 9, Align: align.Right})),
			col.New(2).Add(text.New(fmt.Sprintf("%.2f", q.NetPayout), props.Text{Size: 9, Align: align.Right})),
		))
	}
	m.AddRows(rows...)
	m.AddRow(4, col.New(12).Add(spacer()))
}

func addFYExpenses(m core.Maroto, stmt *FYStatement) {
	m.AddRow(8, col.New(12).Add(text.New("BUSINESS EXPENSES (SELF-DECLARED)", props.Text{Size: 10, Style: fontstyle.Bold})))
	if stmt.Expenses == nil || len(stmt.Expenses.ByCategory) == 0 {
		m.AddRow(6, col.New(12).Add(text.New("No expenses recorded for this financial year.", props.Text{Size: 9, Style: fontstyle.Italic})))
		m.AddRow(4, col.New(12).Add(spacer()))
		return
	}
	rows := make([]core.Row, 0, len(stmt.Expenses.ByCategory)+1)
	for _, ct := range stmt.Expenses.ByCategory {
		label := fmt.Sprintf("%s (%d entr%s)", ExpenseCategoryLabel(ct.Category), ct.Count, pluralYIes(ct.Count))
		rows = append(rows, moneyRow(label, ct.Amount, false))
	}
	rows = append(rows, moneyRow("Total expenses", stmt.TotalExpenses, true))
	m.AddRows(rows...)
	m.AddRow(4, col.New(12).Add(spacer()))
}

func pluralYIes(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func addFYNetIncome(m core.Maroto, stmt *FYStatement) {
	m.AddRow(9,
		col.New(8).Add(text.New("NET INCOME (before income tax)", props.Text{Top: 1, Size: 11, Style: fontstyle.Bold})),
		col.New(4).Add(text.New(fmt.Sprintf("INR %.2f", stmt.NetIncome), props.Text{Top: 1, Size: 11, Style: fontstyle.Bold, Align: align.Right})),
	)
}

func addFYFooter(m core.Maroto) {
	m.AddRow(12, col.New(12).Add(spacer()))
	m.AddRow(5, col.New(12).Add(text.New(
		"Expenses in this statement are self-declared by the seller and are not verified by Home Chef. "+
			"This document is a working summary for GST and income-tax preparation — it is not a GST return, "+
			"Form 16A, or audited financial statement. Please consult a tax professional before filing.",
		props.Text{Size: 7, Align: align.Center, Color: &props.Color{Red: 120, Green: 120, Blue: 120}, Style: fontstyle.Italic},
	)))
	m.AddRow(4, col.New(12).Add(text.New(
		fmt.Sprintf("Generated %s · Home Chef Marketplace", time.Now().In(istLoc).Format("02 Jan 2006 15:04 IST")),
		props.Text{Size: 7, Align: align.Center, Color: &props.Color{Red: 120, Green: 120, Blue: 120}},
	)))
}
