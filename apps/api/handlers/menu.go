package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type MenuHandler struct{}

func NewMenuHandler() *MenuHandler {
	return &MenuHandler{}
}

// ---------- Menu Items ----------

// GetChefMenuItems returns all menu items for the authenticated chef.
// GET /chef/menu
func (h *MenuHandler) GetChefMenuItems(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		// No profile yet = no menu items
		c.JSON(http.StatusOK, []models.MenuItem{})
		return
	}

	var items []models.MenuItem
	database.DB.Where("chef_id = ?", chef.ID).Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Order("sort_order ASC, created_at DESC").Find(&items)

	// Ensure nil slices are returned as empty arrays in JSON
	for i := range items {
		if items[i].DietaryTags == nil {
			items[i].DietaryTags = pq.StringArray{}
		}
		if items[i].Allergens == nil {
			items[i].Allergens = pq.StringArray{}
		}
		if items[i].Ingredients == nil {
			items[i].Ingredients = pq.StringArray{}
		}
		if items[i].Images == nil {
			items[i].Images = []models.MenuItemImage{}
		}
	}

	c.JSON(http.StatusOK, items)
}

// GetMenuItem returns a single menu item by ID (must belong to authenticated chef).
// GET /chef/menu/items/:itemId
func (h *MenuHandler) GetMenuItem(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	itemID := c.Param("itemId")

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var item models.MenuItem
	if err := database.DB.Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Where("id = ? AND chef_id = ?", itemID, chef.ID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
		return
	}

	if item.DietaryTags == nil {
		item.DietaryTags = pq.StringArray{}
	}
	if item.Allergens == nil {
		item.Allergens = pq.StringArray{}
	}
	if item.Ingredients == nil {
		item.Ingredients = pq.StringArray{}
	}
	if item.Images == nil {
		item.Images = []models.MenuItemImage{}
	}

	c.JSON(http.StatusOK, item)
}

// CreateMenuItem creates a new menu item for the authenticated chef.
// POST /chef/menu/items
func (h *MenuHandler) CreateMenuItem(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	chef, err := getOrCreateChefProfile(userID)
	if err != nil {
		log.Printf("Failed to get/create chef profile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize chef profile"})
		return
	}

	var req CreateMenuItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item := models.MenuItem{
		ChefID:       chef.ID,
		Name:         req.Name,
		Description:  req.Description,
		Price:        req.Price,
		ComparePrice: req.ComparePrice,
		ImageURL:     req.ImageURL,
		DietaryTags:  ensureStringArray(req.DietaryTags),
		Allergens:    ensureStringArray(req.Allergens),
		Ingredients:  ensureStringArray(req.Ingredients),
		PrepTime:     req.PrepTime,
		PortionSize:  req.PortionSize,
		Serves:       max(req.Serves, 1),
		SpiceLevel:   req.SpiceLevel,
		IsAvailable:  true,
		IsFeatured:   req.IsFeatured,
	}

	if req.CategoryID != "" {
		catID, err := uuid.Parse(req.CategoryID)
		if err == nil {
			item.CategoryID = &catID
		}
	}

	if err := database.DB.Create(&item).Error; err != nil {
		log.Printf("Failed to create menu item: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create menu item"})
		return
	}

	// Create approval request for admin review
	submittedData, _ := json.Marshal(map[string]interface{}{
		"name": item.Name, "description": item.Description,
		"price": item.Price, "comparePrice": item.ComparePrice,
		"dietaryTags": item.DietaryTags, "allergens": item.Allergens,
		"prepTime": item.PrepTime, "portionSize": item.PortionSize,
		"serves": item.Serves, "isFeatured": item.IsFeatured,
	})
	approvalReq := models.ApprovalRequest{
		Type:          models.ApprovalMenuItemNew,
		Status:        models.ApprovalPending,
		Priority:      "normal",
		ChefID:        &chef.ID,
		SubmittedByID: userID,
		EntityType:    "menu_item",
		EntityID:      item.ID,
		Title:         fmt.Sprintf("New Menu Item: %s", item.Name),
		Description:   fmt.Sprintf("New item ₹%.0f - %s", item.Price, item.Name),
		SubmittedData: string(submittedData),
	}
	database.DB.Create(&approvalReq)
	services.PublishEvent(services.SubjectApprovalCreated, "approval.created", userID, map[string]interface{}{
		"approval_id": approvalReq.ID.String(), "type": string(approvalReq.Type),
		"chef_id": chef.ID.String(), "title": approvalReq.Title,
	})

	c.JSON(http.StatusCreated, item)
}

