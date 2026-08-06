package services

import (
	"bytes"
	"fmt"
	"math"
	"strings"
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

// GenerateOrderInvoicePDF renders a GSTIN-formatted invoice PDF for a
// delivered order. The output is a tax-compliant document with the
// chef's GSTIN + FSSAI, line items grouped by HSN code, and the tax
// breakdown — what the customer needs for input-tax-credit claims.
//
// Returns the PDF as a byte buffer ready to be streamed back over
// HTTP or uploaded to GCS for archival.
func GenerateOrderInvoicePDF(orderID uuid.UUID) ([]byte, string, error) {
	var order models.Order
	if err := database.DB.Preload("Items").Preload("Chef").Preload("Chef.User").Preload("Customer").
		First(&order, orderID).Error; err != nil {
		return nil, "", fmt.Errorf("order not found: %w", err)
	}

	// Resolve any item-level HSN overrides — fall back to the chef's
	// default SAC for restaurant services if a line is missing one.
	itemMeta := loadInvoiceItemMeta(order.Items)

	cfg := config.NewBuilder().
		WithPageNumber().
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		Build()
	m := maroto.New(cfg)

	addInvoiceHeader(m, &order)
	addInvoiceParties(m, &order)
	addInvoiceItems(m, &order, itemMeta)
	addInvoiceTotals(m, &order)
	addInvoiceFooter(m, &order)

	doc, err := m.Generate()
	if err != nil {
		return nil, "", fmt.Errorf("generate pdf: %w", err)
	}
	var buf bytes.Buffer
	if _, err := buf.Write(doc.GetBytes()); err != nil {
		return nil, "", fmt.Errorf("buffer pdf: %w", err)
	}
	docKind := "invoice"
	if !orderInvoiceIsTaxInvoice(&order) {
		docKind = "receipt"
	}
	filename := fmt.Sprintf("%s-%s.pdf", docKind, order.OrderNumber)
	return buf.Bytes(), filename, nil
}

func loadInvoiceItemMeta(items []models.OrderItem) map[uuid.UUID]string {
	if len(items) == 0 {
		return map[uuid.UUID]string{}
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.MenuItemID)
	}
	var menuItems []models.MenuItem
	_ = database.DB.Where("id IN ?", ids).Find(&menuItems).Error
	meta := make(map[uuid.UUID]string, len(menuItems))
	for _, mi := range menuItems {
		hsn := mi.HSN
		if hsn == "" {
			hsn = "996331"
		}
		meta[mi.ID] = hsn
	}
	return meta
}

// orderInvoiceIsTaxInvoice reports whether the document is a formal GSTIN TAX
// INVOICE (a completed, delivered sale) versus a PAYMENT RECEIPT. A tax invoice
// must not be issued for an order that was refunded or never delivered — that
// would claim a taxable sale that did not complete. Both documents are otherwise
// identical (the refund line already shows on a refunded order).
func orderInvoiceIsTaxInvoice(order *models.Order) bool {
	return order.Status == models.OrderStatusDelivered && order.RefundAmount <= 0
}

func addInvoiceHeader(m core.Maroto, order *models.Order) {
	title := "TAX INVOICE"
	if !orderInvoiceIsTaxInvoice(order) {
		title = "PAYMENT RECEIPT"
	}
	// Masthead: brand lockup left, document type right, both on one baseline —
	// the same hierarchy as the in-app receipt so the two read as one document.
	m.AddRow(12,
		col.New(6).Add(
			text.New(BrandName, props.Text{Top: 1, Size: 18, Style: fontstyle.Bold}),
			text.New(BrandWebsite, props.Text{Top: 8, Size: 8, Color: docMutedColor()}),
		),
		col.New(6).Add(
			text.New(title, props.Text{Top: 2, Size: 13, Style: fontstyle.Bold, Align: align.Right, Color: docMutedColor()}),
		),
	)
	m.AddRow(2, col.New(12).Add(spacer()))

	// The amount is what the document is FOR, so it leads; the reference and
	// date sit beside it rather than above it.
	curr := strOrDefault(order.Currency, "INR")
	m.AddRows(
		// Order numbers carry the kitchen's name, so they run to 40 characters —
		// on one line beside the amount they wrapped back over it. The reference
		// gets its own line under a small label, at 8pt, and the amount column
		// narrows to leave it room.
		row.New(16).Add(
			col.New(5).Add(
				text.New("AMOUNT PAID", props.Text{Top: 3.5, Size: 7, Style: fontstyle.Bold, Color: docMutedColor()}),
				text.New(fmt.Sprintf("%s %.2f", curr, order.ToResponse().Total), props.Text{Top: 6.5, Size: 16, Style: fontstyle.Bold}),
			),
			col.New(7).Add(
				text.New("INVOICE #", props.Text{Top: 3.5, Size: 7, Style: fontstyle.Bold, Align: align.Right, Color: docMutedColor()}),
				text.New(order.OrderNumber, props.Text{Top: 7, Size: 8, Align: align.Right}),
				text.New(order.CreatedAt.Format("02 Jan 2006"), props.Text{Top: 11.5, Size: 8, Align: align.Right, Color: docMutedColor()}),
			),
		).WithStyle(&props.Cell{BackgroundColor: docTintColor()}),
	)
	m.AddRow(6, col.New(12).Add(spacer()))
}

