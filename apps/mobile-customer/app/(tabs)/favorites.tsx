import React, { useState } from 'react';
import {
  Alert,
  FlatList,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { Image } from 'expo-image';
import { Heart, ChefHat, Star, ChevronRight } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useFavorites, useToggleFavorite, useFavoriteDishes } from '../../hooks/useFavorites';
import type { FavoriteChefEntry, FavoriteDishEntry } from '../../hooks/useFavorites';
import type { Chef } from '../../types/customer';
import { friendlyErrorMessage } from '../../lib/errors';
import { useDockClearance } from '../../components/navigation/Dock';
import { ScreenTitle } from '../../components/shared/ScreenTitle';
import { MenuItemCard } from '../../components/chef/MenuItemCard';

// ─── Compact saved-chef row ──────────────────────────────────────────────────
// The Saved tab is a shortlist, not a discovery feed, so it uses a dense
// horizontal row (thumbnail + a line of metadata) rather than the big photo card
// from Home — more chefs per screen, less scrolling, cleaner. The thumbnail is
// the same chef image downscaled by expo-image, and the whole row taps through to
// the full chef page.

function SavedChefRow({
  chef,
  onUnsave,
  unsaving,
}: {
  chef: Chef;
  onUnsave: () => void;
  unsaving: boolean;
}) {
  const router = useRouter();
  const initial = (chef.name?.trim()[0] ?? '·').toUpperCase();

  return (
    <Pressable
      onPress={() => router.push(`/chef/${chef.id}`)}
      accessibilityRole="button"
      accessibilityLabel={`View ${chef.name}`}
      style={({ pressed }) => [styles.row, pressed && styles.rowPressed]}
    >
      {/* Thumbnail — downscaled from the same image the chef page shows. */}
      <View style={styles.thumb}>
        {chef.imageUrl ? (
          <Image
            source={{ uri: chef.imageUrl }}
            style={styles.thumbImg}
            contentFit="cover"
            transition={150}
          />
        ) : (
          <View style={[styles.thumbImg, styles.thumbFallback]}>
            <Text style={styles.thumbInitial}>{initial}</Text>
          </View>
        )}
      </View>

      {/* Metadata — name, rating, cuisine, open/min on one compact stack. */}
      <View style={styles.meta}>
        <Text style={styles.name} numberOfLines={1}>
          {chef.name}
        </Text>
        <View style={styles.ratingRow}>
          <Star size={12} color={customerColors.charcoal.DEFAULT} fill={customerColors.charcoal.DEFAULT} />
          <Text style={styles.rating}>
            {chef.rating.toFixed(1)}
            {chef.reviewCount > 0 ? ` (${chef.reviewCount})` : ''}
          </Text>
          {chef.cuisine ? (
            <>
              <Text style={styles.dot}>·</Text>
              <Text style={styles.cuisine} numberOfLines={1}>
                {chef.cuisine}
              </Text>
            </>
          ) : null}
        </View>
        <View style={styles.statusRow}>
          <View
            style={[
              styles.openDot,
              chef.isOpen ? styles.openDotOpen : styles.openDotClosed,
            ]}
          />
          <Text style={styles.statusText}>
            {chef.isOpen ? 'Open' : 'Closed'}
            {chef.minimumOrder != null ? ` · Min ₹${chef.minimumOrder}` : ''}
          </Text>
        </View>
      </View>

      {/* Unsave — hitSlop keeps the 44px target without bloating the row. */}
      <Pressable
        onPress={onUnsave}
        disabled={unsaving}
        hitSlop={12}
        accessibilityRole="button"
        accessibilityLabel={`Remove ${chef.name} from saved`}
        style={styles.heartBtn}
      >
        <Heart
          size={20}
          color={customerColors.coral.DEFAULT}
          fill={customerColors.coral.DEFAULT}
        />
      </Pressable>
      <ChevronRight size={18} color={customerColors.charcoal.soft} />
    </Pressable>
  );
}

// ─── Loading skeleton ────────────────────────────────────────────────────────

function SkeletonCard() {
  return (
    <View className="flex-1 rounded-xl overflow-hidden bg-surface-soft">
      <View className="w-full h-40 bg-hairline" />
      <View className="p-3 gap-2">
        <View className="h-4 rounded bg-hairline w-3/4" />
        <View className="h-3 rounded bg-hairline w-1/2" />
        <View className="h-3 rounded bg-hairline w-2/5" />
      </View>
    </View>
  );
}

function LoadingGrid() {
  return (
    <View className="flex-1 bg-canvas px-4 pt-4">
      {/* Header skeleton */}
      <View className="mb-4 gap-2">
        <View className="h-7 rounded bg-hairline w-36" />
        <View className="h-4 rounded bg-hairline w-20" />
      </View>
      {/* Card skeleton rows */}
      <View className="flex-row gap-3 mb-3">
        <SkeletonCard />
        <SkeletonCard />
      </View>
      <View className="flex-row gap-3 mb-3">
        <SkeletonCard />
        <SkeletonCard />
      </View>
    </View>
  );
}

