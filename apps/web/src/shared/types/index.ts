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

export interface Order {
  id: string;
  orderNumber: string;
  customerId: string;
  chefId: string;
  deliveryPartnerId?: string;
  status: OrderStatus;
  items: OrderItem[];
  deliveryAddress: Address;
  subtotal: number;
  deliveryFee: number;
  serviceFee: number;
  tax: number;
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
  createdAt: string;
}

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

// Social feed types
export interface SocialPost {
  id: string;
  chefId: string;
  chef?: Pick<Chef, 'id' | 'businessName' | 'profileImage'>;
  content: string;
  images: string[];
  taggedMenuItems?: string[];
  hashtags: string[];
  likesCount: number;
  commentsCount: number;
  savesCount: number;
  isLiked?: boolean;
  isSaved?: boolean;
  createdAt: string;
}

export interface SocialComment {
  id: string;
  postId: string;
  userId: string;
  userName: string;
  userAvatar?: string;
  content: string;
  createdAt: string;
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
