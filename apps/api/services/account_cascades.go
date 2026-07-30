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
	"strings"

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
	return purgeUserPersonalData(tx, userID)
}

// customerPurgeUserTables are the personal tables keyed by user_id that the
// purge erases outright. Every row here belongs to the user alone — no
// counterparty needs it and no statute requires it — so at the end of the
// restore window it simply goes.
//
// This list replaced a single `DELETE FROM addresses`: testing a real purge
// end-to-end left rows in ten tables, including customer_profiles — which
// holds date_of_birth and food_allergies, the most sensitive data the platform
// collects — still keyed to a "deleted" account.
//
// Deliberately NOT here:
//   - orders, order_invoices — the counterparty's financial record; PII is
//     scrubbed in place instead (scrubOrderPII / scrubInvoicePII).
//   - meal_plans, credit_notes, ledger/earnings/transactions rows — statutory
//     money records (CGST §36); carry amounts, not personal data.
//   - reviews, tips, dish_ratings — the chef's rating and earnings record; the
//     author becomes anonymous once the users row is gone.
//   - audit_logs — its own retention cron (audit_retention_cron.go) expires
//     them on the legal audit clock.
var customerPurgeUserTables = []string{
	"addresses",
	"campaign_deliveries",
	"customer_profiles",
	"email_verification_tokens",
	"favorite_chefs",
	"favorite_dishes",
	"group_order_participants",
	"loyalty_accounts",
	"loyalty_earn_batches",
	"loyalty_transactions",
	"mfa_backup_codes",
	"notification_preferences",
	"notifications",
	"password_reset_tokens",
	"payment_methods",
	"post_comments",
	"post_likes",
	"promo_code_usages",
	"referral_codes",
	"refresh_tokens",
	"subscriptions",
	"trusted_devices",
	"user_mfa_settings",
	"wallet_txns",
	"wallets",
	"winback_offers",
}

// purgeUserPersonalData erases the customer-side footprint every account owns
// regardless of role — chefs and drivers place orders and hold wallets too, so
// all three cascades run this. Tables are guarded with HasTable (mirroring
// PurgeTestSession) so unit-test fixtures that create only a few tables still
// work; in production every table exists and nothing is skipped.
func purgeUserPersonalData(tx *gorm.DB, userID uuid.UUID) error {
	// The avatar object first, while the row still tells us where it is.
	var u models.User
	if err := tx.Unscoped().Select("avatar").First(&u, "id = ?", userID).Error; err == nil {
		removeStoredObjectURL(u.Avatar)
	}

	for _, table := range customerPurgeUserTables {
		if !tx.Migrator().HasTable(table) {
			continue
		}
		if err := tx.Exec(`DELETE FROM `+table+` WHERE user_id = ?`, userID).Error; err != nil {
			return fmt.Errorf("account: purge %s: %w", table, err)
		}
	}

	// customer_id-keyed personal rows.
	for _, table := range []string{"catering_requests", "meal_trials"} {
		if !tx.Migrator().HasTable(table) {
			continue
		}
		if err := tx.Exec(`DELETE FROM `+table+` WHERE customer_id = ?`, userID).Error; err != nil {
			return fmt.Errorf("account: purge %s: %w", table, err)
		}
	}

	// Blocks in either direction — a block row names both parties.
	if tx.Migrator().HasTable("user_blocks") {
		if err := tx.Exec(`DELETE FROM user_blocks WHERE blocker_id = ? OR blocked_id = ?`,
			userID, userID).Error; err != nil {
			return fmt.Errorf("account: purge user_blocks: %w", err)
		}
	}

	// Conversations: children first (messages key on the room, not the user).
	if tx.Migrator().HasTable("chat_rooms") {
		if tx.Migrator().HasTable("chat_messages") {
			if err := tx.Exec(`DELETE FROM chat_messages WHERE chat_room_id IN
				(SELECT id FROM chat_rooms WHERE customer_id = ?)`, userID).Error; err != nil {
				return fmt.Errorf("account: purge chat_messages: %w", err)
			}
		}
		if err := tx.Exec(`DELETE FROM chat_rooms WHERE customer_id = ?`, userID).Error; err != nil {
			return fmt.Errorf("account: purge chat_rooms: %w", err)
		}
	}

	// Carts: items key on the cart.
	if tx.Migrator().HasTable("carts") {
		if tx.Migrator().HasTable("cart_items") {
			if err := tx.Exec(`DELETE FROM cart_items WHERE cart_id IN
				(SELECT id FROM carts WHERE user_id = ?)`, userID).Error; err != nil {
				return fmt.Errorf("account: purge cart_items: %w", err)
			}
		}
		if err := tx.Exec(`DELETE FROM carts WHERE user_id = ?`, userID).Error; err != nil {
			return fmt.Errorf("account: purge carts: %w", err)
		}
	}

	if err := scrubOrderPII(tx, userID); err != nil {
		return err
	}
	return scrubInvoicePII(tx, userID)
}

