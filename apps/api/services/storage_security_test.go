package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUploadExtension_ComesFromValidatedContentType(t *testing.T) {
	assert.Equal(t, ".png", uploadExtension("avatar.html", "image/png"))
	assert.Equal(t, ".pdf", uploadExtension("license.exe", "application/pdf"))
	assert.Equal(t, ".mp4", uploadExtension("tour.txt", "video/mp4"))
}
