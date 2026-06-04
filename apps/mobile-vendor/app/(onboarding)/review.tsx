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
import { Pencil, CheckCircle } from 'lucide-react-native';
import { OnboardingScaffold } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { api } from '../../lib/api';
import { useVendorOnboardingStore } from '../../store/onboarding-store';

const POLICY_LABELS: Record<string, string> = {
  no_cancellations: 'No cancellations after order accepted',
  up_to_1_hour: 'Up to 1 hour before prep start',
  up_to_30_mins: 'Up to 30 mins before prep start',
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
  return (
    <View style={styles.section}>
      <View style={styles.sectionHeaderRow}>
        <Text style={styles.sectionTitle}>{title}</Text>
        <Pressable
          onPress={() => router.push(editRoute as Parameters<typeof router.push>[0])}
          hitSlop={8}
          style={({ pressed }) => [styles.editBtn, pressed && { opacity: 0.6 }]}
          accessibilityRole="button"
          accessibilityLabel={`Edit ${title}`}
        >
          <Pencil size={12} color={theme.colors.ink.soft} strokeWidth={2} />
          <Text style={styles.editBtnLabel}>Edit</Text>
        </Pressable>
      </View>
      <View style={styles.sectionBody}>{children}</View>
    </View>
  );
}

export default function ReviewScreen() {
  const store = useVendorOnboardingStore();
  const [submitting, setSubmitting] = useState(false);

  const { personalInfo, kitchenDetails, operations, documents, policies } = store;

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
        operatingHours: operations.operatingHours,
        acceptedTerms: policies.acceptedTerms,
        cancellationPolicy: policies.cancellationPolicy,
      });
      store.reset();
      router.replace('/(onboarding)/pending');
    } catch (error: unknown) {
      const serverError =
        (error as { response?: { data?: { error?: string } } } | null)?.response?.data?.error;
      const fallback =
        error instanceof Error ? error.message : 'Submission failed. Please try again.';
      Alert.alert('Submission error', serverError ?? fallback);
    } finally {
      setSubmitting(false);
    }
  }

  // Document status pills
  const idStatus = documents.idProofUri ? 'Uploaded' : 'Missing';
  const fssaiStatus = documents.fssaiUri ? 'Uploaded' : 'Missing';
  const docsComplete = Boolean(documents.idProofUri && documents.fssaiUri);

  return (
    <OnboardingScaffold
      step={6}
      total={6}
      title="Review & submit"
      subtitle="Confirm your details before sending the application."
      primaryLabel={submitting ? '' : 'Submit application'}
      onPrimary={onSubmit}
      primaryLoading={submitting}
      primaryDisabled={submitting}
    >
      {/* Readiness indicator */}
      <View style={[styles.readinessRow, docsComplete && styles.readinessComplete]}>
        <CheckCircle
          size={16}
          color={docsComplete ? theme.colors.herb.DEFAULT : theme.colors.ink.muted}
          strokeWidth={2}
        />
        <Text style={[styles.readinessText, docsComplete && styles.readinessTextComplete]}>
          {docsComplete
            ? 'All required documents uploaded. Ready to submit.'
            : 'Missing required documents — go back to Documents to upload.'}
        </Text>
      </View>

      {/* ── PERSONAL ─────────────────────────────────────────── */}
      <Section title="PERSONAL" editRoute="/(onboarding)/personal-info">
        <RowItem label="Full name" value={personalInfo.fullName || '—'} />
        <RowItem label="Phone" value={personalInfo.phone || '—'} />
        <RowItem label="Email" value={personalInfo.email || '—'} isLast />
      </Section>

      {/* ── KITCHEN ──────────────────────────────────────────── */}
      <Section title="KITCHEN" editRoute="/(onboarding)/kitchen-details">
        <RowItem label="Business name" value={kitchenDetails.businessName || '—'} />
        <RowItem
          label="Cuisines"
          value={kitchenDetails.cuisines.length > 0 ? kitchenDetails.cuisines.join(', ') : '—'}
        />
        <RowItem label="Description" value={kitchenDetails.description || '—'} />
        <RowItem label="Address" value={fullAddress || '—'} isLast />
      </Section>

      {/* ── OPERATIONS ───────────────────────────────────────── */}
      <Section title="OPERATIONS" editRoute="/(onboarding)/operations">
        <RowItem label="Open days" value={openDaysShort.length > 0 ? openDaysShort : '—'} />
        <RowItem label="Prep time" value={operations.prepTime || '—'} />
        <RowItem
          label="Radius"
          value={operations.serviceRadius > 0 ? `${operations.serviceRadius} km` : '—'}
          isLast
        />
      </Section>

      {/* ── DOCUMENTS ────────────────────────────────────────── */}
      <Section title="DOCUMENTS" editRoute="/(onboarding)/documents">
        <RowItem label="ID proof" value={idStatus} />
        <RowItem label="FSSAI license" value={fssaiStatus} isLast />
      </Section>

      {/* ── POLICIES ─────────────────────────────────────────── */}
      <Section title="POLICIES" editRoute="/(onboarding)/policies">
        <RowItem label="Terms" value={policies.acceptedTerms ? 'Accepted' : 'Not accepted'} />
        <RowItem
          label="Cancellation"
          value={
            policies.cancellationPolicy
              ? (POLICY_LABELS[policies.cancellationPolicy] ?? policies.cancellationPolicy)
              : '—'
          }
          isLast
        />
      </Section>

      {/* Submitting indicator */}
      {submitting ? (
        <View style={styles.submittingRow}>
          <ActivityIndicator size="small" color={theme.colors.ink.DEFAULT} />
          <Text style={styles.submittingLabel}>Submitting your application…</Text>
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
    borderColor: theme.colors.herb.DEFAULT,
    backgroundColor: 'rgba(194, 65, 12, 0.06)',
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
