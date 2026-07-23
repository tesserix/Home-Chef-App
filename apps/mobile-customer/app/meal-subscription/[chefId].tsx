// Configure + subscribe to a chef's daily tiffin (#283). Pick meals, days, veg/
// nonveg and cadence; the per-cycle price previews live; subscribe sets it up
// (the Razorpay UPI-Autopay mandate is the billing phase / owner-tested seam).

import { useEffect, useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft, Check } from 'lucide-react-native';
import { customerColors, customerTheme } from '@homechef/mobile-shared/theme';
import {
  useMealChefOffer,
  usePreviewMealPrice,
  useSubscribeMeal,
} from '../../hooks/useMealSubscription';

// Android ripple tints — translucent tokens, never a new literal colour.
const ICON_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const CANVAS_RIPPLE = `${customerColors.canvas}33`;
const CHIP_RIPPLE = `${customerColors.charcoal.DEFAULT}0F`;

const DAYS = [
  { v: 1, l: 'Mon' }, { v: 2, l: 'Tue' }, { v: 3, l: 'Wed' }, { v: 4, l: 'Thu' },
  { v: 5, l: 'Fri' }, { v: 6, l: 'Sat' }, { v: 0, l: 'Sun' },
];

function money(n: number): string {
  return `₹${Math.round(n).toLocaleString('en-IN')}`;
}

export default function MealSubscribeScreen() {
  const { chefId } = useLocalSearchParams<{ chefId: string }>();
  const { data: offer, isLoading, isError, refetch } = useMealChefOffer(chefId);
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
        <Pressable
          onPress={() => router.back()}
          hitSlop={10}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: ICON_RIPPLE, borderless: true }}
        >
          <ChevronLeft size={26} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.headerTitle}>Daily tiffin</Text>
        <View style={{ width: 26 }} />
      </View>

      {isLoading ? (
        <View style={styles.scroll}>
          {['Meals', 'Days', 'Preference', 'Plan'].map((label) => (
            <View key={label}>
              <Text style={styles.label}>{label}</Text>
              <View style={styles.chipRow}>
                <View style={[styles.skeletonChip, { width: 72 }]} />
                <View style={[styles.skeletonChip, { width: 72 }]} />
                <View style={[styles.skeletonChip, { width: 72 }]} />
              </View>
            </View>
          ))}
        </View>
      ) : isError ? (
        // Distinct from "no offer" below — this is a failed fetch, not a chef
        // that genuinely has no tiffin plan, so it gets a retry instead of a
        // dead-end sentence.
        <View style={styles.centered}>
          <Text style={styles.muted}>Something went wrong. Please try again.</Text>
          <Pressable
            onPress={() => void refetch()}
            accessibilityRole="button"
            accessibilityLabel="Retry loading tiffin plan"
            android_ripple={{ color: CANVAS_RIPPLE, borderless: false }}
            style={styles.retryButton}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.retryButtonInner,
                  pressed && Platform.OS === 'ios' && styles.retryButtonPressed,
                ]}
              >
                <Text style={styles.retryButtonText}>Try again</Text>
              </View>
            )}
          </Pressable>
        </View>
      ) : !offer?.available ? (
        <View style={styles.centered}>
          <Text style={styles.muted}>This chef doesn’t offer a tiffin subscription yet.</Text>
        </View>
      ) : (
        <>
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
          </ScrollView>

          {/* Sticky CTA bar per spec §2.5 — white, top hairline + shadow[2],
              coral filled, radius 8, 52pt. Distinct disabled state. */}
          <View style={styles.footer}>
            <Pressable
              onPress={onSubscribe}
              disabled={!selectionValid || subscribe.isPending}
              accessibilityRole="button"
              accessibilityLabel={`Subscribe${price != null ? ` · ₹${Math.round(price).toLocaleString('en-IN')}` : ''}`}
              android_ripple={
                !selectionValid || subscribe.isPending
                  ? undefined
                  : { color: CANVAS_RIPPLE, borderless: false }
              }
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.cta,
                    (!selectionValid || subscribe.isPending) && styles.ctaDisabled,
                    pressed &&
                      Platform.OS === 'ios' &&
                      selectionValid &&
                      !subscribe.isPending &&
                      styles.ctaPressed,
                  ]}
                >
                  {subscribe.isPending ? (
                    <ActivityIndicator color={customerColors.canvas} />
                  ) : (
                    <Text style={styles.ctaText}>Subscribe{price != null ? ` · ${money(price)}` : ''}</Text>
                  )}
                </View>
              )}
            </Pressable>
          </View>
        </>
      )}
    </SafeAreaView>
  );
}