// UpdateMenuItem updates an existing menu item.
// PUT /chef/menu/items/:itemId
func (h *MenuHandler) UpdateMenuItem(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	itemID := c.Param("itemId")

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var item models.MenuItem
	if err := database.DB.Where("id = ? AND chef_id = ?", itemID, chef.ID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
		return
	}

	var req UpdateMenuItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build updates map — only update fields that are present
	updates := map[string]interface{}{}

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Price != nil {
		updates["price"] = *req.Price
	}
	if req.ComparePrice != nil {
		updates["compare_price"] = *req.ComparePrice
	}
	if req.ImageURL != nil {
		updates["image_url"] = *req.ImageURL
	}
	if req.DietaryTags != nil {
		updates["dietary_tags"] = pq.StringArray(*req.DietaryTags)
	}
	if req.Allergens != nil {
		updates["allergens"] = pq.StringArray(*req.Allergens)
	}
	if req.Ingredients != nil {
		updates["ingredients"] = pq.StringArray(*req.Ingredients)
	}
	if req.PrepTime != nil {
		updates["prep_time"] = *req.PrepTime
	}
	if req.PortionSize != nil {
		updates["portion_size"] = *req.PortionSize
	}
	if req.Serves != nil {
		updates["serves"] = *req.Serves
	}
	if req.SpiceLevel != nil {
		updates["spice_level"] = *req.SpiceLevel
	}
	if req.IsAvailable != nil {
		updates["is_available"] = *req.IsAvailable
	}
	if req.IsFeatured != nil {
		updates["is_featured"] = *req.IsFeatured
	}
	if req.CategoryID != nil {
		if *req.CategoryID == "" {
			updates["category_id"] = nil
		} else {
			catID, err := uuid.Parse(*req.CategoryID)
			if err == nil {
				updates["category_id"] = catID
			}
		}
	}

	// Track if price changed for approval
	oldPrice := item.Price
	priceChanged := false
	if req.Price != nil && *req.Price != oldPrice {
		priceChanged = true
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&item).Updates(updates).Error; err != nil {
			log.Printf("Failed to update menu item: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update menu item"})
			return
		}
	}

	// Create approval request for significant changes (price, name)
	if priceChanged {
		submittedData, _ := json.Marshal(map[string]interface{}{
			"itemName": item.Name, "currentPrice": oldPrice,
			"newPrice": *req.Price, "reason": "Price updated by chef",
		})
		approvalReq := models.ApprovalRequest{
			Type:          models.ApprovalPricingChange,
			Status:        models.ApprovalPending,
			Priority:      "normal",
			ChefID:        &chef.ID,
			SubmittedByID: userID,
			EntityType:    "menu_item",
			EntityID:      item.ID,
			Title:         fmt.Sprintf("Price Change: %s", item.Name),
			Description:   fmt.Sprintf("₹%.0f → ₹%.0f for %s", oldPrice, *req.Price, item.Name),
			SubmittedData: string(submittedData),
		}
		database.DB.Create(&approvalReq)
		services.PublishEvent(services.SubjectApprovalCreated, "approval.created", userID, map[string]interface{}{
			"approval_id": approvalReq.ID.String(), "type": string(approvalReq.Type),
			"chef_id": chef.ID.String(), "title": approvalReq.Title,
		})
	}

	// Reload to return the updated item
	database.DB.First(&item, "id = ?", itemID)

	if item.DietaryTags == nil {
		item.DietaryTags = pq.StringArray{}
	}
	if item.Allergens == nil {
		item.Allergens = pq.StringArray{}
	}
	if item.Ingredients == nil {
		item.Ingredients = pq.StringArray{}
	}

	c.JSON(http.StatusOK, item)
}

