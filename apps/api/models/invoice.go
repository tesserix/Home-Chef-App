package models

import (
	"gorm.io/gorm"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// InvoiceLineItem represents a single line item in an order invoice
type InvoiceLineItem struct {
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
	Total     float64 `json:"total"`
}

// OrderInvoice stores generated invoice data for customer food orders
type OrderInvoice struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"orderId"`
	InvoiceNumber string    `gorm:"uniqueIndex;not null" json:"invoiceNumber"`
	CustomerID    uuid.UUID `gorm:"type:uuid;not null;index" json:"customerId"`
	ChefID        uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`

	// Amounts
	Subtotal    float64 `gorm:"not null" json:"subtotal"`
	FoodTax     float64 `gorm:"default:0" json:"foodTax"`
	DeliveryFee float64 `gorm:"default:0" json:"deliveryFee"`
	DeliveryTax float64 `gorm:"default:0" json:"deliveryTax"`
	ServiceFee  float64 `gorm:"default:0" json:"serviceFee"`
	ServiceTax  float64 `gorm:"default:0" json:"serviceTax"`
	Tip         float64 `gorm:"default:0" json:"tip"`
	Discount    float64 `gorm:"default:0" json:"discount"`
	TotalAmount float64 `gorm:"not null" json:"totalAmount"`

	// Tax details
	CountryCode        string  `gorm:"type:varchar(2)" json:"countryCode"`
	Currency           string  `gorm:"type:varchar(3)" json:"currency"`
	TaxName            string  `gorm:"" json:"taxName"`
	FoodTaxPercent     float64 `gorm:"default:0" json:"foodTaxPercent"`
	ServiceTaxPercent  float64 `gorm:"default:0" json:"serviceTaxPercent"`
	DeliveryTaxPercent float64 `gorm:"default:0" json:"deliveryTaxPercent"`

	// Parties
	CustomerName    string `gorm:"" json:"customerName"`
	CustomerEmail   string `gorm:"" json:"customerEmail"`
	CustomerPhone   string `gorm:"" json:"customerPhone"`
	CustomerAddress string `gorm:"type:text" json:"customerAddress"`
	ChefName        string `gorm:"" json:"chefName"`
	ChefAddress     string `gorm:"type:text" json:"chefAddress"`
	// PII companions (#710 P1). This table is a denormalized copy of customer
	// and chef PII frozen at invoice time — it must be encrypted in lockstep
	// with users, or it stays a plaintext replica of the same data.
	CustomerNameEnc    EncryptedString `gorm:"column:customer_name_enc;type:text" json:"-"`
	CustomerEmailEnc   EncryptedString `gorm:"column:customer_email_enc;type:text" json:"-"`
	CustomerEmailBidx  string          `gorm:"column:customer_email_bidx;type:text;index" json:"-"`
	CustomerPhoneEnc   EncryptedString `gorm:"column:customer_phone_enc;type:text" json:"-"`
	CustomerPhoneBidx  string          `gorm:"column:customer_phone_bidx;type:text;index" json:"-"`
	CustomerAddressEnc EncryptedString `gorm:"column:customer_address_enc;type:text" json:"-"`
	ChefNameEnc        EncryptedString `gorm:"column:chef_name_enc;type:text" json:"-"`
	ChefAddressEnc     EncryptedString `gorm:"column:chef_address_enc;type:text" json:"-"`

	// Company details (Fe3dr)
	CompanyName    string `gorm:"" json:"companyName"`
	CompanyAddress string `gorm:"type:text" json:"companyAddress"`
	CompanyTaxID   string `gorm:"" json:"companyTaxId"`

	// Items (JSONB - serialized line items)
	LineItems string `gorm:"type:jsonb;default:'[]'" json:"-"`

	// Metadata
	IssuedAt  time.Time `gorm:"not null" json:"issuedAt"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`

	// Relationships
	Order    Order `gorm:"foreignKey:OrderID" json:"-"`
	Customer User  `gorm:"foreignKey:CustomerID" json:"-"`
}

