package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// referral.go — customer-facing referral endpoints (#38): my code + stats,
// history, and accepting a code at signup. The reward grant lives server-side
// (services/referral_reward.go), triggered by the referee's first paid order.

type ReferralHandler struct{}

func NewReferralHandler() *ReferralHandler { return &ReferralHandler{} }

// GetMyReferral returns the caller's referral code, shareable link, the
// configured reward amounts (so the UI shows "you both get ₹X"), and their stats.
func (h *ReferralHandler) GetMyReferral(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	code, err := services.GetOrCreateReferralCode(database.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load your referral code"})
		return
	}

	cfg := services.GetReferralConfig(database.DB)
	stats := services.GetReferralStats(database.DB, userID)

	c.JSON(http.StatusOK, gin.H{
		"code":           code,
		"link":           services.ReferralLink(code),
		"enabled":        cfg.Enabled,
		"referrerReward": cfg.ReferrerReward,
		"refereeReward":  cfg.RefereeReward,
		"currency":       "INR",
		"stats":          stats,
	})
}

type referralHistoryItem struct {
	RefereeName string    `json:"refereeName"`
	Status      string    `json:"status"`
	Reward      float64   `json:"reward"`
	CreatedAt   time.Time `json:"createdAt"`
}

// GetReferralHistory lists the caller's referrals (most recent first) with the
// referee's first name only (privacy) + status + reward.
func (h *ReferralHandler) GetReferralHistory(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var refs []models.Referral
	database.DB.Where("referrer_user_id = ?", userID).Order("created_at DESC").Find(&refs)

	names := refereeFirstNames(refs)
	out := make([]referralHistoryItem, len(refs))
	for i, r := range refs {
		name := names[r.RefereeUserID]
		if name == "" {
			name = "A friend"
		}
		out[i] = referralHistoryItem{
			RefereeName: name,
			Status:      string(r.Status),
			Reward:      r.ReferrerReward,
			CreatedAt:   r.CreatedAt,
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "count": len(out)})
}

// refereeFirstNames batch-loads the referees' first names for the history view.
func refereeFirstNames(refs []models.Referral) map[uuid.UUID]string {
	out := map[uuid.UUID]string{}
	if len(refs) == 0 {
		return out
	}
	ids := make([]uuid.UUID, len(refs))
	for i, r := range refs {
		ids[i] = r.RefereeUserID
	}
	type row struct {
		ID        uuid.UUID
		FirstName string
	}
	var rows []row
	database.DB.Model(&models.User{}).Select("id, first_name").Where("id IN ?", ids).Scan(&rows)
	for _, r := range rows {
		out[r.ID] = r.FirstName
	}
	return out
}

// AcceptReferral records that the caller signed up with a referral code. Called
// once, right after signup, from the client. Idempotent on the same code.
func (h *ReferralHandler) AcceptReferral(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	// Best-effort device snapshot for fraud dedupe.
	var user models.User
	database.DB.Select("fcm_token").First(&user, "id = ?", userID)

	ref, err := services.AcceptReferral(database.DB, services.AcceptReferralInput{
		RefereeUserID: userID,
		Code:          req.Code,
		Device:        user.FCMToken,
		IP:            c.ClientIP(),
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrReferralCodeInvalid):
			c.JSON(http.StatusNotFound, gin.H{"error": "That referral code isn't valid."})
		case errors.Is(err, services.ErrReferralSelf):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrReferralAlreadyUsed):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrReferralNotNewUser):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not apply the referral code"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": string(ref.Status), "code": ref.Code})
}
