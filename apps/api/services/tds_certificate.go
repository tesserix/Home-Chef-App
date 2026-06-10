package services

// tds_certificate.go — annual TDS summary (Form 16A style) for a chef.
//
// Aggregates the TDS withheld under Section 194-O across an Indian financial
// year (1 Apr – 31 Mar), broken down by the four statutory quarters the way
// a Form 16A presents them. This is a chef-facing SUMMARY for their records —
// the legally-authoritative Form 16A is issued by the deductor via TRACES;
// the footer says so explicitly.

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

// FinancialYearWindow returns the [start, end) bounds — in UTC — of the Indian
// financial year that begins on 1 Apr of fyStartYear (e.g. fyStartYear=2025 →
// 1 Apr 2025 IST .. 1 Apr 2026 IST). Reckoned in IST.
func FinancialYearWindow(fyStartYear int) (time.Time, time.Time) {
	start := time.Date(fyStartYear, time.April, 1, 0, 0, 0, 0, istLocation)
	end := time.Date(fyStartYear+1, time.April, 1, 0, 0, 0, 0, istLocation)
	return start.UTC(), end.UTC()
}

// CurrentFinancialYearStart returns the FY start-year that contains now (IST).
// Jan–Mar belong to the FY that started the previous calendar April.
func CurrentFinancialYearStart(now time.Time) int {
	ist := now.In(istLocation)
	if ist.Month() < time.April {
		return ist.Year() - 1
	}
	return ist.Year()
}

// tdsQuarter accumulates one statutory quarter's figures.
type tdsQuarter struct {
	label string
	gross float64
	tds   float64
}

// GenerateTDSCertificatePDF renders the annual TDS summary for a chef and FY.
func GenerateTDSCertificatePDF(chefID uuid.UUID, fyStartYear int) ([]byte, string, error) {
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		return nil, "", fmt.Errorf("chef not found: %w", err)
	}

	start, end := FinancialYearWindow(fyStartYear)

	var rows []statementOrderRow
	err := database.DB.Raw(`
		SELECT o.id, o.order_number, o.delivered_at, o.subtotal, o.delivery_fee,
		       o.chef_tip, o.delivery_address_state,
		       o.chef_id, c.user_id, c.state AS chef_state
		FROM   orders o
		JOIN   chef_profiles c ON c.id = o.chef_id
		WHERE  o.chef_id       = ?
		AND    o.status        = 'delivered'
		AND    o.delivered_at >= ?
		AND    o.delivered_at  < ?
		AND    o.deleted_at    IS NULL
		ORDER  BY o.delivered_at ASC
	`, chefID, start, end).Scan(&rows).Error
	if err != nil {
		return nil, "", fmt.Errorf("load fy orders: %w", err)
	}

	// Quarters: Q1 Apr–Jun, Q2 Jul–Sep, Q3 Oct–Dec, Q4 Jan–Mar.
	quarters := []tdsQuarter{
		{label: "Q1 (Apr–Jun)"},
		{label: "Q2 (Jul–Sep)"},
		{label: "Q3 (Oct–Dec)"},
		{label: "Q4 (Jan–Mar)"},
	}
	var totalGross, totalTDS float64
	for _, r := range rows {
		e := ComputeOrderEarnings(EarningsInput{
			ItemRevenue:   r.ItemRevenue,
			DeliveryFee:   r.DeliveryFee,
			ChefTip:       r.ChefTip,
			DeliveryState: r.DeliveryState,
		}, chef.State)
		qi := financialQuarterIndex(r.CompletedAt)
		quarters[qi].gross += e.Gross
		quarters[qi].tds += e.TDS
		totalGross += e.Gross
		totalTDS += e.TDS
	}
	for i := range quarters {
		quarters[i].gross = Round2(quarters[i].gross)
		quarters[i].tds = Round2(quarters[i].tds)
	}
	totalGross = Round2(totalGross)
	totalTDS = Round2(totalTDS)

	cfg := config.NewBuilder().
		WithPageNumber().
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		Build()
	m := maroto.New(cfg)

	addTDSHeader(m, fyStartYear)
	addTDSParties(m, &chef)
	addTDSQuarters(m, quarters, totalGross, totalTDS)
	addTDSFooter(m)

	doc, err := m.Generate()
	if err != nil {
		return nil, "", fmt.Errorf("generate tds pdf: %w", err)
	}
	var buf bytes.Buffer
	if _, err := buf.Write(doc.GetBytes()); err != nil {
		return nil, "", fmt.Errorf("buffer tds pdf: %w", err)
	}
	filename := fmt.Sprintf("tds-certificate-FY%d-%02d.pdf", fyStartYear, (fyStartYear+1)%100)
	return buf.Bytes(), filename, nil
}

// financialQuarterIndex maps a delivery date (IST) to 0..3 (Q1..Q4) of the
// Indian financial year. Apr=month 4 → Q1; Jan/Feb/Mar → Q4.
func financialQuarterIndex(t time.Time) int {
	m := int(t.In(istLocation).Month())
	switch {
	case m >= 4 && m <= 6:
		return 0
	case m >= 7 && m <= 9:
		return 1
	case m >= 10 && m <= 12:
		return 2
	default: // Jan, Feb, Mar
		return 3
	}
}

