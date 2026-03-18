# HomeChef Platform - Database Schema

**Version:** 1.0
**Database:** PostgreSQL 15+
**Last Updated:** December 2024

---

## Table of Contents

1. [Overview](#1-overview)
2. [Schema Diagram](#2-schema-diagram)
3. [Tables](#3-tables)
4. [Indexes](#4-indexes)
5. [Migrations](#5-migrations)

---

## 1. Overview

### 1.1 Design Principles

- **UUID Primary Keys**: Using UUIDs for distributed system compatibility
- **Soft Deletes**: `deleted_at` column for audit trail
- **Timestamps**: `created_at` and `updated_at` on all tables
- **Normalization**: 3NF with strategic denormalization for performance
- **JSONB**: Used for flexible schema fields (operating hours, preferences)

### 1.2 Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Tables | snake_case, plural | `menu_items` |
| Columns | snake_case | `first_name` |
| Primary Keys | `id` | `id` |
| Foreign Keys | `<table>_id` | `user_id` |
| Indexes | `idx_<table>_<columns>` | `idx_users_email` |
| Constraints | `<type>_<table>_<column>` | `fk_orders_customer_id` |

---

## 2. Schema Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              HOMECHEF DATABASE SCHEMA                           │
└─────────────────────────────────────────────────────────────────────────────────┘

┌──────────────────┐       ┌──────────────────┐       ┌──────────────────┐
│      users       │       │  chef_profiles   │       │   menu_items     │
├──────────────────┤       ├──────────────────┤       ├──────────────────┤
│ id (PK)          │──┐    │ id (PK)          │──┐    │ id (PK)          │
│ email            │  │    │ user_id (FK)     │──┘    │ chef_id (FK)     │──┘
│ phone            │  │    │ business_name    │       │ name             │
│ password_hash    │  │    │ description      │       │ description      │
│ first_name       │  │    │ cuisines[]       │       │ price            │
│ last_name        │  │    │ specialties[]    │       │ image_url        │
│ avatar           │  │    │ profile_image    │       │ category_id (FK) │──┐
│ role             │  │    │ banner_image     │       │ dietary_tags[]   │  │
│ email_verified   │  │    │ status           │       │ allergens[]      │  │
│ phone_verified   │  │    │ rating           │       │ prep_time        │  │
│ is_active        │  │    │ total_reviews    │       │ is_available     │  │
│ last_login_at    │  │    │ total_orders     │       │ sort_order       │  │
│ created_at       │  │    │ is_online        │       │ created_at       │  │
│ updated_at       │  │    │ accepting_orders │       │ updated_at       │  │
│ deleted_at       │  │    │ latitude         │       │ deleted_at       │  │
└────────┬─────────┘  │    │ longitude        │       └──────────────────┘  │
         │            │    │ service_radius   │                             │
         │            │    │ operating_hours  │       ┌──────────────────┐  │
         │            │    │ min_order        │       │ menu_categories  │  │
         │            │    │ delivery_fee     │       ├──────────────────┤  │
         │            │    │ prep_time        │       │ id (PK)          │<─┘
         │            │    │ id_verified      │       │ chef_id (FK)     │
         │            │    │ kitchen_verified │       │ name             │
         │            │    │ food_license     │       │ description      │
         │            │    │ verified_at      │       │ sort_order       │
         │            │    │ bank_account_id  │       │ is_active        │
         │            │    │ created_at       │       └──────────────────┘
         │            │    │ updated_at       │
         │            │    │ deleted_at       │
         │            │    └──────────────────┘
         │            │
┌────────▼─────────┐  │    ┌──────────────────┐       ┌──────────────────┐
│    addresses     │  │    │      orders      │       │   order_items    │
├──────────────────┤  │    ├──────────────────┤       ├──────────────────┤
│ id (PK)          │  │    │ id (PK)          │──┐    │ id (PK)          │
│ user_id (FK)     │<─┼────│ customer_id (FK) │  │    │ order_id (FK)    │<─┐
│ label            │  │    │ chef_id (FK)     │──┘    │ menu_item_id(FK) │  │
│ line1            │  │    │ delivery_id (FK) │───┐   │ name             │  │
│ line2            │  │    │ order_number     │   │   │ price            │  │
│ city             │  │    │ status           │   │   │ quantity         │  │
│ state            │  │    │ delivery_addr_id │   │   │ subtotal         │  │
│ postal_code      │  │    │ subtotal         │   │   │ notes            │  │
│ country          │  │    │ delivery_fee     │   │   │ created_at       │  │
│ latitude         │  │    │ service_fee      │   │   └──────────────────┘  │
│ longitude        │  │    │ tax              │   │                         │
│ is_default       │  │    │ discount         │   │                         │
│ created_at       │  │    │ tip              │   │                         │
│ updated_at       │  │    │ total            │   │                         │
└──────────────────┘  │    │ promo_code       │   │                         │
                      │    │ instructions     │   │                         │
                      │    │ scheduled_for    │   │                         │
                      │    │ payment_id       │   │                         │
                      │    │ payment_status   │   │                         │
                      │    │ payment_method   │   │                         │
                      │    │ accepted_at      │   │                         │
                      │    │ prepared_at      │   │                         │
                      │    │ picked_up_at     │   │                         │
                      │    │ delivered_at     │   │                         │
                      │    │ cancelled_at     │   │                         │
                      │    │ cancel_reason    │   │                         │
                      │    │ created_at       │───┴─────────────────────────┘
                      │    │ updated_at       │
                      │    │ deleted_at       │
                      │    └────────┬─────────┘
                      │             │
┌──────────────────┐  │    ┌────────▼─────────┐       ┌──────────────────┐
│delivery_partners │  │    │   deliveries     │       │     reviews      │
├──────────────────┤  │    ├──────────────────┤       ├──────────────────┤
│ id (PK)          │──┼────│ partner_id (FK)  │       │ id (PK)          │
│ user_id (FK)     │<─┘    │ id (PK)          │       │ order_id (FK)    │
│ vehicle_type     │       │ order_id (FK)    │<──────│ customer_id (FK) │
│ vehicle_number   │       │ status           │       │ chef_id (FK)     │
│ license_number   │       │ assigned_at      │       │ delivery_id (FK) │
│ license_expiry   │       │ accepted_at      │       │ food_rating      │
│ is_online        │       │ picked_up_at     │       │ delivery_rating  │
│ is_available     │       │ delivered_at     │       │ comment          │
│ current_lat      │       │ distance_km      │       │ images[]         │
│ current_lng      │       │ delivery_fee     │       │ is_public        │
│ rating           │       │ tip              │       │ chef_response    │
│ total_deliveries │       │ delivery_proof   │       │ responded_at     │
│ status           │       │ notes            │       │ created_at       │
│ verified         │       │ created_at       │       │ updated_at       │
│ verified_at      │       │ updated_at       │       └──────────────────┘
│ created_at       │       └──────────────────┘
│ updated_at       │
│ deleted_at       │       ┌──────────────────┐       ┌──────────────────┐
└──────────────────┘       │    payments      │       │   promo_codes    │
                           ├──────────────────┤       ├──────────────────┤
                           │ id (PK)          │       │ id (PK)          │
                           │ order_id (FK)    │       │ code             │
                           │ amount           │       │ description      │
                           │ currency         │       │ discount_type    │
                           │ status           │       │ discount_value   │
                           │ method           │       │ min_order        │
                           │ provider         │       │ max_discount     │
                           │ provider_ref     │       │ usage_limit      │
                           │ metadata         │       │ used_count       │
                           │ refunded_amount  │       │ user_limit       │
                           │ refund_reason    │       │ valid_from       │
                           │ created_at       │       │ valid_until      │
                           │ updated_at       │       │ is_active        │
                           └──────────────────┘       │ created_at       │
                                                      │ updated_at       │
                                                      └──────────────────┘
```

---

## 3. Tables

### 3.1 Users Table

```sql
-- Users: Core user accounts for all roles
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    avatar VARCHAR(500),
    role VARCHAR(20) NOT NULL DEFAULT 'customer',
    email_verified BOOLEAN DEFAULT FALSE,
    phone_verified BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT uq_users_email UNIQUE (email),
    CONSTRAINT uq_users_phone UNIQUE (phone),
    CONSTRAINT chk_users_role CHECK (role IN ('customer', 'chef', 'delivery', 'admin', 'fleet_manager'))
);

-- Indexes
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_phone ON users(phone) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role ON users(role) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_created_at ON users(created_at);

-- Comments
COMMENT ON TABLE users IS 'Core user accounts for all platform roles';
COMMENT ON COLUMN users.role IS 'User role: customer, chef, delivery, admin, fleet_manager';
COMMENT ON COLUMN users.preferences IS 'JSON object storing user preferences like dietary, notifications';
```

### 3.2 Addresses Table

```sql
-- Addresses: User delivery/pickup addresses
CREATE TABLE addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label VARCHAR(50) DEFAULT 'Home',
    line1 VARCHAR(255) NOT NULL,
    line2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    country VARCHAR(100) NOT NULL DEFAULT 'United States',
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    is_default BOOLEAN DEFAULT FALSE,
    delivery_instructions TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_addresses_user_id ON addresses(user_id);
CREATE INDEX idx_addresses_location ON addresses USING GIST (
    ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)
);

-- Ensure only one default address per user
CREATE UNIQUE INDEX idx_addresses_user_default
ON addresses(user_id)
WHERE is_default = TRUE;

COMMENT ON TABLE addresses IS 'User addresses for delivery and chef pickup locations';
```

### 3.3 Social Logins Table

```sql
-- Social Logins: OAuth provider connections
CREATE TABLE social_logins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    access_token TEXT,
    refresh_token TEXT,
    token_expires_at TIMESTAMPTZ,
    profile_data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_social_provider_user UNIQUE (provider, provider_user_id),
    CONSTRAINT chk_social_provider CHECK (provider IN ('google', 'facebook', 'apple'))
);

CREATE INDEX idx_social_logins_user_id ON social_logins(user_id);
CREATE INDEX idx_social_logins_provider ON social_logins(provider, provider_user_id);

COMMENT ON TABLE social_logins IS 'OAuth social login provider connections';
```

### 3.4 Chef Profiles Table

```sql
-- Chef Profiles: Extended profile for chef users
CREATE TABLE chef_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    business_name VARCHAR(255) NOT NULL,
    description TEXT,
    cuisines TEXT[] DEFAULT '{}',
    specialties TEXT[] DEFAULT '{}',
    profile_image VARCHAR(500),
    banner_image VARCHAR(500),
    kitchen_images TEXT[] DEFAULT '{}',

    -- Status
    status VARCHAR(20) DEFAULT 'pending',

    -- Ratings & Stats
    rating DECIMAL(3, 2) DEFAULT 0,
    total_reviews INTEGER DEFAULT 0,
    total_orders INTEGER DEFAULT 0,

    -- Availability
    is_online BOOLEAN DEFAULT FALSE,
    accepting_orders BOOLEAN DEFAULT FALSE,

    -- Location
    address_id UUID REFERENCES addresses(id),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    service_radius DECIMAL(5, 2) DEFAULT 10,

    -- Operations
    operating_hours JSONB DEFAULT '{}',
    min_order_amount DECIMAL(10, 2) DEFAULT 0,
    delivery_fee DECIMAL(10, 2) DEFAULT 0,
    avg_prep_time INTEGER DEFAULT 30,

    -- Verification
    id_verified BOOLEAN DEFAULT FALSE,
    kitchen_verified BOOLEAN DEFAULT FALSE,
    food_license VARCHAR(100),
    license_verified BOOLEAN DEFAULT FALSE,
    health_certificate VARCHAR(500),
    verified_at TIMESTAMPTZ,
    verified_by UUID REFERENCES users(id),

    -- Financial
    bank_account_id VARCHAR(255),
    stripe_account_id VARCHAR(255),
    commission_rate DECIMAL(5, 4) DEFAULT 0.15,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT uq_chef_user UNIQUE (user_id),
    CONSTRAINT chk_chef_status CHECK (status IN ('pending', 'approved', 'rejected', 'suspended')),
    CONSTRAINT chk_chef_rating CHECK (rating >= 0 AND rating <= 5)
);

-- Indexes
CREATE INDEX idx_chef_profiles_user_id ON chef_profiles(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_chef_profiles_status ON chef_profiles(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_chef_profiles_rating ON chef_profiles(rating DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_chef_profiles_online ON chef_profiles(is_online, accepting_orders) WHERE deleted_at IS NULL;
CREATE INDEX idx_chef_profiles_cuisines ON chef_profiles USING GIN(cuisines);

-- Spatial index for location-based search
CREATE INDEX idx_chef_profiles_location ON chef_profiles USING GIST (
    ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)
) WHERE deleted_at IS NULL;

COMMENT ON TABLE chef_profiles IS 'Extended profiles for home chef users';
COMMENT ON COLUMN chef_profiles.operating_hours IS 'JSON: {"monday": {"open": "09:00", "close": "21:00"}, ...}';
COMMENT ON COLUMN chef_profiles.commission_rate IS 'Platform commission rate (e.g., 0.15 = 15%)';
```

### 3.5 Menu Categories Table

```sql
-- Menu Categories: Chef's menu organization
CREATE TABLE menu_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chef_id UUID NOT NULL REFERENCES chef_profiles(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    image_url VARCHAR(500),
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_menu_categories_chef ON menu_categories(chef_id, sort_order);

COMMENT ON TABLE menu_categories IS 'Menu item categories for chef organization';
```

### 3.6 Menu Items Table

```sql
-- Menu Items: Chef's food offerings
CREATE TABLE menu_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chef_id UUID NOT NULL REFERENCES chef_profiles(id) ON DELETE CASCADE,
    category_id UUID REFERENCES menu_categories(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    compare_price DECIMAL(10, 2),
    image_url VARCHAR(500),
    images TEXT[] DEFAULT '{}',

    -- Dietary Information
    dietary_tags TEXT[] DEFAULT '{}',
    allergens TEXT[] DEFAULT '{}',
    ingredients TEXT[] DEFAULT '{}',
    nutritional_info JSONB DEFAULT '{}',

    -- Portions
    portion_size VARCHAR(50),
    serves INTEGER DEFAULT 1,

    -- Operations
    prep_time INTEGER DEFAULT 30,
    is_available BOOLEAN DEFAULT TRUE,
    available_quantity INTEGER,
    max_daily_orders INTEGER,
    daily_orders_count INTEGER DEFAULT 0,

    -- Display
    sort_order INTEGER DEFAULT 0,
    is_featured BOOLEAN DEFAULT FALSE,
    is_special BOOLEAN DEFAULT FALSE,
    special_valid_until TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT chk_menu_price CHECK (price > 0),
    CONSTRAINT chk_menu_prep_time CHECK (prep_time > 0)
);

-- Indexes
CREATE INDEX idx_menu_items_chef ON menu_items(chef_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_menu_items_category ON menu_items(category_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_menu_items_available ON menu_items(chef_id, is_available) WHERE deleted_at IS NULL;
CREATE INDEX idx_menu_items_dietary ON menu_items USING GIN(dietary_tags);
CREATE INDEX idx_menu_items_featured ON menu_items(chef_id, is_featured) WHERE is_featured = TRUE AND deleted_at IS NULL;

-- Full-text search index
CREATE INDEX idx_menu_items_search ON menu_items
USING GIN(to_tsvector('english', name || ' ' || COALESCE(description, '')));

COMMENT ON TABLE menu_items IS 'Food items offered by chefs';
COMMENT ON COLUMN menu_items.dietary_tags IS 'Array: vegetarian, vegan, gluten-free, halal, kosher, etc.';
COMMENT ON COLUMN menu_items.allergens IS 'Array: nuts, dairy, gluten, shellfish, etc.';
COMMENT ON COLUMN menu_items.nutritional_info IS 'JSON: {"calories": 450, "protein": 25, "carbs": 40, "fat": 15}';
```

### 3.7 Orders Table

```sql
-- Orders: Customer food orders
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number VARCHAR(50) NOT NULL,
    customer_id UUID NOT NULL REFERENCES users(id),
    chef_id UUID NOT NULL REFERENCES chef_profiles(id),
    delivery_partner_id UUID REFERENCES delivery_partners(id),

    -- Status
    status VARCHAR(20) DEFAULT 'pending',

    -- Delivery Address (snapshot)
    delivery_address_id UUID REFERENCES addresses(id),
    delivery_address_line1 VARCHAR(255),
    delivery_address_line2 VARCHAR(255),
    delivery_city VARCHAR(100),
    delivery_postal_code VARCHAR(20),
    delivery_lat DECIMAL(10, 8),
    delivery_lng DECIMAL(11, 8),
    delivery_instructions TEXT,

    -- Pricing
    subtotal DECIMAL(10, 2) NOT NULL,
    delivery_fee DECIMAL(10, 2) DEFAULT 0,
    service_fee DECIMAL(10, 2) DEFAULT 0,
    tax DECIMAL(10, 2) DEFAULT 0,
    discount DECIMAL(10, 2) DEFAULT 0,
    tip DECIMAL(10, 2) DEFAULT 0,
    total DECIMAL(10, 2) NOT NULL,

    -- Promo
    promo_code VARCHAR(50),
    promo_discount DECIMAL(10, 2) DEFAULT 0,

    -- Instructions
    special_instructions TEXT,

    -- Scheduling
    order_type VARCHAR(20) DEFAULT 'delivery',
    scheduled_for TIMESTAMPTZ,
    estimated_ready_at TIMESTAMPTZ,
    estimated_delivery_at TIMESTAMPTZ,

    -- Timestamps
    accepted_at TIMESTAMPTZ,
    preparing_at TIMESTAMPTZ,
    ready_at TIMESTAMPTZ,
    picked_up_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancel_reason TEXT,
    cancelled_by UUID REFERENCES users(id),

    -- Payment
    payment_id VARCHAR(255),
    payment_status VARCHAR(20) DEFAULT 'pending',
    payment_method VARCHAR(50),

    -- Metadata
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT uq_order_number UNIQUE (order_number),
    CONSTRAINT chk_order_status CHECK (status IN (
        'pending', 'accepted', 'preparing', 'ready',
        'picked_up', 'delivering', 'delivered',
        'cancelled', 'refunded'
    )),
    CONSTRAINT chk_order_type CHECK (order_type IN ('delivery', 'pickup')),
    CONSTRAINT chk_order_payment_status CHECK (payment_status IN (
        'pending', 'processing', 'completed', 'failed', 'refunded', 'partial_refund'
    ))
);

-- Indexes
CREATE INDEX idx_orders_customer ON orders(customer_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_chef ON orders(chef_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_delivery_partner ON orders(delivery_partner_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_status ON orders(status, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_order_number ON orders(order_number);
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX idx_orders_scheduled ON orders(scheduled_for) WHERE scheduled_for IS NOT NULL;

-- Partial index for active orders
CREATE INDEX idx_orders_active ON orders(chef_id, status)
WHERE status IN ('pending', 'accepted', 'preparing', 'ready', 'picked_up', 'delivering')
AND deleted_at IS NULL;

COMMENT ON TABLE orders IS 'Customer food orders';
COMMENT ON COLUMN orders.order_number IS 'Human-readable order identifier (e.g., HC123456)';
```

### 3.8 Order Items Table

```sql
-- Order Items: Line items in an order
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    menu_item_id UUID REFERENCES menu_items(id),

    -- Denormalized item details (snapshot at time of order)
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 1,
    subtotal DECIMAL(10, 2) NOT NULL,

    -- Customization
    notes TEXT,
    customizations JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_order_item_quantity CHECK (quantity > 0),
    CONSTRAINT chk_order_item_price CHECK (price >= 0)
);

CREATE INDEX idx_order_items_order ON order_items(order_id);
CREATE INDEX idx_order_items_menu_item ON order_items(menu_item_id);

COMMENT ON TABLE order_items IS 'Individual items within an order';
COMMENT ON COLUMN order_items.customizations IS 'JSON: {"spice_level": "medium", "extra_sauce": true}';
```

### 3.9 Delivery Partners Table

```sql
-- Delivery Partners: Delivery personnel profiles
CREATE TABLE delivery_partners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Vehicle Info
    vehicle_type VARCHAR(50) NOT NULL,
    vehicle_make VARCHAR(100),
    vehicle_model VARCHAR(100),
    vehicle_color VARCHAR(50),
    vehicle_number VARCHAR(50),

    -- License
    license_number VARCHAR(100),
    license_state VARCHAR(50),
    license_expiry DATE,
    license_image VARCHAR(500),

    -- Insurance
    insurance_provider VARCHAR(100),
    insurance_number VARCHAR(100),
    insurance_expiry DATE,

    -- Status
    status VARCHAR(20) DEFAULT 'pending',
    is_online BOOLEAN DEFAULT FALSE,
    is_available BOOLEAN DEFAULT FALSE,
    current_latitude DECIMAL(10, 8),
    current_longitude DECIMAL(11, 8),
    last_location_update TIMESTAMPTZ,

    -- Stats
    rating DECIMAL(3, 2) DEFAULT 0,
    total_deliveries INTEGER DEFAULT 0,
    total_earnings DECIMAL(10, 2) DEFAULT 0,

    -- Verification
    id_verified BOOLEAN DEFAULT FALSE,
    license_verified BOOLEAN DEFAULT FALSE,
    vehicle_verified BOOLEAN DEFAULT FALSE,
    background_check_status VARCHAR(20) DEFAULT 'pending',
    verified_at TIMESTAMPTZ,

    -- Financial
    bank_account_id VARCHAR(255),

    -- Operational
    service_areas TEXT[] DEFAULT '{}',
    max_distance_km DECIMAL(5, 2) DEFAULT 20,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,

    CONSTRAINT uq_delivery_user UNIQUE (user_id),
    CONSTRAINT chk_delivery_status CHECK (status IN ('pending', 'approved', 'suspended', 'inactive')),
    CONSTRAINT chk_delivery_vehicle CHECK (vehicle_type IN ('bicycle', 'motorcycle', 'scooter', 'car')),
    CONSTRAINT chk_delivery_rating CHECK (rating >= 0 AND rating <= 5)
);

-- Indexes
CREATE INDEX idx_delivery_partners_user ON delivery_partners(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_delivery_partners_status ON delivery_partners(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_delivery_partners_available ON delivery_partners(is_online, is_available)
    WHERE deleted_at IS NULL AND status = 'approved';
CREATE INDEX idx_delivery_partners_location ON delivery_partners USING GIST (
    ST_SetSRID(ST_MakePoint(current_longitude, current_latitude), 4326)
) WHERE deleted_at IS NULL AND is_online = TRUE;

COMMENT ON TABLE delivery_partners IS 'Delivery personnel profiles and status';
```

### 3.10 Deliveries Table

```sql
-- Deliveries: Delivery assignment and tracking
CREATE TABLE deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id),
    partner_id UUID NOT NULL REFERENCES delivery_partners(id),

    -- Status
    status VARCHAR(20) DEFAULT 'assigned',

    -- Route
    pickup_lat DECIMAL(10, 8),
    pickup_lng DECIMAL(11, 8),
    pickup_address TEXT,
    dropoff_lat DECIMAL(10, 8),
    dropoff_lng DECIMAL(11, 8),
    dropoff_address TEXT,

    -- Distance & Time
    distance_km DECIMAL(6, 2),
    estimated_duration INTEGER,
    actual_duration INTEGER,

    -- Earnings
    delivery_fee DECIMAL(10, 2) DEFAULT 0,
    tip DECIMAL(10, 2) DEFAULT 0,
    bonus DECIMAL(10, 2) DEFAULT 0,
    total_earnings DECIMAL(10, 2) DEFAULT 0,

    -- Timestamps
    assigned_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    accepted_at TIMESTAMPTZ,
    arrived_at_pickup_at TIMESTAMPTZ,
    picked_up_at TIMESTAMPTZ,
    arrived_at_dropoff_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancel_reason TEXT,

    -- Proof of Delivery
    delivery_proof_type VARCHAR(20),
    delivery_proof_image VARCHAR(500),
    delivery_otp VARCHAR(6),
    signature_image VARCHAR(500),

    -- Notes
    partner_notes TEXT,
    customer_notes TEXT,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_delivery_order UNIQUE (order_id),
    CONSTRAINT chk_delivery_status CHECK (status IN (
        'assigned', 'accepted', 'at_pickup', 'picked_up',
        'at_dropoff', 'delivered', 'cancelled', 'failed'
    ))
);

-- Indexes
CREATE INDEX idx_deliveries_order ON deliveries(order_id);
CREATE INDEX idx_deliveries_partner ON deliveries(partner_id, created_at DESC);
CREATE INDEX idx_deliveries_status ON deliveries(status, created_at DESC);

COMMENT ON TABLE deliveries IS 'Delivery assignments and tracking details';
```

### 3.11 Payments Table

```sql
-- Payments: Payment transactions
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id),
    user_id UUID NOT NULL REFERENCES users(id),

    -- Amount
    amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',

    -- Status
    status VARCHAR(20) DEFAULT 'pending',

    -- Method
    payment_method VARCHAR(50) NOT NULL,
    card_last_four VARCHAR(4),
    card_brand VARCHAR(20),

    -- Provider
    provider VARCHAR(50) NOT NULL,
    provider_transaction_id VARCHAR(255),
    provider_payment_intent_id VARCHAR(255),
    provider_response JSONB DEFAULT '{}',

    -- Refund
    refunded_amount DECIMAL(10, 2) DEFAULT 0,
    refund_reason TEXT,
    refunded_at TIMESTAMPTZ,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_payment_status CHECK (status IN (
        'pending', 'processing', 'completed', 'failed',
        'refunded', 'partial_refund', 'disputed'
    )),
    CONSTRAINT chk_payment_amount CHECK (amount > 0)
);

-- Indexes
CREATE INDEX idx_payments_order ON payments(order_id);
CREATE INDEX idx_payments_user ON payments(user_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_provider_ref ON payments(provider_transaction_id);

COMMENT ON TABLE payments IS 'Payment transactions for orders';
```

### 3.12 Payouts Table

```sql
-- Payouts: Chef and delivery partner payouts
CREATE TABLE payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_id UUID NOT NULL REFERENCES users(id),
    recipient_type VARCHAR(20) NOT NULL,

    -- Amount
    amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',

    -- Breakdown
    gross_amount DECIMAL(10, 2) NOT NULL,
    platform_fee DECIMAL(10, 2) DEFAULT 0,
    other_deductions DECIMAL(10, 2) DEFAULT 0,
    net_amount DECIMAL(10, 2) NOT NULL,

    -- Status
    status VARCHAR(20) DEFAULT 'pending',

    -- Period
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,

    -- Provider
    provider VARCHAR(50),
    provider_payout_id VARCHAR(255),
    provider_response JSONB DEFAULT '{}',

    -- Timing
    scheduled_for TIMESTAMPTZ,
    initiated_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    failure_reason TEXT,

    -- Reference
    orders_count INTEGER DEFAULT 0,
    order_ids UUID[] DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_payout_recipient_type CHECK (recipient_type IN ('chef', 'delivery')),
    CONSTRAINT chk_payout_status CHECK (status IN (
        'pending', 'scheduled', 'processing', 'completed', 'failed'
    ))
);

CREATE INDEX idx_payouts_recipient ON payouts(recipient_id, created_at DESC);
CREATE INDEX idx_payouts_status ON payouts(status);
CREATE INDEX idx_payouts_period ON payouts(period_start, period_end);

COMMENT ON TABLE payouts IS 'Payout records for chefs and delivery partners';
```

### 3.13 Reviews Table

```sql
-- Reviews: Customer reviews for orders
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id),
    customer_id UUID NOT NULL REFERENCES users(id),
    chef_id UUID NOT NULL REFERENCES chef_profiles(id),
    delivery_partner_id UUID REFERENCES delivery_partners(id),

    -- Ratings
    food_rating INTEGER NOT NULL,
    packaging_rating INTEGER,
    value_rating INTEGER,
    delivery_rating INTEGER,
    overall_rating DECIMAL(3, 2) NOT NULL,

    -- Content
    title VARCHAR(255),
    comment TEXT,
    images TEXT[] DEFAULT '{}',

    -- Chef Response
    chef_response TEXT,
    responded_at TIMESTAMPTZ,

    -- Moderation
    is_public BOOLEAN DEFAULT TRUE,
    is_flagged BOOLEAN DEFAULT FALSE,
    flag_reason TEXT,
    moderated_at TIMESTAMPTZ,
    moderated_by UUID REFERENCES users(id),

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_review_order UNIQUE (order_id),
    CONSTRAINT chk_review_food_rating CHECK (food_rating >= 1 AND food_rating <= 5),
    CONSTRAINT chk_review_packaging CHECK (packaging_rating IS NULL OR (packaging_rating >= 1 AND packaging_rating <= 5)),
    CONSTRAINT chk_review_value CHECK (value_rating IS NULL OR (value_rating >= 1 AND value_rating <= 5)),
    CONSTRAINT chk_review_delivery CHECK (delivery_rating IS NULL OR (delivery_rating >= 1 AND delivery_rating <= 5)),
    CONSTRAINT chk_review_overall CHECK (overall_rating >= 1 AND overall_rating <= 5)
);

-- Indexes
CREATE INDEX idx_reviews_chef ON reviews(chef_id, created_at DESC) WHERE is_public = TRUE;
CREATE INDEX idx_reviews_customer ON reviews(customer_id, created_at DESC);
CREATE INDEX idx_reviews_delivery_partner ON reviews(delivery_partner_id) WHERE delivery_partner_id IS NOT NULL;
CREATE INDEX idx_reviews_rating ON reviews(chef_id, overall_rating DESC);

COMMENT ON TABLE reviews IS 'Customer reviews and ratings for orders';
```

### 3.14 Promo Codes Table

```sql
-- Promo Codes: Discount codes and promotions
CREATE TABLE promo_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL,
    description TEXT,

    -- Discount
    discount_type VARCHAR(20) NOT NULL,
    discount_value DECIMAL(10, 2) NOT NULL,
    max_discount DECIMAL(10, 2),

    -- Conditions
    min_order_amount DECIMAL(10, 2) DEFAULT 0,
    applicable_to VARCHAR(20) DEFAULT 'all',
    applicable_items UUID[] DEFAULT '{}',
    applicable_chefs UUID[] DEFAULT '{}',

    -- Usage Limits
    usage_limit INTEGER,
    used_count INTEGER DEFAULT 0,
    per_user_limit INTEGER DEFAULT 1,

    -- Validity
    valid_from TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,

    -- Targeting
    new_users_only BOOLEAN DEFAULT FALSE,
    first_order_only BOOLEAN DEFAULT FALSE,

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_promo_code UNIQUE (code),
    CONSTRAINT chk_promo_discount_type CHECK (discount_type IN ('percentage', 'fixed', 'free_delivery')),
    CONSTRAINT chk_promo_applicable CHECK (applicable_to IN ('all', 'items', 'chefs', 'categories'))
);

-- Indexes
CREATE INDEX idx_promo_codes_code ON promo_codes(code) WHERE is_active = TRUE;
CREATE INDEX idx_promo_codes_validity ON promo_codes(valid_from, valid_until) WHERE is_active = TRUE;

COMMENT ON TABLE promo_codes IS 'Promotional discount codes';
```

### 3.15 Promo Code Usage Table

```sql
-- Promo Code Usage: Track promo code usage per user
CREATE TABLE promo_code_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    promo_code_id UUID NOT NULL REFERENCES promo_codes(id),
    user_id UUID NOT NULL REFERENCES users(id),
    order_id UUID NOT NULL REFERENCES orders(id),
    discount_applied DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_promo_usage_order UNIQUE (order_id)
);

CREATE INDEX idx_promo_usage_code ON promo_code_usage(promo_code_id);
CREATE INDEX idx_promo_usage_user ON promo_code_usage(user_id, promo_code_id);

COMMENT ON TABLE promo_code_usage IS 'Tracks usage of promo codes by users';
```

### 3.16 Notifications Table

```sql
-- Notifications: User notifications
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),

    -- Content
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    image_url VARCHAR(500),

    -- Action
    action_type VARCHAR(50),
    action_data JSONB DEFAULT '{}',

    -- Status
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMPTZ,

    -- Delivery
    channels TEXT[] DEFAULT '{push}',
    push_sent BOOLEAN DEFAULT FALSE,
    push_sent_at TIMESTAMPTZ,
    email_sent BOOLEAN DEFAULT FALSE,
    email_sent_at TIMESTAMPTZ,
    sms_sent BOOLEAN DEFAULT FALSE,
    sms_sent_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ,

    CONSTRAINT chk_notification_type CHECK (type IN (
        'order_placed', 'order_accepted', 'order_preparing', 'order_ready',
        'order_picked_up', 'order_delivered', 'order_cancelled',
        'new_order', 'delivery_assigned', 'review_received',
        'payout_completed', 'promo', 'system', 'account'
    ))
);

