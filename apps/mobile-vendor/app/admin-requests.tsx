import { useEffect, useMemo, useState } from 'react';
import { ActivityIndicator, Alert, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { AlertTriangle, BellRing, ChevronLeft } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import {
  AdminRequest,
  useAdminRequests,
  useRemindAdminRequest,
} from '../hooks/useAdminRequests';

// #697 — how long until the chef can bump again, phrased the way a person waits.
// Rounds UP so the label never claims "0h" while the button is still locked.
function untilLabel(iso: string, now: number): string {
  const ms = new Date(iso).getTime() - now;
  if (ms <= 0) return '';
  const mins = Math.ceil(ms / 60_000);
  if (mins < 60) return `${mins}m`;
  const hrs = Math.ceil(mins / 60);
  if (hrs < 24) return `${hrs}h`;
  return `${Math.ceil(hrs / 24)}d`;
}

// A minute is the right resolution for a 6-24h countdown: it keeps the label
// honest without re-rendering the list every second.
function useNowEveryMinute(): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 60_000);
    return () => clearInterval(t);
  }, []);
  return now;
}

// The bump control. Shown only where it means something: an undecided request
// the chef is waiting on.
function RemindControl({ request, now }: { request: AdminRequest; now: number }) {
  const remind = useRemindAdminRequest();
  const escalated = request.reminderCount >= 3;

  // Trust the server's verdict, but re-check the deadline locally so the button
  // unlocks on the countdown rather than on the next refetch.
  const unlocked =
    request.canRemind ||
    (!!request.nextRemindAt && new Date(request.nextRemindAt).getTime() <= now);
  const waitingFor = request.nextRemindAt ? untilLabel(request.nextRemindAt, now) : '';

  function onPress() {
    remind.mutate(request.id, {
      onError: (err) => {
        const status = err?.response?.status;
        Alert.alert(
          status === 429 ? 'Not yet' : "Couldn't send the reminder",
          status === 429
            ? 'This request was reminded recently. The button will unlock again shortly.'
            : (err?.response?.data?.error ?? 'Please try again in a moment.'),
        );
      },
    });
  }

  return (
    <View style={styles.remindRow}>
      {request.reminderCount > 0 ? (
        <Text style={styles.remindCount}>
          {escalated
            ? `Escalated · reminded ${request.reminderCount}×`
            : `Reminded ${request.reminderCount}×`}
        </Text>
      ) : (
        <Text style={styles.remindHint}>
          {unlocked ? 'Waiting on admin' : `Remind in ${waitingFor}`}
        </Text>
      )}
      <Pressable
        onPress={onPress}
        disabled={!unlocked || remind.isPending}
        accessibilityRole="button"
        accessibilityLabel={
          unlocked
            ? `Remind admin about ${request.title}`
            : `Remind admin about ${request.title}. Available in ${waitingFor}`
        }
        accessibilityState={{ disabled: !unlocked || remind.isPending, busy: remind.isPending }}
        style={({ pressed }) => [
          styles.remindBtn,
          !unlocked && styles.remindBtnLocked,
          pressed && unlocked && styles.remindBtnPressed,
        ]}
      >
        <BellRing
          size={14}
          color={unlocked ? theme.colors.ink.DEFAULT : theme.colors.ink.soft}
        />
        <Text style={[styles.remindBtnText, !unlocked && styles.remindBtnTextLocked]}>
          {remind.isPending ? 'Sending…' : unlocked ? 'Remind' : waitingFor}
        </Text>
      </Pressable>
    </View>
  );
}

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

