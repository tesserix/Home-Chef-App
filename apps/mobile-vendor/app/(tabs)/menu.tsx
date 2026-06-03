import { useMemo, useState } from 'react';
import {
  FlatList,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { theme } from '@homechef/mobile-shared/theme';
import { Skeleton } from '@homechef/mobile-shared/ui';
import { useVendorMenu } from '../../hooks/useVendorMenu';
import type { MenuItem } from '../../hooks/useVendorMenu';
import { MenuItemRow } from '../../components/vendor/MenuItemRow';

const ALL_CATEGORIES = '__all__';

// One coherent list entry — either a sticky-ish section header in
// "All" view, or a single menu item. Mixed into a FlatList so we keep
// virtualization without a separate SectionList API.
interface ListEntry {
  type: 'header' | 'item';
  key: string;
  label?: string;
  item?: MenuItem;
}

export default function MenuScreen() {
  const { data, isLoading, isError, refetch, isRefetching } = useVendorMenu();
  const [selectedCategoryId, setSelectedCategoryId] =
    useState<string>(ALL_CATEGORIES);

  const categories = data?.categories ?? [];
  const items = data?.items ?? [];
  const categoryNameById = useMemo(
    () => new Map(categories.map((c) => [c.id, c.name])),
    [categories],
  );

  const filteredItems = useMemo<MenuItem[]>(() => {
    if (selectedCategoryId === ALL_CATEGORIES) return items;
    return items.filter((i) => i.categoryId === selectedCategoryId);
  }, [items, selectedCategoryId]);

  // When "All" is selected, group items by category with section
  // headers. When a category is filtered, a flat list with no headers.
  const listEntries = useMemo<ListEntry[]>(() => {
    if (selectedCategoryId !== ALL_CATEGORIES) {
      return filteredItems.map((i) => ({
        type: 'item',
        key: i.id,
        item: i,
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
      for (const item of catItems) {
        out.push({ type: 'item', key: item.id, item });
      }
    }
    if (uncategorized.length > 0) {
      out.push({ type: 'header', key: 'h-uncat', label: 'Uncategorized' });
      for (const item of uncategorized) {
        out.push({ type: 'item', key: item.id, item });
      }
    }
    return out;
  }, [items, selectedCategoryId, categoryNameById]);

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
      {/* Zone A — Command bar */}
      <View style={styles.commandBar}>
        <Text style={styles.commandTitle}>Menu</Text>
        {hasItems && (
          <Pressable
            onPress={() => router.push('/menu/new' as never)}
            hitSlop={8}
            style={({ pressed }) => [
              styles.addBtn,
              pressed && { opacity: 0.7 },
            ]}
            accessibilityRole="button"
            accessibilityLabel="Add menu item"
          >
            <Text style={styles.addBtnLabel}>+ Add</Text>
          </Pressable>
        )}
      </View>

      {/* Category tabs — bare text + persimmon underline. Hidden until
          there are items, and hidden if there's only one category to pick. */}
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

      {/* List / loading / empty */}
      {isLoading ? (
        <View style={styles.skeletonStack}>
          <Skeleton height={64} style={{ marginBottom: 1 }} />
          <Skeleton height={64} style={{ marginBottom: 1 }} />
          <Skeleton height={64} style={{ marginBottom: 1 }} />
        </View>
      ) : !hasItems ? (
        <EmptyMenu onAdd={() => router.push('/menu/new' as never)} />
      ) : filteredItems.length === 0 ? (
        <View style={styles.filterEmptyBlock}>
          <Text style={styles.filterEmptyText}>
            No items in this category. Tap{' '}
            <Text style={styles.filterEmptyAccent}>+ Add</Text> to create one.
          </Text>
        </View>
      ) : (
        <FlatList
          data={listEntries}
          keyExtractor={(entry) => entry.key}
          contentContainerStyle={styles.listContent}
          refreshControl={
            <RefreshControl
              refreshing={isRefetching}
              onRefresh={refetch}
              tintColor={theme.colors.ink.DEFAULT}
            />
          }
          renderItem={({ item: entry }) =>
            entry.type === 'header' ? (
              <Text style={styles.sectionHeader}>
                {(entry.label ?? '').toUpperCase()}
              </Text>
            ) : entry.item ? (
              <MenuItemRow
                item={entry.item}
                onPress={() =>
                  router.push(`/menu/${entry.item!.id}/edit` as never)
                }
              />
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
      <Text style={[tabStyles.label, active && tabStyles.labelActive]}>
        {label}
      </Text>
      <View
        style={[
          tabStyles.indicator,
          active && tabStyles.indicatorActive,
        ]}
      />
    </Pressable>
  );
}

interface EmptyMenuProps {
  onAdd: () => void;
}

function EmptyMenu({ onAdd }: EmptyMenuProps) {
  return (
    <View style={styles.emptyMenu}>
      <Text style={styles.emptyHeadline}>Your menu is empty</Text>
      <Text style={styles.emptyBody}>
        Add your first dish so customers can start ordering.
      </Text>
      <Pressable
        onPress={onAdd}
        style={({ pressed }) => [
          styles.emptyCta,
          pressed && { opacity: 0.85 },
        ]}
        accessibilityRole="button"
      >
        <Text style={styles.emptyCtaLabel}>Add first item</Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },

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
  addBtn: {
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    minHeight: 36,
    justifyContent: 'center',
  },
  addBtnLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    textDecorationLine: 'underline',
    letterSpacing: 0.1,
  },

  // Category tab bar
  tabBarWrap: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  tabBar: {
    paddingHorizontal: theme.spacing[4],
    gap: theme.spacing[5],
    alignItems: 'flex-start',
  },

  // List
  listContent: {
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

  // 0-item empty
  emptyMenu: {
    flex: 1,
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[16],
    alignItems: 'flex-start',
  },
  emptyHeadline: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.h2.size,
    color: theme.colors.ink.DEFAULT,
    letterSpacing: -0.2,
    marginBottom: theme.spacing[2],
  },
  emptyBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.soft,
    lineHeight: 22,
    marginBottom: theme.spacing[6],
    maxWidth: 340,
  },
  emptyCta: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingVertical: theme.spacing[4],
    paddingHorizontal: theme.spacing[6],
    alignItems: 'center',
    alignSelf: 'stretch',
  },
  emptyCtaLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
    letterSpacing: 0.2,
  },

  // Error
  errorScreen: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
    backgroundColor: theme.colors.paper,
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
  root: { paddingTop: theme.spacing[2], paddingBottom: theme.spacing[2] },
  label: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    paddingBottom: 6,
  },
  labelActive: { color: theme.colors.ink.DEFAULT },
  indicator: {
    height: 2,
    backgroundColor: 'transparent',
    borderRadius: 1,
  },
  indicatorActive: { backgroundColor: theme.colors.herb.DEFAULT },
});
