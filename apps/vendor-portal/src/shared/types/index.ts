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
export type OnboardingStatus = 'not_started' | 'in_progress' | 'submitted' | 'approved' | 'rejected';

export interface Chef {
  id: string;
  userId: string;
  businessName: string;
  description: string;
  cuisines: string[];
  specialties: string[];
  profileImage?: string;
  bannerImage?: string;
  kitchenPhotos?: string[];
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
  onboardingStatus?: OnboardingStatus;
  onboardingStep?: number;
  createdAt: string;
}

// Kitchen & Document types for onboarding
export type KitchenType = 'home_kitchen' | 'cloud_kitchen' | 'shared_kitchen';

export interface KitchenAddress {
  line1: string;
  line2?: string;
  country: string;
  city: string;
  state: string;
  postalCode: string;
  landmark?: string;
}

export interface DocumentUpload {
  id?: string;
  type: DocumentType;
  fileName: string;
  fileUrl?: string;
  status: 'pending' | 'verified' | 'rejected';
  rejectionReason?: string;
  uploadedAt?: string;
}

export type DocumentType =
  | 'fssai_license'
  | 'pan_card'
  | 'aadhaar_card'
  | 'kitchen_photo_1'
  | 'kitchen_photo_2'
  | 'kitchen_photo_3'
  | 'food_safety_cert'
  | 'cancelled_cheque';

export interface OnboardingData {
  // Step 1: Personal Info
  fullName: string;
  phone: string;
  email: string;
  kitchenAddress: KitchenAddress;

  // Step 2: Kitchen Details
  businessName: string;
  description: string;
  kitchenType: KitchenType;
  cuisines: string[];
  specialties: string[];
  yearsOfExperience: string;
  mealsPerDay: string;

  // Step 3: Operations
  prepTime: string;
  serviceRadius: number;
  minimumOrder: number;
  deliveryFee: number;
  operatingHours: OperatingHours;

  // Profile image
  profileImage?: string;

  // Step 4: Documents
  fssaiLicenseNumber?: string;
  panNumber?: string;
  documents: DocumentUpload[];

  // Step 5: Policies
  acceptedTerms: boolean;
  acceptedHygienePolicy: boolean;
  acceptedCancellationPolicy: boolean;
  bankAccountName?: string;
  bankAccountNumber?: string;
  bankIfscCode?: string;
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

export interface MenuItemImage {
  id: string;
  menuItemId: string;
  url: string;
  isPrimary: boolean;
  sortOrder: number;
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
  images?: MenuItemImage[];
  dietaryTags: string[];
  allergens: string[];
  prepTime: number;
  isAvailable: boolean;
  isApproved: boolean;
  isFeatured: boolean;
  portionSize?: string;
  serves: number;
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
