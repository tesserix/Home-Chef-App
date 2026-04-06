import { useRef, useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  Pressable,
  ScrollView,
  Text,
  View,
} from 'react-native';
import { Image } from 'expo-image';
import { useLocalSearchParams } from 'expo-router';
import { Star } from 'lucide-react-native';
import BottomSheet from '@gorhom/bottom-sheet';
import { useChef, useChefMenu } from '../../hooks/useChefs';
import { MenuItemCard } from '../../components/chef/MenuItemCard';
import { CartBar } from '../../components/cart/CartBar';
import { CartSheet } from '../../components/cart/CartSheet';

export default function ChefDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const cartSheetRef = useRef<BottomSheet>(null);

  const { data: chefData, isLoading: chefLoading, isError: chefError } = useChef(id ?? '');
  const { data: menuData, isLoading: menuLoading, isError: menuError } = useChefMenu(id ?? '');

  const chef = chefData?.data;
  const menuItems = menuData?.data ?? [];

  // Derive unique categories from menu items
  const categories = Array.from(
    new Set(menuItems.map((item) => item.category ?? 'Other'))
  );

  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);

  const activeCategory = selectedCategory ?? categories[0] ?? null;

  const filteredItems = activeCategory
    ? menuItems.filter((item) => (item.category ?? 'Other') === activeCategory)
    : menuItems;

  const openCart = () => {
    cartSheetRef.current?.expand();
  };

  if (chefLoading || menuLoading) {
    return (
      <View className="flex-1 items-center justify-center bg-white">
        <ActivityIndicator size="large" color="#F97316" />
      </View>
    );
  }

  if (chefError || menuError || !chef) {
    return (
      <View className="flex-1 items-center justify-center bg-white px-6">
        <Text className="text-base text-gray-500 text-center">
          Failed to load chef details. Please try again.
        </Text>
      </View>
    );
  }

  return (
    <View className="flex-1 bg-gray-50">
      <FlatList
        data={filteredItems}
        keyExtractor={(item) => item.id}
        contentContainerStyle={{ paddingBottom: 100, paddingHorizontal: 16, paddingTop: 8 }}
        renderItem={({ item }) => (
          <MenuItemCard item={item} chefId={chef.id} chefName={chef.name} />
        )}
        ListEmptyComponent={
          <View className="py-10 items-center">
            <Text className="text-gray-400">No items in this category</Text>
          </View>
        }
        ListHeaderComponent={
          <>
            {/* Hero image */}
            <Image
              source={chef.imageUrl ? { uri: chef.imageUrl } : null}
              style={{ width: '100%', height: 200, marginHorizontal: -16 }}
              contentFit="cover"
              placeholder={{ blurhash: 'L6PZfSi_.AyE_3t7t7R**0o#DgR4' }}
              transition={300}
            />

            {/* Chef info */}
            <View className="bg-white rounded-2xl p-4 mt-3 mb-3 shadow-sm border border-gray-100">
              <View className="flex-row items-start justify-between">
                <View className="flex-1">
                  <Text className="text-xl font-bold text-gray-900">{chef.name}</Text>
                  <Text className="text-sm text-gray-500 mt-0.5">{chef.cuisine}</Text>
                </View>
                <View
                  className={`px-3 py-1 rounded-full ${chef.isOpen ? 'bg-green-100' : 'bg-gray-100'}`}
                >
                  <Text
                    className={`text-xs font-semibold ${chef.isOpen ? 'text-green-700' : 'text-gray-500'}`}
                  >
                    {chef.isOpen ? 'Open' : 'Closed'}
                  </Text>
                </View>
              </View>

              <View className="flex-row items-center gap-3 mt-2">
                <View className="flex-row items-center gap-1">
                  <Star size={14} color="#F59E0B" fill="#F59E0B" />
                  <Text className="text-sm font-semibold text-gray-700">
                    {chef.rating.toFixed(1)}
                  </Text>
                  <Text className="text-xs text-gray-400">({chef.reviewCount})</Text>
                </View>
                {chef.deliveryTime ? (
                  <Text className="text-xs text-gray-500">{chef.deliveryTime}</Text>
                ) : null}
                {chef.deliveryFee != null ? (
                  <Text className="text-xs text-gray-500">
                    {chef.deliveryFee === 0 ? 'Free delivery' : `₹${chef.deliveryFee} delivery`}
                  </Text>
                ) : null}
              </View>
            </View>

            {/* Category tabs */}
            {categories.length > 0 && (
              <ScrollView
                horizontal
                showsHorizontalScrollIndicator={false}
                contentContainerStyle={{ gap: 8, paddingVertical: 4, paddingHorizontal: 4, marginBottom: 8 }}
              >
                {categories.map((cat) => (
                  <Pressable
                    key={cat}
                    onPress={() => setSelectedCategory(cat)}
                    className={`px-4 py-2 rounded-full border ${
                      activeCategory === cat
                        ? 'bg-orange-500 border-orange-500'
                        : 'bg-white border-gray-200'
                    }`}
                    accessibilityRole="button"
                    accessibilityLabel={`Filter by ${cat}`}
                  >
                    <Text
                      className={`text-sm font-medium ${
                        activeCategory === cat ? 'text-white' : 'text-gray-700'
                      }`}
                    >
                      {cat}
                    </Text>
                  </Pressable>
                ))}
              </ScrollView>
            )}
          </>
        }
      />

      {/* Floating cart bar */}
      <CartBar onPress={openCart} />

      {/* Cart bottom sheet */}
      <CartSheet ref={cartSheetRef} />
    </View>
  );
}
