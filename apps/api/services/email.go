package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/homechef/api/config"
)

// maskEmail redacts a recipient address for logs, keeping the first character
// and the domain so entries stay traceable without leaking the full PII
// address. An empty string stays empty; a value with no usable "@" is reduced
// to its first character plus "***".
func maskEmail(e string) string {
	if e == "" {
		return ""
	}
	at := strings.LastIndex(e, "@")
	if at < 1 {
		return e[:1] + "***"
	}
	return e[:1] + "***@" + e[at+1:]
}

// EmailService handles sending transactional emails. The provider chain is
// built in InitEmailService: SendGrid primary, Resend fallback.
type EmailService struct {
	mailer Mailer
}

// OrderItemSummary represents a single item in an order email
type OrderItemSummary struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

var (
	emailService *EmailService
	emailOnce    sync.Once
)

// GetEmailService returns the singleton email service
func GetEmailService() *EmailService {
	emailOnce.Do(func() {
		emailService = &EmailService{}
	})
	return emailService
}

// InitEmailService initialises the email service from config.
//
// SendGrid is the primary provider; Resend is ALWAYS the fallback when both
// keys are set, so a SendGrid outage degrades to a provider switch instead
// of dropped mail. With only one key, that provider runs alone. With no
// keys, emails are skipped (and logged) — same behaviour as before the
// fallback existed.
func InitEmailService() {
	svc := GetEmailService()
	cfg := config.AppConfig

	switch {
	case cfg.SendGridAPIKey != "" && cfg.ResendAPIKey != "":
		svc.mailer = NewFallbackMailer(
			NewSendGridMailer(cfg.SendGridAPIKey, cfg.FromEmail, cfg.FromName),
			NewResendMailer(cfg.ResendAPIKey, cfg.FromEmail, cfg.FromName),
		)
		log.Println("Email service initialised (SendGrid primary, Resend fallback)")
	case cfg.SendGridAPIKey != "":
		log.Println("Warning: RESEND_API_KEY not configured — SendGrid only, no fallback provider")
		svc.mailer = NewSendGridMailer(cfg.SendGridAPIKey, cfg.FromEmail, cfg.FromName)
		log.Println("Email service initialised (SendGrid)")
	case cfg.ResendAPIKey != "":
		log.Println("Warning: SENDGRID_API_KEY not configured — using Resend as the only provider")
		svc.mailer = NewResendMailer(cfg.ResendAPIKey, cfg.FromEmail, cfg.FromName)
		log.Println("Email service initialised (Resend)")
	default:
		log.Println("Warning: SendGrid API key not configured — emails will be skipped")
	}
}

// send dispatches an email via the configured provider chain.
// Returns nil immediately if no API key is configured.
// Send dispatches a raw HTML email. Exported wrapper around send for generic
// senders such as the Temporal notification activity (see services/temporal_notify.go).
func (s *EmailService) Send(to, subject, htmlBody string) error {
	return s.send(to, subject, htmlBody)
}

func (s *EmailService) send(to, subject, htmlBody string) error {
	if s.mailer == nil {
		log.Printf("Email skipped (no API key): to=%s subject=%s", maskEmail(to), subject)
		return nil
	}

	if err := s.mailer.Send(context.Background(), to, subject, htmlBody); err != nil {
		return err
	}

	log.Printf("Email sent: to=%s subject=%s", maskEmail(to), subject)
	return nil
}

// SendOrderConfirmation sends a branded order confirmation email to the customer.
// details is optional — when populated, the email renders a CGST §31-compliant
// invoice (GST breakup, HSN/SAC, supplier particulars, chef + ETA + address);
// when nil, the email falls back to the legacy minimal layout so existing
// callers keep working.
//
// TODO(CW-01e-backend): wire the order pipeline to pass invoice details so
// the confirmation email is invoice-shaped end-to-end.
func (s *EmailService) SendOrderConfirmation(to, orderNumber string, items []OrderItemSummary, total float64, details *OrderInvoiceDetails) error {
	subject, html := OrderConfirmationHTML(orderNumber, items, total, details)
	return s.send(to, subject, html)
}

// SendOrderStatusUpdate notifies a customer about an order status change
func (s *EmailService) SendOrderStatusUpdate(to, orderNumber, status string) error {
	subject, html := OrderStatusUpdateHTML(orderNumber, status)
	return s.send(to, subject, html)
}

// SendPasswordResetEmail sends a branded password reset link
func (s *EmailService) SendPasswordResetEmail(to, resetToken string) error {
	resetLink := fmt.Sprintf("https://fe3dr.com/reset-password?token=%s", resetToken)
	subject, html := PasswordResetHTML(resetLink)
	return s.send(to, subject, html)
}

// SendChefVerificationApproved notifies a chef that their profile was approved
func (s *EmailService) SendChefVerificationApproved(to, chefName string) error {
	subject, html := ChefVerificationApprovedHTML(chefName)
	return s.send(to, subject, html)
}

