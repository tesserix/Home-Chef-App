import {
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft, ChevronRight, Plus } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { EmptyState, Skeleton } from '@homechef/mobile-shared/ui';
import {
  CATEGORY_LABEL,
  useSupportTickets,
  type SupportTicket,
} from '../../hooks/useSupport';
import { TicketStatusChip } from '../../components/vendor/TicketStatusChip';

// Compact relative date — "2h ago", "3d ago", or a short date past a week.
function relativeDate(iso: string): string {
  try {
    const then = new Date(iso).getTime();
    const diffMs = Date.now() - then;
    const min = Math.floor(diffMs / 60000);
    if (min < 1) return 'just now';
    if (min < 60) return `${min}m ago`;
    const hr = Math.floor(min / 60);
    if (hr < 24) return `${hr}h ago`;
    const day = Math.floor(hr / 24);
    if (day < 7) return `${day}d ago`;
    return new Date(iso).toLocaleDateString('en-IN', {
      day: 'numeric',
      month: 'short',
    });
  } catch {
    return '';
  }
}

function isUrgent(t: SupportTicket): boolean {
  return t.priority === 'urgent' || t.priority === 'high';
}

function TicketRow({ ticket }: { ticket: SupportTicket }) {
  return (
    <Pressable
      onPress={() => router.push(`/support/${ticket.id}`)}
      accessibilityRole="button"
      accessibilityLabel={`Open ticket: ${ticket.subject}`}
      android_ripple={{ color: `${theme.colors.ink.DEFAULT}0F`, borderless: false }}
    >
      {({ pressed }) => (
        <View
          style={[
            styles.card,
            pressed && Platform.OS === 'ios' && styles.cardPressed,
          ]}
        >
          <View style={styles.cardMain}>
            <View style={styles.subjectRow}>
              {isUrgent(ticket) ? <View style={styles.urgentDot} /> : null}
              <Text style={styles.subject} numberOfLines={1}>
                {ticket.subject}
              </Text>
            </View>
            <View style={styles.metaRow}>
              <TicketStatusChip status={ticket.status} />
              <Text style={styles.meta} numberOfLines={1}>
                {CATEGORY_LABEL[ticket.category] ?? ticket.category}
              </Text>
            </View>
            <Text style={styles.subMeta} numberOfLines={1}>
              {ticket.ticketNumber} · {relativeDate(ticket.createdAt)}
            </Text>
          </View>
          <ChevronRight
            size={18}
            color={theme.colors.ink.muted}
            strokeWidth={1.75}
          />
        </View>
      )}
    </Pressable>
  );
}

export default function SupportTicketsScreen() {
  const { data, isLoading, isError, refetch } = useSupportTickets();
  const tickets = data ?? [];

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.commandBar}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Back"
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
                size={26}
                color={theme.colors.ink.DEFAULT}
                strokeWidth={1.75}
              />
            </View>
          )}
        </Pressable>
        <Text style={styles.commandTitle}>Support</Text>
      </View>

      {isLoading ? (
        <View style={styles.skeletonStack}>
          <Skeleton height={90} style={{ borderRadius: theme.radius.lg, marginBottom: theme.spacing[3] }} />
          <Skeleton height={90} style={{ borderRadius: theme.radius.lg, marginBottom: theme.spacing[3] }} />
          <Skeleton height={90} style={{ borderRadius: theme.radius.lg }} />
        </View>
      ) : isError ? (
        // Shared triad primitive — consistent with the rest of the app's
        // list screens (cancel-requests, admin-requests, reviews, etc.).
        <EmptyState
          title="Couldn't load your tickets"
          body="Check your connection and try again."
          ctaLabel="Retry"
          onCtaPress={() => refetch()}
        />
      ) : tickets.length === 0 ? (
        <EmptyState
          title="No tickets yet"
          body="Raise an issue or request a feature — tap New ticket to start."
          ctaLabel="New ticket"
          onCtaPress={() => router.push('/support/new')}
        />
      ) : (
        <ScrollView
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
          showsVerticalScrollIndicator={false}
        >
          {tickets.map((t) => (
            <TicketRow key={t.id} ticket={t} />
          ))}
        </ScrollView>
      )}

      {/* Persistent New-ticket action when the list is non-empty. */}
      {!isLoading && !isError && tickets.length > 0 ? (
        <View style={styles.footer}>
          <Pressable
            onPress={() => router.push('/support/new')}
            accessibilityRole="button"
            accessibilityLabel="New ticket"
            android_ripple={{ color: `${theme.colors.paper}30`, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.primaryBtn,
                  pressed && Platform.OS === 'ios' && styles.btnPressed,
                ]}
              >
                <Plus size={18} color={theme.colors.paper} strokeWidth={2} />
                <Text style={styles.primaryLabel}>New ticket</Text>
              </View>
            )}
          </Pressable>
        </View>
      ) : null}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
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
  scrollContent: {
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[10],
  },
  skeletonStack: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[1],
  },

  card: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    marginBottom: theme.spacing[3],
    ...theme.shadow[1],
  },
  cardPressed: { backgroundColor: theme.colors.bone },
  cardMain: { flex: 1, gap: theme.spacing[2] },
  subjectRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
  },
  urgentDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: theme.colors.destructive.DEFAULT,
  },
  subject: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  metaRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
  },
  meta: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  subMeta: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    fontVariant: ['tabular-nums'],
  },

  footer: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[4],
    backgroundColor: theme.colors.bone,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
  },
  primaryBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: theme.spacing[2],
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    paddingVertical: 15,
    minHeight: theme.touchTarget.vendor,
  },
  btnPressed: { opacity: 0.85 },
  primaryLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
    letterSpacing: 0.3,
  },
});
