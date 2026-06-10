---
phase: 260610-grk
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - apps/mobile-vendor/components/vendor/PendingOrderCard.tsx
  - apps/mobile-vendor/app/orders/[orderId].tsx
  - apps/mobile-vendor/app/(tabs)/menu.tsx
  - apps/mobile-vendor/app/menu/MenuItemForm.tsx
  - apps/mobile-vendor/app/(auth)/login.tsx
  - apps/mobile-vendor/app/notification-preferences.tsx
  - apps/mobile-vendor/app/reviews.tsx
  - apps/mobile-vendor/app/payout.tsx
  - apps/mobile-vendor/app/admin-requests.tsx
  - apps/mobile-vendor/app/analytics.tsx
  - apps/mobile-vendor/app/(tabs)/orders.tsx
  - apps/mobile-vendor/app/profile.tsx
autonomous: true
requirements: [design-critique-ux]
must_haves:
  truths:
    - "Order age text turns amber at ≥5 min and destructive red at ≥10 min and re-ticks every 30-60s"
    - "Waiting-for-driver status shows a live elapsed-minutes counter"
    - "All identified touch targets are ≥44px"
    - "Login sign-in failures surface Alert.alert with the server message"
    - "Menu screen has a working search input that combines with category filter"
    - "Review filter badges show counts and ≤2★ is relabelled"
    - "Success toasts fire after category create, notification save, and kitchen photo upload"
    - "Quiet-hours inputs show inline error when HH:MM format is invalid"
    - "Payout re-entry helper text visible for masked fields; save button label reflects dirty state"
    - "admin-requests badge and notes block use theme.radius tokens"
    - "Analytics and orders queue empty-state copy is user-reassuring, not implementation detail"
  artifacts:
    - path: "apps/mobile-vendor/components/vendor/PendingOrderCard.tsx"
      provides: "Live age escalation with interval ticker"
    - path: "apps/mobile-vendor/app/orders/[orderId].tsx"
      provides: "Waiting-for-driver live counter + 44px contact buttons"
    - path: "apps/mobile-vendor/app/(tabs)/menu.tsx"
      provides: "Search TextInput + 44px addBtn"
    - path: "apps/mobile-vendor/app/menu/MenuItemForm.tsx"
      provides: "44px photo-remove hitSlop + category-created toast"
    - path: "apps/mobile-vendor/app/(auth)/login.tsx"
      provides: "try/catch on all sign-in handlers"
    - path: "apps/mobile-vendor/app/notification-preferences.tsx"
      provides: "44px quiet-hours inputs + onSuccess toast + inline HH:MM error"
    - path: "apps/mobile-vendor/app/reviews.tsx"
      provides: "Count badges on filter tabs + relabelled Low filter"
    - path: "apps/mobile-vendor/app/payout.tsx"
      provides: "Re-entry helper text + contextual save button label"
    - path: "apps/mobile-vendor/app/admin-requests.tsx"
      provides: "theme.radius tokens on badge and notesBlock"
    - path: "apps/mobile-vendor/app/analytics.tsx"
      provides: "Updated empty-state copy"
    - path: "apps/mobile-vendor/app/(tabs)/orders.tsx"
      provides: "Updated QueueEmpty copy"
    - path: "apps/mobile-vendor/app/profile.tsx"
      provides: "Toast after kitchen photo upload"
  key_links:
    - from: "PendingOrderCard"
      to: "theme.colors.amber.DEFAULT / theme.colors.destructive.DEFAULT"
      via: "age-derived style on meta Text"
      pattern: "setInterval.*30_000"
    - from: "login.tsx handleGoogleSignIn / handleAppleSignIn / handleBiometricLogin"
      to: "Alert.alert"
      via: "try/catch wrapping each handler body"
      pattern: "try.*handleGoogleSignIn"
    - from: "menu.tsx searchQuery state"
      to: "filteredItems useMemo"
      via: "case-insensitive name filter combined with category filter"
      pattern: "searchQuery.*toLowerCase"
---

<objective>
Apply 10 UX improvements identified in the vendor-app design critique.

