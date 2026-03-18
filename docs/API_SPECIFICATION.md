# HomeChef Platform - API Specification

**Version:** 1.0
**Base URL:** `https://api.homechef.com/v1`
**Last Updated:** December 2024

---

## Table of Contents

1. [Overview](#1-overview)
2. [Authentication](#2-authentication)
3. [Common Patterns](#3-common-patterns)
4. [API Endpoints](#4-api-endpoints)
5. [WebSocket Events](#5-websocket-events)
6. [Error Codes](#6-error-codes)

---

## 1. Overview

### 1.1 API Design Principles

- **RESTful**: Resource-based URLs with standard HTTP methods
- **JSON**: All requests and responses use JSON
- **Versioned**: API version in URL path (`/v1/`)
- **Authenticated**: JWT Bearer tokens for protected endpoints
- **Rate Limited**: 100 requests/minute per user
- **Paginated**: List endpoints support pagination

### 1.2 Base URLs

| Environment | URL |
|-------------|-----|
| Production | `https://api.homechef.com/v1` |
| Staging | `https://api-staging.homechef.com/v1` |
| Development | `http://localhost:8080/v1` |
| Mock | `http://localhost:3000/api/mock/v1` |

### 1.3 Request Headers

```http
Content-Type: application/json
Accept: application/json
Authorization: Bearer <jwt_token>
X-Request-ID: <uuid>  # Optional, for request tracing
X-Client-Version: 1.0.0  # Optional, client app version
```

---

## 2. Authentication

### 2.1 Authentication Flow

```
┌──────────┐      ┌──────────┐      ┌──────────┐
│  Client  │      │   API    │      │  OAuth   │
│   App    │      │  Server  │      │ Provider │
└────┬─────┘      └────┬─────┘      └────┬─────┘
     │                 │                  │
     │  POST /auth/login                  │
     │────────────────>│                  │
     │                 │                  │
     │  { token, refreshToken }           │
     │<────────────────│                  │
     │                 │                  │
     │  POST /auth/social                 │
     │────────────────>│                  │
     │                 │  Verify Token    │
     │                 │─────────────────>│
     │                 │  User Info       │
     │                 │<─────────────────│
     │  { token, refreshToken }           │
     │<────────────────│                  │
```

### 2.2 JWT Token Structure

```json
{
  "sub": "user-uuid",
  "email": "user@example.com",
  "role": "customer",
  "exp": 1700003600,
  "iat": 1700000000
}
```

### 2.3 Token Refresh

Access tokens expire in 1 hour. Use refresh tokens (7-day validity) to get new access tokens.

---

## 3. Common Patterns

### 3.1 Success Response

```json
{
  "success": true,
  "data": { ... },
  "meta": {
    "timestamp": "2024-01-01T12:00:00Z",
    "requestId": "req_abc123"
  }
}
```

### 3.2 Error Response

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input",
    "details": [
      {
        "field": "email",
        "message": "Invalid email format"
      }
    ]
  },
  "meta": {
    "timestamp": "2024-01-01T12:00:00Z",
    "requestId": "req_abc123"
  }
}
```

### 3.3 Pagination

**Request:**
```http
GET /chefs?page=1&limit=20&sort=rating&order=desc
```

**Response:**
```json
{
  "success": true,
  "data": [ ... ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 156,
    "totalPages": 8,
    "hasNext": true,
    "hasPrev": false
  }
}
```

### 3.4 Filtering

```http
GET /orders?status=pending,accepted&chef_id=uuid&from=2024-01-01&to=2024-12-31
```

---

## 4. API Endpoints

### 4.1 Authentication Endpoints

#### POST /auth/register
Register a new user account.

**Request:**
```json
{
  "email": "user@example.com",
  "phone": "+1234567890",
  "password": "securePassword123",
  "firstName": "John",
  "lastName": "Doe",
  "role": "customer"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "firstName": "John",
      "lastName": "Doe",
      "role": "customer",
      "emailVerified": false,
      "phoneVerified": false
    },
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "refresh_token_here",
    "expiresIn": 3600
  }
}
```

---

#### POST /auth/login
Login with email and password.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "securePassword123"
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "firstName": "John",
      "lastName": "Doe",
      "role": "customer",
      "avatar": "https://..."
    },
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "refresh_token_here",
    "expiresIn": 3600
  }
}
```

---

#### POST /auth/social
Authenticate via social provider.

**Request:**
```json
{
  "provider": "google",
  "token": "oauth_token_from_provider",
  "role": "customer"
}
```

**Response (200):** Same as `/auth/login`

---

#### POST /auth/verify-phone
Send OTP to phone number.

**Request:**
```json
{
  "phone": "+1234567890"
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "message": "OTP sent successfully",
    "expiresIn": 300
  }
}
```

---

#### POST /auth/verify-otp
Verify phone OTP.

**Request:**
```json
{
  "phone": "+1234567890",
  "otp": "123456"
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "verified": true,
    "user": { ... },
    "token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

---

#### POST /auth/refresh
Refresh access token.

**Request:**
```json
{
  "refreshToken": "refresh_token_here"
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "token": "new_access_token",
    "refreshToken": "new_refresh_token",
    "expiresIn": 3600
  }
}
```

---

#### POST /auth/logout
Logout and invalidate tokens.

**Headers:** `Authorization: Bearer <token>`

**Response (200):**
```json
{
  "success": true,
  "data": {
    "message": "Logged out successfully"
  }
}
```

---

### 4.2 User Endpoints

#### GET /users/me
Get current user profile.

**Headers:** `Authorization: Bearer <token>`

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "phone": "+1234567890",
    "firstName": "John",
    "lastName": "Doe",
    "avatar": "https://...",
    "role": "customer",
    "emailVerified": true,
    "phoneVerified": true,
    "preferences": {
      "dietary": ["vegetarian"],
      "notifications": {
        "push": true,
        "email": true,
        "sms": false
      }
    },
    "createdAt": "2024-01-01T00:00:00Z"
  }
}
```

---

#### PUT /users/me
Update current user profile.

**Request:**
```json
{
  "firstName": "John",
  "lastName": "Smith",
  "avatar": "https://...",
  "preferences": {
    "dietary": ["vegetarian", "gluten-free"]
  }
}
```

**Response (200):** Updated user object

---

#### GET /users/me/addresses
Get user's saved addresses.

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "label": "Home",
      "line1": "123 Main St",
      "line2": "Apt 4B",
      "city": "San Francisco",
      "state": "CA",
      "postalCode": "94102",
      "country": "United States",
      "latitude": 37.7749,
      "longitude": -122.4194,
      "isDefault": true,
      "deliveryInstructions": "Ring doorbell twice"
    }
  ]
}
```

---

#### POST /users/me/addresses
Add new address.

**Request:**
```json
{
  "label": "Work",
  "line1": "456 Market St",
  "city": "San Francisco",
  "state": "CA",
  "postalCode": "94103",
  "country": "United States",
  "latitude": 37.7899,
  "longitude": -122.4000,
  "isDefault": false
}
```

**Response (201):** Created address object

---

#### PUT /users/me/addresses/:id
Update address.

**Response (200):** Updated address object

---

#### DELETE /users/me/addresses/:id
Delete address.

**Response (204):** No content

---

### 4.3 Chef Discovery Endpoints

#### GET /chefs
List chefs with filtering and search.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `lat` | float | User latitude (required for nearby) |
| `lng` | float | User longitude (required for nearby) |
| `radius` | int | Search radius in km (default: 10) |
| `cuisine` | string | Filter by cuisine type |
| `search` | string | Search by name, description |
| `rating` | float | Minimum rating (1-5) |
| `priceRange` | string | Price range ($, $$, $$$) |
| `dietary` | string | Dietary filters (vegetarian, vegan, etc.) |
| `isOpen` | boolean | Currently accepting orders |
| `sort` | string | Sort field (rating, distance, orders) |
| `order` | string | Sort order (asc, desc) |
| `page` | int | Page number |
| `limit` | int | Items per page (max 50) |

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "businessName": "Meena's Kitchen",
      "description": "Authentic South Indian home cooking...",
      "cuisines": ["South Indian", "Tamil"],
      "specialties": ["Dosa", "Biryani"],
      "profileImage": "https://...",
      "rating": 4.8,
      "totalReviews": 156,
      "totalOrders": 423,
      "isOnline": true,
      "acceptingOrders": true,
      "distance": 2.5,
      "prepTime": "30-45 mins",
      "priceRange": "$$",
      "deliveryFee": 2.99,
      "minimumOrder": 15
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45,
    "totalPages": 3
  }
}
```

