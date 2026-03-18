# Razorpay Integration Setup Guide

## Overview

Fe3dr uses **Razorpay Route** (split payments) so that customer payments go **directly** to chefs and drivers — never through Fe3dr's account. Fe3dr's only revenue comes from subscription fees.

### Payment Architecture

```
Customer pays ₹630 for an order:
  ├── ₹500 (food)        → Chef's bank account (via Razorpay Route)
  ├── ₹50  (delivery fee) → Driver's bank account (via Razorpay Route)
  ├── ₹30  (chef tip)     → Chef's bank account
  ├── ₹50  (driver tip)   → Driver's bank account
  └── ₹0                  → Fe3dr (we take nothing from orders)

Fe3dr revenue (separate):
  └── Subscription fee (₹299/mo driver, ₹499/mo chef) auto-deducted via Razorpay
```

---

## Step 1: Create a Razorpay Account

1. Go to [https://dashboard.razorpay.com/signup](https://dashboard.razorpay.com/signup)
2. Sign up with your business email
3. Complete KYC verification (PAN, Aadhaar, business docs)
4. Activate your account — this is Fe3dr's **primary account**

## Step 2: Enable Razorpay Route

1. In the Razorpay Dashboard → **Route** (left sidebar)
2. Click **"Request Access"** to enable Route (split payments)
3. Submit the required documents for Route activation
4. Once approved, you can create linked accounts and transfers

> **Note:** Route requires business verification. Processing time: 2-3 business days.

## Step 3: Get API Keys

1. Dashboard → **Settings** → **API Keys**
2. Generate a new key pair:
   - **Key ID** (starts with `rzp_live_` for production, `rzp_test_` for test mode)
   - **Key Secret** (shown only once — save it securely)
3. For testing, use the **Test Mode** toggle to get test keys

## Step 4: Set Up Webhooks

1. Dashboard → **Settings** → **Webhooks**
2. Add a new webhook:
   - **Webhook URL:** `https://api.yourdomain.com/webhooks/razorpay`
   - **Secret:** Generate a random secret (save it)
   - **Events to subscribe:**
     - `payment.captured`
     - `payment.failed`
     - `refund.processed`
     - `transfer.processed`
     - `subscription.charged`
     - `subscription.halted`
     - `subscription.cancelled`

## Step 5: Configure Environment Variables

Add these to your deployment environment (GKE secrets, `.env`, etc.):

```bash
RAZORPAY_KEY_ID=rzp_live_xxxxxxxxxxxxx
RAZORPAY_KEY_SECRET=xxxxxxxxxxxxxxxxxxxx
RAZORPAY_WEBHOOK_SECRET=your_webhook_secret_here
```

For the **Kubernetes deployment** (HomeChef), add these to the GCP Secret Manager:
- `prod-homechef-razorpay-key-id`
- `prod-homechef-razorpay-key-secret`
- `prod-homechef-razorpay-webhook-secret`

Then reference them in the Helm chart's ExternalSecret.

## Step 6: Create Linked Accounts for Chefs & Drivers

When a chef or driver completes onboarding and gets approved, create a Razorpay linked account:

### How it works:
1. Chef/Driver provides bank details during Razorpay's KYC flow
2. Our backend calls `POST /accounts` on Razorpay API
3. Razorpay verifies the account and assigns an `account_id`
4. We store this `account_id` in `ChefProfile.RazorpayAccountID` or `DeliveryPartner.RazorpayAccountID`
5. All future order payments are routed to this account

### API Call (backend handles this automatically):
```
POST https://api.razorpay.com/v2/accounts
{
  "email": "chef@example.com",
  "phone": "9876543210",
  "legal_business_name": "Chef's Kitchen",
  "business_type": "individual",
  "contact_name": "John Doe"
}
```

## Step 7: Frontend Checkout Integration

### Install Razorpay Checkout SDK

Add to your HTML/index.html:
```html
<script src="https://checkout.razorpay.com/v1/checkout.js"></script>
```

### Customer Payment Flow:

```typescript
// 1. Create Razorpay order via our API
const response = await apiClient.post(`/payments/order/${orderId}/create`);

// 2. Open Razorpay checkout
const options = {
  key: response.razorpayKeyId,
  amount: response.amount,
  currency: response.currency,
  name: "Fe3dr",
  description: `Order #${response.orderNumber}`,
  order_id: response.razorpayOrderId,
  prefill: response.prefill,
  handler: async function (rzResponse) {
    // 3. Verify payment on our backend
    await apiClient.post(`/payments/order/${orderId}/verify`, {
      razorpayPaymentId: rzResponse.razorpay_payment_id,
      razorpayOrderId: rzResponse.razorpay_order_id,
      razorpaySignature: rzResponse.razorpay_signature,
    });
  },
};

const rzp = new window.Razorpay(options);
rzp.open();
```

---

## Payment Flows

### 1. Customer Places Order

```
Customer → [Checkout] → Razorpay → [Auto-splits]:
  ├── Chef linked account: Subtotal + Tax + Chef Tip
  ├── Driver linked account: Delivery Fee + Driver Tip
  └── Fe3dr account: ₹0
```

### 2. Refund Flow

```
Refund initiated (by chef or admin)
  → Razorpay reverses the payment to customer
  → Route transfers to chef/driver are auto-reversed
  → Chef/driver see reversal in their settlement
```

**Refund rules:**
- **Before pickup:** Full refund, chef initiated
- **After pickup / preparing:** Chef decides amount (partial or full)
- **After delivery:** No auto-refund — handled as a dispute by admin
- **Admin override:** Admin can force refund at any stage

### 3. Subscription Auto-Deduction

```
Driver earns ≥ ₹3000 threshold
  → System generates invoice
  → Razorpay auto-charges from driver's payment method
  → ₹299 monthly fee goes to Fe3dr's Razorpay account
```

### 4. Tips

Tips are split separately:
- **Chef Tip** → Goes directly to chef's linked account
- **Driver Tip** → Goes directly to driver's linked account
- Fe3dr does not take any cut from tips

---

## API Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/payments/order/:id/create` | POST | Customer | Create Razorpay order with Route splits |
| `/payments/order/:id/verify` | POST | Customer | Verify payment after checkout |
| `/payments/order/:id/refund` | POST | Chef/Admin | Initiate refund |
| `/webhooks/razorpay` | POST | None (HMAC) | Razorpay webhook receiver |

---

## Testing

### Test Mode

1. Use test API keys (`rzp_test_*`)
2. Use Razorpay test cards:
   - Success: `4111 1111 1111 1111`
   - Failure: `4111 1111 1111 1234` (set to fail)
3. Test UPI: `success@razorpay` / `failure@razorpay`

### Webhook Testing

Use Razorpay's webhook testing tool in the dashboard, or use [ngrok](https://ngrok.com/) to expose your local server.

---

## Settlement Timeline

| Event | Settlement Time |
|-------|----------------|
| Payment captured | T+2 business days (to chef/driver linked account) |
| Refund processed | 5-10 business days (back to customer) |
| Subscription charge | Immediate deduction from payment method |

---

## Security

- All API calls use **HTTP Basic Auth** (Key ID + Key Secret)
- Webhooks verified via **HMAC-SHA256** signature
- Sensitive keys stored in **GCP Secret Manager** (not in code)
- Bank details handled by Razorpay (we never store card/bank info)
- PCI DSS compliance handled by Razorpay's checkout SDK
