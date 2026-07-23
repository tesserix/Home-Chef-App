import { useEffect, useState } from 'react';
import { Linking, Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import { useLocalSearchParams } from 'expo-router';
import { theme } from '@homechef/mobile-shared/theme';
import { appVersion } from '../lib/app-version';

// Hard wall — shown when the running app is below the backend's
// MIN_VERSION_VENDOR_<PLATFORM>. No back gesture escapes this screen;
// the root layout pins it via router.replace whenever the min-version
// check flips. Mounted as a regular Expo Router route so deep links
// (e.g. from a push notification) bypass it correctly when the version
// is current again — _layout.tsx is the authoritative gate.
export default function UpgradeRequiredScreen() {
  const params = useLocalSearchParams<{ minVersion?: string; storeUrl?: string }>();
  const minVersion = typeof params.minVersion === 'string' ? params.minVersion : null;
  const storeUrl = typeof params.storeUrl === 'string' ? params.storeUrl : null;

  const [opening, setOpening] = useState(false);

  // Pre-warm Linking.canOpenURL so the CTA reads as enabled or
  // disabled without a flash of state mid-tap.
  const [canOpen, setCanOpen] = useState(false);
  useEffect(() => {
    if (!storeUrl) return;
    Linking.canOpenURL(storeUrl)
      .then(setCanOpen)
      .catch(() => setCanOpen(false));
  }, [storeUrl]);

  async function openStore(): Promise<void> {
    if (!storeUrl) return;
    setOpening(true);
    try {
      await Linking.openURL(storeUrl);
    } catch {
      // Silently swallow — Linking failure usually means user backs out
      // of the system prompt or the URL is malformed. The wall stays up
      // either way; nothing more we can do here.
    } finally {
      setOpening(false);
    }
  }

  return (
    <View style={styles.container}>
      <Text style={styles.label}>HOME CHEF · VENDOR</Text>
      <Text style={styles.title}>Update required</Text>
      <Text style={styles.body}>
        This version of the app is no longer supported. Update from the App Store
        to keep accepting orders.
      </Text>
      <View style={styles.metaBlock}>
        <Text style={styles.metaLabel}>You're on</Text>
        <Text style={styles.metaValue}>{appVersion}</Text>
        {minVersion && (
          <>
            <Text style={[styles.metaLabel, styles.metaSpacer]}>Required</Text>
            <Text style={styles.metaValue}>{minVersion} or newer</Text>
          </>
        )}
      </View>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel="Open App Store to update"
        onPress={openStore}
        disabled={!storeUrl || !canOpen || opening}
        android_ripple={
          !storeUrl || !canOpen
            ? undefined
            : { color: `${theme.colors.paper}30`, borderless: false }
        }
      >
        {({ pressed }) => (
          <View
            style={[
              styles.cta,
              {
                opacity:
                  !storeUrl || !canOpen ? 0.5 : pressed && Platform.OS === 'ios' ? 0.85 : 1,
              },
            ]}
          >
            <Text style={styles.ctaText}>
              {opening ? 'Opening App Store…' : 'Update now'}
            </Text>
          </View>
        )}
      </Pressable>
      {!storeUrl && (
        <Text style={styles.hint}>
          Search "Fe3dr Vendor" in the App Store.
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: theme.colors.bone,
    paddingHorizontal: 24,
    paddingTop: 96,
    paddingBottom: 32,
  },
  label: {
    fontFamily: 'Inter-Medium',
    fontSize: 11,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    marginBottom: 24,
  },
  title: {
    fontFamily: 'Geist-Bold',
    fontSize: 32,
    lineHeight: 38,
    color: theme.colors.ink.DEFAULT,
    marginBottom: 12,
  },
  body: {
    fontFamily: 'Inter',
    fontSize: 15,
    lineHeight: 22,
    color: theme.colors.ink.soft,
    marginBottom: 32,
  },
  metaBlock: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.strong,
    paddingVertical: 18,
    marginBottom: 32,
  },
  metaLabel: {
    fontFamily: 'Inter-Medium',
    fontSize: 11,
    letterSpacing: 1.2,
    color: theme.colors.ink.muted,
  },
  metaValue: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: theme.colors.ink.DEFAULT,
    marginTop: 4,
  },
  metaSpacer: { marginTop: 14 },
  cta: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    paddingVertical: 18,
    alignItems: 'center',
    justifyContent: 'center',
  },
  ctaText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: theme.colors.paper,
    letterSpacing: 0.2,
  },
  hint: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: theme.colors.ink.muted,
    marginTop: 16,
    textAlign: 'center',
  },
});