---

#### GET /chefs/:id
Get chef details.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "businessName": "Meena's Kitchen",
    "description": "Authentic South Indian home cooking with a modern twist...",
    "cuisines": ["South Indian", "Tamil"],
    "specialties": ["Dosa", "Biryani", "Sambar"],
    "profileImage": "https://...",
    "bannerImage": "https://...",
    "rating": 4.8,
    "totalReviews": 156,
    "totalOrders": 423,
    "isOnline": true,
    "acceptingOrders": true,
    "prepTime": "30-45 mins",
    "priceRange": "$$",
    "deliveryFee": 2.99,
    "minimumOrder": 15,
    "serviceRadius": 10,
    "verified": true,
    "verifiedAt": "2024-01-15T10:00:00Z",
    "operatingHours": {
      "monday": { "open": "08:00", "close": "21:00" },
      "tuesday": { "open": "08:00", "close": "21:00" },
      "wednesday": { "open": "08:00", "close": "21:00" },
      "thursday": { "open": "08:00", "close": "21:00" },
      "friday": { "open": "08:00", "close": "22:00" },
      "saturday": { "open": "09:00", "close": "22:00" },
      "sunday": { "open": "09:00", "close": "20:00" }
    },
    "location": {
      "city": "San Francisco",
      "state": "CA",
      "distance": 2.5
    }
  }
}
```

---

#### GET /chefs/:id/menu
Get chef's menu items.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `category` | string | Filter by category |
| `available` | boolean | Only available items |

**Response (200):**
```json
{
  "success": true,
  "data": {
    "categories": [
      {
        "id": "uuid",
        "name": "Breakfast",
        "description": "Morning specialties",
        "items": [
          {
            "id": "uuid",
            "name": "Masala Dosa",
            "description": "Crispy crepe with spiced potato filling...",
            "price": 8.99,
            "imageUrl": "https://...",
            "dietaryTags": ["vegetarian", "gluten-free"],
            "allergens": [],
            "prepTime": 20,
            "isAvailable": true,
            "isFeatured": true,
            "portionSize": "Regular",
            "serves": 1
          }
        ]
      }
    ],
    "featuredItems": [ ... ],
    "totalItems": 24
  }
}
```

---

#### GET /chefs/:id/reviews
Get chef reviews.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `rating` | int | Filter by rating |
| `sort` | string | Sort by (recent, rating) |
| `page` | int | Page number |
| `limit` | int | Items per page |

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "customer": {
        "firstName": "John",
        "avatar": "https://..."
      },
      "foodRating": 5,
      "overallRating": 4.5,
      "comment": "Amazing dosas! Just like my mom makes...",
      "images": ["https://..."],
      "chefResponse": "Thank you for the kind words!",
      "createdAt": "2024-01-01T12:00:00Z"
    }
  ],
  "stats": {
    "averageRating": 4.8,
    "totalReviews": 156,
    "distribution": {
      "5": 120,
      "4": 25,
      "3": 8,
      "2": 2,
      "1": 1
    }
  },
  "pagination": { ... }
}
```

