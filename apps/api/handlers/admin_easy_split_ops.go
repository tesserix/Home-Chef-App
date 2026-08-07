package handlers

// admin_easy_split_ops.go — the two reads behind the Easy Split control surface
// in tesserix-home (#1085). Read-only: every write on that screen already has
// an endpoint (the rollout switch, the settings, the vendor registration).

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

const (
	opsDefaultWindowDays = 7
	opsMaxWindowDays     = 90
	opsDefaultLimit      = 100
	opsMaxLimit          = 500
)

func opsWindowDays(c *gin.Context) int {
	days, err := strconv.Atoi(strings.TrimSpace(c.Query("days")))
	if err != nil || days <= 0 {
		return opsDefaultWindowDays
	}
	if days > opsMaxWindowDays {
		return opsMaxWindowDays
	}
	return days
}

type easySplitRosterRow struct {
	ChefID        string `json:"chefId"`
	BusinessName  string `json:"businessName"`
	Mode          string `json:"mode"`
	VendorID      string `json:"vendorId"`
	VendorStatus  string `json:"vendorStatus"`
	EasySplitMode string `json:"easySplitMode"`
	Effective     bool   `json:"effective"`
	Payable       bool   `json:"payable"`
	Blocker       string `json:"blocker"`
	SplitOrders   int    `json:"splitOrders"`
	SplitPaise    int    `json:"splitPaise"`
}

// GetEasySplitRoster is the vendor roster: every chef, and whether an order for
// them would split today. "The flag is on" is not the same question — a chef can
// be enabled and still unpayable on this rail, which is what the blocker names.
//
// GET /admin/payouts/easy-split/roster?days=&query=&state=payable|blocked
func (h *AdminPayoutRailHandler) GetEasySplitRoster(c *gin.Context) {
	db := database.DB
	days := opsWindowDays(c)
	since := time.Now().AddDate(0, 0, -days)

	q := db.Model(&models.ChefProfile{})
	if name := strings.TrimSpace(c.Query("query")); name != "" {
		q = q.Where("LOWER(business_name) LIKE ?", "%"+strings.ToLower(name)+"%")
	}
	var chefs []models.ChefProfile
	if err := q.Order("business_name ASC").Find(&chefs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load chefs"})
		return
	}

	type splitAgg struct {
		ChefID string
		Orders int
		Paise  int
	}
	var aggs []splitAgg
	if err := db.Model(&models.Order{}).
		Select("chef_id, COUNT(*) as orders, COALESCE(SUM(gateway_split_paise), 0) as paise").
		Where("gateway_split_paise > 0 AND created_at >= ?", since).
		Group("chef_id").Scan(&aggs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load split totals"})
		return
	}
	byChef := make(map[string]splitAgg, len(aggs))
	for _, a := range aggs {
		byChef[a.ChefID] = a
	}

	state := strings.ToLower(strings.TrimSpace(c.Query("state")))
	rows := make([]easySplitRosterRow, 0, len(chefs))
	for i := range chefs {
		chef := &chefs[i]
		blocker := services.EasySplitChefBlocker(db, chef)
		if (state == "payable" && blocker != "") || (state == "blocked" && blocker == "") {
			continue
		}
		agg := byChef[chef.ID.String()]
		rows = append(rows, easySplitRosterRow{
			ChefID: chef.ID.String(), BusinessName: chef.BusinessName, Mode: chef.Mode,
			VendorID: chef.CashfreeVendorID, VendorStatus: chef.CashfreeVendorStatus,
			EasySplitMode: chef.EasySplitMode,
			Effective:     services.EasySplitEnabledForChef(db, chef),
			Payable:       blocker == "", Blocker: blocker,
			SplitOrders: agg.Orders, SplitPaise: agg.Paise,
		})
	}

	feeMinor, feeReadable := services.PlatformFeeFlatMinor(db)
	c.JSON(http.StatusOK, gin.H{
		"globalEnabled":    services.EasySplitEnabled(db),
		"windowFits":       services.EasySplitWindowFits(db),
		"platformFeeMinor": feeMinor,
		"feeReadable":      feeReadable,
		"days":             days,
		"chefs":            rows,
	})
}

