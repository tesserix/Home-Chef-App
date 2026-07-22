// apps/mobile-vendor/app/(onboarding)/review.tsx
// Step 6/6 — Review submitted info + Submit Application.
// StyleSheet only — no NativeWind className.
// Layout: hairline-divided sections with edit links.

import {
  ActivityIndicator,
  Alert,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { useState } from 'react';
import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';
import { Pencil, CheckCircle } from 'lucide-react-native';
import { OnboardingScaffold } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { api } from '../../lib/api';
import { useVendorOnboardingStore } from '../../store/onboarding-store';

interface CachedOnboardingStatus {
  data: {
    status: string;
    completed: boolean;
    step: number;
    chefId: string | null;
    profile: object | null;
  };
}

// Maps cancellation policy value to its i18n key (under `onboarding`).
const POLICY_LABEL_KEYS: Record<string, string> = {
  no_cancellations: 'onboarding.policyNoCancellations',
  up_to_1_hour: 'onboarding.policyUpTo1Hour',
  up_to_30_mins: 'onboarding.policyUpTo30Mins',
};

const DAY_LABELS: Record<string, string> = {
  monday: 'Mon',
  tuesday: 'Tue',
  wednesday: 'Wed',
  thursday: 'Thu',
  friday: 'Fri',
  saturday: 'Sat',
  sunday: 'Sun',
};

// RowItem — label / value pair within a section
function RowItem({
  label,
  value,
  isLast = false,
}: {
  label: string;
  value: string;
  isLast?: boolean;
}): React.ReactElement {
  return (
    <View style={[styles.row, !isLast && styles.rowDivider]}>
      <Text style={styles.rowLabel}>{label}</Text>
      <Text style={styles.rowValue} numberOfLines={3}>
        {value}
      </Text>
    </View>
  );
}

// Section — caps label header + edit link + hairline-divided rows
function Section({
  title,
  editRoute,
  children,
}: {
  title: string;
  editRoute: string;
  children: React.ReactNode;
}): React.ReactElement {
  const { t } = useTranslation();
  return (
    <View style={styles.section}>
      <View style={styles.sectionHeaderRow}>
        <Text style={styles.sectionTitle}>{title}</Text>
        <Pressable
          onPress={() => router.push(editRoute as Parameters<typeof router.push>[0])}
          hitSlop={8}
          style={({ pressed }) => [styles.editBtn, pressed && { opacity: 0.6 }]}
          accessibilityRole="button"
          accessibilityLabel={`${t('onboarding.edit')} ${title}`}
        >
          <Pencil size={12} color={theme.colors.ink.soft} strokeWidth={2} />
          <Text style={styles.editBtnLabel}>{t('onboarding.edit')}</Text>
        </Pressable>
      </View>
      <View style={styles.sectionBody}>{children}</View>
    </View>
  );
}

export default function ReviewScreen() {
  const { t } = useTranslation();
  const store = useVendorOnboardingStore();
  const queryClient = useQueryClient();
  const [submitting, setSubmitting] = useState(false);

  const { personalInfo, kitchenDetails, operations, documents, policies, payout } = store;

  const openDaysShort = Object.entries(operations.operatingHours)
    .filter(([, hours]) => !hours.closed)
    .map(([day]) => DAY_LABELS[day] ?? day)
    .join(', ');

  const fullAddress = [
    kitchenDetails.addressLine1,
    kitchenDetails.addressLine2,
    kitchenDetails.city,
    kitchenDetails.state,
    kitchenDetails.postalCode,
  ]
    .filter(Boolean)
    .join(', ');

  async function onSubmit(): Promise<void> {
    setSubmitting(true);
    try {
      await api.post('/chef/onboarding', {
        fullName: personalInfo.fullName,
        phone: personalInfo.phone,
        email: personalInfo.email,
        businessName: kitchenDetails.businessName,
        description: kitchenDetails.description,
        cuisines: kitchenDetails.cuisines,
        kitchenAddress: {
          line1: kitchenDetails.addressLine1,
          line2: kitchenDetails.addressLine2,
          city: kitchenDetails.city,
          state: kitchenDetails.state,
          postalCode: kitchenDetails.postalCode,
        },
        prepTime: operations.prepTime,
        serviceRadius: operations.serviceRadius,
        offersPickup: operations.offersPickup,
        offersSelfDelivery: operations.offersSelfDelivery,
        operatingHours: operations.operatingHours,
        acceptedTerms: policies.acceptedTerms,
        cancellationPolicy: policies.cancellationPolicy,
        // Regulatory IDs — backend persists fssaiLicenseNumber on
        // chef_profiles so Wave 3 invoicing can print it and FoSCoS
        // validation can query against it. GSTIN is optional; backend
        // only validates length when provided.
        fssaiLicenseNumber: documents.fssaiLicenseNumber,
        gstin: documents.gstin || undefined,
        // Kitchen compliance media — both photo and video URLs go in this one
        // array (the video is just another URL); backend persists them to
        // chef_profiles.kitchen_photos for admin review.
        kitchenPhotos: documents.kitchenMedia.map((m) => m.url),
      });
      // Optimistically flip the cached onboarding/status to pending_review
      // so the global routing effect in _layout.tsx sees the user as
      // post-submit immediately. Without this, the routing effect computes
      // expectedPath from the stale `in_progress` cache, sees a mismatch
      // with /pending, and bounces the chef back to /policies. Invalidate
      // afterwards so the server's truth eventually reconciles.
      queryClient.setQueryData<CachedOnboardingStatus>(
        ['chef', 'onboarding', 'status'],
        (old) =>
          old
            ? {
                ...old,
                data: { ...old.data, status: 'pending_review', completed: true },
              }
            : old,
      );
      queryClient.invalidateQueries({ queryKey: ['chef', 'onboarding', 'status'] });
      store.reset();
      router.replace('/(onboarding)/pending');
    } catch (error: unknown) {
      const serverError =
        (error as { response?: { data?: { error?: string } } } | null)?.response?.data?.error;
      const fallback =
        error instanceof Error ? error.message : t('onboarding.submissionFailed');
      Alert.alert(t('onboarding.submissionError'), serverError ?? fallback);
    } finally {
      setSubmitting(false);
    }
  }

  // Document status pills
  const idStatus = documents.idProofUri ? t('onboarding.uploaded') : t('onboarding.missing');
  const fssaiStatus = documents.fssaiUri ? t('onboarding.uploaded') : t('onboarding.missing');
  const kitchenPhotoCount = documents.kitchenMedia.filter((m) => m.type === 'image').length;
  const kitchenVideoCount = documents.kitchenMedia.filter((m) => m.type === 'video').length;
  const kitchenMediaComplete = kitchenPhotoCount > 0 && kitchenVideoCount > 0;
  const kitchenStatus = kitchenMediaComplete
    ? t('onboarding.kitchenMediaCount', {
        photos: kitchenPhotoCount,
        videos: kitchenVideoCount,
      })
    : t('onboarding.missing');
  const docsComplete = Boolean(
    documents.idProofUri && documents.fssaiUri && kitchenMediaComplete,
  );

  return (
    <OnboardingScaffold
      step={7}
      total={7}
      stepName={t('onboarding.stepReview')}
      title={t('onboarding.reviewTitle')}
      subtitle={t('onboarding.reviewSubtitle')}
      primaryLabel={submitting ? '' : t('onboarding.submitApplication')}
      onPrimary={onSubmit}
      primaryLoading={submitting}
      primaryDisabled={submitting}
    >
      {/* Readiness indicator */}
      <View style={[styles.readinessRow, docsComplete && styles.readinessComplete]}>
        <CheckCircle
          size={16}
          color={docsComplete ? theme.colors.success.DEFAULT : theme.colors.ink.muted}
          strokeWidth={2}
        />
        <Text style={[styles.readinessText, docsComplete && styles.readinessTextComplete]}>
          {docsComplete
            ? t('onboarding.readyToSubmit')
            : t('onboarding.missingDocuments')}
        </Text>
      </View>

      {/* ── PERSONAL ─────────────────────────────────────────── */}
      <Section title={t('onboarding.personal')} editRoute="/(onboarding)/personal-info">
        <RowItem label={t('onboarding.fullName')} value={personalInfo.fullName || '—'} />
        <RowItem label={t('onboarding.phone')} value={personalInfo.phone || '—'} />
        <RowItem label={t('onboarding.email')} value={personalInfo.email || '—'} isLast />
      </Section>

      {/* ── KITCHEN ──────────────────────────────────────────── */}
      <Section title={t('onboarding.kitchen')} editRoute="/(onboarding)/kitchen-details">
        <RowItem label={t('onboarding.businessNameLabel')} value={kitchenDetails.businessName || '—'} />
        <RowItem
          label={t('onboarding.cuisines')}
          value={kitchenDetails.cuisines.length > 0 ? kitchenDetails.cuisines.join(', ') : '—'}
        />
        <RowItem label={t('onboarding.description')} value={kitchenDetails.description || '—'} />
        <RowItem label={t('onboarding.address')} value={fullAddress || '—'} isLast />
      </Section>

      {/* ── OPERATIONS ───────────────────────────────────────── */}
      <Section title={t('onboarding.operations')} editRoute="/(onboarding)/operations">
        <RowItem label={t('onboarding.openDays')} value={openDaysShort.length > 0 ? openDaysShort : '—'} />
        <RowItem label={t('onboarding.prepTimeLabel')} value={operations.prepTime || '—'} />
        <RowItem
          label={t('onboarding.fulfillmentLabel')}
          value={
            [
              operations.offersPickup ? t('onboarding.pickupTitle') : null,
              operations.offersSelfDelivery ? t('onboarding.selfDeliveryTitle') : null,
            ]
              .filter(Boolean)
              .join(' · ') || '—'
          }
        />
        <RowItem
          label={t('onboarding.radius')}
          value={
            operations.offersSelfDelivery && operations.serviceRadius > 0
              ? t('onboarding.radiusKm', { count: operations.serviceRadius })
              : '—'
          }
          isLast
        />
      </Section>

      {/* ── DOCUMENTS ────────────────────────────────────────── */}
      <Section title={t('onboarding.documents')} editRoute="/(onboarding)/documents">
        <RowItem label={t('onboarding.idProof')} value={idStatus} />
        <RowItem label={t('onboarding.fssaiLicense')} value={fssaiStatus} />
        <RowItem label={t('onboarding.kitchenMediaLabel')} value={kitchenStatus} isLast />
      </Section>

      {/* ── POLICIES ─────────────────────────────────────────── */}
      <Section title={t('onboarding.policies')} editRoute="/(onboarding)/policies">
        <RowItem label={t('onboarding.terms')} value={policies.acceptedTerms ? t('onboarding.accepted') : t('onboarding.notAccepted')} />
        <RowItem
          label={t('onboarding.cancellation')}
          value={
            policies.cancellationPolicy
              ? (POLICY_LABEL_KEYS[policies.cancellationPolicy]
                  ? t(POLICY_LABEL_KEYS[policies.cancellationPolicy]!)
                  : policies.cancellationPolicy)
              : '—'
          }
          isLast
        />
      </Section>

      {/* ── PAYOUT ───────────────────────────────────────────── */}
      <Section title="Payouts" editRoute="/(onboarding)/payout">
        <RowItem
          label="Paid to"
          value={payout.configured ? payout.summary : 'Not set up'}
          isLast
        />
      </Section>

      {/* Submitting indicator */}
      {submitting ? (
        <View style={styles.submittingRow}>
          <ActivityIndicator size="small" color={theme.colors.ink.DEFAULT} />
          <Text style={styles.submittingLabel}>{t('onboarding.submitting')}</Text>
        </View>
      ) : null}

      <View style={styles.bottomSpacer} />
    </OnboardingScaffold>
  );
}

const styles = StyleSheet.create({
  // Readiness indicator strip
  readinessRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: theme.spacing[2],
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
    borderRadius: theme.radius.sm,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    backgroundColor: theme.colors.bone,
    marginBottom: theme.spacing[2],
  },
  readinessComplete: {
    borderColor: theme.colors.success.DEFAULT,
    backgroundColor: 'rgba(0, 138, 5, 0.08)',
  },
  readinessText: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.soft,
    lineHeight: theme.typography.size.caption.size * 1.5,
  },
  readinessTextComplete: {
    color: theme.colors.ink.DEFAULT,
    fontFamily: 'Inter-Medium',
  },

  // Section — hairline top border + spacing
  section: {
    marginTop: theme.spacing[4],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
  },

  sectionHeaderRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[1],
  },

  sectionTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    letterSpacing: 1.2,
    color: theme.colors.ink.muted,
  },

  editBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
    paddingVertical: 4,
    paddingHorizontal: 4,
  },
  editBtnLabel: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.soft,
  },

  sectionBody: {
    // Rows carry their own vertical rhythm
  },

  // Label / value row
  row: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    paddingVertical: theme.spacing[3],
    gap: theme.spacing[4],
    minHeight: theme.touchTarget.vendor,
  },
  rowDivider: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  rowLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    flexShrink: 0,
    maxWidth: '38%',
    lineHeight: theme.typography.size.bodySm.size * 1.45,
  },
  rowValue: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    flex: 1,
    textAlign: 'right',
    lineHeight: theme.typography.size.bodySm.size * 1.45,
    fontVariant: ['tabular-nums'],
  },

  submittingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: theme.spacing[2],
    paddingVertical: theme.spacing[4],
  },
  submittingLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
  },

  bottomSpacer: {
    height: theme.spacing[6],
  },
});
