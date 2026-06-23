package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	natsclient "github.com/nats-io/nats.go"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"gorm.io/gorm"
)

// wsUpgrader upgrades HTTP connections to WebSocket.
//
// Browsers always send Origin on WS upgrade; mobile/native clients omit it.
// We accept the request when EITHER:
//   - Origin is empty (native client), OR
//   - Origin matches the API CORS allowlist (browser).
//
// Anything else is rejected to prevent cross-site WS hijacking.
var wsUpgrader = websocket.Upgrader{
	CheckOrigin:     allowedWSOrigin,
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func allowedWSOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // native / curl / non-browser
	}
	host := r.Host
	// Same-origin (proxied via Cloudflare/Istio) is always fine.
	if strings.EqualFold(origin, "https://"+host) || strings.EqualFold(origin, "http://"+host) {
		return true
	}
	// Trusted production + dev origins. Keep in sync with routes.allowedOrigins().
	for _, allowed := range []string{
		"https://fe3dr.com",
		"https://www.fe3dr.com",
		"https://vendors.fe3dr.com",
		"https://admin.fe3dr.com",
		"https://auth.fe3dr.com",
		"https://delivery.fe3dr.com",
		"http://localhost:5173",
		"http://localhost:3000",
	} {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}

type OrderHandler struct{}

func NewOrderHandler() *OrderHandler {
	return &OrderHandler{}
}

// CreateOrderRequest represents the order creation payload
type CreateOrderRequest struct {
	ChefID               uuid.UUID             `json:"chefId" binding:"required"`
	Items                []CreateOrderItem     `json:"items" binding:"required,min=1"`
	DeliveryAddressID    *uuid.UUID            `json:"deliveryAddressId"`
	DeliveryAddress      *CreateAddressRequest `json:"deliveryAddress"`
	DeliveryInstructions string                `json:"deliveryInstructions"`
	SpecialInstructions  string                `json:"specialInstructions"`
	Tip                  float64               `json:"tip"`
	PromoCode            string                `json:"promoCode"`
	PaymentMethodID      *uuid.UUID            `json:"paymentMethodId"`
	ScheduledFor         *time.Time            `json:"scheduledFor"`
	// Scheduled delivery slot (#51). When DeliverySlot is "lunch"/"dinner" and
	// the chef offers it, the server resolves the slot window → ScheduledFor and
	// reserves the chef's per-slot daily capacity. DeliveryDate is "YYYY-MM-DD"
	// IST (empty = today, bounded by the booking horizon).
	DeliverySlot string `json:"deliverySlot"`
	DeliveryDate string `json:"deliveryDate"`
	// FulfillmentType is "delivery" (default, 3PL), "pickup" (customer collects),
	// or "chef_delivery" (reserved for Phase 2). Empty defaults to delivery.
	FulfillmentType string `json:"fulfillmentType"`
}

type CreateOrderItem struct {
	MenuItemID uuid.UUID `json:"menuItemId" binding:"required"`
	Quantity   int       `json:"quantity" binding:"required,min=1"`
	Notes      string    `json:"notes"`
	// ModifierOptionIDs are the selected add-on options for this line (#232).
	// The server validates them against the item's groups, prices them, and
	// snapshots the selection onto the order line.
	ModifierOptionIDs []uuid.UUID `json:"modifierOptionIds"`
}

