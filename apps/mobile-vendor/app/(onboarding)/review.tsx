// apps/mobile-vendor/app/(onboarding)/review.tsx
// Step 6/6 — Review submitted info + Submit Application.
// StyleSheet only — no NativeWind className.
// Layout: hairline-divided sections (not bordered cards).

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
import { OnboardingScaffold } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { api } from '../../lib/api';
import { useVendorOnboardingStore } from '../../store/onboarding-store';

// Map the raw cancellation policy value to a human-readable label.
const POLICY_LABELS: Record<string, string> = {
  no_cancellations: 'No cancellations after order accepted',
  up_to_1_hour: 'Up to 1 hour before prep start',
  up_to_30_mins: 'Up to 30 mins before prep start',
};

const DAY_LABELS: Record<string, string> = {
  monday: 'Monday',
  tuesday: 'Tuesday',
  wednesday: 'Wednesday',
  thursday: 'Thursday',
  friday: 'Friday',
  saturday: 'Saturday',
  sunday: 'Sunday',
};

// RowItem — a label / value pair within a section.
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

// SectionHeader — uppercase label above a hairline-divided block.
function SectionHeader({ title }: { title: string }): React.ReactElement {
  return (
    <View style={styles.sectionHeader}>
      <Text style={styles.sectionTitle}>{title}</Text>
    </View>
  );
}

// Section — groups a header and rows without a bordered card.
// Rationale: the review screen is a long read, not an interactive surface.
// Hairline-divided sections keep visual hierarchy without adding card chrome
// on top of already-dense content. Bordered cards would create a nested
// box-inside-box feel once the scaffold's outer scroll container is considered.
function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}): React.ReactElement {
  return (
    <View style={styles.section}>
      <SectionHeader title={title} />
      <View style={styles.sectionBody}>{children}</View>
    </View>
  );
}

export default function ReviewScreen() {
  const store = useVendorOnboardingStore();
  const [submitting, setSubmitting] = useState(false);

  const { personalInfo, kitchenDetails, operations, documents, policies } = store;

  // Format open-days string from the hours map.
  const openDaysFull = Object.entries(operations.operatingHours)
    .filter(([, hours]) => !hours.closed)
    .map(([day]) => DAY_LABELS[day] ?? day)
    .join(', ');

  // Build the full address string, filtering empty parts.
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

  return (
    <OnboardingScaffold
      step={6}
      total={6}
      title="Review & submit"
      subtitle="Check everything looks right before sending your application."
      primaryLabel={submitting ? '' : 'Submit application'}
      onPrimary={onSubmit}
      primaryLoading={submitting}
      primaryDisabled={submitting}
    >
      {/* Top editorial note */}
      <View style={styles.noteBlock}>
        <Text style={styles.noteText}>
          Our team reviews applications within 24–48 hours. You'll receive a
          notification once your kitchen is approved.
        </Text>
      </View>

      {/* — Personal — */}
      <Section title="Personal information">
        <RowItem label="Full name" value={personalInfo.fullName || '—'} />
        <RowItem label="Phone" value={personalInfo.phone || '—'} />
        <RowItem label="Email" value={personalInfo.email || '—'} isLast />
      </Section>

      {/* — Kitchen — */}
      <Section title="Kitchen">
        <RowItem label="Business name" value={kitchenDetails.businessName || '—'} />
        <RowItem
          label="Cuisines"
          value={kitchenDetails.cuisines.length > 0 ? kitchenDetails.cuisines.join(', ') : '—'}
        />
        <RowItem
          label="Description"
          value={kitchenDetails.description || '—'}
        />
        <RowItem label="Address" value={fullAddress || '—'} isLast />
      </Section>

      {/* — Operations — */}
      <Section title="Operations">
        <RowItem
          label="Open days"
          value={openDaysFull.length > 0 ? openDaysFull : '—'}
        />
        <RowItem label="Prep time" value={operations.prepTime || '—'} />
        <RowItem
          label="Service radius"
          value={operations.serviceRadius > 0 ? `${operations.serviceRadius} km` : '—'}
          isLast
        />
      </Section>

      {/* — Documents — */}
      <Section title="Documents">
        <RowItem
          label="ID proof"
          value={documents.idProofUri ? 'Uploaded' : 'Not uploaded'}
        />
        <RowItem
          label="FSSAI license"
          value={documents.fssaiUri ? 'Uploaded' : 'Not uploaded'}
          isLast
        />
      </Section>

      {/* — Policies — */}
      <Section title="Policies">
        <RowItem label="Terms accepted" value={policies.acceptedTerms ? 'Yes' : 'No'} />
        <RowItem
          label="Cancellation policy"
          value={
            policies.cancellationPolicy
              ? (POLICY_LABELS[policies.cancellationPolicy] ?? policies.cancellationPolicy)
              : '—'
          }
          isLast
        />
      </Section>

      {/* Loading indicator sits above the scaffold's sticky CTA when submitting. */}
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
  // Soft note block at top of the review.
  noteBlock: {
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.sm,
    paddingVertical: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    borderLeftWidth: 2,
    borderLeftColor: theme.colors.herb.DEFAULT,
  },

  noteText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    lineHeight: theme.typography.size.bodySm.size * 1.55,
    color: theme.colors.ink.soft,
  },

  // Section — no card, just spacing + hairline.
  section: {
    marginTop: theme.spacing[5],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
  },

  sectionHeader: {
    paddingVertical: theme.spacing[3],
  },

  sectionTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    letterSpacing: 0.6,
    color: theme.colors.ink.muted,
    textTransform: 'uppercase',
  },

  sectionBody: {
    // No extra padding — rows carry their own vertical rhythm.
  },

  // Individual label / value row within a section.
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
    height: theme.spacing[4],
  },
});
