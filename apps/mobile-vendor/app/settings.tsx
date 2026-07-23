import { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft, ChevronRight } from 'lucide-react-native';
import * as Notifications from 'expo-notifications';
import { theme } from '@homechef/mobile-shared/theme';
import { useToast } from '@homechef/mobile-shared/ui';
import { hasPasswordProvider } from '@homechef/mobile-shared/auth';
import { api } from '../lib/api';
import { useVendorPendingOrders } from '../hooks/useVendorOrders';

// ---- Types ------------------------------------------------------------------

// Matches the backend shape from GET /chef/settings. The earlier version of
// this file modelled a `notificationPrefs.newOrderNotifications` shape that
// the backend never returned — once `data` arrived from the API the screen
// would crash reading those undefined fields. Naming kept aligned with
// chefs.go so future edits stay grep-able across stacks.
interface NotificationPrefs {
  pushNewOrder: boolean;
  pushOrderUpdate: boolean;
  emailDailySummary: boolean;
  emailWeeklyReport: boolean;
  smsNewOrder: boolean;
}

interface ChefSettings {
  notifications: NotificationPrefs;
  autoAcceptOrders: boolean;
  autoAcceptThreshold: number;
  acceptingOrders: boolean;
}

const DEFAULT_NOTIFICATION_PREFS: NotificationPrefs = {
  pushNewOrder: true,
  pushOrderUpdate: true,
  emailDailySummary: true,
  emailWeeklyReport: true,
  smsNewOrder: false,
};

// ---- Hooks (co-located — no dedicated file exists yet) ----------------------

function useChefSettings() {
  return useQuery<ChefSettings>({
    queryKey: ['chef', 'settings'],
    queryFn: () => api.get<ChefSettings>('/chef/settings').then((r) => r.data),
    staleTime: 60_000,
  });
}

// PUT shape mirrors GET — the backend's UpdateChefSettings handler expects
// the full notifications object plus the autoAccept + acceptingOrders flags.
interface UpdateSettingsPayload {
  notifications?: NotificationPrefs;
  autoAcceptOrders?: boolean;
  autoAcceptThreshold?: number;
  acceptingOrders?: boolean;
}

function useUpdateSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: UpdateSettingsPayload) =>
      api.put('/chef/settings', payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef', 'settings'] });
    },
  });
}

// ---- Sub-components ---------------------------------------------------------

interface ToggleRowProps {
  label: string;
  caption: string;
  value: boolean;
  disabled?: boolean;
  onValueChange: (next: boolean) => void;
  hasBorderBottom?: boolean;
}

function ToggleRow({
  label,
  caption,
  value,
  disabled,
  onValueChange,
  hasBorderBottom = true,
}: ToggleRowProps) {
  return (
    <>
      <View style={styles.toggleRow}>
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
      {/* Inset hairline — skipped on the last row of a group card */}
      {hasBorderBottom ? <View style={styles.separator} /> : null}
    </>
  );
}

interface NavRowProps {
  label: string;
  onPress: () => void;
  destructive?: boolean;
  hasBorderBottom?: boolean;
}

function NavRow({
  label,
  onPress,
  destructive = false,
  hasBorderBottom = true,
}: NavRowProps) {
  return (
    <>
      <Pressable
        onPress={onPress}
        accessibilityRole="button"
        accessibilityLabel={label}
        android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
      >
        {({ pressed }) => (
          // Visual layer on the inner View — iOS Pressable with a
          // function-based `style` prop can strip flex/bg styles.
          <View
            style={[
              styles.navRow,
              pressed && Platform.OS === 'ios' && styles.rowPressed,
            ]}
          >
            <Text
              style={[styles.rowLabel, destructive && styles.destructiveLabel]}
            >
              {label}
            </Text>
            {!destructive && (
              <ChevronRight
                size={18}
                color={theme.colors.ink.muted}
                strokeWidth={1.75}
              />
            )}
          </View>
        )}
      </Pressable>
      {hasBorderBottom ? <View style={styles.separator} /> : null}
    </>
  );
}

// ---- Screen -----------------------------------------------------------------

