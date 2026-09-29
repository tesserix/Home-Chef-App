import { useRef, useState } from 'react';
import { ActivityIndicator, Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import { useQueryClient } from '@tanstack/react-query';
import { customerColors } from '@homechef/mobile-shared/theme';

// Android ripple tints — translucent tokens, never new literals.
const BTN_RIPPLE = `${customerColors.canvas}33`;
const LINK_RIPPLE = `${customerColors.coral.DEFAULT}1F`;
import {
  orderCancellable,
  useCancellationRequest,
  useDisputeCancellation,
  useRequestCancellation,
  type CancellationRequest,
} from '../../hooks/useCancellation';
import { useAlert, type SheetHandle } from '@homechef/mobile-shared/ui';
import { useRouter } from 'expo-router';
import { friendlyErrorMessage } from '../../lib/errors';
import { formatMoney } from '../../lib/format';
import { refundDestinationLine } from '../../lib/refund-destination';
import { estimateCancellationRefund, toPaise } from '../../lib/cancellation-refund-estimate';
import type { Order } from '../../types/customer';
import { DisputeReasonSheet } from './DisputeReasonSheet';
import { CancelRefundSheet } from './CancelRefundSheet';
import { HAIRLINE } from '../../lib/hairline';

/** The order's money, in RUPEES, as the detail screen already holds it. Needed
 *  here for the pre-cancellation refund estimate (#1032) — the customer must see
 *  what a cancellation costs BEFORE the request is sent, not on the cancelled-
 *  order screen afterwards. */
export interface CancellationPricing {
  subtotal: number;
  /** Effective delivery fee (deliveryFeeFinal ?? deliveryFee, #703). */
  deliveryFee: number;
  platformFee: number;
  tax: number;
  discount: number;
  total: number;
  /** Already refunded through any other channel — caps the estimate (#642). */
  alreadyRefunded: number;
  /** Wallet + loyalty credit that funded the order, for the pro-rata note. */
  creditApplied: number;
}

// Which orders the GENERIC cancellation endpoint refuses, and where they are
// actually cancelled. Keys mirror OrderResponse.source; a missing entry means the
// generic flow owns the order and the request action is offered normally.
const OWNING_FLOW: Partial<
  Record<NonNullable<Order['source']>, { title: string; body: string; cta: string; href?: string }>
> = {
  meal_plan: {
    title: 'Cancel this from your meal plan',
    body: "This meal is part of a meal plan, so it's cancelled from the plan itself — that's what releases the right refund for the day. Skipping or cancelling a single day is done there too.",
    cta: 'Go to my meal plans',
    href: '/meal-plans',
  },
  group: {
    title: 'Cancel this from the group order',
    body: 'This is part of a group order. The organiser cancels it from the group order, which refunds everyone who paid in.',
    cta: 'Go to the group order',
  },
};

// Customer cancellation on the order detail (#478). If a request exists it shows
// the vendor's decision + refund (and a dispute action); otherwise, for a still-
// cancellable order, it offers to request one.
// The "request cancellation" step below still uses inline expansion, not a
// sheet — unchanged from the original design. The dispute action (#876) does
// use a bottom sheet (DisputeReasonSheet, built on SheetBase): the anti-modal
// note this comment used to carry was about @gorhom/bottom-sheet silently
// no-op'ing on this app's gorhom+reanimated pairing, not sheets in general —
// SheetBase (a plain RN Modal + Animated implementation, zero gorhom/
// reanimated) is the established pattern every other sheet in this app uses,
// including on this exact screen (app/order/[id]/index.tsx).
//
// The refund destination is NOT a customer choice: refunds go back to the
// original payment method. This screen used to offer a wallet-vs-card picker
// that defaulted to WALLET, which made unspendable store credit the normal
// outcome of a cancellation — wallet checkout (WALLET_CHECKOUT_ENABLED, #141) is
// off in production, so that credit can't be applied to an order, and no refund
// reached the gateway. The server now derives the destination from the order's
// payment (handlers/cancellation.go resolveRefundDestination), so the client
// no longer sends one.
// Money renders through lib/format.ts like every other figure in the app. The
// local `(paise / 100).toFixed(0)` this replaced turned ₹377.07 into "₹377" —
// three rupees adrift from what the customer can check against their bank.
const money = (paise: number) => formatMoney(paise / 100);

export function CancellationSection({
  orderId,
  status,
  paymentStatus,
  source,
  walletRefunded,
  loyaltyRefunded,
  pricing,
}: {
  orderId: string;
  status: string;
  paymentStatus?: string;
  source?: Order['source'];
  walletRefunded?: number;
  loyaltyRefunded?: number;
  pricing?: CancellationPricing;
}) {
  const { showAlert } = useAlert();
  const router = useRouter();
  const qc = useQueryClient();
  const { data: request, isLoading } = useCancellationRequest(orderId);
  const req = useRequestCancellation();
  const dispute = useDisputeCancellation();
  const [expanded, setExpanded] = useState(false);
  const disputeSheetRef = useRef<SheetHandle>(null);
  const refundSheetRef = useRef<SheetHandle>(null);

  if (isLoading) return null;

  // A customer must not be able to open a second dispute on an order that
  // already has one open. useDisputeCancellation's own onSuccess invalidates
  // ['order', orderId, 'cancel-request'], but that refetch is async — between
  // the success alert and the refetch landing, `request.status` is still
  // 'approved' and the trigger stays tappable (indefinitely, if the refetch
  // never lands: offline, dropped request). Write the disputed status into
  // the cache immediately so the card flips to "Under review" right away;
  // the invalidate above stays as reconciliation, same lesson as #868.
  function onDisputeSubmit(reason: string) {
    dispute.mutate(
      { orderId, reason },
      {
        onSuccess: () => {
          qc.setQueryData<CancellationRequest | null>(
            ['order', orderId, 'cancel-request'],
            (old) => (old ? { ...old, status: 'disputed' } : old),
          );
          showAlert('Dispute raised', 'Our team will review it and get back to you.');
        },
        onError: (err) =>
          showAlert(
            "Couldn't raise the dispute",
            friendlyErrorMessage(err, 'Please try again in a moment.'),
          ),
      },
    );
  }

  if (request) {
    return (
      <View style={styles.card}>
        <StatusView
          request={request}
          orderId={orderId}
          onDispute={() => disputeSheetRef.current?.present()}
          disputePending={dispute.isPending}
          walletRefunded={walletRefunded}
          loyaltyRefunded={loyaltyRefunded}
        />
        <DisputeReasonSheet ref={disputeSheetRef} onSubmit={onDisputeSubmit} />
      </View>
    );
  }

  if (!orderCancellable(status, paymentStatus)) return null;

  // A meal-plan day / group order is refund-managed by THAT flow on a separate
  // idempotency keyspace, so handlers/cancellation.go refuses a generic request
  // with 422 no matter how many times it's retried. Offering the button here was a
  // dead end: the request always failed and the screen said "Please try again".
  // Say why, and hand the customer the flow that CAN cancel it.
  const owningFlow = OWNING_FLOW[source ?? 'alacarte'];
  if (owningFlow) {
    return (
      <View style={styles.card}>
        <Text style={styles.label}>{owningFlow.title}</Text>
        <Text style={styles.hint}>{owningFlow.body}</Text>
        {owningFlow.href ? (
          <Pressable
            onPress={() => router.push(owningFlow.href as never)}
            accessibilityRole="button"
            accessibilityLabel={owningFlow.cta}
          >
            <Text style={styles.link}>{owningFlow.cta}</Text>
          </Pressable>
        ) : null}
      </View>
    );
  }

  function onRequest() {
    req.mutate(
      { orderId },
      {
        // The server takes one of two paths and says which: a pre-acceptance
        // cancel is refunded on the spot (`auto_refunded`), anything later waits
        // for the chef (`pending_vendor`). This used to show the vendor-review
        // copy for both, so a customer whose money was already back was told to
        // wait for an outcome that had happened seconds earlier — with the screen
        // behind the dialog already reading "Order cancelled".
        onSuccess: (data) => {
          setExpanded(false);
          const settled = data?.request as CancellationRequest | undefined;
          if (settled?.status === 'auto_refunded') {
            // Deliberately no destination here. The rail split lives on the order
            // row, which this screen has not refetched yet at the moment the alert
            // fires — naming a destination from stale data would reproduce exactly
            // the bug this change fixes. The card below states the split once the
            // refreshed order lands.
            showAlert(
              'Order cancelled',
              `${money(settled.refundTotalPaise ?? 0)} has been refunded. The breakdown is on your order.`,
            );
            return;
          }
          showAlert(
            'Cancellation requested',
            "We've asked the chef to confirm. You'll be notified of the outcome and any refund.",
          );
        },
        // Surface the server's own reason (friendlyErrorMessage strips transport
        // noise). "Please try again" hid explanations the customer needed and
        // invited a retry that could never work.
        onError: (err) =>
          showAlert(
            "Couldn't request cancellation",
            friendlyErrorMessage(err, 'Please try again in a moment.'),
          ),
      },
    );
  }

  // The refund estimate the confirmation sheet shows (#1032). Bounds, not a
  // figure: on an accepted order the chef picks the tier when they confirm, so
  // no exact number exists yet — see lib/cancellation-refund-estimate.ts.
  const estimate = pricing
    ? estimateCancellationRefund({
        status,
        subtotalPaise: toPaise(pricing.subtotal),
        discountPaise: toPaise(pricing.discount),
        deliveryFeePaise: toPaise(pricing.deliveryFee),
        platformFeePaise: toPaise(pricing.platformFee),
        taxPaise: toPaise(pricing.tax),
        totalPaise: toPaise(pricing.total),
        alreadyRefundedPaise: toPaise(pricing.alreadyRefunded),
      })
    : null;

  // With an estimate, the request goes through the confirmation sheet. Without
  // one (the screen didn't pass pricing) the button still submits directly —
  // a missing prop must not leave a customer unable to cancel at all.
  function onRequestPress() {
    if (estimate) {
      refundSheetRef.current?.present();
      return;
    }
    onRequest();
  }

  return (
    <View style={styles.card}>
      {!expanded ? (
        <Pressable
          onPress={() => setExpanded(true)}
          accessibilityRole="button"
          accessibilityLabel="Request cancellation"
        >
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
          <Pressable
            onPress={onRequestPress}
            disabled={req.isPending}
            accessibilityRole="button"
            accessibilityLabel="Request cancellation"
            android_ripple={req.isPending ? undefined : { color: BTN_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[styles.btn, pressed && Platform.OS === 'ios' && !req.isPending && styles.btnPressed]}
              >
                {req.isPending ? (
                  <ActivityIndicator color={customerColors.canvas} />
                ) : (
                  <Text style={styles.btnText}>Request cancellation</Text>
                )}
              </View>
            )}
          </Pressable>
        </>
      )}
      {/* Mounted regardless of `expanded` so dismissing the sheet can't unmount
          it mid-exit-animation. SheetBase renders nothing until presented. */}
      {estimate ? (
        <CancelRefundSheet
          ref={refundSheetRef}
          estimate={estimate}
          creditApplied={pricing?.creditApplied ?? 0}
          onConfirm={onRequest}
        />
      ) : null}
    </View>
  );
}