// scrubOrderPII de-identifies the retained order rows. The rows themselves stay:
// they are the chef's financial record too (earnings, statements, TDS), and the
// PII-free archive (ArchiveAccountFinancials) plus these de-identified rows is
// what "financial records retained, personal data erased" actually means. The
// street address, coordinates and free-text instructions are the identifying
// parts; city/state stay for regional reporting.
func scrubOrderPII(tx *gorm.DB, userID uuid.UUID) error {
	// Column-level guard, not just table-level: test fixtures create a minimal
	// orders table without the address columns.
	if !tx.Migrator().HasTable("orders") ||
		!tx.Migrator().HasColumn(&models.Order{}, "delivery_address_line1") {
		return nil
	}
	if err := tx.Exec(`UPDATE orders SET
			delivery_address_line1 = '', delivery_address_line2 = '',
			delivery_address_postal_code = '',
			delivery_latitude = 0, delivery_longitude = 0,
			delivery_instructions = '', special_instructions = ''
		WHERE customer_id = ?`, userID).Error; err != nil {
		return fmt.Errorf("account: scrub order PII: %w", err)
	}
	// Encrypted companions exist only where PII crypto migrated them (#710);
	// SQLite fixtures and pre-migration schemas don't have the columns.
	if tx.Migrator().HasColumn(&models.Order{}, "delivery_address_line1_enc") {
		if err := tx.Exec(`UPDATE orders SET
				delivery_address_line1_enc = '', delivery_address_line2_enc = ''
			WHERE customer_id = ?`, userID).Error; err != nil {
			return fmt.Errorf("account: scrub order PII (enc): %w", err)
		}
	}
	return nil
}