function Chip({ label, active, onPress }: { label: string; active: boolean; onPress: () => void }) {
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
          {active ? <Check size={13} color={customerColors.coral.pressed} strokeWidth={3} /> : null}
          <Text style={[styles.chipText, active && styles.chipTextActive]}>{label}</Text>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  // White-first canvas per surface model §1 — the price card + chips separate
  // by hairline/border, not a grey page background.
  root: { flex: 1, backgroundColor: customerColors.canvas },
  header: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', paddingHorizontal: 16, paddingVertical: 12 },
  headerTitle: { fontFamily: 'Geist-Bold', fontSize: 20, color: customerColors.charcoal.DEFAULT },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  scroll: { padding: 16, paddingBottom: 24, gap: 6 },
  label: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT, marginTop: 14, marginBottom: 6 },
  chipRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  chip: { flexDirection: 'row', alignItems: 'center', gap: 4, borderWidth: 1, borderColor: customerColors.hairline, borderRadius: 9999, paddingHorizontal: 14, paddingVertical: 8, minHeight: 44, backgroundColor: customerColors.canvas },
  chipActive: { borderColor: customerColors.coral.DEFAULT, backgroundColor: customerColors.coral.tint },
  chipPressed: { backgroundColor: customerColors.surface.soft },
  chipText: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  chipTextActive: { color: customerColors.coral.pressed, fontFamily: 'Inter-SemiBold' },
  skeletonChip: { height: 38, borderRadius: 9999, backgroundColor: customerColors.hairline },
  priceCard: {
    marginTop: 20,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    backgroundColor: customerColors.surface.soft,
    borderRadius: 12,
    padding: 16,
  },
  priceLabel: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft },
  priceValue: {
    fontFamily: 'Geist-Bold',
    fontSize: 22,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  muted: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, marginTop: 8, textAlign: 'center', fontVariant: ['tabular-nums'] },
  retryButton: { marginTop: 16 },
  retryButtonInner: {
    backgroundColor: customerColors.coral.DEFAULT,
    borderRadius: 8,
    paddingHorizontal: 20,
    minHeight: 44,
    alignItems: 'center',
    justifyContent: 'center',
  },
  retryButtonPressed: { backgroundColor: customerColors.coral.pressed },
  retryButtonText: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.canvas },
  // Sticky CTA footer — white, top hairline + shadow[2] (spec §1 floating elements).
  footer: {
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 16,
    backgroundColor: customerColors.canvas,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
    shadowColor: customerTheme.shadow[2].shadowColor,
    shadowOffset: { width: 0, height: -2 },
    shadowOpacity: customerTheme.shadow[2].shadowOpacity,
    shadowRadius: customerTheme.shadow[2].shadowRadius,
    elevation: customerTheme.shadow[2].elevation,
  },
  cta: { backgroundColor: customerColors.coral.DEFAULT, borderRadius: 8, minHeight: 52, alignItems: 'center', justifyContent: 'center' },
  ctaPressed: { backgroundColor: customerColors.coral.pressed },
  ctaDisabled: { backgroundColor: customerColors.surface.soft, borderWidth: 1, borderColor: customerColors.hairline },
  ctaText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: customerColors.canvas, fontVariant: ['tabular-nums'] },
});
