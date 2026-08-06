package services

import (
	"encoding/json"
	"testing"

	"github.com/homechef/api/models"
)

// One line name for every surface: the app, the invoice rows and the PDF.
func TestInvoiceLineName(t *testing.T) {
	bakery, _ := json.Marshal(models.OrderItemBakery{
		WeightKg:      1.5,
		Selections:    []models.BakerySelection{{Kind: "shape", Label: "Shape", Name: "Heart"}},
		MessageOnCake: "Happy Birthday",
	})
	bakeryJSON := string(bakery)
	mods, _ := json.Marshal([]models.OrderItemModifier{{GroupName: "Sides", OptionName: "Extra raita"}})

	cases := []struct {
		name string
		item models.OrderItem
		want string
	}{
		{"plain", models.OrderItem{Name: "Dal Khichdi"}, "Dal Khichdi"},
		{
			"bakery",
			models.OrderItem{Name: "Truffle Cake", BakeryDetails: &bakeryJSON},
			"Truffle Cake — 1.5 kg · Heart · “Happy Birthday”",
		},
		{
			"modifiers",
			models.OrderItem{Name: "Thali", Modifiers: string(mods)},
			"Thali (Extra raita)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := InvoiceLineName(tc.item); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
