package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// chefbook.go — ChefBook: chef-authored culinary articles.
//
// Articles live in MongoDB as whole documents (see services/chefbook.go);
// `posts` and its handlers are untouched and continue to serve short updates.
//
// Reading is public — an article is meant to be shareable to someone who has
// never opened the app. Writing requires a chef; reacting and commenting
// require any signed-in user.

type ChefBookHandler struct{}

func NewChefBookHandler() *ChefBookHandler { return &ChefBookHandler{} }

// respondStorage maps the storage-layer errors onto status codes. Mongo being
// down is a 503, not a 500: it's transient and the client should say "try
// again shortly" rather than "something broke".
func respondStorage(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, services.ErrChefBookUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "ChefBook is temporarily unavailable. Please try again shortly.",
		})
		return true
	case errors.Is(err, services.ErrArticleNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return true
	case errors.Is(err, services.ErrCommentLimit):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return true
	}
	return false
}

func pageParams(c *gin.Context) (limit, skip, page int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	return limit, (page - 1) * limit, page
}

func viewerID(c *gin.Context) string {
	if uid, ok := middleware.GetUserID(c); ok {
		return uid.String()
	}
	return ""
}

// GetFeed lists published articles.
// GET /chefbook/articles
func (h *ChefBookHandler) GetFeed(c *gin.Context) {
	limit, skip, page := pageParams(c)

	list, total, err := services.ListPublishedArticles(c.Request.Context(), services.ArticleFeedQuery{
		ChefID: c.Query("chefId"),
		Tag:    c.Query("tag"),
		Search: c.Query("q"),
		Limit:  limit,
		Skip:   skip,
	})
	if respondStorage(c, err) {
		return
	}
	if err != nil {
		log.Printf("chefbook: feed failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load articles"})
		return
	}

	viewer := viewerID(c)
	out := make([]models.ArticleResponse, 0, len(list))
	for i := range list {
		out = append(out, list[i].ToResponse(viewer, false))
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "total": total, "page": page, "limit": limit})
}

// GetArticle returns one published article, body included.
// GET /chefbook/articles/:slug
func (h *ChefBookHandler) GetArticle(c *gin.Context) {
	a, err := services.GetArticleBySlug(c.Request.Context(), c.Param("slug"))
	if respondStorage(c, err) {
		return
	}
	if err != nil {
		log.Printf("chefbook: get article failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load article"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": a.ToResponse(viewerID(c), true)})
}

// articleInput is the write shape for both create and update.
type articleInput struct {
	Title  string                `json:"title"`
	Cover  string                `json:"cover"`
	Blocks []models.ArticleBlock `json:"blocks"`
	Tags   []string              `json:"tags"`
	Status models.ArticleStatus  `json:"status"`
}

// validate checks the shape and returns the article's readable text.
func (in *articleInput) validate() (string, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return "", errors.New("A title is required")
	}
	if len(in.Blocks) == 0 {
		return "", errors.New("An article needs some content")
	}
	for i, b := range in.Blocks {
		if !b.Type.IsValid() {
			return "", errors.New("Unsupported block type in block " + strconv.Itoa(i+1))
		}
		if b.Type == models.BlockImage && strings.TrimSpace(b.URL) == "" {
			return "", errors.New("An image block needs an image")
		}
	}
	if in.Status == "" {
		in.Status = models.ArticleStatusDraft
	}
	if in.Status != models.ArticleStatusDraft && in.Status != models.ArticleStatusPublished {
		return "", errors.New("status must be 'draft' or 'published'")
	}

	tmp := models.ChefBookArticle{Blocks: in.Blocks}
	return tmp.BodyText(), nil
}

// chefFor resolves the signed-in user's chef profile.
func chefFor(c *gin.Context) (*models.ChefProfile, bool) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return nil, false
	}
	return &chef, true
}

