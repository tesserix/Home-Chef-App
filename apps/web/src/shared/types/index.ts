export * from './auth';

// Common API response types
export interface ApiResponse<T> {
  success: boolean;
  data: T;
  meta?: {
    timestamp: string;
    requestId: string;
  };
}

export interface ApiError {
  success: false;
  error: {
    code: string;
    message: string;
    details?: Array<{
      field: string;
      message: string;
    }>;
  };
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
    hasNext: boolean;
    hasPrev: boolean;
  };
}

// Chef types
export interface Chef {
  id: string;
  userId: string;
  businessName: string;
  description: string;
  cuisines: string[];
  specialties: string[];
  profileImage?: string;
  bannerImage?: string;
  rating: number;
  totalReviews: number;
  totalOrders: number;
  isOnline: boolean;
  acceptingOrders: boolean;
  serviceRadius: number;
  latitude: number;
  longitude: number;
  prepTime: string;
  priceRange: string;
  deliveryFee: number;
  minimumOrder: number;
  operatingHours: OperatingHours;
  verified: boolean;
  verifiedAt?: string;
  // foodSafetyBadge: chef holds a verified, non-expired FSSAI licence (#35).
  // Returned by the API on the chef-list card response.
  foodSafetyBadge?: boolean;
  // TODO(CW-01c): Expose chef FSSAI licence number on the public /chefs/:id API
  // payload. Backend currently stores it on the chef profile (vendor-portal) but
  // does not surface it to the customer storefront. Until then this is undefined
  // and the badge falls back to a neutral state.
  fssaiLicenseNumber?: string;
  createdAt: string;
}

export interface OperatingHours {
  monday?: DayHours;
  tuesday?: DayHours;
  wednesday?: DayHours;
  thursday?: DayHours;
  friday?: DayHours;
  saturday?: DayHours;
  sunday?: DayHours;
}

export interface DayHours {
  open: string;
  close: string;
}

// Menu types
export interface MenuCategory {
  id: string;
  chefId: string;
  name: string;
  description?: string;
  sortOrder: number;
  isActive: boolean;
}

export interface MenuItem {
  id: string;
  chefId: string;
  categoryId?: string;
  name: string;
  description?: string;
  price: number;
  comparePrice?: number;
  imageUrl?: string;
  images?: string[];
  dietaryTags: string[];
  allergens: string[];
  prepTime: number;
  isAvailable: boolean;
  isFeatured: boolean;
  portionSize?: string;
  serves: number;
  // Per-dish rating rolled up from DishRating (#145).
  rating?: number;
  totalReviews?: number;
  // Capacity & cutoff controls (#48). dailyCapacity null/absent = unlimited;
  // remainingToday/soldOut are server-derived for capped dishes today (IST).
  dailyCapacity?: number | null;
  remainingToday?: number | null;
  soldOut?: boolean;
  // Add-ons / combos (#52).
  isCombo?: boolean;
  modifierGroups?: ModifierGroup[];
  comboItems?: ComboItemRef[];
}

/** Per-item add-on modifier group (#232). */
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
  maxSelect: number;
  options: ModifierOption[];
}
/** Included dish in a combo (#233). */
export interface ComboItemRef {
  menuItemId: string;
  name: string;
  quantity: number;
}
/** A selected modifier on a cart line (#232). */
export interface SelectedModifier {
  groupId: string;
  groupName: string;
  optionId: string;
  optionName: string;
  priceDelta: number;
}

// Weekly fixed menu (#1) — a chef's published Mon–Sun × lunch/dinner menu.
export type MealSlot = 'lunch' | 'dinner';
export type MealVariant = 'veg' | 'nonveg';
export interface WeeklyMenuItem {
  id?: string;
  dayOfWeek: number; // 0=Sun .. 6=Sat
  slot: MealSlot;
  variant: MealVariant;
  name: string;
  description?: string;
  price: number;
  imageUrl?: string;
}
export interface WeeklyMenu {
  isPublished: boolean;
  publishedAt?: string | null;
  items: WeeklyMenuItem[];
}

// Order types
export type OrderStatus =
  | 'pending'
  | 'accepted'
  | 'preparing'
  | 'ready'
  | 'picked_up'
  | 'delivering'
  | 'delivered'
  | 'cancelled'
  | 'refunded';

/**
 * How an order reaches the customer. Mirrors models.FulfillmentType.
 *
 * The customer only ever chooses `delivery` vs `pickup`. WHO carries a delivery
 * order — the chef themselves or a 3PL rider — is the chef's call at Mark Ready,
 * which is why `chef_delivery` appears on responses but is never sent by this app.
 */
export type FulfillmentType = 'delivery' | 'chef_delivery' | 'pickup';

