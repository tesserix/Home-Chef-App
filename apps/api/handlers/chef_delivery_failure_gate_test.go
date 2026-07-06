package handlers

// chef_delivery_failure_gate_test.go — #393 chef self-delivery parity. The pure gate:
// only a self-delivery chef, on an order that is out for delivery, may report a delivery
// failure. 3PL `delivery` orders fail via the courier pipeline; pickup is customer-
// collected; an already-delivered/terminal order can't be re-failed.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestChefMayReportDeliveryFailure(t *testing.T) {
	cases := []struct {
		name   string
		status models.OrderStatus
		ft     models.FulfillmentType
		want   bool
	}{
		{"chef-delivery en route", models.OrderStatusDelivering, models.FulfillmentChefDelivery, true},
		{"chef-delivery ready", models.OrderStatusReady, models.FulfillmentChefDelivery, true},
		{"chef-delivery picked up", models.OrderStatusPickedUp, models.FulfillmentChefDelivery, true},
		{"already delivered blocked", models.OrderStatusDelivered, models.FulfillmentChefDelivery, false},
		{"not yet out for delivery", models.OrderStatusPreparing, models.FulfillmentChefDelivery, false},
		{"3PL delivery blocked (courier pipeline owns it)", models.OrderStatusDelivering, models.FulfillmentDelivery, false},
		{"pickup blocked (customer-collected)", models.OrderStatusReady, models.FulfillmentPickup, false},
		{"unset fulfillment blocked", models.OrderStatusDelivering, models.FulfillmentType(""), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, chefMayReportDeliveryFailure(tc.status, tc.ft))
		})
	}
}