// validateAndPriceModifiers checks the selected option ids against a menu item's
// modifier groups — enforcing required/min/max per group and option availability —
// and returns the per-unit price delta + the snapshot to persist (#232). Pure
// (no DB) so it's unit-tested. Returns a customer-facing error for a bad selection.
func validateAndPriceModifiers(groups []models.ModifierGroup, selected []uuid.UUID) (float64, []models.OrderItemModifier, error) {
	sel := make(map[uuid.UUID]bool, len(selected))
	for _, id := range selected {
		sel[id] = true
	}

	var delta float64
	snapshot := []models.OrderItemModifier{}
	matched := map[uuid.UUID]bool{}

	for _, g := range groups {
		count := 0
		for _, o := range g.Options {
			if !sel[o.ID] {
				continue
			}
			if !o.IsAvailable {
				return 0, nil, fmt.Errorf("%s is no longer available", o.Name)
			}
			count++
			delta += o.PriceDelta
			snapshot = append(snapshot, models.OrderItemModifier{
				GroupName: g.Name, OptionName: o.Name, PriceDelta: o.PriceDelta,
			})
			matched[o.ID] = true
		}
		minSel := g.MinSelect
		if g.Required && minSel < 1 {
			minSel = 1
		}
		if count < minSel {
			return 0, nil, fmt.Errorf("please choose an option for %q", g.Name)
		}
		if g.MaxSelect > 0 && count > g.MaxSelect {
			return 0, nil, fmt.Errorf("too many options chosen for %q", g.Name)
		}
	}

	// Every selected id must belong to one of this item's groups.
	for _, id := range selected {
		if !matched[id] {
			return 0, nil, fmt.Errorf("invalid add-on selection")
		}
	}
	return delta, snapshot, nil
}

// resolveFulfillment maps the customer's requested mode (delivery vs pickup) to
// the actual fulfillment type. The CHEF controls who delivers, not the customer:
// a "delivery" order to a chef who self-delivers becomes a chef-delivered order
// (the chef drives it), otherwise it routes to 3PL. The customer never chooses
// chef_delivery directly.
func resolveFulfillment(req CreateOrderRequest, chef models.ChefProfile) (models.FulfillmentType, error) {
	switch models.FulfillmentType(req.FulfillmentType) {
	case "", models.FulfillmentDelivery:
		// "I'll have it delivered" → the chef decides who carries it.
		if chef.OffersSelfDelivery {
			return models.FulfillmentChefDelivery, nil
		}
		return models.FulfillmentDelivery, nil
	case models.FulfillmentPickup:
		if !chef.OffersPickup {
			return "", fmt.Errorf("this kitchen does not offer pickup")
		}
		return models.FulfillmentPickup, nil
	case models.FulfillmentChefDelivery:
		// Customers no longer request this directly, but accept it defensively
		// from older clients when the chef does offer self-delivery.
		if !chef.OffersSelfDelivery {
			return "", fmt.Errorf("this kitchen does not offer chef delivery")
		}
		return models.FulfillmentChefDelivery, nil
	default:
		return "", fmt.Errorf("unsupported fulfillment option")
	}
}

