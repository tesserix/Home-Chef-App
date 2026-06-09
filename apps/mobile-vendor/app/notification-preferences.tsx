import { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { useToast } from '@homechef/mobile-shared/ui';
import { api } from '../lib/api';

// Matches the backend shape from GET /chef/notification-preferences (see
// apps/api/models/notification_preferences.go). Field names are the
// camelCase JSON keys the handler emits — not the Go struct names.
interface NotificationPreferences {
  newOrders: boolean;
  payouts: boolean;
  customerMessages: boolean;
  promo: boolean;
  quietHoursEnabled: boolean;
  quietHoursStart: string;
  quietHoursEnd: string;
  timezone: string;
}

const DEFAULT_PREFS: NotificationPreferences = {
  newOrders: true,
  payouts: true,
  customerMessages: true,
  promo: false,
  quietHoursEnabled: false,
  quietHoursStart: '22:00',
  quietHoursEnd: '07:00',
  timezone: 'Asia/Kolkata',
};

const HHMM_REGEX = /^([01][0-9]|2[0-3]):[0-5][0-9]$/;

function useNotificationPreferences() {
  return useQuery<NotificationPreferences>({
    queryKey: ['chef', 'notification-preferences'],
    queryFn: () =>
      api
        .get<NotificationPreferences>('/chef/notification-preferences')
        .then((r) => r.data),
    staleTime: 60_000,
  });
}

function useUpdateNotificationPreferences() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: Partial<NotificationPreferences>) =>
      api.put('/chef/notification-preferences', payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chef', 'notification-preferences'] });
    },
  });
}

interface ToggleRowProps {
  label: string;
  caption: string;
  value: boolean;
  onValueChange: (v: boolean) => void;
  hasBorderBottom?: boolean;
  disabled?: boolean;
}

function ToggleRow({
  label,
  caption,
  value,
  onValueChange,
  hasBorderBottom = true,
  disabled = false,
}: ToggleRowProps) {
  return (
    <View
      style={[styles.toggleRow, hasBorderBottom && styles.rowBorderBottom]}
    >
      <View style={styles.toggleText}>
        <Text style={styles.rowLabel}>{label}</Text>
        <Text style={styles.rowCaption}>{caption}</Text>
      </View>
      <Switch
        value={value}
        onValueChange={onValueChange}
        disabled={disabled}
        trackColor={{
          false: theme.colors.mist.strong,
          true: theme.colors.ink.DEFAULT,
        }}
        thumbColor={theme.colors.paper}
        ios_backgroundColor={theme.colors.mist.strong}
      />
    </View>
  );
}

