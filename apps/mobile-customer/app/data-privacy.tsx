// Your Data — functional DPDP Act 2023 data-subject screen.
// Right to Access (download a copy of your data) + Right to Erasure (delete
// account). Backed by /customer/me/export and /customer/me/delete.

import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Platform,
  Pressable,
  Share,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { Download, PauseCircle, ShieldAlert } from 'lucide-react-native';

import { customerColors } from '@homechef/mobile-shared/theme';
import { KeyboardAwareScrollView } from '@homechef/mobile-shared/ui';
import { ScreenHeader } from '../components/ScreenHeader';
import { useProfile } from '../hooks/useProfile';
import {
  useExportMyData,
  useDeleteAccount,
  useDeactivateAccount,
  useDeletionEligibility,
} from '../hooks/useDataPrivacy';
import { friendlyErrorMessage } from '../lib/errors';
import { useAuthStore } from '../store/auth-store';

// Android ripple tints — translucent tokens, never a new literal colour.
const CANVAS_RIPPLE = `${customerColors.canvas}33`;
const DESTRUCTIVE_RIPPLE = `${customerColors.destructive.DEFAULT}14`;

export default function DataPrivacyScreen() {
  const router = useRouter();
  const { data: profile } = useProfile();
  const exportData = useExportMyData();
  const deleteAccount = useDeleteAccount();
  const deactivate = useDeactivateAccount();
  const eligibility = useDeletionEligibility();
  const [confirmEmail, setConfirmEmail] = useState('');

  const email = profile?.email ?? '';
  const blockers = eligibility.data?.blockers ?? [];
  const blocked = blockers.length > 0;
  const retentionDays = eligibility.data?.retentionDays ?? 180;
  const canDelete =
    !blocked &&
    confirmEmail.trim().toLowerCase() === email.trim().toLowerCase() &&
    email.length > 0;

  function handleDeactivate() {
    Alert.alert(
      'Pause my account',
      'Your profile is hidden and notifications stop. Nothing is deleted — sign in again any time to pick up where you left off.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Pause',
          onPress: () =>
            deactivate.mutate(undefined, {
              // Route to the paused screen rather than signing out. The session
              // token stays valid (the account is paused, not deleted) and
              // /me/reactivate is the one endpoint a paused account may call,
              // so the user can undo this immediately. Signing out here also
              // raced the api-client's own 403 redirect to the same screen.
              onSuccess: () => router.replace('/account-paused'),
              onError: (error) =>
                Alert.alert(
                  'Could not pause',
                  friendlyErrorMessage(error, 'Please try again.'),
                ),
            }),
        },
      ],
    );
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
          // User dismissed the share sheet — nothing to do.
        }
      },
      onError: (error) =>
        Alert.alert(
          'Export failed',
          friendlyErrorMessage(error, 'Could not prepare your data. Please try again.'),
        ),
    });
  }

  function handleDelete() {
    Alert.alert(
      'Delete account',
      `Your account is removed straight away and your sign-in stops working. If you change your mind, signing up again with this email within ${retentionDays} days restores your history — after that everything is erased for good. Continue?`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: () =>
            deleteAccount.mutate(confirmEmail.trim(), {
              onSuccess: () => {
                Alert.alert(
                  'Account deleted',
                  `Sorry to see you go. Sign up again with this email within ${retentionDays} days if you want your history back.`,
                  [
                    {
                      text: 'OK',
                      onPress: () => {
                        useAuthStore.getState().logout();
                        router.replace('/(auth)/login');
                      },
                    },
                  ],
                );
              },
              onError: (error) =>
                Alert.alert(
                  'Delete failed',
                  friendlyErrorMessage(error, 'Could not delete your account. Please try again.'),
                ),
            }),
        },
      ],
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      {/* Pushed screen — headerShown is false app-wide, so draw the back
          affordance ourselves (this screen shipped without one). */}
      <ScreenHeader title="Your Data" />
      <KeyboardAwareScrollView contentContainerStyle={{ paddingBottom: 40 }}>
        <View className="px-4 pt-3 pb-2">
          <Text className="text-sm text-charcoal-soft">
            Access or delete the personal data we hold, per India's DPDP Act 2023.
          </Text>
        </View>

        {/* ── Right to Access ── */}
        <View className="mx-4 mt-4 rounded-xl overflow-hidden border border-hairline bg-canvas p-4">
          <View className="flex-row items-center gap-2">
            <Download size={18} color={customerColors.charcoal.soft} />
            <Text className="text-base font-semibold text-charcoal">Download my data</Text>
          </View>
          <Text className="text-sm text-charcoal-soft mt-1">
            Get a machine-readable copy of your profile, orders, addresses, wallet, and more.
          </Text>
          <Pressable
            onPress={handleExport}
            disabled={exportData.isPending}
            accessibilityRole="button"
            accessibilityLabel="Download my data"
            android_ripple={exportData.isPending ? undefined : { color: CANVAS_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                className={`mt-3 rounded-lg min-h-[48px] items-center justify-center ${
                  pressed && Platform.OS === 'ios' ? 'bg-coral-pressed' : 'bg-coral'
                }`}
              >
                {exportData.isPending ? (
                  <ActivityIndicator color={customerColors.canvas} />
                ) : (
                  <Text className="text-base font-semibold text-white">Download</Text>
                )}
              </View>
            )}
          </Pressable>
        </View>

        {/* ── Pause (reversible) — offered before the destructive option so
            someone who just wants a break does not reach for Delete. ── */}
        <View className="mx-4 mt-6 rounded-xl overflow-hidden border border-hairline bg-canvas p-4">
          <View className="flex-row items-center gap-2">
            <PauseCircle size={18} color={customerColors.charcoal.soft} />
            <Text className="text-base font-semibold text-charcoal">Pause my account</Text>
          </View>
          <Text className="text-sm text-charcoal-soft mt-1">
            Hide your profile and stop notifications. Nothing is deleted, and you can come back any
            time by signing in again.
          </Text>
          <Pressable
            onPress={handleDeactivate}
            disabled={deactivate.isPending}
            accessibilityRole="button"
            accessibilityLabel="Pause my account"
            android_ripple={
              deactivate.isPending ? undefined : { color: CANVAS_RIPPLE, borderless: false }
            }
          >
            {({ pressed }) => (
              <View
                className={`mt-3 rounded-lg min-h-[48px] items-center justify-center border border-hairline ${
                  pressed && Platform.OS === 'ios' ? 'bg-surface-soft' : 'bg-canvas'
                }`}
              >
                {deactivate.isPending ? (
                  <ActivityIndicator color={customerColors.charcoal.soft} />
                ) : (
                  <Text className="text-base font-semibold text-charcoal">Pause account</Text>
                )}
              </View>
            )}
          </Pressable>
        </View>

        {/* ── Right to Erasure ── */}
        <View className="mx-4 mt-6 rounded-xl overflow-hidden border border-destructive/30 bg-canvas p-4">
          <View className="flex-row items-center gap-2">
            <ShieldAlert size={18} color={customerColors.destructive.DEFAULT} />
            <Text className="text-base font-semibold text-destructive">Delete my account</Text>
          </View>

          {/* Blockers first: nothing is deleted while money or an order is in
              flight, so say so before asking anyone to type their email. */}
          {blocked ? (
            <View className="mt-2 rounded-lg bg-surface-soft p-3">
              <Text className="text-sm font-semibold text-charcoal">
                Finish these before deleting
              </Text>
              {blockers.map((b) => (
                <Text key={b.code} className="text-sm text-charcoal-soft mt-1">
                  • {b.label}
                  {b.amount ? ` (₹${b.amount.toFixed(2)})` : b.count ? ` (${b.count})` : ''}
                </Text>
              ))}
              <Text className="text-xs text-charcoal-soft mt-2">
                You can pause your account instead — that works right away.
              </Text>
            </View>
          ) : null}

          <Text className="text-sm text-charcoal-soft mt-1">
            To confirm, type your email{email ? ` (${email})` : ''} below. Your sign-in stops
            working immediately. Sign up again with this email within {retentionDays} days to
            restore your history — after that it is erased permanently.
          </Text>
          <TextInput
            className="mt-3 text-base text-charcoal bg-transparent border border-hairline rounded-lg px-3 min-h-[48px]"
            value={confirmEmail}
            onChangeText={setConfirmEmail}
            placeholder="Type your email to confirm"
            placeholderTextColor={customerColors.charcoal.soft}
            autoCapitalize="none"
            keyboardType="email-address"
            accessibilityLabel="Confirm email to delete account"
          />
          <Pressable
            onPress={handleDelete}
            disabled={!canDelete || deleteAccount.isPending}
            accessibilityRole="button"
            accessibilityLabel="Delete my account"
            android_ripple={
              !canDelete || deleteAccount.isPending
                ? undefined
                : { color: DESTRUCTIVE_RIPPLE, borderless: false }
            }
          >
            {({ pressed }) => (
              <View
                className={`mt-3 rounded-lg min-h-[48px] items-center justify-center border ${
                  !canDelete
                    ? 'border-hairline bg-surface-soft'
                    : pressed && Platform.OS === 'ios'
                      ? 'border-destructive bg-surface-soft'
                      : 'border-destructive bg-canvas'
                }`}
              >
                {deleteAccount.isPending ? (
                  <ActivityIndicator color={customerColors.destructive.DEFAULT} />
                ) : (
                  <Text
                    className={`text-base font-semibold ${
                      canDelete ? 'text-destructive' : 'text-charcoal-soft'
                    }`}
                  >
                    Delete account
                  </Text>
                )}
              </View>
            )}
          </Pressable>
        </View>
      </KeyboardAwareScrollView>
    </SafeAreaView>
  );
}
