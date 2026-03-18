export * from './auth';

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

export type DeliveryStatus =
  | 'assigned'
  | 'picked_up'
  | 'in_transit'
  | 'delivered'
  | 'cancelled';

export interface DeliveryPartner {
  id: string;
  userId: string;
  vehicleType: string;
  vehicleNumber: string;
  licenseNumber: string;
  verified: boolean;
  verifiedAt?: string;
  isActive: boolean;
  isOnline: boolean;
  currentLatitude: number;
  currentLongitude: number;
  rating: number;
  totalDeliveries: number;
  totalReviews: number;
  createdAt: string;
  name?: string;
  email?: string;
  phone?: string;
  avatar?: string;
}

export interface Delivery {
  id: string;
  orderId: string;
  deliveryPartnerId: string;
  status: DeliveryStatus;
  pickup: {
    address: string;
    city: string;
    latitude: number;
    longitude: number;
  };
  dropoff: {
    address: string;
    city: string;
    latitude: number;
    longitude: number;
  };
  distance: number;
  estimatedDuration: number;
  actualDuration?: number;
  deliveryFee: number;
  tip: number;
  totalPayout: number;
  assignedAt: string;
  pickedUpAt?: string;
  deliveredAt?: string;
  order?: {
    id: string;
    orderNumber: string;
    status: string;
    total: number;
    itemCount: number;
    items: OrderItem[];
    specialInstructions?: string;
    deliveryInstructions?: string;
  };
  chef?: {
    name: string;
  };
}

export interface OrderItem {
  id: string;
  menuItemId: string;
  name: string;
  price: number;
  quantity: number;
  subtotal: number;
  notes?: string;
}

export interface AvailableDelivery {
  orderId: string;
  orderNumber: string;
  chefName: string;
  itemCount: number;
  pickupAddress: string;
  dropoffAddress: string;
  distance: number;
  estimatedPayout: number;
  createdAt: string;
}

export interface EarningsData {
  period: string;
  totalDeliveries: number;
  totalEarnings: number;
  totalTips: number;
  deliveryFees: number;
  avgPerDelivery: number;
  daily: DailyEarning[];
}

export interface DailyEarning {
  date: string;
  deliveries: number;
  earnings: number;
}

export interface DashboardStats {
  partner: {
    id: string;
    isOnline: boolean;
    rating: number;
  };
  today: { deliveries: number; earnings: number };
  week: { deliveries: number; earnings: number };
  month: { deliveries: number; earnings: number };
  active: number;
  availableOrders: number;
  totalDeliveries: number;
  totalReviews: number;
}
