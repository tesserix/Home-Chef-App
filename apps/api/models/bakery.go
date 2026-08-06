package models

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// bakery.go — the bakery vertical (#1065). A bakery is a chef like any other:
// same orders, payments, invoices and payouts. What differs is the product —
// a cake is configured (weight, shape, flavour, egg/sugar choice, message)
// before it can be priced, and it needs advance notice to bake.
//
// BakerySpec hangs off a MenuItem and turns it into a configurable product.
// Generic ModifierGroups still cover loose add-ons (candles, a knife, a topper);
// the spec covers what add-ons can't: per-kg pricing and the fixed vocabulary
// the configurator UI is built around.

// BakeryProductType is the shelf a bakery item sits on.
type BakeryProductType string

const (
	BakeryProductCake       BakeryProductType = "cake"
	BakeryProductCupcake    BakeryProductType = "cupcake"
	BakeryProductPastry     BakeryProductType = "pastry"
	BakeryProductBread      BakeryProductType = "bread"
	BakeryProductCookie     BakeryProductType = "cookie"
	BakeryProductDessertBox BakeryProductType = "dessert_box"
	BakeryProductHamper     BakeryProductType = "hamper"
	BakeryProductSavoury    BakeryProductType = "savoury"
)

// BakeryProductTypes is the canonical vocabulary, ordered for display.
var BakeryProductTypes = []BakeryProductType{
	BakeryProductCake, BakeryProductCupcake, BakeryProductPastry,
	BakeryProductBread, BakeryProductCookie, BakeryProductDessertBox,
	BakeryProductHamper, BakeryProductSavoury,
}

// BakeryOptionKind groups the choices a customer makes on one product. Every
// kind is single-select and required when the chef has defined options for it —
// a cake with no flavour chosen is not an order anyone can bake.
type BakeryOptionKind string

const (
	BakeryOptionShape     BakeryOptionKind = "shape"
	BakeryOptionFlavour   BakeryOptionKind = "flavour"
	BakeryOptionSponge    BakeryOptionKind = "sponge"
	BakeryOptionFrosting  BakeryOptionKind = "frosting"
	BakeryOptionEgg       BakeryOptionKind = "egg"
	BakeryOptionSweetness BakeryOptionKind = "sweetness"
	BakeryOptionTier      BakeryOptionKind = "tier"
)

// BakeryOptionKinds is the canonical ordering the configurator renders in.
var BakeryOptionKinds = []BakeryOptionKind{
	BakeryOptionShape, BakeryOptionTier, BakeryOptionFlavour, BakeryOptionSponge,
	BakeryOptionFrosting, BakeryOptionEgg, BakeryOptionSweetness,
}

// BakeryOptionKindLabels are the customer-facing headings per kind.
var BakeryOptionKindLabels = map[BakeryOptionKind]string{
	BakeryOptionShape:     "Shape",
	BakeryOptionTier:      "Tiers",
	BakeryOptionFlavour:   "Flavour",
	BakeryOptionSponge:    "Sponge",
	BakeryOptionFrosting:  "Frosting",
	BakeryOptionEgg:       "Egg preference",
	BakeryOptionSweetness: "Sweetness",
}

// Price modes for a BakeryOption surcharge.
const (
	// BakeryPriceFlat adds the delta once, regardless of size.
	BakeryPriceFlat = "flat"
	// BakeryPricePerKg scales the delta with the chosen weight — how a premium
	// flavour or a fondant finish actually costs a baker.
	BakeryPricePerKg = "per_kg"
)

// BakeryOccasions is the canonical occasion vocabulary customers filter by.
var BakeryOccasions = []string{
	"birthday", "anniversary", "wedding", "baby-shower", "engagement",
	"house-party", "corporate", "festival", "farewell", "graduation",
}