CREATE INDEX idx_notifications_user ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications(user_id, is_read) WHERE is_read = FALSE;

COMMENT ON TABLE notifications IS 'User notifications across all channels';
```

### 3.17 Support Tickets Table

```sql
-- Support Tickets: Customer support requests
CREATE TABLE support_tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_number VARCHAR(50) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    order_id UUID REFERENCES orders(id),

    -- Ticket Details
    category VARCHAR(50) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,

    -- Status
    status VARCHAR(20) DEFAULT 'open',
    priority VARCHAR(20) DEFAULT 'normal',

    -- Assignment
    assigned_to UUID REFERENCES users(id),
    assigned_at TIMESTAMPTZ,

    -- Resolution
    resolution TEXT,
    resolved_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES users(id),

    -- Metadata
    attachments TEXT[] DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_ticket_number UNIQUE (ticket_number),
    CONSTRAINT chk_ticket_status CHECK (status IN ('open', 'in_progress', 'waiting', 'resolved', 'closed')),
    CONSTRAINT chk_ticket_priority CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    CONSTRAINT chk_ticket_category CHECK (category IN (
        'order_issue', 'payment_issue', 'delivery_issue', 'quality_issue',
        'account_issue', 'refund_request', 'feedback', 'other'
    ))
);

CREATE INDEX idx_support_tickets_user ON support_tickets(user_id, created_at DESC);
CREATE INDEX idx_support_tickets_status ON support_tickets(status, priority);
CREATE INDEX idx_support_tickets_assigned ON support_tickets(assigned_to) WHERE assigned_to IS NOT NULL;

