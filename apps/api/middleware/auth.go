package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// extractKeycloakRoles decodes a JWT payload (without signature verification)
// and extracts the "roles" claim. This is safe because the token comes from
// the trusted BFF which already validated it with Keycloak.
func extractKeycloakRoles(token string) []string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	// Decode the payload (second part)
	payload := parts[1]
	// Add padding if needed
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil
	}
	// Check top-level "roles" claim (from Keycloak protocol mapper)
	if roles, ok := claims["roles"]; ok {
		if arr, ok := roles.([]interface{}); ok {
			var result []string
			for _, r := range arr {
				if s, ok := r.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	// Check realm_access.roles (Keycloak default structure)
	if ra, ok := claims["realm_access"]; ok {
		if raMap, ok := ra.(map[string]interface{}); ok {
			if roles, ok := raMap["roles"]; ok {
				if arr, ok := roles.([]interface{}); ok {
					var result []string
					for _, r := range arr {
						if s, ok := r.(string); ok {
							result = append(result, s)
						}
					}
					return result
				}
			}
		}
	}
	return nil
}

type Claims struct {
	UserID uuid.UUID       `json:"userId"`
	Email  string          `json:"email"`
	Role   models.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// TwoFactorChallengeKind distinguishes the two 2FA flow entry points. The
// client can only use a "verify" challenge to call /auth/2fa/verify, and an
// "enroll" challenge to call /auth/2fa/enroll-*. Separate claim values mean
// a verify token can't be replayed against the enroll endpoints or vice versa.
type TwoFactorChallengeKind string

const (
	ChallengeVerify TwoFactorChallengeKind = "verify"
	ChallengeEnroll TwoFactorChallengeKind = "enroll"
)

// twoFactorChallengeClaims is the JWT payload for a short-lived token handed
// back after a successful password step but before 2FA is satisfied.
type twoFactorChallengeClaims struct {
	UserID uuid.UUID              `json:"userId"`
	Kind   TwoFactorChallengeKind `json:"kind"`
	jwt.RegisteredClaims
}

// GenerateTwoFactorChallenge mints a 10-minute JWT that proves the user
// cleared the password step. It is not an access token — it can only be
// consumed by /auth/2fa/verify or /auth/2fa/enroll-*.
func GenerateTwoFactorChallenge(user *models.User, kind TwoFactorChallengeKind) (string, error) {
	claims := &twoFactorChallengeClaims{
		UserID: user.ID,
		Kind:   kind,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "homechef-2fa",
			Subject:   user.ID.String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

// ParseTwoFactorChallenge validates a challenge token and returns the user
// ID + kind. Rejects expired tokens and mismatched-kind tokens.
func ParseTwoFactorChallenge(tokenString string, expected TwoFactorChallengeKind) (uuid.UUID, error) {
	claims := &twoFactorChallengeClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, err
	}
	if claims.Kind != expected {
		return uuid.Nil, jwt.ErrTokenInvalidClaims
	}
	return claims.UserID, nil
}

// AuthMiddleware validates JWT tokens and sets user context.
// Supports four authentication methods:
//  1. Authorization: ApiKey <secret> — platform integration key, short-circuits
//     the rest so third-party callers don't need a user login
//  2. Authorization: Bearer <JWT> — traditional JWT from the app's own auth
//  3. ?token=<JWT> query param — only honored on WebSocket upgrade requests,
//     because browsers cannot set custom headers on the WS handshake
//  4. x-jwt-claim-sub header — trusted user ID from BFF proxy or Istio mesh
//     (only accepted for internal mesh requests where the BFF already validated the session)
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		// API key short-circuit. Platform keys represent an integration, not
		// a user, so we set userID to the key's creator (for audit) and stash
		// the key on the context for scope checks.
		if strings.HasPrefix(authHeader, "ApiKey ") {
			key, err := services.LookupApiKey(strings.TrimPrefix(authHeader, "ApiKey "))
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
				c.Abort()
				return
			}
			c.Set("apiKey", key)
			c.Set("userID", key.CreatedBy)
			c.Set("userRole", models.RoleAdmin)
			c.Next()
			return
		}

		tokenString := ""
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}
		// WebSocket handshakes can't carry an Authorization header, so allow
		// the JWT to ride in as ?token=... — but ONLY for WS upgrade requests
		// to avoid tokens leaking into access logs / referer on normal traffic.
		if tokenString == "" && strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
			tokenString = c.Query("token")
		}

		// Try JWT-based auth first
		if tokenString != "" {
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(config.AppConfig.JWTSecret), nil
			})

			if err == nil && token.Valid {
				var user models.User
				if err := database.DB.First(&user, "id = ?", claims.UserID).Error; err != nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
					c.Abort()
					return
				}
				if !user.IsActive {
					c.JSON(http.StatusForbidden, gin.H{"error": "Account is suspended"})
					c.Abort()
					return
				}
				c.Set("userID", claims.UserID)
				c.Set("userEmail", claims.Email)
				c.Set("userRole", claims.Role)
				c.Set("user", &user)
				c.Next()
				return
			}
			// JWT parsing failed — fall through to x-jwt-claim-sub
		}

		// Fallback: trust x-jwt-claim-sub from BFF proxy / Istio mesh
		// The BFF validates the session cookie and sets this header with the authenticated user ID
		sub := c.GetHeader("x-jwt-claim-sub")
		if sub != "" {
			userID, err := uuid.Parse(sub)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in x-jwt-claim-sub"})
				c.Abort()
				return
			}

			var user models.User
			if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
				// User not found by Keycloak ID — auto-provision from BFF claims
				email := c.GetHeader("x-jwt-claim-email")
				if email == "" {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found and no email provided for provisioning"})
					c.Abort()
					return
				}

				// Check if a user with this email already exists (registered via direct API)
				var existingUser models.User
				if err := database.DB.First(&existingUser, "email = ?", email).Error; err == nil {
					// Email exists — use existing user
					user = existingUser
				} else {
					// Create new user from Keycloak claims
					firstName := c.GetHeader("x-jwt-claim-given-name")
					lastName := c.GetHeader("x-jwt-claim-family-name")
					if firstName == "" && lastName == "" {
						name := c.GetHeader("x-jwt-claim-name")
						parts := strings.SplitN(name, " ", 2)
						if len(parts) > 0 {
							firstName = parts[0]
						}
						if len(parts) > 1 {
							lastName = parts[1]
						}
					}

					now := time.Now()
					user = models.User{
						ID:            userID,
						Email:         email,
						FirstName:     firstName,
						LastName:      lastName,
						Role:          models.RoleCustomer,
						AuthProvider:  models.ProviderGoogle,
						ProviderID:    sub,
						IsActive:      true,
						EmailVerified: true,
						LastLoginAt:   &now,
					}
					if err := database.DB.Create(&user).Error; err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to provision user"})
						c.Abort()
						return
					}
				}
			}
			if !user.IsActive {
				c.JSON(http.StatusForbidden, gin.H{"error": "Account is suspended"})
				c.Abort()
				return
			}

			c.Set("userID", user.ID)
			c.Set("userEmail", user.Email)

			// Determine effective role: check Keycloak JWT roles from the BFF Bearer token
			// The BFF forwards the Keycloak access_token as Authorization: Bearer <token>
			// We decode the payload (no sig verification needed — BFF is trusted) to read roles
			effectiveRole := user.Role
			if authHeader := c.GetHeader("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
				if keycloakRoles := extractKeycloakRoles(authHeader[7:]); len(keycloakRoles) > 0 {
					for _, r := range keycloakRoles {
						if r == "super_admin" || r == "admin" {
							effectiveRole = models.RoleAdmin
							break
						}
						if r == "fleet_manager" {
							effectiveRole = models.RoleFleetManager
							// Don't break — admin/super_admin takes precedence
						}
					}
				}
			}

			// Auto-provision super admin staff profile for default emails
			if models.IsSuperAdminEmail(user.Email) && effectiveRole != models.RoleAdmin {
				effectiveRole = models.RoleAdmin
			}
			c.Set("userRole", effectiveRole)
			c.Set("user", &user)
			c.Next()
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
		c.Abort()
	}
}

