package services

import (
	"context"
	"errors"
	"testing"
)

// recordingMailer captures every Send call and returns a scripted error.
type recordingMailer struct {
	calls           int
	attachmentCalls int
	lastTo          string
	err             error
}

func (r *recordingMailer) Send(_ context.Context, to, _, _ string) error {
	r.calls++
	r.lastTo = to
	return r.err
}

func (r *recordingMailer) SendWithAttachment(_ context.Context, to, _, _ string, _ Attachment) error {
	r.attachmentCalls++
	r.lastTo = to
	return r.err
}

func TestFallbackMailer_PrimarySucceeds_FallbackNotCalled(t *testing.T) {
	primary := &recordingMailer{}
	fallback := &recordingMailer{}
	m := NewFallbackMailer(primary, fallback)

	if err := m.Send(context.Background(), "user@example.com", "subject", "<p>hi</p>"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if primary.calls != 1 || fallback.calls != 0 {
		t.Fatalf("calls primary=%d fallback=%d, want 1/0 — fallback must not fire when primary delivers", primary.calls, fallback.calls)
	}
}

func TestFallbackMailer_PrimaryFails_FallbackDelivers(t *testing.T) {
	primary := &recordingMailer{err: errors.New("sendgrid 503")}
	fallback := &recordingMailer{}
	m := NewFallbackMailer(primary, fallback)

	if err := m.Send(context.Background(), "user@example.com", "subject", "<p>hi</p>"); err != nil {
		t.Fatalf("Send: %v — fallback delivery must count as success", err)
	}
	if fallback.calls != 1 || fallback.lastTo != "user@example.com" {
		t.Fatalf("fallback calls=%d to=%q, want the original message retried once", fallback.calls, fallback.lastTo)
	}
}

func TestFallbackMailer_BothFail_ErrorsJoined(t *testing.T) {
	primaryErr := errors.New("sendgrid 503")
	fallbackErr := errors.New("resend 500")
	m := NewFallbackMailer(&recordingMailer{err: primaryErr}, &recordingMailer{err: fallbackErr})

	err := m.Send(context.Background(), "user@example.com", "subject", "<p>hi</p>")
	if !errors.Is(err, primaryErr) || !errors.Is(err, fallbackErr) {
		t.Fatalf("error %v must wrap both provider errors", err)
	}
}

func TestFallbackMailer_AttachmentSend_FallbackFires(t *testing.T) {
	primary := &recordingMailer{err: errors.New("sendgrid 503")}
	fallback := &recordingMailer{}
	m := NewFallbackMailer(primary, fallback)

	att := Attachment{Filename: "invoice.pdf", ContentType: "application/pdf", Content: []byte("%PDF")}
	if err := m.SendWithAttachment(context.Background(), "user@example.com", "subject", "<p>hi</p>", att); err != nil {
		t.Fatalf("SendWithAttachment: %v — fallback delivery must count as success", err)
	}
	if fallback.attachmentCalls != 1 {
		t.Fatalf("fallback attachment calls=%d, want 1 — the attachment must ride along on the retry", fallback.attachmentCalls)
	}
}

func TestFallbackMailer_ContextCancelled_SkipsFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	primary := &recordingMailer{err: context.Canceled}
	fallback := &recordingMailer{}
	m := NewFallbackMailer(primary, fallback)

	if err := m.Send(ctx, "user@example.com", "subject", "<p>hi</p>"); err == nil {
		t.Fatal("Send: want error on cancelled context")
	}
	if fallback.calls != 0 {
		t.Fatalf("fallback calls=%d, want 0 — a dead context must not burn the fallback provider", fallback.calls)
	}
}
