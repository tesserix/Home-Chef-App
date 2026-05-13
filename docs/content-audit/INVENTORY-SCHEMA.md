# Inventory Row Schema

Every row in `CONTENT-INVENTORY.md` follows this shape.

## Columns

| Column | Type | Description | Example |
|---|---|---|---|
| `surface_id` | string | Stable ID across phases. Format: `<app>-<category>-<slug>` | `web-marketing-landing-hero` |
| `app` | enum | One of: `web`, `vendor-portal`, `delivery-portal`, `admin-portal`, `mobile-customer`, `mobile-vendor`, `mobile-delivery`, `api` | `web` |
| `category` | enum | One of the 9 buckets (see below) | `marketing` |
| `route_or_file` | string | Route path or source-file path (with line range when relevant) | `apps/web/src/app/routes/HomePage.tsx:42-78` |
| `audience` | enum | Primary persona seeing this surface | `customer` |
| `word_count` | int | Approx word count of visible copy | 47 |
| `current_text_excerpt` | string | First 200 chars of visible text (for grep/identification) | `"Discover home chefs near you..."` |
| `shared_component_origin` | string \| null | Upstream source if this string comes from a shared package | `@tesserix/web Button` or `null` |
| `last_edited` | ISO date \| null | Git blame date of last change to this string | `2026-04-22` |
| `notes` | string | Free-text: special handling, known issues, related surfaces | `"Hero CTA also embedded in mobile-customer onboarding"` |

## Categories (9 buckets)

| Category | Scope |
|---|---|
| `legal` | T&C, Privacy, Refund/Cancellation, Cookie, Acceptable Use, Chef Agreement, Driver Agreement, Allergen/Food-safety disclosures |
| `marketing` | Landing/home, About, How-it-works, For-Chefs, For-Drivers, Pricing, FAQ |
| `auth-onboarding` | Sign up, Login, MFA, Password reset, OAuth consent, Chef onboarding wizard, Driver verification, Email-OTP |
| `core-ux` | In-app verbs and labels: browse, search, cart, checkout, order tracking, menu builder, order management, delivery navigation |
| `errors-empty` | Validation errors, system errors (4xx/5xx), empty states, offline states, loading states |
| `transactional` | Backend-generated: email templates, push notifications, SMS, in-app toasts/banners |
| `microcopy` | Tooltips, helper text, placeholders, success confirmations, button labels, modal subtitles |
| `help` | FAQ articles, contact, support flows, in-app hints, "learn more" links |
| `seo-meta` | `<title>`, `<meta description>`, OG tags, structured data, sitemap copy |

## Audience values

`customer`, `chef`, `driver`, `admin`, `multi` (if surface is shared across personas)

## Surface ID conventions

- Lowercase, kebab-case
- Prefix with app shortcode: `web-`, `vp-`, `dp-`, `ap-`, `mc-`, `mv-`, `md-`, `api-`
- Followed by category: `legal-`, `mkt-`, `auth-`, `ux-`, `err-`, `tx-`, `mc-`, `help-`, `seo-`
- Then a slug identifying the surface

Examples:
- `web-mkt-landing-hero`
- `vp-auth-chef-onboarding-step-3`
- `api-tx-order-confirmed-email`
- `mc-err-network-offline-banner`

## What is NOT in the inventory

- Server log strings
- Internal admin debug strings
- Code comments
- Alt-text on decorative icons (covered by prior a11y sweep)
- DevTools strings
- Test fixture strings
