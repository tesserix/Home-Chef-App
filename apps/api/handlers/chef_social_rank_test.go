package handlers

// chef_social_rank_test.go — the weights discovery ranks kitchens by. Asserted
// as a score rather than a string so the test says what the rule means: a
// kitchen cannot out-post a kitchen that customers actually subscribed to.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSocialRankOrder_IsDescendingTiebreaker(t *testing.T) {
	require.Contains(t, socialRankOrder(), "DESC")
}

// A subscription is a standing commitment; a like is one tap. 3:1.
func TestSocialScore_SubscriptionOutweighsLike(t *testing.T) {
	require.Greater(t, socialScore(1, 0, 0), socialScore(0, 2, 0))
	require.Equal(t, socialScore(1, 0, 0), socialScore(0, 3, 0))
}

// Blog engagement counts, but a chef controls how many articles they post and
// not how many people subscribe — so its contribution is capped.
func TestSocialScore_ArticleReactionsAreCapped(t *testing.T) {
	require.Equal(t, socialScore(0, 0, 10), socialScore(0, 10, 0))

	atCap := socialScore(0, 0, articleReactionRankCap)
	require.Equal(t, atCap, socialScore(0, 0, articleReactionRankCap+500))

	// The cap has to be low enough that a prolific blogger cannot overtake a
	// kitchen with a real subscriber base.
	require.Greater(t, socialScore(100, 0, 0), socialScore(0, 0, 10_000))
}
