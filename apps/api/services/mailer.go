package services

import "context"

// Mailer dispatches a single rendered email. Implementations:
//   - SendGridMailer — production primary (SendGrid v3 API)
//   - ResendMailer   — always-on fallback (Resend API)
//   - FallbackMailer — chains primary → fallback
//
// EmailService depends on this interface, not on a concrete provider, so
// InitEmailService can swap or chain providers without touching the
// template methods.
type Mailer interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
	// SendWithAttachment is Send plus a single file attachment (e.g. the
	// PDF invoice mailed on order.delivered). Kept as a separate method —
	// not a variadic on Send — so the common no-attachment path stays
	// trivially readable.
	SendWithAttachment(ctx context.Context, to, subject, htmlBody string, att Attachment) error
}

// Attachment is one file riding along on an outgoing email. Content is the
// raw bytes; each provider adapter base64-encodes on the wire.
type Attachment struct {
	Filename    string
	ContentType string // e.g. "application/pdf"
	Content     []byte
}
