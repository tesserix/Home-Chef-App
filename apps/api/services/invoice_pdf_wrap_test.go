package services

// A party block stacks its optional lines at fixed 5mm steps, which assumed
// every line was one line. A real chef address is 70+ characters and wraps, so
// GSTIN printed on top of the address and FSSAI on top of that — the invoice
// came out overlapping and unreadable.

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/config"
)

func TestTextLines(t *testing.T) {
	// A half-page column: A4 (210mm) less 15mm margins, halved.
	const half = 90.0

	cases := []struct {
		name string
		text string
		size float64
		want int
	}{
		{"empty occupies nothing", "", 9, 0},
		{"a short line stays one line", "Bengaluru, Karnataka", 9, 1},
		{
			"a real chef address wraps",
			"324, 12th Main Road, HAL 2nd Stage, Indiranagar, Bengaluru, Karnataka, 560071",
			9, 2,
		},
		{"smaller type fits more before wrapping", "GSTIN: 29ABCDE1234F1Z5", 7, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := textLines(tc.text, tc.size, half); got != tc.want {
				t.Fatalf("textLines(%q, %v) = %d, want %d", tc.text, tc.size, got, tc.want)
			}
		})
	}
}

// The regression itself: the long-address order that produced the overlapping
// document. Renders every block, so a stacking change cannot ship broken.
func TestInvoiceRendersALongAddressWithoutOverlap(t *testing.T) {
	order := models.Order{
		OrderNumber: "SAFFRON-HOME-KITCHEN-HC26080515161573",
		CreatedAt:   time.Date(2026, 8, 6, 12, 30, 0, 0, time.UTC),
		Status:      models.OrderStatusDelivered,
		Currency:    "INR",
		Subtotal:    320,
		DeliveryFee: 39.12,
		PlatformFee: 13.53,
		Total:       393.05,
		Chef: models.ChefProfile{
			BusinessName:       "Saffron Home Kitchen",
			AddressLine1:       "324, 12th Main Road",
			AddressLine2:       "HAL 2nd Stage, Indiranagar",
			City:               "Bengaluru",
			State:              "Karnataka",
			PostalCode:         "560071",
			GSTIN:              "29ABCDE1234F1Z5",
			FSSAILicenseNumber: "11203445555577",
			User:               models.User{FirstName: "Arjun", LastName: "Menon"},
		},
		Customer:                  models.User{FirstName: "Priya", LastName: "Sharma"},
		DeliveryAddressLine1:      "48, 5th Cross",
		DeliveryAddressLine2:      "Domlur",
		DeliveryAddressCity:       "Bengaluru",
		DeliveryAddressState:      "Karnataka",
		DeliveryAddressPostalCode: "560071",
		Items: []models.OrderItem{
			{MenuItemID: uuid.New(), Name: "Butter Chicken", Quantity: 1, Price: 320, Subtotal: 320},
		},
	}

	// The stacked block must reserve room for the wrapped address, or the next
	// line lands on top of it.
	got := chefPartyBlockHeight(&order.Chef)
	if got < 30 {
		t.Fatalf("chef block height %v is too short for a wrapped address + GSTIN + FSSAI", got)
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
	if out := os.Getenv("INVOICE_PDF_OUT"); out != "" {
		_ = os.WriteFile(out, doc.GetBytes(), 0o600)
	}
}
