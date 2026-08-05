import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Stripe Connect onboarding for chefs who are paid outside India, plus the
// provider switch (#payouts). Wired to the same four endpoints the vendor
// portal's Settings page uses:
//   GET  /chef/stripe/status          — live capability state
//   POST /chef/stripe/connect         — create the Express account + first link
//   POST /chef/stripe/onboarding-link — re-link an account that needs more KYC
//   PUT  /chef/payment-provider       — choose which provider settles payouts
//
// Without these the mobile app could only collect Indian bank details
// (/chef/payout), so an international chef had no way to get paid from the
// phone at all.

export type PaymentProvider = 'razorpay' | 'stripe';

export interface StripeConnectStatus {
  connected: boolean;
  accountId: string;
  chargesEnabled: boolean;
  payoutsEnabled: boolean;
  detailsSubmitted: boolean;
  country: string;
  paymentProvider?: PaymentProvider;
  /** Set when Stripe was unreachable and the cached flags are being shown. */
  warning?: string;
}

export interface OnboardingLink {
  accountId?: string;
  onboardingUrl: string;
  expiresAt?: number;
  country?: string;
}

/** Countries Stripe Connect Express supports, narrowed to where chefs operate. */
export const STRIPE_COUNTRIES: { code: string; name: string }[] = [
  { code: 'US', name: 'United States' },
  { code: 'GB', name: 'United Kingdom' },
  { code: 'CA', name: 'Canada' },
  { code: 'AU', name: 'Australia' },
  { code: 'NZ', name: 'New Zealand' },
  { code: 'SG', name: 'Singapore' },
  { code: 'HK', name: 'Hong Kong' },
  { code: 'AE', name: 'United Arab Emirates' },
  { code: 'DE', name: 'Germany' },
  { code: 'FR', name: 'France' },
  { code: 'IT', name: 'Italy' },
  { code: 'ES', name: 'Spain' },
  { code: 'NL', name: 'Netherlands' },
  { code: 'IE', name: 'Ireland' },
  { code: 'IN', name: 'India' },
];

export function useStripeConnectStatus(enabled = true) {
  return useQuery<StripeConnectStatus>({
    queryKey: ['chef', 'stripe-status'],
    queryFn: () => api.get<StripeConnectStatus>('/chef/stripe/status').then((r) => r.data),
    enabled,
    // KYC completes on Stripe's hosted pages, outside the app, and nothing
    // calls back into the client — so poll while an account exists but is not
    // yet cleared, and stop once there is nothing left to wait for.
    refetchInterval: (query) => {
      const d = query.state.data;
      if (d?.connected && !(d.chargesEnabled && d.payoutsEnabled)) return 60_000;
      return false;
    },
  });
}

export function useCreateStripeAccount() {
  const queryClient = useQueryClient();
  return useMutation<OnboardingLink, Error, string>({
    mutationFn: (country: string) =>
      api.post<OnboardingLink>('/chef/stripe/connect', { country }).then((r) => r.data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['chef', 'stripe-status'] }),
  });
}

export function useRefreshStripeOnboardingLink() {
  return useMutation<OnboardingLink, Error, void>({
    mutationFn: () =>
      api.post<OnboardingLink>('/chef/stripe/onboarding-link', {}).then((r) => r.data),
  });
}

export function useSetPaymentProvider() {
  const queryClient = useQueryClient();
  return useMutation<{ paymentProvider: string }, Error, PaymentProvider>({
    mutationFn: (provider) =>
      api
        .put<{ paymentProvider: string }>('/chef/payment-provider', { provider })
        .then((r) => r.data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['chef', 'stripe-status'] });
      // The payout screen's own card reads the active provider too.
      void queryClient.invalidateQueries({ queryKey: ['chef', 'payout'] });
    },
  });
}
