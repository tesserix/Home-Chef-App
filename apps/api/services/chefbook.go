package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// chefbook.go — the Mongo-backed store for ChefBook articles.
//
// Mongo is optional for the rest of the platform (see mongo.go): when it isn't
// connected the messaging features no-op. ChefBook can't no-op the same way —
// an article that silently fails to save is worse than a clear error — so every
// entry point here returns ErrChefBookUnavailable instead, and the handler
// turns that into a 503 the client can actually show.

const collArticles = "chefbook_articles"

// ErrChefBookUnavailable is returned when Mongo isn't connected.
var ErrChefBookUnavailable = errors.New("chefbook storage unavailable")

// ErrArticleNotFound is returned for a missing or non-visible article.
var ErrArticleNotFound = errors.New("article not found")

func articles() (*mongo.Collection, error) {
	mc := GetMongoClient()
	if !mc.IsConnected() {
		return nil, ErrChefBookUnavailable
	}
	return mc.Collection(collArticles), nil
}

// EnsureChefBookIndexes creates the indexes ChefBook queries rely on. Safe to
// call repeatedly; Mongo treats an identical index spec as a no-op.
func EnsureChefBookIndexes(ctx context.Context) error {
	col, err := articles()
	if err != nil {
		return err
	}
	_, err = col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		// The public feed: published articles, newest first.
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "publishedAt", Value: -1}}},
		// A chef's own list, including drafts.
		{Keys: bson.D{{Key: "chefId", Value: 1}, {Key: "updatedAt", Value: -1}}},
		// Slug lookups from a shared link. Unique, and sparse so the many
		// drafts without a slug don't collide with each other on null.
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		// Free-text search over what a reader actually reads.
		{Keys: bson.D{{Key: "title", Value: "text"}, {Key: "excerpt", Value: "text"}}},
	})
	return err
}

var slugStrip = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify builds a URL-safe slug. The random suffix is supplied by the caller
// rather than generated here so the result is deterministic and testable.
func Slugify(title, suffix string) string {
	s := slugStrip.ReplaceAllString(strings.ToLower(strings.TrimSpace(title)), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "article"
	}
	// Keep slugs short enough to stay readable in a shared link.
	if len(s) > 60 {
		s = strings.Trim(s[:60], "-")
	}
	if suffix == "" {
		return s
	}
	return s + "-" + suffix
}

// CreateArticle inserts a new article. The caller is responsible for having
// already run the culinary check and PII filter.
func CreateArticle(ctx context.Context, a *models.ChefBookArticle) error {
	col, err := articles()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	if a.Status == models.ArticleStatusPublished && a.PublishedAt == nil {
		a.PublishedAt = &now
	}
	res, err := col.InsertOne(ctx, a)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		a.ID = oid
	}
	return nil
}

// ArticleFeedQuery narrows the public feed.
type ArticleFeedQuery struct {
	ChefID string
	Tag    string
	Search string
	Limit  int
	Skip   int
}

// ListPublishedArticles returns the public feed, newest first.
func ListPublishedArticles(ctx context.Context, q ArticleFeedQuery) ([]models.ChefBookArticle, int64, error) {
	col, err := articles()
	if err != nil {
		return nil, 0, err
	}

	filter := bson.M{
		"status": models.ArticleStatusPublished,
		// Moderated-down content must not resurface in the feed.
		"moderation.isModerated": bson.M{"$ne": true},
	}
	if q.ChefID != "" {
		filter["chefId"] = q.ChefID
	}
	if q.Tag != "" {
		filter["tags"] = strings.ToLower(q.Tag)
	}
	if q.Search != "" {
		filter["$text"] = bson.M{"$search": q.Search}
	}

	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "publishedAt", Value: -1}}).
		SetLimit(int64(q.Limit)).
		SetSkip(int64(q.Skip)).
		// The feed renders excerpts, so comment bodies and article blocks are
		// dead weight — but the comments' hidden flags must survive, or
		// VisibleComments() counts zero on every listing.
		SetProjection(feedProjection)
	cur, err := col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var out []models.ChefBookArticle
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// feedProjection strips article bodies and comment payloads from listings
// while keeping each comment's hidden flag, so comment counts stay correct.
var feedProjection = bson.M{
	"blocks":              0,
	"comments.body":       0,
	"comments.userId":     0,
	"comments.userName":   0,
	"comments.userAvatar": 0,
	"comments.createdAt":  0,
}

