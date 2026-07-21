# PLAN — #462 payout medium cluster (scoped)

7-item audit cluster. This PR takes the two bounded, high-value correctness items; the rest are split out (see bottom). Epic #403, flag-gated.

## In scope

### Item 1 — GST split rounding (earnings.go, money-math)
`ComputeOrderEarnings` computes `halfGST = Round2(RateGST/2 * commission)` and sets `cgst = sgst = halfGST`. For odd-paise commission, `cgst + sgst` (= `2·Round2(0.09·commission)`) can be ±1 paise off the true GST on commission (`Round2(0.18·commission)`) — the CGST+SGST split doesn't reconcile to the total GST liability.

**Fix:** compute the full GST once and split with the remainder on SGST so the halves ALWAYS sum to the full GST:
```
fullGST := Round2(RateGST * commission)
// intra-state:
cgst = Round2(fullGST / 2)
sgst = Round2(fullGST - cgst)   // remainder → cgst + sgst == fullGST exactly
// inter-state:
igst = fullGST
```
(The issue suggested `cgst = sgst = Round2(fullGST/2)`, but that can still be ±1 paise off for odd-paise fullGST; the remainder split is exact.) Drop the `halfGST` line.

### Item 5 — BulkReleasePayouts error labeling + cap + audit IDs (handlers/admin_payout.go)
`BulkReleasePayouts` labels EVERY `ReleaseHold` error as `not_eligible` — so a real failure (DB/gateway error) is silently reported as a benign skip and never re-driven. It also has no size cap and the audit row records only counts.

**Fix:**
- Distinguish `errors.Is(err, services.ErrHoldNotEligible)` → `skipped` (benign, racing/duplicate) from any other error → new `failed` bucket (reconcile candidates), returned in the response and recorded in the audit row.
- Record the released / skipped / failed IDs in the `payout.bulk_released` audit row (not just counts) for traceability.
- Cap the batch: `const maxBulkReleaseItems = 500`. If the resolved item set exceeds it, return `400 too_many_items` with a message to narrow the batch (no silent truncation).

## Tests (TDD)
- `TestComputeOrderEarnings_GSTSplitReconciles` (NEW) — odd-paise commission (e.g. ItemRevenue 5 + CommissionRate 0.10 → commission 0.50): assert `CGST+SGST == Round2(RateGST*commission)` intra-state and `IGST == fullGST` inter-state. RED on the old `halfGST` code.
- Existing `TestComputeOrderEarnings_IntraState` / `_InterState` (commission 60 → 5.4/5.4, 10.8) stay green.
- Bulk handler tests (NEW): a real (non-eligible) failure lands in `failed`, an `ErrHoldNotEligible` lands in `skipped`, released IDs returned; oversized batch → 400. (Uses/extends the admin_payout handler harness.)

## Out of scope → follow-ups
- **Item 4 (ReleaseDayPayout idempotency):** ALREADY handled by #459 (`isAlreadyReleasedErr` tolerate). The `FetchTransfer` OnHold check is a documented follow-up — leave as noted.
- **Item 2 (partial reverse):** not in scope — the current model is full-clawback (#457). File follow-up if partial disputes land.
- **Item 3 (queue Amount = gross):** display-only DTO (documented). File follow-up to compute captured/net for the queue.
- **Item 6 (webhook replay ts/nonce):** larger security change → own PR/issue.
- **Item 7 (wallet-only orders bypass hold):** needs a reconciliation check → own follow-up.

## Verification
`go build ./...`, `go vet`, `go test ./services/ ./handlers/ -count=1`, `go test ./...`. Money-path gates: plan-check + independent verify.
