// Home / discovery screen.
//
// Header collapses from five stacked filter rows to three visual rows:
//   1. Search pill  + "Search dishes →" / "Map view →" quick links
//   2. Cuisine category scroller (the primary browse axis — always visible)
//   3. Slim filter bar — Open Now pill (most-used quick toggle) + "Filters"
//      pill with an active-count badge + Social Feed & Catering nav entries
//
// Secondary filters (diet, price, sort) live in FilterSheet — a bottom sheet
// that opens on the "Filters" tap and drives the SAME state variables/setters
// that previously powered three separate chip rows.
//
// ALL filter state (selectedDiet, maxPrice, sort, isOpenOnly) is defined
// here and passed down — FilterSheet holds zero state of its own.

import { useEffect, useRef, useState } from 'react';
import {
  FlatList,
  Platform,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import Animated, {
  Easing,
  FadeInDown,
  useReducedMotion,
} from 'react-native-reanimated';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { Map, Search, SlidersHorizontal } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { AddressSwitcher } from '../../components/address/AddressSwitcher';
import { AddressSwitcherSheet } from '../../components/address/AddressSwitcherSheet';
import { useDockClearance } from '../../components/navigation/Dock';
import { CART_FAB_CLEARANCE } from '../../components/navigation/DockCartPill';
import { useCartStore } from '../../store/cart-store';
import { ChefCard } from '../../components/chef/ChefCard';
import { ActiveOrderStack } from '../../components/orders/ActiveOrderStack';
import { WinbackBanner } from '../../components/home/WinbackBanner';
import { ActivePlanChip } from '../../components/meal-plan/ActivePlanChip';
import { FilterSheet } from '../../components/home/FilterSheet';
import { CATERING_ENABLED, SOCIAL_ENABLED } from '../../lib/features';
import { type SheetHandle } from '@homechef/mobile-shared/ui';
import { useActiveOrder } from '../../hooks/useActiveOrder';
import { useOrderStatusWS } from '../../hooks/useOrderStatusWS';
import { useChefs } from '../../hooks/useChefs';
import type { ChefFilters } from '../../hooks/useChefs';
import { useCustomerCoords, useActiveAddress } from '../../hooks/useCustomerCoords';

// Entrance easing — ease-out-quart, matches the app-wide motion spec.
const ENTRANCE_EASING = Easing.bezier(0.22, 1, 0.36, 1);

// Android ripple tint for pills/rows on white or light surfaces — translucent
// ink derived from the charcoal token (never a new literal colour), matching
// the primitive Button's `withAlpha` convention.
const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

const CUISINES = [
  'All',
  'North Indian',
  'South Indian',
  'Chinese',
  'Continental',
  'Italian',
  'Healthy',
];

// Counts how many secondary filters are active (non-default) so the badge
// on the Filters pill reflects the applied state.
function countActiveFilters(params: {
  selectedDiet: string;
  maxPrice: number | undefined;
  sort: ChefFilters['sort'];
  isOpenOnly: boolean;
}): number {
  let n = 0;
  if (params.selectedDiet !== '') n += 1;
  if (params.maxPrice !== undefined) n += 1;
  if (params.sort !== 'rating') n += 1;
  // isOpenOnly is surfaced separately in the quick bar; don't double-count.
  return n;
}

// Skeleton card matching the photo-led 4:3 ChefCard shape.
function SkeletonCard() {
  return (
    <View style={styles.skeletonCard}>
      <View style={styles.skeletonPhoto} />
      <View style={styles.skeletonBody}>
        <View style={styles.skeletonLine1} />
        <View style={styles.skeletonLine2} />
        <View style={styles.skeletonLine3} />
      </View>
    </View>
  );
}

export default function HomeScreen() {
  // ── Filter state ────────────────────────────────────────────────────────
  // All five filter dimensions live here and flow into useChefs(filters).
  // FilterSheet receives setters as props so it drives the same state.
  // Text search lives on the unified search screen (/search-dishes) — the
  // home search pill is a navigation button, not an inline input.
  const [selectedCuisine, setSelectedCuisine] = useState<string>('All');
  const [selectedDiet, setSelectedDiet] = useState<string>('');
  const [isOpenOnly, setIsOpenOnly] = useState(false);
  const [sort, setSort] = useState<ChefFilters['sort']>('rating');
  const [maxPrice, setMaxPrice] = useState<number | undefined>(undefined);

  const { orders: activeOrders } = useActiveOrder();
  // Stack up to 3 active-order cards in the floating anchor; more than that is
  // rare and would bury the feed. Each card is ~106px tall (incl. its margin).
  const visibleActiveOrders = activeOrders.slice(0, 3);

  // Live stage changes on the active-order card (#716). The notification stream
  // is user-scoped, so one socket covers every card in the stack — no orderId.
  // Only while an order is actually in flight, so we don't hold a socket open
  // on an idle Home screen; useActiveOrder's poll stays as the fallback.
  useOrderStatusWS(undefined, activeOrders.length > 0);

  // Ref for opening the FilterSheet imperatively on Filters pill tap.
  const filterSheetRef = useRef<SheetHandle>(null);
  const addressSheetRef = useRef<SheetHandle>(null);

  // ── Data fetching ────────────────────────────────────────────────────────
  // Customer location drives the chef delivery-area gate: a chef who only
  // delivers (no pickup) and can't reach the customer is hidden, and chefs shown
  // for pickup-only carry deliverableToYou=false. Omitted when unknown, so
  // discovery falls back to un-located. We pass a very large radius so the legacy
  // 15km near-me box doesn't ALSO hard-cap the feed — reach is decided per-chef
  // by the delivery-area gate, not a fixed circle around the customer (otherwise
  // a customer with no chef within 15km sees an empty feed).
  const coords = useCustomerCoords();
  const { address: activeAddress } = useActiveAddress();
  const filters: ChefFilters = {
    cuisine: selectedCuisine !== 'All' ? selectedCuisine : undefined,
    dietary: selectedDiet || undefined,
    isOpen: isOpenOnly || undefined,
    maxPrice,
    sort,
    lat: coords?.lat,
    lng: coords?.lng,
    // Real near-me radius (km) — the API caps it. Keeps the feed to nearby
    // kitchens instead of surfacing ones in other cities/states.
    radius: coords ? 25 : undefined,
    // Hard region gate: only kitchens in the delivery address's state.
    state: activeAddress?.state || undefined,
    limit: 20,
  };

  const { data, isLoading, isFetching, refetch } = useChefs(filters);
  const chefs = data?.data ?? [];

  // Staggered card entrances (reduced-motion gated). No bounce — ease-out-quart.
  const reduceMotion = useReducedMotion();

  // Floating dock: scenes span the full screen, so scroll content + the
  // floating active-order card anchor above the dock instead of a tab bar.
  // With items in the cart, the CartFab hovers above the dock's right end —
  // lift the order card past it so the two floating layers never overlap.
  const dockClearance = useDockClearance();
  const cartHasItems = useCartStore((s) => s.items.length > 0);
  const orderStackBottom =
    dockClearance - 4 + (cartHasItems ? CART_FAB_CLEARANCE : 0);

  // ── Derived values ───────────────────────────────────────────────────────
  const activeFilterCount = countActiveFilters({ selectedDiet, maxPrice, sort, isOpenOnly });

  // ── Handlers ─────────────────────────────────────────────────────────────
  function handleOpenFilters() {
    filterSheetRef.current?.present();
  }

  // ── Header component ─────────────────────────────────────────────────────
  const renderHeader = () => (
    <>
      {/* ── Row 0: Active delivery address + map entry on one row. The address
          pill opens the switcher sheet; the circular map button replaces the
          old "Map view →" text link. ── */}
      <View style={styles.addressRow}>
        <View style={styles.addressRowPill}>
          <AddressSwitcher onOpen={() => addressSheetRef.current?.present()} />
        </View>
        <Pressable
          onPress={() => router.push('/chefs-map')}
          accessibilityRole="button"
          accessibilityLabel="View chefs on a map"
          android_ripple={{ color: ROW_RIPPLE, borderless: true, radius: 20 }}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.mapButton,
                pressed && Platform.OS === 'ios' && styles.pressedIOS,
              ]}
            >
              <Map size={18} color={customerColors.charcoal.DEFAULT} />
            </View>
          )}
        </Pressable>
      </View>

      {/* ── Row 1: Search entry. A button, not an inline input — typing happens
          on the unified search screen (dishes + chefs) with proper focus. The
          old "Search dishes →" link folded into this. ── */}
      <View style={styles.searchPillWrapper}>
        <Pressable
          onPress={() => router.push('/search-dishes')}
          accessibilityRole="button"
          accessibilityLabel="Search dishes and chefs"
          android_ripple={{ color: ROW_RIPPLE, borderless: false }}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.searchPill,
                pressed && Platform.OS === 'ios' && styles.pressedIOS,
              ]}
            >
              <Search
                size={18}
                color={customerColors.charcoal.soft}
                accessibilityElementsHidden
              />
              <Text style={styles.searchPlaceholder}>What are you craving?</Text>
            </View>
          )}
        </Pressable>
      </View>

      {/* Win-back offer (#42) — surfaces an active offer; auto-prefills checkout. */}
      <WinbackBanner />

      {/* Glanceable "your tiffin plan" entry (#434) — only renders when the
          customer has a live plan; taps open the per-day fulfilment sheet. */}
      <ActivePlanChip />

      {/* ── Row 2: Cuisine category scroller (primary browse axis) ── */}
      {/* Airbnb-style: selected = charcoal text + 2px charcoal underline;
          unselected = charcoal-soft, no fill, no border. */}
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={styles.chipRowContent}
        style={styles.chipRow}
        accessibilityRole="tablist"
      >
        {CUISINES.map((cuisine) => {
          const isSelected = selectedCuisine === cuisine;
          return (
            // iOS Pressable inner-View pattern: visual styles stay on the
            // inner View. `style` here is a static object (not a function),
            // so it's safe under the array-style bug — and it's REQUIRED:
            // without it, this Pressable is the actual flex-row child inside
            // the ScrollView's contentContainerStyle, and on-device it was
            // being treated as shrinkable, compressing every chip below its
            // label's natural width instead of letting the row scroll (R3).
            <Pressable
              key={cuisine}
              onPress={() => setSelectedCuisine(cuisine)}
              accessibilityRole="tab"
              accessibilityState={{ selected: isSelected }}
              accessibilityLabel={`Filter by ${cuisine}`}
              android_ripple={{ color: ROW_RIPPLE, borderless: false }}
              style={styles.chipPressable}
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.chip,
                    isSelected && styles.chipSelected,
                    pressed && Platform.OS === 'ios' && styles.chipPressedIOS,
                  ]}
                >
                  <Text
                    style={[
                      styles.chipLabel,
                      isSelected ? styles.chipLabelSelected : styles.chipLabelDefault,
                    ]}
                  >
                    {cuisine}
                  </Text>
                </View>
              )}
            </Pressable>
          );
        })}
      </ScrollView>

      {/* ── Row 3: Slim filter/sort bar ── */}
      {/* Open Now quick-toggle (most frequently used, warrants top-level placement)
          + Filters pill (opens FilterSheet) with an active-count badge
          + Social Feed / Catering navigation entries (discovery, not filters) */}
      <View style={styles.filterBar}>
        {/* Open Now quick-toggle */}
        <Pressable
          onPress={() => setIsOpenOnly((prev) => !prev)}
          accessibilityRole="button"
          accessibilityLabel={isOpenOnly ? 'Showing open chefs only' : 'Show open chefs only'}
          accessibilityState={{ checked: isOpenOnly }}
          android_ripple={{ color: ROW_RIPPLE, borderless: false }}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.openNowPill,
                isOpenOnly && styles.openNowPillActive,
                pressed && Platform.OS === 'ios' && styles.pressedIOS,
              ]}
            >
              <View
                style={[
                  styles.openNowDot,
                  isOpenOnly ? styles.openNowDotActive : styles.openNowDotInactive,
                ]}
              />
              <Text
                style={[
                  styles.openNowLabel,
                  isOpenOnly ? styles.openNowLabelActive : styles.openNowLabelDefault,
                ]}
              >
                Open Now
              </Text>
            </View>
          )}
        </Pressable>

        {/* Filters pill — opens the sheet; badge shows count of active secondary filters */}
        <Pressable
          onPress={handleOpenFilters}
          accessibilityRole="button"
          accessibilityLabel={
            activeFilterCount > 0
              ? `Filters — ${activeFilterCount} active`
              : 'Open filters'
          }
          android_ripple={{ color: ROW_RIPPLE, borderless: false }}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.filtersPill,
                activeFilterCount > 0 && styles.filtersPillActive,
                pressed && Platform.OS === 'ios' && styles.pressedIOS,
              ]}
            >
              <SlidersHorizontal
                size={14}
                color={
                  activeFilterCount > 0
                    ? customerColors.canvas
                    : customerColors.charcoal.soft
                }
                accessibilityElementsHidden
              />
              <Text
                style={[
                  styles.filtersPillLabel,
                  activeFilterCount > 0 && styles.filtersPillLabelActive,
                ]}
              >
                Filters
              </Text>
              {activeFilterCount > 0 && (
                <View style={styles.filterBadge}>
                  <Text style={styles.filterBadgeText}>{activeFilterCount}</Text>
                </View>
              )}
            </View>
          )}
        </Pressable>

        {/* Right side: Social Feed + Catering — navigation, not filter controls.
            Moved from their own row to the same slim bar, saving one full row. */}
        <View style={styles.navLinks}>
          {/* Social Feed — DEFERRED for v1 (stub, no real feed yet). */}
          {SOCIAL_ENABLED ? (
            <Pressable
              onPress={() => router.push('/social')}
              accessibilityRole="button"
              accessibilityLabel="Go to Social Feed"
              android_ripple={{ color: ROW_RIPPLE, borderless: false }}
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.navLinkPill,
                    pressed && Platform.OS === 'ios' && styles.pressedIOS,
                  ]}
                >
                  <Text style={styles.navLinkLabel}>Social Feed</Text>
                </View>
              )}
            </Pressable>
          ) : null}
          {/* Catering — DEFERRED for v1 (CATERING_DEPOSIT_ENABLED off). */}
          {CATERING_ENABLED ? (
            <Pressable
              onPress={() => router.push('/catering')}
              accessibilityRole="button"
              accessibilityLabel="Go to Catering"
              android_ripple={{ color: ROW_RIPPLE, borderless: false }}
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.navLinkPill,
                    pressed && Platform.OS === 'ios' && styles.pressedIOS,
                  ]}
                >
                  <Text style={styles.navLinkLabel}>Catering</Text>
                </View>
              )}
            </Pressable>
          ) : null}
        </View>
      </View>
    </>
  );

  // A single persistent FlatList drives every state (loading, results, empty).
  // The search pill + filter chips live in ListHeaderComponent so they NEVER
  // reposition when a filter tap triggers a refetch — swapping the whole screen
  // layout between a loading branch and a list branch caused the header to
  // jump. Skeletons and the empty message render inside the list body instead.
  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* Outer View is flex:1 so the floating card can be positioned absolutely
          above the tab bar without affecting FlatList scroll behaviour. */}
      <View style={styles.screenWrapper}>
        <FlatList
          data={isLoading ? [] : chefs}
          keyExtractor={(item) => item.id}
          numColumns={2}
          columnWrapperStyle={styles.columnWrapper}
          contentContainerStyle={[
            styles.listContent,
            { paddingBottom: dockClearance },
            // Pad the bottom for the floating active-order card. The stack is
            // COLLAPSED by default (~1 card + a peek), so reserve one card's
            // worth; expanding it is a deliberate, temporary overlay.
            visibleActiveOrders.length > 0 && {
              paddingBottom: dockClearance + 112,
            },
          ]}
          ListHeaderComponent={renderHeader}
          keyboardShouldPersistTaps="handled"
          renderItem={({ item, index }) => (
            <Animated.View
              style={{ flex: 1 }}
              entering={
                reduceMotion
                  ? undefined
                  // §3.5: stagger steps 40-60ms, max 3 steps — cap the delay
                  // at the 3rd card so a long grid doesn't cascade for a full
                  // second; every card past that fades in at the same delay.
                  : FadeInDown.delay(Math.min(index, 2) * 60)
                      .duration(250)
                      .easing(ENTRANCE_EASING)
              }
            >
              <ChefCard chef={item} />
            </Animated.View>
          )}
          refreshControl={
            <RefreshControl
              refreshing={isFetching && !isLoading}
              onRefresh={refetch}
              tintColor={customerColors.coral.DEFAULT}
            />
          }
          ListEmptyComponent={
            isLoading ? (
              <View style={styles.skeletonGrid}>
                <View style={styles.skeletonCol}><SkeletonCard /></View>
                <View style={styles.skeletonCol}><SkeletonCard /></View>
                <View style={styles.skeletonCol}><SkeletonCard /></View>
                <View style={styles.skeletonCol}><SkeletonCard /></View>
              </View>
            ) : (
              <View style={styles.emptyState}>
                <Text style={styles.emptyTitle}>No chefs found</Text>
                <Text style={styles.emptyBody}>Try adjusting your filters</Text>
              </View>
            )
          }
        />

        {/* Floating active-order stack — pinned just above the tab bar. With
            more than one in-flight order it collapses into a card stack (front
            card + peeking layers); tap to expand the full list. Absolute
            positioning keeps it out of the scroll flow. */}
        {visibleActiveOrders.length > 0 && (
          <View
            style={[styles.activeOrderAnchor, { bottom: orderStackBottom }]}
            pointerEvents="box-none"
          >
            <ActiveOrderStack orders={visibleActiveOrders} />
          </View>
        )}

        {/* FilterSheet — mounts a Modal only once presented (SheetBase's
            `present()`); nothing renders until the Filters pill is tapped. */}
        <FilterSheet
          ref={filterSheetRef}
          selectedDiet={selectedDiet}
          onDietChange={setSelectedDiet}
          maxPrice={maxPrice}
          onMaxPriceChange={setMaxPrice}
          sort={sort}
          onSortChange={setSort}
          isOpenOnly={isOpenOnly}
          onIsOpenOnlyChange={setIsOpenOnly}
        />

        {/* AddressSwitcherSheet — mounted at the screen root (sibling of the
            FlatList), NOT inside the list header, so its Modal overlays the
            whole screen instead of being trapped in the header's layout box.
            Renders nothing until the address pill is tapped
            (addressSheetRef.present()). */}
        <AddressSwitcherSheet ref={addressSheetRef} />
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: customerColors.canvas,
  },

  // Wrapper that lets us layer the floating card and FilterSheet above the FlatList
  screenWrapper: {
    flex: 1,
  },

  // Extra bottom padding when the active-order card is showing so the last
  // chef card isn't obscured behind it (card ~90pt + 16pt gap = ~106pt).
  listContentWithCard: {
    paddingBottom: 106,
  },

  // Shared iOS-only pressed treatment for the header's button-like pills
  // (Android relies on android_ripple on each Pressable instead). Per §3.5
  // motion contract: pressed scale 0.97.
  pressedIOS: {
    opacity: 0.85,
    transform: [{ scale: 0.97 }],
  },

  // Absolute anchor for the floating card — sits just above the tab bar.
  // `bottom` is set dynamically — anchored just above the floating dock.
  activeOrderAnchor: {
    position: 'absolute',
    left: 0,
    right: 0,
  },

  // ── Row 1: Search pill ────────────────────────────────────────────────────
  // ── Row 0: Address + map entry ────────────────────────────────────────────
  addressRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingRight: 16,
  },
  addressRowPill: {
    flex: 1,
  },
  mapButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.surface.DEFAULT,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
  },

  searchPillWrapper: {
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 12,
  },
  searchPill: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: customerColors.surface.DEFAULT,
    borderRadius: 9999,
    paddingHorizontal: 16,
    paddingVertical: 14,
    gap: 10,
    // shadow[2] per spec
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 6,
    elevation: 2,
  },
  searchPlaceholder: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: 15,
    color: customerColors.charcoal.soft,
  },

  // ── Row 2: Cuisine chip row ───────────────────────────────────────────────
  chipRow: {
    marginBottom: 4,
    // RN gives every ScrollView flexGrow: 1 (ScrollView.js — baseHorizontal), so
    // a horizontal chip row grows into any free vertical space it is offered.
    // Keep it content-height.
    flexGrow: 0,
    // Belt-and-suspenders: never let the ScrollView itself be compressed by
    // an ancestor — a horizontal ScrollView must always measure at its full
    // natural/available width so its CONTENT can legitimately exceed it and
    // scroll (that's the whole point of it being a ScrollView).
    flexShrink: 0,
  },
  // Explicit row direction + no-wrap: a horizontal ScrollView already
  // defaults to this, but leaving it implicit is exactly how the previous
  // clipping regression went unnoticed by a static styles read — spelling
  // it out removes any doubt about how these chips are meant to lay out.
  chipRowContent: {
    flexDirection: 'row',
    flexWrap: 'nowrap',
    gap: 0,
    paddingHorizontal: 16,
    paddingBottom: 4,
  },
  // This is the Pressable's OWN style (not the inner `chip` View below) —
  // the Pressable is the actual flex item living inside chipRowContent's
  // row. Root cause of the on-device truncation: this Pressable had no
  // style at all, so Yoga's flex algorithm was free to compress it (and
  // everything inside it) once the row's total content width exceeded the
  // viewport, instead of leaving that overflow to the ScrollView's own
  // horizontal scroll. flexShrink: 0 makes every chip's true content width
  // non-negotiable.
  chipPressable: {
    flexShrink: 0,
  },
  chip: {
    flexShrink: 0,
    paddingHorizontal: 12,
    paddingVertical: 10,
  },
  chipSelected: {
    borderBottomWidth: 2,
    borderBottomColor: customerColors.charcoal.DEFAULT,
  },
  // iOS-only pressed treatment — opacity only (no scale) so the underline
  // stays put under the moving text on a horizontal-scroll tab.
  chipPressedIOS: {
    opacity: 0.55,
  },
  chipLabel: {
    flexShrink: 0,
    fontFamily: 'Inter',
    fontSize: 14,
    letterSpacing: 0,
  },
  chipLabelSelected: {
    color: customerColors.charcoal.DEFAULT,
    fontFamily: 'Inter-SemiBold',
  },
  chipLabelDefault: {
    color: customerColors.charcoal.soft,
  },

  // ── Row 3: Filter bar (Open Now + Filters + nav links) ───────────────────
  filterBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingBottom: 10,
    gap: 8,
  },

  // Open Now pill — Airbnb-style: charcoal fill when active
  openNowPill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: 12,
    paddingVertical: 7,
    borderRadius: 9999,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.surface.DEFAULT,
    minHeight: 34,
  },
  openNowPillActive: {
    borderColor: customerColors.charcoal.DEFAULT,
    backgroundColor: customerColors.charcoal.DEFAULT,
  },
  openNowDot: {
    width: 7,
    height: 7,
    borderRadius: 4,
  },
  // Accent discipline: the SELECTED state is the accent — the idle dot stays
  // neutral so the chip doesn't compete with the dock pill / cart pill.
  openNowDotActive: {
    backgroundColor: customerColors.coral.DEFAULT,
  },
  openNowDotInactive: {
    backgroundColor: customerColors.charcoal.soft,
    opacity: 0.5,
  },
  openNowLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
  },
  openNowLabelActive: {
    color: customerColors.canvas,
  },
  openNowLabelDefault: {
    color: customerColors.charcoal.soft,
  },

  // Filters pill — shows active-count badge when secondary filters are applied
  filtersPill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 5,
    paddingHorizontal: 12,
    paddingVertical: 7,
    borderRadius: 9999,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.surface.DEFAULT,
    minHeight: 34,
  },
  filtersPillActive: {
    borderColor: customerColors.charcoal.DEFAULT,
    backgroundColor: customerColors.charcoal.DEFAULT,
  },
  filtersPillLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    color: customerColors.charcoal.soft,
  },
  filtersPillLabelActive: {
    color: customerColors.canvas,
  },
  filterBadge: {
    width: 16,
    height: 16,
    borderRadius: 8,
    backgroundColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  filterBadgeText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    color: customerColors.canvas,
    fontVariant: ['tabular-nums'],
  },

  // Social Feed + Catering nav links — pushed to the right end of the filter bar
  navLinks: {
    flex: 1,
    flexDirection: 'row',
    justifyContent: 'flex-end',
    gap: 6,
  },
  navLinkPill: {
    paddingHorizontal: 10,
    paddingVertical: 7,
    borderRadius: 9999,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.surface.DEFAULT,
    minHeight: 34,
    justifyContent: 'center',
  },
  navLinkLabel: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.DEFAULT,
  },

  // ── List layout ───────────────────────────────────────────────────────────
  columnWrapper: {
    gap: 12,
    paddingHorizontal: 16,
  },
  // Bottom padding is applied dynamically (useDockClearance) so the last
  // row scrolls clear of the floating dock.
  listContent: {
    gap: 12,
    paddingTop: 4,
  },

  // ── Empty state ───────────────────────────────────────────────────────────
  emptyState: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 64,
  },
  emptyTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.charcoal.DEFAULT,
    marginBottom: 4,
  },
  emptyBody: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
  },

  // ── Skeleton grid ─────────────────────────────────────────────────────────
  skeletonGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 12,
    paddingHorizontal: 16,
    paddingTop: 4,
  },
  skeletonCol: {
    flex: 1,
    minWidth: '45%',
  },
  skeletonCard: {
    flex: 1,
    borderRadius: 12,
    overflow: 'hidden',
    backgroundColor: customerColors.surface.soft,
  },
  skeletonPhoto: {
    width: '100%',
    aspectRatio: 4 / 3,
    backgroundColor: customerColors.hairline,
  },
  skeletonBody: {
    padding: 10,
    gap: 6,
  },
  skeletonLine1: {
    height: 14,
    width: '75%',
    borderRadius: 4,
    backgroundColor: customerColors.hairline,
  },
  skeletonLine2: {
    height: 12,
    width: '55%',
    borderRadius: 4,
    backgroundColor: customerColors.hairline,
  },
  skeletonLine3: {
    height: 12,
    width: '40%',
    borderRadius: 4,
    backgroundColor: customerColors.hairline,
  },
});
