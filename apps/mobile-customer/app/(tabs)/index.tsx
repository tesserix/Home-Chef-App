import { useEffect, useRef, useState } from 'react';
import {
  FlatList,
  Pressable,
  RefreshControl,
  ScrollView,
  Text,
  TextInput,
  View,
} from 'react-native';
import { router } from 'expo-router';
import { Search, SlidersHorizontal } from 'lucide-react-native';
import { ChefCard } from '../../components/chef/ChefCard';
import { useChefs } from '../../hooks/useChefs';
import type { ChefFilters } from '../../hooks/useChefs';

const CUISINES = ['All', 'North Indian', 'South Indian', 'Chinese', 'Continental', 'Italian', 'Healthy'];

const SORT_OPTIONS: { label: string; value: ChefFilters['sort'] }[] = [
  { label: 'Recommended', value: 'rating' },
  { label: 'Top Rated', value: 'rating' },
  { label: 'Newest', value: 'newest' },
  { label: 'Price', value: 'price' },
];

function SkeletonCard() {
  return (
    <View className="flex-1 rounded-2xl overflow-hidden bg-gray-100">
      <View className="w-full h-40 bg-gray-200" />
      <View className="p-3 gap-2">
        <View className="h-4 w-3/4 rounded bg-gray-200" />
        <View className="h-3 w-1/2 rounded bg-gray-200" />
        <View className="h-3 w-1/3 rounded bg-gray-200" />
      </View>
    </View>
  );
}

export default function HomeScreen() {
  const [searchText, setSearchText] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [selectedCuisine, setSelectedCuisine] = useState<string>('All');
  const [isOpenOnly, setIsOpenOnly] = useState(false);
  const [sort, setSort] = useState<ChefFilters['sort']>('rating');

  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }
    debounceTimerRef.current = setTimeout(() => {
      setDebouncedSearch(searchText);
    }, 400);

    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [searchText]);

  const filters: ChefFilters = {
    search: debouncedSearch || undefined,
    cuisine: selectedCuisine !== 'All' ? selectedCuisine : undefined,
    isOpen: isOpenOnly || undefined,
    sort,
    limit: 20,
  };

  const { data, isLoading, isFetching, refetch } = useChefs(filters);

  const chefs = data?.data ?? [];

  const renderHeader = () => (
    <>
      {/* Search bar */}
      <View className="flex-row items-center bg-white border border-gray-200 rounded-xl mx-4 mt-4 mb-3 px-3 gap-2">
        <Search size={18} color="#9CA3AF" />
        <TextInput
          value={searchText}
          onChangeText={setSearchText}
          placeholder="Search chefs, cuisines..."
          placeholderTextColor="#9CA3AF"
          className="flex-1 py-3 text-sm text-gray-900"
          returnKeyType="search"
          accessibilityLabel="Search chefs"
        />
      </View>

      {/* Cuisine filter chips */}
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={{ gap: 8, paddingHorizontal: 16, paddingBottom: 4 }}
        className="mb-3"
      >
        {CUISINES.map((cuisine) => (
          <Pressable
            key={cuisine}
            onPress={() => setSelectedCuisine(cuisine)}
            className={`px-4 py-2 rounded-full border ${
              selectedCuisine === cuisine
                ? 'bg-orange-500 border-orange-500'
                : 'bg-white border-gray-200'
            }`}
            accessibilityRole="button"
            accessibilityLabel={`Filter by ${cuisine}`}
          >
            <Text
              className={`text-sm font-medium ${
                selectedCuisine === cuisine ? 'text-white' : 'text-gray-700'
              }`}
            >
              {cuisine}
            </Text>
          </Pressable>
        ))}
      </ScrollView>

      {/* Open Now toggle + Sort row */}
      <View className="flex-row items-center justify-between px-4 mb-3 gap-2">
        <Pressable
          onPress={() => setIsOpenOnly((prev: boolean) => !prev)}
          className={`flex-row items-center gap-1.5 px-3 py-1.5 rounded-full border ${
            isOpenOnly ? 'bg-green-500 border-green-500' : 'bg-white border-gray-200'
          }`}
          accessibilityRole="button"
          accessibilityLabel="Toggle open now filter"
        >
          <View
            className={`w-2 h-2 rounded-full ${isOpenOnly ? 'bg-white' : 'bg-green-400'}`}
          />
          <Text
            className={`text-xs font-medium ${isOpenOnly ? 'text-white' : 'text-gray-700'}`}
          >
            Open Now
          </Text>
        </Pressable>

        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={{ gap: 6 }}
        >
          {SORT_OPTIONS.map((opt, index) => (
            <Pressable
              key={index}
              onPress={() => setSort(opt.value)}
              className={`px-3 py-1.5 rounded-full border ${
                sort === opt.value && SORT_OPTIONS.findIndex((o) => o.value === sort) === index
                  ? 'bg-orange-100 border-orange-300'
                  : 'bg-white border-gray-200'
              }`}
              accessibilityRole="button"
              accessibilityLabel={`Sort by ${opt.label}`}
            >
              <Text className="text-xs text-gray-700 font-medium">{opt.label}</Text>
            </Pressable>
          ))}
        </ScrollView>

        <SlidersHorizontal size={18} color="#9CA3AF" />
      </View>

      {/* Quick-access row: Social Feed and Catering */}
      <View className="flex-row gap-2 px-4 py-2 mb-1">
        <Pressable
          onPress={() => router.push('/social')}
          className="bg-white border border-gray-200 rounded-full px-4 py-1.5 active:bg-gray-50"
          accessibilityRole="button"
          accessibilityLabel="Go to Social Feed"
        >
          <Text className="text-sm text-gray-800">Social Feed</Text>
        </Pressable>
        <Pressable
          onPress={() => router.push('/catering')}
          className="bg-white border border-gray-200 rounded-full px-4 py-1.5 active:bg-gray-50"
          accessibilityRole="button"
          accessibilityLabel="Go to Catering"
        >
          <Text className="text-sm text-gray-800">Catering</Text>
        </Pressable>
      </View>
    </>
  );

  if (isLoading) {
    return (
      <View className="flex-1 bg-gray-50">
        {renderHeader()}
        <View className="flex-row flex-wrap gap-3 px-4 pt-2">
          <View className="flex-1"><SkeletonCard /></View>
          <View className="flex-1"><SkeletonCard /></View>
          <View className="flex-1"><SkeletonCard /></View>
          <View className="flex-1"><SkeletonCard /></View>
        </View>
      </View>
    );
  }

  return (
    <View className="flex-1 bg-gray-50">
      <FlatList
        data={chefs}
        keyExtractor={(item) => item.id}
        numColumns={2}
        columnWrapperStyle={{ gap: 12, paddingHorizontal: 16 }}
        contentContainerStyle={{ paddingBottom: 100, gap: 12, paddingTop: 4 }}
        ListHeaderComponent={renderHeader}
        renderItem={({ item }) => <ChefCard chef={item} />}
        refreshControl={
          <RefreshControl
            refreshing={isFetching && !isLoading}
            onRefresh={refetch}
          />
        }
        ListEmptyComponent={
          <View className="flex-1 items-center justify-center py-16">
            <Text className="text-gray-400 text-base">No chefs found</Text>
            <Text className="text-gray-300 text-sm mt-1">Try adjusting your filters</Text>
          </View>
        }
      />
    </View>
  );
}