---

### 4.4 Cart Endpoints

#### GET /cart
Get current cart.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "chefId": "uuid",
    "chef": {
      "businessName": "Meena's Kitchen",
      "profileImage": "https://...",
      "deliveryFee": 2.99,
      "minimumOrder": 15
    },
    "items": [
      {
        "id": "uuid",
        "menuItemId": "uuid",
        "name": "Masala Dosa",
        "price": 8.99,
        "quantity": 2,
        "subtotal": 17.98,
        "notes": "Extra chutney please",
        "imageUrl": "https://..."
      }
    ],
    "subtotal": 25.97,
    "itemCount": 3
  }
}
```

---

#### POST /cart/items
Add item to cart.

**Request:**
```json
{
  "menuItemId": "uuid",
  "quantity": 2,
  "notes": "Extra spicy",
  "customizations": {
    "spiceLevel": "hot"
  }
}
```

**Response (200):** Updated cart object

---

#### PUT /cart/items/:id
Update cart item.

**Request:**
```json
{
  "quantity": 3,
  "notes": "Less spicy please"
}
```

**Response (200):** Updated cart object

---

#### DELETE /cart/items/:id
Remove item from cart.

**Response (200):** Updated cart object

---

#### DELETE /cart
Clear entire cart.

**Response (204):** No content

---

### 4.5 Order Endpoints

#### POST /orders
Create a new order.

**Request:**
```json
{
  "chefId": "uuid",
  "items": [
    {
      "menuItemId": "uuid",
      "quantity": 2,
      "notes": "Extra chutney"
    }
  ],
  "deliveryAddressId": "uuid",
  "orderType": "delivery",
  "scheduledFor": null,
  "promoCode": "SAVE10",
  "tip": 5.00,
  "specialInstructions": "Please ring the doorbell",
  "paymentMethod": "card"
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "orderNumber": "HC1234567890",
    "status": "pending",
    "chef": {
      "id": "uuid",
      "businessName": "Meena's Kitchen",
      "phone": "+1234567890"
    },
    "items": [ ... ],
    "deliveryAddress": {
      "line1": "123 Main St",
      "city": "San Francisco",
      "postalCode": "94102"
    },
    "pricing": {
      "subtotal": 25.97,
      "deliveryFee": 2.99,
      "serviceFee": 1.30,
      "tax": 2.34,
      "discount": 2.60,
      "tip": 5.00,
      "total": 35.00
    },
    "promoCode": "SAVE10",
    "estimatedReadyAt": "2024-01-01T12:30:00Z",
    "estimatedDeliveryAt": "2024-01-01T12:45:00Z",
    "payment": {
      "status": "pending",
      "clientSecret": "pi_xxx_secret_xxx"
    },
    "createdAt": "2024-01-01T12:00:00Z"
  }
}
```

---

#### GET /orders
Get customer's orders.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | string | Filter by status(es) |
| `from` | date | Start date |
| `to` | date | End date |
| `page` | int | Page number |
| `limit` | int | Items per page |

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "orderNumber": "HC1234567890",
      "status": "delivered",
      "chef": {
        "businessName": "Meena's Kitchen",
        "profileImage": "https://..."
      },
      "items": [
        {
          "name": "Masala Dosa",
          "quantity": 2,
          "price": 8.99
        }
      ],
      "total": 35.00,
      "deliveredAt": "2024-01-01T12:45:00Z",
      "canReview": true,
      "canReorder": true,
      "createdAt": "2024-01-01T12:00:00Z"
    }
  ],
  "pagination": { ... }
}
```