Purpose: Raise order-management responsiveness, safety, and polish for kitchen-hands-dirty context where chefs process orders under time pressure.
Output: 12 modified files covering urgency cues, touch safety, error handling, search, feedback toasts, copy, and visual alignment.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md

<!-- iOS Pressable constraint: function-style Pressable styles drop flex/bg/padding on iOS.
     Visual layer MUST go on an inner View. Outer Pressable carries only opacity/flex.
     See feedback_ios_pressable_array_style.md and existing patterns in PendingOrderCard.tsx. -->
<!-- Theme tokens: import { theme } from '@homechef/mobile-shared/theme'
     Colors: theme.colors.amber.DEFAULT, theme.colors.destructive.DEFAULT,
             theme.colors.ink.DEFAULT, theme.colors.herb.DEFAULT, theme.colors.mist.strong
     Radius: theme.radius.DEFAULT (8px), theme.radius.sm
     Spacing: theme.spacing[N]
     NO hardcoded color/radius values. -->
<!-- Toast: const { show: showToast } = useToast() from '@homechef/mobile-shared/ui'
     showToast({ message: '...', tone: 'success' | 'error' })
     See pattern in orders/[orderId].tsx submitCancel. -->
<!-- noUncheckedIndexedAccess is ON: all array accesses must be guarded. -->
<!-- Inter/Geist font families only, matching existing usage in each file. -->
</context>

<tasks>

<task type="auto">
  <name>Task 1: Order urgency — age escalation + waiting-for-driver counter</name>
  <files>
    apps/mobile-vendor/components/vendor/PendingOrderCard.tsx,
    apps/mobile-vendor/app/orders/[orderId].tsx
  </files>
  <action>
**PendingOrderCard.tsx — live age escalation:**

1. Add `useEffect` + `useState` imports (already has them via React Native; add `useEffect, useState` from 'react').
2. Add a `now` state ticking every 45 000 ms:
   ```ts
   const [now, setNow] = useState(Date.now);
   useEffect(() => {
     const id = setInterval(() => setNow(Date.now()), 45_000);
     return () => clearInterval(id);
   }, []);
   ```
3. Compute age in minutes from `order.createdAt` using `now`:
   ```ts
   const ageMins = Math.max(0, Math.floor((now - new Date(order.createdAt).getTime()) / 60_000));
   ```
4. Derive age color:
   ```ts
   const ageColor =
     ageMins >= 10 ? theme.colors.destructive.DEFAULT
     : ageMins >= 5 ? theme.colors.amber.DEFAULT
     : theme.colors.ink.soft;
   ```
5. Replace the static `styles.meta` on the age portion. The meta line is currently one `<Text>` node. Split into two inline `<Text>` children so only the age portion gets the escalated color:
   ```tsx
   <Text style={styles.meta} numberOfLines={1}>
     {formatItemsSummary(order.items)}
     {'  ·  '}
     <Text style={{ color: ageColor }}>{formatMinutesAgo(order.createdAt)}</Text>
   </Text>
   ```
   Pass `now` into `formatMinutesAgo` so it uses the same clock tick:
   ```ts
   function formatMinutesAgo(iso: string, nowMs: number): string {
     const t = new Date(iso).getTime();
     if (Number.isNaN(t)) return '';
     const mins = Math.max(0, Math.floor((nowMs - t) / 60_000));
     ...same body...
   }
   ```
   Call as `formatMinutesAgo(order.createdAt, now)`.

**orders/[orderId].tsx — waiting-for-driver counter:**

The `caption` map at ~line 357 sets `ready: 'Waiting for driver to pick up.'`. This is rendered in the `FooterBlock` component. Locate the `FooterBlock` (or the inline render of `ready` caption) and augment only the `ready` case:

1. Add `useEffect, useState` imports if not already present for this component scope.
2. The `FooterBlock` receives `order` (or at least `status` and a timestamp). Check what props it receives. If it only receives `status`, also pass `updatedAt: string | undefined` as a prop from the caller site.
3. Add ticker state in `FooterBlock` (or the ready-caption render site):
   ```ts
   const [now, setNow] = useState(Date.now);
   useEffect(() => {
     if (status !== 'ready') return;
     const id = setInterval(() => setNow(Date.now()), 45_000);
     return () => clearInterval(id);
   }, [status]);
   ```
