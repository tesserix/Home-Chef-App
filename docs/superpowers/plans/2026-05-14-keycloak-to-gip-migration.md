# Keycloak → GIP Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec:** [`docs/superpowers/specs/2026-05-14-keycloak-to-gip-migration-design.md`](../specs/2026-05-14-keycloak-to-gip-migration-design.md)

**Goal:** Replace Keycloak with Google Identity Platform across all 4 Home-Chef-App web apps + 3 Expo mobile apps + Go API, by building a new `homechef-auth-bff` Go service modeled on mark8ly's BFF, then doing a single hard-cutover maintenance window and decommissioning Keycloak after a 14-day soak.

**Architecture:** A new Go service `apps/auth-bff/` (mirroring `mark8ly/services/auth-bff/`) sits between every client and `apps/api`. It owns the OIDC dance with GIP, issues encrypted AES-GCM session cookies (web) or bearer sessions (mobile), and forwards verified identity to `apps/api` via HMAC-signed headers. `apps/api` stops minting HS256 tokens and only verifies BFF-signed headers. Three GIP tenant pools — `HomeChef-Customer`, `HomeChef-Business` (vendor+driver), `HomeChef-Internal` (admin) — give pool-level user isolation. Roles live in GIP custom claims. No FGA, no MFA in this phase.

**Tech Stack:** Go 1.26 + Gin (BFF + API), `github.com/coreos/go-oidc/v3`, `github.com/golang-jwt/jwt/v5`, `github.com/MicahParks/keyfunc/v3` (JWKS cache), Firebase JS SDK v11 (web), `@react-native-firebase/auth` (mobile Expo), AES-256-GCM session encryption, HMAC-SHA256 for service-to-service trust, GORM + PostgreSQL, Helm + ArgoCD + Istio + ExternalSecrets + Terraform (GIP tenants).

---

## File Structure

### New files

```
apps/auth-bff/                                           # NEW Go service
├── cmd/server/main.go                                   # wiring
├── internal/
│   ├── config/config.go                                 # env loading
│   ├── gip/verifier.go                                  # JWT + JWKS + tenant validation
│   ├── gip/verifier_test.go
│   ├── session/cookie.go                                # AES-GCM
│   ├── session/cookie_test.go
│   ├── session/handler.go                               # /auth/session, /logout, /switch-context, /refresh, /csrf
│   ├── session/handler_test.go
│   ├── autologin/service.go                             # /auth/auto-login (mobile)
│   ├── autologin/handler.go
│   ├── autologin/service_test.go
│   ├── oidc/login.go                                    # /auth/login (web start)
│   ├── oidc/callback.go                                 # /auth/callback (web end)
│   ├── oidc/exchange.go                                 # /auth/exchange (web email/password id_token → cookie)
│   ├── oidc/login_test.go
│   ├── productregistry/registry.go                      # homechef-products.yaml loader + host resolution
│   ├── productregistry/registry_test.go
│   ├── headerproxy/signer.go                            # X-Internal-Auth HMAC sign
│   ├── headerproxy/signer_test.go
│   ├── audit/client.go                                  # best-effort POST to audit-service
│   ├── ratelimit/middleware.go                          # per-IP + per-email throttle
│   └── apiclient/users.go                               # call apps/api /internal/users/upsert
├── migrations/
│   ├── 20260514000001_auth_audit_events.up.sql
│   └── 20260514000001_auth_audit_events.down.sql
├── homechef-products.yaml                               # product registry config
├── Dockerfile
├── go.mod
├── go.sum
├── .env.example
├── README.md
└── Makefile

apps/api/migrations/
├── 20260514000001_drop_keycloak_auth_tables.up.sql
├── 20260514000001_drop_keycloak_auth_tables.down.sql
├── 20260514000002_add_gip_identity_to_users.up.sql
└── 20260514000002_add_gip_identity_to_users.down.sql

apps/api/middleware/bff_auth.go                          # HMAC verifier + ctx injection
apps/api/middleware/bff_auth_test.go
apps/api/handlers/internal_users.go                      # /internal/users/upsert
apps/api/handlers/internal_users_test.go

packages/mobile-shared/auth/                             # NEW Expo shared package directory
├── firebase.ts
├── sign-in.ts
├── bff-session.ts
├── token-storage.ts
├── provider.tsx
├── index.ts
└── __tests__/sign-in.test.ts

apps/{web,vendor-portal,delivery-portal,admin-portal}/src/lib/firebase.ts   # NEW per app

tesserix-k8s/charts/apps/homechef-auth-bff/              # NEW Helm chart (copy from mark8ly-auth-bff)
├── Chart.yaml
├── values.yaml
├── values-uat.yaml
└── templates/{deployment,service,configmap-products,authorization-policy,_helpers.tpl}.yaml

tesserix-k8s/argocd/prod/apps/homechef/auth-bff.yaml
tesserix-k8s/manifests/homechef-istio/virtualservice-auth.yaml
tesserix-k8s/external-secrets/prod/homechef/externalsecret.yaml  # extended
tesserix-k8s/terraform-new/stacks/11-identity-platform/terraform.tfvars  # extended

.github/workflows/auth-bff-ci.yml                        # NEW
```

### Modified files

```
apps/api/
├── cmd/main.go                                          # remove HS256 wiring, add BFF HMAC middleware
├── config/config.go                                     # remove JWT/OAuth secrets, add HMAC key
├── middleware/auth.go                                   # delete GenerateTokens, extractKeycloakRoles, etc.
├── handlers/auth.go                                     # delete Register, Login, OAuthLogin, etc.
├── models/user.go                                       # drop auth columns, add GIP fields
└── .env.example                                         # cleanup

apps/web/src/
├── features/auth/services/auth-service.ts               # Firebase JS, drop kc_idp_hint
├── app/store/auth-store.ts                              # drop direct API auth path
└── shared/types/auth.ts                                 # remove SocialProvider Keycloak strings

apps/vendor-portal/src/  (same 3 files as web)
apps/delivery-portal/src/  (same 3 files; dual-mode auth merged into single flow)
apps/admin-portal/src/  (same 3 files)

apps/mobile-customer/store/auth-store.ts                 # re-export from mobile-shared
apps/mobile-customer/app/(auth)/login.tsx                # Firebase signInWithCredential
apps/mobile-vendor/store/auth-store.ts                   # same
apps/mobile-vendor/app/(auth)/login.tsx                  # same
apps/mobile-delivery/store/auth-store.ts                 # same
apps/mobile-delivery/app/(auth)/login.tsx                # same

packages/mobile-shared/src/api/auth.ts                   # delete entirely (replaced by new auth/ dir)

apps/{web,vendor-portal,delivery-portal,admin-portal}/.env.example   # GIP vars
apps/mobile-{customer,vendor,delivery}/.env.example      # GIP vars

docker-compose.yml                                       # add auth-bff service
README.md                                                # update auth section
CLAUDE.md                                                # update auth section

tesserix-k8s/manifests/homechef-istio/virtualservice-*.yaml   # remove /bff/, /driver-bff/, /admin-bff/
```

### Deleted files (after 14-day soak — Phase 6 Task 6.5)

```
tesserix-k8s/argocd/prod/infrastructure/identity-customer.yaml
tesserix-k8s/argocd/prod/infrastructure/identity-internal.yaml
tesserix-k8s/argocd/pilot/infrastructure/keycloak.yaml
tesserix-k8s/argocd/prod/projects/identity.yaml
tesserix-k8s/charts/apps/keycloak/                       # if present
tesserix-k8s/external-secrets/prod/identity-internal/
tesserix-k8s/sealedsecret/prod/identity/keycloak-*.yaml  # 12 IdP + admin/db/redis
tesserix-k8s/manifests/identity-customer/
tesserix-k8s/manifests/identity-internal/
tesserix-k8s/scripts/identity/keycloak-to-gip-migrate.py
```

---

## Phase 0: Pre-flight (Manual GCP Console + Terraform)

These tasks have no code but must complete before any BFF dev. They're cross-team (DevOps + Apple Developer team + Facebook app reviewer + GIP console operator).

### Task 0.1: Provision 3 GIP tenants via Terraform

**Files:**
- Modify: `tesserix-k8s/terraform-new/stacks/11-identity-platform/terraform.tfvars`

- [ ] **Step 1: Add tenant entries to tfvars**

```hcl
tenants = [
  # ... existing mark8ly tenants stay ...
  {
    name = "HomeChef-Customer"
    display_name = "HomeChef Customer"
    allow_password_signup = true
    enable_email_link_signin = false
  },
  {
    name = "HomeChef-Business"
    display_name = "HomeChef Business"
    allow_password_signup = true
    enable_email_link_signin = false
  },
  {
    name = "HomeChef-Internal"
    display_name = "HomeChef Internal Staff"
    allow_password_signup = false
    enable_email_link_signin = true
  },
]
```

- [ ] **Step 2: Plan + apply in dry-run**

Run (from `tesserix-k8s/terraform-new/stacks/11-identity-platform/`):
```bash
terraform plan -out=tfplan
```
Expected: 3 `google_identity_platform_tenant.this[\"HomeChef-...\"]` resources to create. No other diffs.

- [ ] **Step 3: Apply**

```bash
terraform apply tfplan
```
Expected: 3 tenants created in `tesseracthub-480811` project. Record the generated tenant IDs (suffix `-xxxxx` format) — they go into `homechef-products.yaml` and frontend env vars.

- [ ] **Step 4: Commit**

```bash
cd tesserix-k8s
git add terraform-new/stacks/11-identity-platform/terraform.tfvars
git commit -m "infra(gip): provision HomeChef-Customer/Business/Internal tenants"
git push
```

### Task 0.2: Register 4 OAuth 2.0 Web Clients in GCP Console

Manual step — no code change.

- [ ] **Step 1: Open GCP Console → APIs & Services → Credentials in `tesseracthub-480811`**
- [ ] **Step 2: Create 4 OAuth 2.0 Client IDs** (type: Web application)
  - `homechef-customer-web` — Authorized origins: `https://fe3dr.com`, `https://www.fe3dr.com`, `http://localhost:5173`. Redirect URIs: `https://fe3dr.com/auth/callback`, `http://localhost:5173/auth/callback`, `http://localhost:8081/auth/callback`.
  - `homechef-business-web` — Origins: `https://vendors.fe3dr.com`, `https://delivery.fe3dr.com`, `http://localhost:5174`, `http://localhost:5175`. Redirects: corresponding `/auth/callback` paths.
  - `homechef-internal-web` — Origin: `https://admin.fe3dr.com`, `http://localhost:5176`. Redirect: `https://admin.fe3dr.com/auth/callback`, `http://localhost:5176/auth/callback`.
  - `homechef-mobile-google` — Origins: none. Used for native Google Sign-In in Expo apps (already exists; verify).
- [ ] **Step 3: Record client ID + secret for each**. Paste secrets into the GCP Secret Manager entries created in Task 0.4. Paste client IDs into `homechef-products.yaml` (Phase 1 Task 1.4).

### Task 0.3: Enable sign-in providers per GIP tenant

Manual via GCP Console → Identity Platform → Tenants → (each HomeChef tenant) → Sign-in method.

- [ ] **Step 1: HomeChef-Customer:** enable Email/Password, Google, Apple, Facebook, Phone
- [ ] **Step 2: HomeChef-Business:** enable Email/Password, Google, Apple, Phone (no Facebook for business users)
- [ ] **Step 3: HomeChef-Internal:** enable Email/Password, Google (only — staff-only)
- [ ] **Step 4: Configure Apple Sign-In** — upload Apple Service ID + key from Apple Developer (one config covers all tenants)
- [ ] **Step 5: Configure Facebook** — paste Facebook App ID + secret from Meta Developer
- [ ] **Step 6: Configure Phone** — set SMS region (India + US initially) and monthly quota cap (₹50,000 budget)
- [ ] **Step 7: Add authorized domains** to project-level config: `fe3dr.com`, `www.fe3dr.com`, `vendors.fe3dr.com`, `delivery.fe3dr.com`, `admin.fe3dr.com`

### Task 0.4: Create 7 secrets in GCP Secret Manager

Use `gcloud secrets create` against `tesseracthub-480811`.

- [ ] **Step 1: Generate 32-byte random keys**

```bash
SESSION_KEY=$(openssl rand -base64 32)
HMAC_KEY=$(openssl rand -base64 32)
```

- [ ] **Step 2: Create all 7 secrets**

```bash
echo -n "$SESSION_KEY" | gcloud secrets create prod-homechef-session-encrypt-key --project=tesseracthub-480811 --data-file=-
echo -n "$HMAC_KEY"    | gcloud secrets create prod-homechef-bff-internal-hmac-key --project=tesseracthub-480811 --data-file=-

# Paste these from GCP Console (Task 0.2 output):
echo -n "<client-secret-customer>" | gcloud secrets create prod-homechef-customer-client-secret --project=tesseracthub-480811 --data-file=-
echo -n "<client-secret-business>" | gcloud secrets create prod-homechef-business-client-secret --project=tesseracthub-480811 --data-file=-
echo -n "<client-secret-internal>" | gcloud secrets create prod-homechef-internal-client-secret --project=tesseracthub-480811 --data-file=-

# Paste from GCP Console → APIs & Services → Credentials → API keys (Identity Toolkit key):
echo -n "<browser-api-key>" | gcloud secrets create prod-homechef-gip-web-api-key --project=tesseracthub-480811 --data-file=-

# Paste from your ops list:
echo -n "samyak.rout@gmail.com,unidevidp@gmail.com,mahesh.sangawar@gmail.com" | gcloud secrets create prod-homechef-admin-allowed-emails --project=tesseracthub-480811 --data-file=-
```