// DeleteMenuItem soft-deletes a menu item.
// DELETE /chef/menu/items/:itemId
func (h *MenuHandler) DeleteMenuItem(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	itemID := c.Param("itemId")

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	result := database.DB.Where("id = ? AND chef_id = ?", itemID, chef.ID).Delete(&models.MenuItem{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Menu item deleted"})
}

// ---------- Categories ----------

// GetCategories returns all categories for the authenticated chef.
// GET /chef/menu/categories
func (h *MenuHandler) GetCategories(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		// No profile yet = no categories
		c.JSON(http.StatusOK, []models.MenuCategory{})
		return
	}

	var categories []models.MenuCategory
	database.DB.Where("chef_id = ?", chef.ID).Order("sort_order ASC, name ASC").Find(&categories)

	c.JSON(http.StatusOK, categories)
}

// CreateCategory creates a new menu category.
// POST /chef/menu/categories
func (h *MenuHandler) CreateCategory(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	chef, err := getOrCreateChefProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize chef profile"})
		return
	}

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Case-insensitive duplicate check
	var existing models.MenuCategory
	if err := database.DB.Where("chef_id = ? AND LOWER(name) = LOWER(?)", chef.ID, req.Name).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "A category with this name already exists"})
		return
	}

	category := models.MenuCategory{
		ChefID:      chef.ID,
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		IsActive:    true,
	}

	if err := database.DB.Create(&category).Error; err != nil {
		log.Printf("Failed to create category: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, category)
}

// UpdateCategory updates an existing category.
// PUT /chef/menu/categories/:categoryId
func (h *MenuHandler) UpdateCategory(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	categoryID := c.Param("categoryId")

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var category models.MenuCategory
	if err := database.DB.Where("id = ? AND chef_id = ?", categoryID, chef.ID).First(&category).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		// Case-insensitive duplicate check (exclude self)
		var existing models.MenuCategory
		if err := database.DB.Where("chef_id = ? AND LOWER(name) = LOWER(?) AND id != ?", chef.ID, *req.Name, category.ID).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "A category with this name already exists"})
			return
		}
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) > 0 {
		database.DB.Model(&category).Updates(updates)
	}

	database.DB.First(&category, "id = ?", categoryID)
	c.JSON(http.StatusOK, category)
}

