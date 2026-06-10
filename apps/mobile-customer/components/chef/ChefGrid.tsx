import { FlatList, RefreshControl, StyleSheet, View } from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ChefCard } from './ChefCard';
import type { Chef } from '../../types/customer';

interface ChefGridProps {
  chefs: Chef[];
  isLoading: boolean;
  onRefresh?: () => void;
  isRefreshing?: boolean;
}

// Skeleton matching the photo-led 4:3 ChefCard shape.
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

export function ChefGrid({
  chefs,
  isLoading,
  onRefresh,
  isRefreshing = false,
}: ChefGridProps) {
  if (isLoading) {
    return (
      <View style={styles.skeletonGrid}>
        <View style={styles.skeletonCol}><SkeletonCard /></View>
        <View style={styles.skeletonCol}><SkeletonCard /></View>
        <View style={styles.skeletonCol}><SkeletonCard /></View>
        <View style={styles.skeletonCol}><SkeletonCard /></View>
      </View>
    );
  }

  return (
    <FlatList
      data={chefs}
      keyExtractor={(item) => item.id}
      numColumns={2}
      columnWrapperStyle={styles.columnWrapper}
      contentContainerStyle={styles.listContent}
      renderItem={({ item }) => <ChefCard chef={item} />}
      refreshControl={
        onRefresh ? (
          <RefreshControl
            refreshing={isRefreshing}
            onRefresh={onRefresh}
            tintColor={customerColors.coral.DEFAULT}
          />
        ) : undefined
      }
    />
  );
}

const styles = StyleSheet.create({
  columnWrapper: {
    gap: 12,
    paddingHorizontal: 16,
  },
  listContent: {
    paddingBottom: 100,
    gap: 12,
    paddingTop: 8,
  },

  // Skeleton grid — matches the 2-col layout
  skeletonGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 12,
    paddingHorizontal: 16,
    paddingTop: 8,
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
