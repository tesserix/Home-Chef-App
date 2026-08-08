package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

// Hand-DDL'd for the same reason as setupMFADB: gen_random_uuid() defaults
// don't exist on sqlite.
func setupDeviceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	createUserDevicesTable(t, db)
	return db
}

func TestRecordDevice_FirstSightIsNew(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()

	seen, err := RecordDevice(db, RecordDeviceInput{
		UserID: uid, DeviceID: "device-a", App: "customer", Platform: "ios",
	})
	require.NoError(t, err)
	require.False(t, seen.Known, "a device never seen before must report as new")
}

func TestRecordDevice_SecondSightIsKnown(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()
	in := RecordDeviceInput{UserID: uid, DeviceID: "device-a", App: "customer", Platform: "ios"}

	_, err := RecordDevice(db, in)
	require.NoError(t, err)

	seen, err := RecordDevice(db, in)
	require.NoError(t, err)
	require.True(t, seen.Known, "a repeat sign-in must not re-trigger the security email")
}

// The whole point of the change: two devices on one account coexist.
func TestRecordDevice_SecondDeviceDoesNotDisplaceTheFirst(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()

	_, err := RecordDevice(db, RecordDeviceInput{UserID: uid, DeviceID: "device-a", App: "customer"})
	require.NoError(t, err)
	_, err = RecordDevice(db, RecordDeviceInput{UserID: uid, DeviceID: "device-b", App: "customer"})
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.UserDevice{}).Where("user_id = ?", uid).Count(&count).Error)
	require.Equal(t, int64(2), count)
}

// The same handset signed into two accounts is two rows, not a stolen one.
func TestRecordDevice_SameDeviceTwoAccountsAreSeparateRows(t *testing.T) {
	db := setupDeviceDB(t)
	a, b := uuid.New(), uuid.New()

	_, err := RecordDevice(db, RecordDeviceInput{UserID: a, DeviceID: "shared", App: "customer"})
	require.NoError(t, err)
	_, err = RecordDevice(db, RecordDeviceInput{UserID: b, DeviceID: "shared", App: "customer"})
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&models.UserDevice{}).Count(&count).Error)
	require.Equal(t, int64(2), count)
}

// Postgres stores timestamps at microsecond precision while Go carries
// nanoseconds, so a stored first_seen_at never equals the Go time that wrote
// it. Deciding "new device" by comparing against that Go value would report
// every sign-in as new and email the user each time. Simulate the truncation
// sqlite doesn't apply.
func TestRecordDevice_KnownDespiteTimestampTruncation(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()
	seeded := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	require.NoError(t, db.Create(&models.UserDevice{
		ID: uuid.New(), UserID: uid, DeviceID: "device-a", App: "customer",
		FirstSeenAt: seeded, LastSeenAt: seeded,
	}).Error)

	seen, err := RecordDevice(db, RecordDeviceInput{
		UserID: uid, DeviceID: "device-a", App: "customer",
	})
	require.NoError(t, err)
	require.True(t, seen.Known, "an existing device must not re-trigger the email")
}

// The insert path must leave first_seen_at and last_seen_at identical, because
// that equality — not a comparison against a Go timestamp the database will
// truncate — is what marks a device as new.
func TestRecordDevice_InsertLeavesTimestampsIdentical(t *testing.T) {
	db := setupDeviceDB(t)

	_, err := RecordDevice(db, RecordDeviceInput{
		UserID: uuid.New(), DeviceID: "device-a", App: "customer",
	})
	require.NoError(t, err)

	var stored models.UserDevice
	require.NoError(t, db.First(&stored).Error)
	require.Equal(t, stored.FirstSeenAt.UTC(), stored.LastSeenAt.UTC())
}

func TestRecordDevice_RefreshesLastSeenAndOrigin(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()

	_, err := RecordDevice(db, RecordDeviceInput{
		UserID: uid, DeviceID: "device-a", App: "customer", IP: "1.1.1.1", Country: "IN",
	})
	require.NoError(t, err)
	var first models.UserDevice
	require.NoError(t, db.First(&first).Error)

	time.Sleep(2 * time.Millisecond)
	_, err = RecordDevice(db, RecordDeviceInput{
		UserID: uid, DeviceID: "device-a", App: "customer", IP: "2.2.2.2", Country: "AU", City: "Sydney",
	})
	require.NoError(t, err)

	var again models.UserDevice
	require.NoError(t, db.First(&again).Error)
	require.Equal(t, "2.2.2.2", again.LastIP)
	require.Equal(t, "AU", again.LastCountry)
	require.Equal(t, "Sydney", again.LastCity)
	require.True(t, again.LastSeenAt.After(first.LastSeenAt))
	require.Equal(t, first.FirstSeenAt.UTC(), again.FirstSeenAt.UTC(), "first_seen_at must not move")
}