// DeleteCategory soft-deletes a category and nullifies categoryId on its items.
// DELETE /chef/menu/categories/:categoryId
func (h *MenuHandler) DeleteCategory(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	categoryID := c.Param("categoryId")

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	result := database.DB.Where("id = ? AND chef_id = ?", categoryID, chef.ID).Delete(&models.MenuCategory{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Nullify category_id on items that used this category
	database.DB.Model(&models.MenuItem{}).Where("category_id = ? AND chef_id = ?", categoryID, chef.ID).Update("category_id", nil)

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted"})
}

// ---------- Menu Item Images ----------

// UploadMenuItemImage uploads an image for a menu item to GCS (public bucket).
// POST /chef/menu/items/:itemId/images — multipart/form-data with field: file
// Max 5 images per item.
func (h *MenuHandler) UploadMenuItemImage(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	itemID := c.Param("itemId")

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var item models.MenuItem
	if err := database.DB.Where("id = ? AND chef_id = ?", itemID, chef.ID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
		return
	}

	// Check existing image count (max 5)
	var imageCount int64
	database.DB.Model(&models.MenuItemImage{}).Where("menu_item_id = ?", item.ID).Count(&imageCount)
	if imageCount >= 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 5 images per menu item"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	// Validate file size (5MB max)
	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum 5 MB."})
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !services.IsImageContentType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Allowed: JPEG, PNG, WebP."})
		return
	}

	// Upload to public bucket: chefs/{chefID}/menu/{itemID}/{uuid}.{ext}
	folder := fmt.Sprintf("chefs/%s/menu/%s", chef.ID.String(), item.ID.String())
	fileURL, err := services.UploadPublicFile(c.Request.Context(), folder, header.Filename, file, contentType)
	if err != nil {
		log.Printf("Failed to upload menu item image: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
		return
	}

	// Determine sort order (append to end)
	isPrimary := imageCount == 0

	img := models.MenuItemImage{
		MenuItemID: item.ID,
		URL:        fileURL,
		IsPrimary:  isPrimary,
		SortOrder:  int(imageCount),
	}
	if err := database.DB.Create(&img).Error; err != nil {
		log.Printf("Failed to save menu item image: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
		return
	}

	// Also set the first image as the item's main imageUrl
	if isPrimary {
		database.DB.Model(&item).Update("image_url", fileURL)
	}

	c.JSON(http.StatusCreated, img)
}

// DeleteMenuItemImage deletes a menu item image from GCS and the database.
// DELETE /chef/menu/items/:itemId/images/:imageId
func (h *MenuHandler) DeleteMenuItemImage(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	itemID := c.Param("itemId")
	imageID := c.Param("imageId")

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	// Verify the item belongs to this chef
	var item models.MenuItem
	if err := database.DB.Where("id = ? AND chef_id = ?", itemID, chef.ID).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Menu item not found"})
		return
	}

	var img models.MenuItemImage
	if err := database.DB.Where("id = ? AND menu_item_id = ?", imageID, item.ID).First(&img).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// Delete from GCS (best effort — don't fail if GCS delete fails)
	bucket := config.AppConfig.GCSPublicBucket
	// Extract object path from full URL: https://storage.googleapis.com/{bucket}/{path}
	prefix := fmt.Sprintf("https://storage.googleapis.com/%s/", bucket)
	if objectPath := img.URL; len(objectPath) > len(prefix) {
		objectPath = objectPath[len(prefix):]
		if err := services.DeleteFile(c.Request.Context(), bucket, objectPath); err != nil {
			log.Printf("Warning: failed to delete GCS object %s: %v", objectPath, err)
		}
	}

	database.DB.Delete(&img)

	// If deleted image was primary, promote next one
	if img.IsPrimary {
		var nextImg models.MenuItemImage
		if err := database.DB.Where("menu_item_id = ?", item.ID).Order("sort_order ASC").First(&nextImg).Error; err == nil {
			database.DB.Model(&nextImg).Update("is_primary", true)
			database.DB.Model(&item).Update("image_url", nextImg.URL)
		} else {
			// No images left
			database.DB.Model(&item).Update("image_url", "")
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Image deleted"})
}

// ---------- Request types ----------

type CreateMenuItemRequest struct {
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description"`
	Price        float64  `json:"price" binding:"required,gt=0"`
	ComparePrice float64  `json:"comparePrice"`
	CategoryID   string   `json:"categoryId"`
	ImageURL     string   `json:"imageUrl"`
	DietaryTags  []string `json:"dietaryTags"`
	Allergens    []string `json:"allergens"`
	Ingredients  []string `json:"ingredients"`
	PrepTime     int      `json:"prepTime"`
	PortionSize  string   `json:"portionSize"`
	Serves       int      `json:"serves"`
	SpiceLevel   int      `json:"spiceLevel"`
	IsFeatured   bool     `json:"isFeatured"`
}

type UpdateMenuItemRequest struct {
	Name         *string   `json:"name"`
	Description  *string   `json:"description"`
	Price        *float64  `json:"price"`
	ComparePrice *float64  `json:"comparePrice"`
	CategoryID   *string   `json:"categoryId"`
	ImageURL     *string   `json:"imageUrl"`
	DietaryTags  *[]string `json:"dietaryTags"`
	Allergens    *[]string `json:"allergens"`
	Ingredients  *[]string `json:"ingredients"`
	PrepTime     *int      `json:"prepTime"`
	PortionSize  *string   `json:"portionSize"`
	Serves       *int      `json:"serves"`
	SpiceLevel   *int      `json:"spiceLevel"`
	IsAvailable  *bool     `json:"isAvailable"`
	IsFeatured   *bool     `json:"isFeatured"`
}

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
}

type UpdateCategoryRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sortOrder"`
	IsActive    *bool   `json:"isActive"`
}

// ---------- Helpers ----------

func ensureStringArray(arr []string) pq.StringArray {
	if arr == nil {
		return pq.StringArray{}
	}
	return pq.StringArray(arr)
}
