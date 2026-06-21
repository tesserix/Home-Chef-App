# Flexible Fulfillment — Phase 1 (Pickup) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a customer "pickup" fulfillment mode (collect from the chef, no delivery fee), capture real chef coordinates, and make the shipped address-masking conditional so pickup reveals the chef's exact address while 3PL stays masked.

**Architecture:** Add `Order.FulfillmentType` (`delivery`|`chef_delivery`|`pickup`, default `delivery`) and `ChefProfile.OffersPickup`. Chef coordinates are resolved server-side by forward-geocoding the chef's stored address via the existing Photon endpoint (on profile save + a one-time backfill). Order responses become mode-aware: a `pickup` order returns the chef's exact coords + address to the customer and skips 3PL dispatch + delivery fee; `delivery` orders keep the masking shipped in PR #320. Customer checkout gains a Delivery/Pickup selector (shown only when the chef offers pickup); the vendor profile gains a pickup toggle.

**Tech Stack:** Go 1.26 (Gin, GORM AutoMigrate), Expo/React Native + TypeScript, TanStack Query. Spec: `docs/superpowers/specs/2026-06-21-flexible-fulfillment-design.md`.

## Global Constraints

- Branch off `main`; open a PR (owner merges). Do NOT push to `main`.
- Schema changes are via GORM **AutoMigrate** (struct tags add columns) — no hand-written SQL migration files needed. Order & ChefProfile are already in the `DB.AutoMigrate(...)` list (`apps/api/database/database.go`).
- Every mobile app `package.json` stays on `@babel/core ^7.29.7` + `tailwindcss ^3.x` (dependabot breaks Metro otherwise).
- Never surface raw error codes to users — use the friendly mappers already in the apps.
- Currency is INR; money is `float64` rupees.
- Deploy is Kargo + the GAR-mirror dance; merge to `main` triggers the API CI build.
- Keep PR #320 masking for `delivery`/3PL — only `pickup` reveals the chef address. (`chef_delivery` is Phase 2; wire the enum value but no behavior beyond default.)

---

### Task 1: Fulfillment + pickup-offering model fields & responses

**Files:**
- Modify: `apps/api/models/order.go` (Order struct ~line 56–145; OrderResponse ~212–248; ToResponse ~323)
- Modify: `apps/api/models/chef.go` (ChefProfile ~line 20–60; ChefProfileResponse ~189–227; ToResponse ~243)
- Modify: `apps/api/handlers/chefs.go` (UpdateChefProfileRequest ~617–625; apply-fields block ~682–695)
- Test: `apps/api/models/order_test.go`, `apps/api/models/chef_fulfillment_test.go`

**Interfaces:**
- Produces: `models.FulfillmentType` (`FulfillmentDelivery`/`FulfillmentChefDelivery`/`FulfillmentPickup`); `Order.FulfillmentType`; `OrderResponse.FulfillmentType`; `ChefProfile.OffersPickup`; `ChefProfileResponse.OffersPickup`; `UpdateChefProfileRequest.OffersPickup *bool`.

- [ ] **Step 1: Write the failing test** — append to `apps/api/models/order_test.go`:

```go
func TestToResponse_FulfillmentTypeDefaultsToDelivery(t *testing.T) {
	o := Order{} // unset
	if got := o.ToResponse().FulfillmentType; got != FulfillmentDelivery {
		t.Fatalf("empty FulfillmentType should normalize to delivery, got %q", got)
	}
	o2 := Order{FulfillmentType: FulfillmentPickup}
	if got := o2.ToResponse().FulfillmentType; got != FulfillmentPickup {
		t.Fatalf("want pickup, got %q", got)
	}
}
```

And create `apps/api/models/chef_fulfillment_test.go`:

```go
package models

import "testing"

func TestChefToResponse_OffersPickup(t *testing.T) {
	c := ChefProfile{OffersPickup: true}
	if !c.ToResponse().OffersPickup {
		t.Fatal("ChefProfileResponse should carry OffersPickup")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/api && go test ./models/ -run 'Fulfillment|OffersPickup' -v`
