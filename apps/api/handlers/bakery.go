package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// bakery.go — the bakery vertical (#1065): the baker's configurator save path
// and the customer's bakery discovery surface. Everything downstream of the
// order (payments, invoices, payouts) is shared with the meals vertical
// untouched; a cake is just a line item that had to be configured first.

type BakeryHandler struct{}

func NewBakeryHandler() *BakeryHandler { return &BakeryHandler{} }

// PreloadBakery attaches a menu item's configurator (options in canonical
// order). Used by every read that renders a menu.
func PreloadBakery(db *gorm.DB) *gorm.DB {
	return db.Preload("Bakery").Preload("Bakery.Options", func(d *gorm.DB) *gorm.DB {
		return d.Order("kind, sort_order")
	})
}

// parseBakeryUpdate decodes the update request's raw bakery field. It reports
// whether the client mentioned the configurator at all — an omitted key leaves
// it untouched, an explicit null strips it.
func parseBakeryUpdate(raw json.RawMessage) (*BakerySpecInput, bool, error) {
	if len(raw) == 0 {
		return nil, false, nil
	}
	if string(bytes.TrimSpace(raw)) == "null" {
		return nil, true, nil
	}
	var in BakerySpecInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, true, fmt.Errorf("we couldn't read the bakery options on this item")
	}
	return &in, true, nil
}

