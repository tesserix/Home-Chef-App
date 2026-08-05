package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/services"
)

// In-app feedback and ideas from the customer and vendor apps, filed onto the
// product backlog as labelled GitHub issues (services/github_feedback.go).

type FeedbackHandler struct {
	client *services.GitHubFeedbackClient
}

func NewFeedbackHandler() *FeedbackHandler {
	return &FeedbackHandler{client: services.NewGitHubFeedbackClient()}
}

// SubmitFeedback files one report.
// POST /api/v1/feedback
func (h *FeedbackHandler) SubmitFeedback(c *gin.Context) {
	var req struct {
		Kind       string `json:"kind" binding:"required"`
		Title      string `json:"title" binding:"required"`
		Message    string `json:"message" binding:"required"`
		App        string `json:"app"`
		Platform   string `json:"platform"`
		AppVersion string `json:"appVersion"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind != services.FeedbackKindFeedback && kind != services.FeedbackKindIdea {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kind must be 'feedback' or 'idea'"})
		return
	}
	title := strings.TrimSpace(req.Title)
	message := strings.TrimSpace(req.Message)
	if title == "" || message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A title and a message are both required"})
		return
	}
	if len(title) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title must be 200 characters or fewer"})
		return
	}
	if len(message) > 5000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message must be 5000 characters or fewer"})
		return
	}
	app := strings.ToLower(strings.TrimSpace(req.App))
	if app != "customer" && app != "vendor" && app != "delivery" {
		app = "unknown"
	}

	userID, _ := middleware.GetUserID(c)
	res, err := h.client.Submit(c.Request.Context(), services.FeedbackSubmission{
		Kind:       kind,
		Title:      title,
		Message:    message,
		App:        app,
		Platform:   strings.ToLower(strings.TrimSpace(req.Platform)),
		AppVersion: strings.TrimSpace(req.AppVersion),
		UserID:     userID.String(),
	})
	if err != nil {
		if errors.Is(err, services.ErrFeedbackNotConfigured) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Feedback isn't available right now"})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "Your feedback couldn't be sent. Please try again."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    gin.H{"reference": res.Number, "url": res.URL},
	})
}
