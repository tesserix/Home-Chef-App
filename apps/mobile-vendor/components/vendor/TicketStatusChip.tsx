import { StyleSheet, Text, View } from 'react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { STATUS_LABEL, type TicketStatus } from '../../hooks/useSupport';

// Toned status pill — tint background + same-hue text, never a solid fill
// (matches the admin-requests badge language). Tones per the tickets spec:
//   open                       → neutral grey
//   in_progress / waiting_*    → amber (work in flight)
//   resolved                   → success green
//   closed                     → muted grey
function chipTone(status: TicketStatus): { bg: string; fg: string } {
  switch (status) {
    case 'resolved':
      return { bg: theme.colors.success.tint, fg: theme.colors.success.soft };
    case 'in_progress':
    case 'waiting_on_customer':
    case 'waiting_on_chef':
      return { bg: theme.colors.amber.tint, fg: theme.colors.ink.DEFAULT };
    case 'closed':
      return { bg: theme.colors.mist.DEFAULT, fg: theme.colors.ink.muted };
    case 'open':
    default:
      return { bg: theme.colors.mist.DEFAULT, fg: theme.colors.ink.soft };
  }
}

export function TicketStatusChip({ status }: { status: TicketStatus }) {
  const tone = chipTone(status);
  return (
    <View style={[styles.chip, { backgroundColor: tone.bg }]}>
      <Text style={[styles.chipText, { color: tone.fg }]}>
        {STATUS_LABEL[status] ?? status}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  chip: {
    paddingHorizontal: theme.spacing[2],
    paddingVertical: 3,
    borderRadius: theme.radius.full,
    alignSelf: 'flex-start',
  },
  chipText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.2,
  },
});
