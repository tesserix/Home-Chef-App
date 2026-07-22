package services

// chef_availability_event.go — fan-out for a chef opening or closing shop.
//
// Closing a kitchen used to tell nobody: the handler flipped accepting_orders
// and returned. The customer app derives isOpen from that flag but only learns
// about it when its chef query refetches, which on a two-minute staleTime means
// a closed kitchen keeps taking orders for minutes.
//
// Staging the change on the outbox makes it durable and lets it fan out the
// same way every other state change does.

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChefAvailabilityEvent is the payload consumers act on.
//
// It carries the resulting state rather than just "something changed", so a
// consumer never has to re-read the row — which would race the next toggle and
// could publish a stale value.
type ChefAvailabilityEvent struct {
	ChefID          uuid.UUID `json:"chefId"`
	AcceptingOrders bool      `json:"acceptingOrders"`
}

// EnqueueChefAvailabilityChanged stages the change on the outbox.
//
// Pass the transaction that flipped the flag so the event cannot be published
// for a change that later rolls back — the dual-write problem #121 exists to
// avoid.
func EnqueueChefAvailabilityChanged(tx *gorm.DB, chefID uuid.UUID, acceptingOrders bool) error {
	return EnqueueOutbox(
		tx,
		SubjectChefAvailabilityChanged,
		"chef",
		chefID.String(),
		ChefAvailabilityEvent{ChefID: chefID, AcceptingOrders: acceptingOrders},
	)
}
