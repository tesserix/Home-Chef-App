package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// maxMenuItemImages caps the gallery per dish. Enforced under a row lock in
// UploadMenuItemImage — a plain count-then-insert races when the picker
// uploads a multi-selection concurrently.
const maxMenuItemImages = 5

// errMenuItemImageLimit is a sentinel so the transaction can reject a late
// arrival and the caller can distinguish "full" from a real failure.
var errMenuItemImageLimit = errors.New("menu item image limit reached")

type MenuHandler struct{}

func NewMenuHandler() *MenuHandler {
	return &MenuHandler{}
}

// ---------- Menu Items ----------

// GetChefMenuItems returns all menu items + categories for the authenticated
// chef. GET /chef/menu
//
// Response shape: {"items": [...], "categories": [...]} — the mobile app
// reads both arrays from the same call to render the menu list and the
// category picker without a second round-trip.
func (h *MenuHandler) GetChefMenuItems(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	emptyResponse := gin.H{
		"items":      []models.MenuItem{},
		"categories": []models.MenuCategory{},
	}

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		// No profile yet = no menu items, no categories.
		c.JSON(http.StatusOK, emptyResponse)
		return
	}

	var items []models.MenuItem
	database.DB.Where("chef_id = ?", chef.ID).
		Preload("Images", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order ASC") }).
		Preload("ModifierGroups", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order") }).
		Preload("ModifierGroups.Options", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order") }).
		Preload("ComboItems", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order") }).
		Order("sort_order ASC, created_at DESC").Find(&items)

	// Ensure nil slices are returned as empty arrays in JSON
	capDay := services.CapacityDay(time.Now())
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
		// Derive today's remaining/sold-out from the capacity counter (#48).
		if rem, soldOut := services.RemainingToday(items[i].ID, items[i].DailyCapacity, capDay); rem != nil {
			items[i].RemainingToday = rem
			items[i].SoldOut = soldOut
		}
	}

	var categories []models.MenuCategory
	database.DB.Where("chef_id = ? AND is_active = true", chef.ID).
		Order("sort_order ASC, name ASC").Find(&categories)
	if categories == nil {
		categories = []models.MenuCategory{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":      items,
		"categories": categories,
	})
}

type setCapacityRequest struct {
	DailyCapacity *int `json:"dailyCapacity"` // nil or <= 0 = unlimited
}

// SetMenuItemCapacity — PUT /chef/menu/items/:itemId/capacity (#48). Owner-scoped.
func (h *MenuHandler) SetMenuItemCapacity(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item id"})
		return
	}
	var req setCapacityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.DailyCapacity != nil && *req.DailyCapacity < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dailyCapacity cannot be negative (use 0 or null for unlimited)"})
		return
	}
	// Owner-scoped update → 404 (not 403) if the item isn't this chef's.
	res := database.DB.Model(&models.MenuItem{}).
		Where("id = ? AND chef_id = ?", itemID, chef.ID).
		Updates(map[string]interface{}{"daily_capacity": req.DailyCapacity})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update capacity"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": true, "dailyCapacity": req.DailyCapacity})
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
	if err := database.DB.
		Preload("Images", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order ASC") }).
		Preload("ModifierGroups", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order") }).
		Preload("ModifierGroups.Options", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order") }).
		Preload("ComboItems", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order") }).
		Where("id = ? AND chef_id = ?", itemID, chef.ID).First(&item).Error; err != nil {
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
		ChefID:        chef.ID,
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		ComparePrice:  req.ComparePrice,
		ImageURL:      req.ImageURL,
		DietaryTags:   ensureStringArray(req.DietaryTags),
		Allergens:     ensureStringArray(req.Allergens),
		Ingredients:   ensureStringArray(req.Ingredients),
		PrepTime:      req.PrepTime,
		PortionSize:   req.PortionSize,
		Serves:        max(req.Serves, 1),
		SpiceLevel:    req.SpiceLevel,
		IsVeg:         req.IsVeg,
		IsAvailable:   true,
		AvailableDays: sanitizeWeekdays(req.AvailableDays),
		IsFeatured:    req.IsFeatured,
		HSN:           req.HSN,
		IsCombo:       req.IsCombo,
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

	// Save add-on groups + combo components (#52, replace-all).
	if err := saveItemModifiers(item.ID, chef.ID, req.ModifierGroups, req.ComboItems); err != nil {
		log.Printf("Failed to save item modifiers/combo: %v", err)
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
	if err := services.EnqueueEvent(database.DB, services.SubjectApprovalCreated, "approval.created", userID, map[string]interface{}{
		"approval_id": approvalReq.ID.String(), "type": string(approvalReq.Type),
		"chef_id": chef.ID.String(), "title": approvalReq.Title,
	}); err != nil {
		log.Printf("failed to enqueue approval.created event: %v", err)
	}

	c.JSON(http.StatusCreated, gin.H{"item": item})
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
	if req.IsVeg != nil {
		updates["is_veg"] = *req.IsVeg
	}
	if req.IsAvailable != nil {
		updates["is_available"] = *req.IsAvailable
	}
	if req.IsFeatured != nil {
		updates["is_featured"] = *req.IsFeatured
	}
	if req.AvailableDays != nil {
		updates["available_days"] = pq.Int64Array(sanitizeWeekdays(*req.AvailableDays))
	}
	if req.HSN != nil {
		updates["hsn"] = *req.HSN
	}
	if req.IsCombo != nil {
		updates["is_combo"] = *req.IsCombo
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

	// Replace add-on groups + combo components when the chef edited them (#52).
	if req.ModifierGroups != nil || req.ComboItems != nil {
		groups := []ModifierGroupInput{}
		if req.ModifierGroups != nil {
			groups = *req.ModifierGroups
		}
		combos := []ComboItemInput{}
		if req.ComboItems != nil {
			combos = *req.ComboItems
		}
		if err := saveItemModifiers(item.ID, chef.ID, groups, combos); err != nil {
			log.Printf("Failed to save item modifiers/combo: %v", err)
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
		if err := services.EnqueueEvent(database.DB, services.SubjectApprovalCreated, "approval.created", userID, map[string]interface{}{
			"approval_id": approvalReq.ID.String(), "type": string(approvalReq.Type),
			"chef_id": chef.ID.String(), "title": approvalReq.Title,
		}); err != nil {
			log.Printf("failed to enqueue approval.created event: %v", err)
		}
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

	c.JSON(http.StatusOK, gin.H{"item": item})
}

// ToggleMenuItemAvailability marks a menu item as available or out-of-stock.
// The chef must own the item.
// PUT /chef/menu/items/:itemId/availability — body: {"isAvailable": bool}
func (h *MenuHandler) ToggleMenuItemAvailability(c *gin.Context) {
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

	var req struct {
		IsAvailable bool `json:"isAvailable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Model(&item).Update("is_available", req.IsAvailable).Error; err != nil {
		log.Printf("Failed to update item availability: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update availability"})
		return
	}

	// Reload to return the up-to-date record
	database.DB.Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).First(&item, "id = ?", itemID)

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

	// The declared multipart Content-Type is client-controlled, so also sniff
	// the real bytes — the same guard every upload.go path already applies.
	// This one lands in the PUBLIC bucket, so an SVG or HTML payload wearing an
	// image/jpeg header would be served back to browsers verbatim.
	sniffed, serr := sniffContentType(file)
	if serr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not read file"})
		return
	}
	if !services.IsImageContentType(sniffed) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File contents don't match an allowed image type."})
		return
	}

	// Cheap pre-check so an obviously-full item doesn't pay for a GCS upload.
	// It is NOT the enforcement point — the transaction below is.
	var preCount int64
	database.DB.Model(&models.MenuItemImage{}).Where("menu_item_id = ?", item.ID).Count(&preCount)
	if preCount >= maxMenuItemImages {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 5 images per menu item"})
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

	img, txErr := claimMenuItemImageSlot(database.DB, item.ID, fileURL)

	if errors.Is(txErr, errMenuItemImageLimit) {
		// The object already landed in GCS; drop it rather than leave an
		// orphan nothing references. Best effort, same as DeleteMenuItemImage.
		bucket := config.AppConfig.GCSPublicBucket
		prefix := fmt.Sprintf("https://storage.googleapis.com/%s/", bucket)
		if objectPath := fileURL; len(objectPath) > len(prefix) {
			objectPath = objectPath[len(prefix):]
			if delErr := services.DeleteFile(c.Request.Context(), bucket, objectPath); delErr != nil {
				log.Printf("Warning: orphaned menu image %s after limit hit: %v", objectPath, delErr)
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 5 images per menu item"})
		return
	}
	if txErr != nil {
		log.Printf("Failed to save menu item image: %v", txErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
		return
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
	// IsVeg is nullable: send true/false to set, omit/null to leave unset.
	IsVeg      *bool `json:"isVeg"`
	IsFeatured bool  `json:"isFeatured"`
	// HSN/SAC code for GST classification. Optional — empty string
	// causes the DB default ("996331", restaurant services) to apply.
	HSN string `json:"hsn"`
	// AvailableDays is the weekly-menu schedule (0=Sun..6=Sat). Empty/omitted =
	// available every day. Invalid values are dropped server-side.
	AvailableDays []int `json:"availableDays"`
	// Add-ons / combos (#52). ModifierGroups + ComboItems replace-all on save.
	IsCombo        bool                 `json:"isCombo"`
	ModifierGroups []ModifierGroupInput `json:"modifierGroups"`
	ComboItems     []ComboItemInput     `json:"comboItems"`
}

// ModifierGroupInput / ModifierOptionInput / ComboItemInput are the nested save
// shapes a chef sends with a menu item (#52).
type ModifierGroupInput struct {
	Name      string                `json:"name"`
	Required  bool                  `json:"required"`
	MinSelect int                   `json:"minSelect"`
	MaxSelect int                   `json:"maxSelect"`
	Options   []ModifierOptionInput `json:"options"`
}

type ModifierOptionInput struct {
	Name        string  `json:"name"`
	PriceDelta  float64 `json:"priceDelta"`
	IsAvailable *bool   `json:"isAvailable"`
}

type ComboItemInput struct {
	MenuItemID string `json:"menuItemId"`
	Quantity   int    `json:"quantity"`
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
	// IsVeg: send true/false to set, omit to leave unchanged.
	// Note: to explicitly clear the flag (reset to "not set"), the API would
	// need a tri-state sentinel — deferred for v2.
	IsVeg       *bool   `json:"isVeg"`
	IsAvailable *bool   `json:"isAvailable"`
	HSN         *string `json:"hsn"`
	IsFeatured  *bool   `json:"isFeatured"`
	// AvailableDays: send the weekly-menu schedule (0=Sun..6=Sat) to replace it;
	// an empty array clears it (back to "every day"). Omit to leave unchanged.
	AvailableDays *[]int `json:"availableDays"`
	// Add-ons / combos (#52) — when present, replace-all the item's groups/combo.
	IsCombo        *bool                 `json:"isCombo"`
	ModifierGroups *[]ModifierGroupInput `json:"modifierGroups"`
	ComboItems     *[]ComboItemInput     `json:"comboItems"`
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

// sanitizeWeekdays validates a weekly-menu schedule: keeps only weekdays 0..6,
// de-dupes, and preserves order. Invalid values are dropped rather than
// rejected. Returns an empty (non-nil) array for "every day".
func sanitizeWeekdays(days []int) pq.Int64Array {
	out := pq.Int64Array{}
	seen := map[int]bool{}
	for _, d := range days {
		if d < 0 || d > 6 || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, int64(d))
	}
	return out
}

// saveItemModifiers replaces a menu item's modifier groups (+ options) and combo
// components in one transaction (#52). Replace-all keeps the editor simple: the
// chef sends the full desired set on every save. Blank-named entries are skipped;
// combo components are validated to belong to the same chef (name snapshotted).
func saveItemModifiers(itemID, chefID uuid.UUID, groups []ModifierGroupInput, combos []ComboItemInput) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Replace modifier groups + their options.
		var old []models.ModifierGroup
		tx.Where("menu_item_id = ?", itemID).Find(&old)
		for _, g := range old {
			if err := tx.Where("group_id = ?", g.ID).Delete(&models.ModifierOption{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("menu_item_id = ?", itemID).Delete(&models.ModifierGroup{}).Error; err != nil {
			return err
		}
		for gi, g := range groups {
			if g.Name == "" {
				continue
			}
			grp := models.ModifierGroup{
				MenuItemID: itemID,
				Name:       g.Name,
				Required:   g.Required,
				MinSelect:  g.MinSelect,
				MaxSelect:  g.MaxSelect,
				SortOrder:  gi,
			}
			if err := tx.Create(&grp).Error; err != nil {
				return err
			}
			for oi, o := range g.Options {
				if o.Name == "" {
					continue
				}
				avail := true
				if o.IsAvailable != nil {
					avail = *o.IsAvailable
				}
				if err := tx.Create(&models.ModifierOption{
					GroupID:     grp.ID,
					Name:        o.Name,
					PriceDelta:  o.PriceDelta,
					IsAvailable: avail,
					SortOrder:   oi,
				}).Error; err != nil {
					return err
				}
			}
		}

		// Replace combo components.
		if err := tx.Where("combo_id = ?", itemID).Delete(&models.ComboItem{}).Error; err != nil {
			return err
		}
		for ci, comp := range combos {
			mid, err := uuid.Parse(comp.MenuItemID)
			if err != nil || mid == itemID {
				continue // skip invalid ids and self-reference
			}
			var inc models.MenuItem
			if tx.Select("name").Where("id = ? AND chef_id = ?", mid, chefID).First(&inc).Error != nil {
				continue // only the chef's own items can be bundled
			}
			qty := comp.Quantity
			if qty < 1 {
				qty = 1
			}
			if err := tx.Create(&models.ComboItem{
				ComboID:    itemID,
				MenuItemID: mid,
				Name:       inc.Name,
				Quantity:   qty,
				SortOrder:  ci,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// claimMenuItemImageSlot inserts one gallery row for itemID, deciding the
// count, primary flag and sort order under a row lock on the parent item.
//
// The lock is the whole point. The picker uploads a multi-selection
// concurrently; without it several requests read the same count and all pass,
// which puts the gallery past its cap, gives more than one row IsPrimary (so
// menu_items.image_url is written twice, last write wins) and collides
// sort_order. Reproduced against Postgres in menu_image_race_test.go: eight
// racers on a 4-image item produced 12 rows.
func claimMenuItemImageSlot(db *gorm.DB, itemID uuid.UUID, url string) (models.MenuItemImage, error) {
	var img models.MenuItemImage
	err := db.Transaction(func(tx *gorm.DB) error {
		var locked models.MenuItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", itemID).First(&locked).Error; err != nil {
			return err
		}

		var count int64
		if err := tx.Model(&models.MenuItemImage{}).
			Where("menu_item_id = ?", locked.ID).Count(&count).Error; err != nil {
			return err
		}
		if count >= maxMenuItemImages {
			return errMenuItemImageLimit
		}

		isPrimary := count == 0
		img = models.MenuItemImage{
			MenuItemID: locked.ID,
			URL:        url,
			IsPrimary:  isPrimary,
			SortOrder:  int(count),
		}
		if err := tx.Create(&img).Error; err != nil {
			return err
		}
		if isPrimary {
			return tx.Model(&models.MenuItem{}).
				Where("id = ?", locked.ID).Update("image_url", url).Error
		}
		return nil
	})
	return img, err
}
