package services

import (
	"context"
	"errors"
	"log"
)

// FallbackMailer chains two Mailers: every email goes to primary first, and
// only when primary errors does it retry on fallback. Production wiring is
// SendGrid (primary) → Resend (fallback) so a SendGrid outage, rate limit
// or account suspension degrades to a provider switch instead of dropped
// transactional mail.
//
// The failover is per-message and stateless — no circuit breaker. SendGrid
// is retried first on every send, so recovery is automatic and there is no
// shared state to reset.
type FallbackMailer struct {
	primary  Mailer
	fallback Mailer
}

// NewFallbackMailer constructs a FallbackMailer.
func NewFallbackMailer(primary, fallback Mailer) *FallbackMailer {
	return &FallbackMailer{primary: primary, fallback: fallback}
}

// Send tries primary, then fallback.
func (f *FallbackMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	return f.deliver(ctx, to, subject, func(m Mailer) error {
		return m.Send(ctx, to, subject, htmlBody)
	})
}

// SendWithAttachment tries primary, then fallback, with the attachment
// riding along on both attempts.
func (f *FallbackMailer) SendWithAttachment(ctx context.Context, to, subject, htmlBody string, att Attachment) error {
	return f.deliver(ctx, to, subject, func(m Mailer) error {
		return m.SendWithAttachment(ctx, to, subject, htmlBody, att)
	})
}

func (f *FallbackMailer) deliver(ctx context.Context, to, subject string, send func(Mailer) error) error {
	primaryErr := send(f.primary)
	if primaryErr == nil {
		return nil
	}
	// A cancelled context will fail on the fallback too — bail early.
	if ctx.Err() != nil {
		return primaryErr
	}

	log.Printf("Warning: primary email provider failed, retrying on fallback: to=%s subject=%s error=%v", to, subject, primaryErr)

	fallbackErr := send(f.fallback)
	if fallbackErr == nil {
		log.Printf("Email delivered via fallback provider: to=%s subject=%s", to, subject)
		return nil
	}
	return errors.Join(primaryErr, fallbackErr)
}