---

#### GET /orders/:id
Get order details.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "orderNumber": "HC1234567890",
    "status": "preparing",
    "statusHistory": [
      { "status": "pending", "timestamp": "2024-01-01T12:00:00Z" },
      { "status": "accepted", "timestamp": "2024-01-01T12:02:00Z" },
      { "status": "preparing", "timestamp": "2024-01-01T12:05:00Z" }
    ],
    "chef": {
      "id": "uuid",
      "businessName": "Meena's Kitchen",
      "profileImage": "https://...",
      "phone": "+1234567890",
      "location": {
        "latitude": 37.7749,
        "longitude": -122.4194
      }
    },
    "items": [
      {
        "id": "uuid",
        "name": "Masala Dosa",
        "description": "Crispy crepe...",
        "price": 8.99,
        "quantity": 2,
        "subtotal": 17.98,
        "imageUrl": "https://...",
        "notes": "Extra chutney"
      }
    ],
    "deliveryAddress": {
      "line1": "123 Main St",
      "line2": "Apt 4B",
      "city": "San Francisco",
      "state": "CA",
      "postalCode": "94102",
      "latitude": 37.7849,
      "longitude": -122.4094
    },
    "delivery": {
      "partner": {
        "firstName": "Dinesh",
        "phone": "+1234567890",
        "photo": "https://...",
        "rating": 4.9
      },
      "currentLocation": {
        "latitude": 37.7799,
        "longitude": -122.4144
      },
      "status": "picked_up",
      "eta": "10 mins"
    },
    "pricing": {
      "subtotal": 25.97,
      "deliveryFee": 2.99,
      "serviceFee": 1.30,
      "tax": 2.34,
      "discount": 2.60,
      "tip": 5.00,
      "total": 35.00
    },
    "payment": {
      "status": "completed",
      "method": "card",
      "last4": "4242"
    },
    "estimatedDeliveryAt": "2024-01-01T12:45:00Z",
    "specialInstructions": "Please ring the doorbell",
    "createdAt": "2024-01-01T12:00:00Z"
  }
}
```

---

#### PUT /orders/:id/cancel
Cancel an order.

**Request:**
```json
{
  "reason": "Changed my mind"
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "status": "cancelled",
    "cancelledAt": "2024-01-01T12:10:00Z",
    "refund": {
      "amount": 35.00,
      "status": "processing",
      "estimatedDate": "2024-01-03T00:00:00Z"
    }
  }
}
```

---

#### POST /orders/:id/review
Submit order review.

**Request:**
```json
{
  "foodRating": 5,
  "packagingRating": 4,
  "valueRating": 5,
  "deliveryRating": 5,
  "comment": "Amazing food! The dosas were perfect.",
  "images": ["https://..."]
}
```

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "foodRating": 5,
    "overallRating": 4.75,
    "comment": "Amazing food!...",
    "createdAt": "2024-01-01T13:00:00Z"
  }
}
```

