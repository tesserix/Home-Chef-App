// FilterSheet — secondary filter controls for the Home/discovery screen.
//
// Consolidates diet, price, sort, and Open-Now behind a single bottom sheet
// so the main header collapses to: search + cuisine row + slim filter bar.
// Each filter group is wired to the SAME state/handlers as before — this
// component owns no state; all values and setters flow in via props.
//
// Design: radius-lg (16) on the sheet; hairline group separators; coral
// active state; iOS Pressable inner-View pattern throughout (never
// function-style style array on Pressable).
//
// Built on the shared SheetBase (plain React Native Modal + Animated) —
// @gorhom/bottom-sheet's raw BottomSheet + `.expand()` silently no-ops on
// this app's gorhom 5.2.8 + reanimated 4.4.1 pairing, confirmed on-device.

import React, { forwardRef, useCallback } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SheetBase, type SheetHandle } from '@homechef/mobile-shared/ui';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { ChefFilters } from '../../hooks/useChefs';

// ---- Constant option sets ---------------------------------------------------
// Mirrored from index.tsx — the source of truth remains the screen; the
// constants are defined there and passed in, but we keep them here too so
// FilterSheet is self-contained and testable in isolation.

export const DIET_FILTER_OPTIONS: { label: string; value: string }[] = [
  { label: 'All diets', value: '' },
  { label: 'Vegetarian', value: 'vegetarian' },
  { label: 'Vegan', value: 'vegan' },
  { label: 'Jain', value: 'jain' },
  { label: 'Eggetarian', value: 'eggetarian' },
  { label: 'Halal', value: 'halal' },
];

export const PRICE_FILTER_OPTIONS: { label: string; value: number | undefined }[] = [
  { label: 'Any price', value: undefined },
  { label: '< ₹100', value: 100 },
  { label: '< ₹250', value: 250 },
  { label: '< ₹500', value: 500 },
];

export const SORT_OPTIONS: { label: string; value: ChefFilters['sort']; key: string }[] = [
  { label: 'Recommended', value: 'rating', key: 'recommended' },
  { label: 'Top Rated', value: 'rating', key: 'top-rated' },
  { label: 'Newest', value: 'newest', key: 'newest' },
  { label: 'Price', value: 'price', key: 'price' },
  { label: 'Nearest', value: 'distance', key: 'distance' },
];

// ---- Props ------------------------------------------------------------------

export interface FilterSheetProps {
  // Diet
  selectedDiet: string;
  onDietChange: (value: string) => void;
  // Price
  maxPrice: number | undefined;
  onMaxPriceChange: (value: number | undefined) => void;
  // Sort
  sort: ChefFilters['sort'];
  onSortChange: (value: ChefFilters['sort']) => void;
  // Open Now
  isOpenOnly: boolean;
  onIsOpenOnlyChange: (value: boolean) => void;
}

// ---- Sub-components ---------------------------------------------------------

interface SectionHeaderProps {
  title: string;
}

function SectionHeader({ title }: SectionHeaderProps) {
  return (
    <Text style={styles.sectionTitle}>{title}</Text>
  );
}

interface FilterChipProps {
  label: string;
  isSelected: boolean;
  onPress: () => void;
  accessibilityLabel?: string;
}

// iOS Pressable inner-View pattern: visual styles live on the inner View;
// Pressable is a plain interaction wrapper with no style prop.
function FilterChip({ label, isSelected, onPress, accessibilityLabel }: FilterChipProps) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityState={{ selected: isSelected }}
      accessibilityLabel={accessibilityLabel ?? label}
    >
      <View style={[styles.chip, isSelected && styles.chipActive]}>
        <Text style={[styles.chipLabel, isSelected && styles.chipLabelActive]}>
          {label}
        </Text>
      </View>
    </Pressable>
  );
}

// ---- Main component ---------------------------------------------------------

