package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ResendMailer sends emails via the Resend HTTP API (https://resend.com).
//
// Same thin-HTTP-client philosophy as SendGridMailer: the whole integration
// is one POST request, so the official SDK is not worth the dependency.
// Resend is wired as the FALLBACK provider behind SendGrid — see
// FallbackMailer — so transactional mail keeps flowing when SendGrid is
// down, rate-limiting, or rejecting the account.
type ResendMailer struct {
	apiKey   string
	from     string
	fromName string
	client   *http.Client
}

// NewResendMailer constructs a ResendMailer. apiKey must be set; an empty
// key fails-fast on Send rather than silently dropping mail.
func NewResendMailer(apiKey, from, fromName string) *ResendMailer {
	return &ResendMailer{
		apiKey:   apiKey,
		from:     from,
		fromName: fromName,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

// Send dispatches a single email via Resend.
func (s *ResendMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	return s.send(ctx, to, subject, htmlBody, nil)
}

// SendWithAttachment dispatches a single email with one attachment.
func (s *ResendMailer) SendWithAttachment(ctx context.Context, to, subject, htmlBody string, att Attachment) error {
	return s.send(ctx, to, subject, htmlBody, []resendAttachment{{
		Content:     base64.StdEncoding.EncodeToString(att.Content),
		Filename:    att.Filename,
		ContentType: att.ContentType,
	}})
}

func (s *ResendMailer) send(ctx context.Context, to, subject, htmlBody string, attachments []resendAttachment) error {
	if s.apiKey == "" {
		return fmt.Errorf("email: Resend API key is not configured")
	}

	// Resend takes the display name inline in the from field
	// ("Name <email>") instead of a separate field like SendGrid.
	from := s.from
	if s.fromName != "" {
		from = fmt.Sprintf("%s <%s>", s.fromName, s.from)
	}

	body := resendRequest{
		From:        from,
		To:          []string{to},
		Subject:     subject,
		HTML:        htmlBody,
		Attachments: attachments,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("email: failed to marshal resend request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.resend.com/emails", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("email: failed to create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("email: resend request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("email: Resend returned status %d: %s", resp.StatusCode, string(respBody))
}

// Resend API request shape — minimum viable subset. Text is part of the
// shape for parity but stays empty: HomeChef templates are HTML-only.
type resendRequest struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	Subject     string             `json:"subject"`
	HTML        string             `json:"html,omitempty"`
	Text        string             `json:"text,omitempty"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
}

type resendAttachment struct {
	Content     string `json:"content"` // base64-encoded
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
}