func TestSetDeviceToken_KeepsOtherDevicesDeliverable(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()
	longA, longB := repeatToken("a"), repeatToken("b")

	require.NoError(t, SetDeviceToken(db, uid, "device-a", "customer", longA))
	require.NoError(t, SetDeviceToken(db, uid, "device-b", "customer", longB))

	tokens, err := ActiveDeviceTokens(db, uid)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{longA, longB}, tokens)
}

func TestSetDeviceToken_EmptyClearsOnlyThatDevice(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()
	longA, longB := repeatToken("a"), repeatToken("b")
	require.NoError(t, SetDeviceToken(db, uid, "device-a", "customer", longA))
	require.NoError(t, SetDeviceToken(db, uid, "device-b", "customer", longB))

	// Signing out on device B must not silence device A.
	require.NoError(t, SetDeviceToken(db, uid, "device-b", "customer", ""))

	tokens, err := ActiveDeviceTokens(db, uid)
	require.NoError(t, err)
	require.Equal(t, []string{longA}, tokens)
}

// A handset re-used by a second account keeps only the newest owner, so a
// re-installed phone never keeps delivering to a previous account.
func TestSetDeviceToken_TokenMovesToTheNewOwner(t *testing.T) {
	db := setupDeviceDB(t)
	a, b := uuid.New(), uuid.New()
	token := repeatToken("x")

	require.NoError(t, SetDeviceToken(db, a, "shared", "customer", token))
	require.NoError(t, SetDeviceToken(db, b, "shared", "customer", token))

	oldOwner, err := ActiveDeviceTokens(db, a)
	require.NoError(t, err)
	require.Empty(t, oldOwner)

	newOwner, err := ActiveDeviceTokens(db, b)
	require.NoError(t, err)
	require.Equal(t, []string{token}, newOwner)
}

func TestActiveDeviceTokens_SkipsRevokedAndEmpty(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()
	live := repeatToken("l")
	require.NoError(t, SetDeviceToken(db, uid, "live", "customer", live))
	require.NoError(t, SetDeviceToken(db, uid, "dead", "customer", repeatToken("d")))
	require.NoError(t, SetDeviceToken(db, uid, "blank", "customer", ""))

	require.NoError(t, DropDeviceToken(db, uid, repeatToken("d")))

	tokens, err := ActiveDeviceTokens(db, uid)
	require.NoError(t, err)
	require.Equal(t, []string{live}, tokens)
}

// DropDeviceToken prunes exactly the token FCM rejected — a dead token on one
// device must not disarm the user's other devices.
func TestDropDeviceToken_LeavesSiblingsAlone(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()
	good, bad := repeatToken("g"), repeatToken("z")
	require.NoError(t, SetDeviceToken(db, uid, "good", "customer", good))
	require.NoError(t, SetDeviceToken(db, uid, "bad", "customer", bad))

	require.NoError(t, DropDeviceToken(db, uid, bad))

	tokens, err := ActiveDeviceTokens(db, uid)
	require.NoError(t, err)
	require.Equal(t, []string{good}, tokens)
}

// Referral self-dealing is caught by two accounts sharing a handset. The
// signal used to be a shared users.fcm_token; with per-device rows the handset
// itself is the stronger and more durable fingerprint.
func TestSharesDevice(t *testing.T) {
	db := setupDeviceDB(t)
	referrer, referee, stranger := uuid.New(), uuid.New(), uuid.New()

	for _, uid := range []uuid.UUID{referrer, referee} {
		_, err := RecordDevice(db, RecordDeviceInput{UserID: uid, DeviceID: "one-phone", App: "customer"})
		require.NoError(t, err)
	}
	_, err := RecordDevice(db, RecordDeviceInput{UserID: stranger, DeviceID: "other-phone", App: "customer"})
	require.NoError(t, err)

	shared, err := SharesDevice(db, referrer, referee)
	require.NoError(t, err)
	require.True(t, shared)

	shared, err = SharesDevice(db, referrer, stranger)
	require.NoError(t, err)
	require.False(t, shared)
}

// An account compared against itself trivially shares every device; treating
// that as fraud would reject on a signal that means nothing.
func TestSharesDevice_SameUserIsNotASharedDevice(t *testing.T) {
	db := setupDeviceDB(t)
	uid := uuid.New()
	_, err := RecordDevice(db, RecordDeviceInput{UserID: uid, DeviceID: "one-phone", App: "customer"})
	require.NoError(t, err)

	shared, err := SharesDevice(db, uid, uid)
	require.NoError(t, err)
	require.False(t, shared)
}

func repeatToken(seed string) string {
	out := ""
	for len(out) < 140 {
		out += seed + "0123456789"
	}
	return out
}
