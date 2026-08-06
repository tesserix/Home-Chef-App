import React, { useState } from 'react';
import { ActivityIndicator, Platform, Pressable, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import {
  ChevronRight,
  UtensilsCrossed,
  ScrollText,
  Wallet,
  Gift,
  Award,
  DatabaseZap,
  KeyRound,
  UserMinus,
  Bookmark,
  Receipt,
  Salad,
  LifeBuoy,
  MessageSquarePlus,
  ShieldOff,
  BookOpen,
} from 'lucide-react-native';
import { useProfile } from '../../hooks/useProfile';
import {
  CATERING_ENABLED,
  WALLET_ENABLED,
  REWARDS_ENABLED,
  REFERRAL_ENABLED,
} from '../../lib/features';
import { useAuthStore } from '../../store/auth-store';
import { customerColors } from '@homechef/mobile-shared/theme';
import { KeyboardAwareScrollView, useDialog } from '@homechef/mobile-shared/ui';
import { hasPasswordProvider } from '@homechef/mobile-shared/auth';
import { useDockClearance } from '../../components/navigation/Dock';
import { Alert } from 'react-native';
import { GuestGate } from '../../components/GuestGate';
import { useIsGuest } from '../../hooks/useRequireAccount';

// Profile — a HUB, not a form.
//
// This screen used to open with a name/phone form and a pair of preference
// pickers, which pushed every navigational destination below the fold: the
// things people come here to reach were the hardest things to find. The forms
// now live at /profile/edit and /profile/preferences, and what remains is
// identity, three high-traffic tiles, and quiet rows.

const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const DESTRUCTIVE_RIPPLE = `${customerColors.destructive.DEFAULT}14`;

interface NavRowProps {
  icon: React.ReactNode;
  label: string;
  onPress: () => void;
  /** Account-ending actions read in the destructive colour, as in the chef app. */
  destructive?: boolean;
}

function NavRow({ icon, label, onPress, destructive }: NavRowProps) {
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
          {/* Destructive rows get a tinted chip instead of the neutral grey
              every other row uses — the one place an icon carries meaning
              rather than just decoration, so "this ends something" is
              scannable before the label is even read. */}
          <View
            className={`w-9 h-9 rounded-full items-center justify-center mr-3 ${
              destructive ? 'bg-destructive-tint' : 'bg-surface-soft'
            }`}
          >
            {icon}
          </View>
          <Text
            className="flex-1 text-base font-sans"
            style={{
              color: destructive
                ? customerColors.destructive.DEFAULT
                : customerColors.charcoal.DEFAULT,
            }}
          >
            {label}
          </Text>
          <ChevronRight size={18} color={customerColors.charcoal.soft} />
        </View>
      )}
    </Pressable>
  );
}

function NavRowDivider() {
  return <View className="h-px bg-hairline ml-16" />;
}

/** Names a group of nav rows so the grouping reads as intentional rather
 *  than an accidental gap in the scroll. Sentence case, not all-caps —
 *  a label, not a shout. */
function SectionLabel({ children }: { children: string }) {
  return (
    <Text
      className="px-4 pb-1 pt-1 text-[13px] font-semibold font-sans"
      style={{ color: customerColors.charcoal.soft, letterSpacing: 0.1 }}
    >
      {children}
    </Text>
  );
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
  // App Review 5.1.1(iv): everything on this screen is tied to an identity,
  // so a guest gets a way in rather than a profile full of blanks.
  //
  // Wrapper around a separate body, not an early return above the body's hooks
  // — an auth flip while mounted would otherwise change this component's hook
  // count and crash ("Rendered fewer hooks"). See OrdersScreen.
  const isGuest = useIsGuest();
  if (isGuest) {
    return (
      <GuestGate
        icon={UserMinus}
        title="Your account"
        body="Sign in to manage addresses, payments, your wallet and your data."
      />
    );
  }
  return <ProfileScreenBody />;
}

