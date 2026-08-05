package handlers

import "fmt"

// Audience size, earned rather than bought: a kitchen customers subscribe to,
// like, and react to on ChefBook ranks above one they don't.
//
// The weights say what each signal costs the customer. A subscription is a
// standing commitment, so it carries 3; a like is one tap, so it carries 1.
//
// ChefBook reactions carry 1 each but are capped, because they are the one
// signal a chef partly controls: they choose how many articles to post, and
// each new article is a fresh chance for the same reader to react. The cap
// keeps a good blog worth roughly a dozen subscribers and no more.
const articleReactionRankCap = 40

// socialScore is the ranking contribution of one kitchen's audience. It exists
// so the weights can be tested as arithmetic instead of as a SQL string;
// socialRankOrder is the same expression for Postgres.
func socialScore(subscribers, likes, articleReactions int) int {
	if articleReactions > articleReactionRankCap {
		articleReactions = articleReactionRankCap
	}
	return subscribers*3 + likes + articleReactions
}

// socialRankOrder is always DESC and always a TIEBREAKER, never the primary
// key — someone who asked to sort by price wants price order, so popularity
// only separates chefs the chosen sort ties. The one exception is the default
// sort, where no key was asked for and popularity leads.
//
// CASE rather than LEAST for the cap: the same expression has to run on
// Postgres in prod and SQLite under test, and LEAST does not exist on SQLite.
func socialRankOrder() string {
	capped := fmt.Sprintf(
		"CASE WHEN article_reaction_count > %d THEN %d ELSE article_reaction_count END",
		articleReactionRankCap, articleReactionRankCap)
	return "(subscriber_count * 3 + like_count + " + capped + ") DESC"
}
