package services

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// InvoiceCompanyInfo holds Fe3dr company details for invoices
type InvoiceCompanyInfo struct {
	Name    string
	Address string
	Email   string
	Phone   string
	Website string
	TaxID   string
}

// GetCompanyInfo loads Fe3dr company details from PlatformSettings
func GetCompanyInfo() (*InvoiceCompanyInfo, error) {
	info := &InvoiceCompanyInfo{
		Name:    "Fe3dr Technologies Pvt. Ltd.",
		Address: "",
		Email:   "hello@fe3dr.com",
		Phone:   "",
		Website: "https://fe3dr.com",
		TaxID:   "",
	}

	var settings []models.PlatformSettings
	database.DB.Where("key LIKE ?", "invoice.company_%").Find(&settings)

	for _, s := range settings {
		switch s.Key {
		case "invoice.company_name":
			info.Name = s.Value
		case "invoice.company_address":
			info.Address = s.Value
		case "invoice.company_email":
			info.Email = s.Value
		case "invoice.company_phone":
			info.Phone = s.Value
		case "invoice.company_website":
			info.Website = s.Value
		case "invoice.company_tax_id":
			info.TaxID = s.Value
		}
	}

	return info, nil
}

// GenerateOrderInvoice creates an invoice for a completed customer food order.
// The order must be preloaded with Items, Chef (with User), and Customer.
func GenerateOrderInvoice(order *models.Order) (*models.OrderInvoice, error) {
	// Check if invoice already exists for this order
	var existing models.OrderInvoice
	if err := database.DB.Where("order_id = ?", order.ID).First(&existing).Error; err == nil {
		return &existing, nil
	}

	// Country drives the tax config. Tax is owed where the CUSTOMER
	// takes delivery, not where the chef operates, so we read
	// DeliveryAddressCountry which was stamped on the order at creation
	// (from the customer's delivery address). Falls back to chef country
	// then IN for orders created before the column was populated.
	countryCode := order.DeliveryAddressCountry
	if countryCode == "" {
		if order.Chef.PayoutCountry != "" {
			countryCode = order.Chef.PayoutCountry
		} else {
			countryCode = "IN"
		}
	}

	rates := order.SnapshotRates()

	// Load company info
	companyInfo, err := GetCompanyInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to load company info: %w", err)
	}

	// Calculate food subtotal from items
	var subtotal float64
	lineItems := make([]models.InvoiceLineItem, len(order.Items))
	for i, item := range order.Items {
		itemTotal := models.RoundAmount(item.Price * float64(item.Quantity))
		subtotal += itemTotal
		// Append the selected add-ons to the line name so the invoice shows what
		// the customer paid for (the deltas are already in item.Price) (#232).
		name := item.Name
		if mods := item.ParsedModifiers(); len(mods) > 0 {
			parts := make([]string, len(mods))
			for j, m := range mods {
				parts[j] = m.OptionName
			}
			name = name + " (" + strings.Join(parts, ", ") + ")"
		}
		lineItems[i] = models.InvoiceLineItem{
			Name:      name,
			Quantity:  item.Quantity,
			UnitPrice: item.Price,
			Total:     itemTotal,
		}
	}
	subtotal = models.RoundAmount(subtotal)

	// The tax the order was CHARGED, per supply — never recomputed here. This used
	// to re-derive it from live rates and produced a stored invoice whose total
	// disagreed with the payment (₹266.14 against a ₹264.57 charge on
	// HC26080306287649), which GET /orders/:id/invoice then served to the customer.
	p := order.ToResponse()
	subtotal = p.Subtotal
	foodTax := order.TaxFood
	deliveryFee := p.DeliveryFee
	deliveryTax := order.TaxDelivery
	platformFee := p.PlatformFee
	serviceTax := order.TaxService
	tip := p.Tip
	discount := p.Discount
	// Orders placed before tax was split per supply carry only a total; it was one
	// supply at one rate, so it all sits on the food line.
	if foodTax == 0 && serviceTax == 0 && deliveryTax == 0 {
		foodTax = p.Tax
	}
	totalAmount := p.Total

	// Serialize line items to JSON
	lineItemsJSON, err := json.Marshal(lineItems)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal line items: %w", err)
	}

	// Generate invoice number
	invoiceNumber := generateOrderInvoiceNumber()

	// Build customer address string
	customerAddress := strings.Join(filterEmpty([]string{
		order.DeliveryAddressLine1,
		order.DeliveryAddressLine2,
		order.DeliveryAddressCity,
		order.DeliveryAddressState,
		order.DeliveryAddressPostalCode,
	}), ", ")

	// Build chef address string
	chefAddress := strings.Join(filterEmpty([]string{
		order.Chef.AddressLine1,
		order.Chef.AddressLine2,
		order.Chef.City,
		order.Chef.State,
		order.Chef.PostalCode,
	}), ", ")

	// Currency preference order: the order's stamped currency (set at
	// creation from the chef's settlement country) → the platform_settings
	// override for this country → INR default. Using the order's currency
	// first keeps the invoice line items consistent with what was charged.
	currency := order.Currency
	if currency == "" {
		currency = "INR"
		var currSetting models.PlatformSettings
		if err := database.DB.Where("key = ?", fmt.Sprintf("currency.%s.default", countryCode)).
			First(&currSetting).Error; err == nil {
			currency = currSetting.Value
		}
	}

	// Determine company tax ID — the jurisdiction's own registration wins over the
	// global company one when set.
	companyTaxID := companyInfo.TaxID
	if jurisdiction := ResolveTaxRate(countryCode, order.DeliveryAddressState); jurisdiction.CompanyTaxID != "" {
		companyTaxID = jurisdiction.CompanyTaxID
	}

	now := time.Now().UTC()

	invoice := models.OrderInvoice{
		OrderID:       order.ID,
		InvoiceNumber: invoiceNumber,
		CustomerID:    order.CustomerID,
		ChefID:        order.ChefID,

		Subtotal:    subtotal,
		FoodTax:     foodTax,
		DeliveryFee: deliveryFee,
		DeliveryTax: deliveryTax,
		PlatformFee: platformFee,
		ServiceTax:  serviceTax,
		Tip:         tip,
		Discount:    discount,
		TotalAmount: totalAmount,

		CountryCode: countryCode,
		Currency:    currency,
		// The rates FROZEN on the order, not today's — an admin retuning a rate must
		// not restate an invoice already issued.
		TaxName:            order.TaxName,
		FoodTaxPercent:     rates.Food,
		ServiceTaxPercent:  rates.Service,
		DeliveryTaxPercent: rates.Delivery,

		CustomerName:    order.Customer.FirstName + " " + order.Customer.LastName,
		CustomerEmail:   order.Customer.Email,
		CustomerPhone:   order.Customer.Phone,
		CustomerAddress: customerAddress,
		ChefName:        order.Chef.BusinessName,
		ChefAddress:     chefAddress,

		CompanyName:    companyInfo.Name,
		CompanyAddress: companyInfo.Address,
		CompanyTaxID:   companyTaxID,

		LineItems: string(lineItemsJSON),
		IssuedAt:  now,
	}

	if err := database.DB.Create(&invoice).Error; err != nil {
		return nil, fmt.Errorf("failed to create order invoice: %w", err)
	}

	log.Printf("Generated order invoice %s for order %s", invoiceNumber, order.ID)
	return &invoice, nil
}

