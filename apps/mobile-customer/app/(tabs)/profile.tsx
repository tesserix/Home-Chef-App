import React, { useState } from 'react';
import { ActivityIndicator, Platform, Pressable, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import {
  CalendarDays,
  RefreshCw,
  ChevronRight,
  MessageSquare,
  UtensilsCrossed,
  ScrollText,
  Wallet,
  Gift,
  Award,
  DatabaseZap,
  KeyRound,
  Heart,
  Receipt,
  Salad,
} from 'lucide-react-native';
import { useProfile } from '../../hooks/useProfile';
import {
  TIFFIN_ENABLED,
  CATERING_ENABLED,
  WALLET_ENABLED,
  REWARDS_ENABLED,
  REFERRAL_ENABLED,
  SOCIAL_ENABLED,
} from '../../lib/features';
import { useAuthStore } from '../../store/auth-store';
import { customerColors } from '@homechef/mobile-shared/theme';
import { KeyboardAwareScrollView } from '@homechef/mobile-shared/ui';
import { hasPasswordProvider } from '@homechef/mobile-shared/auth';
import { useDockClearance } from '../../components/navigation/Dock';
import { Alert } from 'react-native';

// Profile — a HUB, not a form.
//
// This screen used to open with a name/phone form and a pair of preference
// pickers, which pushed every navigational destination below the fold: the
// things people come here to reach were the hardest things to find. The forms
// now live at /profile/edit and /profile/preferences, and what remains is
// identity, three high-traffic tiles, and quiet rows.

const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const DESTRUCTIVE_RIPPLE = `${customerColors.destructive.DEFAULT}14`;

function SectionLabel({ children }: { children: string }) {
  return (
    <Text className="text-xs font-semibold text-charcoal-soft px-4 pt-5 pb-2">{children}</Text>
  );
}

interface NavRowProps {
  icon: React.ReactNode;
  label: string;
  onPress: () => void;
  isLast?: boolean;
}

function NavRow({ icon, label, onPress }: NavRowProps) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={label}
      android_ripple={{ color: ROW_RIPPLE, borderless: false }}
    >
      {({ pressed }) => (
        <View
          className={`flex-row items-center px-4 py-3 min-h-[52px] ${pressed ? 'bg-surface-soft' : 'bg-canvas'}`}
        >
          <View className="w-9 h-9 rounded-full bg-surface-soft items-center justify-center mr-3">
            {icon}
          </View>
          <Text className="flex-1 text-base text-charcoal font-sans">{label}</Text>
          <ChevronRight size={18} color={customerColors.charcoal.soft} />
        </View>
      )}
    </Pressable>
  );
}

function NavRowDivider() {
  return <View className="h-px bg-hairline ml-16" />;
}

/** One of the three high-traffic destinations at the top of the hub. */
function QuickTile({
  icon,
  label,
  onPress,
}: {
  icon: React.ReactNode;
  label: string;
  onPress: () => void;
}) {
  return (
    <Pressable
      onPress={onPress}
      className="flex-1"
      accessibilityRole="button"
      accessibilityLabel={label}
      android_ripple={{ color: ROW_RIPPLE, borderless: false }}
    >
      {({ pressed }) => (
        <View
          className="min-h-[88px] items-center justify-center gap-2 rounded-xl"
          style={{
            backgroundColor: customerColors.surface.soft,
            opacity: pressed && Platform.OS === 'ios' ? 0.7 : 1,
          }}
        >
          {icon}
          <Text className="text-[14px] font-medium" style={{ color: customerColors.charcoal.DEFAULT }}>
            {label}
          </Text>
        </View>
      )}
    </Pressable>
  );
}

