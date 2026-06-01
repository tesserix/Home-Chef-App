# Home Chef Mobile — UX Patterns

> The rules for *how* the apps behave. The "what they look like" lives in `.impeccable.md`.
> Read this before adding a new screen, mutation, or destructive action.

## 1. Feedback channel by intent

Match the user's expectation, not the API's return value.

| Intent | Channel | Component |
|---|---|---|
| Confirm a non-blocking success ("Profile saved") | Toast | `useToast().show({ tone: 'success' })` |
| Inform without blocking ("New version available") | Toast | `useToast().show({ tone: 'info' })` |
| Surface a recoverable error ("Network slow, retrying") | Toast | `useToast().show({ tone: 'error', action: { label: 'Retry', ... } })` |
| Confirm an irreversible action ("Delete this item?") | Sheet | `<Sheet primaryDestructive ...>` |
| Force acknowledgement of a critical error | `Alert.alert` | last resort — blocks the thread |
| Optimistic delete with grace period | UndoSnackbar | `useUndoSnackbar().show({ onCommit, onUndo })` |
| Validation error inline to a form field | Input error prop | `<Input error="..." />` |
| Validation error at form level | Sheet or inline banner | depends on user fix-ability |

**Never use `Alert.alert` for routine confirmations.** It interrupts. Sheet or Toast first.

## 2. Save policy

| Field type | When to save |
|---|---|
| Free-text / multiline (bio, description) | Explicit Save button. The user is composing, not setting. |
| Single-pick / toggle (cuisine, isAvailable, isVeg) | On change (optimistic). Toast on confirm. |
| Currency / number (price) | On blur with validation. Show debounce hint if remote. |
| Step-wizard (onboarding) | On step advance — never lose progress across screens. |
| Destructive (delete) | UndoSnackbar with 5s window. |

**Persistence rule:** any local Zustand store that holds in-progress user input must persist to MMKV/SecureStore. The onboarding store currently doesn't — fix this before adding new wizards.

## 3. Loading state ladder

Pick the lowest rung that matches the wait:

```
0–150ms   → no indicator. Latency masking is honest only when invisible.
150–500ms → button spinner ON the action that triggered it.
500ms–2s  → skeleton placeholders matching the eventual layout.
2s+       → skeleton + small ink-muted text: "Still loading…" after 2s,
            "Network might be slow." after 5s.
8s+       → offer manual retry with the actual error.
```

`<Skeleton>` is for shapes you can describe ahead of time (list rows, card grids). `<ActivityIndicator>` is for unknown-duration spinners on action targets.

## 4. Destructive action checklist

Before shipping any `DELETE` or `cancel`:

- [ ] Optimistic removal from the UI on tap
- [ ] `<UndoSnackbar>` with 5-second window (vendor) or 7 seconds (driver)
- [ ] API call deferred until `onCommit` fires
- [ ] If undone, restore item *exactly* — re-fetch don't reconstruct
- [ ] If commit fails, restore item + show error toast with retry

For *irreversible* actions (cancel order with payment refund, archive year-end data), skip the undo pattern and use `<Sheet primaryDestructive>` with explicit name confirm — type the order number to confirm.

## 5. Empty state composition

`<EmptyState>` is the canonical layout — icon, one-sentence title, optional body, single CTA.

**Always:** centred, paper background, no decorative graphics. The first thing the empty state should answer is "what can I do here?" — that's the CTA. The icon is decoration; the CTA is content.

Bad: "No items found." Empty. No action.
Good: "No menu items yet — add your first dish." + `[+ Add menu item]` CTA.

## 6. Form patterns

Every form field uses `<Input>`. Mandatory props:

- `label` — every field has a visible label. No placeholder-as-label.
- `accessibilityHint` — auto-derived from `error` when present.

Errors:
- Inline `<Input error="...">` for field-level
- Banner at top for form-level (e.g. "Couldn't reach the kitchen API")
- Never both for the same condition

Long forms (onboarding, menu item editor) chunk into sections with one h2 each. Save-on-blur for atomic fields; explicit submit at the end.

## 7. Navigation feedback

- Push: full slide on iOS, fade-up on Android (default Expo Router)
- Replace: cross-fade — user shouldn't see two screens transitioning
- Modal: full bottom-up slide (Stack with `presentation: 'modal'`)
- Sheet: bottom sheet at 40% or dynamic — never full screen unless content demands it

After any successful mutation that closes a screen, dispatch a Toast on the *next* screen — never on the same screen that's animating away.

## 8. Per-role density

`.impeccable.md` mandates per-role density. Apply via `theme.roleScale`:

| Role | textBase | padding | gap |
|---|---|---|---|
| Customer | 16 | 16 | 12 |
| Vendor | 15 | 12 | 8 |
| Driver | 18 | 20 | 16 |

Vendor screens pack tighter (more data per viewport, faster scanning). Customer breathes. Driver strips down.

## 9. Accessibility floor

- All `Pressable`s have `accessibilityLabel` + `accessibilityRole`
- Touch target ≥ 44pt (vendor/customer), 48pt (driver) — use `min-h-touch`
- Persimmon focus ring for keyboard navigation (Tab on iPad keyboard) — 2px solid, 2px offset
- Form errors set `accessibilityHint` so VoiceOver announces them
- All ActivityIndicators set `accessibilityLabel="Loading"` and don't double up with text saying the same

## 10. Error boundary

`<ErrorBoundary>` is mounted at root in every app's `_layout.tsx`. It catches render crashes and shows a recovery UI with the error + Try Again button. It does NOT catch:

- Async errors (use try/catch + `getServerErrorMessage`)
- API errors (use the same)
- Crashes inside event handlers (use try/catch in the handler)

Anything the ErrorBoundary catches is a *bug*. Log it.
