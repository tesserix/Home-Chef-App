import { FlatList, RefreshControl, View } from 'react-native';
import { ChefCard } from './ChefCard';
import type { Chef } from '../../types/customer';

interface ChefGridProps {
  chefs: Chef[];
  isLoading: boolean;
  onRefresh?: () => void;
  isRefreshing?: boolean;
}

function SkeletonCard() {
  return (
    <View className="flex-1 rounded-2xl overflow-hidden bg-mist">
      <View className="w-full h-40 bg-mist" />
      <View className="p-3 gap-2">
        <View className="h-4 w-3/4 rounded bg-mist" />
        <View className="h-3 w-1/2 rounded bg-mist" />
        <View className="h-3 w-1/3 rounded bg-mist" />
      </View>
    </View>
  );
}

export function ChefGrid({ chefs, isLoading, onRefresh, isRefreshing = false }: ChefGridProps) {
  if (isLoading) {
    return (
      <View className="flex-row flex-wrap gap-3 px-4 pt-2">
        <View className="flex-1"><SkeletonCard /></View>
        <View className="flex-1"><SkeletonCard /></View>
        <View className="flex-1"><SkeletonCard /></View>
        <View className="flex-1"><SkeletonCard /></View>
      </View>
    );
  }

  return (
    <FlatList
      data={chefs}
      keyExtractor={(item) => item.id}
      numColumns={2}
      columnWrapperStyle={{ gap: 12, paddingHorizontal: 16 }}
      contentContainerStyle={{ paddingBottom: 100, gap: 12, paddingTop: 8 }}
      renderItem={({ item }) => <ChefCard chef={item} />}
      refreshControl={
        onRefresh ? (
          <RefreshControl refreshing={isRefreshing} onRefresh={onRefresh} />
        ) : undefined
      }
    />
  );
}
