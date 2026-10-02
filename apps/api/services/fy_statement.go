package services

// fy_statement.go — annual income & expense statement for a chef, in their
// market's tax year (India/NZ Apr–Mar, Australia Jul–Jun).
//
// Earnings side re-derives from delivered orders exactly like the TDS
// certificate — including gateway-split (Easy Split) orders, because the FY
// statement reports INCOME, not what the weekly settlement rail still owes.
// Expense side sums the chef's self-declared ChefExpense rows. The result
// backs both the analytics JSON and the downloadable PDF used at tax time.

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// ParseISTDate parses a YYYY-MM-DD string as an IST calendar day (UTC stored).
func ParseISTDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", s, istLoc)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// FYLabel renders "FY 2025-26" for a start year.
func FYLabel(fyStartYear int) string {
	return fmt.Sprintf("FY %d-%02d", fyStartYear, (fyStartYear+1)%100)
}

// ExpenseCategoryTotal is one category's FY total.
type ExpenseCategoryTotal struct {
	Category models.ChefExpenseCategory `json:"category"`
	Amount   float64                    `json:"amount"`
	Count    int                        `json:"count"`
}

// ExpenseMonthTotal is one FY month's total (YYYY-MM in IST).
type ExpenseMonthTotal struct {
	Month  string  `json:"month"`
	Amount float64 `json:"amount"`
}

// ExpenseSummary is the FY expense aggregate for analytics.
type ExpenseSummary struct {
	FYStartYear int                    `json:"fyStartYear"`
	FYLabel     string                 `json:"fyLabel"`
	Currency    string                 `json:"currency"`
	Total       float64                `json:"total"`
	Count       int                    `json:"count"`
	ByCategory  []ExpenseCategoryTotal `json:"byCategory"`
	ByMonth     []ExpenseMonthTotal    `json:"byMonth"`
}

// ComputeExpenseSummary aggregates one chef's expenses across a financial year.
func ComputeExpenseSummary(chefID uuid.UUID, country string, fyStartYear int) (*ExpenseSummary, error) {
	cal := FiscalCalendarFor(country)
	start, end := cal.Window(fyStartYear)

	var expenses []models.ChefExpense
	if err := database.DB.
		Where("chef_id = ? AND expense_date >= ? AND expense_date < ?", chefID, start, end).
		Order("expense_date ASC").
		Find(&expenses).Error; err != nil {
		return nil, fmt.Errorf("load fy expenses: %w", err)
	}

	summary := &ExpenseSummary{
		FYStartYear: fyStartYear,
		FYLabel:     FYLabel(fyStartYear),
		Currency:    ReportingCurrency(country),
		ByCategory:  []ExpenseCategoryTotal{},
		ByMonth:     []ExpenseMonthTotal{},
	}

	byCat := map[models.ChefExpenseCategory]*ExpenseCategoryTotal{}
	byMonth := map[string]float64{}
	monthOrder := []string{}
	for _, e := range expenses {
		summary.Total += e.Amount
		summary.Count++
		ct := byCat[e.Category]
		if ct == nil {
			ct = &ExpenseCategoryTotal{Category: e.Category}
			byCat[e.Category] = ct
		}
		ct.Amount += e.Amount
		ct.Count++
		month := cal.Month(e.ExpenseDate)
		if _, seen := byMonth[month]; !seen {
			monthOrder = append(monthOrder, month)
		}
		byMonth[month] += e.Amount
	}

	// Categories in declared order so the UI renders stably.
	for _, cat := range []models.ChefExpenseCategory{
		models.ExpenseIngredients, models.ExpenseGas, models.ExpenseUtensils,
		models.ExpensePackaging, models.ExpenseTransport, models.ExpenseEquipment,
		models.ExpenseUtilities, models.ExpenseOther,
	} {
		if ct := byCat[cat]; ct != nil {
			ct.Amount = Round2(ct.Amount)
			summary.ByCategory = append(summary.ByCategory, *ct)
		}
	}
	for _, month := range monthOrder {
		summary.ByMonth = append(summary.ByMonth, ExpenseMonthTotal{
			Month: month, Amount: Round2(byMonth[month]),
		})
	}
	summary.Total = Round2(summary.Total)
	return summary, nil
}

// FYQuarter is one statutory quarter's earnings line on the FY statement.
type FYQuarter struct {
	Label       string  `json:"label"`
	OrdersCount int     `json:"ordersCount"`
	Gross       float64 `json:"gross"`
	TDS         float64 `json:"tds"`
	NetPayout   float64 `json:"netPayout"`
}