Expected: FAIL — undefined `FulfillmentDelivery` / `FulfillmentPickup` / field `FulfillmentType` / `OffersPickup`.

- [ ] **Step 3: Add the model fields, constants, and response wiring**

In `apps/api/models/order.go`, after the `PaymentStatus` const block (~line 54), add:

```go
// FulfillmentType is how an order reaches the customer.
type FulfillmentType string

const (
	FulfillmentDelivery     FulfillmentType = "delivery"      // 3PL rider (default)
	FulfillmentChefDelivery FulfillmentType = "chef_delivery" // chef delivers (Phase 2)
	FulfillmentPickup       FulfillmentType = "pickup"        // customer collects
)
```

In the `Order` struct, next to `PaymentMethod` (~line 64), add:

```go
	FulfillmentType FulfillmentType `gorm:"type:varchar(16);default:'delivery'" json:"fulfillmentType"`
```

In `OrderResponse` (after `Status`), add:

```go
	FulfillmentType FulfillmentType `json:"fulfillmentType"`
```

In `ToResponse()`, set it (normalizing the empty default), inside the returned struct literal:

```go
		Status:          o.Status,
		FulfillmentType: func() FulfillmentType {
			if o.FulfillmentType == "" {
				return FulfillmentDelivery
			}
			return o.FulfillmentType
		}(),
```

In `apps/api/models/chef.go`, add to `ChefProfile` (near `ServiceRadius`, ~line 28):

```go
	OffersPickup bool `gorm:"default:false" json:"offersPickup"`
```

Add to `ChefProfileResponse` (near `ServiceRadius`):

```go
	OffersPickup bool `json:"offersPickup"`
```

And in the chef `ToResponse()` struct literal, add:

```go
		OffersPickup: c.OffersPickup,
```

In `apps/api/handlers/chefs.go`, add to `UpdateChefProfileRequest` (after `AcceptingOrders`):

```go
	OffersPickup *bool `json:"offersPickup"`
```

And in the apply-fields block of `UpdateChefProfile` (after the `AcceptingOrders` block, ~line 681):

```go
	if req.OffersPickup != nil {
		chef.OffersPickup = *req.OffersPickup
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test ./models/ -run 'Fulfillment|OffersPickup' -v && go build ./...`
Expected: PASS, build OK.

- [ ] **Step 5: Commit**

```bash
git add apps/api/models/order.go apps/api/models/chef.go apps/api/handlers/chefs.go apps/api/models/order_test.go apps/api/models/chef_fulfillment_test.go
git commit -m "feat(fulfillment): add Order.FulfillmentType + ChefProfile.OffersPickup model fields"
```

---

### Task 2: Server-side forward geocoder (Photon)

**Files:**
- Create: `apps/api/services/geocode.go`
- Test: `apps/api/services/geocode_test.go`

**Interfaces:**
- Produces: `services.GeocodeAddress(query string) (lat, lng float64, ok bool)` — best-effort forward geocode; `ok=false` on empty query, network error, or no result.

- [ ] **Step 1: Write the failing test** — `apps/api/services/geocode_test.go`:

```go
package services

import "testing"

func TestParsePhotonGeocode_TopResult(t *testing.T) {
	// Photon geometry is [lon, lat].
	body := []byte(`{"features":[{"geometry":{"coordinates":[72.7967,19.1499]}},{"geometry":{"coordinates":[1,2]}}]}`)
	lat, lng, ok := parsePhotonGeocode(body)
	if !ok || lat != 19.1499 || lng != 72.7967 {
		t.Fatalf("want 19.1499,72.7967 ok; got %v,%v ok=%v", lat, lng, ok)
	}
}

func TestParsePhotonGeocode_Empty(t *testing.T) {
	if _, _, ok := parsePhotonGeocode([]byte(`{"features":[]}`)); ok {
		t.Fatal("empty features must return ok=false")
	}
}

func TestGeocodeAddress_BlankQuery(t *testing.T) {
	if _, _, ok := GeocodeAddress("   "); ok {
		t.Fatal("blank query must return ok=false without a network call")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/api && go test ./services/ -run Geocode -v`