type CreateAddressRequest struct {
	Line1      string `json:"line1" binding:"required"`
	Line2      string `json:"line2"`
	City       string `json:"city" binding:"required"`
	State      string `json:"state" binding:"required"`
	PostalCode string `json:"postalCode" binding:"required"`
	// ISO-3166 alpha-2. Optional on the wire — drives per-country tax
	// lookup, so the frontend should pass it when known. Falls back to
	// the saved address's country or India when absent.
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// CreateOrder creates a new order
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Platform operating-hours gate — admins can set open/close times in
	// Settings → Platform, and IsPlatformOpen evaluates them against the
	// configured timezone. Unconfigured = always open (current behavior).
	if open, msg := services.IsPlatformOpen(); !open {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	// Verify chef exists and is active
	var chef models.ChefProfile
	if err := database.DB.Where("id = ? AND is_active = ?", req.ChefID, true).
		First(&chef).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chef not found or not available"})
		return
	}

	if !chef.AcceptingOrders {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chef is not accepting orders"})
		return
	}

	// FSSAI hard lockout (#32): an India chef whose food-safety (FSSAI) licence
	// has lapsed must not take new orders until a renewal is verified — this
	// protects customers from food prepared under an expired licence and the
	// platform from the legal/reputational fallout. Customer-facing copy stays
	// generic (the chef's compliance state is private).
	if services.IsChefFSSAIExpired(&chef) {
		middleware.RecordFSSAILockout("order_blocked")
		c.JSON(http.StatusBadRequest, gin.H{"error": "This kitchen is temporarily unavailable. Please choose another chef."})
		return
	}

	// Capacity cutoff (#48): a chef can auto-close ordering once their meal
	// cutoffs have passed (extends pause-receiving).
	capSettings := services.GetChefCapacitySettings(req.ChefID)
	if services.IsPastDailyClose(capSettings, time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This kitchen has closed ordering for today."})
		return
	}

	// Resolve and validate the fulfillment mode. Pickup is only allowed when
	// the chef explicitly offers it; unknown modes are rejected early.
	fulfillment, err := resolveFulfillment(req, chef)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Scheduled delivery slot (#51): when the customer picks a lunch/dinner slot
	// the chef offers, resolve the slot window → ScheduledFor and remember the
	// per-slot daily capacity to reserve inside the order transaction below.
	var slotScheduledFor *time.Time
	var slotBookingDay time.Time
	slotToReserve, slotCap := "", 0
	if req.DeliverySlot != "" {
		sf, day, err := services.ResolveSlotSchedule(capSettings, req.DeliverySlot, req.DeliveryDate, time.Now())
		if err != nil {
			switch {
			case errors.Is(err, services.ErrSlotClosed):
				c.JSON(http.StatusConflict, gin.H{"error": "That delivery slot has closed. Please pick another time."})
			case errors.Is(err, services.ErrSlotDateInvalid):
				c.JSON(http.StatusBadRequest, gin.H{"error": "Please choose a valid delivery date."})
			default: // ErrSlotNotOffered
				c.JSON(http.StatusBadRequest, gin.H{"error": "This chef doesn't offer that delivery slot."})
			}
			return
		}
		slotScheduledFor = &sf
		slotBookingDay = day
		if capLimit := services.SlotCapacity(capSettings, req.DeliverySlot); capLimit != nil {
			slotToReserve, slotCap = req.DeliverySlot, *capLimit
		}
	}

	// Get menu items and calculate totals
	var subtotal float64
	orderItems := make([]models.OrderItem, len(req.Items))

	// Daily-capacity reservations to apply atomically inside the transaction (#48).
	type capReservation struct {
		itemID uuid.UUID
		name   string
		qty    int
		cap    int
	}
	var capReservations []capReservation

	for i, item := range req.Items {
		var menuItem models.MenuItem
		if err := database.DB.Where("id = ? AND chef_id = ? AND is_available = ?",
			item.MenuItemID, req.ChefID, true).First(&menuItem).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Menu item %s not found or unavailable", item.MenuItemID)})
			return
		}

		// Validate + price the selected add-on modifiers (#232). The per-unit
		// price includes the modifier deltas so Price × Quantity == Subtotal.
		var groups []models.ModifierGroup
		database.DB.Preload("Options").Where("menu_item_id = ?", item.MenuItemID).
			Order("sort_order").Find(&groups)
		modDelta, modSnapshot, merr := validateAndPriceModifiers(groups, item.ModifierOptionIDs)
		if merr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": merr.Error()})
			return
		}
		modJSON, _ := json.Marshal(modSnapshot)

		unitPrice := menuItem.Price + modDelta
		itemSubtotal := unitPrice * float64(item.Quantity)
		subtotal += itemSubtotal

		orderItems[i] = models.OrderItem{
			MenuItemID: item.MenuItemID,
			Name:       menuItem.Name,
			Price:      unitPrice,
			Quantity:   item.Quantity,
			Subtotal:   itemSubtotal,
			Notes:      item.Notes,
			Modifiers:  string(modJSON),
		}

		if menuItem.DailyCapacity != nil && *menuItem.DailyCapacity > 0 {
			capReservations = append(capReservations, capReservation{
				itemID: item.MenuItemID, name: menuItem.Name, qty: item.Quantity, cap: *menuItem.DailyCapacity,
			})
		}
	}

	// Check minimum order
	if subtotal < chef.MinimumOrder {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":        fmt.Sprintf("Minimum order is $%.2f", chef.MinimumOrder),
			"minimumOrder": chef.MinimumOrder,
		})
		return
	}

	// Calculate fees from the platform policy (Settings → Platform).
	// Defaults match the prior hardcoded values (10% service, 8% tax, $2.99
	// delivery) so behavior doesn't change until an admin edits the policy.
	// deliveryFee starts at the flat policy fee and is replaced below with a
	// live 3PL quote once the delivery address (and its coords) is resolved.
	policy := services.GetPlatformPolicy()
	deliveryFee := policy.BaseDeliveryFee
	serviceFee := subtotal * (policy.ServiceFeePercent / 100.0)

	// Tax is resolved per customer country (and state/region when known)
	// from the tax_rates table rather than the single global TaxPercent.
	// GST for India, VAT for EU, HST for Ontario, etc. — all picked by
	// the delivery address the customer chose.
	tip := req.Tip
	discount := 0.0

	// Apply promo code discount
	var appliedPromo *models.PromoCode
	chefFundedDiscount := 0.0
	if req.PromoCode != "" {
		promoDiscount, promo, promoErr := validateAndCalculateDiscount(req.PromoCode, userID, chef.ID, subtotal)
		if promoErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": promoErr})
			return
		}
		discount = promoDiscount
		appliedPromo = promo
		// Chef-funded promos are billed to the chef at settlement (#39).
		chefFundedDiscount = services.ChefFundedPortion(promo, promoDiscount)
	}

	// Total is computed below, after tax is resolved from the delivery
	// address. Until then we only have the pre-tax pieces.

	// Get or create delivery address
	var deliveryAddr CreateAddressRequest
	if req.DeliveryAddressID != nil {
		var addr models.Address
		if err := database.DB.Where("id = ? AND user_id = ?", *req.DeliveryAddressID, userID).
			First(&addr).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Delivery address not found"})
			return
		}
		deliveryAddr = CreateAddressRequest{
			Line1:      addr.Line1,
			Line2:      addr.Line2,
			City:       addr.City,
			State:      addr.State,
			PostalCode: addr.PostalCode,
			Country:    addr.Country,
			Latitude:   addr.Latitude,
			Longitude:  addr.Longitude,
		}
	} else if req.DeliveryAddress != nil {
		deliveryAddr = *req.DeliveryAddress
	} else if fulfillment == models.FulfillmentPickup {
		// Pickup: no delivery address — the customer collects from the chef.
		// Leave the address empty; country defaults to IN for tax resolution.
		deliveryAddr = CreateAddressRequest{Country: "IN"}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Delivery address required"})
		return
	}

	// Country defaults to India if the address was created before the
	// column was populated; tax resolver falls back to a zero-rate rule
	// so the order still goes through.
	deliveryCountry := deliveryAddr.Country
	if deliveryCountry == "" {
		deliveryCountry = "IN"
	}

	// Live 3PL delivery quote, computed now that the drop address is known.
	// Pickup orders pay no delivery fee and skip the 3PL quote entirely.
	// For delivery, falls back to the flat policy fee (already in deliveryFee)
	// when the address has no coordinates yet or no provider can serve the leg,
	// so checkout never blocks on the quote.
	if fulfillment == models.FulfillmentPickup {
		deliveryFee = 0
	} else if fulfillment == models.FulfillmentChefDelivery {
		// Chef delivers themselves — distance-based self-delivery fee, no 3PL quote.
		deliveryFee = services.ComputeSelfDeliveryFee(chef, deliveryAddr.Latitude, deliveryAddr.Longitude)
	} else if fee, ok := services.QuoteCheckoutDeliveryFee(chef, deliveryAddr.City, deliveryCountry, deliveryAddr.Latitude, deliveryAddr.Longitude); ok {
		deliveryFee = fee
	}

	taxRule := services.ResolveTaxRate(deliveryCountry, deliveryAddr.State)
	// Tax base: subtotal + deliveryFee + serviceFee, after promo discount.
	// Tip is excluded (direct pass-through to chef/driver).
	taxBase := subtotal + deliveryFee + serviceFee - discount
	if taxBase < 0 {
		taxBase = 0
	}
	var tax float64
	if taxRule.Inclusive {
		// Inclusive rate: the base already contains the tax; back it out
		// for line-item display but don't add it to total.
		tax = taxBase - (taxBase / (1 + taxRule.Rate/100.0))
	} else {
		tax = taxBase * (taxRule.Rate / 100.0)
	}

	// Order total. For inclusive-tax jurisdictions (EU VAT etc.) the tax
	// line is informational — it's part of taxBase already, so we don't
	// add it again or the customer would be charged twice.
	var total float64
	if taxRule.Inclusive {
		total = subtotal + deliveryFee + serviceFee + tip - discount
	} else {
		total = subtotal + deliveryFee + serviceFee + tax + tip - discount
	}

	// If the admin has configured delivery zones, enforce coverage. When no
	// zones exist we skip — feature is opt-in so existing customers aren't
	// suddenly unable to order. Pickup orders skip this gate entirely since
	// there is no drop-off address to check.
	if fulfillment != models.FulfillmentPickup && services.HasActiveZones() {
		if deliveryAddr.Latitude == 0 && deliveryAddr.Longitude == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Delivery address needs location coordinates. Please select your address from the map.",
			})
			return
		}
		if zone := services.FindZoneForAddress(deliveryAddr.Latitude, deliveryAddr.Longitude); zone == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "We don't deliver to this address yet. Please pick a supported area.",
			})
			return
		}
	}

	// Generate order number
	orderNumber := generateOrderNumber()

	// Inherit the chef's current gateway so VerifyPayment / refund later
	// read the same provider this order was created against. Falls back to
	// razorpay for older chefs without the column populated.
	orderProvider := chef.PaymentProvider
	if orderProvider == "" {
		orderProvider = "razorpay"
	}

	// Order currency derives from the chef's settlement country so Stripe
	// charges and chef payouts settle in a consistent currency. Defaults
	// to INR for legacy chefs with empty PayoutCountry.
	orderCurrency := strings.ToUpper(services.CurrencyForCountry(chef.PayoutCountry))

	// Create order
	order := models.Order{
		OrderNumber:               orderNumber,
		CustomerID:                userID,
		ChefID:                    chef.ID,
		PaymentProvider:           orderProvider,
		Currency:                  orderCurrency,
		Status:                    models.OrderStatusPending,
		PaymentStatus:             models.PaymentPending,
		Subtotal:                  subtotal,
		DeliveryFee:               deliveryFee,
		ServiceFee:                serviceFee,
		Tax:                       tax,
		TaxRate:                   taxRule.Rate,
		TaxName:                   taxRule.TaxName,
		Tip:                       tip,
		Discount:                  discount,
		ChefFundedDiscount:        chefFundedDiscount,
		Total:                     total,
		PromoCode:                 req.PromoCode,
		DeliveryAddressLine1:      deliveryAddr.Line1,
		DeliveryAddressLine2:      deliveryAddr.Line2,
		DeliveryAddressCity:       deliveryAddr.City,
		DeliveryAddressState:      deliveryAddr.State,
		DeliveryAddressPostalCode: deliveryAddr.PostalCode,
		DeliveryAddressCountry:    deliveryCountry,
		DeliveryLatitude:          deliveryAddr.Latitude,
		DeliveryLongitude:         deliveryAddr.Longitude,
		DeliveryInstructions:      req.DeliveryInstructions,
		SpecialInstructions:       req.SpecialInstructions,
		ScheduledFor:              req.ScheduledFor,
		DeliverySlot:              req.DeliverySlot,
		FulfillmentType:           fulfillment,
		EstimatedPrepTime:         30, // Default, could be calculated
	}
	// A resolved slot window is authoritative over any client-sent ScheduledFor.
	if slotScheduledFor != nil {
		order.ScheduledFor = slotScheduledFor
	}

	// Start transaction
	tx := database.DB.Begin()

	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Add order items
	for i := range orderItems {
		orderItems[i].OrderID = order.ID
	}

	if err := tx.Create(&orderItems).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order items"})
		return
	}

	// Atomically reserve daily capacity for capped dishes (#48) — oversell-safe.
	capDay := services.CapacityDay(time.Now())
	for _, cr := range capReservations {
		if err := services.ReserveCapacity(tx, req.ChefID, cr.itemID, cr.qty, cr.cap, capDay); err != nil {
			tx.Rollback()
			if errors.Is(err, services.ErrSoldOut) {
				c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("%s is sold out for today", cr.name)})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reserve capacity"})
			return
		}
	}

	// Reserve the scheduled delivery slot's per-day capacity (#51) — one booking
	// per order, keyed to its delivery day. Oversell-safe.
	if slotToReserve != "" {
		if err := services.ReserveSlot(tx, req.ChefID, slotToReserve, 1, slotCap, slotBookingDay); err != nil {
			tx.Rollback()
			if errors.Is(err, services.ErrSlotFull) {
				c.JSON(http.StatusConflict, gin.H{"error": "That delivery slot just filled up. Please pick another time."})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reserve delivery slot"})
			return
		}
	}

	// Record promo code usage. Claim a redemption slot ATOMICALLY first: the
	// conditional WHERE re-checks the global usage limit and the budget cap inside
	// the UPDATE, so concurrent checkouts can't push usage_count past usage_limit
	// or budget_spent past budget_cap (#39). If the slot is gone (claimed by a
	// racing order between validation and now), roll back so we never charge full
	// price silently — the customer re-validates.
	if appliedPromo != nil {
		claimed, claimErr := services.ClaimPromoRedemption(tx, appliedPromo.ID, discount)
		if claimErr != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply promo code"})
			return
		}
		if !claimed {
			tx.Rollback()
			c.JSON(http.StatusConflict, gin.H{"error": "This promo code is no longer available"})
			return
		}
		// Per-user cap, race-safe: the claim above locked the promo row, so this
		// count sees all committed prior redemptions and two concurrent same-user
		// orders can't both slip past a "N per user" limit (#39). Roll back the
		// claim's increments if the user is already at their limit.
		if appliedPromo.PerUserLimit > 0 &&
			services.UserPromoRedemptions(tx, appliedPromo.ID, userID) >= int64(appliedPromo.PerUserLimit) {
			tx.Rollback()
			c.JSON(http.StatusConflict, gin.H{"error": "You have already used this promo code the maximum number of times"})
			return
		}
		usage := models.PromoCodeUsage{
			PromoCodeID: appliedPromo.ID,
			UserID:      userID,
			OrderID:     &order.ID,
			Discount:    discount,
		}
		if err := tx.Create(&usage).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record promo usage"})
			return
		}
	}

	// Clear user's cart for this chef
	tx.Where("user_id = ? AND chef_id = ?", userID, chef.ID).Delete(&models.Cart{})

	// Stage order events in the SAME transaction (transactional outbox) so they
	// can never be lost between commit and publish (#131). The relay publishes
	// them to JetStream with PubAck after commit.
	orderEvent := services.OrderEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		CustomerID:  order.CustomerID,
		ChefID:      order.ChefID,
		Status:      string(order.Status),
		Total:       order.Total,
	}
	if err := services.EnqueueOrderEvent(tx, services.SubjectOrderCreated, orderEvent); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record order event"})
		return
	}
	// NOTE: the actionable chef "new order" push (SubjectChefNewOrder) is NOT
	// enqueued here. Orders are created BEFORE payment, so notifying the chef at
	// creation pushed unpaid/abandoned orders to the kitchen. It now fires at
	// payment confirmation instead — see notifyChefNewOrderTx wired into each
	// paid-transition path in handlers/payment.go.

	tx.Commit()

	// Load the created order with items
	database.DB.Preload("Items").First(&order, order.ID)

	// Record metrics (events now flow durably via the outbox relay)
	middleware.RecordOrder(string(order.Status), order.Total)

	c.JSON(http.StatusCreated, order.ToResponse())
}

