import { useState } from 'react';
import { Alert, Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';
import { Button, Screen } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { Badge, Empty, LoadingList, ScreenHeader } from '../components/kit';
import {
  CANCEL_REASONS,
  useAdminCancellations,
  useResolveCancellation,
  type AdminCancellationRequest,
} from '../hooks/useAdminCancellations';
import { errorMessage } from '../lib/format';

// Admin arbitration queue (#480) — the mobile twin of the admin-portal page.
// Disputes + vendor timeouts; the admin picks the correct tier and the refund is
// issued (timeout) or topped up to the difference (dispute).
const money = (paise: number) => `₹${(paise / 100).toFixed(0)}`;

export default function CancellationsScreen() {
  const q = useAdminCancellations();
  const items = q.data?.data ?? [];

  return (
    <Screen>
      <ScreenHeader title="Cancellation arbitration" subtitle="Disputes & vendor timeouts" back />
      {q.isLoading ? (
        <LoadingList />
      ) : items.length === 0 ? (
        <Empty title="Nothing to review" body="Disputes and vendor timeouts will show up here." />
      ) : (
        <ScrollView contentContainerStyle={styles.scroll}>
          {items.map((r) => (
            <ArbitrationCard key={r.id} req={r} />
          ))}
        </ScrollView>
      )}
    </Screen>
  );
}

function ArbitrationCard({ req }: { req: AdminCancellationRequest }) {
  const [reason, setReason] = useState<string | null>(null);
  const [note, setNote] = useState('');
  const resolve = useResolveCancellation();

  function onResolve() {
    if (!reason) return;
    resolve.mutate(
      { id: req.id, reason, note },
      {
        onSuccess: () => Alert.alert('Resolved', 'Any additional refund has been issued to the customer.'),
        onError: (e) => Alert.alert('Could not resolve', errorMessage(e)),
      },
    );
  }

  return (
    <View style={styles.card}>
      <View style={styles.rowBetween}>
        <Text style={styles.orderNo}>Order #{req.orderId.slice(0, 8)}</Text>
        <Badge
          label={req.status === 'disputed' ? 'Disputed' : 'Timeout'}
          tone={req.status === 'disputed' ? 'danger' : 'warning'}
        />
      </View>
      {req.customerReason ? <Text style={styles.sub}>Customer: “{req.customerReason}”</Text> : null}
      {req.disputeReason ? <Text style={styles.sub}>Dispute: “{req.disputeReason}”</Text> : null}
      {req.refundExecuted ? (
        <Text style={styles.meta}>
          Already refunded {money(req.refundTotalPaise)} · vendor kept {money(req.vendorKeptPaise)}
        </Text>
      ) : null}

      <Text style={styles.q}>Correct tier</Text>
      {CANCEL_REASONS.map((r) => {
        const active = reason === r.value;
        return (
          <Pressable
            key={r.value}
            onPress={() => setReason(r.value)}
            style={[styles.reason, active && styles.reasonActive]}
            accessibilityRole="radio"
            accessibilityState={{ selected: active }}
          >
            <Text style={[styles.reasonLabel, active && styles.reasonLabelActive]}>{r.label}</Text>
            <Text style={styles.reasonHint}>{r.hint}</Text>
          </Pressable>
        );
      })}

      <TextInput
        value={note}
        onChangeText={setNote}
        placeholder="Internal note (optional)"
        placeholderTextColor={theme.colors.ink.soft}
        style={styles.note}
        multiline
      />

      <Button
        label="Resolve"
        onPress={onResolve}
        disabled={!reason || resolve.isPending}
        loading={resolve.isPending}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  scroll: { padding: 16, gap: 12 },
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: 16,
    gap: 6,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
  },
  rowBetween: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  orderNo: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.ink.DEFAULT },
  sub: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.soft },
  meta: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.soft, marginTop: 2 },
  q: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: theme.colors.ink.DEFAULT, marginTop: 8 },
  reason: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.md,
    padding: 12,
  },
  reasonActive: { borderColor: theme.colors.herb.DEFAULT, backgroundColor: theme.colors.herb.tint },
  reasonLabel: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: theme.colors.ink.DEFAULT },
  reasonLabelActive: { color: theme.colors.herb.DEFAULT },
  reasonHint: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.soft, marginTop: 2 },
  note: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.md,
    padding: 10,
    marginTop: 8,
    minHeight: 44,
    fontFamily: 'Inter',
    fontSize: 13,
    color: theme.colors.ink.DEFAULT,
  },
});
