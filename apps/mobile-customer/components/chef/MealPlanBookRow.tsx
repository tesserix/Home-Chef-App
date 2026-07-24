// MealPlanBookRow — one selectable meal in the book-meal-plan list. The booking
// screen used the photo-forward WeeklyMenuDishCard, but tiffin cells rarely have
// photos, so every card was a big empty gray box in a cramped horizontal scroll.
// This is a compact, photo-free hairline row instead: left FSSAI diet mark, the
// dish name (+ a Thali pill and its components when it's a combo), a tabular
// price, and a selection circle that fills coral when picked.
//
// Layout lives on a plain View inside the Pressable — iOS Fabric drops flex/bg
// returned from a Pressable style function.

import { Pressable, StyleSheet, Text, View } from 'react-native';
import * as Haptics from 'expo-haptics';
import { Check } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { DietIcon } from '@homechef/mobile-shared/ui';
import { comboLabel } from '../../lib/combo-label';

export interface MealPlanBookRowProps {
  name: string;
  variant: 'veg' | 'nonveg';
  price: number;
  isCombo?: boolean;
  comboComponents?: string[];
  selected: boolean;
  onPress: () => void;
  /** Hairline above the row when it isn't the first in its slot group. */
  divided?: boolean;
}

export function MealPlanBookRow({
  name,
  variant,
  price,
  isCombo,
  comboComponents,
  selected,
  onPress,
  divided,
}: MealPlanBookRowProps) {
  const isVeg = variant === 'veg';
  const priceLabel = `₹${Math.round(price).toLocaleString('en-IN')}`;
  const parts =
    isCombo && comboComponents && comboComponents.length > 0
      ? comboComponents.join(' · ')
      : null;

  const handlePress = () => {
    void Haptics.selectionAsync();
    onPress();
  };

  return (
    <Pressable
      onPress={handlePress}
      accessibilityRole="button"
      accessibilityState={{ selected }}
      accessibilityLabel={`${name}, ${isVeg ? 'veg' : 'non-veg'}${
        isCombo ? `, ${comboLabel()}` : ''
      }, ${priceLabel}`}
    >
      {({ pressed }) => (
        <View
          style={[
            styles.row,
            divided && styles.rowDivided,
            selected && styles.rowSelected,
            pressed && styles.rowPressed,
          ]}
        >
          <DietIcon kind={isVeg ? 'veg' : 'non-veg'} size={14} />

          <View style={styles.body}>
            <View style={styles.titleRow}>
              <Text style={styles.name} numberOfLines={1}>
                {name}
              </Text>
              {isCombo ? (
                <View style={styles.comboPill}>
                  <Text style={styles.comboPillText}>{comboLabel()}</Text>
                </View>
              ) : null}
            </View>
            {parts ? (
              <Text style={styles.parts} numberOfLines={1}>
                {parts}
              </Text>
            ) : null}
          </View>

          <Text style={styles.price}>{priceLabel}</Text>
          <View style={[styles.check, selected && styles.checkOn]}>
            {selected ? (
              <Check size={13} color={customerColors.canvas} strokeWidth={3} />
            ) : null}
          </View>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingVertical: 12,
    paddingHorizontal: 14,
    minHeight: 56,
    backgroundColor: customerColors.canvas,
  },
  rowDivided: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
  },
  rowSelected: { backgroundColor: customerColors.coral.tint },
  rowPressed: { opacity: 0.7 },

  body: { flex: 1, gap: 2 },
  titleRow: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  name: {
    flexShrink: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },
  // Bordered (not filled) pill so it stays legible on both the canvas row and
  // the coral-tint selected row.
  comboPill: {
    borderWidth: 1,
    borderColor: customerColors.coral.DEFAULT,
    borderRadius: 6,
    paddingHorizontal: 6,
    paddingVertical: 1,
  },
  comboPillText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    letterSpacing: 0.3,
    color: customerColors.coral.pressed,
  },
  parts: {
    fontFamily: 'Inter',
    fontSize: 12,
    lineHeight: 16,
    color: customerColors.charcoal.soft,
  },
  price: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  check: {
    width: 24,
    height: 24,
    borderRadius: 9999,
    borderWidth: 2,
    borderColor: customerColors.hairline,
    alignItems: 'center',
    justifyContent: 'center',
  },
  checkOn: {
    backgroundColor: customerColors.coral.DEFAULT,
    borderColor: customerColors.coral.DEFAULT,
  },
});
