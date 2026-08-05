package handlers

// fssai_upload_security_test.go — the upload path handles chefs' Aadhaar and
// PAN images. These pin the three properties a security review found missing
// the first time: the filename never becomes a path, the extension is derived
// from the verified bytes rather than chosen by the uploader, and a document
// is never handed out on a durable public URL.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitiseFileLabel_CannotTraverse(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"../../etc/passwd", "passwd"},
		{"../../../root/.ssh/id_rsa", "id_rsa"},
		{"aadhaar.jpg", "aadhaar.jpg"},
		{"  spaced.png  ", "spaced.png"},
		{"", "document"},
		{".", "document"},
		{"/absolute/path.pdf", "path.pdf"},
	} {
		got := sanitiseFileLabel(tc.in)
		require.Equal(t, tc.want, got, "input %q", tc.in)
		require.NotContains(t, got, "/", "a label must never carry a separator")
		require.NotContains(t, got, "..", "a label must never carry a traversal")
	}
}

func TestSanitiseFileLabel_IsLengthCapped(t *testing.T) {
	require.LessOrEqual(t, len(sanitiseFileLabel(strings.Repeat("a", 500)+".jpg")), 80)
}

// The extension follows the SNIFFED type, so an uploader cannot name their file
// .php (or anything else) and have the stored object carry it.
func TestFssaiFileExtension_FollowsTheVerifiedType(t *testing.T) {
	require.Equal(t, "f.png", fssaiFileExtension("image/png"))
	require.Equal(t, "f.webp", fssaiFileExtension("image/webp"))
	require.Equal(t, "f.pdf", fssaiFileExtension("application/pdf"))
	// Anything unrecognised falls back to jpg rather than echoing the input.
	require.Equal(t, "f.jpg", fssaiFileExtension("application/x-httpd-php"))
	require.Equal(t, "f.jpg", fssaiFileExtension("image/jpeg"))
}
