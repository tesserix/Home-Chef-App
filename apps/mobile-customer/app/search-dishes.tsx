import React, { useEffect, useRef, useState } from 'react';
import { FlatList, Platform, Pressable, StyleSheet, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { Image } from 'expo-image';
import { ChefHat, ChevronRight, Search, UtensilsCrossed } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ScreenHeader } from '../components/ScreenHeader';
import { useSearchDishes, type DishResult } from '../hooks/useSearchDishes';
import { useChefs } from '../hooks/useChefs';
import type { Chef } from '../types/customer';

// Android ripple tints — translucent tokens derived from existing colours,
// never a new literal colour (matches the ChefCard `withAlpha` convention).
const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const CTA_RIPPLE = `${customerColors.canvas}33`;

function formatMoney(amount: number): string {
  try {
    return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(amount);
  } catch {
    return `₹${amount.toFixed(0)}`;
  }
}

// ─── Dish result row ──────────────────────────────────────────────────────────
// Photo on the right (radius 12, R2 fallback), tabular price, and a wayfinding
// "attribution" caption. The dish-search endpoint (GET /v1/search/dishes)
// returns the owning chefId but not the chef's name — the row still routes to
// that chef's page on tap, so the caption stays honest about what happens
// next rather than inventing a name the API doesn't provide.
function DishRow({ item, onPress }: { item: DishResult; onPress: () => void }) {
  const hasImage = Boolean(item.imageUrl);
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={`View ${item.name}`}
      android_ripple={{ color: ROW_RIPPLE, borderless: false }}
    >
      {({ pressed }) => (
        <View
          style={[styles.dishRow, pressed && Platform.OS === 'ios' && styles.rowPressedIOS]}
        >
          <View style={styles.dishTextCol}>
            <Text style={styles.dishName} numberOfLines={1}>
              {item.name}
            </Text>
            {item.description ? (
              <Text style={styles.dishDescription} numberOfLines={1}>
                {item.description}
              </Text>
            ) : null}
            <View style={styles.dishMetaRow}>
              <Text style={styles.dishPrice}>{formatMoney(item.price)}</Text>
              {item.rating ? (
                <>
                  <Text style={styles.dishMetaDot}>·</Text>
                  <Text style={styles.dishRating}>★ {item.rating.toFixed(1)}</Text>
                </>
              ) : null}
            </View>
            {/* Attribution / wayfinding line — the row navigates to the chef
                that makes this dish. */}
            <View style={styles.attributionRow}>
              <ChefHat size={12} color={customerColors.charcoal.soft} />
              <Text style={styles.attributionText}>View this chef's menu</Text>
            </View>
          </View>

          <View style={styles.dishPhotoWrap}>
            {hasImage ? (
              <Image
                source={{ uri: item.imageUrl }}
                style={styles.dishPhoto}
                contentFit="cover"
                placeholder={{ blurhash: 'L6PZfSi_.AyE_3t7t7R**0o#DgR4' }}
                transition={150}
                accessibilityElementsHidden
              />
            ) : (
              <View style={[styles.dishPhoto, styles.dishPhotoPlaceholder]}>
                <UtensilsCrossed size={20} color={customerColors.charcoal.soft} />
              </View>
            )}
          </View>
        </View>
      )}
    </Pressable>
  );
}

// ─── Chef result row ──────────────────────────────────────────────────────────
// Deliberately avatar-less (kept light — chef search is a lighter-weight
// shortcut than the dish list); adds ripple + chevron for a consistent
// navigable-row affordance with the rest of the app.
function ChefRow({ chef, onPress }: { chef: Chef; onPress: () => void }) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={`View chef ${chef.name}`}
      android_ripple={{ color: ROW_RIPPLE, borderless: false }}
    >
      {({ pressed }) => (
        <View style={[styles.chefRow, pressed && Platform.OS === 'ios' && styles.rowPressedIOS]}>
          <View style={styles.chefAvatar}>
            <ChefHat size={16} color={customerColors.charcoal.soft} />
          </View>
          <View style={styles.chefTextCol}>
            <Text style={styles.chefName} numberOfLines={1}>
              {chef.name}
            </Text>
            <Text style={styles.chefCuisine} numberOfLines={1}>
              {chef.cuisine}
            </Text>
          </View>
          {chef.rating > 0 ? (
            <Text style={styles.chefRating}>★ {chef.rating.toFixed(1)}</Text>
          ) : null}
          <ChevronRight size={16} color={customerColors.charcoal.soft} />
        </View>
      )}
    </Pressable>
  );
}

