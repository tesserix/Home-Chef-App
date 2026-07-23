import { useState } from 'react';
import {
  Alert,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { getServerErrorMessage } from '@homechef/mobile-shared/api';
import { theme } from '@homechef/mobile-shared/theme';
import { Button, EmptyState, Skeleton } from '@homechef/mobile-shared/ui';
import {
  CANCEL_REASONS,
  useCancellationRequests,
  useConfirmCancellation,
  type CancellationRequest,
} from '../hooks/useCancellations';

// Vendor arbitration queue (#475) — the mobile twin of the web vendor-portal
// screen: a customer asked to cancel; the chef picks the reason and the API
// issues the tiered refund. Same API + same CANCEL_REASONS as web.
export default function CancelRequestsScreen() {
  const { data, isLoading, isError, refetch } = useCancellationRequests('pending_vendor');
  const requests = data?.data ?? [];

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
        >
          {({ pressed }) => (
            <View style={pressed && Platform.OS === 'ios' ? { opacity: 0.6 } : undefined}>
              <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text style={styles.title}>Cancellation requests</Text>
        <View style={{ width: 24 }} />
      </View>

      {isLoading ? (
        <View style={styles.skeletonStack}>
          <Skeleton height={160} style={{ borderRadius: theme.radius.lg }} />
        </View>
      ) : isError ? (
        <EmptyState
          title="Couldn't load requests"
          body="Check your connection and try again."
          ctaLabel="Retry"
          onCtaPress={() => refetch()}
        />
      ) : requests.length === 0 ? (
        <EmptyState
          title="No requests right now"
          body="When a customer asks to cancel an order you're preparing, it appears here for you to confirm."
        />
      ) : (
        <ScrollView contentContainerStyle={styles.scroll}>
          {requests.map((r) => (
            <RequestCard key={r.id} req={r} />
          ))}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

function RequestCard({ req }: { req: CancellationRequest }) {
  const [reason, setReason] = useState<string | null>(null);
  const confirm = useConfirmCancellation();

  function onConfirm() {
    if (!reason) return;
    confirm.mutate(
      { id: req.id, reason },
      {
        onSuccess: () =>
          Alert.alert('Cancellation confirmed', 'The refund has been issued to the customer.'),
        onError: (e) => Alert.alert('Could not confirm', getServerErrorMessage(e, 'Please try again.')),
      },
    );
  }

  return (
    <View style={styles.card}>
      <Text style={styles.orderNo}>Order #{req.orderId.slice(0, 8)}</Text>
      {req.customerReason ? <Text style={styles.custReason}>“{req.customerReason}”</Text> : null}
      <Text style={styles.q}>Where is this order?</Text>
      {CANCEL_REASONS.map((r) => {
        const active = reason === r.value;
        return (
          <Pressable
            key={r.value}
            onPress={() => setReason(r.value)}
            accessibilityRole="radio"
            accessibilityLabel={r.label}
            accessibilityState={{ selected: active }}
            android_ripple={{ color: `${theme.colors.ink.DEFAULT}0F`, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.reason,
                  active && styles.reasonActive,
                  pressed && Platform.OS === 'ios' && styles.reasonPressed,
                ]}
              >
                <Text style={styles.reasonLabel}>{r.label}</Text>
                <Text style={styles.reasonHint}>{r.hint}</Text>
              </View>
            )}
          </Pressable>
        );
      })}
      <Button
        label="Confirm cancellation"
        onPress={onConfirm}
        disabled={!reason || confirm.isPending}
        loading={confirm.isPending}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  title: { fontFamily: 'Geist-Bold', fontSize: 20, color: theme.colors.ink.DEFAULT },
  scroll: { padding: theme.spacing[4], gap: theme.spacing[3] },
  skeletonStack: { padding: theme.spacing[4] },
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    gap: theme.spacing[2],
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
  },
  orderNo: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.ink.DEFAULT },
  custReason: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.soft, fontStyle: 'italic' },
  q: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: theme.colors.ink.DEFAULT, marginTop: theme.spacing[2] },
  reason: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.md,
    padding: theme.spacing[3],
  },
  reasonActive: { borderColor: theme.colors.ink.DEFAULT, backgroundColor: theme.colors.bone },
  reasonPressed: { backgroundColor: theme.colors.bone },
  reasonLabel: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: theme.colors.ink.DEFAULT },
  reasonHint: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.soft, marginTop: 2 },
});
