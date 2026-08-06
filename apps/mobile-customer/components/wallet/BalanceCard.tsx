import { useState } from 'react';
import { Modal, Platform, Pressable, Text, View } from 'react-native';
import { Award, ChevronRight, Info, Wallet, X } from 'lucide-react-native';

import { customerColors } from '@homechef/mobile-shared/theme';

// Shadow[2] — the only two surfaces on this screen (canvas and card) are both
// pure white, so a hairline border alone is invisible; this is what actually
// separates the card from the page. Lives on an outer, unclipped wrapper
// because `overflow: hidden` (needed to clip the tinted header to the rounded
// corners) would otherwise clip the iOS shadow to nothing.
const cardShadow = {
  shadowColor: '#000000',
  shadowOffset: { width: 0, height: 4 },
  shadowOpacity: 0.1,
  shadowRadius: 12,
  elevation: 4,
} as const;

// BalanceCard — one headline figure, two sources beneath.
//
// A customer holding both wallet credit and loyalty points previously had to
// visit two screens to answer "how much can I spend?". This answers it once and
// then itemises, because the two balances genuinely behave differently: wallet
// credit never expires and has no per-order cap, points expire after a year and
// are capped per order and per month. The ⓘ on each row states that difference
// rather than leaving the customer to discover it at checkout.

export interface BalanceCardProps {
  walletBalance: number;
  pointsBalance: number;
  /** Rupee value of the points, computed server-side from the live redeem rate. */
  pointsValue: number;
  /** Caps for the points explainer, from the live loyalty config. */
  maxRedeemPct?: number;
  monthlyCap?: number;
  expiryDays?: number;
  /** ISO currency code, from the live wallet balance. Defaults to INR. */
  currency?: string;
  onPressWallet?: () => void;
  onPressPoints?: () => void;
}

// Same Intl-based formatting the transaction list uses (app/wallet.tsx), so a
// figure never renders two different ways on one screen — a four-digit refund
// showed as "₹1234.00" here next to "₹1,234.00" in the list below it before
// this matched them up.
function money(n: number, currency = 'INR'): string {
  try {
    return new Intl.NumberFormat('en-IN', { style: 'currency', currency }).format(n);
  } catch {
    return `${currency} ${n.toFixed(2)}`;
  }
}
const points = (n: number) => n.toLocaleString('en-IN');

interface Explainer {
  title: string;
  body: string;
}

