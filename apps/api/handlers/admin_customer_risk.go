package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// admin_customer_risk.go — the investigation surface for customer refund abuse (#937).
// The engine only ever scores and flags; every action that touches a real person's
// account happens here, by a named admin, with the score they saw recorded alongside.

// riskQueueRow is one line of the investigation queue: the profile plus enough identity
// to recognise who it is without a second round trip.
type riskQueueRow struct {
	models.CustomerRiskProfile
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

// GetCustomerRiskQueue lists risk profiles, worst first.
// GET /admin/customer-risk?band=&status=&minScore=&page=&limit=
func (h *AdminHandler) GetCustomerRiskQueue(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}

	q := database.DB.Model(&models.CustomerRiskProfile{})
	if band := c.Query("band"); band != "" {
		if !models.ValidRiskBand(models.RiskBand(band)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid band"})
			return
		}
		q = q.Where("band = ?", band)
	}
	if status := c.Query("status"); status != "" {
		if !models.ValidRiskStatus(models.RiskStatus(status)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
			return
		}
		q = q.Where("status = ?", status)
	}
	if minScore := c.Query("minScore"); minScore != "" {
		if v, err := strconv.ParseFloat(minScore, 64); err == nil {
			q = q.Where("score >= ?", v)
		}
	}

	var total int64
	q.Count(&total)

	var profiles []models.CustomerRiskProfile
	if err := q.Order("score DESC, updated_at DESC").
		Offset((page - 1) * limit).Limit(limit).Find(&profiles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load the risk queue"})
		return
	}

	rows := make([]riskQueueRow, 0, len(profiles))
	for _, p := range profiles {
		row := riskQueueRow{CustomerRiskProfile: p}
		var u models.User
		if err := database.DB.Select("first_name", "last_name", "email").
			First(&u, "id = ?", p.UserID).Error; err == nil {
			row.FirstName, row.LastName, row.Email = u.FirstName, u.LastName, u.Email
		}
		rows = append(rows, row)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  rows,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetCustomerRiskDetail returns one profile with the evidence behind it: the raw event
// timeline and the claims it was built from. An admin must never act on the score alone.
// GET /admin/customer-risk/:userId
func (h *AdminHandler) GetCustomerRiskDetail(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	// Recompute on open so the admin reads live numbers, not a cache from the last event.
	profile, err := services.RecomputeCustomerRisk(database.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load the risk profile"})
		return
	}

	var events []models.CustomerRiskEvent
	database.DB.Where("customer_id = ?", userID).
		Order("occurred_at DESC").Limit(200).Find(&events)

	var issues []models.OrderIssue
	database.DB.Where("customer_id = ?", userID).
		Order("created_at DESC").Limit(100).Find(&issues)

	var actions []models.CustomerRiskAction
	database.DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&actions)

	var user models.User
	database.DB.Select("id", "first_name", "last_name", "email", "phone", "is_active", "created_at").
		First(&user, "id = ?", userID)

	c.JSON(http.StatusOK, gin.H{
		"profile": profile,
		"user":    user,
		"events":  events,
		"issues":  issues,
		"actions": actions,
		"config":  services.GetCustomerRiskConfig(database.DB),
	})
}

// ReviewCustomerRisk applies an admin's decision to a profile.
// POST /admin/customer-risk/:userId/review — { status, note }
//
// `blocked` reuses the existing account suspension rather than inventing a second way to
// disable a login; clearing a block reverses exactly that, and nothing else — an account
// suspended for an unrelated reason stays suspended.
func (h *AdminHandler) ReviewCustomerRisk(c *gin.Context) {
	adminID, _ := middleware.GetUserID(c)
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A status is required"})
		return
	}
	status := models.RiskStatus(req.Status)
	if !models.ValidRiskStatus(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}
	// Restricting or blocking a real person needs a reason on the record. An admin who
	// cannot say why in one line has not finished investigating.
	if (status == models.RiskStatusRestricted || status == models.RiskStatusBlocked) && req.Note == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A note is required when restricting or blocking an account"})
		return
	}

	profile := services.GetCustomerRiskProfile(database.DB, userID)
	if profile == nil {
		if profile, err = services.RecomputeCustomerRisk(database.DB, userID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No risk profile for this user"})
			return
		}
	}

	from := profile.Status
	now := time.Now()
	profile.Status = status
	profile.StatusReason = req.Note
	profile.ReviewedAt = &now
	profile.ReviewedBy = &adminID
	if err := database.DB.Save(profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save the decision"})
		return
	}

	action := models.CustomerRiskAction{
		UserID:        userID,
		ActorID:       adminID,
		FromStatus:    from,
		ToStatus:      status,
		Note:          req.Note,
		ScoreSnapshot: profile.Score,
		BandSnapshot:  profile.Band,
	}
	if err := database.DB.Create(&action).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not record the decision"})
		return
	}

	switch {
	case status == models.RiskStatusBlocked:
		database.DB.Model(&models.User{}).Where("id = ?", userID).Update("is_active", false)
	case from == models.RiskStatusBlocked:
		database.DB.Model(&models.User{}).Where("id = ?", userID).Update("is_active", true)
	}

	services.LogAudit(c, "customer_risk.review", "user", userID.String(),
		gin.H{"status": string(from)},
		gin.H{"status": string(status), "note": req.Note, "score": profile.Score, "band": string(profile.Band)})

	c.JSON(http.StatusOK, gin.H{"profile": profile, "action": action})
}