function ProfileScreenBody() {
  const router = useRouter();
  const { data: profile, isLoading } = useProfile();
  const dockClearance = useDockClearance();

  // Only email/password accounts can change a password; SSO accounts have no
  // password credential, so that row is hidden for them.
  const [canChangePassword] = useState(() => hasPasswordProvider());
  const dialog = useDialog();

  function handleLogout() {
    dialog.confirm({
      title: 'Log out?',
      message: "You'll need to sign in again to place an order.",
      accentColor: customerColors.coral.DEFAULT,
      actions: [
        { label: 'Cancel', cancel: true },
        {
          label: 'Log out',
          destructive: true,
          onPress: () => {
            useAuthStore.getState().logout();
            router.replace('/(auth)/login');
          },
        },
      ],
    });
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
              {/* Without this the row reads as a static header and nobody
                  discovers that their personal details live behind it. */}
              <ChevronRight size={20} color={customerColors.charcoal.soft} />
            </View>
          )}
        </Pressable>

        {/* ── Three high-traffic destinations ── */}
        <View className="flex-row gap-3 px-4">
          <QuickTile
            icon={<Bookmark size={22} color={customerColors.charcoal.DEFAULT} />}
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
                  // Coral tint, not the same flat grey as the quick tiles above —
                  // this is the one nudge on the screen worth a beat of accent
                  // colour, and giving it its own tone is what lets it outrank
                  // three identical-looking utility tiles instead of blending in.
                  backgroundColor: customerColors.coral.tint,
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
                    Share Fe3dr with a friend and you both earn points
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
            v1 (wallet/rewards/referral/catering/tiffin) is hidden, and the
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
            {
              icon: <BookOpen size={18} color={customerColors.charcoal.soft} />,
              label: 'ChefBook',
              route: '/chefbook',
            },
            CATERING_ENABLED && {
              icon: <UtensilsCrossed size={18} color={customerColors.charcoal.soft} />,
              label: 'Catering',
              route: '/catering',
            },
            // "My meal plans" sat here. Removed: the Plans dock tab is one tap
            // from anywhere and is now the single home for the meal-plan list,
            // which was previously reachable from four places at once (this row,
            // the Plans tab, an Orders-tab segment and the /meal-plans route).
            // The route itself stays — deep links and the post-booking redirect
            // both land on it.
            // "My subscriptions" sat here (#696, as the fix for /subscriptions
            // being an orphan route). Removed with the rest of the recurring
            // tiffin UI in #1035 — with no way to subscribe, a management screen
            // for subscriptions has nothing to manage.
          ].filter(Boolean) as { icon: React.ReactNode; label: string; route: string }[];

          if (moreRows.length === 0) return null;

          return (
            <View className="mt-6">
              <SectionLabel>Preferences</SectionLabel>
              {moreRows.map((r, i) => (
                <View key={r.label}>
                  {i > 0 ? <NavRowDivider /> : null}
                  <NavRow
                    icon={r.icon}
                    label={r.label}
                    onPress={() => router.push(r.route as never)}
                  />
                </View>
              ))}
            </View>
          );
        })()}

        {/* ═══════════════════════════════════════════════════════════════════
            Section — Privacy & Legal. Two rows: "Your Data" stays top-level
            (DPDP action center — export/delete, functional not reference) and
            the four reference documents consolidate behind one "Legal" row
            (app/legal.tsx index).
        ═══════════════════════════════════════════════════════════════════ */}
        {/* A labelled section, not a bare grey gap — the gap alone read as an
            accidental seam in the scroll rather than a deliberate boundary
            between "things I configure" and "things I do to my account." */}
        <View className="mt-6">
          <SectionLabel>Support & legal</SectionLabel>
        </View>

        <NavRow
          icon={<DatabaseZap size={18} color={customerColors.charcoal.soft} />}
          label="Download my data"
          onPress={() => router.push('/data-privacy')}
        />
        <NavRowDivider />
        {/* App Review 1.2 asks for blocking to be reversible. The block itself
            is offered inline on the content (the ⋯ menu on a post or review),
            which is where people reach for it; this row is where they undo it. */}
        <NavRow
          icon={<ShieldOff size={18} color={customerColors.charcoal.soft} />}
          label="Blocked accounts"
          onPress={() => router.push('/blocked-accounts')}
        />
        <NavRowDivider />
        <NavRow
          icon={<LifeBuoy size={18} color={customerColors.charcoal.soft} />}
          label="Help & support"
          onPress={() => router.push('/support-chat')}
        />
        <NavRowDivider />
        <NavRow
          icon={<MessageSquarePlus size={18} color={customerColors.charcoal.soft} />}
          label="Send feedback or an idea"
          onPress={() => router.push('/feedback')}
        />
        <NavRowDivider />
        <NavRow
          icon={<ScrollText size={18} color={customerColors.charcoal.soft} />}
          label="Legal"
          onPress={() => router.push('/legal')}
        />

        {/* ── Account — password change, gated to email/password accounts.
            Google/Apple (SSO) accounts have no password credential, so this
            section is hidden for them (hasPasswordProvider). ── */}
        {canChangePassword ? (
          <View className="mt-6">
            <SectionLabel>Account</SectionLabel>
            <NavRow
              icon={<KeyRound size={18} color={customerColors.charcoal.soft} />}
              label="Change password"
              onPress={() => router.push('/(auth)/forgot-password' as never)}
            />
          </View>
        ) : null}

        {/* Apple 5.1.1(v) requires account deletion to be initiated IN the app,
            and a customer has to be able to FIND it. Pausing and deleting have
            always been on the data-privacy screen, but that screen was reachable
            only behind a row called "Your data" — accurate for the export half,
            and invisible to anyone looking to leave. This is the chef app's
            wording (settings.tsx), pointed at the screen we already have.

            No section label here, deliberately — the extra breathing room
            plus the destructive tint chip already mark this as the point of
            no return; a caption would just be more text to read on the way
            to leaving. */}
        <View className="mt-6">
          <NavRow
            icon={<UserMinus size={18} color={customerColors.destructive.DEFAULT} />}
            label="Pause or delete account"
            onPress={() => router.push('/data-privacy')}
            destructive
          />
        </View>

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

      {/* Branded confirmations, replacing the stock platform alert. */}
      {dialog.element}
    </SafeAreaView>
  );
}
