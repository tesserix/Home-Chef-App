package models

import (
	"testing"

	"github.com/google/uuid"
)

// Every partitioned model must default to live when constructed zero-valued.
// NormalizeMode is what guarantees this at read time; this test guards against
// someone "helpfully" changing the default to test during development.
func TestZeroValuedModeReadsLive(t *testing.T) {
	if NormalizeMode((&Order{}).Mode) != ChefModeLive {
		t.Fatal("a zero-valued Order must read as live")
	}
	if NormalizeMode((&Tip{}).Mode) != ChefModeLive {
		t.Fatal("a zero-valued Tip must read as live")
	}
	if NormalizeMode((&Review{}).Mode) != ChefModeLive {
		t.Fatal("a zero-valued Review must read as live")
	}
	if NormalizeMode((&MenuItem{}).Mode) != ChefModeLive {
		t.Fatal("a zero-valued MenuItem must read as live")
	}
	if (&Order{}).IsTest() {
		t.Fatal("a zero-valued Order is not a test row")
	}
}

// A clone is a historical replica of a real customer's order. Distinguishing it
// from something actually done in the sandbox is what keeps it out of that
// customer's order history.
func TestIsCloneTracksProvenance(t *testing.T) {
	if (&Order{}).IsClone() {
		t.Fatal("an ordinary order is not a clone")
	}
	src := uuid.New()
	o := &Order{}
	o.ClonedFromID = &src
	if !o.IsClone() {
		t.Fatal("a row with ClonedFromID set is a clone")
	}
}