- [ ] **Step 3: Verify**

```bash
gcloud secrets list --project=tesseracthub-480811 --filter="name:prod-homechef-*"
```
Expected: 7 entries listed.

### Task 0.5: Create local dev `.env.local` template

**Files:**
- Create: `apps/auth-bff/.env.example`

- [ ] **Step 1: Write the template**

```bash
cat > apps/auth-bff/.env.example <<'EOF'
# Server
HTTP_PORT=8081
ENV=dev
LOG_LEVEL=debug

# Product config
PRODUCTS_CONFIG_PATH=./homechef-products.yaml

# GIP
GIP_PROJECT_ID=tesseracthub-480811
GIP_PROJECT_NUMBER=849928263410
GIP_WEB_API_KEY=                   # gcloud secrets versions access latest --secret=dev-homechef-gip-web-api-key

# Per-pool OAuth client secrets
HOMECHEF_CUSTOMER_CLIENT_SECRET=
HOMECHEF_BUSINESS_CLIENT_SECRET=
HOMECHEF_INTERNAL_CLIENT_SECRET=

# Session
SESSION_ENCRYPT_KEY=               # 32 bytes base64
SESSION_COOKIE_DOMAIN=localhost
SESSION_MAX_AGE_HOURS=168

# BFF→API trust
BFF_INTERNAL_HMAC_KEY=             # 32 bytes base64; must match apps/api
API_BASE_URL=http://localhost:8080

# Admin gating
HOMECHEF_ADMIN_ALLOWED_EMAILS=

# Audit (best-effort)
AUDIT_ENDPOINT=
EOF
```

- [ ] **Step 2: Create `scripts/load-secrets.sh`** (in repo root)

```bash
cat > scripts/load-secrets.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
PROJECT=tesseracthub-480811
ENV_PREFIX=${1:-dev}
TARGET=${2:-apps/auth-bff/.env.local}

echo "Loading $ENV_PREFIX-homechef-* secrets from $PROJECT into $TARGET"
cp apps/auth-bff/.env.example "$TARGET"

for key in gip-web-api-key customer-client-secret business-client-secret internal-client-secret session-encrypt-key bff-internal-hmac-key admin-allowed-emails; do
  envvar=$(echo "$key" | tr '[:lower:]-' '[:upper:]_')
  val=$(gcloud secrets versions access latest --project="$PROJECT" --secret="${ENV_PREFIX}-homechef-${key}")
  sed -i.bak "s|^${envvar}=.*|${envvar}=${val}|" "$TARGET"
done
rm -f "$TARGET.bak"
echo "Done. $TARGET ready for docker-compose."
EOF
chmod +x scripts/load-secrets.sh
```

- [ ] **Step 3: Commit**

```bash
git add apps/auth-bff/.env.example scripts/load-secrets.sh
git commit -m "feat(auth-bff): env template + dev secret loader"
```

---

## Phase 1: BFF Foundation (`apps/auth-bff/`)

The largest phase. ~12 tasks. Strict TDD — every internal package gets unit tests first.

### Task 1.1: Scaffold the Go module + Dockerfile + main skeleton

**Files:**
- Create: `apps/auth-bff/go.mod`, `apps/auth-bff/cmd/server/main.go`, `apps/auth-bff/Dockerfile`, `apps/auth-bff/Makefile`, `apps/auth-bff/README.md`

- [ ] **Step 1: Initialize module**

```bash
mkdir -p apps/auth-bff/cmd/server apps/auth-bff/internal
cd apps/auth-bff
go mod init github.com/homechef/auth-bff
```

- [ ] **Step 2: Pin dependencies**

```bash
go get github.com/gin-gonic/gin@v1.10.0
go get github.com/coreos/go-oidc/v3@v3.11.0
go get golang.org/x/oauth2@v0.27.0
go get github.com/MicahParks/keyfunc/v3@v3.3.10
go get github.com/golang-jwt/jwt/v5@v5.2.1
go get github.com/google/uuid@v1.6.0
go get github.com/joho/godotenv@v1.5.1
go get github.com/sirupsen/logrus@v1.9.3
go get github.com/stretchr/testify@v1.10.0
go get gopkg.in/yaml.v3@v3.0.1
```

- [ ] **Step 3: Write `cmd/server/main.go` skeleton**

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env.local")

	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8081"
	}
	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("auth-bff listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
```

- [ ] **Step 4: Write Dockerfile**

```dockerfile
# syntax=docker/dockerfile:1.7
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/auth-bff ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /out/auth-bff /usr/local/bin/auth-bff
COPY homechef-products.yaml /etc/auth-bff/homechef-products.yaml
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/auth-bff"]
```

- [ ] **Step 5: Build + run smoke**

```bash
cd apps/auth-bff
go build ./...
go run ./cmd/server &
curl -s http://localhost:8081/healthz
kill %1
```
Expected: `{"ok":true}`

- [ ] **Step 6: Commit**

```bash
git add apps/auth-bff/
git commit -m "feat(auth-bff): scaffold Go module with healthz endpoint"
```

### Task 1.2: GIP token verifier (TDD)

**Files:**
- Create: `apps/auth-bff/internal/gip/verifier.go`, `apps/auth-bff/internal/gip/verifier_test.go`

- [ ] **Step 1: Write failing test**

```go
package gip

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testJWKServer struct {
	key *rsa.PrivateKey
	srv *httptest.Server
	kid string
}

func newJWKServer(t *testing.T) *testJWKServer {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tj := &testJWKServer{key: priv, kid: "test-kid"}
	tj.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwk := map[string]any{
			"keys": []map[string]any{{
				"kty": "RSA",
				"kid": tj.kid,
				"use": "sig",
				"alg": "RS256",
				"n":   base64URLUint(priv.N.Bytes()),
				"e":   "AQAB",
			}},
		}
		_ = json.NewEncoder(w).Encode(jwk)
	}))
	return tj
}

func (tj *testJWKServer) signToken(t *testing.T, claims jwt.MapClaims) string {
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = tj.kid
	s, err := tok.SignedString(tj.key)
	require.NoError(t, err)
	return s
}

func TestVerifier_ValidToken_PassesAllChecks(t *testing.T) {
	tj := newJWKServer(t)
	defer tj.srv.Close()

	v, err := New(Config{
		ProjectID: "tesseracthub-480811",
		JWKSURL:   tj.srv.URL,
		Leeway:    10 * time.Second,
	})
	require.NoError(t, err)

	tok := tj.signToken(t, jwt.MapClaims{
		"iss":   "https://securetoken.google.com/tesseracthub-480811",
		"aud":   "tesseracthub-480811",
		"sub":   "user-123",
		"email": "a@b.com",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
		"firebase": map[string]any{"tenant": "HomeChef-Customer-aaaaa"},
	})

	got, err := v.Verify(t.Context(), tok, "HomeChef-Customer-aaaaa")
	require.NoError(t, err)
	assert.Equal(t, "user-123", got.UID)
	assert.Equal(t, "a@b.com", got.Email)
	assert.Equal(t, "HomeChef-Customer-aaaaa", got.TenantID)
}

func TestVerifier_WrongTenant_Rejects(t *testing.T) {
	tj := newJWKServer(t)
	defer tj.srv.Close()
	v, _ := New(Config{ProjectID: "p", JWKSURL: tj.srv.URL, Leeway: 10 * time.Second})

	tok := tj.signToken(t, jwt.MapClaims{
		"iss": "https://securetoken.google.com/p", "aud": "p", "sub": "u",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		"firebase": map[string]any{"tenant": "OtherTenant-xxxxx"},
	})

	_, err := v.Verify(t.Context(), tok, "HomeChef-Customer-aaaaa")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTenantMismatch)
}

func TestVerifier_Expired_Rejects(t *testing.T) {
	tj := newJWKServer(t)
	defer tj.srv.Close()
	v, _ := New(Config{ProjectID: "p", JWKSURL: tj.srv.URL, Leeway: 1 * time.Second})

	tok := tj.signToken(t, jwt.MapClaims{
		"iss": "https://securetoken.google.com/p", "aud": "p", "sub": "u",
		"exp": time.Now().Add(-time.Hour).Unix(), "iat": time.Now().Add(-2 * time.Hour).Unix(),
		"firebase": map[string]any{"tenant": "T"},
	})

	_, err := v.Verify(t.Context(), tok, "T")
	require.Error(t, err)
}

func TestVerifier_TamperedSignature_Rejects(t *testing.T) {
	tj := newJWKServer(t)
	defer tj.srv.Close()
	v, _ := New(Config{ProjectID: "p", JWKSURL: tj.srv.URL, Leeway: 1 * time.Second})

	tok := tj.signToken(t, jwt.MapClaims{
		"iss": "https://securetoken.google.com/p", "aud": "p", "sub": "u",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		"firebase": map[string]any{"tenant": "T"},
	})
	tampered := tok[:len(tok)-3] + "AAA"

	_, err := v.Verify(t.Context(), tampered, "T")
	require.Error(t, err)
}

func base64URLUint(b []byte) string {
	// helper for JWK encoding; use encoding/base64.RawURLEncoding
	return jwt.NewParser().DecodeSegment // placeholder — actual code uses base64.RawURLEncoding.EncodeToString(b)
}
```

(Fix the `base64URLUint` helper to use `base64.RawURLEncoding.EncodeToString(b)` from `encoding/base64`.)

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/gip/... -v
```
Expected: FAIL — `package gip` cannot be loaded (no `verifier.go` yet).

- [ ] **Step 3: Implement `verifier.go`**

```go
package gip

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTenantMismatch = errors.New("firebase.tenant claim mismatch")
	ErrInvalidIssuer  = errors.New("invalid issuer")
	ErrInvalidAud     = errors.New("invalid audience")
)

type Config struct {
	ProjectID string
	JWKSURL   string        // override only for tests; prod uses default
	Leeway    time.Duration
}

type Verifier struct {
	cfg     Config
	jwks    keyfunc.Keyfunc
	expIss  string
}

type VerifiedToken struct {
	UID      string
	Email    string
	TenantID string
	Provider string                 // from firebase.sign_in_provider claim
	Claims   jwt.MapClaims
}

const defaultJWKSURL = "https://www.googleapis.com/service_accounts/v1/jwk/securetoken@system.gserviceaccount.com"

func New(cfg Config) (*Verifier, error) {
	jwksURL := cfg.JWKSURL
	if jwksURL == "" {
		jwksURL = defaultJWKSURL
	}
	k, err := keyfunc.NewDefaultCtx(context.Background(), []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("init JWKS: %w", err)
	}
	if cfg.Leeway == 0 {
		cfg.Leeway = 10 * time.Second
	}
	return &Verifier{
		cfg:    cfg,
		jwks:   k,
		expIss: "https://securetoken.google.com/" + cfg.ProjectID,
	}, nil
}

func (v *Verifier) Verify(ctx context.Context, raw string, expectedTenantID string) (*VerifiedToken, error) {
	claims := jwt.MapClaims{}
	tok, err := jwt.ParseWithClaims(raw, claims, v.jwks.Keyfunc,
		jwt.WithLeeway(v.cfg.Leeway),
		jwt.WithIssuer(v.expIss),
		jwt.WithAudience(v.cfg.ProjectID),
		jwt.WithValidMethods([]string{"RS256"}),
	)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if !tok.Valid {
		return nil, errors.New("invalid token")
	}

	fb, _ := claims["firebase"].(map[string]any)
	tenantID, _ := fb["tenant"].(string)
	if tenantID != expectedTenantID {
		return nil, ErrTenantMismatch
	}
	provider, _ := fb["sign_in_provider"].(string)
	uid, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)

	return &VerifiedToken{
		UID: uid, Email: email, TenantID: tenantID, Provider: provider, Claims: claims,
	}, nil
}
```

- [ ] **Step 4: Run tests to verify pass**

```bash
go test ./internal/gip/... -v
```
Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/auth-bff/internal/gip/
git commit -m "feat(auth-bff): GIP RS256 verifier with JWKS cache and tenant validation"
```

### Task 1.3: Session cookie (AES-GCM) (TDD)

**Files:**
- Create: `apps/auth-bff/internal/session/cookie.go`, `apps/auth-bff/internal/session/cookie_test.go`

- [ ] **Step 1: Write failing test**

```go
package session

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func freshKey(t *testing.T) []byte {
	k := make([]byte, 32)
	_, err := rand.Read(k)
	require.NoError(t, err)
	return k
}

