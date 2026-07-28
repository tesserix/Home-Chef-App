package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// query_filters.go — shared parsing for list-shaped query parameters.
//
// Every order list in the product has tabs that span several statuses ("Live
// orders" is pending + accepted + preparing + ready), and every client
// expresses that the obvious way: one repeated parameter, comma separated.
//
//	GET /chef/orders?status=pending,accepted,preparing
//
// The handlers read it with c.Query("status") and compared it with `=`, so the
// SQL asked for a single row whose status was literally the string
// "pending,accepted,preparing". No row can match that, so the endpoint answered
// 200 with an empty list — indistinguishable from "you have no orders".
//
// That is exactly what a chef saw: a paid order sitting in the database, the
// vendor dashboard insisting there was nothing there, and no error anywhere to
// suggest otherwise.

// statusValues splits a `status` query parameter into the statuses it names.
//
// Returns nil when nothing usable was sent, which callers MUST treat as "no
// filter" rather than "match nothing" — an empty IN () excludes everything and
// would reintroduce the same silent-empty bug from the other direction.
func statusValues(c *gin.Context) []string {
	return splitCSVParam(c.Query("status"))
}

// splitCSVParam turns "a, b ,,c" into ["a","b","c"]. Trims each value and drops
// empties so a trailing comma or a stray space can't smuggle "" into an IN list
// and match rows with no status.
func splitCSVParam(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