// GenerateSubscriptionInvoiceData takes an existing SubscriptionInvoice and returns
// a structured map with all fields needed to render the invoice (for PDF generation, emails, etc.)
func GenerateSubscriptionInvoiceData(invoice *models.SubscriptionInvoice) (map[string]interface{}, error) {
	// Load subscription with user
	var sub models.Subscription
	if err := database.DB.Preload("User").First(&sub, invoice.SubscriptionID).Error; err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	taxCfg := ResolveTaxRate(sub.CountryCode, "")

	// Load company info
	companyInfo, err := GetCompanyInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to load company info: %w", err)
	}

	// Determine plan label
	planLabel := string(sub.BillingInterval) + " " + string(sub.SubscriberType) + " subscription"

	data := map[string]interface{}{
		// Invoice details
		"invoiceId":     invoice.ID.String(),
		"invoiceNumber": invoice.InvoiceNumber,
		"status":        string(invoice.Status),
		"issuedAt":      invoice.CreatedAt.Format(time.RFC3339),

		// Company details
		"company": map[string]interface{}{
			"name":    companyInfo.Name,
			"address": companyInfo.Address,
			"email":   companyInfo.Email,
			"phone":   companyInfo.Phone,
			"website": companyInfo.Website,
			"taxId":   companyInfo.TaxID,
		},

		// Subscriber details
		"subscriber": map[string]interface{}{
			"name":  sub.User.FirstName + " " + sub.User.LastName,
			"email": sub.User.Email,
			"phone": sub.User.Phone,
			"type":  string(sub.SubscriberType),
		},

		// Plan details
		"plan": map[string]interface{}{
			"label":       planLabel,
			"interval":    string(sub.BillingInterval),
			"amount":      sub.PlanAmount,
			"currency":    sub.Currency,
			"countryCode": sub.CountryCode,
		},

		// Billing period
		"period": map[string]interface{}{
			"start": invoice.PeriodStart.Format(time.RFC3339),
			"end":   invoice.PeriodEnd.Format(time.RFC3339),
		},

		// Tax breakdown
		"tax": map[string]interface{}{
			"name":      taxCfg.TaxName,
			"percent":   taxCfg.SubscriptionPercent,
			"amount":    invoice.TaxAmount,
			"idLabel":   taxCfg.RegistrationIDLabel,
			"companyId": taxCfg.CompanyTaxID,
		},

		// Totals
		"subtotal":    invoice.Amount,
		"taxAmount":   invoice.TaxAmount,
		"totalAmount": invoice.TotalAmount,
		"currency":    invoice.Currency,

		// Payment info
		"payment": map[string]interface{}{
			"gateway":  invoice.PaymentGateway,
			"paidAt":   invoice.PaidAt,
			"attempts": invoice.AttemptCount,
		},

		// Earnings context
		"earningsAtGeneration": invoice.EarningsAtGeneration,
	}

	return data, nil
}

