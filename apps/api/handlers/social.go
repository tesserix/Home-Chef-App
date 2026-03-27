package handlers

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"github.com/lib/pq"
)

type SocialHandler struct{}

func NewSocialHandler() *SocialHandler {
	return &SocialHandler{}
}

var hashtagRegex = regexp.MustCompile(`#(\w+)`)

// extractHashtags extracts unique hashtags from content.
func extractHashtags(content string) []string {
	matches := hashtagRegex.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var tags []string
	for _, m := range matches {
		tag := strings.ToLower(m[1])
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	return tags
}

// GetFeed returns a public paginated feed of published posts, newest first.
// GET /social/feed
func (h *SocialHandler) GetFeed(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	hashtag := c.Query("hashtag")

	var posts []models.Post
	var total int64

	query := database.DB.Where("status = ?", models.PostStatusPublished)
	if hashtag != "" {
		query = query.Where("? = ANY(hashtags)", strings.ToLower(hashtag))
	}

	query.Model(&models.Post{}).Count(&total)

	if err := query.
		Preload("Chef").
		Preload("MenuItem").
		Preload("Likes").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feed"})
		return
	}

	// Get current user ID if authenticated (for isLiked)
	var userIDPtr *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		userIDPtr = &uid
	}

	responses := make([]models.PostResponse, len(posts))
	for i, post := range posts {
		responses[i] = post.ToResponse(userIDPtr)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetPost returns a single post with its comments.
// GET /social/posts/:id
func (h *SocialHandler) GetPost(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	var post models.Post
	if err := database.DB.
		Preload("Chef").
		Preload("MenuItem").
		Preload("Likes").
		Preload("Comments", "parent_id IS NULL AND is_hidden = false").
		Preload("Comments.User").
		Preload("Comments.Replies", "is_hidden = false").
		Preload("Comments.Replies.User").
		Where("status = ?", models.PostStatusPublished).
		First(&post, "id = ?", postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	var userIDPtr *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		userIDPtr = &uid
	}

	// Build comment responses
	comments := make([]models.CommentResponse, 0, len(post.Comments))
	for _, comment := range post.Comments {
		replies := make([]models.CommentResponse, 0, len(comment.Replies))
		for _, reply := range comment.Replies {
			replies = append(replies, models.CommentResponse{
				ID:      reply.ID,
				Content: reply.Content,
				User: models.CommentUserInfo{
					ID:        reply.User.ID,
					FirstName: reply.User.FirstName,
					Avatar:    reply.User.Avatar,
				},
				CreatedAt: reply.CreatedAt,
			})
		}
		comments = append(comments, models.CommentResponse{
			ID:      comment.ID,
			Content: comment.Content,
			User: models.CommentUserInfo{
				ID:        comment.User.ID,
				FirstName: comment.User.FirstName,
				Avatar:    comment.User.Avatar,
			},
			Replies:   replies,
			CreatedAt: comment.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     post.ToResponse(userIDPtr),
		"comments": comments,
	})
}

// LikePost toggles a like on a post. If already liked, it unlikes.
// POST /social/posts/:id/like
func (h *SocialHandler) LikePost(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	// Verify the post exists and is published
	var post models.Post
	if err := database.DB.Where("id = ? AND status = ?", postID, models.PostStatusPublished).First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Check if already liked
	var existing models.PostLike
	if err := database.DB.Where("post_id = ? AND user_id = ?", postID, userID).First(&existing).Error; err == nil {
		// Unlike: remove the like
		database.DB.Delete(&existing)
		database.DB.Model(&post).Update("likes_count", post.LikesCount-1)
		c.JSON(http.StatusOK, gin.H{"liked": false, "likesCount": post.LikesCount - 1})
		return
	}

	// Like the post
	like := models.PostLike{
		PostID: postID,
		UserID: userID,
	}
	if err := database.DB.Create(&like).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to like post"})
		return
	}
	database.DB.Model(&post).Update("likes_count", post.LikesCount+1)

	c.JSON(http.StatusOK, gin.H{"liked": true, "likesCount": post.LikesCount + 1})
}

// AddComment adds a comment to a post. Content is PII-filtered.
// POST /social/posts/:id/comments
func (h *SocialHandler) AddComment(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	var input struct {
		Content  string     `json:"content" binding:"required"`
		ParentID *uuid.UUID `json:"parentId"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content is required"})
		return
	}

	if len(strings.TrimSpace(input.Content)) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment cannot be empty"})
		return
	}

	// Verify post exists and is published
	var post models.Post
	if err := database.DB.Where("id = ? AND status = ?", postID, models.PostStatusPublished).First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// If replying, verify parent comment exists
	if input.ParentID != nil {
		var parent models.PostComment
		if err := database.DB.Where("id = ? AND post_id = ?", *input.ParentID, postID).First(&parent).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Parent comment not found"})
			return
		}
	}

	// PII filter the comment
	sanitized, _, _ := services.FilterChatMessage(input.Content)

	comment := models.PostComment{
		PostID:   postID,
		UserID:   userID,
		ParentID: input.ParentID,
		Content:  sanitized,
	}
	if err := database.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add comment"})
		return
	}

	// Increment comment count
	database.DB.Model(&post).Update("comments_count", post.CommentsCount+1)

	// Load user for response
	database.DB.First(&comment.User, "id = ?", userID)

	c.JSON(http.StatusCreated, gin.H{
		"data": models.CommentResponse{
			ID:      comment.ID,
			Content: comment.Content,
			User: models.CommentUserInfo{
				ID:        comment.User.ID,
				FirstName: comment.User.FirstName,
				Avatar:    comment.User.Avatar,
			},
			CreatedAt: comment.CreatedAt,
		},
	})
}

// ---- Chef Post Management ----

// GetChefPosts returns the authenticated chef's posts.
// GET /chef/posts
func (h *SocialHandler) GetChefPosts(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	status := c.Query("status")

	var posts []models.Post
	var total int64

	query := database.DB.Where("chef_id = ?", chef.ID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Model(&models.Post{}).Count(&total)

	if err := query.
		Preload("Chef").
		Preload("MenuItem").
		Preload("Likes").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}

	userIDPtr := &userID
	responses := make([]models.PostResponse, len(posts))
	for i, post := range posts {
		responses[i] = post.ToResponse(userIDPtr)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// CreatePost creates a new social post. Content is PII-filtered.
// POST /chef/posts — multipart/form-data with fields: content, menuItemId, images (up to 4)
func (h *SocialHandler) CreatePost(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	// Parse multipart form (32 MB max)
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	content := c.PostForm("content")
	if strings.TrimSpace(content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content is required"})
		return
	}

	// PII filter the content
	sanitized, hasPII, violations := services.FilterChatMessage(content)
	if hasPII {
		log.Printf("PII detected in social post from chef %s: %v", chef.ID, violations)
	}

	// Extract hashtags
	hashtags := extractHashtags(sanitized)

	// Optional menu item link
	var menuItemID *uuid.UUID
	if mid := c.PostForm("menuItemId"); mid != "" {
		parsed, err := uuid.Parse(mid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid menuItemId"})
			return
		}
		// Verify the menu item belongs to this chef
		var item models.MenuItem
		if err := database.DB.
			Joins("JOIN menu_categories ON menu_categories.id = menu_items.category_id").
			Where("menu_items.id = ? AND menu_categories.chef_id = ?", parsed, chef.ID).
			First(&item).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Menu item not found"})
			return
		}
		menuItemID = &parsed
	}

	// Handle image uploads (optional, max 4)
	var imageURLs []string
	form := c.Request.MultipartForm
	if form != nil && form.File["images"] != nil {
		files := form.File["images"]
		if len(files) > 4 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 4 images per post"})
			return
		}

		for _, header := range files {
			if header.Size > 5*1024*1024 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Each image must be under 5 MB"})
				return
			}

			contentType := header.Header.Get("Content-Type")
			if !services.IsImageContentType(contentType) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image type. Allowed: JPEG, PNG, WebP."})
				return
			}

			file, err := header.Open()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read image"})
				return
			}
			defer file.Close()

			folder := fmt.Sprintf("social/%s", chef.ID.String())
			fileURL, err := services.UploadPublicFile(c.Request.Context(), folder, header.Filename, file, contentType)
			if err != nil {
				log.Printf("Failed to upload social post image: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
				return
			}
			imageURLs = append(imageURLs, fileURL)
		}
	}

	post := models.Post{
		ChefID:              chef.ID,
		Content:             sanitized,
		Images:              pq.StringArray(imageURLs),
		Hashtags:            pq.StringArray(hashtags),
		MenuItemID:          menuItemID,
		Status:              models.PostStatusPublished,
		ContactInfoDetected: hasPII,
	}

	if err := database.DB.Create(&post).Error; err != nil {
		log.Printf("Failed to create social post: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
		return
	}

	// Reload with relationships
	database.DB.Preload("Chef").Preload("MenuItem").First(&post, "id = ?", post.ID)

	c.JSON(http.StatusCreated, gin.H{"data": post.ToResponse(&userID)})
}

// UpdatePost updates an existing social post.
// PUT /chef/posts/:id
func (h *SocialHandler) UpdatePost(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	var post models.Post
	if err := database.DB.Where("id = ? AND chef_id = ?", postID, chef.ID).First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	var input struct {
		Content    *string    `json:"content"`
		MenuItemID *uuid.UUID `json:"menuItemId"`
		Status     *string    `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if input.Content != nil {
		if strings.TrimSpace(*input.Content) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Content cannot be empty"})
			return
		}
		sanitized, hasPII, violations := services.FilterChatMessage(*input.Content)
		if hasPII {
			log.Printf("PII detected in post update from chef %s: %v", chef.ID, violations)
		}
		post.Content = sanitized
		post.Hashtags = pq.StringArray(extractHashtags(sanitized))
		post.ContactInfoDetected = hasPII
	}

	if input.MenuItemID != nil {
		// Verify menu item belongs to chef
		var item models.MenuItem
		if err := database.DB.
			Joins("JOIN menu_categories ON menu_categories.id = menu_items.category_id").
			Where("menu_items.id = ? AND menu_categories.chef_id = ?", *input.MenuItemID, chef.ID).
			First(&item).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Menu item not found"})
			return
		}
		post.MenuItemID = input.MenuItemID
	}

	if input.Status != nil {
		status := models.PostStatus(*input.Status)
		if status != models.PostStatusPublished && status != models.PostStatusDraft && status != models.PostStatusArchived {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
			return
		}
		post.Status = status
	}

	if err := database.DB.Save(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update post"})
		return
	}

	database.DB.Preload("Chef").Preload("MenuItem").Preload("Likes").First(&post, "id = ?", post.ID)

	c.JSON(http.StatusOK, gin.H{"data": post.ToResponse(&userID)})
}

// DeletePost soft-deletes a chef's post.
// DELETE /chef/posts/:id
func (h *SocialHandler) DeletePost(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	result := database.DB.Where("id = ? AND chef_id = ?", postID, chef.ID).Delete(&models.Post{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete post"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Post deleted"})
}
