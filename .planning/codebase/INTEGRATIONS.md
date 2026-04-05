# External Integrations

**Analysis Date:** 2026-04-05

## Payment Processing

**Primary Payment Gateway:**
- Razorpay - Payment processing with Route (distributed ledger for splits)
  - SDK/Client: Custom wrapper in `apps/api/services/razorpay.go`
  - Publishable Key: `RAZORPAY_KEY_ID` (safe to expose to frontend)
  - Secret Key: `RAZORPAY_KEY_SECRET` (secure, cleared from config after init)
  - Webhook Secret: `RAZORPAY_WEBHOOK_SECRET` (held inside RazorpayClient struct)
  - Features:
    - Linked accounts (Razorpay Route) for chef/driver payouts
    - Order creation with payment splits
    - Refund handling
    - Webhook signature verification (HMAC-SHA256)
  - Integration Pattern: API-based, authenticated via Basic Auth with key ID and secret
  - Endpoint: `https://api.razorpay.com/v1`

**Legacy Gateway (Stripe):**
- Stripe - Configured but deprecated in favor of Razorpay
  - Keys: `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY`, `STRIPE_WEBHOOK_SECRET`
  - Status: Support code present but not actively used

## Email & Communications

**Email Service:**
- SendGrid v3 REST API
  - API Key: `SENDGRID_API_KEY` (required)
  - From Address: `FROM_EMAIL` (default: `noreply@homechef.com`)
  - From Name: `FROM_NAME` (default: `HomeChef`)
  - Implementation: `apps/api/services/email.go`
  - Endpoint: `https://api.sendgrid.com/v3/mail/send`
  - Email Types Sent:
    - Order confirmation (`SendOrderConfirmation`)
    - Order status updates (`SendOrderStatusUpdate`)
    - Password reset (`SendPasswordResetEmail`)
    - Chef verification approved (`SendChefVerificationApproved`)
    - Chef new order notification (`SendChefNewOrder`)
    - Delivery assigned (`SendDeliveryAssigned`)
    - Support ticket created/updated (`SendSupportTicketCreated`, `SendSupportTicketUpdate`)
    - Welcome emails (`SendWelcomeEmail`)
    - Staff invitations (`SendStaffInvitation`)
  - Graceful Degradation: Logs skip message if API key missing; no crash

**SMS Service (Twilio):**
- Twilio - SMS for transactional messages
  - Account SID: `TWILIO_ACCOUNT_SID`
  - Auth Token: `TWILIO_AUTH_TOKEN`
  - Phone Number: `TWILIO_PHONE_NUMBER` (sender ID)
  - Status: Configured but not actively integrated in handlers (placeholder for future)

## Data Storage

**Primary Database:**
- PostgreSQL 16 (Cloud SQL in production, docker-compose container in dev)
  - Connection String: `DATABASE_URL` or constructed from `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
  - Client: GORM v1.25.12 (`gorm.io/gorm`, `gorm.io/driver/postgres`)
  - SSL Mode: Configurable via `DB_SSLMODE`

**File Storage:**
- Google Cloud Storage (GCS)
  - Project ID: `GCS_PROJECT_ID` (default: `tesseracthub-480811`)
  - Public Bucket: `GCS_PUBLIC_BUCKET` (default: `homechef-prod-assets-in`)
    - Used for: Chef photos, dish images, user avatars
    - Returns: Public HTTPS URL (`https://storage.googleapis.com/{bucket}/{path}`)
  - Private Bucket: `GCS_PRIVATE_BUCKET` (default: `homechef-prod-docs-in`)
    - Used for: Verification documents, payment receipts, sensitive files
    - Access: Via signed URLs with configurable expiry
  - Client: `cloud.google.com/go/storage` v1.61.2
  - Implementation: `apps/api/services/storage.go`
  - Authentication: Workload Identity (GKE), ADC (Application Default Credentials) locally

