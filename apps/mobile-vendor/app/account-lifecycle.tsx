// Account — pause or delete your chef account, plus a copy of your data.
//
// Apple 5.1.1(v) requires deletion to be initiated in-app; this screen replaces
// the "email support to delete your account" alert that Settings used to show,
// which would have failed review.
//
// Deleting is confirm-email gated and refuses while orders are unfulfilled or
// payouts unreleased — a chef must not be able to walk away from money the
// platform still owes them or work a customer has already paid for.

import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Platform,
  Pressable,
  ScrollView,
  Share,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';

import {
  useDeactivateAccount,
  useDeleteAccount,
  useDeletionEligibility,
  useExportMyData,
} from '../hooks/useAccountLifecycle';
import { useAuthStore } from '../store/auth-store';

export default function AccountLifecycleScreen() {
  const profile = useAuthStore((s) => s.user);
  const eligibility = useDeletionEligibility();
  const deleteAccount = useDeleteAccount();
  const deactivate = useDeactivateAccount();
  const exportData = useExportMyData();
  const [confirmEmail, setConfirmEmail] = useState('');

  const email = profile?.email ?? '';
  const blockers = eligibility.data?.blockers ?? [];
  const blocked = blockers.length > 0;
  const retentionDays = eligibility.data?.retentionDays ?? 180;
  const canDelete =
    !blocked &&
    email.length > 0 &&
    confirmEmail.trim().toLowerCase() === email.trim().toLowerCase();

  function signOutToLogin() {
    useAuthStore.getState().logout();
    router.replace('/(auth)/login' as never);
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
      'Pause my kitchen',
      'Your kitchen is hidden from customers and stops taking orders. Your menu, history and approval stay exactly as they are — sign in again any time to reopen.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Pause',
          onPress: () =>
            deactivate.mutate(undefined, {
              onSuccess: () =>
                Alert.alert(
                  'Kitchen paused',
                  'Sign in again whenever you are ready to reopen.',
                  [{ text: 'OK', onPress: signOutToLogin }],
                ),
              onError: () => Alert.alert('Could not pause', 'Please try again.'),
            }),
        },
      ],
    );
  }

  function handleDelete() {
    Alert.alert(
      'Delete account',
      `Your account is removed straight away and your sign-in stops working. If you sign up again with this email within ${retentionDays} days your menu and history come back — but your kitchen has to be approved again and you will need to re-upload your identity documents. After ${retentionDays} days everything is erased for good.`,
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
                  `Sorry to see you go. Sign up again with this email within ${retentionDays} days if you want your kitchen back.`,
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
    <SafeAreaView style={styles.safe} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={8}
        >
          <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} />
        </Pressable>
        <Text style={styles.headerTitle}>Account</Text>
        <View style={{ width: 24 }} />
      </View>

      <ScrollView contentContainerStyle={{ paddingBottom: 40 }}>
        {/* Right to Access */}
        <Text style={styles.sectionLabel}>YOUR DATA</Text>
        <View style={styles.card}>
          <Text style={styles.body}>
            Download a machine-readable copy of your profile, menu, orders and documents.
          </Text>
          <Pressable
            onPress={handleExport}
            disabled={exportData.isPending}
            accessibilityRole="button"
            accessibilityLabel="Download my data"
            style={({ pressed }) => [
              styles.buttonNeutral,
              pressed && Platform.OS === 'ios' && styles.pressedNeutral,
            ]}
          >
            {exportData.isPending ? (
              <ActivityIndicator color={theme.colors.ink.DEFAULT} />
            ) : (
              <Text style={styles.buttonNeutralText}>Download my data</Text>
            )}
          </Pressable>
        </View>

        {/* Reversible pause, offered first so a chef taking a break does not
            reach for the destructive option. */}
        <Text style={styles.sectionLabel}>TAKE A BREAK</Text>
        <View style={styles.card}>
          <Text style={styles.body}>
            Pause your kitchen instead of deleting it. Customers stop seeing you and no new orders
            come in, but nothing is lost and you keep your approval.
          </Text>
          <Pressable
            onPress={handleDeactivate}
            disabled={deactivate.isPending}
            accessibilityRole="button"
            accessibilityLabel="Pause my kitchen"
            style={({ pressed }) => [
              styles.buttonNeutral,
              pressed && Platform.OS === 'ios' && styles.pressedNeutral,
            ]}
          >
            {deactivate.isPending ? (
              <ActivityIndicator color={theme.colors.ink.DEFAULT} />
            ) : (
              <Text style={styles.buttonNeutralText}>Pause my kitchen</Text>
            )}
          </Pressable>
        </View>

        {/* Deletion */}
        <Text style={styles.sectionLabel}>DELETE ACCOUNT</Text>
        <View style={[styles.card, styles.cardDanger]}>
          {blocked ? (
            <View style={styles.blockerBox}>
              <Text style={styles.blockerTitle}>Finish these before deleting</Text>
              {blockers.map((b) => (
                <Text key={b.code} style={styles.blockerItem}>
                  {'•'} {b.label}
                  {b.amount
                    ? ` (₹${b.amount.toFixed(2)})`
                    : b.count
                      ? ` (${b.count})`
                      : ''}
                </Text>
              ))}
              <Text style={styles.blockerHint}>
                You can pause your kitchen instead — that works right away.
              </Text>
            </View>
          ) : null}

          <Text style={styles.body}>
            To confirm, type your email{email ? ` (${email})` : ''} below. Signing up again with
            this email within {retentionDays} days restores your menu and history, but your kitchen
            must be approved again and identity documents re-uploaded.
          </Text>

          <TextInput
            style={styles.input}
            value={confirmEmail}
            onChangeText={setConfirmEmail}
            placeholder="Type your email to confirm"
            placeholderTextColor={theme.colors.ink.muted}
            autoCapitalize="none"
            keyboardType="email-address"
            accessibilityLabel="Confirm email to delete account"
          />

          <Pressable
            onPress={handleDelete}
            disabled={!canDelete || deleteAccount.isPending}
            accessibilityRole="button"
            accessibilityLabel="Delete my account"
            style={({ pressed }) => [
              styles.buttonDanger,
              !canDelete && styles.buttonDisabled,
              canDelete && pressed && Platform.OS === 'ios' && styles.pressedDanger,
            ]}
          >
            {deleteAccount.isPending ? (
              <ActivityIndicator color={theme.colors.destructive.DEFAULT} />
            ) : (
              <Text style={[styles.buttonDangerText, !canDelete && styles.buttonDisabledText]}>
                Delete account
              </Text>
            )}
          </Pressable>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: theme.colors.paper },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  headerTitle: { fontSize: 18, fontWeight: '600', color: theme.colors.ink.DEFAULT },
  sectionLabel: {
    fontSize: 12,
    fontWeight: '600',
    color: theme.colors.ink.muted,
    letterSpacing: 0.5,
    marginTop: 20,
    marginBottom: 8,
    marginHorizontal: 16,
  },
  card: {
    marginHorizontal: 16,
    padding: 16,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    backgroundColor: theme.colors.paper,
  },
  cardDanger: { borderColor: theme.colors.destructive.DEFAULT },
  body: { fontSize: 14, color: theme.colors.ink.muted, lineHeight: 20 },
  input: {
    marginTop: 12,
    minHeight: 48,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: 8,
    paddingHorizontal: 12,
    fontSize: 16,
    color: theme.colors.ink.DEFAULT,
  },
  buttonNeutral: {
    marginTop: 12,
    minHeight: 48,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  pressedNeutral: { backgroundColor: theme.colors.bone },
  buttonNeutralText: { fontSize: 16, fontWeight: '600', color: theme.colors.ink.DEFAULT },
  buttonDanger: {
    marginTop: 12,
    minHeight: 48,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: theme.colors.destructive.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  pressedDanger: { backgroundColor: theme.colors.destructive.tint },
  buttonDangerText: { fontSize: 16, fontWeight: '600', color: theme.colors.destructive.DEFAULT },
  buttonDisabled: { borderColor: theme.colors.mist.DEFAULT },
  buttonDisabledText: { color: theme.colors.ink.muted },
  blockerBox: {
    marginBottom: 12,
    padding: 12,
    borderRadius: 8,
    backgroundColor: theme.colors.bone,
  },
  blockerTitle: { fontSize: 14, fontWeight: '600', color: theme.colors.ink.DEFAULT },
  blockerItem: { fontSize: 14, color: theme.colors.ink.muted, marginTop: 4 },
  blockerHint: { fontSize: 12, color: theme.colors.ink.muted, marginTop: 8 },
});
