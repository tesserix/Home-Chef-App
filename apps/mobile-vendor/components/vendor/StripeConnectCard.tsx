import { useState } from 'react';
import { ActivityIndicator, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import * as WebBrowser from 'expo-web-browser';
import { CheckCircle2, ChevronDown, ChevronUp, Globe, TriangleAlert } from 'lucide-react-native';
import { getServerErrorMessage } from '@homechef/mobile-shared/api';
import { theme } from '@homechef/mobile-shared/theme';
import { useAlert } from '@homechef/mobile-shared/ui';
import {
  STRIPE_COUNTRIES,
  useCreateStripeAccount,
  useRefreshStripeOnboardingLink,
  useSetPaymentProvider,
  useStripeConnectStatus,
} from '../../hooks/useStripeConnect';

// International payouts via Stripe Connect — the mobile counterpart of the
// vendor portal's StripeConnectCard. Collapsed by default: most chefs are paid
// into an Indian bank account through Cashfree and never need this, so it sits
// under a disclosure row rather than competing with the bank-details form.
//
// KYC happens on Stripe's hosted pages. We open them in the system browser
// (not a WebView) because Stripe blocks embedded webviews for identity
// verification, and poll the status afterwards since the redirect lands on the
// web portal, never back in the app.

export function StripeConnectCard() {
  const { showAlert } = useAlert();
  const [open, setOpen] = useState(false);
  const [country, setCountry] = useState('US');

  const { data, isLoading, refetch } = useStripeConnectStatus(open);
  const createAccount = useCreateStripeAccount();
  const refreshLink = useRefreshStripeOnboardingLink();
  const setProvider = useSetPaymentProvider();

  const connected = data?.connected ?? false;
  const ready = Boolean(data?.connected && data.chargesEnabled && data.payoutsEnabled);
  const activeProvider = data?.paymentProvider ?? 'cashfree';
  const busy = createAccount.isPending || refreshLink.isPending;

  async function openOnboarding(url: string): Promise<void> {
    await WebBrowser.openBrowserAsync(url);
    // The chef is back from Stripe — re-read capabilities rather than waiting
    // for the poll interval, so a completed KYC reflects immediately.
    void refetch();
  }

  function onStart(): void {
    createAccount.mutate(country, {
      onSuccess: (link) => void openOnboarding(link.onboardingUrl),
      onError: (err) =>
        showAlert(
          'Could not start Stripe onboarding',
          getServerErrorMessage(err, 'Please try again in a moment.'),
        ),
    });
  }

  function onResume(): void {
    refreshLink.mutate(undefined, {
      onSuccess: (link) => void openOnboarding(link.onboardingUrl),
      onError: (err) =>
        showAlert(
          'Could not open Stripe',
          getServerErrorMessage(err, 'Please try again in a moment.'),
        ),
    });
  }

  function onSwitchProvider(): void {
    const next = activeProvider === 'stripe' ? 'cashfree' : 'stripe';
    showAlert(
      next === 'stripe' ? 'Get paid through Stripe?' : 'Get paid through Cashfree?',
      next === 'stripe'
        ? 'Future payouts will settle to your Stripe account instead of your Indian bank account.'
        : 'Future payouts will settle to your Indian bank account instead of Stripe.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Switch',
          onPress: () =>
            setProvider.mutate(next, {
              onError: (err) =>
                showAlert('Could not switch', getServerErrorMessage(err, 'Please try again.')),
            }),
        },
      ],
    );
  }

  return (
    <View style={styles.card}>
      <Pressable
        onPress={() => setOpen((v) => !v)}
        accessibilityRole="button"
        accessibilityState={{ expanded: open }}
        accessibilityLabel={open ? 'Hide international payouts' : 'Show international payouts'}
      >
        {({ pressed }) => (
          <View style={[styles.headerRow, pressed && { opacity: 0.7 }]}>
            <View style={styles.headerLeft}>
              <Globe size={18} color={theme.colors.ink.soft} />
              <View style={{ flex: 1 }}>
                <Text style={styles.title}>International payouts</Text>
                <Text style={styles.caption}>
                  Paid outside India? Get settled in your local currency via Stripe.
                </Text>
              </View>
            </View>
            {open ? (
              <ChevronUp size={18} color={theme.colors.ink.muted} />
            ) : (
              <ChevronDown size={18} color={theme.colors.ink.muted} />
            )}
          </View>
        )}
      </Pressable>

      {open ? (
        isLoading ? (
          <View style={styles.loading}>
            <ActivityIndicator size="small" color={theme.colors.ink.muted} />
          </View>
        ) : (
          <View style={styles.body}>
            {data?.warning ? (
              <View style={styles.warning}>
                <TriangleAlert size={14} color={theme.colors.ink.soft} />
                <Text style={styles.warningText}>{data.warning}</Text>
              </View>
            ) : null}

            {!connected ? (
              <>
                <Text style={styles.para}>
                  Stripe handles identity and bank verification on their own pages. Pick the country
                  your bank account is in, then follow the steps.
                </Text>
                <Text style={styles.fieldLabel}>Country</Text>
                <ScrollView
                  horizontal
                  showsHorizontalScrollIndicator={false}
                  contentContainerStyle={styles.countryRow}
                >
                  {STRIPE_COUNTRIES.map((c) => (
                    <Pressable
                      key={c.code}
                      onPress={() => setCountry(c.code)}
                      accessibilityRole="radio"
                      accessibilityState={{ selected: country === c.code }}
                      accessibilityLabel={c.name}
                    >
                      {({ pressed }) => (
                        <View
                          style={[
                            styles.chip,
                            country === c.code && styles.chipSelected,
                            pressed && { opacity: 0.8 },
                          ]}
                        >
                          <Text
                            style={[
                              styles.chipText,
                              country === c.code && styles.chipTextSelected,
                            ]}
                          >
                            {c.name}
                          </Text>
                        </View>
                      )}
                    </Pressable>
                  ))}
                </ScrollView>
                <PrimaryButton
                  label={busy ? 'Opening Stripe…' : 'Set up Stripe payouts'}
                  disabled={busy}
                  loading={busy}
                  onPress={onStart}
                />
              </>
            ) : (
              <>
                <View style={ready ? styles.statusReady : styles.statusPending}>
                  {ready ? (
                    <CheckCircle2 size={14} color={theme.colors.success.soft} />
                  ) : (
                    <TriangleAlert size={14} color={theme.colors.ink.soft} />
                  )}
                  <Text style={ready ? styles.statusReadyText : styles.statusPendingText}>
                    {ready ? 'Connected · ready for payouts' : 'Stripe needs more information'}
                  </Text>
                </View>

                <Text style={styles.para}>
                  {ready
                    ? `Your Stripe account${data?.country ? ` (${data.country})` : ''} is verified.`
                    : 'Stripe has not finished verifying you yet. Reopen their pages to complete the remaining steps.'}
                </Text>

                {!ready ? (
                  <PrimaryButton
                    label={busy ? 'Opening Stripe…' : 'Continue on Stripe'}
                    disabled={busy}
                    loading={busy}
                    onPress={onResume}
                  />
                ) : null}

                {/* Provider switch — only offered once Stripe can actually pay
                    out, so a chef can't strand their earnings on an account
                    Stripe has not cleared. */}
                {ready ? (
                  <>
                    <View style={styles.providerRow}>
                      <Text style={styles.fieldLabel}>Payouts settle via</Text>
                      <Text style={styles.providerValue}>
                        {activeProvider === 'stripe' ? 'Stripe' : 'Cashfree'}
                      </Text>
                    </View>
                    <SecondaryButton
                      label={
                        activeProvider === 'stripe'
                          ? 'Switch back to Cashfree'
                          : 'Use Stripe for payouts'
                      }
                      disabled={setProvider.isPending}
                      loading={setProvider.isPending}
                      onPress={onSwitchProvider}
                    />
                  </>
                ) : null}
              </>
            )}
          </View>
        )
      ) : null}
    </View>
  );
}