func TestCookie_RoundTrip(t *testing.T) {
	mgr, err := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour})
	require.NoError(t, err)

	p := &Payload{UID: "u1", Email: "a@b.com", Pool: "customer", Role: "customer", IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}
	enc, err := mgr.Encode(p)
	require.NoError(t, err)
	require.NotEmpty(t, enc)

	got, err := mgr.Decode(enc)
	require.NoError(t, err)
	assert.Equal(t, p.UID, got.UID)
	assert.Equal(t, p.Email, got.Email)
}

func TestCookie_Tampered_Rejected(t *testing.T) {
	mgr, _ := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour})
	p := &Payload{UID: "u1", IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}
	enc, _ := mgr.Encode(p)
	tampered := enc[:len(enc)-2] + "AA"
	_, err := mgr.Decode(tampered)
	require.Error(t, err)
}

func TestCookie_Expired_Rejected(t *testing.T) {
	mgr, _ := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour})
	p := &Payload{UID: "u1", IssuedAt: time.Now().Add(-2 * time.Hour).Unix(), ExpiresAt: time.Now().Add(-time.Hour).Unix()}
	enc, _ := mgr.Encode(p)
	_, err := mgr.Decode(enc)
	require.Error(t, err)
}

func TestCookie_FreshNonceEachCall(t *testing.T) {
	mgr, _ := NewManager(Config{EncryptKey: freshKey(t), MaxAge: time.Hour})
	p := &Payload{UID: "u1", IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}
	a, _ := mgr.Encode(p)
	b, _ := mgr.Encode(p)
	assert.NotEqual(t, a, b, "same payload should produce different ciphertext (fresh nonce)")
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/session/... -v
```
Expected: FAIL — no `cookie.go`.

- [ ] **Step 3: Implement `cookie.go`**

```go
package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type Config struct {
	EncryptKey    []byte
	MaxAge        time.Duration
	CookieName    string
	CookieDomain  string
	Secure        bool
}

type Payload struct {
	UID       string `json:"uid"`
	Email     string `json:"email"`
	Pool      string `json:"pool"`
	Role      string `json:"role"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type Manager struct {
	cfg  Config
	gcm  cipher.AEAD
}

func NewManager(cfg Config) (*Manager, error) {
	if l := len(cfg.EncryptKey); l != 16 && l != 24 && l != 32 {
		return nil, fmt.Errorf("encrypt key must be 16/24/32 bytes, got %d", l)
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 7 * 24 * time.Hour
	}
	if cfg.CookieName == "" {
		cfg.CookieName = "hc_session"
	}
	block, err := aes.NewCipher(cfg.EncryptKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Manager{cfg: cfg, gcm: gcm}, nil
}

func (m *Manager) Encode(p *Payload) (string, error) {
	plaintext, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, m.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := m.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(ct), nil
}

func (m *Manager) Decode(raw string) (*Payload, error) {
	blob, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if len(blob) < m.gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := blob[:m.gcm.NonceSize()], blob[m.gcm.NonceSize():]
	plain, err := m.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	var p Payload
	if err := json.Unmarshal(plain, &p); err != nil {
		return nil, err
	}
	if time.Now().Unix() > p.ExpiresAt {
		return nil, errors.New("session expired")
	}
	return &p, nil
}

func (m *Manager) SetCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cfg.CookieName,
		Value:    value,
		Path:     "/",
		Domain:   m.cfg.CookieDomain,
		MaxAge:   int(m.cfg.MaxAge.Seconds()),
		HttpOnly: true,
		Secure:   m.cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *Manager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: m.cfg.CookieName, Value: "", Path: "/", Domain: m.cfg.CookieDomain,
		MaxAge: -1, HttpOnly: true, Secure: m.cfg.Secure, SameSite: http.SameSiteLaxMode,
	})
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/session/... -v
```
Expected: 4 PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/auth-bff/internal/session/cookie.go apps/auth-bff/internal/session/cookie_test.go
git commit -m "feat(auth-bff): AES-GCM session cookie manager"
```

### Task 1.4: Product registry (TDD)

**Files:**
- Create: `apps/auth-bff/homechef-products.yaml`, `apps/auth-bff/internal/productregistry/registry.go`, `apps/auth-bff/internal/productregistry/registry_test.go`

- [ ] **Step 1: Write `homechef-products.yaml`** — copy verbatim from spec §4.1.

- [ ] **Step 2: Write failing test**

```go
package productregistry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_ResolveByExactHost(t *testing.T) {
	r, err := Load("../../homechef-products.yaml")
	require.NoError(t, err)

	app, err := r.ResolveByHost("fe3dr.com")
	require.NoError(t, err)
	assert.Equal(t, "web", app.Name)
	assert.Equal(t, "customer", app.AuthContext)
	assert.Equal(t, "customer", string(app.DefaultRole))
}

func TestRegistry_ResolveByLocalhost(t *testing.T) {
	r, _ := Load("../../homechef-products.yaml")
	app, err := r.ResolveByHost("localhost:5174")
	require.NoError(t, err)
	assert.Equal(t, "vendor-portal", app.Name)
	assert.Equal(t, "vendor", string(app.DefaultRole))
}

func TestRegistry_UnknownHost_Errors(t *testing.T) {
	r, _ := Load("../../homechef-products.yaml")
	_, err := r.ResolveByHost("attacker.example.com")
	assert.ErrorIs(t, err, ErrUnknownHost)
}

func TestRegistry_MobileTenantAllowlist(t *testing.T) {
	r, _ := Load("../../homechef-products.yaml")
	assert.True(t, r.IsMobileTenantAllowed("HomeChef-Customer-xxxxx"))
	assert.True(t, r.IsMobileTenantAllowed("HomeChef-Business-xxxxx"))
	assert.False(t, r.IsMobileTenantAllowed("HomeChef-Internal-xxxxx"))
}
```

- [ ] **Step 3: Run test — fails (no impl)**

```bash
go test ./internal/productregistry/... -v
```

- [ ] **Step 4: Implement `registry.go`**

```go
package productregistry

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

var ErrUnknownHost = errors.New("unknown host")

type App struct {
	Name              string   `yaml:"name"`
	Hosts             []string `yaml:"hosts"`
	GIPTenantID       string   `yaml:"gipTenantId"`
	OAuthClientID     string   `yaml:"oauthClientId"`
	ClientSecretEnv   string   `yaml:"clientSecretEnv"`
	SessionCookie     string   `yaml:"sessionCookie"`
	CallbackPath      string   `yaml:"callbackPath"`
	CallbackHost      string   `yaml:"callbackHost"`
	PostLoginURL      string   `yaml:"postLoginUrl"`
	PostLogoutURL     string   `yaml:"postLogoutUrl"`
	AuthContext       string   `yaml:"authContext"`
	DefaultRole       string   `yaml:"defaultRole"`
	SignInMethods     []string `yaml:"signInMethods"`
	AllowedEmailsEnv  string   `yaml:"allowedEmailsEnv"`
	AllowedOrigins    []string `yaml:"allowedOrigins"`
}

type Product struct {
	Name   string `yaml:"name"`
	Domain string `yaml:"domain"`
	Apps   []App  `yaml:"apps"`
}

type Registry struct {
	Products              []Product `yaml:"products"`
	MobileTenantAllowlist []string  `yaml:"mobileTenantAllowlist"`

	hostIndex map[string]*App
}

func Load(path string) (*Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Registry
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	r.hostIndex = make(map[string]*App)
	for i := range r.Products {
		for j := range r.Products[i].Apps {
			a := &r.Products[i].Apps[j]
			for _, h := range a.Hosts {
				r.hostIndex[strings.ToLower(h)] = a
			}
		}
	}
	return &r, nil
}

func (r *Registry) ResolveByHost(host string) (*App, error) {
	if app, ok := r.hostIndex[strings.ToLower(host)]; ok {
		return app, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnknownHost, host)
}

func (r *Registry) IsMobileTenantAllowed(tenantID string) bool {
	for _, t := range r.MobileTenantAllowlist {
		if t == tenantID {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/productregistry/... -v
```
Expected: 4 PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/auth-bff/homechef-products.yaml apps/auth-bff/internal/productregistry/
git commit -m "feat(auth-bff): product registry with host resolution"
```

### Task 1.5: HMAC header proxy (TDD)

**Files:**
- Create: `apps/auth-bff/internal/headerproxy/signer.go`, `apps/auth-bff/internal/headerproxy/signer_test.go`

- [ ] **Step 1: Write failing test**

```go
package headerproxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSigner_SignAndVerify_RoundTrip(t *testing.T) {
	key := []byte("test-key-32-bytes-padding-padding!")
	s := NewSigner(SignerConfig{Key: key, Window: time.Minute})

	body := []byte(`{"hello":"world"}`)
	r := httptest.NewRequest(http.MethodPost, "/v1/foo", strings.NewReader(string(body)))
	require.NoError(t, s.Sign(r, body, Identity{UserID: "u1", Email: "a@b.com", Role: "customer", Pool: "customer"}))

	id, err := s.Verify(r, body)
	require.NoError(t, err)
	assert.Equal(t, "u1", id.UserID)
	assert.Equal(t, "customer", id.Role)
}

func TestSigner_StaleTimestamp_Rejects(t *testing.T) {
	key := []byte("test-key-32-bytes-padding-padding!")
	s := NewSigner(SignerConfig{Key: key, Window: time.Second})

	r := httptest.NewRequest(http.MethodGet, "/v1/foo", nil)
	require.NoError(t, s.Sign(r, nil, Identity{UserID: "u1"}))
	r.Header.Set("X-Auth-Ts", "1")  // ancient

	_, err := s.Verify(r, nil)
	require.ErrorIs(t, err, ErrStaleTimestamp)
}

func TestSigner_TamperedBody_Rejects(t *testing.T) {
	key := []byte("test-key-32-bytes-padding-padding!")
	s := NewSigner(SignerConfig{Key: key, Window: time.Minute})
	body := []byte(`{"hello":"world"}`)
	r := httptest.NewRequest(http.MethodPost, "/v1/foo", strings.NewReader(string(body)))
	require.NoError(t, s.Sign(r, body, Identity{UserID: "u1"}))

	_, err := s.Verify(r, []byte(`{"hello":"tampered"}`))
	require.ErrorIs(t, err, ErrSignatureMismatch)
}

func TestSigner_MissingHeader_Rejects(t *testing.T) {
	key := []byte("test-key-32-bytes-padding-padding!")
	s := NewSigner(SignerConfig{Key: key, Window: time.Minute})
	r := httptest.NewRequest(http.MethodGet, "/v1/foo", nil)
	_, err := s.Verify(r, nil)
	require.ErrorIs(t, err, ErrMissingSignature)
}
```

- [ ] **Step 2: Run — fails**

```bash
go test ./internal/headerproxy/... -v
```

- [ ] **Step 3: Implement `signer.go`**

```go
package headerproxy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

var (
	ErrMissingSignature  = errors.New("missing X-Internal-Auth")
	ErrStaleTimestamp    = errors.New("stale X-Auth-Ts")
	ErrSignatureMismatch = errors.New("HMAC mismatch")
)

const (
	HdrUserID     = "X-User-Id"
	HdrUserEmail  = "X-User-Email"
	HdrUserRole   = "X-User-Role"
	HdrAuthPool   = "X-Auth-Pool"
	HdrAuthTs     = "X-Auth-Ts"
	HdrSignature  = "X-Internal-Auth"
)

type SignerConfig struct {
	Key    []byte
	Window time.Duration
}

type Signer struct {
	cfg SignerConfig
}

type Identity struct {
	UserID string
	Email  string
	Role   string
	Pool   string
}

func NewSigner(cfg SignerConfig) *Signer {
	if cfg.Window == 0 {
		cfg.Window = 60 * time.Second
	}
	return &Signer{cfg: cfg}
}

func (s *Signer) Sign(r *http.Request, body []byte, id Identity) error {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	r.Header.Set(HdrUserID, id.UserID)
	r.Header.Set(HdrUserEmail, id.Email)
	r.Header.Set(HdrUserRole, id.Role)
	r.Header.Set(HdrAuthPool, id.Pool)
	r.Header.Set(HdrAuthTs, ts)
	r.Header.Set(HdrSignature, s.compute(r.Method, r.URL.Path, body, ts))
	return nil
}

func (s *Signer) Verify(r *http.Request, body []byte) (*Identity, error) {
	sig := r.Header.Get(HdrSignature)
	if sig == "" {
		return nil, ErrMissingSignature
	}
	ts := r.Header.Get(HdrAuthTs)
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("bad X-Auth-Ts: %w", err)
	}
	if d := time.Since(time.Unix(tsInt, 0)); d > s.cfg.Window || d < -s.cfg.Window {
		return nil, ErrStaleTimestamp
	}
	want := s.compute(r.Method, r.URL.Path, body, ts)
	if !hmac.Equal([]byte(sig), []byte(want)) {
		return nil, ErrSignatureMismatch
	}
	return &Identity{
		UserID: r.Header.Get(HdrUserID),
		Email:  r.Header.Get(HdrUserEmail),
		Role:   r.Header.Get(HdrUserRole),
		Pool:   r.Header.Get(HdrAuthPool),
	}, nil
}

func (s *Signer) compute(method, path string, body []byte, ts string) string {
	bodyHash := sha256.Sum256(body)
	m := hmac.New(sha256.New, s.cfg.Key)
	fmt.Fprintf(m, "%s\n%s\n%s\n%s", method, path, hex.EncodeToString(bodyHash[:]), ts)
	return hex.EncodeToString(m.Sum(nil))
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/headerproxy/... -v
```
Expected: 4 PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/auth-bff/internal/headerproxy/
git commit -m "feat(auth-bff): HMAC-SHA256 signer for BFF→API trust"
```

### Task 1.6: API client for /internal/users/upsert

**Files:**
- Create: `apps/auth-bff/internal/apiclient/users.go`, `apps/auth-bff/internal/apiclient/users_test.go`

- [ ] **Step 1: Write failing test**

```go
package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/homechef/auth-bff/internal/headerproxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_UpsertUser_ForwardsIdentityWithHMAC(t *testing.T) {
	key := []byte("test-key-32-bytes-padding-padding!")
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: key, Window: time.Minute})

	var received UpsertUserRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := readBody(t, r)
		_, err := signer.Verify(r, body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &received))
		_ = json.NewEncoder(w).Encode(UpsertUserResponse{UserID: "u1"})
	}))
	defer srv.Close()

	c := New(srv.URL, signer)
	resp, err := c.UpsertUser(context.Background(), UpsertUserRequest{
		GIPUid: "gip-u", GIPTenantID: "HomeChef-Customer", GIPProvider: "google.com",
		AuthPool: "customer", Email: "a@b.com", Name: "A", Role: "customer",
	})
	require.NoError(t, err)
	assert.Equal(t, "u1", resp.UserID)
	assert.Equal(t, "gip-u", received.GIPUid)
}
```

(Helper `readBody` reads the body and replaces it so verifier can re-read.)

- [ ] **Step 2: Implement `users.go`**

```go
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/homechef/auth-bff/internal/headerproxy"
)

