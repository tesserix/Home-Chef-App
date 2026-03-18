package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type DriverOnboardingHandler struct{}

func NewDriverOnboardingHandler() *DriverOnboardingHandler {
	return &DriverOnboardingHandler{}
}

// GetDriverOnboardingStatus returns the current onboarding status for the driver
// GET /driver/onboarding/status
func (h *DriverOnboardingHandler) GetDriverOnboardingStatus(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Preload("User").Preload("Documents").
		Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"step":   0,
			"status": "not_started",
		})
		return
	}

	var docCount int64
	database.DB.Model(&models.DeliveryPartnerDocument{}).
		Where("partner_id = ?", partner.ID).Count(&docCount)

	payoutSet := partner.PayoutMethod != ""

	// Map verification status to a simple status field the frontend expects
	status := "in_progress"
	switch partner.VerificationStatus {
	case models.VerificationInReview:
		if partner.OnboardingComplete {
			status = "in_review"
		} else {
			status = "submitted"
		}
	case models.VerificationApproved:
		status = "approved"
	case models.VerificationRejected:
		status = "rejected"
	default:
		if partner.OnboardingComplete {
			status = "submitted"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"step":               partner.OnboardingStep,
		"status":             status,
		"onboardingComplete": partner.OnboardingComplete,
		"verificationStatus": partner.VerificationStatus,
		"rejectionReason":    partner.RejectionReason,
		"profile": gin.H{
			"city":                partner.City,
			"emergencyContact":    partner.EmergencyContact,
			"emergencyPhone":      partner.EmergencyPhone,
			"vehicleType":         partner.VehicleType,
			"vehicleMake":         partner.VehicleMake,
			"vehicleModel":        partner.VehicleModel,
			"vehicleYear":         partner.VehicleYear,
			"vehicleColor":        partner.VehicleColor,
			"vehicleNumber":       partner.VehicleNumber,
			"licenseNumber":       partner.LicenseNumber,
			"hasDeliveryBoxSpace": partner.HasDeliveryBoxSpace,
			"payoutMethod":        partner.PayoutMethod,
			"bankAccountName":     func() string { v, _ := services.GetDriverSecret(c.Request.Context(), partner.ID.String(), "bank-account-name"); return v }(),
			"bankAccountNumber":   func() string { v, _ := services.GetDriverSecret(c.Request.Context(), partner.ID.String(), "bank-account-number"); return maskBankAccount(v) }(),
			"bankIFSC":            func() string { v, _ := services.GetDriverSecret(c.Request.Context(), partner.ID.String(), "bank-ifsc"); return v }(),
			"upiId":               func() string { v, _ := services.GetDriverSecret(c.Request.Context(), partner.ID.String(), "upi-id"); return maskEmail(v) }(),
		},
		"documentCount": docCount,
		"payoutSet":     payoutSet,
	})
}