**Caching Layer:**
- Redis 7
  - Connection: `REDIS_URL` (default: `redis://localhost:6379`)
  - Client: `github.com/redis/go-redis/v9` v9.18.0
  - Implementation: `apps/api/services/redis.go`
  - Uses: Session caching, rate limiting, temporary data

## Secret Management

**GCP Secret Manager:**
- Used for: Vendor/driver payment field secrets (Razorpay linked account credentials)
- Client: `cloud.google.com/go/secretmanager` v1.16.0
- Implementation: `apps/api/services/secrets.go`
- Secret ID Format:
  - `prod-vendor-payment-{vendorId}-{field}` (production)
  - `dev-vendor-payment-{vendorId}-{field}` (development)
  - Similar format for drivers: `prod-driver-payment-{driverId}-{field}`
- Operations:
  - `StoreVendorSecret(ctx, vendorID, field, value)` - Create/update secret
  - `GetVendorSecret(ctx, vendorID, field)` - Retrieve secret
  - `GetDriverSecret(ctx, driverID, field)` - Retrieve driver secret
- Authentication: Workload Identity (GKE), ADC locally

## Authentication & Identity

**Auth Provider:**
- Custom JWT-based authentication (OAuth-ready)
- Token Type: Bearer token in Authorization header
- Secret: `JWT_SECRET` (must be strong in production)
- Token Expiration: `JWT_EXPIRATION_HOURS` (default: 24)
- Refresh Token Expiration: `REFRESH_TOKEN_DAYS` (default: 30)
- Implementation: `apps/api/middleware/auth.go`

**OAuth Integrations (Configured, implementation-ready):**
- Google OAuth2
  - Client ID: `GOOGLE_CLIENT_ID`
  - Client Secret: `GOOGLE_CLIENT_SECRET`
  - Scope: Profile, email
- Facebook OAuth
  - App ID: `FACEBOOK_APP_ID`
  - App Secret: `FACEBOOK_APP_SECRET`
- Apple Sign-In
  - Client ID: `APPLE_CLIENT_ID`
  - Team ID: `APPLE_TEAM_ID`
  - Key ID: `APPLE_KEY_ID`

## Messaging & Events

**Event Broker:**
- NATS 2.10 with JetStream
  - URL: `NATS_URL` (default: `nats://localhost:4222`)
  - Client: `github.com/nats-io/nats.go` v1.47.0
  - Implementation: `apps/api/services/nats.go`
  - JetStream enabled for persistent message store
  - Usage Pattern: Publish-subscribe with queue groups for worker patterns

**Event Subjects Published:**
- Order events: `orders.created`, `orders.updated`, `orders.cancelled`, `orders.delivered`
- Payment events: `payments.success`, `payments.failed`
- Chef events: `chef.new_order`, `chef.verified`
- Delivery events: `delivery.assigned`, `delivery.picked_up`
- User events: `users.registered`, `reviews.posted`
- Notification events: `notifications.email`, `notifications.push`, `notifications.sms`
- Approval events: `approvals.created`, `approvals.approved`, `approvals.rejected`, `approvals.info_requested`
- Driver events: `driver.onboarding.submitted`, `driver.delivery.created`, `provider.delivery.updated`
- Subscription events: `subscription.created`, `subscription.activated`, `subscription.cancelled`, `subscription.invoice.created`
- Catering events: `catering.request`, `catering.quote`

**Notification Service:**
- Implementation: `apps/api/services/notifications.go`
- Subscribes to order, user, chef, delivery, and approval events
- Routes notifications to email, push, and SMS channels
- Queue-based processing with worker pattern: `notification-workers`

## Monitoring & Observability

**Metrics:**
- Prometheus metrics
  - Client: `github.com/prometheus/client_golang` v1.23.2
  - Standard Go HTTP metrics collected automatically
  - Custom metrics can be registered via Prometheus registry

**Logging:**
- Standard Go `log` package for startup and warnings
- JSON structured logs to stdout (suitable for GCP Cloud Logging)
- Log levels: Info, Warning, Error
- Pattern: Service startup logs, configuration checks, error context

