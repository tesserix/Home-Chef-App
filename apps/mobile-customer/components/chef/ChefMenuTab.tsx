// "Menu" tab pane on the chef detail screen — the à-la-carte landing view.
//
// Every category renders as its own SECTION in one continuous scroll, rather
// than the chip row filtering the list down to one category at a time. Filtering
// hid the menu: a customer had to already know a dessert section existed to go
// looking for it. Sectioning reveals the whole menu, and the floating Menu pill
// (owned by the screen) jumps between sections.
//
// Section offsets are reported upward via onSectionLayout so the screen — which
// owns the ScrollView — can scroll to one. Measuring real laid-out positions is
// both simpler and more accurate than a SectionList's scrollToLocation, which
// needs getItemLayout to be precise and cannot be, since these rows vary in
// height with photo presence and description length.
//
// Presentational: category state and the startGroupOrder flow stay in the screen.

import { Platform, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
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
  /** The section currently in view — drives the chip underline. */
  activeCategory: string | null;
  /** Tapping a chip scrolls to that section rather than filtering to it. */
  onSelectCategory: (category: string) => void;
  /** ALL items, grouped into sections here. */
  items: MenuItem[];
  /** True when the chef has no menu at all. */
  menuIsEmpty: boolean;
  /** Reports each section's y-offset within the scroll content, for jumping. */
  onSectionLayout: (category: string, y: number) => void;
  onStartGroupOrder: () => void;
}

export function ChefMenuTab({
  chefId,
  chefName,
  categories,
  activeCategory,
  onSelectCategory,
  items,
  menuIsEmpty,
  onSectionLayout,
  onStartGroupOrder,
}: ChefMenuTabProps) {
  // Group once per render, preserving the category order the screen derived.
  const sections = categories
    .map((name) => ({ name, items: items.filter((i) => (i.category ?? 'Other') === name) }))
    .filter((sec) => sec.items.length > 0);

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
                accessibilityLabel={`Jump to ${cat}`}
                accessibilityState={{ selected: activeCategory === cat }}
                android_ripple={{ color: CHIP_RIPPLE }}
              >
                {({ pressed }) => (
                  // Inner View: visual styles here to dodge iOS Pressable bug
                  <View
                    style={[
                      styles.categoryChip,
                      activeCategory === cat && styles.categoryChipActive,
                      pressed && Platform.OS === 'ios' && styles.categoryChipPressed,
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
                )}
              </Pressable>
            ))}
          </ScrollView>
          <View style={styles.hairline} />
        </>
      ) : null}

      {/* ── MENU SECTIONS ── */}
      {menuIsEmpty ? (
        <View style={styles.emptyMenu}>
          <Text style={styles.emptyMenuText}>
            This kitchen hasn&apos;t published a menu right now — check back soon.
          </Text>
        </View>
      ) : (
        sections.map((section) => (
          <View
            key={section.name}
            // Absolute y within the ScrollView content — what scrollTo needs.
            onLayout={(e) => onSectionLayout(section.name, e.nativeEvent.layout.y)}
          >
            {/* The heading is omitted for a single-category menu, where it would
                only repeat what the screen already says. */}
            {sections.length > 1 ? (
              <Text style={styles.sectionHeading}>{section.name}</Text>
            ) : null}
            <View style={styles.menuList}>
              {section.items.map((item) => (
                <MenuItemCard key={item.id} item={item} chefId={chefId} chefName={chefName} />
              ))}
            </View>
          </View>
        ))
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
  sectionHeading: {
    fontSize: 20,
    fontWeight: '700',
    color: customerColors.charcoal.DEFAULT,
    paddingHorizontal: 20,
    paddingTop: 24,
    paddingBottom: 4,
  },
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
  // Opacity-only iOS press feedback (Android gets android_ripple instead).
  categoryChipPressed: {
    opacity: 0.6,
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
