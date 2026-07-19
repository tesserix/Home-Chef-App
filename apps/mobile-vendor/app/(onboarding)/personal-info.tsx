import { Alert, ScrollView, StyleSheet, Text, View } from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import { User, Phone, Mail } from 'lucide-react-native';
import { Input, OnboardingScaffold } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { useAuthStore } from '../../store/auth-store';
import { useVendorOnboardingStore } from '../../store/onboarding-store';

const schema = z.object({
  fullName: z.string().min(2, 'onboarding.errFullNameMin'),
  phone: z
    .string()
    .regex(/^[6-9]\d{9}$/, 'onboarding.errPhone'),
  email: z.string().email('onboarding.errEmail'),
});

type FormValues = z.infer<typeof schema>;

export default function PersonalInfoScreen() {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const { personalInfo, updatePersonalInfo, setStep } = useVendorOnboardingStore();

  const {
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    // Seed from the persisted draft first, then fall back to the details the
    // vendor already gave at sign-up (name / phone / email captured on the user),
    // so this step arrives pre-filled instead of blank — they shouldn't have to
    // retype what they just entered. RHF captures defaultValues once at mount,
    // which is what we want here — each Edit tap mounts a fresh instance.
    defaultValues: {
      fullName:
        personalInfo.fullName ||
        [user?.firstName, user?.lastName].filter(Boolean).join(' '),
      phone: personalInfo.phone || (user?.phone ?? ''),
      email: personalInfo.email || (user?.email ?? ''),
    },
  });

  function onSubmit(data: FormValues): void {
    updatePersonalInfo(data);
    setStep(2);
    router.push('/(onboarding)/kitchen-details');
  }

  function onInvalid(errs: typeof errors): void {
    const firstError = Object.values(errs)[0];
    if (firstError?.message) Alert.alert(t('onboarding.checkDetails'), t(firstError.message));
  }

  return (
    <OnboardingScaffold
      step={1}
      total={6}
      stepName={t('onboarding.stepPersonal')}
      title={t('onboarding.personalTitle')}
      subtitle={t('onboarding.personalSubtitle')}
      primaryLabel={t('onboarding.continue')}
      onPrimary={handleSubmit(onSubmit, onInvalid)}
    >
      {/* ── WHO YOU ARE ─────────────────────────────────────── */}
      <View style={styles.sectionLabel}>
        <User size={12} color={theme.colors.ink.muted} strokeWidth={2} />
        <Text style={styles.sectionLabelText}>{t('onboarding.identity')}</Text>
      </View>

      <View style={styles.fieldCard}>
        <Controller
          control={control}
          name="fullName"
          render={({ field: { onChange, onBlur, value } }) => (
            <Input
              label={t('onboarding.fullName')}
              placeholder={t('onboarding.fullNamePlaceholder')}
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              autoCapitalize="words"
              error={errors.fullName?.message ? t(errors.fullName.message) : undefined}
            />
          )}
        />
      </View>

      {/* ── HOW WE REACH YOU ────────────────────────────────── */}
      <View style={styles.hairline} />

      <View style={styles.sectionLabel}>
        <Phone size={12} color={theme.colors.ink.muted} strokeWidth={2} />
        <Text style={styles.sectionLabelText}>{t('onboarding.contact')}</Text>
      </View>

      <View style={styles.fieldCard}>
        <Controller
          control={control}
          name="phone"
          render={({ field: { onChange, onBlur, value } }) => (
            <Input
              label={t('onboarding.phoneNumber')}
              placeholder={t('onboarding.phonePlaceholder')}
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              keyboardType="phone-pad"
              maxLength={10}
              helper={t('onboarding.phoneHelper')}
              error={errors.phone?.message ? t(errors.phone.message) : undefined}
            />
          )}
        />

        {/* Hairline between phone and email inside the contact group */}
        <View style={styles.innerHairline} />

        <Controller
          control={control}
          name="email"
          render={({ field: { value } }) => (
            <View style={styles.lockedWrap}>
              <View style={styles.lockedLabelRow}>
                <Mail size={12} color={theme.colors.ink.muted} strokeWidth={2} />
                <Text style={styles.lockedLabel}>{t('onboarding.email')}</Text>
              </View>
              <View style={styles.lockedField}>
                <Text style={styles.lockedValue} numberOfLines={1}>{value}</Text>
                <View style={styles.lockedBadge}>
                  <Text style={styles.lockedBadgeText}>{t('onboarding.locked')}</Text>
                </View>
              </View>
              <Text style={styles.lockedHint}>{t('onboarding.emailLockedHint')}</Text>
            </View>
          )}
        />
      </View>

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
    marginTop: theme.spacing[1],
  },
  sectionLabelText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    letterSpacing: 1.2,
    color: theme.colors.ink.muted,
  },

  // Hairline separator between sections
  hairline: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginVertical: theme.spacing[4],
  },

  // Inner hairline — between fields inside the same group
  innerHairline: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginVertical: theme.spacing[3],
  },

  // Card: hairline border, no shadow, no elevation
  fieldCard: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    padding: theme.spacing[4],
    backgroundColor: theme.colors.paper,
    gap: theme.spacing[0],
  },

  // Locked email presentation
  lockedWrap: {
    gap: theme.spacing[1],
  },
  lockedLabelRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
    marginBottom: theme.spacing[1],
  },
  lockedLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    letterSpacing: 1.0,
    color: theme.colors.ink.muted,
  },
  lockedField: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.sm,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
    minHeight: theme.touchTarget.vendor,
    gap: theme.spacing[2],
  },
  lockedValue: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.muted,
  },
  lockedBadge: {
    paddingHorizontal: theme.spacing[2],
    paddingVertical: 2,
    borderRadius: theme.radius.full,
    backgroundColor: theme.colors.mist.DEFAULT,
  },
  lockedBadgeText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    letterSpacing: 0.5,
    color: theme.colors.ink.soft,
  },
  lockedHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[1],
  },

  bottomSpacer: {
    height: theme.spacing[4],
  },
});