4. Compute elapsed for ready state:
   ```ts
   const readyElapsed = (() => {
     if (status !== 'ready') return '';
     const ts = updatedAt ?? order?.updatedAt ?? order?.createdAt;
     if (!ts) return '';
     const mins = Math.max(0, Math.floor((now - new Date(ts).getTime()) / 60_000));
     return mins < 1 ? '' : ` · ${mins}m`;
   })();
   ```
5. Change the ready caption text to:
   ```tsx
   <Text style={styles.footerCaption}>{`Waiting for driver to pick up${readyElapsed}.`}</Text>
   ```
   Only the `ready` branch changes; all other captions remain static strings.
  </action>
  <verify>
    <automated>cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App/apps/mobile-vendor && npx tsc --noEmit 2>&1 | head -40</automated>
  </verify>
  <done>
    - TypeScript compiles with no errors for the two modified files.
    - PendingOrderCard renders age in ink.soft when &lt;5 min, amber at ≥5 min, destructive.DEFAULT at ≥10 min.
    - setInterval clears on unmount.
    - Waiting-for-driver footer appends " · Nm" only when ready and elapsed ≥1 min.
  </done>
</task>

<task type="auto">
  <name>Task 2: Touch targets, login error handling, and menu search</name>
  <files>
    apps/mobile-vendor/app/menu/MenuItemForm.tsx,
    apps/mobile-vendor/app/notification-preferences.tsx,
    apps/mobile-vendor/app/orders/[orderId].tsx,
    apps/mobile-vendor/app/(tabs)/menu.tsx,
    apps/mobile-vendor/app/(auth)/login.tsx
  </files>
  <action>
**44px touch targets:**

a) MenuItemForm.tsx — photo remove button (~line 354 styles):
   The `removeBtn` is 20×20. Keep the visual 20×20 circle. Add `hitSlop={{ top: 12, right: 12, bottom: 12, left: 12 }}` to the `Pressable` wrapping `removeBtn`. The Pressable's `removeBtnWrap` style (absolute, top: -6, right: -6) stays; hitSlop expands the touch zone without layout change.

b) notification-preferences.tsx — quiet-hours TextInputs (~line 349 styles):
   In `quietInput` StyleSheet entry, change `paddingVertical: 10` to ensure `minHeight: 44` (add `minHeight: 44`), and change `borderRadius: 6` to `borderRadius: theme.radius.DEFAULT` (which is 8).

c) orders/[orderId].tsx — contact buttons (~line 1026 styles):
   In `contactBtn` StyleSheet entry, change `minHeight: 36` to `minHeight: 44`.

d) menu.tsx — addBtn (~line 291 styles):
   In `addBtn` StyleSheet entry, change `minHeight: 36` to `minHeight: 44`.

**Login error handling — login.tsx:**

Each of the three handlers (`handleGoogleSignIn`, `handleAppleSignIn`, `handleBiometricLogin`) and the inline `onLogin` callback inside `<LoginScreen>` currently throw without a catch boundary visible to the UI.

Wrap each handler body in try/catch. Import `Alert` from 'react-native' (already used in the file? check imports — add if missing).

Pattern for each:
```ts
const handleGoogleSignIn = async () => {
  try {
    await GoogleSignin.hasPlayServices({ showPlayServicesUpdateDialog: true });
    const result = ...;
    ...existing body up to router.replace...
  } catch (err: unknown) {
    const msg =
      err instanceof Error ? err.message : 'Something went wrong. Please try again.';
    Alert.alert('Sign-in failed', msg);
  }
};
```

Apply the same pattern to `handleAppleSignIn`, `handleBiometricLogin`, and the `onLogin` prop callback. FCM registration blocks (`try { const fcmToken = ... } catch { /* non-fatal */ }`) stay exactly as-is inside the outer try block — the inner catch absorbs FCM errors without alerting.

**Menu search — menu.tsx:**