COMMENT ON TABLE support_tickets IS 'Customer support tickets';
```

### 3.18 Audit Logs Table

```sql
-- Audit Logs: System audit trail
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),

    -- Action
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID,

    -- Details
    old_values JSONB,
    new_values JSONB,

    -- Context
    ip_address INET,
    user_agent TEXT,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Partition by month for performance
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id, created_at DESC);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action, created_at DESC);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);

COMMENT ON TABLE audit_logs IS 'System audit trail for compliance and debugging';
```

---

## 4. Indexes

### 4.1 Index Summary

```sql
-- Performance-critical indexes summary

-- Users
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_phone ON users(phone) WHERE deleted_at IS NULL;

-- Chef Profiles - Location search
CREATE INDEX idx_chef_profiles_location ON chef_profiles USING GIST (
    ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)
) WHERE deleted_at IS NULL;

-- Menu Items - Full-text search
CREATE INDEX idx_menu_items_search ON menu_items
USING GIN(to_tsvector('english', name || ' ' || COALESCE(description, '')));

-- Orders - Active orders for chef dashboard
CREATE INDEX idx_orders_active ON orders(chef_id, status)
WHERE status IN ('pending', 'accepted', 'preparing', 'ready', 'picked_up', 'delivering')
AND deleted_at IS NULL;

