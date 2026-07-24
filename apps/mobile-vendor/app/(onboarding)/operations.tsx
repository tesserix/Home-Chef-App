// apps/mobile-vendor/app/(onboarding)/operations.tsx
// Step 3/6 — Operating hours per day, prep time, service radius.
// StyleSheet only — no NativeWind className.

import {
  Alert,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  View,
} from 'react-native';
import { useRef, useState } from 'react';
import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import { Clock, Timer, MapPin } from 'lucide-react-native';
import { Input, OnboardingScaffold } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { useVendorOnboardingStore } from '../../store/onboarding-store';

type DayHours = { open: string; close: string; closed: boolean };
type HoursMap = Record<string, DayHours>;

const DAYS = [
  'monday',
  'tuesday',
  'wednesday',
  'thursday',
  'friday',
  'saturday',
  'sunday',
] as const;
type Day = (typeof DAYS)[number];

const DAY_LABELS: Record<Day, string> = {
  monday: 'Mon',
  tuesday: 'Tue',
  wednesday: 'Wed',
  thursday: 'Thu',
  friday: 'Fri',
  saturday: 'Sat',
  sunday: 'Sun',
};

const PREP_TIME_OPTIONS = ['15min', '30min', '45min', '60min', '90min'] as const;

export default function OperationsScreen() {
  const { t } = useTranslation();
  const { operations, updateOperations, setStep } = useVendorOnboardingStore();

  const [hours, setHours] = useState<HoursMap>(operations.operatingHours);
  const [prepTime, setPrepTime] = useState<string>(operations.prepTime);
  const [serviceRadius, setServiceRadius] = useState<string>(
    String(operations.serviceRadius),
  );
  const [offersPickup, setOffersPickup] = useState<boolean>(operations.offersPickup);
  const [offersSelfDelivery, setOffersSelfDelivery] = useState<boolean>(
    operations.offersSelfDelivery,
  );

  // R14 — scroll the section with the validation problem into view instead
  // of leaving the alert as the only signal. OnboardingScaffold owns the
  // ScrollView, so we forward a ref into it via its `scrollRef` prop.
  const scrollRef = useRef<ScrollView>(null);
  const fulfillmentFieldY = useRef(0);
  const radiusFieldY = useRef(0);

  function updateDay(day: Day, field: 'open' | 'close', value: string): void {
    setHours((prev) => ({
      ...prev,
      [day]: { ...prev[day], [field]: value },
    }));
  }

  function toggleDay(day: Day, isOpen: boolean): void {
    setHours((prev) => ({
      ...prev,
      [day]: { ...prev[day], closed: !isOpen },
    }));
  }

  function onNext(): void {
    // At least one fulfillment method — a kitchen offering neither can't be
    // activated by admin, so block it here with a clear reason instead.
    if (!offersPickup && !offersSelfDelivery) {
      scrollRef.current?.scrollTo({ y: Math.max(0, fulfillmentFieldY.current - 16), animated: true });
      Alert.alert(t('onboarding.validationError'), t('onboarding.fulfillmentError'));
      return;
    }
    // Radius only matters when the chef self-delivers.
    const radius = parseInt(serviceRadius, 10);
    if (offersSelfDelivery && (Number.isNaN(radius) || radius < 1 || radius > 50)) {
      scrollRef.current?.scrollTo({ y: Math.max(0, radiusFieldY.current - 16), animated: true });
      Alert.alert(t('onboarding.validationError'), t('onboarding.radiusError'));
      return;
    }
    updateOperations({
      operatingHours: hours,
      prepTime,
      serviceRadius: Number.isNaN(radius) ? operations.serviceRadius : radius,
      offersPickup,
      offersSelfDelivery,
    });
    setStep(4);
    router.push('/(onboarding)/documents');
  }

  return (
    <OnboardingScaffold
      step={3}
      total={6}
      stepName={t('onboarding.stepOperations')}
      title={t('onboarding.operationsTitle')}
      subtitle={t('onboarding.operationsSubtitle')}
      primaryLabel={t('onboarding.continue')}
      onPrimary={onNext}
      scrollRef={scrollRef}
    >
      {/* ── WHEN YOU COOK ──────────────────────────────────────── */}
      <View style={styles.sectionLabel}>
        <Clock size={12} color={theme.colors.ink.muted} strokeWidth={2} />
        <Text style={styles.sectionLabelText}>{t('onboarding.whenYouCook')}</Text>
      </View>

      <View style={styles.sectionHint}>
        <Text style={styles.hintText}>{t('onboarding.whenYouCookHint')}</Text>
      </View>

      <View style={styles.hoursCard}>
        {DAYS.map((day, idx) => {
          const dayData = hours[day] ?? { open: '09:00', close: '21:00', closed: false };
          const isLast = idx === DAYS.length - 1;

          return (
            <View
              key={day}
              style={[styles.dayRow, !isLast && styles.dayRowBorder]}
            >
              {/* Day label — fixed width keeps alignment across rows */}
              <Text style={styles.dayLabel}>{DAY_LABELS[day]}</Text>

              {/* Switch — ink track / paper thumb */}
              <Switch
                value={!dayData.closed}
                onValueChange={(val) => toggleDay(day, val)}
                trackColor={{
                  false: theme.colors.mist.strong,
                  true: theme.colors.ink.DEFAULT,
                }}
                thumbColor={theme.colors.paper}
                ios_backgroundColor={theme.colors.mist.strong}
              />

              {/* Hours inputs or "Closed" label */}
              {!dayData.closed ? (
                <View style={styles.hoursInputRow}>
                  <View style={styles.timeInputWrap}>
                    <Input
                      value={dayData.open}
                      onChangeText={(val) => updateDay(day, 'open', val)}
                      placeholder="09:00"
                      maxLength={5}
                      keyboardType="numbers-and-punctuation"
                      inputStyle={styles.timeInput}
                      label=""
                      accessibilityLabel={`${DAY_LABELS[day]} opening time`}
                    />
                  </View>
                  <Text style={styles.timeRangeDash}>–</Text>
                  <View style={styles.timeInputWrap}>
                    <Input
                      value={dayData.close}
                      onChangeText={(val) => updateDay(day, 'close', val)}
                      placeholder="21:00"
                      maxLength={5}
                      keyboardType="numbers-and-punctuation"
                      inputStyle={styles.timeInput}
                      label=""
                      accessibilityLabel={`${DAY_LABELS[day]} closing time`}
                    />
                  </View>
                </View>
              ) : (
                <Text style={styles.closedLabel}>{t('onboarding.closed')}</Text>
              )}
            </View>
          );
        })}
      </View>

      {/* ── HOW FAST ───────────────────────────────────────────── */}
      <View style={styles.hairline} />

      <View style={styles.sectionLabel}>
        <Timer size={12} color={theme.colors.ink.muted} strokeWidth={2} />
        <Text style={styles.sectionLabelText}>{t('onboarding.prepTime')}</Text>
      </View>
      <Text style={styles.hintText}>{t('onboarding.prepTimeHint')}</Text>

      <View style={styles.chipRow}>
        {PREP_TIME_OPTIONS.map((option) => {
          const selected = prepTime === option;
          return (
            <Pressable
              key={option}
              onPress={() => setPrepTime(option)}
              accessibilityRole="button"
              accessibilityLabel={option}
              accessibilityState={{ selected }}
              android_ripple={{
                color: selected
                  ? `${theme.colors.paper}30`
                  : `${theme.colors.ink.DEFAULT}14`,
                borderless: false,
              }}
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.chip,
                    selected && styles.chipActive,
                    pressed && Platform.OS === 'ios' && { opacity: 0.75 },
                  ]}
                >
                  <Text style={[styles.chipLabel, selected && styles.chipLabelActive]}>
                    {option}
                  </Text>
                </View>
              )}
            </Pressable>
          );
        })}
      </View>

      {/* ── HOW CUSTOMERS GET THEIR FOOD ───────────────────────── */}
      <View style={styles.hairline} />

      <View style={styles.sectionLabel}>
        <MapPin size={12} color={theme.colors.ink.muted} strokeWidth={2} />
        <Text style={styles.sectionLabelText}>{t('onboarding.fulfillmentLabel')}</Text>
      </View>
      <View style={styles.sectionHint}>
        <Text style={styles.hintText}>{t('onboarding.fulfillmentHint')}</Text>
      </View>

      <View
        style={styles.fieldCard}
        onLayout={(e) => {
          fulfillmentFieldY.current = e.nativeEvent.layout.y;
        }}
      >
        <View style={styles.fulfillRow}>
          <View style={styles.fulfillText}>
            <Text style={styles.fulfillTitle}>{t('onboarding.pickupTitle')}</Text>
            <Text style={styles.fulfillSub}>{t('onboarding.pickupSub')}</Text>
          </View>
          <Switch
            value={offersPickup}
            onValueChange={setOffersPickup}
            trackColor={{ false: theme.colors.mist.strong, true: theme.colors.ink.DEFAULT }}
            thumbColor={theme.colors.paper}
            ios_backgroundColor={theme.colors.mist.strong}
          />
        </View>

        <View style={styles.innerHairline} />

        <View style={styles.fulfillRow}>
          <View style={styles.fulfillText}>
            <Text style={styles.fulfillTitle}>{t('onboarding.selfDeliveryTitle')}</Text>
            <Text style={styles.fulfillSub}>{t('onboarding.selfDeliverySub')}</Text>
          </View>
          <Switch
            value={offersSelfDelivery}
            onValueChange={setOffersSelfDelivery}
            trackColor={{ false: theme.colors.mist.strong, true: theme.colors.ink.DEFAULT }}
            thumbColor={theme.colors.paper}
            ios_backgroundColor={theme.colors.mist.strong}
          />
        </View>
      </View>

      {/* ── HOW FAR — only relevant when the chef self-delivers ──── */}
      {offersSelfDelivery ? (
        <>
          <View style={styles.hairline} />
          <View style={styles.sectionLabel}>
            <MapPin size={12} color={theme.colors.ink.muted} strokeWidth={2} />
            <Text style={styles.sectionLabelText}>{t('onboarding.deliveryArea')}</Text>
          </View>
          <View
            style={styles.fieldCard}
            onLayout={(e) => {
              radiusFieldY.current = e.nativeEvent.layout.y;
            }}
          >
            <Input
              label={t('onboarding.serviceRadius')}
              placeholder={t('onboarding.serviceRadiusPlaceholder')}
              value={serviceRadius}
              onChangeText={setServiceRadius}
              keyboardType="number-pad"
              maxLength={2}
              helper={t('onboarding.serviceRadiusHelper')}
            />
          </View>
        </>
      ) : null}

      <View style={styles.bottomSpacer} />
    </OnboardingScaffold>
  );
}