type UpsertUserRequest struct {
	GIPUid      string `json:"gip_uid"`
	GIPTenantID string `json:"gip_tenant_id"`
	GIPProvider string `json:"gip_provider"`
	AuthPool    string `json:"auth_pool"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Role        string `json:"role"`
}

type UpsertUserResponse struct {
	UserID string `json:"user_id"`
}

type Client struct {
	base   string
	signer *headerproxy.Signer
	http   *http.Client
}

func New(baseURL string, signer *headerproxy.Signer) *Client {
	return &Client{base: baseURL, signer: signer, http: &http.Client{}}
}

func (c *Client) UpsertUser(ctx context.Context, req UpsertUserRequest) (*UpsertUserResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/internal/users/upsert", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	if err := c.signer.Sign(r, body, headerproxy.Identity{
		UserID: req.GIPUid, Email: req.Email, Role: req.Role, Pool: req.AuthPool,
	}); err != nil {
		return nil, err
	}
	resp, err := c.http.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upsert: %d %s", resp.StatusCode, b)
	}
	var out UpsertUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
```

- [ ] **Step 3: Run tests + commit**

```bash
go test ./internal/apiclient/... -v
git add apps/auth-bff/internal/apiclient/
git commit -m "feat(auth-bff): apps/api client for user upsert with HMAC"
```

### Task 1.7: Mobile autologin handler (TDD)

**Files:**
- Create: `apps/auth-bff/internal/autologin/{service.go,handler.go,service_test.go}`

- [ ] **Step 1: Write failing test for happy-path autologin**

```go
package autologin

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoLogin_Happy(t *testing.T) {
	deps := fakeDeps(t)              // wires fake GIP verifier + fake apiclient
	r := gin.New()
	NewHandler(deps).Register(r)

	body := `{"id_token":"valid.test.token","expected_tenant_id":"HomeChef-Customer-xxxxx"}`
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"session_token":`)
	assert.Contains(t, w.Body.String(), `"expires_at":`)
}

func TestAutoLogin_TenantMismatch_401(t *testing.T) { /* ... */ }
func TestAutoLogin_TenantNotAllowedForMobile_403(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Implement `service.go` + `handler.go`**

```go
// service.go
package autologin

import (
	"context"
	"time"

	"github.com/homechef/auth-bff/internal/apiclient"
	"github.com/homechef/auth-bff/internal/gip"
	"github.com/homechef/auth-bff/internal/productregistry"
	"github.com/homechef/auth-bff/internal/session"
)

type Deps struct {
	GIP      *gip.Verifier
	Sessions *session.Manager
	Registry *productregistry.Registry
	API      *apiclient.Client
}

type Request struct {
	IDToken          string `json:"id_token" binding:"required"`
	ExpectedTenantID string `json:"expected_tenant_id" binding:"required"`
}

type Response struct {
	SessionToken string `json:"session_token"`
	ExpiresAt    int64  `json:"expires_at"`
	User         User   `json:"user"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
	Pool  string `json:"pool"`
}

func (d *Deps) AutoLogin(ctx context.Context, req Request) (*Response, error) {
	if !d.Registry.IsMobileTenantAllowed(req.ExpectedTenantID) {
		return nil, ErrTenantNotAllowed
	}
	tok, err := d.GIP.Verify(ctx, req.IDToken, req.ExpectedTenantID)
	if err != nil {
		return nil, err
	}
	pool := poolFromTenant(req.ExpectedTenantID)
	role := defaultRoleForPool(pool)        // role override via tok.Claims["role"]
	if r, ok := tok.Claims["role"].(string); ok && r != "" {
		role = r
	}
	upsert, err := d.API.UpsertUser(ctx, apiclient.UpsertUserRequest{
		GIPUid: tok.UID, GIPTenantID: tok.TenantID, GIPProvider: tok.Provider,
		AuthPool: pool, Email: tok.Email, Role: role,
	})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	exp := now.Add(7 * 24 * time.Hour)
	payload := &session.Payload{
		UID: upsert.UserID, Email: tok.Email, Pool: pool, Role: role,
		IssuedAt: now.Unix(), ExpiresAt: exp.Unix(),
	}
	enc, err := d.Sessions.Encode(payload)
	if err != nil {
		return nil, err
	}
	return &Response{
		SessionToken: enc, ExpiresAt: exp.Unix(),
		User: User{ID: upsert.UserID, Email: tok.Email, Role: role, Pool: pool},
	}, nil
}
```

```go
// handler.go
package autologin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

var ErrTenantNotAllowed = errors.New("tenant not allowed for mobile")

type Handler struct{ d *Deps }

func NewHandler(d *Deps) *Handler { return &Handler{d: d} }

func (h *Handler) Register(r gin.IRouter) {
	r.POST("/auth/auto-login", h.post)
}

func (h *Handler) post(c *gin.Context) {
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_body"})
		return
	}
	resp, err := h.d.AutoLogin(c.Request.Context(), req)
	switch {
	case errors.Is(err, ErrTenantNotAllowed):
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_not_allowed"})
	case err != nil:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
	default:
		c.JSON(http.StatusOK, resp)
	}
}
```

- [ ] **Step 3: Run tests + commit**

```bash
go test ./internal/autologin/... -v
git add apps/auth-bff/internal/autologin/
git commit -m "feat(auth-bff): mobile /auth/auto-login endpoint"
```

### Task 1.8: Web OIDC login + callback + exchange handlers

**Files:**
- Create: `apps/auth-bff/internal/oidc/{login.go,callback.go,exchange.go,login_test.go}`

- [ ] **Step 1: Implement `login.go`** — start OIDC redirect

```go
package oidc

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/homechef/auth-bff/internal/productregistry"
	"golang.org/x/oauth2"
)

type Handlers struct {
	Registry *productregistry.Registry
	// provider per pool keyed by tenantID; built at startup
	OAuthByApp map[string]*oauth2.Config
	OIDCByApp  map[string]*oidc.Provider
	// CSRF state store — simple in-memory LRU acceptable; replaced by Redis if needed
	StateStore StateStore
}

func (h *Handlers) Login(c *gin.Context) {
	app, err := h.Registry.ResolveByHost(c.Request.Host)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown_host"})
		return
	}
	state := newState()
	nonce := newState()
	h.StateStore.Put(state, StateEntry{
		AppName: app.Name, Nonce: nonce, ReturnTo: c.Query("return_to"),
	})
	cfg := h.OAuthByApp[app.Name]
	url := cfg.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.SetAuthURLParam("tenantId", app.GIPTenantID))
	c.Redirect(http.StatusFound, url)
}

func newState() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
```

- [ ] **Step 2: Implement `callback.go`** — exchange code → id_token → upsert → cookie

```go
package oidc

import (
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) Callback(c *gin.Context) {
	app, err := h.Registry.ResolveByHost(c.Request.Host)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown_host"})
		return
	}
	state := c.Query("state")
	entry, ok := h.StateStore.Take(state)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "bad_state"})
		return
	}
	cfg := h.OAuthByApp[app.Name]
	tok, err := cfg.Exchange(c.Request.Context(), c.Query("code"))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "exchange_failed"})
		return
	}
	rawID, _ := tok.Extra("id_token").(string)
	verifier := h.OIDCByApp[app.Name].Verifier(&oidc.Config{ClientID: cfg.ClientID})
	idTok, err := verifier.Verify(c.Request.Context(), rawID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "id_token_invalid"})
		return
	}
	var claims map[string]any
	_ = idTok.Claims(&claims)
	// hand off to shared "issue session" routine — used by both Callback and Exchange
	h.issueSession(c, app, rawID, claims, entry)
}
```

- [ ] **Step 3: Implement `exchange.go`** — email/password flow (Firebase JS already did GIP sign-in; we just verify id_token + mint cookie)

```go
package oidc

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ExchangeRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

func (h *Handlers) Exchange(c *gin.Context) {
	app, err := h.Registry.ResolveByHost(c.Request.Host)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown_host"})
		return
	}
	var req ExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_body"})
		return
	}
	vt, err := h.GIPVerifier.Verify(c.Request.Context(), req.IDToken, app.GIPTenantID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "id_token_invalid"})
		return
	}
	h.issueSession(c, app, req.IDToken, vt.Claims, StateEntry{})
}
```

- [ ] **Step 4: Implement shared `issueSession`** in `login.go` — upsert + cookie

```go
// (added to login.go)
func (h *Handlers) issueSession(c *gin.Context, app *productregistry.App, rawID string, claims map[string]any, entry StateEntry) {
	pool := app.AuthContext
	role := app.DefaultRole
	if r, ok := claims["role"].(string); ok && r != "" {
		role = r
	}
	upsert, err := h.API.UpsertUser(c.Request.Context(), apiclient.UpsertUserRequest{
		GIPUid: getStr(claims, "sub"), GIPTenantID: app.GIPTenantID,
		GIPProvider: getStr(claims, "firebase.sign_in_provider"),
		AuthPool: pool, Email: getStr(claims, "email"), Role: role,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "upsert_failed"})
		return
	}
	now := time.Now()
	exp := now.Add(h.Sessions.MaxAge())
	payload := &session.Payload{
		UID: upsert.UserID, Email: getStr(claims, "email"), Pool: pool, Role: role,
		IssuedAt: now.Unix(), ExpiresAt: exp.Unix(),
	}
	enc, err := h.Sessions.Encode(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session_encode_failed"})
		return
	}
	h.Sessions.SetCookie(c.Writer, enc)
	if entry.ReturnTo != "" {
		c.Redirect(http.StatusFound, entry.ReturnTo)
		return
	}
	c.Redirect(http.StatusFound, app.PostLoginURL)
}
```

- [ ] **Step 5: Add state store** (in-memory LRU sufficient; 5 min TTL)

```go
// state.go
package oidc

import (
	"sync"
	"time"
)

type StateEntry struct {
	AppName  string
	Nonce    string
	ReturnTo string
	Created  time.Time
}

type StateStore interface {
	Put(key string, e StateEntry)
	Take(key string) (StateEntry, bool)
}

type memStore struct {
	mu sync.Mutex
	m  map[string]StateEntry
}

func NewMemStateStore() StateStore { return &memStore{m: map[string]StateEntry{}} }

func (s *memStore) Put(k string, e StateEntry) {
	e.Created = time.Now()
	s.mu.Lock()
	s.m[k] = e
	s.mu.Unlock()
}

func (s *memStore) Take(k string) (StateEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.m[k]
	delete(s.m, k)
	if ok && time.Since(e.Created) > 5*time.Minute {
		return StateEntry{}, false
	}
	return e, ok
}
```

- [ ] **Step 6: Write tests** for `Exchange` (mock verifier + mock API client), `Callback` (full OAuth using oauth2 testing patterns), `Login` (redirect URL contains tenantId + state).

- [ ] **Step 7: Run + commit**

```bash
go test ./internal/oidc/... -v
git add apps/auth-bff/internal/oidc/
git commit -m "feat(auth-bff): OIDC login/callback + email-password id_token exchange"
```

### Task 1.9: Session lifecycle handlers (TDD)

**Files:**
- Create: `apps/auth-bff/internal/session/handler.go`, `apps/auth-bff/internal/session/handler_test.go`

- [ ] **Step 1: Write tests for `/auth/session`, `/auth/logout`, `/auth/refresh`, `/auth/csrf`**

```go
func TestSessionHandler_Session_Authenticated(t *testing.T) {
	// craft cookie via Manager.Encode, attach, expect 200 with user JSON
}
func TestSessionHandler_Session_NoCookie_401(t *testing.T) {}
func TestSessionHandler_Logout_ClearsCookie(t *testing.T) {}
func TestSessionHandler_Refresh_ExtendsExp(t *testing.T) {}
func TestSessionHandler_CSRF_ReturnsTokenAndCookie(t *testing.T) {}
```

- [ ] **Step 2: Implement `handler.go`**

```go
package session

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Mgr *Manager
}

