package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

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
	// Real-3PL fields (Shadowfax needs pincodes/city/state + a pickup contact +
	// the order value). PickupName/Phone identify the chef to the rider.
	ClientOrderID  string // our order id as the provider's client_order_id
	PickupName     string
	PickupPhone    string
	PickupCity     string
	PickupState    string
	PickupPincode  string
	DropoffCity    string
	DropoffState   string
	DropoffPincode string
	OrderValue     float64 // order subtotal — sent as product_value
	// ScheduledPickupAt is the order's scheduled delivery-slot pickup time (#51),
	// nil for ASAP orders. Adapters that support scheduled bookings should pass
	// it to the 3PL so the rider is timed to the slot; adapters that don't may
	// ignore it. NOTE: deferring the dispatch *trigger* itself until close to the
	// slot (vs booking immediately at chef-ready) is tracked separately under #7.
	ScheduledPickupAt *time.Time
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

	// Filter by supported country. json.Marshal escapes embedded quotes so
	// untrusted values can't break out of the JSON literal passed to @>.
	if countryCode != "" {
		b, _ := json.Marshal([]string{countryCode})
		db = db.Where("supported_countries @> ?", string(b))
	}

	// Filter by supported city
	if city != "" {
		b, _ := json.Marshal([]string{city})
		db = db.Where("supported_cities @> ?", string(b))
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

// AnyProviderEnabled reports whether at least one 3PL delivery provider is
// enabled AND active. This is the single source of truth for whether the 3PL /
// rider surface is live at all: when it's false everything third-party is dark
// — no rider dispatch, no "hand to a rider" UI, and "delivery" is only offered
// for chefs who self-deliver. Flipping a provider's is_enabled=true (e.g. once
// Borzo is validated) lights the whole path back up with no code change.
func (s *ProviderService) AnyProviderEnabled() bool {
	var count int64
	database.DB.Model(&models.DeliveryProvider{}).
		Where("is_enabled = ? AND is_active = ?", true, true).
		Count(&count)
	return count > 0
}

// ThirdPartyDeliveryEnabled is a package-level convenience over
// ProviderService.AnyProviderEnabled for handlers that don't hold a service.
func ThirdPartyDeliveryEnabled() bool {
	return NewProviderService().AnyProviderEnabled()
}

// CreateProviderDelivery books a delivery with a third-party provider. When an
// outbound adapter exists for the provider's code (ClientFor) the real API is
// called; providers without an adapter yet get a deterministic mock response so
// onboarding isn't blocked. Either way a single NATS event is published.
func (s *ProviderService) CreateProviderDelivery(provider *models.DeliveryProvider, req ProviderDeliveryRequest) (*ProviderDeliveryResponse, error) {
	log.Printf("Creating provider delivery: provider=%s order=%s", provider.Code, req.OrderID)

	var response *ProviderDeliveryResponse
	if client, ok := ClientFor(provider); ok {
		resp, err := client.CreateTask(context.Background(), req)
		if err != nil {
			return nil, fmt.Errorf("provider %s create task: %w", provider.Code, err)
		}
		response = resp
	} else {
		response = s.mockProviderDelivery(provider, req)
	}

	// Durable event publication via the transactional outbox.
	if err := EnqueueEvent(database.DB, SubjectProviderDeliveryCreated, "provider.delivery.created", uuid.Nil, map[string]interface{}{
		"provider_id":          provider.ID.String(),
		"provider_code":        provider.Code,
		"order_id":             req.OrderID.String(),
		"external_delivery_id": response.ExternalDeliveryID,
		"cost":                 response.Cost,
	}); err != nil {
		log.Printf("failed to enqueue provider.delivery.created event: %v", err)
	}

	return response, nil
}

// mockProviderDelivery returns a deterministic fake booking for providers that
// don't have a real outbound adapter yet. Cost + ETA are derived from the
// provider's configured pricing/timing so admin stats stay sane in testing.
func (s *ProviderService) mockProviderDelivery(provider *models.DeliveryProvider, req ProviderDeliveryRequest) *ProviderDeliveryResponse {
	cost := s.calculateCost(provider, req)
	distance := haversineDistance(req.PickupLat, req.PickupLng, req.DropoffLat, req.DropoffLng)
	estimatedMinutes := provider.AvgPickupTime + int(distance*3) // rough estimate: 3 min/km

	now := time.Now()
	return &ProviderDeliveryResponse{
		ExternalDeliveryID: fmt.Sprintf("%s_%s", provider.Code, uuid.New().String()[:8]),
		ExternalTrackingID: fmt.Sprintf("TRK_%s_%d", provider.Code, now.UnixMilli()),
		TrackingURL:        fmt.Sprintf("%s/track/%s_%s", provider.APIBaseURL, provider.Code, uuid.New().String()[:8]),
		EstimatedPickup:    now.Add(time.Duration(provider.AvgPickupTime) * time.Minute),
		EstimatedDelivery:  now.Add(time.Duration(estimatedMinutes) * time.Minute),
		Cost:               cost,
	}
}

// GetProviderQuote returns a price + ETA for a leg. Uses the provider's outbound
// adapter when available; otherwise computes a quote from the provider's
// configured pricing model (always serviceable). Returns Serviceable=false only
// when a real adapter says the leg can't be served.
func (s *ProviderService) GetProviderQuote(provider *models.DeliveryProvider, req QuoteRequest) (*QuoteResponse, error) {
	if client, ok := ClientFor(provider); ok {
		return client.GetQuote(context.Background(), req)
	}

	distance := haversineDistance(req.PickupLat, req.PickupLng, req.DropoffLat, req.DropoffLng)
	var fee float64
	switch provider.PricingModel {
	case "per_km":
		fee = math.Round((provider.BaseCost+provider.PerKmCost*distance)*100) / 100
	default: // flat_rate, per_delivery
		fee = provider.BaseCost
	}
	currency := provider.Currency
	if currency == "" {
		currency = "INR"
	}
	return &QuoteResponse{
		Fee:         fee,
		Currency:    currency,
		ETAMinutes:  provider.AvgPickupTime + int(distance*3),
		Serviceable: true,
	}, nil
}

// CancelProviderDelivery cancels a booked task. No-op (logged) for providers
// without a real adapter.
func (s *ProviderService) CancelProviderDelivery(provider *models.DeliveryProvider, externalID, reason string) error {
	if client, ok := ClientFor(provider); ok {
		return client.CancelTask(context.Background(), externalID, reason)
	}
	log.Printf("mock cancel provider delivery: provider=%s external=%s reason=%s", provider.Code, externalID, reason)
	return nil
}

// HandleProviderWebhook processes an inbound webhook from a provider
// terminalize3PLDeliveryFailure freezes a 3PL delivery's money state for admin fault
// resolution when the provider reports a terminal non-delivery (failed / returned-RTO):
// it opens a pending delivery_failed OrderIssue and disputes the order's payout hold, so
// the chef is not paid until an admin confirms fault (#393). No money moves here. A raw
// provider status carries no reliable fault signal, so the freeze uses FailureOther →
// FaultAmbiguous: the admin picks the concrete outcome (refund vs release). Idempotent —
// TerminalizeDeliveryFailure no-ops on a re-fired failure. Returns a (transient) error on
// a genuine failure so the webhook layer retries and the money is eventually frozen.
func terminalize3PLDeliveryFailure(orderID uuid.UUID, providerCode, deliveryID string) error {
	var order models.Order
	if err := database.DB.First(&order, "id = ?", orderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// A genuinely-missing order won't appear on retry — PERMANENT, so the webhook
			// layer acks and stops (no infinite retry-storm). The delivery still references
			// this order id, so this is effectively unreachable, but fail closed to permanent.
			return fmt.Errorf("delivery-failure: 3PL order %s not found: %w", orderID, ErrWebhookPermanent)
		}
		return fmt.Errorf("delivery-failure: load 3PL order %s: %w", orderID, err)
	}
	froze, err := TerminalizeDeliveryFailure(database.DB, &order, models.FailureOther, "3pl:"+providerCode,
		map[string]any{"delivery_id": deliveryID})
	if err != nil {
		return fmt.Errorf("delivery-failure: terminalize 3PL order %s: %w", orderID, err)
	}
	if !froze {
		// froze=false with no error means the order shape has no freeze handler yet —
		// today that's a consolidated GROUP order (no razorpay_order_id, not a meal-plan
		// day): RecordDeliveryFailure skips it and no group-failure mirror exists. Surface
		// it loudly rather than silently 200-ack so the order isn't invisibly stranded
		// (group-order failure freeze is a tracked follow-up).
		log.Printf("delivery-failure: 3PL order %s failed/returned but was NOT frozen (unsupported order shape, e.g. group order) — needs manual admin resolution", orderID)
	}
	return nil
}

