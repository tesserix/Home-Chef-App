# Vendor App UI Modernization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the vendor app's flush custom tab bar with a floating monochrome dock (solid-ink active pill), re-anchor tab-screen scroll padding, and adopt the shared `EmptyState` in Menu — matching the customer app's modernization in the vendor's Uber-monochrome system.

**Architecture:** Port the customer `Dock.tsx` + `dock-metrics.ts` into `apps/mobile-vendor/components/navigation/`, swapping the coral accent for `ink`/`paper` and dropping the cart FAB. Swap the `CustomTabBar` wiring in `(tabs)/_layout.tsx`. Add `useDockClearance()` bottom padding to the 4 tab screens. Replace Menu's bespoke empty state with the shared component.

**Tech Stack:** Expo / React Native, expo-router `<Tabs>`, `react-native-reanimated` (FadeIn), `react-native-safe-area-context`, `lucide-react-native`, `@homechef/mobile-shared/theme` tokens.

## Global Constraints

- Scope: `apps/mobile-vendor` only. No backend changes. Light mode.
- **No color-token changes.** Use only existing `colors.ink`, `colors.paper`, `colors.mist` from `@homechef/mobile-shared/theme`. `herb` (persimmon) stays retired. Shadow color `#000` is the only literal.
- Active tab = solid `ink` pill, white (`paper`) icon+label. Inactive = icon-only `ink.muted`.
- No cart FAB / no 5th slot (vendor has no cart).
- Motion: `FadeIn` 250ms `Easing.bezier(0.22, 1, 0.36, 1)`; gated on `useReducedMotion()` → instant. No bounce/overshoot.
- Do NOT port the customer `ScreenTitle`. Do NOT touch existing chips, section cards, segmented control, or dashboard structure.
- Every dock slot ≥44px touch target; `accessibilityRole` tablist/tab + selected state + label.
- Verification is `npx tsc --noEmit` + iOS-sim Debug build + screenshots (no jest unit tests for these RN UI files). Local sim builds only (no EAS) per `feedback_local_sim_builds_only.md`.
- Git: feature branch `feat/vendor-ui-modernization` → PR to `main` (no direct push). Single-line commit messages, no signatures.

---

### Task 1: Dock geometry module

**Files:**
- Create: `apps/mobile-vendor/components/navigation/dock-metrics.ts`

**Interfaces:**
- Produces: `DOCK_HEIGHT: number` (64), `DOCK_BOTTOM_GAP: number` (4), `useDockClearance(): number`.

- [ ] **Step 1: Create the module** (verbatim port from customer — geometry is app-agnostic)

```typescript
// dock-metrics — shared geometry for the floating dock and the layers that
// anchor to it (screen scroll padding). Lives in its own module so Dock and
// screens import it without a circular dependency.

import { useSafeAreaInsets } from 'react-native-safe-area-context';

/** Dock bar height. */
export const DOCK_HEIGHT = 64;

/** Gap between the dock and the bottom safe-area inset. */
export const DOCK_BOTTOM_GAP = 4;

/**
 * Bottom padding a screen needs so scrollable content clears the floating
 * dock (dock height + gap + safe-area + a breathing row of space).
 */
export function useDockClearance(): number {
  const insets = useSafeAreaInsets();
  return insets.bottom + DOCK_BOTTOM_GAP + DOCK_HEIGHT + 12;
}
```

- [ ] **Step 2: Typecheck**

Run: `cd apps/mobile-vendor && npx tsc --noEmit`
Expected: no new errors from this file.

- [ ] **Step 3: Commit**

```bash
git add apps/mobile-vendor/components/navigation/dock-metrics.ts
git commit -m "feat(mobile-vendor): dock geometry metrics + useDockClearance"
```

---

### Task 2: The Dock component

**Files:**
- Create: `apps/mobile-vendor/components/navigation/Dock.tsx`

**Interfaces:**
- Consumes: `DOCK_HEIGHT`, `DOCK_BOTTOM_GAP`, `useDockClearance` from `./dock-metrics`; `colors` from `@homechef/mobile-shared/theme`.
- Produces: `export function Dock(props)` accepting expo-router tabBar props (`state`, `descriptors`, `navigation`); re-exports `DOCK_HEIGHT`, `DOCK_BOTTOM_GAP`, `useDockClearance`.

- [ ] **Step 1: Create the component**

