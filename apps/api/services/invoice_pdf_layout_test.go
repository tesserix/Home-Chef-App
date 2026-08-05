package services

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/config"
)

// The invoice is assembled from four blocks; this renders all of them for a
// realistic order so a layout change can never ship as a document that fails to
// generate (the DB-backed entry point can't be exercised in a unit test).
func TestInvoiceDocumentRenders(t *testing.T) {
	order := models.Order{
		OrderNumber: "HC-2026-0042",
		CreatedAt:   time.Date(2026, 8, 5, 12, 30, 0, 0, time.UTC),
		Status:      models.OrderStatusDelivered,
		Currency:    "INR",
		Subtotal:    450,
		DeliveryFee: 39.15,
		PlatformFee: 12,
		Tip:         20,
		Total:       561.32,
		Chef: models.ChefProfile{
			BusinessName:       "Anita's Kitchen",
			AddressLine1:       "12 Residency Road",
			City:               "Bengaluru",
			State:              "Karnataka",
			PostalCode:         "560025",
			GSTIN:              "29AABCU9603R1ZM",
			FSSAILicenseNumber: "12345678901234",
		},
		Customer: models.User{FirstName: "Ravi", LastName: "Kumar"},
		Items: []models.OrderItem{
			{MenuItemID: uuid.New(), Name: "Bisi bele bath", Quantity: 2, Price: 150, Subtotal: 300},
			{MenuItemID: uuid.New(), Name: "Curd rice", Quantity: 1, Price: 150, Subtotal: 150},
		},
	}

	m := maroto.New(config.NewBuilder().WithLeftMargin(15).WithTopMargin(15).WithRightMargin(15).Build())
	addInvoiceHeader(m, &order)
	addInvoiceParties(m, &order)
	addInvoiceItems(m, &order, map[uuid.UUID]string{})
	addInvoiceTotals(m, &order)
	addInvoiceFooter(m, &order)

	doc, err := m.Generate()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(doc.GetBytes()) < 1000 {
		t.Fatalf("document looks empty: %d bytes", len(doc.GetBytes()))
	}
	if out := os.Getenv("INVOICE_PDF_OUT"); out != "" {
		_ = os.WriteFile(out, doc.GetBytes(), 0o600)
	}
}

func TestInvoiceDocumentRendersARefundedReceipt(t *testing.T) {
	order := models.Order{
		OrderNumber:  "HC-2026-0043",
		CreatedAt:    time.Now(),
		Status:       models.OrderStatusCancelled,
		Currency:     "INR",
		Subtotal:     200,
		Total:        200,
		RefundAmount: 200,
		Chef:         models.ChefProfile{BusinessName: "Anita's Kitchen"},
		Customer:     models.User{FirstName: "Ravi"},
		Items:        []models.OrderItem{{MenuItemID: uuid.New(), Name: "Curd rice", Quantity: 1, Price: 200, Subtotal: 200}},
	}

	m := maroto.New(config.NewBuilder().Build())
	addInvoiceHeader(m, &order)
	addInvoiceParties(m, &order)
	addInvoiceItems(m, &order, map[uuid.UUID]string{})
	addInvoiceTotals(m, &order)
	addInvoiceFooter(m, &order)

	if _, err := m.Generate(); err != nil {
		t.Fatalf("generate: %v", err)
	}
}
