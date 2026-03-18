package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// ProviderDeliveryRequest represents a request to create a delivery via a provider
type ProviderDeliveryRequest struct {
	OrderID         uuid.UUID
	PickupAddress   string
	PickupLat       float64
	PickupLng       float64
	DropoffAddress  string
	DropoffLat      float64
	DropoffLng      float64
	CustomerName    string
	CustomerPhone   string
	ItemDescription string
	Weight          float64 // estimated weight in kg
}

// ProviderDeliveryResponse represents the provider's response
type ProviderDeliveryResponse struct {
	ExternalDeliveryID string
	ExternalTrackingID string
	TrackingURL        string
	EstimatedPickup    time.Time
	EstimatedDelivery  time.Time
	Cost               float64
}

// ProviderService handles third-party delivery provider operations
type ProviderService struct{}

// NewProviderService creates a new ProviderService
func NewProviderService() *ProviderService {
	return &ProviderService{}
}

// FindAvailableProvider finds the best available provider for a delivery
func (s *ProviderService) FindAvailableProvider(city string, distance float64, countryCode string) (*models.DeliveryProvider, error) {
	var providers []models.DeliveryProvider
	db := database.DB.Where("is_enabled = ? AND is_active = ?", true, true)

	// Filter by supported country
	if countryCode != "" {
		db = db.Where("supported_countries @> ?", fmt.Sprintf(`["%s"]`, countryCode))
	}

	// Filter by supported city
	if city != "" {
		db = db.Where("supported_cities @> ?", fmt.Sprintf(`["%s"]`, city))
	}

	// Filter by max distance
	if distance > 0 {
		db = db.Where("max_distance >= ?", distance)
	}

	// Order by priority (lower number = higher priority)
	if err := db.Order("priority ASC").Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("failed to query providers: %w", err)
	}

	// Check each provider for capacity
	for _, provider := range providers {
		// Check concurrent delivery limit
		var activeCount int64
		database.DB.Model(&models.Delivery{}).
			Where("provider_id = ? AND status NOT IN ?", provider.ID, []string{
				string(models.DeliveryDelivered),
				string(models.DeliveryFailed),
				string(models.DeliveryCancelled),
				string(models.DeliveryReturned),
			}).Count(&activeCount)

		if int(activeCount) >= provider.MaxConcurrentDeliveries {
			continue
		}

		// Check daily limit
		if provider.DailyLimit > 0 {
			today := time.Now().Truncate(24 * time.Hour)
			var todayCount int64
			database.DB.Model(&models.Delivery{}).
				Where("provider_id = ? AND assigned_at >= ?", provider.ID, today).
				Count(&todayCount)

			if int(todayCount) >= provider.DailyLimit {
				continue
			}
		}

		return &provider, nil
	}

	return nil, nil // no provider available
}

// CreateProviderDelivery creates a delivery request with a third-party provider
// Phase 1: returns a mock response; actual API integration is Phase 2
func (s *ProviderService) CreateProviderDelivery(provider *models.DeliveryProvider, req ProviderDeliveryRequest) (*ProviderDeliveryResponse, error) {
	log.Printf("Creating provider delivery: provider=%s order=%s", provider.Code, req.OrderID)

	// Calculate estimated cost based on pricing model
	cost := s.calculateCost(provider, req)

	// Calculate distance for time estimates
	distance := haversineDistance(req.PickupLat, req.PickupLng, req.DropoffLat, req.DropoffLng)
	estimatedMinutes := provider.AvgPickupTime + int(distance*3) // rough estimate: 3 min/km

	now := time.Now()
	response := &ProviderDeliveryResponse{
		ExternalDeliveryID: fmt.Sprintf("%s_%s", provider.Code, uuid.New().String()[:8]),
		ExternalTrackingID: fmt.Sprintf("TRK_%s_%d", provider.Code, now.UnixMilli()),
		TrackingURL:        fmt.Sprintf("%s/track/%s_%s", provider.APIBaseURL, provider.Code, uuid.New().String()[:8]),
		EstimatedPickup:    now.Add(time.Duration(provider.AvgPickupTime) * time.Minute),
		EstimatedDelivery:  now.Add(time.Duration(estimatedMinutes) * time.Minute),
		Cost:               cost,
	}

	// Publish NATS event
	_ = PublishEvent(SubjectProviderDeliveryCreated, "provider.delivery.created", uuid.Nil, map[string]interface{}{
		"provider_id":          provider.ID.String(),
		"provider_code":        provider.Code,
		"order_id":             req.OrderID.String(),
		"external_delivery_id": response.ExternalDeliveryID,
		"cost":                 response.Cost,
	})

	return response, nil
}

