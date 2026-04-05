# Feature Landscape — Home Chef Mobile Apps

**Domain:** Food delivery mobile applications (Customer, Vendor/Chef, Delivery Driver)
**Researched:** 2026-04-05
**Confidence:** HIGH (based on codebase analysis + established industry patterns from Swiggy, Zomato, Uber Eats, DoorDash)

---

## Context: What the Web Already Has

The following features exist in web portals and need mobile parity — they are NOT differentiated by being on mobile:

- Auth (login, register, forgot password)
- Browse/discover chefs, view menus
- Cart, checkout, payment (Razorpay)
- Order history, order details
- Order tracking (map view)
- Favorites
- Social feed
- Catering requests and quotes
- Vendor: menu management, live orders, earnings, analytics, reviews
- Driver: available deliveries, active delivery, GPS navigation, earnings, fleet/staff management

The question this document answers is: **what mobile-native capabilities go beyond web parity?**

---

## Table Stakes

Features users expect on mobile. Missing = app feels broken or incomplete. Users will leave for a competitor.

| Feature | Why Expected | App(s) | Complexity | Notes |
|---------|--------------|--------|------------|-------|
| Push notifications — order lifecycle | Every food app sends these. Without them users don't know when order is accepted, ready, or delivered. | Customer, Vendor, Driver | Low | Backend FCM already exists (`services/push.go`). Expo Notifications handles FCM token registration. Need Expo token → FCM bridge. |
| Push notifications — actionable (accept/reject) | Chefs on mobile must be able to accept orders from lock screen without opening app. iOS notification actions and Android notification buttons are expected by vendors running kitchens. | Vendor | Medium | Requires notification categories with action buttons (expo-notifications supports this). Needs background task handler. |
| Biometric authentication (Face ID / Fingerprint) | Users hate typing passwords on mobile. Every banking and food app offers biometric login after first sign-in. | Customer, Vendor, Driver | Low | `expo-local-authentication` covers iOS Face ID / Touch ID and Android Fingerprint. Store JWT refresh token in Keychain (expo-secure-store), re-auth via biometric on app resume. |
| Location permissions + background GPS (Driver) | Drivers cannot navigate without location access. Background location is needed while screen is off during delivery. | Driver | Medium | `expo-location` with `background` permission. Requires foreground service on Android. Battery-drain tradeoff must be managed (stop tracking after delivery complete). |
| Deep links / universal links | SMS and email notifications link directly to order detail or chef page. Without deep links users land on home screen and churn. | Customer, Vendor, Driver | Low | Expo Router supports deep linking. Backend already sends emails with order IDs. Need URL scheme mapping (`homechef://orders/:id`). |
| App icon badge (unread count) | iOS and Android show notification badge count. Users expect this for unread orders and notifications. | Customer, Vendor | Low | `expo-notifications` `setBadgeCountAsync`. Backend already has `/notifications/unread-count` endpoint. |
| Haptic feedback on key interactions | "Order placed" confirmation, accept/reject order, delivery completion. Mobile users expect tactile confirmation. Absence feels cheap. | Customer, Vendor, Driver | Low | `expo-haptics` — trivial to add at confirmation moments. |
| Pull-to-refresh | Standard mobile pattern on every list view (orders, notifications, deliveries). | All | Low | React Native `RefreshControl` on `ScrollView` / `FlatList`. Already handled by React Query `refetch`. |
| Skeleton loading screens | Mobile networks are slower and less reliable. Blank screens with no feedback cause users to assume the app crashed. | All | Low | Replace spinners with skeleton placeholders on list/detail views. |
| Offline error state (graceful, not full offline mode) | Show clear "No connection" state rather than blank screen or cryptic error when network drops. | All | Low | React Native `NetInfo` (`@react-native-community/netinfo`) + error boundary. Not full offline mode — just a graceful degraded state. |
| Camera — food photos (Vendor) | Chefs adding menu items must be able to take or upload a photo from their phone. | Vendor | Low | `expo-image-picker` with `launchCameraAsync` + `launchImageLibraryAsync`. Upload to GCS via existing `POST /api/v1/upload` endpoint. |
| Camera — document upload (Driver onboarding) | Drivers submitting ID / vehicle docs need camera access. Web portal requires file upload; mobile needs native camera sheet. | Driver | Low | Same `expo-image-picker` pattern as above. |
| Secure token storage | JWT and refresh tokens must not be stored in AsyncStorage (plain text). Mobile keychain is table stakes for security. | All | Low | `expo-secure-store` wraps iOS Keychain and Android Keystore. Replace any AsyncStorage token usage. |
| Auto-refresh tokens in background | Session expiry on mobile is jarring. Apps silently refresh in background rather than kicking the user to a login screen mid-session. | All | Low | Intercept 401 responses in API client, call refresh endpoint, retry original request. Axios interceptor pattern. |
| Social login — Google / Apple Sign-In | Apple Sign-In is App Store mandatory for apps that have any social login. Google is expected. | Customer, Vendor, Driver | Medium | `expo-auth-session` + `@invertase/react-native-google-signin`. Apple Sign-In via `expo-apple-authentication`. Backend already has Apple/Google OAuth config. |

