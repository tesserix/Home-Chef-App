import { useEffect, useRef, useState } from 'react';
import {
  FlatList,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { Search, SlidersHorizontal } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ChefCard } from '../../components/chef/ChefCard';
import { useChefs } from '../../hooks/useChefs';
import type { ChefFilters } from '../../hooks/useChefs';

const CUISINES = [
  'All',
  'North Indian',
  'South Indian',
  'Chinese',
  'Continental',
  'Italian',
  'Healthy',
];

const SORT_OPTIONS: { label: string; value: ChefFilters['sort'] }[] = [
  { label: 'Recommended', value: 'rating' },
  { label: 'Top Rated', value: 'rating' },
  { label: 'Newest', value: 'newest' },
  { label: 'Price', value: 'price' },
];

// Skeleton card matching the photo-led 4:3 ChefCard shape.
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
      {/* Floating white search pill — the brand's first impression.
          radius-full, shadow[2], charcoal-soft magnifier + placeholder. */}
      <View style={styles.searchPillWrapper}>
        <View style={styles.searchPill}>
          <Search
            size={18}
            color={customerColors.charcoal.soft}
            accessibilityElementsHidden
          />
          <TextInput
            value={searchText}
            onChangeText={setSearchText}
            placeholder="What are you craving?"
            placeholderTextColor={customerColors.charcoal.soft}
            style={styles.searchInput}
            returnKeyType="search"
            accessibilityLabel="Search chefs"
          />
        </View>
      </View>

      {/* Airbnb category chip row: selected = charcoal text + 2px charcoal
          underline, unselected = charcoal-soft, NO fill / NO border. */}
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={styles.chipRowContent}
        style={styles.chipRow}
        accessibilityRole="tablist"
      >
        {CUISINES.map((cuisine) => {
          const isSelected = selectedCuisine === cuisine;
          return (
            // className-based Pressable is safe on iOS (no function-style style prop).
            <Pressable
              key={cuisine}
              onPress={() => setSelectedCuisine(cuisine)}
              accessibilityRole="tab"
              accessibilityState={{ selected: isSelected }}
              accessibilityLabel={`Filter by ${cuisine}`}
            >
              <View style={[styles.chip, isSelected && styles.chipSelected]}>
                <Text
                  style={[
                    styles.chipLabel,
                    isSelected ? styles.chipLabelSelected : styles.chipLabelDefault,
                  ]}
                >
                  {cuisine}
                </Text>
              </View>
            </Pressable>
          );
        })}
      </ScrollView>

      {/* Open-Now toggle + Sort chips + filter icon row.
          Coral only for selected state; unselected = charcoal-soft. */}
      <View style={styles.filterRow}>
        {/* Open Now toggle */}
        <Pressable
          onPress={() => setIsOpenOnly((prev) => !prev)}
          accessibilityRole="button"
          accessibilityLabel={isOpenOnly ? 'Showing open chefs only' : 'Show open chefs only'}
          accessibilityState={{ checked: isOpenOnly }}
        >
          <View
            style={[
              styles.openNowPill,
              isOpenOnly && styles.openNowPillActive,
            ]}
          >
            <View
              style={[
                styles.openNowDot,
                isOpenOnly ? styles.openNowDotActive : styles.openNowDotInactive,
              ]}
            />
            <Text
              style={[
                styles.openNowLabel,
                isOpenOnly ? styles.openNowLabelActive : styles.openNowLabelDefault,
              ]}
            >
              Open Now
            </Text>
          </View>
        </Pressable>

        {/* Sort chips — horizontal scroll in a flex-1 container */}
        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={styles.sortScrollContent}
          style={styles.sortScroll}
        >
          {SORT_OPTIONS.map((opt, index) => {
            const isActive =
              sort === opt.value &&
              SORT_OPTIONS.findIndex((o) => o.value === sort) === index;
            return (
              <Pressable
                key={index}
                onPress={() => setSort(opt.value)}
                accessibilityRole="button"
                accessibilityLabel={`Sort by ${opt.label}`}
                accessibilityState={{ selected: isActive }}
              >
                <View style={[styles.sortChip, isActive && styles.sortChipActive]}>
                  <Text
                    style={[
                      styles.sortChipLabel,
                      isActive ? styles.sortChipLabelActive : styles.sortChipLabelDefault,
                    ]}
                  >
                    {opt.label}
                  </Text>
                </View>
              </Pressable>
            );
          })}
        </ScrollView>

        <SlidersHorizontal
          size={18}
          color={customerColors.charcoal.soft}
          accessibilityElementsHidden
        />
      </View>

      {/* Quick-access: Social Feed + Catering — clean charcoal pills on white.
          Ghost outline style, no fill, no persimmon. */}
      <View style={styles.quickLinks}>
        <Pressable
          onPress={() => router.push('/social')}
          accessibilityRole="button"
          accessibilityLabel="Go to Social Feed"
        >
          <View style={styles.quickLinkPill}>
            <Text style={styles.quickLinkLabel}>Social Feed</Text>
          </View>
        </Pressable>
        <Pressable
          onPress={() => router.push('/catering')}
          accessibilityRole="button"
          accessibilityLabel="Go to Catering"
        >
          <View style={styles.quickLinkPill}>
            <Text style={styles.quickLinkLabel}>Catering</Text>
          </View>
        </Pressable>
      </View>
    </>
  );

  // A single persistent FlatList drives every state (loading, results, empty).
  // The search pill + filter chips live in ListHeaderComponent so they NEVER
  // reposition when a filter tap triggers a refetch — swapping the whole screen
  // layout between a loading branch and a list branch caused the header to
  // jump. Skeletons and the empty message render inside the list body instead.
  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <FlatList
        data={isLoading ? [] : chefs}
        keyExtractor={(item) => item.id}
        numColumns={2}
        columnWrapperStyle={styles.columnWrapper}
        contentContainerStyle={styles.listContent}
        ListHeaderComponent={renderHeader}
        keyboardShouldPersistTaps="handled"
        renderItem={({ item }) => <ChefCard chef={item} />}
        refreshControl={
          <RefreshControl
            refreshing={isFetching && !isLoading}
            onRefresh={refetch}
            tintColor={customerColors.coral.DEFAULT}
          />
        }
        ListEmptyComponent={
          isLoading ? (
            <View style={styles.skeletonGrid}>
              <View style={styles.skeletonCol}><SkeletonCard /></View>
              <View style={styles.skeletonCol}><SkeletonCard /></View>
              <View style={styles.skeletonCol}><SkeletonCard /></View>
              <View style={styles.skeletonCol}><SkeletonCard /></View>
            </View>
          ) : (
            <View style={styles.emptyState}>
              <Text style={styles.emptyTitle}>No chefs found</Text>
              <Text style={styles.emptyBody}>Try adjusting your filters</Text>
            </View>
          )
        }
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: customerColors.canvas,
  },

  // --- Floating search pill ---
  // Outer wrapper provides horizontal margins and top spacing.
  searchPillWrapper: {
    paddingHorizontal: 16,
    paddingTop: 16,
    paddingBottom: 12,
  },
  searchPill: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: customerColors.surface.DEFAULT,
    borderRadius: 9999, // radius-full
    paddingHorizontal: 16,
    paddingVertical: 14,
    gap: 10,
    // shadow[2] per spec
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 6,
    elevation: 2,
  },
  searchInput: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    // Remove any default padding React Native adds on Android
    padding: 0,
  },

  // --- Airbnb category chip row ---
  chipRow: {
    marginBottom: 4,
  },
  chipRowContent: {
    gap: 0, // chips are spaced by their own padding
    paddingHorizontal: 16,
    paddingBottom: 4,
  },
  chip: {
    paddingHorizontal: 12,
    paddingVertical: 10,
    // No fill, no border on default state
  },
  chipSelected: {
    borderBottomWidth: 2,
    borderBottomColor: customerColors.charcoal.DEFAULT,
  },
  chipLabel: {
    fontFamily: 'Inter',
    fontSize: 14,
    letterSpacing: 0,
  },
  chipLabelSelected: {
    color: customerColors.charcoal.DEFAULT,
    fontFamily: 'Inter-SemiBold',
  },
  chipLabelDefault: {
    color: customerColors.charcoal.soft,
  },

  // --- Open Now toggle + Sort row ---
  filterRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    marginBottom: 8,
    gap: 8,
  },
  openNowPill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: 12,
    paddingVertical: 7,
    borderRadius: 9999,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.surface.DEFAULT,
    minHeight: 34,
  },
  openNowPillActive: {
    borderColor: customerColors.charcoal.DEFAULT,
    backgroundColor: customerColors.charcoal.DEFAULT,
  },
  openNowDot: {
    width: 7,
    height: 7,
    borderRadius: 4,
  },
  openNowDotActive: {
    backgroundColor: customerColors.coral.DEFAULT,
  },
  openNowDotInactive: {
    backgroundColor: customerColors.coral.DEFAULT,
    opacity: 0.6,
  },
  openNowLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
  },
  openNowLabelActive: {
    color: customerColors.canvas,
  },
  openNowLabelDefault: {
    color: customerColors.charcoal.soft,
  },
  sortScroll: {
    flex: 1,
  },
  sortScrollContent: {
    gap: 6,
  },
  sortChip: {
    paddingHorizontal: 12,
    paddingVertical: 7,
    borderRadius: 9999,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.surface.DEFAULT,
    minHeight: 34,
    justifyContent: 'center',
  },
  sortChipActive: {
    borderColor: customerColors.charcoal.DEFAULT,
    backgroundColor: customerColors.charcoal.DEFAULT,
  },
  sortChipLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
  },
  sortChipLabelActive: {
    color: customerColors.canvas,
  },
  sortChipLabelDefault: {
    color: customerColors.charcoal.soft,
  },

  // --- Quick access links ---
  quickLinks: {
    flexDirection: 'row',
    gap: 8,
    paddingHorizontal: 16,
    paddingVertical: 8,
    marginBottom: 4,
  },
  quickLinkPill: {
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 9999,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.surface.DEFAULT,
    minHeight: 36,
    justifyContent: 'center',
  },
  quickLinkLabel: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.DEFAULT,
  },

  // --- List layout ---
  columnWrapper: {
    gap: 12,
    paddingHorizontal: 16,
  },
  listContent: {
    paddingBottom: 100,
    gap: 12,
    paddingTop: 4,
  },

  // --- Empty state ---
  emptyState: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 64,
  },
  emptyTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.charcoal.DEFAULT,
    marginBottom: 4,
  },
  emptyBody: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
  },

  // --- Skeleton grid ---
  skeletonGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 12,
    paddingHorizontal: 16,
    paddingTop: 4,
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