Expected: FAIL — undefined `parsePhotonGeocode` / `GeocodeAddress`.

- [ ] **Step 3: Implement** — `apps/api/services/geocode.go`:

```go
package services

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	geocodePhotonAPI  = "https://photon.komoot.io/api/"
	geocodeUserAgent  = "homechef-api (+https://fe3dr.com)"
)

var geocodeClient = &http.Client{Timeout: 4 * time.Second}

// parsePhotonGeocode pulls lat/lng from the top Photon feature.
// Photon geometry coordinates are [lon, lat].
func parsePhotonGeocode(body []byte) (lat, lng float64, ok bool) {
	var resp struct {
		Features []struct {
			Geometry struct {
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, false
	}
	for _, f := range resp.Features {
		if len(f.Geometry.Coordinates) == 2 {
			return f.Geometry.Coordinates[1], f.Geometry.Coordinates[0], true
		}
	}
	return 0, 0, false
}

// GeocodeAddress forward-geocodes a free-text address via Photon. Best-effort:
// returns ok=false on blank input, transport error, or no match. Never panics —
// callers treat a miss as "no coordinates yet".
func GeocodeAddress(query string) (lat, lng float64, ok bool) {
	q := strings.TrimSpace(query)
	if q == "" {
		return 0, 0, false
	}
	pu, err := url.Parse(geocodePhotonAPI)
	if err != nil {
		return 0, 0, false
	}
	params := url.Values{}
	params.Set("q", q)
	params.Set("limit", "1")
	pu.RawQuery = params.Encode()

	req, err := http.NewRequest(http.MethodGet, pu.String(), nil)
	if err != nil {
		return 0, 0, false
	}
	req.Header.Set("User-Agent", geocodeUserAgent)

	resp, err := geocodeClient.Do(req)
	if err != nil {
		return 0, 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, false
	}
	var buf [1 << 16]byte
	n, _ := resp.Body.Read(buf[:])
	return parsePhotonGeocode(buf[:n])
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test ./services/ -run Geocode -v && go vet ./services/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/services/geocode.go apps/api/services/geocode_test.go
git commit -m "feat(geocode): server-side Photon forward geocoder for chef coordinates"
```

---

### Task 3: Resolve chef coordinates on profile save + one-time backfill

**Files:**
- Modify: `apps/api/handlers/chefs.go` (`UpdateChefProfile`, after `database.DB.Save(&chef)` ~line 697)
- Create: `apps/api/services/chef_geocode_backfill.go`
- Modify: `apps/api/main.go` (after DB init, before router run — call the backfill in a goroutine)

**Interfaces:**
- Consumes: `services.GeocodeAddress` (Task 2).
- Produces: `services.BackfillChefCoordinates(db *gorm.DB)`.

- [ ] **Step 1: Geocode on profile save** — in `UpdateChefProfile`, immediately after the successful `database.DB.Save(&chef)` block, add:

```go
	// Resolve kitchen coordinates from the address (best-effort) so pickup +
	// self-delivery distance work. Only when we have a street + city and the
	// address actually changed enough to matter; failures are non-fatal.
	if chef.AddressLine1 != "" && chef.City != "" {
		full := joinAddress(chef.AddressLine1, chef.AddressLine2, chef.City, chef.State, chef.PostalCode)
		if lat, lng, ok := services.GeocodeAddress(full); ok {
			chef.Latitude, chef.Longitude = lat, lng
			database.DB.Model(&chef).Updates(map[string]any{"latitude": lat, "longitude": lng})
		}
	}
```

(`joinAddress` already exists in `services` — if not importable here, inline a comma-join of the non-empty parts.)

- [ ] **Step 2: Implement the backfill** — `apps/api/services/chef_geocode_backfill.go`:

