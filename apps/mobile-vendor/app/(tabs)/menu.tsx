import { useMemo, useState } from 'react';
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
import { theme } from '@homechef/mobile-shared/theme';
import { Skeleton, EmptyState } from '@homechef/mobile-shared/ui';
import { useVendorMenu } from '../../hooks/useVendorMenu';
import type { MenuItem } from '../../hooks/useVendorMenu';
import { MenuItemRow } from '../../components/vendor/MenuItemRow';
import { useDockClearance } from '../../components/navigation/Dock';

const ALL_CATEGORIES = '__all__';

// One coherent list entry — either a sticky-ish section header in
// "All" view, or a single menu item. Mixed into a FlatList so we keep
// virtualization without a separate SectionList API.
interface ListEntry {
  type: 'header' | 'item';
  key: string;
  label?: string;
  item?: MenuItem;
  // Position within the section — drives group-card corner radii
  // (style-based grouping keeps the flat-list virtualization).
  first?: boolean;
  last?: boolean;
}

export default function MenuScreen() {
  const { data, isLoading, isError, refetch } = useVendorMenu();
  const [selectedCategoryId, setSelectedCategoryId] =
    useState<string>(ALL_CATEGORIES);
  const [searchQuery, setSearchQuery] = useState('');
  const dockClearance = useDockClearance();

  // Pull-to-refresh spinner gated to USER action only; React Query's
  // `isRefetching` would also fire for background refetches (window focus,
  // invalidations from other screens), keeping the spinner stuck.
  const [isPulling, setIsPulling] = useState(false);
  async function onPullRefresh(): Promise<void> {
    setIsPulling(true);
    try {
      await refetch();
    } finally {
      setIsPulling(false);
    }
  }

  const categories = data?.categories ?? [];
  const items = data?.items ?? [];
  const categoryNameById = useMemo(
    () => new Map(categories.map((c) => [c.id, c.name])),
    [categories],
  );

  const filteredItems = useMemo<MenuItem[]>(() => {
    const byCat =
      selectedCategoryId === ALL_CATEGORIES
        ? items
        : items.filter((i) => i.categoryId === selectedCategoryId);
    if (!searchQuery.trim()) return byCat;
    const q = searchQuery.trim().toLowerCase();
    return byCat.filter((i) => i.name.toLowerCase().includes(q));
  }, [items, selectedCategoryId, searchQuery]);

  // When "All" is selected (and no search query), group items by category
  // with section headers. When a category is filtered or a search query is
  // active, show a flat list without headers.
  const listEntries = useMemo<ListEntry[]>(() => {
    if (selectedCategoryId !== ALL_CATEGORIES || searchQuery.trim()) {
      return filteredItems.map((i, idx) => ({
        type: 'item' as const,
        key: i.id,
        item: i,
        first: idx === 0,
        last: idx === filteredItems.length - 1,
      }));
    }
    const buckets = new Map<string, MenuItem[]>();
    const uncategorized: MenuItem[] = [];
    for (const item of items) {
      if (!item.categoryId) {
        uncategorized.push(item);
        continue;
      }
      const list = buckets.get(item.categoryId);
      if (list) list.push(item);
      else buckets.set(item.categoryId, [item]);
    }
    const out: ListEntry[] = [];
    for (const [catId, catItems] of buckets) {
      out.push({
        type: 'header',
        key: `h-${catId}`,
        label: categoryNameById.get(catId) ?? 'Other',
      });
      catItems.forEach((item, idx) => {
        out.push({
          type: 'item',
          key: item.id,
          item,
          first: idx === 0,
          last: idx === catItems.length - 1,
        });
      });
    }
    if (uncategorized.length > 0) {
      out.push({ type: 'header', key: 'h-uncat', label: 'Uncategorized' });
      uncategorized.forEach((item, idx) => {
        out.push({
          type: 'item',
          key: item.id,
          item,
          first: idx === 0,
          last: idx === uncategorized.length - 1,
        });
      });
    }
    return out;
  }, [items, filteredItems, selectedCategoryId, searchQuery, categoryNameById]);

  const hasItems = items.length > 0;

  if (isError) {
    return (
      <SafeAreaView style={styles.errorScreen}>
        <Text style={styles.errorBody}>Failed to load menu</Text>
        <Pressable onPress={() => refetch()} style={styles.errorBtn}>
          <Text style={styles.errorBtnLabel}>Retry</Text>
        </Pressable>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* Zone A — Command bar. The "+ Add" affordance is always visible so
          chefs have a discoverable entry point even when the menu is empty
          (and as a fallback if the empty-state CTA fails to render). */}
      <View style={styles.commandBar}>
        <Text style={styles.commandTitle}>Menu</Text>
        <Pressable
          onPress={() => router.push('/menu/new' as never)}
          hitSlop={8}
          style={({ pressed }) => ({ opacity: pressed ? 0.85 : 1 })}
          accessibilityRole="button"
          accessibilityLabel="Add menu item"
        >
          <View style={styles.addBtn}>
            <Text style={styles.addBtnLabel}>+ Add</Text>
          </View>
        </Pressable>
      </View>

      {/* Category chips (UI-V2-SPEC §5) — ink-filled pill when active,
          paper + mist border when inactive. Hidden until there are items,
          and hidden if there's no category to pick. */}
      {hasItems && categories.length > 0 && (
        <View style={styles.tabBarWrap}>
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={styles.tabBar}
          >
            <CategoryTab
              label="All"
              active={selectedCategoryId === ALL_CATEGORIES}
              onPress={() => setSelectedCategoryId(ALL_CATEGORIES)}
            />
            {categories.map((cat) => (
              <CategoryTab
                key={cat.id}
                label={cat.name}
                active={selectedCategoryId === cat.id}
                onPress={() => setSelectedCategoryId(cat.id)}
              />
            ))}
          </ScrollView>
        </View>
      )}

      {/* Search input — visible only when there are items to search through */}
      {hasItems && (
        <View style={styles.searchWrap}>
          <TextInput
            value={searchQuery}
            onChangeText={setSearchQuery}
            placeholder="Search items…"
            placeholderTextColor={theme.colors.ink.muted}
            style={styles.searchInput}
            returnKeyType="search"
            clearButtonMode="while-editing"
            accessibilityLabel="Search menu items"
          />
        </View>
      )}

      {/* List / loading / empty */}
      {isLoading ? (
        <View style={styles.skeletonStack}>
          <Skeleton height={64} style={{ marginBottom: 1 }} />
          <Skeleton height={64} style={{ marginBottom: 1 }} />
          <Skeleton height={64} style={{ marginBottom: 1 }} />
        </View>
      ) : !hasItems ? (
        <EmptyState
          title="Your menu is empty"
          body="Add your first dish so customers can start ordering."
          ctaLabel="Add first item"
          onCtaPress={() => router.push('/menu/new' as never)}
        />
      ) : filteredItems.length === 0 ? (
        <View style={styles.filterEmptyBlock}>
          <Text style={styles.filterEmptyText}>
            {searchQuery.trim()
              ? `No items match "${searchQuery.trim()}".`
              : 'No items in this category. Tap '}
            {!searchQuery.trim() && (
              <Text style={styles.filterEmptyAccent}>+ Add</Text>
            )}
            {!searchQuery.trim() && ' to create one.'}
          </Text>
        </View>
      ) : (
        <FlatList
          data={listEntries}
          keyExtractor={(entry) => entry.key}
          contentContainerStyle={[styles.listContent, { paddingBottom: dockClearance }]}
          refreshControl={
            <RefreshControl
              refreshing={isPulling}
              onRefresh={onPullRefresh}
              tintColor={theme.colors.ink.DEFAULT}
            />
          }
          renderItem={({ item: entry }) =>
            entry.type === 'header' ? (
              <Text style={styles.sectionHeader}>
                {(entry.label ?? '').toUpperCase()}
              </Text>
            ) : entry.item ? (
              // Group-card segment (UI-V2-SPEC §1): shadow on the outer
              // wrapper, overflow clip on the inner so corner radii hold.
              <View
                style={[
                  styles.cardSegment,
                  entry.first && styles.cardTop,
                  entry.last && styles.cardBottom,
                ]}
              >
                <View
                  style={[
                    styles.cardClip,
                    entry.first && styles.cardTop,
                    entry.last && styles.cardBottom,
                  ]}
                >
                  <MenuItemRow
                    item={entry.item}
                    onPress={() =>
                      router.push(`/menu/${entry.item!.id}/edit` as never)
                    }
                  />
                </View>
              </View>
            ) : null
          }
          showsVerticalScrollIndicator={false}
        />
      )}
    </SafeAreaView>
  );
}