// Schedules a local notification that mimics the FCM payload shape used by
// the backend's new-order push. Confirms (1) foreground alert handler is
// firing, (2) Android channel + iOS category are registered, (3) tap
// routing reads the orderId and lands on the detail screen. The Accept /
// Reject lock-screen actions exercise the SecureStore + fetch path.
async function triggerTestNotification(orderId: string | null): Promise<void> {
  // Honor permission state first — schedule on a denied permission silently
  // no-ops and the chef thinks the button is broken.
  const settings = await Notifications.getPermissionsAsync();
  if (settings.status !== 'granted') {
    Alert.alert(
      'Notifications blocked',
      'Enable notifications for Fe3dr Vendor in Settings to test.',
    );
    return;
  }

  await Notifications.scheduleNotificationAsync({
    content: {
      title: 'New order',
      body: orderId
        ? 'Tap to open the most recent pending order'
        : 'Tap to open the queue (no pending orders cached)',
      data: {
        type: 'new_order',
        // When no real pending orderId is available the detail screen will
        // show the "Couldn't load this order" fallback — itself useful for
        // testing the cold-cache deep-link path.
        orderId: orderId ?? 'test-no-cache',
      },
      sound: 'default',
      categoryIdentifier: Platform.OS === 'ios' ? 'new_order' : undefined,
    },
    trigger: {
      type: Notifications.SchedulableTriggerInputTypes.TIME_INTERVAL,
      seconds: 2,
      channelId: Platform.OS === 'android' ? 'new-orders' : undefined,
    },
  });
}

// Show the developer test row in any build pointed at localhost or a non-prod
// host. `__DEV__` alone is false under the Release-config local-sim profile
// so we fall back to the API URL — `vendors.fe3dr.com` (preview/production)
// hides the section, anything else (localhost, ngrok, internal staging) keeps
// it visible for debugging the push pipeline.
const SHOW_DEV_TOOLS =
  __DEV__ ||
  !(process.env.EXPO_PUBLIC_API_URL ?? '').includes('vendors.fe3dr.com');

