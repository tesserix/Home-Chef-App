// Win-back offer banner (#42) — a quiet, on-brand nudge at the top of Home when
// the customer has an active win-back offer. Shows the discount, code and expiry;
// the code auto-prefills the promo field at checkout (useWinback). Dismissible
// for the session.

import { useState } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { Gift, X } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useWinback } from '../../hooks/useWinback';

function daysLeft(iso: string): number {
  const ms = new Date(iso).getTime() - Date.now();
  return Math.max(0, Math.ceil(ms / (24 * 60 * 60 * 1000)));
}

export function WinbackBanner() {
  const { data: offer } = useWinback();
  const [dismissed, setDismissed] = useState(false);

  if (!offer || dismissed) return null;

  const left = daysLeft(offer.expiresAt);

  return (
    <View style={styles.card} accessibilityRole="summary">
      <View style={styles.iconWrap}>
        <Gift size={20} color={customerColors.coral.DEFAULT} />
      </View>
      <View style={styles.body}>
        <Text style={styles.title}>We miss you — {Math.round(offer.discountPercent)}% off your next order</Text>
        <Text style={styles.sub}>
          Use code <Text style={styles.code}>{offer.code}</Text> at checkout
          {left > 0 ? ` · ${left} day${left === 1 ? '' : 's'} left` : ''}
        </Text>
      </View>
      <Pressable
        onPress={() => setDismissed(true)}
        hitSlop={10}
        accessibilityRole="button"
        accessibilityLabel="Dismiss offer"
      >
        <X size={18} color={customerColors.charcoal.soft} />
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    marginHorizontal: 16,
    marginBottom: 12,
    paddingHorizontal: 14,
    paddingVertical: 12,
    borderRadius: 12,
    backgroundColor: customerColors.coral.tint,
    borderWidth: 1,
    borderColor: customerColors.coral.DEFAULT,
  },
  iconWrap: {
    width: 36,
    height: 36,
    borderRadius: 18,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.canvas,
  },
  body: { flex: 1 },
  title: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  sub: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft, marginTop: 2 },
  code: { fontFamily: 'Inter-SemiBold', color: customerColors.coral.pressed },
});
