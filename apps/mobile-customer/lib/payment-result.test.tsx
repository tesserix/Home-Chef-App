import { jest, beforeEach, afterEach, it, expect } from '@jest/globals';
import React from 'react';
import { act, create } from 'react-test-renderer';

jest.mock('react-native-css-interop/jsx-runtime', () => require('react/jsx-runtime'));
jest.mock('react-native-css-interop/jsx-dev-runtime', () => require('react/jsx-dev-runtime'));
jest.mock('react-native', () => ({
  ActivityIndicator: 'ActivityIndicator',
  Platform: { OS: 'ios' },
  Pressable: 'Pressable',
  Text: 'Text',
  View: 'View',
  StyleSheet: { create: (x: unknown) => x },
}));
jest.mock('react-native-safe-area-context', () => ({
  SafeAreaView: 'SafeAreaView',
}));
jest.mock('lucide-react-native', () => ({
  CheckCircle2: 'CheckCircle2',
  XCircle: 'XCircle',
}));
jest.mock('expo-router', () => ({
  useRouter: () => ({ replace: jest.fn(), setParams: jest.fn() }),
  useLocalSearchParams: jest.fn(),
}));
jest.mock('../hooks/useOrderHistory', () => ({ useOrder: jest.fn() }));
jest.mock('./payment', () => ({
  startOrderPayment: jest.fn(),
  isPaymentSetupRejection: () => false,
}));
import { useLocalSearchParams } from 'expo-router';
import { useOrder } from '../hooks/useOrderHistory';
import { startOrderPayment } from './payment';
import PaymentResult from '../app/payment/result';

(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;
let screen: ReturnType<typeof create>;
beforeEach(() => {
  jest.useFakeTimers();
  jest.clearAllMocks();
  jest
    .mocked(useLocalSearchParams)
    .mockReturnValue({ order_id: 'existing-order', not_started: '1' });
  jest.mocked(useOrder).mockReturnValue({
    data: { data: { paymentStatus: 'pending', status: 'pending' } },
  } as never);
});
afterEach(async () => {
  if (screen) await act(async () => screen.unmount());
  jest.useRealTimers();
});
it('shows a setup rejection immediately and retries the same order', async () => {
  await act(async () => {
    screen = create(<PaymentResult />);
  });
  expect(JSON.stringify(screen.toJSON())).toContain('Payment not started');
  const retry = screen.root.findByProps({
    accessibilityLabel: 'Retry payment',
  });
  jest.mocked(startOrderPayment).mockRejectedValueOnce(new Error('timeout'));
  await act(async () => {
    await retry.props.onPress();
  });
  expect(startOrderPayment).toHaveBeenCalledWith('existing-order');
  expect(JSON.stringify(screen.toJSON())).not.toContain('Payment not started');
  expect(screen.root.findAllByProps({ accessibilityLabel: 'Retry payment' })).toHaveLength(0);
});
it('keeps unknown payment outcomes in confirmation without retry', async () => {
  jest.mocked(useLocalSearchParams).mockReturnValue({ order_id: 'existing-order' });
  await act(async () => {
    screen = create(<PaymentResult />);
  });
  await act(async () => {
    jest.advanceTimersByTime(30000);
  });
  expect(JSON.stringify(screen.toJSON())).toContain('Still confirming your payment');
  expect(screen.root.findAllByProps({ accessibilityLabel: 'Retry payment' })).toHaveLength(0);
});
it('lets authoritative completion override an earlier setup error', async () => {
  jest.mocked(useOrder).mockReturnValue({
    data: { data: { paymentStatus: 'completed', status: 'confirmed' } },
  } as never);
  await act(async () => {
    screen = create(<PaymentResult />);
  });
  expect(JSON.stringify(screen.toJSON())).not.toContain('Payment not started');
  expect(screen.root.findAllByProps({ accessibilityLabel: 'Retry payment' })).toHaveLength(0);
});
