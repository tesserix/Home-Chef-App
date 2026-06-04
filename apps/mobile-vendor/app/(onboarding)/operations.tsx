// apps/mobile-vendor/app/(onboarding)/operations.tsx
// Step 3/6 — Operating hours per day, prep time, service radius.
// StyleSheet only — no NativeWind className.

import {
  Alert,
  StyleSheet,
  Switch,
  Text,
  View,
} from 'react-native';
import { useState } from 'react';
import { router } from 'expo-router';
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
  const { operations, updateOperations, setStep } = useVendorOnboardingStore();

  const [hours, setHours] = useState<HoursMap>(operations.operatingHours);
  const [prepTime, setPrepTime] = useState<string>(operations.prepTime);
  const [serviceRadius, setServiceRadius] = useState<string>(
    String(operations.serviceRadius),
  );

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
    const radius = parseInt(serviceRadius, 10);
    if (Number.isNaN(radius) || radius < 1 || radius > 50) {
      Alert.alert('Validation Error', 'Service radius must be between 1 and 50 km');
      return;
    }
    updateOperations({ operatingHours: hours, prepTime, serviceRadius: radius });
    setStep(4);
    router.push('/(onboarding)/documents');
  }

  return (
    <OnboardingScaffold
      step={3}
      total={6}
      title="Operations"
      subtitle="When you cook, how fast, and how far you deliver."
      primaryLabel="Continue"
      onPrimary={onNext}
    >
      {/* Operating hours — compact stacked list.
          Rationale: 7 rows fits without horizontal scroll and the switch + hours
          per row is the natural mental model ("is Monday open? yes — 9am–9pm").
          Grouping by weekday/weekend adds a second navigation layer without
          saving space. Compact stacked list wins. */}
      <View style={styles.sectionHeader}>
        <Text style={styles.sectionTitle}>Operating hours</Text>
        <Text style={styles.sectionHint}>Toggle to mark closed</Text>
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

              {/* Switch — ink track / paper thumb (vendor density) */}
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
                <Text style={styles.closedLabel}>Closed</Text>
              )}
            </View>
          );
        })}
      </View>

      {/* Prep time chips */}
      <View style={styles.sectionGap}>
        <Text style={styles.sectionTitle}>Prep time</Text>
        <Text style={styles.sectionHint}>Average time per order</Text>
      </View>
      <View style={styles.chipRow}>
        {PREP_TIME_OPTIONS.map((option) => {
          const selected = prepTime === option;
          return (
            <View
              key={option}
              style={[styles.chip, selected && styles.chipActive]}
            >
              <Text
                style={[styles.chipLabel, selected && styles.chipLabelActive]}
                onPress={() => setPrepTime(option)}
                suppressHighlighting
              >
                {option}
              </Text>
            </View>
          );
        })}
      </View>

      {/* Service radius */}
      <View style={styles.sectionGap}>
        <Input
          label="Service radius (km)"
          placeholder="1–50 km"
          value={serviceRadius}
          onChangeText={setServiceRadius}
          keyboardType="number-pad"
          maxLength={2}
          helper="Maximum distance you will accept orders from."
        />
      </View>

      <View style={styles.bottomSpacer} />
    </OnboardingScaffold>
  );
}

const styles = StyleSheet.create({
  // Section header: title left, hint right.
  sectionHeader: {
    flexDirection: 'row',
    alignItems: 'baseline',
    justifyContent: 'space-between',
    marginBottom: theme.spacing[2],
  },

  sectionGap: {
    marginTop: theme.spacing[5],
  },

  sectionTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },

  sectionHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
  },

  // The 7-row hours block sits inside a hairline-bordered card. This
  // groups the rows visually without adding heavy card elevation.
  hoursCard: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    overflow: 'hidden',
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
    width: 32, // fixed — keeps switch column aligned across all rows
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

  // Compact time input — no label, centre-aligned value.
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

  // Prep time outlined pill chips — same pattern as kitchen-details cuisines.
  chipRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: theme.spacing[2],
    marginTop: theme.spacing[2],
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

  bottomSpacer: {
    height: theme.spacing[2],
  },
});