function StatusView({
  request,
  onDispute,
  disputePending,
  walletRefunded,
  loyaltyRefunded,
}: {
  request: CancellationRequest;
  orderId: string;
  onDispute: () => void;
  disputePending: boolean;
  walletRefunded?: number;
  loyaltyRefunded?: number;
}) {
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
            {refundDestinationLine(
              request.refundTotalPaise,
              walletRefunded,
              loyaltyRefunded,
              request.refundDestination,
            )}
          </Text>
          {request.status === 'approved' ? (
            <Pressable
              onPress={onDispute}
              disabled={disputePending}
              accessibilityRole="button"
              accessibilityLabel="Dispute the refund amount"
              android_ripple={disputePending ? undefined : { color: LINK_RIPPLE, borderless: false }}
            >
              <View style={styles.linkRow}>
                {disputePending ? (
                  <ActivityIndicator size="small" color={customerColors.coral.DEFAULT} />
                ) : (
                  <Text style={styles.link}>Dispute the refund amount</Text>
                )}
              </View>
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
    borderWidth: HAIRLINE,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.canvas,
    gap: 8,
  },
  link: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.coral.DEFAULT },
  linkRow: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  label: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  hint: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft, lineHeight: 16 },
  // Spec §3 primary button radius (8) — was 10.
  btn: {
    marginTop: 4,
    borderRadius: 8,
    minHeight: 44,
    backgroundColor: customerColors.coral.DEFAULT,
    paddingVertical: 12,
    alignItems: 'center',
    justifyContent: 'center',
  },
  btnPressed: { backgroundColor: customerColors.coral.pressed },
  btnText: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.canvas },
  statusTitle: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  statusBody: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, fontVariant: ['tabular-nums'] },
});