## Currency & Exchange Rates

**Exchange Rate Services (Configured for future use):**
- Open Exchange Rates API
  - App ID: `OPENEXCHANGERATES_APP_ID`
  - Endpoint: `https://openexchangerates.io/api/latest`
  - Implementation: `apps/api/services/forex.go`
  - Usage: Currency conversion, multi-currency pricing

- ExchangeRates-API
  - API Key: `EXCHANGERATES_API_KEY`
  - Endpoint: `https://api.exchangerates.api.io/`
  - Backup service for exchange rate data

## Webhooks & Callbacks

**Incoming Webhooks:**
- Razorpay Webhooks
  - Endpoint: Handler in `apps/api/handlers/payment.go`
  - Event: Payment success/failure, refund completion
  - Verification: HMAC-SHA256 signature verification via `VerifyWebhookSignature()`
  - Payload: Razorpay payment event with order and transfer details

**Outgoing Webhooks:**
- Currently: None actively implemented
- Future: Provider delivery events, approval status changes (catering/support tickets)

## CI/CD & Deployment

**Containerization:**
- Docker multi-stage builds
  - `apps/api/Dockerfile` - Go API
  - `apps/web/Dockerfile` - Frontend apps
  - Base images: Alpine 3.19 (Go), Node 22 (frontends)

**Local Development:**
- Docker Compose (`docker-compose.yml`)
  - Services: postgres, redis, nats, api, web, vendor-portal, admin-portal, delivery-portal, adminer, mailhog
  - Network: `homechef-network`
  - Volumes: `postgres_data`, `redis_data`, `nats_data`, `api_uploads`

**Production Deployment (Infrastructure not in this repo):**
- Expects GCP infrastructure:
  - Cloud SQL PostgreSQL 16
  - GCS buckets (public, private)
  - Secret Manager
  - Cloud Run or GKE for containerized services
- Requires environment variables injected at runtime

## Environment Configuration

**Required Environment Variables (Production):**

Database:
- `DATABASE_URL` - Full connection string, or `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`

Authentication:
- `JWT_SECRET` - Strong random string
- `JWT_EXPIRATION_HOURS` - Token lifetime in hours
- `REFRESH_TOKEN_DAYS` - Refresh token lifetime in days

Payments:
- `RAZORPAY_KEY_ID` - Razorpay publishable key
- `RAZORPAY_KEY_SECRET` - Razorpay secret key
- `RAZORPAY_WEBHOOK_SECRET` - Webhook signature secret

Email:
- `SENDGRID_API_KEY` - API key for transactional emails
- `FROM_EMAIL` - Sender email address
- `FROM_NAME` - Sender display name

Storage & Secrets:
- `GCS_PROJECT_ID` - GCP project ID
- `GCS_PUBLIC_BUCKET` - Public assets bucket
- `GCS_PRIVATE_BUCKET` - Private documents bucket

Messaging:
- `REDIS_URL` - Redis connection string
- `NATS_URL` - NATS server connection string

Optional (OAuth, SMS, legacy):
- `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`
- `FACEBOOK_APP_ID`, `FACEBOOK_APP_SECRET`
- `APPLE_CLIENT_ID`, `APPLE_TEAM_ID`, `APPLE_KEY_ID`
- `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_PHONE_NUMBER`
- `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY`, `STRIPE_WEBHOOK_SECRET`
- `OPENEXCHANGERATES_APP_ID`, `EXCHANGERATES_API_KEY`

**Secrets Location:**
- Development: `.env` file (not committed, loaded via `godotenv`)
- Production: Injected as environment variables at container runtime
- GCP Secrets: Stored in Secret Manager, accessed via `cloud.google.com/go/secretmanager`

## Feature Flags

**Mock Mode:**
- Flag: `ENABLE_MOCK_MODE` (boolean, default: false)
- When true: Disables real API calls for testing
- Configuration: `apps/api/config/config.go`

---

*Integration audit: 2026-04-05*
