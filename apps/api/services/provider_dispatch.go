package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// DispatchOrderDelivery books a 3PL delivery for an order and links it back.
//
// Idempotent: a no-op if the order already has a delivery (safe to call on every
// "ready" transition / retry). Provider is chosen by serviceable city + distance
// + cost via FindAvailableProvider, then CreateProviderDelivery books the task.
//
// Own fleet is retired — there is NO fallback to an internal driver. If no 3PL
// provider is serviceable the order is left for manual handling and logged; this
// is not treated as an error so the chef's status update still succeeds.
func DispatchOrderDelivery(orderID uuid.UUID) error {
	var order models.Order
	if err := database.DB.Preload("Chef").Preload("Customer").
		First(&order, "id = ?", orderID).Error; err != nil {
		return fmt.Errorf("dispatch: load order: %w", err)
	}

	// Idempotency: skip if a delivery already exists for this order.
	if order.DeliveryID != nil {
		return nil
	}
	var existing models.Delivery
	if err := database.DB.Where("order_id = ?", orderID).First(&existing).Error; err == nil {
		return nil
	}

	distance := haversineDistance(
		order.Chef.Latitude, order.Chef.Longitude,
		order.DeliveryLatitude, order.DeliveryLongitude,
	)
	city := order.DeliveryAddressCity
	country := order.DeliveryAddressCountry
	if country == "" {
		country = "IN"
	}

	svc := NewProviderService()
	provider, err := svc.FindAvailableProvider(city, distance, country)
	if err != nil {
		return fmt.Errorf("dispatch: find provider: %w", err)
	}
	if provider == nil {
		log.Printf("dispatch: no 3PL provider available for order=%s city=%q — left for manual handling", orderID, city)
		return nil
	}

	req := ProviderDeliveryRequest{
		OrderID:         order.ID,
		PickupAddress:   joinAddress(order.Chef.AddressLine1, order.Chef.City),
		PickupLat:       order.Chef.Latitude,
		PickupLng:       order.Chef.Longitude,
		DropoffAddress:  joinAddress(order.DeliveryAddressLine1, order.DeliveryAddressCity),
		DropoffLat:      order.DeliveryLatitude,
		DropoffLng:      order.DeliveryLongitude,
		CustomerName:    strings.TrimSpace(order.Customer.FirstName + " " + order.Customer.LastName),
		CustomerPhone:   order.Customer.Phone,
		ItemDescription: fmt.Sprintf("Order %s", order.OrderNumber),
		Weight:          1.0, // TODO: derive from item count/weights when available
	}

	resp, err := svc.CreateProviderDelivery(provider, req)
	if err != nil {
		return fmt.Errorf("dispatch: create provider delivery (provider=%s): %w", provider.Code, err)
	}

	delivery := models.Delivery{
		OrderID:             order.ID,
		Status:              models.DeliveryAssigned,
		AssignmentType:      models.AssignmentThirdParty,
		ProviderID:          &provider.ID,
		ExternalDeliveryID:  resp.ExternalDeliveryID,
		ExternalTrackingID:  resp.ExternalTrackingID,
		ExternalTrackingURL: resp.TrackingURL,
		ProviderCost:        resp.Cost,
		PickupAddressLine1:  order.Chef.AddressLine1,
		PickupAddressCity:   order.Chef.City,
		PickupLatitude:      order.Chef.Latitude,
		PickupLongitude:     order.Chef.Longitude,
		DropoffAddressLine1: order.DeliveryAddressLine1,
		DropoffAddressCity:  order.DeliveryAddressCity,
		DropoffLatitude:     order.DeliveryLatitude,
		DropoffLongitude:    order.DeliveryLongitude,
		Distance:            distance,
		DeliveryFee:         order.DeliveryFee,
	}
	if !resp.EstimatedPickup.IsZero() && resp.EstimatedDelivery.After(resp.EstimatedPickup) {
		delivery.EstimatedDuration = int(resp.EstimatedDelivery.Sub(resp.EstimatedPickup).Minutes())
	}

	if err := database.DB.Create(&delivery).Error; err != nil {
		return fmt.Errorf("dispatch: persist delivery: %w", err)
	}

	// Link delivery to order + stamp provider usage.
	database.DB.Model(&order).Update("delivery_id", delivery.ID)
	now := time.Now()
	database.DB.Model(provider).Update("last_used_at", now)

	log.Printf("dispatch: order=%s booked provider=%s external=%s cost=%.2f", orderID, provider.Code, resp.ExternalDeliveryID, resp.Cost)
	return nil
}

// joinAddress joins non-empty address parts with ", ".
func joinAddress(parts ...string) string {
	var nonEmpty []string
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			nonEmpty = append(nonEmpty, s)
		}
	}
	return strings.Join(nonEmpty, ", ")
}
