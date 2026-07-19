// Your Data — functional DPDP Act 2023 data-subject screen.
// Right to Access (download a copy of your data) + Right to Erasure (delete
// account). Backed by /customer/me/export and /customer/me/delete.

import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Pressable,
  Share,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { Download, ShieldAlert } from 'lucide-react-native';

import { customerColors } from '@homechef/mobile-shared/theme';
import { KeyboardAwareScrollView } from '@homechef/mobile-shared/ui';
import { ScreenHeader } from '../components/ScreenHeader';
import { useProfile } from '../hooks/useProfile';
import { useExportMyData, useDeleteAccount } from '../hooks/useDataPrivacy';
import { friendlyErrorMessage } from '../lib/errors';
import { useAuthStore } from '../store/auth-store';

export default function DataPrivacyScreen() {
  const router = useRouter();
  const { data: profile } = useProfile();
  const exportData = useExportMyData();
  const deleteAccount = useDeleteAccount();
  const [confirmEmail, setConfirmEmail] = useState('');

  const email = profile?.email ?? '';
  const canDelete =
    confirmEmail.trim().toLowerCase() === email.trim().toLowerCase() && email.length > 0;

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
      'This hides your account immediately and permanently erases your data after a 30-day window. This cannot be undone after that window. Continue?',
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
                  'Your account is now hidden. Contact support within 30 days to cancel.',
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
          >
            {({ pressed }) => (
              <View
                className={`mt-3 rounded-lg min-h-[48px] items-center justify-center ${
                  pressed ? 'bg-coral-pressed' : 'bg-coral'
                }`}
              >
                {exportData.isPending ? (
                  <ActivityIndicator color="#fff" />
                ) : (
                  <Text className="text-base font-semibold text-white">Download</Text>
                )}
              </View>
            )}
          </Pressable>
        </View>

        {/* ── Right to Erasure ── */}
        <View className="mx-4 mt-6 rounded-xl overflow-hidden border border-destructive/30 bg-canvas p-4">
          <View className="flex-row items-center gap-2">
            <ShieldAlert size={18} color={customerColors.destructive?.DEFAULT ?? '#B22B0E'} />
            <Text className="text-base font-semibold text-destructive">Delete my account</Text>
          </View>
          <Text className="text-sm text-charcoal-soft mt-1">
            To confirm, type your email{email ? ` (${email})` : ''} below. Your account is hidden
            immediately and erased after 30 days.
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
          >
            {({ pressed }) => (
              <View
                className={`mt-3 rounded-lg min-h-[48px] items-center justify-center border ${
                  !canDelete
                    ? 'border-hairline bg-surface-soft'
                    : pressed
                      ? 'border-destructive bg-surface-soft'
                      : 'border-destructive bg-canvas'
                }`}
              >
                {deleteAccount.isPending ? (
                  <ActivityIndicator color={customerColors.destructive?.DEFAULT ?? '#B22B0E'} />
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
