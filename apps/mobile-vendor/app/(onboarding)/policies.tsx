// apps/mobile-vendor/app/(onboarding)/policies.tsx
// Step 5/6 — Terms checkbox + cancellation policy radio.
// StyleSheet only — no NativeWind className.

import {
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { useState } from 'react';
import { router } from 'expo-router';
import { Check, RotateCcw } from 'lucide-react-native';
import { OnboardingScaffold } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { useVendorOnboardingStore } from '../../store/onboarding-store';
import {
  VENDOR_TERMS_TEXT,
  CANCELLATION_POLICY_OPTIONS,
  type CancellationPolicy,
} from '../../constants/terms';

// Bullet points extracted from the terms text for scannable display
const TERMS_BULLETS = [
  'Maintain food hygiene standards per FSSAI regulations',
  'Ensure accurate menu descriptions and pricing',
  'Prepare orders within your stated prep time',
  'Comply with all applicable local food safety laws',
  'HomeChef may suspend accounts for repeated hygiene complaints',
];

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
      subtitle="Agree to our terms and set how you handle cancellations."
      primaryLabel="Continue"
      onPrimary={onNext}
      primaryDisabled={!canContinue}
    >
      {/* ── TERMS & CONDITIONS ────────────────────────────────── */}
      <View style={styles.sectionLabel}>
        <Text style={styles.sectionLabelText}>TERMS & CONDITIONS</Text>
      </View>

      {/* Scannable bullets instead of a dense text block */}
      <View style={styles.termsBulletCard}>
        {TERMS_BULLETS.map((bullet, idx) => (
          <View
            key={idx}
            style={[styles.bulletRow, idx < TERMS_BULLETS.length - 1 && styles.bulletRowBorder]}
          >
            <View style={styles.bulletDot} />
            <Text style={styles.bulletText}>{bullet}</Text>
          </View>
        ))}
        <View style={styles.termsFooterRow}>
          <Text style={styles.termsFooterText}>
            Full terms at{' '}
            <Text style={styles.termsLink}>homechef.in/vendor-terms</Text>
          </Text>
        </View>
      </View>

      {/* Acceptance row — ink checkbox */}
      <Pressable
        onPress={() => setAcceptedTerms((prev) => !prev)}
        style={({ pressed }) => [
          styles.checkRow,
          acceptedTerms && styles.checkRowAccepted,
          pressed && { opacity: 0.75 },
        ]}
        accessibilityRole="checkbox"
        accessibilityState={{ checked: acceptedTerms }}
        accessibilityLabel="I accept the terms and conditions"
      >
        <View style={[styles.checkbox, acceptedTerms && styles.checkboxChecked]}>
          {acceptedTerms ? (
            <Check size={12} color={theme.colors.paper} strokeWidth={3} />
          ) : null}
        </View>
        <Text style={[styles.checkLabel, acceptedTerms && styles.checkLabelAccepted]}>
          I accept the terms and conditions
        </Text>
      </Pressable>

      {/* ── CANCELLATION POLICY ───────────────────────────────── */}
      <View style={styles.hairline} />

      <View style={styles.sectionLabel}>
        <RotateCcw size={12} color={theme.colors.ink.muted} strokeWidth={2} />
        <Text style={styles.sectionLabelText}>CANCELLATION POLICY</Text>
      </View>
      <Text style={styles.sectionHint}>
        Choose how customers can cancel orders placed with you.
      </Text>

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
                selected && styles.radioRowSelected,
                pressed && !selected && styles.radioRowPressed,
              ]}
              accessibilityRole="radio"
              accessibilityState={{ checked: selected }}
              accessibilityLabel={option.label}
            >
              {/* Radio circle */}
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
  // Section caps label
  sectionLabel: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
    marginBottom: theme.spacing[2],
    marginTop: theme.spacing[1],
  },
  sectionLabelText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    letterSpacing: 1.2,
    color: theme.colors.ink.muted,
  },

  sectionHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[3],
  },

  // Hairline separator
  hairline: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginVertical: theme.spacing[4],
  },

  // Terms bullets card
  termsBulletCard: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    backgroundColor: theme.colors.paper,
    overflow: 'hidden',
    marginBottom: theme.spacing[3],
  },

  bulletRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  bulletRowBorder: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  bulletDot: {
    width: 5,
    height: 5,
    borderRadius: 2.5,
    backgroundColor: theme.colors.ink.soft,
    marginTop: 7,
    flexShrink: 0,
  },
  bulletText: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    lineHeight: theme.typography.size.bodySm.size * 1.5,
  },

  termsFooterRow: {
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    backgroundColor: theme.colors.bone,
  },
  termsFooterText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
  },
  termsLink: {
    fontFamily: 'Inter-Medium',
    color: theme.colors.ink.soft,
    textDecorationLine: 'underline',
  },

  // Checkbox acceptance row
  checkRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingVertical: theme.spacing[3],
    paddingHorizontal: theme.spacing[3],
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    minHeight: theme.touchTarget.vendor,
    backgroundColor: theme.colors.paper,
  },
  checkRowAccepted: {
    borderColor: theme.colors.ink.DEFAULT,
    backgroundColor: theme.colors.bone,
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

  checkLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    flex: 1,
    lineHeight: theme.typography.size.bodySm.size * 1.45,
  },
  checkLabelAccepted: {
    fontFamily: 'Inter-Medium',
    color: theme.colors.ink.DEFAULT,
  },

  // Radio group — hairline card
  radioGroup: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    overflow: 'hidden',
    backgroundColor: theme.colors.paper,
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
  radioRowSelected: {
    backgroundColor: theme.colors.bone,
  },
  radioRowPressed: {
    backgroundColor: theme.colors.bone,
  },

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
    height: theme.spacing[4],
  },
});
