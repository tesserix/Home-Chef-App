package models

import "testing"

func TestChefToResponse_OffersPickup(t *testing.T) {
	c := ChefProfile{OffersPickup: true}
	if !c.ToResponse().OffersPickup {
		t.Fatal("ChefProfileResponse should carry OffersPickup")
	}
}