// BakerySpec makes one MenuItem configurable. One row per item.
type BakerySpec struct {
	// Live/test data partition. See models.ModePartition.
	ModePartition

	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MenuItemID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"menuItemId"`
	ChefID     uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`

	ProductType BakeryProductType `gorm:"type:varchar(20);not null;index" json:"productType"`

	// Weight pricing. PricePerKg > 0 switches the line off MenuItem.Price and
	// onto PricePerKg × weight — the way cakes are actually sold. Zero leaves
	// the item flat-priced (a loaf, a box of six cupcakes).
	PricePerKg   float64 `gorm:"default:0" json:"pricePerKg"`
	MinWeightKg  float64 `gorm:"default:0" json:"minWeightKg"`
	MaxWeightKg  float64 `gorm:"default:0" json:"maxWeightKg"`
	WeightStepKg float64 `gorm:"default:0.5" json:"weightStepKg"`
	// ServesPerKg drives the "serves ~10" hint next to each weight.
	ServesPerKg int `gorm:"default:8" json:"servesPerKg"`

	AllowMessage        bool `gorm:"default:false" json:"allowMessage"`
	MaxMessageChars     int  `gorm:"default:40" json:"maxMessageChars"`
	AllowReferencePhoto bool `gorm:"default:false" json:"allowReferencePhoto"`

	// LeadTimeHours is the advance notice the baker needs. Enforced against the
	// order's scheduled slot at checkout.
	LeadTimeHours int `gorm:"default:0" json:"leadTimeHours"`

	// Occasions this product is offered for; empty means "any occasion".
	Occasions pq.StringArray `gorm:"type:text[]" json:"occasions"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Options []BakeryOption `gorm:"foreignKey:SpecID" json:"options"`
}

// BakeryOption is one choice within a kind (Heart shape, Belgian chocolate,
// Eggless, Sugar-free). Dietary tags and allergens ride on the option because
// choosing "Eggless" genuinely changes what the customer is eating.
type BakeryOption struct {
	ID     uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SpecID uuid.UUID        `gorm:"type:uuid;not null;index" json:"specId"`
	Kind   BakeryOptionKind `gorm:"type:varchar(16);not null;index" json:"kind"`
	Name   string           `gorm:"not null" json:"name"`

	PriceDelta float64 `gorm:"default:0" json:"priceDelta"`
	PriceMode  string  `gorm:"type:varchar(8);default:'flat'" json:"priceMode"`

	// DietaryTags / Allergens the option adds to the line (e.g. an "Egg" option
	// declares the eggs allergen; "Eggless" declares the eggless diet tag).
	DietaryTags pq.StringArray `gorm:"type:text[]" json:"dietaryTags"`
	Allergens   pq.StringArray `gorm:"type:text[]" json:"allergens"`

	ImageURL    string    `gorm:"" json:"imageUrl,omitempty"`
	IsAvailable bool      `gorm:"default:true" json:"isAvailable"`
	IsDefault   bool      `gorm:"default:false" json:"isDefault"`
	SortOrder   int       `gorm:"default:0" json:"sortOrder"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"-"`
}

// BakerySelection is one resolved choice on an order line.
type BakerySelection struct {
	Kind       BakeryOptionKind `json:"kind"`
	Label      string           `json:"label"`
	Name       string           `json:"name"`
	PriceDelta float64          `json:"priceDelta"`
}

// OrderItemBakery is the immutable snapshot of a configured bakery line, stored
// as JSON on OrderItem.BakeryDetails. The chef bakes from this and the invoice
// prints from it, so it must stand alone once the menu item changes.
type OrderItemBakery struct {
	ProductType       BakeryProductType `json:"productType"`
	WeightKg          float64           `json:"weightKg,omitempty"`
	Serves            int               `json:"serves,omitempty"`
	BasePrice         float64           `json:"basePrice"`
	Selections        []BakerySelection `json:"selections"`
	MessageOnCake     string            `json:"messageOnCake,omitempty"`
	ReferencePhotoURL string            `json:"referencePhotoUrl,omitempty"`
	Occasion          string            `json:"occasion,omitempty"`
	// DietaryTags / Allergens resolved from the chosen options, so the kitchen
	// docket and the customer's receipt both carry the eggless / sugar-free call.
	DietaryTags []string `json:"dietaryTags,omitempty"`
	Allergens   []string `json:"allergens,omitempty"`
}

// Summary renders the configuration as one line for the cart, the kitchen
// docket, the invoice and the receipt — every surface prints the same string.
func (b OrderItemBakery) Summary() string {
	parts := []string{}
	if b.WeightKg > 0 {
		parts = append(parts, strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.2f", b.WeightKg), "0"), ".")+" kg")
	}
	for _, s := range b.Selections {
		parts = append(parts, s.Name)
	}
	if b.MessageOnCake != "" {
		parts = append(parts, "“"+b.MessageOnCake+"”")
	}
	return strings.Join(parts, " · ")
}

// ParsedBakery decodes an order line's bakery snapshot. Returns nil when the
// line is an ordinary dish.
func (oi *OrderItem) ParsedBakery() *OrderItemBakery {
	if strings.TrimSpace(oi.BakeryDetails) == "" {
		return nil
	}
	var b OrderItemBakery
	if err := json.Unmarshal([]byte(oi.BakeryDetails), &b); err != nil {
		return nil
	}
	return &b
}

// WeightChoices expands the spec's min/max/step into the sizes the configurator
// offers. Empty for a flat-priced product. Lives on the model so ToResponse can
// render it without importing the service layer.
func (s *BakerySpec) WeightChoices() []float64 {
	if s == nil || s.PricePerKg <= 0 {
		return nil
	}
	min, max, step := s.MinWeightKg, s.MaxWeightKg, s.WeightStepKg
	if min <= 0 {
		min = 0.5
	}
	if max <= 0 || max < min {
		max = min
	}
	if step <= 0 {
		step = 0.5
	}
	out := []float64{}
	for w := min; w <= max+0.001 && len(out) < 40; w += step {
		out = append(out, math.Round(w*100)/100)
	}
	return out
}

// BakeryOptionsByKind groups a spec's available options in canonical kind order,
// which is how both the configurator and the vendor editor render them.
func (s *BakerySpec) BakeryOptionsByKind() map[BakeryOptionKind][]BakeryOption {
	out := map[BakeryOptionKind][]BakeryOption{}
	for _, o := range s.Options {
		out[o.Kind] = append(out[o.Kind], o)
	}
	return out
}