```typescript
// Dock — floating bottom navigation for the vendor app (replaces the flush
// custom tab bar). A detached rounded bar hovering above the home indicator:
// solid white (paper) surface, mist hairline border, soft shadow —
// deliberately NO blur/glass. Inactive tabs are icon-only; the active tab
// expands into a solid ink pill with a white label (the dock's single accent).
// No cart FAB — the vendor app has no cart.
//
// Rendered via expo-router's <Tabs tabBar={...}>. The root View is absolutely
// positioned so scenes get full screen height and content scrolls visibly
// through the gap beneath — screens pad their scroll bottom by
// useDockClearance() so the last row clears the dock.

import { Pressable, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import Animated, { Easing, FadeIn, useReducedMotion } from 'react-native-reanimated';
import {
  LayoutDashboard,
  ClipboardList,
  UtensilsCrossed,
  MoreHorizontal,
  type LucideIcon,
} from 'lucide-react-native';
import { colors } from '@homechef/mobile-shared/theme';
import { DOCK_BOTTOM_GAP, DOCK_HEIGHT, useDockClearance } from './dock-metrics';

// Screens keep importing clearance helpers from here too.
export { DOCK_BOTTOM_GAP, DOCK_HEIGHT, useDockClearance };

// Entrances use the app-standard ease-out-quart — no bounce, no overshoot.
const ENTRANCE_EASING = Easing.bezier(0.22, 1, 0.36, 1);

const TAB_ICONS: Record<string, LucideIcon> = {
  index: LayoutDashboard,
  orders: ClipboardList,
  menu: UtensilsCrossed,
  more: MoreHorizontal,
};

// Minimal structural types for the react-navigation tabBar props — the
// package isn't a direct dependency (expo-router wraps it), so we type the
// slice we consume.
interface DockRoute {
  key: string;
  name: string;
}
interface DockProps {
  state: { index: number; routes: DockRoute[] };
  descriptors: Record<string, { options: { title?: string } }>;
  navigation: {
    emit: (event: {
      type: 'tabPress';
      target: string;
      canPreventDefault: true;
    }) => { defaultPrevented: boolean };
    navigate: (name: string) => void;
  };
}

export function Dock({ state, descriptors, navigation }: DockProps) {
  const insets = useSafeAreaInsets();
  const reduceMotion = useReducedMotion();

  return (
    <View
      style={[styles.root, { bottom: insets.bottom + DOCK_BOTTOM_GAP }]}
      pointerEvents="box-none"
    >
      <View style={styles.bar} accessibilityRole="tablist">
        {state.routes.map((route, index) => {
          const { options } = descriptors[route.key] ?? { options: {} };
          const label = options?.title ?? route.name;
          const isActive = state.index === index;
          const Icon = TAB_ICONS[route.name] ?? LayoutDashboard;

          const onPress = () => {
            const event = navigation.emit({
              type: 'tabPress',
              target: route.key,
              canPreventDefault: true,
            });
            if (!isActive && !event.defaultPrevented) {
              navigation.navigate(route.name);
            }
          };

          return (
            <Pressable
              key={route.key}
              onPress={onPress}
              accessibilityRole="tab"
              accessibilityState={{ selected: isActive }}
              accessibilityLabel={label}
              style={styles.slot}
            >
              {isActive ? (
                <Animated.View
                  entering={
                    reduceMotion
                      ? undefined
                      : FadeIn.duration(250).easing(ENTRANCE_EASING)
                  }
                  style={styles.activePill}
                >
                  <Icon size={20} color={colors.paper} />
                  <Text style={styles.activeLabel} numberOfLines={1}>
                    {label}
                  </Text>
                </Animated.View>
              ) : (
                <Icon size={22} color={colors.ink.muted} />
              )}
            </Pressable>
          );
        })}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    position: 'absolute',
    left: 16,
    right: 16,
  },
  // Solid white pill — hairline border + soft shadow, no blur.
  bar: {
    flexDirection: 'row',
    alignItems: 'center',
    height: DOCK_HEIGHT,
    borderRadius: 28,
    backgroundColor: colors.paper,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: colors.mist.DEFAULT,
    paddingHorizontal: 8,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.1,
    shadowRadius: 16,
    elevation: 10,
  },
  // Each slot flexes equally; ≥44px touch target comes from the bar height.
  slot: {
    flex: 1,
    height: '100%',
    alignItems: 'center',
    justifyContent: 'center',
  },
  activePill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: 14,
    paddingVertical: 9,
    borderRadius: 20,
    backgroundColor: colors.ink.DEFAULT,
  },
  activeLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: colors.paper,
  },
});
```

- [ ] **Step 2: Typecheck**

