// Order receipt / tax invoice (#receipt). Customers asked to view the receipt
// for an order they placed and paid for — including a cancelled+refunded one,
// which the order detail shows the money for but offered no document.
//
// Rendered from the order the app already has (no persisting invoice row — that
// stays a delivered-only, tax-numbered record on the backend). The heading
// mirrors the backend PDF's rule exactly: a TAX INVOICE only for a completed,
// unrefunded delivery; a PAYMENT RECEIPT otherwise — so the app and the
// downloadable PDF never disagree about what the document is.

import { Share, ScrollView, StyleSheet, Text, View, Pressable } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { ChevronLeft, Share2 } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useOrder } from '../../../hooks/useOrderHistory';

function money(n: number): string {
  return `₹${n.toFixed(2)}`;
}

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export default function OrderReceiptScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { data, isLoading, isError } = useOrder(id ?? '');
  const order = data?.data;

  // Same rule as the backend PDF: a tax invoice is only for a completed sale.
  const isTaxInvoice = order?.status === 'delivered' && (order?.refundAmount ?? 0) <= 0;
  const docTitle = isTaxInvoice ? 'Tax Invoice' : 'Payment Receipt';

  const subtotal = (order?.items ?? []).reduce((s, it) => s + it.price * it.quantity, 0);
  const deliveryFee = order?.deliveryFee ?? 0;
  const serviceFee = order?.serviceFee ?? 0;
  const tax = order?.tax ?? 0;
  const discount = order?.discount ?? 0;
  const refund = order?.refundAmount ?? 0;

  async function onShare() {
    if (!order) return;
    // No file-system/PDF export dep in this app, so share a plain-text receipt
    // via the OS sheet — enough to forward or save the record. The formal PDF is
    // served by the API for anyone who needs the tax document.
    const lines = [
      `${docTitle} — Home Chef`,
      `Order ${order.orderNumber}`,
      order.chef?.name ? `Kitchen: ${order.chef.name}` : '',
      `Date: ${formatDateTime(order.createdAt)}`,
      '',
      ...(order.items ?? []).map((it) => `${it.quantity} × ${it.name}  ${money(it.price * it.quantity)}`),
      '',
      `Subtotal: ${money(subtotal)}`,
      deliveryFee > 0 ? `Delivery: ${money(deliveryFee)}` : '',
      serviceFee > 0 ? `Service fee: ${money(serviceFee)}` : '',
      tax > 0 ? `Tax: ${money(tax)}` : '',
      discount > 0 ? `Discount: -${money(discount)}` : '',
      `Total: ${money(order.totalAmount)}`,
      refund > 0 ? `Refunded: -${money(refund)}` : '',
    ].filter(Boolean);
    try {
      await Share.share({ message: lines.join('\n') });
    } catch {
      // User dismissed the share sheet — nothing to do.
    }
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable onPress={() => router.back()} hitSlop={10} accessibilityLabel="Go back" accessibilityRole="button">
          <ChevronLeft size={26} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.headerTitle}>Receipt</Text>
        {order ? (
          <Pressable onPress={onShare} hitSlop={10} accessibilityLabel="Share receipt" accessibilityRole="button">
            <Share2 size={22} color={customerColors.charcoal.DEFAULT} />
          </Pressable>
        ) : (
          <View style={{ width: 22 }} />
        )}
      </View>

      {isLoading ? (
        <View style={styles.centered}><Text style={styles.muted}>Loading…</Text></View>
      ) : isError || !order ? (
        <View style={styles.centered}><Text style={styles.muted}>Receipt not available.</Text></View>
      ) : (
        <ScrollView contentContainerStyle={styles.scroll}>
          <View style={styles.doc}>
            {/* Masthead */}
            <View style={styles.masthead}>
              <Text style={styles.docTitle}>{docTitle.toUpperCase()}</Text>
              <Text style={styles.brand}>Home Chef</Text>
            </View>
            <Text style={styles.meta}>Order #{order.orderNumber}</Text>
            <Text style={styles.meta}>{formatDateTime(order.createdAt)}</Text>
            {!isTaxInvoice ? (
              <Text style={styles.notTaxNote}>This is a payment receipt, not a tax invoice.</Text>
            ) : null}

            <View style={styles.rule} />

            {/* Parties */}
            {order.chef?.name ? (
              <View style={styles.party}>
                <Text style={styles.partyLabel}>From</Text>
                <Text style={styles.partyValue}>{order.chef.name}</Text>
              </View>
            ) : null}
            {order.deliveryAddress ? (
              <View style={styles.party}>
                <Text style={styles.partyLabel}>Deliver to</Text>
                <Text style={styles.partyValue}>
                  {[order.deliveryAddress.addressLine1, order.deliveryAddress.addressLine2]
                    .filter(Boolean)
                    .join(', ')}
                  {'\n'}
                  {order.deliveryAddress.city}, {order.deliveryAddress.state}{' '}
                  {order.deliveryAddress.pincode}
                </Text>
              </View>
            ) : null}

            <View style={styles.rule} />

            {/* Items */}
            {(order.items ?? []).map((it, i) => (
              <View key={`${it.menuItemId}-${i}`} style={styles.itemRow}>
                <Text style={styles.itemName} numberOfLines={2}>
                  {it.quantity} × {it.name}
                </Text>
                <Text style={styles.itemAmount}>{money(it.price * it.quantity)}</Text>
              </View>
            ))}

            <View style={styles.rule} />

            {/* Totals */}
            <Line label="Subtotal" value={money(subtotal)} />
            {deliveryFee > 0 ? <Line label="Delivery" value={money(deliveryFee)} /> : null}
            {serviceFee > 0 ? <Line label="Service fee" value={money(serviceFee)} /> : null}
            {tax > 0 ? <Line label={'Tax'} value={money(tax)} /> : null}
            {discount > 0 ? <Line label="Discount" value={`-${money(discount)}`} /> : null}
            <Line label="Total" value={money(order.totalAmount)} bold />
            {refund > 0 ? (
              <Line label="Refunded" value={`-${money(refund)}`} refund />
            ) : null}

            <View style={styles.rule} />
            <Text style={styles.footer}>
              This is a computer-generated document and does not require a physical
              signature.
            </Text>
          </View>
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

function Line({
  label,
  value,
  bold,
  refund,
}: {
  label: string;
  value: string;
  bold?: boolean;
  refund?: boolean;
}) {
  return (
    <View style={styles.totalRow}>
      <Text style={[styles.totalLabel, bold && styles.totalBold, refund && styles.refundText]}>
        {label}
      </Text>
      <Text
        style={[styles.totalValue, bold && styles.totalBold, refund && styles.refundText]}
      >
        {value}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.surface.soft },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  headerTitle: { fontFamily: 'Geist-Bold', fontSize: 20, color: customerColors.charcoal.DEFAULT },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  muted: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft },
  scroll: { padding: 16 },
  doc: {
    backgroundColor: customerColors.canvas,
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    padding: 20,
  },
  masthead: { flexDirection: 'row', alignItems: 'baseline', justifyContent: 'space-between' },
  docTitle: { fontFamily: 'Geist-Bold', fontSize: 18, color: customerColors.charcoal.DEFAULT, letterSpacing: 0.5 },
  brand: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  meta: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, marginTop: 2 },
  notTaxNote: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    marginTop: 8,
  },
  rule: { height: StyleSheet.hairlineWidth, backgroundColor: customerColors.hairline, marginVertical: 14 },
  party: { marginBottom: 10 },
  partyLabel: { fontFamily: 'Inter-SemiBold', fontSize: 11, color: customerColors.charcoal.soft, textTransform: 'uppercase', letterSpacing: 0.4 },
  partyValue: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.DEFAULT, marginTop: 2, lineHeight: 20 },
  itemRow: { flexDirection: 'row', justifyContent: 'space-between', gap: 12, paddingVertical: 5 },
  itemName: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.DEFAULT, flex: 1 },
  itemAmount: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.DEFAULT, fontVariant: ['tabular-nums'] },
  totalRow: { flexDirection: 'row', justifyContent: 'space-between', paddingVertical: 3 },
  totalLabel: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft },
  totalValue: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.DEFAULT, fontVariant: ['tabular-nums'] },
  totalBold: { fontFamily: 'Inter-SemiBold', color: customerColors.charcoal.DEFAULT },
  refundText: { color: customerColors.coral.pressed },
  footer: { fontFamily: 'Inter', fontSize: 11, color: customerColors.charcoal.soft, textAlign: 'center', lineHeight: 16 },
});
