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
      <View className="flex-1 items-center justify-center bg-bone">
        <ActivityIndicator size="large" color="#3e6b3c" />
      </View>
    );
  }

  if (chefError || menuError || !chef) {
    return (
      <View className="flex-1 items-center justify-center bg-bone px-6">
        <Text className="text-base text-ink-muted text-center">
          Failed to load chef details. Please try again.
        </Text>
      </View>
    );
  }

  return (
    <View className="flex-1 bg-paper">
      <FlatList
        data={filteredItems}
        keyExtractor={(item) => item.id}
        contentContainerStyle={{ paddingBottom: 100, paddingHorizontal: 16, paddingTop: 8 }}
        renderItem={({ item }) => (
          <MenuItemCard item={item} chefId={chef.id} chefName={chef.name} />
        )}
        ListEmptyComponent={
          <View className="py-10 items-center">
            <Text className="text-ink-muted">No items in this category</Text>
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
            <View className="bg-bone rounded-2xl p-4 mt-3 mb-3 shadow-sm border border-mist">
              <View className="flex-row items-start justify-between">
                <View className="flex-1">
                  <Text className="text-xl font-semibold text-ink">{chef.name}</Text>
                  <Text className="text-sm text-ink-muted mt-0.5">{chef.cuisine}</Text>
                </View>
                <View
                  className={`px-3 py-1 rounded-full ${chef.isOpen ? 'bg-herb-tint' : 'bg-mist'}`}
                >
                  <Text
                    className={`text-xs font-semibold ${chef.isOpen ? 'text-herb' : 'text-ink-muted'}`}
                  >
                    {chef.isOpen ? 'Open' : 'Closed'}
                  </Text>
                </View>
              </View>

              <View className="flex-row items-center gap-3 mt-2">
                <View className="flex-row items-center gap-1">
                  <Star size={14} color="#d1a64a" fill="#d1a64a" />
                  <Text className="text-sm font-semibold text-ink-soft">
                    {chef.rating.toFixed(1)}
                  </Text>
                  <Text className="text-xs text-ink-muted">({chef.reviewCount})</Text>
                </View>
                {chef.deliveryTime ? (
                  <Text className="text-xs text-ink-muted">{chef.deliveryTime}</Text>
                ) : null}
                {chef.deliveryFee != null ? (
                  <Text className="text-xs text-ink-muted">
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
                        ? 'bg-herb border-herb'
                        : 'bg-bone border-mist'
                    }`}
                    accessibilityRole="button"
                    accessibilityLabel={`Filter by ${cat}`}
                  >
                    <Text
                      className={`text-sm font-medium ${
                        activeCategory === cat ? 'text-paper' : 'text-ink-soft'
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