-- Delivery Partners - Available partners for assignment
CREATE INDEX idx_delivery_partners_available ON delivery_partners USING GIST (
    ST_SetSRID(ST_MakePoint(current_longitude, current_latitude), 4326)
) WHERE deleted_at IS NULL AND is_online = TRUE AND is_available = TRUE;
```

---

## 5. Migrations

### 5.1 Initial Migration

```sql
-- migrations/001_initial_schema.up.sql

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Create all tables (see sections above)
-- ...

-- Seed initial data
INSERT INTO users (id, email, password_hash, first_name, last_name, role, email_verified)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@homechef.com',
    '$2a$10$...', -- bcrypt hash of admin password
    'System',
    'Admin',
    'admin',
    TRUE
);
```

### 5.2 Rollback Migration

```sql
-- migrations/001_initial_schema.down.sql

DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS support_ticket_messages CASCADE;
DROP TABLE IF EXISTS support_tickets CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS promo_code_usage CASCADE;
DROP TABLE IF EXISTS promo_codes CASCADE;
DROP TABLE IF EXISTS reviews CASCADE;
DROP TABLE IF EXISTS payouts CASCADE;
DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS deliveries CASCADE;
DROP TABLE IF EXISTS delivery_partners CASCADE;
DROP TABLE IF EXISTS order_items CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS menu_items CASCADE;
DROP TABLE IF EXISTS menu_categories CASCADE;
DROP TABLE IF EXISTS chef_profiles CASCADE;
DROP TABLE IF EXISTS social_logins CASCADE;
DROP TABLE IF EXISTS addresses CASCADE;
DROP TABLE IF EXISTS users CASCADE;

DROP EXTENSION IF EXISTS "pg_trgm";
DROP EXTENSION IF EXISTS "postgis";
DROP EXTENSION IF EXISTS "uuid-ossp";
```

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | Dec 2024 | HomeChef Team | Initial database schema |

---

*This document is confidential and intended for internal development use only.*
