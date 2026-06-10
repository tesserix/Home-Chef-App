import { ActivityIndicator, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { AdminRequest, useAdminRequests } from '../hooks/useAdminRequests';

// Human-readable status labels — backend enum strings are snake_case and
// not chef-facing. Order kept canonical so badge colors below match the
// switch in `statusBadgeColor`.
const STATUS_LABEL: Record<AdminRequest['status'], string> = {
  pending: 'Under review',
  approved: 'Approved',
  rejected: 'Rejected',
  info_requested: 'Action required',
  cancelled: 'Cancelled',
};

function statusBadgeColor(status: AdminRequest['status']): {
  bg: string;
  fg: string;
} {
  switch (status) {
    case 'info_requested':
      return { bg: theme.colors.destructive.DEFAULT, fg: theme.colors.paper };
    case 'approved':
      return { bg: theme.colors.ink.DEFAULT, fg: theme.colors.paper };
    case 'rejected':
      return { bg: theme.colors.destructive.DEFAULT, fg: theme.colors.paper };
    case 'cancelled':
      return { bg: theme.colors.mist.strong, fg: theme.colors.ink.muted };
    case 'pending':
    default:
      return { bg: theme.colors.bone, fg: theme.colors.ink.DEFAULT };
  }
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString('en-IN', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    });
  } catch {
    return iso;
  }
}

function RequestCard({ request }: { request: AdminRequest }) {
  const colors = statusBadgeColor(request.status);
  // Only info_requested cards are tappable — the chef has nothing to
  // do for pending/approved/rejected (admin owns the state); making
  // them tappable would invite confused taps that land on a useless
  // screen.
  const isInteractive = request.status === 'info_requested';
  const inner = (
    <View style={styles.card}>
      <View style={styles.cardTopRow}>
        <View style={[styles.badge, { backgroundColor: colors.bg }]}>
          <Text style={[styles.badgeText, { color: colors.fg }]}>
            {STATUS_LABEL[request.status] ?? request.status}
          </Text>
        </View>
        <Text style={styles.cardDate}>{formatDate(request.createdAt)}</Text>
      </View>
      <Text style={styles.cardTitle} numberOfLines={2}>
        {request.title}
      </Text>
      {request.description ? (
        <Text style={styles.cardBody} numberOfLines={3}>
          {request.description}
        </Text>
      ) : null}
      {request.adminNotes && request.status === 'info_requested' ? (
        <View style={styles.notesBlock}>
          <Text style={styles.notesLabel}>Admin notes</Text>
          <Text style={styles.notesBody}>{request.adminNotes}</Text>
        </View>
      ) : null}
      {isInteractive ? (
        <Text style={styles.cardCta}>Respond to admin →</Text>
      ) : null}
    </View>
  );
  if (!isInteractive) return inner;
  return (
    <Pressable
      onPress={() => router.push(`/admin-requests/${request.id}`)}
      accessibilityRole="button"
      accessibilityLabel={`Respond to ${request.title}`}
    >
      {inner}
    </Pressable>
  );
}

export default function AdminRequestsScreen() {
  const { data, isLoading, isError } = useAdminRequests();

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
        <Text style={styles.commandTitle}>Admin requests</Text>
      </View>

      {isLoading ? (
        <View style={styles.center}>
          <ActivityIndicator size="large" color={theme.colors.ink.DEFAULT} />
        </View>
      ) : isError ? (
        <View style={styles.center}>
          <Text style={styles.muted}>Could not load your admin requests.</Text>
        </View>
      ) : (data?.length ?? 0) === 0 ? (
        <View style={styles.center}>
          <Text style={styles.muted}>No admin requests yet.</Text>
        </View>
      ) : (
        <ScrollView
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
        >
          {data!.map((r) => (
            <RequestCard key={r.id} request={r} />
          ))}
        </ScrollView>
      )}
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
  scrollContent: {
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[10],
    gap: theme.spacing[3],
  },

  card: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    paddingVertical: theme.spacing[3],
    gap: theme.spacing[2],
  },
  cardTopRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  badge: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: theme.radius.sm,
  },
  badgeText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    letterSpacing: 0.6,
  },
  cardDate: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
  },
  cardTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    lineHeight: 22,
  },
  cardBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    lineHeight: 20,
  },

  notesBlock: {
    backgroundColor: theme.colors.bone,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    borderLeftWidth: 3,
    borderLeftColor: theme.colors.destructive.DEFAULT,
    borderRadius: theme.radius.sm,
    gap: 4,
  },
  notesLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    letterSpacing: 1.2,
    color: theme.colors.ink.muted,
  },
  notesBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    lineHeight: 20,
  },
  cardCta: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.destructive.DEFAULT,
    marginTop: theme.spacing[1],
  },

  center: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
  },
  muted: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.muted,
    textAlign: 'center',
  },
});