function ExplainerSheet({ item, onClose }: { item: Explainer | null; onClose: () => void }) {
  return (
    <Modal visible={!!item} transparent animationType="fade" onRequestClose={onClose}>
      <Pressable
        className="flex-1 justify-end"
        style={{ backgroundColor: '#00000066' }}
        onPress={onClose}
        accessibilityRole="button"
        accessibilityLabel="Close"
      >
        <Pressable
          className="rounded-t-2xl px-5 pb-10 pt-5"
          style={{ backgroundColor: customerColors.canvas }}
          onPress={(e) => e.stopPropagation()}
        >
          <View className="flex-row items-start justify-between gap-4">
            <Text
              className="flex-1 text-[18px]"
              style={{ color: customerColors.charcoal.DEFAULT, fontFamily: 'Inter-SemiBold' }}
            >
              {item?.title}
            </Text>
            <Pressable
              onPress={onClose}
              className="h-11 w-11 items-center justify-center"
              accessibilityRole="button"
              accessibilityLabel="Close"
            >
              <X size={20} color={customerColors.charcoal.soft} />
            </Pressable>
          </View>
          <Text
            className="mt-2 text-[15px]"
            style={{ color: customerColors.charcoal.soft, lineHeight: 22, fontFamily: 'Inter' }}
          >
            {item?.body}
          </Text>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

/** One funding source, on its own bounded row.
 *
 * The two sources previously ran together with a matching ⓘ apiece, which made
 * them read as one list of similar things. They are not similar: one is credit,
 * one is points, and they behave differently at checkout. Each now carries its
 * own icon and sits in a separated row so the difference is visible before the
 * label is read. */
function SourceRow({
  icon,
  amount,
  label,
  caption,
  onInfo,
  onPress,
}: {
  icon: React.ReactNode;
  amount: string;
  label: string;
  caption?: string;
  onInfo: () => void;
  onPress?: () => void;
}) {
  return (
    <Pressable
      onPress={onPress}
      disabled={!onPress}
      className="min-h-[72px] flex-row items-center gap-3 px-4 py-3"
      accessibilityRole={onPress ? 'button' : undefined}
      accessibilityLabel={onPress ? `${label}, ${amount}` : undefined}
    >
      {({ pressed }) => (
        <>
          <View
            className="h-10 w-10 items-center justify-center rounded-full"
            style={{
              backgroundColor: customerColors.surface.soft,
              opacity: pressed && Platform.OS === 'ios' ? 0.6 : 1,
            }}
          >
            {icon}
          </View>

          <View className="flex-1">
            <View className="flex-row items-center gap-1.5">
              <Text
                className="text-[15px]"
                style={{ color: customerColors.charcoal.soft, fontFamily: 'Inter' }}
              >
                {label}
              </Text>
              {/* Its own hit target, so tapping "what is this?" never navigates. */}
              <Pressable
                onPress={onInfo}
                hitSlop={12}
                accessibilityRole="button"
                accessibilityLabel={`About ${label}`}
              >
                <Info size={15} color={customerColors.charcoal.soft} />
              </Pressable>
            </View>
            <Text
              className="mt-0.5 text-[17px]"
              style={{
                color: customerColors.charcoal.DEFAULT,
                fontVariant: ['tabular-nums'],
                fontFamily: 'Inter-SemiBold',
              }}
            >
              {amount}
              {caption ? (
                <Text
                  className="text-[15px]"
                  style={{
                    color: customerColors.charcoal.soft,
                    fontVariant: ['tabular-nums'],
                    fontFamily: 'Inter',
                  }}
                >
                  {'  '}
                  {caption}
                </Text>
              ) : null}
            </Text>
          </View>

          {onPress ? <ChevronRight size={18} color={customerColors.charcoal.soft} /> : null}
        </>
      )}
    </Pressable>
  );
}

export function BalanceCard({
  walletBalance,
  pointsBalance,
  pointsValue,
  maxRedeemPct = 0.1,
  monthlyCap = 300,
  expiryDays = 365,
  currency = 'INR',
  onPressWallet,
  onPressPoints,
}: BalanceCardProps) {
  const [explainer, setExplainer] = useState<Explainer | null>(null);
  const total = walletBalance + pointsValue;
  const totalLabel = money(total, currency);

  return (
    <>
      {/* Outer: shadow + hairline border, unclipped so the shadow can render on iOS. */}
      <View
        className="rounded-2xl border"
        style={{
          backgroundColor: customerColors.canvas,
          borderColor: customerColors.hairline,
          ...cardShadow,
        }}
      >
        {/* Inner: clips the tinted header and rows to the rounded corners. */}
        <View className="overflow-hidden rounded-2xl">
          {/* Headline block — tinted so the total reads as a summary of the rows
              beneath it rather than as another row. One accessible node so a
              screen reader announces "Total balance, ₹X" as a single sentence
              instead of the label and figure as two disconnected fragments. */}
          <View
            className="flex-row items-center px-5 pb-5 pt-5"
            style={{ backgroundColor: customerColors.surface.soft }}
            accessible
            accessibilityLabel={`Total balance, ${totalLabel}`}
          >
            <View className="flex-1">
              <Text
                className="text-[15px]"
                style={{ color: customerColors.charcoal.soft, fontFamily: 'Inter' }}
              >
                Fe3dr balance
              </Text>
              <Text
                className="mt-1 text-[40px]"
                style={{
                  color: customerColors.charcoal.DEFAULT,
                  fontVariant: ['tabular-nums'],
                  letterSpacing: -0.5,
                  fontFamily: 'Geist-Bold',
                }}
              >
                {totalLabel}
              </Text>
            </View>
            {/* The purse marks what this card is at a glance. Coral tint rather
                than a coral fill: it is a label, not something to press. */}
            <View
              className="h-14 w-14 items-center justify-center rounded-full"
              style={{ backgroundColor: customerColors.coral.tint }}
            >
              <Wallet size={26} color={customerColors.coral.DEFAULT} />
            </View>
          </View>

          {/* Sources, each bounded by a hairline. */}
          <View className="h-px" style={{ backgroundColor: customerColors.hairline }} />

          <SourceRow
            icon={<Wallet size={19} color={customerColors.charcoal.DEFAULT} />}
            label="Wallet credit"
            amount={money(walletBalance, currency)}
            onPress={onPressWallet}
            onInfo={() =>
              setExplainer({
                title: 'Wallet credit',
                body:
                  'Credit from refunds, referrals and promotions. It never expires and there is no limit on how much you can use in one order. It can be spent on Fe3dr only — it cannot be withdrawn to a bank account.',
              })
            }
          />

          <View
            className="h-px"
            style={{ backgroundColor: customerColors.hairline, marginLeft: 68 }}
          />

          <SourceRow
            icon={<Award size={19} color={customerColors.coral.DEFAULT} />}
            label="Loyalty points"
            amount={money(pointsValue, currency)}
            caption={`${points(pointsBalance)} pts`}
            onPress={onPressPoints}
            onInfo={() =>
              setExplainer({
                title: 'Loyalty points',
                body: `Points you earn on delivered orders. Unlike wallet credit they expire ${
                  expiryDays === 365 ? 'a year' : `${expiryDays} days`
                } after you earn them, and there are limits on spending: up to ${Math.round(
                  maxRedeemPct * 100,
                )}% of the food total on any one order, and ${money(
                  monthlyCap,
                  currency,
                )} across a rolling 30 days.`,
              })
            }
          />

          <View className="h-px" style={{ backgroundColor: customerColors.hairline }} />

          {/* The one rule a customer must understand about credit, phrased exactly
              as it is on the checkout credits card. */}
          <Text
            className="px-4 py-3 text-[13px]"
            style={{
              color: customerColors.charcoal.soft,
              backgroundColor: customerColors.surface.soft,
              fontFamily: 'Inter',
            }}
          >
            Usable on food &amp; delivery. Fees and taxes are paid separately.
          </Text>
        </View>
      </View>

      <ExplainerSheet item={explainer} onClose={() => setExplainer(null)} />
    </>
  );
}