type easySplitOrderRow struct {
	OrderID       string    `json:"orderId"`
	OrderNumber   string    `json:"orderNumber"`
	ChefID        string    `json:"chefId"`
	ChefName      string    `json:"chefName"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
	TotalPaise    int       `json:"totalPaise"`
	ExpectedPaise int       `json:"expectedPaise"`
	SplitPaise    int       `json:"splitPaise"`
	DeltaPaise    int       `json:"deltaPaise"`
	Rail          string    `json:"rail"`
	Reason        string    `json:"reason"`
	Exception     bool      `json:"exception"`
}

// GetEasySplitOrders is the per-order settlement view and, filtered, the
// exception queue: which rail settled each paid order, what the chef's share
// should have been, and what actually left the capture.
//
// Reconciliation here is expected-vs-stamped, not expected-vs-Cashfree: the
// stamp is written only after Cashfree accepts the split, so a delta means our
// own two figures disagree — which is the case an operator can act on. A
// settlement-report comparison needs Cashfree's payout file and is separate.
//
// GET /admin/payouts/easy-split/orders?days=&rail=split|payout|exception&limit=
func (h *AdminPayoutRailHandler) GetEasySplitOrders(c *gin.Context) {
	db := database.DB
	days := opsWindowDays(c)
	since := time.Now().AddDate(0, 0, -days)

	limit, err := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	if err != nil || limit <= 0 {
		limit = opsDefaultLimit
	}
	if limit > opsMaxLimit {
		limit = opsMaxLimit
	}

	var orders []models.Order
	if err := db.Preload("Chef").
		// Refunded orders stay in view: a refund does not un-split the order, and
		// the clawback is exactly the kind of delta this screen exists to show.
		Where("payment_status IN ? AND created_at >= ?",
			[]models.PaymentStatus{models.PaymentCompleted, models.PaymentRefunded}, since).
		Order("created_at DESC").Limit(limit).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load orders"})
		return
	}

	reasons := easySplitSkipReasons(orderIDStrings(orders))
	fee, feeReadable := services.PlatformFeeFlatMinor(db)

	rail := strings.ToLower(strings.TrimSpace(c.Query("rail")))
	rows := make([]easySplitOrderRow, 0, len(orders))
	var splitCount, splitPaise, payoutCount, exceptionCount, deltaPaise int
	for i := range orders {
		order := &orders[i]
		expected := services.ToPaise(services.ChefNetPayoutFor(order))
		if feeReadable {
			expected -= int(fee)
		}
		if expected < 0 {
			expected = 0
		}
		row := easySplitOrderRow{
			OrderID: order.ID.String(), OrderNumber: order.OrderNumber,
			ChefID: order.ChefID.String(), ChefName: order.Chef.BusinessName,
			Status: string(order.Status), CreatedAt: order.CreatedAt,
			TotalPaise: services.ToPaise(order.Total), ExpectedPaise: expected,
			SplitPaise: order.GatewaySplitPaise, Rail: "payout",
			Reason: reasons[order.ID.String()],
		}
		if order.GatewaySplitPaise > 0 {
			row.Rail = "split"
			row.DeltaPaise = order.GatewaySplitPaise - expected
			row.Exception = row.DeltaPaise != 0
		} else {
			// An unexplained payout-rail order for a chef who is split-payable
			// right now is the case worth looking at: something refused at release
			// time and left no reason behind.
			row.Exception = row.Reason == "" &&
				services.EasySplitChefBlocker(db, &order.Chef) == ""
		}

		switch row.Rail {
		case "split":
			splitCount++
			splitPaise += order.GatewaySplitPaise
			deltaPaise += row.DeltaPaise
		default:
			payoutCount++
		}
		if row.Exception {
			exceptionCount++
		}

		switch rail {
		case "split", "payout":
			if row.Rail != rail {
				continue
			}
		case "exception":
			if !row.Exception {
				continue
			}
		}
		rows = append(rows, row)
	}

	c.JSON(http.StatusOK, gin.H{
		"days":   days,
		"orders": rows,
		"summary": gin.H{
			"splitCount": splitCount, "splitPaise": splitPaise,
			"payoutCount": payoutCount, "exceptionCount": exceptionCount,
			"deltaPaise": deltaPaise,
		},
	})
}

func orderIDStrings(orders []models.Order) []string {
	ids := make([]string, 0, len(orders))
	for i := range orders {
		ids = append(ids, orders[i].ID.String())
	}
	return ids
}

// easySplitSkipReasons reads back the reason each order took the payout rail
// from the audit row the release wrote (#1084) — the stored answer, not one
// re-derived from chef state that has since moved on.
func easySplitSkipReasons(orderIDs []string) map[string]string {
	out := map[string]string{}
	if len(orderIDs) == 0 {
		return out
	}
	var logs []models.AuditLog
	if err := database.DB.
		Where("action = ? AND entity_id IN ?", "order.payout.easy_split_skipped", orderIDs).
		Order("created_at ASC").Find(&logs).Error; err != nil {
		return out
	}
	for i := range logs {
		var payload struct {
			Reason string `json:"reason"`
		}
		if json.Unmarshal([]byte(logs[i].NewValue), &payload) == nil && payload.Reason != "" {
			out[logs[i].EntityID] = payload.Reason
		}
	}
	return out
}
