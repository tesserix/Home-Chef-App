import { useState } from 'react';
import { Modal, Platform, Pressable, Text, View } from 'react-native';
import { Award, ChevronRight, Info, Wallet, X } from 'lucide-react-native';

import { customerColors } from '@homechef/mobile-shared/theme';

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
  onPressWallet?: () => void;
  onPressPoints?: () => void;
}

const money = (n: number) => `₹${n.toFixed(2)}`;
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
              className="flex-1 text-[18px] font-semibold"
              style={{ color: customerColors.charcoal.DEFAULT }}
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
            style={{ color: customerColors.charcoal.soft, lineHeight: 22 }}
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
                style={{ color: customerColors.charcoal.soft }}
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
              className="mt-0.5 text-[17px] font-semibold"
              style={{
                color: customerColors.charcoal.DEFAULT,
                fontVariant: ['tabular-nums'],
              }}
            >
              {amount}
              {caption ? (
                <Text
                  className="text-[15px] font-normal"
                  style={{ color: customerColors.charcoal.soft }}
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
  onPressWallet,
  onPressPoints,
}: BalanceCardProps) {
  const [explainer, setExplainer] = useState<Explainer | null>(null);
  const total = walletBalance + pointsValue;

  return (
    <>
      <View
        className="overflow-hidden rounded-2xl border"
        style={{
          backgroundColor: customerColors.canvas,
          borderColor: customerColors.hairline,
        }}
      >
        {/* Headline block — tinted so the total reads as a summary of the rows
            beneath it rather than as another row. */}
        <View
          className="flex-row items-center px-5 pb-5 pt-5"
          style={{ backgroundColor: customerColors.surface.soft }}
        >
          <View className="flex-1">
            <Text className="text-[15px]" style={{ color: customerColors.charcoal.soft }}>
              Fe3dr balance
            </Text>
            <Text
              className="mt-1 text-[40px] font-bold"
              style={{
                color: customerColors.charcoal.DEFAULT,
                fontVariant: ['tabular-nums'],
                letterSpacing: -0.5,
              }}
            >
              {money(total)}
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
          amount={money(walletBalance)}
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
          amount={money(pointsValue)}
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
              )} across a rolling 30 days.`,
            })
          }
        />

        <View className="h-px" style={{ backgroundColor: customerColors.hairline }} />

        {/* The one rule a customer must understand about credit, phrased exactly
            as it is on the checkout credits card. */}
        <Text
          className="px-4 py-3 text-[13px]"
          style={{ color: customerColors.charcoal.soft, backgroundColor: customerColors.surface.soft }}
        >
          Usable on food &amp; delivery. Fees and taxes are paid separately.
        </Text>
      </View>

      <ExplainerSheet item={explainer} onClose={() => setExplainer(null)} />
    </>
  );
}