---

## Differentiators

Features that set the app apart. Not expected by default, but meaningfully improve retention and NPS.

| Feature | Value Proposition | App(s) | Complexity | Notes |
|---------|-------------------|--------|------------|-------|
| Live order tracking with animated map | Web has a static map. Mobile can show a moving pin with smooth animation as driver location updates in real time. This is the single biggest UX delta vs web. | Customer | High | Requires WebSocket or polling for driver location from backend. `react-native-maps` + `Animated` API for smooth pin movement. Backend needs `PUT /deliveries/:id/location` and a way to broadcast to customer (NATS → WebSocket or polling). |
| Driver turn-by-turn navigation (native handoff) | Instead of showing a map inside the app, offer a "Navigate" button that opens Google Maps / Apple Maps / Waze with the destination pre-loaded. Best-practice for driver apps — navigating in a dedicated nav app is safer and more accurate. | Driver | Low | `Linking.openURL` with platform-specific deep link (`maps://`, `comgooglemaps://`, `waze://`). Fallback to browser Google Maps URL. |
| Notification preference management | Let users choose which notifications they want (new order accepted vs. every status change). Reduces opt-out rate. | Customer, Vendor | Medium | Store preferences in user profile. Backend already has notification infrastructure. UI is a settings screen with toggles. |
| Shake to report issue | Shake the phone to trigger a support ticket pre-filled with the current order context. Reduces friction for bug reports. Common in food delivery driver apps (Swiggy Genie, Porter). | Driver | Low | `expo-sensors` accelerometer threshold detection. Opens the support ticket flow with context auto-populated. |
| Rating prompt at the right moment | Post-delivery in-app review prompt (App Store / Play Store rating request) when user is most satisfied. Dramatically improves app store rating. | Customer | Low | `expo-store-review` — show after order delivered, first time only, no more than once per 30 days. |
| Scheduled order reminders | Push notification 30 min before a scheduled catering or pre-order delivery. | Customer | Medium | Requires backend scheduled notification job (NATS-based cron or a new endpoint). Expo can't schedule far-future local notifications reliably — must be server-side push. |
| Photo confirmation on delivery | Driver takes a photo of delivered order at the door (proof of delivery), especially for contactless. Visible to customer in order detail. | Driver, Customer | Medium | `expo-camera` + upload to GCS. Add `delivery_photo_url` field to delivery model. Common in Swiggy/Zomato's driver app. |
| Order reorder shortcut | One-tap reorder from order history. Mobile home screen widget (future). | Customer | Low | `POST /orders` with previous order's items. Simple button on order history card. No new API needed. |
| Chef availability toggle widget (iOS/Android Home Screen) | Vendors toggle "I'm available today" from a home screen widget without opening the app. Useful for home chefs who cook part-time. | Vendor | High | Requires `expo-widgets` (iOS WidgetKit) or Android AppWidget. Complex — out of scope v1, flag for v2. |
| Smart ETA display | Show "arrives in ~12 min" updating in real time on Customer app home screen / order tracker rather than static time. | Customer | Medium | Client-side timer countdown from server ETA + live driver location polling. |

---

## Anti-Features

Features to deliberately NOT build in this milestone.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Full offline mode with sync | Mentioned as out of scope in PROJECT.md. Requires complex conflict resolution infrastructure for orders, menus, delivery state. Risk outweighs benefit for v1. | Show clear offline error states. Allow cached read-only views (React Query stale cache). |
| In-app payment UI (Apple Pay / Google Pay native sheet) | PROJECT.md explicitly deferred to v1 in favor of Razorpay web checkout. Building a native payment sheet is high effort (Razorpay SDK integration, PCI scope, Apple Pay merchant registration). | Use Razorpay's WebView-based checkout (their SDK supports React Native via WebView wrapper). |
| Custom video calling | Not in web version. Home chefs and customers rarely need synchronous video. Adds significant infrastructure (WebRTC, media servers). | Use in-app chat (already exists) with photo sharing. |
| AR menu preview | Future enhancement per PROJECT.md. ARKit/ARCore integration is complex and device-dependent. | High-quality static food photos via camera upload are sufficient. |
| Tablet-optimized layouts | PROJECT.md explicitly phone-first. Responsive tablet layout doubles design effort. | Target phones (375px–430px width). Let tablet users get phone layout scaled up. |
| Background location tracking for Customer | Customers do not need GPS tracked. Privacy-invasive, battery draining. App Store reviewers scrutinize this. | Only request "when in use" location for address auto-fill and chef discovery. |
| Infinite notification history | Storing and displaying thousands of notifications creates scroll performance issues and storage cost. | Paginated list (already supported by existing notifications endpoint). Auto-expire after 90 days server-side. |
| Live chat video/audio within the app | Between customer and chef or driver. Complex, liability for inappropriate content, outside scope. | Text-only in-app chat (already exists in API: `handlers/chat.go`). |

