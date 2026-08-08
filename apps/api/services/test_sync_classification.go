package services

import (
	"fmt"
	"sort"
	"strings"
)

// test_sync_classification.go — the allow-list that governs what the live→test
// clone may copy (#797).
//
// WHY THIS EXISTS. cloneRows used to derive its column list straight from the
// live schema, documented as a convenience: "a column added later is copied
// automatically without touching this code". That is precisely the behaviour
// the isolation spec prohibits —
//
//	UNKNOWN FIELD → AUTOMATIC COPY         forbidden
//	UNKNOWN FIELD → DENY UNTIL CLASSIFIED  required
//
// — because it means adding a bank account number, a payment token or a PAN to
// any cloned table silently ships it into the test partition on the next clone,
// with nobody reviewing the decision.
//
// So the schema no longer decides. This file does. A column that exists in the
// database but is absent here FAILS the clone loudly (see classifyColumns).
// That is deliberate: a build-breaking error is a cheap price for never again
// leaking a field into test by omission.
//
// WHEN YOU ADD A COLUMN to one of these tables, the clone will start failing
// with the column named. Classify it here — do not reach for a wildcard.

// syncClass is how one column is treated when a live row is copied into test.
type syncClass int

const (
	// classCopy — safe to copy verbatim from live.
	classCopy syncClass = iota

	// classBlank — the column is emptied on the copy. Used for gateway
	// identifiers: a cloned order must be inspectable and drivable through its
	// status machine, but must never carry an id that could be charged or
	// refunded against a real payment.
	classBlank

	// classBlankNull — classBlank for a column whose type has no empty value:
	// a uuid foreign key cannot hold '', so emptiness is NULL.
	classBlankNull

	// classDerive — the value is computed from the original rather than copied.
	// Used where a column is globally unique and a verbatim copy would collide
	// with its own source (order_number).
	classDerive

	// classPartition — mode/test_session_id/cloned_from_id. Set by the clone
	// itself, never taken from the source row.
	classPartition

	// classNeverCopy — must never enter the test dataset. Listing a column here
	// is stronger than omitting it: omission is an unclassified-field error,
	// this is a recorded decision that the value stays behind.
	classNeverCopy
)