---

### 4.6 Chef Dashboard Endpoints

#### GET /chef/profile
Get chef's own profile.

**Response (200):** Full chef profile object

---

#### PUT /chef/profile
Update chef profile.

**Request:**
```json
{
  "businessName": "Meena's Kitchen",
  "description": "Updated description...",
  "cuisines": ["South Indian", "Tamil"],
  "operatingHours": { ... },
  "minimumOrder": 15,
  "deliveryFee": 2.99
}
```

**Response (200):** Updated profile

---

#### PUT /chef/availability
Update availability status.

**Request:**
```json
{
  "isOnline": true,
  "acceptingOrders": true
}
```

**Response (200):** Updated status

---

#### GET /chef/menu
Get chef's menu for management.

**Response (200):** Full menu with categories

---

#### POST /chef/menu
Add menu item.

**Request:**
```json
{
  "categoryId": "uuid",
  "name": "Butter Chicken",
  "description": "Creamy tomato-based chicken curry...",
  "price": 14.99,
  "imageUrl": "https://...",
  "dietaryTags": ["gluten-free"],
  "allergens": ["dairy"],
  "prepTime": 25,
  "isAvailable": true,
  "portionSize": "Regular",
  "serves": 2
}
```

**Response (201):** Created menu item

---

#### PUT /chef/menu/:id
Update menu item.

**Response (200):** Updated menu item

---

#### DELETE /chef/menu/:id
Delete menu item.

**Response (204):** No content

---

#### GET /chef/orders
Get chef's orders.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `status` | string | Filter by status |
| `from` | date | Start date |
| `to` | date | End date |

**Response (200):**
```json
{
  "success": true,
  "data": {
    "pending": [
      {
        "id": "uuid",
        "orderNumber": "HC123",
        "customer": {
          "firstName": "John",
          "phone": "+1234567890"
        },
        "items": [ ... ],
        "total": 35.00,
        "scheduledFor": null,
        "specialInstructions": "...",
        "createdAt": "2024-01-01T12:00:00Z"
      }
    ],
    "active": [ ... ],
    "completed": [ ... ]
  }
}
```

---

#### PUT /chef/orders/:id/status
Update order status.

**Request:**
```json
{
  "status": "preparing",
  "estimatedReadyAt": "2024-01-01T12:30:00Z"
}
```

**Response (200):** Updated order

---

#### GET /chef/earnings
Get earnings summary.

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `period` | string | daily, weekly, monthly |
| `from` | date | Start date |
| `to` | date | End date |

**Response (200):**
```json
{
  "success": true,
  "data": {
    "summary": {
      "totalEarnings": 4523.50,
      "totalOrders": 156,
      "averageOrderValue": 29.00,
      "platformFees": 678.52,
      "netEarnings": 3844.98
    },
    "breakdown": [
      {
        "date": "2024-01-01",
        "orders": 12,
        "gross": 348.00,
        "fees": 52.20,
        "net": 295.80
      }
    ],
    "pendingPayout": {
      "amount": 845.30,
      "scheduledDate": "2024-01-07"
    }
  }
}
```

---