// generateOrderInvoiceNumber generates a unique invoice number for order invoices
// Format: FE3DR-ORD-YYYYMMDD-XXXX
func generateOrderInvoiceNumber() string {
	now := time.Now().UTC()
	dateStr := now.Format("20060102")

	// Load prefix from settings, default to FE3DR-ORD
	prefix := "FE3DR-ORD"
	var prefixSetting models.PlatformSettings
	if err := database.DB.Where("key = ?", "invoice.order_prefix").First(&prefixSetting).Error; err == nil {
		prefix = prefixSetting.Value
	}

	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based
		return fmt.Sprintf("%s-%s-%04d", prefix, dateStr, now.UnixNano()%10000)
	}

	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 4)
	for i, v := range b {
		result[i] = chars[int(v)%len(chars)]
	}

	return fmt.Sprintf("%s-%s-%s", prefix, dateStr, string(result))
}

// filterEmpty removes empty strings from a slice
func filterEmpty(ss []string) []string {
	var result []string
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			result = append(result, s)
		}
	}
	return result
}

// GetOrderInvoiceByOrderID fetches or generates an invoice for an order
func GetOrderInvoiceByOrderID(orderID uuid.UUID) (*models.OrderInvoice, error) {
	// Try to find existing invoice
	var invoice models.OrderInvoice
	if err := database.DB.Where("order_id = ?", orderID).First(&invoice).Error; err == nil {
		return &invoice, nil
	}

	// Load the full order and generate on the fly
	var order models.Order
	if err := database.DB.Preload("Items").Preload("Chef").Preload("Customer").
		First(&order, orderID).Error; err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// Only generate invoices for delivered orders
	if order.Status != models.OrderStatusDelivered {
		return nil, fmt.Errorf("invoice not available: order status is %s", order.Status)
	}

	return GenerateOrderInvoice(&order)
}
