// apps/mobile-vendor/app/(onboarding)/payout.tsx
// Step 6/7 — where the chef's money goes (#739).
//
// Payout setup used to be an optional Settings action that onboarding actively
// deferred ("you can add this later"). That let a chef go live, take a
// customer's money, cook, deliver and accrue released holds with no payable
// destination on file — the payout engine would then build a batch for a payee
// it cannot pay. Collecting it here, while the chef is motivated, is both
// better for them and the only point at which we can guarantee it exists.
//
// Sensitive fields go straight to POST /chef/payout, which stores them in GCP
// Secret Manager. Only a masked summary is kept in the onboarding draft.

import { useRef, useState } from 'react';
import { ActivityIndicator, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { router } from 'expo-router';
import { useMutation } from '@tanstack/react-query';
import * as ImagePicker from 'expo-image-picker';
import * as DocumentPicker from 'expo-document-picker';
import { CheckCircle, FileText, Image as ImageIcon, Landmark, ShieldCheck } from 'lucide-react-native';
import { Input, OnboardingScaffold, useAlert, useToast } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { getServerErrorMessage, multipartConfig } from '@homechef/mobile-shared/api';
import { api } from '../../lib/api';
import { useVendorOnboardingStore } from '../../store/onboarding-store';
import { useCancelOnboarding } from '../../lib/use-cancel-onboarding';
import {
  buildPayoutPayload,
  emptyPayoutForm,
  summarisePayout,
  validatePayoutInput,
  type PayoutFormValues,
  type PayoutValidationError,
} from '../../lib/payout';

// UPI is not an accepted payout method (#767): Route settles to a bank account
// only. Bank transfer is the only option.
export default function PayoutStep() {
  const cancelOnboarding = useCancelOnboarding();
  const { showAlert } = useAlert();
  const { updatePayout, setStep } = useVendorOnboardingStore();

  const [values, setValues] = useState<PayoutFormValues>(emptyPayoutForm);
  const [errors, setErrors] = useState<PayoutValidationError[]>([]);
  const { show: showToast } = useToast();
  // Optional last-3-months bank statement — speeds up payout verification.
  const [statementUploaded, setStatementUploaded] = useState(false);
  const [statementUploading, setStatementUploading] = useState(false);

  // Nothing typed yet = the step is skippable. Missing payout details never
  // remove the account — they only hold payouts (payout gate #739) — so the
  // chef may defer this and add it from Settings within 30 days.
  const untouched =
    !values.bankAccountName.trim() && !values.bankAccountNumber.trim() && !values.bankIFSC.trim();

  async function uploadStatement(uri: string, mimeType: string, name: string): Promise<void> {
    setStatementUploading(true);
    try {
      const form = new FormData();
      form.append('type', 'bank_statement');
      form.append('file', { uri, name, type: mimeType } as unknown as Blob);
      await api.post('/chef/documents', form, multipartConfig());
      setStatementUploaded(true);
      showToast({ message: 'Bank statement uploaded', tone: 'success' });
    } catch (err: unknown) {
      showToast({
        message: getServerErrorMessage(err, 'Could not upload the statement'),
        tone: 'error',
      });
    } finally {
      setStatementUploading(false);
    }
  }

  async function pickStatementImage(): Promise<void> {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) return;
    const result = await ImagePicker.launchImageLibraryAsync({ mediaTypes: ['images'], quality: 0.85 });
    if (!result.canceled && result.assets[0]) {
      await uploadStatement(result.assets[0].uri, 'image/jpeg', result.assets[0].uri.split('/').pop() ?? 'statement.jpg');
    }
  }

  async function pickStatementPdf(): Promise<void> {
    const result = await DocumentPicker.getDocumentAsync({ type: ['application/pdf'], copyToCacheDirectory: true });
    if (!result.canceled && result.assets[0]) {
      await uploadStatement(result.assets[0].uri, 'application/pdf', result.assets[0].name ?? 'statement.pdf');
    }
  }

  // R14 — scroll to the first invalid field on a failed submit instead of
  // leaving an already-scrolled-away chef staring at nothing happening.
  // OnboardingScaffold owns the ScrollView, forwarded via `scrollRef`.
  const scrollRef = useRef<ScrollView>(null);
  const bankAccountNameY = useRef(0);
  const bankAccountNumberY = useRef(0);
  const bankIFSCY = useRef(0);

  const save = useMutation({
    mutationFn: () => api.post('/chef/payout', buildPayoutPayload(values)),
  });

  function set(field: keyof PayoutFormValues, value: string): void {
    setValues((prev) => ({ ...prev, [field]: value }));
    // Clear this field's error as soon as the chef starts correcting it,
    // rather than leaving stale red text until the next submit.
    setErrors((prev) => prev.filter((e) => e.field !== field));
  }

  function errorFor(field: keyof PayoutFormValues): string | undefined {
    return errors.find((e) => e.field === field)?.message;
  }

  function onNext(): void {
    if (untouched) {
      // Deferred: payouts stay on hold until details are added from Settings.
      setStep(7);
      router.push('/(onboarding)/review');
      return;
    }
    const found = validatePayoutInput(values);
    if (found.length > 0) {
      setErrors(found);
      const order: { field: keyof PayoutFormValues; y: typeof bankAccountNameY }[] = [
        { field: 'bankAccountName', y: bankAccountNameY },
        { field: 'bankAccountNumber', y: bankAccountNumberY },
        { field: 'bankIFSC', y: bankIFSCY },
      ];
      const first = order.find((f) => found.some((e) => e.field === f.field));
      if (first) {
        scrollRef.current?.scrollTo({ y: Math.max(0, first.y.current - 16), animated: true });
      }
      return;
    }

    save.mutate(undefined, {
      onSuccess: () => {
        // Persist only the masked summary — never the account number.
        updatePayout({
          configured: true,
          method: 'bank_transfer',
          summary: summarisePayout(values),
        });
        setStep(7);
        router.push('/(onboarding)/review');
      },
      onError: (err) =>
        showAlert(
          'Could not save payout details',
          getServerErrorMessage(err, 'Please check your details and try again.'),
        ),
    });
  }

  return (
    <OnboardingScaffold
      onCancel={cancelOnboarding}
      step={6}
      total={7}
      stepName="Payouts"
      title="Where should we send your earnings?"
      subtitle="You'll be paid here after each order is delivered and confirmed. You can change this any time from Settings."
      primaryLabel={untouched ? 'Skip for now' : 'Save and continue'}
      onPrimary={onNext}
      primaryLoading={save.isPending}
      onBack={() => router.back()}
      scrollRef={scrollRef}
    >
      {untouched ? (
        <View style={styles.deferCard}>
          <Text style={styles.deferTitle}>You can add this later — within 30 days</Text>
          <Text style={styles.deferBody}>
            Until your bank details are in, earnings from delivered orders are held and payouts
            are paused. Add them any time from Settings — sooner means no payment delays.
          </Text>
        </View>
      ) : null}

      <View style={styles.methods}>
        <View style={[styles.method, styles.methodActive]}>
          <Landmark size={20} color={theme.colors.ink.DEFAULT} />
          <View style={styles.methodText}>
            <Text style={[styles.methodLabel, styles.methodLabelActive]}>Bank account</Text>
            <Text style={styles.methodHint}>Settled directly to your account by NEFT/IMPS</Text>
          </View>
        </View>
      </View>

      <View onLayout={(e) => { bankAccountNameY.current = e.nativeEvent.layout.y; }}>
        <Input
          label="Account holder name"
          value={values.bankAccountName}
          onChangeText={(t) => set('bankAccountName', t)}
          placeholder="As printed on your passbook"
          autoCapitalize="words"
          error={errorFor('bankAccountName')}
        />
      </View>
      <View onLayout={(e) => { bankAccountNumberY.current = e.nativeEvent.layout.y; }}>
        <Input
          label="Account number"
          value={values.bankAccountNumber}
          onChangeText={(t) => set('bankAccountNumber', t)}
          placeholder="e.g. 123456789012"
          keyboardType="number-pad"
          error={errorFor('bankAccountNumber')}
        />
      </View>
      <View onLayout={(e) => { bankIFSCY.current = e.nativeEvent.layout.y; }}>
        <Input
          label="IFSC code"
          value={values.bankIFSC}
          onChangeText={(t) => set('bankIFSC', t)}
          placeholder="e.g. HDFC0001234"
          autoCapitalize="characters"
          error={errorFor('bankIFSC')}
        />
      </View>

      {/* Optional bank statement — private bucket, admin-only access. */}
      <View style={styles.statementCard}>
        <View style={styles.statementHeader}>
          <Text style={styles.statementTitle}>Bank statement (optional)</Text>
          {statementUploaded ? (
            <CheckCircle size={16} color={theme.colors.success.DEFAULT} strokeWidth={2} />
          ) : null}
        </View>
        <Text style={styles.statementHint}>
          Last 3 months, matching this account — speeds up payout verification. Stored privately;
          only our verification team can open it.
        </Text>
        {statementUploading ? (
          <ActivityIndicator size="small" color={theme.colors.ink.DEFAULT} style={styles.statementSpinner} />
        ) : (
          <View style={styles.statementActions}>
            <Pressable
              onPress={pickStatementImage}
              accessibilityRole="button"
              style={styles.statementBtn}
            >
              <ImageIcon size={15} color={theme.colors.ink.soft} strokeWidth={2} />
              <Text style={styles.statementBtnLabel}>Gallery</Text>
            </Pressable>
            <Pressable
              onPress={pickStatementPdf}
              accessibilityRole="button"
              style={styles.statementBtn}
            >
              <FileText size={15} color={theme.colors.ink.soft} strokeWidth={2} />
              <Text style={styles.statementBtnLabel}>PDF</Text>
            </Pressable>
          </View>
        )}
      </View>

      <View style={styles.assurance}>
        <ShieldCheck size={16} color={theme.colors.ink.muted} />
        <Text style={styles.assuranceText}>
          Your details are stored encrypted and are never shown in full — not even back to you.
        </Text>
      </View>
    </OnboardingScaffold>
  );
}

