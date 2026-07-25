// Kitchen paused — the way back from a self-service pause.
//
// A deactivated account 403s on every protected request, so without this screen
// a paused chef signs in and lands in a dead app with no explanation and no way
// back. /me/reactivate is the one endpoint the auth middleware lets a
// deactivated account reach.

import { ActivityIndicator, Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { theme } from '@homechef/mobile-shared/theme';

import { useReactivateAccount } from '../hooks/useAccountLifecycle';
import { useAuthStore } from '../store/auth-store';
import { useAlert } from '@homechef/mobile-shared/ui';

export default function AccountPausedScreen() {
  const { showAlert } = useAlert();
  const reactivate = useReactivateAccount();
  const { status } = useLocalSearchParams<{ status?: string }>();

  // A deleted account cannot be revived here — the restore path runs at
  // sign-up, not sign-in, because the credential itself is gone.
  const deleted = status === 'account_deleted';

  function handleReactivate() {
    reactivate.mutate(undefined, {
      onSuccess: () =>
        showAlert('Welcome back', 'Your kitchen is active again. Reopen when you are ready.', [
          { text: 'OK', onPress: () => router.replace('/(tabs)' as never) },
        ]),
      onError: () =>
        showAlert('Could not reactivate', 'Please try again, or contact support.'),
    });
  }

  function handleSignOut() {
    useAuthStore.getState().logout();
    router.replace('/(auth)/login' as never);
  }

  return (
    <SafeAreaView style={styles.safe} edges={['top', 'left', 'right', 'bottom']}>
      <View style={styles.body}>
        <Text style={styles.title}>
          {deleted ? 'This account was deleted' : 'Your kitchen is paused'}
        </Text>
        <Text style={styles.copy}>
          {deleted
            ? 'You can come back any time — sign up again with the same email within 180 days and your menu and history come with you. Your kitchen will need approving again before customers can order.'
            : 'Your kitchen is resting. Your menu, reviews and approval are all safe, and customers simply will not see you until you reopen. No orders can come in while you are paused.'}
        </Text>

        {!deleted ? (
          <Pressable
            onPress={handleReactivate}
            disabled={reactivate.isPending}
            accessibilityRole="button"
            accessibilityLabel="Reactivate my kitchen"
            style={styles.primary}
          >
            {reactivate.isPending ? (
              <ActivityIndicator color={theme.colors.paper} />
            ) : (
              <Text style={styles.primaryText}>Reopen my kitchen</Text>
            )}
          </Pressable>
        ) : null}

        <Pressable
          onPress={handleSignOut}
          accessibilityRole="button"
          accessibilityLabel="Sign out"
          style={styles.secondary}
        >
          <Text style={styles.secondaryText}>Sign out</Text>
        </Pressable>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: theme.colors.paper },
  body: { flex: 1, alignItems: 'center', justifyContent: 'center', paddingHorizontal: 24 },
  title: {
    fontSize: 24,
    fontWeight: '600',
    color: theme.colors.ink.DEFAULT,
    textAlign: 'center',
  },
  copy: {
    marginTop: 8,
    fontSize: 16,
    lineHeight: 24,
    color: theme.colors.ink.muted,
    textAlign: 'center',
  },
  primary: {
    marginTop: 32,
    alignSelf: 'stretch',
    minHeight: 48,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: theme.colors.ink.DEFAULT,
  },
  primaryText: { fontSize: 16, fontWeight: '600', color: theme.colors.paper },
  secondary: {
    marginTop: 12,
    alignSelf: 'stretch',
    minHeight: 48,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  secondaryText: { fontSize: 16, fontWeight: '600', color: theme.colors.ink.DEFAULT },
});
