# Chef-delivery Streamline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the chef the one who chooses, at "Mark Ready", whether to deliver a Delivery order themselves or hand it to a rider (3PL) — and stop chef-delivery orders vanishing into History before they're delivered.

**Architecture:** Orders are created as plain `delivery`/`pickup` (no creation-time carrier routing). For a self-delivering chef, the "Mark Ready" action carries a `carrier` choice that flips `FulfillmentType` to `chef_delivery` (chef carries it) or leaves it `delivery` (3PL). Active/History bucketing becomes fulfillment-aware so chef-delivery stays active until Delivered. 3PL *dispatch* itself is unchanged (deferred to Shadowfax).

**Tech Stack:** Go 1.26 + Gin + GORM (backend); Expo/React Native + TypeScript (mobile-customer, mobile-vendor).

## Global Constraints

- Go: `gofmt` before commit; tests via `go test ./...`.
- Mobile: `@babel/core ^7` + `tailwindcss ^3` pins; `npx tsc --noEmit` must be clean except the pre-existing `app/reviews.tsx` router-typing error (vendor). Jest is broken repo-wide — frontend tasks verify via `tsc` + manual sim, not unit tests.
- Customer never picks the carrier; the customer only sees Delivery vs Pickup.
- Privacy: full customer address/phone is exposed to the chef ONLY when `FulfillmentType == chef_delivery`. `delivery` and `pickup` stay area-only (existing `ToChefResponse`).
- 3PL dispatch is deferred to the Shadowfax workstream — "Hand to a rider" just leaves the order `delivery` (the existing `EnqueueDeliveryDispatch` path); do not build rider tracking here.
- Single-line commit messages, conventional prefix, no signature.

---

### Task 1: Stop auto-routing delivery → chef_delivery at order creation

PR #344 made `resolveFulfillment` return `chef_delivery` for any delivery order to a self-delivering chef. The carrier is now decided at Mark Ready instead, so creation must keep the order `delivery`.

**Files:**
- Modify: `apps/api/handlers/orders.go` (`resolveFulfillment`)
- Test: `apps/api/handlers/orders_fulfillment_test.go`

**Interfaces:**
- Produces: `resolveFulfillment(req CreateOrderRequest, chef models.ChefProfile) (models.FulfillmentType, error)` — `""`/`"delivery"` → `FulfillmentDelivery` (regardless of `OffersSelfDelivery`); `"pickup"` → `FulfillmentPickup` (if `OffersPickup`); `"chef_delivery"` → `FulfillmentChefDelivery` (if `OffersSelfDelivery`, defensive for old clients).

- [ ] **Step 1: Update the test for the new behavior**

In `apps/api/handlers/orders_fulfillment_test.go`, replace the two self-delivery assertions added in #344 so a self-delivering chef's `delivery`/empty order resolves to `FulfillmentDelivery`:

```go
	chefSelfDelivers := models.ChefProfile{OffersSelfDelivery: true}

	// Carrier is chosen at Mark Ready now, NOT at creation: a delivery order to a
	// self-delivering chef is still created as plain delivery.
	if ft, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "delivery"}, chefSelfDelivers); err != nil || ft != models.FulfillmentDelivery {
		t.Fatalf("delivery should stay delivery at creation, got %q err=%v", ft, err)
	}
	if ft, err := resolveFulfillment(CreateOrderRequest{}, chefSelfDelivers); err != nil || ft != models.FulfillmentDelivery {
		t.Fatalf("empty mode should be delivery at creation, got %q err=%v", ft, err)
	}
	// Defensive: an explicit chef_delivery from an old client is still honored.
	if ft, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "chef_delivery"}, chefSelfDelivers); err != nil || ft != models.FulfillmentChefDelivery {
		t.Fatalf("explicit chef_delivery should be honored, got %q err=%v", ft, err)
	}
```

- [ ] **Step 2: Run the test, verify it FAILS**