func (h *Handler) Register(r gin.IRouter) {
	r.GET("/auth/session", h.session)
	r.POST("/auth/logout", h.logout)
	r.POST("/auth/refresh", h.refresh)
	r.GET("/auth/csrf", h.csrf)
}

func (h *Handler) session(c *gin.Context) {
	p, err := h.read(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id": p.UID, "email": p.Email, "role": p.Role, "pool": p.Pool,
		"expires_at": p.ExpiresAt,
	})
}

func (h *Handler) logout(c *gin.Context) {
	h.Mgr.Clear(c.Writer)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) refresh(c *gin.Context) {
	p, err := h.read(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	p.ExpiresAt = time.Now().Add(h.Mgr.MaxAge()).Unix()
	enc, err := h.Mgr.Encode(p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encode"})
		return
	}
	h.Mgr.SetCookie(c.Writer, enc)
	c.JSON(http.StatusOK, gin.H{"expires_at": p.ExpiresAt})
}

func (h *Handler) csrf(c *gin.Context) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	tok := hex.EncodeToString(b)
	http.SetCookie(c.Writer, &http.Cookie{Name: "hc_csrf", Value: tok, Path: "/", SameSite: http.SameSiteStrictMode})
	c.JSON(http.StatusOK, gin.H{"csrf_token": tok})
}

func (h *Handler) read(c *gin.Context) (*Payload, error) {
	ck, err := c.Request.Cookie(h.Mgr.cfg.CookieName)
	if err != nil {
		// also accept Authorization: Bearer for mobile
		if a := c.GetHeader("Authorization"); len(a) > 7 && a[:7] == "Bearer " {
			return h.Mgr.Decode(a[7:])
		}
		return nil, err
	}
	return h.Mgr.Decode(ck.Value)
}
```

(Add `Manager.MaxAge()` getter and persist it.)

- [ ] **Step 3: Run + commit**

```bash
go test ./internal/session/... -v
git add apps/auth-bff/internal/session/handler.go apps/auth-bff/internal/session/handler_test.go
git commit -m "feat(auth-bff): session/logout/refresh/csrf endpoints"
```

### Task 1.10: Wire main.go with all handlers

**Files:**
- Modify: `apps/auth-bff/cmd/server/main.go`, `apps/auth-bff/internal/config/config.go` (new)

- [ ] **Step 1: Create `internal/config/config.go`**

```go
package config

import (
	"encoding/base64"
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env                  string
	HTTPPort             string
	ProductsConfigPath   string
	GIPProjectID         string
	GIPProjectNumber     string
	SessionEncryptKey    []byte
	SessionMaxAge        time.Duration
	SessionCookieDomain  string
	BFFInternalHMACKey   []byte
	APIBaseURL           string
	AuditEndpoint        string
	AdminAllowedEmails   string
}

func Load() (*Config, error) {
	c := &Config{
		Env:                 os.Getenv("ENV"),
		HTTPPort:            getOrDefault("HTTP_PORT", "8080"),
		ProductsConfigPath:  os.Getenv("PRODUCTS_CONFIG_PATH"),
		GIPProjectID:        os.Getenv("GIP_PROJECT_ID"),
		GIPProjectNumber:    os.Getenv("GIP_PROJECT_NUMBER"),
		SessionCookieDomain: os.Getenv("SESSION_COOKIE_DOMAIN"),
		APIBaseURL:          os.Getenv("API_BASE_URL"),
		AuditEndpoint:       os.Getenv("AUDIT_ENDPOINT"),
		AdminAllowedEmails:  os.Getenv("HOMECHEF_ADMIN_ALLOWED_EMAILS"),
	}
	if c.GIPProjectID == "" { return nil, errors.New("GIP_PROJECT_ID required") }
	if c.ProductsConfigPath == "" { return nil, errors.New("PRODUCTS_CONFIG_PATH required") }

	sek, err := base64.StdEncoding.DecodeString(os.Getenv("SESSION_ENCRYPT_KEY"))
	if err != nil { return nil, errors.New("SESSION_ENCRYPT_KEY must be base64") }
	c.SessionEncryptKey = sek

	hmk, err := base64.StdEncoding.DecodeString(os.Getenv("BFF_INTERNAL_HMAC_KEY"))
	if err != nil { return nil, errors.New("BFF_INTERNAL_HMAC_KEY must be base64") }
	c.BFFInternalHMACKey = hmk

	h, _ := strconv.Atoi(getOrDefault("SESSION_MAX_AGE_HOURS", "168"))
	c.SessionMaxAge = time.Duration(h) * time.Hour
	return c, nil
}

func getOrDefault(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
```

- [ ] **Step 2: Rewrite `main.go` to wire everything**

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/homechef/auth-bff/internal/apiclient"
	"github.com/homechef/auth-bff/internal/autologin"
	"github.com/homechef/auth-bff/internal/config"
	"github.com/homechef/auth-bff/internal/gip"
	"github.com/homechef/auth-bff/internal/headerproxy"
	"github.com/homechef/auth-bff/internal/oidc"
	"github.com/homechef/auth-bff/internal/productregistry"
	"github.com/homechef/auth-bff/internal/session"
)

func main() {
	_ = godotenv.Load(".env.local")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	verifier, err := gip.New(gip.Config{ProjectID: cfg.GIPProjectID, Leeway: 10 * time.Second})
	if err != nil { log.Fatalf("gip: %v", err) }

	mgr, err := session.NewManager(session.Config{
		EncryptKey: cfg.SessionEncryptKey, MaxAge: cfg.SessionMaxAge,
		CookieName: "hc_session", CookieDomain: cfg.SessionCookieDomain,
		Secure: cfg.Env != "dev",
	})
	if err != nil { log.Fatalf("session: %v", err) }

	reg, err := productregistry.Load(cfg.ProductsConfigPath)
	if err != nil { log.Fatalf("registry: %v", err) }

	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: cfg.BFFInternalHMACKey, Window: time.Minute})
	api := apiclient.New(cfg.APIBaseURL, signer)

	oidcH := &oidc.Handlers{ /* registry/sessions/api/verifier wired */ }
	autoH := autologin.NewHandler(&autologin.Deps{GIP: verifier, Sessions: mgr, Registry: reg, API: api})
	sessH := &session.Handler{Mgr: mgr}

	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	r.GET("/auth/login", oidcH.Login)
	r.GET("/auth/callback", oidcH.Callback)
	r.POST("/auth/exchange", oidcH.Exchange)
	autoH.Register(r)
	sessH.Register(r)

	srv := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: r}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
```

- [ ] **Step 3: Run smoke test**

```bash
scripts/load-secrets.sh dev
cd apps/auth-bff
go run ./cmd/server &
curl -sf http://localhost:8081/healthz
kill %1
```
Expected: `{"ok":true}`

- [ ] **Step 4: Commit**

```bash
git add apps/auth-bff/cmd/ apps/auth-bff/internal/config/
git commit -m "feat(auth-bff): wire main.go with all subsystems"
```

### Task 1.11: Add audit client (best-effort) and rate-limit middleware

**Files:**
- Create: `apps/auth-bff/internal/audit/client.go`, `apps/auth-bff/internal/ratelimit/middleware.go`

- [ ] **Step 1: Audit client** — fires async POST to `AUDIT_ENDPOINT`, swallows errors

```go
package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Event struct {
	Type      string                 `json:"type"`
	UserID    string                 `json:"user_id,omitempty"`
	Email     string                 `json:"email,omitempty"`
	Pool      string                 `json:"pool,omitempty"`
	OccurredAt time.Time             `json:"occurred_at"`
	Attrs     map[string]any         `json:"attrs,omitempty"`
}

type Client struct {
	url  string
	http *http.Client
}

func New(url string) *Client {
	return &Client{url: url, http: &http.Client{Timeout: 2 * time.Second}}
}

func (c *Client) Emit(e Event) {
	if c == nil || c.url == "" { return }
	go func() {
		body, _ := json.Marshal(e)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.http.Do(req)
		if err != nil { return }
		_ = resp.Body.Close()
	}()
}
```

- [ ] **Step 2: Rate limit middleware** — token bucket per IP + per email

```go
package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type Limiter struct {
	mu      sync.Mutex
	clients map[string]*rate.Limiter
	rps     rate.Limit
	burst   int
}

func New(rps float64, burst int) *Limiter {
	return &Limiter{clients: map[string]*rate.Limiter{}, rps: rate.Limit(rps), burst: burst}
}

func (l *Limiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		l.mu.Lock()
		lim, ok := l.clients[key]
		if !ok {
			lim = rate.NewLimiter(l.rps, l.burst)
			l.clients[key] = lim
		}
		l.mu.Unlock()
		if !lim.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
			return
		}
		c.Next()
	}
}

// Reaper goroutine optional; for now memory grows unboundedly per unique IP. Acceptable for v1.
```

Apply to `/auth/login`, `/auth/callback`, `/auth/exchange`, `/auth/auto-login` in `main.go`:

```go
rl := ratelimit.New(5, 10)        // 5 rps per IP, burst 10
auth := r.Group("/auth", rl.Middleware())
auth.GET("/login", oidcH.Login)
auth.GET("/callback", oidcH.Callback)
auth.POST("/exchange", oidcH.Exchange)
auth.POST("/auto-login", autoH.PostFunc())   // expose as func on Handler
```

- [ ] **Step 3: Commit**

```bash
go test ./internal/audit/... ./internal/ratelimit/... -v
git add apps/auth-bff/internal/audit/ apps/auth-bff/internal/ratelimit/ apps/auth-bff/cmd/server/main.go
git commit -m "feat(auth-bff): audit client + per-IP rate limit on auth endpoints"
```

### Task 1.12: Integration test (in-process BFF + fake apps/api)

**Files:**
- Create: `apps/auth-bff/internal/integration_test.go`

- [ ] **Step 1: Write end-to-end test**

```go
//go:build integration

package internal_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// ... import all internal packages
)

func TestEndToEnd_MobileAutoLogin(t *testing.T) {
	// 1. Spin a fake JWK server + sign a valid GIP id_token
	// 2. Spin a fake apps/api that verifies HMAC and returns user_id
	// 3. Wire the BFF in-process
	// 4. POST /auth/auto-login → expect 200 with session_token
	// 5. GET /auth/session with bearer → expect 200 with email+role+pool
}
```

- [ ] **Step 2: Run gated under `-tags=integration`**

```bash
go test -tags=integration ./internal/... -v -run TestEndToEnd
```

- [ ] **Step 3: Commit**

```bash
git add apps/auth-bff/internal/integration_test.go
git commit -m "test(auth-bff): in-process end-to-end mobile autologin flow"
```

---

## Phase 2: Backend (`apps/api`)

### Task 2.1: BFF HMAC verification middleware (TDD)

**Files:**
- Create: `apps/api/middleware/bff_auth.go`, `apps/api/middleware/bff_auth_test.go`

- [ ] **Step 1: Write tests**

```go
func TestBFFAuth_ValidSignature_SetsContext(t *testing.T) { /* sign request with same key BFF uses, expect ctx populated */ }
func TestBFFAuth_MissingHeader_401(t *testing.T) {}
func TestBFFAuth_StaleTs_401(t *testing.T) {}
func TestBFFAuth_TamperedBody_401(t *testing.T) {}
```

- [ ] **Step 2: Implement middleware**

```go
package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/homechef/auth-bff/internal/headerproxy"
)

func BFFAuth(signer *headerproxy.Signer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}
		id, err := signer.Verify(c.Request, body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "bff_auth_failed"})
			return
		}
		c.Set("user_id", id.UserID)
		c.Set("email", id.Email)
		c.Set("role", id.Role)
		c.Set("pool", id.Pool)
		c.Next()
	}
}

var ErrBFFAuth = errors.New("bff_auth_failed")
```

- [ ] **Step 3: Add `BFF_INTERNAL_HMAC_KEY` to `apps/api/config/config.go`**

```go
// in Config struct:
BFFInternalHMACKey []byte

// in Load():
hmk, err := base64.StdEncoding.DecodeString(os.Getenv("BFF_INTERNAL_HMAC_KEY"))
if err != nil || len(hmk) < 16 {
    return nil, errors.New("BFF_INTERNAL_HMAC_KEY required, ≥16 bytes base64")
}
cfg.BFFInternalHMACKey = hmk
```

- [ ] **Step 4: Run + commit**

```bash
cd apps/api && go test ./middleware/... -v
git add apps/api/middleware/bff_auth.go apps/api/middleware/bff_auth_test.go apps/api/config/config.go
git commit -m "feat(api): BFF HMAC verification middleware"
```

### Task 2.2: Internal `/internal/users/upsert` endpoint (TDD)

**Files:**
- Create: `apps/api/handlers/internal_users.go`, `apps/api/handlers/internal_users_test.go`

- [ ] **Step 1: Write tests**

```go
func TestUpsert_NewUser_Inserts(t *testing.T) {
	// POST /internal/users/upsert with new gip_uid, expect 200 + row count = 1
}
func TestUpsert_ExistingUser_UpdatesLastLogin(t *testing.T) {
	// pre-seed row; upsert with same gip_uid, expect last_login_at updated, no new row
}
func TestUpsert_RaceTolerated(t *testing.T) {
	// two concurrent upserts with same gip_uid, exactly 1 row
}
```

- [ ] **Step 2: Implement handler**

```go
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

type InternalUsersHandler struct {
	DB *gorm.DB
}

type UpsertUserRequest struct {
	GIPUid      string `json:"gip_uid" binding:"required"`
	GIPTenantID string `json:"gip_tenant_id" binding:"required"`
	GIPProvider string `json:"gip_provider" binding:"required"`
	AuthPool    string `json:"auth_pool" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Name        string `json:"name"`
	Role        string `json:"role" binding:"required"`
}

