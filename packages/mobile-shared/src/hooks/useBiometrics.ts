// Biometric authentication hook (D-08)
// Behavior: optional setting, unlock cached JWT on app resume from background
// Does NOT re-authenticate with the server — only unlocks the secure store token
// Source: expo-local-authentication v55 docs

import { useEffect, useRef, useState } from 'react';
import { AppState, AppStateStatus } from 'react-native';
import * as LocalAuthentication from 'expo-local-authentication';
import { useAuthStore } from './useAuth';

export async function checkBiometricCapability(): Promise<{
  hasHardware: boolean;
  isEnrolled: boolean;
}> {
  const [hasHardware, isEnrolled] = await Promise.all([
    LocalAuthentication.hasHardwareAsync(),
    LocalAuthentication.isEnrolledAsync(),
  ]);
  return { hasHardware, isEnrolled };
}

export async function authenticateWithBiometrics(): Promise<boolean> {
  const { hasHardware, isEnrolled } = await checkBiometricCapability();
  if (!hasHardware || !isEnrolled) return false;

  const result = await LocalAuthentication.authenticateAsync({
    promptMessage: 'Authenticate to continue',
    fallbackLabel: 'Use password',
    cancelLabel: 'Cancel',
  });
  return result.success;
}

/**
 * Hook that prompts for biometrics when the app returns from background.
 * Only active when ALL three conditions are true:
 *   1. isLoading === false  (hydrateFromStorage has completed — store values are real)
 *   2. isAuthenticated === true
 *   3. biometricsEnabled === true
 *
 * If any condition is false the AppState listener is not registered, preventing
 * the race condition where the prompt fires before hydrateFromStorage resolves.
 *
 * If biometric auth fails or is cancelled, onLockout is called — caller should
 * navigate to /(auth)/login.
 */
export function useBiometricLock(options: { onLockout: () => void }) {
  const { isAuthenticated, isLoading, biometricsEnabled } = useAuthStore();
  const [isLocked, setIsLocked] = useState(false);
  const appState = useRef<AppStateStatus>(AppState.currentState);

  useEffect(() => {
    // Gate: do not register the listener until hydration is complete.
    // isLoading starts as true and flips to false only after hydrateFromStorage() resolves.
    // Registering before that would read stale isAuthenticated/biometricsEnabled values.
    if (isLoading) return;
    if (!isAuthenticated || !biometricsEnabled) return;

    const subscription = AppState.addEventListener(
      'change',
      async (nextState: AppStateStatus) => {
        if (
          appState.current.match(/inactive|background/) &&
          nextState === 'active'
        ) {
          // App came to foreground — prompt for biometrics
          setIsLocked(true);
          const success = await authenticateWithBiometrics();
          if (success) {
            setIsLocked(false);
          } else {
            setIsLocked(false);
            options.onLockout();
          }
        }
        appState.current = nextState;
      }
    );

    return () => subscription.remove();
  // Re-run when hydration completes (isLoading flips false) or auth state changes.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoading, isAuthenticated, biometricsEnabled]);

  return { isLocked };
}
