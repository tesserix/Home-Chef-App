package services

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// easy_split.go — split-at-capture policy.
//
// With Easy Split ON, a Cashfree checkout carries an order_split paying the
// chef's NET share (the same ComputeOrderEarnings figure the weekly statement
// would have paid) straight from capture, minus the flat platform fee. The
// remainder — commission, GST, TDS, delivery fee, tips-for-rider and the flat
// fee — stays with the platform merchant account and settles to the company's
// current account on Cashfree's schedule. The platform never holds the chef's
// money, which is the regulatory point.
//
// Every guard here fails toward NO SPLIT: full capture with the statement path
// paying the chef later is the safe, pre-existing behaviour.

const (
	// SettingEasySplitEnabled turns split-at-capture on. Default OFF.
	SettingEasySplitEnabled = "easy_split_enabled"
	// SettingPlatformFeeFlatMinor is the flat per-transaction platform fee in
	// paise, deducted from the chef's split share. Absent means zero.
	SettingPlatformFeeFlatMinor = "platform_fee_flat_minor"
)

// EasySplitEnabled reads the flag; any read failure means off.
func EasySplitEnabled(db *gorm.DB) bool {
	return settingBool(db, SettingEasySplitEnabled)
}

// PlatformFeeFlatMinor reads the flat fee. ok=false means the setting exists
// but cannot be parsed — callers must then refuse to split rather than guess.
func PlatformFeeFlatMinor(db *gorm.DB) (feeMinor int64, ok bool) {
	value := strings.TrimSpace(settingValue(db, SettingPlatformFeeFlatMinor))
	if value == "" {
		return 0, true
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n < 0 {
		log.Printf("easy-split: platform fee %q unparseable — splits disabled", value)
		return 0, false
	}
	return n, true
}

// BuildOrderSplit decides the Easy Split allocation for one checkout, or nil
// for a full platform capture.
//
// nil when: the flag is off, credit part-funds the order (the capture no
// longer covers the chef's share, so the statement path must settle it), the
// chef has no ACTIVE vendor registration, the chef's FSSAI licence has lapsed
// (payout is withheld, so the money must stay at the platform), the fee
// setting is unreadable, or the computed share rounds to nothing.
func BuildOrderSplit(db *gorm.DB, order *models.Order, capturePaise, creditPaise int) *CashfreeOrderSplit {
	if order == nil || !EasySplitEnabled(db) || creditPaise > 0 {
		return nil
	}
	chef := &order.Chef
	if chef.CashfreeVendorID == "" || !strings.EqualFold(chef.CashfreeVendorStatus, CashfreeVendorActive) {
		return nil
	}
	if IsChefFSSAIExpired(chef) {
		return nil
	}
	fee, ok := PlatformFeeFlatMinor(db)
	if !ok {
		return nil
	}

	share := int64(ToPaise(ChefNetPayoutFor(order))) - fee
	if share <= 0 {
		return nil
	}
	if share > int64(capturePaise) {
		share = int64(capturePaise)
	}
	return &CashfreeOrderSplit{
		VendorID:    chef.CashfreeVendorID,
		AmountPaise: CashfreeAmountFromPaise(int(share)),
	}
}

// EasySplitVendorIDFor is the deterministic vendor id for a chef —
// re-registration updates the same vendor instead of minting a parallel one.
func EasySplitVendorIDFor(chefID uuid.UUID) string {
	return "hc_" + strings.ReplaceAll(chefID.String(), "-", "")
}

var easySplitNameStrip = regexp.MustCompile(`[^a-zA-Z0-9 ./\-&]`)

func easySplitName(name string) string {
	clean := strings.TrimSpace(easySplitNameStrip.ReplaceAllString(name, " "))
	if clean == "" {
		return "Fe3dr Kitchen"
	}
	return clean
}

var easySplitPhoneStrip = regexp.MustCompile(`[^0-9]`)

func easySplitPhone(phone string) string {
	digits := easySplitPhoneStrip.ReplaceAllString(phone, "")
	digits = strings.TrimPrefix(digits, "91")
	if len(digits) < 8 || len(digits) > 12 {
		return ""
	}
	return digits
}

// EnsureEasySplitVendorWith registers (or updates) the chef's Easy Split
// vendor from bank details the caller already holds — the payout-details save
// path, which must not read secrets back mid-write.
func EnsureEasySplitVendorWith(
	ctx context.Context, db *gorm.DB, chef *models.ChefProfile,
	bank CashfreeVendorBank, email, phone string,
) (*CashfreeVendorResponse, error) {
	cf := GetCashfreeFor(chef.Mode)
	if cf == nil {
		return nil, fmt.Errorf("easy-split: cashfree is not configured for the chef's %q mode", chef.Mode)
	}
	if bank.AccountNumber == "" || bank.IFSC == "" || bank.AccountHolder == "" {
		return nil, fmt.Errorf("easy-split: chef %s has no bank details on file", chef.ID)
	}
	p := easySplitPhone(phone)
	if p == "" {
		return nil, fmt.Errorf("easy-split: chef %s has no usable phone number on file", chef.ID)
	}

	req := &CashfreeVendorRequest{
		VendorID:      EasySplitVendorIDFor(chef.ID),
		Status:        CashfreeVendorActive,
		Name:          easySplitName(chef.BusinessName),
		Email:         email,
		Phone:         p,
		VerifyAccount: true,
		Bank:          &bank,
		KYC: CashfreeVendorKYC{
			AccountType:  "savings",
			BusinessType: "Food and Beverages",
			PAN:          strings.ToUpper(strings.TrimSpace(chef.PanNumber)),
		},
	}
	vendor, err := cf.CreateVendor(req)
	if err != nil {
		return nil, err
	}

	if uErr := db.Model(&models.ChefProfile{}).Where("id = ?", chef.ID).Updates(map[string]any{
		"cashfree_vendor_id":     vendor.VendorID,
		"cashfree_vendor_status": vendor.Status,
	}).Error; uErr != nil {
		return vendor, fmt.Errorf("easy-split: vendor %s registered but not persisted: %w", vendor.VendorID, uErr)
	}
	chef.CashfreeVendorID = vendor.VendorID
	chef.CashfreeVendorStatus = vendor.Status
	_ = ctx
	return vendor, nil
}

// EnsureEasySplitVendor is the admin-triggered variant: bank details come from
// Secret Manager, identity from the chef's user row.
func EnsureEasySplitVendor(ctx context.Context, db *gorm.DB, chef *models.ChefProfile) (*CashfreeVendorResponse, error) {
	vendorID := chef.ID.String()
	holder, _ := GetVendorSecret(ctx, vendorID, "bank-account-name")
	account, _ := GetVendorSecret(ctx, vendorID, "bank-account-number")
	ifsc, _ := GetVendorSecret(ctx, vendorID, "bank-ifsc")

	var user models.User
	if err := db.First(&user, "id = ?", chef.UserID).Error; err != nil {
		return nil, fmt.Errorf("easy-split: load user for chef %s: %w", chef.ID, err)
	}
	return EnsureEasySplitVendorWith(ctx, db, chef,
		CashfreeVendorBank{AccountNumber: account, AccountHolder: holder, IFSC: ifsc},
		user.Email, user.Phone)
}

// RefreshEasySplitVendor re-reads the vendor's status from Cashfree — the
// answer to "has the destination finished verifying?" — and persists it.
func RefreshEasySplitVendor(_ context.Context, db *gorm.DB, chef *models.ChefProfile) (*CashfreeVendorResponse, error) {
	cf := GetCashfreeFor(chef.Mode)
	if cf == nil {
		return nil, fmt.Errorf("easy-split: cashfree is not configured for the chef's %q mode", chef.Mode)
	}
	id := chef.CashfreeVendorID
	if id == "" {
		id = EasySplitVendorIDFor(chef.ID)
	}
	vendor, err := cf.FetchVendor(id)
	if err != nil {
		return nil, err
	}
	if uErr := db.Model(&models.ChefProfile{}).Where("id = ?", chef.ID).Updates(map[string]any{
		"cashfree_vendor_id":     vendor.VendorID,
		"cashfree_vendor_status": vendor.Status,
	}).Error; uErr != nil {
		return vendor, fmt.Errorf("easy-split: vendor %s refreshed but not persisted: %w", vendor.VendorID, uErr)
	}
	chef.CashfreeVendorID = vendor.VendorID
	chef.CashfreeVendorStatus = vendor.Status
	return vendor, nil
}
