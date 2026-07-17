import { useState } from 'react';
import { ActivityIndicator, Alert, Pressable, StyleSheet, Text, View } from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import {
  orderCancellable,
  useCancellationRequest,
  useDisputeCancellation,
  useRequestCancellation,
  type CancellationRequest,
} from '../../hooks/useCancellation';

// Customer cancellation on the order detail (#478). If a request exists it shows
// the vendor's decision + refund (and a dispute action); otherwise, for a still-
// cancellable order, it offers to request one.
// Inline expansion — no bottom-sheet — deliberately, to avoid the modal-provider
// class of crash.
//
// The refund destination is NOT a customer choice: refunds go back to the
// original payment method. This screen used to offer a wallet-vs-card picker
// that defaulted to WALLET, which made unspendable store credit the normal
// outcome of a cancellation — wallet checkout (WALLET_CHECKOUT_ENABLED, #141) is
// off in production, so that credit can't be applied to an order, and no refund
// reached Razorpay. The server now derives the destination from the order's
// payment (handlers/cancellation.go resolveRefundDestination), so the client
// no longer sends one.
const money = (paise: number) => `₹${(paise / 100).toFixed(0)}`;

export function CancellationSection({ orderId, status }: { orderId: string; status: string }) {
  const { data: request, isLoading } = useCancellationRequest(orderId);
  const req = useRequestCancellation();
  const dispute = useDisputeCancellation();
  const [expanded, setExpanded] = useState(false);

  if (isLoading) return null;

  if (request) {
    return (
      <View style={styles.card}>
        <StatusView request={request} orderId={orderId} onDispute={() => dispute.mutate(
          { orderId },
          { onSuccess: () => Alert.alert('Dispute raised', 'Our team will review it and get back to you.') },
        )} />
      </View>
    );
  }

  if (!orderCancellable(status)) return null;

  function onRequest() {
    req.mutate(
      { orderId },
      {
        onSuccess: () => {
          setExpanded(false);
          Alert.alert(
            'Cancellation requested',
            "We've asked the chef to confirm. You'll be notified of the outcome and any refund.",
          );
        },
        onError: () => Alert.alert('Could not request', 'Please try again.'),
      },
    );
  }

  return (
    <View style={styles.card}>
      {!expanded ? (
        <Pressable onPress={() => setExpanded(true)} accessibilityRole="button">
          <Text style={styles.link}>Request cancellation</Text>
        </Pressable>
      ) : (
        <>
          <Text style={styles.label}>Refund to your original payment method</Text>
          <Text style={styles.hint}>
            Any refund goes back to the card or account you paid with, and can take 5–7 days to
            appear. The chef confirms and issues the right refund based on how far along the order
            is. The platform fee isn't refundable.
          </Text>
          <Pressable onPress={onRequest} disabled={req.isPending} style={styles.btn} accessibilityRole="button">
            {req.isPending ? (
              <ActivityIndicator color="#fff" />
            ) : (
              <Text style={styles.btnText}>Request cancellation</Text>
            )}
          </Pressable>
        </>
      )}
    </View>
  );
}

function StatusView({
  request,
  onDispute,
}: {
  request: CancellationRequest;
  orderId: string;
  onDispute: () => void;
}) {
  const dest = request.refundDestination === 'original' ? 'card' : 'wallet';
  switch (request.status) {
    case 'pending_vendor':
      return (
        <>
          <Text style={styles.statusTitle}>Cancellation requested</Text>
          <Text style={styles.statusBody}>Waiting for the chef to confirm — you'll be notified.</Text>
        </>
      );
    case 'approved':
    case 'auto_refunded':
    case 'resolved':
      return (
        <>
          <Text style={styles.statusTitle}>Order cancelled</Text>
          <Text style={styles.statusBody}>
            {money(request.refundTotalPaise)} refunded to your {dest}.
          </Text>
          {request.status === 'approved' ? (
            <Pressable onPress={onDispute} accessibilityRole="button">
              <Text style={styles.link}>Dispute the refund amount</Text>
            </Pressable>
          ) : null}
        </>
      );
    case 'disputed':
    case 'admin_review':
      return (
        <>
          <Text style={styles.statusTitle}>Under review</Text>
          <Text style={styles.statusBody}>Our team is reviewing your cancellation.</Text>
        </>
      );
    default:
      return <Text style={styles.statusBody}>Cancellation status: {request.status}</Text>;
  }
}

const styles = StyleSheet.create({
  card: {
    marginHorizontal: 16,
    marginTop: 12,
    padding: 16,
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.canvas,
    gap: 8,
  },
  link: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.coral.DEFAULT },
  label: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  hint: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft, lineHeight: 16 },
  btn: {
    marginTop: 4,
    borderRadius: 10,
    backgroundColor: customerColors.coral.DEFAULT,
    paddingVertical: 12,
    alignItems: 'center',
  },
  btnText: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: '#fff' },
  statusTitle: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  statusBody: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft },
});
