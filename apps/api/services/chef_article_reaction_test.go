package services

// chef_article_reaction_test.go — ChefBook reactions feeding the kitchen's
// ranking counter. The reactions themselves live in Mongo; what is tested here
// is the Postgres counter discovery ranks by, and the delta rule that decides
// when it moves at all.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func articleReactionCount(t *testing.T, db *gorm.DB, chefID uuid.UUID) int {
	t.Helper()
	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	return chef.ArticleReactionCount
}

// Switching reaction type is not new engagement — the reader already counted.
func TestArticleReactionDelta(t *testing.T) {
	cases := []struct {
		name string
		had  models.ReactionType
		now  models.ReactionType
		want int64
	}{
		{"first reaction", "", models.ReactionYum, 1},
		{"removed", models.ReactionYum, "", -1},
		{"switched type", models.ReactionYum, models.ReactionLove, 0},
		{"same again", models.ReactionYum, models.ReactionYum, 0},
		{"never had, still none", "", "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, ArticleReactionDelta(tc.had, tc.now))
		})
	}
}

func TestApplyArticleReaction_MovesCounter(t *testing.T) {
	db, chefID := setupAudienceDB(t)

	require.NoError(t, ApplyArticleReaction(chefID, 1))
	require.Equal(t, 1, articleReactionCount(t, db, chefID))

	require.NoError(t, ApplyArticleReaction(chefID, 1))
	require.Equal(t, 2, articleReactionCount(t, db, chefID))

	require.NoError(t, ApplyArticleReaction(chefID, -1))
	require.Equal(t, 1, articleReactionCount(t, db, chefID))
}

// A zero delta must not issue a write at all — switching reaction type is the
// common case and it should cost nothing.
func TestApplyArticleReaction_ZeroIsNoOp(t *testing.T) {
	db, chefID := setupAudienceDB(t)

	require.NoError(t, ApplyArticleReaction(chefID, 1))
	require.NoError(t, ApplyArticleReaction(chefID, 0))
	require.Equal(t, 1, articleReactionCount(t, db, chefID))
}

// Reactions removed after a manual data fix must never drive the public count
// negative, which would rank the kitchen below one with no blog at all.
func TestApplyArticleReaction_ClampsAtZero(t *testing.T) {
	db, chefID := setupAudienceDB(t)

	require.NoError(t, ApplyArticleReaction(chefID, -1))
	require.Equal(t, 0, articleReactionCount(t, db, chefID))

	require.NoError(t, ApplyArticleReaction(chefID, 1))
	require.NoError(t, ApplyArticleReaction(chefID, -5))
	require.Equal(t, 0, articleReactionCount(t, db, chefID))
}