function SectionLabel({ label }: { label: string }) {
  return <Text style={styles.sectionLabel}>{label}</Text>;
}

// ─── Skeleton rows (R8 — no full-screen spinners) ─────────────────────────────
function SkeletonRow() {
  return (
    <View style={styles.dishRow}>
      <View style={styles.dishTextCol}>
        <View style={styles.skeletonLine1} />
        <View style={styles.skeletonLine2} />
        <View style={styles.skeletonLine3} />
      </View>
      <View style={[styles.dishPhoto, styles.skeletonPhoto]} />
    </View>
  );
}

function ResultsSkeleton() {
  return (
    <View>
      <SkeletonRow />
      <SkeletonRow />
      <SkeletonRow />
    </View>
  );
}

// ─── Error / retry state (R8 triad) ───────────────────────────────────────────
function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <View style={styles.centeredState}>
      <View style={styles.stateIconWrap}>
        <Search size={26} color={customerColors.charcoal.soft} />
      </View>
      <Text style={styles.stateTitle}>Something went wrong</Text>
      <Text style={styles.stateBody}>We couldn't load results. Please try again.</Text>
      <Pressable
        onPress={onRetry}
        accessibilityRole="button"
        accessibilityLabel="Retry search"
        android_ripple={{ color: CTA_RIPPLE, borderless: false }}
      >
        <View style={styles.retryButton}>
          <Text style={styles.retryLabel}>Try again</Text>
        </View>
      </Pressable>
    </View>
  );
}

