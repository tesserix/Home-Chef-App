package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// HeaderAdminMode is the header the HomeChef admin console sends to say which
// world the operator is currently looking at.
const HeaderAdminMode = "X-HomeChef-Mode"

// adminMode resolves the console's active Live/Test toggle for this request.
//
// Absent or unrecognised values resolve to live, so an older admin build — or a
// proxy that strips the header — keeps seeing exactly the real data it saw
// before this feature existed.
func adminMode(c *gin.Context) string {
	return models.NormalizeMode(c.GetHeader(HeaderAdminMode))
}

// adminModeScope filters a partitioned table to the console's active mode.
//
// Every admin list and aggregate goes through this rather than through a blanket
// "exclude test" filter, because an admin investigating a sandbox order needs to
// SEE it — just never mixed in with real ones. Live is the default, so a fake
// order can't drift into a revenue figure by accident.
func adminModeScope(c *gin.Context) func(*gorm.DB) *gorm.DB {
	return services.ModeScope(adminMode(c))
}