---

## Feature Dependencies

```
Biometric auth → Secure token storage (must exist first)
Push notifications (actionable) → Push notification token registration (must exist first)
Live order map (animated driver pin) → Driver background GPS tracking (must exist first)
Photo confirmation (delivery) → Camera permissions → GCS upload endpoint (already exists)
Social login (Apple) → Apple Sign-In entitlement in EAS build config (must exist first)
App icon badge → Push notification token registration (must exist first)
Scheduled order reminders → Backend scheduled push (new capability needed)
Reorder shortcut → Order history screen (must exist first)
```

---

## MVP Recommendation

### Customer App — MVP Must-Haves (beyond web parity)
1. Push notifications — order lifecycle (accepted, ready, picked up, delivered)
2. Secure token storage + auto-refresh
3. Biometric auth (Face ID / Fingerprint)
4. Deep links (order detail from notification tap)
5. App icon badge (unread count)
6. Social login — Google + Apple Sign-In
7. Haptic feedback on order placed / payment confirmed
8. Pull-to-refresh on all list views
9. Skeleton loading states
10. Offline error state (graceful)

### Vendor/Chef App — MVP Must-Haves
1. Push notifications — new order with actionable accept/reject from lock screen
2. Camera — food photo upload for menu items
3. Secure token storage + auto-refresh
4. Biometric auth
5. Deep links (new order notification → order detail)
6. App icon badge (pending orders count)
7. Haptic on order accepted/rejected
8. Pull-to-refresh

### Driver App — MVP Must-Haves
1. Background location permission + GPS (active delivery tracking)
2. Push notifications — delivery assigned
3. Camera — document upload (onboarding) + proof-of-delivery photo
4. Native navigation handoff (Google Maps / Apple Maps deep link)
5. Secure token storage + auto-refresh
6. Biometric auth
7. Deep links
8. Haptic on delivery completion

### Defer to v2
- Animated live driver tracking pin on Customer map (needs backend WebSocket/polling infra)
- Chef home screen widget (iOS WidgetKit complexity)
- Scheduled push notifications (needs backend cron)
- Shake-to-report (nice-to-have, low priority)
- App Store review prompt (add after launch once review volume is needed)

---

## Phase-Specific Notes for Roadmap

| Phase Topic | Mobile-Native Feature Concern | Recommendation |
|-------------|-------------------------------|----------------|
| Expo setup / infra | Push notification token registration must be wired up in the shared infra phase, before any app-specific work | Add Expo Notifications setup + `PUT /profile/device-token` call to login flow in Phase 1 |
| Auth screens | Biometric auth requires `expo-local-authentication` + `expo-secure-store` as dependencies from day one | Add in auth phase, not as a separate phase |
| Camera / uploads | All three apps need it (menu photos, doc uploads, proof of delivery) — same `expo-image-picker` component | Build a shared `<ImageUploadPicker>` component in Phase 1 / infra |
| GPS / maps | Driver GPS needs background location entitlement declared in `app.json` from the start — cannot add later without a new EAS build | Add to EAS build config in infra phase |
| Social login | Apple Sign-In requires `com.apple.developer.applesignin` entitlement — needs Apple Developer account configured before first TestFlight | Block social login implementation on Apple entitlement setup |
| Deep links | URL scheme and associated domains require EAS build config + Apple/Google verification before any feature uses them | Configure in infra phase |

---

## Confidence Assessment

| Area | Confidence | Basis |
|------|------------|-------|
| Push notification infrastructure | HIGH | Backend FCM service confirmed in codebase (`services/push.go`). FCM token registration endpoint exists (`PUT /profile/device-token`). Expo Notifications is the standard Expo integration path. |
| Biometric auth pattern | HIGH | `expo-local-authentication` + `expo-secure-store` is the canonical Expo pattern. No conflicting signals. |
| Driver background GPS | HIGH | Background location for delivery tracking is universal in driver apps. `expo-location` background task support is well-documented. |
| Native nav handoff (driver) | HIGH | `Linking.openURL` with map app deep links is a simple, proven pattern used by DoorDash/Swiggy driver apps. |
| Live animated driver tracking | MEDIUM | Requires backend changes (real-time location broadcasting) not currently present. Complexity is non-trivial. |
| Apple Sign-In requirement | HIGH | Apple App Store Review Guidelines §4.8 requires apps with third-party social login to also offer Sign in with Apple. This is non-negotiable. |
| Photo proof of delivery | MEDIUM | Common industry feature (Swiggy, Zomato, Amazon delivery). API changes needed (new field on delivery model). Backend is straightforward but not yet present. |

---

*Research basis: Codebase analysis (existing API handlers, push infrastructure, auth patterns) + established food delivery app patterns (Swiggy, Zomato, Uber Eats, DoorDash, Dunzo). No web sources available during this research session.*