// HandleProviderWebhook processes an inbound webhook from a provider
func (s *ProviderService) HandleProviderWebhook(providerCode string, payload []byte) error {
	log.Printf("Processing webhook from provider: %s", providerCode)

	var provider models.DeliveryProvider
	if err := database.DB.Where("code = ?", providerCode).First(&provider).Error; err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	// Parse the webhook payload — generic structure
	var webhookData map[string]interface{}
	if err := json.Unmarshal(payload, &webhookData); err != nil {
		return fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Extract external delivery ID from payload
	externalID, ok := webhookData["delivery_id"].(string)
	if !ok {
		// Try alternative field names
		if eid, ok := webhookData["order_id"].(string); ok {
			externalID = eid
		} else if eid, ok := webhookData["id"].(string); ok {
			externalID = eid
		}
	}

	if externalID == "" {
		return fmt.Errorf("no delivery ID found in webhook payload")
	}

	// Look up the delivery by external ID
	var delivery models.Delivery
	if err := database.DB.Where("external_delivery_id = ? AND provider_id = ?", externalID, provider.ID).
		First(&delivery).Error; err != nil {
		return fmt.Errorf("delivery not found for external ID %s: %w", externalID, err)
	}

	// Extract provider status
	providerStatus, _ := webhookData["status"].(string)
	if providerStatus == "" {
		return fmt.Errorf("no status found in webhook payload")
	}

	// Map provider status to Fe3dr status using StatusMapping
	var statusMap map[string]string
	if err := json.Unmarshal([]byte(provider.StatusMapping), &statusMap); err != nil {
		return fmt.Errorf("failed to parse provider status mapping: %w", err)
	}

	fe3drStatus, mapped := statusMap[providerStatus]
	if !mapped {
		log.Printf("Unmapped provider status: provider=%s status=%s", providerCode, providerStatus)
		return fmt.Errorf("unmapped provider status: %s", providerStatus)
	}

	// Update delivery status
	updates := map[string]interface{}{
		"status": fe3drStatus,
	}

	// Set timestamps based on status
	now := time.Now()
	switch models.DeliveryStatus(fe3drStatus) {
	case models.DeliveryPickedUp:
		updates["picked_up_at"] = now
	case models.DeliveryDelivered:
		updates["delivered_at"] = now
	case models.DeliveryCancelled:
		updates["cancelled_at"] = now
		if reason, ok := webhookData["reason"].(string); ok {
			updates["cancel_reason"] = reason
		}
	case models.DeliveryFailed:
		if reason, ok := webhookData["reason"].(string); ok {
			updates["failure_reason"] = reason
		}
	}

	// Update tracking URL if provided
	if trackingURL, ok := webhookData["tracking_url"].(string); ok && trackingURL != "" {
		updates["external_tracking_url"] = trackingURL
	}

	if err := database.DB.Model(&delivery).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update delivery: %w", err)
	}

	// Publish NATS event
	_ = PublishEvent(SubjectProviderDeliveryUpdated, "provider.delivery.updated", uuid.Nil, map[string]interface{}{
		"provider_id":          provider.ID.String(),
		"provider_code":        provider.Code,
		"delivery_id":          delivery.ID.String(),
		"external_delivery_id": externalID,
		"old_status":           string(delivery.Status),
		"new_status":           fe3drStatus,
		"provider_status":      providerStatus,
	})

	return nil
}

// calculateCost calculates the estimated cost based on the provider's pricing model
func (s *ProviderService) calculateCost(provider *models.DeliveryProvider, req ProviderDeliveryRequest) float64 {
	distance := haversineDistance(req.PickupLat, req.PickupLng, req.DropoffLat, req.DropoffLng)

	switch provider.PricingModel {
	case "per_km":
		cost := provider.BaseCost + (provider.PerKmCost * distance)
		return math.Round(cost*100) / 100
	case "flat_rate":
		return provider.BaseCost
	case "per_delivery":
		fallthrough
	default:
		return provider.BaseCost
	}
}

// haversineDistance calculates the distance between two lat/lng points in km
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}
