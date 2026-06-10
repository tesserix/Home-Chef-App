package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env                 string
	HTTPPort            string
	ProductsConfigPath  string
	GIPProjectID        string
	GIPProjectNumber    string
	SessionEncryptKey   []byte
	SessionMaxAge       time.Duration
	SessionCookieDomain string
	BFFInternalHMACKey  []byte
	APIBaseURL          string
	AuditEndpoint       string
	AdminAllowedEmails  string

	// Observability. TraceProjectID is the GCP project Cloud Trace spans are
	// written to — set GCP_PROJECT_ID to the GKE project (e.g. tesseracthub-480811)
	// so auth-bff spans land in the SAME trace view as homechef-api. Defaults to
	// the GIP project when unset. Empty disables tracing (local dev).
	TraceProjectID   string
	OTelSamplingRate float64
	AppVersion       string
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
	if c.GIPProjectID == "" {
		return nil, errors.New("GIP_PROJECT_ID required")
	}
	if c.ProductsConfigPath == "" {
		return nil, errors.New("PRODUCTS_CONFIG_PATH required")
	}
	if c.APIBaseURL == "" {
		return nil, errors.New("API_BASE_URL required")
	}

	sek := os.Getenv("SESSION_ENCRYPT_KEY")
	if sek == "" {
		return nil, errors.New("SESSION_ENCRYPT_KEY required (base64, 16/24/32 bytes)")
	}
	decoded, err := base64.StdEncoding.DecodeString(sek)
	if err != nil {
		return nil, fmt.Errorf("SESSION_ENCRYPT_KEY must be valid base64: %w", err)
	}
	c.SessionEncryptKey = decoded

	hk := os.Getenv("BFF_INTERNAL_HMAC_KEY")
	if hk == "" {
		return nil, errors.New("BFF_INTERNAL_HMAC_KEY required (base64, 16+ bytes)")
	}
	decoded, err = base64.StdEncoding.DecodeString(hk)
	if err != nil {
		return nil, fmt.Errorf("BFF_INTERNAL_HMAC_KEY must be valid base64: %w", err)
	}
	if len(decoded) < 16 {
		return nil, errors.New("BFF_INTERNAL_HMAC_KEY must decode to at least 16 bytes")
	}
	c.BFFInternalHMACKey = decoded

	h, err := strconv.Atoi(getOrDefault("SESSION_MAX_AGE_HOURS", "168"))
	if err != nil {
		return nil, fmt.Errorf("SESSION_MAX_AGE_HOURS must be integer: %w", err)
	}
	c.SessionMaxAge = time.Duration(h) * time.Hour

	// Cloud Trace target — GCP_PROJECT_ID wins, else fall back to the GIP project.
	c.TraceProjectID = getOrDefault("GCP_PROJECT_ID", c.GIPProjectID)
	c.AppVersion = os.Getenv("APP_VERSION")
	c.OTelSamplingRate = 0.1
	if v := os.Getenv("OTEL_SAMPLING_RATE"); v != "" {
		if f, perr := strconv.ParseFloat(v, 64); perr == nil {
			c.OTelSamplingRate = f
		}
	}

	return c, nil
}

func getOrDefault(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