export const FilterSheet = forwardRef<SheetHandle, FilterSheetProps>(
  (props, ref) => {
    const {
      selectedDiet,
      onDietChange,
      maxPrice,
      onMaxPriceChange,
      sort,
      onSortChange,
      isOpenOnly,
      onIsOpenOnlyChange,
    } = props;

    const handleClose = useCallback(() => {
      if (ref && 'current' in ref && ref.current) {
        ref.current.dismiss();
      }
    }, [ref]);

    return (
      // SheetBase's own ScrollView owns the max-height + scroll bounding
      // (setting maxHeight on an ancestor View instead of the ScrollView
      // itself doesn't actually bound a ScrollView child — RN's scrollable-
      // area recipe needs maxHeight on the ScrollView). maxHeightRatio caps
      // growth at the same ~75% ceiling the old snapPoints={['75%']} used;
      // the panel now hugs its (fixed, known) content instead of always
      // padding out to a fixed height — sensible for content-height sheets.
      <SheetBase ref={ref} maxHeightRatio={0.75} panelStyle={styles.sheetBackground}>
        <View style={styles.scrollContent}>
          {/* ── Sheet header ── */}
          <View style={styles.sheetHeader}>
            <Text style={styles.sheetTitle}>Filters</Text>
            <Pressable
              onPress={handleClose}
              accessibilityRole="button"
              accessibilityLabel="Close filters"
            >
              <View style={styles.doneButton}>
                <Text style={styles.doneLabel}>Done</Text>
              </View>
            </Pressable>
          </View>

          {/* ── Open Now toggle ── */}
          <View style={styles.section}>
            <SectionHeader title="Availability" />
            <Pressable
              onPress={() => onIsOpenOnlyChange(!isOpenOnly)}
              accessibilityRole="switch"
              accessibilityState={{ checked: isOpenOnly }}
              accessibilityLabel={isOpenOnly ? 'Showing open chefs only' : 'Show open chefs only'}
            >
              <View style={styles.toggleRow}>
                <View style={styles.toggleLeft}>
                  <View style={[styles.openDot, isOpenOnly && styles.openDotActive]} />
                  <Text style={styles.toggleLabel}>Open now</Text>
                </View>
                {/* Custom toggle track */}
                <View style={[styles.track, isOpenOnly && styles.trackActive]}>
                  <View style={[styles.thumb, isOpenOnly && styles.thumbActive]} />
                </View>
              </View>
            </Pressable>
          </View>

          <View style={styles.hairline} />

          {/* ── Diet ── */}
          <View style={styles.section}>
            <SectionHeader title="Diet" />
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              style={styles.chipScroll}
              contentContainerStyle={styles.chipRow}
            >
              {DIET_FILTER_OPTIONS.map((opt) => (
                <FilterChip
                  key={opt.value || 'all-diets'}
                  label={opt.label}
                  isSelected={selectedDiet === opt.value}
                  onPress={() => onDietChange(opt.value)}
                  accessibilityLabel={`Filter by ${opt.label}`}
                />
              ))}
            </ScrollView>
          </View>

          <View style={styles.hairline} />

          {/* ── Price ── */}
          <View style={styles.section}>
            <SectionHeader title="Price" />
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              style={styles.chipScroll}
              contentContainerStyle={styles.chipRow}
            >
              {PRICE_FILTER_OPTIONS.map((opt) => (
                <FilterChip
                  key={opt.label}
                  label={opt.label}
                  isSelected={maxPrice === opt.value}
                  onPress={() => onMaxPriceChange(opt.value)}
                  accessibilityLabel={`Filter by ${opt.label}`}
                />
              ))}
            </ScrollView>
          </View>

          <View style={styles.hairline} />

          {/* ── Sort ── */}
          <View style={styles.section}>
            <SectionHeader title="Sort by" />
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              style={styles.chipScroll}
              contentContainerStyle={styles.chipRow}
            >
              {SORT_OPTIONS.map((opt) => (
                <FilterChip
                  key={opt.key}
                  label={opt.label}
                  isSelected={sort === opt.value && SORT_OPTIONS.find((o) => o.value === sort)?.key === opt.key}
                  onPress={() => onSortChange(opt.value)}
                  accessibilityLabel={`Sort by ${opt.label}`}
                />
              ))}
            </ScrollView>
          </View>

          {/* Bottom breathing room above the safe-area notch */}
          <View style={{ height: 32 }} />
        </View>
      </SheetBase>
    );
  }
);

FilterSheet.displayName = 'FilterSheet';

// ---- Styles -----------------------------------------------------------------

const styles = StyleSheet.create({
  // Sheet chrome — background/radius only. Shadow lives on SheetBase's own
  // outer (unclipped) shadow view; putting shadow props here too would sit
  // on the same node as `overflow: hidden` and iOS would drop them.
  sheetBackground: {
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
    backgroundColor: customerColors.canvas,
  },
  scrollContent: {
    paddingTop: 8,
    paddingBottom: 0,
  },

  // Header row: "Filters" title + "Done" button
  sheetHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 20,
    paddingTop: 4,
    paddingBottom: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
    marginBottom: 4,
  },
  sheetTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 17,
    color: customerColors.charcoal.DEFAULT,
    letterSpacing: -0.2,
  },
  doneButton: {
    paddingHorizontal: 12,
    paddingVertical: 6,
    minHeight: 36,
    justifyContent: 'center',
  },
  doneLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.coral.DEFAULT,
  },

  // Section wrapper
  section: {
    paddingHorizontal: 20,
    paddingVertical: 16,
    gap: 12,
  },
  sectionTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    letterSpacing: 0.4,
    textTransform: 'uppercase',
  },

  // Hairline section divider (not a heavy border)
  hairline: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: customerColors.hairline,
    marginHorizontal: 20,
  },

  // Horizontal chip scroll row
  // flexGrow: 0 — RN's ScrollView base style is flexGrow: 1 (ScrollView.js,
  // baseHorizontal), so a horizontal chip row grows into any free vertical
  // space rather than hugging its chips. Pin it to stay content-height.
  chipScroll: {
    flexGrow: 0,
  },
  chipRow: {
    gap: 8,
    paddingRight: 8,
  },

  // Filter chips — selected: charcoal fill + canvas text
  chip: {
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: 9999,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.surface.DEFAULT,
    minHeight: 36,
    justifyContent: 'center',
  },
  chipActive: {
    borderColor: customerColors.charcoal.DEFAULT,
    backgroundColor: customerColors.charcoal.DEFAULT,
  },
  chipLabel: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
  },
  chipLabelActive: {
    color: customerColors.canvas,
    fontFamily: 'Inter-SemiBold',
  },

  // Open-Now toggle row
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    minHeight: 44,
    paddingVertical: 2,
  },
  toggleLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    flex: 1,
  },
  openDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: customerColors.coral.DEFAULT,
    opacity: 0.5,
  },
  openDotActive: {
    opacity: 1,
  },
  toggleLabel: {
    fontFamily: 'Inter',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },

  // Toggle track + thumb — native-look switch approximation
  track: {
    width: 44,
    height: 26,
    borderRadius: 13,
    backgroundColor: customerColors.hairline,
    justifyContent: 'center',
    paddingHorizontal: 2,
  },
  trackActive: {
    backgroundColor: customerColors.coral.DEFAULT,
  },
  thumb: {
    width: 22,
    height: 22,
    borderRadius: 11,
    backgroundColor: customerColors.canvas,
    alignSelf: 'flex-start',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.12,
    shadowRadius: 2,
    elevation: 2,
  },
  thumbActive: {
    alignSelf: 'flex-end',
  },
});