#### GET /chef/analytics
Get analytics data.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "overview": {
      "totalOrders": 423,
      "rating": 4.8,
      "repeatCustomerRate": 0.42,
      "acceptanceRate": 0.95
    },
    "popularItems": [
      {
        "id": "uuid",
        "name": "Masala Dosa",
        "orders": 245,
        "revenue": 2202.55
      }
    ],
    "peakHours": [
      { "hour": 12, "orders": 45 },
      { "hour": 19, "orders": 62 }
    ],
    "customerDemographics": {
      "newVsReturning": { "new": 0.58, "returning": 0.42 },
      "topLocations": [ ... ]
    }
  }
}
```

---

### 4.7 Delivery Partner Endpoints

#### GET /delivery/profile
Get delivery partner profile.

**Response (200):** Full partner profile

---

#### PUT /delivery/availability
Update availability.

**Request:**
```json
{
  "isOnline": true,
  "currentLatitude": 37.7749,
  "currentLongitude": -122.4194
}
```

**Response (200):** Updated status

---

#### POST /delivery/location
Update current location.

**Request:**
```json
{
  "latitude": 37.7749,
  "longitude": -122.4194
}
```

**Response (200):** Acknowledged

---

#### GET /delivery/orders
Get assigned deliveries.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "current": {
      "id": "uuid",
      "order": {
        "id": "uuid",
        "orderNumber": "HC123",
        "items": [ ... ],
        "specialInstructions": "..."
      },
      "pickup": {
        "chef": {
          "businessName": "Meena's Kitchen",
          "phone": "+1234567890"
        },
        "address": "...",
        "latitude": 37.7749,
        "longitude": -122.4194
      },
      "dropoff": {
        "customer": {
          "firstName": "John",
          "phone": "+1234567890"
        },
        "address": "...",
        "latitude": 37.7849,
        "longitude": -122.4094,
        "instructions": "Ring doorbell"
      },
      "status": "picked_up",
      "earnings": {
        "base": 4.00,
        "distance": 1.50,
        "tip": 5.00,
        "total": 10.50
      }
    },
    "available": [ ... ],
    "completed": [ ... ]
  }
}
```

---

#### PUT /delivery/orders/:id/status
Update delivery status.

**Request:**
```json
{
  "status": "picked_up"
}
```

---

#### POST /delivery/orders/:id/complete
Complete delivery with proof.

**Request:**
```json
{
  "proofType": "photo",
  "proofImage": "https://...",
  "notes": "Left at door"
}
```

**Response (200):** Completed delivery details

---

#### GET /delivery/earnings
Get earnings summary.

**Response (200):** Similar to chef earnings

---

### 4.8 Admin Endpoints

#### GET /admin/dashboard
Get admin dashboard stats.

**Response (200):**
```json
{
  "success": true,
  "data": {
    "today": {
      "orders": 1234,
      "revenue": 45678.90,
      "newCustomers": 89,
      "newChefs": 5,
      "activeDeliveries": 45
    },
    "growth": {
      "ordersChange": 0.12,
      "revenueChange": 0.15
    },
    "alerts": [
      {
        "type": "pending_verification",
        "count": 12,
        "message": "12 chefs awaiting verification"
      }
    ]
  }
}
```

---

#### GET /admin/users
List all users with filters.

**Query Parameters:**
- `role`, `status`, `search`, `page`, `limit`

**Response (200):** Paginated user list

---

#### GET /admin/chefs
List all chefs.

---

#### PUT /admin/chefs/:id/verify
Verify/reject chef.

**Request:**
```json
{
  "status": "approved",
  "notes": "All documents verified"
}
```

---

#### GET /admin/orders
List all orders.

---

#### POST /admin/orders/:id/refund
Process refund.

**Request:**
```json
{
  "amount": 35.00,
  "reason": "Customer complaint - quality issue"
}
```

---

#### GET /admin/analytics
Get platform analytics.

---

#### POST /admin/promo-codes
Create promo code.

**Request:**
```json
{
  "code": "SAVE20",
  "discountType": "percentage",
  "discountValue": 20,
  "maxDiscount": 50,
  "minOrderAmount": 30,
  "usageLimit": 1000,
  "perUserLimit": 1,
  "validFrom": "2024-01-01T00:00:00Z",
  "validUntil": "2024-01-31T23:59:59Z",
  "newUsersOnly": true
}
```

---

### 4.9 Payment Endpoints

#### POST /payments/intent
Create payment intent.

**Request:**
```json
{
  "orderId": "uuid",
  "paymentMethod": "card"
}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "clientSecret": "pi_xxx_secret_xxx",
    "paymentIntentId": "pi_xxx",
    "amount": 3500,
    "currency": "usd"
  }
}
```