// OrderInvoiceResponse is the API response DTO for an order invoice
type OrderInvoiceResponse struct {
	ID            uuid.UUID `json:"id"`
	OrderID       uuid.UUID `json:"orderId"`
	InvoiceNumber string    `json:"invoiceNumber"`
	CustomerID    uuid.UUID `json:"customerId"`
	ChefID        uuid.UUID `json:"chefId"`

	// Amounts
	Subtotal    float64 `json:"subtotal"`
	FoodTax     float64 `json:"foodTax"`
	DeliveryFee float64 `json:"deliveryFee"`
	DeliveryTax float64 `json:"deliveryTax"`
	ServiceFee  float64 `json:"serviceFee"`
	ServiceTax  float64 `json:"serviceTax"`
	Tip         float64 `json:"tip"`
	Discount    float64 `json:"discount"`
	TotalAmount float64 `json:"totalAmount"`

	// Tax details
	CountryCode        string  `json:"countryCode"`
	Currency           string  `json:"currency"`
	TaxName            string  `json:"taxName"`
	FoodTaxPercent     float64 `json:"foodTaxPercent"`
	ServiceTaxPercent  float64 `json:"serviceTaxPercent"`
	DeliveryTaxPercent float64 `json:"deliveryTaxPercent"`

	// Parties
	CustomerName    string `json:"customerName"`
	CustomerEmail   string `json:"customerEmail"`
	CustomerPhone   string `json:"customerPhone"`
	CustomerAddress string `json:"customerAddress"`
	ChefName        string `json:"chefName"`
	ChefAddress     string `json:"chefAddress"`

	// Company
	CompanyName    string `json:"companyName"`
	CompanyAddress string `json:"companyAddress"`
	CompanyTaxID   string `json:"companyTaxId"`

	// Items
	LineItems []InvoiceLineItem `json:"lineItems"`

	// Metadata
	IssuedAt  time.Time `json:"issuedAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// ToResponse converts OrderInvoice to OrderInvoiceResponse
func (inv *OrderInvoice) ToResponse() OrderInvoiceResponse {
	var items []InvoiceLineItem
	if inv.LineItems != "" {
		json.Unmarshal([]byte(inv.LineItems), &items)
	}
	if items == nil {
		items = []InvoiceLineItem{}
	}

	return OrderInvoiceResponse{
		ID:                 inv.ID,
		OrderID:            inv.OrderID,
		InvoiceNumber:      inv.InvoiceNumber,
		CustomerID:         inv.CustomerID,
		ChefID:             inv.ChefID,
		Subtotal:           inv.Subtotal,
		FoodTax:            inv.FoodTax,
		DeliveryFee:        inv.DeliveryFee,
		DeliveryTax:        inv.DeliveryTax,
		ServiceFee:         inv.ServiceFee,
		ServiceTax:         inv.ServiceTax,
		Tip:                inv.Tip,
		Discount:           inv.Discount,
		TotalAmount:        inv.TotalAmount,
		CountryCode:        inv.CountryCode,
		Currency:           inv.Currency,
		TaxName:            inv.TaxName,
		FoodTaxPercent:     inv.FoodTaxPercent,
		ServiceTaxPercent:  inv.ServiceTaxPercent,
		DeliveryTaxPercent: inv.DeliveryTaxPercent,
		CustomerName:       inv.CustomerName,
		CustomerEmail:      inv.CustomerEmail,
		CustomerPhone:      inv.CustomerPhone,
		CustomerAddress:    inv.CustomerAddress,
		ChefName:           inv.ChefName,
		ChefAddress:        inv.ChefAddress,
		CompanyName:        inv.CompanyName,
		CompanyAddress:     inv.CompanyAddress,
		CompanyTaxID:       inv.CompanyTaxID,
		LineItems:          items,
		IssuedAt:           inv.IssuedAt,
		CreatedAt:          inv.CreatedAt,
	}
}

// BeforeSave mirrors the invoice PII snapshot into its encrypted companions
// (#710 P1).
func (i *OrderInvoice) BeforeSave(*gorm.DB) error {
	i.CustomerNameEnc = encOf(i.CustomerName)
	i.CustomerEmailEnc = encOf(i.CustomerEmail)
	i.CustomerEmailBidx = bidxOf(i.CustomerEmail)
	i.CustomerPhoneEnc = encOf(i.CustomerPhone)
	i.CustomerPhoneBidx = bidxOf(i.CustomerPhone)
	i.CustomerAddressEnc = encOf(i.CustomerAddress)
	i.ChefNameEnc = encOf(i.ChefName)
	i.ChefAddressEnc = encOf(i.ChefAddress)
	return nil
}
