package services

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

type capturedMail struct {
	to      string
	subject string
	body    string
}

type recordingMailer struct{ sent []capturedMail }

func (m *recordingMailer) Send(_ context.Context, to, subject, body string) error {
	m.sent = append(m.sent, capturedMail{to, subject, body})
	return nil
}

func (m *recordingMailer) SendWithAttachment(ctx context.Context, to, subject, body string, _ Attachment) error {
	return m.Send(ctx, to, subject, body)
}

func withRecordingMailer(t *testing.T) *recordingMailer {
	t.Helper()
	svc := GetEmailService()
	prev := svc.mailer
	m := &recordingMailer{}
	svc.mailer = m
	t.Cleanup(func() { svc.mailer = prev })
	return m
}

func loginUser() models.User {
	return models.User{ID: uuid.New(), Email: "asha@example.com", FirstName: "Asha"}
}

func TestNoteLoginDevice_EmailsOnAnUnrecognisedDevice(t *testing.T) {
	db := setupDeviceDB(t)
	withMiniredis(t)
	geoProvider(t, `{"status":"success","country":"India","countryCode":"IN","city":"Bengaluru"}`, http.StatusOK)
	mail := withRecordingMailer(t)
	u := loginUser()

	NoteLoginDevice(db, u, LoginSighting{
		DeviceID: "device-a", App: "customer", Platform: "ios", Label: "iPhone", IP: "49.207.1.1",
	})

	require.Len(t, mail.sent, 1)
	require.Equal(t, u.Email, mail.sent[0].to)
	require.Contains(t, mail.sent[0].body, "Bengaluru, India")
}

func TestNoteLoginDevice_StaysQuietOnAKnownDevice(t *testing.T) {
	db := setupDeviceDB(t)
	withMiniredis(t)
	mail := withRecordingMailer(t)
	u := loginUser()
	in := LoginSighting{DeviceID: "device-a", App: "customer"}

	NoteLoginDevice(db, u, in)
	NoteLoginDevice(db, u, in)

	require.Len(t, mail.sent, 1, "only the first sign-in on a device is worth an email")
}

// Signing in on a second device is the case this whole change exists for: it
// must be recorded and announced, not treated as a takeover of the first.
func TestNoteLoginDevice_EmailsAgainForASecondDevice(t *testing.T) {
	db := setupDeviceDB(t)
	withMiniredis(t)
	mail := withRecordingMailer(t)
	u := loginUser()

	NoteLoginDevice(db, u, LoginSighting{DeviceID: "device-a", App: "customer"})
	NoteLoginDevice(db, u, LoginSighting{DeviceID: "device-b", App: "customer"})

	require.Len(t, mail.sent, 2)
	var count int64
	require.NoError(t, db.Model(&models.UserDevice{}).Where("user_id = ?", u.ID).Count(&count).Error)
	require.Equal(t, int64(2), count, "both devices stay registered")
}

// A brand-new account's first device is not news — the user is looking at the
// app they just signed up on.
func TestNoteLoginDevice_SkipsTheEmailOnSignup(t *testing.T) {
	db := setupDeviceDB(t)
	withMiniredis(t)
	mail := withRecordingMailer(t)

	NoteLoginDevice(db, loginUser(), LoginSighting{DeviceID: "device-a", App: "customer", FirstLogin: true})

	require.Empty(t, mail.sent)
	var count int64
	require.NoError(t, db.Model(&models.UserDevice{}).Count(&count).Error)
	require.Equal(t, int64(1), count, "the device is still registered, only the email is skipped")
}

// Clients that don't yet send a device id must not all collapse onto one
// shared row, and must never be emailed about a device we can't identify.
func TestNoteLoginDevice_IgnoresASightingWithNoDeviceID(t *testing.T) {
	db := setupDeviceDB(t)
	withMiniredis(t)
	mail := withRecordingMailer(t)

	NoteLoginDevice(db, loginUser(), LoginSighting{App: "customer", IP: "49.207.1.1"})

	require.Empty(t, mail.sent)
	var count int64
	require.NoError(t, db.Model(&models.UserDevice{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestNoteLoginDevice_StoresTheResolvedOrigin(t *testing.T) {
	db := setupDeviceDB(t)
	withMiniredis(t)
	geoProvider(t, `{"status":"success","country":"India","countryCode":"IN","city":"Bengaluru"}`, http.StatusOK)
	withRecordingMailer(t)
	u := loginUser()

	NoteLoginDevice(db, u, LoginSighting{DeviceID: "device-a", App: "customer", IP: "49.207.1.1"})

	var d models.UserDevice
	require.NoError(t, db.First(&d).Error)
	require.Equal(t, "IN", d.LastCountry)
	require.Equal(t, "Bengaluru", d.LastCity)
	require.Equal(t, "49.207.1.1", d.LastIP)
}
