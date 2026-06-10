import { useEffect } from 'react';
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import type { TFunction } from 'i18next';
import { useQuery } from '@tanstack/react-query';
import { theme } from '@homechef/mobile-shared/theme';
import { api } from '../../lib/api';
import { useAuthStore } from '../../store/auth-store';

interface OnboardingStatusResponse {
  status:
    | 'not_started'
    | 'in_progress'
    | 'pending_review'
    | 'submitted'
    | 'verified'
    | 'rejected';
  completed: boolean;
  step: number;
  chefId: string | null;
  profile: { rejectionReason?: string; submittedAt?: string } | null;
}

function formatSubmitted(iso: string | undefined): string | null {
  if (!iso) return null;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return null;
  return d.toLocaleDateString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

// Relative-time helper for the timeline step. Renders as "Today at 3:42 PM",
// "Yesterday", or "X days ago" — same vocabulary the chef sees elsewhere.
function relativeFrom(iso: string | undefined, t: TFunction): string | null {
  if (!iso) return null;
  const ts = new Date(iso).getTime();
  if (Number.isNaN(ts)) return null;
  const now = Date.now();
  const diffMs = now - ts;
  const diffMins = Math.floor(diffMs / 60_000);
  if (diffMins < 1) return t('onboarding.justNow');
  if (diffMins < 60) return `${diffMins}m ago`;
  const diffHrs = Math.floor(diffMins / 60);
  if (diffHrs < 24) {
    const time = new Date(ts).toLocaleTimeString('en-IN', {
      hour: 'numeric',
      minute: '2-digit',
      hour12: true,
    });
    return t('onboarding.todayAt', { time });
  }
  const diffDays = Math.floor(diffHrs / 24);
  if (diffDays === 1) return t('onboarding.yesterday');
  if (diffDays < 7) return t('onboarding.daysAgo', { count: diffDays });
  return formatSubmitted(iso) ?? '';
}

export default function PendingScreen() {
  const { t } = useTranslation();
  const { logout } = useAuthStore();

  const { data, isLoading, isRefetching, refetch } = useQuery({
    queryKey: ['chef', 'onboarding', 'status'],
    queryFn: () =>
      api.get<OnboardingStatusResponse>('/chef/onboarding/status'),
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
  });

  const status = data?.data?.status;
  const rejectionReason = data?.data?.profile?.rejectionReason;
  const submittedAt = formatSubmitted(data?.data?.profile?.submittedAt);
  const isRejected = status === 'rejected';

  useEffect(() => {
    if (status === 'verified') {
      router.replace('/(tabs)');
    }
  }, [status]);

  function handleLogout(): void {
    logout();
    router.replace('/(auth)/login');
  }

  function handleReapply(): void {
    router.replace('/(onboarding)/personal-info');
  }

  if (isLoading) {
    return (
      <SafeAreaView style={styles.loadingScreen}>
        <ActivityIndicator color={theme.colors.ink.DEFAULT} size="large" />
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* Header bar */}
      <View style={styles.header}>
        <Text style={styles.headerTitle}>{t('onboarding.applicationStatus')}</Text>
        <Pressable onPress={handleLogout} hitSlop={8} style={styles.logoutBtn}>
          <Text style={styles.logoutLabel}>{t('onboarding.logout')}</Text>
        </Pressable>
      </View>

      <ScrollView
        contentContainerStyle={styles.scrollContent}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={refetch}
            tintColor={theme.colors.ink.DEFAULT}
          />
        }
      >
        {isRejected ? (
          <>
            <View style={styles.statusRow}>
              <View
                style={[
                  styles.statusDot,
                  { backgroundColor: theme.colors.destructive.DEFAULT },
                ]}
              />
              <Text style={styles.statusLabel}>{t('onboarding.notApproved')}</Text>
            </View>
            <Text style={styles.headline}>{t('onboarding.applicationNotApproved')}</Text>
            <Text style={styles.body}>
              {rejectionReason
                ? t('onboarding.reviewerFeedback', { reason: rejectionReason })
                : t('onboarding.notApprovedBody')}
            </Text>

            {submittedAt && (
              <View style={styles.infoBlock}>
                <Text style={styles.infoKey}>{t('onboarding.submitted')}</Text>
                <Text style={styles.infoVal}>{submittedAt}</Text>
              </View>
            )}

            <Pressable
              onPress={handleReapply}
              style={({ pressed }) => [
                styles.primaryBtn,
                pressed && { opacity: 0.85 },
              ]}
              accessibilityRole="button"
            >
              <Text style={styles.primaryBtnLabel}>{t('onboarding.reapply')}</Text>
            </Pressable>
          </>
        ) : (
          <>
            <View style={styles.statusRow}>
              <View
                style={[
                  styles.statusDot,
                  { backgroundColor: theme.colors.herb.DEFAULT },
                ]}
              />
              <Text style={styles.statusLabel}>{t('onboarding.underReview')}</Text>
            </View>
            <Text style={styles.headline}>{t('onboarding.reviewingHeadline')}</Text>
            <Text style={styles.body}>
              {t('onboarding.reviewingBody')}
            </Text>

            {/* Application timeline — three steps, current in persimmon.
                Gives a sense of forward motion without faking progress. */}
            <View style={styles.timeline}>
              <TimelineStep
                state="done"
                label={t('onboarding.submitted')}
                meta={relativeFrom(data?.data?.profile?.submittedAt, t) ?? ''}
              />
              <TimelineConnector active />
              <TimelineStep
                state="active"
                label={t('onboarding.reviewing')}
                meta={t('onboarding.now')}
              />
              <TimelineConnector />
              <TimelineStep
                state="pending"
                label={t('onboarding.approved')}
                meta={t('onboarding.approvedEta')}
              />
            </View>

            {/* What you'll get — sets expectations and reduces wait anxiety */}
            <View style={styles.unlockBlock}>
              <Text style={styles.unlockLabel}>{t('onboarding.whatYoullGet')}</Text>
              <UnlockItem text={t('onboarding.unlockMenu')} />
              <UnlockItem text={t('onboarding.unlockOrders')} />
              <UnlockItem text={t('onboarding.unlockPayouts')} />
            </View>

            {/* Manual refresh button. Pull-to-refresh still works, but a
                visible CTA is the discoverable affordance — and it gives
                a stuck user a clear action while they wait. */}
            <Pressable
              onPress={() => refetch()}
              disabled={isRefetching}
              style={({ pressed }) => [
                styles.primaryBtn,
                pressed && { opacity: 0.85 },
                isRefetching && { opacity: 0.5 },
              ]}
              accessibilityRole="button"
              accessibilityLabel={t('onboarding.checkStatus')}
            >
              {isRefetching ? (
                <ActivityIndicator color={theme.colors.paper} />
              ) : (
                <Text style={styles.primaryBtnLabel}>{t('onboarding.checkStatus')}</Text>
              )}
            </Pressable>

            <Text style={styles.helperLine}>
              {t('onboarding.pendingHelper')}
            </Text>
          </>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}

// ---- Inline sub-components --------------------------------------------------

type StepState = 'done' | 'active' | 'pending';

interface TimelineStepProps {
  state: StepState;
  label: string;
  meta: string;
}

function TimelineStep({ state, label, meta }: TimelineStepProps) {
  return (
    <View style={timelineStyles.step}>
      <View
        style={[
          timelineStyles.dot,
          state === 'done' && timelineStyles.dotDone,
          state === 'active' && timelineStyles.dotActive,
          state === 'pending' && timelineStyles.dotPending,
        ]}
      >
        {state === 'done' ? (
          <Text style={timelineStyles.checkmark}>✓</Text>
        ) : null}
      </View>
      <Text
        style={[
          timelineStyles.label,
          state === 'pending' && { color: theme.colors.ink.muted },
        ]}
      >
        {label}
      </Text>
      <Text style={timelineStyles.meta} numberOfLines={2}>
        {meta}
      </Text>
    </View>
  );
}

function TimelineConnector({ active = false }: { active?: boolean }) {
  return (
    <View
      style={[
        timelineStyles.connector,
        active && { backgroundColor: theme.colors.ink.DEFAULT },
      ]}
    />
  );
}

function UnlockItem({ text }: { text: string }) {
  return (
    <View style={styles.unlockRow}>
      <View style={styles.unlockBullet} />
      <Text style={styles.unlockText}>{text}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },
  loadingScreen: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: theme.colors.paper,
  },

  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[3],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  headerTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  logoutBtn: { paddingVertical: 4, paddingHorizontal: 4 },
  logoutLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    textDecorationLine: 'underline',
  },

  scrollContent: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[6],
    paddingBottom: theme.spacing[10],
  },

  statusRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    marginBottom: theme.spacing[3],
  },
  statusDot: { width: 8, height: 8, borderRadius: 4 },
  statusLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.soft,
  },

  headline: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
    marginBottom: theme.spacing[3],
  },
  body: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    lineHeight: 22,
    color: theme.colors.ink.soft,
    marginBottom: theme.spacing[6],
    maxWidth: 380,
  },

  // Application timeline (3 steps)
  timeline: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    marginBottom: theme.spacing[6],
  },

  // "What you'll get" preview
  unlockBlock: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    paddingTop: theme.spacing[4],
    marginBottom: theme.spacing[6],
  },
  unlockLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[3],
  },
  unlockRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingVertical: theme.spacing[2],
  },
  unlockBullet: {
    width: 6,
    height: 6,
    borderRadius: 3,
    backgroundColor: theme.colors.ink.DEFAULT,
  },
  unlockText: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },

  infoGroup: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    marginBottom: theme.spacing[6],
  },
  infoBlock: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: theme.spacing[3],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  infoKey: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
  },
  infoVal: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },

  helperLine: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    textAlign: 'center',
    letterSpacing: 0.3,
    marginTop: theme.spacing[4],
  },

  primaryBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingVertical: theme.spacing[4],
    alignItems: 'center',
    marginTop: theme.spacing[2],
  },
  primaryBtnLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
    letterSpacing: 0.2,
  },
});

const timelineStyles = StyleSheet.create({
  step: {
    flex: 1,
    alignItems: 'center',
    gap: 4,
  },
  dot: {
    width: 22,
    height: 22,
    borderRadius: 11,
    alignItems: 'center',
    justifyContent: 'center',
  },
  dotDone: {
    backgroundColor: theme.colors.ink.DEFAULT,
  },
  dotActive: {
    backgroundColor: theme.colors.herb.DEFAULT,
  },
  dotPending: {
    borderWidth: 1.5,
    borderColor: theme.colors.mist.strong,
    backgroundColor: theme.colors.paper,
  },
  checkmark: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: theme.colors.paper,
    lineHeight: 14,
  },
  connector: {
    flex: 1,
    height: 1.5,
    marginTop: 10,
    marginHorizontal: -2,
    backgroundColor: theme.colors.mist.strong,
  },
  label: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.DEFAULT,
    marginTop: theme.spacing[2],
    letterSpacing: 0.1,
  },
  meta: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.2,
    textAlign: 'center',
  },
});
