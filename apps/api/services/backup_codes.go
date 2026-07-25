package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// Backup codes — the recovery path when both enrolled channels are out of reach.
//
// Stored as HMAC-SHA256 under a server-side key, not a bare hash. Codes are
// short enough to be typed by a human, so a bare hash of a leaked table would
// be brute-forceable offline in seconds; the keyed MAC means an attacker needs
// the key as well as the rows.

var (
	ErrBackupCodeInvalid  = errors.New("invalid code")
	ErrBackupCodesMissing = errors.New("no backup codes issued")
	ErrBackupKeyMissing   = errors.New("backup code key not configured")
)

const (
	// BackupCodeCount is how many codes are issued per generation.
	BackupCodeCount = 8
	// LowBackupCodeThreshold is when we start nudging the user to regenerate.
	LowBackupCodeThreshold = 2

	backupCodeEntropyBytes = 10 // 80 bits → 16 base32 chars, shown as XXXX-XXXX-XXXX-XXXX
)

// backupCodeAlphabet omits padding so codes stay clean to read aloud.
var backupCodeAlphabet = base32.StdEncoding.WithPadding(base32.NoPadding)

// NormalizeBackupCode makes user input comparable: case and separators vary
// wildly when someone reads a code off paper.
func NormalizeBackupCode(code string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(code)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// FormatBackupCode groups a normalized code into readable quads.
func FormatBackupCode(norm string) string {
	var parts []string
	for i := 0; i < len(norm); i += 4 {
		end := min(i+4, len(norm))
		parts = append(parts, norm[i:end])
	}
	return strings.Join(parts, "-")
}

func backupCodeHMAC(key, normalized string) string {
	m := hmac.New(sha256.New, []byte(key))
	m.Write([]byte(normalized))
	return hex.EncodeToString(m.Sum(nil))
}

func generateBackupCode() (string, error) {
	raw := make([]byte, backupCodeEntropyBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return backupCodeAlphabet.EncodeToString(raw), nil
}

// GenerateBackupCodes replaces any existing codes with a fresh set and returns
// the plaintext, formatted for display. This is the only time the codes exist
// in readable form.
//
// Regenerating deletes the old set: leaving both live would mean a user who
// regenerates because they think the old sheet leaked still has the leaked
// codes working.
func GenerateBackupCodes(db *gorm.DB, userID uuid.UUID, key string) ([]string, error) {
	if key == "" {
		return nil, ErrBackupKeyMissing
	}
	codes := make([]string, 0, BackupCodeCount)
	rows := make([]models.MFABackupCode, 0, BackupCodeCount)
	now := time.Now()

	for range BackupCodeCount {
		norm, err := generateBackupCode()
		if err != nil {
			return nil, err
		}
		codes = append(codes, FormatBackupCode(norm))
		rows = append(rows, models.MFABackupCode{
			ID:        uuid.New(),
			UserID:    userID,
			CodeHMAC:  backupCodeHMAC(key, norm),
			CreatedAt: now,
		})
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.MFABackupCode{}).Error; err != nil {
			return err
		}
		return tx.Create(&rows).Error
	})
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// RedeemBackupCode consumes one unused code and reports how many remain.
//
// Returns ErrBackupCodeInvalid both for a wrong code and for one already spent,
// so a caller cannot learn which codes were previously valid.
func RedeemBackupCode(db *gorm.DB, userID uuid.UUID, submitted, key string) (remaining int, err error) {
	if key == "" {
		return 0, ErrBackupKeyMissing
	}
	norm := NormalizeBackupCode(submitted)
	if norm == "" {
		return 0, ErrBackupCodeInvalid
	}
	want := backupCodeHMAC(key, norm)

	var rows []models.MFABackupCode
	if err := db.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, ErrBackupCodesMissing
	}

	// Walk every row rather than querying by hash, and compare in constant
	// time, so redemption takes the same shape regardless of which code (if
	// any) matched.
	var matched *models.MFABackupCode
	for i := range rows {
		if rows[i].UsedAt != nil {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(rows[i].CodeHMAC), []byte(want)) == 1 {
			matched = &rows[i]
		}
	}
	if matched == nil {
		return 0, ErrBackupCodeInvalid
	}

	now := time.Now()
	// Guard on used_at inside the write too: two concurrent redemptions of the
	// same code must not both succeed.
	res := db.Model(&models.MFABackupCode{}).
		Where("id = ? AND used_at IS NULL", matched.ID).
		Update("used_at", now)
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		return 0, ErrBackupCodeInvalid
	}

	return CountUnusedBackupCodes(db, userID)
}

// CountUnusedBackupCodes reports how many codes the user has left.
func CountUnusedBackupCodes(db *gorm.DB, userID uuid.UUID) (int, error) {
	var n int64
	err := db.Model(&models.MFABackupCode{}).
		Where("user_id = ? AND used_at IS NULL", userID).Count(&n).Error
	return int(n), err
}