1. Add `searchQuery` state: `const [searchQuery, setSearchQuery] = useState('');`
2. Add `TextInput` import from 'react-native' (already present? grep the imports — add if missing).
3. Modify `filteredItems` useMemo to additionally filter by `searchQuery`:
   ```ts
   const filteredItems = useMemo<MenuItem[]>(() => {
     const byCat = selectedCategoryId === ALL_CATEGORIES
       ? items
       : items.filter((i) => i.categoryId === selectedCategoryId);
     if (!searchQuery.trim()) return byCat;
     const q = searchQuery.trim().toLowerCase();
     return byCat.filter((i) => i.name.toLowerCase().includes(q));
   }, [items, selectedCategoryId, searchQuery]);
   ```
4. Add a search input below the tab bar (or below command bar if no tabs), visible only when `hasItems`:
   ```tsx
   {hasItems && (
     <View style={styles.searchWrap}>
       <TextInput
         value={searchQuery}
         onChangeText={setSearchQuery}
         placeholder="Search items…"
         placeholderTextColor={theme.colors.ink.muted}
         style={styles.searchInput}
         returnKeyType="search"
         clearButtonMode="while-editing"
         accessibilityLabel="Search menu items"
       />
     </View>
   )}
   ```
5. Add to StyleSheet:
   ```ts
   searchWrap: {
     paddingHorizontal: theme.spacing[4],
     paddingBottom: theme.spacing[2],
     paddingTop: theme.spacing[1],
   },
   searchInput: {
     fontFamily: 'Inter',
     fontSize: theme.typography.size.body.size,
     color: theme.colors.ink.DEFAULT,
     minHeight: 44,
     borderWidth: StyleSheet.hairlineWidth,
     borderColor: theme.colors.mist.strong,
     borderRadius: theme.radius.DEFAULT,
     backgroundColor: theme.colors.paper,
     paddingHorizontal: theme.spacing[3],
   },
   ```
6. Update the "no items match" empty state: when `hasItems && filteredItems.length === 0`, existing `filterEmptyBlock` text already says "No items in this category." Extend the condition: if `searchQuery.trim()` is non-empty, show "No items match \"<query>\"." instead.
   ```tsx
   : filteredItems.length === 0 ? (
     <View style={styles.filterEmptyBlock}>
       <Text style={styles.filterEmptyText}>
         {searchQuery.trim()
           ? `No items match "${searchQuery.trim()}".`
           : <>No items in this category. Tap <Text style={styles.filterEmptyAccent}>+ Add</Text> to create one.</>}
       </Text>
     </View>
   )
   ```
  </action>
  <verify>
    <automated>cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App/apps/mobile-vendor && npx tsc --noEmit 2>&1 | head -40</automated>
  </verify>
  <done>
    - TypeScript compiles clean.
    - All four touch targets have minHeight 44 (contact buttons, addBtn, quietInput) or hitSlop 12 on all sides (photo remove).
    - quietInput borderRadius uses theme.radius.DEFAULT.
    - Login handlers have try/catch with Alert.alert; FCM blocks remain inner non-fatal catches.
    - Menu search input visible when hasItems; filters by name case-insensitively combined with category; "No items match" copy shown when query is set and no results.
  </done>
</task>

<task type="auto">
  <name>Task 3: Feedback toasts, review filter counts, payout UX, radius alignment, and empty-state copy</name>
  <files>
    apps/mobile-vendor/app/menu/MenuItemForm.tsx,
    apps/mobile-vendor/app/notification-preferences.tsx,
    apps/mobile-vendor/app/profile.tsx,
    apps/mobile-vendor/app/reviews.tsx,
    apps/mobile-vendor/app/payout.tsx,
    apps/mobile-vendor/app/admin-requests.tsx,
    apps/mobile-vendor/app/analytics.tsx,
    apps/mobile-vendor/app/(tabs)/orders.tsx
  </files>
  <action>
**Success toasts:**

a) MenuItemForm.tsx — after category created successfully (~line 509):
   Import `useToast` at the top: `import { useToast } from '@homechef/mobile-shared/ui';`
   Add `const { show: showToast } = useToast();` inside the component.
   In `handleCreateCategory`, after `setCategoryId(created.id); setNewCatName(''); setShowNewCatInput(false);`, add:
   ```ts
   showToast({ message: `Category "${trimmed}" created.`, tone: 'success' });
   ```