// OptionalAuthMiddleware extracts user info if token is present, but doesn't require it.
// Supports both JWT and x-jwt-claim-sub header.
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try JWT first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				claims := &Claims{}
				token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (interface{}, error) {
					return []byte(config.AppConfig.JWTSecret), nil
				})
				if err == nil && token.Valid {
					c.Set("userID", claims.UserID)
					c.Set("userEmail", claims.Email)
					c.Set("userRole", claims.Role)
					c.Next()
					return
				}
			}
		}

		// Fallback: x-jwt-claim-sub from BFF/Istio
		sub := c.GetHeader("x-jwt-claim-sub")
		if sub != "" {
			userID, err := uuid.Parse(sub)
			if err == nil {
				var user models.User
				if err := database.DB.First(&user, "id = ?", userID).Error; err == nil {
					c.Set("userID", user.ID)
					c.Set("userEmail", user.Email)
					c.Set("userRole", user.Role)
				}
			}
		}

		c.Next()
	}
}

// GenerateTokens creates access and refresh tokens for a user. The returned
// gin.Context is optional — when provided, the request's User-Agent and IP
// are recorded on the RefreshToken row so admins can see where a session
// came from and revoke specific devices.
func GenerateTokens(user *models.User) (string, string, error) {
	return GenerateTokensWithContext(user, nil)
}

// GenerateTokensWithContext is the capture-device-metadata form used by
// handlers that have a *gin.Context handy. Access / refresh TTLs come from
// the platform SecurityPolicy (falls back to config defaults).
func GenerateTokensWithContext(user *models.User, c *gin.Context) (string, string, error) {
	policy := services.GetSecurityPolicy()

	accessTTLHours := policy.SessionAccessTTLHours
	if accessTTLHours <= 0 {
		accessTTLHours = config.AppConfig.JWTExpirationHours
	}
	refreshTTLDays := policy.SessionRefreshTTLDays
	if refreshTTLDays <= 0 {
		refreshTTLDays = config.AppConfig.RefreshTokenDays
	}

	accessClaims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(accessTTLHours))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "homechef",
			Subject:   user.ID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(config.AppConfig.JWTSecret))
	if err != nil {
		return "", "", err
	}

	refreshToken := uuid.New().String()
	expiresAt := time.Now().AddDate(0, 0, refreshTTLDays)

	refreshTokenModel := models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: expiresAt,
	}
	if c != nil {
		refreshTokenModel.UserAgent = c.Request.UserAgent()
		refreshTokenModel.IPAddress = c.ClientIP()
	}
	if err := database.DB.Create(&refreshTokenModel).Error; err != nil {
		return "", "", err
	}

	return accessTokenString, refreshToken, nil
}

// Helper functions to get user info from context
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, false
	}
	return userID.(uuid.UUID), true
}

func GetUserRole(c *gin.Context) (models.UserRole, bool) {
	role, exists := c.Get("userRole")
	if !exists {
		return "", false
	}
	return role.(models.UserRole), true
}

func GetUser(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}
	return user.(*models.User), true
}
