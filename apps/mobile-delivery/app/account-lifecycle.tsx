// Account — pause or delete your driver account, plus a copy of your data.
//
// Apple 5.1.1(v) requires deletion to be initiated in-app; this screen replaces
// the "email support@homechef.in to request account deletion" alert that
// Settings used to show, which would have failed review.
//
// Deletion refuses while a delivery is still in hand: a driver must not be able
// to disappear mid-route with a customer's order.

import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  ScrollView,
  Share,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';

import {
  useDeactivateAccount,
  useDeleteAccount,
  useDeletionEligibility,
  useExportMyData,
} from '../hooks/useAccountLifecycle';
import { useAuthStore } from '../store/auth-store';

export default function AccountLifecycleScreen() {
  const user = useAuthStore((s) => s.user);
  const eligibility = useDeletionEligibility();
  const deleteAccount = useDeleteAccount();
  const deactivate = useDeactivateAccount();
  const exportData = useExportMyData();
  const [confirmEmail, setConfirmEmail] = useState('');

  const email = user?.email ?? '';
  const blockers = eligibility.data?.blockers ?? [];
  const blocked = blockers.length > 0;
  const retentionDays = eligibility.data?.retentionDays ?? 180;
  const canDelete =
    !blocked &&
    email.length > 0 &&
    confirmEmail.trim().toLowerCase() === email.trim().toLowerCase();

  function signOutToLogin() {
    useAuthStore.getState().logout();
    router.replace('/(auth)/login');
  }

  function handleExport() {
    exportData.mutate(undefined, {
      onSuccess: async (data) => {
        try {
          await Share.share({
            title: 'My Home Chef data',
            message: JSON.stringify(data, null, 2),
          });
        } catch {
          // Share sheet dismissed — nothing to do.
        }
      },
      onError: () =>
        Alert.alert('Export failed', 'Could not prepare your data. Please try again.'),
    });
  }

  function handleDeactivate() {
    Alert.alert(
      'Pause my account',
      'You go offline and stop receiving delivery requests. Nothing is deleted — sign in again any time to start taking deliveries.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Pause',
          onPress: () =>
            deactivate.mutate(undefined, {
              // Route to the paused screen rather than signing out — the
              // session stays valid and /me/reactivate is reachable, so the
              // driver can come back immediately. Signing out also raced the
              // api client's own 403 redirect to the same screen.
              onSuccess: () => router.replace('/account-paused' as never),
              onError: () => Alert.alert('Could not pause', 'Please try again.'),
            }),
        },
      ],
    );
  }

  function handleDelete() {
    Alert.alert(
      'Delete account',
      `Your account is removed straight away and your sign-in stops working. If you sign up again with this email within ${retentionDays} days your history comes back — but you will need to be verified again and re-upload your documents. After ${retentionDays} days everything is erased for good.`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: () =>
            deleteAccount.mutate(confirmEmail.trim(), {
              onSuccess: () =>
                Alert.alert(
                  'Account deleted',
                  `Sorry to see you go. Sign up again with this email within ${retentionDays} days if you want your history back.`,
                  [{ text: 'OK', onPress: signOutToLogin }],
                ),
              onError: () =>
                Alert.alert(
                  'Delete failed',
                  'Could not delete your account. Please try again or contact support.',
                ),
            }),
        },
      ],
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-paper" edges={['top', 'left', 'right']}>
      <View className="flex-row items-center px-4 py-3">
        <TouchableOpacity
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={8}
        >
          <ChevronLeft size={24} color="#0E0E0C" />
        </TouchableOpacity>
        <Text className="ml-2 font-display text-2xl font-semibold text-ink">Account</Text>
      </View>

      <ScrollView contentContainerStyle={{ paddingBottom: 40 }}>
        {/* Right to Access */}
        <Text className="text-xs font-semibold text-ink-muted uppercase tracking-wider px-4 mb-1 mt-4">
          Your data
        </Text>
        <View className="mx-4 rounded-lg border border-mist bg-paper p-4">
          <Text className="text-sm text-ink-muted">
            Download a copy of your profile, deliveries and documents.
          </Text>
          <TouchableOpacity
            onPress={handleExport}
            disabled={exportData.isPending}
            accessibilityRole="button"
            accessibilityLabel="Download my data"
            className="mt-3 min-h-[48px] items-center justify-center rounded-lg border border-mist"
          >
            {exportData.isPending ? (
              <ActivityIndicator color="#0E0E0C" />
            ) : (
              <Text className="text-base font-semibold text-ink">Download my data</Text>
            )}
          </TouchableOpacity>
        </View>

        {/* Reversible pause, offered before the destructive option. */}
        <Text className="text-xs font-semibold text-ink-muted uppercase tracking-wider px-4 mb-1 mt-6">
          Take a break
        </Text>
        <View className="mx-4 rounded-lg border border-mist bg-paper p-4">
          <Text className="text-sm text-ink-muted">
            Go offline and stop getting delivery requests. Nothing is deleted and you keep your
            verification.
          </Text>
          <TouchableOpacity
            onPress={handleDeactivate}
            disabled={deactivate.isPending}
            accessibilityRole="button"
            accessibilityLabel="Pause my account"
            className="mt-3 min-h-[48px] items-center justify-center rounded-lg border border-mist"
          >
            {deactivate.isPending ? (
              <ActivityIndicator color="#0E0E0C" />
            ) : (
              <Text className="text-base font-semibold text-ink">Pause my account</Text>
            )}
          </TouchableOpacity>
        </View>

        {/* Deletion */}
        <Text className="text-xs font-semibold text-ink-muted uppercase tracking-wider px-4 mb-1 mt-6">
          Delete account
        </Text>
        <View className="mx-4 rounded-lg border border-destructive bg-paper p-4">
          {blocked ? (
            <View className="mb-3 rounded-lg bg-bone p-3">
              <Text className="text-sm font-semibold text-ink">Finish these before deleting</Text>
              {blockers.map((b) => (
                <Text key={b.code} className="text-sm text-ink-muted mt-1">
                  {'•'} {b.label}
                  {b.amount ? ` (₹${b.amount.toFixed(2)})` : b.count ? ` (${b.count})` : ''}
                </Text>
              ))}
              <Text className="text-xs text-ink-muted mt-2">
                You can pause your account instead — that works right away.
              </Text>
            </View>
          ) : null}

          <Text className="text-sm text-ink-muted">
            To confirm, type your email{email ? ` (${email})` : ''} below. Signing up again with
            this email within {retentionDays} days restores your history, but you will need to be
            verified again.
          </Text>

          <TextInput
            className="mt-3 min-h-[48px] rounded-lg border border-mist px-3 text-base text-ink"
            value={confirmEmail}
            onChangeText={setConfirmEmail}
            placeholder="Type your email to confirm"
            placeholderTextColor="#888888"
            autoCapitalize="none"
            keyboardType="email-address"
            accessibilityLabel="Confirm email to delete account"
          />

          <TouchableOpacity
            onPress={handleDelete}
            disabled={!canDelete || deleteAccount.isPending}
            accessibilityRole="button"
            accessibilityLabel="Delete my account"
            className={`mt-3 min-h-[48px] items-center justify-center rounded-lg border ${
              canDelete ? 'border-destructive' : 'border-mist'
            }`}
          >
            {deleteAccount.isPending ? (
              <ActivityIndicator color="#B22B0E" />
            ) : (
              <Text
                className={`text-base font-semibold ${
                  canDelete ? 'text-destructive' : 'text-ink-muted'
                }`}
              >
                Delete account
              </Text>
            )}
          </TouchableOpacity>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