// DriverOnboardingPersonal handles Step 1 of driver onboarding (personal details)
// POST /driver/onboarding/personal
func (h *DriverOnboardingHandler) DriverOnboardingPersonal(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		City             string  `json:"city" binding:"required"`
		EmergencyContact string  `json:"emergencyContact" binding:"required"`
		EmergencyPhone   string  `json:"emergencyPhone" binding:"required"`
		DateOfBirth      *string `json:"dateOfBirth"`
		VehicleType      string  `json:"vehicleType" binding:"required"`
		ReferralCode     string  `json:"referralCode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var dob *time.Time
	if req.DateOfBirth != nil && *req.DateOfBirth != "" {
		parsed, err := time.Parse("2006-01-02", *req.DateOfBirth)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date of birth format, use YYYY-MM-DD"})
			return
		}
		dob = &parsed
	}

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var partner models.DeliveryPartner
	isNew := false
	if err := tx.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		// Create new partner
		isNew = true
		partner = models.DeliveryPartner{
			UserID:      userID,
			IsVerified:  false,
			IsActive:    true,
			IsOnline:    false,
			VehicleType: req.VehicleType,
		}
	}

	partner.City = req.City
	partner.EmergencyContact = req.EmergencyContact
	partner.EmergencyPhone = req.EmergencyPhone
	partner.DateOfBirth = dob
	partner.VehicleType = req.VehicleType

	// If previously rejected, reset status so driver can resubmit
	if partner.VerificationStatus == models.VerificationRejected {
		partner.VerificationStatus = models.VerificationPending
		partner.RejectionReason = ""
		partner.OnboardingComplete = false
	}

	if partner.OnboardingStep < 1 {
		partner.OnboardingStep = 1
	}

	// Handle referral code
	if req.ReferralCode != "" {
		var referrer models.DeliveryPartner
		if err := tx.Where("referral_code = ? AND is_verified = ?", req.ReferralCode, true).
			First(&referrer).Error; err == nil {
			// Valid referral - set referredByID and create referral record
			partner.ReferredByID = &referrer.ID

			if isNew {
				// Create partner first so we have an ID for the referral
				if err := tx.Create(&partner).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create delivery partner profile"})
					return
				}
				isNew = false // Already created
			}

			referral := models.DriverReferral{
				ReferrerID:   referrer.ID,
				RefereeID:    partner.ID,
				ReferralCode: req.ReferralCode,
				Status:       models.ReferralPending,
				ExpiresAt:    time.Now().AddDate(0, 3, 0), // Expires in 3 months
			}
			if err := tx.Create(&referral).Error; err != nil {
				log.Printf("Failed to create referral record: %v", err)
			}
		}
	}

	if isNew {
		if err := tx.Create(&partner).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create delivery partner profile"})
			return
		}
	} else {
		if err := tx.Save(&partner).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update delivery partner profile"})
			return
		}
	}

	// Update user role to delivery
	tx.Model(&models.User{}).Where("id = ?", userID).Update("role", models.RoleDelivery)

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save onboarding data"})
		return
	}

	// Reload with associations for response
	database.DB.Preload("User").Preload("Documents").First(&partner, "id = ?", partner.ID)

	c.JSON(http.StatusOK, partner.ToDetailResponse())
}

// DriverOnboardingVehicle handles Step 2 of driver onboarding (vehicle details)
// POST /driver/onboarding/vehicle
func (h *DriverOnboardingHandler) DriverOnboardingVehicle(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		VehicleType         string `json:"vehicleType" binding:"required"`
		VehicleMake         string `json:"vehicleMake"`
		VehicleModel        string `json:"vehicleModel"`
		VehicleYear         int    `json:"vehicleYear"`
		VehicleColor        string `json:"vehicleColor"`
		VehicleNumber       string `json:"vehicleNumber"`
		LicenseNumber       string `json:"licenseNumber"`
		HasDeliveryBoxSpace *bool  `json:"hasDeliveryBoxSpace"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields based on vehicle type
	if req.VehicleType == "bicycle" {
		if req.HasDeliveryBoxSpace == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Please indicate if your bicycle can carry a delivery box"})
			return
		}
	} else {
		if strings.TrimSpace(req.VehicleNumber) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Vehicle registration number is required"})
			return
		}
		if strings.TrimSpace(req.LicenseNumber) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Driving license number is required"})
			return
		}
	}

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Please complete step 1 (personal details) first"})
		return
	}

	partner.VehicleType = req.VehicleType
	partner.VehicleMake = req.VehicleMake
	partner.VehicleModel = req.VehicleModel
	partner.VehicleYear = req.VehicleYear
	partner.VehicleColor = req.VehicleColor

	if req.VehicleType == "bicycle" {
		partner.VehicleNumber = ""
		partner.LicenseNumber = ""
		if req.HasDeliveryBoxSpace != nil {
			partner.HasDeliveryBoxSpace = *req.HasDeliveryBoxSpace
		}
	} else {
		partner.VehicleNumber = req.VehicleNumber
		partner.LicenseNumber = req.LicenseNumber
	}

	if partner.OnboardingStep < 2 {
		partner.OnboardingStep = 2
	}

	if err := database.DB.Save(&partner).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vehicle details"})
		return
	}

	database.DB.Preload("User").Preload("Documents").First(&partner, "id = ?", partner.ID)
	c.JSON(http.StatusOK, partner.ToDetailResponse())
}

