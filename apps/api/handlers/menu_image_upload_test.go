package handlers

// menu_image_upload_test.go — UploadMenuItemImage hardening.
//
// Menu images are the only upload path that skipped sniffContentType, and they
// land in the PUBLIC bucket, so a payload wearing an image/jpeg header would be
// served straight back to browsers. The sniff runs before any GCS call, which
// is what makes this testable without storage.

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
)

const menuImgChefDDL = `CREATE TABLE chef_profiles (
	id text PRIMARY KEY, user_id text, business_name text,
	is_active integer DEFAULT 1, created_at datetime, updated_at datetime,
	deleted_at datetime
)`

const menuImgItemDDL = `CREATE TABLE menu_items (
	id text PRIMARY KEY, chef_id text, name text, price real,
	image_url text DEFAULT '', is_available integer DEFAULT 1,
	serves integer DEFAULT 1, sort_order integer DEFAULT 0,
	created_at datetime, updated_at datetime, deleted_at datetime
)`

const menuImgImagesDDL = `CREATE TABLE menu_item_images (
	id text PRIMARY KEY, menu_item_id text, url text,
	is_primary integer DEFAULT 0, sort_order integer DEFAULT 0,
	created_at datetime, updated_at datetime, deleted_at datetime
)`

func setupMenuImageDB(t *testing.T) (uuid.UUID, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{menuImgChefDDL, menuImgItemDDL, menuImgImagesDDL} {
		require.NoError(t, db.Exec(s).Error)
	}

	userID, chefID, itemID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, is_active) VALUES (?,?,?,1)`,
		chefID.String(), userID.String(), "Test Kitchen").Error)
	require.NoError(t, db.Exec(
		`INSERT INTO menu_items (id, chef_id, name, price) VALUES (?,?,?,?)`,
		itemID.String(), chefID.String(), "Dal Fry", 120.0).Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return userID, itemID
}

// postImage builds a multipart upload whose declared part Content-Type is
// `declared` while the actual bytes are `body`.
func postImage(t *testing.T, userID, itemID uuid.UUID, filename, declared string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="` + filename + `"`}
	h["Content-Type"] = []string{declared}
	part, err := w.CreatePart(h)
	require.NoError(t, err)
	_, err = part.Write(body)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.POST("/chef/menu/items/:itemId/images", (&MenuHandler{}).UploadMenuItemImage)

	req := httptest.NewRequest(http.MethodPost, "/chef/menu/items/"+itemID.String()+"/images", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestMenuImage_RejectsSpoofedContentType(t *testing.T) {
	userID, itemID := setupMenuImageDB(t)

	// A GIF (disallowed) declaring itself as image/jpeg. Before the fix this
	// sailed through on the header alone and was stored publicly.
	gif := append([]byte("GIF89a"), bytes.Repeat([]byte{0x00, 0x11, 0x22}, 200)...)
	rec := postImage(t, userID, itemID, "dish.jpg", "image/jpeg", gif)

	require.Equal(t, http.StatusBadRequest, rec.Code,
		"bytes that aren't an allowed image must be rejected regardless of the declared header")
	require.Contains(t, rec.Body.String(), "File contents don't match")
}

func TestMenuImage_RejectsHTMLDisguisedAsImage(t *testing.T) {
	// The public-bucket case that matters: markup served back to a browser.
	userID, itemID := setupMenuImageDB(t)
	html := []byte("<html><body><script>alert(1)</script></body></html>")
	rec := postImage(t, userID, itemID, "dish.png", "image/png", html)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "File contents don't match")
}

func TestMenuImage_RejectsDeclaredNonImageBeforeSniffing(t *testing.T) {
	userID, itemID := setupMenuImageDB(t)
	rec := postImage(t, userID, itemID, "notes.pdf", "application/pdf", []byte("%PDF-1.4 stub"))

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Invalid file type")
}

func TestMenuImage_RejectsOversizeBeforeUpload(t *testing.T) {
	userID, itemID := setupMenuImageDB(t)
	// 5 MB + 1 byte, with a real JPEG header so only the size check can trip.
	big := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x42}, 5*1024*1024)...)
	rec := postImage(t, userID, itemID, "huge.jpg", "image/jpeg", big)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "File too large")
}

func TestMenuImage_LimitIsRejectedWhenAlreadyFull(t *testing.T) {
	userID, itemID := setupMenuImageDB(t)
	for i := 0; i < maxMenuItemImages; i++ {
		require.NoError(t, database.DB.Exec(
			`INSERT INTO menu_item_images (id, menu_item_id, url, is_primary, sort_order) VALUES (?,?,?,?,?)`,
			uuid.NewString(), itemID.String(), "https://example/img.jpg", i == 0, i).Error)
	}

	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x11}, 600)...)
	rec := postImage(t, userID, itemID, "sixth.jpg", "image/jpeg", jpeg)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Maximum 5 images")
}

func TestMenuImage_RejectsForeignChefsItem(t *testing.T) {
	// Ownership is scoped before anything else — another chef's item is a 404,
	// not a 403, so the endpoint doesn't confirm the item exists.
	userID, _ := setupMenuImageDB(t)
	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x11}, 600)...)
	rec := postImage(t, userID, uuid.New(), "x.jpg", "image/jpeg", jpeg)

	require.Equal(t, http.StatusNotFound, rec.Code)
}
