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

import { useState } from 'react';
import { Alert, StyleSheet, Text, View } from 'react-native';
import { router } from 'expo-router';
import { useMutation } from '@tanstack/react-query';
import { Landmark, ShieldCheck } from 'lucide-react-native';
import { Input, OnboardingScaffold } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { getServerErrorMessage } from '@homechef/mobile-shared/api';
import { api } from '../../lib/api';
import { useVendorOnboardingStore } from '../../store/onboarding-store';
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
  const { updatePayout, setStep } = useVendorOnboardingStore();

  const [values, setValues] = useState<PayoutFormValues>(emptyPayoutForm);
  const [errors, setErrors] = useState<PayoutValidationError[]>([]);

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
    const found = validatePayoutInput(values);
    if (found.length > 0) {
      setErrors(found);
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
        Alert.alert(
          'Could not save payout details',
          getServerErrorMessage(err, 'Please check your details and try again.'),
        ),
    });
  }

  return (
    <OnboardingScaffold
      step={6}
      total={7}
      stepName="Payouts"
      title="Where should we send your earnings?"
      subtitle="You'll be paid here after each order is delivered and confirmed. You can change this any time from Settings."
      primaryLabel="Save and continue"
      onPrimary={onNext}
      primaryLoading={save.isPending}
      onBack={() => router.back()}
    >
      <View style={styles.methods}>
        <View style={[styles.method, styles.methodActive]}>
          <Landmark size={20} color={theme.colors.ink.DEFAULT} />
          <View style={styles.methodText}>
            <Text style={[styles.methodLabel, styles.methodLabelActive]}>Bank account</Text>
            <Text style={styles.methodHint}>Settled directly to your account by NEFT/IMPS</Text>
          </View>
        </View>
      </View>

      <Input
        label="Account holder name"
        value={values.bankAccountName}
        onChangeText={(t) => set('bankAccountName', t)}
        placeholder="As printed on your passbook"
        autoCapitalize="words"
        error={errorFor('bankAccountName')}
      />
      <Input
        label="Account number"
        value={values.bankAccountNumber}
        onChangeText={(t) => set('bankAccountNumber', t)}
        placeholder="e.g. 123456789012"
        keyboardType="number-pad"
        error={errorFor('bankAccountNumber')}
      />
      <Input
        label="IFSC code"
        value={values.bankIFSC}
        onChangeText={(t) => set('bankIFSC', t)}
        placeholder="e.g. HDFC0001234"
        autoCapitalize="characters"
        error={errorFor('bankIFSC')}
      />

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
