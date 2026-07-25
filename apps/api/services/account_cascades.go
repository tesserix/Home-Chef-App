package services

// account_cascades.go — the role-specific half of each account transition.
//
// The re-approval rule lives here: restoring an account returns the user's data
// but NOT their standing. A chef may well have been deleted *because* of a
// compliance problem, and their FSSAI licence or PAN may have expired or been
// revoked while they were gone. So restore clears every approval flag, drops
// the identity documents, and raises a fresh approval request — a restored chef
// re-enters the queue exactly like a first-time applicant.
//
// Kitchen photos, the profile image, menu photos and descriptions survive: they
// are the user's own content, they carry no compliance claim, and making people
// re-shoot their whole menu would defeat the point of keeping the data at all.

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// identityDocTypes are the compliance documents that must be re-uploaded after
// a restore. Anything asserting a licence, an identity or a bank destination is
// in this set; photographs of the kitchen are not.
func identityDocTypes() []models.DocumentType {
	return []models.DocumentType{
		models.DocPanCard,
		models.DocAadhaarCard,
		models.DocFSSAILicense,
		models.DocFoodSafetyCert,
		models.DocCancelledCheque,
	}
}

// ---------------------------------------------------------------- customer

type customerCascade struct{}

func (customerCascade) OnDeactivate(tx *gorm.DB, userID uuid.UUID) error {
	// Stop push and marketing while paused. The FCM token is cleared rather
	// than kept so a paused account stops receiving notifications immediately.
	return tx.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{"fcm_token": "", "marketing_consent": false}).Error
}

func (c customerCascade) OnDelete(tx *gorm.DB, userID uuid.UUID) error {
	// Addresses are deliberately NOT deleted here. models.Address has no
	// DeletedAt, so deleting it is a HARD delete — the row would be gone
	// forever and the restore window would be a promise we cannot keep. The
	// account is already unreachable (soft-deleted user + is_active=false), and
	// addresses are only ever readable through the owner's own session, so
	// leaving them in place exposes nothing. The purge sweeper removes them.
	return c.OnDeactivate(tx, userID)
}

func (customerCascade) OnRestore(tx *gorm.DB, _ uuid.UUID) error {
	// Customers carry no approval, and nothing was destroyed on delete, so
	// there is nothing to undo. Marketing consent stays off: it was withdrawn,
	// and DPDP §6 requires a fresh opt-in rather than a silent reinstatement.
	return nil
}

func (customerCascade) Purge(tx *gorm.DB, userID uuid.UUID) error {
	return tx.Exec(`DELETE FROM addresses WHERE user_id = ?`, userID).Error
}

// ------------------------------------------------------------------- chef

type chefCascade struct{}

// chefProfileID resolves the profile id for a user. chef_profiles has no
// soft-delete column, so the row is simply present until the purge sweeper
// removes it — no Unscoped needed.
func chefProfileID(tx *gorm.DB, userID uuid.UUID) (uuid.UUID, bool) {
	var chef models.ChefProfile
	if err := tx.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		return uuid.Nil, false
	}
	return chef.ID, true
}

func (chefCascade) OnDeactivate(tx *gorm.DB, userID uuid.UUID) error {
	chefID, ok := chefProfileID(tx, userID)
	if !ok {
		return nil
	}
	// auto_schedule_enabled MUST be cleared alongside accepting_orders: the
	// kitchen-schedule cron flips accepting_orders back on from the chef's
	// operating hours, so without this a deactivated kitchen silently reopens
	// on the next scan.
	return tx.Model(&models.ChefProfile{}).
		Where("id = ?", chefID).
		Updates(map[string]any{
			"accepting_orders":      false,
			"auto_schedule_enabled": false,
			"is_active":             false,
		}).Error
}

func (c chefCascade) OnDelete(tx *gorm.DB, userID uuid.UUID) error {
	if err := c.OnDeactivate(tx, userID); err != nil {
		return err
	}
	chefID, ok := chefProfileID(tx, userID)
	if !ok {
		return nil
	}
	// Take the menu offline so customers do not see a phantom kitchen during
	// the retention window.
	//
	// The ChefProfile row itself is deliberately left in place. models.ChefProfile
	// has no DeletedAt, so deleting it is a HARD delete: the kitchen would be
	// unrecoverable and every order's chef_id would be orphaned. is_active=false
	// (set by OnDeactivate above) already removes the kitchen from every
	// customer-facing query — see handlers/chefs.go:90 and :319 — so the account
	// is invisible without destroying anything. The purge sweeper removes it at
	// the end of the retention window.
	return tx.Model(&models.MenuItem{}).
		Where("chef_id = ?", chefID).
		Update("is_available", false).Error
}