export default function NotificationPreferencesScreen() {
  const { data, isLoading, isError } = useNotificationPreferences();
  const update = useUpdateNotificationPreferences();
  const { show: showToast } = useToast();

  const [prefs, setPrefs] = useState<NotificationPreferences>(DEFAULT_PREFS);

  useEffect(() => {
    if (!data) return;
    // Backend always returns a full payload (defaults fill missing rows)
    // so a partial spread isn't strictly needed, but defending against a
    // future field rename costs nothing.
    setPrefs({ ...DEFAULT_PREFS, ...data });
  }, [data]);

  function persist(next: Partial<NotificationPreferences>) {
    const merged = { ...prefs, ...next };
    setPrefs(merged);
    update.mutate(next, {
      onError: () => {
        // Roll back the optimistic update — we only mutated `next` so
        // restoring the previous slice from `prefs` is the safe undo.
        setPrefs(prefs);
        showToast({ message: 'Could not save. Try again.', tone: 'error' });
      },
    });
  }

  function handleQuietHoursInput(field: 'quietHoursStart' | 'quietHoursEnd', value: string) {
    // Cheap client-side guard — backend also validates. We persist
    // immediately only if the value parses; otherwise update local
    // state so the user can keep typing without losing characters.
    setPrefs({ ...prefs, [field]: value });
    if (HHMM_REGEX.test(value)) {
      persist({ [field]: value });
    }
  }

  if (isLoading) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <View style={styles.loadingContainer}>
          <ActivityIndicator size="large" color={theme.colors.ink.DEFAULT} />
        </View>
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <View style={styles.loadingContainer}>
          <Text style={styles.errorText}>Could not load preferences.</Text>
          <Pressable onPress={() => router.back()} style={styles.errorBack}>
            <Text style={styles.errorBackLabel}>Go back</Text>
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.commandBar}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          style={styles.backBtn}
          accessibilityRole="button"
          accessibilityLabel="Back"
        >
          <ChevronLeft size={26} color={theme.colors.ink.DEFAULT} strokeWidth={1.75} />
        </Pressable>
        <Text style={styles.commandTitle}>Notifications</Text>
      </View>

      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        keyboardShouldPersistTaps="handled"
      >
        <Text style={styles.sectionLabel}>CATEGORIES</Text>
        <View style={styles.sectionGroup}>
          <ToggleRow
            label="New orders"
            caption="Push when a customer places an order"
            value={prefs.newOrders}
            onValueChange={(v) => persist({ newOrders: v })}
          />
          <ToggleRow
            label="Payouts"
            caption="Push when a payout is sent to your bank"
            value={prefs.payouts}
            onValueChange={(v) => persist({ payouts: v })}
          />
          <ToggleRow
            label="Customer messages"
            caption="Push when a customer messages you about an order"
            value={prefs.customerMessages}
            onValueChange={(v) => persist({ customerMessages: v })}
          />
          <ToggleRow
            label="Promotions"
            caption="Tips, news, and platform updates from Home Chef"
            value={prefs.promo}
            onValueChange={(v) => persist({ promo: v })}
            hasBorderBottom={false}
          />
        </View>

        <Text style={styles.sectionLabel}>QUIET HOURS</Text>
        <View style={styles.sectionGroup}>
          <ToggleRow
            label="Mute non-critical pushes"
            caption="New-order pushes always come through. Payout, message, and promo pushes are silenced."
            value={prefs.quietHoursEnabled}
            onValueChange={(v) => persist({ quietHoursEnabled: v })}
            hasBorderBottom={prefs.quietHoursEnabled}
          />
          {prefs.quietHoursEnabled && (
            <View style={styles.quietHoursRow}>
              <View style={styles.quietHoursField}>
                <Text style={styles.quietLabel}>Start</Text>
                <TextInput
                  value={prefs.quietHoursStart}
                  onChangeText={(v) => handleQuietHoursInput('quietHoursStart', v)}
                  placeholder="22:00"
                  placeholderTextColor={theme.colors.ink.muted}
                  style={styles.quietInput}
                  keyboardType="numbers-and-punctuation"
                  maxLength={5}
                  autoCorrect={false}
                />
              </View>
              <View style={styles.quietHoursField}>
                <Text style={styles.quietLabel}>End</Text>
                <TextInput
                  value={prefs.quietHoursEnd}
                  onChangeText={(v) => handleQuietHoursInput('quietHoursEnd', v)}
                  placeholder="07:00"
                  placeholderTextColor={theme.colors.ink.muted}
                  style={styles.quietInput}
                  keyboardType="numbers-and-punctuation"
                  maxLength={5}
                  autoCorrect={false}
                />
              </View>
            </View>
          )}
        </View>

        <Text style={styles.helperFootnote}>
          Times are interpreted in your local timezone ({prefs.timezone}).
          Changes save automatically.
        </Text>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },

  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
    gap: theme.spacing[2],
  },
  backBtn: { marginRight: theme.spacing[1] },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },

  scroll: { flex: 1 },
  scrollContent: { paddingBottom: theme.spacing[10] },

  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[6],
    paddingBottom: theme.spacing[2],
  },

  sectionGroup: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  rowBorderBottom: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },

  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    minHeight: 44,
    gap: theme.spacing[4],
  },
  toggleText: { flex: 1 },

  rowLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  rowCaption: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: 2,
    lineHeight: 16,
  },

  quietHoursRow: {
    flexDirection: 'row',
    gap: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  quietHoursField: { flex: 1 },
  quietLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginBottom: 4,
  },
  quietInput: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    paddingVertical: 10,
    paddingHorizontal: 12,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.strong,
    borderRadius: 6,
    backgroundColor: theme.colors.paper,
  },

  helperFootnote: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    lineHeight: 16,
  },

  loadingContainer: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  errorText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[4],
  },
  errorBack: { paddingVertical: theme.spacing[2] },
  errorBackLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    textDecorationLine: 'underline',
  },
});
