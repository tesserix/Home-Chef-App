package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/services"
)

// campaign_public.go — #56. Unauthenticated endpoints an email client hits: the
// open-tracking pixel and the one-click unsubscribe. Keyed on the delivery id (a
// per-send UUID), so they're unguessable and need no session.

// trackingPixel is a 1×1 transparent GIF returned by the open-tracking endpoint.
var trackingPixel = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00,
	0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02,
	0x44, 0x01, 0x00, 0x3b,
}

// CampaignHandler serves the public campaign tracking + unsubscribe surface.
type CampaignHandler struct{}

func NewCampaignHandler() *CampaignHandler { return &CampaignHandler{} }

// TrackOpen records an email open (first hit only) and returns a 1×1 GIF.
// GET /api/v1/campaigns/track/open/:id
func (h *CampaignHandler) TrackOpen(c *gin.Context) {
	if id, err := uuid.Parse(c.Param("id")); err == nil {
		services.MarkCampaignDeliveryOpened(database.DB, id)
	}
	// Always return the pixel — never leak whether the id was valid.
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
	c.Data(http.StatusOK, "image/gif", trackingPixel)
}

// Unsubscribe withdraws the recipient's marketing consent and shows a small
// confirmation page.
// GET /api/v1/campaigns/unsubscribe/:id
func (h *CampaignHandler) Unsubscribe(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err == nil {
		_ = services.UnsubscribeMarketingByDelivery(database.DB, id)
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `<!doctype html><html><head><meta name="viewport" content="width=device-width,initial-scale=1"><title>Unsubscribed</title></head><body style="font-family:system-ui,sans-serif;max-width:480px;margin:64px auto;padding:0 24px;color:#222"><h1 style="font-size:20px">You're unsubscribed</h1><p style="color:#666">You won't receive marketing emails from Fe3dr anymore. You'll still get important account and order updates. Changed your mind? Update your preferences in the app.</p></body></html>`)
}
