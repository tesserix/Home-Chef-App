package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerAvatar_RejectsHTMLDisguisedAsImage(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	headers := make(textproto.MIMEHeader)
	headers.Set("Content-Disposition", `form-data; name="file"; filename="avatar.png"`)
	headers.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(headers)
	require.NoError(t, err)
	_, err = part.Write([]byte("<html><script>alert(1)</script></html>"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", uuid.New()); c.Next() })
	r.POST("/customer/avatar", (&CustomerHandler{}).UploadAvatar)
	req := httptest.NewRequest(http.MethodPost, "/customer/avatar", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "File contents don't match")
}

func TestAttachmentDisposition_RemovesHeaderInjection(t *testing.T) {
	value := attachmentDisposition("inline", "photo.png\r\nX-Attacker: injected")
	assert.False(t, strings.ContainsAny(value, "\r\n"))
	assert.Contains(t, value, "filename")
}
