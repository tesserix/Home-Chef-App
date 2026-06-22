// API response shapes for the /admin/* endpoints the admin app consumes.
// Mirrors apps/api/handlers/admin.go, approval.go, delivery.go, staff.go,
// admin_wallet.go, admin_reviews.go, meal_plan.go. Fields the UI doesn't use
// are intentionally omitted; extras on the wire are ignored.

export interface PaginationMeta {
  page: number;
  limit: number;
  total: number;
  totalPages: number;
  hasNext: boolean;
  hasPrev: boolean;
}

export interface Paginated<T> {
  data: T[];
  pagination: PaginationMeta;
}

// ---- Dashboard --------------------------------------------------------------
export interface AdminStats {
  totalUsers: number;
  newUsersToday: number;
  totalChefs: number;
  pendingVerifications: number;
  totalOrders: number;
  ordersToday: number;
  revenue: number;
  revenueToday: number;
  revenueChange: number;
  ordersChange: number;
}

export interface Activity {
  id: string;
  type: 'order' | 'user' | 'chef' | string;
  title: string;
  description: string;
  timestamp: string;
}

export interface AdminAnalytics {
  overview: {
    totalRevenue: number;
    totalOrders: number;
    avgOrderValue: number;
    activeUsers: number;
  };
  ordersByStatus: Record<string, number>;
}

// ---- Users ------------------------------------------------------------------
export interface UserWithStats {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  phone: string;
  role: string;
  isActive: boolean;
  totalOrders: number;
  totalSpent: number;
  lastOrderAt?: string | null;
  createdAt?: string;
}

// ---- Chefs ------------------------------------------------------------------
export interface ChefWithStats {
  id: string;
  userId: string;
  businessName: string;
  cuisines: string[];
  kitchenType?: string;
  isVerified: boolean;
  isActive: boolean;
  acceptingOrders: boolean;
  rating: number;
  createdAt: string;
  ownerName: string;
  ownerEmail: string;
  ownerPhone: string;
  totalOrders: number;
  totalRevenue: number;
  menuItemCount: number;
  documentCount: number;
  onlineStatus: 'online' | 'away' | 'offline' | string;
}

export interface FSSAILockedChef {
  chefId: string;
  userId: string;
  businessName: string;
  fssaiExpiry?: string | null;
  daysSinceExpiry: number;
  overrideUntil?: string | null;
  overrideReason?: string;
  overrideBy?: string | null;
}

export interface FSSAILockResponse {
  locked: FSSAILockedChef[];
  overridden: FSSAILockedChef[];
  lockedCount: number;
  overriddenCount: number;
  missingExpiryCount: number;
}

// ---- Orders -----------------------------------------------------------------
export interface OrderRow {
  id: string;
  orderNumber: string;
  status: string;
  total: number;
  paymentStatus: string;
  createdAt: string;
  customerName: string;
  chefName: string;
  itemCount: number;
}

export interface OrderDetail {
  order: Record<string, unknown> & {
    id: string;
    orderNumber?: string;
    status?: string;
    total?: number;
    subtotal?: number;
    deliveryFee?: number;
    tax?: number;
    paymentStatus?: string;
    createdAt?: string;
    deliveryAddress?: string;
    items?: { name: string; quantity: number; price: number }[];
  };
  customer: { id: string; name: string; email: string; phone: string; createdAt?: string };
  chef: { id: string; businessName: string; city?: string };
}

// ---- Approvals --------------------------------------------------------------
export type ApprovalType =
  | 'kitchen_onboarding'
  | 'document_verification'
  | 'menu_item_new'
  | 'menu_item_update'
  | 'pricing_change'
  | 'kitchen_update'
  | 'driver_onboarding'
  | 'driver_document'
  | string;

export type ApprovalStatus = 'pending' | 'approved' | 'rejected' | 'info_requested';
export type ApprovalPriority = 'urgent' | 'high' | 'normal' | 'low';

export interface ApprovalRequest {
  id: string;
  type: ApprovalType;
  status: ApprovalStatus;
  priority: ApprovalPriority;
  chefId?: string | null;
  partnerId?: string | null;
  submittedById: string;
  reviewedById?: string | null;
  entityType: string;
  entityId: string;
  title: string;
  description: string;
  submittedData?: Record<string, unknown>;
  adminNotes?: string;
  reviewedAt?: string | null;
  createdAt: string;
  updatedAt: string;
  chef?: Record<string, unknown>;
  partner?: Record<string, unknown>;
  documents?: ApprovalDocument[];
  fssaiLooksCommercial?: boolean | null;
  kitchenTypeNonHome?: boolean | null;
}

export interface ApprovalDocument {
  id: string;
  type?: string;
  fileName?: string;
  status?: string;
}

export interface ApprovalCounts {
  byStatus: Record<string, number>;
  byType: Record<string, number>;
  total: number;
}

// ---- Reviews ----------------------------------------------------------------
export interface ReviewRow {
  id: string;
  chefId: string;
  customerId: string;
  rating: number;
  text?: string;
  comment?: string;
  isHidden: boolean;
  hiddenReason?: string;
  createdAt: string;
}

// ---- Wallet -----------------------------------------------------------------
export interface WalletTxn {
  id: string;
  userId: string;
  amount: number;
  type: 'credit' | 'debit' | string;
  source: string;
  reason: string;
  balanceAfter: number;
  createdAt: string;
}

export interface WalletResponse {
  balance: number;
  currency: string;
  transactions: WalletTxn[];
  count: number;
}

// ---- Meal Plans -------------------------------------------------------------
export interface MealPlanRow {
  id: string;
  chefId: string;
  customerId: string;
  status: string;
  startDate?: string;
  endDate?: string;
  mealCount?: number;
  daysPerWeek?: number;
  pricePerMeal?: number;
  totalPrice?: number;
  createdAt: string;
}

// ---- Delivery ---------------------------------------------------------------
export interface DeliveryStats {
  totalPartners: number;
  verifiedPartners: number;
  onlinePartners: number;
  totalDeliveries: number;
  activeDeliveries: number;
  todayDeliveries: number;
  todayEarnings: number;
}

export interface DeliveryPartnerSummary {
  id: string;
  userId: string;
  name: string;
  email: string;
  phone: string;
  vehicleType: string;
  vehicleNumber?: string;
  isVerified: boolean;
  isOnline: boolean;
  isActive: boolean;
  rating: number;
  totalDeliveries: number;
  createdAt: string;
}

export interface DeliveryRow {
  id: string;
  orderId: string;
  status: string;
  distance?: number;
  deliveryFee?: number;
  tip?: number;
  totalPayout?: number;
  pickup?: { address?: string; city?: string };
  dropoff?: { address?: string; city?: string };
  createdAt?: string;
}

// ---- Staff ------------------------------------------------------------------
export type StaffRole =
  | 'super_admin'
  | 'admin'
  | 'support'
  | 'fleet_manager'
  | 'delivery_ops'
  | string;

export interface StaffMember {
  id: string;
  userId: string;
  user?: { email?: string; firstName?: string; lastName?: string };
  staffRole: StaffRole;
  department?: string;
  title?: string;
  isActive: boolean;
  permissions?: string[];
  lastActiveAt?: string;
  createdAt: string;
}