func addTDSHeader(m core.Maroto, fyStartYear int) {
	m.AddRow(12,
		col.New(8).Add(text.New("TDS CERTIFICATE", props.Text{Top: 2, Size: 15, Style: fontstyle.Bold})),
		col.New(4).Add(text.New("Home Chef", props.Text{Top: 2, Size: 14, Style: fontstyle.Bold, Align: align.Right})),
	)
	m.AddRow(6,
		col.New(12).Add(text.New(
			fmt.Sprintf("Annual summary under Section 194-O · FY %d-%02d (AY %d-%02d)",
				fyStartYear, (fyStartYear+1)%100, fyStartYear+1, (fyStartYear+2)%100),
			props.Text{Size: 9})),
	)
	m.AddRow(4, col.New(12).Add(spacer()))
}

func addTDSParties(m core.Maroto, chef *models.ChefProfile) {
	deductor := []core.Component{
		text.New("DEDUCTOR", props.Text{Size: 8, Style: fontstyle.Bold, Color: &props.Color{Red: 90, Green: 90, Blue: 90}}),
		text.New("Home Chef Marketplace", props.Text{Top: 4, Size: 11, Style: fontstyle.Bold}),
		text.New("TAN appears on the official TRACES Form 16A", props.Text{Top: 9, Size: 8, Color: &props.Color{Red: 120, Green: 120, Blue: 120}, Style: fontstyle.Italic}),
	}

	deductee := []core.Component{
		text.New("DEDUCTEE (You)", props.Text{Size: 8, Style: fontstyle.Bold, Color: &props.Color{Red: 90, Green: 90, Blue: 90}}),
		text.New(strOrDefault(chef.BusinessName, "Chef"), props.Text{Top: 4, Size: 11, Style: fontstyle.Bold}),
	}
	if chef.PanNumber != "" {
		deductee = append(deductee, text.New("PAN: "+chef.PanNumber, props.Text{Top: 9, Size: 9, Style: fontstyle.Bold}))
	} else {
		deductee = append(deductee, text.New("PAN: not on file — add it in your profile", props.Text{Top: 9, Size: 8, Color: &props.Color{Red: 178, Green: 43, Blue: 14}}))
	}
	addr := joinNonEmpty([]string{chef.City, chef.State, chef.PostalCode}, ", ")
	if addr != "" {
		deductee = append(deductee, text.New(addr, props.Text{Top: 14, Size: 9}))
	}

	m.AddRow(26,
		col.New(6).Add(deductor...),
		col.New(6).Add(deductee...),
	)
	m.AddRow(4, col.New(12).Add(spacer()))
}

func addTDSQuarters(m core.Maroto, quarters []tdsQuarter, totalGross, totalTDS float64) {
	m.AddRow(7,
		col.New(5).Add(text.New("Quarter", props.Text{Size: 9, Style: fontstyle.Bold})),
		col.New(4).Add(text.New("Amount paid", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
		col.New(3).Add(text.New("TDS deducted", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
	)
	rows := make([]core.Row, 0, len(quarters)+1)
	for _, q := range quarters {
		rows = append(rows, row.New(6).Add(
			col.New(5).Add(text.New(q.label, props.Text{Size: 9})),
			col.New(4).Add(text.New(fmt.Sprintf("INR %.2f", q.gross), props.Text{Size: 9, Align: align.Right})),
			col.New(3).Add(text.New(fmt.Sprintf("INR %.2f", q.tds), props.Text{Size: 9, Align: align.Right})),
		))
	}
	rows = append(rows, row.New(7).Add(
		col.New(5).Add(text.New("TOTAL", props.Text{Size: 9, Style: fontstyle.Bold})),
		col.New(4).Add(text.New(fmt.Sprintf("INR %.2f", totalGross), props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
		col.New(3).Add(text.New(fmt.Sprintf("INR %.2f", totalTDS), props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
	))
	m.AddRows(rows...)
}

func addTDSFooter(m core.Maroto) {
	m.AddRow(14, col.New(12).Add(spacer()))
	m.AddRow(5, col.New(12).Add(text.New(
		"This is a summary of TDS withheld for your records. The legally-authoritative Form 16A is "+
			"issued quarterly by the deductor and can be downloaded from the TRACES portal (incometax.gov.in). "+
			"TDS is deposited against the PAN shown above under Section 194-O.",
		props.Text{Size: 7, Align: align.Center, Color: &props.Color{Red: 120, Green: 120, Blue: 120}, Style: fontstyle.Italic},
	)))
	m.AddRow(4, col.New(12).Add(text.New(
		fmt.Sprintf("Generated %s · Home Chef Marketplace", time.Now().In(istLocation).Format("02 Jan 2006 15:04 IST")),
		props.Text{Size: 7, Align: align.Center, Color: &props.Color{Red: 120, Green: 120, Blue: 120}},
	)))
}
