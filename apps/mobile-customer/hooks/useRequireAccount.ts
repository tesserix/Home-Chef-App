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
              // Guest mode is NOT cleared here. It used to be, as the signal that
              // told the root gate to show login — but browsing is now the default
              // for anyone signed out, so clearing it would just re-enter guest on
              // the next render. The root gate leaves the (auth) group alone
              // instead, so pushing the screen is enough.
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
 *
 * Derived purely from "signed out, and we know it" — it deliberately does NOT
 * read the persisted `isGuest` flag. That flag is written asynchronously, so
 * anything that re-ran hydration could read it back before the write landed and
 * flip this to false for a frame; with the root layout writing it on every pass,
 * the two fought each other and the header visibly oscillated between the guest
 * and signed-in treatments. Persisted state is the wrong tool for a question
 * that `isAuthenticated` already answers exactly.
 *
 * `isLoading` keeps the first frames neutral, so a returning signed-in customer
 * never gets a flash of the guest UI while the session rehydrates.
 */
export function useIsGuest(): boolean {
  return useAuthStore((s) => !s.isAuthenticated && !s.isLoading);
}
