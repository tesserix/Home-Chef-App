package services

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// TaxConfig holds per-country tax configuration loaded from PlatformSettings
type TaxConfig struct {
	CountryCode         string
	TaxName             string  // GST, VAT, Sales Tax
	FoodPercent         float64
	ServicePercent      float64
	DeliveryPercent     float64
	SubscriptionPercent float64
	RegistrationIDLabel string // GSTIN, ABN, TIN
	CompanyTaxID        string
}

// InvoiceCompanyInfo holds Fe3dr company details for invoices
type InvoiceCompanyInfo struct {
	Name    string
	Address string
	Email   string
	Phone   string
	Website string
	TaxID   string
}

// Default tax configurations per country
var defaultTaxConfigs = map[string]TaxConfig{
	"IN": {CountryCode: "IN", TaxName: "GST", FoodPercent: 5, ServicePercent: 18, DeliveryPercent: 18, SubscriptionPercent: 18, RegistrationIDLabel: "GSTIN", CompanyTaxID: ""},
	"AU": {CountryCode: "AU", TaxName: "GST", FoodPercent: 10, ServicePercent: 10, DeliveryPercent: 10, SubscriptionPercent: 10, RegistrationIDLabel: "ABN", CompanyTaxID: ""},
	"PK": {CountryCode: "PK", TaxName: "Sales Tax", FoodPercent: 17, ServicePercent: 17, DeliveryPercent: 17, SubscriptionPercent: 17, RegistrationIDLabel: "NTN", CompanyTaxID: ""},
	"BD": {CountryCode: "BD", TaxName: "VAT", FoodPercent: 15, ServicePercent: 15, DeliveryPercent: 15, SubscriptionPercent: 15, RegistrationIDLabel: "TIN", CompanyTaxID: ""},
	"LK": {CountryCode: "LK", TaxName: "VAT", FoodPercent: 8, ServicePercent: 8, DeliveryPercent: 8, SubscriptionPercent: 8, RegistrationIDLabel: "TIN", CompanyTaxID: ""},
	"NP": {CountryCode: "NP", TaxName: "VAT", FoodPercent: 13, ServicePercent: 13, DeliveryPercent: 13, SubscriptionPercent: 13, RegistrationIDLabel: "PAN", CompanyTaxID: ""},
}

// GetTaxConfig loads tax configuration for a country from PlatformSettings, falling back to defaults
func GetTaxConfig(countryCode string) (*TaxConfig, error) {
	cc := strings.ToUpper(countryCode)

	// Start with defaults
	cfg := TaxConfig{
		CountryCode:         cc,
		TaxName:             "Tax",
		FoodPercent:         0,
		ServicePercent:      0,
		DeliveryPercent:     0,
		SubscriptionPercent: 0,
		RegistrationIDLabel: "Tax ID",
		CompanyTaxID:        "",
	}

	if defaults, ok := defaultTaxConfigs[cc]; ok {
		cfg = defaults
	}

	// Try to load overrides from PlatformSettings
	prefix := fmt.Sprintf("tax.%s.", cc)
	var settings []models.PlatformSettings
	database.DB.Where("key LIKE ?", prefix+"%").Find(&settings)

	for _, s := range settings {
		key := strings.TrimPrefix(s.Key, prefix)
		val := s.Value

		switch key {
		case "tax_name":
			cfg.TaxName = val
		case "food_percent":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.FoodPercent = v
			}
		case "service_percent":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.ServicePercent = v
			}
		case "delivery_percent":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.DeliveryPercent = v
			}
		case "subscription_percent":
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.SubscriptionPercent = v
			}
		case "registration_id_label":
			cfg.RegistrationIDLabel = val
		case "company_tax_id":
			cfg.CompanyTaxID = val
		}
	}

	return &cfg, nil
}

// GetCompanyInfo loads Fe3dr company details from PlatformSettings
func GetCompanyInfo() (*InvoiceCompanyInfo, error) {
	info := &InvoiceCompanyInfo{
		Name:    "Fe3dr Technologies Pvt. Ltd.",
		Address: "",
		Email:   "billing@fe3dr.com",
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

	// Determine country code from chef address or default to "IN"
	countryCode := "IN"
	if order.Chef.State != "" {
		// Try to look up country from address - for now use IN as default
		// In production, you'd resolve this from the chef's registered country
		var chefCountry models.PlatformSettings
		if err := database.DB.Where("key = ?", fmt.Sprintf("chef.%s.country_code", order.Chef.ID.String())).
			First(&chefCountry).Error; err == nil {
			countryCode = chefCountry.Value
		}
	}

	// Load tax config
	taxCfg, err := GetTaxConfig(countryCode)
	if err != nil {
		return nil, fmt.Errorf("failed to load tax config: %w", err)
	}

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
		lineItems[i] = models.InvoiceLineItem{
			Name:      item.Name,
			Quantity:  item.Quantity,
			UnitPrice: item.Price,
			Total:     itemTotal,
		}
	}
	subtotal = models.RoundAmount(subtotal)

	// Calculate taxes
	foodTax := models.RoundAmount(subtotal * taxCfg.FoodPercent / 100)
	deliveryFee := order.DeliveryFee
	deliveryTax := models.RoundAmount(deliveryFee * taxCfg.DeliveryPercent / 100)
	serviceFee := order.ServiceFee
	serviceTax := models.RoundAmount(serviceFee * taxCfg.ServicePercent / 100)
	tip := order.Tip
	discount := order.Discount

	totalAmount := models.RoundAmount(subtotal + foodTax + deliveryFee + deliveryTax + serviceFee + serviceTax + tip - discount)

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

	// Determine currency from plan settings or default
	currency := "INR"
	var currSetting models.PlatformSettings
	if err := database.DB.Where("key = ?", fmt.Sprintf("currency.%s.default", countryCode)).
		First(&currSetting).Error; err == nil {
		currency = currSetting.Value
	}

	// Determine company tax ID
	companyTaxID := companyInfo.TaxID
	if taxCfg.CompanyTaxID != "" {
		companyTaxID = taxCfg.CompanyTaxID
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
		ServiceFee:  serviceFee,
		ServiceTax:  serviceTax,
		Tip:         tip,
		Discount:    discount,
		TotalAmount: totalAmount,

		CountryCode:        countryCode,
		Currency:           currency,
		TaxName:            taxCfg.TaxName,
		FoodTaxPercent:     taxCfg.FoodPercent,
		ServiceTaxPercent:  taxCfg.ServicePercent,
		DeliveryTaxPercent: taxCfg.DeliveryPercent,

		CustomerName:  order.Customer.FirstName + " " + order.Customer.LastName,
		CustomerEmail: order.Customer.Email,
		CustomerPhone: order.Customer.Phone,
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

	// Load tax config
	taxCfg, err := GetTaxConfig(sub.CountryCode)
	if err != nil {
		return nil, fmt.Errorf("failed to load tax config: %w", err)
	}

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
			"name":       taxCfg.TaxName,
			"percent":    taxCfg.SubscriptionPercent,
			"amount":     invoice.TaxAmount,
			"idLabel":    taxCfg.RegistrationIDLabel,
			"companyId":  taxCfg.CompanyTaxID,
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