// CreateArticle publishes or drafts a new article.
// POST /chef/chefbook/articles
func (h *ChefBookHandler) CreateArticle(c *gin.Context) {
	chef, ok := chefFor(c)
	if !ok {
		return
	}

	var in articleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	body, err := in.validate()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ChefBook is for food and cooking. No culinary signal at all is refused;
	// a weak signal is allowed through and flagged for a human.
	topic := services.CheckCulinaryTopic(in.Title, body)
	if !topic.OnTopic {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "ChefBook is for food and cooking. Tell us about the dish, the ingredients or the method.",
			"code":  "off_topic",
		})
		return
	}

	// Same PII filter the short posts use — a chef must not publish their
	// personal phone number or a way to take orders off-platform.
	sanitizedBody, hasPII, violations := services.FilterChatMessage(body)
	if hasPII {
		log.Printf("chefbook: PII detected in article from chef %s: %v", chef.ID, violations)
	}
	_ = sanitizedBody // blocks are stored as authored; the flag drives review

	article := models.ChefBookArticle{
		ChefID:         chef.ID.String(),
		ChefName:       chef.BusinessName,
		ChefImage:      chef.ProfileImage,
		Title:          in.Title,
		Slug:           services.Slugify(in.Title, bson.NewObjectID().Hex()[:6]),
		Cover:          strings.TrimSpace(in.Cover),
		Excerpt:        models.BuildExcerpt(body, 200),
		Blocks:         in.Blocks,
		Tags:           normaliseTags(in.Tags),
		Status:         in.Status,
		ReadingMinutes: models.EstimateReadingMinutes(body),
		Moderation: models.ArticleModeration{
			ContactInfoDetected: hasPII,
			TopicFlagged:        topic.Weak,
		},
	}

	// A cover wasn't given but the article opens with an image — use it rather
	// than making the chef upload the same picture twice.
	if article.Cover == "" {
		for _, b := range in.Blocks {
			if b.Type == models.BlockImage && b.URL != "" {
				article.Cover = b.URL
				break
			}
		}
	}

	if err := services.CreateArticle(c.Request.Context(), &article); err != nil {
		if respondStorage(c, err) {
			return
		}
		log.Printf("chefbook: create failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save article"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": article.ToResponse("", true)})
}

func normaliseTags(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(t, "#")))
		if t == "" {
			continue
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
		if len(out) == 10 {
			break
		}
	}
	return out
}

// ListMine returns the signed-in chef's own articles, drafts included.
// GET /chef/chefbook/articles
func (h *ChefBookHandler) ListMine(c *gin.Context) {
	chef, ok := chefFor(c)
	if !ok {
		return
	}
	limit, skip, page := pageParams(c)
	list, total, err := services.ListChefArticles(c.Request.Context(), chef.ID.String(), limit, skip)
	if respondStorage(c, err) {
		return
	}
	if err != nil {
		log.Printf("chefbook: list mine failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load articles"})
		return
	}
	out := make([]models.ArticleResponse, 0, len(list))
	for i := range list {
		out = append(out, list[i].ToResponse("", false))
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "total": total, "page": page, "limit": limit})
}