// One palette for every generated document, so an invoice, a statement and a
// TDS certificate never drift into three greys.
func docMutedColor() *props.Color  { return &props.Color{Red: 90, Green: 90, Blue: 90} }
func docFaintColor() *props.Color  { return &props.Color{Red: 120, Green: 120, Blue: 120} }
func docTintColor() *props.Color   { return &props.Color{Red: 246, Green: 244, Blue: 241} }
func docRuleColor() *props.Color   { return &props.Color{Red: 222, Green: 219, Blue: 214} }
func docAccentColor() *props.Color { return &props.Color{Red: 194, Green: 65, Blue: 12} }

// hairline is a full-width rule — the document's only separator, so sections are
// spaced apart rather than boxed in.
func hairline() core.Row {
	return row.New(0.4).Add(col.New(12).Add(spacer())).WithStyle(&props.Cell{BackgroundColor: docRuleColor()})
}

// Half of the A4 content width — the party columns are 6/12 each.
const halfColumnMM = 90.0

// Line box for the 9pt body lines in a party block.
const partyLineMM = 4.6

// textLines estimates how many lines a string occupies in a column of widthMM at
// the given font size. maroto positions text absolutely and reports no measured
// height, so a stacked block has to predict the wrap itself or the next line
// lands on top of the previous one.
func textLines(s string, fontSize, widthMM float64) int {
	if strings.TrimSpace(s) == "" {
		return 0
	}
	// Helvetica averages ~0.5em per character; pt→mm is 25.4/72.
	perChar := fontSize * 0.5 * 25.4 / 72
	perLine := math.Max(1, math.Floor(widthMM/perChar))
	return int(math.Ceil(float64(len([]rune(s))) / perLine))
}

// chefPartyLines is the SOLD BY block's optional lines in print order.
func chefPartyLines(chef *models.ChefProfile) []string {
	var lines []string
	if owner := strings.TrimSpace(chef.User.FirstName + " " + chef.User.LastName); owner != "" {
		lines = append(lines, "Chef: "+owner)
	}
	if addr := joinNonEmpty([]string{chef.AddressLine1, chef.AddressLine2, chef.City, chef.State, chef.PostalCode}, ", "); addr != "" {
		lines = append(lines, addr)
	}
	if chef.GSTIN != "" {
		lines = append(lines, "GSTIN: "+chef.GSTIN)
	}
	if chef.FSSAILicenseNumber != "" {
		lines = append(lines, "FSSAI: "+chef.FSSAILicenseNumber)
	}
	return lines
}

// chefPartyBlockHeight is the room the SOLD BY block needs once its lines wrap.
func chefPartyBlockHeight(chef *models.ChefProfile) float64 {
	top := 9.0
	for _, line := range chefPartyLines(chef) {
		top += partyLineMM * float64(textLines(line, 9, halfColumnMM))
	}
	return top + 2
}