func (chefCascade) OnRestore(tx *gorm.DB, userID uuid.UUID) error {
	chefID, ok := chefProfileID(tx, userID)
	if !ok {
		return nil
	}

	// Bring the profile back, but pending: not approved, not accepting orders,
	// not auto-scheduling. Nothing about this kitchen is customer-visible until
	// an admin re-approves it.
	// Chef approval is is_verified + verified_at (there is no is_approved column
	// on chef_profiles — that one belongs to menu_items). Clearing both is what
	// puts the kitchen back in the approval queue.
	// No deleted_at here — chef_profiles has no such column. The row was never
	// deleted, only taken offline, so restoring is purely a matter of clearing
	// verification and leaving the kitchen closed until an admin re-approves.
	if err := tx.Exec(`UPDATE chef_profiles SET is_verified = ?, verified_at = NULL,
			accepting_orders = ?, auto_schedule_enabled = ?, is_active = ?
		WHERE id = ?`, false, false, false, true, chefID).Error; err != nil {
		return fmt.Errorf("account: restore chef profile: %w", err)
	}

	// Every menu item returns unapproved, so the menu-approval gate re-reviews
	// it. is_available stays false until the chef reopens.
	if err := tx.Model(&models.MenuItem{}).
		Where("chef_id = ?", chefID).
		Updates(map[string]any{"is_approved": false, "is_available": false}).Error; err != nil {
		return fmt.Errorf("account: reset menu approval: %w", err)
	}

	// Drop the compliance documents — rows and stored objects both. They may
	// have expired or been revoked during the absence, so they must be
	// re-uploaded and re-verified rather than silently reinstated.
	if err := purgeIdentityDocuments(tx, chefID); err != nil {
		return err
	}

	return queueReapproval(tx, models.ApprovalKitchenOnboarding, "chef_profile", chefID, userID)
}

func (chefCascade) Purge(tx *gorm.DB, userID uuid.UUID) error {
	chefID, ok := chefProfileID(tx, userID)
	if !ok {
		return nil
	}
	// Children before parent. Orders and meal plans are deliberately NOT deleted
	// here: they are the counterparty's financial record too, and the deletion
	// blockers already guaranteed none are live.
	//
	// Raw DELETEs rather than GORM model deletes: several of these models carry
	// composite unique indexes and soft-delete columns, and a model-scoped
	// Delete quietly becomes an UPDATE that leaves rows behind. The purge must
	// actually remove them.
	for _, table := range []string{
		"chef_documents", "chef_schedules", "chef_settings", "menu_items",
	} {
		if err := tx.Exec(`DELETE FROM `+table+` WHERE chef_id = ?`, chefID).Error; err != nil {
			return fmt.Errorf("account: purge %s: %w", table, err)
		}
	}
	return tx.Exec(`DELETE FROM chef_profiles WHERE id = ?`, chefID).Error
}

// ----------------------------------------------------------------- driver

type driverCascade struct{}

func driverPartnerID(tx *gorm.DB, userID uuid.UUID) (uuid.UUID, bool) {
	var p models.DeliveryPartner
	if err := tx.Where("user_id = ?", userID).First(&p).Error; err != nil {
		return uuid.Nil, false
	}
	return p.ID, true
}

func (driverCascade) OnDeactivate(tx *gorm.DB, userID uuid.UUID) error {
	partnerID, ok := driverPartnerID(tx, userID)
	if !ok {
		return nil
	}
	// is_online false + is_active false removes the driver from dispatch, which
	// selects on exactly these columns.
	return tx.Model(&models.DeliveryPartner{}).
		Where("id = ?", partnerID).
		Updates(map[string]any{"is_online": false, "is_active": false}).Error
}

func (d driverCascade) OnDelete(tx *gorm.DB, userID uuid.UUID) error {
	return d.OnDeactivate(tx, userID)
}

