package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/middleware"
)

// WSTicketHandler mints short-lived WebSocket tickets for the authenticated
// caller. Registered on the normal authenticated REST group, so the caller is
// already identified (BFF HMAC or Bearer session) before a ticket exists.
type WSTicketHandler struct{ hmacKey []byte }

func NewWSTicketHandler(hmacKey []byte) *WSTicketHandler {
	return &WSTicketHandler{hmacKey: hmacKey}
}

// Mint — POST /v1/realtime/ws-ticket. Returns a ticket the client appends to a
// /ws/* URL as `?ticket=`. The identity is taken from the authenticated
// request, never from the body, so a caller can only ever mint for itself.
func (h *WSTicketHandler) Mint(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	email, _ := c.Get(middleware.CtxUserEmail)
	emailStr, _ := email.(string)
	role, _ := middleware.GetUserRole(c)
	pool, _ := middleware.GetAuthPool(c)

	ticket, err := middleware.MintWSTicket(h.hmacKey, middleware.BFFIdentity{
		UserID: userID.String(),
		Email:  emailStr,
		Role:   string(role),
		Pool:   string(pool),
	}, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not mint ticket"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ticket": ticket,
		// Seconds, so the client can refresh before expiry rather than
		// discovering it on a failed upgrade.
		"expires_in": int(middleware.WSTicketTTL.Seconds()),
	})
}
