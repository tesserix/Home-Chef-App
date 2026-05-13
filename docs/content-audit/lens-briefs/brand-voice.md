# Brand Voice Lens Brief

You are auditing for **cross-surface and cross-app voice consistency** against `.impeccable.md` + `STYLE-GUIDE.md`. Your lens is: does this whole product sound like one brand?

## Inputs you will receive
- A slice of `CONTENT-INVENTORY.md` (the category's rows ACROSS all 7 apps + API)
- `STYLE-GUIDE.md`
- `.impeccable.md` (source of truth for voice principles)

## What to flag

### Cross-app drift
- Same concept named differently across apps (e.g., "Cart" in web, "Bag" in mobile)
- Same action labeled differently (e.g., "Place order" vs "Confirm order" vs "Submit")
- Different greeting tones in welcome emails per persona (when they should share voice DNA)

### Persona-tone violations
- Customer-facing copy that's too operational (sounds like vendor portal)
- Vendor-facing copy that's marketing-y (sounds like customer landing)
- Driver-facing copy that's verbose (>4 words where ≤4 would do)

### Brand-personality drift
- "Confident · Appetizing · Quietly modern" — flag copy that's the opposite:
  - **Loud** ("HURRY!", "AMAZING DEAL", "🎉🎉🎉")
  - **Bland** ("Welcome to our platform", "Your order has been received")
  - **Trend-chasing** ("vibes", "no cap", emoji-as-bullet patterns from 2022 DTC)
  - **Faux-artisanal** ("handcrafted with love", "lovingly prepared", "homestyle goodness")

### Anti-references (from `.impeccable.md`)
- Terracotta-era legacy copy that survived the migration ("artisanal", "cream-and-terra", "warm-cozy", "Playfair Display"-era headlines)
- Swiggy/Zomato-style red-saturated urgency ("Order now!", "Last few left!")
- AI-slop maximalism (cyan-on-dark dark-mode headlines, gradient promises)
- Generic SaaS dashboard ("Hero metrics", "Onboarding journey", "Stakeholder")

### Voice consistency checks
- Pronouns: "we"/"you" enforced everywhere (not "the Company"/"the User"/"the Driver Partner")
- Punctuation: exclamation-mark budget (1 per page customer-facing, 0 vendor/driver)
- Emoji: customer-facing OK in occasional moments; vendor/driver/admin = 0 emoji
- Capitalization: sentence case for buttons and labels; Title Case banned outside of proper nouns

## Output format

```yaml
- finding_id: BV-<NNN>
  surface_id: <from inventory>
  lens: brand-voice
  severity: P0|P1|P2|P3
  issue: "<one-line summary>"
  evidence_excerpt: "<actual text>"
  related_surfaces: ["<other surface_ids showing the same drift>"]
  recommendation: "<rewrite or rule>"
  depends_on: null
```

## Severity guide (Brand Voice lens)

- **P0** — Cross-surface contradiction that breaks brand trust (e.g., "Cart" on web, "Bag" on mobile — customer thinks it's a different product)
- **P1** — Personality drift on entry surfaces (landing, signup, welcome email, push notifications)
- **P2** — Inconsistent persona tone on secondary surfaces
- **P3** — Punctuation / capitalization drift in deep surfaces

## Out of scope for Brand Voice lens
- Within-surface microcopy formulas (TW lens — you focus on cross-surface)
- Legal compliance (Legal lens)
- Conversion psychology (BA lens)
- Visual brand identity (`.impeccable.md` and design system handle this)
