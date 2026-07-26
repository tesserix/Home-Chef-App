package models

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// chefbook.go — ChefBook articles, stored in MongoDB.
//
// A ChefBook article is a whole entity in Mongo rather than a row in
// `posts`: the body is a block document, and reactions and comments are
// embedded on it. `posts` stays what it always was — short chef updates — and
// is untouched by this file.
//
// The cost of that split is deliberate and worth stating: chef identity is
// DENORMALISED onto every article (ChefName/ChefImage), because there is no
// join back to chef_profiles from here. RefreshChefIdentity exists so a chef
// renaming their kitchen doesn't leave stale bylines behind.
//
// Embedded reactions and comments have a ceiling: a Mongo document is capped at
// 16 MB, so an article with a very large number of comments will eventually
// need them moved to their own collection. That is fine at the volumes this
// launches into and is called out at CommentSoftCap below.

// ArticleStatus mirrors the lifecycle PostStatus uses for short posts, so the
// two surfaces read the same way in admin tooling.
type ArticleStatus string

const (
	ArticleStatusDraft     ArticleStatus = "draft"
	ArticleStatusPublished ArticleStatus = "published"
	ArticleStatusArchived  ArticleStatus = "archived"
	ArticleStatusFlagged   ArticleStatus = "flagged"
)

// ReactionType — deliberately food-flavoured rather than the generic
// like/love/haha set. ChefBook is a cooking community; "want to try" is a
// meaningful signal here in a way "haha" is not.
type ReactionType string

const (
	ReactionYum       ReactionType = "yum"
	ReactionLove      ReactionType = "love"
	ReactionWantToTry ReactionType = "want_to_try"
	ReactionClever    ReactionType = "clever"
)

// AllReactionTypes is the allowed set, in display order.
var AllReactionTypes = []ReactionType{
	ReactionYum, ReactionLove, ReactionWantToTry, ReactionClever,
}

// IsValid reports whether r is one of the four supported reactions.
func (r ReactionType) IsValid() bool {
	for _, t := range AllReactionTypes {
		if r == t {
			return true
		}
	}
	return false
}

// BlockType is the kind of content in an article block. Keeping the body as
// typed blocks rather than a blob of HTML means the clients render it natively
// (and identically on web and mobile) and there is no HTML to sanitise.
type BlockType string

const (
	BlockParagraph BlockType = "paragraph"
	BlockHeading   BlockType = "heading"
	BlockImage     BlockType = "image"
	BlockQuote     BlockType = "quote"
	BlockList      BlockType = "list"
)

// IsValid reports whether b is a supported block type.
func (b BlockType) IsValid() bool {
	switch b {
	case BlockParagraph, BlockHeading, BlockImage, BlockQuote, BlockList:
		return true
	}
	return false
}

// ArticleBlock is one unit of an article body.
type ArticleBlock struct {
	Type BlockType `bson:"type" json:"type"`
	// Text carries paragraph/heading/quote content.
	Text string `bson:"text,omitempty" json:"text,omitempty"`
	// URL and Caption are used by image blocks.
	URL     string `bson:"url,omitempty" json:"url,omitempty"`
	Caption string `bson:"caption,omitempty" json:"caption,omitempty"`
	// Items carries list blocks.
	Items []string `bson:"items,omitempty" json:"items,omitempty"`
}

// PlainText is the block's contribution to the article's readable text. Used
// for the culinary check, the excerpt and the reading estimate, none of which
// should see image URLs.
func (b ArticleBlock) PlainText() string {
	switch b.Type {
	case BlockParagraph, BlockHeading, BlockQuote:
		return b.Text
	case BlockList:
		return strings.Join(b.Items, " ")
	default:
		return ""
	}
}

// ArticleReaction is one reader's reaction. One per user per article: reacting
// again with a different type switches it rather than accumulating.
type ArticleReaction struct {
	UserID    string       `bson:"userId" json:"userId"`
	Type      ReactionType `bson:"type" json:"type"`
	CreatedAt time.Time    `bson:"createdAt" json:"createdAt"`
}

