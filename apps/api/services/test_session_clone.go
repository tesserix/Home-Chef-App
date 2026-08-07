package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// partitionedTables is every table carrying the mode/test_session_id/
// cloned_from_id triple. It drives both the clone and the purge, so a table
// added to one is never forgotten by the other.
var partitionedTables = []string{
	"orders", "order_items",
	"group_orders",
	"meal_plans", "meal_plan_days",
	"meal_subscriptions", "meal_trials",
	"catering_requests",
	"tips",
	"chef_promotions",
	"reviews",
	"menu_items",
	"weekly_menus", "weekly_menu_items",
	"daily_menus", "daily_menu_items",
	"chef_schedules",
}

// CloneChefIntoSession copies a live kitchen into a fresh test session.
//
// Depth is CONFIGURATION IN FULL plus a bounded window of order history. That
// reproduces essentially any production issue while completing in seconds; a
// recursive copy of a year of ledger entries would be slower, far more fragile,
// and no more useful for debugging.
//
// Four invariants hold for every step, and the tests enforce all four:
//
//  1. INSERT ONLY, NO HOOKS. Rows are written with SkipHooks so no
//     BeforeSave/AfterCreate fires. Cloning 118 orders must not push 118
//     notifications at a real customer, enqueue 118 NATS events, or start 118
//     Temporal workflows. This is the single biggest correctness risk here.
//  2. NO GATEWAY IDENTIFIERS. Razorpay order/payment/transfer ids are cleared,
//     so a cloned order can be inspected and driven through its status machine
//     but can never be charged or refunded against a real payment.
//  3. PROVENANCE. Every row carries mode=test, the session id, and
//     cloned_from_id pointing at its original — so a clone is always
//     distinguishable from something actually done in the sandbox, and the
//     purge can find every row it created.
//  4. ALL OR NOTHING. The caller runs this inside one transaction. A partial
//     clone would leave a kitchen half-copied and in test mode with no way to
//     tell what is missing.
//
// PII columns (including the #710 encrypted companions) are copied verbatim
// rather than re-encrypted, so no key material is touched.
//
// NOT cloned: ledger entries, payout records, statements, invoices, wallet
// balances and loyalty lots. Those are real-money artefacts — a copy is
// meaningless in a sandbox and dangerous if it ever leaked into reporting.
//
// Returns a JSON object of per-table row counts for the session summary.
func CloneChefIntoSession(tx *gorm.DB, chefID uuid.UUID, session *models.ChefTestSession, windowDays int) (string, error) {
	if windowDays <= 0 {
		windowDays = models.DefaultTestCloneWindowDays
	}
	since := time.Now().AddDate(0, 0, -windowDays)
	counts := map[string]int{}

	// Configuration: copied in full, so the sandbox kitchen behaves exactly like
	// the real one — same menu, same schedule, same capacity, same prices.
	//
	// ORDER MATTERS: a menu's items are remapped onto the cloned header via its
	// cloned_from_id, so the header must already exist when the items are copied.
	cfg := []struct {
		table     string
		where     string
		args      []any
		join      string
		overrides map[string]any
	}{
		{table: "menu_items", where: "t.chef_id = ? AND t.mode = ? AND t.deleted_at IS NULL", args: []any{chefID, models.ChefModeLive}},
		{table: "chef_schedules", where: "t.chef_id = ? AND t.mode = ?", args: []any{chefID, models.ChefModeLive}},
		{table: "weekly_menus", where: "t.chef_id = ? AND t.mode = ?", args: []any{chefID, models.ChefModeLive}},
		// Keyed by chef_id rather than by the weekly_menus row, so no remap.
		{table: "weekly_menu_items", where: "t.chef_id = ? AND t.mode = ?", args: []any{chefID, models.ChefModeLive}},
		{table: "daily_menus", where: "t.chef_id = ? AND t.mode = ?", args: []any{chefID, models.ChefModeLive}},
		// daily_menu_items DOES point at its header by id, so daily_menu_id has to
		// be rewritten to the clone's id — copying it verbatim would leave the
		// sandbox menu reading through to live rows.
		{
			table: "daily_menu_items",
			where: "t.chef_id = ? AND t.mode = ?",
			args:  []any{chefID, models.ChefModeLive},
			join: "JOIN daily_menus p ON p.cloned_from_id = t.daily_menu_id AND p.test_session_id = '" +
				session.ID.String() + "'",
			overrides: map[string]any{"daily_menu_id": sqlExpr("p.id")},
		},
	}
	for _, c := range cfg {
		n, err := cloneRows(tx, c.table, session, c.where, c.args, c.overrides, c.join)
		if err != nil {
			return "", err
		}
		counts[c.table] = n
	}

	// History: a bounded window, enough to debug against the order that broke.
	// Gateway identifiers are blanked so a replica can never move real money.
	//
	// order_number is DERIVED, not copied: it is globally unique (invoicing keys
	// off it), so a verbatim copy collides with its own original. The session
	// suffix also keeps two sessions over the same window from colliding with
	// each other, since a closed session's rows survive until an explicit purge.
	n, err := cloneRows(tx, "orders", session,
		"t.chef_id = ? AND t.mode = ? AND t.created_at >= ? AND t.deleted_at IS NULL",
		[]any{chefID, models.ChefModeLive, since},
		map[string]any{
			"order_number":        sqlExpr("t.order_number || '-T" + shortID(session.ID) + "'"),
			"gateway_order_id":   "",
			"gateway_payment_id": "",
			"payout_transfer_id":  "",
			"refund_id":           "",
		}, "")
	if err != nil {
		return "", err
	}
	counts["orders"] = n

	raw, err := json.Marshal(counts)
	if err != nil {
		return "", fmt.Errorf("clone: marshal summary: %w", err)
	}
	return string(raw), nil
}

