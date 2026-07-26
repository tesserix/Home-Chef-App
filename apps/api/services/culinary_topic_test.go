package services

import (
	"strings"
	"testing"
)

// The gate's whole job is the difference between "clearly about food",
// "clearly not", and "can't tell" — so those three outcomes are what's tested,
// including the cases where naive substring matching would get it wrong.

func TestCheckCulinaryTopic(t *testing.T) {
	long := func(filler string, n int) string {
		out := ""
		for i := 0; i < n; i++ {
			out += filler + " "
		}
		return out
	}

	cases := []struct {
		name        string
		title, body string
		wantOnTopic bool
		wantWeak    bool
	}{
		{
			name:        "a recipe post is on topic",
			title:       "Dalma, three ways",
			body:        "Pressure cook the dal with pumpkin, then finish with a ghee tadka.",
			wantOnTopic: true,
		},
		{
			name:        "signal from the title alone still counts",
			title:       "My biryani method",
			body:        "It took me years to get this right and I want to share it.",
			wantOnTopic: true,
		},
		{
			name:        "unrelated content is refused",
			title:       "Buy cheap flights now",
			body:        "Click the link below for the best deals on travel insurance today.",
			wantOnTopic: false,
		},
		{
			name:        "empty content is refused",
			wantOnTopic: false,
		},
		{
			// One passing mention buried in a long off-topic essay is exactly
			// the case a plain contains() check would wave through.
			name:        "a lone mention in a long body is weak",
			title:       "A long week",
			body:        "rice " + long("we drove around and talked about the weather and the traffic", 12),
			wantOnTopic: true,
			wantWeak:    true,
		},
		{
			name:        "several terms in a long body is not weak",
			title:       "Sunday cooking",
			body:        "paneer masala roti " + long("we drove around and talked about the weather", 12),
			wantOnTopic: true,
			wantWeak:    false,
		},
		{
			// Word-boundary matching: neither of these contains a culinary
			// word, but substring matching would find "pan" and "art"/"tart".
			name:        "substrings do not count as matches",
			title:       "Spanner and artwork",
			body:        "I spent the weekend sorting the garage shelves and the toolbox.",
			wantOnTopic: false,
		},
		{
			name:        "punctuation and hashtags do not break matching",
			title:       "#Paneer!",
			body:        "Marinate, then grill.",
			wantOnTopic: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckCulinaryTopic(tc.title, tc.body)
			if got.OnTopic != tc.wantOnTopic {
				t.Fatalf("OnTopic = %v, want %v (matches=%d)", got.OnTopic, tc.wantOnTopic, got.Matches)
			}
			if got.OnTopic && got.Weak != tc.wantWeak {
				t.Fatalf("Weak = %v, want %v (matches=%d)", got.Weak, tc.wantWeak, got.Matches)
			}
		})
	}
}

// Slugify builds the shareable URL, so its edge cases are worth pinning: an
// emoji-only title still has to produce a usable link.
func TestSlugify(t *testing.T) {
	cases := []struct{ in, suffix, want string }{
		{"Dalma, three ways", "abc123", "dalma-three-ways-abc123"},
		{"  Spaced   Out  ", "x", "spaced-out-x"},
		{"Ghee & Garlic!!", "y", "ghee-garlic-y"},
		// No alphanumerics survive, so fall back rather than emit a bare suffix.
		{"🍛🍛🍛", "z", "article-z"},
		{"No suffix", "", "no-suffix"},
	}
	for _, tc := range cases {
		if got := Slugify(tc.in, tc.suffix); got != tc.want {
			t.Errorf("Slugify(%q, %q) = %q, want %q", tc.in, tc.suffix, got, tc.want)
		}
	}

	// Long titles are truncated without leaving a trailing hyphen.
	long := "the quick brown fox jumps over the lazy dog while cooking a very large pot of dalma for everyone"
	got := Slugify(long, "s")
	if len(got) > 70 {
		t.Errorf("slug too long: %q", got)
	}
	if strings.Contains(got, "--") {
		t.Errorf("slug has a doubled hyphen: %q", got)
	}
}