// GetOrders returns the user's orders
func (h *OrderHandler) GetOrders(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	query := database.DB.Where("customer_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&models.Order{}).Count(&total)

	var orders []models.Order
	if err := query.Preload("Items").Preload("Chef").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}

	responses := make([]models.OrderResponse, len(orders))
	for i, order := range orders {
		responses[i] = order.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
			"hasNext":    int64(offset+limit) < total,
			"hasPrev":    page > 1,
		},
	})
}

// GetOrder returns a single order
func (h *OrderHandler) GetOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("id")

	var order models.Order
	if err := database.DB.Preload("Items").Preload("Chef").Preload("Delivery").
		Where("id = ? AND customer_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, order.ToResponse())
}

// CancelOrder cancels an order
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("id")

	var order models.Order
	if err := database.DB.Preload("Items").Where("id = ? AND customer_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Check if order can be cancelled
	if order.Status != models.OrderStatusPending && order.Status != models.OrderStatusAccepted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be cancelled at this stage"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	now := time.Now()
	order.Status = models.OrderStatusCancelled
	order.CancelledAt = &now
	order.CancelReason = req.Reason

	// Persist the cancellation and stage the event atomically (transactional
	// outbox): the order.cancelled event is delivered durably by the relay (#131).
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		// Release the reserved daily capacity back to the chef (#48), keyed to the
		// order's original IST day so a next-day cancel doesn't touch today's count.
		capDay := services.CapacityDay(order.CreatedAt)
		for _, it := range order.Items {
			if err := services.ReleaseCapacity(tx, it.MenuItemID, it.Quantity, capDay); err != nil {
				return err
			}
		}
		// Release the scheduled delivery-slot booking (#51), keyed to the order's
		// scheduled delivery day (not CreatedAt — the slot may be for a future day).
		if order.DeliverySlot != "" && order.ScheduledFor != nil {
			if err := services.ReleaseSlot(tx, order.ChefID, order.DeliverySlot, 1, services.CapacityDay(*order.ScheduledFor)); err != nil {
				return err
			}
		}
		return services.EnqueueOrderEvent(tx, services.SubjectOrderCancelled, services.OrderEvent{
			OrderID:     order.ID,
			OrderNumber: order.OrderNumber,
			CustomerID:  order.CustomerID,
			ChefID:      order.ChefID,
			Status:      string(order.Status),
			Total:       order.Total,
		})
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order"})
		return
	}

	// Cancel any booked 3PL delivery (no-op if none exists yet). Off the
	// response path; failure must not fail the order cancellation.
	go func() {
		if err := services.CancelOrderDelivery(order.ID, req.Reason); err != nil {
			log.Printf("Failed to cancel 3PL delivery for order %s: %v", order.ID, err)
			services.CaptureBackgroundError(err)
		}
	}()

	c.JSON(http.StatusOK, order.ToResponse())
}