// saveItemBakerySpec replaces a menu item's bakery configurator (replace-all,
// same contract as the add-on editor). A nil input strips it. Only a kitchen
// that sells bakes may attach one — a meals-only chef sending a spec is a client
// bug, not a silently-accepted product.
func saveItemBakerySpec(itemID, chefID uuid.UUID, offersBakery bool, in *BakerySpecInput) error {
	if in == nil {
		return database.DB.Transaction(func(tx *gorm.DB) error {
			return deleteBakerySpec(tx, itemID)
		})
	}
	if !offersBakery {
		return errBakeryOnly
	}
	if err := validateBakerySpecInput(in); err != nil {
		return err
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := deleteBakerySpec(tx, itemID); err != nil {
			return err
		}
		spec := models.BakerySpec{
			MenuItemID:          itemID,
			ChefID:              chefID,
			ProductType:         models.BakeryProductType(in.ProductType),
			PricePerKg:          in.PricePerKg,
			MinWeightKg:         in.MinWeightKg,
			MaxWeightKg:         in.MaxWeightKg,
			WeightStepKg:        in.WeightStepKg,
			ServesPerKg:         in.ServesPerKg,
			AllowMessage:        in.AllowMessage,
			MaxMessageChars:     in.MaxMessageChars,
			AllowReferencePhoto: in.AllowReferencePhoto,
			LeadTimeHours:       in.LeadTimeHours,
			Occasions:           pq.StringArray(sanitizeOccasions(in.Occasions)),
		}
		if spec.WeightStepKg <= 0 {
			spec.WeightStepKg = 0.5
		}
		if spec.ServesPerKg <= 0 {
			spec.ServesPerKg = 8
		}
		if spec.MaxMessageChars <= 0 {
			spec.MaxMessageChars = 40
		}
		if err := tx.Create(&spec).Error; err != nil {
			return err
		}

		for i, o := range in.Options {
			name := strings.TrimSpace(o.Name)
			if name == "" || !validBakeryKind(o.Kind) {
				continue
			}
			avail := true
			if o.IsAvailable != nil {
				avail = *o.IsAvailable
			}
			mode := o.PriceMode
			if mode != models.BakeryPricePerKg {
				mode = models.BakeryPriceFlat
			}
			if err := tx.Create(&models.BakeryOption{
				SpecID:      spec.ID,
				Kind:        models.BakeryOptionKind(o.Kind),
				Name:        name,
				PriceDelta:  o.PriceDelta,
				PriceMode:   mode,
				DietaryTags: pq.StringArray(o.DietaryTags),
				Allergens:   pq.StringArray(o.Allergens),
				ImageURL:    o.ImageURL,
				IsAvailable: avail,
				IsDefault:   o.IsDefault,
				SortOrder:   i,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func deleteBakerySpec(tx *gorm.DB, itemID uuid.UUID) error {
	var existing []models.BakerySpec
	tx.Where("menu_item_id = ?", itemID).Find(&existing)
	for _, s := range existing {
		if err := tx.Where("spec_id = ?", s.ID).Delete(&models.BakeryOption{}).Error; err != nil {
			return err
		}
	}
	return tx.Unscoped().Where("menu_item_id = ?", itemID).Delete(&models.BakerySpec{}).Error
}

// errBakeryOnly is returned verbatim to the client, so it reads as guidance.
var errBakeryOnly = &bakeryError{"Turn on \"I also sell cakes and bakes\" in Profile before adding a cake configurator."}

type bakeryError struct{ msg string }

func (e *bakeryError) Error() string { return e.msg }

func validateBakerySpecInput(in *BakerySpecInput) error {
	if !validBakeryProductType(in.ProductType) {
		return &bakeryError{"Choose what kind of bakery product this is"}
	}
	if in.PricePerKg < 0 {
		return &bakeryError{"Price per kg cannot be negative"}
	}
	if in.PricePerKg > 0 {
		if in.MinWeightKg <= 0 || in.MaxWeightKg <= 0 {
			return &bakeryError{"Set the smallest and largest size you'll bake"}
		}
		if in.MaxWeightKg < in.MinWeightKg {
			return &bakeryError{"The largest size must be at least the smallest size"}
		}
	}
	if in.LeadTimeHours < 0 || in.LeadTimeHours > 24*30 {
		return &bakeryError{"Notice period must be between 0 and 720 hours"}
	}
	return nil
}

func validBakeryProductType(t string) bool {
	for _, p := range models.BakeryProductTypes {
		if string(p) == t {
			return true
		}
	}
	return false
}

func validBakeryKind(k string) bool {
	for _, kind := range models.BakeryOptionKinds {
		if string(kind) == k {
			return true
		}
	}
	return false
}

// sanitizeOccasions keeps only the canonical vocabulary — a free-text occasion
// would never match a filter chip.
func sanitizeOccasions(in []string) []string {
	allowed := map[string]bool{}
	for _, o := range models.BakeryOccasions {
		allowed[o] = true
	}
	out := []string{}
	seen := map[string]bool{}
	for _, o := range in {
		k := strings.ToLower(strings.TrimSpace(o))
		if allowed[k] && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

// ---------- Customer-facing ----------

// GetBakeryOptions returns the bakery vocabulary the apps render their pickers
// and filter chips from, so the client never hardcodes a list that drifts.
// GET /bakery/options
func (h *BakeryHandler) GetBakeryOptions(c *gin.Context) {
	types := make([]gin.H, 0, len(models.BakeryProductTypes))
	for _, t := range models.BakeryProductTypes {
		types = append(types, gin.H{"value": string(t), "label": bakeryProductLabels[t]})
	}
	kinds := make([]gin.H, 0, len(models.BakeryOptionKinds))
	for _, k := range models.BakeryOptionKinds {
		kinds = append(kinds, gin.H{"value": string(k), "label": models.BakeryOptionKindLabels[k]})
	}
	occasions := make([]gin.H, 0, len(models.BakeryOccasions))
	for _, o := range models.BakeryOccasions {
		occasions = append(occasions, gin.H{"value": o, "label": occasionLabel(o)})
	}

	c.JSON(http.StatusOK, gin.H{
		"productTypes": types,
		"optionKinds":  kinds,
		"occasions":    occasions,
		"diets":        services.BakeryDietOptions,
		"allergens":    services.BakeryAllergenOptions,
	})
}

var bakeryProductLabels = map[models.BakeryProductType]string{
	models.BakeryProductCake:       "Cakes",
	models.BakeryProductCupcake:    "Cupcakes",
	models.BakeryProductPastry:     "Pastries",
	models.BakeryProductBread:      "Breads",
	models.BakeryProductCookie:     "Cookies",
	models.BakeryProductDessertBox: "Dessert boxes",
	models.BakeryProductHamper:     "Hampers",
	models.BakeryProductSavoury:    "Savoury bakes",
}

// occasionLabel turns a canonical token into display copy ("baby-shower" →
// "Baby Shower").
func occasionLabel(token string) string {
	words := strings.Split(token, "-")
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// GetBakeryProducts is the customer's bakery browse: every approved, available
// bakery product across visible bakeries, filtered by product type, occasion,
// diet and price. GET /bakery/products
func (h *BakeryHandler) GetBakeryProducts(c *gin.Context) {
	limit := 40
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 && v <= 100 {
		limit = v
	}
	offset := 0
	if v, err := strconv.Atoi(c.Query("offset")); err == nil && v > 0 {
		offset = v
	}

	// Same customer-visibility gates as the rest of the menu: approved, available,
	// live partition, and never a kitchen that has only ever existed in test.
	q := database.DB.Model(&models.MenuItem{}).
		Joins("JOIN bakery_specs bs ON bs.menu_item_id = menu_items.id AND bs.deleted_at IS NULL").
		Joins("JOIN chef_profiles cp ON cp.id = menu_items.chef_id").
		Where("menu_items.is_approved = ? AND menu_items.is_available = ?", true, true).
		Where("menu_items.deleted_at IS NULL").
		Where("menu_items.mode = ?", models.ChefModeLive).
		Where("NOT (cp.mode = ? AND cp.first_live_at IS NULL)", models.ChefModeTest).
		Where("(cp.vertical = ? OR cp.sells_bakery = true)", models.VerticalBakery)

	if pt := strings.TrimSpace(c.Query("productType")); pt != "" {
		q = q.Where("bs.product_type = ?", pt)
	}
	if occ := strings.TrimSpace(c.Query("occasion")); occ != "" {
		// An empty occasions array means "any occasion", so it must still match.
		q = q.Where("(cardinality(bs.occasions) = 0 OR ? = ANY(bs.occasions))", strings.ToLower(occ))
	}
	if diet := strings.TrimSpace(c.Query("dietary")); diet != "" {
		// The tag may sit on the dish itself or on one of its options (an
		// eggless cake is usually eggless *by choice*, not by default).
		q = q.Where(`(EXISTS (SELECT 1 FROM unnest(menu_items.dietary_tags) t WHERE lower(t) = lower(?))
			OR EXISTS (SELECT 1 FROM bakery_options bo WHERE bo.spec_id = bs.id AND bo.is_available
				AND EXISTS (SELECT 1 FROM unnest(bo.dietary_tags) t WHERE lower(t) = lower(?))))`, diet, diet)
	}
	if v, err := strconv.ParseFloat(c.Query("maxPrice"), 64); err == nil && v > 0 {
		q = q.Where("COALESCE(NULLIF(bs.price_per_kg, 0), menu_items.price) <= ?", v)
	}
	if city := strings.TrimSpace(c.Query("city")); city != "" {
		q = q.Where("LOWER(cp.city) = LOWER(?)", city)
	}
	if chefID := strings.TrimSpace(c.Query("chefId")); chefID != "" {
		if id, err := uuid.Parse(chefID); err == nil {
			q = q.Where("menu_items.chef_id = ?", id)
		}
	}

	var total int64
	q.Count(&total)

	var items []models.MenuItem
	q.Preload("Images", func(d *gorm.DB) *gorm.DB { return d.Order("sort_order ASC") }).
		Scopes(PreloadBakery).
		Order("menu_items.is_featured DESC, menu_items.rating DESC, menu_items.total_orders DESC").
		Limit(limit).Offset(offset).Find(&items)

	chefIDs := make([]uuid.UUID, 0, len(items))
	for i := range items {
		chefIDs = append(chefIDs, items[i].ChefID)
	}
	var profiles []models.ChefProfile
	if len(chefIDs) > 0 {
		database.DB.Where("id IN ?", chefIDs).Find(&profiles)
	}
	chefs := make(map[uuid.UUID]models.ChefProfile, len(profiles))
	for i := range profiles {
		chefs[profiles[i].ID] = profiles[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"products": attachBakeryChefs(items, chefs),
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// BakeryProductResponse is a bakery item plus the kitchen that bakes it, so the
// customer's browse list can attribute a cake without a second round-trip.
type BakeryProductResponse struct {
	models.MenuItemResponse
	ChefName   string  `json:"chefName,omitempty"`
	ChefCity   string  `json:"chefCity,omitempty"`
	ChefRating float64 `json:"chefRating,omitempty"`
}

func attachBakeryChefs(items []models.MenuItem, chefs map[uuid.UUID]models.ChefProfile) []BakeryProductResponse {
	out := make([]BakeryProductResponse, 0, len(items))
	for i := range items {
		row := BakeryProductResponse{MenuItemResponse: items[i].ToResponse()}
		if chef, ok := chefs[items[i].ChefID]; ok {
			row.ChefName = chef.BusinessName
			row.ChefCity = chef.City
			row.ChefRating = chef.Rating
		}
		out = append(out, row)
	}
	return out
}
