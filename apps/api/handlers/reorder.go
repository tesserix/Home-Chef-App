package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// reorder.go — #238. One-tap reorder of a past order. Rather than blindly
// re-creating the order (prices, availability and add-ons drift over time), the
// endpoint returns a validated *preview*: each line re-checked against the live
// menu, with current modifier option IDs resolved from the stored name snapshot.
// The client populates the cart with the available lines and tells the customer
// what changed — then the normal checkout + CreateOrder path takes over (which
// re-validates capacity atomically), so reorder adds no new trust surface.

// ReorderModifier is a fully-resolved add-on on a reorder line. Its shape mirrors
// the clients' SelectedModifier so the cart line can be rebuilt 1:1 without a
// second lookup.
type ReorderModifier struct {
	GroupID    uuid.UUID `json:"groupId"`
	GroupName  string    `json:"groupName"`
	OptionID   uuid.UUID `json:"optionId"`
	OptionName string    `json:"optionName"`
	PriceDelta float64   `json:"priceDelta"`
}

// resolveModifierOptions matches an order line's modifier snapshot (group +
// option NAMES, from #232) against the dish's CURRENT modifier groups, returning
// the live, fully-resolved add-ons. allMatched is false when any snapshot option
// was renamed, removed, or turned unavailable — the client then flags that line
// so the customer re-picks instead of silently losing an add-on.
func resolveModifierOptions(snapshot []models.OrderItemModifier, groups []models.ModifierGroup) (mods []ReorderModifier, allMatched bool) {
	allMatched = true
	for _, snap := range snapshot {
		found := false
		for _, g := range groups {
			if g.Name != snap.GroupName {
				continue
			}
			for _, o := range g.Options {
				if o.Name == snap.OptionName && o.IsAvailable {
					mods = append(mods, ReorderModifier{
						GroupID:    g.ID,
						GroupName:  g.Name,
						OptionID:   o.ID,
						OptionName: o.Name,
						PriceDelta: o.PriceDelta,
					})
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			allMatched = false
		}
	}
	return mods, allMatched
}

// ReorderItemResponse is one re-validated line of a reorder preview.
type ReorderItemResponse struct {
	MenuItemID  uuid.UUID         `json:"menuItemId"`
	Name        string            `json:"name"`
	Quantity    int               `json:"quantity"`
	Notes       string            `json:"notes,omitempty"`
	ImageURL    string            `json:"imageUrl,omitempty"`
	Modifiers   []ReorderModifier `json:"modifiers"`
	UnitPrice   float64           `json:"unitPrice"`
	Available   bool              `json:"available"`
	Reason      string            `json:"reason,omitempty"`      // why it's unavailable
	NeedsReview bool              `json:"needsReview,omitempty"` // add-ons changed; re-pick
}

// ReorderResponse is the cart-fill preview returned for a past order.
type ReorderResponse struct {
	ChefID        uuid.UUID             `json:"chefId"`
	ChefName      string                `json:"chefName"`
	ChefAccepting bool                  `json:"chefAccepting"`
	Items         []ReorderItemResponse `json:"items"`
}

// ReorderOrder builds a reorder preview for one of the caller's past orders.
func (h *OrderHandler) ReorderOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order id"})
		return
	}

	var order models.Order
	if err := database.DB.Preload("Items").
		Where("id = ? AND customer_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", order.ChefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	day := services.CapacityDay(time.Now())
	items := make([]ReorderItemResponse, 0, len(order.Items))
	for _, oi := range order.Items {
		// Per-line cancellations were never delivered — don't reorder them.
		if oi.IsCancelled {
			continue
		}

		line := ReorderItemResponse{
			MenuItemID: oi.MenuItemID,
			Name:       oi.Name,
			Quantity:   oi.Quantity,
			Notes:      oi.Notes,
		}

		var mi models.MenuItem
		if err := database.DB.Preload("ModifierGroups.Options").
			First(&mi, "id = ?", oi.MenuItemID).Error; err != nil {
			line.Available = false
			line.Reason = "No longer on the menu"
			items = append(items, line)
			continue
		}

		line.ImageURL = mi.ImageURL
		switch {
		case !mi.IsAvailable:
			line.Available = false
			line.Reason = "Currently unavailable"
		default:
			if _, soldOut := services.RemainingToday(mi.ID, mi.DailyCapacity, day); soldOut {
				line.Available = false
				line.Reason = "Sold out today"
				break
			}
			mods, allMatched := resolveModifierOptions(oi.ParsedModifiers(), mi.ModifierGroups)
			var delta float64
			for _, m := range mods {
				delta += m.PriceDelta
			}
			line.Available = true
			line.Modifiers = mods
			line.UnitPrice = mi.Price + delta
			line.NeedsReview = !allMatched
		}
		items = append(items, line)
	}

	c.JSON(http.StatusOK, ReorderResponse{
		ChefID:        chef.ID,
		ChefName:      chef.BusinessName,
		ChefAccepting: chef.AcceptingOrders,
		Items:         items,
	})
}
