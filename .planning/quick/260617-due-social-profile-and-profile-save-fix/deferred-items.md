# Deferred Items — 260617-due

Out-of-scope discoveries logged during execution. NOT fixed (unrelated to this plan's task changes).

## Pre-existing typecheck errors in packages/mobile-shared

`npx tsc --noEmit -p tsconfig.json` reports 2 errors, both in
`packages/mobile-shared/src/components/OfflineBanner.tsx` (lines 27, 28):

```
src/components/OfflineBanner.tsx(27,11): error TS2769: ... Property 'className' does not exist on type ... View
src/components/OfflineBanner.tsx(28,13): error TS2769: ... Property 'className' does not exist on type ... Text
```

Root cause: NativeWind `className` JSX prop is not picked up by the package's
tsconfig (missing `nativewind/types` reference or jsxImportSource config in the
package's typecheck context). Pre-existing on the plan-base commit — confirmed
by stashing this plan's changes and re-running tsc (count stayed at 2). Not
caused by Task 3's `sign-in.ts` change (which adds zero errors). Out of scope.

## Pre-existing typecheck errors in apps/mobile-customer

`npx tsc --noEmit` reports 3 errors, all in files NOT touched by this plan:

```
app/_layout.tsx(24,35): error TS2322: NotificationBehavior missing shouldShowBanner, shouldShowList
components/cart/CartSheet.tsx(115,5): error TS2578: Unused '@ts-expect-error' directive.
components/cart/CartSheet.tsx(123,8): error TS2578: Unused '@ts-expect-error' directive.
```

Root cause: expo-notifications / expo SDK type drift (`_layout.tsx`) and stale
`@ts-expect-error` directives (`CartSheet.tsx`). Pre-existing on the plan-base
commit — confirmed by stashing this plan's `useProfile.ts` + `profile.tsx`
changes and re-running tsc (count stayed at 3). Task 4's changes add zero new
errors. Out of scope.
