import { router } from 'expo-router';
import { useAlert } from '@homechef/mobile-shared/ui';
import { useAuthStore } from '../store/auth-store';
import { useCustomerOnboardingStore } from '../store/onboarding-store';

// Shared "Cancel" handler for the onboarding wizard: confirm, wipe the
// persisted draft, sign out, and land back on the login screen. Onboarding
// gates the tabs, so cancelling means leaving — the account itself survives
// and the wizard restarts fresh on the next sign-in.
export function useCancelOnboarding() {
  const { showAlert } = useAlert();

  return () =>
    showAlert(
      'Cancel onboarding?',
      'Everything entered so far will be cleared, and you can pick up again whenever you sign back in.',
      [
        { text: 'Keep going', style: 'cancel' },
        {
          text: 'Discard & sign out',
          style: 'destructive',
          onPress: () => {
            useCustomerOnboardingStore.getState().reset();
            // Navigate first, flip auth state a tick later — tearing down auth
            // while the wizard is mounted re-renders it mid-teardown.
            router.replace('/(auth)/login' as never);
            setTimeout(() => {
              void useAuthStore.getState().logout();
            }, 0);
          },
        },
      ],
    );
}
