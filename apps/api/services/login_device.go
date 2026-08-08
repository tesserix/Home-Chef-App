package services

import (
	"context"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// LoginSighting is what a client tells us about the install it just signed in
// from. Everything but DeviceID is decoration for the security email.
type LoginSighting struct {
	DeviceID   string
	App        string
	Platform   string
	Label      string
	AppVersion string
	IP         string
	// FirstLogin marks the sign-in that created the account, whose device is
	// self-evidently the user's own and so is registered without an email.
	FirstLogin bool
}

// NoteLoginDevice registers the device this sign-in came from and, when the
// account has never used it before, emails the owner. It reports nothing:
// authentication must not depend on the device registry, the geolocation
// provider, or the mail provider, so every failure here is logged and dropped.
//
// Callers run this off the request path (see handlers/internal_users.go).
func NoteLoginDevice(db *gorm.DB, user models.User, in LoginSighting) {
	deviceID := strings.TrimSpace(in.DeviceID)
	if deviceID == "" {
		return // pre-#1164 client; nothing stable to key a device on
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	loc := LookupIPLocation(ctx, in.IP)

	seen, err := RecordDevice(db, RecordDeviceInput{
		UserID:     user.ID,
		DeviceID:   deviceID,
		App:        in.App,
		Platform:   in.Platform,
		Label:      in.Label,
		AppVersion: in.AppVersion,
		IP:         in.IP,
		Country:    loc.CountryCode,
		City:       loc.City,
	})
	if err != nil {
		log.Printf("login-device: could not record device for user %s: %v", user.ID, err)
		return
	}
	if seen.Known || in.FirstLogin || user.Email == "" {
		return
	}

	subject, html := NewDeviceLoginHTML(user.FirstName, NewDeviceLogin{
		App:      in.App,
		Platform: in.Platform,
		Label:    in.Label,
		Location: loc,
		IP:       in.IP,
		DeviceID: deviceID,
		Signed:   seen.Device.FirstSeenAt,
	})
	if err := GetEmailService().Send(user.Email, subject, html); err != nil {
		log.Printf("login-device: could not email new-device notice to user %s: %v", user.ID, err)
	}
}