```go
package services

import (
	"log"
	"strings"

	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// BackfillChefCoordinates fills lat/lng for chefs that have an address but no
// coordinates yet (legacy rows created before geocoding existed). Idempotent and
// best-effort — safe to run on every boot.
func BackfillChefCoordinates(db *gorm.DB) {
	var chefs []models.ChefProfile
	if err := db.Where("(latitude = 0 OR latitude IS NULL) AND address_line1 <> ''").
		Find(&chefs).Error; err != nil {
		log.Printf("chef-coord backfill: query failed: %v", err)
		return
	}
	for _, ch := range chefs {
		full := strings.TrimSpace(strings.Join([]string{
			ch.AddressLine1, ch.AddressLine2, ch.City, ch.State, ch.PostalCode,
		}, ", "))
		lat, lng, ok := GeocodeAddress(full)
		if !ok {
			continue
		}
		db.Model(&models.ChefProfile{}).Where("id = ?", ch.ID).
			Updates(map[string]any{"latitude": lat, "longitude": lng})
	}
	log.Printf("chef-coord backfill: processed %d chef(s)", len(chefs))
}
```

- [ ] **Step 3: Call the backfill at startup** — in `apps/api/main.go`, after the DB is initialized and migrated (before `router.Run`), add:

```go
	// One-time-ish: geocode chefs missing coordinates. Background so boot isn't
	// blocked by Photon latency.
	go services.BackfillChefCoordinates(database.DB)
```

- [ ] **Step 4: Verify build + vet**

Run: `cd apps/api && go build ./... && go vet ./services/ ./handlers/`
Expected: clean. (Geocoding hits the network; no unit test here — `GeocodeAddress` is covered in Task 2 and the call sites are best-effort.)

- [ ] **Step 5: Commit**

```bash
git add apps/api/handlers/chefs.go apps/api/services/chef_geocode_backfill.go apps/api/main.go
git commit -m "feat(chef): geocode kitchen address to coordinates on save + startup backfill"
```

---

### Task 4: CreateOrder — accept fulfillment, validate pickup, zero fee + skip 3PL quote

**Files:**
- Modify: `apps/api/handlers/orders.go` (`CreateOrderRequest` ~75; chef load ~188; fee block ~314 & ~380; order struct ~445)
- Test: `apps/api/handlers/orders_fulfillment_test.go`

**Interfaces:**
- Consumes: `models.FulfillmentPickup`, `models.FulfillmentDelivery`, `ChefProfile.OffersPickup` (Task 1).
- Produces: `resolveFulfillment(req CreateOrderRequest, chef models.ChefProfile) (models.FulfillmentType, error)` (package `handlers`).

- [ ] **Step 1: Write the failing test** — `apps/api/handlers/orders_fulfillment_test.go`:

```go
package handlers

import (
	"testing"

	"github.com/homechef/api/models"
)

func TestResolveFulfillment(t *testing.T) {
	chefPickup := models.ChefProfile{OffersPickup: true}
	chefNoPickup := models.ChefProfile{OffersPickup: false}

	// default → delivery
	if ft, err := resolveFulfillment(CreateOrderRequest{}, chefNoPickup); err != nil || ft != models.FulfillmentDelivery {
		t.Fatalf("default should be delivery, got %q err=%v", ft, err)
	}
	// pickup allowed when chef offers it
	if ft, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "pickup"}, chefPickup); err != nil || ft != models.FulfillmentPickup {
		t.Fatalf("pickup should be allowed, got %q err=%v", ft, err)
	}
	// pickup rejected when chef does not offer it
	if _, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "pickup"}, chefNoPickup); err == nil {
		t.Fatal("pickup must be rejected when chef does not offer pickup")
	}
	// unknown value rejected
	if _, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "teleport"}, chefPickup); err == nil {
		t.Fatal("unknown fulfillment must be rejected")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd apps/api && go test ./handlers/ -run TestResolveFulfillment -v`
Expected: FAIL — undefined `resolveFulfillment`, undefined `CreateOrderRequest.FulfillmentType`.

- [ ] **Step 3: Implement**

Add `FulfillmentType` to `CreateOrderRequest` (after `DeliveryInstructions`):

