package services

// statement_pdf.go — renders a WeeklyStatement as a settlement PDF.
//
// The frozen totals come from the persisted statement row; the per-order
// line table is re-derived from the (immutable, post-delivery) orders in
// the week window — they always reconcile to the frozen totals. Shares the
// maroto layout helpers (spacer / joinNonEmpty / strOrDefault) with
// invoice_pdf.go.

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

// GenerateWeeklyStatementPDF renders the settlement statement identified by
// statementID. Returns the PDF bytes ready to stream and a download filename.
func GenerateWeeklyStatementPDF(statementID uuid.UUID) ([]byte, string, error) {
	var stmt models.WeeklyStatement
	if err := database.DB.First(&stmt, statementID).Error; err != nil {
		return nil, "", fmt.Errorf("statement not found: %w", err)
	}

	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", stmt.ChefID).Error; err != nil {
		return nil, "", fmt.Errorf("chef not found: %w", err)
	}

	rows, err := loadStatementOrderRows(stmt.WeekStart, stmt.WeekEnd)
	if err != nil {
		return nil, "", fmt.Errorf("load statement orders: %w", err)
	}
	lines := make([]OrderEarnings, 0, len(rows))
	for _, r := range rows {
		if r.ChefID != stmt.ChefID {
			continue
		}
		lines = append(lines, ComputeOrderEarnings(EarningsInput{
			OrderID:       r.OrderID,
			OrderNumber:   r.OrderNumber,
			CompletedAt:   r.CompletedAt,
			ItemRevenue:   r.ItemRevenue,
			DeliveryFee:   r.DeliveryFee,
			ChefTip:       r.ChefTip,
			DeliveryState: r.DeliveryState,
		}, chef.State))
	}

	cfg := config.NewBuilder().
		WithPageNumber().
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		Build()
	m := maroto.New(cfg)

	addStatementHeader(m, &stmt)
	addStatementChef(m, &chef, &stmt)
	addStatementSummary(m, &stmt)
	addStatementOrders(m, lines)
	addStatementFooter(m)

	doc, err := m.Generate()
	if err != nil {
		return nil, "", fmt.Errorf("generate statement pdf: %w", err)
	}
	var buf bytes.Buffer
	if _, err := buf.Write(doc.GetBytes()); err != nil {
		return nil, "", fmt.Errorf("buffer statement pdf: %w", err)
	}
	filename := fmt.Sprintf("statement-%s.pdf", stmt.WeekStart.In(istLocation).Format("2006-01-02"))
	return buf.Bytes(), filename, nil
}

func addStatementHeader(m core.Maroto, stmt *models.WeeklyStatement) {
	periodEnd := stmt.WeekEnd.In(istLocation).AddDate(0, 0, -1)
	m.AddRow(12,
		col.New(8).Add(
			text.New("WEEKLY SETTLEMENT STATEMENT", props.Text{Top: 2, Size: 15, Style: fontstyle.Bold}),
		),
		col.New(4).Add(
			text.New("Home Chef", props.Text{Top: 2, Size: 14, Style: fontstyle.Bold, Align: align.Right}),
		),
	)
	m.AddRow(6,
		col.New(8).Add(
			text.New(fmt.Sprintf("Period: %s – %s",
				stmt.WeekStart.In(istLocation).Format("02 Jan 2006"),
				periodEnd.Format("02 Jan 2006")),
				props.Text{Size: 9}),
			text.New(fmt.Sprintf("Issued: %s", stmt.CreatedAt.In(istLocation).Format("02 Jan 2006")),
				props.Text{Top: 4, Size: 9}),
		),
		col.New(4).Add(
			text.New("www.homechef.app", props.Text{Size: 8, Align: align.Right, Color: &props.Color{Red: 90, Green: 90, Blue: 90}}),
		),
	)
	m.AddRow(4, col.New(12).Add(spacer()))
}

func addStatementChef(m core.Maroto, chef *models.ChefProfile, stmt *models.WeeklyStatement) {
	block := []core.Component{
		text.New("SETTLED TO", props.Text{Size: 8, Style: fontstyle.Bold, Color: &props.Color{Red: 90, Green: 90, Blue: 90}}),
		text.New(strOrDefault(chef.BusinessName, "Chef"), props.Text{Top: 4, Size: 11, Style: fontstyle.Bold}),
	}
	addr := joinNonEmpty([]string{chef.AddressLine1, chef.AddressLine2, chef.City, chef.State, chef.PostalCode}, ", ")
	if addr != "" {
		block = append(block, text.New(addr, props.Text{Top: 9, Size: 9}))
	}
	if chef.GSTIN != "" {
		block = append(block, text.New("GSTIN: "+chef.GSTIN, props.Text{Top: 14, Size: 9, Style: fontstyle.Bold}))
	}
	m.AddRow(22,
		col.New(8).Add(block...),
		col.New(4).Add(
			text.New("ORDERS", props.Text{Size: 8, Align: align.Right, Color: &props.Color{Red: 90, Green: 90, Blue: 90}}),
			text.New(fmt.Sprintf("%d", stmt.OrdersCount), props.Text{Top: 4, Size: 18, Style: fontstyle.Bold, Align: align.Right}),
		),
	)
	m.AddRow(4, col.New(12).Add(spacer()))
}