// UpdateArticle edits an article the chef owns.
// PUT /chef/chefbook/articles/:id
func (h *ChefBookHandler) UpdateArticle(c *gin.Context) {
	chef, ok := chefFor(c)
	if !ok {
		return
	}
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article id"})
		return
	}

	var in articleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	body, verr := in.validate()
	if verr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": verr.Error()})
		return
	}

	topic := services.CheckCulinaryTopic(in.Title, body)
	if !topic.OnTopic {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "ChefBook is for food and cooking. Tell us about the dish, the ingredients or the method.",
			"code":  "off_topic",
		})
		return
	}
	_, hasPII, _ := services.FilterChatMessage(body)

	set := bson.M{
		"title":                          in.Title,
		"cover":                          strings.TrimSpace(in.Cover),
		"blocks":                         in.Blocks,
		"tags":                           normaliseTags(in.Tags),
		"status":                         in.Status,
		"excerpt":                        models.BuildExcerpt(body, 200),
		"readingMinutes":                 models.EstimateReadingMinutes(body),
		"moderation.topicFlagged":        topic.Weak,
		"moderation.contactInfoDetected": hasPII,
	}
	// Publishing for the first time stamps publishedAt; re-saving a published
	// article must not keep bumping it to the top of the feed.
	if in.Status == models.ArticleStatusPublished {
		existing, err := services.GetArticleByIDForChef(c.Request.Context(), id, chef.ID.String())
		if respondStorage(c, err) {
			return
		}
		if err == nil && existing.PublishedAt == nil {
			set["publishedAt"] = time.Now().UTC()
		}
	}

	if err := services.UpdateArticle(c.Request.Context(), id, chef.ID.String(), set); err != nil {
		if respondStorage(c, err) {
			return
		}
		log.Printf("chefbook: update failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save article"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Article saved"})
}

// DeleteArticle removes an article the chef owns.
// DELETE /chef/chefbook/articles/:id
func (h *ChefBookHandler) DeleteArticle(c *gin.Context) {
	chef, ok := chefFor(c)
	if !ok {
		return
	}
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article id"})
		return
	}
	if err := services.DeleteArticle(c.Request.Context(), id, chef.ID.String()); err != nil {
		if respondStorage(c, err) {
			return
		}
		log.Printf("chefbook: delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete article"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Article deleted"})
}

// React sets, switches or clears the signed-in reader's reaction.
// POST /chefbook/articles/:id/react
func (h *ChefBookHandler) React(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sign in to react"})
		return
	}
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article id"})
		return
	}

	var body struct {
		Reaction models.ReactionType `json:"reaction"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if body.Reaction != "" && !body.Reaction.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported reaction"})
		return
	}

	if err := services.SetReaction(c.Request.Context(), id, uid.String(), body.Reaction); err != nil {
		if respondStorage(c, err) {
			return
		}
		log.Printf("chefbook: react failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save reaction"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Reaction saved"})
}

// AddComment posts a comment on an article.
// POST /chefbook/articles/:id/comments
func (h *ChefBookHandler) AddComment(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sign in to comment"})
		return
	}
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article id"})
		return
	}

	var body struct {
		Body string `json:"body"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	text := strings.TrimSpace(body.Body)
	if text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment can't be empty"})
		return
	}
	if len([]rune(text)) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment is too long (1000 characters max)"})
		return
	}

	// Comments go through the same PII filter as chat: a reader must not be
	// able to move the conversation off-platform in a comment thread.
	sanitized, _, _ := services.FilterChatMessage(text)

	var user models.User
	if err := database.DB.First(&user, "id = ?", uid).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sign in to comment"})
		return
	}

	err = services.AddComment(c.Request.Context(), id, models.ArticleComment{
		UserID:     uid.String(),
		UserName:   strings.TrimSpace(user.FirstName + " " + user.LastName),
		UserAvatar: user.Avatar,
		Body:       sanitized,
	})
	if err != nil {
		if respondStorage(c, err) {
			return
		}
		log.Printf("chefbook: comment failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to post comment"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Comment posted"})
}

// DeleteComment removes a comment. A reader may remove their own; the article's
// author may remove any on their article.
// DELETE /chefbook/articles/:id/comments/:commentId
func (h *ChefBookHandler) DeleteComment(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sign in first"})
		return
	}
	id, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article id"})
		return
	}
	commentID, err := bson.ObjectIDFromHex(c.Param("commentId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment id"})
		return
	}

	// Is this user the chef who wrote the article?
	isAuthor := false
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", uid).First(&chef).Error; err == nil {
		if a, gerr := services.GetArticleByIDForChef(c.Request.Context(), id, chef.ID.String()); gerr == nil && a != nil {
			isAuthor = true
		}
	}

	if err := services.DeleteComment(c.Request.Context(), id, commentID, uid.String(), isAuthor); err != nil {
		if respondStorage(c, err) {
			return
		}
		log.Printf("chefbook: delete comment failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted"})
}