Run: `cd apps/mobile-vendor && npx tsc --noEmit`
Expected: no new errors. (If iOS drops `flex` on the `Pressable` slot at sim-test time, replace `Pressable` with `TouchableOpacity activeOpacity={0.7}` per `feedback_ios_pressable_array_style.md` — verified at Task 5.)

- [ ] **Step 3: Commit**

```bash
git add apps/mobile-vendor/components/navigation/Dock.tsx
git commit -m "feat(mobile-vendor): floating monochrome dock component"
```

---

### Task 3: Wire the Dock into the tabs layout

**Files:**
- Modify: `apps/mobile-vendor/app/(tabs)/_layout.tsx`

**Interfaces:**
- Consumes: `Dock` from `../../components/navigation/Dock`.

- [ ] **Step 1: Replace the CustomTabBar.** Delete the entire `CustomTabBar` function, its `styles` StyleSheet, and now-unused imports (`StyleSheet`, `Text`, `TouchableOpacity`, `View`, `useSafeAreaInsets`, `BottomTabBarProps`, the per-icon imports stay only if still referenced by `Tabs.Screen`). Keep the `Tabs.Screen` definitions unchanged (titles + `tabBarIcon`). The icons are still needed by `Tabs.Screen` options, so keep the lucide imports. Result file:

```typescript
import { Tabs } from 'expo-router';
import {
  LayoutDashboard,
  ClipboardList,
  UtensilsCrossed,
  MoreHorizontal,
} from 'lucide-react-native';
import { Dock } from '../../components/navigation/Dock';

export default function VendorTabsLayout() {
  return (
    <Tabs
      tabBar={(props) => <Dock {...props} />}
      screenOptions={{
        headerShown: false,
      }}
    >
      <Tabs.Screen
        name="index"
        options={{
          title: 'Dashboard',
          tabBarIcon: ({ color, focused, size }) => (
            <LayoutDashboard
              size={size}
              color={String(color)}
              strokeWidth={focused ? 2.4 : 2}
            />
          ),
        }}
      />
      <Tabs.Screen
        name="orders"
        options={{
          title: 'Orders',
          tabBarIcon: ({ color, focused, size }) => (
            <ClipboardList
              size={size}
              color={String(color)}
              strokeWidth={focused ? 2.4 : 2}
            />
          ),
        }}
      />
      <Tabs.Screen
        name="menu"
        options={{
          title: 'Menu',
          tabBarIcon: ({ color, focused, size }) => (
            <UtensilsCrossed
              size={size}
              color={String(color)}
              strokeWidth={focused ? 2.4 : 2}
            />
          ),
        }}
      />
      <Tabs.Screen
        name="more"
        options={{
          title: 'More',
          tabBarIcon: ({ color, focused, size }) => (
            <MoreHorizontal
              size={size}
              color={String(color)}
              strokeWidth={focused ? 2.4 : 2}
            />
          ),
        }}
      />
    </Tabs>
  );
}
```

Note: `Dock` renders its own icons via `TAB_ICONS`, so the `Tabs.Screen tabBarIcon` options are now unused by the custom `tabBar` but kept harmless for registration parity. (Leaving them avoids churn; they cost nothing since `Dock` ignores them.)

- [ ] **Step 2: Typecheck**

Run: `cd apps/mobile-vendor && npx tsc --noEmit`
Expected: no errors, no unused-import errors (`noUnusedLocals` is on — ensure every remaining import is referenced).

- [ ] **Step 3: Commit**

```bash
git add apps/mobile-vendor/app/(tabs)/_layout.tsx
git commit -m "feat(mobile-vendor): mount floating dock as tabs tabBar"
```

---

### Task 4: Re-anchor tab-screen scroll padding

**Files:**
- Modify: `apps/mobile-vendor/app/(tabs)/index.tsx`
- Modify: `apps/mobile-vendor/app/(tabs)/orders.tsx`
- Modify: `apps/mobile-vendor/app/(tabs)/menu.tsx`
- Modify: `apps/mobile-vendor/app/(tabs)/more.tsx`

**Interfaces:**
- Consumes: `useDockClearance` from `../../components/navigation/Dock`.

- [ ] **Step 1: Dashboard (`index.tsx`).** Import the hook and apply the clearance as extra bottom padding on the `ScrollView`'s `contentContainerStyle`. The existing `scrollContent.paddingBottom` is `theme.spacing[6]`; override at runtime so we don't lose the quiet-block anchoring.

