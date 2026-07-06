package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// group_order.go — group / office orders (#46). A host starts a shared cart from
// one chef, invites others via a link, each participant adds their own items and
// pays their split share; it consolidates into ONE Order with a single delivery.
// Guardrails: single chef, participant cap, item-owner isolation (404 not 403),
// host-only lock/cancel, status-guarded transitions, expiry sweep, feature flag.

const (
	groupOrderTTL      = 24 * time.Hour // open carts auto-expire
	groupOrderLeadTime = 0              // group orders are immediate (no future lead like tiffin)
)

type GroupOrderHandler struct{}

func NewGroupOrderHandler() *GroupOrderHandler { return &GroupOrderHandler{} }

// groupOrdersEnabled is true when EITHER the startup env flag
// (GROUP_ORDERS_ENABLED) or the runtime platform_policy override is on. The
// policy override lets admins flip group orders from the console without a
// redeploy — env-only flags were a dead toggle in the admin UI before this.
func groupOrdersEnabled() bool {
	if config.AppConfig != nil && config.AppConfig.GroupOrdersEnabled {
		return true
	}
	return services.GetPlatformPolicy().GroupOrdersEnabled
}

func generateJoinToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func userDisplayName(userID uuid.UUID) string {
	var u models.User
	if err := database.DB.First(&u, "id = ?", userID).Error; err != nil {
		return "A guest"
	}
	name := fmt.Sprintf("%s %s", u.FirstName, u.LastName)
	if len(name) > 1 {
		return name
	}
	// Never fall back to the email — the display name is shown to every
	// co-participant, so a nameless user would leak their email to strangers.
	return "A guest"
}

// scrubGroupChef replaces the preloaded raw ChefProfile on a group order with a
// minimal, privacy-safe projection before it is serialized to customers. The raw
// profile exposes the (often home) chef's exact street address, GPS, FSSAI and
// GSTIN; group members only need the business identity and an approximate area —
// matching what the public chef listing exposes (fuzzed geo, no street address).
func scrubGroupChef(g *models.GroupOrder) {
	if g == nil || g.Chef == nil {
		return
	}
	lat, lng := services.FuzzCoordinate(g.Chef.Latitude, g.Chef.Longitude, g.Chef.ID.String())
	g.Chef = &models.ChefProfile{
		ID:           g.Chef.ID,
		BusinessName: g.Chef.BusinessName,
		ProfileImage: g.Chef.ProfileImage,
		City:         g.Chef.City,
		State:        g.Chef.State,
		Latitude:     lat,
		Longitude:    lng,
	}
}

// loadGroupForParticipant loads a group + the caller's participant row, enforcing
// that the caller is a participant (IDOR-safe: 404 if not theirs).
func loadGroupForParticipant(groupID uuid.UUID, userID uuid.UUID) (models.GroupOrder, models.GroupOrderParticipant, bool) {
	var g models.GroupOrder
	if err := database.DB.Preload("Participants").Preload("Items").Preload("Chef").
		First(&g, "id = ?", groupID).Error; err != nil {
		return g, models.GroupOrderParticipant{}, false
	}
	for i := range g.Participants {
		if g.Participants[i].UserID == userID {
			return g, g.Participants[i], true
		}
	}
	return g, models.GroupOrderParticipant{}, false
}

// ───────────────────────── Create / join ─────────────────────────

type createGroupOrderRequest struct {
	ChefID      string `json:"chefId"`
	Type        string `json:"type"`      // office | personal
	SplitMode   string `json:"splitMode"` // split | host
	Title       string `json:"title"`
	CompanyName string `json:"companyName"`
}