Run: `cd apps/api && go test ./handlers/ -run TestResolveFulfillment -v`
Expected: FAIL (current code returns `chef_delivery` for the delivery+self-deliver case).

- [ ] **Step 3: Remove the creation-time auto-route**

In `apps/api/handlers/orders.go`, change the `case "", models.FulfillmentDelivery:` branch of `resolveFulfillment` to always return delivery:

```go
	case "", models.FulfillmentDelivery:
		// The carrier (chef vs 3PL) is chosen by the chef at Mark Ready, not at
		// creation — so a delivery order is always created as plain delivery.
		return models.FulfillmentDelivery, nil
```

(Leave the `models.FulfillmentChefDelivery` and `models.FulfillmentPickup` cases unchanged.)

- [ ] **Step 4: Run the test, verify it PASSES**

Run: `cd apps/api && go test ./handlers/ -run TestResolveFulfillment -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd apps/api && gofmt -w handlers/orders.go handlers/orders_fulfillment_test.go
git add handlers/orders.go handlers/orders_fulfillment_test.go
git commit -m "fix(fulfillment): create delivery orders as 3PL; carrier chosen at Mark Ready not creation"
```

---

### Task 2: Accept a carrier choice on the Mark-Ready transition

`UpdateOrderStatus` currently takes only `{status}`. Extend it so that when a self-delivering chef transitions a `delivery` order to `ready` (or while it is still `ready`, before anyone is en route), an optional `carrier` flips `FulfillmentType`. `"chef_delivery"` → chef carries it (suppresses the 3PL auto-dispatch + reveals the address); `"delivery"` → 3PL (the existing dispatch path).

**Files:**
- Modify: `apps/api/handlers/chefs.go` (`UpdateOrderStatus`, ~line 863 request struct + carrier handling)
- Test: `apps/api/handlers/orders_fulfillment_test.go` (add `resolveReadyCarrier` unit test)

**Interfaces:**
- Produces: helper `resolveReadyCarrier(current models.FulfillmentType, requested string, chef models.ChefProfile) (models.FulfillmentType, error)` — pure, testable. Rules: empty requested → keep `current`; `chef_delivery` requires `chef.OffersSelfDelivery` AND `current` is `delivery`/`chef_delivery` → returns `chef_delivery`; `delivery` (when switching back) requires `current` is `delivery`/`chef_delivery` → returns `delivery`; `pickup` orders reject any carrier; unknown → error.

- [ ] **Step 1: Write the failing unit test for `resolveReadyCarrier`**

Add to `apps/api/handlers/orders_fulfillment_test.go`:

```go
func TestResolveReadyCarrier(t *testing.T) {
	self := models.ChefProfile{OffersSelfDelivery: true}
	noSelf := models.ChefProfile{}

	// no carrier requested → keep current
	if ft, err := resolveReadyCarrier(models.FulfillmentDelivery, "", self); err != nil || ft != models.FulfillmentDelivery {
		t.Fatalf("empty keeps current; got %q err=%v", ft, err)
	}
	// self-delivering chef picks "I'll deliver"
	if ft, err := resolveReadyCarrier(models.FulfillmentDelivery, "chef_delivery", self); err != nil || ft != models.FulfillmentChefDelivery {
		t.Fatalf("want chef_delivery; got %q err=%v", ft, err)
	}
	// switch back to rider while still delivery-side
	if ft, err := resolveReadyCarrier(models.FulfillmentChefDelivery, "delivery", self); err != nil || ft != models.FulfillmentDelivery {
		t.Fatalf("want delivery; got %q err=%v", ft, err)
	}
	// chef who doesn't self-deliver cannot pick chef_delivery
	if _, err := resolveReadyCarrier(models.FulfillmentDelivery, "chef_delivery", noSelf); err == nil {
		t.Fatal("non-self-delivering chef must be rejected")
	}
	// pickup orders reject any carrier choice
	if _, err := resolveReadyCarrier(models.FulfillmentPickup, "chef_delivery", self); err == nil {
		t.Fatal("pickup orders must reject a carrier choice")
	}
}
```