func addStatementSummary(m core.Maroto, stmt *models.WeeklyStatement) {
	curr := stmt.Currency
	if curr == "" {
		curr = "INR"
	}
	line := func(label string, amount float64, bold, negative bool) core.Row {
		style := fontstyle.Normal
		if bold {
			style = fontstyle.Bold
		}
		amtStr := fmt.Sprintf("%s %.2f", curr, amount)
		if negative {
			amtStr = fmt.Sprintf("-%s %.2f", curr, amount)
		}
		return row.New(6).Add(
			col.New(8).Add(text.New(label, props.Text{Size: 9, Style: style})),
			col.New(4).Add(text.New(amtStr, props.Text{Size: 9, Style: style, Align: align.Right})),
		)
	}

	m.AddRow(6, col.New(12).Add(text.New("SETTLEMENT SUMMARY", props.Text{Size: 8, Style: fontstyle.Bold, Color: &props.Color{Red: 90, Green: 90, Blue: 90}})))
	rows := []core.Row{
		line("Gross revenue", stmt.GrossRevenue, false, false),
		line(fmt.Sprintf("Platform commission (%.0f%%)", RateCommission*100), stmt.PlatformCommission, false, true),
	}
	if stmt.IGST > 0 {
		rows = append(rows, line(fmt.Sprintf("GST · IGST (%.0f%%)", RateGST*100), stmt.IGST, false, true))
	} else {
		rows = append(rows, line(fmt.Sprintf("GST · CGST+SGST (%.0f%%)", RateGST*100), stmt.CGST+stmt.SGST, false, true))
	}
	rows = append(rows,
		line(fmt.Sprintf("TDS u/s 194-O (%.0f%%)", RateTDS*100), stmt.TDS, false, true),
		line("NET PAYOUT", stmt.NetPayout, true, false),
	)
	m.AddRows(rows...)
	m.AddRow(4, col.New(12).Add(spacer()))
}

func addStatementOrders(m core.Maroto, lines []OrderEarnings) {
	m.AddRow(7,
		col.New(4).Add(text.New("Order", props.Text{Size: 9, Style: fontstyle.Bold})),
		col.New(2).Add(text.New("Date", props.Text{Size: 9, Style: fontstyle.Bold})),
		col.New(2).Add(text.New("Gross", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
		col.New(2).Add(text.New("Deductions", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
		col.New(2).Add(text.New("Net", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
	)
	rows := make([]core.Row, 0, len(lines))
	for _, e := range lines {
		deductions := Round2(e.PlatformCommission + e.TDS)
		rows = append(rows, row.New(6).Add(
			col.New(4).Add(text.New("#"+e.OrderNumber, props.Text{Size: 9})),
			col.New(2).Add(text.New(e.CompletedAt.In(istLocation).Format("02 Jan"), props.Text{Size: 9})),
			col.New(2).Add(text.New(fmt.Sprintf("%.2f", e.Gross), props.Text{Size: 9, Align: align.Right})),
			col.New(2).Add(text.New(fmt.Sprintf("-%.2f", deductions), props.Text{Size: 9, Align: align.Right})),
			col.New(2).Add(text.New(fmt.Sprintf("%.2f", e.NetPayout), props.Text{Size: 9, Align: align.Right, Style: fontstyle.Bold})),
		))
	}
	m.AddRows(rows...)
}

func addStatementFooter(m core.Maroto) {
	m.AddRow(12, col.New(12).Add(spacer()))
	m.AddRow(5, col.New(12).Add(text.New(
		"GST shown is levied on the platform commission and is not deducted from your payout. "+
			"TDS is deposited against your PAN under Section 194-O.",
		props.Text{Size: 7, Align: align.Center, Color: &props.Color{Red: 120, Green: 120, Blue: 120}, Style: fontstyle.Italic},
	)))
	m.AddRow(4, col.New(12).Add(text.New(
		fmt.Sprintf("Generated %s · Home Chef Marketplace", time.Now().In(istLocation).Format("02 Jan 2006 15:04 IST")),
		props.Text{Size: 7, Align: align.Center, Color: &props.Color{Red: 120, Green: 120, Blue: 120}},
	)))
}
