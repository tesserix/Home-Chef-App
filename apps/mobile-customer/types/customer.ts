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
