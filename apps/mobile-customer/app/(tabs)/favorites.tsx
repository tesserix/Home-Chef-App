import React, { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  FlatList,
  Pressable,
  RefreshControl,
  SafeAreaView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { Image } from 'expo-image';
import { useRouter } from 'expo-router';
import { Heart, Star } from 'lucide-react-native';
import { useFavorites, useToggleFavorite } from '../../hooks/useFavorites';
import type { FavoriteChefEntry } from '../../hooks/useFavorites';

function FavoriteChefCard({
  entry,
  onUnfavorite,
}: {
  entry: FavoriteChefEntry;
  onUnfavorite: (chefId: string) => void;
}) {
  const router = useRouter();
  const chef = entry.chef;

  return (
    <Pressable
      style={styles.card}
      onPress={() => router.push(`/chef/${chef.id}`)}
      accessibilityLabel={`View ${chef.name}`}
    >
      <Image
        source={chef.imageUrl ? { uri: chef.imageUrl } : null}
        style={styles.cardImage}
        contentFit="cover"
        placeholder={{ blurhash: 'L6PZfSi_.AyE_3t7t7R**0o#DgR4' }}
        transition={200}
      />

      <View style={styles.cardBody}>
        <View style={styles.cardHeader}>
          <Text style={styles.chefName} numberOfLines={1}>
            {chef.name}
          </Text>
          <Pressable
            onPress={() => onUnfavorite(chef.id)}
            hitSlop={8}
            accessibilityLabel={`Remove ${chef.name} from favorites`}
          >
            <Heart size={20} color="#EF4444" fill="#EF4444" />
          </Pressable>
        </View>

        <Text style={styles.cuisine} numberOfLines={1}>
          {chef.cuisine}
        </Text>

        <View style={styles.metaRow}>
          <Star size={12} color="#F59E0B" fill="#F59E0B" />
          <Text style={styles.rating}>{chef.rating.toFixed(1)}</Text>
          <Text style={styles.reviewCount}>({chef.reviewCount})</Text>

          <View
            style={[
              styles.openBadge,
              { backgroundColor: chef.isOpen ? '#22C55E' : '#9CA3AF' },
            ]}
          >
            <Text style={styles.openBadgeText}>
              {chef.isOpen ? 'Open' : 'Closed'}
            </Text>
          </View>
        </View>
      </View>
    </Pressable>
  );
}

export default function FavoritesScreen() {
  const { data, isLoading, isRefetching, refetch } = useFavorites();
  const toggleFavorite = useToggleFavorite();
  // Optimistic local state — list of chef IDs being removed
  const [removingIds, setRemovingIds] = useState<Set<string>>(new Set());

  function handleUnfavorite(chefId: string) {
    // Optimistic removal
    setRemovingIds((prev) => new Set([...prev, chefId]));

    toggleFavorite.mutate(
      { chefId, isFavorited: true },
      {
        onError: () => {
          // Restore on error
          setRemovingIds((prev) => {
            const next = new Set(prev);
            next.delete(chefId);
            return next;
          });
          Alert.alert('Error', 'Could not remove from favorites. Please try again.');
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

  if (isLoading) {
    return (
      <SafeAreaView style={styles.container}>
        <View style={styles.centered}>
          <ActivityIndicator size="large" color="#F97316" />
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.title}>Saved Chefs</Text>
        {data && (
          <Text style={styles.subtitle}>
            {visibleEntries.length}/{data.max} saved
          </Text>
        )}
      </View>

      <FlatList
        data={visibleEntries}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => (
          <FavoriteChefCard entry={item} onUnfavorite={handleUnfavorite} />
        )}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={() => void refetch()}
            tintColor="#F97316"
          />
        }
        ListEmptyComponent={
          <View style={styles.emptyState}>
            <Text style={styles.emptyIcon}>❤️</Text>
            <Text style={styles.emptyTitle}>No saved chefs yet</Text>
            <Text style={styles.emptySubtitle}>
              Tap the heart on any chef to save them!
            </Text>
          </View>
        }
        contentContainerStyle={
          visibleEntries.length === 0 ? styles.emptyContent : styles.listContent
        }
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F9FAFB',
  },
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  header: {
    paddingHorizontal: 16,
    paddingTop: 8,
    paddingBottom: 4,
    flexDirection: 'row',
    alignItems: 'baseline',
    gap: 8,
  },
  title: {
    fontSize: 24,
    fontWeight: '700',
    color: '#111827',
  },
  subtitle: {
    fontSize: 13,
    color: '#9CA3AF',
  },
  listContent: {
    padding: 16,
    gap: 12,
  },
  emptyContent: {
    flex: 1,
  },
  emptyState: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingTop: 80,
  },
  emptyIcon: {
    fontSize: 48,
    marginBottom: 16,
  },
  emptyTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#374151',
    marginBottom: 8,
  },
  emptySubtitle: {
    fontSize: 14,
    color: '#9CA3AF',
    textAlign: 'center',
    paddingHorizontal: 40,
  },
  // Card styles
  card: {
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    overflow: 'hidden',
    marginBottom: 12,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.08,
    shadowRadius: 4,
    elevation: 2,
  },
  cardImage: {
    width: '100%',
    height: 140,
  },
  cardBody: {
    padding: 12,
    gap: 4,
  },
  cardHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  chefName: {
    fontSize: 16,
    fontWeight: '700',
    color: '#111827',
    flex: 1,
    marginRight: 8,
  },
  cuisine: {
    fontSize: 13,
    color: '#6B7280',
  },
  metaRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    marginTop: 4,
  },
  rating: {
    fontSize: 12,
    fontWeight: '600',
    color: '#374151',
  },
  reviewCount: {
    fontSize: 12,
    color: '#9CA3AF',
  },
  openBadge: {
    marginLeft: 'auto',
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 12,
  },
  openBadgeText: {
    fontSize: 11,
    color: '#FFFFFF',
    fontWeight: '600',
  },
});