// sqlExpr is an override rendered as raw SQL instead of a quoted literal, so a
// cloned column can be DERIVED from the row being copied (a per-session order
// number) or from a joined table (a remapped parent id) rather than being set to
// a constant. Only ever constructed from values this package supplies.
type sqlExpr string

// shortID is the leading segment of a UUID — enough to keep per-session derived
// values distinct without making them unwieldy in an admin UI.
func shortID(id uuid.UUID) string { return id.String()[:8] }

// cloneRows copies matching rows of one table into the test partition.
//
// Implemented as INSERT … SELECT rather than load-mutate-save because it is a
// pure data copy: no model hooks fire, no events are enqueued, and no
// notifications are sent.
//
// The column list is NOT taken from the live schema. It is the intersection of
// the live schema with the allow-list in test_sync_classification.go, and a
// live column missing from that allow-list FAILS the clone. Deriving the list
// from the schema — as this used to — meant any column added later was copied
// into the test partition automatically, with nobody reviewing whether it was a
// payment token or a bank account number (#797).
//
// The source table is aliased `t`, so `where` and `overrides` must qualify their
// columns. An optional `join` brings in another table — used to remap a child
// row onto its cloned parent via the parent's cloned_from_id.
//
// overrides blanks or replaces specific columns on the copy: a plain value
// becomes a literal, an sqlExpr is spliced in as SQL.
func cloneRows(tx *gorm.DB, table string, session *models.ChefTestSession, where string, args []any, overrides map[string]any, join string) (int, error) {
	live, err := tableColumns(tx, table)
	if err != nil {
		return 0, fmt.Errorf("clone: columns for %s: %w", table, err)
	}
	if len(live) == 0 {
		return 0, nil // table absent in this environment (sqlite fixtures)
	}

	// Fail closed: an unclassified column stops the clone rather than riding
	// along unnoticed.
	cols, classes, err := classifyColumns(table, live)
	if err != nil {
		return 0, err
	}

	selects := make([]string, 0, len(cols))
	for _, col := range cols {
		if v, ok := overrides[col]; ok {
			selects = append(selects, quoteLiteral(v))
			continue
		}
		switch col {
		case "id":
			// A fresh identity per copy. The original is recorded in
			// cloned_from_id, so provenance survives.
			selects = append(selects, newIDExpr(tx))
			continue
		case "mode":
			selects = append(selects, "'"+models.ChefModeTest+"'")
			continue
		case "test_session_id":
			selects = append(selects, "'"+session.ID.String()+"'")
			continue
		case "cloned_from_id":
			selects = append(selects, "t.id")
			continue
		}
		switch classes[col] {
		case classBlank:
			// Gateway identifiers. Blanked from the classification rather than
			// relying on the caller remembering an override — a forgotten
			// override would silently ship a live payment id into test.
			selects = append(selects, "''")
		case classDerive:
			// Derived columns MUST be supplied by the caller. Falling back to a
			// verbatim copy would reintroduce the collision the class exists to
			// prevent (a globally-unique order_number duplicated from its own
			// source), so this fails closed instead.
			return 0, fmt.Errorf(
				"clone: %s.%s is classified as derived but no override was supplied — "+
					"a derived column must never fall back to a verbatim copy", table, col)
		default:
			selects = append(selects, "t."+col)
		}
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s t %s WHERE %s",
		table, joinCols(cols), joinCols(selects), table, join, where)
	res := tx.Exec(sql, args...)
	if res.Error != nil {
		return 0, fmt.Errorf("clone: copy %s: %w", table, res.Error)
	}
	return int(res.RowsAffected), nil
}

// newIDExpr returns the dialect's fresh-UUID expression. Postgres has
// gen_random_uuid(); the sqlite test driver has no UUID function, so tests get
// a deterministic-enough hex string of the right shape.
func newIDExpr(tx *gorm.DB) string {
	if tx.Dialector.Name() == "sqlite" {
		return "lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || " +
			"substr(hex(randomblob(2)),2) || '-a' || substr(hex(randomblob(2)),2) || " +
			"'-' || hex(randomblob(6)))"
	}
	return "gen_random_uuid()"
}

func joinCols(c []string) string {
	out := ""
	for i, s := range c {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

// quoteLiteral renders an override value as a SQL literal. Only ever called
// with values this package supplies (blank gateway ids), never with user input.
func quoteLiteral(v any) string {
	switch t := v.(type) {
	case sqlExpr:
		return string(t) // raw SQL by construction — see sqlExpr
	case string:
		if t == "" {
			return "''"
		}
		return "'" + t + "'"
	case nil:
		return "NULL"
	default:
		return fmt.Sprintf("%v", t)
	}
}

// tableColumns lists a table's columns, so the clone copies whatever the schema
// currently has rather than a list that silently rots.
func tableColumns(tx *gorm.DB, table string) ([]string, error) {
	names, err := tx.Migrator().ColumnTypes(table)
	if err != nil {
		// A table that doesn't exist in this environment is not an error — the
		// sqlite fixtures only create the tables a given test needs.
		return nil, nil
	}
	out := make([]string, 0, len(names))
	for _, c := range names {
		out = append(out, c.Name())
	}
	return out, nil
}
