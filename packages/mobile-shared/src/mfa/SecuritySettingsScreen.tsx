import { useState } from 'react';
import { ActivityIndicator, StyleSheet, Text, View } from 'react-native';

import type { AxiosInstance } from 'axios';

import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Screen } from '../ui/Screen';
import { useAlert } from '../ui/DialogProvider';
import { theme } from '../theme/tokens';
import { createMFAApi } from './api';
import { useMFAStatus, useTrustedDevices } from './useMFA';

// Security settings — turn two-factor on, manage channels, see remembered
// devices.
//
// The backup codes are shown once, at the moment they are generated, and cannot
// be retrieved afterwards. That is why enabling is a two-step flow here: the
// codes appear on their own panel and the user has to acknowledge them before
// the screen moves on. Slipping them into a toast would guarantee a support
// queue of locked-out users.

export interface SecuritySettingsScreenProps {
  api: AxiosInstance;
  accentColor?: string;
  /**
   * Runs Firebase phone verification and resolves with a fresh ID token whose
   * claims carry the verified number. Supplied per app because each one wires
   * its own Firebase SDK; omit it to hide the phone option entirely.
   */
  onVerifyPhone?: () => Promise<string | null>;
}

export function SecuritySettingsScreen({
  api,
  accentColor,
  onVerifyPhone,
}: SecuritySettingsScreenProps) {
  const { status, loading, error, reload } = useMFAStatus(api);
  const devicesState = useTrustedDevices(api);
  const { showAlert } = useAlert();
  const mfa = createMFAApi(api);

  const [busy, setBusy] = useState(false);
  const [enrollingEmail, setEnrollingEmail] = useState(false);
  const [emailCode, setEmailCode] = useState('');
  const [freshCodes, setFreshCodes] = useState<string[] | null>(null);

  async function run(fn: () => Promise<void>, failure: string) {
    setBusy(true);
    try {
      await fn();
    } catch (err: unknown) {
      const message =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? failure;
      showAlert('Something went wrong', message);
    } finally {
      setBusy(false);
    }
  }

  if (loading) {
    return (
      <Screen>
        <View style={styles.center}>
          <ActivityIndicator />
        </View>
      </Screen>
    );
  }

  if (error || !status) {
    return (
      <Screen>
        <View style={styles.center}>
          <Text style={styles.body}>{error ?? 'Could not load your security settings.'}</Text>
          <Button label="Try again" onPress={() => void reload()} accentColor={accentColor} />
        </View>
      </Screen>
    );
  }

  if (!status.featureEnabled) {
    return (
      <Screen>
        <View style={styles.center}>
          <Text style={styles.body}>Two-factor authentication isn't available yet.</Text>
        </View>
      </Screen>
    );
  }

  // Backup codes panel — takes over the screen so they cannot be missed.
  if (freshCodes) {
    return (
      <Screen scroll>
        <View style={styles.section}>
          <Text style={styles.title}>Save your backup codes</Text>
          <Text style={styles.body}>
            Each code works once. They're the only way back in if you lose access to your email
            and phone. This is the only time they're shown.
          </Text>
          <View style={styles.codeBox}>
            {freshCodes.map((c) => (
              <Text key={c} style={styles.code}>
                {c}
              </Text>
            ))}
          </View>
          <Button
            label="I've saved them"
            fullWidth
            accentColor={accentColor}
            onPress={() => {
              setFreshCodes(null);
              void reload();
            }}
          />
        </View>
      </Screen>
    );
  }

  return (
    <Screen scroll>
      <View style={styles.section}>
        <Text style={styles.title}>Two-factor authentication</Text>
        <Text style={styles.body}>
          {status.enabled
            ? "You'll be asked for a code when you sign in on a device you haven't remembered."
            : 'Add a second step at sign-in so a stolen password on its own is not enough.'}
        </Text>
      </View>

      {/* Channels */}
      <View style={styles.section}>
        <Text style={styles.heading}>Where we send your code</Text>

        <View style={styles.row}>
          <View style={styles.rowText}>
            <Text style={styles.rowTitle}>Email</Text>
            <Text style={styles.rowSub}>
              {status.emailEnrolled ? status.maskedEmail : 'Not set up'}
            </Text>
          </View>
          {!status.emailEnrolled && !enrollingEmail && (
            <Button
              label="Set up"
              size="md"
              accentColor={accentColor}
              loading={busy}
              onPress={() =>
                run(async () => {
                  await mfa.requestEmailEnrollment();
                  setEnrollingEmail(true);
                }, 'Could not send the code.')
              }
            />
          )}
        </View>

        {enrollingEmail && (
          <View style={styles.inlineForm}>
            <Input
              label="Enter the 6-digit code"
              value={emailCode}
              onChangeText={(t) => setEmailCode(t.replace(/[^0-9]/g, '').slice(0, 6))}
              keyboardType="number-pad"
              textContentType="oneTimeCode"
              maxLength={6}
            />
            <Button
              label="Confirm"
              accentColor={accentColor}
              loading={busy}
              disabled={emailCode.length !== 6}
              onPress={() =>
                run(async () => {
                  await mfa.verifyEmailEnrollment(emailCode);
                  setEnrollingEmail(false);
                  setEmailCode('');
                  await reload();
                }, 'That code did not work.')
              }
            />
          </View>
        )}

        {onVerifyPhone && (
          <View style={styles.row}>
            <View style={styles.rowText}>
              <Text style={styles.rowTitle}>Phone</Text>
              <Text style={styles.rowSub}>
                {status.phoneEnrolled ? status.maskedPhone : 'Not set up'}
              </Text>
            </View>
            {!status.phoneEnrolled && (
              <Button
                label="Set up"
                size="md"
                accentColor={accentColor}
                loading={busy}
                onPress={() =>
                  run(async () => {
                    const token = await onVerifyPhone();
                    if (!token) return; // user cancelled
                    await mfa.enrollPhone(token);
                    await reload();
                  }, 'Could not verify that number.')
                }
              />
            )}
          </View>
        )}
      </View>

      {/* Enable / disable */}
      <View style={styles.section}>
        {status.enabled ? (
          <>
            <Text style={styles.rowSub}>
              {status.backupCodesRemaining} backup code
              {status.backupCodesRemaining === 1 ? '' : 's'} left
            </Text>
            <Button
              label="Generate new backup codes"
              variant="secondary"
              loading={busy}
              onPress={() =>
                run(async () => {
                  const { backupCodes } = await mfa.regenerateBackupCodes();
                  setFreshCodes(backupCodes);
                }, 'Could not generate new codes.')
              }
            />
            <Button
              label="Turn off two-factor"
              variant="ghost"
              loading={busy}
              onPress={() =>
                showAlert(
                  'Turn off two-factor?',
                  'Your password alone will be enough to sign in, and every remembered device will be forgotten.',
                  [
                    { text: 'Keep it on', style: 'cancel' },
                    {
                      text: 'Turn off',
                      style: 'destructive',
                      onPress: () =>
                        void run(async () => {
                          await mfa.disable();
                          await reload();
                        }, 'Could not turn two-factor off.'),
                    },
                  ]
                )
              }
            />
          </>
        ) : (
          <Button
            label="Turn on two-factor"
            fullWidth
            accentColor={accentColor}
            loading={busy}
            disabled={!status.emailEnrolled && !status.phoneEnrolled}
            onPress={() =>
              run(async () => {
                const { backupCodes } = await mfa.enable();
                setFreshCodes(backupCodes);
              }, 'Could not turn two-factor on.')
            }
          />
        )}
        {!status.enabled && !status.emailEnrolled && !status.phoneEnrolled && (
          <Text style={styles.rowSub}>Set up email or phone first.</Text>
        )}
      </View>

      {/* Remembered devices */}
      {status.enabled && (
        <View style={styles.section}>
          <Text style={styles.heading}>Remembered devices</Text>
          <Text style={styles.rowSub}>
            These skip the code at sign-in. They stay trusted until you remove them.
          </Text>
          {devicesState.devices.length === 0 ? (
            <Text style={styles.rowSub}>No remembered devices.</Text>
          ) : (
            devicesState.devices.map((d) => (
              <View key={d.id} style={styles.row}>
                <View style={styles.rowText}>
                  <Text style={styles.rowTitle}>{d.label || d.platform || 'Unknown device'}</Text>
                  <Text style={styles.rowSub}>
                    {d.app} · last used {new Date(d.lastSeenAt).toLocaleDateString()}
                  </Text>
                </View>
                <Button label="Remove" size="md" variant="ghost" onPress={() => void devicesState.revoke(d.id)} />
              </View>
            ))
          )}
          {devicesState.devices.length > 0 && (
            <Button
              label="Remove all devices"
              variant="ghost"
              onPress={() => void devicesState.revokeAll()}
            />
          )}
        </View>
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: theme.spacing[3] },
  section: { paddingVertical: theme.spacing[4], gap: theme.spacing[3] },
  title: { fontSize: 22, fontWeight: '700', color: theme.colors.ink.DEFAULT, letterSpacing: -0.3 },
  heading: { fontSize: 16, fontWeight: '600', color: theme.colors.ink.DEFAULT },
  body: { fontSize: 15, lineHeight: 21, color: theme.colors.ink.soft },
  row: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[3] },
  rowText: { flex: 1 },
  rowTitle: { fontSize: 15, fontWeight: '600', color: theme.colors.ink.DEFAULT },
  rowSub: { fontSize: 13, lineHeight: 18, color: theme.colors.ink.soft },
  inlineForm: { gap: theme.spacing[2] },
  codeBox: {
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.md,
    padding: theme.spacing[4],
    gap: theme.spacing[2],
  },
  code: {
    fontSize: 16,
    fontVariant: ['tabular-nums'],
    letterSpacing: 1,
    color: theme.colors.ink.DEFAULT,
    textAlign: 'center',
  },
});