func addInvoiceParties(m core.Maroto, order *models.Order) {
	chef := order.Chef
	cust := order.Customer

	chefBlock := []core.Component{
		text.New("SOLD BY", props.Text{Size: 7, Style: fontstyle.Bold, Color: docMutedColor()}),
		text.New(chef.BusinessName, props.Text{Top: 4, Size: 11, Style: fontstyle.Bold}),
	}
	// Running vertical offset, advanced by the WRAPPED height of each line so
	// GSTIN never prints on top of a two-line address.
	top := 9.0
	for _, line := range chefPartyLines(&chef) {
		style := fontstyle.Normal
		if strings.HasPrefix(line, "GSTIN: ") {
			style = fontstyle.Bold
		}
		chefBlock = append(chefBlock, text.New(line, props.Text{Top: top, Size: 9, Style: style}))
		top += partyLineMM * float64(textLines(line, 9, halfColumnMM))
	}

	custBlock := []core.Component{
		text.New("BILL TO", props.Text{Size: 7, Style: fontstyle.Bold, Color: docMutedColor()}),
		text.New(strOrDefault(cust.FirstName+" "+cust.LastName, "Customer"), props.Text{Top: 4, Size: 11, Style: fontstyle.Bold}),
	}
	custTop := 9.0
	custAddr := joinNonEmpty([]string{order.DeliveryAddressLine1, order.DeliveryAddressLine2, order.DeliveryAddressCity, order.DeliveryAddressState, order.DeliveryAddressPostalCode}, ", ")
	if custAddr != "" {
		custBlock = append(custBlock, text.New(custAddr, props.Text{Top: custTop, Size: 9}))
		custTop += partyLineMM * float64(textLines(custAddr, 9, halfColumnMM))
	}

	m.AddRow(math.Max(math.Max(top, custTop)+2, 26),
		col.New(6).Add(chefBlock...),
		col.New(6).Add(custBlock...),
	)
	m.AddRows(hairline())
	m.AddRow(5, col.New(12).Add(spacer()))
}

func addInvoiceItems(m core.Maroto, order *models.Order, hsnMeta map[uuid.UUID]string) {
	m.AddRows(
		row.New(8).Add(
			col.New(5).Add(text.New("ITEM", props.Text{Top: 2.4, Left: 1.5, Size: 7, Style: fontstyle.Bold, Color: docMutedColor()})),
			col.New(2).Add(text.New("HSN/SAC", props.Text{Top: 2.4, Size: 7, Style: fontstyle.Bold, Align: align.Center, Color: docMutedColor()})),
			col.New(1).Add(text.New("QTY", props.Text{Top: 2.4, Size: 7, Style: fontstyle.Bold, Align: align.Right, Color: docMutedColor()})),
			col.New(2).Add(text.New("UNIT", props.Text{Top: 2.4, Size: 7, Style: fontstyle.Bold, Align: align.Right, Color: docMutedColor()})),
			col.New(2).Add(text.New("AMOUNT", props.Text{Top: 2.4, Right: 1.5, Size: 7, Style: fontstyle.Bold, Align: align.Right, Color: docMutedColor()})),
		).WithStyle(&props.Cell{BackgroundColor: docTintColor()}),
	)
	rows := make([]core.Row, 0, len(order.Items))
	for _, it := range order.Items {
		if it.IsCancelled {
			continue
		}
		hsn := hsnMeta[it.MenuItemID]
		if hsn == "" {
			hsn = "996331"
		}
		rows = append(rows, row.New(7).Add(
			col.New(5).Add(text.New(InvoiceLineName(it), props.Text{Top: 1.8, Left: 1.5, Size: 9})),
			col.New(2).Add(text.New(hsn, props.Text{Top: 1.8, Size: 8, Align: align.Center, Color: docMutedColor()})),
			col.New(1).Add(text.New(fmt.Sprintf("%d", it.Quantity), props.Text{Top: 1.8, Size: 9, Align: align.Right})),
			col.New(2).Add(text.New(fmt.Sprintf("%.2f", it.Price), props.Text{Top: 1.8, Size: 9, Align: align.Right})),
			col.New(2).Add(text.New(fmt.Sprintf("%.2f", it.Subtotal), props.Text{Top: 1.8, Right: 1.5, Size: 9, Align: align.Right})),
		), hairline())
	}
	m.AddRows(rows...)
	m.AddRow(4, col.New(12).Add(spacer()))
}