b) notification-preferences.tsx — onSuccess toast:
   `useToast` is already imported. Find `useUpdateNotificationPreferences` — its `onSuccess` currently only invalidates the query. Add a toast in the component `persist` call-site using `useMutation`'s `onSuccess` option, or call `showToast` in a wrapper around `persist`.
   The cleanest approach: inside the component, wrap each `persist()` call in a consistent helper or add `onSuccess` to the mutation options at call-site. Since `persist` is called inline (e.g., `onValueChange={(v) => persist({ newOrders: v })}`), add an `onSuccess` callback to the `updateMutation.mutate()` call inside `persist`.
   Find the `persist` function definition and add `onSuccess`:
   ```ts
   function persist(patch: Partial<NotificationPreferences>) {
     updateMutation.mutate(patch, {
       onSuccess: () => showToast({ message: 'Preferences saved.', tone: 'success' }),
     });
   }
   ```
   Also add inline error text for invalid quiet-hours format. `HHMM_REGEX` is already defined. Add a `quietTimeError` state:
   ```ts
   const [quietTimeError, setQuietTimeError] = useState<string | null>(null);
   ```
   In `handleQuietHoursInput`, set the error when regex fails:
   ```ts
   function handleQuietHoursInput(field: 'quietHoursStart' | 'quietHoursEnd', value: string) {
     setPrefs({ ...prefs, [field]: value });
     if (HHMM_REGEX.test(value)) {
       setQuietTimeError(null);
       persist({ [field]: value });
     } else if (value.length === 5) {
       // Only show error once user has typed a full 5-char string
       setQuietTimeError('Use HH:MM format (e.g. 22:00)');
     } else {
       setQuietTimeError(null);
     }
   }
   ```
   Render below the quiet-hours row (after the two TextInput fields):
   ```tsx
   {quietTimeError ? (
     <Text style={styles.quietErrorText}>{quietTimeError}</Text>
   ) : null}
   ```
   Add to StyleSheet:
   ```ts
   quietErrorText: {
     fontFamily: 'Inter',
     fontSize: theme.typography.size.caption.size,
     color: theme.colors.destructive.DEFAULT,
     paddingHorizontal: theme.spacing[4],
     paddingTop: 4,
   },
   ```

c) profile.tsx — kitchen photo upload toast (~line 455):
   `useToast` and `showToast` are already set up. Add `onSuccess` to the `uploadKitchenPhotoMutation.mutate` call:
   ```ts
   uploadKitchenPhotoMutation.mutate(result.assets[0].uri, {
     onError: (err) =>
       Alert.alert('Upload failed', getServerErrorMessage(err, 'Failed to upload photo.')),
     onSuccess: () => showToast({ message: 'Kitchen photo added.', tone: 'success' }),
   });
   ```

**Review filter badges — reviews.tsx:**

1. Compute per-star counts from loaded reviews using `useMemo`:
   ```ts
   const filterCounts = useMemo<Record<StarFilter, number>>(() => ({
     all: reviews.length,
     '5': reviews.filter((r) => r.rating === 5).length,
     '4': reviews.filter((r) => r.rating === 4).length,
     '3': reviews.filter((r) => r.rating === 3).length,
     '2-': reviews.filter((r) => r.rating <= 2).length,
   }), [reviews]);
   ```
   `reviews` comes from `data?.reviews ?? []` — derive this before the return.
2. Update `FILTER_LABELS` or render labels dynamically. Since `FILTER_LABELS` is a module-level constant, render the label + count inline at the render site:
   ```tsx
   {FILTER_LABELS.map(({ key, label }) => (
     <FilterTab
       key={key}
       label={`${label} (${filterCounts[key]})`}
       ...
     />
   ))}
   ```
   When data is loading/absent, counts will be 0 — that's acceptable.
3. Change the `≤2★` label in `FILTER_LABELS` to `'Low (1–2★)'`. Update the entry:
   ```ts
   { key: '2-', label: 'Low (1–2★)' },
   ```

**Payout UX — payout.tsx:**