// Status chip colors per spec §2 — tint background + darker text of the
// same hue, never solid fills.
function statusBadgeColor(status: AdminRequest['status']): {
  bg: string;
  fg: string;
} {
  switch (status) {
    case 'info_requested':
      return {
        bg: theme.colors.destructive.tint,
        fg: theme.colors.destructive.DEFAULT,
      };
    case 'approved':
      return { bg: theme.colors.success.tint, fg: theme.colors.success.soft };
    case 'rejected':
      return {
        bg: theme.colors.destructive.tint,
        fg: theme.colors.destructive.DEFAULT,
      };
    case 'cancelled':
      return { bg: theme.colors.mist.DEFAULT, fg: theme.colors.ink.muted };
    case 'pending':
    default:
      return { bg: theme.colors.amber.tint, fg: theme.colors.ink.DEFAULT };
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

function RequestCard({ request, now }: { request: AdminRequest; now: number }) {
  const colors = statusBadgeColor(request.status);
  // Bumping only means something while the request is undecided — an approved
  // request blocks nobody, and the server refuses it anyway.
  const remindable = request.status === 'pending' || request.status === 'info_requested';
  const escalated = request.reminderCount >= 3;
  // Only info_requested cards are tappable — the chef has nothing to
  // do for pending/approved/rejected (admin owns the state); making
  // them tappable would invite confused taps that land on a useless
  // screen.
  const isInteractive = request.status === 'info_requested';
  const inner = (
    <View style={styles.card}>
      <View style={styles.cardTopRow}>
        <View style={styles.badgeGroup}>
          <View style={[styles.badge, { backgroundColor: colors.bg }]}>
            <Text style={[styles.badgeText, { color: colors.fg }]}>
              {STATUS_LABEL[request.status] ?? request.status}
            </Text>
          </View>
          {/* Escalation marker. The icon is paired with text rather than
              standing alone on colour — a warning nobody can read is decoration. */}
          {escalated ? (
            <View style={styles.escalatedBadge}>
              <AlertTriangle size={11} color={theme.colors.destructive.DEFAULT} />
              <Text style={styles.escalatedText}>Escalated</Text>
            </View>
          ) : null}
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
      {remindable ? <RemindControl request={request} now={now} /> : null}
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
  const now = useNowEveryMinute();

  // Reminded requests float to the top of the CHEF's list too, not just the
  // admin's. A chef who chased something wants to see its state without
  // scrolling past a month of approved items — and the escalation marker is only
  // useful if it is where they look first.
  const sorted = useMemo(() => {
    const rank = (r: AdminRequest) =>
      r.reminderCount >= 3 ? 0 : r.reminderCount > 0 ? 1 : 2;
    return [...(data ?? [])].sort(
      (a, b) =>
        rank(a) - rank(b) ||
        b.reminderCount - a.reminderCount ||
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
    );
  }, [data]);

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
          {sorted.map((r) => (
            <RequestCard key={r.id} request={r} now={now} />
          ))}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  badgeGroup: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  escalatedBadge: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 3,
    borderRadius: 9999,
    paddingHorizontal: 7,
    paddingVertical: 2,
    backgroundColor: theme.colors.destructive.tint,
  },
  escalatedText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    color: theme.colors.destructive.DEFAULT,
  },
  remindRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 12,
    marginTop: 12,
  },
  remindCount: { fontFamily: 'Inter-SemiBold', fontSize: 12, color: theme.colors.ink.soft, flex: 1 },
  remindHint: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.soft, flex: 1 },
  // 44px floor — a kitchen-context tap with messy hands.
  remindBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    minHeight: 44,
    paddingHorizontal: 14,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
  },
  remindBtnLocked: { opacity: 0.45 },
  remindBtnPressed: { opacity: 0.7 },
  remindBtnText: { fontFamily: 'Inter-SemiBold', fontSize: 13, color: theme.colors.ink.DEFAULT },
  remindBtnTextLocked: { color: theme.colors.ink.soft },
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

  // Request card — white surface on the bone canvas (spec §1)
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    marginBottom: theme.spacing[3],
    gap: theme.spacing[2],
    ...theme.shadow[1],
  },
  cardTopRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  // Pill chip per spec §2 — tint bg + colored text, radius.full
  badge: {
    paddingHorizontal: theme.spacing[2],
    paddingVertical: 3,
    borderRadius: theme.radius.full,
  },
  badgeText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.2,
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
    borderRadius: theme.radius.DEFAULT,
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
  // Text link per spec §3 — herb, no underline
  cardCta: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
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