func addInvoiceTotals(m core.Maroto, order *models.Order) {
	curr := order.Currency
	if curr == "" {
		curr = "INR"
	}
	totalRow := func(label string, amount float64, bold bool) core.Row {
		style := fontstyle.Normal
		labelColor := docMutedColor()
		if bold {
			style = fontstyle.Bold
			labelColor = nil
		}
		return row.New(5).Add(
			col.New(6).Add(spacer()),
			col.New(4).Add(text.New(label, props.Text{Size: 9, Style: style, Align: align.Right, Color: labelColor})),
			col.New(2).Add(text.New(fmt.Sprintf("%s %.2f", curr, amount), props.Text{Right: 1.5, Size: 9, Style: style, Align: align.Right})),
		)
	}

	// Same breakdown the app and the web page render (models/pricing.go), so the
	// downloadable document and the in-app receipt can never disagree — including
	// the GST split, which this file resolved through the states table while the
	// app compared the two spellings as raw strings.
	p := order.ToResponse()

	rows := []core.Row{totalRow("Subtotal", p.Subtotal, false)}
	if p.DeliveryFee > 0 {
		rows = append(rows, totalRow("Delivery", p.DeliveryFee, false))
	}
	if p.PlatformFee > 0 {
		rows = append(rows, totalRow("Platform fee", p.PlatformFee, false))
	}
	// The per-RATE breakdown, not the summary the apps show: this is the tax
	// invoice, and Rule 46 wants the rate and amount of each head stated.
	for _, line := range p.TaxBreakdown {
		rows = append(rows, totalRow(line.Label, line.Amount, false))
	}
	if p.Discount > 0 {
		rows = append(rows, totalRow("Discount", -p.Discount, false))
	}
	// The tip was charged to the customer and belongs on the invoice; without it a
	// tipped order's rows summed to less than its own TOTAL.
	if p.Tip > 0 {
		rows = append(rows, totalRow("Tip", p.Tip, false))
	}
	if p.Rounding != 0 {
		rows = append(rows, totalRow("Rounding", p.Rounding, false))
	}
	rows = append(rows,
		row.New(3).Add(col.New(6).Add(spacer()), col.New(6).Add(spacer())),
		row.New(0.4).Add(col.New(6).Add(spacer()), col.New(6).Add(spacer())).WithStyle(&props.Cell{BackgroundColor: docRuleColor()}),
		totalRow("TOTAL", p.Total, true),
	)
	if order.RefundAmount > 0 {
		rows = append(rows,
			row.New(5).Add(
				col.New(6).Add(spacer()),
				col.New(4).Add(text.New("Refunded", props.Text{Size: 9, Style: fontstyle.Italic, Align: align.Right, Color: docAccentColor()})),
				col.New(2).Add(text.New(fmt.Sprintf("-%s %.2f", curr, order.RefundAmount), props.Text{Right: 1.5, Size: 9, Style: fontstyle.Italic, Align: align.Right, Color: docAccentColor()})),
			),
		)
	}
	m.AddRows(rows...)
}

func addInvoiceFooter(m core.Maroto, order *models.Order) {
	m.AddRow(15, col.New(12).Add(spacer()))
	// A receipt (refunded / not delivered) must say so — it is not a tax invoice.
	if !orderInvoiceIsTaxInvoice(order) {
		m.AddRow(5,
			col.New(12).Add(text.New(
				"This is a payment receipt, not a tax invoice.",
				props.Text{Size: 7, Align: align.Center, Color: docFaintColor(), Style: fontstyle.Bold},
			)),
		)
	}
	m.AddRow(5,
		col.New(12).Add(text.New(
			"This is a computer-generated document and does not require a physical signature.",
			props.Text{Size: 7, Align: align.Center, Color: docFaintColor(), Style: fontstyle.Italic},
		)),
	)
	m.AddRow(4,
		col.New(12).Add(text.New(
			fmt.Sprintf("Generated %s · "+BrandLegalName, time.Now().UTC().Format("02 Jan 2006 15:04 MST")),
			props.Text{Size: 7, Align: align.Center, Color: docFaintColor()},
		)),
	)
}

func joinNonEmpty(parts []string, sep string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out == "" {
			out = p
		} else {
			out += sep + p
		}
	}
	return out
}

func strOrDefault(s, def string) string {
	if s == "" || s == " " {
		return def
	}
	return s
}

// spacer returns an empty text component for layout-only rows.
func spacer() core.Component {
	return text.New("", props.Text{Size: 1})
}
