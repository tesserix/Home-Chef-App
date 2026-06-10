// apps/mobile-vendor/app/(onboarding)/policies.tsx
// Step 5/6 — Terms checkbox + cancellation policy radio.
// StyleSheet only — no NativeWind className.

import {
  Alert,
  Linking,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { useState } from 'react';
import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import { Check, RotateCcw } from 'lucide-react-native';
import { OnboardingScaffold } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { useVendorOnboardingStore } from '../../store/onboarding-store';
import {
  CANCELLATION_POLICY_OPTIONS,
  type CancellationPolicy,
} from '../../constants/terms';

const VENDOR_TERMS_URL = 'https://fe3dr.com/vendor-terms';
const VENDOR_TERMS_LABEL = 'fe3dr.com/vendor-terms';

// i18n keys for the scannable terms bullets (under the `onboarding` namespace)
const TERMS_BULLET_KEYS = [
  'onboarding.termsBullet1',
  'onboarding.termsBullet2',
  'onboarding.termsBullet3',
  'onboarding.termsBullet4',
  'onboarding.termsBullet5',
];

export default function PoliciesScreen() {
  const { t } = useTranslation();
  const { policies, updatePolicies, setStep } = useVendorOnboardingStore();

  const [acceptedTerms, setAcceptedTerms] = useState<boolean>(policies.acceptedTerms);
  const [cancellationPolicy, setCancellationPolicy] = useState<CancellationPolicy | ''>(
    (policies.cancellationPolicy as CancellationPolicy) || '',
  );

  function onNext(): void {
    if (!acceptedTerms) {
      Alert.alert(t('onboarding.termsRequired'), t('onboarding.termsRequiredBody'));
      return;
    }
    if (!cancellationPolicy) {
      Alert.alert(t('onboarding.policyRequired'), t('onboarding.policyRequiredBody'));
      return;
    }
    updatePolicies({ acceptedTerms, cancellationPolicy });
    setStep(6);
    router.push('/(onboarding)/review');
  }

  const canContinue = acceptedTerms && cancellationPolicy !== '';

  // Maps each cancellation policy value to its i18n key, falling back to the
  // constant's English label if no key is registered.
  const policyLabelKey: Record<string, string> = {
    no_cancellations: 'onboarding.policyNoCancellations',
    up_to_1_hour: 'onboarding.policyUpTo1Hour',
    up_to_30_mins: 'onboarding.policyUpTo30Mins',
  };

  return (
    <OnboardingScaffold
      step={5}
      total={6}
      title={t('onboarding.policiesTitle')}
      subtitle={t('onboarding.policiesSubtitle')}
      primaryLabel={t('onboarding.continue')}
      onPrimary={onNext}
      primaryDisabled={!canContinue}
    >
      {/* ── TERMS & CONDITIONS ────────────────────────────────── */}
      <View style={styles.sectionLabel}>
        <Text style={styles.sectionLabelText}>{t('onboarding.termsConditions')}</Text>
      </View>

      {/* Scannable bullets instead of a dense text block */}
      <View style={styles.termsBulletCard}>
        {TERMS_BULLET_KEYS.map((bulletKey, idx) => (
          <View
            key={idx}
            style={[styles.bulletRow, idx < TERMS_BULLET_KEYS.length - 1 && styles.bulletRowBorder]}
          >
            <View style={styles.bulletDot} />
            <Text style={styles.bulletText}>{t(bulletKey)}</Text>
          </View>
        ))}
        <View style={styles.termsFooterRow}>
          <Text style={styles.termsFooterText}>
            {t('onboarding.fullTermsAt')}
            <Text
              style={styles.termsLink}
              onPress={() => {
                Linking.openURL(VENDOR_TERMS_URL).catch(() => undefined);
              }}
              accessibilityRole="link"
              accessibilityLabel={`Open ${VENDOR_TERMS_LABEL} in browser`}
            >
              {VENDOR_TERMS_LABEL}
            </Text>
          </Text>
        </View>
      </View>

      {/* Acceptance row — ink checkbox. Outer Pressable carries only the
          pressed opacity; inner View owns flex/bg/border. iOS drops those
          styles when applied via the Pressable's function-style prop. */}
      <Pressable
        onPress={() => setAcceptedTerms((prev) => !prev)}
        style={({ pressed }) => ({ opacity: pressed ? 0.75 : 1 })}
        accessibilityRole="checkbox"
        accessibilityState={{ checked: acceptedTerms }}
        accessibilityLabel={t('onboarding.acceptTerms')}
      >
        <View style={[styles.checkRow, acceptedTerms && styles.checkRowAccepted]}>
          <View style={[styles.checkbox, acceptedTerms && styles.checkboxChecked]}>
            {acceptedTerms ? (
              <Check size={12} color={theme.colors.paper} strokeWidth={3} />
            ) : null}
          </View>
          <Text style={[styles.checkLabel, acceptedTerms && styles.checkLabelAccepted]}>
            {t('onboarding.acceptTerms')}
          </Text>
        </View>
      </Pressable>

      {/* ── CANCELLATION POLICY ───────────────────────────────── */}
      <View style={styles.hairline} />

      <View style={styles.sectionLabel}>
        <RotateCcw size={12} color={theme.colors.ink.muted} strokeWidth={2} />
        <Text style={styles.sectionLabelText}>{t('onboarding.cancellationPolicy')}</Text>
      </View>
      <Text style={styles.sectionHint}>
        {t('onboarding.cancellationHint')}
      </Text>

      <View style={styles.radioGroup}>
        {CANCELLATION_POLICY_OPTIONS.map((option, idx) => {
          const selected = cancellationPolicy === option.value;
          const isLast = idx === CANCELLATION_POLICY_OPTIONS.length - 1;
          const optionLabel = policyLabelKey[option.value]
            ? t(policyLabelKey[option.value]!)
            : option.label;

          return (
            <Pressable
              key={option.value}
              onPress={() => setCancellationPolicy(option.value)}
              style={({ pressed }) => ({
                opacity: pressed && !selected ? 0.85 : 1,
              })}
              accessibilityRole="radio"
              accessibilityState={{ checked: selected }}
              accessibilityLabel={optionLabel}
            >
              <View
                style={[
                  styles.radioRow,
                  !isLast && styles.radioRowBorder,
                  selected && styles.radioRowSelected,
                ]}
              >
                <View style={[styles.radio, selected && styles.radioSelected]}>
                  {selected ? <View style={styles.radioDot} /> : null}
                </View>
                <Text style={[styles.radioLabel, selected && styles.radioLabelSelected]}>
                  {optionLabel}
                </Text>
              </View>
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
    borderWidth: 1,
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
    borderBottomWidth: 1,
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
    borderTopWidth: 1,
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
    color: theme.colors.ink.DEFAULT,
    textDecorationLine: 'underline',
    textDecorationColor: theme.colors.ink.DEFAULT,
  },

  // Checkbox acceptance row
  checkRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingVertical: theme.spacing[4],
    paddingHorizontal: theme.spacing[4],
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
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
    borderWidth: 1,
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
    borderBottomWidth: 1,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  radioRowSelected: {
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
