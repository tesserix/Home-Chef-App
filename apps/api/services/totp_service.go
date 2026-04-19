package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"

	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/pquerna/otp/totp"
)

// encodePNG encodes an image into PNG bytes. Separate helper so the handler
// doesn't deal with image codecs directly.
func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// TOTP secrets are stored in GCP Secret Manager, one secret per user, keyed
// by user ID. The DB only holds a boolean flag (TOTPEnabled) — we never keep
// the shared secret in PostgreSQL.
const totpIssuer = "Fe3dr Admin"

func totpSecretName(userID uuid.UUID) string {
	env := "prod"
	if config.IsDevelopment() {
		env = "dev"
	}
	// Product-scoped so it can't collide with another app storing TOTP
	// secrets in the same GCP project.
	return fmt.Sprintf("%s-%s-totp-user-%s", env, ProductPrefix, userID.String())
}

// TOTPEnrollment is what EnrollTOTP returns so the admin can finish setup.
type TOTPEnrollment struct {
	Secret      string // Base32 — user can type it into their app manually
	OtpAuthURL  string // otpauth:// URI the frontend renders as a QR
	QRCodePNG   []byte // rendered PNG, ~200x200, base64 it on the wire if needed
}

// EnrollTOTP generates a fresh secret for the user, writes it to GCP Secret
// Manager, and returns the otpauth URL so the frontend can render the QR.
// The user is NOT marked as TOTPEnabled yet — that happens after they prove
// they have the secret by calling VerifyTOTPEnrollment.
func EnrollTOTP(ctx context.Context, userID uuid.UUID, accountLabel string) (*TOTPEnrollment, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: accountLabel,
	})
	if err != nil {
		return nil, fmt.Errorf("generate totp: %w", err)
	}

	// Persist the secret in GCP Secret Manager — this is also what we read
	// at verification time, so the DB never sees it.
	if err := StorePlatformSecret(ctx, totpSecretName(userID), key.Secret()); err != nil {
		return nil, fmt.Errorf("store totp secret: %w", err)
	}

	img, err := key.Image(200, 200)
	if err != nil {
		return nil, fmt.Errorf("render totp qr: %w", err)
	}
	png, err := encodePNG(img)
	if err != nil {
		return nil, fmt.Errorf("encode totp qr: %w", err)
	}

	return &TOTPEnrollment{
		Secret:     key.Secret(),
		OtpAuthURL: key.URL(),
		QRCodePNG:  png,
	}, nil
}

// VerifyTOTPCode checks a 6-digit code against the user's stored secret.
// Returns nil if the code is valid, error otherwise. Acceptable clock skew
// is +/- 30 seconds (the default for the library).
func VerifyTOTPCode(ctx context.Context, userID uuid.UUID, code string) error {
	secret, err := GetPlatformSecret(ctx, totpSecretName(userID))
	if err != nil {
		return fmt.Errorf("read totp secret: %w", err)
	}
	if secret == "" {
		return fmt.Errorf("no totp secret enrolled")
	}
	if !totp.Validate(code, secret) {
		return fmt.Errorf("invalid code")
	}
	return nil
}

// DisableTOTPSecret is called from the handler after the user has proved
// they can still disable 2FA (password re-verified). We do not destroy the
// secret here — a future re-enroll adds a new version and Secret Manager
// keeps the audit trail. The caller flips User.TOTPEnabled to false in DB.
func DisableTOTPSecret(ctx context.Context, userID uuid.UUID) error {
	_ = ctx
	_ = userID
	return nil
}
