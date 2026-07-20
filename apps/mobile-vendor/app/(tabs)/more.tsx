import { Alert, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import {
  BarChart2,
  Bell,
  CalendarDays,
  ChefHat,
  Gauge,
  ChevronRight,
  DollarSign,
  FileText,
  Languages,
  LifeBuoy,
  LogOut,
  Scale,
  Settings,
  ShieldCheck,
  Star,
  User,
  XCircle,
} from 'lucide-react-native';
import { useTranslation } from 'react-i18next';
import { theme } from '@homechef/mobile-shared/theme';
import { useAuthStore } from '../../store/auth-store';
import { useDockClearance } from '../../components/navigation/Dock';
import { CATERING_ENABLED } from '../../constants/features';

interface NavRow {
  /** i18n key under the "more" namespace for the row label. */
  labelKey: string;
  caption?: string;
  route: string;
  Icon: typeof User;
}

interface NavSection {
  /** i18n key under "more.sections" for the ALL-CAPS group header. */
  titleKey: string;
  rows: NavRow[];
}

// Rows are grouped into labelled sections so the More tab reads as a small set
// of related groups rather than one long flat list. Feature-gated rows keep
// their entry here (filtered below) so a hidden feature stays type-checked and
// re-enabling is a one-line flag flip.
const ALL_SECTIONS: NavSection[] = [
  {
    titleKey: 'kitchen',
    rows: [
      { labelKey: 'mealPlans', caption: 'Weekly menu and tiffin requests', route: '/meal-plans', Icon: CalendarDays },
      { labelKey: 'capacity', caption: 'Daily caps and order cutoffs', route: '/capacity', Icon: Gauge },
      { labelKey: 'catering', caption: 'Event requests, quotes, bookings', route: '/catering', Icon: ChefHat },
      { labelKey: 'reviews', caption: 'Ratings and customer replies', route: '/reviews', Icon: Star },
    ],
  },
  {
    titleKey: 'money',
    rows: [
      { labelKey: 'earnings', caption: 'Payouts and transactions', route: '/earnings', Icon: DollarSign },
      { labelKey: 'analytics', caption: 'Orders, revenue, trends', route: '/analytics', Icon: BarChart2 },
    ],
  },
  {
    titleKey: 'requests',
    rows: [
      { labelKey: 'cancellations', caption: 'Confirm customer cancellations', route: '/cancel-requests', Icon: XCircle },
      { labelKey: 'adminRequests', caption: 'Verification and info requests', route: '/admin-requests', Icon: ShieldCheck },
      { labelKey: 'documents', caption: 'Renew or re-upload expired docs', route: '/documents/renew', Icon: FileText },
    ],
  },
  {
    titleKey: 'account',
    rows: [
      { labelKey: 'profile', caption: 'Name, kitchen details, cuisines', route: '/profile', Icon: User },
      { labelKey: 'notifications', caption: 'Categories and quiet hours', route: '/notification-preferences', Icon: Bell },
      { labelKey: 'language', caption: 'English · हिन्दी', route: '/language', Icon: Languages },
      { labelKey: 'settings', caption: 'Account, auto-accept, advanced', route: '/settings', Icon: Settings },
      { labelKey: 'support', caption: 'Report an issue or request a feature', route: '/support', Icon: LifeBuoy },
      // Legal docs are collapsed behind one row (rarely opened) — the /legal
      // hub screen lists Privacy, Terms, Chef agreement, and EULA.
      { labelKey: 'legal', caption: 'Privacy, terms, agreement, licence', route: '/legal', Icon: Scale },
    ],
  },
];

// Catering (#55) is built and working but hidden for now — see
// constants/features.ts. Filtering here (rather than deleting the row) keeps it
// type-checked; empty sections after filtering are dropped so no header floats
// over a blank card.
const NAV_SECTIONS: NavSection[] = ALL_SECTIONS.map((section) => ({
  ...section,
  rows: section.rows.filter(
    (row) => row.route !== '/catering' || CATERING_ENABLED,
  ),
})).filter((section) => section.rows.length > 0);

function deriveDisplayName(
  user: { name?: string; email?: string } | null | undefined,
): string {
  if (user?.name && user.name.trim().length > 0) return user.name.trim();
  const email = user?.email ?? '';
  if (email.includes('@')) {
    const local = email.split('@')[0]?.split('+')[0] ?? '';
    if (local.length > 0) {
      return local
        .split(/[._-]/)
        .filter(Boolean)
        .map((p) => p.charAt(0).toUpperCase() + p.slice(1))
        .join(' ');
    }
  }
  return 'Chef';
}

function deriveInitials(name: string): string {
  const parts = name.split(/\s+/).filter(Boolean);
  if (parts.length === 0) return 'C';
  if (parts.length === 1) return parts[0]!.slice(0, 2).toUpperCase();
  return ((parts[0]![0] ?? '') + (parts[parts.length - 1]![0] ?? '')).toUpperCase();
}

export default function MoreScreen() {
  const { t } = useTranslation();
  const { logout, user } = useAuthStore();
  const displayName = deriveDisplayName(
    user as { name?: string; email?: string } | null,
  );
  const initials = deriveInitials(displayName);
  const email = user?.email ?? '';
  const dockClearance = useDockClearance();

  function handleLogout() {
    Alert.alert('Log out?', 'You can sign back in any time.', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Log out',
        style: 'destructive',
        onPress: () => {
          logout();
          router.replace('/(auth)/login' as never);
        },
      },
    ]);
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <ScrollView
        contentContainerStyle={[
          styles.scrollContent,
          { paddingBottom: dockClearance },
        ]}
        showsVerticalScrollIndicator={false}
      >
        {/* Zone A — Command bar (matches dashboard/orders/menu) */}
        <View style={styles.commandBar}>
          <Text style={styles.commandTitle}>{t('more.title')}</Text>
        </View>

        {/* Identity block — white group card on the bone canvas (spec §1) */}
        <View style={[styles.card, styles.identityCard]}>
          <View style={styles.identityRow}>
            <View style={styles.avatar}>
              <Text style={styles.avatarLabel}>{initials}</Text>
            </View>
            <View style={styles.identityText}>
              <Text style={styles.identityName} numberOfLines={1}>
                {displayName}
              </Text>
              {email ? (
                <Text style={styles.identityEmail} numberOfLines={1}>
                  {email}
                </Text>
              ) : null}
            </View>
          </View>
        </View>

        {/* Grouped nav — one ALL-CAPS header + white group card per section,
            with inset hairline separators between rows (spec §1 / §9). */}
        {NAV_SECTIONS.map((section) => (
          <View key={section.titleKey}>
            <Text style={styles.sectionLabel}>
              {t(`more.sections.${section.titleKey}`).toUpperCase()}
            </Text>
            <View style={styles.card}>
              <View style={styles.cardInner}>
                {section.rows.map((row, index) => (
                  <NavRowItem
                    key={row.route}
                    row={row}
                    isLast={index === section.rows.length - 1}
                  />
                ))}
              </View>
            </View>
          </View>
        ))}

        {/* Log out — its own group card so it doesn't read as an orphaned
            text link. Differentiated as an action (not navigation) by
            destructive label tone, LogOut icon, no caption, no chevron.
            Matches the Delete-account row pattern in Settings. */}
        <View style={[styles.card, styles.logoutCard]}>
          <View style={styles.cardInner}>
            <Pressable onPress={handleLogout} accessibilityRole="button">
              {({ pressed }) => (
                // Inner-View pattern — keeps iOS Pressable flex layout intact.
                <View
                  style={[styles.navRow, pressed && styles.rowPressed]}
                >
                  <View style={styles.navIcon}>
                    <LogOut
                      size={20}
                      color={theme.colors.destructive.DEFAULT}
                      strokeWidth={1.75}
                    />
                  </View>
                  <View style={styles.navText}>
                    <Text style={styles.logoutLabel}>{t('more.logOut')}</Text>
                  </View>
                </View>
              )}
            </Pressable>
          </View>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

interface NavRowItemProps {
  row: NavRow;
  isLast: boolean;
}

function NavRowItem({ row, isLast }: NavRowItemProps) {
  const { t } = useTranslation();
  const { Icon, labelKey, caption, route } = row;
  return (
    <>
      <Pressable
        onPress={() => router.push(route as never)}
        accessibilityRole="button"
      >
        {({ pressed }) => (
          // Visual layer on an inner View. iOS Pressable with a
          // function-based `style` prop strips flexbox under some
          // conditions — same trick used by the dashboard status button.
          <View style={[styles.navRow, pressed && styles.rowPressed]}>
            <View style={styles.navIcon}>
              <Icon
                size={20}
                color={theme.colors.ink.soft}
                strokeWidth={1.75}
              />
            </View>
            <View style={styles.navText}>
              <Text style={styles.navLabel}>{t(`more.${labelKey}`)}</Text>
              {caption ? (
                <Text style={styles.navCaption}>{caption}</Text>
              ) : null}
            </View>
            <ChevronRight
              size={18}
              color={theme.colors.ink.muted}
              strokeWidth={1.75}
            />
          </View>
        )}
      </Pressable>
      {/* Inset hairline separator — aligned to the text, not edge-to-edge */}
      {!isLast ? <View style={styles.separator} /> : null}
    </>
  );
}

// 36pt icon circle (spec §9); separator inset lines up with the row text:
// row paddingHorizontal + circle width + row gap.
const ICON_CIRCLE = 36;
const SEPARATOR_INSET = theme.spacing[4] + ICON_CIRCLE + theme.spacing[3];

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  scrollContent: {
    paddingBottom: theme.spacing[10],
  },

  // Group cards — white surfaces on the bone canvas (spec §1).
  // Shadow lives on the outer card; `cardInner` clips pressed-row
  // backgrounds to the radius (iOS masksToBounds would kill the shadow
  // if `overflow: 'hidden'` sat on the shadowed view itself).
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
  separator: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginLeft: SEPARATOR_INSET,
  },

  // Section label — ALL-CAPS caption-spaced group header. paddingTop supplies
  // the inter-section rhythm, so group cards carry no bottom margin. Matches
  // the section headers in Settings.
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[6],
    paddingBottom: theme.spacing[2],
  },

  // Zone A — Command bar
  commandBar: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
  },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },

  // Identity card — sits above the first section label, which supplies the
  // gap below it (hence no bottom margin here).
  identityCard: {
    marginBottom: 0,
  },
  identityRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[4],
  },
  avatar: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: theme.colors.ink.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  avatarLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.paper,
    letterSpacing: 0.5,
  },
  identityText: { flex: 1 },
  identityName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  identityEmail: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    marginTop: 2,
  },

  // Rows inside group cards
  navRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[2],
    minHeight: 56,
  },
  navIcon: {
    width: ICON_CIRCLE,
    height: ICON_CIRCLE,
    borderRadius: theme.radius.full,
    backgroundColor: theme.colors.bone,
    alignItems: 'center',
    justifyContent: 'center',
  },
  navText: { flex: 1 },
  navLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  navCaption: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: 2,
  },

  // Logout — separate group card below the nav sections so it reads as a
  // distinct action surface, not a tail of the nav.
  logoutCard: {
    marginTop: theme.spacing[6],
  },
  logoutLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.destructive.DEFAULT,
  },
});
