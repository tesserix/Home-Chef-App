// Customer app domain types matching the Go API response shapes.
// Reference: apps/api/handlers/chefs.go, orders.go, delivery.go

export interface Chef {
  id: string;
  name: string;
  cuisine: string;
  rating: number;
  reviewCount: number;
  isOpen: boolean;
  imageUrl?: string;
  latitude?: number;
  longitude?: number;
  deliveryTime?: string;
  minimumOrder?: number;
  deliveryFee?: number;
  // foodSafetyBadge (#35): chef holds a verified, non-expired FSSAI licence;
  // set from the chef-list API response via mapChef.
  foodSafetyBadge?: boolean;
}

export interface MenuItem {
  id: string;
  chefId: string;
  name: string;
  description?: string;
  price: number;
  imageUrl?: string;
  category?: string;
  isAvailable: boolean;
  dietaryTags?: string[];
  // Dietary & allergen profile (#41): declared allergens + veg flag, used for
  // per-item badges and at-checkout conflict warnings.
  allergens?: string[];
  isVeg?: boolean | null;
  // Per-dish rating rolled up from DishRating (#145).
  rating?: number;
  reviewCount?: number;
  // Capacity & cutoff controls (#48). dailyCapacity absent/0 = unlimited;
  // remainingToday/soldOut are server-derived for capped dishes today (IST).
  dailyCapacity?: number;
  remainingToday?: number;
  soldOut?: boolean;
}

export interface CartItem {
  menuItemId: string;
  name: string;
  price: number;
  quantity: number;
  imageUrl?: string;
  /** Per-item special instructions, e.g. "no onions". Sent as `notes` per item on CreateOrder. */
  instructions?: string;
}

export interface Address {
  id?: string;
  /** Home / Work / Other. Backend requires it (defaults to "Home" server-side). */
  label?: string;
  addressLine1: string;
  addressLine2?: string;
  city: string;
  state: string;
  pincode: string;
  /** Geocoded coords (from address autocomplete). 0/absent → server uses a flat delivery fee + skips zone checks. */
  latitude?: number;
  longitude?: number;
  isDefault?: boolean;
}

export interface OrderItem {
  menuItemId: string;
  name: string;
  price: number;
  quantity: number;
}

export interface Order {
  id: string;
  orderNumber: string;
  status: 'pending' | 'confirmed' | 'preparing' | 'ready' | 'picked_up' | 'delivered' | 'cancelled';
  // Payment lifecycle — distinct from `status` (order lifecycle). A paid order
  // can still sit at status='pending' until the chef accepts, so this is the
  // only reliable signal of whether money has been captured.
  paymentStatus?: 'pending' | 'completed' | 'failed' | 'refunded';
  // Optional: the order API's OrderResponse (ToResponse) carries no chef object
  // or chefId, so this is undefined on list/detail. Render defensively until the
  // backend adds it. See hooks/useOrderHistory.ts mapper.
  chef?: Chef;
  items: OrderItem[];
  totalAmount: number;
  deliveryAddress: Address;
  createdAt: string;
  estimatedDeliveryTime?: string;
}

export interface TrackingResponse {
  orderId: string;
  orderNumber: string;
  status: Order['status'];
  chef: Chef;
  estimatedDeliveryTime?: string;
  delivery?: {
    id: string;
    status: string;
    currentLatitude?: number;    // driver live location (from DeliveryPartner.CurrentLatitude)
    currentLongitude?: number;   // driver live location (from DeliveryPartner.CurrentLongitude)
    dropoffLatitude?: number;    // customer destination (from Delivery.DropoffLatitude)
    dropoffLongitude?: number;   // customer destination (from Delivery.DropoffLongitude)
    // NOTE: Backend gap — DeliveryResponse DTO (ToResponse()) omits dropoff and driver coords.
    // These fields will be undefined until backend adds them to the TrackOrder handler response.
    // Fallback: use chef.latitude / chef.longitude as destination marker when dropoff is missing.
  };
}
