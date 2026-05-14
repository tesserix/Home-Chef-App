package handlers

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// ExportsHandler serves admin CSV downloads. Every endpoint streams rows
// directly to the client so we don't have to hold a full export in memory.
type ExportsHandler struct{}

func NewExportsHandler() *ExportsHandler { return &ExportsHandler{} }

// parseDateRange pulls ?from=YYYY-MM-DD&to=YYYY-MM-DD off the query string
// and returns (from, to, applied). When neither is provided, applied=false
// and the caller should not filter by date.
func parseDateRange(c *gin.Context) (from, to time.Time, applied bool) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" && toStr == "" {
		return
	}
	if fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = t
			applied = true
		}
	}
	if toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			// Inclusive end-of-day.
			to = t.Add(24 * time.Hour).Add(-time.Second)
			applied = true
		}
	}
	return
}

// writeHeaders sets response headers so the browser treats the payload as a
// CSV download and picks a timestamped filename.
func writeHeaders(c *gin.Context, name string) *csv.Writer {
	filename := fmt.Sprintf("%s-%s.csv", name, time.Now().UTC().Format("2006-01-02"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-store")
	return csv.NewWriter(c.Writer)
}

// flushAndLog is the deferred cleanup for every export handler. It flushes
// the CSV writer's buffer and logs any IO errors so we don't silently ship
// a truncated file with HTTP 200.
func flushAndLog(w *csv.Writer, name string) {
	w.Flush()
	if err := w.Error(); err != nil {
		log.Printf("exports.%s: csv write error: %v", name, err)
	}
}

// ExportUsers dumps the users table. Columns are stable so downstream
// scripts can rely on ordering; edit cautiously.
func (h *ExportsHandler) ExportUsers(c *gin.Context) {
	// Run the query before we write CSV headers so we can still return a
	// clean JSON error if the DB fails. Once headers are flushed a JSON
	// error body would confuse the browser into saving a bad .csv file.
	rows, err := database.DB.Model(&models.User{}).Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	w := writeHeaders(c, "homechef-users")
	defer flushAndLog(w, "users")

	_ = w.Write([]string{"id", "email", "first_name", "last_name", "phone", "role", "is_active", "created_at", "last_login_at"})

	for rows.Next() {
		var u models.User
		if err := database.DB.ScanRows(rows, &u); err != nil {
			continue
		}
		last := ""
		if u.LastLoginAt != nil {
			last = u.LastLoginAt.UTC().Format(time.RFC3339)
		}
		_ = w.Write([]string{
			u.ID.String(),
			u.Email,
			u.FirstName,
			u.LastName,
			u.Phone,
			string(u.Role),
			strconv.FormatBool(u.IsActive),
			u.CreatedAt.UTC().Format(time.RFC3339),
			last,
		})
	}
}

// ExportOrders dumps orders in the optional date range. Filterable by status.
func (h *ExportsHandler) ExportOrders(c *gin.Context) {
	from, to, applied := parseDateRange(c)
	status := c.Query("status")

	query := database.DB.Model(&models.Order{})
	if applied {
		query = query.Where("created_at BETWEEN ? AND ?", from, to)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	rows, err := query.Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	w := writeHeaders(c, "homechef-orders")
	defer flushAndLog(w, "orders")

	_ = w.Write([]string{"order_number", "customer_id", "chef_id", "status", "payment_status", "subtotal", "delivery_fee", "service_fee", "tax", "tip", "discount", "total", "created_at"})

	for rows.Next() {
		var o models.Order
		if err := database.DB.ScanRows(rows, &o); err != nil {
			continue
		}
		_ = w.Write([]string{
			o.OrderNumber,
			o.CustomerID.String(),
			o.ChefID.String(),
			string(o.Status),
			string(o.PaymentStatus),
			strconv.FormatFloat(o.Subtotal, 'f', 2, 64),
			strconv.FormatFloat(o.DeliveryFee, 'f', 2, 64),
			strconv.FormatFloat(o.ServiceFee, 'f', 2, 64),
			strconv.FormatFloat(o.Tax, 'f', 2, 64),
			strconv.FormatFloat(o.Tip, 'f', 2, 64),
			strconv.FormatFloat(o.Discount, 'f', 2, 64),
			strconv.FormatFloat(o.Total, 'f', 2, 64),
			o.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
}

// ExportRevenue is a per-day rollup: date, order_count, gross, service_fees,
// delivery_fees, tax. Handy for finance teams that want a quick trend rather
// than the full order log.
func (h *ExportsHandler) ExportRevenue(c *gin.Context) {
	from, to, applied := parseDateRange(c)

	type dailyRow struct {
		Date         time.Time
		Orders       int64
		Gross        float64
		ServiceFees  float64
		DeliveryFees float64
		Tax          float64
	}
	var rows []dailyRow

	q := database.DB.
		Table("orders").
		Select(`DATE(created_at) AS date,
			COUNT(*) AS orders,
			COALESCE(SUM(total), 0) AS gross,
			COALESCE(SUM(service_fee), 0) AS service_fees,
			COALESCE(SUM(delivery_fee), 0) AS delivery_fees,
			COALESCE(SUM(tax), 0) AS tax`).
		Where("payment_status = ?", "paid").
		Group("DATE(created_at)").
		Order("date DESC")
	if applied {
		q = q.Where("created_at BETWEEN ? AND ?", from, to)
	}
	if err := q.Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	w := writeHeaders(c, "homechef-revenue")
	defer flushAndLog(w, "revenue")

	_ = w.Write([]string{"date", "orders", "gross_revenue", "service_fees", "delivery_fees", "tax_collected"})

	for _, r := range rows {
		_ = w.Write([]string{
			r.Date.UTC().Format("2006-01-02"),
			strconv.FormatInt(r.Orders, 10),
			strconv.FormatFloat(r.Gross, 'f', 2, 64),
			strconv.FormatFloat(r.ServiceFees, 'f', 2, 64),
			strconv.FormatFloat(r.DeliveryFees, 'f', 2, 64),
			strconv.FormatFloat(r.Tax, 'f', 2, 64),
		})
	}
}