// ─── Error state ─────────────────────────────────────────────────────────────

function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <View className="flex-1 items-center justify-center px-8 gap-4">
      <View className="w-16 h-16 rounded-full bg-surface-soft items-center justify-center">
        <Heart size={28} color="#EBEBEB" />
      </View>
      <Text className="text-lg font-semibold text-charcoal text-center font-display">
        Something went wrong
      </Text>
      <Text className="text-sm text-charcoal-soft text-center">
        We could not load your saved chefs. Please try again.
      </Text>
      {/* Coral CTA — visual styles on inner View per iOS Pressable bug */}
      <Pressable onPress={onRetry} accessibilityRole="button" accessibilityLabel="Retry loading favorites">
        <View className="bg-coral rounded-lg px-6 py-3 min-h-[44px] items-center justify-center">
          <Text className="text-canvas font-semibold text-sm">Try again</Text>
        </View>
      </Pressable>
    </View>
  );
}

// ─── Empty state ─────────────────────────────────────────────────────────────

function EmptyState() {
  const router = useRouter();
  return (
    <View className="flex-1 items-center justify-center px-8 gap-4 pt-16">
      {/* Surface-soft icon circle */}
      <View className="w-20 h-20 rounded-full bg-surface-soft items-center justify-center">
        <ChefHat size={36} color="#717171" />
      </View>
      <View className="items-center gap-2">
        <Text className="text-xl font-bold text-charcoal text-center font-display">
          No saved chefs yet
        </Text>
        <Text className="text-sm text-charcoal-soft text-center leading-5">
          Tap the heart on any chef card to save them here for easy ordering.
        </Text>
      </View>
      {/* Coral "Browse chefs" CTA — visual styles on inner View */}
      <Pressable
        onPress={() => router.push('/(tabs)')}
        accessibilityRole="button"
        accessibilityLabel="Browse chefs"
      >
        <View className="bg-coral rounded-lg px-8 py-3 min-h-[44px] items-center justify-center mt-2">
          <Text className="text-canvas font-semibold text-base">Browse chefs</Text>
        </View>
      </Pressable>
    </View>
  );
}

// Saved-dishes empty state (#237) — distinct copy from the chefs empty state.
function EmptyDishesState() {
  const router = useRouter();
  return (
    <View className="flex-1 items-center justify-center px-8 gap-4 pt-16">
      <View className="w-20 h-20 rounded-full bg-surface-soft items-center justify-center">
        <Heart size={34} color="#717171" />
      </View>
      <View className="items-center gap-2">
        <Text className="text-xl font-bold text-charcoal text-center font-display">
          No saved dishes yet
        </Text>
        <Text className="text-sm text-charcoal-soft text-center leading-5">
          Tap the heart on any dish to save it here for one-tap ordering later.
        </Text>
      </View>
      <Pressable
        onPress={() => router.push('/(tabs)')}
        accessibilityRole="button"
        accessibilityLabel="Browse chefs"
      >
        <View className="bg-coral rounded-lg px-8 py-3 min-h-[44px] items-center justify-center mt-2">
          <Text className="text-canvas font-semibold text-base">Browse chefs</Text>
        </View>
      </Pressable>
    </View>
  );
}

// ─── Segmented control (Chefs | Dishes) ──────────────────────────────────────

type FavTab = 'chefs' | 'dishes';

function FavoriteTabs({
  tab,
  onChange,
  chefCount,
  dishCount,
}: {
  tab: FavTab;
  onChange: (t: FavTab) => void;
  chefCount?: number;
  dishCount?: number;
}) {
  const segments: { key: FavTab; label: string; count?: number }[] = [
    { key: 'chefs', label: 'Chefs', count: chefCount },
    { key: 'dishes', label: 'Dishes', count: dishCount },
  ];
  return (
    <View className="flex-row gap-2 px-4 pb-2">
      {segments.map((s) => {
        const active = tab === s.key;
        return (
          <Pressable
            key={s.key}
            onPress={() => onChange(s.key)}
            accessibilityRole="tab"
            accessibilityState={{ selected: active }}
            accessibilityLabel={`${s.label}${s.count != null ? `, ${s.count}` : ''}`}
          >
            {/* Dock language: active segment = coral-tint pill + coral text
                (same DNA as the dock's active tab), not a solid coral block. */}
            <View
              className={`rounded-full px-4 py-1.5 ${active ? 'bg-coral-tint' : 'bg-surface-soft'}`}
            >
              <Text
                className={`text-sm font-semibold ${active ? 'text-coral' : 'text-charcoal-soft'}`}
              >
                {s.label}
                {s.count != null ? ` ${s.count}` : ''}
              </Text>
            </View>
          </Pressable>
        );
      })}
    </View>
  );
}

// ─── Main screen ─────────────────────────────────────────────────────────────