Add import near the other component imports:
```typescript
import { useDockClearance } from '../../components/navigation/Dock';
```
Inside `DashboardScreen`, after the other hooks:
```typescript
  const dockClearance = useDockClearance();
```
Change the `ScrollView` opening tag's `contentContainerStyle`:
```tsx
      <ScrollView
        contentContainerStyle={[
          styles.scrollContent,
          { paddingBottom: dockClearance },
        ]}
```

- [ ] **Step 2: Orders (`orders.tsx`).** Import the hook; add the clearance to the `FlatList` `contentContainerStyle` bottom padding. Locate the `FlatList` and merge padding:
```typescript
import { useDockClearance } from '../../components/navigation/Dock';
```
```typescript
  const dockClearance = useDockClearance();
```
On the `FlatList`:
```tsx
        contentContainerStyle={[
          styles.listContent,
          { paddingBottom: dockClearance },
        ]}
```
If `styles.listContent` does not exist, apply `contentContainerStyle={{ paddingBottom: dockClearance }}` merged with whatever content style is already present (read the file first; preserve existing horizontal/top padding).

- [ ] **Step 3: Menu (`menu.tsx`).** Same pattern on the menu list (`FlatList`/`ScrollView`). Read the file, find the scroll container, merge `{ paddingBottom: dockClearance }` into its content container style, preserving existing padding.

- [ ] **Step 4: More (`more.tsx`).** Same pattern on the `ScrollView` content container.

- [ ] **Step 5: Typecheck**

Run: `cd apps/mobile-vendor && npx tsc --noEmit`
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add apps/mobile-vendor/app/(tabs)/index.tsx apps/mobile-vendor/app/(tabs)/orders.tsx apps/mobile-vendor/app/(tabs)/menu.tsx apps/mobile-vendor/app/(tabs)/more.tsx
git commit -m "feat(mobile-vendor): pad tab screens for floating dock clearance"
```

---

### Task 5: Adopt shared EmptyState in Menu

**Files:**
- Modify: `apps/mobile-vendor/app/(tabs)/menu.tsx`

**Interfaces:**
- Consumes: `EmptyState` from `@homechef/mobile-shared/ui`.

- [ ] **Step 1: Read the current empty-state block** (the `emptyMenu` composition + the filter/search "no results" block) to capture existing copy and the "Add first item" route.

- [ ] **Step 2: Replace the bespoke "menu is empty" composition** with the shared component, preserving copy and CTA route:

```tsx
import { EmptyState } from '@homechef/mobile-shared/ui';
```
```tsx
<EmptyState
  title="Your menu is empty"
  body="Add your first dish so customers can start ordering."
  ctaLabel="Add first item"
  onCtaPress={() => router.push('/menu/new')}
/>
```
(Use the exact `body` copy currently shown and the exact route the current "Add first item" CTA uses — read them in Step 1; do not invent.)

- [ ] **Step 3: Replace the search/filter "no results" block** with a CTA-less EmptyState:
```tsx
<EmptyState title={`No items match "${searchQuery.trim()}".`} />
```

- [ ] **Step 4: Remove the now-unused styles** (`emptyMenu`, `emptyHeadline`, `emptyBody`, `emptyCta`, `emptyCtaLabel`) so `noUnusedLocals` stays clean.

- [ ] **Step 5: Typecheck**

Run: `cd apps/mobile-vendor && npx tsc --noEmit`
Expected: no errors, no unused-style warnings.

- [ ] **Step 6: Commit**

```bash
git add apps/mobile-vendor/app/(tabs)/menu.tsx
git commit -m "refactor(mobile-vendor): menu uses shared EmptyState"
```

---

### Task 6: Simulator verification

**Files:** none (build + capture only).

- [ ] **Step 1: Full typecheck**

Run: `cd apps/mobile-vendor && npx tsc --noEmit`
Expected: clean.

- [ ] **Step 2: Build + boot Debug on iOS sim** (per `project_ios_native_build_from_source.md` / `feedback_local_sim_builds_only.md`): build the Debug scheme with `xcodebuild`, install to a booted simulator with `simctl`, launch, and start Metro.

- [ ] **Step 3: Screenshot each tab** with `xcrun simctl io booted screenshot`: Dashboard (quiet + with pending), Orders (New + History), Menu (populated + empty), More. Confirm:
  - Active tab shows a solid-ink pill with white icon+label; inactive tabs icon-only grey.
  - Dock floats with content visible in the gap beneath; last row/quiet block clears the dock.
  - Opening the kitchen-status action sheet overlays the dock.
  - Tab switching navigates and the pill animates (fade, no bounce).

- [ ] **Step 4: Report** — attach before/after screenshots; note any `Pressable` flex fallback applied.