a) Re-entry helper text: Locate the `Field` components for "Account number" (bank) and "UPI ID". After each, add a helper `<Text>` that renders when the existing payout method is set (i.e., `data` is truthy, meaning there is already a saved method):
   ```tsx
   {data && (
     <Text style={styles.reentryHelper}>(re-enter required)</Text>
   )}
   ```
   Place this Text immediately after the Account number Field and after the UPI ID Field, inside their respective conditional blocks.
   Add to StyleSheet:
   ```ts
   reentryHelper: {
     fontFamily: 'Inter',
     fontSize: theme.typography.size.caption.size,
     color: theme.colors.ink.muted,
     paddingHorizontal: theme.spacing[4],
     paddingTop: 4,
     paddingBottom: theme.spacing[2],
   },
   ```

b) Contextual save button label (~line 422): The button currently always shows "Save payout details". Change to:
   ```tsx
   <Text style={styles.saveBtnLabel}>
     {isDirty ? 'Save changes' : 'Save payout details'}
   </Text>
   ```

**Radius alignment — admin-requests.tsx:**

a) `badge` style (~line 183): change `borderRadius: 4` to `borderRadius: theme.radius.sm`.
b) `notesBlock` style (~line 214): change `borderRadius: 2` to `borderRadius: theme.radius.sm`.
   (theme.radius.sm is 4px by convention in this codebase — keeps the visual tight while aligning to the token system. Use `theme.radius.DEFAULT` only if sm does not exist on the theme object; verify via existing usage in the file.)

**Empty-state copy:**

a) analytics.tsx (~line 217): Replace:
   ```
   "Analytics appear once you have completed orders. Check back after your first few deliveries."
   ```
   with:
   ```
   "Analytics appear after your first completed order."
   ```
   Two sentences → one crisp sentence. Remove the "Check back after your first few deliveries" tail.

b) orders.tsx `QueueEmpty` (~line 174): Replace:
   ```
   "New orders arrive here automatically. We poll every 10 seconds."
   ```
   with:
   ```
   "New orders will appear here automatically. Your queue is up to date."
   ```
   Removes implementation detail; adds reassurance.
  </action>
  <verify>
    <automated>cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App/apps/mobile-vendor && npx tsc --noEmit 2>&1 | head -40</automated>
  </verify>
  <done>
    - TypeScript compiles clean across all 8 modified files.
    - Category-created toast fires in MenuItemForm after successful creation.
    - Notification preferences show "Preferences saved." toast on each persist call; quiet-hours inline error shows on malformed HH:MM (after 5 chars typed).
    - Kitchen photo upload shows success toast in profile.tsx.
    - Review filter tabs show "(N)" counts; "≤2★" renamed to "Low (1–2★)".
    - Payout "(re-enter required)" helper visible under masked fields when data exists; button label is "Save changes" when isDirty.
    - admin-requests badge and notesBlock use theme.radius.sm (not hardcoded 4 or 2).
    - Analytics empty-state copy is one sentence with no delivery-count reference.
    - QueueEmpty copy has no "We poll every 10 seconds" implementation detail.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Sign-in handler → Alert | Error messages from auth errors may contain server-returned strings; these are display-only in a native Alert, no XSS surface |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-grk-01 | Information Disclosure | login.tsx Alert.alert error message | accept | Error message shown only to the authenticated user on their own device; no sensitive credential data leaks (tokens are never in error strings from Google/Apple SDKs) |
| T-grk-02 | Denial of Service | setInterval in PendingOrderCard | mitigate | Interval clears on unmount via useEffect cleanup return; no accumulation across re-renders |
</threat_model>

<verification>
Run after all three tasks:

```bash
cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App/apps/mobile-vendor && npx tsc --noEmit
```

Must exit 0 with no errors.
</verification>

<success_criteria>
All 10 design-critique items implemented across 12 files with zero TypeScript errors. Each item maps to a specific observable behaviour described in the task `<done>` blocks. No hardcoded colors, radii, or font families introduced. iOS Pressable inner-View pattern preserved throughout. Persimmon (theme.colors.herb) not used as a new accent.
</success_criteria>

<output>
After completion, create `.planning/quick/260610-grk-vendor-app-ux-improvements-from-design-c/260610-grk-SUMMARY.md` following the summary template.
</output>
