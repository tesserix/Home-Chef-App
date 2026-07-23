// "Menu" tab pane on the chef detail screen — the à-la-carte landing view.
// Category chip row (Airbnb underline style) + MenuItemCard list + the small
// group-order secondary action at the bottom. Presentational: category state
// and the startGroupOrder flow stay in the screen and come in as props.

import { Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { Users } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { MenuItem } from '../../types/customer';
import { GROUP_ORDERS_ENABLED } from '../../lib/features';
import { MenuItemCard } from './MenuItemCard';
import { ChefActionRow } from './ChefActionRow';

// Android ripple tint — translucent charcoal derived from the token (never a
// new literal colour), matching the ChefCard `withAlpha` convention.
const CHIP_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

export interface ChefMenuTabProps {
  chefId: string;
  chefName: string;
  categories: string[];
  activeCategory: string | null;
  onSelectCategory: (category: string) => void;
  /** Items already filtered by the active category. */
  filteredItems: MenuItem[];
  /** True when the chef has no menu at all (vs. just an empty category). */
  menuIsEmpty: boolean;
  onStartGroupOrder: () => void;
}

export function ChefMenuTab({
  chefId,
  chefName,
  categories,
  activeCategory,
  onSelectCategory,
  filteredItems,
  menuIsEmpty,
  onStartGroupOrder,
}: ChefMenuTabProps) {
  return (
    <>
      {/* ── CATEGORY CHIP ROW (Airbnb underline style, spec §2 item 2) ── */}
      {categories.length > 1 ? (
        <>
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            style={styles.categoryScroll}
            contentContainerStyle={styles.categoryRow}
          >
            {categories.map((cat) => (
              <Pressable
                key={cat}
                onPress={() => onSelectCategory(cat)}
                accessibilityRole="button"
                accessibilityLabel={`Filter by ${cat}`}
                accessibilityState={{ selected: activeCategory === cat }}
                android_ripple={{ color: CHIP_RIPPLE }}
              >
                {/* Inner View: visual styles here to dodge iOS Pressable bug */}
                <View
                  style={[
                    styles.categoryChip,
                    activeCategory === cat && styles.categoryChipActive,
                  ]}
                >
                  <Text
                    style={[
                      styles.categoryChipLabel,
                      activeCategory === cat && styles.categoryChipLabelActive,
                    ]}
                  >
                    {cat}
                  </Text>
                  {/* 2px underline for selected (Airbnb category bar) */}
                  {activeCategory === cat ? (
                    <View style={styles.categoryChipUnderline} />
                  ) : null}
                </View>
              </Pressable>
            ))}
          </ScrollView>
          <View style={styles.hairline} />
        </>
      ) : null}

      {/* ── MENU ITEMS ── */}
      {filteredItems.length === 0 ? (
        <View style={styles.emptyMenu}>
          <Text style={styles.emptyMenuText}>
            {menuIsEmpty
              ? "This kitchen hasn't published a menu right now — check back soon."
              : 'No items in this category'}
          </Text>
        </View>
      ) : (
        <View style={styles.menuList}>
          {filteredItems.map((item) => (
            <MenuItemCard
              key={item.id}
              item={item}
              chefId={chefId}
              chefName={chefName}
            />
          ))}
        </View>
      )}

      {/* Group / office order (#46) — small secondary action at the bottom of
          the menu (was a big tinted card in the header). Hidden until the
          split-pay flow is live. */}
      {GROUP_ORDERS_ENABLED ? (
        <View style={styles.groupRowWrap}>
          <ChefActionRow
            icon={
              <Users
                size={18}
                color={customerColors.charcoal.soft}
                strokeWidth={2}
              />
            }
            title="Start a group / office order"
            caption="Everyone adds their own items, split the bill"
            onPress={onStartGroupOrder}
            accessibilityLabel="Start a group or office order"
          />
        </View>
      ) : null}
    </>
  );
}

const styles = StyleSheet.create({
  hairline: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: customerColors.hairline,
    marginHorizontal: 20,
  },

  // ── Category chip row (Airbnb underline style, spec §2 item 2) ───────────
  // flexGrow: 0 — RN's ScrollView base style is flexGrow: 1 (ScrollView.js,
  // baseHorizontal), so a horizontal category row grows into free vertical
  // space rather than hugging its chips. Pin it to stay content-height.
  categoryScroll: {
    flexGrow: 0,
  },
  categoryRow: {
    paddingHorizontal: 20,
    paddingVertical: 0,
    gap: 0,
  },
  categoryChip: {
    paddingHorizontal: 4,
    paddingTop: 14,
    paddingBottom: 10,
    marginRight: 20,
    alignItems: 'center',
    position: 'relative',
  },
  categoryChipActive: {
    // Underline drawn as a child View (see below)
  },
  categoryChipLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    letterSpacing: 0.1,
  },
  categoryChipLabelActive: {
    // Selected = charcoal text (spec §2 item 2).
    color: customerColors.charcoal.DEFAULT,
  },
  // 2px charcoal underline for selected chip (Airbnb category-bar style).
  categoryChipUnderline: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    height: 2,
    borderRadius: 1,
    backgroundColor: customerColors.charcoal.DEFAULT,
  },

  // ── Menu list ─────────────────────────────────────────────────────────────
  menuList: {
    paddingHorizontal: 20,
    // Last item has a hairline bottom — that is sufficient; no extra padding needed.
  },
  emptyMenu: {
    paddingVertical: 40,
    alignItems: 'center',
  },
  emptyMenuText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
  },
  // Group-order secondary action at the bottom of the Menu tab.
  groupRowWrap: {
    paddingHorizontal: 20,
    paddingTop: 4,
  },
});