// CreateGroupOrder — POST /group-orders. The host starts a shared cart from a chef.
func (h *GroupOrderHandler) CreateGroupOrder(c *gin.Context) {
	if !groupOrdersEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Group orders aren't available yet"})
		return
	}
	hostID, _ := middleware.GetUserID(c)
	var req createGroupOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	chefID, err := uuid.Parse(req.ChefID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chefId"})
		return
	}
	gType := models.GroupOrderPersonal
	if req.Type == string(models.GroupOrderOffice) {
		gType = models.GroupOrderOffice
	}
	splitMode := models.GroupSplitEqualByItems
	if req.SplitMode == string(models.GroupSplitHostPays) {
		splitMode = models.GroupSplitHostPays
	}

	var chef models.ChefProfile
	if err := database.DB.Where("id = ? AND is_active = ?", chefID, true).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	if services.IsChefFSSAIExpired(&chef) {
		c.JSON(http.StatusConflict, gin.H{"error": "This chef isn't accepting orders right now"})
		return
	}

	now := time.Now()
	g := models.GroupOrder{
		HostID:      hostID,
		ChefID:      chefID,
		Type:        gType,
		SplitMode:   splitMode,
		Title:       req.Title,
		CompanyName: req.CompanyName,
		JoinToken:   generateJoinToken(),
		Status:      models.GroupOrderOpen,
		Currency:    services.CurrencyForCountry(chef.PayoutCountry),
		ExpiresAt:   now.Add(groupOrderTTL),
	}
	host := models.GroupOrderParticipant{
		UserID:      hostID,
		Role:        models.GroupRoleHost,
		DisplayName: userDisplayName(hostID),
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&g).Error; err != nil {
			return err
		}
		host.GroupOrderID = g.ID
		return tx.Create(&host).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group order"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"groupOrder": g,
		"joinToken":  g.JoinToken,
		"joinUrl":    fmt.Sprintf("homechef-customer://group/%s", g.JoinToken),
	})
}

// GroupJoinPreview — GET /group-invites/:token (PUBLIC). Lets an invitee see what
// they're joining before authenticating.
func (h *GroupOrderHandler) GroupJoinPreview(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}
	var g models.GroupOrder
	if err := database.DB.Preload("Chef").Preload("Participants").
		Where("join_token = ?", token).First(&g).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invite not found"})
		return
	}
	chefName := ""
	if g.Chef != nil {
		chefName = g.Chef.BusinessName
	}
	c.JSON(http.StatusOK, gin.H{
		"title":    g.Title,
		"type":     g.Type,
		"chefName": chefName,
		"chefId":   g.ChefID,
		"status":   g.Status,
		"joinable": g.Status == models.GroupOrderOpen && time.Now().Before(g.ExpiresAt),
		"hostName": hostDisplayName(&g),
	})
}

func hostDisplayName(g *models.GroupOrder) string {
	for i := range g.Participants {
		if g.Participants[i].Role == models.GroupRoleHost {
			return g.Participants[i].DisplayName
		}
	}
	return ""
}

// JoinGroupOrder — POST /group-invites/:token/accept. Authenticated join.
func (h *GroupOrderHandler) JoinGroupOrder(c *gin.Context) {
	if !groupOrdersEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Group orders aren't available yet"})
		return
	}
	userID, _ := middleware.GetUserID(c)
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	var g models.GroupOrder
	if err := database.DB.Preload("Participants").Where("join_token = ?", token).First(&g).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invite not found"})
		return
	}
	if g.Status != models.GroupOrderOpen || time.Now().After(g.ExpiresAt) {
		c.JSON(http.StatusConflict, gin.H{"error": "This group order is no longer open"})
		return
	}
	for i := range g.Participants {
		if g.Participants[i].UserID == userID {
			c.JSON(http.StatusOK, gin.H{"groupOrder": g, "alreadyJoined": true})
			return
		}
	}
	if len(g.Participants) >= models.GroupOrderMaxParticipants {
		c.JSON(http.StatusConflict, gin.H{"error": "This group order is full"})
		return
	}

	p := models.GroupOrderParticipant{
		GroupOrderID: g.ID,
		UserID:       userID,
		Role:         models.GroupRoleGuest,
		DisplayName:  userDisplayName(userID),
	}
	if err := database.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groupOrderId": g.ID, "joined": true})
}

// ───────────────────────── Read ─────────────────────────

// GetMyGroupOrders — GET /group-orders. Groups the caller hosts or joined.
func (h *GroupOrderHandler) GetMyGroupOrders(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var ids []uuid.UUID
	database.DB.Model(&models.GroupOrderParticipant{}).
		Where("user_id = ?", userID).Pluck("group_order_id", &ids)
	var groups []models.GroupOrder
	if len(ids) > 0 {
		database.DB.Preload("Chef").Where("id IN ?", ids).
			Order("created_at DESC").Find(&groups)
	}
	for i := range groups {
		scrubGroupChef(&groups[i])
	}
	c.JSON(http.StatusOK, gin.H{"data": groups})
}

