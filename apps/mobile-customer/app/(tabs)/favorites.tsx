import React, { useState } from 'react';
import {
  Alert,
  FlatList,
  Pressable,
  RefreshControl,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { Heart, ChefHat } from 'lucide-react-native';
import { useFavorites, useToggleFavorite } from '../../hooks/useFavorites';
import type { FavoriteChefEntry } from '../../hooks/useFavorites';
import { friendlyErrorMessage } from '../../lib/errors';
import { ChefCard } from '../../components/chef/ChefCard';

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

// ─── Main screen ─────────────────────────────────────────────────────────────

export default function FavoritesScreen() {
  const { data, isLoading, isError, isRefetching, refetch } = useFavorites();
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
    data?.data.filter((e) => !removingIds.has(e.chefId)) ?? [];

  // ── Loading ──────────────────────────────────────────────────────────────
  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
        <LoadingGrid />
      </SafeAreaView>
    );
  }

  // ── Error ────────────────────────────────────────────────────────────────
  if (isError) {
    return (
      <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
        <ErrorState onRetry={() => void refetch()} />
      </SafeAreaView>
    );
  }

  // ── List (including empty) ───────────────────────────────────────────────
  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      {/* ── Header ── */}
      <View className="px-4 pt-3 pb-2 flex-row items-baseline gap-2">
        <Text className="text-2xl font-bold text-charcoal tracking-tight font-display">
          Favorites
        </Text>
        {data != null && (
          <Text className="text-sm text-charcoal-soft">
            {visibleEntries.length}/{data.max}
          </Text>
        )}
      </View>

      {/* ── 2-col photo grid ── */}
      <FlatList<FavoriteChefEntry>
        data={visibleEntries}
        keyExtractor={(item) => item.id}
        numColumns={2}
        columnWrapperStyle={{ gap: 12, paddingHorizontal: 16 }}
        contentContainerStyle={
          visibleEntries.length === 0
            ? { flexGrow: 1 }
            : { paddingTop: 8, paddingBottom: 24, gap: 12 }
        }
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={() => void refetch()}
            tintColor="#FF385C"
          />
        }
        renderItem={({ item }) => (
          // flex-1 wrapper so each column cell fills the available width.
          // The Pressable inside ChefCard is already `flex-1`.
          <View className="flex-1">
            <ChefCard chef={item.chef} />
          </View>
        )}
        ListEmptyComponent={<EmptyState />}
      />
    </SafeAreaView>
  );
}