export default function SettingsScreen() {
  const { data, isLoading, isError } = useChefSettings();
  const updateMutation = useUpdateSettings();
  const { data: pendingResp } = useVendorPendingOrders();
  const { show: showToast } = useToast();

  const [notifPrefs, setNotifPrefs] = useState<NotificationPrefs>(
    DEFAULT_NOTIFICATION_PREFS,
  );
  const [acceptingOrders, setAcceptingOrders] = useState(false);
  const [autoSchedule, setAutoSchedule] = useState(false);

  // Only email/password accounts can change a password. Google/Apple (SSO)
  // accounts have no password credential, so the "Change password" row is
  // hidden for them. Read once on mount — providerData is stable for the
  // session and the user is already authenticated on this screen.
  const [canChangePassword] = useState(() => hasPasswordProvider());

  // Auto open/close reads/writes the chef profile (not /chef/settings), via the
  // pointer-based PUT /chef/profile so it never disturbs other settings.
  const { data: profileData } = useQuery<{ autoScheduleEnabled?: boolean }>({
    queryKey: ['chef', 'profile', 'auto-schedule'],
    queryFn: () => api.get<{ autoScheduleEnabled?: boolean }>('/chef/profile').then((r) => r.data),
  });
  useEffect(() => {
    if (profileData) setAutoSchedule(profileData.autoScheduleEnabled ?? false);
  }, [profileData]);
  const autoScheduleMutation = useMutation({
    mutationFn: (value: boolean) => api.put('/chef/profile', { autoScheduleEnabled: value }),
  });
  function handleAutoScheduleToggle(value: boolean) {
    const previous = autoSchedule;
    setAutoSchedule(value);
    autoScheduleMutation.mutate(value, {
      onError: () => {
        setAutoSchedule(previous);
        showToast({ message: 'Could not update auto open/close. Try again.', tone: 'error' });
      },
    });
  }

  useEffect(() => {
    if (!data) return;
    // Defend against a partial payload — e.g. a half-migrated backend that
    // omits the notifications block. Without this guard the screen crashed
    // on `notifPrefs.pushNewOrder` after data arrived undefined.
    setNotifPrefs({
      ...DEFAULT_NOTIFICATION_PREFS,
      ...(data.notifications ?? {}),
    });
    setAcceptingOrders(data.acceptingOrders ?? false);
  }, [data]);

  function handleNotifToggle(key: keyof NotificationPrefs, value: boolean) {
    const previous = { ...notifPrefs };
    const updated = { ...notifPrefs, [key]: value };
    setNotifPrefs(updated);
    updateMutation.mutate(
      { notifications: updated },
      {
        onError: () => {
          // Roll back optimistic update on failure and surface error to chef.
          setNotifPrefs(previous);
          showToast({ message: 'Could not update preference. Try again.', tone: 'error' });
        },
      },
    );
  }

  function handleAcceptingOrdersToggle(value: boolean) {
    const previous = acceptingOrders;
    setAcceptingOrders(value);
    updateMutation.mutate(
      { acceptingOrders: value },
      {
        onError: () => {
          setAcceptingOrders(previous);
          showToast({ message: 'Could not update availability. Try again.', tone: 'error' });
        },
      },
    );
  }

  function handleDeleteAccount() {
    Alert.alert(
      'Delete account',
      'To delete your account, contact our support team. This action cannot be undone.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Contact support',
          onPress: () =>
            Alert.alert(
              'Support',
              'Email support@fe3dr.com to request account deletion.',
            ),
        },
      ],
    );
  }

  if (isLoading) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <View style={styles.commandBar}>
          <Pressable
            onPress={() => router.back()}
            hitSlop={8}
            accessibilityLabel="Go back"
            accessibilityRole="button"
            android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.backBtn,
                  pressed && Platform.OS === 'ios' && { opacity: 0.6 },
                ]}
              >
                <ChevronLeft
                  size={24}
                  color={theme.colors.ink.DEFAULT}
                  strokeWidth={1.75}
                />
              </View>
            )}
          </Pressable>
          <Text style={styles.commandTitle}>Settings</Text>
        </View>
        <View style={styles.loadingContainer}>
          <ActivityIndicator size="large" color={theme.colors.ink.DEFAULT} />
        </View>
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <View style={styles.commandBar}>
          <Pressable
            onPress={() => router.back()}
            hitSlop={8}
            accessibilityLabel="Go back"
            accessibilityRole="button"
            android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.backBtn,
                  pressed && Platform.OS === 'ios' && { opacity: 0.6 },
                ]}
              >
                <ChevronLeft
                  size={24}
                  color={theme.colors.ink.DEFAULT}
                  strokeWidth={1.75}
                />
              </View>
            )}
          </Pressable>
          <Text style={styles.commandTitle}>Settings</Text>
        </View>
        <View style={styles.loadingContainer}>
          <Text style={styles.errorText}>Failed to load settings</Text>
          <Pressable
            onPress={() => router.back()}
            hitSlop={8}
            accessibilityRole="button"
            accessibilityLabel="Go back"
            android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.errorBack,
                  pressed && Platform.OS === 'ios' && { opacity: 0.7 },
                ]}
              >
                <Text style={styles.errorBackLabel}>Go back</Text>
              </View>
            )}
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* Command bar — back chevron left, Geist-Bold 28pt title */}
      <View style={styles.commandBar}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityLabel="Go back"
          accessibilityRole="button"
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.backBtn,
                pressed && Platform.OS === 'ios' && { opacity: 0.6 },
              ]}
            >
              <ChevronLeft
                size={24}
                color={theme.colors.ink.DEFAULT}
                strokeWidth={1.75}
              />
            </View>
          )}
        </Pressable>
        <Text style={styles.commandTitle}>Settings</Text>
      </View>

      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
      >
        {/* NOTIFICATION PREFERENCES section. Labels match the backend
            channels the chef-settings endpoint actually persists; the
            previous "Payouts" / "Reviews" toggles weren't wired to any
            real backend field. */}
        <Text style={styles.sectionLabel}>NOTIFICATION PREFERENCES</Text>
        <View style={styles.card}>
          <View style={styles.cardInner}>
          <ToggleRow
            label="New orders"
            caption="Push notification when a new order arrives"
            value={notifPrefs.pushNewOrder}
            disabled={updateMutation.isPending}
            onValueChange={(v) => handleNotifToggle('pushNewOrder', v)}
          />
          <ToggleRow
            label="Order updates"
            caption="Push when an order changes status"
            value={notifPrefs.pushOrderUpdate}
            disabled={updateMutation.isPending}
            onValueChange={(v) => handleNotifToggle('pushOrderUpdate', v)}
          />
          <ToggleRow
            label="Daily summary email"
            caption="One email each evening with the day's orders"
            value={notifPrefs.emailDailySummary}
            disabled={updateMutation.isPending}
            onValueChange={(v) => handleNotifToggle('emailDailySummary', v)}
          />
          <ToggleRow
            label="Weekly summary email"
            caption="Weekly performance recap email"
            value={notifPrefs.emailWeeklyReport}
            disabled={updateMutation.isPending}
            onValueChange={(v) => handleNotifToggle('emailWeeklyReport', v)}
          />
          <ToggleRow
            label="SMS for new orders"
            caption="Text message in addition to the push"
            value={notifPrefs.smsNewOrder}
            disabled={updateMutation.isPending}
            onValueChange={(v) => handleNotifToggle('smsNewOrder', v)}
            hasBorderBottom={false}
          />
          </View>
        </View>

        {/* AVAILABILITY section
            The "Accepting Orders" toggle is retained here as a settings-level
            audit point — the dashboard's Open/Closed button is the operational
            shortcut for a running session, while this toggle persists the
            preference across sessions (e.g. set to Off at end of day and it
            stays off on next launch). Both write to the same backend field. */}
        <Text style={styles.sectionLabel}>AVAILABILITY</Text>
        <View style={styles.card}>
          <View style={styles.cardInner}>
            <ToggleRow
              label="Accepting orders"
              caption="Persists across sessions — use dashboard button for quick toggle"
              value={acceptingOrders}
              disabled={updateMutation.isPending || autoSchedule}
              onValueChange={handleAcceptingOrdersToggle}
              hasBorderBottom
            />
            <ToggleRow
              label="Auto open/close by hours"
              caption="Open and close your kitchen automatically to match your operating hours"
              value={autoSchedule}
              disabled={autoScheduleMutation.isPending}
              onValueChange={handleAutoScheduleToggle}
              hasBorderBottom={false}
            />
          </View>
        </View>

        {/* ACCOUNT section — navigation rows in a white group card */}
        <Text style={styles.sectionLabel}>ACCOUNT</Text>
        <View style={styles.card}>
          <View style={styles.cardInner}>
            {canChangePassword ? (
              <NavRow
                label="Change password"
                onPress={() => router.push('/(auth)/forgot-password' as never)}
              />
            ) : null}
            <NavRow
              label="Delete account"
              onPress={handleDeleteAccount}
              destructive
              hasBorderBottom={false}
            />
          </View>
        </View>

        {/* DEVELOPER section — local-sim + dev builds only. Lets us exercise
            the push pipeline without waiting for a real order to come in.
            Hidden in preview/production builds via SHOW_DEV_TOOLS gate. */}
        {SHOW_DEV_TOOLS ? (
          <>
            <Text style={styles.sectionLabel}>DEVELOPER</Text>
            <View style={styles.card}>
              <View style={styles.cardInner}>
                <NavRow
                  label="Send test notification"
                  onPress={() => {
                    const firstPending = pendingResp?.orders?.[0]?.id ?? null;
                    triggerTestNotification(firstPending).catch((err: unknown) => {
                      Alert.alert(
                        'Test notification failed',
                        err instanceof Error ? err.message : 'Unknown error',
                      );
                    });
                  }}
                  hasBorderBottom={false}
                />
              </View>
            </View>
          </>
        ) : null}
      </ScrollView>
    </SafeAreaView>
  );
}

