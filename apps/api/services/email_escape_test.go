package services

// email_escape_test.go — security audit #7: transactional emails are assembled
// with fmt.Sprintf, so user-controlled free text (a chef's business name/menu
// item, a customer's name) must be HTML-escaped or it injects markup into an
// email from the trusted Fe3dr brand. These prove the escaping holds and that
// system URLs are left intact.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWelcomeEmail_EscapesName(t *testing.T) {
	_, html := WelcomeEmailHTML(`<script>alert(1)</script>`)
	require.False(t, strings.Contains(html, "<script>"), "raw script must not appear")
	require.Contains(t, html, "&lt;script&gt;", "name is HTML-escaped")
}

func TestStaffInvite_EscapesNameButNotURL(t *testing.T) {
	_, html := StaffInvitationHTML(`<a href="https://evil.tld">Pay now</a>`, "admin", "https://fe3dr.com/accept?t=abc")
	require.False(t, strings.Contains(html, `<a href="https://evil.tld"`), "injected anchor must be escaped")
	require.Contains(t, html, "&lt;a href")
	// The legitimate accept URL is a system href and must remain usable.
	require.Contains(t, html, "https://fe3dr.com/accept?t=abc")
}
