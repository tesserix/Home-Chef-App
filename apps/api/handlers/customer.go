package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"github.com/lib/pq"
)

type PreferenceHandler struct{}

func NewPreferenceHandler() *PreferenceHandler {
	return &PreferenceHandler{}
}

// GetPreferenceOptions returns all active preference options grouped by category.
func (h *PreferenceHandler) GetPreferenceOptions(c *gin.Context) {
	category := c.Query("category")

	var options []models.PreferenceOption
	query := database.DB.Where("is_active = ?", true).Order("category, sort_order")
	if category != "" {
		query = query.Where("category = ?", category)
	}
	query.Find(&options)

	// Group by category
	grouped := map[string][]models.PreferenceOption{}
	for _, opt := range options {
		grouped[opt.Category] = append(grouped[opt.Category], opt)
	}

	c.JSON(http.StatusOK, grouped)
}

type CustomerHandler struct{}

func NewCustomerHandler() *CustomerHandler {
	return &CustomerHandler{}
}

// GetCustomerProfile returns the full customer profile, lazy-creating a CustomerProfile row if needed.
func (h *CustomerHandler) GetCustomerProfile(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var profile models.CustomerProfile
	result := database.DB.Where("user_id = ?", userID).First(&profile)
	if result.Error != nil {
		// Lazy-create
		profile = models.CustomerProfile{
			UserID: userID,
		}
		if err := database.DB.Create(&profile).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create customer profile"})
			return
		}
	}

	c.JSON(http.StatusOK, profile.ToResponse(&user))
}