const styles = StyleSheet.create({
  deferCard: {
    backgroundColor: theme.colors.amber.tint,
    borderLeftWidth: 3,
    borderLeftColor: theme.colors.amber.DEFAULT,
    borderRadius: theme.radius.sm,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    marginBottom: theme.spacing[4],
  },
  deferTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    marginBottom: 2,
  },
  deferBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.label.size,
    lineHeight: theme.typography.size.label.size * 1.45,
    color: theme.colors.ink.soft,
  },

  statementCard: {
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.md,
    padding: theme.spacing[4],
    marginTop: theme.spacing[4],
  },
  statementHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 2,
  },
  statementTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  statementHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.label.size,
    lineHeight: theme.typography.size.label.size * 1.45,
    color: theme.colors.ink.soft,
    marginBottom: theme.spacing[3],
  },
  statementSpinner: { alignSelf: 'flex-start' },
  statementActions: { flexDirection: 'row', gap: theme.spacing[2] },
  statementBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[2],
    minHeight: 40,
  },
  statementBtnLabel: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },

  methods: { gap: theme.spacing[2], marginBottom: theme.spacing[4] },
  method: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingVertical: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    borderRadius: 8,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    minHeight: 44,
  },
  methodActive: {
    borderColor: theme.colors.ink.DEFAULT,
    backgroundColor: theme.colors.bone,
  },
  methodText: { flex: 1 },
  methodLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.muted,
  },
  methodLabelActive: { color: theme.colors.ink.DEFAULT },
  methodHint: {
    fontFamily: 'Inter-Regular',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    marginTop: 2,
  },
  assurance: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: theme.spacing[2],
    marginTop: theme.spacing[4],
  },
  assuranceText: {
    flex: 1,
    fontFamily: 'Inter-Regular',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    lineHeight: 18,
  },
});
