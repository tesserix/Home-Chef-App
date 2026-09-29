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
}));
jest.mock('react-native-safe-area-context', () => ({
  SafeAreaView: 'SafeAreaView',
}));
jest.mock('react-native-webview', () => ({ WebView: 'WebView' }));
jest.mock('lucide-react-native', () => ({ ChevronLeft: 'ChevronLeft' }));
jest.mock('expo-router', () => ({
  router: { replace: jest.fn(), back: jest.fn() },
  useLocalSearchParams: () => ({
    orderId: 'order-1',
    cashfreeOrderId: 'cf-1',
    paymentSessionId: 'session-1',
    env: 'SANDBOX',
  }),
}));
jest.mock('@tanstack/react-query', () => ({
  useQueryClient: () => ({ invalidateQueries: jest.fn() }),
}));
jest.mock('./api', () => ({ api: { post: jest.fn() } }));
import { api } from './api';
import { router } from 'expo-router';
import Checkout from '../app/payment/cashfree';

(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;
let screen: ReturnType<typeof create>;
beforeEach(() => {
  jest.useFakeTimers();
  jest.clearAllMocks();
  jest.mocked(api.post).mockResolvedValue({ data: { status: 'pending' } });
});
afterEach(async () => {
  if (screen) await act(async () => screen.unmount());
  jest.useRealTimers();
});

it('offers server verification when the gateway never completes, without starting another charge', async () => {
  await act(async () => {
    screen = create(<Checkout />);
  });
  await act(async () => {
    jest.advanceTimersByTime(60000);
  });
  const recovery = screen.root.findByProps({
    accessibilityLabel: 'Check payment status',
  });
  expect(router.replace).not.toHaveBeenCalled();
  jest.mocked(api.post).mockRejectedValueOnce(new Error('provider timeout'));
  await act(async () => {
    await recovery.props.onPress();
  });
  expect(api.post).toHaveBeenLastCalledWith('/v1/payments/order/order-1/verify', {
    cashfreeOrderId: 'cf-1',
  });
  expect(router.replace).toHaveBeenCalledTimes(1);
  expect(router.replace).toHaveBeenCalledWith('/payment/result?order_id=order-1');
  expect(jest.mocked(api.post).mock.calls.every(([path]) => String(path).endsWith('/verify'))).toBe(
    true,
  );
});

it('offers recovery immediately when the checkout document fails to load', async () => {
  await act(async () => {
    screen = create(<Checkout />);
  });
  const webview = screen.root.findByType('WebView' as any);
  await act(async () => {
    webview.props.onError({
      nativeEvent: { description: 'Network timed out' },
    });
  });
  expect(screen.root.findByProps({ accessibilityLabel: 'Check payment status' })).toBeTruthy();
  expect(router.replace).not.toHaveBeenCalled();
});

it('does not overlap slow payment verifications', async () => {
  jest.mocked(api.post).mockImplementation(() => new Promise(() => {}));
  await act(async () => {
    screen = create(<Checkout />);
  });
  await act(async () => {
    jest.advanceTimersByTime(20000);
  });
  expect(api.post).toHaveBeenCalledTimes(1);
});

it('shares an in-flight verification with gateway completion and navigates once', async () => {
  let resolveVerification!: () => void;
  const response = new Promise<void>((resolve) => {
    resolveVerification = resolve;
  }).then(() => ({ data: { status: 'completed' } }));
  jest.mocked(api.post).mockReturnValue(response);
  await act(async () => {
    screen = create(<Checkout />);
  });
  await act(async () => {
    jest.advanceTimersByTime(4000);
  });
  const webview = screen.root.findByType('WebView' as any);
  await act(async () => {
    webview.props.onMessage({ nativeEvent: { data: '{"type":"settled"}' } });
    webview.props.onMessage({ nativeEvent: { data: '{"type":"error"}' } });
  });
  expect(api.post).toHaveBeenCalledTimes(1);
  await act(async () => {
    resolveVerification();
  });
  expect(router.replace).toHaveBeenCalledTimes(1);
  expect(router.replace).toHaveBeenCalledWith('/payment/result?order_id=order-1');
});