const styles = StyleSheet.create({
  // Section label row: small icon + caps text
  sectionLabel: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
    marginBottom: theme.spacing[2],
  },
  sectionLabelText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    letterSpacing: 1.2,
    color: theme.colors.ink.muted,
  },

  sectionHint: {
    marginBottom: theme.spacing[2],
  },
  fulfillRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: theme.spacing[3],
    paddingVertical: theme.spacing[1],
  },
  innerHairline: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginVertical: theme.spacing[3],
  },
  fulfillText: { flex: 1, gap: 2 },
  fulfillTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  fulfillSub: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
  },
  hintText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[2],
  },

  // Hairline separator between sections
  hairline: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginVertical: theme.spacing[4],
  },

  // The 7-row hours block — hairline-bordered card
  hoursCard: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    overflow: 'hidden',
    backgroundColor: theme.colors.paper,
  },

  dayRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    gap: theme.spacing[3],
    backgroundColor: theme.colors.paper,
    minHeight: theme.touchTarget.vendor,
  },

  dayRowBorder: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },

  dayLabel: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    width: 32,
  },

  hoursInputRow: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
    justifyContent: 'flex-end',
  },

  timeInputWrap: {
    width: 64,
  },

  timeInput: {
    textAlign: 'center',
    fontSize: theme.typography.size.bodySm.size,
    fontVariant: ['tabular-nums'],
    paddingHorizontal: theme.spacing[2],
    paddingVertical: theme.spacing[1],
  },

  timeRangeDash: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
  },

  closedLabel: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    textAlign: 'right',
  },

  // Prep time chips
  chipRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: theme.spacing[2],
    marginTop: theme.spacing[1],
  },

  chip: {
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    borderRadius: theme.radius.full,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[2],
    minHeight: theme.touchTarget.vendor,
    alignItems: 'center',
    justifyContent: 'center',
  },

  chipActive: {
    borderColor: theme.colors.ink.DEFAULT,
    backgroundColor: theme.colors.ink.DEFAULT,
  },

  chipLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },

  chipLabelActive: {
    fontFamily: 'Inter-Medium',
    color: theme.colors.paper,
  },

  // Service radius field card
  fieldCard: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    padding: theme.spacing[4],
    backgroundColor: theme.colors.paper,
  },

  bottomSpacer: {
    height: theme.spacing[4],
  },
});
