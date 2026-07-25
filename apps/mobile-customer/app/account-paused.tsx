// Account paused — the way back from a self-service pause.
//
// A deactivated account 403s on every protected request, so without this screen
// a paused user signs in successfully and then lands in a dead app: every
// query fails and nothing explains why. Pausing would be a one-way door only
// support could open, which defeats the point of offering it as the gentle
// alternative to deletion.
//
// The api client routes here on 403 + status=account_deactivated. /me/reactivate
// is the one endpoint the auth middleware lets a deactivated account reach.

import { ActivityIndicator, Platform, Pressable, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { PauseCircle } from 'lucide-react-native';

import { customerColors } from '@homechef/mobile-shared/theme';
import { useReactivateAccount } from '../hooks/useDataPrivacy';
import { friendlyErrorMessage } from '../lib/errors';
import { useAuthStore } from '../store/auth-store';
import { useAlert } from '@homechef/mobile-shared/ui';

export default function AccountPausedScreen() {
  const { showAlert } = useAlert();
  const router = useRouter();
  const reactivate = useReactivateAccount();
  const { status } = useLocalSearchParams<{ status?: string }>();

  // A deleted account cannot be revived from here — the restore path runs at
  // sign-up, not sign-in, because the credential itself is gone.
  const deleted = status === 'account_deleted';

  function handleReactivate() {
    reactivate.mutate(undefined, {
      onSuccess: () => {
        showAlert('Welcome back', 'Your account is active again.', [
          { text: 'OK', onPress: () => router.replace('/(tabs)') },
        ]);
      },
      onError: (error) =>
        showAlert(
          'Could not reactivate',
          friendlyErrorMessage(error, 'Please try again, or contact support.'),
        ),
    });
  }

  function handleSignOut() {
    useAuthStore.getState().logout();
    router.replace('/(auth)/login');
  }

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right', 'bottom']}>
      <View className="flex-1 items-center justify-center px-6">
        <PauseCircle size={48} color={customerColors.charcoal.soft} />

        <Text className="mt-4 text-center font-display text-2xl font-semibold text-charcoal">
          {deleted ? 'This account was deleted' : 'Your account is paused'}
        </Text>

        {/* Deliberately warm and reassuring. A pause is a choice the user made,
            not a punishment — the copy should make coming back feel easy and
            make clear nothing was lost. */}
        <Text className="mt-2 text-center text-base text-charcoal-soft">
          {deleted
            ? 'You can come back any time — sign up again with the same email within 180 days and your history comes with you.'
            : 'Everything is safe and waiting for you. Your orders, saved addresses and wallet balance are exactly as you left them, and ordering is switched off until you are ready.'}
        </Text>

        {!deleted ? (
          <Text className="mt-3 text-center text-sm text-charcoal-soft">
            Turn it back on whenever you like — it takes effect straight away.
          </Text>
        ) : null}

        {!deleted ? (
          <Pressable
            onPress={handleReactivate}
            disabled={reactivate.isPending}
            accessibilityRole="button"
            accessibilityLabel="Reactivate my account"
            className="mt-8 w-full"
          >
            {({ pressed }) => (
              <View
                className={`min-h-[48px] items-center justify-center rounded-lg ${
                  pressed && Platform.OS === 'ios' ? 'bg-coral-pressed' : 'bg-coral'
                }`}
              >
                {reactivate.isPending ? (
                  <ActivityIndicator color={customerColors.canvas} />
                ) : (
                  <Text className="text-base font-semibold text-white">Turn my account back on</Text>
                )}
              </View>
            )}
          </Pressable>
        ) : null}

        <Pressable
          onPress={handleSignOut}
          accessibilityRole="button"
          accessibilityLabel="Sign out"
          className="mt-3 w-full"
        >
          <View className="min-h-[48px] items-center justify-center rounded-lg border border-hairline">
            <Text className="text-base font-semibold text-charcoal">Sign out</Text>
          </View>
        </Pressable>
      </View>
    </SafeAreaView>
  );
}