// RecomputeCustomerRiskProfile forces a rescore from the event ledger.
// POST /admin/customer-risk/:userId/recompute
func (h *AdminHandler) RecomputeCustomerRiskProfile(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}
	profile, err := services.RecomputeCustomerRisk(database.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not recompute the profile"})
		return
	}
	c.JSON(http.StatusOK, profile)
}

// GetCustomerRiskConfig returns the effective risk policy.
func (h *AdminHandler) GetCustomerRiskConfig(c *gin.Context) {
	c.JSON(http.StatusOK, services.GetCustomerRiskConfig(database.DB))
}

// UpdateCustomerRiskConfig upserts the provided fields (partial PUT).
func (h *AdminHandler) UpdateCustomerRiskConfig(c *gin.Context) {
	adminID, _ := middleware.GetUserID(c)

	var req struct {
		Enabled                *bool    `json:"enabled"`
		EnforcementEnabled     *bool    `json:"enforcementEnabled"`
		WindowDays             *int     `json:"windowDays"`
		MinOrders              *int     `json:"minOrders"`
		ChefMinOrders          *int     `json:"chefMinOrders"`
		WatchScore             *float64 `json:"watchScore"`
		ElevatedScore          *float64 `json:"elevatedScore"`
		SevereScore            *float64 `json:"severeScore"`
		ClaimRateFull          *float64 `json:"claimRateFull"`
		RefundedShareFull      *float64 `json:"refundedShareFull"`
		SpreadFullChefs        *float64 `json:"spreadFullChefs"`
		SignalFull             *float64 `json:"signalFull"`
		WeightClaimRate        *float64 `json:"weightClaimRate"`
		WeightMoney            *float64 `json:"weightMoney"`
		WeightSpread           *float64 `json:"weightSpread"`
		WeightSignal           *float64 `json:"weightSignal"`
		ChefIssueRateFloor     *float64 `json:"chefIssueRateFloor"`
		SuppressAutoRefundBand *string  `json:"suppressAutoRefundBand"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	bools := map[string]*bool{
		"customer_risk.enabled":             req.Enabled,
		"customer_risk.enforcement_enabled": req.EnforcementEnabled,
	}
	for key, v := range bools {
		if v != nil {
			setPlatformSetting(key, strconv.FormatBool(*v), adminID)
		}
	}
	ints := map[string]*int{
		"customer_risk.window_days":     req.WindowDays,
		"customer_risk.min_orders":      req.MinOrders,
		"customer_risk.chef_min_orders": req.ChefMinOrders,
	}
	for key, v := range ints {
		if v != nil {
			setPlatformSetting(key, strconv.Itoa(*v), adminID)
		}
	}
	floats := map[string]*float64{
		"customer_risk.watch_score":           req.WatchScore,
		"customer_risk.elevated_score":        req.ElevatedScore,
		"customer_risk.severe_score":          req.SevereScore,
		"customer_risk.claim_rate_full":       req.ClaimRateFull,
		"customer_risk.refunded_share_full":   req.RefundedShareFull,
		"customer_risk.spread_full_chefs":     req.SpreadFullChefs,
		"customer_risk.signal_full":           req.SignalFull,
		"customer_risk.weight_claim_rate":     req.WeightClaimRate,
		"customer_risk.weight_money":          req.WeightMoney,
		"customer_risk.weight_spread":         req.WeightSpread,
		"customer_risk.weight_signal":         req.WeightSignal,
		"customer_risk.chef_issue_rate_floor": req.ChefIssueRateFloor,
	}
	for key, v := range floats {
		if v != nil {
			setPlatformSetting(key, strconv.FormatFloat(*v, 'f', -1, 64), adminID)
		}
	}
	if req.SuppressAutoRefundBand != nil {
		band := models.RiskBand(*req.SuppressAutoRefundBand)
		if !models.ValidRiskBand(band) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suppressAutoRefundBand"})
			return
		}
		setPlatformSetting("customer_risk.suppress_auto_refund_band", string(band), adminID)
	}

	services.LogAudit(c, "customer_risk.config_updated", "platform_settings", "customer_risk", nil, req)
	c.JSON(http.StatusOK, services.GetCustomerRiskConfig(database.DB))
}
