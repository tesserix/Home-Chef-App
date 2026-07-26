# For legal review: accountability and enforcement clauses

**Date:** 26 July 2026
**Prepared by:** engineering, at the founder's request
**Status:** DRAFT. Not reviewed by a lawyer. Do not ship without sign-off.

---

## What was asked for

Strengthen the customer terms and the chef agreement so that:

1. Chefs who sell unauthorised or unsafe food face serious consequences.
2. Customers who make fake refund claims or damage the platform's reputation face serious consequences.
3. The same applies to vendors.
4. The company and the apps are protected, with responsibility resting on customers and chefs.

## What was written

Plain English, short sentences, no legalese, so it can be read quickly and rewritten by counsel.

### Customer terms — new sections

| Section | Covers |
|---|---|
| 8. Acceptable use (rewritten) | Broken out into a list; adds the consequence pointer |
| 9. Refund fraud and payment disputes | Fake non-delivery claims, fake quality claims, borrowed photos, serial claiming, chargeback abuse. Sets out how we investigate, and the consequences: refusal, account closure, recovery as a debt, referral to authorities |
| 10. False statements that damage a chef or Fe3dr | Narrow: knowingly false statements of **fact**. Explicitly preserves honest negative reviews |
| 11. Information you give us | Allergies, delivery details, who collects |
| 12. Costs you cause us | Customer-side indemnity, carved back for our own fault and for what the law disallows |

### Chef agreement — new sections

| Section | Covers |
|---|---|
| 5. FSSAI (extended) | Adds the duty to stop selling and notify the same day if a licence lapses |
| 6. What you may sell, and what you may not | Positive list plus nine prohibitions: lapsed licence, unregistered kitchen, reselling restaurant food as home cooking, spoiled or out-of-date food, unsafe cold chain, adulterants, false veg/vegan/jain/halal/allergen claims, cooking while ill. Names these "serious breaches" |
| 7. What happens after a serious breach | Immediate delisting, payout hold, order cancellation, termination; then the off-platform consequences: mandatory reporting to FSSAI, prosecution exposure under the FSS Act 2006, claims lying against the chef, cost recovery. Carve-out for self-reported honest mistakes |
| 12. Honest dealing, ratings and reputation | Fake orders, self-reviews, review-buying, pressuring reviewers, off-platform diversion, knowingly false statements. Explicitly preserves honest criticism of Fe3dr |
| 13. Liability (extended) | Chef indemnity, and a statement that food-safety duties cannot be transferred to Fe3dr by contract |

Mirrored across four surfaces so they cannot drift:

- `apps/mobile-customer/app/terms.tsx`
- `apps/mobile-vendor/app/chef-agreement.tsx`
- `apps/web-landing/app/terms/page.tsx`
- `apps/web-landing/app/vendor-terms/page.tsx`

---

## Please look hard at these. Engineering could not resolve them.

### 1. "All responsibility with customers and chefs" is not achievable as stated

This was the explicit ask, and the drafting pushes as far toward it as looked defensible — but it cannot go the whole way, and pretending otherwise would give the business false comfort.

- The **Consumer Protection (E-Commerce) Rules, 2020** place duties directly on a marketplace entity — grievance officer, no unfair trade practice, liability where the platform gives an express warranty or fails its own duties. These do not disappear because the terms say the chef is responsible.
- **FSSAI** treats an e-commerce food business operator as an FBO in its own right. The platform is expected to be licensed and to ensure listed FBOs are licensed. Our own compliance duty is not delegable.
- Under the **Consumer Protection Act, 2019**, terms that are unfair to a consumer can be struck down regardless of what was signed.

The clauses are therefore written to put responsibility on the party genuinely at fault, not to make Fe3dr immune. **Counsel should tell the founder plainly how far this can actually go**, because the current expectation is broader than the law appears to allow.

### 2. The anti-reputation clauses are deliberately narrow — please confirm the scope

Both the customer and chef versions are limited to **statements of fact known to be untrue**, and both say in terms that honest criticism is protected.

That was a deliberate choice. A clause penalising a customer for a truthful bad review would likely be an unfair term, could itself be an unfair trade practice, and would read very badly if it surfaced publicly. If the business wants this wider, counsel should be the one to say whether that is possible.

### 3. Payout withholding vs the RBI Payment Aggregator framework

Chef agreement §7 lets us hold pending payouts during a serious-breach investigation. Settlement timelines under the PA framework are prescriptive. **Is an indefinite investigative hold compatible with them, and what is the maximum defensible hold period?** The clause currently says only "no longer than the investigation needs", which is probably too vague.

### 4. Consumer-side indemnity (customer terms §12)

Indemnities given by consumers are frequently unenforceable. It is carved back for our own fault and for what the law disallows. **Is it worth keeping at all, or does an unenforceable clause do more harm than good in an App Review or a consumer complaint?**

### 5. Mandatory reporting and data sharing

Chef agreement §7 commits us to reporting suspected food-safety offences to FSSAI and handing over registration details, kitchen address and related records.

- Confirm the reporting obligation is stated correctly.
- Confirm this disclosure is covered by the privacy policy and lawful under the **DPDP Act, 2023**. The privacy policy needs to say we disclose to regulators — please check the wording lines up.

### 6. The operating-entity question, now more urgent

The terms name **Tesserix Pty Ltd (ACN 694 070 865), New South Wales, Australia** as the operator, and the governing-law clause points to NSW courts.

The new clauses lean heavily on **Indian** law — FSS Act 2006, CPA 2019, DPDP 2023, RBI PA framework. Payments settle through Razorpay in India; chefs hold Indian FSSAI registrations; customers are Indian consumers.

**An Australian entity relying on Indian statutory consequences, under NSW governing law, needs reconciling before launch.** This also affects who is named as the seller on the App Store and Play listings.

### 7. Age

Terms require users to be 18+. The apps are rated 4+ / Everyone. That is defensible (a contractual limit, not a content rating), but please confirm you are comfortable.

---

## Not changed

- Privacy policy, refund policy, EULA — untouched. If §5 above lands, privacy needs a matching edit.
- No enforcement mechanism was built for these clauses beyond what already exists (report/block, admin triage, payout holds, account suspension). The terms describe what we *may* do, not what is automated.

## Files

```
apps/mobile-customer/app/terms.tsx
apps/mobile-vendor/app/chef-agreement.tsx
apps/web-landing/app/terms/page.tsx
apps/web-landing/app/vendor-terms/page.tsx
apps/web-landing/lib/site.ts          # LEGAL_LAST_UPDATED -> 26 July 2026
```

All four typecheck. The date bump in `site.ts` is shared, so **every** landing legal page now shows 26 July 2026 — including privacy, refund, EULA and account-deletion, whose text did not change. Counsel may prefer per-document dates; say so and engineering will split them.