// scrubInvoicePII does the same for the frozen invoice copies, which duplicate
// the customer's name, email, phone and address at invoice time (#710 P1).
func scrubInvoicePII(tx *gorm.DB, userID uuid.UUID) error {
	if !tx.Migrator().HasTable("order_invoices") ||
		!tx.Migrator().HasColumn(&models.OrderInvoice{}, "customer_name") {
		return nil
	}
	if err := tx.Exec(`UPDATE order_invoices SET
			customer_name = '', customer_email = '', customer_phone = '',
			customer_address = ''
		WHERE customer_id = ?`, userID).Error; err != nil {
		return fmt.Errorf("account: scrub invoice PII: %w", err)
	}
	if tx.Migrator().HasColumn(&models.OrderInvoice{}, "customer_name_enc") {
		if err := tx.Exec(`UPDATE order_invoices SET
				customer_name_enc = '', customer_email_enc = '', customer_email_bidx = '',
				customer_phone_enc = '', customer_phone_bidx = '', customer_address_enc = ''
			WHERE customer_id = ?`, userID).Error; err != nil {
			return fmt.Errorf("account: scrub invoice PII (enc): %w", err)
		}
	}
	return nil
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

// chefPurgeTables are the chef_id-keyed tables the purge erases outright: the
// kitchen's configuration, menu, content and compliance records. Orders, meal
// plans, group orders and weekly_statements are deliberately NOT here — they
// are the counterparty's financial record too, and the deletion blockers
// already guaranteed none are live. Reviews, ratings and favorites pointing at
// the kitchen DO go: content about a kitchen that no longer exists serves
// nobody, and leaving it keyed to a purged chef is retention without purpose.
var chefPurgeTables = []string{
	"catering_quotes",
	"chef_capacity_settings",
	"chef_documents",
	"chef_mode_stats",
	"chef_notification_preferences",
	"chef_penalties",
	"chef_promotions",
	"chef_schedules",
	"chef_settings",
	"chef_slot_daily_bookings",
	"chef_subscription_configs",
	"chef_test_sessions",
	"daily_menu_items",
	"daily_menus",
	"dish_ratings",
	"favorite_chefs",
	"menu_categories",
	"menu_item_daily_sales",
	"posts",
	"promo_codes",
	"reviews",
	"weekly_menu_items",
	"weekly_menus",
	// menu_items last of the menu family — menu_item_images is deleted via a
	// subselect on it first, below.
	"menu_items",
}

func (chefCascade) Purge(tx *gorm.DB, userID uuid.UUID) error {
	chefID, ok := chefProfileID(tx, userID)
	if !ok {
		// No profile — still erase the customer-side footprint this user owns.
		return purgeUserPersonalData(tx, userID)
	}

	// Stored objects first, while the rows still say where they are. ALL
	// document types, not just the identity subset the restore path drops: the
	// old purge deleted chef_documents rows and left every uploaded
	// PAN/FSSAI/bank object in the bucket forever.
	var docs []models.ChefDocument
	if err := tx.Where("chef_id = ?", chefID).Find(&docs).Error; err != nil {
		return fmt.Errorf("account: load chef documents for purge: %w", err)
	}
	for _, d := range docs {
		removeStoredObject(d.Bucket, d.FilePath)
	}
	var chef models.ChefProfile
	if err := tx.Select("profile_image").First(&chef, "id = ?", chefID).Error; err == nil {
		removeStoredObjectURL(chef.ProfileImage)
	}

	// Payout bank details live in Secret Manager, not the database — the most
	// sensitive thing a chef gives us, and the old purge left them behind.
	// Best-effort: a missing secret or an unconfigured client must not stall
	// the erasure, and DeleteVendorSecret treats not-found as success.
	for _, field := range []string{"bank-account-number", "bank-account-name", "bank-ifsc", "upi-id"} {
		if err := DeleteVendorSecret(context.Background(), chefID.String(), field); err != nil {
			log.Printf("account: could not delete vendor secret %s for chef=%s: %v", field, chefID, err)
		}
	}

	// menu_item_images keys on the menu item, not the chef — delete via
	// subselect while menu_items still exists.
	if tx.Migrator().HasTable("menu_item_images") {
		if err := tx.Exec(`DELETE FROM menu_item_images WHERE menu_item_id IN
			(SELECT id FROM menu_items WHERE chef_id = ?)`, chefID).Error; err != nil {
			return fmt.Errorf("account: purge menu_item_images: %w", err)
		}
	}

	// Raw DELETEs rather than GORM model deletes: several of these models carry
	// composite unique indexes and soft-delete columns, and a model-scoped
	// Delete quietly becomes an UPDATE that leaves rows behind. The purge must
	// actually remove them. HasTable-guarded for test fixtures, as elsewhere.
	for _, table := range chefPurgeTables {
		if !tx.Migrator().HasTable(table) {
			continue
		}
		if err := tx.Exec(`DELETE FROM `+table+` WHERE chef_id = ?`, chefID).Error; err != nil {
			return fmt.Errorf("account: purge %s: %w", table, err)
		}
	}

	if err := tx.Exec(`DELETE FROM chef_profiles WHERE id = ?`, chefID).Error; err != nil {
		return fmt.Errorf("account: purge chef_profiles: %w", err)
	}
	// The chef is also a customer: wallet, addresses, notifications and the
	// rest of the personal footprint go the same way as for any other user.
	return purgeUserPersonalData(tx, userID)
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
		// No partner row — still erase the customer-side footprint.
		return purgeUserPersonalData(tx, userID)
	}
	// purgeDriverDocuments (not a raw DELETE): it removes the stored licence and
	// identity objects before dropping the rows. The old purge deleted the rows
	// directly and left every uploaded document in the bucket forever.
	if err := purgeDriverDocuments(tx, partnerID); err != nil {
		return err
	}
	if err := tx.Exec(`DELETE FROM driver_referrals WHERE referrer_id = ?`,
		userID).Error; err != nil {
		return fmt.Errorf("account: purge driver referrals: %w", err)
	}
	if err := tx.Exec(`DELETE FROM delivery_partners WHERE id = ?`, partnerID).Error; err != nil {
		return fmt.Errorf("account: purge delivery_partners: %w", err)
	}
	// The driver is also a customer — same personal footprint as everyone else.
	return purgeUserPersonalData(tx, userID)
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

// removeStoredObjectURL deletes an object addressed by its public URL — the
// form UploadPublicFile returns and profile images / avatars store
// (https://storage.googleapis.com/{bucket}/{path}). Anything that isn't a URL
// in that shape (an external avatar from a social login, an empty string) is
// simply not ours to delete and is skipped.
func removeStoredObjectURL(rawURL string) {
	const prefix = "https://storage.googleapis.com/"
	if !strings.HasPrefix(rawURL, prefix) {
		return
	}
	bucket, objectPath, found := strings.Cut(strings.TrimPrefix(rawURL, prefix), "/")
	if !found {
		return
	}
	removeStoredObject(bucket, objectPath)
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