/**
 * Where the customer's suggested fulfilment time got to (#709).
 *
 * `requested` — the customer proposed it and the chef hasn't answered yet.
 * `confirmed` — the chef agreed to the requested time.
 * `proposed`  — the chef countered with a different time.
 * `declined`  — the chef can't do a specific time.
 */
export type FulfillmentTimeStatus = 'requested' | 'confirmed' | 'proposed' | 'declined';

/**
 * Chef identity as it appears on an order.
 *
 * The API never serializes the raw ChefProfile onto an order (that leaked
 * profile fields to the customer); it projects this narrow shape instead —
 * see `OrderChefResponse` in `apps/api/models/order.go`. Only present when
 * the handler preloaded the relation, hence `chef?` below.
 */
export interface OrderChef {
  id: string;
  /** Mirrors businessName; kept for backward-compat with older responses. */
  name: string;
  businessName?: string;
  /** The proprietor behind the kitchen — printed on the official receipt. */
  ownerName?: string;
  imageUrl?: string;
  fssaiLicenseNumber?: string;
  gstin?: string;
  /** Supplier state — the receipt uses it to split GST into CGST+SGST vs IGST. */
  state?: string;
}

/**
 * One statutory tax row, split and labelled by the API (models/pricing.go) so the
 * web page, the mobile app and the invoice PDF print identical wording and
 * identical paise. `label` already carries the rate ("CGST (2.5%)").
 */
export interface TaxLine {
  code: "cgst" | "sgst" | "igst" | "tax";
  label: string;
  rate: number;
  amount: number;
}

export interface Order {
  id: string;
  orderNumber: string;
  customerId: string;
  chefId: string;
  chef?: OrderChef;
  deliveryPartnerId?: string;
  status: OrderStatus;
  items: OrderItem[];
  deliveryAddress: Address;
  subtotal: number;
  deliveryFee: number;
  platformFee: number;
  tax: number;
  /** Render these instead of deriving CGST/SGST here — see TaxLine. */
  taxLines?: TaxLine[];
  /** Paise needed to make the lines sum to total. 0 for orders priced after #977. */
  rounding?: number;
  discount: number;
  tip: number;
  total: number;
  promoCode?: string;
  specialInstructions?: string;
  scheduledFor?: string;
  estimatedReadyAt?: string;
  estimatedDeliveryAt?: string;
  acceptedAt?: string;
  preparedAt?: string;
  pickedUpAt?: string;
  deliveredAt?: string;
  cancelledAt?: string;
  cancelReason?: string;
  paymentId?: string;
  paymentStatus: PaymentStatus;
  paymentMethod?: string;
  // Escrow payout hold (#387/#617). A delivered, gateway-charged order parks at
  // `awaiting_customer_confirmation` until the customer confirms receipt — that
  // confirmation is what makes the chef's payout release-eligible.
  payoutHoldStatus?: PayoutHoldStatus;
  customerConfirmedAt?: string;
  // Fulfilment mode. Absent on orders placed before the field existed, which is
  // why every consumer must treat undefined as 'delivery' (the server default)
  // rather than assuming it is always present.
  fulfillmentType?: FulfillmentType;
  // The home-tiffin time handshake (#709): what the customer asked for, what the
  // chef settled on, and where that negotiation stands.
  requestedFulfillmentAt?: string;
  confirmedFulfillmentAt?: string;
  fulfillmentTimeStatus?: FulfillmentTimeStatus;
  createdAt: string;
}

/** Mirrors models/payout_hold.go PayoutHoldStatus. '' (or absent) = no hold. */
export type PayoutHoldStatus =
  | ''
  | 'awaiting_customer_confirmation'
  | 'release_eligible'
  | 'released'
  | 'disputed'
  | 'withheld'
  | 'reversed';

export interface OrderItem {
  id: string;
  menuItemId: string;
  name: string;
  description?: string;
  price: number;
  quantity: number;
  subtotal: number;
  notes?: string;
  imageUrl?: string;
}

export type PaymentStatus =
  | 'pending'
  | 'processing'
  | 'completed'
  | 'failed'
  | 'refunded';

// Address types
export interface Address {
  id: string;
  userId: string;
  label: string;
  line1: string;
  line2?: string;
  city: string;
  state: string;
  postalCode: string;
  country: string;
  latitude?: number;
  longitude?: number;
  isDefault: boolean;
  deliveryInstructions?: string;
}

// Review types
export interface Review {
  id: string;
  orderId: string;
  overallRating: number;
  foodRating: number;
  deliveryRating?: number;
  valueRating?: number;
  // Ratings 2.0 sub-scores (#35)
  packagingRating?: number;
  hygieneRating?: number;
  title?: string;
  comment?: string;
  images?: string[];
  chefResponse?: string;
  chefRespondedAt?: string;
  helpfulCount: number;
  customerName: string;
  customerAvatar?: string;
  createdAt: string;
}

