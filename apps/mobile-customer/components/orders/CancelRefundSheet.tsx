import { forwardRef } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { Sheet, type SheetHandle } from '@homechef/mobile-shared/ui';
import { customerColors } from '@homechef/mobile-shared/theme';

import { formatMoney } from '../../lib/format';
import type { CancellationRefundEstimate } from '../../lib/cancellation-refund-estimate';

// CancelRefundSheet — the confirmation shown BEFORE a cancellation request is
// sent (#1032). Cancelling is not free: the platform fee and its GST are always
// retained, and on an accepted order the chef's tier decides how much of the
// food comes back. Learning that afterwards, on the cancelled-order screen, is
// what generated the refund disputes this sheet exists to pre-empt.
//
// Built on <Sheet> rather than <Dialog>/useDialog on purpose: useDialog renders
// title + message + actions only, and a four-line money breakdown collapsed into
// one prose paragraph loses the right-aligned tabular column that lets a
// customer check the figures against their bank statement. <Sheet> is the
// codebase's canonical confirmation surface ("confirming a destructive action …
// cancel order" — Sheet.tsx) and takes arbitrary `children`, so this is reuse of
// the existing primitive, not a bespoke modal.
//
// The vocabulary here is deliberately identical to the POST-cancellation
// breakdown on app/order/[id]/index.tsx ("Platform fee (non-refundable)",
// "GST on the amount retained"): the estimate and the outcome must read as the
// same story, or the second one looks like a different policy.

// The two states the customer can be in, worded from the API's own branch
// (services.ClassifyCancellation): a pending order settles immediately at the
// 100% path, anything the chef has accepted goes to them to tier.
const EXPLAINER_SETTLED =
  "Your chef hasn't accepted this order yet, so nothing has been cooked — everything except the platform fee comes back.";
const EXPLAINER_CHEF_DECIDES =
  "Your chef decides how much of the food comes back when they confirm — up to 90% if they haven't started cooking, less once they have.";

interface CancelRefundSheetProps {
  estimate: CancellationRefundEstimate;
  /** Rupees of wallet + loyalty credit that funded the order. Non-zero means the
   *  refund comes back split pro-rata across the rails that paid for it, which
   *  is the second surprise #1032 records ("the wallet came back short"). The
   *  split itself is NOT quoted: the server derives it at refund time from the
   *  captured payment, and a number invented here would be the very thing this
   *  sheet is meant to stop. */
  creditApplied?: number;
  /** Submits the cancellation request. The sheet dismisses itself first. */
  onConfirm: () => void;
}

export const CancelRefundSheet = forwardRef<SheetHandle, CancelRefundSheetProps>(
  function CancelRefundSheet({ estimate, creditApplied = 0, onConfirm }, ref) {
    const {
      foodPaise,
      deliveryRefundPaise,
      platformFeeKeptPaise,
      taxPaise,
      minRefundPaise,
      maxRefundPaise,
      exact,
    } = estimate;

    const rupees = (paise: number) => formatMoney(paise / 100);

    // Lead with the floor when there is one. On a pickup order cancelled after
    // the chef accepted, the floor is genuinely ₹0 (0% tier, no delivery fee to
    // return) — "You'll get back at least ₹0" is a worse thing to say than
    // naming the ceiling and letting the chef's decision carry the sentence.
    const hasFloor = minRefundPaise > 0;
    const headlineLabel = exact
      ? "You'll get back"
      : hasFloor
        ? "You'll get back at least"
        : 'You could get back up to';
    const headlineAmount = hasFloor || exact ? minRefundPaise : maxRefundPaise;

    return (
      <Sheet
        ref={ref}
        title="Cancel this order?"
        primaryLabel="Cancel order"
        primaryDestructive
        onPrimaryPress={onConfirm}
        cancelLabel="Keep order"
      >
        <View style={styles.content}>
          {headlineAmount > 0 || exact ? (
            <View>
              <Text style={styles.headlineLabel}>{headlineLabel}</Text>
              <Text style={styles.headlineAmount}>{rupees(headlineAmount)}</Text>
              {!exact && hasFloor && maxRefundPaise > minRefundPaise ? (
                <Text style={styles.headlineNote}>
                  Up to {rupees(maxRefundPaise)} if your chef hasn&apos;t started cooking.
                </Text>
              ) : null}
            </View>
          ) : null}

          <Text style={styles.explainer}>{exact ? EXPLAINER_SETTLED : EXPLAINER_CHEF_DECIDES}</Text>

          <View style={styles.rows}>
            <Row
              label="Food"
              amount={rupees(foodPaise)}
              note={exact ? 'Refunded in full' : "Refunded in part — your chef's call"}
            />
            {deliveryRefundPaise > 0 ? (
              <Row
                label="Delivery"
                amount={rupees(deliveryRefundPaise)}
                note="Refunded — no driver has collected this yet"
              />
            ) : null}
            {platformFeeKeptPaise > 0 ? (
              <Row
                label="Platform fee"
                amount={rupees(platformFeeKeptPaise)}
                note="Non-refundable"
              />
            ) : null}
            {taxPaise > 0 ? (
              <Row
                label="GST"
                amount={rupees(taxPaise)}
                note="The GST on whatever is refunded comes back with it; the GST on the amount retained does not"
              />
            ) : null}
          </View>

          <Text style={styles.footnote}>
            The amounts above leave GST out, so what actually lands is a little more. The exact
            figure is confirmed on this order once the cancellation goes through, and the money
            returns to the payment method you used — usually within 5–7 days.
          </Text>

          {creditApplied > 0 ? (
            <Text style={styles.footnote}>
              You paid {formatMoney(creditApplied)} of this order with credit, so the refund is
              split back across the same methods in the same proportion — part to your card, part
              to your wallet.
            </Text>
          ) : null}
        </View>
      </Sheet>
    );
  },
);

function Row({ label, amount, note }: { label: string; amount: string; note: string }) {
  return (
    // One accessibility node per row: a screen reader that reads "Food", "₹300",
    // "Refunded in full" as three unrelated stops makes the breakdown unusable.
    <View style={styles.row} accessible accessibilityLabel={`${label}, ${amount}. ${note}`}>
      <View style={styles.rowTop}>
        <Text style={styles.rowLabel}>{label}</Text>
        <Text style={styles.rowAmount}>{amount}</Text>
      </View>
      <Text style={styles.rowNote}>{note}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  content: { gap: 16 },
  headlineLabel: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft },
  headlineAmount: {
    fontFamily: 'Geist',
    fontSize: 32,
    color: customerColors.charcoal.DEFAULT,
    // Tabular figures on every money value in the app, so columns and headline
    // agree digit for digit.
    fontVariant: ['tabular-nums'],
    marginTop: 2,
  },
  headlineNote: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
    marginTop: 4,
  },
  explainer: {
    fontFamily: 'Inter',
    fontSize: 14,
    lineHeight: 20,
    color: customerColors.charcoal.DEFAULT,
  },
  // Hairline-separated rows, not bordered cards (.impeccable.md §Surface).
  rows: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
  },
  row: {
    paddingVertical: 10,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
    gap: 2,
  },
  rowTop: { flexDirection: 'row', alignItems: 'baseline', justifyContent: 'space-between', gap: 12 },
  rowLabel: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  rowAmount: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  rowNote: { fontFamily: 'Inter', fontSize: 12, lineHeight: 16, color: customerColors.charcoal.soft },
  footnote: {
    fontFamily: 'Inter',
    fontSize: 12,
    lineHeight: 17,
    color: customerColors.charcoal.soft,
  },
});