// FYStatement is the full income & expense picture for one financial year.
type FYStatement struct {
	FYStartYear int    `json:"fyStartYear"`
	FYLabel     string `json:"fyLabel"`
	Period      string `json:"period"`
	Currency    string `json:"currency"`

	// Income side — derived from delivered orders (Easy Split included).
	OrdersCount   int     `json:"ordersCount"`
	FoodRevenue   float64 `json:"foodRevenue"`  // item revenue net of chef-funded discounts
	GSTCollected  float64 `json:"gstCollected"` // food GST charged to customers (outward supply tax)
	Tips          float64 `json:"tips"`
	GrossReceipts float64 `json:"grossReceipts"` // foodRevenue + gstCollected + tips

	PlatformCommission float64 `json:"platformCommission"`
	CommissionCGST     float64 `json:"commissionCgst"` // GST charged on commission (ITC-eligible)
	CommissionSGST     float64 `json:"commissionSgst"`
	CommissionIGST     float64 `json:"commissionIgst"`
	TDSWithheld        float64 `json:"tdsWithheld"` // Section 194-O, credit vs income tax
	NetEarnings        float64 `json:"netEarnings"` // grossReceipts − commission − TDS

	Quarters []FYQuarter `json:"quarters"`

	// Expense side — the chef's self-declared books.
	Expenses      *ExpenseSummary `json:"expenses"`
	TotalExpenses float64         `json:"totalExpenses"`

	// NetIncome = netEarnings − totalExpenses (before income tax).
	NetIncome float64 `json:"netIncome"`
}

// ComputeFYStatement aggregates one chef's FY earnings and expenses.
func ComputeFYStatement(chefID uuid.UUID, fyStartYear int) (*FYStatement, error) {
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		return nil, fmt.Errorf("chef not found: %w", err)
	}

	cal := FiscalCalendarFor(chef.PayoutCountry)
	start, end := cal.Window(fyStartYear)

	// Same income basis as the TDS certificate: every delivered order in the
	// FY, split-settled or not, with the per-order frozen commission rate.
	var rows []statementOrderRow
	err := database.DB.Raw(`
		SELECT o.id, o.order_number, o.delivered_at, o.subtotal, o.tax, o.currency, o.tax_inclusive,
		       o.tax_food, o.tax_service, o.chef_funded_discount,
		       o.delivery_fee, o.chef_tip, o.driver_tip, o.delivery_address_state, o.commission_rate,
		       o.fulfillment_type, o.delivery_fee_final,
		       o.chef_id, c.user_id, c.state AS chef_state
		FROM   orders o
		JOIN   chef_profiles c ON c.id = o.chef_id
		WHERE  o.chef_id       = ?
		AND    o.status        = 'delivered'
		AND    o.delivered_at >= ?
		AND    o.delivered_at  < ?
		AND    o.deleted_at    IS NULL
		ORDER  BY o.delivered_at ASC
	`, chefID, start, end).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load fy orders: %w", err)
	}

	stmt := &FYStatement{
		FYStartYear: fyStartYear,
		FYLabel:     FYLabel(fyStartYear),
		Period:      cal.PeriodLabel(fyStartYear),
		Currency:    ReportingCurrency(chef.PayoutCountry),
	}
	for _, label := range cal.QuarterLabels() {
		stmt.Quarters = append(stmt.Quarters, FYQuarter{Label: label})
	}

	for _, r := range rows {
		// No live rate is resolved here, so a legacy 0 falls through to
		// DefaultCommissionRate inside ComputeOrderEarnings — intended.
		e := ComputeOrderEarnings(r.earningsInput(0), chef.State)

		stmt.OrdersCount++
		stmt.FoodRevenue += r.ItemRevenue - r.ChefFundedDiscount
		stmt.GSTCollected += ChefTaxOf(r.Tax, r.TaxFood, r.TaxService)
		stmt.Tips += r.ChefTip
		stmt.GrossReceipts += e.Gross
		stmt.PlatformCommission += e.PlatformCommission
		stmt.CommissionCGST += e.CGST
		stmt.CommissionSGST += e.SGST
		stmt.CommissionIGST += e.IGST
		stmt.TDSWithheld += e.TDS
		stmt.NetEarnings += e.NetPayout

		q := &stmt.Quarters[cal.Quarter(r.CompletedAt)]
		q.OrdersCount++
		q.Gross += e.Gross
		q.TDS += e.TDS
		q.NetPayout += e.NetPayout
	}

	expenses, err := ComputeExpenseSummary(chefID, chef.PayoutCountry, fyStartYear)
	if err != nil {
		return nil, err
	}
	stmt.Expenses = expenses
	stmt.TotalExpenses = expenses.Total

	stmt.FoodRevenue = Round2(stmt.FoodRevenue)
	stmt.GSTCollected = Round2(stmt.GSTCollected)
	stmt.Tips = Round2(stmt.Tips)
	stmt.GrossReceipts = Round2(stmt.GrossReceipts)
	stmt.PlatformCommission = Round2(stmt.PlatformCommission)
	stmt.CommissionCGST = Round2(stmt.CommissionCGST)
	stmt.CommissionSGST = Round2(stmt.CommissionSGST)
	stmt.CommissionIGST = Round2(stmt.CommissionIGST)
	stmt.TDSWithheld = Round2(stmt.TDSWithheld)
	stmt.NetEarnings = Round2(stmt.NetEarnings)
	for i := range stmt.Quarters {
		stmt.Quarters[i].Gross = Round2(stmt.Quarters[i].Gross)
		stmt.Quarters[i].TDS = Round2(stmt.Quarters[i].TDS)
		stmt.Quarters[i].NetPayout = Round2(stmt.Quarters[i].NetPayout)
	}
	stmt.NetIncome = Round2(stmt.NetEarnings - stmt.TotalExpenses)
	return stmt, nil
}