func (driverCascade) OnRestore(tx *gorm.DB, userID uuid.UUID) error {
	partnerID, ok := driverPartnerID(tx, userID)
	if !ok {
		return nil
	}
	if err := tx.Model(&models.DeliveryPartner{}).
		Where("id = ?", partnerID).
		Updates(map[string]any{
			"is_verified": false,
			"is_online":   false,
			"is_active":   false, // stays out of dispatch until re-approved
		}).Error; err != nil {
		return fmt.Errorf("account: restore driver: %w", err)
	}
	if err := purgeDriverDocuments(tx, partnerID); err != nil {
		return err
	}
	return queueReapproval(tx, models.ApprovalDriverOnboarding, "delivery_partner", partnerID, userID)
}

func (driverCascade) Purge(tx *gorm.DB, userID uuid.UUID) error {
	partnerID, ok := driverPartnerID(tx, userID)
	if !ok {
		return nil
	}
	if err := tx.Exec(`DELETE FROM delivery_partner_documents WHERE partner_id = ?`,
		partnerID).Error; err != nil {
		return fmt.Errorf("account: purge driver documents: %w", err)
	}
	if err := tx.Exec(`DELETE FROM driver_referrals WHERE referrer_id = ?`,
		userID).Error; err != nil {
		return fmt.Errorf("account: purge driver referrals: %w", err)
	}
	return tx.Exec(`DELETE FROM delivery_partners WHERE id = ?`, partnerID).Error
}

// --------------------------------------------------------------- helpers

// purgeIdentityDocuments deletes a chef's compliance documents, best-effort
// removing the stored objects too. A storage failure is logged, not fatal: the
// database row going away is what actually revokes the claim, and leaving an
// orphaned object is preferable to failing the whole restore.
func purgeIdentityDocuments(tx *gorm.DB, chefID uuid.UUID) error {
	var docs []models.ChefDocument
	if err := tx.Where("chef_id = ? AND type IN ?", chefID, identityDocTypes()).
		Find(&docs).Error; err != nil {
		return fmt.Errorf("account: load identity documents: %w", err)
	}
	for _, d := range docs {
		removeStoredObject(d.Bucket, d.FilePath)
	}
	return tx.Unscoped().
		Where("chef_id = ? AND type IN ?", chefID, identityDocTypes()).
		Delete(&models.ChefDocument{}).Error
}

// purgeDriverDocuments does the same for a delivery partner. Every driver
// document is an identity or licence claim, so all of them go.
func purgeDriverDocuments(tx *gorm.DB, partnerID uuid.UUID) error {
	var docs []models.DeliveryPartnerDocument
	if err := tx.Where("partner_id = ?", partnerID).Find(&docs).Error; err != nil {
		return fmt.Errorf("account: load driver documents: %w", err)
	}
	for _, d := range docs {
		removeStoredObject(d.Bucket, d.FilePath)
	}
	return tx.Unscoped().Where("partner_id = ?", partnerID).
		Delete(&models.DeliveryPartnerDocument{}).Error
}

// removeStoredObject deletes an uploaded object best-effort. A storage failure
// is logged, never fatal: the database row going away is what actually revokes
// the compliance claim, and an orphaned object in GCS is far preferable to a
// failed restore or a stalled purge.
func removeStoredObject(bucket, path string) {
	if bucket == "" || path == "" {
		return
	}
	if err := DeleteFile(context.Background(), bucket, path); err != nil {
		log.Printf("account: could not remove object %s/%s: %v", bucket, path, err)
	}
}

// queueReapproval raises a pending approval request for a restored account,
// flagged so the admin sees this is a returning user rather than a new signup
// and can weigh why the account was deleted before approving.
func queueReapproval(tx *gorm.DB, kind models.ApprovalRequestType, entityType string, entityID, userID uuid.UUID) error {
	req := models.ApprovalRequest{
		Type:          kind,
		Status:        models.ApprovalPending,
		EntityType:    entityType,
		EntityID:      entityID,
		SubmittedByID: userID,
		Title:         "Restored account — re-approval required",
		Description: "This account was previously deleted and has been restored by the user " +
			"within the retention window. Approval was reset and the identity documents were " +
			"removed: they must be re-uploaded and re-verified before this account goes live.",
		SubmittedData: `{"restored_account":true}`,
	}
	switch entityType {
	case "chef_profile":
		req.ChefID = &entityID
	case "delivery_partner":
		req.PartnerID = &entityID
	}
	if err := tx.Create(&req).Error; err != nil {
		return fmt.Errorf("account: queue re-approval: %w", err)
	}
	return nil
}