func (s *ProviderService) HandleProviderWebhook(providerCode string, payload []byte) error {
	log.Printf("Processing webhook from provider: %s", providerCode)

	var provider models.DeliveryProvider
	if err := database.DB.Where("code = ?", providerCode).First(&provider).Error; err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	// Shadowfax's Push Callback keys on our order number (not the AWB) and uses a
	// code-based status map, so it takes a dedicated path rather than the generic
	// external_delivery_id + StatusMapping flow below.
	if provider.Code == "shadowfax" {
		return s.handleShadowfaxWebhook(&provider, payload)
	}

	// Parse the webhook payload — generic structure
	var webhookData map[string]interface{}
	if err := json.Unmarshal(payload, &webhookData); err != nil {
		// A malformed body will never parse on retry — permanent.
		return fmt.Errorf("failed to parse webhook payload (%v): %w", err, ErrWebhookPermanent)
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
		return fmt.Errorf("no delivery ID found in webhook payload: %w", ErrWebhookPermanent)
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
		return fmt.Errorf("no status found in webhook payload: %w", ErrWebhookPermanent)
	}

	// Map provider status to Fe3dr status using StatusMapping
	var statusMap map[string]string
	if err := json.Unmarshal([]byte(provider.StatusMapping), &statusMap); err != nil {
		return fmt.Errorf("failed to parse provider status mapping (%v): %w", err, ErrWebhookPermanent)
	}

	fe3drStatus, mapped := statusMap[providerStatus]
	if !mapped {
		log.Printf("Unmapped provider status: provider=%s status=%s", providerCode, providerStatus)
		// An unmapped status won't map on retry — permanent, ACK and stop retries.
		return fmt.Errorf("unmapped provider status %q: %w", providerStatus, ErrWebhookPermanent)
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

	// Persist the raw provider status + live rider details so the customer map
	// shows the real 3PL rider (closes the live-map gap). Field names vary by
	// provider; accept the common aliases.
	updates["provider_status"] = providerStatus
	if name := firstWebhookString(webhookData, "rider_name", "agent_name", "driver_name"); name != "" {
		updates["rider_name"] = name
	}
	if phone := firstWebhookString(webhookData, "rider_phone", "agent_phone", "driver_phone", "rider_contact"); phone != "" {
		updates["rider_phone"] = phone
	}
	if lat, ok := firstWebhookFloat(webhookData, "rider_latitude", "latitude", "lat", "current_latitude"); ok {
		updates["rider_latitude"] = lat
	}
	if lng, ok := firstWebhookFloat(webhookData, "rider_longitude", "longitude", "lng", "lon", "current_longitude"); ok {
		updates["rider_longitude"] = lng
	}

	if err := database.DB.Model(&delivery).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update delivery: %w", err)
	}

	// On 3PL delivery, mark the meal-plan/group order delivered and park the
	// regular order's payout in a customer-confirmation hold (#387) — delivery no
	// longer releases funds; the customer confirming (or #388) drives release.
	// All three are status-guarded no-ops for orders they don't apply to.
	if models.DeliveryStatus(fe3drStatus) == models.DeliveryDelivered && delivery.OrderID != uuid.Nil {
		MarkMealPlanDayDelivered(delivery.OrderID)
		MarkGroupOrderDelivered(delivery.OrderID)
		if err := SetOrderHoldAwaitingConfirmation(database.DB, delivery.OrderID); err != nil {
			log.Printf("payout-hold: park order %s on 3PL delivery failed: %v", delivery.OrderID, err)
		}
	}

	// A 3PL delivery that terminally failed or was returned (RTO) freezes the order's
	// money for admin fault resolution — never leave it stranded unfrozen (#393). Placed
	// before the event enqueue so a transient freeze error returns for a webhook retry
	// (idempotent) without double-emitting the informational update event.
	if st := models.DeliveryStatus(fe3drStatus); (st == models.DeliveryFailed || st == models.DeliveryReturned) && delivery.OrderID != uuid.Nil {
		if err := terminalize3PLDeliveryFailure(delivery.OrderID, provider.Code, delivery.ID.String()); err != nil {
			return err
		}
	}

	// A 3PL `cancelled` AFTER pickup (food already collected by the rider) but BEFORE
	// delivery is effectively a failed delivery — the customer won't get the food, so freeze
	// the money for admin fault resolution exactly like failed/returned (#594). Two guards on
	// the loaded (pre-update) row: PickedUpAt != nil (the rider has the food; a PRE-pickup
	// cancel is a normal cancellation, no freeze) AND DeliveredAt == nil (a late/replayed
	// `cancelled` after a `delivered` webhook must NOT dispute a correctly-delivered order —
	// delivered parks the hold at awaiting_confirmation, which SetOrderHoldDisputed would
	// otherwise flip). Idempotent via terminalize's existing-issue check.
	if models.DeliveryStatus(fe3drStatus) == models.DeliveryCancelled &&
		delivery.PickedUpAt != nil && delivery.DeliveredAt == nil && delivery.OrderID != uuid.Nil {
		if err := terminalize3PLDeliveryFailure(delivery.OrderID, provider.Code, delivery.ID.String()); err != nil {
			return err
		}
	}

	// Durable event publication via the transactional outbox.
	if err := EnqueueEvent(database.DB, SubjectProviderDeliveryUpdated, "provider.delivery.updated", uuid.Nil, map[string]interface{}{
		"provider_id":          provider.ID.String(),
		"provider_code":        provider.Code,
		"delivery_id":          delivery.ID.String(),
		"external_delivery_id": externalID,
		"old_status":           string(delivery.Status),
		"new_status":           fe3drStatus,
		"provider_status":      providerStatus,
	}); err != nil {
		log.Printf("failed to enqueue provider.delivery.updated event: %v", err)
	}

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

// firstWebhookString returns the first non-empty string value among the given
// keys in a decoded webhook payload. Lets one extraction handle field-name
// variation across providers.
func firstWebhookString(data map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := data[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// firstWebhookFloat returns the first numeric value among the given keys.
// JSON numbers decode as float64; a stringified number is also accepted.
func firstWebhookFloat(data map[string]interface{}, keys ...string) (float64, bool) {
	for _, k := range keys {
		switch v := data[k].(type) {
		case float64:
			return v, true
		case json.Number:
			if f, err := v.Float64(); err == nil {
				return f, true
			}
		}
	}
	return 0, false
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
