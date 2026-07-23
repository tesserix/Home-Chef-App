import { Platform, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import {
  ChevronLeft,
  ChevronRight,
  FileText,
  Scale,
  ScrollText,
  Shield,
  User,
} from 'lucide-react-native';
import { useTranslation } from 'react-i18next';
import { theme } from '@homechef/mobile-shared/theme';

interface LegalRow {
  /** i18n key under the "more" namespace for the row label. */
  labelKey: string;
  caption: string;
  route: string;
  Icon: typeof User;
}

// Legal reference docs, collapsed here off the More tab (they're rarely opened).
// Each row deep-links to an existing standalone LegalScreen route.
const LEGAL_ROWS: LegalRow[] = [
  { labelKey: 'legalPrivacy', caption: 'How we handle your data', route: '/privacy', Icon: Shield },
  { labelKey: 'legalTerms', caption: 'Using the vendor app', route: '/terms', Icon: FileText },
  { labelKey: 'legalAgreement', caption: 'Commission, payouts, food safety', route: '/chef-agreement', Icon: Scale },
  { labelKey: 'legalEula', caption: 'App licence terms', route: '/eula', Icon: ScrollText },
];

export default function LegalScreen() {
  const { t } = useTranslation();

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* Zone A — Command bar: back chevron + title on the bone canvas */}
      <View style={styles.commandBar}>
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={8}
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
        >
          {({ pressed }) => (
            // Inner-View pattern — visual styles on the View, never a
            // function-style array on the Pressable (iOS drops them).
            <View
              style={[
                styles.backButton,
                pressed && Platform.OS === 'ios' && { opacity: 0.6 },
              ]}
            >
              <ChevronLeft
                size={22}
                color={theme.colors.ink.DEFAULT}
                strokeWidth={2}
              />
            </View>
          )}
        </Pressable>
        <Text style={styles.commandTitle}>{t('more.legal')}</Text>
      </View>

      <ScrollView
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
      >
        <View style={styles.card}>
          <View style={styles.cardInner}>
            {LEGAL_ROWS.map((row, index) => (
              <LegalRowItem
                key={row.route}
                row={row}
                isLast={index === LEGAL_ROWS.length - 1}
              />
            ))}
          </View>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

interface LegalRowItemProps {
  row: LegalRow;
  isLast: boolean;
}

function LegalRowItem({ row, isLast }: LegalRowItemProps) {
  const { t } = useTranslation();
  const { Icon, labelKey, caption, route } = row;
  return (
    <>
      <Pressable
        onPress={() => router.push(route as never)}
        accessibilityRole="button"
        accessibilityLabel={t(`more.${labelKey}`)}
        android_ripple={{ color: `${theme.colors.ink.DEFAULT}0F`, borderless: false }}
      >
        {({ pressed }) => (
          <View
            style={[styles.navRow, pressed && Platform.OS === 'ios' && styles.rowPressed]}
          >
            <View style={styles.navIcon}>
              <Icon size={20} color={theme.colors.ink.soft} strokeWidth={1.75} />
            </View>
            <View style={styles.navText}>
              <Text style={styles.navLabel}>{t(`more.${labelKey}`)}</Text>
              <Text style={styles.navCaption}>{caption}</Text>
            </View>
            <ChevronRight
              size={18}
              color={theme.colors.ink.muted}
              strokeWidth={1.75}
            />
          </View>
        )}
      </Pressable>
      {!isLast ? <View style={styles.separator} /> : null}
    </>
  );
}

const ICON_CIRCLE = 36;
const SEPARATOR_INSET = theme.spacing[4] + ICON_CIRCLE + theme.spacing[3];

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  scrollContent: {
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[10],
  },

  // Command bar — back chevron + title
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
    gap: theme.spacing[3],
  },
  backButton: {
    minWidth: 28,
    minHeight: 44,
    justifyContent: 'center',
  },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
    flex: 1,
  },

  // Group card — white surface on the bone canvas (matches More).
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
});
