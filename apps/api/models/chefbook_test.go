package models

import "testing"

// The article helpers carry real behaviour — reaction switching, excerpt
// trimming, comment visibility — so they're worth pinning independently of
// Mongo, which none of this touches.

func TestReactionCountsAndViewer(t *testing.T) {
	a := &ChefBookArticle{Reactions: []ArticleReaction{
		{UserID: "u1", Type: ReactionYum},
		{UserID: "u2", Type: ReactionYum},
		{UserID: "u3", Type: ReactionWantToTry},
		// A junk value that reached the DB somehow must not be counted, or the
		// totals stop adding up.
		{UserID: "u4", Type: ReactionType("shrug")},
	}}

	counts := a.ReactionCounts()
	if counts[ReactionYum] != 2 {
		t.Fatalf("yum = %d, want 2", counts[ReactionYum])
	}
	if counts[ReactionWantToTry] != 1 {
		t.Fatalf("want_to_try = %d, want 1", counts[ReactionWantToTry])
	}
	if counts[ReactionLove] != 0 {
		t.Fatalf("love = %d, want 0", counts[ReactionLove])
	}
	// Every supported type is present so a client can render the full row
	// without checking for missing keys.
	if len(counts) != len(AllReactionTypes) {
		t.Fatalf("counts has %d keys, want %d", len(counts), len(AllReactionTypes))
	}

	if got := a.ViewerReaction("u3"); got != ReactionWantToTry {
		t.Fatalf("viewer u3 = %q, want want_to_try", got)
	}
	if got := a.ViewerReaction("nobody"); got != "" {
		t.Fatalf("viewer nobody = %q, want empty", got)
	}
	// An anonymous reader has no reaction, and must not accidentally match a
	// row whose userId is somehow empty.
	if got := a.ViewerReaction(""); got != "" {
		t.Fatalf("anonymous viewer = %q, want empty", got)
	}
}

func TestVisibleCommentsHidesModerated(t *testing.T) {
	a := &ChefBookArticle{Comments: []ArticleComment{
		{Body: "lovely"},
		{Body: "spam", Hidden: true},
		{Body: "thanks"},
	}}
	got := a.VisibleComments()
	if len(got) != 2 {
		t.Fatalf("visible = %d, want 2", len(got))
	}
	for _, c := range got {
		if c.Hidden {
			t.Fatal("a hidden comment leaked into VisibleComments")
		}
	}
	// The response must count only what a reader can actually see.
	if resp := a.ToResponse("", true); resp.CommentsCount != 2 {
		t.Fatalf("CommentsCount = %d, want 2", resp.CommentsCount)
	}
}

func TestToResponseBodyOnlyOnDetail(t *testing.T) {
	a := &ChefBookArticle{
		Blocks:   []ArticleBlock{{Type: BlockParagraph, Text: "Fry the onions."}},
		Comments: []ArticleComment{{Body: "yum"}},
	}
	if feed := a.ToResponse("", false); len(feed.Blocks) != 0 || len(feed.Comments) != 0 {
		t.Fatal("feed listing must not carry the body or comments")
	}
	if detail := a.ToResponse("", true); len(detail.Blocks) != 1 || len(detail.Comments) != 1 {
		t.Fatal("detail view must carry the body and comments")
	}
}

func TestBodyTextIgnoresImageURLs(t *testing.T) {
	a := &ChefBookArticle{Blocks: []ArticleBlock{
		{Type: BlockHeading, Text: "Dalma"},
		{Type: BlockImage, URL: "https://example.com/pumpkin-curry-recipe.jpg"},
		{Type: BlockList, Items: []string{"toor dal", "pumpkin"}},
	}}
	body := a.BodyText()
	// An image URL full of food words would otherwise game the culinary check.
	if want := "Dalma toor dal pumpkin"; body != want {
		t.Fatalf("BodyText = %q, want %q", body, want)
	}
}

func TestBuildExcerpt(t *testing.T) {
	if got := BuildExcerpt("Short one.", 200); got != "Short one." {
		t.Fatalf("short text was altered: %q", got)
	}
	// Collapses whitespace so a multi-paragraph body doesn't render with gaps.
	if got := BuildExcerpt("a\n\n  b   c", 200); got != "a b c" {
		t.Fatalf("whitespace not collapsed: %q", got)
	}
	long := "The secret to a good dalma is roasting the dal before it ever meets water"
	got := BuildExcerpt(long, 20)
	if len([]rune(got)) > 21 { // 20 + the ellipsis
		t.Fatalf("excerpt too long: %q", got)
	}
	if got[len(got)-3:] != "…" {
		t.Fatalf("excerpt should end with an ellipsis: %q", got)
	}
	// Cut on a word boundary — never mid-word.
	if got == "The secret to a goo…" {
		t.Fatalf("excerpt cut mid-word: %q", got)
	}
}

func TestEstimateReadingMinutes(t *testing.T) {
	if got := EstimateReadingMinutes(""); got != 0 {
		t.Fatalf("empty = %d, want 0", got)
	}
	// Anything non-empty is at least a minute — "0 min read" reads as a bug.
	if got := EstimateReadingMinutes("one two three"); got != 1 {
		t.Fatalf("short = %d, want 1", got)
	}
	words := ""
	for range 450 {
		words += "word "
	}
	if got := EstimateReadingMinutes(words); got != 2 {
		t.Fatalf("450 words = %d, want 2", got)
	}
}
