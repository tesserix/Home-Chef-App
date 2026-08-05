package handlers

// chef_audience.go — the customer-facing like and subscribe endpoints.
//
// Every mutation returns the full audience state (viewer's own flags + public
// totals) so the app never has to follow a write with a read to repaint the
// button, and two devices cannot disagree about the count.

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"gorm.io/gorm"
)

type ChefAudienceHandler struct{}

func NewChefAudienceHandler() *ChefAudienceHandler {
	return &ChefAudienceHandler{}
}

// GetAudienceState returns the viewer's like/subscribe state and the public
// totals. GET /chefs/:id/audience
func (h *ChefAudienceHandler) GetAudienceState(c *gin.Context) {
	chefID, ok := parseChefIDParam(c)
	if !ok {
		return
	}
	// Unauthenticated is fine here: the totals are public, and the viewer's own
	// flags simply come back false.
	userID, _ := middleware.GetUserID(c)

	state, err := services.GetChefAudienceState(userID, chefID)
	if err != nil {
		respondAudienceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"audience": state})
}

// LikeChef records a like. POST /chefs/:id/like
func (h *ChefAudienceHandler) LikeChef(c *gin.Context) {
	h.mutate(c, services.LikeChef)
}

// UnlikeChef removes a like. DELETE /chefs/:id/like
func (h *ChefAudienceHandler) UnlikeChef(c *gin.Context) {
	h.mutate(c, services.UnlikeChef)
}

// Subscribe starts a subscription. POST /chefs/:id/subscribe
func (h *ChefAudienceHandler) Subscribe(c *gin.Context) {
	h.mutate(c, services.SubscribeToChef)
}

// Unsubscribe ends a subscription. DELETE /chefs/:id/subscribe
func (h *ChefAudienceHandler) Unsubscribe(c *gin.Context) {
	h.mutate(c, services.UnsubscribeFromChef)
}

// UpdateNotifyPrefs flips the per-kind flags on a subscription.
// PATCH /chefs/:id/subscribe
func (h *ChefAudienceHandler) UpdateNotifyPrefs(c *gin.Context) {
	chefID, ok := parseChefIDParam(c)
	if !ok {
		return
	}
	userID, _ := middleware.GetUserID(c)

	var req struct {
		NotifyMenu         *bool `json:"notifyMenu"`
		NotifyPriceChange  *bool `json:"notifyPriceChange"`
		NotifyAvailability *bool `json:"notifyAvailability"`
		NotifyArticles     *bool `json:"notifyArticles"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prefs := map[string]bool{}
	for kind, v := range map[string]*bool{
		models.ChefNotifyMenu:         req.NotifyMenu,
		models.ChefNotifyPriceChange:  req.NotifyPriceChange,
		models.ChefNotifyAvailability: req.NotifyAvailability,
		models.ChefNotifyArticles:     req.NotifyArticles,
	} {
		if v != nil {
			prefs[kind] = *v
		}
	}
	if len(prefs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No notification preferences supplied"})
		return
	}

	if err := services.UpdateSubscriptionNotifyPrefs(userID, chefID, prefs); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not subscribed to this kitchen"})
			return
		}
		respondAudienceError(c, err)
		return
	}

	state, err := services.GetChefAudienceState(userID, chefID)
	if err != nil {
		respondAudienceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"audience": state})
}

// ListSubscriptions returns the kitchens the customer follows.
// GET /me/subscriptions
func (h *ChefAudienceHandler) ListSubscriptions(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	subs, err := services.ListChefSubscriptions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load subscriptions"})
		return
	}

	out := make([]models.ChefSubscriptionResponse, 0, len(subs))
	for i := range subs {
		out = append(out, models.ChefSubscriptionResponse{
			ID:                 subs[i].ID,
			ChefID:             subs[i].ChefID,
			Chef:               subs[i].Chef.ToResponse(),
			NotifyMenu:         subs[i].NotifyMenu,
			NotifyPriceChange:  subs[i].NotifyPriceChange,
			NotifyAvailability: subs[i].NotifyAvailability,
			NotifyArticles:     subs[i].NotifyArticles,
			CreatedAt:          subs[i].CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"subscriptions": out, "total": len(out)})
}

// mutate runs one of the four like/subscribe operations and returns the
// resulting state — they differ only in which service call they make.
func (h *ChefAudienceHandler) mutate(c *gin.Context, op func(uuid.UUID, uuid.UUID) (models.ChefAudienceState, error)) {
	chefID, ok := parseChefIDParam(c)
	if !ok {
		return
	}
	userID, _ := middleware.GetUserID(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sign in to continue"})
		return
	}

	state, err := op(userID, chefID)
	if err != nil {
		respondAudienceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"audience": state})
}

func parseChefIDParam(c *gin.Context) (uuid.UUID, bool) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return uuid.Nil, false
	}
	return chefID, true
}

func respondAudienceError(c *gin.Context, err error) {
	if errors.Is(err, services.ErrChefNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kitchen not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
}