```go
	FulfillmentType string `json:"fulfillmentType"`
```

Add the resolver (top-level func in `orders.go`):

```go
// resolveFulfillment validates the requested fulfillment mode against what the
// chef offers, defaulting to 3PL delivery. chef_delivery is reserved for Phase 2.
func resolveFulfillment(req CreateOrderRequest, chef models.ChefProfile) (models.FulfillmentType, error) {
	switch models.FulfillmentType(req.FulfillmentType) {
	case "", models.FulfillmentDelivery:
		return models.FulfillmentDelivery, nil
	case models.FulfillmentPickup:
		if !chef.OffersPickup {
			return "", fmt.Errorf("this kitchen does not offer pickup")
		}
		return models.FulfillmentPickup, nil
	default:
		return "", fmt.Errorf("unsupported fulfillment option")
	}
}
```

In `CreateOrder`, after the chef is loaded and validated (~line 209, after the FSSAI/capacity checks), resolve the mode:

```go
	fulfillment, err := resolveFulfillment(req, chef)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
```

Guard the delivery-fee quote so pickup pays nothing — wrap the existing block at ~line 380:

```go
	if fulfillment == models.FulfillmentPickup {
		deliveryFee = 0
	} else if fee, ok := services.QuoteCheckoutDeliveryFee(chef, deliveryAddr.City, deliveryCountry, deliveryAddr.Latitude, deliveryAddr.Longitude); ok {
		deliveryFee = fee
	}
```

Set the field on the `order := models.Order{...}` literal (~line 445):

```go
		FulfillmentType:           fulfillment,
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd apps/api && go test ./handlers/ -run TestResolveFulfillment -v && go build ./...`
Expected: PASS, build OK.

- [ ] **Step 5: Commit**

```bash
git add apps/api/handlers/orders.go apps/api/handlers/orders_fulfillment_test.go
git commit -m "feat(orders): accept + validate fulfillment mode; pickup is free and skips 3PL quote"
```

---

### Task 5: Dispatch 3PL only for delivery orders

**Files:**
- Modify: `apps/api/handlers/chefs.go` (`UpdateOrderStatus`, the `OrderStatusReady` dispatch block ~865)

**Interfaces:**
- Consumes: `Order.FulfillmentType`, `models.FulfillmentDelivery`.

- [ ] **Step 1: Guard the dispatch** — change the ready-dispatch block to:

```go
	// Auto-dispatch a 3PL delivery once food is ready — ONLY for 3PL orders.
	// Pickup orders are collected by the customer; chef_delivery (Phase 2) is
	// carried by the chef. Neither dispatches a provider.
	if order.Status == models.OrderStatusReady && order.FulfillmentType == models.FulfillmentDelivery {
		services.EnqueueDeliveryDispatch(order.ID)
	}
```

- [ ] **Step 2: Build + existing handler tests**

Run: `cd apps/api && go build ./... && go test ./handlers/ -count=1`
Expected: build OK, tests pass.

- [ ] **Step 3: Commit**

```bash
git add apps/api/handlers/chefs.go
git commit -m "fix(orders): only dispatch a 3PL delivery for delivery-mode orders"
```

---

### Task 6: Reveal exact chef address + coords for pickup orders (conditional masking)

**Files:**
- Modify: `apps/api/handlers/orders.go` (`TrackOrder` ~743)
- Test: `apps/api/handlers/orders_fulfillment_test.go` (extend)

**Interfaces:**
- Consumes: `services.FuzzCoordinate`, `Order.FulfillmentType`, `models.FulfillmentPickup`.
- Produces: `chefTrackCoords(order models.Order) (lat, lng float64, exact bool)` (package `handlers`).

- [ ] **Step 1: Write the failing test** — append:

