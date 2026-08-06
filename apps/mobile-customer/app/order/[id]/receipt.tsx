// Order receipt / tax invoice (#receipt). Customers asked to view the receipt
// for an order they placed and paid for — including a cancelled+refunded one,
// which the order detail shows the money for but offered no document.
//
// Rendered from the order the app already has (no persisting invoice row — that
// stays a delivered-only, tax-numbered record on the backend). The heading
// mirrors the backend PDF's rule exactly: a TAX INVOICE only for a completed,
// unrefunded delivery; a PAYMENT RECEIPT otherwise — so the app and the
// downloadable PDF never disagree about what the document is.

import { useState } from 'react';
import { ActivityIndicator, Platform, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { Image } from 'expo-image';
import * as WebBrowser from 'expo-web-browser';
import { ChevronLeft, Download, Share2 } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useOrder, fetchInvoiceDownloadUrl } from '../../../hooks/useOrderHistory';
import { useAlert } from '@homechef/mobile-shared/ui';
import { receiptFileName } from '../../../lib/receipt-file';
import { shareReceiptPdf } from '../../../lib/share-pdf';

// Android ripple tints — translucent tokens, never a new literal colour.
const ICON_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const PDF_RIPPLE = `${customerColors.canvas}33`;

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
  const { showAlert } = useAlert();
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { data, isLoading, isError } = useOrder(id ?? '');
  const order = data?.data;
  const [openingPdf, setOpeningPdf] = useState(false);
  const [sharing, setSharing] = useState(false);

  // Open the official PDF (the same document web downloads) in the in-app
  // browser, via a short-lived signed URL — iOS then offers save/share/print.
  // This is how mobile reaches parity without a file-system module.
  async function onOpenPdf() {
    if (!order || openingPdf) return;
    setOpeningPdf(true);
    try {
      const url = await fetchInvoiceDownloadUrl(order.id);
      await WebBrowser.openBrowserAsync(url);
    } catch {
      showAlert(
        "Couldn't open the PDF",
        'The receipt could not be opened right now. Please try again in a moment.',
      );
    } finally {
      setOpeningPdf(false);
    }
  }

  // Same rule as the backend PDF: a tax invoice is only for a completed sale.
  const isTaxInvoice = order?.status === 'delivered' && (order?.refundAmount ?? 0) <= 0;
  const docTitle = isTaxInvoice ? 'Tax Invoice' : 'Payment Receipt';

  // Sharing sends the issued PDF, never a re-typed text copy: a plain-text
  // summary can drift from the tax document it claims to be.
  async function onShare() {
    if (!order || sharing) return;
    setSharing(true);
    try {
      const url = await fetchInvoiceDownloadUrl(order.id);
      await shareReceiptPdf(url, receiptFileName(order.orderNumber, isTaxInvoice));
    } catch {
      showAlert(
        "Couldn't share the PDF",
        'The document could not be prepared right now. Please try again in a moment.',
      );
    } finally {
      setSharing(false);
    }
  }

  // Robust deliver-to lines: skip empty fields so we never render a stray comma
  // (the old bug), and drop the whole block when no address is present.
  const addr = order?.deliveryAddress;
  const deliveryAddressLines = [
    [addr?.addressLine1, addr?.addressLine2].filter(Boolean).join(', '),
    [[addr?.city, addr?.state].filter(Boolean).join(', '), addr?.pincode]
      .filter(Boolean)
      .join(' '),
  ]
    .map((l) => l.trim())
    .filter((l) => l.length > 0);

  // Every figure below comes from the API's one breakdown (models/pricing.go),
  // including the CGST/SGST split and the rounding row that makes the lines reach
  // the total. This screen used to do its own arithmetic: it printed lines that
  // summed to a paise MORE than the total underneath them, and compared the two
  // state spellings as raw strings, so it could say IGST where the PDF said
  // CGST+SGST for the same order.
  const subtotal =
    order?.subtotal ?? (order?.items ?? []).reduce((s, it) => s + it.price * it.quantity, 0);
  const deliveryFee = order?.deliveryFee ?? 0;
  const platformFee = order?.platformFee ?? 0;
  const tip = order?.tip ?? 0;
  const taxLines = order?.taxLines ?? [];
  const rounding = order?.rounding ?? 0;
  const discount = order?.discount ?? 0;
  const refund = order?.refundAmount ?? 0;

  // What actually left the customer's payment method (#1027). A receipt headed
  // "PAYMENT RECEIPT" whose Total is the gross order value states an amount the
  // customer never paid: wallet and loyalty credit are settled before the
  // gateway is charged. Same fields and same arithmetic as the order-detail
  // screen (`order/[id]/index.tsx`), so the two screens can never disagree.
  const walletApplied = order?.walletApplied ?? 0;
  const loyaltyApplied = order?.loyaltyApplied ?? 0;
  const creditApplied = walletApplied + loyaltyApplied;
  const amountCharged = Math.max((order?.totalAmount ?? 0) - creditApplied, 0);

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={10}
          accessibilityLabel="Go back"
          accessibilityRole="button"
          android_ripple={{ color: ICON_RIPPLE, borderless: true, radius: 20 }}
        >
          {({ pressed }) => (
            <View style={pressed && Platform.OS === 'ios' ? styles.iconPressed : undefined}>
              <ChevronLeft size={26} color={customerColors.charcoal.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text style={styles.headerTitle}>Receipt</Text>
        {order ? (
          <Pressable
            onPress={onShare}
            disabled={sharing}
            hitSlop={10}
            accessibilityLabel="Share receipt as PDF"
            accessibilityRole="button"
            accessibilityState={{ disabled: sharing }}
            android_ripple={{ color: ICON_RIPPLE, borderless: true, radius: 20 }}
          >
            {({ pressed }) =>
              sharing ? (
                <ActivityIndicator size="small" color={customerColors.charcoal.DEFAULT} />
              ) : (
                <View style={pressed && Platform.OS === 'ios' ? styles.iconPressed : undefined}>
                  <Share2 size={22} color={customerColors.charcoal.DEFAULT} />
                </View>
              )
            }
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
            {/* Masthead: brand lockup on the left, document type on the right,
                both on one baseline so the two never fight for the eye. */}
            <View style={styles.masthead}>
              <View style={styles.brandLockup}>
                <Image
                  source={require('../../../assets/icon.png')}
                  style={styles.brandMark}
                  contentFit="cover"
                  accessibilityLabel="Fe3dr"
                />
                <Text style={styles.brand}>Fe3dr</Text>
              </View>
              <Text style={styles.docTitle}>{docTitle.toUpperCase()}</Text>
            </View>

            {/* The amount is what the document is FOR, so it leads — the order
                number and date sit beside it as reference, not as the headline. */}
            <View style={styles.amountBlock}>
              <Text style={styles.amountLabel}>{refund > 0 ? 'Amount paid' : 'Total paid'}</Text>
              <Text style={styles.amount}>{money(amountCharged)}</Text>
            </View>

            <View style={styles.metaGrid}>
              <View style={styles.metaCell}>
                <Text style={styles.metaLabel}>ORDER</Text>
                <Text style={styles.metaValue}>#{order.orderNumber}</Text>
              </View>
              <View style={[styles.metaCell, styles.metaCellRight]}>
                <Text style={styles.metaLabel}>DATE</Text>
                <Text style={styles.metaValue}>{formatDateTime(order.createdAt)}</Text>
              </View>
            </View>

            {!isTaxInvoice ? (
              <Text style={styles.notTaxNote}>This is a payment receipt, not a tax invoice.</Text>
            ) : null}

            <View style={styles.rule} />

            {/* Parties — official seller block: business, proprietor, FSSAI, GSTIN */}
            {order.chef ? (
              <View style={styles.party}>
                <Text style={styles.partyLabel}>SOLD BY</Text>
                <Text style={styles.sellerName}>
                  {order.chef.businessName || order.chef.name}
                </Text>
                {order.chef.ownerName ? (
                  <Text style={styles.partyValue}>Chef {order.chef.ownerName}</Text>
                ) : null}
                {order.chef.fssaiLicenseNumber ? (
                  <Text style={styles.regLine}>FSSAI Lic. No. {order.chef.fssaiLicenseNumber}</Text>
                ) : null}
                {order.chef.gstin ? (
                  <Text style={styles.regLine}>GSTIN {order.chef.gstin}</Text>
                ) : null}
              </View>
            ) : null}

            {/* Deliver to (delivery orders) / Pickup (collection orders) */}
            {order.fulfillmentType === 'pickup' ? (
              <View style={styles.party}>
                <Text style={styles.partyLabel}>FULFILMENT</Text>
                <Text style={styles.partyValue}>Pickup from the kitchen</Text>
              </View>
            ) : deliveryAddressLines.length > 0 ? (
              <View style={styles.party}>
                <Text style={styles.partyLabel}>DELIVER TO</Text>
                {deliveryAddressLines.map((l, i) => (
                  <Text key={i} style={styles.partyValue}>
                    {l}
                  </Text>
                ))}
              </View>
            ) : null}

            <View style={styles.rule} />

            {/* Items — the quantity sits in its own gutter so names start on one
                left edge and the amounts stay a clean right-hand column. */}
            <Text style={styles.sectionLabel}>ITEMS</Text>
            {(order.items ?? []).map((it, i) => (
              <View key={`${it.menuItemId}-${i}`} style={styles.itemRow}>
                <Text style={styles.itemQty}>{it.quantity}×</Text>
                <View style={styles.itemBody}>
                  <Text style={styles.itemName} numberOfLines={2}>
                    {it.name}
                  </Text>
                  {it.quantity > 1 ? (
                    <Text style={styles.itemUnit}>{money(it.price)} each</Text>
                  ) : null}
                </View>
                <Text style={styles.itemAmount}>{money(it.price * it.quantity)}</Text>
              </View>
            ))}

            <View style={styles.rule} />

            {/* Totals */}
            <Line label="Subtotal" value={money(subtotal)} />
            {deliveryFee > 0 ? <Line label="Delivery" value={money(deliveryFee)} /> : null}
            {platformFee > 0 ? <Line label="Platform fee" value={money(platformFee)} /> : null}
            {taxLines.map((t) => (
              <Line key={t.code} label={t.label} value={money(t.amount)} />
            ))}
            {discount > 0 ? <Line label="Discount" value={`-${money(discount)}`} /> : null}
            {tip > 0 ? <Line label="Tip" value={money(tip)} /> : null}
            {rounding !== 0 ? <Line label="Rounding" value={money(rounding)} /> : null}

            <View style={styles.totalRule} />
            <Line label="Total" value={money(order.totalAmount)} bold />
            {/* Credit settled before the gateway, then the figure that was
                actually charged (#1027). Only rendered when credit was applied,
                so an ordinary order keeps the one bold Total it had. */}
            {walletApplied > 0.005 ? (
              <Line label="Wallet credit" value={`-${money(walletApplied)}`} refund />
            ) : null}
            {loyaltyApplied > 0.005 ? (
              <Line label="Loyalty points" value={`-${money(loyaltyApplied)}`} refund />
            ) : null}
            {creditApplied > 0.005 ? (
              <Line label="Paid by card / UPI" value={money(amountCharged)} bold />
            ) : null}
            {refund > 0 ? (
              <Line label="Refunded" value={`-${money(refund)}`} refund />
            ) : null}

            <View style={styles.rule} />
            <Text style={styles.footer}>
              This is a computer-generated document and does not require a physical
              signature.
            </Text>
            <Text style={styles.footerBrand}>Fe3dr Marketplace · www.fe3dr.com</Text>
          </View>

          {/* The official PDF — same document the web downloads. Opens in the
              in-app browser (iOS save/share/print), the mobile route to parity. */}
          <Pressable
            onPress={onOpenPdf}
            disabled={openingPdf}
            accessibilityRole="button"
            accessibilityLabel="Open the PDF receipt"
            android_ripple={openingPdf ? undefined : { color: PDF_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.pdfBtn,
                  pressed && Platform.OS === 'ios' && !openingPdf && styles.pdfBtnPressed,
                ]}
              >
                {openingPdf ? (
                  <ActivityIndicator size="small" color={customerColors.canvas} />
                ) : (
                  <>
                    <Download size={18} color={customerColors.canvas} />
                    <Text style={styles.pdfBtnText}>Open PDF receipt</Text>
                  </>
                )}
              </View>
            )}
          </Pressable>
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
      {/* Money coming BACK is green, and only the figure carries the colour —
          the label stays charcoal. Both of those match the order-detail
          breakdown, which is the screen a customer compares this against; the
          receipt was reading them in coral, so the same refund looked like a
          warning here and a credit there. */}
      <Text style={[styles.totalLabel, bold && styles.totalBold]}>{label}</Text>
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
  iconPressed: { opacity: 0.6 },
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
  masthead: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 12,
  },
  brandLockup: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  brandMark: { width: 28, height: 28, borderRadius: 8 },
  docTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    letterSpacing: 0.8,
    textAlign: 'right',
    flexShrink: 1,
  },
  brand: { fontFamily: 'Geist-Bold', fontSize: 20, color: customerColors.charcoal.DEFAULT, letterSpacing: -0.2 },
  amountBlock: { marginTop: 20 },
  amountLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    letterSpacing: 0.6,
    textTransform: 'uppercase',
    color: customerColors.charcoal.soft,
  },
  amount: {
    fontFamily: 'Geist-Bold',
    fontSize: 32,
    lineHeight: 38,
    marginTop: 2,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
    letterSpacing: -0.5,
  },
  metaGrid: { flexDirection: 'row', gap: 16, marginTop: 16 },
  metaCell: { flex: 1 },
  metaCellRight: { alignItems: 'flex-end' },
  metaLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    letterSpacing: 0.6,
    color: customerColors.charcoal.soft,
  },
  metaValue: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.DEFAULT,
    marginTop: 2,
    fontVariant: ['tabular-nums'],
  },
  notTaxNote: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    marginTop: 12,
  },
  rule: { height: StyleSheet.hairlineWidth, backgroundColor: customerColors.hairline, marginVertical: 16 },
  totalRule: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: customerColors.hairline,
    marginTop: 10,
    marginBottom: 8,
  },
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    letterSpacing: 0.6,
    color: customerColors.charcoal.soft,
    marginBottom: 8,
  },
  party: { marginBottom: 16 },
  partyLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    color: customerColors.charcoal.soft,
    letterSpacing: 0.6,
  },
  partyValue: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.DEFAULT, marginTop: 3, lineHeight: 20 },
  sellerName: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT, marginTop: 4, lineHeight: 20 },
  regLine: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft, marginTop: 3, letterSpacing: 0.2 },
  itemRow: { flexDirection: 'row', alignItems: 'flex-start', gap: 10, paddingVertical: 6 },
  itemQty: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    minWidth: 28,
    fontVariant: ['tabular-nums'],
  },
  itemBody: { flex: 1 },
  itemName: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.DEFAULT, lineHeight: 20 },
  itemUnit: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft, marginTop: 1 },
  itemAmount: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  totalRow: { flexDirection: 'row', justifyContent: 'space-between', gap: 12, paddingVertical: 4 },
  totalLabel: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft },
  totalValue: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.DEFAULT, fontVariant: ['tabular-nums'] },
  totalBold: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  refundText: { color: customerColors.success.DEFAULT },
  footer: { fontFamily: 'Inter', fontSize: 11, color: customerColors.charcoal.soft, textAlign: 'center', lineHeight: 16 },
  footerBrand: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
    marginTop: 4,
  },
  pdfBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    marginTop: 16,
    minHeight: 48,
    borderRadius: 12,
    backgroundColor: customerColors.coral.DEFAULT,
  },
  pdfBtnText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.canvas },
  pdfBtnPressed: { backgroundColor: customerColors.coral.pressed },
});
