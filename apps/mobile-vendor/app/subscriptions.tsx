import { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Platform,
  Pressable,
  StyleSheet,
  Switch,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { getServerErrorMessage } from '@homechef/mobile-shared/api';
import { theme } from '@homechef/mobile-shared/theme';
import { Button, KeyboardAwareScrollView, useAlert } from '@homechef/mobile-shared/ui';
import {
  EMPTY_SUBSCRIPTION_CONFIG,
  SUBSCRIPTION_CADENCES,
  SUBSCRIPTION_SLOTS,
  useSubscriptionConfig,
  useUpdateSubscriptionConfig,
  type SubscriptionConfig,
} from '../hooks/useSubscriptionConfig';

const HHMM = /^([01]\d|2[0-3]):[0-5]\d$/;

/** Blank or nonsense reads as zero, never NaN — the API rejects NaN outright. */
function toAmount(raw: string): number {
  const n = parseFloat(raw.trim());
  return Number.isFinite(n) && n >= 0 ? n : 0;
}

function toCount(raw: string): number {
  const n = parseInt(raw.trim(), 10);
  return Number.isFinite(n) && n > 0 ? n : 0;
}

/** "" for a zero so the field shows a placeholder rather than a literal 0. */
function amountText(n: number): string {
  return n > 0 ? String(n) : '';
}

// Chef tiffin-subscription setup (#284) — the mobile counterpart of the vendor
// portal's Subscriptions page. A chef who runs their kitchen from the phone
// could previously configure everything EXCEPT the recurring offer, which meant
// opening a laptop to set the one price a subscriber is actually charged.
export default function SubscriptionsScreen() {
  const { showAlert } = useAlert();
  const { data, isLoading } = useSubscriptionConfig();
  const save = useUpdateSubscriptionConfig();

  const hasPublishedMenu = data?.hasPublishedMenu ?? false;

  const [form, setForm] = useState<SubscriptionConfig>(EMPTY_SUBSCRIPTION_CONFIG);
  // Money/count fields are held as text so a half-typed "12." doesn't get
  // rounded out from under the chef mid-keystroke.
  const [perMeal, setPerMeal] = useState('');
  const [deliveryFee, setDeliveryFee] = useState('');
  const [dailyCapacity, setDailyCapacity] = useState('');
  const [trialPrice, setTrialPrice] = useState('');
  const [trialDays, setTrialDays] = useState('');
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    if (hydrated || !data?.config) return;
    const cfg = {
      ...EMPTY_SUBSCRIPTION_CONFIG,
      ...data.config,
      slots: data.config.slots ?? EMPTY_SUBSCRIPTION_CONFIG.slots,
      cadences: data.config.cadences ?? EMPTY_SUBSCRIPTION_CONFIG.cadences,
      cutoffTime: data.config.cutoffTime || EMPTY_SUBSCRIPTION_CONFIG.cutoffTime,
    };
    setForm(cfg);
    setPerMeal(amountText(cfg.perMealPrice));
    setDeliveryFee(amountText(cfg.deliveryFee));
    setDailyCapacity(amountText(cfg.dailyCapacity));
    setTrialPrice(amountText(cfg.trialPrice));
    setTrialDays(cfg.trialDurationDays > 0 ? String(cfg.trialDurationDays) : '');
    setHydrated(true);
  }, [data, hydrated]);

  function toggleIn(list: string[], value: string): string[] {
    return list.includes(value) ? list.filter((v) => v !== value) : [...list, value];
  }

  function onSave(): void {
    const cutoff = form.cutoffTime.trim() || '21:00';
    if (!HHMM.test(cutoff)) {
      showAlert('Invalid cutoff', 'Order cutoff must be HH:MM (24h), e.g. 21:00.');
      return;
    }
    // Mirror the server's two enable-time rules so the chef is told what is
    // wrong before a round-trip, not after a 400.
    if (form.enabled && !hasPublishedMenu) {
      showAlert(
        'Publish a weekly menu first',
        'Subscriptions need a published weekly menu so customers can see what they are signing up for.',
      );
      return;
    }
    if (form.enabled && toAmount(perMeal) <= 0) {
      showAlert('Set a per-meal price', 'A subscription needs a price per meal before it can go live.');
      return;
    }
    if (form.enabled && form.slots.length === 0) {
      showAlert('Pick at least one meal', 'Choose which meals the subscription covers.');
      return;
    }
    if (form.enabled && form.cadences.length === 0) {
      showAlert('Pick at least one plan length', 'Choose weekly, monthly, or both.');
      return;
    }

    save.mutate(
      {
        ...form,
        cutoffTime: cutoff,
        perMealPrice: toAmount(perMeal),
        deliveryFee: toAmount(deliveryFee),
        dailyCapacity: toCount(dailyCapacity),
        trialPrice: toAmount(trialPrice),
        trialDurationDays: toCount(trialDays),
      },
      {
        onSuccess: () => showAlert('Saved', 'Your subscription settings are updated.'),
        // Server messages here are already chef-readable ("Set a per-meal
        // price"), so pass them straight through rather than flattening every
        // rejection into one generic line.
        onError: (err) =>
          showAlert('Could not save', getServerErrorMessage(err, 'Please try again.')),
      },
    );
  }

  if (isLoading) {
    return (
      <SafeAreaView style={styles.centered}>
        <ActivityIndicator color={theme.colors.ink.DEFAULT} />
      </SafeAreaView>
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
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
        >
          {({ pressed }) => (
            <View style={pressed && Platform.OS === 'ios' ? { opacity: 0.6 } : null}>
              <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text style={styles.title}>Subscriptions</Text>
        <View style={{ width: 24 }} />
      </View>

      <KeyboardAwareScrollView contentContainerStyle={styles.scroll}>
        {/* Master switch */}
        <View style={styles.card}>
          <View style={styles.rowBetween}>
            <View style={{ flex: 1 }}>
              <Text style={styles.cardTitle}>Offer subscriptions</Text>
              <Text style={styles.caption}>
                Let customers subscribe to a repeating tiffin instead of ordering each day.
              </Text>
            </View>
            <Switch
              value={form.enabled}
              onValueChange={(enabled) => setForm((f) => ({ ...f, enabled }))}
              disabled={!hasPublishedMenu && !form.enabled}
              trackColor={{ true: theme.colors.ink.DEFAULT }}
            />
          </View>
          {!hasPublishedMenu ? (
            <Pressable
              onPress={() => router.push('/meal-plans/weekly-menu' as never)}
              accessibilityRole="button"
              accessibilityLabel="Publish a weekly menu"
            >
              {({ pressed }) => (
                <View style={[styles.notice, pressed && { opacity: 0.7 }]}>
                  <Text style={styles.noticeText}>
                    Publish a weekly menu before turning this on — customers need to see what a
                    subscription includes.
                  </Text>
                  <Text style={styles.noticeAction}>Set up weekly menu →</Text>
                </View>
              )}
            </Pressable>
          ) : null}
        </View>

        {/* Meals covered */}
        <View style={styles.card}>
          <Text style={styles.cardTitle}>Meals covered</Text>
          <Text style={styles.caption}>Which meals a subscription delivers.</Text>
          <View style={styles.chipRow}>
            {SUBSCRIPTION_SLOTS.map((slot) => (
              <Chip
                key={slot.value}
                label={slot.label}
                selected={form.slots.includes(slot.value)}
                onPress={() => setForm((f) => ({ ...f, slots: toggleIn(f.slots, slot.value) }))}
              />
            ))}
          </View>
        </View>

        {/* Plan lengths */}
        <View style={styles.card}>
          <Text style={styles.cardTitle}>Plan lengths</Text>
          <Text style={styles.caption}>How long a customer can commit for.</Text>
          <View style={styles.chipRow}>
            {SUBSCRIPTION_CADENCES.map((c) => (
              <Chip
                key={c.value}
                label={c.label}
                selected={form.cadences.includes(c.value)}
                onPress={() => setForm((f) => ({ ...f, cadences: toggleIn(f.cadences, c.value) }))}
              />
            ))}
          </View>
        </View>

        {/* Pricing */}
        <View style={styles.card}>
          <Text style={styles.cardTitle}>Pricing</Text>
          <FieldRow
            label="Price per meal"
            prefix="₹"
            value={perMeal}
            onChange={setPerMeal}
            placeholder="0"
            keyboardType="decimal-pad"
            accessibilityLabel="Price per meal in rupees"
          />
          <FieldRow
            label="Delivery fee per meal"
            prefix="₹"
            value={deliveryFee}
            onChange={setDeliveryFee}
            placeholder="0"
            keyboardType="decimal-pad"
            accessibilityLabel="Delivery fee per meal in rupees"
          />
          <Text style={styles.hint}>
            Charged on every meal in the plan. Leave the fee at 0 if delivery is included.
          </Text>
        </View>

        {/* Capacity + cutoff */}
        <View style={styles.card}>
          <Text style={styles.cardTitle}>Capacity & cutoff</Text>
          <FieldRow
            label="Subscribers per day"
            value={dailyCapacity}
            onChange={setDailyCapacity}
            placeholder="No limit"
            keyboardType="number-pad"
            accessibilityLabel="Maximum subscribers per day"
          />
          <FieldRow
            label="Order cutoff (24h)"
            value={form.cutoffTime}
            onChange={(cutoffTime) => setForm((f) => ({ ...f, cutoffTime }))}
            placeholder="21:00"
            keyboardType="numbers-and-punctuation"
            accessibilityLabel="Daily order cutoff time, 24 hour HH:MM"
          />
          <Text style={styles.hint}>
            Leave the daily limit blank for no cap. The cutoff is the last time a subscriber can
            change tomorrow&apos;s meal.
          </Text>
        </View>

        {/* Free / discounted trial */}
        <View style={styles.card}>
          <View style={styles.rowBetween}>
            <View style={{ flex: 1 }}>
              <Text style={styles.cardTitle}>Trial period</Text>
              <Text style={styles.caption}>
                Offer the first few days at a lower price so customers can try your food.
              </Text>
            </View>
            <Switch
              value={form.trialEnabled}
              onValueChange={(trialEnabled) => setForm((f) => ({ ...f, trialEnabled }))}
              trackColor={{ true: theme.colors.ink.DEFAULT }}
            />
          </View>
          {form.trialEnabled ? (
            <>
              <FieldRow
                label="Trial length (days)"
                value={trialDays}
                onChange={setTrialDays}
                placeholder="3"
                keyboardType="number-pad"
                accessibilityLabel="Trial length in days"
              />
              <FieldRow
                label="Trial price per meal"
                prefix="₹"
                value={trialPrice}
                onChange={setTrialPrice}
                placeholder="0"
                keyboardType="decimal-pad"
                accessibilityLabel="Trial price per meal in rupees"
              />
            </>
          ) : null}
        </View>

        <Button
          label="Save settings"
          variant="primary"
          size="lg"
          loading={save.isPending}
          onPress={onSave}
        />
      </KeyboardAwareScrollView>
    </SafeAreaView>
  );
}

function Chip({
  label,
  selected,
  onPress,
}: {
  label: string;
  selected: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="checkbox"
      accessibilityState={{ checked: selected }}
      accessibilityLabel={label}
    >
      {({ pressed }) => (
        <View style={[styles.chip, selected && styles.chipSelected, pressed && { opacity: 0.8 }]}>
          <Text style={[styles.chipText, selected && styles.chipTextSelected]}>{label}</Text>
        </View>
      )}
    </Pressable>
  );
}

function FieldRow({
  label,
  value,
  onChange,
  placeholder,
  prefix,
  keyboardType,
  accessibilityLabel,
}: {
  label: string;
  value: string;
  onChange: (s: string) => void;
  placeholder: string;
  prefix?: string;
  keyboardType: 'decimal-pad' | 'number-pad' | 'numbers-and-punctuation';
  accessibilityLabel: string;
}) {
  return (
    <View style={styles.fieldRow}>
      <Text style={styles.fieldLabel}>{label}</Text>
      <View style={styles.fieldInputWrap}>
        {prefix ? <Text style={styles.fieldPrefix}>{prefix}</Text> : null}
        <TextInput
          style={styles.fieldInput}
          value={value}
          onChangeText={onChange}
          placeholder={placeholder}
          placeholderTextColor={theme.colors.ink.muted}
          keyboardType={keyboardType}
          accessibilityLabel={accessibilityLabel}
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  title: { fontFamily: 'Geist-Bold', fontSize: 20, color: theme.colors.ink.DEFAULT },
  scroll: { padding: theme.spacing[4], gap: theme.spacing[4], paddingBottom: theme.spacing[10] },
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    padding: theme.spacing[4],
    ...theme.shadow[1],
  },
  rowBetween: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    minHeight: 56,
  },
  cardTitle: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: theme.colors.ink.DEFAULT },
  caption: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.muted, marginTop: 2 },
  hint: {
    fontFamily: 'Inter',
    fontSize: 12,
    lineHeight: 17,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[3],
  },
  notice: {
    marginTop: theme.spacing[3],
    backgroundColor: theme.colors.amber.tint,
    borderRadius: theme.radius.DEFAULT,
    padding: theme.spacing[3],
    gap: 4,
  },
  noticeText: {
    fontFamily: 'Inter',
    fontSize: 13,
    lineHeight: 18,
    color: theme.colors.ink.DEFAULT,
  },
  noticeAction: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: theme.colors.ink.DEFAULT,
  },
  chipRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: theme.spacing[2],
    marginTop: theme.spacing[3],
  },
  chip: {
    minHeight: 40,
    justifyContent: 'center',
    borderRadius: theme.radius.full,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    paddingHorizontal: theme.spacing[4],
  },
  chipSelected: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderColor: theme.colors.ink.DEFAULT,
  },
  chipText: { fontFamily: 'Inter-Medium', fontSize: 14, color: theme.colors.ink.DEFAULT },
  chipTextSelected: { color: theme.colors.paper },
  fieldRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: theme.spacing[3],
    marginTop: theme.spacing[3],
    minHeight: 44,
  },
  fieldLabel: { flex: 1, fontFamily: 'Inter', fontSize: 15, color: theme.colors.ink.DEFAULT },
  fieldInputWrap: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 2,
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[2],
    minWidth: 110,
  },
  fieldPrefix: { fontFamily: 'Inter', fontSize: 15, color: theme.colors.ink.muted },
  fieldInput: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: theme.colors.ink.DEFAULT,
    textAlign: 'right',
    paddingVertical: theme.spacing[2],
  },
});