// UpdateCustomerProfile updates user + customer profile fields.
func (h *CustomerHandler) UpdateCustomerProfile(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		FirstName          *string  `json:"firstName"`
		LastName           *string  `json:"lastName"`
		Phone              *string  `json:"phone"`
		DateOfBirth        *string  `json:"dateOfBirth"`
		DietaryPreferences []string `json:"dietaryPreferences"`
		FoodAllergies      []string `json:"foodAllergies"`
		CuisinePreferences []string `json:"cuisinePreferences"`
		SpiceTolerance     *string  `json:"spiceTolerance"`
		HouseholdSize      *string  `json:"householdSize"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update user fields
	userUpdates := map[string]interface{}{}
	if req.FirstName != nil {
		userUpdates["first_name"] = *req.FirstName
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		userUpdates["last_name"] = *req.LastName
		user.LastName = *req.LastName
	}
	if req.Phone != nil {
		userUpdates["phone"] = *req.Phone
		user.Phone = *req.Phone
	}
	if len(userUpdates) > 0 {
		database.DB.Model(&user).Updates(userUpdates)
	}

	// Get or create customer profile
	var profile models.CustomerProfile
	result := database.DB.Where("user_id = ?", userID).First(&profile)
	if result.Error != nil {
		profile = models.CustomerProfile{UserID: userID}
		database.DB.Create(&profile)
	}

	// Update profile fields
	profileUpdates := map[string]interface{}{}
	if req.DateOfBirth != nil {
		if *req.DateOfBirth == "" {
			profileUpdates["date_of_birth"] = nil
			profile.DateOfBirth = nil
		} else {
			t, err := time.Parse("2006-01-02", *req.DateOfBirth)
			if err == nil {
				profileUpdates["date_of_birth"] = t
				profile.DateOfBirth = &t
			}
		}
	}
	if req.DietaryPreferences != nil {
		profileUpdates["dietary_preferences"] = pq.StringArray(req.DietaryPreferences)
		profile.DietaryPreferences = req.DietaryPreferences
	}
	if req.FoodAllergies != nil {
		profileUpdates["food_allergies"] = pq.StringArray(req.FoodAllergies)
		profile.FoodAllergies = req.FoodAllergies
	}
	if req.CuisinePreferences != nil {
		profileUpdates["cuisine_preferences"] = pq.StringArray(req.CuisinePreferences)
		profile.CuisinePreferences = req.CuisinePreferences
	}
	if req.SpiceTolerance != nil {
		profileUpdates["spice_tolerance"] = *req.SpiceTolerance
		profile.SpiceTolerance = *req.SpiceTolerance
	}
	if req.HouseholdSize != nil {
		profileUpdates["household_size"] = *req.HouseholdSize
		profile.HouseholdSize = *req.HouseholdSize
	}
	if len(profileUpdates) > 0 {
		database.DB.Model(&profile).Updates(profileUpdates)
	}

	c.JSON(http.StatusOK, profile.ToResponse(&user))
}

// GetOnboardingStatus returns lightweight onboarding status.
func (h *CustomerHandler) GetOnboardingStatus(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var profile models.CustomerProfile
	result := database.DB.Where("user_id = ?", userID).First(&profile)
	if result.Error != nil {
		// No profile yet = not completed
		c.JSON(http.StatusOK, gin.H{
			"onboardingCompleted": false,
			"onboardingStep":     0,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"onboardingCompleted": profile.OnboardingCompleted,
		"onboardingStep":      profile.OnboardingStep,
	})
}

// CompleteOnboarding saves profile data and marks onboarding as complete.
func (h *CustomerHandler) CompleteOnboarding(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		FirstName          string   `json:"firstName"`
		LastName           string   `json:"lastName"`
		Phone              string   `json:"phone"`
		DateOfBirth        string   `json:"dateOfBirth"`
		DietaryPreferences []string `json:"dietaryPreferences"`
		FoodAllergies      []string `json:"foodAllergies"`
		CuisinePreferences []string `json:"cuisinePreferences"`
		SpiceTolerance     string   `json:"spiceTolerance"`
		HouseholdSize      string   `json:"householdSize"`
		// Address fields
		AddressLabel      string  `json:"addressLabel"`
		AddressLine1      string  `json:"addressLine1"`
		AddressLine2      string  `json:"addressLine2"`
		AddressCity       string  `json:"addressCity"`
		AddressState      string  `json:"addressState"`
		AddressPostalCode string  `json:"addressPostalCode"`
		AddressCountry    string  `json:"addressCountry"`
		AddressLatitude   float64 `json:"addressLatitude"`
		AddressLongitude  float64 `json:"addressLongitude"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update user fields if provided
	userUpdates := map[string]interface{}{}
	if req.FirstName != "" {
		userUpdates["first_name"] = req.FirstName
	}
	if req.LastName != "" {
		userUpdates["last_name"] = req.LastName
	}
	if req.Phone != "" {
		userUpdates["phone"] = req.Phone
	}
	if len(userUpdates) > 0 {
		database.DB.Model(&user).Updates(userUpdates)
	}

	// Get or create customer profile
	var profile models.CustomerProfile
	result := database.DB.Where("user_id = ?", userID).First(&profile)
	if result.Error != nil {
		profile = models.CustomerProfile{UserID: userID}
		database.DB.Create(&profile)
	}

	// Update profile
	profileUpdates := map[string]interface{}{
		"onboarding_completed": true,
		"onboarding_step":      3,
	}
	if req.DateOfBirth != "" {
		if t, err := time.Parse("2006-01-02", req.DateOfBirth); err == nil {
			profileUpdates["date_of_birth"] = t
		}
	}
	if req.DietaryPreferences != nil {
		profileUpdates["dietary_preferences"] = pq.StringArray(req.DietaryPreferences)
	}
	if req.FoodAllergies != nil {
		profileUpdates["food_allergies"] = pq.StringArray(req.FoodAllergies)
	}
	if req.CuisinePreferences != nil {
		profileUpdates["cuisine_preferences"] = pq.StringArray(req.CuisinePreferences)
	}
	if req.SpiceTolerance != "" {
		profileUpdates["spice_tolerance"] = req.SpiceTolerance
	}
	if req.HouseholdSize != "" {
		profileUpdates["household_size"] = req.HouseholdSize
	}
	database.DB.Model(&profile).Updates(profileUpdates)

	// Create address if provided
	if req.AddressLine1 != "" {
		country := req.AddressCountry
		if country == "" {
			country = "IN"
		}
		label := req.AddressLabel
		if label == "" {
			label = "Home"
		}
		address := models.Address{
			UserID:     userID,
			Label:      label,
			Line1:      req.AddressLine1,
			Line2:      req.AddressLine2,
			City:       req.AddressCity,
			State:      req.AddressState,
			PostalCode: req.AddressPostalCode,
			Country:    country,
			Latitude:   req.AddressLatitude,
			Longitude:  req.AddressLongitude,
			IsDefault:  true,
		}
		database.DB.Create(&address)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             "Onboarding completed",
		"onboardingCompleted": true,
	})
}

// SkipOnboarding marks onboarding as complete without saving data.
// POST /customer/onboarding/skip — no request body required
func (h *CustomerHandler) SkipOnboarding(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var profile models.CustomerProfile
	result := database.DB.Where("user_id = ?", userID).First(&profile)
	if result.Error != nil {
		profile = models.CustomerProfile{
			UserID:              userID,
			OnboardingCompleted: true,
			OnboardingStep:      3,
		}
		if err := database.DB.Create(&profile).Error; err != nil {
			log.Printf("Failed to create customer profile on skip: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to skip onboarding"})
			return
		}
	} else {
		database.DB.Model(&profile).Updates(map[string]interface{}{
			"onboarding_completed": true,
			"onboarding_step":      3,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             "Onboarding skipped",
		"onboardingCompleted": true,
	})
}

// UploadAvatar handles customer profile picture uploads.
// POST /customer/avatar — multipart/form-data with field: file
func (h *CustomerHandler) UploadAvatar(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum 5 MB."})
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !services.IsImageContentType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Allowed: JPEG, PNG, WebP."})
		return
	}

	folder := fmt.Sprintf("customers/%s/avatar", userID.String())
	fileURL, err := services.UploadPublicFile(c.Request.Context(), folder, header.Filename, file, contentType)
	if err != nil {
		log.Printf("Failed to upload customer avatar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	database.DB.Model(&models.User{}).Where("id = ?", userID).Update("avatar", fileURL)

	c.JSON(http.StatusOK, gin.H{"url": fileURL})
}
