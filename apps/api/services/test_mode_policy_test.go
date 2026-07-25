package services

import "testing"

func TestMayViewTestChefs(t *testing.T) {
	p := TestModePolicy{ViewerEmails: []string{"Samyak.Rout@Gmail.com", " unidevidp@gmail.com "}}

	for _, ok := range []string{
		"samyak.rout@gmail.com", "SAMYAK.ROUT@GMAIL.COM", "unidevidp@gmail.com", " unidevidp@gmail.com ",
	} {
		if !p.MayViewTestChefs(ok) {
			t.Fatalf("%q must be allowed (case-insensitive, whitespace-trimmed)", ok)
		}
	}
	for _, no := range []string{"", "   ", "someone@else.com", "samyak.rout@gmail.com.evil.com"} {
		if p.MayViewTestChefs(no) {
			t.Fatalf("%q must NOT be allowed", no)
		}
	}
}

// An anonymous caller has no email. It must never match, including against a
// policy that has accidentally been saved with an empty string in the list —
// otherwise every logged-out visitor would see every sandbox kitchen.
func TestAnonymousNeverMatches(t *testing.T) {
	p := TestModePolicy{ViewerEmails: []string{"", "  "}}
	if p.MayViewTestChefs("") {
		t.Fatal("anonymous callers must never see test chefs")
	}
	if p.MayViewTestChefs("   ") {
		t.Fatal("a whitespace-only email must never match a whitespace-only entry")
	}
}

func TestDefaultPolicySeedsTheThreeTesters(t *testing.T) {
	d := DefaultTestModePolicy()
	for _, want := range []string{
		"samyak.rout@gmail.com", "unidevidp@gmail.com", "mahesh.sangawar@gmail.com",
	} {
		if !d.MayViewTestChefs(want) {
			t.Fatalf("default policy must seed %s", want)
		}
	}
	if d.MayViewTestChefs("stranger@example.com") {
		t.Fatal("the default policy must not admit anyone else")
	}
}
