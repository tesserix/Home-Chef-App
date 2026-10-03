import { jest, it, expect, afterEach } from '@jest/globals';
import React from 'react';
import { act, create } from 'react-test-renderer';
jest.mock('react-native', () => ({ Platform: { OS: 'ios' }, Modal: 'Modal', ActivityIndicator: 'ActivityIndicator', Image: 'Image', TextInput: 'TextInput', Pressable: 'Pressable', ScrollView: 'ScrollView', Text: 'Text', View: 'View', StyleSheet: { create: (x: unknown) => x } }));
jest.mock('react-native-safe-area-context', () => ({ SafeAreaView: 'SafeAreaView' }));
jest.mock('../components/vendor/FssaiOfferCard', () => ({ FssaiOfferCard: 'FssaiOfferCard' }));
jest.mock('lucide-react-native', () => new Proxy({}, { get: (_, name) => String(name) }));
jest.mock('@homechef/mobile-shared/ui', () => ({ OnboardingScaffold: 'OnboardingScaffold', Input: 'Input', Skeleton: 'Skeleton', EmptyState: 'EmptyState', useAlert: () => ({ showAlert: jest.fn() }), useToast: () => ({ show: jest.fn() }) }));
jest.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
jest.mock('@tanstack/react-query', () => ({ useMutation: () => ({ isPending: false }), useQuery: jest.fn(), useQueryClient: () => ({ invalidateQueries: jest.fn() }) }));
jest.mock('expo-image-picker', () => ({}));
jest.mock('expo-document-picker', () => ({}));
jest.mock('expo-router', () => ({ router: { push: jest.fn(), back: jest.fn() } }));
jest.mock('./api', () => ({ api: { post: jest.fn() } }));
jest.mock('./ocr', () => ({}));
jest.mock('./use-cancel-onboarding', () => ({ useCancelOnboarding: () => jest.fn() }));
jest.mock('../store/onboarding-store', () => ({ useVendorOnboardingStore: jest.fn() }));
import { useVendorOnboardingStore } from '../store/onboarding-store';
import DocumentsScreen from '../app/(onboarding)/documents';
import PayoutStep from '../app/(onboarding)/payout';
import { router } from 'expo-router';
(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
let screen: ReturnType<typeof create>;
afterEach(async () => { if (screen) await act(async () => screen.unmount()); jest.clearAllMocks(); });
function draft(country: string) {
  jest.mocked(useVendorOnboardingStore).mockReturnValue({ kitchenDetails: { country }, documents: { kitchenMedia: [], fssaiLicenseNumber: '', fssaiExpiryDate: '', gstin: '' }, updateDocuments: jest.fn(), updatePayout: jest.fn(), setStep: jest.fn() } as never);
}
it.each(['AU', 'NZ'])('shows local registration without Indian tax fields for %s', async (country) => {
  draft(country);
  await act(async () => { screen = create(<DocumentsScreen />); });
  const output = JSON.stringify(screen.toJSON());
  expect(output).toContain('onboarding.foodRegCert');
  expect(output).not.toContain('onboarding.licenseNumber');
  expect(output).not.toContain('onboarding.gstinOptional');
  expect(screen.root.findAllByProps({ accessibilityLabel: 'Food safety registration guide' })).toHaveLength(1);
});
it('preserves Indian registration fields', async () => {
  draft('IN');
  await act(async () => { screen = create(<DocumentsScreen />); });
  expect(JSON.stringify(screen.toJSON())).toContain('onboarding.licenseNumber');
  expect(screen.root.findAllByProps({ accessibilityLabel: 'Food safety registration guide' })).toHaveLength(0);
});
it.each(['AU', 'NZ'])('defers Stripe connection until application submission for %s', async (country) => {
  draft(country);
  await act(async () => { screen = create(<PayoutStep />); });
  expect(JSON.stringify(screen.toJSON())).toContain('Stripe');
  expect(screen.root.findAllByType('Input' as never)).toHaveLength(0);
  await act(async () => { screen.root.findByType('OnboardingScaffold' as never).props.onPrimary(); });
  expect(router.push).toHaveBeenCalledWith('/(onboarding)/review');
});

import DocumentsRenewScreen from '../app/documents/renew';
import { useQuery } from '@tanstack/react-query';
const renewQueries = (market: { data?: unknown; isError?: boolean }) => (options: any) =>
  (options.queryKey.includes('market')
    ? { data: market.data, isLoading: false, isError: !!market.isError, refetch: jest.fn() }
    : { data: [], isLoading: false, isError: false, refetch: jest.fn() }) as never;
it.each(['AU', 'NZ'])('asks a %s kitchen to renew its council food registration', async (country) => {
  jest.mocked(useQuery).mockImplementation(renewQueries({ data: { country } }));
  await act(async () => { screen = create(<DocumentsRenewScreen />); });
  expect(screen.root.findAllByProps({ accessibilityLabel: 'Upload Food business registration' })).toHaveLength(1);
  expect(screen.root.findAllByProps({ accessibilityLabel: 'Upload FSSAI license' })).toHaveLength(0);
});
it('shows a retry when the kitchen market cannot be loaded', async () => {
  jest.mocked(useQuery).mockImplementation(renewQueries({ isError: true }));
  await act(async () => { screen = create(<DocumentsRenewScreen />); });
  expect(screen.root.findAllByType('EmptyState' as never)).toHaveLength(1);
});