// ListChefArticles returns one chef's articles including drafts.
func ListChefArticles(ctx context.Context, chefID string, limit, skip int) ([]models.ChefBookArticle, int64, error) {
	col, err := articles()
	if err != nil {
		return nil, 0, err
	}
	filter := bson.M{"chefId": chefID}
	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "updatedAt", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(skip)).
		SetProjection(feedProjection)
	cur, err := col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	var out []models.ChefBookArticle
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// GetArticleBySlug loads a published article for a reader.
func GetArticleBySlug(ctx context.Context, slug string) (*models.ChefBookArticle, error) {
	col, err := articles()
	if err != nil {
		return nil, err
	}
	var a models.ChefBookArticle
	err = col.FindOne(ctx, bson.M{
		"slug":                   slug,
		"status":                 models.ArticleStatusPublished,
		"moderation.isModerated": bson.M{"$ne": true},
	}).Decode(&a)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrArticleNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// GetArticleByIDForChef loads an article the chef owns, at any status, so the
// author can open their own drafts.
func GetArticleByIDForChef(ctx context.Context, id bson.ObjectID, chefID string) (*models.ChefBookArticle, error) {
	col, err := articles()
	if err != nil {
		return nil, err
	}
	var a models.ChefBookArticle
	err = col.FindOne(ctx, bson.M{"_id": id, "chefId": chefID}).Decode(&a)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrArticleNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateArticle replaces the editable fields of an article the chef owns.
func UpdateArticle(ctx context.Context, id bson.ObjectID, chefID string, set bson.M) error {
	col, err := articles()
	if err != nil {
		return err
	}
	set["updatedAt"] = time.Now().UTC()
	res, err := col.UpdateOne(ctx, bson.M{"_id": id, "chefId": chefID}, bson.M{"$set": set})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrArticleNotFound
	}
	return nil
}

// DeleteArticle removes an article the chef owns.
func DeleteArticle(ctx context.Context, id bson.ObjectID, chefID string) error {
	col, err := articles()
	if err != nil {
		return err
	}
	res, err := col.DeleteOne(ctx, bson.M{"_id": id, "chefId": chefID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrArticleNotFound
	}
	return nil
}

// SetReaction records or switches a reader's reaction. Passing an empty
// reaction removes it, which is what tapping the active one again means.
//
// Two writes rather than one: Mongo can't pull and push the same array in a
// single update. The pull is unconditional so switching reaction type can never
// leave a user with two.
func SetReaction(ctx context.Context, id bson.ObjectID, userID string, r models.ReactionType) error {
	col, err := articles()
	if err != nil {
		return err
	}
	if r != "" && !r.IsValid() {
		return fmt.Errorf("unsupported reaction %q", r)
	}

	// FindOneAndUpdate rather than UpdateOne so the pull also hands back the
	// pre-change document: whether this reader already had a reaction, and which
	// kitchen to credit. Reading it separately would race a concurrent tap.
	var before models.ChefBookArticle
	err = col.FindOneAndUpdate(ctx,
		bson.M{"_id": id},
		bson.M{"$pull": bson.M{"reactions": bson.M{"userId": userID}}},
		options.FindOneAndUpdate().
			SetReturnDocument(options.Before).
			SetProjection(bson.M{"chefId": 1, "status": 1, "reactions": 1}),
	).Decode(&before)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrArticleNotFound
	}
	if err != nil {
		return err
	}

	if r != "" {
		if _, err := col.UpdateOne(ctx,
			bson.M{"_id": id},
			bson.M{"$push": bson.M{"reactions": models.ArticleReaction{
				UserID:    userID,
				Type:      r,
				CreatedAt: time.Now().UTC(),
			}}}); err != nil {
			return err
		}
	}

	creditArticleReaction(before, userID, r)
	return nil
}

// creditArticleReaction moves the kitchen's ranking counter for a reaction that
// just changed. Best-effort by design: the reaction is already saved, and
// failing the reader's request because a ranking counter would not budge is the
// wrong trade.
func creditArticleReaction(before models.ChefBookArticle, userID string, now models.ReactionType) {
	// A draft is not published work, so reacting to one earns no ranking. Without
	// this a chef could stack unlisted drafts and react to each.
	if before.Status != models.ArticleStatusPublished {
		return
	}
	delta := ArticleReactionDelta(before.ViewerReaction(userID), now)
	if delta == 0 {
		return
	}
	chefID, err := uuid.Parse(before.ChefID)
	if err != nil {
		return
	}
	// A chef reacting to their own article is not an endorsement by anyone else.
	if owner, err := chefOwnerUserID(database.DB, chefID); err == nil && owner.String() == userID {
		return
	}
	if err := ApplyArticleReaction(chefID, delta); err != nil {
		log.Printf("chefbook: credit article reaction for chef %s: %v", chefID, err)
	}
}

// AddComment appends a comment. Returns ErrCommentLimit once an article is past
// the soft cap, rather than growing a document toward Mongo's 16 MB ceiling.
var ErrCommentLimit = errors.New("this article has reached its comment limit")

func AddComment(ctx context.Context, id bson.ObjectID, c models.ArticleComment) error {
	col, err := articles()
	if err != nil {
		return err
	}

	var existing models.ChefBookArticle
	if err := col.FindOne(ctx, bson.M{"_id": id},
		options.FindOne().SetProjection(bson.M{"comments": 1})).Decode(&existing); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrArticleNotFound
		}
		return err
	}
	if len(existing.Comments) >= models.CommentSoftCap {
		return ErrCommentLimit
	}

	c.ID = bson.NewObjectID()
	c.CreatedAt = time.Now().UTC()
	_, err = col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$push": bson.M{"comments": c}})
	return err
}

// DeleteComment removes a comment. The author of the comment and the author of
// the article may both remove it; the caller decides which applies.
func DeleteComment(ctx context.Context, id, commentID bson.ObjectID, userID string, isArticleAuthor bool) error {
	col, err := articles()
	if err != nil {
		return err
	}
	pull := bson.M{"_id": commentID}
	if !isArticleAuthor {
		// A reader may only remove their own comment.
		pull["userId"] = userID
	}
	res, err := col.UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$pull": bson.M{"comments": pull}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrArticleNotFound
	}
	return nil
}

// RefreshChefIdentity re-stamps the denormalised chef name and image across
// that chef's articles. Called when a chef edits their profile — without it a
// renamed kitchen keeps its old byline on everything already published.
func RefreshChefIdentity(ctx context.Context, chefID, name, image string) error {
	col, err := articles()
	if err != nil {
		return err
	}
	_, err = col.UpdateMany(ctx,
		bson.M{"chefId": chefID},
		bson.M{"$set": bson.M{"chefName": name, "chefImage": image}})
	return err
}