export default function SearchDishesScreen() {
  const router = useRouter();
  const { q: initialQ } = useLocalSearchParams<{ q?: string }>();
  const [text, setText] = useState(initialQ ?? '');
  const [query, setQuery] = useState(initialQ ?? '');
  const [focused, setFocused] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Debounce the query so we don't hit the API on every keystroke.
  useEffect(() => {
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => setQuery(text), 350);
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, [text]);

  // Unified search (home's single search entry): chefs by name alongside
  // dishes. Chef matches render as a compact section above the dish list.
  const chefQueryActive = query.trim().length >= 2;
  const chefsQuery = useChefs(
    chefQueryActive ? { search: query.trim(), limit: 5 } : { limit: 0 },
  );
  const dishesQuery = useSearchDishes(query);

  // ── R13 zero-flicker: keep the previous results rendered while a new
  // debounced query is in flight, instead of blanking to empty while
  // useQuery's `data` is undefined between key changes. Component-side only
  // — no change to the hooks' query keys/contracts. Cleared once the query
  // drops below the 2-char search threshold, since that renders a different
  // ("type to search") branch anyway. ──
  const [stableDishes, setStableDishes] = useState<DishResult[]>([]);
  const [stableChefs, setStableChefs] = useState<Chef[]>([]);
  useEffect(() => {
    if (dishesQuery.data) setStableDishes(dishesQuery.data);
  }, [dishesQuery.data]);
  useEffect(() => {
    if (chefQueryActive && chefsQuery.data) setStableChefs(chefsQuery.data.data ?? []);
  }, [chefQueryActive, chefsQuery.data]);
  useEffect(() => {
    if (!chefQueryActive) {
      setStableDishes([]);
      setStableChefs([]);
    }
  }, [chefQueryActive]);

  const isFirstLoad =
    chefQueryActive && dishesQuery.isLoading && stableDishes.length === 0 && stableChefs.length === 0;
  const isError = chefQueryActive && (dishesQuery.isError || chefsQuery.isError);
  const showEmpty =
    chefQueryActive &&
    !isFirstLoad &&
    !isError &&
    !dishesQuery.isFetching &&
    stableDishes.length === 0 &&
    stableChefs.length === 0;

  function handleRetry() {
    void dishesQuery.refetch();
    void chefsQuery.refetch();
  }

  return (
    <SafeAreaView edges={['top', 'left', 'right']} style={styles.screen}>
      <ScreenHeader title="Search" />
      <View style={styles.searchBarWrap}>
        <View style={[styles.searchBar, focused && styles.searchBarFocused]}>
          <Search size={18} color={customerColors.charcoal.soft} />
          <TextInput
            value={text}
            onChangeText={setText}
            onFocus={() => setFocused(true)}
            onBlur={() => setFocused(false)}
            autoFocus={!initialQ}
            placeholder="What are you craving?"
            placeholderTextColor={customerColors.charcoal.soft}
            style={styles.searchInput}
            returnKeyType="search"
            accessibilityLabel="Search dishes and chefs"
          />
        </View>
      </View>

      {isFirstLoad ? (
        <ResultsSkeleton />
      ) : isError ? (
        <ErrorState onRetry={handleRetry} />
      ) : showEmpty ? (
        <View style={styles.centeredState}>
          <View style={styles.stateIconWrap}>
            <UtensilsCrossed size={26} color={customerColors.charcoal.soft} />
          </View>
          <Text style={styles.stateTitle}>Nothing found</Text>
          <Text style={styles.stateBody}>
            No dishes or chefs matched “{query}”. Try a different search.
          </Text>
        </View>
      ) : !chefQueryActive ? (
        <View style={styles.centeredState}>
          <Text style={styles.hintText}>Type at least 2 characters to search dishes and chefs.</Text>
        </View>
      ) : (
        <FlatList
          data={stableDishes}
          keyExtractor={(d) => d.id}
          renderItem={({ item }) => (
            <DishRow item={item} onPress={() => item.chefId && router.push(`/chef/${item.chefId}`)} />
          )}
          keyboardShouldPersistTaps="handled"
          ListHeaderComponent={
            <>
              {stableChefs.length > 0 && (
                <View>
                  <SectionLabel label="Chefs" />
                  {stableChefs.map((chef) => (
                    <ChefRow key={chef.id} chef={chef} onPress={() => router.push(`/chef/${chef.id}`)} />
                  ))}
                </View>
              )}
              {stableDishes.length > 0 && <SectionLabel label="Dishes" />}
            </>
          }
        />
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: customerColors.canvas },

  // ── Search pill — radius-full, visible 2px coral focus ring ──
  searchBarWrap: { padding: 16 },
  searchBar: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: customerColors.surface.soft,
    borderRadius: 9999,
    paddingHorizontal: 16,
    height: 44,
    borderWidth: 2,
    borderColor: 'transparent',
  },
  searchBarFocused: {
    borderColor: customerColors.coral.DEFAULT,
  },
  searchInput: {
    flex: 1,
    marginLeft: 8,
    fontFamily: 'Inter',
    fontSize: 16,
    color: customerColors.charcoal.DEFAULT,
  },

  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    textTransform: 'uppercase',
    letterSpacing: 0.4,
    paddingHorizontal: 16,
    paddingTop: 16,
    paddingBottom: 6,
  },

  rowPressedIOS: { backgroundColor: customerColors.surface.soft },

  // ── Dish row ──
  dishRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 10,
    gap: 12,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  dishTextCol: { flex: 1, gap: 2 },
  dishName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },
  dishDescription: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
  },
  dishMetaRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    marginTop: 2,
  },
  dishPrice: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  dishMetaDot: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft },
  dishRating: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
  attributionRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    marginTop: 3,
  },
  attributionText: {
    fontFamily: 'Inter',
    fontSize: 11,
    color: customerColors.charcoal.soft,
  },
  dishPhotoWrap: { flexShrink: 0 },
  dishPhoto: {
    width: 56,
    height: 56,
    borderRadius: 12,
  },
  dishPhotoPlaceholder: {
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
  },

  // ── Chef row ──
  chefRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  chefAvatar: {
    width: 34,
    height: 34,
    borderRadius: 17,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
  },
  chefTextCol: { flex: 1 },
  chefName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },
  chefCuisine: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    marginTop: 1,
  },
  chefRating: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },

  // ── Skeleton ──
  skeletonLine1: { height: 14, width: '70%', borderRadius: 4, backgroundColor: customerColors.hairline, marginBottom: 6 },
  skeletonLine2: { height: 11, width: '45%', borderRadius: 4, backgroundColor: customerColors.hairline, marginBottom: 6 },
  skeletonLine3: { height: 11, width: '30%', borderRadius: 4, backgroundColor: customerColors.hairline },
  skeletonPhoto: { backgroundColor: customerColors.hairline },

  // ── Centered states (error / empty / hint) ──
  centeredState: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 32,
    gap: 8,
    paddingTop: 32,
  },
  stateIconWrap: {
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 4,
  },
  stateTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 17,
    color: customerColors.charcoal.DEFAULT,
    textAlign: 'center',
  },
  stateBody: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
    lineHeight: 18,
  },
  hintText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
  },
  retryButton: {
    marginTop: 8,
    backgroundColor: customerColors.coral.DEFAULT,
    borderRadius: 8,
    paddingHorizontal: 20,
    minHeight: 44,
    alignItems: 'center',
    justifyContent: 'center',
  },
  retryLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.canvas,
  },
});
