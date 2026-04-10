package services

import "fmt"

// Enterprise HTML email templates for Fe3dr / HomeChef
// Responsive, branded layout matching the Fe3dr design system.
// Base colors: brand-500 (#FF6B35), brand-600 (#E55A2B), gray-900 (#111827)

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
    .header { background:linear-gradient(135deg,#FF6B35,#E55A2B); padding:32px 40px; text-align:center; border-radius:12px 12px 0 0; }
    .header h1 { margin:0; color:#ffffff; font-size:24px; font-weight:700; letter-spacing:0.5px; }
    .content { background:#ffffff; padding:40px; border-left:1px solid #e5e7eb; border-right:1px solid #e5e7eb; }
    .content h2 { color:#111827; font-size:22px; margin:0 0 16px 0; font-weight:600; }
    .content p { color:#4b5563; font-size:15px; line-height:1.7; margin:0 0 16px 0; }
    .btn { display:inline-block; background:#FF6B35; color:#ffffff !important; text-decoration:none; padding:14px 32px; border-radius:8px; font-weight:600; font-size:15px; margin:8px 0 24px 0; }
    .btn:hover { background:#E55A2B; }
    .info-box { background:#FFF7ED; border:1px solid #FDBA74; border-radius:8px; padding:16px 20px; margin:16px 0; }
    .info-box p { color:#9A3412; margin:0; font-size:14px; }
    .divider { border:none; border-top:1px solid #e5e7eb; margin:24px 0; }
    .footer { background:#f9fafb; padding:24px 40px; text-align:center; border:1px solid #e5e7eb; border-top:none; border-radius:0 0 12px 12px; }
    .footer p { color:#9ca3af; font-size:12px; line-height:1.6; margin:0; }
    .footer a { color:#FF6B35; text-decoration:none; }
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
`, firstName)
	html = emailBase(subject, "Welcome to Fe3dr — discover home-cooked meals near you", body)
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
          <span style="word-break:break-all;color:#FF6B35;">%s</span></p>
`, firstName, verifyURL, verifyURL)
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
`, inviterName, role, acceptURL)
	html = emailBase(subject, fmt.Sprintf("%s invited you to join Fe3dr", inviterName), body)
	return
}

// PasswordResetHTML returns the password reset email
func PasswordResetHTML(resetURL string) (subject, html string) {
	subject = "Reset your password — Fe3dr"
	body := fmt.Sprintf(`
          <h2>Reset your password</h2>
          <p>We received a request to reset the password for your Fe3dr account.</p>
          <a href="%s" class="btn">Reset Password</a>
          <div class="info-box">
            <p>This link expires in 1 hour. If you didn't request a password reset, you can safely ignore this email — your account is secure.</p>
          </div>
          <hr class="divider">
          <p class="muted">Button not working? Copy and paste this URL:<br>
          <span style="word-break:break-all;color:#FF6B35;">%s</span></p>
`, resetURL, resetURL)
	html = emailBase(subject, "Reset your Fe3dr password", body)
	return
}

// OrderConfirmationHTML returns the order confirmation email
func OrderConfirmationHTML(orderNumber string, items []OrderItemSummary, total float64) (subject, html string) {
	subject = fmt.Sprintf("Order #%s confirmed — Fe3dr", orderNumber)
	rows := ""
	for _, item := range items {
		rows += fmt.Sprintf(`<tr>
            <td style="padding:8px 0;border-bottom:1px solid #f3f4f6;color:#374151;">%s</td>
            <td style="padding:8px 0;border-bottom:1px solid #f3f4f6;color:#374151;text-align:center;">%d</td>
            <td style="padding:8px 0;border-bottom:1px solid #f3f4f6;color:#374151;text-align:right;">₹%.2f</td>
          </tr>`, item.Name, item.Quantity, item.Price)
	}
	body := fmt.Sprintf(`
          <h2>Order Confirmed!</h2>
          <p>Your order <strong>#%s</strong> has been placed successfully. Your home chef is preparing your meal!</p>
          <table width="100%%" cellpadding="0" cellspacing="0" style="margin:16px 0;">
            <tr style="background:#f9fafb;">
              <th style="padding:10px 0;text-align:left;color:#6b7280;font-size:13px;font-weight:600;">Item</th>
              <th style="padding:10px 0;text-align:center;color:#6b7280;font-size:13px;font-weight:600;">Qty</th>
              <th style="padding:10px 0;text-align:right;color:#6b7280;font-size:13px;font-weight:600;">Price</th>
            </tr>
            %s
            <tr>
              <td colspan="2" style="padding:12px 0;font-weight:700;color:#111827;">Total</td>
              <td style="padding:12px 0;text-align:right;font-weight:700;color:#FF6B35;font-size:18px;">₹%.2f</td>
            </tr>
          </table>
          <a href="https://fe3dr.com/orders" class="btn">Track Your Order</a>
`, orderNumber, rows, total)
	html = emailBase(subject, fmt.Sprintf("Order #%s confirmed — your meal is being prepared", orderNumber), body)
	return
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
