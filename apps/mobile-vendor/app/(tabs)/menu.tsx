import { useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  RefreshControl,
  ScrollView,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { Plus } from 'lucide-react-native';
import { useVendorMenu, useDeleteMenuItem } from '../../hooks/useVendorMenu';
import type { MenuItem } from '../../hooks/useVendorMenu';
import { MenuItemCard } from '../../components/vendor/MenuItemCard';

function SkeletonCard() {
  return (
    <View className="bg-bone rounded-2xl shadow-sm p-3 mb-2 flex-row animate-pulse">
      <View className="w-20 h-20 rounded-xl bg-mist mr-3" />
      <View className="flex-1">
        <View className="h-4 bg-mist rounded w-2/3 mb-2" />
        <View className="h-3 bg-mist rounded w-1/3 mb-2" />
        <View className="h-3 bg-mist rounded w-1/4" />
      </View>
    </View>
  );
}

export default function MenuScreen() {
  const { data, isLoading, isError, refetch, isRefetching } = useVendorMenu();
  const deleteMutation = useDeleteMenuItem();
  const [selectedCategory, setSelectedCategory] = useState<string>('All');

  const categories = ['All', ...(data?.categories?.map((c) => c.name) ?? [])];

  const filteredItems: MenuItem[] =
    selectedCategory === 'All'
      ? (data?.items ?? [])
      : (data?.items ?? []).filter((item) => item.category === selectedCategory);

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-paper">
        <View className="px-4 pt-4 pb-2">
          <Text className="font-display text-2xl font-semibold text-ink">My Menu</Text>
        </View>
        <View className="px-4 pt-2">
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
        </View>
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView className="flex-1 bg-paper items-center justify-center px-6">
        <Text className="text-ink-muted text-base mb-4">Failed to load menu</Text>
        <TouchableOpacity
          onPress={() => refetch()}
          className="bg-herb px-6 py-3 rounded-xl"
        >
          <Text className="text-paper font-semibold">Retry</Text>
        </TouchableOpacity>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-paper">
      {/* Header */}
      <View className="px-4 pt-4 pb-2">
        <Text className="font-display text-2xl font-semibold text-ink">My Menu</Text>
      </View>

      {/* Category filter */}
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        className="px-4 mb-2"
        contentContainerStyle={{ gap: 8, paddingRight: 16 }}
      >
        {categories.map((cat) => (
          <TouchableOpacity
            key={cat}
            onPress={() => setSelectedCategory(cat)}
            className={`px-4 py-2 rounded-full border ${
              selectedCategory === cat
                ? 'bg-herb border-herb'
                : 'bg-bone border-mist'
            }`}
            activeOpacity={0.7}
          >
            <Text
              className={`text-sm font-medium ${
                selectedCategory === cat ? 'text-paper' : 'text-ink-soft'
              }`}
            >
              {cat}
            </Text>
          </TouchableOpacity>
        ))}
      </ScrollView>

      {/* Menu list */}
      <FlatList
        data={filteredItems}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => (
          <MenuItemCard
            item={item}
            onEdit={() => router.push(`/menu/${item.id}/edit` as never)}
            onDelete={() => deleteMutation.mutate(item.id)}
          />
        )}
        contentContainerStyle={{ paddingHorizontal: 16, paddingBottom: 100 }}
        refreshControl={
          <RefreshControl refreshing={isRefetching} onRefresh={refetch} tintColor="#3e6b3c" />
        }
        ListEmptyComponent={
          <View className="items-center justify-center py-16">
            <Text className="text-ink-muted text-base text-center">
              No menu items yet.{'\n'}Tap + to add your first item.
            </Text>
          </View>
        }
        showsVerticalScrollIndicator={false}
      />

      {/* FAB */}
      <TouchableOpacity
        onPress={() => router.push('/menu/new' as never)}
        className="absolute bottom-6 right-6 w-14 h-14 bg-herb rounded-full items-center justify-center shadow-lg"
        activeOpacity={0.85}
      >
        {deleteMutation.isPending ? (
          <ActivityIndicator color="white" />
        ) : (
          <Plus size={28} color="white" />
        )}
      </TouchableOpacity>
    </SafeAreaView>
  );
}
