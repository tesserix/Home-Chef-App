package services

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
	m.AddRow(12,
		col.New(8).Add(
			text.New(title, props.Text{
				Top: 2, Size: 16, Style: fontstyle.Bold,
			}),
		),
		col.New(4).Add(
			text.New("Home Chef", props.Text{
				Top: 2, Size: 14, Style: fontstyle.Bold, Align: align.Right,
			}),
		),
	)
	m.AddRow(6,
		col.New(8).Add(
			text.New(fmt.Sprintf("Invoice #: %s", order.OrderNumber), props.Text{Size: 9}),
			text.New(fmt.Sprintf("Date: %s", order.CreatedAt.Format("02 Jan 2006")), props.Text{Top: 4, Size: 9}),
		),
		col.New(4).Add(
			text.New("www.homechef.app", props.Text{Size: 8, Align: align.Right, Color: &props.Color{Red: 90, Green: 90, Blue: 90}}),
		),
	)
	m.AddRow(4, col.New(12).Add(spacer()))
}

func addInvoiceParties(m core.Maroto, order *models.Order) {
	chef := order.Chef
	cust := order.Customer

	chefBlock := []core.Component{
		text.New("FROM (Supplier)", props.Text{Size: 8, Style: fontstyle.Bold, Color: &props.Color{Red: 90, Green: 90, Blue: 90}}),
		text.New(chef.BusinessName, props.Text{Top: 4, Size: 11, Style: fontstyle.Bold}),
	}
	// Running vertical offset so optional lines (proprietor, address, GSTIN, FSSAI)
	// stack cleanly with no gaps when any is absent.
	top := 9.0
	if owner := strings.TrimSpace(chef.User.FirstName + " " + chef.User.LastName); owner != "" {
		chefBlock = append(chefBlock, text.New("Chef: "+owner, props.Text{Top: top, Size: 9}))
		top += 5
	}
	addrLine := joinNonEmpty([]string{chef.AddressLine1, chef.AddressLine2, chef.City, chef.State, chef.PostalCode}, ", ")
	if addrLine != "" {
		chefBlock = append(chefBlock, text.New(addrLine, props.Text{Top: top, Size: 9}))
		top += 5
	}
	if chef.GSTIN != "" {
		chefBlock = append(chefBlock, text.New("GSTIN: "+chef.GSTIN, props.Text{Top: top, Size: 9, Style: fontstyle.Bold}))
		top += 5
	}
	if chef.FSSAILicenseNumber != "" {
		chefBlock = append(chefBlock, text.New("FSSAI: "+chef.FSSAILicenseNumber, props.Text{Top: top, Size: 9}))
		top += 5
	}

	custBlock := []core.Component{
		text.New("BILL TO (Customer)", props.Text{Size: 8, Style: fontstyle.Bold, Color: &props.Color{Red: 90, Green: 90, Blue: 90}}),
		text.New(strOrDefault(cust.FirstName+" "+cust.LastName, "Customer"), props.Text{Top: 4, Size: 11, Style: fontstyle.Bold}),
	}
	custAddr := joinNonEmpty([]string{order.DeliveryAddressLine1, order.DeliveryAddressLine2, order.DeliveryAddressCity, order.DeliveryAddressState, order.DeliveryAddressPostalCode}, ", ")
	if custAddr != "" {
		custBlock = append(custBlock, text.New(custAddr, props.Text{Top: 9, Size: 9}))
	}

	m.AddRow(24,
		col.New(6).Add(chefBlock...),
		col.New(6).Add(custBlock...),
	)
	m.AddRow(4, col.New(12).Add(spacer()))
}

