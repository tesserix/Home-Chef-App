import React, { useEffect, useRef, useState } from 'react';
import { ActivityIndicator, FlatList, Pressable, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { Search } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ScreenHeader } from '../components/ScreenHeader';
import { useSearchDishes, type DishResult } from '../hooks/useSearchDishes';

function formatMoney(amount: number): string {
  try {
    return new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(amount);
  } catch {
    return `₹${amount.toFixed(0)}`;
  }
}

export default function SearchDishesScreen() {
  const router = useRouter();
  const { q: initialQ } = useLocalSearchParams<{ q?: string }>();
  const [text, setText] = useState(initialQ ?? '');
  const [query, setQuery] = useState(initialQ ?? '');
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Debounce the query so we don't hit the API on every keystroke.
  useEffect(() => {
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => setQuery(text), 350);
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, [text]);

  const { data: dishes = [], isLoading, isFetching } = useSearchDishes(query);
  const showEmpty = query.trim().length >= 2 && !isLoading && !isFetching && dishes.length === 0;

  const renderItem = ({ item }: { item: DishResult }) => (
    <Pressable
      onPress={() => item.chefId && router.push(`/chef/${item.chefId}`)}
      accessibilityRole="button"
      accessibilityLabel={`View ${item.name}`}
    >
      {({ pressed }) => (
        <View
          style={{
            flexDirection: 'row',
            alignItems: 'center',
            paddingHorizontal: 16,
            paddingVertical: 12,
            backgroundColor: pressed ? customerColors.surface.soft : customerColors.canvas,
            borderBottomWidth: 1,
            borderBottomColor: customerColors.hairline,
          }}
        >
          <View style={{ flex: 1, paddingRight: 12 }}>
            <Text style={{ fontFamily: 'Inter', fontSize: 16, color: customerColors.charcoal.DEFAULT }} numberOfLines={1}>
              {item.name}
            </Text>
            {item.description ? (
              <Text style={{ fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft, marginTop: 2 }} numberOfLines={1}>
                {item.description}
              </Text>
            ) : null}
          </View>
          <View style={{ alignItems: 'flex-end' }}>
            <Text style={{ fontFamily: 'Inter', fontSize: 14, fontWeight: '600', color: customerColors.charcoal.DEFAULT }}>
              {formatMoney(item.price)}
            </Text>
            {item.rating ? (
              <Text style={{ fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft, marginTop: 2 }}>
                ★ {item.rating.toFixed(1)}
              </Text>
            ) : null}
          </View>
        </View>
      )}
    </Pressable>
  );

  return (
    <SafeAreaView edges={['top', 'left', 'right']} style={{ flex: 1, backgroundColor: customerColors.canvas }}>
      <ScreenHeader title="Search dishes" />
      <View style={{ padding: 16 }}>
        <View
          style={{
            flexDirection: 'row',
            alignItems: 'center',
            backgroundColor: customerColors.surface.soft,
            borderRadius: 999,
            paddingHorizontal: 16,
            height: 44,
          }}
        >
          <Search size={18} color={customerColors.charcoal.soft} />
          <TextInput
            value={text}
            onChangeText={setText}
            autoFocus={!initialQ}
            placeholder="Search dishes by name"
            placeholderTextColor={customerColors.charcoal.soft}
            style={{ flex: 1, marginLeft: 8, fontFamily: 'Inter', fontSize: 16, color: customerColors.charcoal.DEFAULT }}
            returnKeyType="search"
            accessibilityLabel="Search dishes"
          />
        </View>
      </View>

      {isLoading && query.trim().length >= 2 ? (
        <ActivityIndicator style={{ marginTop: 24 }} color={customerColors.charcoal.soft} />
      ) : showEmpty ? (
        <Text style={{ textAlign: 'center', marginTop: 32, color: customerColors.charcoal.soft, fontFamily: 'Inter' }}>
          No dishes found for “{query}”.
        </Text>
      ) : query.trim().length < 2 ? (
        <Text style={{ textAlign: 'center', marginTop: 32, color: customerColors.charcoal.soft, fontFamily: 'Inter' }}>
          Type at least 2 characters to search dishes.
        </Text>
      ) : (
        <FlatList data={dishes} keyExtractor={(d) => d.id} renderItem={renderItem} keyboardShouldPersistTaps="handled" />
      )}
    </SafeAreaView>
  );
}