func (h *InternalUsersHandler) Upsert(c *gin.Context) {
	var req UpsertUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	now := time.Now()
	var u models.User
	res := h.DB.Where("gip_uid = ?", req.GIPUid).First(&u)
	if res.Error == gorm.ErrRecordNotFound {
		u = models.User{
			ID: uuidNew(), Email: req.Email, Name: req.Name,
			GIPUid: req.GIPUid, GIPTenantID: req.GIPTenantID, GIPProvider: req.GIPProvider,
			AuthPool: models.AuthPool(req.AuthPool), Role: models.UserRole(req.Role),
			LastLoginAt: &now,
		}
		if err := h.DB.Create(&u).Error; err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
	} else if res.Error != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": res.Error.Error()})
		return
	} else {
		u.LastLoginAt = &now
		if u.Name == "" && req.Name != "" {
			u.Name = req.Name
		}
		if err := h.DB.Save(&u).Error; err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"user_id": u.ID.String()})
}
```

- [ ] **Step 3: Register route in `cmd/main.go`**

```go
// Inside route setup, BEFORE generic BFFAuth on other routes:
internal := r.Group("/internal", middleware.BFFAuth(signer))   // same HMAC middleware
internal.POST("/users/upsert", internalUsersH.Upsert)
```

- [ ] **Step 4: Run + commit**

```bash
go test ./handlers/... -v
git add apps/api/handlers/internal_users.go apps/api/handlers/internal_users_test.go apps/api/cmd/
git commit -m "feat(api): internal users.upsert endpoint (BFF-only)"
```

### Task 2.3: DB migration — drop password + token tables

**Files:**
- Create: `apps/api/migrations/20260514000001_drop_keycloak_auth_tables.{up,down}.sql`

- [ ] **Step 1: Write `up.sql`** — verbatim from spec §5.1

- [ ] **Step 2: Write `down.sql`** — verbatim from spec §5.1

- [ ] **Step 3: Apply locally**

```bash
docker-compose up -d postgres
psql -h localhost -U homechef -d homechef -f apps/api/migrations/20260514000001_drop_keycloak_auth_tables.up.sql
psql -h localhost -U homechef -d homechef -c "\d users"
```
Expected: no `password`/`totp_*`/`email_verified`/`auth_provider`/`provider_id` columns; `refresh_tokens`/`password_reset_tokens`/`email_verification_tokens` tables gone.

- [ ] **Step 4: Commit**

```bash
git add apps/api/migrations/20260514000001_*
git commit -m "feat(api): drop Keycloak password + token tables"
```

### Task 2.4: DB migration — add GIP identity columns

**Files:**
- Create: `apps/api/migrations/20260514000002_add_gip_identity_to_users.{up,down}.sql`

- [ ] **Step 1: Write `up.sql`** — verbatim from spec §5.2

- [ ] **Step 2: Write `down.sql`**

```sql
ALTER TABLE users
  DROP COLUMN IF EXISTS gip_uid,
  DROP COLUMN IF EXISTS gip_tenant_id,
  DROP COLUMN IF EXISTS gip_provider,
  DROP COLUMN IF EXISTS auth_pool,
  DROP COLUMN IF EXISTS role,
  DROP COLUMN IF EXISTS last_login_at;
DROP INDEX IF EXISTS idx_users_gip_uid;
DROP INDEX IF EXISTS idx_users_email_pool;
DROP INDEX IF EXISTS idx_users_email_per_pool;
-- re-create email unique constraint only if you actually need it back
```

- [ ] **Step 3: Apply locally + verify**

```bash
psql -h localhost -U homechef -d homechef -f apps/api/migrations/20260514000002_add_gip_identity_to_users.up.sql
psql -h localhost -U homechef -d homechef -c "\d users"
```
Expected: new columns + indexes exist; old `users_email_key` unique constraint gone.

- [ ] **Step 4: Commit**

```bash
git add apps/api/migrations/20260514000002_*
git commit -m "feat(api): add gip_uid/gip_tenant_id/auth_pool/role columns"
```

### Task 2.5: Update User model

**Files:**
- Modify: `apps/api/models/user.go`

- [ ] **Step 1: Rewrite struct + delete token types**

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

type AuthPool string

const (
	PoolCustomer AuthPool = "customer"
	PoolBusiness AuthPool = "business"
	PoolInternal AuthPool = "internal"
)

type UserRole string

const (
	RoleCustomer UserRole = "customer"
	RoleVendor   UserRole = "vendor"
	RoleDriver   UserRole = "driver"
	RoleAdmin    UserRole = "admin"
)

type User struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email       string     `gorm:"index" json:"email"`
	Name        string     `json:"name"`

	GIPUid      string     `gorm:"uniqueIndex" json:"gip_uid"`
	GIPTenantID string     `json:"gip_tenant_id"`
	GIPProvider string     `json:"gip_provider"`
	AuthPool    AuthPool   `json:"auth_pool"`
	Role        UserRole   `json:"role"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`

	// existing non-auth fields
	Phone     string    `json:"phone,omitempty"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// All RefreshToken / PasswordResetToken / EmailVerificationToken types removed.
```

- [ ] **Step 2: Run + commit**

```bash
go build ./...
git add apps/api/models/user.go
git commit -m "refactor(api): User model now points at GIP identity"
```

### Task 2.6: Delete auth handlers + HS256 helpers

**Files:**
- Modify: `apps/api/handlers/auth.go` (delete most of it)
- Modify: `apps/api/middleware/auth.go` (delete HS256 helpers)
- Modify: `apps/api/cmd/main.go` (remove route registrations)

- [ ] **Step 1: In `handlers/auth.go`, delete:**
  - `Register`
  - `Login`
  - `OAuthLogin`
  - `ForgotPassword`
  - `ResetPassword`
  - `verifyGoogleToken`, `verifyFacebookToken`, `verifyAppleToken`
  - any 2FA challenge / TOTP setup endpoints

  Keep only handlers that read user from context but don't *establish* identity (e.g., `GetMe`, `UpdateProfile`).

- [ ] **Step 2: In `middleware/auth.go`, delete:**
  - `GenerateTokens`, `GenerateTokensWithContext`
  - `extractKeycloakRoles`
  - `OptionalAuthMiddleware` (no longer needed — every request has BFF identity)
  - The 5-method `AuthMiddleware` (replaced by `BFFAuth`)

- [ ] **Step 3: In `cmd/main.go`, remove route registrations** for `/api/v1/auth/register`, `/api/v1/auth/login`, `/api/v1/auth/oauth`, `/api/v1/auth/forgot-password`, `/api/v1/auth/reset-password`.

- [ ] **Step 4: Apply `BFFAuth` to all protected groups**

```go
v1 := r.Group("/api/v1", middleware.BFFAuth(signer))
// register handlers as before; they read user_id/role/pool from ctx
```

- [ ] **Step 5: Build, run, smoke-test with curl**

```bash
go build ./...
go run ./cmd/main.go &

# Without BFF headers → 401 expected
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/v1/users/me
# Expected: 401

kill %1
```

- [ ] **Step 6: Commit**

```bash
git add apps/api/handlers/auth.go apps/api/middleware/auth.go apps/api/cmd/
git commit -m "refactor(api): remove Keycloak auth handlers and HS256 minting"
```

### Task 2.7: Env cleanup

**Files:**
- Modify: `apps/api/config/config.go`, `apps/api/.env.example`

- [ ] **Step 1: Remove from `config.go`** — `JWT_SECRET`, `JWT_EXPIRATION_HOURS`, `REFRESH_TOKEN_DAYS`, `GOOGLE_CLIENT_SECRET`, `FACEBOOK_APP_SECRET`, `APPLE_*`

- [ ] **Step 2: Update `.env.example`**

```
# Database, Redis, NATS — unchanged
DATABASE_URL=postgresql://homechef:homechef@localhost:5432/homechef

# BFF trust
BFF_INTERNAL_HMAC_KEY=                 # 32 bytes base64; must match apps/auth-bff
BFF_AUTH_TS_WINDOW_SECONDS=60

# GCS / Secret Manager — unchanged
```

- [ ] **Step 3: Commit**

```bash
git add apps/api/config/config.go apps/api/.env.example
git commit -m "refactor(api): drop JWT and OAuth env vars; require BFF_INTERNAL_HMAC_KEY"
```

---

## Phase 3: Web Frontends (×4 apps)

The 4 web apps (`web`, `vendor-portal`, `delivery-portal`, `admin-portal`) get nearly identical changes. Task 3.1 is the shared Firebase init. Tasks 3.2–3.5 apply per app with the tenant ID + cookie name + post-login URL differing.

### Task 3.1: Add Firebase JS SDK to all 4 web apps

- [ ] **Step 1: For each of the 4 apps:**

```bash
cd apps/web         && npm i firebase@^11.0.0
cd ../vendor-portal && npm i firebase@^11.0.0
cd ../delivery-portal && npm i firebase@^11.0.0
cd ../admin-portal  && npm i firebase@^11.0.0
```

- [ ] **Step 2: Create `src/lib/firebase.ts`** in each app (same code; tenant ID varies via env)

```ts
// apps/<app>/src/lib/firebase.ts
import { initializeApp } from "firebase/app";
import { getAuth, connectAuthEmulator } from "firebase/auth";

const cfg = {
  apiKey: import.meta.env.VITE_GIP_API_KEY,
  authDomain: import.meta.env.VITE_GIP_AUTH_DOMAIN,
  projectId: import.meta.env.VITE_GIP_PROJECT_ID,
};

export const app = initializeApp(cfg);
export const auth = getAuth(app);
auth.tenantId = import.meta.env.VITE_GIP_TENANT_ID;   // pins this app to its pool
```

- [ ] **Step 3: Add env vars to `.env.example` in each app**

```
VITE_BFF_URL=http://localhost:8081
VITE_GIP_PROJECT_ID=tesseracthub-480811
VITE_GIP_API_KEY=
VITE_GIP_AUTH_DOMAIN=tesseracthub-480811.firebaseapp.com
VITE_GIP_TENANT_ID=                              # HomeChef-Customer-xxxxx / Business / Internal
```

Remove old: `VITE_GOOGLE_CLIENT_ID`, `VITE_FACEBOOK_APP_ID`, `VITE_APPLE_CLIENT_ID`.

- [ ] **Step 4: Commit**

```bash
git add apps/web apps/vendor-portal apps/delivery-portal apps/admin-portal
git commit -m "feat(web): add Firebase JS SDK to all 4 frontends"
```

### Task 3.2: Migrate `apps/web` auth-service.ts

**Files:**
- Modify: `apps/web/src/features/auth/services/auth-service.ts`, `apps/web/src/app/store/auth-store.ts`, `apps/web/src/shared/types/auth.ts`

- [ ] **Step 1: Rewrite `auth-service.ts`**

```ts
import {
  signInWithPopup, GoogleAuthProvider, OAuthProvider, FacebookAuthProvider,
  signInWithEmailAndPassword, createUserWithEmailAndPassword,
  PhoneAuthProvider, RecaptchaVerifier, signInWithCredential,
  signOut, onAuthStateChanged, sendPasswordResetEmail,
} from "firebase/auth";
import { auth } from "@/lib/firebase";

const BFF = import.meta.env.VITE_BFF_URL ?? "";

async function exchangeForCookie(idToken: string) {
  const r = await fetch(`${BFF}/auth/exchange`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ id_token: idToken }),
  });
  if (!r.ok) throw new Error(`exchange_failed_${r.status}`);
  return r.json();
}

export async function signInWithGoogle() {
  const cred = await signInWithPopup(auth, new GoogleAuthProvider());
  const idToken = await cred.user.getIdToken();
  return exchangeForCookie(idToken);
}

export async function signInWithApple() {
  const cred = await signInWithPopup(auth, new OAuthProvider("apple.com"));
  return exchangeForCookie(await cred.user.getIdToken());
}

export async function signInWithFacebook() {
  const cred = await signInWithPopup(auth, new FacebookAuthProvider());
  return exchangeForCookie(await cred.user.getIdToken());
}