- [ ] **Step 2: Run it, verify it FAILS**

Run: `cd apps/api && go test ./handlers/ -run TestResolveReadyCarrier -v`
Expected: FAIL (`resolveReadyCarrier` undefined).

- [ ] **Step 3: Implement `resolveReadyCarrier` in `orders.go`**

Add below `resolveFulfillment` in `apps/api/handlers/orders.go`:

```go
// resolveReadyCarrier resolves the chef's per-order carrier choice made at Mark
// Ready. The customer chose delivery vs pickup; the chef chooses who carries a
// delivery order. Empty keeps the current type. Only delivery↔chef_delivery
// switches are allowed, only for a self-delivering chef, and never for pickup.
func resolveReadyCarrier(current models.FulfillmentType, requested string, chef models.ChefProfile) (models.FulfillmentType, error) {
	if requested == "" {
		return current, nil
	}
	if current == models.FulfillmentPickup {
		return "", fmt.Errorf("pickup orders have no carrier choice")
	}
	switch models.FulfillmentType(requested) {
	case models.FulfillmentChefDelivery:
		if !chef.OffersSelfDelivery {
			return "", fmt.Errorf("this kitchen does not offer self-delivery")
		}
		return models.FulfillmentChefDelivery, nil
	case models.FulfillmentDelivery:
		return models.FulfillmentDelivery, nil
	default:
		return "", fmt.Errorf("unsupported carrier")
	}
}
```

- [ ] **Step 4: Run it, verify it PASSES**

Run: `cd apps/api && go test ./handlers/ -run TestResolveReadyCarrier -v`
Expected: PASS.

- [ ] **Step 5: Wire the carrier into `UpdateOrderStatus`**

In `apps/api/handlers/chefs.go`, extend the request struct (~line 863) and apply the carrier before the timestamp switch. Carrier is only honored while the order has not left the chef's hands (status not yet `picked_up`/`delivered`):

```go
	var req struct {
		Status string `json:"status" binding:"required"`
		// Optional carrier choice the chef makes at Mark Ready (self-delivery
		// chefs only): "chef_delivery" = I'll deliver, "delivery" = hand to a rider.
		Carrier string `json:"carrier"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply the chef's carrier choice — allowed only before the order is en route.
	if req.Carrier != "" {
		if order.Status == models.OrderStatusPickedUp || order.Status == models.OrderStatusDelivered {
			c.JSON(http.StatusBadRequest, gin.H{"error": "carrier is locked once the order is out for delivery"})
			return
		}
		ft, err := resolveReadyCarrier(order.FulfillmentType, req.Carrier, chef)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		order.FulfillmentType = ft
	}

	priorStatus := order.Status
	order.Status = models.OrderStatus(req.Status)
```

The existing auto-dispatch guard (`order.Status == ready && order.FulfillmentType == delivery`) then naturally fires 3PL only when the chef chose "hand to a rider", and not for `chef_delivery`. No change needed there.

- [ ] **Step 6: Build + full backend tests**

Run: `cd apps/api && gofmt -w handlers/chefs.go handlers/orders.go handlers/orders_fulfillment_test.go && go build ./... && go test ./handlers/ ./services/`
Expected: build OK, tests PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/api/handlers/chefs.go apps/api/handlers/orders.go apps/api/handlers/orders_fulfillment_test.go
git commit -m "feat(chef-delivery): chef picks carrier (deliver vs rider) at Mark Ready; locked once en route"
```

---

### Task 3: Surface chef→drop distance on delivery orders (for the Ready decision)