export default function ProfileScreen() {
  const router = useRouter();
  const { data: profile, isLoading } = useProfile();
  const dockClearance = useDockClearance();

  // Only email/password accounts can change a password; SSO accounts have no
  // password credential, so that row is hidden for them.
  const [canChangePassword] = useState(() => hasPasswordProvider());

  function handleLogout() {
    Alert.alert('Log out', 'Are you sure you want to log out?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Log Out',
        style: 'destructive',
        onPress: () => {
          useAuthStore.getState().logout();
          router.replace('/(auth)/login');
        },
      },
    ]);
  }

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color={customerColors.coral.DEFAULT} />
        </View>
      </SafeAreaView>
    );
  }

  const fullName = [profile?.firstName, profile?.lastName].filter(Boolean).join(' ');
  const initials =
    [profile?.firstName?.[0], profile?.lastName?.[0]].filter(Boolean).join('').toUpperCase() || '?';

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      <KeyboardAwareScrollView contentContainerStyle={{ paddingBottom: dockClearance }}>
        {/* ── Identity — the name IS the way into editing, so the form needs no
            row of its own. The monogram stays charcoal: identity is not a call
            to action, and coral is reserved for actions and selection. ── */}
        <Pressable
          onPress={() => router.push('/profile/edit' as never)}
          accessibilityRole="button"
          accessibilityLabel="Edit profile"
          android_ripple={{ color: ROW_RIPPLE, borderless: false }}
        >
          {({ pressed }) => (
            <View
              className="flex-row items-center gap-4 px-4 pb-5 pt-4"
              style={{ opacity: pressed && Platform.OS === 'ios' ? 0.7 : 1 }}
            >
              <View className="flex-1">
                <Text
                  className="text-[30px] font-bold font-display"
                  style={{ color: customerColors.charcoal.DEFAULT, letterSpacing: -0.5 }}
                  numberOfLines={1}
                >
                  {fullName || 'Your profile'}
                </Text>
                <Text className="mt-0.5 text-sm" style={{ color: customerColors.charcoal.soft }}>
                  {profile?.email ?? ''}
                </Text>
              </View>
              <View
                className="h-16 w-16 items-center justify-center rounded-full"
                style={{ backgroundColor: customerColors.charcoal.DEFAULT }}
              >
                <Text
                  className="text-[22px] font-bold font-display"
                  style={{ color: customerColors.canvas, lineHeight: 28 }}
                >
                  {initials}
                </Text>
              </View>
            </View>
          )}
        </Pressable>

        {/* ── Three high-traffic destinations ── */}
        <View className="flex-row gap-3 px-4">
          <QuickTile
            icon={<Heart size={22} color={customerColors.charcoal.DEFAULT} />}
            label="Saved"
            onPress={() => router.push('/(tabs)/favorites' as never)}
          />
          {WALLET_ENABLED ? (
            <QuickTile
              icon={<Wallet size={22} color={customerColors.charcoal.DEFAULT} />}
              label="Wallet"
              onPress={() => router.push('/wallet')}
            />
          ) : null}
          <QuickTile
            icon={<Receipt size={22} color={customerColors.charcoal.DEFAULT} />}
            label="Orders"
            onPress={() => router.push('/(tabs)/orders' as never)}
          />
        </View>

        {/* ── Referral nudge ── */}
        {REFERRAL_ENABLED ? (
          <Pressable
            onPress={() => router.push('/referral')}
            accessibilityRole="button"
            accessibilityLabel="Refer and earn"
            android_ripple={{ color: ROW_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                className="mx-4 mt-4 flex-row items-center gap-4 rounded-xl px-4 py-4"
                style={{
                  backgroundColor: customerColors.surface.soft,
                  opacity: pressed && Platform.OS === 'ios' ? 0.7 : 1,
                }}
              >
                <View className="flex-1">
                  <Text
                    className="text-[17px] font-semibold"
                    style={{ color: customerColors.charcoal.DEFAULT }}
                  >
                    Refer &amp; earn
                  </Text>
                  <Text className="mt-0.5 text-sm" style={{ color: customerColors.charcoal.soft }}>
                    Share Fe3dr with a friend and you both get credit
                  </Text>
                </View>
                <Gift size={28} color={customerColors.coral.DEFAULT} />
              </View>
            )}
          </Pressable>
        ) : null}

        {/* ═══════════════════════════════════════════════════════════════════
            Section — More (iOS grouped nav rows)
        ═══════════════════════════════════════════════════════════════════ */}
        {/* MORE nav rows are each gated by a feature flag — everything deferred for
            v1 (wallet/rewards/referral/social/catering/tiffin) is hidden, and the
            whole section drops out when no row is enabled. Flip the flag in
            lib/features.ts (+ any backend flag) to bring a row back. */}
        {(() => {
          const moreRows = [
            // Food preferences and the dietary profile moved off this screen so
            // it could be a hub; this row is how they stay reachable.
            {
              icon: <Salad size={18} color={customerColors.charcoal.soft} />,
              label: 'Food preferences',
              route: '/profile/preferences',
            },
            // Wallet is a tile above and Refer & Earn is the nudge card, so
            // neither repeats here.
            REWARDS_ENABLED && {
              icon: <Award size={18} color={customerColors.charcoal.soft} />,
              label: 'Rewards',
              route: '/loyalty',
            },
            SOCIAL_ENABLED && {
              icon: <MessageSquare size={18} color={customerColors.charcoal.soft} />,
              label: 'Social Feed',
              route: '/social',
            },
            CATERING_ENABLED && {
              icon: <UtensilsCrossed size={18} color={customerColors.charcoal.soft} />,
              label: 'Catering',
              route: '/catering',
            },
            TIFFIN_ENABLED && {
              icon: <CalendarDays size={18} color={customerColors.charcoal.soft} />,
              label: 'My meal plans',
              route: '/meal-plans',
            },
            // /subscriptions was an ORPHAN route (#696): the only screen that can
            // pause or cancel a RECURRING charge, reachable solely via a one-shot
            // "View" button in the alert shown right after subscribing. Dismiss that
            // alert and the customer could never find it again — they'd have to call
            // support or charge back to stop being billed. This row is the whole fix;
            // the screen and its API already work.
            TIFFIN_ENABLED && {
              icon: <RefreshCw size={18} color={customerColors.charcoal.soft} />,
              label: 'My subscriptions',
              route: '/subscriptions',
            },
          ].filter(Boolean) as { icon: React.ReactNode; label: string; route: string }[];

          if (moreRows.length === 0) return null;

          return (
            <>
              <SectionLabel>More</SectionLabel>
              {/* Shadow on outer View, overflow+radius on clip View — iOS shadow gotcha */}
              <View
                className="mx-4"
                style={{
                  shadowColor: customerColors.charcoal.DEFAULT,
                  shadowOffset: { width: 0, height: 1 },
                  shadowOpacity: 0.06,
                  shadowRadius: 4,
                  elevation: 2,
                }}
              >
                <View className="rounded-xl overflow-hidden">
                  {moreRows.map((r, i) => (
                    <View key={r.label}>
                      {i > 0 ? <NavRowDivider /> : null}
                      <NavRow
                        icon={r.icon}
                        label={r.label}
                        onPress={() => router.push(r.route as never)}
                        isLast={i === moreRows.length - 1}
                      />
                    </View>
                  ))}
                </View>
              </View>
            </>
          );
        })()}

        {/* ═══════════════════════════════════════════════════════════════════
            Section — Privacy & Legal. Two rows: "Your Data" stays top-level
            (DPDP action center — export/delete, functional not reference) and
            the four reference documents consolidate behind one "Legal" row
            (app/legal.tsx index).
        ═══════════════════════════════════════════════════════════════════ */}
        <SectionLabel>Privacy & Legal</SectionLabel>

        {/* Shadow on outer View, overflow+radius on clip View — iOS shadow gotcha */}
        <View
          className="mx-4"
          style={{
            shadowColor: customerColors.charcoal.DEFAULT,
            shadowOffset: { width: 0, height: 1 },
            shadowOpacity: 0.06,
            shadowRadius: 4,
            elevation: 2,
          }}
        >
          <View className="rounded-xl overflow-hidden">
            <NavRow
              icon={<DatabaseZap size={18} color={customerColors.charcoal.soft} />}
              label="Your Data"
              onPress={() => router.push('/data-privacy')}
            />
            <NavRowDivider />
            <NavRow
              icon={<ScrollText size={18} color={customerColors.charcoal.soft} />}
              label="Legal"
              onPress={() => router.push('/legal')}
              isLast
            />
          </View>
        </View>

        {/* ── Account — password change, gated to email/password accounts.
            Google/Apple (SSO) accounts have no password credential, so this
            section is hidden for them (hasPasswordProvider). ── */}
        {canChangePassword ? (
          <>
            <SectionLabel>Account</SectionLabel>
            <View
              className="mx-4"
              style={{
                shadowColor: customerColors.charcoal.DEFAULT,
                shadowOffset: { width: 0, height: 1 },
                shadowOpacity: 0.06,
                shadowRadius: 4,
                elevation: 2,
              }}
            >
              <View className="rounded-xl overflow-hidden">
                <NavRow
                  icon={<KeyRound size={18} color={customerColors.charcoal.soft} />}
                  label="Change password"
                  onPress={() => router.push('/(auth)/forgot-password' as never)}
                  isLast
                />
              </View>
            </View>
          </>
        ) : null}

        {/* ── Logout — destructive action ── */}
        <Pressable
          onPress={handleLogout}
          accessibilityRole="button"
          accessibilityLabel="Log out"
          android_ripple={{ color: DESTRUCTIVE_RIPPLE, borderless: false }}
        >
          {({ pressed }) => (
            <View
              className={`mx-4 mt-6 rounded-lg min-h-[52px] items-center justify-center border border-hairline ${
                pressed ? 'bg-surface-soft' : 'bg-canvas'
              }`}
            >
              <Text className="text-base font-semibold text-destructive">
                Log Out
              </Text>
            </View>
          )}
        </Pressable>

      </KeyboardAwareScrollView>
    </SafeAreaView>
  );
}