func addInvoiceItems(m core.Maroto, order *models.Order, hsnMeta map[uuid.UUID]string) {
	m.AddRow(7,
		col.New(5).Add(text.New("Item", props.Text{Size: 9, Style: fontstyle.Bold})),
		col.New(2).Add(text.New("HSN/SAC", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Center})),
		col.New(1).Add(text.New("Qty", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
		col.New(2).Add(text.New("Unit", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
		col.New(2).Add(text.New("Amount", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
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
		rows = append(rows, row.New(6).Add(
			col.New(5).Add(text.New(it.Name, props.Text{Size: 9})),
			col.New(2).Add(text.New(hsn, props.Text{Size: 9, Align: align.Center})),
			col.New(1).Add(text.New(fmt.Sprintf("%d", it.Quantity), props.Text{Size: 9, Align: align.Right})),
			col.New(2).Add(text.New(fmt.Sprintf("%.2f", it.Price), props.Text{Size: 9, Align: align.Right})),
			col.New(2).Add(text.New(fmt.Sprintf("%.2f", it.Subtotal), props.Text{Size: 9, Align: align.Right})),
		))
	}
	m.AddRows(rows...)
	m.AddRow(2, col.New(12).Add(spacer()))
}

func addInvoiceTotals(m core.Maroto, order *models.Order) {
	curr := order.Currency
	if curr == "" {
		curr = "INR"
	}
	totalRow := func(label string, amount float64, bold bool) core.Row {
		style := fontstyle.Normal
		if bold {
			style = fontstyle.Bold
		}
		return row.New(5).Add(
			col.New(8).Add(spacer()),
			col.New(2).Add(text.New(label, props.Text{Size: 9, Style: style, Align: align.Right})),
			col.New(2).Add(text.New(fmt.Sprintf("%s %.2f", curr, amount), props.Text{Size: 9, Style: style, Align: align.Right})),
		)
	}

	rows := []core.Row{totalRow("Subtotal", order.Subtotal, false)}
	if order.DeliveryFee > 0 {
		rows = append(rows, totalRow("Delivery", order.DeliveryFee, false))
	}
	if order.ServiceFee > 0 {
		rows = append(rows, totalRow("Platform fee", order.ServiceFee, false))
	}
	if order.Tax > 0 {
		// GST-compliant split (#invoice): an Indian tax invoice must show CGST+SGST
		// for an intra-state supply, or IGST for inter-state — never a single "GST".
		// Non-IN keeps the configured tax name.
		if strings.EqualFold(order.DeliveryAddressCountry, "IN") {
			b := SplitIndiaGST(order.Tax, order.TaxRate, order.Chef.State, order.DeliveryAddressState)
			if b.Intra {
				rows = append(rows, totalRow(fmt.Sprintf("CGST @ %.2g%%", b.CGSTRate), b.CGST, false))
				rows = append(rows, totalRow(fmt.Sprintf("SGST @ %.2g%%", b.SGSTRate), b.SGST, false))
			} else {
				rows = append(rows, totalRow(fmt.Sprintf("IGST @ %.2g%%", b.IGSTRate), b.IGST, false))
			}
		} else {
			taxLabel := order.TaxName
			if taxLabel == "" {
				taxLabel = "Tax"
			}
			rows = append(rows, totalRow(taxLabel, order.Tax, false))
		}
	}
	if order.Discount > 0 {
		rows = append(rows, totalRow("Discount", -order.Discount, false))
	}
	rows = append(rows, totalRow("TOTAL", order.Total, true))
	if order.RefundAmount > 0 {
		rows = append(rows,
			row.New(5).Add(
				col.New(8).Add(spacer()),
				col.New(2).Add(text.New("Refunded", props.Text{Size: 9, Style: fontstyle.Italic, Align: align.Right, Color: &props.Color{Red: 178, Green: 43, Blue: 14}})),
				col.New(2).Add(text.New(fmt.Sprintf("-%s %.2f", curr, order.RefundAmount), props.Text{Size: 9, Style: fontstyle.Italic, Align: align.Right, Color: &props.Color{Red: 178, Green: 43, Blue: 14}})),
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
				props.Text{Size: 7, Align: align.Center, Color: &props.Color{Red: 120, Green: 120, Blue: 120}, Style: fontstyle.Bold},
			)),
		)
	}
	m.AddRow(5,
		col.New(12).Add(text.New(
			"This is a computer-generated document and does not require a physical signature.",
			props.Text{Size: 7, Align: align.Center, Color: &props.Color{Red: 120, Green: 120, Blue: 120}, Style: fontstyle.Italic},
		)),
	)
	m.AddRow(4,
		col.New(12).Add(text.New(
			fmt.Sprintf("Generated %s · Home Chef Marketplace", time.Now().UTC().Format("02 Jan 2006 15:04 MST")),
			props.Text{Size: 7, Align: align.Center, Color: &props.Color{Red: 120, Green: 120, Blue: 120}},
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
