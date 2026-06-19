// Configure + subscribe to a chef's daily tiffin (#283). Pick meals, days, veg/
// nonveg and cadence; the per-cycle price previews live; subscribe sets it up
// (the Razorpay UPI-Autopay mandate is the billing phase / owner-tested seam).

import { useEffect, useMemo, useState } from 'react';
import { ActivityIndicator, Alert, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft, Check } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import {
  useMealChefOffer,
  usePreviewMealPrice,
  useSubscribeMeal,
} from '../../hooks/useMealSubscription';

const DAYS = [
  { v: 1, l: 'Mon' }, { v: 2, l: 'Tue' }, { v: 3, l: 'Wed' }, { v: 4, l: 'Thu' },
  { v: 5, l: 'Fri' }, { v: 6, l: 'Sat' }, { v: 0, l: 'Sun' },
];

function money(n: number): string {
  return `₹${Math.round(n).toLocaleString('en-IN')}`;
}

export default function MealSubscribeScreen() {
  const { chefId } = useLocalSearchParams<{ chefId: string }>();
  const { data: offer, isLoading } = useMealChefOffer(chefId);
  const preview = usePreviewMealPrice();
  const subscribe = useSubscribeMeal();

  const [slots, setSlots] = useState<string[]>([]);
  const [days, setDays] = useState<number[]>([1, 2, 3, 4, 5]);
  const [variant, setVariant] = useState<'veg' | 'nonveg'>('veg');
  const [cadence, setCadence] = useState<string>('weekly');
  const [price, setPrice] = useState<number | null>(null);

  // Seed defaults from the offer.
  useEffect(() => {
    if (offer?.available) {
      if (slots.length === 0 && offer.slots?.length) setSlots([offer.slots[0]]);
      if (offer.cadences?.length && !offer.cadences.includes(cadence)) setCadence(offer.cadences[0]);
    }
  }, [offer]); // eslint-disable-line react-hooks/exhaustive-deps

  const selectionValid = slots.length > 0 && days.length > 0 && !!cadence && !!chefId;

  // Live price preview whenever the selection changes.
  useEffect(() => {
    if (!selectionValid) {
      setPrice(null);
      return;
    }
    preview.mutate(
      { chefId: chefId!, slots, days, variant, cadence },
      { onSuccess: (r) => setPrice(r.cycleAmount), onError: () => setPrice(null) },
    );
  }, [slots, days, variant, cadence]); // eslint-disable-line react-hooks/exhaustive-deps

  const cadenceOptions = useMemo(() => offer?.cadences ?? ['weekly', 'monthly'], [offer]);
  const slotOptions = useMemo(() => offer?.slots ?? ['lunch', 'dinner'], [offer]);

  function toggle<T>(list: T[], v: T, set: (x: T[]) => void) {
    set(list.includes(v) ? list.filter((x) => x !== v) : [...list, v]);
  }

  function onSubscribe() {
    if (!selectionValid) return;
    subscribe.mutate(
      { chefId: chefId!, slots, days, variant, cadence },
      {
        onSuccess: () =>
          Alert.alert('Subscription created', 'Your daily tiffin is set up. Manage it under My Subscriptions.', [
            { text: 'View', onPress: () => router.replace('/subscriptions' as never) },
          ]),
        onError: (e) => Alert.alert('Could not subscribe', e.message || 'Please try again.'),
      },
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable onPress={() => router.back()} hitSlop={10} accessibilityLabel="Go back">
          <ChevronLeft size={26} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.headerTitle}>Daily tiffin</Text>
        <View style={{ width: 26 }} />
      </View>

      {isLoading ? (
        <View style={styles.centered}><ActivityIndicator color={customerColors.coral.DEFAULT} /></View>
      ) : !offer?.available ? (
        <View style={styles.centered}>
          <Text style={styles.muted}>This chef doesn’t offer a tiffin subscription yet.</Text>
        </View>
      ) : (
        <ScrollView contentContainerStyle={styles.scroll}>
          <Text style={styles.label}>Meals</Text>
          <View style={styles.chipRow}>
            {slotOptions.map((s) => (
              <Chip key={s} label={s === 'lunch' ? 'Lunch' : 'Dinner'} active={slots.includes(s)} onPress={() => toggle(slots, s, setSlots)} />
            ))}
          </View>

          <Text style={styles.label}>Days</Text>
          <View style={styles.chipRow}>
            {DAYS.map((d) => (
              <Chip key={d.v} label={d.l} active={days.includes(d.v)} onPress={() => toggle(days, d.v, setDays)} />
            ))}
          </View>

          <Text style={styles.label}>Preference</Text>
          <View style={styles.chipRow}>
            <Chip label="Veg" active={variant === 'veg'} onPress={() => setVariant('veg')} />
            <Chip label="Non-veg" active={variant === 'nonveg'} onPress={() => setVariant('nonveg')} />
          </View>

          <Text style={styles.label}>Plan</Text>
          <View style={styles.chipRow}>
            {cadenceOptions.map((c) => (
              <Chip key={c} label={c === 'weekly' ? 'Weekly' : 'Monthly'} active={cadence === c} onPress={() => setCadence(c)} />
            ))}
          </View>

          <View style={styles.priceCard}>
            <Text style={styles.priceLabel}>{cadence === 'monthly' ? 'Per month' : 'Per week'}</Text>
            <Text style={styles.priceValue}>
              {price == null ? '—' : money(price)}
            </Text>
          </View>
          {offer.deliveryFee ? <Text style={styles.muted}>Includes {money(offer.deliveryFee)} flat delivery</Text> : null}

          <Pressable
            onPress={onSubscribe}
            disabled={!selectionValid || subscribe.isPending}
            style={[styles.cta, (!selectionValid || subscribe.isPending) && styles.ctaDisabled]}
            accessibilityRole="button"
          >
            {subscribe.isPending ? (
              <ActivityIndicator color={customerColors.canvas} />
            ) : (
              <Text style={styles.ctaText}>Subscribe{price != null ? ` · ${money(price)}` : ''}</Text>
            )}
          </Pressable>
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

function Chip({ label, active, onPress }: { label: string; active: boolean; onPress: () => void }) {
  return (
    <Pressable onPress={onPress} accessibilityRole="button" accessibilityState={{ selected: active }}>
      <View style={[styles.chip, active && styles.chipActive]}>
        {active ? <Check size={13} color={customerColors.coral.pressed} strokeWidth={3} /> : null}
        <Text style={[styles.chipText, active && styles.chipTextActive]}>{label}</Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.surface.soft },
  header: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', paddingHorizontal: 16, paddingVertical: 12 },
  headerTitle: { fontFamily: 'Geist-Bold', fontSize: 20, color: customerColors.charcoal.DEFAULT },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  scroll: { padding: 16, paddingBottom: 40, gap: 6 },
  label: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT, marginTop: 14, marginBottom: 6 },
  chipRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  chip: { flexDirection: 'row', alignItems: 'center', gap: 4, borderWidth: 1, borderColor: customerColors.hairline, borderRadius: 9999, paddingHorizontal: 14, paddingVertical: 8, backgroundColor: customerColors.canvas },
  chipActive: { borderColor: customerColors.coral.DEFAULT, backgroundColor: customerColors.coral.tint },
  chipText: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  chipTextActive: { color: customerColors.coral.pressed, fontFamily: 'Inter-SemiBold' },
  priceCard: { marginTop: 20, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', backgroundColor: customerColors.canvas, borderRadius: 12, padding: 16 },
  priceLabel: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft },
  priceValue: { fontFamily: 'Geist-Bold', fontSize: 22, color: customerColors.charcoal.DEFAULT },
  muted: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, marginTop: 8, textAlign: 'center' },
  cta: { marginTop: 24, backgroundColor: customerColors.coral.DEFAULT, borderRadius: 12, minHeight: 52, alignItems: 'center', justifyContent: 'center' },
  ctaDisabled: { backgroundColor: customerColors.surface.soft, borderWidth: 1, borderColor: customerColors.hairline },
  ctaText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: customerColors.canvas },
});