// DriverOnboardingPayout handles Step 4 of driver onboarding (payout info)
// Step 3 is documents which uses existing upload endpoints
// POST /driver/onboarding/payout
func (h *DriverOnboardingHandler) DriverOnboardingPayout(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		PayoutMethod      string `json:"payoutMethod" binding:"required"`
		BankAccountNumber string `json:"bankAccountNumber"`
		BankIFSC          string `json:"bankIFSC"`
		BankAccountName   string `json:"bankAccountName"`
		UpiID             string `json:"upiId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate payout method
	if req.PayoutMethod != "bank_transfer" && req.PayoutMethod != "upi" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payoutMethod must be 'bank_transfer' or 'upi'"})
		return
	}

	if req.PayoutMethod == "bank_transfer" {
		if req.BankAccountNumber == "" || req.BankIFSC == "" || req.BankAccountName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bankAccountNumber, bankIFSC, and bankAccountName are required for bank_transfer"})
			return
		}
	}

	if req.PayoutMethod == "upi" {
		if req.UpiID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "upiId is required for upi payout method"})
			return
		}
	}

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Please complete earlier onboarding steps first"})
		return
	}

	driverID := partner.ID.String()

	// DB stores ONLY payout method (non-sensitive). All sensitive data in Secret Manager.
	partner.PayoutMethod = req.PayoutMethod
	partner.BankAccountNumber = ""
	partner.BankIFSC = ""
	partner.BankAccountName = ""
	partner.UpiID = ""

	if partner.OnboardingStep < 4 {
		partner.OnboardingStep = 4
	}

	if err := database.DB.Save(&partner).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save payout information"})
		return
	}

	log.Printf("Payout details saved for driver %s (method: %s)", driverID, req.PayoutMethod)

	// Store sensitive fields in GCP Secret Manager asynchronously
	go func() {
		ctx := context.Background()
		secretFields := map[string]string{
			"bank-account-number": req.BankAccountNumber,
			"bank-account-name":   req.BankAccountName,
			"bank-ifsc":           req.BankIFSC,
			"upi-id":              req.UpiID,
		}
		for field, value := range secretFields {
			if value != "" {
				if err := services.StoreDriverSecret(ctx, driverID, field, value); err != nil {
					log.Printf("Warning: failed to store secret %s for driver %s: %v", field, driverID, err)
				}
			}
		}
		log.Printf("Secrets stored in Secret Manager for driver %s", driverID)
	}()

	database.DB.Preload("User").Preload("Documents").First(&partner, "id = ?", partner.ID)
	c.JSON(http.StatusOK, partner.ToDetailResponse())
}

// DriverOnboardingSubmit handles Step 5 (final submission) of driver onboarding
// POST /driver/onboarding/submit
func (h *DriverOnboardingHandler) DriverOnboardingSubmit(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		TermsAccepted bool `json:"termsAccepted" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !req.TermsAccepted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You must accept the terms and conditions"})
		return
	}

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	// Prevent re-submission if already approved
	if partner.VerificationStatus == models.VerificationApproved {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Your application has already been approved"})
		return
	}

	// Prevent re-submission if already in review
	if partner.VerificationStatus == models.VerificationInReview {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Your application is already under review"})
		return
	}

	if partner.OnboardingStep < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please complete all previous onboarding steps before submitting"})
		return
	}

	// Verify documents have been uploaded
	var docCount int64
	database.DB.Model(&models.DeliveryPartnerDocument{}).
		Where("partner_id = ?", partner.ID).Count(&docCount)
	if docCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please upload at least one document before submitting"})
		return
	}

	now := time.Now()
	partner.TermsAcceptedAt = &now
	partner.OnboardingComplete = true
	partner.VerificationStatus = models.VerificationInReview
	partner.OnboardingStep = 5

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Save(&partner).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit onboarding"})
		return
	}

	// Create approval request
	submittedData, _ := json.Marshal(map[string]interface{}{
		"partnerId":   partner.ID,
		"vehicleType": partner.VehicleType,
		"city":        partner.City,
	})

	approvalReq := models.ApprovalRequest{
		Type:          models.ApprovalDriverOnboarding,
		Status:        models.ApprovalPending,
		Priority:      "high",
		PartnerID:     &partner.ID,
		SubmittedByID: userID,
		EntityType:    "delivery_partner",
		EntityID:      partner.ID,
		Title:         fmt.Sprintf("Driver Onboarding: %s", partner.City),
		Description:   fmt.Sprintf("New driver onboarding submission - %s (%s)", partner.VehicleType, partner.City),
		SubmittedData: string(submittedData),
	}

	if err := tx.Create(&approvalReq).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create approval request"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit onboarding"})
		return
	}

	// Publish NATS event (non-blocking)
	if err := services.PublishEvent(services.SubjectDriverOnboardingSubmitted, "driver.onboarding.submitted", userID, map[string]interface{}{
		"partner_id": partner.ID.String(),
		"city":       partner.City,
	}); err != nil {
		log.Printf("Failed to publish driver onboarding submitted event: %v", err)
	}

	database.DB.Preload("User").Preload("Documents").First(&partner, "id = ?", partner.ID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Onboarding submitted successfully. Your application is now under review.",
		"partner": partner.ToDetailResponse(),
	})
}

