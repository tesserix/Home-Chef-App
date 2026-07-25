import { useEffect, useState } from 'react';
import { Platform, StyleSheet, Text, View } from 'react-native';

import type { AxiosInstance } from 'axios';

import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Screen } from '../ui/Screen';
import { theme } from '../theme/tokens';
import { useMFAChallenge } from './useMFA';
import type { MFAChannel } from './api';

// The challenge screen — shown when the API refuses a request with mfa_required.
//
// This is a blocking screen by design: until the code is accepted, every other
// request 403s, so there is nothing useful behind it. It offers no "skip", only
// the channels the server said this account can be challenged on, plus the
// backup-code escape hatch.

export interface MFAChallengeScreenProps {
  api: AxiosInstance;
  /** Channels the server offered, from the 403 payload. */
  channels: MFAChannel[];
  /** Masked hints from the 403 payload, e.g. { email: "s•••k@gmail.com" }. */
  masked: Partial<Record<MFAChannel, string>>;
  /** Called once the challenge passes and the token is stored. */
  onVerified: () => void;
  /** Signs out — the only way past this screen without a code. */
  onSignOut: () => void;
  /** App accent (coral for customer, persimmon elsewhere). */
  accentColor?: string;
  /** Shown in the device list on other devices. */
  deviceLabel?: string;
}

export function MFAChallengeScreen({
  api,
  channels,
  masked,
  onVerified,
  onSignOut,
  accentColor,
  deviceLabel,
}: MFAChallengeScreenProps) {
  const { sending, verifying, sentOn, masked: sentMasked, error, send, verify } =
    useMFAChallenge(api, onVerified);

  const [code, setCode] = useState('');
  const [backupCode, setBackupCode] = useState('');
  const [useBackup, setUseBackup] = useState(false);
  const [remember, setRemember] = useState(true);

  // Email is the channel we can actually deliver on without further user
  // interaction, so it is sent automatically. Phone requires the Firebase
  // handshake and is opt-in via its button.
  useEffect(() => {
    if (channels.includes('email')) {
      void send('email');
    }
    // Intentionally once, on mount: re-sending on every render would burn the
    // resend cooldown immediately.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const canSubmit = useBackup ? backupCode.trim().length > 0 : code.trim().length === 6;

  async function submit() {
    await verify(
      useBackup
        ? { backupCode: backupCode.trim(), rememberDevice: remember, deviceLabel, platform: Platform.OS }
        : {
            channel: sentOn ?? 'email',
            code: code.trim(),
            rememberDevice: remember,
            deviceLabel,
            platform: Platform.OS,
          }
    );
  }

  return (
    <Screen scroll>
      <View style={styles.body}>
        <Text style={styles.title}>Confirm it's you</Text>
        <Text style={styles.subtitle}>
          {useBackup
            ? 'Enter one of the backup codes you saved when you turned on two-factor.'
            : sentMasked
              ? `We sent a 6-digit code to ${sentMasked}.`
              : 'Choose where to send your verification code.'}
        </Text>

        {useBackup ? (
          <Input
            label="Backup code"
            value={backupCode}
            onChangeText={setBackupCode}
            autoCapitalize="characters"
            autoCorrect={false}
            placeholder="XXXX-XXXX-XXXX-XXXX"
            error={error ?? undefined}
          />
        ) : (
          <Input
            label="Verification code"
            value={code}
            onChangeText={(t) => setCode(t.replace(/[^0-9]/g, '').slice(0, 6))}
            keyboardType="number-pad"
            textContentType="oneTimeCode"
            autoComplete="one-time-code"
            placeholder="123456"
            maxLength={6}
            error={error ?? undefined}
          />
        )}

        <View style={styles.rememberRow}>
          <Button
            label={remember ? '☑  Remember this device' : '☐  Remember this device'}
            variant="ghost"
            onPress={() => setRemember((r) => !r)}
            accessibilityRole="checkbox"
            accessibilityState={{ checked: remember }}
          />
        </View>
        <Text style={styles.rememberHint}>
          {remember
            ? "You won't be asked again on this device until you remove it from Security settings."
            : "You'll be asked again next time you sign in."}
        </Text>

        <Button
          label="Verify"
          onPress={submit}
          disabled={!canSubmit}
          loading={verifying}
          fullWidth
          accentColor={accentColor}
        />

        {!useBackup && channels.includes('email') && (
          <Button
            label={sending ? 'Sending…' : 'Resend code'}
            variant="ghost"
            onPress={() => void send('email')}
            disabled={sending}
          />
        )}

        <Button
          label={useBackup ? 'Use a verification code instead' : "I can't access my email or phone"}
          variant="ghost"
          onPress={() => {
            setUseBackup((b) => !b);
            setCode('');
            setBackupCode('');
          }}
        />

        <View style={styles.footer}>
          <Button label="Sign out" variant="ghost" onPress={onSignOut} />
        </View>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: { paddingVertical: theme.spacing[6], gap: theme.spacing[3] },
  title: { fontSize: 24, fontWeight: '700', color: theme.colors.ink.DEFAULT, letterSpacing: -0.4 },
  subtitle: {
    fontSize: 15,
    lineHeight: 21,
    color: theme.colors.ink.soft,
    marginBottom: theme.spacing[2],
  },
  rememberRow: { alignItems: 'flex-start' },
  rememberHint: {
    fontSize: 13,
    lineHeight: 18,
    color: theme.colors.ink.soft,
    marginBottom: theme.spacing[2],
  },
  footer: { marginTop: theme.spacing[6], alignItems: 'center' },
});