// ---- Styles -----------------------------------------------------------------

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: theme.colors.bone,
  },

  // Command bar — matches dashboard/orders/more geometry
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
    gap: theme.spacing[2],
  },
  backBtn: {
    marginRight: theme.spacing[1],
  },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },

  // Scroll
  scroll: { flex: 1 },
  scrollContent: {
    paddingBottom: theme.spacing[10],
  },

  // Section label — ALL-CAPS caption-spaced, matches `IN PROGRESS` on dashboard
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[6],
    paddingBottom: theme.spacing[2],
  },

  // Group card — white surface on the bone canvas (spec §1). Shadow on
  // the outer card; `cardInner` clips pressed-row backgrounds to the
  // radius (overflow:hidden on the shadowed view would kill the shadow).
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    marginHorizontal: theme.spacing[4],
    ...theme.shadow[1],
  },
  cardInner: {
    borderRadius: theme.radius.lg,
    overflow: 'hidden',
  },
  rowPressed: {
    backgroundColor: theme.colors.bone,
  },
  // Inset hairline separator — aligned to the row text, not edge-to-edge
  separator: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginLeft: theme.spacing[4],
  },

  // Toggle row
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    minHeight: 44,
    gap: theme.spacing[4],
  },
  toggleText: {
    flex: 1,
  },

  // Nav row (chevron rows)
  navRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    minHeight: 44,
  },

  // Row text
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
  destructiveLabel: {
    color: theme.colors.destructive.DEFAULT,
    fontFamily: 'Inter',
  },

  // Loading / error states
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
  errorBack: {
    paddingVertical: theme.spacing[2],
  },
  errorBackLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
});
