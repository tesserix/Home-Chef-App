package services

import (
	"regexp"
	"strings"
)

// culinary_topic.go — the topic gate for ChefBook (#social).
//
// ChefBook is a place for chefs to write about food and cooking. Without a
// gate it becomes a general-purpose blog attached to a food brand, which is
// both off-strategy and a moderation liability.
//
// This is a LEXICON HEURISTIC, not a classifier, and it is deliberately shaped
// around that limitation:
//
//   - Content with NO culinary signal at all is rejected. That case is
//     unambiguous and cheap to detect.
//   - Content with a weak signal is accepted but flagged for a human. Guessing
//     silently in the grey zone is how legitimate posts get eaten.
//
// It is not a safety filter and makes no claim to catch harmful content — the
// existing contact-info moderation and the admin queue remain responsible for
// that. Word-boundary matching keeps "panner" out of "spanner" and stops
// "art" matching inside "tart".

// Ingredients, techniques, dishes, cuisines, equipment and the vocabulary of
// eating. Indian terms are first-class here: this is an India-first product
// and a post about dalma or a bhuna is squarely on topic.
var culinaryTerms = []string{
	// Core nouns
	"food", "foods", "dish", "dishes", "meal", "meals", "recipe", "recipes",
	"cook", "cooks", "cooking", "cooked", "cuisine", "kitchen", "chef", "chefs",
	"ingredient", "ingredients", "flavour", "flavor", "flavours", "flavors",
	"taste", "tasty", "aroma", "seasoning", "menu", "eat", "eating", "edible",
	"breakfast", "lunch", "dinner", "brunch", "snack", "snacks", "dessert",
	"desserts", "appetiser", "appetizer", "starter", "starters", "thali",
	"tiffin", "platter", "portion", "serving", "servings", "leftovers",
	// Techniques
	"bake", "baked", "baking", "boil", "boiled", "boiling", "fry", "fried",
	"frying", "saute", "sauteed", "roast", "roasted", "roasting", "grill",
	"grilled", "grilling", "steam", "steamed", "steaming", "simmer",
	"simmered", "marinate", "marinated", "marinade", "knead", "kneading",
	"ferment", "fermented", "fermentation", "temper", "tempering", "tadka",
	"whisk", "whisked", "blend", "blended", "garnish", "garnished", "plating",
	"caramelise", "caramelize", "braise", "braised", "poach", "poached",
	"sear", "seared", "proof", "proofing", "sourdough", "batter", "dough",
	// Ingredients
	"rice", "wheat", "flour", "atta", "maida", "besan", "lentil", "lentils",
	"dal", "daal", "paneer", "cheese", "butter", "ghee", "oil", "milk",
	"cream", "curd", "yoghurt", "yogurt", "egg", "eggs", "chicken", "mutton",
	"lamb", "fish", "prawn", "prawns", "seafood", "meat", "vegetable",
	"vegetables", "veg", "nonveg", "vegan", "vegetarian", "potato", "onion",
	"tomato", "garlic", "ginger", "chilli", "chili", "pepper", "salt", "sugar",
	"jaggery", "honey", "spice", "spices", "masala", "turmeric", "cumin",
	"coriander", "cardamom", "cinnamon", "clove", "cloves", "mustard",
	"fenugreek", "asafoetida", "hing", "herb", "herbs", "basil", "mint",
	"curry", "gravy", "stock", "broth", "sauce", "chutney", "pickle", "achar",
	"bread", "roti", "chapati", "paratha", "naan", "puri", "dosa", "idli",
	"vada", "sambar", "upma", "poha", "biryani", "pulao", "khichdi", "dalma",
	"kheer", "halwa", "laddu", "jalebi", "gulab", "rasgulla", "barfi",
	"samosa", "pakora", "chaat", "tikka", "kebab", "korma", "bhuna", "kassa",
	"pasta", "pizza", "noodle", "noodles", "soup", "salad", "sandwich",
	"cake", "cookie", "cookies", "pastry", "chocolate", "coffee", "tea",
	"chai", "juice", "smoothie", "lassi", "beverage", "beverages", "drink",
	// Kitchen and craft
	"pan", "pot", "skillet", "wok", "kadai", "tawa", "oven", "stove", "griddle",
	"pressure", "cooker", "utensil", "utensils", "recipe", "nutrition",
	"protein", "calorie", "calories", "diet", "dietary", "allergen",
	"allergens", "gluten", "organic", "farmers", "harvest", "seasonal",
	"pantry", "hygiene", "fssai",
}

var culinarySet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(culinaryTerms))
	for _, t := range culinaryTerms {
		m[t] = struct{}{}
	}
	return m
}()

// Split on anything that isn't a letter or digit so punctuation, hashtags and
// emoji don't weld themselves onto words.
var wordSplit = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// CulinaryVerdict is the outcome of the topic check.
type CulinaryVerdict struct {
	// OnTopic is false when nothing culinary was found at all. The caller
	// should refuse the write.
	OnTopic bool
	// Weak is true when the signal was thin enough to be worth a human look.
	// The caller should still accept the write and set TopicFlagged.
	Weak bool
	// Matches is how many distinct culinary terms were found — useful in logs
	// when someone asks why a post was refused.
	Matches int
}

// Thresholds. A single mention across a long article is more likely a passing
// reference than a food post, so the bar scales with length: short posts need
// one term, longer ones need a second before they stop looking incidental.
const (
	weakSignalWordCount = 60
	strongSignalTerms   = 2
)

// CheckCulinaryTopic decides whether ChefBook content is about food.
//
// Both the title and the body are considered, so "Dalma, three ways" carries
// its own signal even if the body is discursive.
func CheckCulinaryTopic(title, body string) CulinaryVerdict {
	combined := strings.ToLower(strings.TrimSpace(title + " " + body))
	if combined == "" {
		return CulinaryVerdict{OnTopic: false}
	}

	words := wordSplit.Split(combined, -1)
	seen := make(map[string]struct{})
	total := 0
	for _, w := range words {
		if w == "" {
			continue
		}
		total++
		if _, ok := culinarySet[w]; ok {
			seen[w] = struct{}{}
		}
	}

	matches := len(seen)
	if matches == 0 {
		return CulinaryVerdict{OnTopic: false, Matches: 0}
	}

	weak := total > weakSignalWordCount && matches < strongSignalTerms
	return CulinaryVerdict{OnTopic: true, Weak: weak, Matches: matches}
}