// ArticleComment is embedded on the article. User identity is denormalised for
// the same reason the chef's is — there is no join available from Mongo.
type ArticleComment struct {
	ID         bson.ObjectID `bson:"_id" json:"id"`
	UserID     string        `bson:"userId" json:"userId"`
	UserName   string        `bson:"userName" json:"userName"`
	UserAvatar string        `bson:"userAvatar,omitempty" json:"userAvatar,omitempty"`
	Body       string        `bson:"body" json:"body"`
	// Hidden keeps a moderated comment in place (so counts and threading stay
	// stable) while withholding it from readers.
	Hidden    bool      `bson:"hidden" json:"-"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

// ArticleModeration mirrors the moderation fields on Post so an admin sees the
// same shape whichever surface the content came from.
type ArticleModeration struct {
	IsModerated         bool       `bson:"isModerated" json:"isModerated"`
	ModeratedAt         *time.Time `bson:"moderatedAt,omitempty" json:"moderatedAt,omitempty"`
	ModeratorNote       string     `bson:"moderatorNote,omitempty" json:"moderatorNote,omitempty"`
	ContactInfoDetected bool       `bson:"contactInfoDetected" json:"contactInfoDetected"`
	// TopicFlagged marks a weak culinary signal — allowed through, but worth a
	// human look. Content with no culinary signal is refused outright.
	TopicFlagged bool `bson:"topicFlagged" json:"topicFlagged"`
}

// CommentSoftCap is the point past which comments should stop being embedded.
// Mongo caps a document at 16 MB; at ~1 KB per comment this leaves an order of
// magnitude of headroom while still being a number we'd actually notice.
const CommentSoftCap = 2000

// ChefBookArticle is the whole entity.
type ChefBookArticle struct {
	ID bson.ObjectID `bson:"_id,omitempty" json:"id"`

	// Chef identity, denormalised — see the file comment.
	ChefID    string `bson:"chefId" json:"chefId"`
	ChefName  string `bson:"chefName" json:"chefName"`
	ChefImage string `bson:"chefImage,omitempty" json:"chefImage,omitempty"`

	Title   string         `bson:"title" json:"title"`
	Slug    string         `bson:"slug" json:"slug"`
	Cover   string         `bson:"cover,omitempty" json:"cover,omitempty"`
	Excerpt string         `bson:"excerpt" json:"excerpt"`
	Blocks  []ArticleBlock `bson:"blocks" json:"blocks"`
	Tags    []string       `bson:"tags,omitempty" json:"tags,omitempty"`

	Status         ArticleStatus `bson:"status" json:"status"`
	ReadingMinutes int           `bson:"readingMinutes" json:"readingMinutes"`

	Reactions []ArticleReaction `bson:"reactions,omitempty" json:"-"`
	Comments  []ArticleComment  `bson:"comments,omitempty" json:"-"`

	Moderation ArticleModeration `bson:"moderation" json:"-"`

	CreatedAt   time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time  `bson:"updatedAt" json:"updatedAt"`
	PublishedAt *time.Time `bson:"publishedAt,omitempty" json:"publishedAt,omitempty"`
}

// BodyText is the whole readable article, used for the culinary check and the
// reading estimate.
func (a *ChefBookArticle) BodyText() string {
	parts := make([]string, 0, len(a.Blocks))
	for _, b := range a.Blocks {
		if t := strings.TrimSpace(b.PlainText()); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}

// ReactionCounts totals each reaction type for display.
func (a *ChefBookArticle) ReactionCounts() map[ReactionType]int {
	counts := make(map[ReactionType]int, len(AllReactionTypes))
	for _, t := range AllReactionTypes {
		counts[t] = 0
	}
	for _, r := range a.Reactions {
		if r.Type.IsValid() {
			counts[r.Type]++
		}
	}
	return counts
}

// ViewerReaction returns the reaction this user left, if any.
func (a *ChefBookArticle) ViewerReaction(userID string) ReactionType {
	if userID == "" {
		return ""
	}
	for _, r := range a.Reactions {
		if r.UserID == userID {
			return r.Type
		}
	}
	return ""
}

// VisibleComments drops moderated ones. Kept as a method so no handler
// accidentally serves the raw slice.
func (a *ChefBookArticle) VisibleComments() []ArticleComment {
	out := make([]ArticleComment, 0, len(a.Comments))
	for _, c := range a.Comments {
		if !c.Hidden {
			out = append(out, c)
		}
	}
	return out
}

// ArticleResponse is the client-facing shape. Reactions and comments are
// summarised rather than dumped: a feed of 20 articles should not ship every
// reaction row on every one of them.
type ArticleResponse struct {
	ID        string `json:"id"`
	ChefID    string `json:"chefId"`
	ChefName  string `json:"chefName"`
	ChefImage string `json:"chefImage,omitempty"`

	Title   string         `json:"title"`
	Slug    string         `json:"slug"`
	Cover   string         `json:"cover,omitempty"`
	Excerpt string         `json:"excerpt"`
	Blocks  []ArticleBlock `json:"blocks,omitempty"`
	Tags    []string       `json:"tags,omitempty"`

	Status         ArticleStatus `json:"status"`
	ReadingMinutes int           `json:"readingMinutes"`

	ReactionCounts map[ReactionType]int `json:"reactionCounts"`
	ViewerReaction ReactionType         `json:"viewerReaction,omitempty"`
	ReactionsTotal int                  `json:"reactionsTotal"`
	CommentsCount  int                  `json:"commentsCount"`
	Comments       []ArticleComment     `json:"comments,omitempty"`

	CreatedAt   time.Time  `json:"createdAt"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
}

// ToResponse renders the article for a client. includeBody is false in feed
// listings (excerpt only) and true on the detail view.
func (a *ChefBookArticle) ToResponse(viewerID string, includeBody bool) ArticleResponse {
	counts := a.ReactionCounts()
	total := 0
	for _, n := range counts {
		total += n
	}
	visible := a.VisibleComments()

	resp := ArticleResponse{
		ID:             a.ID.Hex(),
		ChefID:         a.ChefID,
		ChefName:       a.ChefName,
		ChefImage:      a.ChefImage,
		Title:          a.Title,
		Slug:           a.Slug,
		Cover:          a.Cover,
		Excerpt:        a.Excerpt,
		Tags:           a.Tags,
		Status:         a.Status,
		ReadingMinutes: a.ReadingMinutes,
		ReactionCounts: counts,
		ViewerReaction: a.ViewerReaction(viewerID),
		ReactionsTotal: total,
		CommentsCount:  len(visible),
		CreatedAt:      a.CreatedAt,
		PublishedAt:    a.PublishedAt,
	}
	if includeBody {
		resp.Blocks = a.Blocks
		resp.Comments = visible
	}
	return resp
}

// BuildExcerpt trims the article's text to a preview of at most max runes,
// cutting on a word boundary so the feed never shows a half-word.
func BuildExcerpt(body string, max int) string {
	body = strings.Join(strings.Fields(body), " ")
	if len([]rune(body)) <= max {
		return body
	}
	runes := []rune(body)[:max]
	if i := strings.LastIndex(string(runes), " "); i > 0 {
		return string(runes[:i]) + "…"
	}
	return string(runes) + "…"
}
