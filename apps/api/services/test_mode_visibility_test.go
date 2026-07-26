package services

import (
	"testing"
	"time"

	"github.com/homechef/api/models"
)

func TestChefVisibilityMatrix(t *testing.T) {
	now := time.Now()
	live := &models.ChefProfile{Mode: models.ChefModeLive}
	bornTest := &models.ChefProfile{Mode: models.ChefModeTest}
	flipped := &models.ChefProfile{Mode: models.ChefModeTest, FirstLiveAt: &now}

	cases := []struct {
		name  string
		chef  *models.ChefProfile
		email string
		want  ChefVisibilityDecision
	}{
		{"live chef, anonymous", live, "", VisibilityFull},
		{"live chef, tester", live, "samyak.rout@gmail.com", VisibilityFull},

		// A kitchen nobody has heard of should not exist for anyone but a tester.
		{"born-test, anonymous", bornTest, "", VisibilityHidden},
		{"born-test, stranger", bornTest, "someone@else.com", VisibilityHidden},
		{"born-test, tester", bornTest, "samyak.rout@gmail.com", VisibilityFull},

		// A kitchen with regulars should not vanish — it should look closed.
		{"flipped, anonymous", flipped, "", VisibilityClosed},
		{"flipped, stranger", flipped, "someone@else.com", VisibilityClosed},
		{"flipped, tester", flipped, "unidevidp@gmail.com", VisibilityFull},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ChefVisibility(tc.chef, tc.email); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// A nil chef must not panic and must not be treated as hidden — a missing
// preload would otherwise blank the customer's whole listing.
func TestChefVisibilityNilIsFull(t *testing.T) {
	if ChefVisibility(nil, "") != VisibilityFull {
		t.Fatal("a nil chef must resolve to full visibility, not hidden")
	}
}
