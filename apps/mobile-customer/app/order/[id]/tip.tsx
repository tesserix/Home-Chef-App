import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { KeyboardAwareScrollView } from '@homechef/mobile-shared/ui';
import { useCreateTip } from '../../../hooks/useTip';
import { friendlyErrorMessage } from '../../../lib/errors';

const PRESETS = [20, 50, 100];

// Android ripple tints — translucent tokens, never a new literal colour.
const BACK_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const CHIP_RIPPLE = `${customerColors.coral.DEFAULT}1F`;
const CTA_RIPPLE = `${customerColors.canvas}33`;

// Post-delivery tip screen (#45): pick an amount for the chef and/or rider;
// 100% pass-through. Creates the charge then opens the shared Razorpay sheet.
export default function TipScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const createTip = useCreateTip();
  const [chefAmount, setChefAmount] = useState(50);
  const [riderAmount, setRiderAmount] = useState(0);

  const total = chefAmount + riderAmount;

  function send() {
    if (!id || total < 1) return;
    createTip.mutate(
      { orderId: id, chefAmount, riderAmount },
      {
        onSuccess: (data) => {
          router.replace({
            pathname: '/payment/checkout',
            params: {
              kind: 'tip',
              tipId: data.tipId,
              orderId: id,
              razorpayOrderId: data.razorpayOrderId,
              razorpayKeyId: data.razorpayKeyId,
              amount: String(data.amount),
              currency: data.currency ?? 'INR',
            },
          });
        },
        onError: (err) =>
          Alert.alert('Could not start tip', friendlyErrorMessage(err, 'Please try again.')),
      },
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: BACK_RIPPLE, borderless: true, radius: 20 }}
        >
          {({ pressed }) => (
            <View style={pressed && Platform.OS === 'ios' ? styles.iconPressed : undefined}>
              <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text style={styles.title}>Add a tip</Text>
        <View style={{ width: 24 }} />
      </View>

      <KeyboardAwareScrollView contentContainerStyle={styles.scroll}>
        <Text style={styles.intro}>
          Say thanks with a tip — 100% goes straight to your chef and rider, with
          no platform cut.
        </Text>

        <TipPicker
          label="Your chef"
          amount={chefAmount}
          onChange={setChefAmount}
        />
        <TipPicker
          label="Your rider"
          caption="Only if a rider delivered your order"
          amount={riderAmount}
          onChange={setRiderAmount}
        />
      </KeyboardAwareScrollView>

      <View style={styles.footer}>
        <View style={styles.totalRow}>
          <Text style={styles.totalLabel}>Total tip</Text>
          <Text style={styles.totalValue}>₹{total}</Text>
        </View>
        <Pressable
          onPress={send}
          disabled={total < 1 || createTip.isPending}
          accessibilityRole="button"
          accessibilityLabel={total < 1 ? 'Choose a tip amount' : `Tip ₹${total}`}
          android_ripple={
            total < 1 || createTip.isPending ? undefined : { color: CTA_RIPPLE, borderless: false }
          }
        >
          {({ pressed }) => (
            <View
              style={[
                styles.cta,
                (total < 1 || createTip.isPending) && styles.ctaDisabled,
                pressed &&
                  Platform.OS === 'ios' &&
                  total >= 1 &&
                  !createTip.isPending &&
                  styles.ctaPressed,
              ]}
            >
              {createTip.isPending ? (
                <ActivityIndicator color={customerColors.canvas} />
              ) : (
                <Text style={styles.ctaText}>{total < 1 ? 'Choose an amount' : `Tip ₹${total}`}</Text>
              )}
            </View>
          )}
        </Pressable>
      </View>
    </SafeAreaView>
  );
}

function TipPicker({
  label,
  caption,
  amount,
  onChange,
}: {
  label: string;
  caption?: string;
  amount: number;
  onChange: (n: number) => void;
}) {
  const isCustom = amount > 0 && !PRESETS.includes(amount);
  return (
    <View style={styles.card}>
      <Text style={styles.cardLabel}>{label}</Text>
      {caption ? <Text style={styles.cardCaption}>{caption}</Text> : null}
      <View style={styles.chips}>
        <Chip label="None" active={amount === 0} onPress={() => onChange(0)} />
        {PRESETS.map((p) => (
          <Chip key={p} label={`₹${p}`} active={amount === p} onPress={() => onChange(p)} />
        ))}
      </View>
      <TextInput
        style={[styles.custom, isCustom && styles.customActive]}
        placeholder="Custom amount (₹)"
        accessibilityLabel="Custom tip amount in rupees"
        placeholderTextColor={customerColors.charcoal.soft}
        keyboardType="numeric"
        value={isCustom ? String(amount) : ''}
        onChangeText={(t) => onChange(Number(t.replace(/[^0-9]/g, '')) || 0)}
      />
    </View>
  );
}

function Chip({
  label,
  active,
  onPress,
}: {
  label: string;
  active: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ selected: active }}
      android_ripple={{ color: CHIP_RIPPLE, borderless: false }}
    >
      {({ pressed }) => (
        <View
          style={[
            styles.chip,
            active && styles.chipActive,
            pressed && Platform.OS === 'ios' && styles.chipPressed,
          ]}
        >
          <Text style={[styles.chipText, active && styles.chipTextActive]}>{label}</Text>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  title: { fontFamily: 'Inter-SemiBold', fontSize: 18, color: customerColors.charcoal.DEFAULT },
  iconPressed: { opacity: 0.6 },
  scroll: { padding: 16, paddingBottom: 24 },
  intro: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    lineHeight: 20,
    marginBottom: 20,
  },
  card: {
    backgroundColor: customerColors.surface.DEFAULT,
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    padding: 16,
    marginBottom: 16,
  },
  cardLabel: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: customerColors.charcoal.DEFAULT },
  cardCaption: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    marginTop: 2,
  },
  chips: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, marginTop: 12 },
  // R3 chip: content-sized (no fixed width) + R5 44pt min touch height.
  chip: {
    paddingHorizontal: 16,
    minHeight: 44,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: 999,
    borderWidth: 1,
    borderColor: customerColors.hairline,
  },
  chipActive: { backgroundColor: customerColors.coral.tint, borderColor: customerColors.coral.DEFAULT },
  chipPressed: { opacity: 0.7 },
  chipText: {
    fontFamily: 'Inter-Medium',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  chipTextActive: { color: customerColors.coral.DEFAULT },
  custom: {
    marginTop: 12,
    fontFamily: 'Inter',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    paddingVertical: 10,
    paddingHorizontal: 12,
    borderRadius: 8,
    backgroundColor: customerColors.surface.soft,
    borderWidth: 1,
    borderColor: 'transparent',
  },
  customActive: { borderColor: customerColors.coral.DEFAULT },
  footer: {
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 16,
    backgroundColor: customerColors.canvas,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
    gap: 12,
  },
  totalRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  totalLabel: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft },
  totalValue: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 20,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  // Spec §3 primary button: radius 8, 52pt min-height.
  cta: {
    height: 52,
    borderRadius: 8,
    backgroundColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  ctaPressed: { backgroundColor: customerColors.coral.pressed },
  ctaDisabled: { backgroundColor: customerColors.charcoal.soft, opacity: 0.5 },
  ctaText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: customerColors.canvas, fontVariant: ['tabular-nums'] },
});