```go
func TestChefTrackCoords_PickupIsExact(t *testing.T) {
	o := models.Order{FulfillmentType: models.FulfillmentPickup}
	o.Chef.ID = uuid.New()
	o.Chef.Latitude, o.Chef.Longitude = 19.1499, 72.7967
	lat, lng, exact := chefTrackCoords(o)
	if !exact || lat != 19.1499 || lng != 72.7967 {
		t.Fatalf("pickup must return EXACT chef coords; got %v,%v exact=%v", lat, lng, exact)
	}
}

func TestChefTrackCoords_DeliveryIsFuzzed(t *testing.T) {
	o := models.Order{FulfillmentType: models.FulfillmentDelivery}
	o.Chef.ID = uuid.New()
	o.Chef.Latitude, o.Chef.Longitude = 19.1499, 72.7967
	lat, lng, exact := chefTrackCoords(o)
	if exact {
		t.Fatal("delivery must be fuzzed, not exact")
	}
	if lat == 19.1499 && lng == 72.7967 {
		t.Fatal("delivery coords must differ from the true location")
	}
}
```

(Ensure `"github.com/google/uuid"` is imported in the test file.)

- [ ] **Step 2: Run to verify it fails**

Run: `cd apps/api && go test ./handlers/ -run TestChefTrackCoords -v`
Expected: FAIL — undefined `chefTrackCoords`.

- [ ] **Step 3: Implement** — add to `orders.go`:

```go
// chefTrackCoords returns the chef coordinates to show the customer for an order.
// Pickup reveals the EXACT kitchen (the customer is collecting); every other mode
// returns an approximate (fuzzed) point so the address stays private.
func chefTrackCoords(order models.Order) (lat, lng float64, exact bool) {
	if order.FulfillmentType == models.FulfillmentPickup {
		return order.Chef.Latitude, order.Chef.Longitude, true
	}
	flat, flng := services.FuzzCoordinate(order.Chef.Latitude, order.Chef.Longitude, order.Chef.ID.String())
	return flat, flng, false
}
```

In `TrackOrder`, replace the `chefAreaLat, chefAreaLng := services.FuzzCoordinate(...)` line with:

```go
	chefLat, chefLng, chefExact := chefTrackCoords(order)
```

Update the `"chef"` map in the response to use these and include the address only when exact (pickup):

```go
		"chef": func() gin.H {
			m := gin.H{
				"name":      order.Chef.BusinessName,
				"latitude":  chefLat,
				"longitude": chefLng,
			}
			if chefExact {
				// Pickup: the customer needs the real address to collect.
				m["address"] = joinAddress(order.Chef.AddressLine1, order.Chef.AddressLine2, order.Chef.City, order.Chef.State, order.Chef.PostalCode)
			}
			return m
		}(),
```

(`joinAddress` lives in `services`; call `services.joinAddress`? It is unexported. Inline a local join in `orders.go`: build a `[]string` of the non-empty parts and `strings.Join(parts, ", ")`. `strings` is already imported in `orders.go`.)

- [ ] **Step 4: Run to verify it passes**

Run: `cd apps/api && go test ./handlers/ -run 'TestChefTrackCoords|TestResolveFulfillment' -v && go build ./...`
Expected: PASS, build OK.

- [ ] **Step 5: Commit**

```bash
git add apps/api/handlers/orders.go apps/api/handlers/orders_fulfillment_test.go
git commit -m "feat(tracking): pickup orders return the chef's exact address + coords; delivery stays fuzzed"
```

---

### Task 7: Customer checkout — Delivery/Pickup selector

**Files:**
- Modify: `apps/mobile-customer/hooks/useOrderCheckout.ts` (`CreateOrderPayload`)
- Modify: `apps/mobile-customer/app/checkout.tsx` (fulfillment state + selector + payload + fee)
- Modify: `apps/mobile-customer/types/customer.ts` (add `offersPickup?: boolean` to `Chef` if absent)

**Interfaces:**
- Consumes: `Chef.offersPickup` (from `useChef`/chef detail), backend `fulfillmentType` field (Task 1/4).

- [ ] **Step 1: Add `fulfillmentType` to the order payload** — in `useOrderCheckout.ts`, add to `CreateOrderPayload`:

```ts
  // 'delivery' (default) | 'pickup'. Omit → server defaults to delivery.
  fulfillmentType?: 'delivery' | 'pickup';
```

- [ ] **Step 2: Add `offersPickup` to the Chef type** — in `types/customer.ts`, in the `Chef` interface add (if not present):

```ts
  offersPickup?: boolean;
```

And ensure the chef mapper (`hooks/useChefs.ts`) copies it: `offersPickup: c.offersPickup,`.

- [ ] **Step 3: Add the selector to checkout** — in `checkout.tsx`:

Add state near the other `useState`s:

```tsx
  const [fulfillment, setFulfillment] = useState<'delivery' | 'pickup'>('delivery');
```

Source the chef's pickup offering. The cart already has `chefId`; fetch the chef detail (reuse the existing `useChef` hook; import it). Near the top of the component:

```tsx
  const { data: chef } = useChef(cartStore.chefId ?? '');
  const offersPickup = !!chef?.offersPickup;
```

Render a segmented control above the address section (only when pickup is available):

```tsx
  {offersPickup ? (
    <View style={styles.fulfillmentRow}>
      {(['delivery', 'pickup'] as const).map((mode) => (
        <Pressable
          key={mode}
          onPress={() => setFulfillment(mode)}
          accessibilityRole="button"
          accessibilityState={{ selected: fulfillment === mode }}
          style={[styles.fulfillmentChip, fulfillment === mode && styles.fulfillmentChipActive]}
        >
          <Text style={[styles.fulfillmentChipText, fulfillment === mode && styles.fulfillmentChipTextActive]}>
            {mode === 'delivery' ? 'Delivery' : 'Pickup'}
          </Text>
        </Pressable>
      ))}
    </View>
  ) : null}
```

Add styles (coral active per the customer system):

```tsx
  fulfillmentRow: { flexDirection: 'row', gap: 8, marginBottom: 16 },
  fulfillmentChip: { flex: 1, minHeight: 44, alignItems: 'center', justifyContent: 'center', borderRadius: 8, borderWidth: 1, borderColor: customerColors.hairline, backgroundColor: customerColors.surface.DEFAULT },
  fulfillmentChipActive: { borderColor: customerColors.coral.DEFAULT, backgroundColor: customerColors.coral.tint },
  fulfillmentChipText: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.soft },
  fulfillmentChipTextActive: { color: customerColors.coral.pressed },
```

When `fulfillment === 'pickup'`, the address picker is not required. Update `canPlaceOrder`:

```tsx
  const canPlaceOrder =
    cartStore.items.length > 0 &&
    (fulfillment === 'pickup' || !!selectedAddressId) &&
    !!cartStore.chefId &&
    !isLoading &&
    acceptedTerms;
```

Pass the mode to order creation (in the `createOrder.mutateAsync({...})` call) and make the address conditional:

```tsx
        chefId: cartStore.chefId,
        items: cartStore.items.map(/* unchanged */),
        deliveryAddressId: fulfillment === 'pickup' ? undefined : selectedAddressId,
        fulfillmentType: fulfillment,
        specialInstructions: note.trim() || undefined,
        deliverySlot: selectedSlot?.slot,
        deliveryDate: selectedSlot?.date,
        promoCode: appliedPromo?.code,
```

`deliveryFee` is already `0` in v1, so the total math needs no change.

- [ ] **Step 4: Typecheck**

Run: `cd apps/mobile-customer && npx tsc --noEmit`
Expected: no errors in `checkout.tsx` / `useOrderCheckout.ts` / `types/customer.ts`.

- [ ] **Step 5: Manual verification + commit**

Manual (sim, customer Metro on 8081): on a chef with pickup enabled, the checkout shows a Delivery/Pickup toggle; choosing Pickup lets you place an order without selecting an address; the order is created with no delivery fee.

```bash
git add apps/mobile-customer/hooks/useOrderCheckout.ts apps/mobile-customer/app/checkout.tsx apps/mobile-customer/types/customer.ts apps/mobile-customer/hooks/useChefs.ts
git commit -m "feat(checkout): Delivery/Pickup fulfillment selector"
```