export default function FavoritesScreen() {
  const dockClearance = useDockClearance();
  const [tab, setTab] = useState<FavTab>('chefs');

  const chefs = useFavorites();
  const dishes = useFavoriteDishes();
  const toggleFavorite = useToggleFavorite();
  // Optimistic local state — set of chef IDs currently being removed
  const [removingIds, setRemovingIds] = useState<Set<string>>(new Set());

  function handleUnfavorite(chefId: string) {
    setRemovingIds((prev) => new Set([...prev, chefId]));

    toggleFavorite.mutate(
      { chefId, isFavorited: true },
      {
        onError: (error) => {
          setRemovingIds((prev) => {
            const next = new Set(prev);
            next.delete(chefId);
            return next;
          });
          Alert.alert(
            'Error',
            friendlyErrorMessage(error, 'Could not remove from favorites. Please try again.'),
          );
        },
        onSuccess: () => {
          setRemovingIds((prev) => {
            const next = new Set(prev);
            next.delete(chefId);
            return next;
          });
        },
      },
    );
  }

  const visibleEntries =
    chefs.data?.data.filter((e) => !removingIds.has(e.chefId)) ?? [];
  const dishEntries = dishes.data?.data ?? [];

  // First-load skeleton only for the active tab's query.
  const activeLoading = tab === 'chefs' ? chefs.isLoading : dishes.isLoading;
  if (activeLoading) {
    return (
      <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
        <Header />
        <FavoriteTabs tab={tab} onChange={setTab} chefCount={chefs.data?.count} dishCount={dishes.data?.count} />
        <LoadingGrid />
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      <Header />
      <FavoriteTabs
        tab={tab}
        onChange={setTab}
        chefCount={chefs.data?.count}
        dishCount={dishes.data?.count}
      />

      {tab === 'chefs' ? (
        chefs.isError ? (
          <ErrorState onRetry={() => void chefs.refetch()} />
        ) : (
          <FlatList<FavoriteChefEntry>
            data={visibleEntries}
            keyExtractor={(item) => item.id}
            contentContainerStyle={
              visibleEntries.length === 0
                ? { flexGrow: 1 }
                : { paddingTop: 8, paddingHorizontal: 16, paddingBottom: dockClearance }
            }
            ItemSeparatorComponent={() => <View style={styles.rowSep} />}
            refreshControl={
              <RefreshControl
                refreshing={chefs.isRefetching}
                onRefresh={() => void chefs.refetch()}
                tintColor="#FF385C"
              />
            }
            renderItem={({ item }) => (
              <SavedChefRow
                chef={item.chef}
                onUnsave={() => handleUnfavorite(item.chefId)}
                unsaving={removingIds.has(item.chefId)}
              />
            )}
            ListEmptyComponent={<EmptyState />}
          />
        )
      ) : dishes.isError ? (
        <ErrorState onRetry={() => void dishes.refetch()} />
      ) : (
        <FlatList<FavoriteDishEntry>
          data={dishEntries}
          keyExtractor={(item) => item.id}
          contentContainerStyle={
            dishEntries.length === 0
              ? { flexGrow: 1 }
              : { paddingHorizontal: 16, paddingBottom: dockClearance }
          }
          refreshControl={
            <RefreshControl
              refreshing={dishes.isRefetching}
              onRefresh={() => void dishes.refetch()}
              tintColor="#FF385C"
            />
          }
          renderItem={({ item }) => (
            <MenuItemCard
              item={item.menuItem}
              chefId={item.chef.id}
              chefName={item.chef.businessName}
            />
          )}
          ListEmptyComponent={<EmptyDishesState />}
        />
      )}
    </SafeAreaView>
  );
}

// Shared header — the per-tab counts live on the segment chips below it.
// Titled "Saved" to match the tab label in the dock.
function Header() {
  return <ScreenTitle title="Saved" />;
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingVertical: 10,
  },
  rowPressed: { opacity: 0.6 },
  rowSep: { height: StyleSheet.hairlineWidth, backgroundColor: customerColors.hairline },
  thumb: { width: 72, height: 72, borderRadius: 12, overflow: 'hidden' },
  thumbImg: { width: '100%', height: '100%' },
  thumbFallback: {
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.surface.soft,
  },
  thumbInitial: {
    fontFamily: 'Geist-Bold',
    fontSize: 26,
    color: customerColors.charcoal.soft,
  },
  meta: { flex: 1, gap: 3 },
  name: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  ratingRow: { flexDirection: 'row', alignItems: 'center', gap: 4 },
  rating: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.DEFAULT },
  dot: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft },
  cuisine: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, flexShrink: 1 },
  statusRow: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  openDot: { width: 7, height: 7, borderRadius: 4 },
  openDotOpen: { backgroundColor: customerColors.success.DEFAULT },
  openDotClosed: { backgroundColor: customerColors.charcoal.soft },
  statusText: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft },
  heartBtn: { padding: 2 },
})
