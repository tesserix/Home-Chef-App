import React, { useEffect, useRef, useState } from 'react';
import { ActivityIndicator, FlatList, Pressable, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { ChefHat, Search } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ScreenHeader } from '../components/ScreenHeader';
import { useSearchDishes, type DishResult } from '../hooks/useSearchDishes';
import { useChefs } from '../hooks/useChefs';
import type { Chef } from '../types/customer';

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

  // Unified search (home's single search entry): chefs by name alongside
  // dishes. Chef matches render as a compact section above the dish list.
  const chefQueryActive = query.trim().length >= 2;
  const { data: chefData } = useChefs(
    chefQueryActive ? { search: query.trim(), limit: 5 } : { limit: 0 },
  );
  const chefMatches = chefQueryActive ? (chefData?.data ?? []) : [];

  const { data: dishes = [], isLoading, isFetching } = useSearchDishes(query);
  const showEmpty =
    chefQueryActive &&
    !isLoading &&
    !isFetching &&
    dishes.length === 0 &&
    chefMatches.length === 0;

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

  // Compact chef row — avatar-less to stay light; taps into the chef page.
  const renderChefRow = (chef: Chef) => (
    <Pressable
      key={chef.id}
      onPress={() => router.push(`/chef/${chef.id}`)}
      accessibilityRole="button"
      accessibilityLabel={`View chef ${chef.name}`}
    >
      {({ pressed }) => (
        <View
          style={{
            flexDirection: 'row',
            alignItems: 'center',
            gap: 12,
            paddingHorizontal: 16,
            paddingVertical: 12,
            backgroundColor: pressed ? customerColors.surface.soft : customerColors.canvas,
            borderBottomWidth: 1,
            borderBottomColor: customerColors.hairline,
          }}
        >
          <View
            style={{
              width: 34,
              height: 34,
              borderRadius: 17,
              backgroundColor: customerColors.surface.soft,
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <ChefHat size={16} color={customerColors.charcoal.soft} />
          </View>
          <View style={{ flex: 1 }}>
            <Text
              style={{ fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT }}
              numberOfLines={1}
            >
              {chef.name}
            </Text>
            <Text
              style={{ fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft, marginTop: 1 }}
              numberOfLines={1}
            >
              {chef.cuisine}
            </Text>
          </View>
          {chef.rating > 0 ? (
            <Text style={{ fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft }}>
              ★ {chef.rating.toFixed(1)}
            </Text>
          ) : null}
        </View>
      )}
    </Pressable>
  );

  const sectionLabel = (label: string) => (
    <Text
      style={{
        fontFamily: 'Inter-SemiBold',
        fontSize: 12,
        color: customerColors.charcoal.soft,
        textTransform: 'uppercase',
        letterSpacing: 0.4,
        paddingHorizontal: 16,
        paddingTop: 16,
        paddingBottom: 6,
      }}
    >
      {label}
    </Text>
  );

  return (
    <SafeAreaView edges={['top', 'left', 'right']} style={{ flex: 1, backgroundColor: customerColors.canvas }}>
      <ScreenHeader title="Search" />
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
            placeholder="What are you craving?"
            placeholderTextColor={customerColors.charcoal.soft}
            style={{ flex: 1, marginLeft: 8, fontFamily: 'Inter', fontSize: 16, color: customerColors.charcoal.DEFAULT }}
            returnKeyType="search"
            accessibilityLabel="Search dishes and chefs"
          />
        </View>
      </View>

      {isLoading && chefQueryActive ? (
        <ActivityIndicator style={{ marginTop: 24 }} color={customerColors.charcoal.soft} />
      ) : showEmpty ? (
        <Text style={{ textAlign: 'center', marginTop: 32, color: customerColors.charcoal.soft, fontFamily: 'Inter' }}>
          Nothing found for “{query}”.
        </Text>
      ) : !chefQueryActive ? (
        <Text style={{ textAlign: 'center', marginTop: 32, color: customerColors.charcoal.soft, fontFamily: 'Inter' }}>
          Type at least 2 characters to search dishes and chefs.
        </Text>
      ) : (
        <FlatList
          data={dishes}
          keyExtractor={(d) => d.id}
          renderItem={renderItem}
          keyboardShouldPersistTaps="handled"
          ListHeaderComponent={
            <>
              {chefMatches.length > 0 && (
                <View>
                  {sectionLabel('Chefs')}
                  {chefMatches.map(renderChefRow)}
                </View>
              )}
              {dishes.length > 0 && sectionLabel('Dishes')}
            </>
          }
        />
      )}
    </SafeAreaView>
  );
}