---

### Task 8: Vendor profile — "Allow customer pickup" toggle

**Files:**
- Modify: `apps/mobile-vendor/app/profile.tsx` (state ~308; `UpdateChefProfilePayload` ~52; payload build ~370; render — add a Switch row)
- Modify: `apps/mobile-vendor/hooks/useVendorProfile.ts` (or wherever `UpdateChefProfilePayload` is declared) — add `offersPickup?: boolean`

**Interfaces:**
- Consumes: backend `UpdateChefProfileRequest.OffersPickup` (Task 1), `ChefProfileResponse.offersPickup`.

- [ ] **Step 1: Add `offersPickup` to the payload type** — in the file declaring `UpdateChefProfilePayload`, add:

```ts
  offersPickup?: boolean;
```

And to the profile data interface that reads the GET response (the one with `acceptingOrders: boolean` ~line 42), add `offersPickup: boolean;`.

- [ ] **Step 2: Add state + hydrate** — in `profile.tsx`, near the other `useState`s:

```tsx
  const [offersPickup, setOffersPickup] = useState(false);
```

Where the screen seeds local state from fetched `data` (the same effect that sets `addressLine1` etc.), add:

```tsx
    setOffersPickup(data.offersPickup ?? false);
```

- [ ] **Step 3: Render the toggle** — add a row (use the same `Switch` component the app already uses; import `Switch` from `react-native` if not imported) in the settings section:

```tsx
  <View style={styles.toggleRow}>
    <View style={{ flex: 1 }}>
      <Text style={styles.toggleLabel}>Allow customer pickup</Text>
      <Text style={styles.toggleHint}>Customers can collect their order from your kitchen.</Text>
    </View>
    <Switch
      value={offersPickup}
      onValueChange={setOffersPickup}
      accessibilityLabel="Allow customer pickup"
    />
  </View>
```

If `toggleRow`/`toggleLabel`/`toggleHint` styles don't exist, add minimal ones consistent with the vendor theme (ink text, `theme.spacing` paddings).

- [ ] **Step 4: Include it in the save payload** — in each `UpdateChefProfilePayload` build (the main save ~line 370 and any address-only save ~427), add:

```tsx
    offersPickup,
```

- [ ] **Step 5: Typecheck**

Run: `cd apps/mobile-vendor && npx tsc --noEmit`
Expected: clean for `profile.tsx` and the payload-type file.

- [ ] **Step 6: Manual verification + commit**

Manual (sim, stop customer Metro first — one Metro on 8081; start vendor Metro): the profile shows an "Allow customer pickup" switch; toggling + saving persists (re-open profile shows it on); the customer app then sees the pickup option for that chef.

```bash
git add apps/mobile-vendor/app/profile.tsx apps/mobile-vendor/hooks/useVendorProfile.ts
git commit -m "feat(vendor): Allow customer pickup toggle in chef profile"
```

---

## Deploy & verify (after all tasks)

- [ ] Push branch, open PR to `main`, owner merges.
- [ ] After merge: confirm the API CI build, then the Kargo/GAR-mirror deploy lands the new revision.
- [ ] Smoke test: enable pickup on a chef (with a geocodable address → coords backfilled), place a pickup order from the customer app, confirm no delivery fee, no 3PL dispatch, and the order's tracking shows the chef's exact address; place a delivery order and confirm the chef stays masked.

## Self-review notes

- Spec coverage: fulfillment field (T1), per-chef offering (T1/T8), self-delivery pricing (Phase 2 — not here), mode-aware visibility (T6 + T5), chef-coord capture (T2/T3), checkout UX (T7), vendor UX (T8). `chef_delivery` enum value exists (T1) with default behavior only — Phase 2 adds its logic.
- Money/INR, AutoMigrate, masking-kept-for-3PL, dep pins, PR workflow: in Global Constraints.
- Pickup pre-commit address visibility is intentionally per-order (revealed after ordering via TrackOrder), not at browse — browse stays fuzzed to avoid pre-order address harvesting.
