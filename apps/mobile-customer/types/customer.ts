// Customer app domain types matching the Go API response shapes.
// Reference: apps/api/handlers/chefs.go, orders.go, delivery.go

import type { PayoutHoldStatus } from '../lib/payout-hold';

export interface Chef {
  id: string;
  name: string;
  /** Official receipt fields (#receipt): the kitchen's registered business name,
   *  the proprietor (home chef), and regulatory IDs printed for the customer. */
  businessName?: string;
  ownerName?: string;
  fssaiLicenseNumber?: string;
  gstin?: string;
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
  // offersPickup: chef allows customers to pick up their order at the kitchen
  // address instead of requesting delivery (backend Task 1 field).
  offersPickup?: boolean;
  // offersSelfDelivery: chef delivers the order themselves (Phase 2) — surfaces
  // the "Chef delivery" mode in checkout.
  offersSelfDelivery?: boolean;
  // offersDelivery: server-computed — whether "Delivery" is fulfillable at all
  // (chef self-delivers OR a 3PL provider is live). Gates the delivery option in
  // checkout so no unfulfillable delivery order is placed. Defaults true when the
  // API omits it (older builds).
  offersDelivery?: boolean;
  // deliverableToYou: per-customer, server-computed from the coordinates the app
  // sent — whether THIS chef can deliver to the customer's location (own-fleet
  // within radius, or any location when a 3PL is live). Undefined when the app
  // sent no coordinates (fall back to offersDelivery). `false` means the chef is
  // shown for pickup only because the customer is outside the chef's delivery
  // range — the app must not offer delivery in that case.
  deliverableToYou?: boolean;
  // Full street address — only surfaced by TrackOrder for a PICKUP order (the
  // customer needs it to collect). Absent/fuzzed for delivery (privacy).
  address?: string;
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
  // Add-ons / combos (#52). modifierGroups drives the add-on picker; comboItems
  // are the included dishes shown on a combo.
  isCombo?: boolean;
  modifierGroups?: ModifierGroup[];
  comboItems?: ComboItemRef[];
}

/** A per-item modifier group with options (#232). */
export interface ModifierOption {
  id: string;
  name: string;
  priceDelta: number;
  isAvailable: boolean;
}
export interface ModifierGroup {
  id: string;
  name: string;
  required: boolean;
  minSelect: number;
  maxSelect: number; // 1 = single choice; >1 or 0 = multi
  options: ModifierOption[];
}
/** One included dish in a combo (#233). */
export interface ComboItemRef {
  menuItemId: string;
  name: string;
  quantity: number;
}

/** A selected add-on modifier on a cart line (#232). */
export interface SelectedModifier {
  groupId: string;
  groupName: string;
  optionId: string;
  optionName: string;
  priceDelta: number;
}

export interface CartItem {
  /** Unique per (menuItemId + modifier selection). Equals menuItemId when no
   *  modifiers are chosen, so the same dish with different add-ons is a separate
   *  line. Cart operations key by this (#232). */
  lineId: string;
  menuItemId: string;
  name: string;
  /** UNIT price including any modifier deltas. */
  price: number;
  quantity: number;
  imageUrl?: string;
  /** Per-item special instructions, e.g. "no onions". Sent as `notes` per item on CreateOrder. */
  instructions?: string;
  /** Selected add-on modifiers for this line (#232). */
  modifiers?: SelectedModifier[];
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
  id?: string;
  menuItemId: string;
  name: string;
  price: number;
  quantity: number;
}

export interface Order {
  id: string;
  orderNumber: string;
  // Matches the backend OrderStatus enum exactly (models/order.go). The chef
  // "accept" sets status='accepted' (the customer-facing label is "Confirmed").
  status:
    | 'pending'
    | 'accepted'
    | 'preparing'
    | 'ready'
    | 'picked_up'
    | 'delivering'
    | 'delivered'
    | 'cancelled'
    | 'refunded';
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
  // How the order reaches the customer (backend OrderResponse.fulfillmentType).
  // 'pickup' → no delivery address/fee; collect from the chef.
  fulfillmentType?: 'delivery' | 'chef_delivery' | 'pickup';
  // Home-tiffin scheduling handshake (#709): the customer's requested time, the
  // chef's confirmed/proposed time (ISO), and the status the detail screen shows.
  requestedFulfillmentAt?: string;
  confirmedFulfillmentAt?: string;
  fulfillmentTimeStatus?: 'requested' | 'confirmed' | 'proposed' | 'declined';
  // Real fee breakdown from OrderResponse — render these instead of deriving
  // "delivery fee" as (total − subtotal), which mislabels service fee + tax.
  deliveryFee?: number;
  serviceFee?: number;
  tax?: number;
  discount?: number;
  // Public URL of the chef's food-ready photo (the prepared dish), shown on the
  // order detail once the chef marks the order ready.
  readyPhotoUrl?: string;
  // Escrow payout-hold state (#617). Present only when a hold exists — i.e. with
  // the escrow flags on. Drives the "Confirm received" CTA + confirmed/disputed
  // states on delivered orders; absent (undefined) when the flags are off.
  payoutHoldStatus?: PayoutHoldStatus;
  customerConfirmedAt?: string;
  // Why a cancelled order was cancelled + the refund (#694). For a platform
  // auto-void these carry the apology ("the kitchen closed before this order was
  // accepted") and the amount refunded, so a cancelled order can explain itself
  // instead of showing a bare grey chip.
  cancelReason?: string;
  refundAmount?: number;
  refundedAt?: string;
}

export interface TrackingResponse {
  orderId: string;
  orderNumber: string;
  status: Order['status'];
  // How the order reaches the customer (TrackOrder response). Drives the
  // pickup-vs-delivery wording on the tracking screen.
  fulfillmentType?: Order['fulfillmentType'];
  chef: Chef;
  estimatedDeliveryTime?: string;
  delivery?: {
    id: string;
    status: string;
    currentLatitude?: number;    // driver live location (from DeliveryPartner.CurrentLatitude)
    currentLongitude?: number;   // driver live location (from DeliveryPartner.CurrentLongitude)
    dropoffLatitude?: number;    // customer destination (from Delivery.DropoffLatitude)
    dropoffLongitude?: number;   // customer destination (from Delivery.DropoffLongitude)
    // 3PL hosted live-tracking page (e.g. Shadowfax customer_track_url). The
    // Unified API exposes no raw rider GPS, so the customer opens this for a live map.
    externalTrackingUrl?: string;
    // NOTE: Backend gap — DeliveryResponse DTO (ToResponse()) omits dropoff and driver coords.
    // These fields will be undefined until backend adds them to the TrackOrder handler response.
    // Fallback: use chef.latitude / chef.longitude as destination marker when dropoff is missing.
  };
}
