# Retention Feature Backlog — curated (2026-07-23)

Owner ask: the best-possible *missing* functionality that would make customers and chefs come
back again and again. Curated — not exhaustive — and filtered against what the platform
already has (meal-plan subscriptions, group orders, catering, wallet, referral, loyalty
points, tips, reviews, favorites, dish search, dietary/allergen profiles, chef-time
handshake, food-safety badges, admin-mediated chat). Ranked by retention impact ÷ build cost.
Document only — nothing here is in the design-sweep scope.

## 1. One-tap Reorder ("Order it again")
A "Order again" rail on Home (last 3 delivered orders → one tap rebuilds the cart) + a
Reorder button on every past-order detail. The single highest-frequency habit loop in food
apps; all data already exists in order history. **Impact: very high · Effort: S.**

## 2. Favorite-chef "cooking today" push
When a favorited chef opens their kitchen or publishes today's/this week's menu → push
"Amma ka kitchen is cooking today · Butter Chicken ₹120". Home kitchens have narrow windows —
this is the demand-capture moment aggregators can't copy. Push + favorites infra exist;
needs an event on kitchen-open/menu-publish. **Impact: very high · Effort: M.**

## 3. Honest queue + live ETA
Post-accept, show "2 orders ahead of you · ready ~7:40 pm" from the chef's real queue and
prep time, with a push when the ETA shifts >10 min. Converts the scariest part of home-chef
ordering (uncertainty) into trust; uniquely honest vs. aggregator fake-precision ETAs.
**Impact: high · Effort: M.**

## 4. Photo reviews + per-dish ratings
Reviews exist; add photo upload and a dish-level star that surfaces on menu rows
("★ 4.8 · 132 ordered"). Social proof exactly at the decision point; photos feed the
photo-forward design system with real content. **Impact: high · Effort: M.**

## 5. "Notify me when open" waitlist
Closed kitchen → one tap "Tell me when they're open" (push at next kitchen-open). Converts
every dead-end visit into a scheduled return. Pairs with #2's event.
**Impact: high · Effort: S.**

## 6. Chef-level punch-card ("5th meal on Amma")
Per-chef loyalty (chef-funded perk: 5th order → free dessert/₹50 off), replacing abstract
platform points with a relationship to *your* chef. Deepens the platform's core moat —
the human bond with a specific cook. **Impact: high · Effort: M.**

## 7. Festival / occasion pre-order calendar
Seasonal windows (Diwali sweets, Onam sadya, Eid biryani) where chefs list limited pre-order
menus with a deadline + pickup/delivery day. Home chefs' highest-margin, highest-demand
moments; spiky AOV + strong annual retention memory. **Impact: high · Effort: L.**

## 8. Post-delivery moment: rate → tip → "same time next week?"
One sheet after confirm-received combining photo review, tip, and a one-tap "repeat this
order next Thursday" (creates a scheduled order via the existing handshake). Captures intent
at peak satisfaction. **Impact: medium-high · Effort: S-M.**

## 9. Household profiles
Multiple dietary profiles per account ("Mom: Jain", "Kids: no nuts") applied per-order.
Tiffin is a family product; ordering for the household is the real job-to-be-done.
**Impact: medium · Effort: M.**

## 10. Chef story on the profile ("Meet your chef")
A short bio card — who cooks, since when, signature dish, kitchen photo strip. Zero backend
beyond fields that mostly exist; directly serves the brand's trust goal. Candidate to fold
into a future design wave. **Impact: medium · Effort: S.**

**Recommended first wave (next quarter): #1, #2, #5, #8** — all S/M effort, all compounding
the same loop: *notice → order → delight → schedule the next one.*