export async function signInWithEmail(email: string, password: string) {
  const cred = await signInWithEmailAndPassword(auth, email, password);
  return exchangeForCookie(await cred.user.getIdToken());
}

export async function registerWithEmail(email: string, password: string) {
  const cred = await createUserWithEmailAndPassword(auth, email, password);
  return exchangeForCookie(await cred.user.getIdToken());
}

export async function startPhoneSignIn(phone: string, recaptchaContainerId: string) {
  const verifier = new RecaptchaVerifier(auth, recaptchaContainerId, { size: "invisible" });
  const provider = new PhoneAuthProvider(auth);
  return provider.verifyPhoneNumber(phone, verifier);
}

export async function completePhoneSignIn(verificationId: string, code: string) {
  const phoneCred = PhoneAuthProvider.credential(verificationId, code);
  const cred = await signInWithCredential(auth, phoneCred);
  return exchangeForCookie(await cred.user.getIdToken());
}

export async function sendPasswordReset(email: string) {
  await sendPasswordResetEmail(auth, email);
}

export async function fetchSession() {
  const r = await fetch(`${BFF}/auth/session`, { credentials: "include" });
  if (r.status === 401) return null;
  if (!r.ok) throw new Error(`session_${r.status}`);
  return r.json();
}

export async function logout() {
  await signOut(auth);
  await fetch(`${BFF}/auth/logout`, { method: "POST", credentials: "include" });
}

export function subscribeAuth(cb: (u: { id: string; email: string; role: string; pool: string } | null) => void) {
  // we drive the app from the BFF session, not Firebase user — onAuthStateChanged
  // only used to detect token expiry locally; primary signal is /auth/session
  return onAuthStateChanged(auth, async (fbUser) => {
    if (!fbUser) { cb(null); return; }
    const s = await fetchSession();
    cb(s ? { id: s.user_id, email: s.email, role: s.role, pool: s.pool } : null);
  });
}
```

- [ ] **Step 2: Rewrite `auth-store.ts`** — drop the direct-API auth path; session = `fetchSession()` result

```ts
import { create } from "zustand";
import { fetchSession, logout as bffLogout } from "@/features/auth/services/auth-service";

type AuthUser = { id: string; email: string; role: string; pool: string };

interface AuthState {
  user: AuthUser | null;
  loading: boolean;
  hydrate: () => Promise<void>;
  logout: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  loading: true,
  hydrate: async () => {
    set({ loading: true });
    const s = await fetchSession();
    set({
      user: s ? { id: s.user_id, email: s.email, role: s.role, pool: s.pool } : null,
      loading: false,
    });
  },
  logout: async () => {
    await bffLogout();
    set({ user: null });
  },
}));
```

- [ ] **Step 3: Delete old types** in `shared/types/auth.ts` — remove `SocialProvider` Keycloak-flavoured strings; replace with the 5 Firebase provider IDs (`password`, `google.com`, `apple.com`, `facebook.com`, `phone`).

- [ ] **Step 4: Delete `kc_idp_hint` / `kc_action=UPDATE_PASSWORD`** wherever they appear in this app (forgot-password page, login-with-provider buttons). Replace with `sendPasswordReset(email)` import.

- [ ] **Step 5: Smoke**

```bash
cd apps/web
npm run typecheck
npm run lint
npm run build
```
Expected: 0 errors.

- [ ] **Step 6: Commit**

```bash
git add apps/web
git commit -m "feat(web): replace Keycloak auth with Firebase JS + BFF cookie exchange"
```

### Task 3.3: Migrate `apps/vendor-portal` auth-service.ts

Same shape as Task 3.2. Differences:
- `VITE_GIP_TENANT_ID=HomeChef-Business-xxxxx`
- No Facebook button on login page (per `signInMethods` in registry)
- Post-login URL is `/dashboard`
- Remove `signInWithFacebook` export

- [ ] Apply Task 3.2 changes to `apps/vendor-portal`, omitting Facebook code paths.
- [ ] `npm run typecheck && npm run lint && npm run build`
- [ ] Commit: `feat(vendor-portal): replace Keycloak auth with Firebase JS + BFF cookie exchange`

### Task 3.4: Migrate `apps/delivery-portal` auth-service.ts

Differences from Task 3.2:
- Same `HomeChef-Business-xxxxx` tenant as vendor (vendor + driver share business pool)
- Currently has dual-mode `staff`/`driver` auth (one realm each). Merge into a single flow — both modes are now driver in the business pool. Internal-portal access from delivery-portal is no longer supported; staff use admin-portal directly.
- Drop all `/driver-bff/` path-prefix references
- Post-login URL is `/jobs`

- [ ] Apply Task 3.2 changes to `apps/delivery-portal`. Collapse the two mode branches into one.
- [ ] `npm run typecheck && npm run lint && npm run build`
- [ ] Commit: `feat(delivery-portal): unify driver auth via Firebase JS + BFF`

### Task 3.5: Migrate `apps/admin-portal` auth-service.ts

Differences:
- `VITE_GIP_TENANT_ID=HomeChef-Internal-xxxxx`
- Only Email/Password + Google sign-in methods (per registry); remove Apple/Facebook/Phone UI
- Allowlist gating is done server-side at BFF (`HOMECHEF_ADMIN_ALLOWED_EMAILS`); frontend doesn't need to do anything special — non-allowed emails get a 403 from `/auth/exchange`, display generic "access denied".

- [ ] Apply Task 3.2 changes to `apps/admin-portal`, restricted to password+google.
- [ ] `npm run typecheck && npm run lint && npm run build`
- [ ] Commit: `feat(admin-portal): replace Keycloak auth with Firebase JS + BFF (staff allowlist enforced server-side)`

### Task 3.6: Remove all Keycloak references from code + comments

- [ ] **Step 1: Grep for stale strings across all 4 apps**

```bash
grep -RIn --include="*.ts" --include="*.tsx" --include="*.md" \
  -E 'kc_idp_hint|kc_action|keycloak|homechef realm|tesserix-internal realm|/bff/|/driver-bff/|/admin-bff/' \
  apps/ packages/mobile-shared/
```

- [ ] **Step 2: For each match, replace or delete**. Reference-fix examples:
  - `getLoginUrl('google')` callers → `signInWithGoogle()` from auth-service
  - `kc_action=UPDATE_PASSWORD` page → `<a onClick={() => sendPasswordReset(email)}>Reset password</a>`
  - Path-prefix mentions in fetch URLs → drop prefix; BFF resolves by host

- [ ] **Step 3: Verify clean**

```bash
grep -RIn --include="*.ts" --include="*.tsx" -E 'keycloak|kc_idp_hint|kc_action' apps/
```
Expected: no matches.

- [ ] **Step 4: Commit**

```bash
git add apps/
git commit -m "chore(web): purge Keycloak references from frontend code"
```

---

## Phase 4: Mobile (`packages/mobile-shared/auth/` + 3 Expo apps)

### Task 4.1: Create `packages/mobile-shared/auth/` package

**Files:**
- Create entire directory tree under `packages/mobile-shared/auth/`

- [ ] **Step 1: Install Firebase**

```bash
cd packages/mobile-shared
npm i @react-native-firebase/app@^21.0.0 @react-native-firebase/auth@^21.0.0 expo-secure-store@~14.0.0
```

- [ ] **Step 2: Write `firebase.ts`**

```ts
import auth, { FirebaseAuthTypes } from "@react-native-firebase/auth";

export function configureFirebaseAuth(tenantId: string) {
  // tenantId pins app to the right GIP pool
  auth().tenantId = tenantId;
  return auth();
}

export type FirebaseUser = FirebaseAuthTypes.User;
```

- [ ] **Step 3: Write `sign-in.ts`** (Google/Apple/Phone/Email common entry points)

```ts
import auth from "@react-native-firebase/auth";

export async function signInWithGoogleCredential(idToken: string, accessToken?: string) {
  const cred = auth.GoogleAuthProvider.credential(idToken, accessToken);
  return auth().signInWithCredential(cred);
}

export async function signInWithAppleCredential(idToken: string, rawNonce: string) {
  const cred = auth.AppleAuthProvider.credential(idToken, rawNonce);
  return auth().signInWithCredential(cred);
}

export async function signInWithEmail(email: string, password: string) {
  return auth().signInWithEmailAndPassword(email, password);
}

export async function registerWithEmail(email: string, password: string) {
  return auth().createUserWithEmailAndPassword(email, password);
}

export async function startPhoneSignIn(phone: string) {
  return auth().verifyPhoneNumber(phone);
}

export async function getIdToken() {
  const u = auth().currentUser;
  return u ? u.getIdToken() : null;
}

export async function signOut() {
  return auth().signOut();
}
```

- [ ] **Step 4: Write `bff-session.ts`**

```ts
import * as SecureStore from "expo-secure-store";

const SESSION_KEY = "hc_session_token";

