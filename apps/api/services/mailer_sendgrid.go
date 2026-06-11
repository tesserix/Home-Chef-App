package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SendGridMailer sends emails via the SendGrid v3 HTTP API.
//
// We use a thin HTTP client instead of the official SendGrid SDK to avoid a
// large dependency for what is fundamentally a single POST request.
type SendGridMailer struct {
	apiKey   string
	from     string
	fromName string
	client   *http.Client
}

// NewSendGridMailer constructs a SendGridMailer. apiKey must be set; an
// empty key fails-fast on Send rather than silently dropping mail.
func NewSendGridMailer(apiKey, from, fromName string) *SendGridMailer {
	return &SendGridMailer{
		apiKey:   apiKey,
		from:     from,
		fromName: fromName,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

// sendgridPayload represents the SendGrid v3 mail/send request body
type sendgridPayload struct {
	Personalizations []sendgridPersonalization `json:"personalizations"`
	From             sendgridAddress           `json:"from"`
	Subject          string                    `json:"subject"`
	Content          []sendgridContent         `json:"content"`
	Attachments      []sendgridAttachment      `json:"attachments,omitempty"`
}

type sendgridAttachment struct {
	Content     string `json:"content"` // base64-encoded
	Type        string `json:"type"`
	Filename    string `json:"filename"`
	Disposition string `json:"disposition"`
}

type sendgridPersonalization struct {
	To []sendgridAddress `json:"to"`
}

type sendgridAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type sendgridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Send dispatches a single email via the SendGrid v3 REST API.
func (s *SendGridMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	return s.send(ctx, to, subject, htmlBody, nil)
}

// SendWithAttachment dispatches a single email with one attachment.
func (s *SendGridMailer) SendWithAttachment(ctx context.Context, to, subject, htmlBody string, att Attachment) error {
	return s.send(ctx, to, subject, htmlBody, []sendgridAttachment{{
		Content:     base64.StdEncoding.EncodeToString(att.Content),
		Type:        att.ContentType,
		Filename:    att.Filename,
		Disposition: "attachment",
	}})
}

func (s *SendGridMailer) send(ctx context.Context, to, subject, htmlBody string, attachments []sendgridAttachment) error {
	if s.apiKey == "" {
		return fmt.Errorf("email: SendGrid API key is not configured")
	}

	payload := sendgridPayload{
		Personalizations: []sendgridPersonalization{
			{To: []sendgridAddress{{Email: to}}},
		},
		From:    sendgridAddress{Email: s.from, Name: s.fromName},
		Subject: subject,
		Content: []sendgridContent{
			{Type: "text/html", Value: htmlBody},
		},
		Attachments: attachments,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("email: failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("email: failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("email: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("email: SendGrid returned status %d", resp.StatusCode)
	}

	return nil
}