// clonedTableColumns classifies every column of every table CloneChefIntoSession
// copies. Tables that are only ever purged (the rest of partitionedTables) do
// not appear — nothing is copied into them, so there is nothing to classify.
//
// NOTE ON PII. Several order columns below carry real customer data — delivery
// address lines and their #710 encrypted companions. They are classCopy today
// because that is the behaviour that already shipped, and silently changing it
// here would be a second undocumented decision. Making it explicit is the point:
// it is now visible and reviewable. #798 removes orders from the clone entirely,
// which retires the question rather than answering it halfway.
var clonedTableColumns = map[string]map[string]syncClass{
	"menu_items": {
		"id": classCopy, "chef_id": classCopy, "category_id": classCopy,
		"name": classCopy, "description": classCopy, "price": classCopy,
		"compare_price": classCopy, "image_url": classCopy, "dietary_tags": classCopy,
		"allergens": classCopy, "ingredients": classCopy, "prep_time": classCopy,
		"portion_size": classCopy, "serves": classCopy, "spice_level": classCopy,
		"is_available": classCopy, "is_approved": classCopy, "is_featured": classCopy,
		"total_orders": classCopy, "rating": classCopy, "total_reviews": classCopy,
		"sort_order": classCopy, "created_at": classCopy, "updated_at": classCopy,
		"deleted_at": classCopy, "is_veg": classCopy, "hsn": classCopy,
		"daily_capacity": classCopy, "is_combo": classCopy, "available_days": classCopy,
		"mode": classPartition, "test_session_id": classPartition, "cloned_from_id": classPartition,
	},
	"chef_schedules": {
		"id": classCopy, "chef_id": classCopy, "day_of_week": classCopy,
		"open_time": classCopy, "close_time": classCopy, "is_closed": classCopy,
		"created_at": classCopy, "updated_at": classCopy,
		"mode": classPartition, "test_session_id": classPartition, "cloned_from_id": classPartition,
	},
	"weekly_menus": {
		"id": classCopy, "chef_id": classCopy, "is_published": classCopy,
		"published_at": classCopy, "created_at": classCopy, "updated_at": classCopy,
		"mode": classPartition, "test_session_id": classPartition, "cloned_from_id": classPartition,
	},
	"weekly_menu_items": {
		"id": classCopy, "chef_id": classCopy, "day_of_week": classCopy,
		"slot": classCopy, "variant": classCopy, "name": classCopy,
		"description": classCopy, "price": classCopy, "image_url": classCopy,
		"dietary_tags": classCopy, "allergens": classCopy, "menu_item_id": classCopy,
		"created_at": classCopy, "updated_at": classCopy, "is_combo": classCopy,
		"combo_components": classCopy,
		"portion_size":     classCopy, "serves": classCopy,
		"mode": classPartition, "test_session_id": classPartition, "cloned_from_id": classPartition,
	},
	"daily_menus": {
		"id": classCopy, "chef_id": classCopy, "date": classCopy,
		"is_published": classCopy, "published_at": classCopy,
		"created_at": classCopy, "updated_at": classCopy,
		"mode": classPartition, "test_session_id": classPartition, "cloned_from_id": classPartition,
	},
	"daily_menu_items": {
		// daily_menu_id is remapped onto the cloned parent by an override, not
		// copied — see CloneChefIntoSession. Classified copy so the column is
		// selected at all; the override then replaces the value.
		"id": classCopy, "daily_menu_id": classCopy, "chef_id": classCopy,
		"date": classCopy, "slot": classCopy, "variant": classCopy,
		"name": classCopy, "description": classCopy, "price": classCopy,
		"image_url": classCopy, "dietary_tags": classCopy, "allergens": classCopy,
		"menu_item_id": classCopy, "sort_order": classCopy, "created_at": classCopy,
		"updated_at": classCopy, "is_combo": classCopy, "combo_components": classCopy,
		"portion_size": classCopy, "serves": classCopy,
		"mode": classPartition, "test_session_id": classPartition, "cloned_from_id": classPartition,
	},
	"orders": {
		"id": classCopy, "order_number": classDerive, "customer_id": classCopy,
		"chef_id": classCopy, "delivery_id": classCopy, "status": classCopy,
		"payment_status": classCopy, "payment_method": classCopy, "subtotal": classCopy,
		"delivery_fee": classCopy, "service_fee": classCopy, "tax": classCopy,
		"tip": classCopy, "chef_tip": classCopy, "driver_tip": classCopy,
		"discount": classCopy, "total": classCopy, "promo_code": classCopy,
		// Real customer PII. classCopy preserves shipped behaviour; #798 removes
		// orders from the clone, which is the actual fix.
		"delivery_address_line1": classCopy, "delivery_address_line2": classCopy,
		"delivery_address_city": classCopy, "delivery_address_state": classCopy,
		"delivery_address_postal_code": classCopy, "delivery_latitude": classCopy,
		"delivery_longitude": classCopy, "delivery_instructions": classCopy,
		"delivery_address_line1_enc": classCopy, "delivery_address_line2_enc": classCopy,
		"delivery_address_country": classCopy,
		"estimated_prep_time":      classCopy, "estimated_delivery_time": classCopy,
		"scheduled_for": classCopy, "accepted_at": classCopy, "prepared_at": classCopy,
		"picked_up_at": classCopy, "delivered_at": classCopy, "cancelled_at": classCopy,
		"cancel_reason": classCopy, "special_instructions": classCopy,
		// Gateway identifiers — blanked so a replica can never move real money.
		"stripe_payment_intent_id": classBlank, "gateway_order_id": classBlank,
		"gateway_payment_id": classBlank, "refund_id": classBlank,
		"refunded_at": classCopy, "refund_amount": classCopy, "refund_reason": classCopy,
		"refund_initiated_by": classCopy, "created_at": classCopy, "updated_at": classCopy,
		"deleted_at": classCopy, "payment_provider": classCopy, "tax_rate": classCopy,
		"tax_name": classCopy, "currency": classCopy, "wallet_applied": classCopy,
		"delivery_slot": classCopy, "chef_funded_discount": classCopy,
		"fulfillment_type": classCopy, "ready_photo_url": classCopy,
		"handover_photo_url": classCopy, "payout_hold_status": classCopy,
		"customer_confirmed_at": classCopy, "payout_settled_at": classCopy,
		"payout_settle_attempts": classCopy, "commission_rate": classCopy,
		"accept_reminder_count": classCopy, "last_accept_reminder_at": classCopy,
		"requested_fulfillment_at": classCopy, "confirmed_fulfillment_at": classCopy,
		"fulfillment_time_status": classCopy, "delivery_fee_final": classCopy,
		"loyalty_applied": classCopy, "loyalty_points_spent": classCopy,
		"wallet_refunded": classCopy, "loyalty_refunded": classCopy,
		"payout_transfer_id": classBlank,
		// The per-component tax breakdown, alongside the tax/tax_rate above.
		"tax_food": classCopy, "tax_delivery": classCopy, "tax_service": classCopy,
		"tax_rate_food": classCopy, "tax_rate_delivery": classCopy, "tax_rate_service": classCopy,
		"tax_inclusive": classCopy, "tax_service_inclusive": classCopy,
		"tax_delivery_by_platform": classCopy,
		"delivery_fee_source":      classCopy, "chef_tip_at": classCopy,
		"stale_reminder_count": classCopy, "last_stale_reminder_at": classCopy,
		"settled_net_payout": classCopy, "gateway_split_paise": classCopy,
		// Dropped, not copied: it points at a real weekly statement the clone was
		// never billed on, and carrying it would tie a replica to real payout.
		"billed_statement_id": classBlankNull,
		"mode":                classPartition, "test_session_id": classPartition, "cloned_from_id": classPartition,
	},
}

