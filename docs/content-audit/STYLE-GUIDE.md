# Home Chef Style Guide

Voice and content conventions for every user-facing string. Aligned with `.impeccable.md` (visual design source of truth). The audit measures every surface against this guide.

## 1. Voice Principles

From `.impeccable.md`: **Confident · Appetizing · Quietly modern.**

### Five rules

**Rule 1 — Confident, not loud.** No urgency tricks. No exclamation marks except in genuine celebration (≤1 per page).

- ✅ "Order placed. Your chef is preparing it now."
- ❌ "🎉 ORDER PLACED! Get ready for AMAZING food! 🍽️"

**Rule 2 — Plain English over jargon.** Choose the shorter, more common word when it carries the same meaning.

- ✅ "We use this to send you order updates."
- ❌ "Pursuant to our communication policy, we utilize this data..."

**Rule 3 — Photo-forward, chrome-light.** UI chrome shrinks; food and faces carry the brand. Copy supports the photo, doesn't compete with it.

- ✅ Hero: "Tonight's dinner, from a kitchen near you." (over photo)
- ❌ Hero: "DISCOVER THE BEST HOME-COOKED MEALS IN YOUR CITY WITH OUR REVOLUTIONARY PLATFORM" (overrides photo)

**Rule 4 — One accent per surface.** One CTA in herb green. One emphasized word per headline.

- ✅ "Order from **home chefs**, not restaurants."
- ❌ "Order from **home chefs**, not **restaurants**, on the **best** platform."

**Rule 5 — Restraint over urgency.** No countdown timers, no "Only 2 left!", no FOMO patterns. Trust the user.

- ✅ "Available today" / "Available tomorrow"
- ❌ "🔥 Selling fast! Order now before it's gone!"

## 2. Persona Tone Matrix

Same voice principles, dialed per audience.

| Persona | Tone | Sentence length | Verb mood | Example |
|---|---|---|---|---|
| Customer | Warmer, sensory | Conversational (10-18 words) | Suggestive ("Discover", "Try") | "Discover chefs cooking in your neighborhood." |
| Chef / Vendor | Functional, time-aware | Crisp (5-12 words) | Operational ("Mark ready", "Pause menu") | "5 orders queued. Earliest pickup: 7:15 PM." |
| Driver | Glanceable, imperative | Telegraphic (≤4 words where possible) | Imperative ("Pick up", "Confirm") | "Pick up at 7 PM. 1.2 km." |
| Admin | Neutral operator | Precise (any length, no fluff) | Direct ("Approve", "Suspend", "Audit") | "Suspend vendor — requires reason code." |

## 3. Vocabulary List

Preferred terms ✅ / banned variants ❌. When a banned variant is in production code, file a finding.

### Identity & roles

- `Sign in` ✅ / `Log in` ❌ / `Login` ❌ (always two words, "sign")
- `Sign out` ✅ / `Log out` ❌
- `Sign up` ✅ (verb) / `Signup` ❌
- `Home chef` ✅ (customer-facing) / `Cook` ❌ / `Vendor` ❌ / `Seller` ❌
- `Chef` ✅ (chef-facing, in their portal) — they self-identify as chefs
- `Driver` ✅ (driver-facing) / `Delivery partner` (customer-facing only)
- `Customer` ✅ (internal/chef-facing) / `User` ❌ (too generic)

### Order verbs

- `Place an order` ✅ / `Purchase` ❌ / `Buy` ❌ (food, not commerce)
- `Order` (noun) ✅ / `Transaction` ❌ (customer side)
- `Pickup` (noun) ✅ / `Pick-up` ❌ / `Pick up` ❌ (when noun)
- `Pick up` (verb, two words) ✅
- `Delivery` ✅ / `Shipping` ❌

### Money & status

- `₹120` ✅ (no space, currency before amount) / `120 ₹` ❌ / `Rs. 120` ❌ / `INR 120` ❌
- `Total` ✅ / `Grand Total` ❌ (over-emphasized)
- `Subtotal` ✅ / `Sub-total` ❌
- `Tax` ✅ / `GST` ✅ when GST line specifically (legally required)
- `Free delivery` ✅ / `Delivery: Free` ❌
- `Paid` ✅ / `Payment successful` ✅ (UI verb forms — pick one per surface)

### Time & dates

- Today: `Today, 7:30 PM` ✅
- Tomorrow: `Tomorrow, 7:30 PM` ✅
- Future: `Fri, 14 May, 7:30 PM` ✅ (en-IN locale, 12-hour clock customer-facing)
- Past relative under 24h: `35 minutes ago` ✅
- Past >24h: `Yesterday` / `2 days ago` / absolute date thereafter

### Banned brand-drift terms