// ValidateReferralCode validates a referral code
// POST /driver/referral/validate
func (h *DriverOnboardingHandler) ValidateReferralCode(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var referrer models.DeliveryPartner
	if err := database.DB.Preload("User").
		Where("referral_code = ? AND is_verified = ?", req.Code, true).
		First(&referrer).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid":        false,
			"referrerName": "",
		})
		return
	}

	referrerName := ""
	if referrer.User.ID != uuid.Nil {
		referrerName = referrer.User.FirstName + " " + referrer.User.LastName
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":        true,
		"referrerName": strings.TrimSpace(referrerName),
	})
}

// GetReferralCode returns the referral code for the authenticated delivery partner
// GET /driver/referral/code
func (h *DriverOnboardingHandler) GetReferralCode(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Preload("User").Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	if partner.ReferralCode == "" {
		// Generate referral code: FE3DR-<first 4 chars of name uppercase>-<4 random alphanumeric>
		name := partner.User.FirstName
		if len(name) > 4 {
			name = name[:4]
		}
		name = strings.ToUpper(name)
		// Pad with X if name is shorter than 4 chars
		for len(name) < 4 {
			name += "X"
		}

		randomPart := generateRandomAlphanumeric(4)
		partner.ReferralCode = fmt.Sprintf("FE3DR-%s-%s", name, randomPart)

		if err := database.DB.Save(&partner).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate referral code"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"referralCode": partner.ReferralCode,
	})
}

// GetReferralStats returns referral statistics for the authenticated delivery partner
// GET /driver/referral/stats
func (h *DriverOnboardingHandler) GetReferralStats(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	var totalReferrals int64
	var completedReferrals int64
	var pendingReferrals int64
	var totalBonus float64

	database.DB.Model(&models.DriverReferral{}).
		Where("referrer_id = ?", partner.ID).
		Count(&totalReferrals)

	database.DB.Model(&models.DriverReferral{}).
		Where("referrer_id = ? AND status = ?", partner.ID, models.ReferralCompleted).
		Count(&completedReferrals)

	database.DB.Model(&models.DriverReferral{}).
		Where("referrer_id = ? AND status = ?", partner.ID, models.ReferralPending).
		Count(&pendingReferrals)

	database.DB.Model(&models.DriverReferral{}).
		Where("referrer_id = ? AND status IN ?", partner.ID, []models.ReferralStatus{models.ReferralCompleted, models.ReferralPaid}).
		Select("COALESCE(SUM(bonus_amount), 0)").
		Scan(&totalBonus)

	c.JSON(http.StatusOK, gin.H{
		"totalReferrals":     totalReferrals,
		"completedReferrals": completedReferrals,
		"pendingReferrals":   pendingReferrals,
		"totalBonusEarned":   totalBonus,
		"referralCode":       partner.ReferralCode,
	})
}

// generateRandomAlphanumeric generates a random alphanumeric string of the given length
func generateRandomAlphanumeric(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// Fallback to a simple default if crypto/rand fails
			result[i] = charset[i%len(charset)]
			continue
		}
		result[i] = charset[n.Int64()]
	}
	return string(result)
}
