package models

import "testing"

func TestToResponse_FulfillmentTypeDefaultsToDelivery(t *testing.T) {
	o := Order{} // unset
	if got := o.ToResponse().FulfillmentType; got != FulfillmentDelivery {
		t.Fatalf("empty FulfillmentType should normalize to delivery, got %q", got)
	}
	o2 := Order{FulfillmentType: FulfillmentPickup}
	if got := o2.ToResponse().FulfillmentType; got != FulfillmentPickup {
		t.Fatalf("want pickup, got %q", got)
	}
}