// Catering types
export type CateringRequestStatus =
  | 'pending'
  | 'quotes_received'
  | 'booked'
  | 'in_progress'
  | 'completed'
  | 'cancelled';

export type CateringServiceType = 'delivery_only' | 'setup' | 'full_service';

export interface CateringRequest {
  id: string;
  customerId: string;
  targetChefId?: string;
  status: CateringRequestStatus;
  eventDate: string;
  eventTime: string;
  eventLocation: Address;
  guestCount: number;
  cuisinePreferences: string[];
  dietaryRequirements: string[];
  budgetMin?: number;
  budgetMax?: number;
  serviceType: CateringServiceType;
  description?: string;
  quotesCount: number;
  createdAt: string;
}

export interface CateringQuote {
  id: string;
  requestId: string;
  chefId: string;
  chef?: Pick<Chef, 'id' | 'businessName' | 'profileImage' | 'rating' | 'totalReviews'>;
  menuItems: CateringMenuItem[];
  pricePerPerson: number;
  totalPrice: number;
  serviceCharge?: number;
  notes?: string;
  validUntil: string;
  status: 'pending' | 'accepted' | 'declined' | 'expired';
  createdAt: string;
}

export interface CateringMenuItem {
  name: string;
  description?: string;
  quantity: number;
  pricePerUnit: number;
}

// Delivery types
export interface DeliveryPartner {
  id: string;
  userId: string;
  vehicleType: 'bicycle' | 'motorcycle' | 'scooter' | 'car';
  vehicleNumber?: string;
  licenseNumber?: string;
  isOnline: boolean;
  isAvailable: boolean;
  rating: number;
  totalDeliveries: number;
  currentLatitude?: number;
  currentLongitude?: number;
  status: 'pending' | 'approved' | 'suspended';
  verified: boolean;
}

export interface Delivery {
  id: string;
  orderId: string;
  partnerId: string;
  status: DeliveryStatus;
  pickupAddress: string;
  dropoffAddress: string;
  distanceKm?: number;
  estimatedDuration?: number;
  deliveryFee: number;
  tip: number;
  assignedAt: string;
  pickedUpAt?: string;
  deliveredAt?: string;
}

export type DeliveryStatus =
  | 'assigned'
  | 'accepted'
  | 'at_pickup'
  | 'picked_up'
  | 'at_dropoff'
  | 'delivered'
  | 'cancelled';

// Filter types
export interface ChefFilters {
  [key: string]: string | number | boolean | undefined;
  lat?: number;
  lng?: number;
  radius?: number;
  cuisine?: string;
  search?: string;
  rating?: number;
  priceRange?: string;
  minPrice?: number;
  maxPrice?: number;
  dietary?: string;
  isOpen?: boolean;
  sort?: 'rating' | 'distance' | 'orders' | 'price';
  order?: 'asc' | 'desc';
  page?: number;
  limit?: number;
}

export interface OrderFilters {
  status?: OrderStatus | OrderStatus[];
  from?: string;
  to?: string;
  page?: number;
  limit?: number;
}

// Customer profile types
export interface CustomerProfile {
  id: string;
  userId: string;
  firstName: string;
  lastName: string;
  email: string;
  phone?: string;
  avatar?: string;
  dateOfBirth?: string;
  dietaryPreferences: string[];
  foodAllergies: string[];
  cuisinePreferences: string[];
  spiceTolerance: string;
  householdSize: string;
  onboardingCompleted: boolean;
  onboardingStep: number;
  preferredCurrency: string;
  authProvider: 'email' | 'google' | 'facebook' | 'apple';
}

export interface OnboardingStatus {
  onboardingCompleted: boolean;
  onboardingStep: number;
}

export type SpiceTolerance = 'mild' | 'medium' | 'hot' | 'extra_hot';
export type HouseholdSize = '1' | '2' | '3-4' | '5-6' | '7+';

// TOTP 2FA types
export interface TotpStatusResponse {
  success: boolean;
  totp_enabled: boolean;
  backup_codes_remaining: number;
}

export interface TotpSetupResponse {
  success: boolean;
  setup_session: string;
  totp_uri: string;
  manual_entry_key: string;
  backup_codes: string[];
}

// Favorites
export interface FavoriteChef {
  id: string;
  chefId: string;
  chef: Chef;
  createdAt: string;
}

/** A saved/favorited menu item (#237). */
export interface FavoriteDish {
  id: string;
  menuItemId: string;
  menuItem: MenuItem;
  chef: { id: string; businessName: string; profileImage?: string };
  createdAt: string;
}
