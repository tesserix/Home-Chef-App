package services

import (
	"fmt"
	"strings"
	"time"
)

// NewDeviceLogin is what a sign-in security email describes: which install, on
// what, from roughly where, and when.
type NewDeviceLogin struct {
	App      string
	Platform string
	Label    string
	Location IPLocation
	IP       string
	DeviceID string
	Signed   time.Time
}

var appDisplayNames = map[string]string{
	"customer": "Fe3dr",
	"vendor":   "Fe3dr for Chefs",
	"chef":     "Fe3dr for Chefs",
	"delivery": "Fe3dr for Drivers",
	"admin":    "Fe3dr Admin",
}

// NewDeviceLoginHTML renders the "we saw a sign-in from a device you haven't
// used before" email.
func NewDeviceLoginHTML(firstName string, in NewDeviceLogin) (subject, html string) {
	subject = "New sign-in to your Fe3dr account"
	greeting := "there"
	if strings.TrimSpace(firstName) != "" {
		greeting = esc(firstName)
	}

	rows := []struct{ label, value string }{
		{"Device", in.deviceDescription()},
		{"App", in.appDescription()},
		{"Location", in.Location.Describe()},
		{"IP address", firstNonEmpty(in.IP, "Unknown")},
		{"When", in.Signed.UTC().Format("2 Jan 2006, 15:04") + " UTC"},
	}
	var detail strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&detail,
			`<p style="margin:0 0 6px 0;"><strong style="color:#111827;">%s:</strong> %s</p>`,
			r.label, esc(r.value))
	}

	body := fmt.Sprintf(`
          <h2>New sign-in, %s</h2>
          <p>Your Fe3dr account was just signed in on a device we haven't seen before. If that was you, nothing more is needed — you stay signed in everywhere else too.</p>
          <div class="info-box">
            %s
          </div>
          <p>Didn't recognise this? Change your password now and you'll be asked to sign in again on every device.</p>
          <a href="https://fe3dr.com/forgot-password" class="btn">Secure my account</a>
          <hr class="divider">
          <p class="muted">Location is approximate and derived from the network address, so it may show a nearby city or your mobile provider's.</p>
`, greeting, detail.String())

	html = emailBase(subject, "A new device just signed in to your Fe3dr account", body)
	return
}

func (in NewDeviceLogin) deviceDescription() string {
	parts := make([]string, 0, 3)
	if in.Label != "" {
		parts = append(parts, in.Label)
	}
	if in.Platform != "" {
		parts = append(parts, in.Platform)
	}
	if tail := deviceIDTail(in.DeviceID); tail != "" {
		parts = append(parts, "…"+tail)
	}
	return firstNonEmpty(strings.Join(parts, " · "), "Unknown device")
}

func (in NewDeviceLogin) appDescription() string {
	if name, ok := appDisplayNames[strings.ToLower(in.App)]; ok {
		return name
	}
	return "Fe3dr"
}

// deviceIDTail keeps enough of the install id for a user to tell two devices
// apart without reproducing an identifier they could be talked into sharing.
func deviceIDTail(id string) string {
	if len(id) <= 6 {
		return id
	}
	return id[len(id)-6:]
}

func firstNonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
