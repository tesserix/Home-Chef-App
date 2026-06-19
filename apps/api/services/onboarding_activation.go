package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// onboarding_activation.go — the chef-onboarding activation, extracted so it can
// run either inline from the approval handler or as a durable Temporal activity
// (#126). It is idempotent (every write is a state-set, repeatable safely), so a
// retried activity converges, and it returns an error so the workflow retries a
// transient DB failure — closing the gap where a crash mid-approval left a chef
// "approved" in the ApprovalRequest table but never actually verified/activated.

// ActivateChefOnboarding verifies + activates the chef behind a kitchen-onboarding
// approval and promotes their user to the chef role. No-op for any other approval
// type. Idempotent.
func ActivateChefOnboarding(db *gorm.DB, approvalID uuid.UUID) error {
	var approval models.ApprovalRequest
	if err := db.First(&approval, "id = ?", approvalID).Error; err != nil {
		return fmt.Errorf("onboarding: load approval %s: %w", approvalID, err)
	}
	if approval.Type != models.ApprovalKitchenOnboarding || approval.ChefID == nil {
		return nil // not a chef onboarding — nothing to activate
	}

	now := time.Now()
	if err := db.Model(&models.ChefProfile{}).Where("id = ?", *approval.ChefID).Updates(map[string]any{
		"is_verified": true,
		"verified_at": &now,
		"is_active":   true,
	}).Error; err != nil {
		return fmt.Errorf("onboarding: verify chef %s: %w", *approval.ChefID, err)
	}

	var chef models.ChefProfile
	if err := db.First(&chef, "id = ?", *approval.ChefID).Error; err != nil {
		return fmt.Errorf("onboarding: load chef %s: %w", *approval.ChefID, err)
	}
	if err := db.Model(&models.User{}).Where("id = ?", chef.UserID).
		Update("role", models.RoleChef).Error; err != nil {
		return fmt.Errorf("onboarding: promote user %s to chef: %w", chef.UserID, err)
	}
	return nil
}