// ErrUnclassifiedColumn is returned when the live schema carries a column this
// file does not classify. Deliberately fatal to the clone rather than skipped:
// a skipped column is a silent behaviour change, and an unclassified one may be
// the sensitive field this whole mechanism exists to catch.
type ErrUnclassifiedColumn struct {
	Table   string
	Columns []string
}

func (e *ErrUnclassifiedColumn) Error() string {
	return fmt.Sprintf(
		"clone: %s has unclassified column(s) %s — classify them in "+
			"services/test_sync_classification.go before they can be cloned "+
			"(unknown fields are denied by default, never copied)",
		e.Table, strings.Join(e.Columns, ", "))
}

// classifyColumns resolves the live column list for a table against the
// allow-list.
//
// Returns the columns to copy and their classes. Errors if the table is not
// classified at all, or if any live column is unclassified — fail closed.
//
// A column classified here but ABSENT from the database is not an error: the
// sqlite fixtures create narrower tables than production, and a column removed
// from prod should not break the clone before the entry is tidied up.
func classifyColumns(table string, liveColumns []string) ([]string, map[string]syncClass, error) {
	classes, ok := clonedTableColumns[table]
	if !ok {
		return nil, nil, fmt.Errorf(
			"clone: table %s is not classified in services/test_sync_classification.go — "+
				"a table may not be cloned until every column it carries is classified", table)
	}

	cols := make([]string, 0, len(liveColumns))
	var unknown []string
	for _, c := range liveColumns {
		class, ok := classes[c]
		if !ok {
			unknown = append(unknown, c)
			continue
		}
		if class == classNeverCopy {
			continue // recorded decision: stays behind
		}
		cols = append(cols, c)
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, nil, &ErrUnclassifiedColumn{Table: table, Columns: unknown}
	}
	return cols, classes, nil
}