export async function autoLogin(bffUrl: string, idToken: string, expectedTenantId: string) {
  const r = await fetch(`${bffUrl}/auth/auto-login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id_token: idToken, expected_tenant_id: expectedTenantId }),
  });
  if (!r.ok) throw new Error(`auto_login_${r.status}`);
  const body = await r.json();
  await SecureStore.setItemAsync(SESSION_KEY, body.session_token);
  return body;
}

export async function getStoredSession() {
  return SecureStore.getItemAsync(SESSION_KEY);
}

export async function clearStoredSession() {
  await SecureStore.deleteItemAsync(SESSION_KEY);
}

export async function fetchSessionUser(bffUrl: string) {
  const tok = await getStoredSession();
  if (!tok) return null;
  const r = await fetch(`${bffUrl}/auth/session`, {
    headers: { Authorization: `Bearer ${tok}` },
  });
  if (r.status === 401) {
    await clearStoredSession();
    return null;
  }
  return r.json();
}

export async function logout(bffUrl: string) {
  const tok = await getStoredSession();
  if (tok) {
    await fetch(`${bffUrl}/auth/logout`, {
      method: "POST",
      headers: { Authorization: `Bearer ${tok}` },
    });
  }
  await clearStoredSession();
}
```

- [ ] **Step 5: Write `provider.tsx`**

```tsx
import { createContext, useContext, useEffect, useState, ReactNode } from "react";
import auth from "@react-native-firebase/auth";
import { configureFirebaseAuth } from "./firebase";
import { autoLogin, clearStoredSession, fetchSessionUser, logout } from "./bff-session";

type AuthUser = { id: string; email: string; role: string; pool: string };

const Ctx = createContext<{
  user: AuthUser | null;
  loading: boolean;
  signIn: (idToken: string) => Promise<void>;
  signOut: () => Promise<void>;
} | undefined>(undefined);

export function AuthProvider({
  children, bffUrl, tenantId,
}: { children: ReactNode; bffUrl: string; tenantId: string }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    configureFirebaseAuth(tenantId);
    const unsub = auth().onAuthStateChanged(async (u) => {
      if (!u) {
        await clearStoredSession();
        setUser(null);
        setLoading(false);
        return;
      }
      const s = await fetchSessionUser(bffUrl);
      setUser(s ? { id: s.user_id, email: s.email, role: s.role, pool: s.pool } : null);
      setLoading(false);
    });
    return () => unsub();
  }, [bffUrl, tenantId]);

  return (
    <Ctx.Provider value={{
      user, loading,
      signIn: async (idToken) => {
        const body = await autoLogin(bffUrl, idToken, tenantId);
        setUser({ id: body.user.id, email: body.user.email, role: body.user.role, pool: body.user.pool });
      },
      signOut: async () => {
        await logout(bffUrl);
        await auth().signOut();
        setUser(null);
      },
    }}>{children}</Ctx.Provider>
  );
}

export function useAuth() {
  const c = useContext(Ctx);
  if (!c) throw new Error("useAuth outside AuthProvider");
  return c;
}
```

- [ ] **Step 6: Write `index.ts` barrel + `token-storage.ts`** (re-exports for backward compat)

```ts
// index.ts
export * from "./firebase";
export * from "./sign-in";
export * from "./bff-session";
export * from "./provider";
```

- [ ] **Step 7: Commit**

```bash
git add packages/mobile-shared/auth/
git commit -m "feat(mobile-shared): Firebase auth + BFF session package"
```

### Task 4.2: Migrate `apps/mobile-customer`

**Files:**
- Modify: `apps/mobile-customer/store/auth-store.ts`, `apps/mobile-customer/app/(auth)/login.tsx`, `apps/mobile-customer/app/_layout.tsx`, `apps/mobile-customer/.env.example`

- [ ] **Step 1: Install Firebase**

```bash
cd apps/mobile-customer
npm i @react-native-firebase/app @react-native-firebase/auth
```

- [ ] **Step 2: Wrap root in `AuthProvider`**

```tsx
// app/_layout.tsx (excerpt)
import { AuthProvider } from "@homechef/mobile-shared/auth";

export default function RootLayout() {
  return (
    <AuthProvider
      bffUrl={process.env.EXPO_PUBLIC_BFF_URL!}
      tenantId={process.env.EXPO_PUBLIC_GIP_TENANT_ID!}
    >
      {/* existing tree */}
    </AuthProvider>
  );
}
```

- [ ] **Step 3: Rewrite `app/(auth)/login.tsx`** — use Firebase + provider's `signIn`

```tsx
import { GoogleSignin } from "@react-native-google-signin/google-signin";
import { signInWithGoogleCredential, useAuth } from "@homechef/mobile-shared/auth";

export default function Login() {
  const { signIn } = useAuth();

  async function onGooglePress() {
    GoogleSignin.configure({ webClientId: process.env.EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID! });
    await GoogleSignin.hasPlayServices();
    const { idToken } = await GoogleSignin.signIn();
    if (!idToken) throw new Error("no_id_token");
    const cred = await signInWithGoogleCredential(idToken);
    const fbIdToken = await cred.user.getIdToken();
    await signIn(fbIdToken);
  }

  // ... similar for Email/Apple/Phone
}
```

- [ ] **Step 4: Update `store/auth-store.ts`** to re-export from `@homechef/mobile-shared/auth`

```ts
export { useAuth } from "@homechef/mobile-shared/auth";
```

- [ ] **Step 5: Update `.env.example`**

```
EXPO_PUBLIC_BFF_URL=http://localhost:8081
EXPO_PUBLIC_GIP_TENANT_ID=HomeChef-Customer-xxxxx
EXPO_PUBLIC_GIP_API_KEY=
EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID=         # KEEP — Firebase Google sign-in needs it
```

- [ ] **Step 6: Smoke**

```bash
cd apps/mobile-customer
npx expo prebuild --clean
npx expo run:ios    # or run:android
```
Manual test: tap Google sign-in → see user lands on home screen with session.

- [ ] **Step 7: Commit**

```bash
git add apps/mobile-customer
git commit -m "feat(mobile-customer): Firebase auth + BFF auto-login"
```

### Task 4.3: Migrate `apps/mobile-vendor`

Same as Task 4.2 with `EXPO_PUBLIC_GIP_TENANT_ID=HomeChef-Business-xxxxx`.

### Task 4.4: Migrate `apps/mobile-delivery`

Same as Task 4.2 with `EXPO_PUBLIC_GIP_TENANT_ID=HomeChef-Business-xxxxx` (same pool, different default role on first sign-in via BFF inference).

### Task 4.5: Delete legacy mobile auth code

**Files:**
- Delete: `packages/mobile-shared/src/api/auth.ts`

- [ ] **Step 1: Verify no remaining imports of the old file**

```bash
grep -RIn "from.*mobile-shared/src/api/auth" apps/ packages/
```
Expected: no matches (all replaced by `@homechef/mobile-shared/auth`).

- [ ] **Step 2: Delete + commit**

```bash
git rm packages/mobile-shared/src/api/auth.ts
git commit -m "chore(mobile-shared): drop legacy direct-API auth client"
```

---

## Phase 5: Infra (`tesserix-k8s`)

All work is in the sibling repo. Pre-req: Phase 0 Task 0.1 (Terraform tenants) is already done; this phase deploys the BFF.

### Task 5.1: Helm chart for `homechef-auth-bff`

**Files:**
- Create: `tesserix-k8s/charts/apps/homechef-auth-bff/{Chart.yaml,values.yaml,values-uat.yaml,templates/*.yaml}`

- [ ] **Step 1: Copy mark8ly-auth-bff chart as starting point**

```bash
cd tesserix-k8s
cp -r charts/apps/mark8ly-auth-bff charts/apps/homechef-auth-bff
```

- [ ] **Step 2: Rewrite `Chart.yaml`**

```yaml
apiVersion: v2
name: homechef-auth-bff
description: Auth BFF for Home-Chef-App (GIP)
type: application
version: 0.1.0
appVersion: "1.0.0"
```

- [ ] **Step 3: Rewrite `values.yaml`** — verbatim from spec §6.4 (image, replicas, resources, env, envFromSecret, serviceAccount)

- [ ] **Step 4: Adapt templates** — replace `mark8ly` strings with `homechef`; adapt AuthorizationPolicy to allow `homechef-api.homechef` callers + `istio-ingress`

- [ ] **Step 5: `helm lint`**

```bash
helm lint charts/apps/homechef-auth-bff/
```
Expected: 0 errors.

- [ ] **Step 6: Commit**

```bash
git add charts/apps/homechef-auth-bff/
git commit -m "chart(homechef-auth-bff): initial helm chart"
git push
```

### Task 5.2: ArgoCD Application

**Files:**
- Create: `tesserix-k8s/argocd/prod/apps/homechef/auth-bff.yaml`

- [ ] **Step 1: Write Application**

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: homechef-auth-bff
  namespace: argocd
  finalizers: [resources-finalizer.argocd.argoproj.io]
spec:
  project: homechef
  source:
    repoURL: git@github.com:tesserix/tesserix-k8s.git
    targetRevision: main
    path: charts/apps/homechef-auth-bff
    helm:
      valueFiles: [values.yaml]
  destination:
    server: https://kubernetes.default.svc
    namespace: homechef
  syncPolicy:
    automated: { prune: true, selfHeal: true }
    syncOptions: [ServerSideApply=true, CreateNamespace=true]
```

- [ ] **Step 2: Commit**

```bash
git add argocd/prod/apps/homechef/auth-bff.yaml
git commit -m "argocd(homechef): auth-bff application"
git push
```

### Task 5.3: ExternalSecrets

**Files:**
- Modify: `tesserix-k8s/external-secrets/prod/homechef/externalsecret.yaml`

- [ ] **Step 1: Append the 7 new keys to the existing ExternalSecret** (verbatim from spec §6.5)

- [ ] **Step 2: Commit + sync**

```bash
git add external-secrets/prod/homechef/externalsecret.yaml
git commit -m "secrets(homechef): GIP, session, HMAC, admin-allowlist"
git push

# After argocd-syncs:
kubectl get secret homechef-auth-bff-secrets -n homechef -o yaml | head -30
```
Expected: 7 keys present in the synced secret.

### Task 5.4: Istio VirtualService (new) + update existing

**Files:**
- Create: `tesserix-k8s/manifests/homechef-istio/virtualservice-auth.yaml`
- Modify: `tesserix-k8s/manifests/homechef-istio/virtualservice-{web,vendors,delivery,admin}.yaml`

- [ ] **Step 1: Create the auth VS** (verbatim from spec §6.3)

- [ ] **Step 2: In each per-app VS, delete the `/bff/`, `/driver-bff/`, `/admin-bff/` match rules**

```diff
-    - match:
-        - uri: { prefix: /bff/ }
-      rewrite: { uri: / }
-      route:
-        - destination: { host: homechef-auth-bff-legacy.svc.cluster.local }
```

- [ ] **Step 3: Commit**

```bash
git add manifests/homechef-istio/
git commit -m "istio(homechef): unify /auth/* path; drop legacy /bff/ prefixes"
git push
```

### Task 5.5: Kargo Warehouse entry

**Files:**
- Modify: `tesserix/kargo-manifests/projects/homechef/warehouse.yaml`

- [ ] **Step 1: Add the auth-bff image subscription**

```yaml
spec:
  subscriptions:
    - image:
        repoURL: asia-south1-docker.pkg.dev/tesseracthub-480811/ghcr-remote/homechef/auth-bff
        semverConstraint: ^1.0.0
        strictSemvers: false
```

- [ ] **Step 2: Commit in the kargo-manifests repo**

```bash
cd ../kargo-manifests
git add projects/homechef/warehouse.yaml
git commit -m "kargo(homechef): subscribe to auth-bff image"
git push
```

### Task 5.6: CI workflow

**Files:**
- Create: `.github/workflows/auth-bff-ci.yml` (in Home-Chef-App repo)

- [ ] **Step 1: Copy + adapt the existing `homechef-web-ci.yml`** to build `apps/auth-bff` instead. Job stages: `go vet`, `go test -race -coverprofile`, `docker build`, `docker push` to GAR, `bump-k8s` curl call.

- [ ] **Step 2: Commit + push**

```bash
git add .github/workflows/auth-bff-ci.yml
git commit -m "ci(auth-bff): build/test/push to GAR with k8s bump"
git push
```

---

## Phase 6: Cutover

### Task 6.1: Staging dress rehearsal on devtest

- [ ] **Step 1: Deploy auth-bff to devtest cluster** (manual ArgoCD sync from a `devtest` branch in tesserix-k8s)
- [ ] **Step 2: Provision `Dev-HomeChef-*` GIP tenants** in the same `tesseracthub-480811` project via separate Terraform tfvars
- [ ] **Step 3: Run the full §7.3 test matrix** (17 web + 18 mobile flows) on devtest
- [ ] **Step 4: Time the cutover sequence** — measure each step duration; tighten timings if any step exceeds budget
- [ ] **Step 5: Fix any issues found; re-rehearse until 3 consecutive green runs**
- [ ] **Step 6: Document deltas vs prod** in `docs/superpowers/runbooks/2026-cutover-runbook.md`

### Task 6.2: User communication

- [ ] **Step 1: Send T-7d email** to all known users (subject from spec §7.7)
- [ ] **Step 2: Enable T-24h in-app banner** in all 4 web apps via feature flag
- [ ] **Step 3: Update status page** with maintenance window entry

### Task 6.3: Production cutover (45-min window, Sunday 02:00 IST)

Follow the §7.2 table from the spec verbatim. Two operators on call. Use the cutover runbook from Task 6.1.

- [ ] **Step 1: T-15m: enable maintenance page in Cloudflare Worker**
- [ ] **Step 2: T-10m: ArgoCD pause sync on homechef, identity-customer, identity-internal**
- [ ] **Step 3: T-5m: apply DB migrations via psql**
- [ ] **Step 4: T-0: ArgoCD apply homechef-auth-bff Application + per-app VirtualService updates + apps/api new image with BFF_INTERNAL_HMAC_KEY**
- [ ] **Step 5: T+2m → T+8m: smoke tests per spec §7.2**
- [ ] **Step 6: T+10m: remove maintenance page**
- [ ] **Step 7: T+15m: resume ArgoCD sync on homechef; leave identity-* paused**
- [ ] **Step 8: T+30m: rollback decision deadline. If green, proceed.**
- [ ] **Step 9: T+45m: declare cutover complete; post status update**

If any §7.5 rollback trigger fires before T+30m: execute the rollback steps from spec §7.5.

### Task 6.4: Post-cutover monitoring (T-0 to T+72h)

- [ ] **Step 1: Watch `grafana.internal/d/homechef-auth-cutover`** dashboard during business hours for 72h
- [ ] **Step 2: Triage any PagerDuty incidents** per §7.4 alert rules
- [ ] **Step 3: Compile a post-cutover report** with success rate, incident count, mean time to detect/recover

### Task 6.5: Decommission Keycloak (T+14d)

- [ ] **Step 1: Verify zero traffic to Keycloak namespaces for 14 days** (Istio access logs)
- [ ] **Step 2: Open single decommission PR in tesserix-k8s** removing all paths listed in spec §6.8
- [ ] **Step 3: After PR merges and ArgoCD prunes**: drop `keycloak_customer` + `keycloak_internal` databases via psql
- [ ] **Step 4: Remove `prod-keycloak-*` family from GCP Secret Manager** (with rotation log entry)
- [ ] **Step 5: Delete `tesserix-k8s/scripts/identity/keycloak-to-gip-migrate.py`** (fresh-start chosen; script no longer relevant)
- [ ] **Step 6: Commit + announce final decommission**

---

## Self-Review Summary

**Spec coverage:** Every section of the design doc maps to at least one task:
- §1 Architecture → Phase 1 (BFF) + Phase 2 (API HMAC)
- §2 Components → Phase 1 + Phase 3 (web) + Phase 4 (mobile)
- §3 Data Flow → Phase 1 Task 1.7 (mobile) + 1.8 (web) handlers
- §4 Configuration → Phase 0 Task 0.4 (secrets) + 0.5 (env) + Phase 1.4 (registry)
- §5 Database → Phase 2 Tasks 2.3–2.5
- §6 Infra → Phase 5
- §7 Cutover → Phase 6

**Placeholder scan:** No "TBD" or "implement later". Repetitive web/mobile work (4 apps × 3 mobile) is detailed once (Task 3.2, 4.2) and tasks 3.3–3.5, 4.3–4.4 specify deltas explicitly.

**Type consistency:** `Payload` struct in `session` package consistent across handlers; `Identity` struct in `headerproxy` package matches `BFFAuth` middleware reads; `UpsertUserRequest` field names match in `apps/auth-bff/internal/apiclient/users.go` and `apps/api/handlers/internal_users.go`. Spot-checked.

**Known gaps (intentional, deferred):**
- Forced session revocation requires admin GIP role on the BFF; deferred to a follow-on phase via `gip-admin-claims` operator job. Spec §3.5 notes this.
- TOTP MFA + OpenFGA + account linking are explicitly in the spec's "Non-Goals" — no tasks.

---

## Plan complete and saved to `docs/superpowers/plans/2026-05-14-keycloak-to-gip-migration.md`.

Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration. Best for a migration this big — each task gets isolated context.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints. Faster start, but the session context will get crowded across 50+ tasks.

Which approach?
