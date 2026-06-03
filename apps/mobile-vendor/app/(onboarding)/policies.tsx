// apps/mobile-vendor/app/(onboarding)/policies.tsx
// Step 5/6 — Terms checkbox + cancellation policy radio.
// StyleSheet only — no NativeWind className.

import {
  Alert,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { useState } from 'react';
import { router } from 'expo-router';
import { OnboardingScaffold } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { useVendorOnboardingStore } from '../../store/onboarding-store';
import {
  VENDOR_TERMS_TEXT,
  CANCELLATION_POLICY_OPTIONS,
  type CancellationPolicy,
} from '../../constants/terms';

export default function PoliciesScreen() {
  const { policies, updatePolicies, setStep } = useVendorOnboardingStore();

  const [acceptedTerms, setAcceptedTerms] = useState<boolean>(policies.acceptedTerms);
  const [cancellationPolicy, setCancellationPolicy] = useState<CancellationPolicy | ''>(
    (policies.cancellationPolicy as CancellationPolicy) || '',
  );

  function onNext(): void {
    if (!acceptedTerms) {
      Alert.alert('Terms required', 'Please accept the terms and conditions to continue.');
      return;
    }
    if (!cancellationPolicy) {
      Alert.alert('Policy required', 'Please select a cancellation policy to continue.');
      return;
    }
    updatePolicies({ acceptedTerms, cancellationPolicy });
    setStep(6);
    router.push('/(onboarding)/review');
  }

  const canContinue = acceptedTerms && cancellationPolicy !== '';

  return (
    <OnboardingScaffold
      step={5}
      total={6}
      title="Policies"
      subtitle="Read, agree, and choose how you handle cancellations."
      primaryLabel="Continue"
      onPrimary={onNext}
      primaryDisabled={!canContinue}
    >
      {/* Terms box — hairline bordered, bone bg */}
      <View style={styles.termsBox}>
        <Text style={styles.termsText}>{VENDOR_TERMS_TEXT}</Text>
      </View>

      {/* Terms checkbox — ink border unchecked, ink fill checked. NOT persimmon. */}
      <Pressable
        onPress={() => setAcceptedTerms((prev) => !prev)}
        style={({ pressed }) => [
          styles.checkRow,
          pressed && styles.checkRowPressed,
        ]}
        accessibilityRole="checkbox"
        accessibilityState={{ checked: acceptedTerms }}
        accessibilityLabel="I accept the terms and conditions"
      >
        <View style={[styles.checkbox, acceptedTerms && styles.checkboxChecked]}>
          {acceptedTerms ? <Text style={styles.checkmark}>✓</Text> : null}
        </View>
        <Text style={styles.checkLabel}>I accept the terms and conditions</Text>
      </Pressable>

      {/* Section divider */}
      <View style={styles.divider} />

      {/* Cancellation policy — three radio rows with description.
          Rationale: three choices each carry a meaningful description
          ("Up to 1 hour before prep start" needs context). A chip row
          truncates labels and forces the user to already know what each
          means. Radio rows with the full label reads clearly at a glance. */}
      <Text style={styles.policyLabel}>Cancellation policy</Text>

      <View style={styles.radioGroup}>
        {CANCELLATION_POLICY_OPTIONS.map((option, idx) => {
          const selected = cancellationPolicy === option.value;
          const isLast = idx === CANCELLATION_POLICY_OPTIONS.length - 1;

          return (
            <Pressable
              key={option.value}
              onPress={() => setCancellationPolicy(option.value)}
              style={({ pressed }) => [
                styles.radioRow,
                !isLast && styles.radioRowBorder,
                pressed && styles.radioRowPressed,
              ]}
              accessibilityRole="radio"
              accessibilityState={{ checked: selected }}
              accessibilityLabel={option.label}
            >
              {/* Radio circle — ink border, ink fill dot when selected. NOT persimmon. */}
              <View style={[styles.radio, selected && styles.radioSelected]}>
                {selected ? <View style={styles.radioDot} /> : null}
              </View>

              <Text style={[styles.radioLabel, selected && styles.radioLabelSelected]}>
                {option.label}
              </Text>
            </Pressable>
          );
        })}
      </View>

      <View style={styles.bottomSpacer} />
    </OnboardingScaffold>
  );
}

const styles = StyleSheet.create({
  // Terms scroll box — bone bg, hairline border, constrained height.
  termsBox: {
    backgroundColor: theme.colors.bone,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    padding: theme.spacing[4],
    maxHeight: 140,
  },

  termsText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    lineHeight: theme.typography.size.bodySm.size * 1.55,
    color: theme.colors.ink.soft,
  },

  // Checkbox row.
  checkRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    minHeight: theme.touchTarget.vendor,
  },

  checkRowPressed: {
    opacity: 0.75,
  },

  checkbox: {
    width: 20,
    height: 20,
    borderRadius: theme.radius.sm,
    borderWidth: 1.5,
    borderColor: theme.colors.ink.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: theme.colors.paper,
    flexShrink: 0,
  },

  checkboxChecked: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderColor: theme.colors.ink.DEFAULT,
  },

  checkmark: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    color: theme.colors.paper,
    lineHeight: 14,
    textAlign: 'center',
  },

  checkLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    flex: 1,
    lineHeight: theme.typography.size.bodySm.size * 1.45,
  },

  divider: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginVertical: theme.spacing[2],
  },

  policyLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    marginBottom: theme.spacing[2],
  },

  // Radio group housed in a hairline card.
  radioGroup: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    overflow: 'hidden',
  },

  radioRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[4],
    minHeight: theme.touchTarget.vendor,
    backgroundColor: theme.colors.paper,
  },

  radioRowBorder: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },

  radioRowPressed: {
    backgroundColor: theme.colors.bone,
  },

  // Radio circle — 20×20, ink border, no fill until selected.
  radio: {
    width: 20,
    height: 20,
    borderRadius: 10,
    borderWidth: 1.5,
    borderColor: theme.colors.mist.strong,
    alignItems: 'center',
    justifyContent: 'center',
    flexShrink: 0,
    backgroundColor: theme.colors.paper,
  },

  radioSelected: {
    borderColor: theme.colors.ink.DEFAULT,
  },

  // Inner dot — ink fill, not persimmon.
  radioDot: {
    width: 10,
    height: 10,
    borderRadius: 5,
    backgroundColor: theme.colors.ink.DEFAULT,
  },

  radioLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    flex: 1,
    lineHeight: theme.typography.size.bodySm.size * 1.45,
  },

  radioLabelSelected: {
    fontFamily: 'Inter-Medium',
    color: theme.colors.ink.DEFAULT,
  },

  bottomSpacer: {
    height: theme.spacing[2],
  },
});