// chefTrackCoords returns the chef coordinates to show the customer for an order.
// Pickup reveals the EXACT kitchen (the customer is collecting); every other mode
// returns an approximate (fuzzed) point so the address stays private.
func chefTrackCoords(order models.Order) (lat, lng float64, exact bool) {
	if order.FulfillmentType == models.FulfillmentPickup {
		return order.Chef.Latitude, order.Chef.Longitude, true
	}
	flat, flng := services.FuzzCoordinate(order.Chef.Latitude, order.Chef.Longitude, order.Chef.ID.String())
	return flat, flng, false
}

// TrackOrder returns real-time tracking info for an order
func (h *OrderHandler) TrackOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("id")

	var order models.Order
	if err := database.DB.Preload("Delivery").Preload("Delivery.DeliveryPartner").Preload("Chef").
		Where("id = ? AND customer_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	chefLat, chefLng, chefExact := chefTrackCoords(order)

	fulfillment := order.FulfillmentType
	if fulfillment == "" {
		fulfillment = models.FulfillmentDelivery
	}

	response := gin.H{
		"orderId":         order.ID,
		"orderNumber":     order.OrderNumber,
		"status":          order.Status,
		"fulfillmentType": fulfillment,
		// Food-ready photo (the prepared dish) for the tracking screen.
		"readyPhotoUrl": order.ReadyPhotoURL,
		"chef": func() gin.H {
			m := gin.H{
				"name":      order.Chef.BusinessName,
				"latitude":  chefLat,
				"longitude": chefLng,
			}
			if chefExact {
				// Pickup: the customer needs the real address to collect.
				var parts []string
				for _, p := range []string{
					order.Chef.AddressLine1,
					order.Chef.AddressLine2,
					order.Chef.City,
					order.Chef.State,
					order.Chef.PostalCode,
				} {
					if p != "" {
						parts = append(parts, p)
					}
				}
				m["address"] = strings.Join(parts, ", ")
			}
			return m
		}(),
		"estimatedPrepTime":     order.EstimatedPrepTime,
		"estimatedDeliveryTime": order.EstimatedDeliveryTime,
		"createdAt":             order.CreatedAt,
		"acceptedAt":            order.AcceptedAt,
		"preparedAt":            order.PreparedAt,
		"pickedUpAt":            order.PickedUpAt,
		"deliveredAt":           order.DeliveredAt,
	}

	if order.Delivery != nil {
		response["delivery"] = order.Delivery.ToResponse()
	} else {
		// No 3PL delivery record yet (pre-dispatch). Still surface the dropoff
		// coords from the order so the map can show chef + destination before a
		// driver is assigned (otherwise the client falls back to a country-wide view).
		response["delivery"] = gin.H{
			"dropoffLatitude":  order.DeliveryLatitude,
			"dropoffLongitude": order.DeliveryLongitude,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetOrderInvoice returns the invoice for an order, generating it on the fly if needed
func (h *OrderHandler) GetOrderInvoice(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("id")

	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	// Verify the order belongs to this user
	var order models.Order
	if err := database.DB.Where("id = ? AND customer_id = ?", orderUUID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	invoice, err := services.GetOrderInvoiceByOrderID(orderUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, invoice.ToResponse())
}

// GetOrderInvoicePDF streams the PDF tax invoice for a delivered order.
// Customer-scoped (only the buyer can pull their own invoice).
// GET /api/v1/orders/:id/invoice.pdf
func (h *OrderHandler) GetOrderInvoicePDF(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("id")

	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND customer_id = ?", orderUUID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if order.Status != models.OrderStatusDelivered {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invoice available after delivery"})
		return
	}

	pdfBytes, filename, err := services.GenerateOrderInvoicePDF(orderUUID)
	if err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate invoice"})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// TrackOrderWS upgrades the HTTP connection to WebSocket and streams real-time driver location
// updates from NATS delivery.location.{deliveryID} to the connected client.
// Security: verifies the authenticated customer owns the order before upgrading (T-04-03).
func (h *OrderHandler) TrackOrderWS(c *gin.Context) {
	orderID := c.Param("id")
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Verify customer owns this order and load the delivery relationship.
	var order models.Order
	if err := database.DB.Preload("Delivery").
		Where("id = ? AND customer_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order_not_found", "message": "Order not found"})
		return
	}
	if order.Delivery == nil || order.Delivery.ID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no_active_delivery", "message": "No active delivery for this order"})
		return
	}
	deliveryID := order.Delivery.ID.String()

	// Upgrade HTTP → WebSocket.
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed for order %s: %v", orderID, err)
		return // Upgrader writes the error response automatically.
	}
	defer conn.Close()

	// Buffered write channel — only one goroutine calls conn.WriteMessage (gorilla is not concurrent-write-safe).
	// T-04-05: non-blocking send drops messages when full rather than blocking.
	writeCh := make(chan []byte, 32)
	defer close(writeCh)

	// Write pump — serialises all WebSocket writes.
	go func() {
		for msg := range writeCh {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("WS write error for order %s: %v", orderID, err)
				return
			}
		}
	}()

	// Subscribe to core NATS (not JetStream) for live location fan-out.
	subject := fmt.Sprintf("%s.%s", services.SubjectDeliveryLocation, deliveryID)
	sub, err := services.GetNATSClient().Subscribe(subject, func(msg *natsclient.Msg) {
		select {
		case writeCh <- msg.Data:
		default:
			// Channel full — drop this message rather than block the NATS dispatcher.
		}
	})
	if err != nil {
		log.Printf("NATS subscribe failed for delivery %s: %v", deliveryID, err)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1011, "Internal error"))
		return
	}
	defer sub.Unsubscribe()

	// Read pump — blocks until client disconnects or sends a close frame.
	// T-04-05: SetReadLimit caps inbound frame size at 512 bytes.
	conn.SetReadLimit(512)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// Helper to generate order number
func generateOrderNumber() string {
	timestamp := time.Now().Format("0601021504")
	random := rand.Intn(9999)
	return fmt.Sprintf("HC%s%04d", timestamp, random)
}