// GetGroupOrder — GET /group-orders/:id. Scoped: caller must be a participant.
func (h *GroupOrderHandler) GetGroupOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	g, me, ok := loadGroupForParticipant(id, userID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group order not found"})
		return
	}
	scrubGroupChef(&g)
	resp := gin.H{"groupOrder": g, "me": me}
	// Only the host gets the invite token/link.
	if me.Role == models.GroupRoleHost {
		resp["joinToken"] = g.JoinToken
		resp["joinUrl"] = fmt.Sprintf("homechef-customer://group/%s", g.JoinToken)
	}
	c.JSON(http.StatusOK, resp)
}

// ───────────────────────── Items ─────────────────────────

type addGroupItemRequest struct {
	MenuItemID string `json:"menuItemId"`
	Quantity   int    `json:"quantity"`
	Notes      string `json:"notes"`
}

// AddGroupItem — POST /group-orders/:id/items. A participant adds their own item.
func (h *GroupOrderHandler) AddGroupItem(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	g, me, ok := loadGroupForParticipant(id, userID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group order not found"})
		return
	}
	if g.Status != models.GroupOrderOpen {
		c.JSON(http.StatusConflict, gin.H{"error": "This group order is no longer open for changes"})
		return
	}
	var req addGroupItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Quantity < 1 {
		req.Quantity = 1
	}
	menuItemID, err := uuid.Parse(req.MenuItemID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid menuItemId"})
		return
	}
	var item models.MenuItem
	if err := database.DB.Where("id = ? AND chef_id = ? AND is_available = ?", menuItemID, g.ChefID, true).
		First(&item).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Item not available from this chef"})
		return
	}
	// Capacity backstop (#48): don't let a sold-out dish into the shared cart.
	// (Group orders consolidate after payment, so this add-time guard is the
	// enforcement point — full reservation-at-consolidation is a tracked follow-up.)
	if _, soldOut := services.RemainingToday(item.ID, item.DailyCapacity, services.CapacityDay(time.Now())); soldOut {
		c.JSON(http.StatusConflict, gin.H{"error": item.Name + " is sold out for today"})
		return
	}
	gi := models.GroupOrderItem{
		GroupOrderID:  g.ID,
		ParticipantID: me.ID,
		MenuItemID:    item.ID,
		Name:          item.Name,
		Price:         item.Price,
		Quantity:      req.Quantity,
		Subtotal:      item.Price * float64(req.Quantity),
		Notes:         req.Notes,
	}
	if err := database.DB.Create(&gi).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add item"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": gi})
}