---

#### POST /payments/confirm
Confirm payment.

**Request:**
```json
{
  "paymentIntentId": "pi_xxx",
  "orderId": "uuid"
}
```

**Response (200):** Payment confirmation

---

### 4.10 Notification Endpoints

#### GET /notifications
Get user notifications.

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "type": "order_delivered",
      "title": "Order Delivered",
      "body": "Your order #HC123 has been delivered!",
      "isRead": false,
      "actionType": "view_order",
      "actionData": { "orderId": "uuid" },
      "createdAt": "2024-01-01T12:45:00Z"
    }
  ],
  "unreadCount": 5
}
```

---

#### PUT /notifications/:id/read
Mark notification as read.

---

#### PUT /notifications/read-all
Mark all as read.

---

### 4.11 Search Endpoints

#### GET /search
Global search.

**Query Parameters:**
- `q`: Search query
- `type`: chefs, items, all
- `lat`, `lng`: Location for relevance

**Response (200):**
```json
{
  "success": true,
  "data": {
    "chefs": [ ... ],
    "items": [ ... ],
    "cuisines": ["Italian", "Indian"]
  }
}
```

---

## 5. WebSocket Events

### 5.1 Connection

```javascript
const socket = new WebSocket('wss://api.homechef.com/ws');

socket.onopen = () => {
  socket.send(JSON.stringify({
    type: 'authenticate',
    token: 'jwt_token_here'
  }));
};
```

### 5.2 Event Types

#### Order Status Update
```json
{
  "type": "ORDER_STATUS_UPDATE",
  "payload": {
    "orderId": "uuid",
    "status": "preparing",
    "estimatedReadyAt": "2024-01-01T12:30:00Z",
    "message": "Your food is being prepared"
  }
}
```

#### Delivery Location Update
```json
{
  "type": "DELIVERY_LOCATION",
  "payload": {
    "orderId": "uuid",
    "location": {
      "latitude": 37.7799,
      "longitude": -122.4144
    },
    "eta": "8 mins",
    "status": "delivering"
  }
}
```

#### New Order (Chef)
```json
{
  "type": "NEW_ORDER",
  "payload": {
    "orderId": "uuid",
    "orderNumber": "HC123",
    "items": [...],
    "total": 35.00,
    "customerName": "John D."
  }
}
```

#### Delivery Assignment (Partner)
```json
{
  "type": "DELIVERY_ASSIGNED",
  "payload": {
    "deliveryId": "uuid",
    "order": {...},
    "pickup": {...},
    "dropoff": {...},
    "earnings": {...}
  }
}
```

---

## 6. Error Codes

### 6.1 HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 204 | No Content |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict |
| 422 | Validation Error |
| 429 | Too Many Requests |
| 500 | Internal Server Error |

### 6.2 Application Error Codes

| Code | Description |
|------|-------------|
| `AUTH_INVALID_CREDENTIALS` | Invalid email/password |
| `AUTH_TOKEN_EXPIRED` | JWT token expired |
| `AUTH_INVALID_TOKEN` | Invalid JWT token |
| `AUTH_SOCIAL_ERROR` | Social login failed |
| `USER_NOT_FOUND` | User does not exist |
| `USER_ALREADY_EXISTS` | Email already registered |
| `CHEF_NOT_FOUND` | Chef does not exist |
| `CHEF_UNAVAILABLE` | Chef not accepting orders |
| `CHEF_OUTSIDE_RANGE` | Address outside service area |
| `MENU_ITEM_UNAVAILABLE` | Item not available |
| `ORDER_NOT_FOUND` | Order does not exist |
| `ORDER_CANNOT_CANCEL` | Order too far in progress |
| `ORDER_MINIMUM_NOT_MET` | Below minimum order |
| `CART_CHEF_MISMATCH` | Items from different chefs |
| `PAYMENT_FAILED` | Payment processing failed |
| `PROMO_INVALID` | Invalid promo code |
| `PROMO_EXPIRED` | Promo code expired |
| `PROMO_LIMIT_REACHED` | Promo usage limit reached |
| `VALIDATION_ERROR` | Input validation failed |
| `RATE_LIMIT_EXCEEDED` | Too many requests |

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | Dec 2024 | HomeChef Team | Initial API specification |

---

*This document is confidential and intended for internal development use only.*