- `Artisanal` ❌
- `Handcrafted with love` ❌
- `Curated` ❌ (overused; use "selected" or just describe directly)
- `Foodie` ❌
- `Hurry!` / `Limited time!` / `Only X left!` ❌ (urgency tricks)
- `Click here` ❌ (use descriptive link text)
- `Learn more →` ✅ when paired with topic context, never standalone

## 4. Microcopy Patterns (formulas)

### Buttons

Format: **verb-first, ≤3 words, sentence case**.

| Action | ✅ | ❌ |
|---|---|---|
| Submit order | `Place order` | `PLACE ORDER`, `Click here to order`, `Submit` |
| Save draft | `Save draft` | `Save it for later`, `Save Draft` |
| Confirm pickup | `Confirm pickup` | `Confirm Pick-Up`, `I picked it up` |

### Errors

Format: **What happened → What to do.** Two sentences max. No blame. No "Oops!" "Uh oh!" exclamation patterns.

- ✅ "Card declined. Try a different payment method."
- ✅ "Network lost. Reconnect to keep tracking."
- ❌ "Oops! Something went wrong! 😢 Please try again later."

### Empty states

Format: **Why it's empty → One action.**

- ✅ "No orders yet. Browse chefs near you."
- ✅ "No menu items. Add your first dish."
- ❌ "Looks like nothing here!" / "It's lonely in here..."

### Success toasts

Format: **past tense, ≤6 words, period.**

- ✅ "Order placed."
- ✅ "Menu published."
- ❌ "Yay! Your order is on its way! 🎉"

### Form labels

Format: **noun, sentence case, no colons.** Helper text in muted tone, under the input.

- Label: `Delivery address`
- Helper: `We'll send your driver here.`
- ❌ `DELIVERY ADDRESS:` / `Address*` (required indicator is asterisk on field, label stays clean)

### Modal subtitles

Format: **explain consequence in one sentence.**

- ✅ "Cancelling this order refunds the customer immediately."
- ❌ "Are you sure? This action cannot be undone." (vague, doesn't explain WHAT can't be undone)

## 5. Legal-Page Tone

Plain language even in T&C. Plain language is a legal feature, not a tradeoff.

- **Short sentences** — average 15-20 words, max 25
- **Headings every ~200 words** — never let a wall of text exceed 200 words without a heading
- **"We" / "you"** — never "the Company", "the User", "the Service Provider"
- **Allergen / refund clauses get callout boxes** — not buried in paragraphs
- **Defined terms in bold first-use** — `**Order**`, `**Pickup Window**`, `**Refund Period**`
- **Active voice** — "We refund within 7 days" not "Refunds are processed within 7 days"
- **One idea per paragraph** — split walls of legal prose
- **Plain-language summary callout at top of every legal page** — "Here's what this page covers, in one paragraph"

Example transformation:

❌ Before:
> "The Company shall not be liable for any damages, losses, or expenses arising from the consumption of food items procured through the Service, including but not limited to allergic reactions, foodborne illness, or any other adverse health consequences, except to the extent such liability cannot be excluded under applicable law."

✅ After:
> "**Allergens.** Home chefs label allergens on every dish. If you have allergies, check the label before ordering and contact the chef with questions. We aren't responsible for allergic reactions unless we're legally required to be (e.g., we hid allergen info from you)."

## 6. Numerals & Formatting

- **Tabular figures everywhere money / IDs / ETAs appear** — `font-feature-settings: "tnum"`
- **Currency** — `₹120` (no space, symbol first) — en-IN locale primary
- **Time** — 24-hour internal-facing (admin, vendor portal scheduling), 12-hour customer-facing
- **Relative time threshold** — under 24h relative ("in 35 min", "2 hours ago"); 24h+ absolute ("Tomorrow, 7:30 PM", "Fri, 14 May")
- **Phone numbers** — `+91 98765 43210` (en-IN format)
- **Order IDs** — `#HC-2026-00001234` — prefix, year, zero-padded
- **Distance** — kilometres customer-facing (`1.2 km`); metres only when <1 km (`850 m`)
- **Plurals** — always: `1 order` / `0 orders` / `2 orders` (never "order(s)")
- **Percentages** — no space: `15%` (never `15 %`)

## 7. Internationalization Readiness

Every rule above survives translation IF:

- Button-length rules allow 30% slack (Hindi/Tamil run longer)
- Sentence templates don't depend on English word order (avoid concatenation like `"Order #" + id + " by " + chef`)
- Plurals use ICU MessageFormat-compatible patterns (or whatever i18n lib is chosen)
- Date formatting goes through `Intl.DateTimeFormat`, never hand-rolled
- Currency formatting goes through `Intl.NumberFormat({ style: 'currency', currency: 'INR' })`

Translation work is **out of scope for this audit**, but every string the audit flags for rewrite must be translation-ready when rewritten.
