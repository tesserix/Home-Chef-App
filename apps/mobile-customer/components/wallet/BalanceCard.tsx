import { useState } from 'react';
import { Modal, Platform, Pressable, Text, View } from 'react-native';
import { ChevronRight, Info, X } from 'lucide-react-native';

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

/** One itemised source row: ⓘ, amount, label, and an optional chevron. */
function SourceRow({
  amount,
  label,
  onInfo,
  onPress,
}: {
  amount: string;
  label: string;
  onInfo: () => void;
  onPress?: () => void;
}) {
  return (
    <View className="flex-row items-center">
      <Pressable
        onPress={onInfo}
        className="h-11 w-11 items-center justify-center"
        accessibilityRole="button"
        accessibilityLabel={`About ${label}`}
        hitSlop={4}
      >
        <Info size={18} color={customerColors.charcoal.soft} />
      </Pressable>
      <Pressable
        onPress={onPress}
        disabled={!onPress}
        className="min-h-[44px] flex-1 flex-row items-center justify-between"
        accessibilityRole={onPress ? 'button' : undefined}
        accessibilityLabel={onPress ? `${amount} ${label}` : undefined}
      >
        {({ pressed }) => (
          <>
            <Text
              className="text-[16px]"
              style={{
                color: customerColors.charcoal.DEFAULT,
                fontVariant: ['tabular-nums'],
                opacity: pressed && Platform.OS === 'ios' ? 0.6 : 1,
              }}
            >
              {amount} — {label}
            </Text>
            {onPress ? <ChevronRight size={18} color={customerColors.charcoal.soft} /> : null}
          </>
        )}
      </Pressable>
    </View>
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
        className="rounded-2xl px-5 pb-5 pt-4"
        style={{ backgroundColor: customerColors.surface.soft }}
      >
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

        <View className="mt-3 gap-0.5">
          <SourceRow
            amount={money(walletBalance)}
            label="Wallet credit"
            onPress={onPressWallet}
            onInfo={() =>
              setExplainer({
                title: 'Wallet credit',
                body:
                  'Credit from refunds, referrals and promotions. It never expires and there is no limit on how much you can use in one order. It can be spent on Fe3dr only — it cannot be withdrawn to a bank account.',
              })
            }
          />
          <SourceRow
            amount={money(pointsValue)}
            label={`${points(pointsBalance)} loyalty points`}
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
        </View>

        {/* The one rule a customer must understand about credit, phrased exactly
            as it is on the checkout credits card. */}
        <Text className="mt-3 text-[13px]" style={{ color: customerColors.charcoal.soft }}>
          Usable on food &amp; delivery. Fees and taxes are paid separately.
        </Text>
      </View>

      <ExplainerSheet item={explainer} onClose={() => setExplainer(null)} />
    </>
  );
}
