// Slim hairline action row for the chef detail screen — replaces the old big
// coral-tint CTA cards (plan a week, subscribe, group order). One small icon
// (caller picks the colour so the coral budget stays with the caller), a title,
// an optional caption, and a chevron. No tinted background — hairline only.

import type { ReactNode } from 'react';
import { Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import { ChevronRight } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';

// Android ripple tint — translucent charcoal derived from the token.
const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

export interface ChefActionRowProps {
  icon: ReactNode;
  title: string;
  caption?: string;
  onPress: () => void;
  accessibilityLabel: string;
}

export function ChefActionRow({
  icon,
  title,
  caption,
  onPress,
  accessibilityLabel,
}: ChefActionRowProps) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel}
      android_ripple={{ color: ROW_RIPPLE }}
    >
      {({ pressed }) => (
        // Visual + layout styles on the inner View — iOS drops flex/bg/padding
        // returned from a Pressable style function.
        <View style={[styles.row, pressed && Platform.OS === 'ios' && styles.rowPressed]}>
          {icon}
          <View style={styles.textCol}>
            <Text style={styles.title}>{title}</Text>
            {caption ? <Text style={styles.caption}>{caption}</Text> : null}
          </View>
          <ChevronRight
            size={18}
            color={customerColors.charcoal.soft}
            strokeWidth={2}
          />
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  // ≥44pt touch target; hairline separator instead of a tinted card.
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    minHeight: 56,
    paddingVertical: 10,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  rowPressed: {
    opacity: 0.7,
  },
  textCol: {
    flex: 1,
  },
  title: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },
  caption: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    marginTop: 1,
  },
});