// SendOrderInvoice emails the customer a copy of their GST tax
// invoice as a PDF attachment after delivery. Goes through the same
// provider chain as every other send, so the attachment fails over
// from SendGrid to Resend too. A missing API key no-ops cleanly like
// every other send helper.
func (s *EmailService) SendOrderInvoice(to, firstName, orderNumber string, pdfBytes []byte, filename string) error {
	if s.mailer == nil {
		log.Printf("Email skipped (no API key): to=%s subject=invoice %s", maskEmail(to), orderNumber)
		return nil
	}
	name := firstName
	if name == "" {
		name = "there"
	}
	subject := fmt.Sprintf("Your Home Chef invoice (%s)", orderNumber)
	body := fmt.Sprintf(`
		<p>Hi %s,</p>
		<p>Thanks for ordering with Home Chef. Your tax invoice for order
		<strong>%s</strong> is attached as a PDF.</p>
		<p>Save it for your records — if you need to claim GST input credit, your
		accountant will want this.</p>
		<p>— Home Chef</p>
	`, name, orderNumber)

	att := Attachment{
		Filename:    filename,
		ContentType: "application/pdf",
		Content:     pdfBytes,
	}
	if err := s.mailer.SendWithAttachment(context.Background(), to, subject, body, att); err != nil {
		return fmt.Errorf("email: invoice send: %w", err)
	}
	log.Printf("Invoice email sent: to=%s order=%s", maskEmail(to), orderNumber)
	return nil
}

// SendApprovalInfoRequested notifies a chef that an admin has asked for
// more information on an outstanding approval request. Mirrors the
// in-app "ACTION REQUIRED" card so a chef who only checks email also
// learns there's something blocking their onboarding.
func (s *EmailService) SendApprovalInfoRequested(to, chefName, requestTitle, adminNotes string) error {
	subject := "Action needed: " + requestTitle
	html := fmt.Sprintf(`
		<p>Hi %s,</p>
		<p>Our team has asked for more information on your application item:
		<strong>%s</strong>.</p>
		<p style="white-space: pre-wrap; padding: 12px; background: #F4F2EC; border-left: 3px solid #C2410C;">%s</p>
		<p>Please open the HomeChef Vendor app to respond — the request is
		waiting under <strong>Action Required</strong> on your dashboard.</p>
		<p>— Home Chef</p>
	`, chefName, requestTitle, adminNotes)
	return s.send(to, subject, html)
}

// SendChefNewOrder notifies a chef about a new incoming order
func (s *EmailService) SendChefNewOrder(to, orderNumber string, total float64) error {
	subject, html := ChefNewOrderHTML(orderNumber, total)
	return s.send(to, subject, html)
}

// SendDeliveryAssigned notifies a customer that a delivery partner was assigned
func (s *EmailService) SendDeliveryAssigned(to, orderNumber, pickupAddress string) error {
	subject, html := DeliveryAssignedHTML(orderNumber, pickupAddress)
	return s.send(to, subject, html)
}

// SendSupportTicketCreated confirms that a support ticket was created
func (s *EmailService) SendSupportTicketCreated(to, ticketNumber, subject string) error {
	html := fmt.Sprintf(`
<h2>Support Ticket Created</h2>
<p>Your support ticket <strong>#%s</strong> has been created.</p>
<p><strong>Subject:</strong> %s</p>
<p>Our team will get back to you shortly.</p>
`, ticketNumber, subject)

	return s.send(to, fmt.Sprintf("Support Ticket #%s — %s", ticketNumber, subject), html)
}

// SendSupportTicketUpdate notifies about a support ticket status change
func (s *EmailService) SendSupportTicketUpdate(to, ticketNumber, status string) error {
	html := fmt.Sprintf(`
<h2>Support Ticket Update</h2>
<p>Your support ticket <strong>#%s</strong> has been updated to: <strong>%s</strong></p>
`, ticketNumber, status)

	return s.send(to, fmt.Sprintf("Ticket Update — #%s", ticketNumber), html)
}

// SendWelcomeEmail sends a branded welcome email to newly registered users
func (s *EmailService) SendWelcomeEmail(to, firstName string) error {
	subject, html := WelcomeEmailHTML(firstName)
	return s.send(to, subject, html)
}

// SendEmailVerification sends an email verification link
func (s *EmailService) SendEmailVerification(to, firstName, verifyURL string) error {
	subject, html := EmailVerificationHTML(firstName, verifyURL)
	return s.send(to, subject, html)
}

// SendEmailOTP sends a 6-digit email verification code.
func (s *EmailService) SendEmailOTP(to, firstName, code string) error {
	subject, html := EmailOTPHTML(firstName, code)
	return s.send(to, subject, html)
}

// SendAccountReminderEmail tells an existing user that someone (possibly
// them) tried to register with their email. The message is intentionally
// neutral so the recipient learns about it without confirming to a
// third party that the account exists.
func (s *EmailService) SendAccountReminderEmail(to, firstName string) error {
	loginURL := "https://fe3dr.com/login"
	resetURL := "https://fe3dr.com/forgot-password"
	subject := "You already have a Fe3dr HomeChef account"
	html := fmt.Sprintf(`<p>Hi %s,</p>
<p>Someone (possibly you) just tried to create a Fe3dr HomeChef account with this email,
but you already have one. You can <a href="%s">sign in here</a>.</p>
<p>If you've forgotten your password, <a href="%s">reset it here</a>.</p>
<p>If this wasn't you, you can safely ignore this email — no changes were made.</p>
<p>— Fe3dr HomeChef</p>`, firstName, loginURL, resetURL)
	return s.send(to, subject, html)
}

// SendStaffInvitation sends a branded staff invitation email
func (s *EmailService) SendStaffInvitation(to, inviterName, role, acceptURL string) error {
	subject, html := StaffInvitationHTML(inviterName, role, acceptURL)
	return s.send(to, subject, html)
}