interface CategoryTabProps {
  label: string;
  active: boolean;
  onPress: () => void;
}

function CategoryTab({ label, active, onPress }: CategoryTabProps) {
  return (
    <Pressable
      onPress={onPress}
      style={tabStyles.root}
      hitSlop={6}
      accessibilityRole="tab"
      accessibilityState={{ selected: active }}
    >
      {/* Inner-View pattern — visual styles live on the View, not the
          Pressable, to dodge the iOS function-style style drop. */}
      <View
        style={[
          tabStyles.chip,
          active ? tabStyles.chipActive : tabStyles.chipInactive,
        ]}
      >
        <Text style={[tabStyles.label, active && tabStyles.labelActive]}>
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },

  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
  },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },
  // Small primary button (UI-V2-SPEC §3, compact variant)
  addBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[3],
    minHeight: 40,
    justifyContent: 'center',
  },
  addBtnLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.paper,
    letterSpacing: 0.1,
  },

  // Category chip row (UI-V2-SPEC §5)
  tabBarWrap: {
    paddingBottom: theme.spacing[1],
  },
  tabBar: {
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[2],
    gap: theme.spacing[2],
    alignItems: 'center',
  },

  // Search — white card field on the bone canvas (UI-V2-SPEC §1)
  searchWrap: {
    marginHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[3],
    paddingTop: theme.spacing[1],
  },
  searchInput: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    minHeight: 44,
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.paper,
    paddingHorizontal: theme.spacing[3],
    ...theme.shadow[1],
  },

  // List
  listContent: {
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[10],
  },
  sectionHeader: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[5],
    paddingBottom: theme.spacing[2],
  },

  // Group-card segment shell (UI-V2-SPEC §1) — wraps MenuItemRow entries.
  cardSegment: {
    marginHorizontal: theme.spacing[4],
    backgroundColor: theme.colors.paper,
    ...theme.shadow[1],
  },
  cardClip: {
    overflow: 'hidden',
    backgroundColor: theme.colors.paper,
  },
  cardTop: {
    borderTopLeftRadius: theme.radius.lg,
    borderTopRightRadius: theme.radius.lg,
  },
  cardBottom: {
    borderBottomLeftRadius: theme.radius.lg,
    borderBottomRightRadius: theme.radius.lg,
  },

  // Loading + filter-empty
  skeletonStack: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[4],
  },
  filterEmptyBlock: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[6],
  },
  filterEmptyText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    lineHeight: 20,
  },
  filterEmptyAccent: {
    fontFamily: 'Inter-SemiBold',
    color: theme.colors.ink.DEFAULT,
  },

  // Error
  errorScreen: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
    backgroundColor: theme.colors.bone,
  },
  errorBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[4],
  },
  errorBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[6],
    paddingVertical: theme.spacing[3],
  },
  errorBtnLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
  },
});

const tabStyles = StyleSheet.create({
  root: {},
  chip: {
    minHeight: 36,
    paddingHorizontal: theme.spacing[3],
    borderRadius: theme.radius.full,
    alignItems: 'center',
    justifyContent: 'center',
  },
  chipActive: {
    backgroundColor: theme.colors.ink.DEFAULT,
  },
  chipInactive: {
    backgroundColor: theme.colors.paper,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
  },
  label: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  labelActive: { color: theme.colors.paper },
});
