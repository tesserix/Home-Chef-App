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

// SendOrderConfirmation sends an order confirmation email to the customer
func (s *EmailService) SendOrderConfirmation(to, orderNumber string, items []OrderItemSummary, total float64) error {
	var itemsHTML string
	for _, item := range items {
		itemsHTML += fmt.Sprintf("<tr><td>%s</td><td>%d</td><td>$%.2f</td></tr>", item.Name, item.Quantity, item.Price)
	}

	html := fmt.Sprintf(`
<h2>Order Confirmed!</h2>
<p>Thank you for your order <strong>#%s</strong>.</p>
<table border="1" cellpadding="8" cellspacing="0">
<tr><th>Item</th><th>Qty</th><th>Price</th></tr>
%s
</table>
<p><strong>Total: $%.2f</strong></p>
<p>We'll notify you when your order is being prepared.</p>
`, orderNumber, itemsHTML, total)

	return s.send(to, fmt.Sprintf("Order Confirmed — #%s", orderNumber), html)
}

// SendOrderStatusUpdate notifies a customer about an order status change
func (s *EmailService) SendOrderStatusUpdate(to, orderNumber, status string) error {
	html := fmt.Sprintf(`
<h2>Order Status Update</h2>
<p>Your order <strong>#%s</strong> status has been updated to: <strong>%s</strong></p>
<p>%s</p>
`, orderNumber, status, getOrderStatusMessage(status))

	return s.send(to, fmt.Sprintf("Order Update — #%s", orderNumber), html)
}

// SendPasswordResetEmail sends a password reset link
func (s *EmailService) SendPasswordResetEmail(to, resetToken string) error {
	// In production the link would point to the storefront reset page
	resetLink := fmt.Sprintf("https://fe3dr.com/reset-password?token=%s", resetToken)

	html := fmt.Sprintf(`
<h2>Password Reset</h2>
<p>You requested a password reset. Click the link below to set a new password:</p>
<p><a href="%s">Reset Password</a></p>
<p>This link expires in 1 hour. If you did not request this, please ignore this email.</p>
`, resetLink)

	return s.send(to, "Reset Your Password", html)
}

// SendChefVerificationApproved notifies a chef that their profile was approved
func (s *EmailService) SendChefVerificationApproved(to, chefName string) error {
	html := fmt.Sprintf(`
<h2>Congratulations, %s!</h2>
<p>Your chef profile has been verified and approved. You can now start listing dishes and accepting orders on HomeChef.</p>
<p><a href="https://vendors.fe3dr.com/dashboard">Go to Dashboard</a></p>
`, chefName)

	return s.send(to, "Your Chef Profile Is Approved!", html)
}

// SendChefNewOrder notifies a chef about a new incoming order
func (s *EmailService) SendChefNewOrder(to, orderNumber string, total float64) error {
	html := fmt.Sprintf(`
<h2>New Order Received!</h2>
<p>You have a new order <strong>#%s</strong> worth <strong>$%.2f</strong>.</p>
<p>Please log in to your dashboard to review and confirm.</p>
<p><a href="https://vendors.fe3dr.com/orders">View Orders</a></p>
`, orderNumber, total)

	return s.send(to, fmt.Sprintf("New Order — #%s", orderNumber), html)
}

// SendDeliveryAssigned notifies a customer that a delivery partner was assigned
func (s *EmailService) SendDeliveryAssigned(to, orderNumber, pickupAddress string) error {
	html := fmt.Sprintf(`
<h2>Delivery Partner Assigned</h2>
<p>A delivery partner has been assigned to your order <strong>#%s</strong>.</p>
<p>Pickup from: %s</p>
<p>You will be notified when your order is on its way!</p>
`, orderNumber, pickupAddress)

	return s.send(to, fmt.Sprintf("Delivery Assigned — #%s", orderNumber), html)
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

// SendStaffInvitation sends a branded staff invitation email
func (s *EmailService) SendStaffInvitation(to, inviterName, role, acceptURL string) error {
	subject, html := StaffInvitationHTML(inviterName, role, acceptURL)
	return s.send(to, subject, html)
}