function PrimaryButton({
  label,
  onPress,
  disabled,
  loading,
}: {
  label: string;
  onPress: () => void;
  disabled?: boolean;
  loading?: boolean;
}) {
  return (
    <Pressable onPress={onPress} disabled={disabled} accessibilityRole="button" accessibilityLabel={label}>
      {({ pressed }) => (
        <View style={[styles.primaryBtn, disabled && styles.btnDisabled, pressed && { opacity: 0.85 }]}>
          {loading ? <ActivityIndicator size="small" color={theme.colors.paper} /> : null}
          <Text style={styles.primaryBtnText}>{label}</Text>
        </View>
      )}
    </Pressable>
  );
}

function SecondaryButton({
  label,
  onPress,
  disabled,
  loading,
}: {
  label: string;
  onPress: () => void;
  disabled?: boolean;
  loading?: boolean;
}) {
  return (
    <Pressable onPress={onPress} disabled={disabled} accessibilityRole="button" accessibilityLabel={label}>
      {({ pressed }) => (
        <View style={[styles.secondaryBtn, pressed && { opacity: 0.85 }]}>
          {loading ? <ActivityIndicator size="small" color={theme.colors.ink.DEFAULT} /> : null}
          <Text style={styles.secondaryBtnText}>{label}</Text>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    padding: theme.spacing[4],
    gap: theme.spacing[2],
    ...theme.shadow[1],
  },
  headerRow: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[2], minHeight: 44 },
  headerLeft: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[2], flex: 1 },
  title: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: theme.colors.ink.DEFAULT },
  caption: { fontFamily: 'Inter', fontSize: 13, lineHeight: 18, color: theme.colors.ink.muted, marginTop: 2 },
  loading: { paddingVertical: theme.spacing[4], alignItems: 'center' },
  body: { gap: theme.spacing[3], paddingTop: theme.spacing[2] },
  para: { fontFamily: 'Inter', fontSize: 13, lineHeight: 19, color: theme.colors.ink.soft },
  fieldLabel: { fontFamily: 'Inter-SemiBold', fontSize: 13, color: theme.colors.ink.DEFAULT },
  countryRow: { flexDirection: 'row', gap: theme.spacing[2], paddingVertical: 2 },
  chip: {
    minHeight: 36,
    justifyContent: 'center',
    borderRadius: theme.radius.full,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    paddingHorizontal: theme.spacing[3],
  },
  chipSelected: { backgroundColor: theme.colors.ink.DEFAULT, borderColor: theme.colors.ink.DEFAULT },
  chipText: { fontFamily: 'Inter-Medium', fontSize: 13, color: theme.colors.ink.DEFAULT },
  chipTextSelected: { color: theme.colors.paper },
  warning: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 6,
    backgroundColor: theme.colors.amber.tint,
    borderRadius: theme.radius.DEFAULT,
    padding: theme.spacing[3],
  },
  warningText: { flex: 1, fontFamily: 'Inter', fontSize: 12, lineHeight: 17, color: theme.colors.ink.DEFAULT },
  statusReady: {
    flexDirection: 'row',
    alignItems: 'center',
    alignSelf: 'flex-start',
    gap: 6,
    backgroundColor: theme.colors.success.tint,
    borderRadius: theme.radius.full,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: 6,
  },
  statusReadyText: { fontFamily: 'Inter-SemiBold', fontSize: 12, color: theme.colors.success.soft },
  statusPending: {
    flexDirection: 'row',
    alignItems: 'center',
    alignSelf: 'flex-start',
    gap: 6,
    backgroundColor: theme.colors.amber.tint,
    borderRadius: theme.radius.full,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: 6,
  },
  statusPendingText: { fontFamily: 'Inter-SemiBold', fontSize: 12, color: theme.colors.ink.DEFAULT },
  providerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    minHeight: 32,
  },
  providerValue: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: theme.colors.ink.DEFAULT,
  },
  primaryBtn: {
    minHeight: 48,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.ink.DEFAULT,
    paddingHorizontal: theme.spacing[5],
  },
  primaryBtnText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.paper },
  btnDisabled: { backgroundColor: theme.colors.mist.strong },
  secondaryBtn: {
    minHeight: 48,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    paddingHorizontal: theme.spacing[5],
  },
  secondaryBtnText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.ink.DEFAULT },
});
