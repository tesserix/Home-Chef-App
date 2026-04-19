package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/homechef/api/config"
)

// EmailService handles sending transactional emails via SendGrid
type EmailService struct {
	apiKey   string
	from     string
	fromName string
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

// InitEmailService initialises the email service from config
func InitEmailService() {
	svc := GetEmailService()
	svc.apiKey = config.AppConfig.SendGridAPIKey
	svc.from = config.AppConfig.FromEmail
	svc.fromName = config.AppConfig.FromName

	if svc.apiKey == "" {
		log.Println("Warning: SendGrid API key not configured — emails will be skipped")
	} else {
		log.Println("Email service initialised (SendGrid)")
	}
}

// sendgridPayload represents the SendGrid v3 mail/send request body
type sendgridPayload struct {
	Personalizations []sendgridPersonalization `json:"personalizations"`
	From             sendgridAddress           `json:"from"`
	Subject          string                    `json:"subject"`
	Content          []sendgridContent         `json:"content"`
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

// send dispatches an email via the SendGrid v3 REST API.
// Returns nil immediately if the API key is not configured.
func (s *EmailService) send(to, subject, htmlBody string) error {
	if s.apiKey == "" {
		log.Printf("Email skipped (no API key): to=%s subject=%s", to, subject)
		return nil
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
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("email: failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("email: failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("email: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("email: SendGrid returned status %d", resp.StatusCode)
	}

	log.Printf("Email sent: to=%s subject=%s", to, subject)
	return nil
}

// SendOrderConfirmation sends a branded order confirmation email to the customer
func (s *EmailService) SendOrderConfirmation(to, orderNumber string, items []OrderItemSummary, total float64) error {
	subject, html := OrderConfirmationHTML(orderNumber, items, total)
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
