package services

// The reminder and the auto-confirm must answer "is this order still waiting to
// be confirmed?" the same way. They did not: a cancelled order whose payout hold
// had never been moved kept pushing "Did your order arrive?" (#931).

import (
	"testing"
	"time"

	"github.com/homechef/api/models"
)

func TestOrderAwaitingReceiptConfirmation(t *testing.T) {
	now := time.Now()
	awaiting := func() models.Order {
		return models.Order{
			Status:           models.OrderStatusDelivered,
			PayoutHoldStatus: models.PayoutHoldAwaitingConfirmation,
		}
	}

	cases := []struct {
		name  string
		order models.Order
		want  bool
	}{
		{"a delivered order on an awaiting hold is the whole point", awaiting(), true},
		{
			name: "already confirmed",
			order: func() models.Order {
				o := awaiting()
				o.CustomerConfirmedAt = &now
				return o
			}(),
			want: false,
		},
		{
			name: "the hold has moved on",
			order: func() models.Order {
				o := awaiting()
				o.PayoutHoldStatus = models.PayoutHoldReleaseEligible
				return o
			}(),
			want: false,
		},
		{
			// The live defect: cancelled, but the hold was never moved off
			// awaiting_customer_confirmation, so the reminder fired anyway.
			name: "cancelled while the hold is still awaiting",
			order: func() models.Order {
				o := awaiting()
				o.Status = models.OrderStatusCancelled
				return o
			}(),
			want: false,
		},
		{
			name: "refunded status",
			order: func() models.Order {
				o := awaiting()
				o.Status = models.OrderStatusRefunded
				return o
			}(),
			want: false,
		},
		{
			name: "rejected by the chef — nothing can have arrived",
			order: func() models.Order {
				o := awaiting()
				o.Status = models.OrderStatusRejected
				return o
			}(),
			want: false,
		},
		{
			// refunded_at set without a terminal status: money went back, so stop
			// asking regardless of what `status` says.
			name: "refunded_at stamped on a still-delivered order",
			order: func() models.Order {
				o := awaiting()
				o.RefundedAt = &now
				return o
			}(),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := orderAwaitingReceiptConfirmation(&tc.order); got != tc.want {
				t.Errorf("orderAwaitingReceiptConfirmation() = %v, want %v", got, tc.want)
			}
		})
	}
}