`GetOrderDetail` only computes `selfDeliveryDistanceKm`/`MaxDistanceKm` for `chef_delivery` orders (#342). The chef now needs that distance on a still-`delivery` order to decide at Mark Ready, and only AFTER they pick "I'll deliver" does the order become `chef_delivery` (revealing the address). So compute the distance whenever the chef self-delivers, regardless of current fulfillment type.

**Files:**
- Modify: `apps/api/handlers/chefs.go` (`GetOrderDetail`, the `if order.FulfillmentType == models.FulfillmentChefDelivery` block)

**Interfaces:**
- Consumes: `services.ComputeSelfDeliveryDistanceKm`, `chef.SelfDeliveryMaxDistanceKm`, `chef.OffersSelfDelivery` (all existing).
- Produces: `ChefOrderDetailResponse.SelfDeliveryDistanceKm` + `SelfDeliveryMaxDistanceKm` populated for `chef_delivery` OR (`delivery` AND `chef.OffersSelfDelivery`).

- [ ] **Step 1: Broaden the distance computation**

In `apps/api/handlers/chefs.go` `GetOrderDetail`, replace the condition:

```go
	// Surface the chef→drop distance + comfort radius for the Mark-Ready carrier
	// decision: on chef_delivery orders AND on delivery orders the chef COULD
	// self-deliver (so the distance is visible before they choose "I'll deliver").
	if order.FulfillmentType == models.FulfillmentChefDelivery ||
		(order.FulfillmentType == models.FulfillmentDelivery && chef.OffersSelfDelivery) {
		detail.SelfDeliveryDistanceKm = services.ComputeSelfDeliveryDistanceKm(
			chef, order.DeliveryLatitude, order.DeliveryLongitude,
		)
		detail.SelfDeliveryMaxDistanceKm = chef.SelfDeliveryMaxDistanceKm
	}
```

- [ ] **Step 2: Build**

Run: `cd apps/api && gofmt -w handlers/chefs.go && go build ./...`
Expected: build OK. (No new behavior to unit-test; distance fn already covered by `TestComputeSelfDeliveryDistanceKm`.)

- [ ] **Step 3: Commit**

```bash
git add apps/api/handlers/chefs.go
git commit -m "feat(chef-delivery): expose chef->drop distance on delivery orders for the Mark Ready decision"
```

---

### Task 4: Fulfillment-aware Active/History bucketing (vendor)

Fixes the core bug: a `chef_delivery` order at `picked_up` (chef out delivering) must stay in the **active** flow until `delivered`; only 3PL pickups belong in History.

**Files:**
- Modify: `apps/mobile-vendor/app/(tabs)/orders.tsx` (`HISTORY_STATUSES` set → a predicate)
- Modify: `apps/mobile-vendor/hooks/useOrderDetail.ts` (already exposes `fulfillmentType` + `selfDeliveryDistanceKm` — no change; referenced for context)

**Interfaces:**
- Produces: `isHistoryOrder(o: { status: string; fulfillmentType?: string }): boolean` — true for `delivered`/`cancelled`/`rejected` always, and for `picked_up` ONLY when `fulfillmentType !== 'chef_delivery'`.

- [ ] **Step 1: Replace the status Set with a predicate**

In `apps/mobile-vendor/app/(tabs)/orders.tsx`, replace the `HISTORY_STATUSES` set and its usage:

```ts
// A chef-delivery order at `picked_up` means the CHEF is out delivering — still
// active. Only 3PL pickups (rider has it, out of the chef's hands) are History.
function isHistoryOrder(o: { status: string; fulfillmentType?: string }): boolean {
  if (o.status === 'delivered' || o.status === 'cancelled' || o.status === 'rejected') {
    return true;
  }
  if (o.status === 'picked_up') return o.fulfillmentType !== 'chef_delivery';
  return false;
}
```

Then update the `HistoryTab` filter (currently `HISTORY_STATUSES.has(o.status)`):

```ts
  const orders = useMemo(
    () => (data?.orders ?? []).filter(isHistoryOrder),
    [data?.orders],
  );
```

- [ ] **Step 2: Typecheck**

Run: `cd apps/mobile-vendor && npx tsc --noEmit 2>&1 | grep -v "app/reviews.tsx"`
Expected: no output (clean).

- [ ] **Step 3: Commit**

```bash
git add apps/mobile-vendor/app/\(tabs\)/orders.tsx
git commit -m "fix(vendor): keep chef-delivery orders active until delivered (only 3PL pickups go to History)"
```

---

### Task 5: Mark-Ready carrier choice UI + distance steering (vendor)

When a self-delivering chef marks a `delivery` order ready, present two actions: "I'll deliver" (sends `carrier: chef_delivery`) and "Hand to a rider" (sends `carrier: delivery`). If the drop is beyond the chef's range, surface the distance warning and visually recommend "Hand to a rider".

**Files:**
- Modify: `apps/mobile-vendor/hooks/useVendorOrders.ts` (`useUpdateOrderStatus` mutation — accept optional `carrier`)
- Modify: `apps/mobile-vendor/app/orders/[orderId].tsx` (Mark-Ready footer for self-deliverable delivery orders)

**Interfaces:**
- Consumes: `useUpdateOrderStatus().mutate({ orderId, status, carrier? })`; `order.selfDeliveryDistanceKm`, `order.selfDeliveryMaxDistanceKm`, `order.fulfillmentType`.
- Produces: `captureAndAdvanceWithCarrier(kind, carrier)` — uploads the ready photo then calls the status mutation with `{ status: 'ready', carrier }`.

- [ ] **Step 1: Extend the status mutation to pass `carrier`**

In `apps/mobile-vendor/hooks/useVendorOrders.ts`, update `useUpdateOrderStatus`'s `mutationFn` and param type:

```ts
    mutationFn: ({
      orderId,
      status,
      carrier,
    }: {
      orderId: string;
      status: Order['status'];
      carrier?: 'chef_delivery' | 'delivery';
    }) => api.put(`/chef/orders/${orderId}/status`, { status, carrier }),
```

- [ ] **Step 2: Render the two carrier actions at Mark Ready**

In `apps/mobile-vendor/app/orders/[orderId].tsx`, the chef self-deliverability is `order.selfDeliveryMaxDistanceKm != null` is NOT a reliable signal — instead the backend exposes a distance only when the chef can self-deliver. Compute eligibility from the presence of distance data: `const canSelfDeliver = order.fulfillmentType === 'delivery' && order.selfDeliveryDistanceKm != null;` Add (near the existing `overSelfDeliveryRange` derivation):

```ts
  const canSelfDeliver =
    order.fulfillmentType === 'delivery' &&
    (order.selfDeliveryMaxDistanceKm > 0 || order.selfDeliveryDistanceKm > 0);
  const overRange =
    canSelfDeliver &&
    order.selfDeliveryMaxDistanceKm > 0 &&
    order.selfDeliveryDistanceKm > order.selfDeliveryMaxDistanceKm;
```

In `FooterActions`, when `status === 'ready'` is reached the existing flow shows "Out for delivery"/3PL caption. Change the **Mark Ready** action (currently `onMarkReady={() => captureAndAdvance('ready', 'ready')}`) so that for a self-deliverable delivery order it offers the carrier choice instead of a single button. Add a `captureAndAdvanceWithCarrier` in the screen component:

```ts
  async function captureAndAdvanceWithCarrier(
    carrier: 'chef_delivery' | 'delivery',
  ): Promise<void> {
    if (!order || uploadPhoto.isPending || updateStatus.isPending) return;
    const asset = await pickReadyPhoto(); // existing camera/library logic, see note
    if (!asset?.uri) return;
    try {
      await uploadPhoto.mutateAsync({ orderId: order.id, kind: 'ready', uri: asset.uri });
      await updateStatus.mutateAsync({ orderId: order.id, status: 'ready', carrier });
    } catch {
      showToast({ message: 'Could not update the order. Please try again.', tone: 'error' });
    }
  }
```

NOTE: extract the camera/library pick from the existing `captureAndAdvance` into a `pickReadyPhoto()` helper that returns the asset (reuse the `Device.isDevice ? camera : library` logic added earlier), so both `captureAndAdvance` and `captureAndAdvanceWithCarrier` share it.

For the pre-ready footer (status `accepted`/`preparing`) of a self-deliverable order, render two buttons instead of one "Mark ready":

```tsx
{canSelfDeliver ? (
  <View style={styles.footerStack}>
    {overRange ? (
      <Text style={styles.carrierHint}>
        {`This drop is ${order.selfDeliveryDistanceKm.toFixed(1)} km away — beyond your ${order.selfDeliveryMaxDistanceKm} km range. Handing it to a rider is recommended.`}
      </Text>
    ) : null}
    <Pressable onPress={() => onReadyCarrier('chef_delivery')} disabled={disabled} accessibilityRole="button" accessibilityLabel="Mark ready, I will deliver it myself">
      <View style={[overRange ? styles.secondaryBtnFull : styles.primaryBtnFull, disabled && { opacity: 0.4 }]}>
        <Text style={overRange ? styles.secondaryLabel : styles.primaryLabel}>Ready · I'll deliver</Text>
      </View>
    </Pressable>
    <Pressable onPress={() => onReadyCarrier('delivery')} disabled={disabled} accessibilityRole="button" accessibilityLabel="Mark ready, hand to a rider">
      <View style={[overRange ? styles.primaryBtnFull : styles.secondaryBtnFull, disabled && { opacity: 0.4 }]}>
        <Text style={overRange ? styles.primaryLabel : styles.secondaryLabel}>Ready · Hand to a rider</Text>
      </View>
    </Pressable>
  </View>
) : (
  /* existing single "Mark ready" button */
)}
```

Thread `onReadyCarrier` through `FooterActions` props to `captureAndAdvanceWithCarrier`. Reuse existing `primaryBtnFull`/`primaryLabel` styles; add `secondaryBtnFull`/`secondaryLabel` (ink outline) + `footerStack` (gap column) + `carrierHint` (amber.tint text) if not present.

- [ ] **Step 3: Typecheck**

Run: `cd apps/mobile-vendor && npx tsc --noEmit 2>&1 | grep -v "app/reviews.tsx"`
Expected: clean.

- [ ] **Step 4: Manual sim verification**

Vendor: open an accepted/preparing delivery order from a self-delivering chef → footer shows "Ready · I'll deliver" + "Ready · Hand to a rider". Picking "I'll deliver" → photo → order becomes chef_delivery, full address shows, "Out for delivery" appears, order stays in active. Over-range order shows the hint and emphasizes "Hand to a rider".

- [ ] **Step 5: Commit**

```bash
git add apps/mobile-vendor/hooks/useVendorOrders.ts apps/mobile-vendor/app/orders/\[orderId\].tsx
git commit -m "feat(vendor): Mark Ready offers I'll-deliver vs hand-to-rider, with over-range nudge"
```

---

### Task 6: Carrier-aware wording + switch-before-en-route (vendor)

Kill "awaiting pickup"/"awaiting driver" copy on chef-delivery; allow the chef to flip the carrier while the order is still `ready` (before "Out for delivery").

**Files:**
- Modify: `apps/mobile-vendor/app/orders/[orderId].tsx` (`statusLabelFor`, the `ready` footer caption, add a "Hand to a rider instead"/"I'll deliver instead" switch link)

**Interfaces:**
- Consumes: `useUpdateOrderStatus().mutate({ orderId, status:'ready', carrier })` (re-sends `ready` with the other carrier to switch).

- [ ] **Step 1: Make `statusLabelFor` carrier-aware**

In `apps/mobile-vendor/app/orders/[orderId].tsx`, update `statusLabelFor` so `picked_up` reads "Out for delivery" for chef_delivery and "Picked up by rider" for 3PL, and `ready` 3PL reads "Ready · awaiting rider":

```ts
function statusLabelFor(status: OrderDetailStatus, fulfillment: FulfillmentType): string {
  if (fulfillment === 'pickup') {
    if (status === 'delivered' || status === 'picked_up') return 'Collected';
    if (status === 'ready') return 'Ready for pickup';
  }
  if (fulfillment === 'chef_delivery' && status === 'picked_up') return 'Out for delivery';
  if (fulfillment === 'delivery' && status === 'picked_up') return 'Picked up by rider';
  return STATUS_LABEL[status] ?? status;
}
```

- [ ] **Step 2: Add the "switch carrier" link in the `ready` footer**

In `FooterActions`, for `status === 'ready'`: chef_delivery shows "Out for delivery" (existing) + a quiet link "Hand to a rider instead" → `onReadyCarrier('delivery')`; the 3PL `ready` caption ("Waiting for driver to pick up") gains a quiet link "I'll deliver this instead" → `onReadyCarrier('chef_delivery')` (only when `canSelfDeliver`-equivalent, i.e., the chef self-delivers). Reuse the `cancelLink`-style quiet link styling.

- [ ] **Step 3: Typecheck + manual sim**

Run: `cd apps/mobile-vendor && npx tsc --noEmit 2>&1 | grep -v "app/reviews.tsx"` → clean.
Sim: a `ready` chef_delivery order shows "Out for delivery" + "Hand to a rider instead"; tapping the link flips it to 3PL `ready` (caption "Ready · awaiting rider") and the 3PL footer offers "I'll deliver this instead".

- [ ] **Step 4: Commit**

```bash
git add apps/mobile-vendor/app/orders/\[orderId\].tsx
git commit -m "feat(vendor): carrier-aware status wording + switch carrier while order is ready"
```

---

### Task 7: Verify customer-facing wording is carrier-neutral

The customer must never see who carries the order; "on the way"/"almost ready" wording (shipped in #344/#345) must read correctly for `chef_delivery`.

**Files:**
- Verify (no change expected): `apps/mobile-customer/lib/orderSteps.ts`, `apps/mobile-customer/app/order/[id]/index.tsx`

- [ ] **Step 1: Confirm wording**

Grep for any carrier-specific customer copy:

```bash
cd apps/mobile-customer && grep -rn "awaiting driver\|awaiting pickup\|chef will\|rider" lib/orderSteps.ts app/order/\[id\]/index.tsx
```
Expected: none (already neutral from #344/#345). `getStatusLine`/`getInlineStatusLabel` `ready` → "Almost ready — heading your way soon"; `picked_up` → "on the way". If any carrier-specific copy is found, neutralize it.

- [ ] **Step 2: Typecheck (if changed)**

Run: `cd apps/mobile-customer && npx tsc --noEmit` → clean.

- [ ] **Step 3: Commit (only if changed)**

```bash
git add apps/mobile-customer/lib/orderSteps.ts apps/mobile-customer/app/order/\[id\]/index.tsx
git commit -m "fix(customer): keep order status wording carrier-neutral"
```

---

## Self-review notes

- **Spec coverage:** customer Delivery/Pickup only (already shipped #344; Task 1 reverts the creation auto-route) ✓; carrier at Mark Ready (Tasks 2,5) ✓; distance steering (Tasks 3,5) ✓; privacy reveal on chef_delivery only (automatic via FulfillmentType flip + existing ToChefResponse; Task 2/3) ✓; active-until-delivered (Task 4) ✓; switch-before-en-route (Tasks 2,6) ✓; carrier-aware wording (Tasks 6,7) ✓; 3PL dispatch deferred (untouched) ✓.
- **Privacy:** address reveals only when FulfillmentType becomes chef_delivery — Task 2 sets it server-side; Task 3 only adds DISTANCE (not address) to delivery orders. ✓
- **3PL "Hand to a rider"** leaves the order `delivery`, so the existing `EnqueueDeliveryDispatch` (ready + delivery) path fires — correct, and parks until Shadowfax. ✓
