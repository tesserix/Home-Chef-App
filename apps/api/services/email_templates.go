package services

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/homechef/api/models"
)

// esc HTML-escapes user-controlled free text before it is interpolated into an
// email template (audit #7). These templates are assembled with fmt.Sprintf (not
// html/template), so without this a chef-controlled value (business name, menu
// item name) or a customer address could inject markup — a phishing link or
// image — into an email sent from the trusted Fe3dr brand. System-generated
// values (URLs, numbers, fixed labels) don't need it; free-text names/addresses/
// tax fields do.
func esc(s string) string { return html.EscapeString(s) }

// Enterprise HTML email templates for Fe3dr / HomeChef
// Responsive, branded layout matching the Fe3dr design system.
// Base colors: neutral ink (#111827), gray-700 (#374151), gray-900 (#111827) — deliberately brandless/monochrome

// emailBase wraps content in the branded Fe3dr email layout
func emailBase(title, preheader, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" xmlns="http://www.w3.org/1999/xhtml">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="x-apple-disable-message-reformatting">
  <title>%s</title>
  <style>
    body { margin:0; padding:0; background-color:#f9fafb; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif; }
    .container { max-width:600px; margin:0 auto; }
    .header { background:linear-gradient(135deg,#111827,#374151); padding:32px 40px; text-align:center; border-radius:12px 12px 0 0; }
    .header h1 { margin:0; color:#ffffff; font-size:24px; font-weight:700; letter-spacing:0.5px; }
    .content { background:#ffffff; padding:40px; border-left:1px solid #e5e7eb; border-right:1px solid #e5e7eb; }
    .content h2 { color:#111827; font-size:22px; margin:0 0 16px 0; font-weight:600; }
    .content p { color:#4b5563; font-size:15px; line-height:1.7; margin:0 0 16px 0; }
    .btn { display:inline-block; background:#111827; color:#ffffff !important; text-decoration:none; padding:14px 32px; border-radius:8px; font-weight:600; font-size:15px; margin:8px 0 24px 0; }
    .btn:hover { background:#374151; }
    .info-box { background:#F3F4F6; border:1px solid #E5E7EB; border-radius:8px; padding:16px 20px; margin:16px 0; }
    .info-box p { color:#374151; margin:0; font-size:14px; }
    .divider { border:none; border-top:1px solid #e5e7eb; margin:24px 0; }
    .footer { background:#f9fafb; padding:24px 40px; text-align:center; border:1px solid #e5e7eb; border-top:none; border-radius:0 0 12px 12px; }
    .footer p { color:#9ca3af; font-size:12px; line-height:1.6; margin:0; }
    .footer a { color:#111827; text-decoration:none; }
    .muted { color:#9ca3af; font-size:13px; }
    @media only screen and (max-width:620px) {
      .container { width:100%% !important; }
      .content, .header, .footer { padding:24px 20px !important; }
    }
  </style>
</head>
<body>
  <!-- Preheader -->
  <div style="display:none;max-height:0;overflow:hidden;mso-hide:all">%s</div>
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f9fafb;padding:40px 16px;">
    <tr><td align="center">
      <div class="container">
        <div class="header">
          <h1>Fe3dr</h1>
        </div>
        <div class="content">
          %s
        </div>
        <div class="footer">
          <p>Fe3dr by HomeChef &middot; Authentic homemade food, delivered</p>
          <p style="margin-top:8px;"><a href="https://fe3dr.com">fe3dr.com</a> &middot; <a href="https://fe3dr.com/privacy">Privacy</a> &middot; <a href="https://fe3dr.com/terms">Terms</a></p>
        </div>
      </div>
    </td></tr>
  </table>
</body>
</html>`, title, preheader, body)
}

// =========================================================================
// Enterprise email templates
// =========================================================================

// WelcomeEmailHTML returns the welcome email for newly registered users
func WelcomeEmailHTML(firstName string) (subject, html string) {
	subject = "Welcome to Fe3dr!"
	body := fmt.Sprintf(`
          <h2>Welcome to Fe3dr, %s!</h2>
          <p>We're thrilled to have you. Discover amazing home-cooked meals from talented local chefs right in your neighbourhood.</p>
          <p>Here's what you can do:</p>
          <ul style="color:#4b5563;font-size:15px;line-height:2;">
            <li>Browse hundreds of home chefs near you</li>
            <li>Order authentic, homemade food</li>
            <li>Track your delivery in real time</li>
            <li>Rate and review your experience</li>
          </ul>
          <a href="https://fe3dr.com" class="btn">Start Exploring</a>
          <p class="muted">If you didn't create this account, please ignore this email.</p>
`, esc(firstName))
	html = emailBase(subject, "Welcome to Fe3dr — discover home-cooked meals near you", body)
	return
}

// EmailOTPHTML returns the 6-digit email verification code email.
func EmailOTPHTML(firstName, code string) (subject, html string) {
	subject = "Your Fe3dr verification code"
	greeting := "there"
	if strings.TrimSpace(firstName) != "" {
		greeting = esc(firstName)
	}
	body := fmt.Sprintf(`
          <h2>Verify your email, %s</h2>
          <p>Use this 6-digit code to verify your email and continue setting up your account.</p>
          <div style="text-align:center;margin:8px 0 24px 0;">
            <span style="display:inline-block;font-size:34px;font-weight:700;letter-spacing:10px;color:#111827;background:#F3F4F6;border:1px solid #E5E7EB;border-radius:8px;padding:16px 28px;">%s</span>
          </div>
          <div class="info-box">
            <p>This code expires in 10 minutes. If you didn't request it, you can safely ignore this email.</p>
          </div>
`, greeting, esc(code))
	html = emailBase(subject, "Your Fe3dr email verification code", body)
	return
}

// EmailVerificationHTML returns the email verification email
func EmailVerificationHTML(firstName, verifyURL string) (subject, html string) {
	subject = "Verify your email — Fe3dr"
	body := fmt.Sprintf(`
          <h2>Verify your email, %s</h2>
          <p>Thanks for signing up! Please verify your email address to activate your account and start ordering.</p>
          <a href="%s" class="btn">Verify Email Address</a>
          <div class="info-box">
            <p>This link expires in 24 hours. If you didn't create a Fe3dr account, you can safely ignore this email.</p>
          </div>
          <hr class="divider">
          <p class="muted">Button not working? Copy and paste this URL into your browser:<br>
          <span style="word-break:break-all;color:#111827;">%s</span></p>
`, esc(firstName), verifyURL, verifyURL)
	html = emailBase(subject, "Verify your email to start using Fe3dr", body)
	return
}

// StaffInvitationHTML returns the staff invitation email
func StaffInvitationHTML(inviterName, role, acceptURL string) (subject, html string) {
	subject = "You're invited to join Fe3dr"
	body := fmt.Sprintf(`
          <h2>You've been invited!</h2>
          <p><strong>%s</strong> has invited you to join the Fe3dr team as <strong>%s</strong>.</p>
          <p>As a team member, you'll help manage deliveries and fleet operations on the Fe3dr platform.</p>
          <a href="%s" class="btn">Accept Invitation</a>
          <div class="info-box">
            <p>This invitation expires in 7 days. After accepting, you'll be guided through a quick onboarding process.</p>
          </div>
          <hr class="divider">
          <p class="muted">If you weren't expecting this invitation, you can safely ignore it.</p>
`, esc(inviterName), esc(role), acceptURL)
	html = emailBase(subject, fmt.Sprintf("%s invited you to join Fe3dr", esc(inviterName)), body)
	return
}

// PasswordResetHTML renders the branded password-reset email.
//
// Sent by US, not by Firebase. Firebase's own message comes from
// noreply@<project>.firebaseapp.com (which Gmail treats as spam), is branded
// with the GCP project name rather than the product, and pastes a raw
// firebaseapp.com URL into the body — three things that together make a
// legitimate security email look exactly like a phishing attempt.
//
// ttl is passed in rather than hardcoded so the stated expiry can never drift
// away from the one actually enforced in services/password_reset.go.
func PasswordResetHTML(resetURL string, ttl time.Duration) (subject, html string) {
	subject = "Reset your Fe3dr password"
	mins := int(ttl.Minutes())
	body := fmt.Sprintf(`
          <h2>Reset your password</h2>
          <p>We received a request to reset the password for your Fe3dr account. Choose a new one using the button below.</p>
          <a href="%s" class="btn">Choose a new password</a>
          <div class="info-box">
            <p><strong>This link expires in %d minutes</strong> and can only be used once.</p>
            <p>If you didn't ask for this, you can safely ignore this email — your password stays as it is, and nobody can change it without this link.</p>
          </div>
          <hr class="divider">
          <p class="muted">We will never ask you for your password, an OTP, or your payment details by email or phone.</p>
          <p class="muted">Button not working? Copy and paste this address into your browser:<br>
          <span style="word-break:break-all;color:#111827;">%s</span></p>
`, esc(resetURL), mins, esc(resetURL))
	html = emailBase(subject, "Reset your Fe3dr password", body)
	return
}

// OrderInvoiceDetails carries optional invoice + fulfilment metadata for the
// order-confirmation email so the message can satisfy CGST Act 2017 §31 +
// Rule 46 particulars (HSN/SAC, GST breakup, supplier, place of supply) for
// B2C food orders. All fields are optional; missing data is rendered as a
// graceful fallback so existing callers that pass nil keep working.
//
// TODO(CW-01e-backend): wire the order pipeline to populate these fields when
// dispatching the confirmation notification (currently the caller in
// services/notifications.go passes nil items + only the total).
type OrderInvoiceDetails struct {
	// Tax breakdown
	Subtotal    float64
	DeliveryFee float64
	PlatformFee float64
	Discount    float64
	Tip         float64
	// TaxLines and Rounding come straight off OrderResponse (models/pricing.go) —
	// the same rows the app, the web page and the PDF show. Building the split here
	// instead is how five surfaces came to disagree about one order's tax.
	TaxLines   []models.TaxLine
	Rounding   float64
	GSTRatePct float64 // e.g. 5 for prepared food via e-commerce
	HSNCode    string  // SAC for service; defaults to "996331" (food via e-commerce)

	// Fulfilment context
	ChefName        string
	ChefFSSAI       string
	DeliveryAddress string
	ETA             string // human-readable window e.g. "Today, 7:30–8:00 PM"

	// Marketplace operator (Tesserix / Fe3dr Pty Ltd) — used as supplier of
	// record for B2C invoices under §9(5).
	SupplierName    string
	SupplierAddress string
	SupplierGSTIN   string
	PlaceOfSupply   string // state name of the recipient
	InvoiceNumber   string // consecutive invoice series; empty if not yet allocated
}

// OrderConfirmationHTML returns the order confirmation email. When details is
// non-nil and populated, the email renders a full CGST §31-compliant invoice
// (GST breakup, HSN, supplier particulars, chef, delivery address, ETA,
// allergen reminder, refund policy link). When details is nil, the email
// renders the legacy minimal layout so existing callers keep working.
func OrderConfirmationHTML(orderNumber string, items []OrderItemSummary, total float64, details *OrderInvoiceDetails) (subject, html string) {
	subject = fmt.Sprintf("Order #%s confirmed — Fe3dr", orderNumber)

	rows := ""
	for _, item := range items {
		rows += fmt.Sprintf(`<tr>
            <td style="padding:8px 0;border-bottom:1px solid #f3f4f6;color:#374151;">%s</td>
            <td style="padding:8px 0;border-bottom:1px solid #f3f4f6;color:#374151;text-align:center;">%d</td>
            <td style="padding:8px 0;border-bottom:1px solid #f3f4f6;color:#374151;text-align:right;">₹%.2f</td>
          </tr>`, esc(item.Name), item.Quantity, item.Price)
	}

	// Resolve invoice details with safe defaults so the template stays
	// readable even when individual fields are missing.
	d := orderInvoiceDetailsOrDefault(details)

	body := fmt.Sprintf(`
          <h2>Order Confirmed!</h2>
          <p><strong>Order #%s</strong> — your home chef is preparing your meal.</p>
          %s
          %s
          <table width="100%%" cellpadding="0" cellspacing="0" style="margin:16px 0;">
            <tr style="background:#f9fafb;">
              <th style="padding:10px 0;text-align:left;color:#6b7280;font-size:13px;font-weight:600;">Item</th>
              <th style="padding:10px 0;text-align:center;color:#6b7280;font-size:13px;font-weight:600;">Qty</th>
              <th style="padding:10px 0;text-align:right;color:#6b7280;font-size:13px;font-weight:600;">Price</th>
            </tr>
            %s
          </table>
          %s
          <table width="100%%" cellpadding="0" cellspacing="0" style="margin:8px 0 24px 0;">
            <tr>
              <td colspan="2" style="padding:12px 0;border-top:2px solid #111827;font-weight:700;color:#111827;font-size:16px;">Grand total</td>
              <td style="padding:12px 0;border-top:2px solid #111827;text-align:right;font-weight:700;color:#111827;font-size:18px;">₹%.2f</td>
            </tr>
          </table>
          <a href="https://fe3dr.com/orders" class="btn">Track Your Order</a>
          <div class="info-box" style="background:#FEF3C7;border-color:#F59E0B;">
            <p style="color:#92400E;"><strong>Allergy notice:</strong> If you have any allergies or dietary restrictions, please contact your chef immediately via the order chat. Home kitchens may handle nuts, dairy, gluten, eggs and other allergens.</p>
          </div>
          %s
          %s
          <hr class="divider">
          <p class="muted">Need help? Visit <a href="https://fe3dr.com/refund" style="color:#111827;">our refund policy</a> or reply to this email. This is a computer-generated invoice — no signature is required.</p>
`,
		orderNumber,
		renderInvoiceHeader(d),
		renderFulfilmentBlock(d),
		rows,
		renderInvoiceSummary(d),
		total,
		renderSupplierFooter(d),
		renderRefundBlurb(),
	)

	html = emailBase(subject, fmt.Sprintf("Order #%s confirmed — your meal is being prepared", orderNumber), body)
	return
}

// orderInvoiceDetailsOrDefault returns a non-nil details struct with safe
// fallbacks so the email template never panics on missing data.
func orderInvoiceDetailsOrDefault(details *OrderInvoiceDetails) *OrderInvoiceDetails {
	if details == nil {
		return &OrderInvoiceDetails{
			HSNCode:         "996331",
			SupplierName:    "Fe3dr by HomeChef",
			SupplierAddress: "Mumbai, India",
			GSTRatePct:      5,
		}
	}
	// Return a copy so callers cannot observe template-side mutation.
	c := *details
	if c.HSNCode == "" {
		c.HSNCode = "996331" // SAC for prepared food supplied via e-commerce
	}
	if c.SupplierName == "" {
		c.SupplierName = "Fe3dr by HomeChef"
	}
	if c.SupplierAddress == "" {
		c.SupplierAddress = "Mumbai, India"
	}
	return &c
}

// renderInvoiceHeader returns the invoice number + HSN block. Empty when
// neither is set so the legacy minimal email stays unchanged.
func renderInvoiceHeader(d *OrderInvoiceDetails) string {
	if d.InvoiceNumber == "" && d.SupplierGSTIN == "" {
		return ""
	}
	parts := ""
	if d.InvoiceNumber != "" {
		parts += fmt.Sprintf(`<p style="margin:4px 0;color:#4b5563;font-size:14px;"><strong>Invoice no.</strong> %s</p>`, esc(d.InvoiceNumber))
	}
	if d.SupplierGSTIN != "" {
		parts += fmt.Sprintf(`<p style="margin:4px 0;color:#4b5563;font-size:14px;"><strong>GSTIN:</strong> %s</p>`, esc(d.SupplierGSTIN))
	}
	if d.HSNCode != "" {
		parts += fmt.Sprintf(`<p style="margin:4px 0;color:#4b5563;font-size:14px;"><strong>HSN/SAC:</strong> %s (prepared food via e-commerce)</p>`, esc(d.HSNCode))
	}
	if d.PlaceOfSupply != "" {
		parts += fmt.Sprintf(`<p style="margin:4px 0;color:#4b5563;font-size:14px;"><strong>Place of supply:</strong> %s</p>`, esc(d.PlaceOfSupply))
	}
	return fmt.Sprintf(`<div style="background:#f9fafb;border-radius:8px;padding:12px 16px;margin:16px 0;">%s</div>`, parts)
}

// renderFulfilmentBlock returns chef + address + ETA. Empty when nothing set.
func renderFulfilmentBlock(d *OrderInvoiceDetails) string {
	if d.ChefName == "" && d.DeliveryAddress == "" && d.ETA == "" {
		return ""
	}
	parts := ""
	if d.ChefName != "" {
		fssai := ""
		if d.ChefFSSAI != "" {
			fssai = fmt.Sprintf(` <span class="muted">(FSSAI %s)</span>`, esc(d.ChefFSSAI))
		}
		parts += fmt.Sprintf(`<p style="margin:4px 0;color:#374151;font-size:14px;"><strong>Chef:</strong> %s%s</p>`, esc(d.ChefName), fssai)
	}
	if d.DeliveryAddress != "" {
		parts += fmt.Sprintf(`<p style="margin:4px 0;color:#374151;font-size:14px;"><strong>Delivering to:</strong> %s</p>`, esc(d.DeliveryAddress))
	}
	if d.ETA != "" {
		parts += fmt.Sprintf(`<p style="margin:4px 0;color:#374151;font-size:14px;"><strong>Expected by:</strong> %s</p>`, esc(d.ETA))
	}
	return fmt.Sprintf(`<div style="border-left:3px solid #111827;padding:8px 12px;margin:16px 0;">%s</div>`, parts)
}

// renderInvoiceSummary returns the GST breakup rows. Empty when no GST data
// is supplied so the legacy "total only" email stays unchanged.
func renderInvoiceSummary(d *OrderInvoiceDetails) string {
	hasBreakdown := d.Subtotal > 0 || d.DeliveryFee > 0 || d.PlatformFee > 0 || len(d.TaxLines) > 0
	if !hasBreakdown {
		return ""
	}
	rows := ""
	rowFmt := `<tr><td style="padding:6px 0;color:#4b5563;font-size:14px;">%s</td><td style="padding:6px 0;text-align:right;color:#374151;font-size:14px;">₹%.2f</td></tr>`
	if d.Subtotal > 0 {
		rows += fmt.Sprintf(rowFmt, "Subtotal", d.Subtotal)
	}
	if d.DeliveryFee > 0 {
		rows += fmt.Sprintf(rowFmt, "Delivery fee", d.DeliveryFee)
	}
	if d.PlatformFee > 0 {
		rows += fmt.Sprintf(rowFmt, "Service / platform fee", d.PlatformFee)
	}
	if d.Discount > 0 {
		rows += fmt.Sprintf(`<tr><td style="padding:6px 0;color:#15803D;font-size:14px;">Discount</td><td style="padding:6px 0;text-align:right;color:#15803D;font-size:14px;">-₹%.2f</td></tr>`, d.Discount)
	}
	for _, line := range d.TaxLines {
		rows += fmt.Sprintf(rowFmt, line.Label, line.Amount)
	}
	if d.Tip > 0 {
		rows += fmt.Sprintf(rowFmt, "Tip", d.Tip)
	}
	if d.Rounding != 0 {
		rows += fmt.Sprintf(rowFmt, "Rounding", d.Rounding)
	}
	return fmt.Sprintf(`<table width="100%%" cellpadding="0" cellspacing="0" style="margin:0 0 8px 0;">%s</table>`, rows)
}

// renderSupplierFooter returns the supplier of record block required by
// CGST Rule 46 (name + address of supplier). Always rendered with sensible
// defaults so the email is invoice-shaped even when the caller passes nil.
func renderSupplierFooter(d *OrderInvoiceDetails) string {
	return fmt.Sprintf(`<p class="muted" style="margin-top:24px;">Supplied by <strong>%s</strong>, %s.</p>`, esc(d.SupplierName), esc(d.SupplierAddress))
}

// renderRefundBlurb returns the short cancellation/refund policy line.
func renderRefundBlurb() string {
	return `<p class="muted">Cancellations and refunds are handled per our <a href="https://fe3dr.com/refund" style="color:#111827;">refund policy</a>. Contact support within 24 hours if anything is wrong with your order.</p>`
}

// OrderStatusUpdateHTML returns the order status update email
func OrderStatusUpdateHTML(orderNumber, status string) (subject, html string) {
	statusLabels := map[string]string{
		"confirmed":  "Your order has been confirmed by the chef",
		"preparing":  "Your chef is preparing your meal",
		"ready":      "Your order is ready for pickup",
		"picked_up":  "Your delivery partner has picked up your order",
		"on_the_way": "Your order is on the way!",
		"delivered":  "Your order has been delivered. Enjoy your meal!",
		"cancelled":  "Your order has been cancelled",
	}
	label := statusLabels[status]
	if label == "" {
		label = fmt.Sprintf("Order status: %s", status)
	}
	subject = fmt.Sprintf("Order #%s — %s", orderNumber, label)
	emoji := "📦"
	switch status {
	case "preparing":
		emoji = "👨‍🍳"
	case "ready":
		emoji = "✅"
	case "picked_up", "on_the_way":
		emoji = "🚗"
	case "delivered":
		emoji = "🎉"
	case "cancelled":
		emoji = "❌"
	}
	body := fmt.Sprintf(`
          <h2>%s Order Update</h2>
          <p>Order <strong>#%s</strong>: %s</p>
          <a href="https://fe3dr.com/orders" class="btn">View Order Details</a>
`, emoji, orderNumber, label)
	html = emailBase(subject, label, body)
	return
}

// ChefVerificationApprovedHTML returns the chef verification approval email
func ChefVerificationApprovedHTML(chefName string) (subject, html string) {
	subject = "Your kitchen is verified — Fe3dr"
	body := fmt.Sprintf(`
          <h2>Congratulations, %s!</h2>
          <p>Your kitchen has been verified and your chef profile is now <strong>live on Fe3dr</strong>. Customers in your area can now discover and order from your menu.</p>
          <p>Here's what to do next:</p>
          <ul style="color:#4b5563;font-size:15px;line-height:2;">
            <li>Set your operating hours and availability</li>
            <li>Add menu items with photos and descriptions</li>
            <li>Start accepting orders!</li>
          </ul>
          <a href="https://vendors.fe3dr.com/dashboard" class="btn">Go to Your Dashboard</a>
`, chefName)
	html = emailBase(subject, fmt.Sprintf("%s, your kitchen is live on Fe3dr!", chefName), body)
	return
}

// ChefNewOrderHTML returns the new order notification email for chefs
func ChefNewOrderHTML(orderNumber string, total float64) (subject, html string) {
	subject = fmt.Sprintf("New order #%s — Fe3dr", orderNumber)
	body := fmt.Sprintf(`
          <h2>🔔 New Order!</h2>
          <p>You have a new order <strong>#%s</strong> worth <strong>₹%.2f</strong>.</p>
          <p>Please review and accept the order from your dashboard.</p>
          <a href="https://vendors.fe3dr.com/orders" class="btn">View Order</a>
          <div class="info-box">
            <p>Customers expect a response within 5 minutes. Please accept or decline promptly.</p>
          </div>
`, orderNumber, total)
	html = emailBase(subject, fmt.Sprintf("New order #%s — ₹%.2f", orderNumber, total), body)
	return
}

// DeliveryAssignedHTML returns the delivery assignment notification
func DeliveryAssignedHTML(orderNumber, pickupAddress string) (subject, html string) {
	subject = fmt.Sprintf("Delivery assigned — Order #%s", orderNumber)
	body := fmt.Sprintf(`
          <h2>🚗 New Delivery!</h2>
          <p>You've been assigned order <strong>#%s</strong>.</p>
          <p><strong>Pickup location:</strong> %s</p>
          <a href="https://delivery.fe3dr.com/dashboard" class="btn">View Delivery Details</a>
`, orderNumber, pickupAddress)
	html = emailBase(subject, fmt.Sprintf("New delivery — pickup at %s", pickupAddress), body)
	return
}

// PasswordResetExpiredHTML is the page shown when a reset link is expired,
// unknown, or already used.
//
// Deliberately ONE page for all three cases. Distinguishing "expired" from
// "never existed" would tell someone probing tokens which of their guesses had
// once been real.
func PasswordResetExpiredHTML() string {
	return `<!doctype html>
<html lang="en"><head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="robots" content="noindex">
<title>Reset link no longer valid — Fe3dr</title>
<style>
  body{margin:0;background:#faf9f7;color:#1c1917;font:16px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",Inter,Roboto,sans-serif;
       display:flex;min-height:100vh;align-items:center;justify-content:center;padding:24px}
  .card{background:#fff;border:1px solid #e7e5e4;border-radius:16px;padding:32px;max-width:440px;
        box-shadow:0 1px 3px rgba(0,0,0,.04)}
  h1{margin:0 0 12px;font-size:20px;font-weight:600}
  p{margin:0 0 12px;color:#57534e}
  .muted{font-size:13px;color:#78716c;margin-top:20px}
</style></head><body>
  <div class="card">
    <h1>This reset link is no longer valid</h1>
    <p>Password reset links expire after 15 minutes and can only be used once.</p>
    <p>Open the Fe3dr app and choose <strong>Forgot password</strong> again to get a fresh one.</p>
    <p class="muted">Your account is unchanged, and your current password still works.</p>
  </div>
</body></html>`
}
