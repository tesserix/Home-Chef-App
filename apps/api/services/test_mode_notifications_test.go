package services

import (
	"testing"

	"github.com/homechef/api/models"
)

func TestTagSubjectForMode(t *testing.T) {
	if got := TagSubjectForMode(models.ChefModeTest, "Your order is on the way"); got != "[TEST] Your order is on the way" {
		t.Fatalf("got %q", got)
	}
	if got := TagSubjectForMode(models.ChefModeLive, "Your order is on the way"); got != "Your order is on the way" {
		t.Fatalf("a live notification must be untouched, got %q", got)
	}
	// Subjects are composed from several helpers; a doubled prefix reads as a bug.
	if got := TagSubjectForMode(models.ChefModeTest, "[TEST] Already tagged"); got != "[TEST] Already tagged" {
		t.Fatalf("tagging must be idempotent, got %q", got)
	}
	// An unknown mode is live (see models.NormalizeMode), so it must not tag.
	if got := TagSubjectForMode("garbage", "Untouched"); got != "Untouched" {
		t.Fatalf("an unrecognised mode must not tag, got %q", got)
	}
}
