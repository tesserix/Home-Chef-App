import { StyleSheet, Text, View } from 'react-native';
import { theme } from '@homechef/mobile-shared/theme';

interface DashboardStatsCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
}

/**
 * <DashboardStatsCard> — a single tile in the dashboard 2x2 grid.
 *
 * Uber-like: ink semibold numerals (was persimmon), hairline border (was
 * `shadow-sm`), tabular-nums so digits line up across cards. The card is
 * quiet; the value's weight does the work.
 */
export function DashboardStatsCard({ title, value, subtitle }: DashboardStatsCardProps) {
  return (
    <View style={styles.root}>
      <Text style={styles.title}>{title}</Text>
      <Text style={styles.value}>{value}</Text>
      {subtitle ? <Text style={styles.subtitle}>{subtitle}</Text> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    backgroundColor: theme.colors.paper,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[4],
  },
  title: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.8,
    textTransform: 'uppercase',
  },
  value: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    color: theme.colors.ink.DEFAULT,
    marginTop: theme.spacing[2],
    letterSpacing: -0.3,
    fontVariant: ['tabular-nums'],
  },
  subtitle: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[1],
  },
});