// RemoveGroupItem — DELETE /group-orders/:id/items/:itemId. Owner-only.
func (h *GroupOrderHandler) RemoveGroupItem(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item id"})
		return
	}
	g, me, ok := loadGroupForParticipant(id, userID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group order not found"})
		return
	}
	if g.Status != models.GroupOrderOpen {
		c.JSON(http.StatusConflict, gin.H{"error": "This group order is no longer open for changes"})
		return
	}
	// Owner-only: scope the delete to this participant's own item (404 if not).
	res := database.DB.Where("id = ? AND group_order_id = ? AND participant_id = ?", itemID, g.ID, me.ID).
		Delete(&models.GroupOrderItem{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove item"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"removed": true})
}

// ───────────────────────── Lock + price ─────────────────────────

type lockGroupOrderRequest struct {
	DeliveryAddressID    string  `json:"deliveryAddressId"`
	DeliveryInstructions string  `json:"deliveryInstructions"`
	ScheduledFor         *string `json:"scheduledFor"`
}

// LockGroupOrder — POST /group-orders/:id/lock. Host sets the single drop, the
// cart is priced once, and each participant's share is computed.
func (h *GroupOrderHandler) LockGroupOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	g, me, ok := loadGroupForParticipant(id, userID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group order not found"})
		return
	}
	if me.Role != models.GroupRoleHost {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the host can lock the order"})
		return
	}
	if g.Status != models.GroupOrderOpen {
		c.JSON(http.StatusConflict, gin.H{"error": "This group order is already locked"})
		return
	}
	if len(g.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Add at least one item before locking"})
		return
	}
	var req lockGroupOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Resolve the single drop from the host's saved address.
	addrID, err := uuid.Parse(req.DeliveryAddressID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A delivery address is required"})
		return
	}
	var addr models.Address
	if err := database.DB.Where("id = ? AND user_id = ?", addrID, userID).First(&addr).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Delivery address not found"})
		return
	}

	// Price the whole cart once (mirrors CreateOrder).
	var subtotal float64
	subByParticipant := map[uuid.UUID]float64{}
	for _, it := range g.Items {
		subtotal += it.Subtotal
		subByParticipant[it.ParticipantID] += it.Subtotal
	}
	if g.Chef != nil && subtotal < g.Chef.MinimumOrder {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Minimum order is %.0f", g.Chef.MinimumOrder)})
		return
	}

	policy := services.GetPlatformPolicy()
	deliveryFee := policy.BaseDeliveryFee
	if g.Chef != nil {
		if quote, ok := services.QuoteCheckoutDeliveryFee(*g.Chef, addr.City, addr.Country, addr.Latitude, addr.Longitude); ok {
			deliveryFee = quote
		}
	}
	serviceFee := subtotal * policy.ServiceFeePercent / 100

	taxRule := services.ResolveTaxRate(addr.Country, addr.State)
	taxBase := subtotal + deliveryFee + serviceFee
	var tax, total float64
	if taxRule != nil && taxRule.Inclusive {
		// Prices already include tax: derive the embedded portion.
		tax = taxBase - (taxBase / (1 + taxRule.Rate/100))
		total = taxBase
	} else if taxRule != nil {
		tax = taxBase * taxRule.Rate / 100
		total = taxBase + tax
	} else {
		total = taxBase
	}
	extras := deliveryFee + serviceFee + tax

	now := time.Now()
	// Compute shares.
	shares := map[uuid.UUID]float64{} // participant id → share
	if g.SplitMode == models.GroupSplitHostPays {
		shares[me.ID] = round2(total)
	} else {
		shares = models.SplitShares(subByParticipant, extras)
		// Participants with no items still owe nothing in split mode.
	}

	// Aggregate quantities per dish so we can reserve à-la-carte daily capacity at
	// lock (#48/#219) — pre-payment, so a sold-out dish blocks the lock rather than
	// stranding a paid group. Released if the group is cancelled.
	qtyByItem := map[uuid.UUID]int{}
	for _, it := range g.Items {
		qtyByItem[it.MenuItemID] += it.Quantity
	}
	capByItem := map[uuid.UUID]int{}
	nameByItem := map[uuid.UUID]string{}
	if len(qtyByItem) > 0 {
		ids := make([]uuid.UUID, 0, len(qtyByItem))
		for id := range qtyByItem {
			ids = append(ids, id)
		}
		var menuItems []models.MenuItem
		database.DB.Where("id IN ?", ids).Find(&menuItems)
		for _, mi := range menuItems {
			nameByItem[mi.ID] = mi.Name
			if mi.DailyCapacity != nil && *mi.DailyCapacity > 0 {
				capByItem[mi.ID] = *mi.DailyCapacity
			}
		}
	}
	capDay := services.CapacityDay(now)
	var soldOutName string

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.GroupOrder{}).
			Where("id = ? AND status = ?", g.ID, models.GroupOrderOpen).
			Updates(map[string]any{
				"status":                       models.GroupOrderLocked,
				"locked_at":                    now,
				"subtotal":                     round2(subtotal),
				"delivery_fee":                 round2(deliveryFee),
				"service_fee":                  round2(serviceFee),
				"tax":                          round2(tax),
				"tax_rate":                     taxRate(taxRule),
				"tax_name":                     taxName(taxRule),
				"total":                        round2(total),
				"delivery_address_line1":       addr.Line1,
				"delivery_address_line2":       addr.Line2,
				"delivery_address_city":        addr.City,
				"delivery_address_state":       addr.State,
				"delivery_address_postal_code": addr.PostalCode,
				"delivery_address_country":     addr.Country,
				"delivery_latitude":            addr.Latitude,
				"delivery_longitude":           addr.Longitude,
				"delivery_instructions":        req.DeliveryInstructions,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errGroupConflict
		}
		// Set each participant's share; 0-share participants are auto-completed.
		for i := range g.Participants {
			p := g.Participants[i]
			share := round2(shares[p.ID])
			upd := map[string]any{"share_amount": share}
			if share <= 0 {
				upd["payment_status"] = models.GroupPayCompleted
			}
			if err := tx.Model(&models.GroupOrderParticipant{}).Where("id = ?", p.ID).
				Updates(upd).Error; err != nil {
				return err
			}
			if share > 0 {
				if err := services.EnqueueEvent(tx, services.SubjectGroupOrderLocked, "group_order.locked", p.UserID, map[string]any{
					"group_order_id": g.ID.String(), "share": share,
				}); err != nil {
					return err
				}
			}
		}
		// Reserve daily capacity for capped dishes (#219) — oversell-safe; a
		// sold-out dish aborts the lock so no one pays for a dish that can't be made.
		for itemID, capLimit := range capByItem {
			if err := services.ReserveCapacity(tx, g.ChefID, itemID, qtyByItem[itemID], capLimit, capDay); err != nil {
				if err == services.ErrSoldOut {
					soldOutName = nameByItem[itemID]
					return errGroupSoldOut
				}
				return err
			}
		}
		return nil
	}); err != nil {
		if err == errGroupSoldOut {
			c.JSON(http.StatusConflict, gin.H{"error": soldOutName + " is sold out for today — remove it before locking"})
			return
		}
		if err == errGroupConflict {
			c.JSON(http.StatusConflict, gin.H{"error": "This group order is already locked"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock group order"})
		return
	}

	// Reload for the response.
	g, _, _ = loadGroupForParticipant(id, userID)
	scrubGroupChef(&g)
	c.JSON(http.StatusOK, gin.H{"groupOrder": g})
}

// ───────────────────────── Pay + consolidate ─────────────────────────

// PayGroupShare — POST /group-orders/:id/pay. Creates the Razorpay charge for the
// caller's share (split) or the full total (host pays).
func (h *GroupOrderHandler) PayGroupShare(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	g, me, ok := loadGroupForParticipant(id, userID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group order not found"})
		return
	}
	if g.Status != models.GroupOrderLocked {
		c.JSON(http.StatusConflict, gin.H{"error": "This group order isn't ready for payment"})
		return
	}
	if me.PaymentStatus == models.GroupPayCompleted {
		c.JSON(http.StatusOK, gin.H{"alreadyPaid": true})
		return
	}
	if me.ShareAmount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You have nothing to pay"})
		return
	}
	rz := services.GetRazorpay()
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}
	rzOrder, err := rz.CreateOrder(&services.OrderRequest{
		Amount:   services.ToPaise(me.ShareAmount),
		Currency: g.Currency,
		Receipt:  fmt.Sprintf("GRP-%s-%s", shortID(g.ID), shortID(me.ID)),
		Notes:    map[string]string{"purpose": "group_order", "group_order_id": g.ID.String(), "participant_id": me.ID.String()},
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Could not start payment"})
		return
	}
	database.DB.Model(&models.GroupOrderParticipant{}).Where("id = ?", me.ID).
		Update("razorpay_order_id", rzOrder.ID)

	c.JSON(http.StatusCreated, gin.H{
		"razorpayOrderId": rzOrder.ID,
		"razorpayKeyId":   rz.GetKeyID(),
		"amount":          rzOrder.Amount,
		"currency":        g.Currency,
	})
}

type verifyGroupShareRequest struct {
	RazorpayPaymentID string `json:"razorpayPaymentId"`
	RazorpayOrderID   string `json:"razorpayOrderId"`
	RazorpaySignature string `json:"razorpaySignature"`
}

// VerifyGroupShare — POST /group-orders/:id/pay/verify. Confirms the caller's
// payment; when everyone required has paid, consolidates into one Order.
func (h *GroupOrderHandler) VerifyGroupShare(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	var req verifyGroupShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, me, ok := loadGroupForParticipant(id, userID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group order not found"})
		return
	}
	rz := services.GetRazorpay()
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}
	if me.PaymentStatus != models.GroupPayCompleted {
		payment, err := rz.FetchPayment(req.RazorpayPaymentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify payment"})
			return
		}
		if payment.Status != "captured" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payment not captured"})
			return
		}
		if payment.OrderID != me.RazorpayOrderID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID mismatch"})
			return
		}
		// SECURITY (#395·4): bind the captured amount + Checkout signature to THIS
		// participant's share, mirroring the main-order VerifyPayment — otherwise an
		// under-amount captured payment on the share's razorpay order (payment.Amount
		// is unforgeable, from Razorpay) could mark the share paid in full, and a
		// captured payment from another order could be reused. Amount is the hard gate.
		if payment.Amount < services.ToPaise(me.ShareAmount) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payment amount does not match the share amount"})
			return
		}
		if req.RazorpaySignature != "" &&
			!services.VerifyPaymentSignature(me.RazorpayOrderID, req.RazorpayPaymentID, req.RazorpaySignature) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payment signature verification failed"})
			return
		}
		if err := database.DB.Model(&models.GroupOrderParticipant{}).
			Where("id = ? AND payment_status <> ?", me.ID, models.GroupPayCompleted).
			Updates(map[string]any{"payment_status": models.GroupPayCompleted, "razorpay_payment_id": req.RazorpayPaymentID}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record payment"})
			return
		}
	}

	// Re-check readiness with fresh data; consolidate exactly once.
	placed, err := h.maybeConsolidate(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment recorded but order placement failed; please contact support"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"paymentVerified": true, "placed": placed})
}

