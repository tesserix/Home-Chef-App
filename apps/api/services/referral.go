package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// ReferralConfig is the admin-tunable program config (#38), stored as
// PlatformSettings `referral.*` keys so the platform owner changes it at runtime.
type ReferralConfig struct {
	Enabled         bool    `json:"enabled"`
	ReferrerReward  float64 `json:"referrerReward"`
	RefereeReward   float64 `json:"refereeReward"`
	MonthlySpendCap float64 `json:"monthlySpendCap"`
}

// GetReferralConfig reads the program config, falling back to sane defaults so
// the feature works before an admin sets anything.
func GetReferralConfig(db *gorm.DB) ReferralConfig {
	cfg := ReferralConfig{Enabled: true, ReferrerReward: 100, RefereeReward: 100, MonthlySpendCap: 100000}
	var settings []models.PlatformSettings
	db.Where("key LIKE ?", "referral.%").Find(&settings)
	for _, s := range settings {
		switch s.Key {
		case "referral.enabled":
			cfg.Enabled = s.Value == "true" || s.Value == "1"
		case "referral.referrer_reward":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.ReferrerReward = v
			}
		case "referral.referee_reward":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.RefereeReward = v
			}
		case "referral.monthly_spend_cap":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.MonthlySpendCap = v
			}
		}
	}
	return cfg
}

// referral.go — customer referral program (#38). Code minting + redemption
// eligibility. The reward grant + admin config live in referral_reward.go.

// referralAlphabet excludes ambiguous characters (0/O, 1/I/L) so codes are easy
// to read aloud and type.
const referralAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
const referralCodeLen = 8

var (
	ErrReferralCodeInvalid = errors.New("referral code not found")
	ErrReferralSelf        = errors.New("you can't use your own referral code")
	ErrReferralAlreadyUsed = errors.New("you've already used a referral code")
	ErrReferralNotNewUser  = errors.New("referral codes are for new customers only")
)

// NormalizeReferralCode upper-cases + trims a user-entered code for comparison.
func NormalizeReferralCode(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// ReferralLink builds the shareable universal link for a code. The web app
// captures `/refer/:code`; the mobile app's universal-link config maps the same
// path back into the app.
func ReferralLink(code string) string {
	return "https://fe3dr.com/refer/" + code
}

// randomReferralCode returns a random code from the safe alphabet. The caller
// retries on a DB uniqueness collision.
func randomReferralCode() string {
	buf := make([]byte, referralCodeLen)
	_, _ = rand.Read(buf)
	out := make([]byte, referralCodeLen)
	for i, b := range buf {
		out[i] = referralAlphabet[int(b)%len(referralAlphabet)]
	}
	return string(out)
}

// GetOrCreateReferralCode returns the user's referral code, minting one (with a
// few collision retries) if they don't have one yet.
func GetOrCreateReferralCode(db *gorm.DB, userID uuid.UUID) (string, error) {
	var existing models.ReferralCode
	if err := db.Where("user_id = ?", userID).First(&existing).Error; err == nil {
		return existing.Code, nil
	}
	for attempt := 0; attempt < 6; attempt++ {
		code := randomReferralCode()
		if err := db.Create(&models.ReferralCode{UserID: userID, Code: code}).Error; err == nil {
			return code, nil
		}
		// Create failed — a code collision (retry) or a concurrent insert for
		// this user (re-read and return the row that won).
		var saved models.ReferralCode
		if e := db.Where("user_id = ?", userID).First(&saved).Error; e == nil {
			return saved.Code, nil
		}
	}
	return "", fmt.Errorf("could not generate a unique referral code")
}

// AcceptReferralInput carries the redemption request + a best-effort fraud
// snapshot (device token + IP at accept time).
type AcceptReferralInput struct {
	RefereeUserID uuid.UUID
	Code          string
	Device        string
	IP            string
}

// AcceptReferral records that the referee signed up with `code`. Guards: code
// exists, not self-referral, referee hasn't already redeemed, referee is new (no
// prior paid orders). Re-submitting the SAME code is idempotent; a different
// code after one is already set is rejected.
func AcceptReferral(db *gorm.DB, in AcceptReferralInput) (*models.Referral, error) {
	code := NormalizeReferralCode(in.Code)
	if code == "" {
		return nil, ErrReferralCodeInvalid
	}

	var rc models.ReferralCode
	if err := db.Where("code = ?", code).First(&rc).Error; err != nil {
		return nil, ErrReferralCodeInvalid
	}
	if rc.UserID == in.RefereeUserID {
		return nil, ErrReferralSelf
	}

	var existing models.Referral
	if err := db.Where("referee_user_id = ?", in.RefereeUserID).First(&existing).Error; err == nil {
		if existing.Code == code {
			return &existing, nil // idempotent re-accept
		}
		return nil, ErrReferralAlreadyUsed
	}

	// New-customer guard: referrals are first-order acquisition, so a referee
	// who already has a paid order isn't eligible.
	var paidCount int64
	db.Model(&models.Order{}).
		Where("customer_id = ? AND payment_status = ?", in.RefereeUserID, models.PaymentCompleted).
		Count(&paidCount)
	if paidCount > 0 {
		return nil, ErrReferralNotNewUser
	}

	ref := models.Referral{
		ReferrerUserID: rc.UserID,
		RefereeUserID:  in.RefereeUserID,
		Code:           code,
		Status:         models.ReferralStatePending,
		RefereeDevice:  in.Device,
		RefereeIP:      in.IP,
	}
	if err := db.Create(&ref).Error; err != nil {
		return nil, err
	}
	return &ref, nil
}

// ReferralStats summarises a referrer's program activity for the Refer & Earn UI.
type ReferralStats struct {
	RewardedCount int     `json:"rewardedCount"`
	PendingCount  int     `json:"pendingCount"`
	TotalEarned   float64 `json:"totalEarned"`
}

// GetReferralStats rolls up a referrer's redemptions.
func GetReferralStats(db *gorm.DB, referrerUserID uuid.UUID) ReferralStats {
	var refs []models.Referral
	db.Where("referrer_user_id = ?", referrerUserID).Find(&refs)
	var s ReferralStats
	for _, r := range refs {
		switch r.Status {
		case models.ReferralStateRewarded:
			s.RewardedCount++
			s.TotalEarned += r.ReferrerReward
		case models.ReferralStatePending:
			s.PendingCount++
		}
	}
	return s
}
