// useRequireAccount — the account gate for guest browsing.
//
// App Review guideline 5.1.1(iv): an app may only require an account for
// features that genuinely need one. Browsing chefs, menus and reviews does not;
// ordering, paying, saving favourites and anything tied to an identity does.
//
// So the login wall moved off the app's front door and onto the specific
// actions that cannot work without an account. Every one of those actions calls
// requireAccount() first.

import { useCallback } from 'react';
import { Alert } from 'react-native';
import { router } from 'expo-router';

import { useAuthStore } from '../store/auth-store';

/**
 * Returns a guard to call at the top of any action that needs an account.
 *
 * ```ts
 * const requireAccount = useRequireAccount();
 * function handleAddToCart() {
 *   if (!requireAccount('add items to your cart')) return;
 *   ...
 * }
 * ```
 *
 * Returns true when the caller may proceed. When it returns false it has
 * already shown the prompt, so the caller just bails.
 *
 * @param action - what the user was trying to do, phrased to follow
 *   "Sign in to ..." — e.g. "place an order", "save this chef".
 */
export function useRequireAccount(): (action: string) => boolean {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  return useCallback(
    (action: string) => {
      if (isAuthenticated) return true;

      Alert.alert(
        'Sign in to continue',
        `You need an account to ${action}. It takes a minute.`,
        [
          { text: 'Not now', style: 'cancel' },
          {
            text: 'Sign in',
            onPress: () => {
              // Leave guest mode before navigating, or the root auth gate sees
              // isGuest and bounces straight back to the tabs.
              void useAuthStore.getState().setGuest(false);
              router.push('/(auth)/login');
            },
          },
        ],
      );
      return false;
    },
    [isAuthenticated],
  );
}

/**
 * True when the user is browsing without an account.
 *
 * For rendering — hiding a "Your orders" section, showing a sign-in banner.
 * Use requireAccount() for the actual gate; a hidden button is not a guard.
 */
export function useIsGuest(): boolean {
  return useAuthStore((s) => s.isGuest && !s.isAuthenticated);
}