// maybeConsolidate places the consolidated Order if everyone required has paid.
// Returns whether placement happened. Idempotent (status-guarded).
func (h *GroupOrderHandler) maybeConsolidate(groupID uuid.UUID) (bool, error) {
	var g models.GroupOrder
	if err := database.DB.Preload("Participants").Preload("Items").Preload("Chef").
		First(&g, "id = ?", groupID).Error; err != nil {
		return false, err
	}
	if g.Status != models.GroupOrderLocked {
		return g.Status == models.GroupOrderPlaced, nil
	}
	if !groupFullyPaid(&g) {
		return false, nil
	}

	now := time.Now()
	chefAccount := ""
	if g.Chef != nil {
		chefAccount = g.Chef.RazorpayAccountID
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Build the single consolidated Order from all participants' items.
		order := models.Order{
			OrderNumber:               generateOrderNumber(),
			CustomerID:                g.HostID,
			ChefID:                    g.ChefID,
			Status:                    models.OrderStatusPending,
			PaymentStatus:             models.PaymentCompleted,
			Currency:                  g.Currency,
			Subtotal:                  g.Subtotal,
			DeliveryFee:               g.DeliveryFee,
			ServiceFee:                g.ServiceFee,
			Tax:                       g.Tax,
			TaxRate:                   g.TaxRate,
			TaxName:                   g.TaxName,
			Total:                     g.Total,
			DeliveryAddressLine1:      g.DeliveryAddressLine1,
			DeliveryAddressLine2:      g.DeliveryAddressLine2,
			DeliveryAddressCity:       g.DeliveryAddressCity,
			DeliveryAddressState:      g.DeliveryAddressState,
			DeliveryAddressPostalCode: g.DeliveryAddressPostalCode,
			DeliveryAddressCountry:    g.DeliveryAddressCountry,
			DeliveryLatitude:          g.DeliveryLatitude,
			DeliveryLongitude:         g.DeliveryLongitude,
			DeliveryInstructions:      g.DeliveryInstructions,
			EstimatedPrepTime:         30,
			ScheduledFor:              g.ScheduledFor,
		}
		// Guarded transition: only the first caller places the order.
		res := tx.Model(&models.GroupOrder{}).
			Where("id = ? AND status = ?", g.ID, models.GroupOrderLocked).
			Update("status", models.GroupOrderPlaced)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errGroupConflict // someone else consolidated
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		items := make([]models.OrderItem, 0, len(g.Items))
		for _, it := range g.Items {
			items = append(items, models.OrderItem{
				OrderID:    order.ID,
				MenuItemID: it.MenuItemID,
				Name:       it.Name,
				Price:      it.Price,
				Quantity:   it.Quantity,
				Subtotal:   it.Subtotal,
				Notes:      it.Notes,
			})
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&models.GroupOrder{}).Where("id = ?", g.ID).
			Updates(map[string]any{"order_id": order.ID, "placed_at": now}).Error; err != nil {
			return err
		}
		// Hold the chef payout (released on delivery).
		if err := services.HoldGroupChefPayout(tx, &g, chefAccount); err != nil {
			return err
		}
		// Feed the normal chef + dispatch pipeline.
		ev := services.OrderEvent{
			OrderID: order.ID, OrderNumber: order.OrderNumber,
			CustomerID: order.CustomerID, ChefID: order.ChefID,
			Status: string(order.Status), Total: order.Total,
		}
		if err := services.EnqueueOrderEvent(tx, services.SubjectOrderCreated, ev); err != nil {
			return err
		}
		if err := services.EnqueueOrderEvent(tx, services.SubjectChefNewOrder, ev); err != nil {
			return err
		}
		return services.EnqueueEvent(tx, services.SubjectGroupOrderPlaced, "group_order.placed", g.HostID, map[string]any{
			"group_order_id": g.ID.String(), "order_id": order.ID.String(),
		})
	})
	if err == errGroupConflict {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ───────────────────────── Cancel / leave ─────────────────────────

// CancelGroupOrder — POST /group-orders/:id/cancel. Host only. Reverses the chef
// payout (if held) and refunds every paid participant to wallet.
func (h *GroupOrderHandler) CancelGroupOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	g, me, ok := loadGroupForParticipant(id, userID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group order not found"})
		return
	}
	if me.Role != models.GroupRoleHost {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the host can cancel"})
		return
	}
	// A delivery-FAILED group is frozen pending admin fault resolution (#594) — the host
	// must not self-cancel it (that would force an unadjudicated refund + chef clawback and
	// lock out the admin resolver, whose claim requires status='failed').
	if g.Status == models.GroupOrderCancelled || g.Status == models.GroupOrderDelivered || g.Status == models.GroupOrderFailed {
		c.JSON(http.StatusConflict, gin.H{"error": "This group order can't be cancelled"})
		return
	}

	now := time.Now()
	// Flag-gated (#456): a no-op while OrderPayoutAutoReleaseEnabled is OFF, so cancel
	// can't move money ungated at launch. The chef payout reverse is now routed through
	// the guarded hold machine AFTER the cancel commits (reverseGroupCancelPayout below,
	// on both the success and already-cancelled recovery paths) — not the old pre-tx
	// best-effort ReverseGroupChefPayout that ran outside the status transition (#456 W-A).
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.GroupOrder{}).
			Where("id = ? AND status NOT IN ?", g.ID, []models.GroupOrderStatus{models.GroupOrderCancelled, models.GroupOrderDelivered, models.GroupOrderFailed}).
			Updates(map[string]any{"status": models.GroupOrderCancelled, "cancelled_at": now, "cancel_reason": "cancelled by host"})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errGroupConflict
		}
		for i := range g.Participants {
			if err := services.RefundGroupParticipant(tx, &g.Participants[i], "group order cancelled"); err != nil {
				return err
			}
			if err := services.EnqueueEvent(tx, services.SubjectGroupOrderCancelled, "group_order.cancelled", g.Participants[i].UserID, map[string]any{
				"group_order_id": g.ID.String(),
			}); err != nil {
				return err
			}
		}
		// Release reserved daily capacity (#219) — only a locked group reserved it.
		// ReleaseCapacity is a no-op for items with no counter row (uncapped).
		if g.LockedAt != nil {
			relQty := map[uuid.UUID]int{}
			for _, it := range g.Items {
				relQty[it.MenuItemID] += it.Quantity
			}
			relDay := services.CapacityDay(*g.LockedAt)
			for itemID, qty := range relQty {
				if err := services.ReleaseCapacity(tx, itemID, qty, relDay); err != nil {
					return err
				}
			}
		}
		// Cancel the consolidated order too if it was placed.
		if g.OrderID != nil {
			tx.Model(&models.Order{}).Where("id = ?", *g.OrderID).
				Update("status", models.OrderStatusCancelled)
		}
		return nil
	}); err != nil {
		if err == errGroupConflict {
			// Already cancelled/delivered. If it was cancelled but the payout reverse
			// never ran (e.g. a crash after the first cancel committed), recover it now —
			// ReverseGroupHoldForCancel self-guards on status==cancelled and is idempotent.
			reverseGroupCancelPayout(g.ID)
			c.JSON(http.StatusConflict, gin.H{"error": "This group order can't be cancelled"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel"})
		return
	}
	// #456 W-A: drive the chef payout hold → reversed and claw back the held direct
	// transfer through the guarded hold machine, only now that the cancel genuinely
	// committed (was an unconditional pre-tx ReverseGroupChefPayout). Best-effort; the
	// payout-reconcile cron re-drives a reversed-but-unsettled group hold.
	reverseGroupCancelPayout(g.ID)
	c.JSON(http.StatusOK, gin.H{"cancelled": true})
}

// reverseGroupCancelPayout drives a cancelled group's chef payout hold → reversed and
// claws back the held direct transfer, best-effort and off the response path. Self-
// guarding (no-op unless status==cancelled) and idempotent, so it's safe to call on
// both the cancel success path and the already-cancelled conflict/retry recovery path.
func reverseGroupCancelPayout(groupID uuid.UUID) {
	if err := services.ReverseGroupHoldForCancel(database.DB, groupID, "group order cancelled"); err != nil {
		log.Printf("group-order: reverse chef payout on cancel for %s failed: %v", groupID, err)
	}
}

// AdminResolveGroupDeliveryFailure executes the admin-confirmed money policy for a
// delivery-FAILED group order (#594 slice B). The group is frozen `disputed` by the freeze;
// the admin confirms fault — customer-fault → chef paid + group→delivered;
// platform/chef-fault → refund all participants + reverse the chef transfer + group→
// cancelled. Mirrors AdminResolveDayDeliveryFailure for the group aggregate.
func (h *GroupOrderHandler) AdminResolveGroupDeliveryFailure(c *gin.Context) {
	adminID, _ := middleware.GetUserID(c)
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group id"})
		return
	}
	var req struct {
		Fault models.DeliveryFaultClass `json:"fault" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A fault class (customer, platform, or chef) is required"})
		return
	}

	// Load the group WITH participants — the refund targets on the platform/chef-fault path.
	var g models.GroupOrder
	if err := database.DB.Preload("Participants").First(&g, "id = ?", groupID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group order not found"})
		return
	}

	switch err := services.ResolveGroupOrderFailure(database.DB, &g, req.Fault, adminID); {
	case errors.Is(err, services.ErrAmbiguousFault):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Confirm a concrete fault: customer, platform, or chef"})
	case errors.Is(err, services.ErrNotDeliveryFailure):
		c.JSON(http.StatusBadRequest, gin.H{"error": "This group is not in a delivery-failed state"})
	case errors.Is(err, services.ErrIssueAlreadyHandled):
		c.JSON(http.StatusConflict, gin.H{"error": "This group has already been resolved"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not resolve the delivery failure"})
	default:
		c.JSON(http.StatusOK, gin.H{"status": "resolved", "fault": string(req.Fault)})
	}
}

// LeaveGroupOrder — POST /group-orders/:id/leave. A guest leaves an open group
// (their items are removed). The host can't leave — they cancel instead.
func (h *GroupOrderHandler) LeaveGroupOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	g, me, ok := loadGroupForParticipant(id, userID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group order not found"})
		return
	}
	if me.Role == models.GroupRoleHost {
		c.JSON(http.StatusConflict, gin.H{"error": "The host can't leave — cancel the group order instead"})
		return
	}
	if g.Status != models.GroupOrderOpen {
		c.JSON(http.StatusConflict, gin.H{"error": "You can't leave once the order is locked"})
		return
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_order_id = ? AND participant_id = ?", g.ID, me.ID).
			Delete(&models.GroupOrderItem{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.GroupOrderParticipant{}, "id = ?", me.ID).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to leave"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"left": true})
}

// ───────────────────────── helpers ─────────────────────────

var errGroupConflict = fmt.Errorf("group order state changed concurrently")

// errGroupSoldOut aborts a lock when a capped dish is sold out for the day (#219).
var errGroupSoldOut = fmt.Errorf("a dish is sold out for today")

// groupFullyPaid reports whether every required participant has paid (split: all
// with a share; host: the host).
func groupFullyPaid(g *models.GroupOrder) bool {
	if g.SplitMode == models.GroupSplitHostPays {
		for i := range g.Participants {
			if g.Participants[i].Role == models.GroupRoleHost {
				return g.Participants[i].PaymentStatus == models.GroupPayCompleted
			}
		}
		return false
	}
	for i := range g.Participants {
		if g.Participants[i].ShareAmount > 0 && g.Participants[i].PaymentStatus != models.GroupPayCompleted {
			return false
		}
	}
	return true
}

func shortID(id uuid.UUID) string { return id.String()[:8] }

func taxRate(r *models.TaxRate) float64 {
	if r == nil {
		return 0
	}
	return r.Rate
}

func taxName(r *models.TaxRate) string {
	if r == nil {
		return ""
	}
	return r.TaxName
}
