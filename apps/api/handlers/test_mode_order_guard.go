package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// errTestChefNotOrderable is returned when a customer who is not on the
// test-mode viewer allowlist tries to transact with a sandbox kitchen.
//
// The copy is deliberately the same "not accepting orders" language a real
// closed kitchen uses: a stranger who somehow reached a test kitchen learns
// nothing about test mode existing.
var errTestChefNotOrderable = errors.New("this kitchen is not accepting orders right now")

// assertMayOrderFromChef is the last line of defence for test-mode isolation.
//
// Discovery filtering is what normally keeps a sandbox kitchen out of sight,
// but discovery is spread across nine endpoints and will grow. This check sits
// on the handful of CREATE paths instead, where the number of doors is small
// and fixed — so a discovery surface someone forgets to filter can never become
// a real customer transacting with a fake kitchen.
//
// Note it demands FULL visibility, not merely "not hidden": an established
// kitchen flipped to test shows as closed to its regulars, and closed means
// closed.
func assertMayOrderFromChef(c *gin.Context, chef *models.ChefProfile) error {
	if services.ChefVisibility(chef, viewerEmail(c)) != services.VisibilityFull {
		return errTestChefNotOrderable
	}
	return nil
}
