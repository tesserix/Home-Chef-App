# Mobile Customer Inventory

Generated: 2026-05-13
App: apps/mobile-customer/
Total rows: 95

| surface_id | app | category | route_or_file | audience | word_count | current_text_excerpt | shared_component_origin | last_edited | notes |
|---|---|---|---|---|---|---|---|---|---|
| mc-auth-login-title | mobile-customer | auth-onboarding | app/(auth)/login.tsx | customer | 2 | "Welcome back" | LoginScreen prop (origin: @homechef/mobile-shared/screens) | 2026-04-06 | Title prop passed to shared LoginScreen; remaining login UI strings live in @homechef/mobile-shared |
| mc-auth-register-host | mobile-customer | auth-onboarding | app/(auth)/register.tsx | customer | 0 | (shared RegisterScreen — no local strings) | RegisterScreen (origin: @homechef/mobile-shared/screens) | 2026-04-06 | All copy in shared screen; flag for cross-app voice audit |
| mc-perm-faceid | mobile-customer | microcopy | app.json (NSFaceIDUsageDescription) | customer | 7 | "Use Face ID to log in quickly" | — | 2026-04-06 | iOS permission prompt — appears in native system dialog |
| mc-perm-camera | mobile-customer | microcopy | app.json (NSCameraUsageDescription) | customer | 4 | "Used to take photos" | — | 2026-04-06 | iOS camera permission prompt; vague — does not explain why |
| mc-perm-location | mobile-customer | microcopy | app.json (NSLocationWhenInUseUsageDescription) | customer | 11 | "Used to show your location on the delivery tracking map" | — | 2026-04-06 | iOS location prompt for delivery tracking |
| mc-onb-step1-progress | mobile-customer | auth-onboarding | app/(onboarding)/user-info.tsx | customer | 4 | "Step 1 of 3" | — | 2026-05-12 | Progress label |
| mc-onb-step1-title | mobile-customer | auth-onboarding | app/(onboarding)/user-info.tsx | customer | 4 | "Tell us about yourself" | — | 2026-05-12 | Section title |
| mc-onb-step1-subtitle | mobile-customer | auth-onboarding | app/(onboarding)/user-info.tsx | customer | 10 | "We need a few details to set up your account." | — | 2026-05-12 | Section subtitle |
| mc-onb-step1-labels | mobile-customer | auth-onboarding | app/(onboarding)/user-info.tsx | customer | 6 | "First Name / Last Name / Phone Number" | — | 2026-05-12 | Three field labels |
| mc-onb-step1-placeholders | mobile-customer | auth-onboarding | app/(onboarding)/user-info.tsx | customer | 12 | "Enter your first name / Enter your last name / 10-digit mobile number" | — | 2026-05-12 | Input placeholders |
| mc-onb-step1-errors | mobile-customer | errors-empty | app/(onboarding)/user-info.tsx | customer | 21 | "First name must be at least 2 characters / Last name must be at least 2 characters / Enter a valid 10-digit Indian mobile number" | — | 2026-05-12 | Zod validation messages |
| mc-onb-step1-cta | mobile-customer | core-ux | app/(onboarding)/user-info.tsx | customer | 1 | "Continue" | — | 2026-05-12 | Primary CTA |
| mc-onb-step2-progress | mobile-customer | auth-onboarding | app/(onboarding)/address.tsx | customer | 4 | "Step 2 of 3" | — | 2026-05-12 | Progress label |
| mc-onb-step2-title | mobile-customer | auth-onboarding | app/(onboarding)/address.tsx | customer | 3 | "Your delivery address" | — | 2026-05-12 | Section title |
| mc-onb-step2-subtitle | mobile-customer | auth-onboarding | app/(onboarding)/address.tsx | customer | 6 | "Where should we deliver your orders?" | — | 2026-05-12 | Section subtitle |
| mc-onb-step2-labels | mobile-customer | auth-onboarding | app/(onboarding)/address.tsx | customer | 4 | "Address / City / State / Pincode" | — | 2026-05-12 | Field labels |
| mc-onb-step2-placeholders | mobile-customer | auth-onboarding | app/(onboarding)/address.tsx | customer | 13 | "House no., street, area / Enter your city / Enter your state / 6-digit pincode" | — | 2026-05-12 | Input placeholders |
| mc-onb-step2-errors | mobile-customer | errors-empty | app/(onboarding)/address.tsx | customer | 21 | "Address must be at least 5 characters / City is required / State is required / Enter a valid 6-digit pincode" | — | 2026-05-12 | Zod validation messages |
| mc-onb-step3-progress | mobile-customer | auth-onboarding | app/(onboarding)/preferences.tsx | customer | 4 | "Step 3 of 3" | — | 2026-05-12 | Progress label |
| mc-onb-step3-title | mobile-customer | auth-onboarding | app/(onboarding)/preferences.tsx | customer | 6 | "What do you love to eat?" | — | 2026-05-12 | Section title |
| mc-onb-step3-subtitle | mobile-customer | auth-onboarding | app/(onboarding)/preferences.tsx | customer | 9 | "Select your favourite cuisines to get personalised recommendations." | — | 2026-05-12 | Uses British "favourite/personalised"; web app may use American — flag for spelling consistency |
| mc-onb-step3-cuisines | mobile-customer | core-ux | app/(onboarding)/preferences.tsx | customer | 11 | "North Indian, South Indian, Chinese, Continental, Italian, Healthy, Street Food, Desserts" | — | 2026-05-12 | Cuisine chip labels |
| mc-onb-step3-cta | mobile-customer | core-ux | app/(onboarding)/preferences.tsx | customer | 2 | "Finish Setup" | — | 2026-05-12 | Final onboarding CTA |
| mc-onb-step3-error | mobile-customer | errors-empty | app/(onboarding)/preferences.tsx | customer | 7 | "Setup failed / Something went wrong. Please try again." | — | 2026-05-12 | Alert.alert error fallback |
| mc-tabs-labels | mobile-customer | microcopy | app/(tabs)/_layout.tsx | customer | 4 | "Home / Orders / Saved / Profile" | — | 2026-05-12 | Bottom tab labels |
| mc-home-search-placeholder | mobile-customer | microcopy | app/(tabs)/index.tsx | customer | 3 | "Search chefs, cuisines..." | — | 2026-05-12 | Search input placeholder |
| mc-home-search-a11y | mobile-customer | microcopy | app/(tabs)/index.tsx | customer | 2 | "Search chefs" | — | 2026-05-12 | accessibilityLabel |
| mc-home-cuisine-filters | mobile-customer | core-ux | app/(tabs)/index.tsx | customer | 10 | "All, North Indian, South Indian, Chinese, Continental, Italian, Healthy" | — | 2026-05-12 | Cuisine filter chips |
| mc-home-open-now | mobile-customer | core-ux | app/(tabs)/index.tsx | customer | 2 | "Open Now" | — | 2026-05-12 | Toggle label |
| mc-home-sort-options | mobile-customer | core-ux | app/(tabs)/index.tsx | customer | 5 | "Recommended, Top Rated, Newest, Price" | — | 2026-05-12 | Sort options; two map to same value 'rating' — likely UX bug |
| mc-home-quick-links | mobile-customer | core-ux | app/(tabs)/index.tsx | customer | 4 | "Social Feed / Catering" | — | 2026-05-12 | Quick-access pills |
| mc-home-empty | mobile-customer | errors-empty | app/(tabs)/index.tsx | customer | 8 | "No chefs found / Try adjusting your filters" | — | 2026-05-12 | Empty state |
| mc-orders-title | mobile-customer | core-ux | app/(tabs)/orders.tsx | customer | 2 | "My Orders" | — | 2026-05-12 | Screen title |
| mc-orders-filters | mobile-customer | core-ux | app/(tabs)/orders.tsx | customer | 4 | "All, Active, Delivered, Cancelled" | — | 2026-05-12 | Status filter chips |
| mc-orders-empty | mobile-customer | errors-empty | app/(tabs)/orders.tsx | customer | 9 | "No orders yet / Browse chefs to place your first order!" | — | 2026-05-12 | Empty state with 🍽️ emoji; flag emoji-in-copy decision |
| mc-favorites-title | mobile-customer | core-ux | app/(tabs)/favorites.tsx | customer | 2 | "Saved Chefs" | — | 2026-05-12 | Screen title; tab label is "Saved" but page title is "Saved Chefs" — minor inconsistency |
| mc-favorites-subtitle | mobile-customer | microcopy | app/(tabs)/favorites.tsx | customer | 3 | "{n}/{max} saved" | — | 2026-05-12 | Dynamic count display |
| mc-favorites-a11y-view | mobile-customer | microcopy | app/(tabs)/favorites.tsx | customer | 3 | "View {chef.name}" | — | 2026-05-12 | Card accessibilityLabel |
| mc-favorites-a11y-remove | mobile-customer | microcopy | app/(tabs)/favorites.tsx | customer | 5 | "Remove {chef.name} from favorites" | — | 2026-05-12 | Heart-button a11y |
| mc-favorites-open-closed | mobile-customer | microcopy | app/(tabs)/favorites.tsx | customer | 2 | "Open / Closed" | — | 2026-05-12 | Chef status badge |
| mc-favorites-error | mobile-customer | errors-empty | app/(tabs)/favorites.tsx | customer | 8 | "Could not remove from favorites. Please try again." | — | 2026-05-12 | Alert.alert error |
| mc-favorites-empty | mobile-customer | errors-empty | app/(tabs)/favorites.tsx | customer | 11 | "No saved chefs yet / Tap the heart on any chef to save them!" | — | 2026-05-12 | Empty state with heart emoji |
| mc-profile-sections | mobile-customer | core-ux | app/(tabs)/profile.tsx | customer | 7 | "Personal Info / Food Preferences / More" | — | 2026-05-12 | Section titles |
| mc-profile-field-labels | mobile-customer | core-ux | app/(tabs)/profile.tsx | customer | 4 | "First Name / Last Name / Phone" | — | 2026-05-12 | Form field labels |
| mc-profile-placeholders | mobile-customer | microcopy | app/(tabs)/profile.tsx | customer | 5 | "First name / Last name / +91 9876543210" | — | 2026-05-12 | Input placeholders |
| mc-profile-errors | mobile-customer | errors-empty | app/(tabs)/profile.tsx | customer | 9 | "First name is required / Last name is required / Invalid phone number" | — | 2026-05-12 | Zod validation messages |
| mc-profile-cuisines | mobile-customer | core-ux | app/(tabs)/profile.tsx | customer | 11 | "North Indian, South Indian, Chinese, Continental, Italian, Healthy, Desserts, Street Food" | — | 2026-05-12 | Cuisine chip options (note: order differs from onboarding step3 — flag for drift) |
| mc-profile-save-ctas | mobile-customer | core-ux | app/(tabs)/profile.tsx | customer | 4 | "Save Changes / Save Preferences" | — | 2026-05-12 | Save buttons |
| mc-profile-save-success | mobile-customer | transactional | app/(tabs)/profile.tsx | customer | 9 | "Saved / Profile updated successfully. / Cuisine preferences updated." | — | 2026-05-12 | Alert.alert success toasts |
| mc-profile-save-error | mobile-customer | errors-empty | app/(tabs)/profile.tsx | customer | 11 | "Error / Could not update profile. Please try again. / Could not save preferences." | — | 2026-05-12 | Alert.alert errors |
| mc-profile-more-rows | mobile-customer | core-ux | app/(tabs)/profile.tsx | customer | 3 | "Social Feed / Catering" | — | 2026-05-12 | More-section rows with emoji icons (📱/🍽️) |
| mc-profile-logout-confirm | mobile-customer | transactional | app/(tabs)/profile.tsx | customer | 14 | "Log out / Are you sure you want to log out? / Cancel / Log Out" | — | 2026-05-12 | Alert.alert confirmation; title-case inconsistency "Log out" vs button "Log Out" |
| mc-profile-logout-button | mobile-customer | core-ux | app/(tabs)/profile.tsx | customer | 2 | "Log Out" | — | 2026-05-12 | Bottom destructive button |
| mc-chef-error-load | mobile-customer | errors-empty | app/chef/[id].tsx | customer | 6 | "Failed to load chef details. Please try again." | — | 2026-05-12 | Error state |
| mc-chef-open-closed | mobile-customer | microcopy | app/chef/[id].tsx | customer | 2 | "Open / Closed" | — | 2026-05-12 | Status pill |
| mc-chef-delivery-meta | mobile-customer | microcopy | app/chef/[id].tsx | customer | 4 | "Free delivery / ₹{n} delivery" | — | 2026-05-12 | Delivery fee display |
| mc-chef-empty-category | mobile-customer | errors-empty | app/chef/[id].tsx | customer | 5 | "No items in this category" | — | 2026-05-12 | Empty list state |
| mc-chef-a11y-filter | mobile-customer | microcopy | app/chef/[id].tsx | customer | 3 | "Filter by {category}" | — | 2026-05-12 | Category chip a11y |
| mc-menuitem-a11y | mobile-customer | microcopy | components/chef/MenuItemCard.tsx | customer | 4 | "Add {item.name} to cart" | — | 2026-05-12 | Add button a11y |
| mc-menuitem-cross-chef | mobile-customer | transactional | components/chef/MenuItemCard.tsx | customer | 13 | "Replace Cart? / You have items from another chef. Replace cart? / Cancel / Replace" | — | 2026-05-12 | Alert.alert cross-chef warning |
| mc-cartbar-cta | mobile-customer | core-ux | components/cart/CartBar.tsx | customer | 2 | "View Cart" | — | 2026-05-12 | Floating cart button |
| mc-cartbar-a11y | mobile-customer | microcopy | components/cart/CartBar.tsx | customer | 7 | "View cart — {n} items, ₹{total}" | — | 2026-05-12 | Cart bar a11y label |
| mc-cartsheet-title | mobile-customer | core-ux | components/cart/CartSheet.tsx | customer | 2 | "Your Cart" | — | 2026-05-12 | Bottom-sheet header |
| mc-cartsheet-subtotal | mobile-customer | core-ux | components/cart/CartSheet.tsx | customer | 1 | "Subtotal" | — | 2026-05-12 | Label |
| mc-cartsheet-checkout-cta | mobile-customer | core-ux | components/cart/CartSheet.tsx | customer | 3 | "Proceed to Checkout" | — | 2026-05-12 | Primary CTA |
| mc-cartsheet-empty | mobile-customer | errors-empty | components/cart/CartSheet.tsx | customer | 4 | "Your cart is empty" | — | 2026-05-12 | Empty state |
| mc-cartsheet-qty-a11y | mobile-customer | microcopy | components/cart/CartSheet.tsx | customer | 8 | "Decrease quantity / Increase quantity / Remove {item.name} / ₹{price} each" | — | 2026-05-12 | Cart row a11y + per-item price label |
| mc-checkout-header | mobile-customer | core-ux | app/checkout.tsx | customer | 1 | "Checkout" | — | 2026-05-12 | Screen title |
| mc-checkout-section-address | mobile-customer | core-ux | app/checkout.tsx | customer | 2 | "Delivery Address" | — | 2026-05-12 | Section title |
| mc-checkout-default-badge | mobile-customer | microcopy | app/checkout.tsx | customer | 1 | "Default" | — | 2026-05-12 | Default address badge |
| mc-checkout-add-address | mobile-customer | core-ux | app/checkout.tsx | customer | 3 | "Add New Address" | — | 2026-05-12 | Inline expansion CTA |
| mc-checkout-address-placeholders | mobile-customer | microcopy | app/checkout.tsx | customer | 11 | "Address line 1 * / Address line 2 (optional) / City * / State * / Pincode *" | — | 2026-05-12 | Form placeholders |
| mc-checkout-address-errors | mobile-customer | errors-empty | app/checkout.tsx | customer | 13 | "Address line 1 is required / City is required / State is required / Pincode must be 6 digits" | — | 2026-05-12 | Zod errors |
| mc-checkout-save-address | mobile-customer | core-ux | app/checkout.tsx | customer | 2 | "Save Address" | — | 2026-05-12 | Save button |
| mc-checkout-summary-section | mobile-customer | core-ux | app/checkout.tsx | customer | 2 | "Order Summary" | — | 2026-05-12 | Section title |
| mc-checkout-empty-cart | mobile-customer | errors-empty | app/checkout.tsx | customer | 4 | "Your cart is empty." | — | 2026-05-12 | List empty state |
| mc-checkout-totals | mobile-customer | core-ux | app/checkout.tsx | customer | 5 | "Subtotal / Delivery fee / Free / Total" | — | 2026-05-12 | Order totals labels |
| mc-checkout-note | mobile-customer | microcopy | app/checkout.tsx | customer | 9 | "Note to chef (optional) / Any special instructions..." | — | 2026-05-12 | Note field label + placeholder |
| mc-checkout-place-order | mobile-customer | core-ux | app/checkout.tsx | customer | 6 | "Place Order · ₹{total} / Processing..." | — | 2026-05-12 | Sticky-bottom primary CTA |
| mc-checkout-errors | mobile-customer | errors-empty | app/checkout.tsx | customer | 27 | "Order creation failed. Please try again. / Payment was not completed. Please try again. / Payment confirmation timed out. Check your order history to confirm status. / Dismiss" | — | 2026-05-12 | Payment error states; verbose timeout copy |
| mc-payment-confirming | mobile-customer | transactional | app/payment/result.tsx | customer | 2 | "Confirming payment..." | — | 2026-05-12 | Razorpay deep-link callback transient screen |
| mc-order-detail-not-found | mobile-customer | errors-empty | app/order/[id]/index.tsx | customer | 3 | "Order not found / Go Back" | — | 2026-05-12 | Error state + recovery CTA |
| mc-order-detail-status-labels | mobile-customer | core-ux | app/order/[id]/index.tsx | customer | 11 | "Pending / Confirmed / Preparing / Ready for Pickup / On the Way / Delivered / Cancelled" | — | 2026-05-12 | Order status badges; uses "Ready for Pickup" vs OrderCard.tsx "Ready" and OrderTimeline "On the Way" — drift |
| mc-order-detail-eta | mobile-customer | microcopy | app/order/[id]/index.tsx | customer | 2 | "ETA: {time}" | — | 2026-05-12 | Estimated delivery display |
| mc-order-detail-track-cta | mobile-customer | core-ux | app/order/[id]/index.tsx | customer | 2 | "Track Order" | — | 2026-05-12 | Primary CTA for active orders |
| mc-order-detail-sections | mobile-customer | core-ux | app/order/[id]/index.tsx | customer | 6 | "Items / Delivery Address / Price Breakdown" | — | 2026-05-12 | Section titles |
| mc-order-detail-price-rows | mobile-customer | core-ux | app/order/[id]/index.tsx | customer | 5 | "Subtotal / Delivery Fee / Total" | — | 2026-05-12 | Price breakdown rows; capitalization differs from checkout "Delivery fee" — flag |
| mc-order-detail-footer | mobile-customer | microcopy | app/order/[id]/index.tsx | customer | 3 | "Ordered on {date}" | — | 2026-05-12 | Order timestamp footer |
| mc-order-card-meta | mobile-customer | microcopy | components/orders/OrderCard.tsx | customer | 4 | "{n} item(s) • {date}" | — | 2026-05-12 | Order card meta line |
| mc-order-card-status | mobile-customer | core-ux | components/orders/OrderCard.tsx | customer | 7 | "Pending / Confirmed / Preparing / Ready / Picked Up / Delivered / Cancelled" | — | 2026-05-12 | Status labels differ from order detail — "Ready" vs "Ready for Pickup", "Picked Up" vs "On the Way" |
| mc-timeline-steps | mobile-customer | core-ux | components/orders/OrderTimeline.tsx | customer | 5 | "Confirmed / Preparing / On the Way / Delivered" | — | 2026-05-12 | Tracking timeline step labels |
| mc-timeline-eta | mobile-customer | microcopy | components/orders/OrderTimeline.tsx | customer | 3 | "Est. arrival: {time}" | — | 2026-05-12 | ETA prefix |
| mc-deliverymap-markers | mobile-customer | microcopy | components/tracking/DeliveryMap.tsx | customer | 4 | "Delivery Address / Chef Location / Driver" | — | 2026-05-12 | Map marker titles |
| mc-social-title | mobile-customer | core-ux | app/social.tsx | customer | 2 | "Social Feed" | — | 2026-05-12 | Screen title |
| mc-social-empty | mobile-customer | errors-empty | app/social.tsx | customer | 8 | "No posts yet / Chefs will share their latest creations here." | — | 2026-05-12 | Empty state w/ 📸 emoji |
| mc-catering-title | mobile-customer | core-ux | app/catering.tsx | customer | 1 | "Catering" | — | 2026-05-12 | Screen title |
| mc-catering-tabs | mobile-customer | core-ux | app/catering.tsx | customer | 4 | "Request Catering / My Requests" | — | 2026-05-12 | Segmented control labels |
| mc-catering-event-types | mobile-customer | core-ux | app/catering.tsx | customer | 9 | "Wedding, Birthday, Corporate, Anniversary, Festival, House Party, Other" | — | 2026-05-12 | Event type chips |
| mc-catering-form-labels | mobile-customer | core-ux | app/catering.tsx | customer | 18 | "Event Type * / Event Date * (YYYY-MM-DD) / Guest Count * / Budget (₹) / City * / State * / Additional Details" | — | 2026-05-12 | Form labels; asterisks for required + raw date format hint is unfriendly UX |
| mc-catering-placeholders | mobile-customer | microcopy | app/catering.tsx | customer | 14 | "2026-06-15 / 50 / 25000 / Mumbai / Maharashtra / Any specific requirements, dietary restrictions, menu preferences..." | — | 2026-05-12 | Input placeholders |
| mc-catering-errors | mobile-customer | errors-empty | app/catering.tsx | customer | 40 | "Event type is required / Date must be YYYY-MM-DD / Event date must be in the future / Guest count must be a number / At least 1 guest required / Budget must be a number / Budget must be positive / City is required / State is required" | — | 2026-05-12 | Zod validation messages |
| mc-catering-submit-success | mobile-customer | transactional | app/catering.tsx | customer | 7 | "Request Submitted! / Chefs will review and send quotes." | — | 2026-05-12 | Alert.alert success |
| mc-catering-submit-error | mobile-customer | errors-empty | app/catering.tsx | customer | 8 | "Error / Could not submit request. Please try again." | — | 2026-05-12 | Alert.alert error |
| mc-catering-submit-cta | mobile-customer | core-ux | app/catering.tsx | customer | 2 | "Submit Request" | — | 2026-05-12 | Primary CTA |
| mc-catering-status-labels | mobile-customer | core-ux | app/catering.tsx | customer | 5 | "Open / Quoted / Accepted / Completed / Cancelled" | — | 2026-05-12 | Request status pills |
| mc-catering-budget-display | mobile-customer | microcopy | app/catering.tsx | customer | 2 | "Budget: ₹{n}" | — | 2026-05-12 | Budget display in request card |
| mc-catering-view-quote-hint | mobile-customer | microcopy | app/catering.tsx | customer | 4 | "Quotes available — view details" | — | 2026-05-12 | Quote-ready hint |
| mc-catering-empty | mobile-customer | errors-empty | app/catering.tsx | customer | 9 | "No requests yet / Submit a catering request to get quotes from chefs." | — | 2026-05-12 | Empty state with 🍽️ emoji |
