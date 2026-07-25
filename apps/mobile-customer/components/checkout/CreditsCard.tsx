import { useState } from 'react';
import { Pressable, Text, TextInput, View } from 'react-native';
import { Check } from 'lucide-react-native';

import { customerColors } from '@homechef/mobile-shared/theme';
import type { CreditQuote, LoyaltyLimit } from '../../hooks/useDeliveryQuote';
import { CreditSlider } from './CreditSlider';

// CreditsCard — "Pay with your credits".
//
// Replaces a bare checkbox that sat between the promo field and the total, where
// the owner missed it entirely while placing a live order. Credit is now a titled
// section of its own, above Price Details.
//
// Every figure here comes from the server quote. The card renders; it does not
// calculate. That is the whole point: the previous screen computed the payable
// from a cached balance and posted it, so any drift showed one number and charged
// another.

export interface CreditsCardProps {
  quote: CreditQuote;
  useWallet: boolean;
  useLoyalty: boolean;
  onChange: (next: {
    useWallet: boolean;
    walletAmount?: number;
    useLoyalty: boolean;
    loyaltyPoints?: number;
  }) => void;
}

const money = (n: number) => `₹${n.toFixed(2)}`;
const points = (n: number) => n.toLocaleString('en-IN');

/** The loyalty caption explains WHY the row is capped, using the server's reason
 *  rather than re-deriving the cap and risking a different answer. */
function loyaltyCaption(quote: CreditQuote, limit: LoyaltyLimit): string {
  switch (limit) {
    case 'per_order_cap':
      return `worth ${money(quote.pointsBalanceValue)} · up to ${money(
        quote.pointsMaxValue,
      )} here`;
    case 'monthly_cap':
      return `${money(quote.pointsMaxValue)} of your monthly limit left`;
    case 'order_covered':
      return 'your wallet already covers this order';
    default:
      return `worth ${money(quote.pointsMaxValue)}`;
  }
}

function Toggle({ on, label }: { on: boolean; label: string }) {
  return (
    <View
      accessibilityRole="checkbox"
      accessibilityState={{ checked: on }}
      accessibilityLabel={label}
      className="h-6 w-6 items-center justify-center rounded-md border-2"
      style={{
        borderColor: on ? customerColors.coral.DEFAULT : customerColors.hairline,
        backgroundColor: on ? customerColors.coral.DEFAULT : customerColors.canvas,
      }}
    >
      {on && <Check size={14} color={customerColors.canvas} />}
    </View>
  );
}

export function CreditsCard({ quote, useWallet, useLoyalty, onChange }: CreditsCardProps) {
  const [walletText, setWalletText] = useState<string | null>(null);
  const [pointsText, setPointsText] = useState<string | null>(null);

  const walletShown = useWallet ? quote.walletApplied : 0;
  const pointsShown = useLoyalty ? quote.pointsApplied : 0;
  const creditApplied = walletShown + quote.pointsValue * (useLoyalty ? 1 : 0);

  const walletUsable = quote.walletEnabled && quote.walletMax > 0;
  const loyaltyUsable = quote.loyaltyEnabled && quote.pointsMax > 0;
  if (!walletUsable && !loyaltyUsable) return null;

  // Touching EITHER control pins BOTH rails to explicit values. Without this,
  // dragging the wallet down would let loyalty silently expand into the gap and
  // spend points the customer was deliberately preserving.
  const pin = (over: Partial<{ useWallet: boolean; walletAmount: number; useLoyalty: boolean; loyaltyPoints: number }>) =>
    onChange({
      useWallet,
      useLoyalty,
      walletAmount: quote.walletApplied,
      loyaltyPoints: quote.pointsApplied,
      ...over,
    });

  return (
    <View className="border-b" style={{ borderColor: customerColors.hairline }}>
      <View className="px-4 pb-3 pt-5">
        <Text
          className="text-[17px] font-semibold"
          style={{ color: customerColors.charcoal.DEFAULT }}
        >
          Pay with your credits
        </Text>
      </View>

      {walletUsable && (
        <View className="px-4 pb-4">
          <Pressable
            onPress={() => pin({ useWallet: !useWallet, walletAmount: undefined })}
            className="min-h-[44px] flex-row items-center justify-between"
            accessibilityRole="button"
            accessibilityLabel="Use wallet credit"
          >
            <View className="flex-1 flex-row items-center gap-3">
              <Toggle on={useWallet} label="Use wallet credit" />
              <View className="flex-1">
                <Text
                  className="text-[15px] font-medium"
                  style={{ color: customerColors.charcoal.DEFAULT }}
                >
                  Wallet credit
                </Text>
                <Text
                  className="text-[13px]"
                  style={{ color: customerColors.charcoal.soft, fontVariant: ['tabular-nums'] }}
                >
                  Balance {money(quote.walletBalance)}
                </Text>
              </View>
            </View>
            <Text
              className="text-[15px] font-semibold"
              style={{
                color: useWallet ? customerColors.success.DEFAULT : customerColors.charcoal.soft,
                fontVariant: ['tabular-nums'],
              }}
            >
              {useWallet ? `−${money(walletShown)}` : money(0)}
            </Text>
          </Pressable>

          {useWallet && (
            <View className="flex-row items-center gap-3">
              <View className="flex-1">
                <CreditSlider
                  value={walletShown}
                  max={quote.walletMax}
                  step={1}
                  accessibilityLabel="Wallet credit to apply"
                  onChange={(v) => pin({ walletAmount: v })}
                  onCommit={(v) => pin({ walletAmount: v })}
                />
              </View>
              <TextInput
                value={walletText ?? String(Math.round(walletShown))}
                onChangeText={setWalletText}
                onBlur={() => {
                  const v = Number(walletText);
                  setWalletText(null);
                  if (!Number.isNaN(v)) pin({ walletAmount: Math.max(0, Math.min(quote.walletMax, v)) });
                }}
                keyboardType="number-pad"
                accessibilityLabel="Wallet amount"
                className="h-11 w-24 rounded-lg px-3 text-right text-[15px]"
                style={{
                  backgroundColor: customerColors.surface.soft,
                  color: customerColors.charcoal.DEFAULT,
                  fontVariant: ['tabular-nums'],
                }}
              />
            </View>
          )}
        </View>
      )}

      {loyaltyUsable && (
        <View className="px-4 pb-4">
          <Pressable
            onPress={() => pin({ useLoyalty: !useLoyalty, loyaltyPoints: undefined })}
            className="min-h-[44px] flex-row items-center justify-between"
            accessibilityRole="button"
            accessibilityLabel="Use loyalty points"
          >
            <View className="flex-1 flex-row items-center gap-3">
              <Toggle on={useLoyalty} label="Use loyalty points" />
              <View className="flex-1">
                <Text
                  className="text-[15px] font-medium"
                  style={{ color: customerColors.charcoal.DEFAULT }}
                >
                  Loyalty points
                </Text>
                <Text
                  className="text-[13px]"
                  style={{ color: customerColors.charcoal.soft, fontVariant: ['tabular-nums'] }}
                >
                  {points(quote.pointsBalance)} pts · {loyaltyCaption(quote, quote.loyaltyLimit)}
                </Text>
              </View>
            </View>
            <Text
              className="text-[15px] font-semibold"
              style={{
                color: useLoyalty ? customerColors.success.DEFAULT : customerColors.charcoal.soft,
                fontVariant: ['tabular-nums'],
              }}
            >
              {useLoyalty ? `−${money(quote.pointsValue)}` : money(0)}
            </Text>
          </Pressable>

          {useLoyalty && (
            <View className="flex-row items-center gap-3">
              <View className="flex-1">
                <CreditSlider
                  value={pointsShown}
                  max={quote.pointsMax}
                  step={1}
                  accessibilityLabel="Loyalty points to apply"
                  onChange={(v) => pin({ loyaltyPoints: v })}
                  onCommit={(v) => pin({ loyaltyPoints: v })}
                />
              </View>
              <TextInput
                value={pointsText ?? String(Math.round(pointsShown))}
                onChangeText={setPointsText}
                onBlur={() => {
                  const v = Number(pointsText);
                  setPointsText(null);
                  if (!Number.isNaN(v)) pin({ loyaltyPoints: Math.max(0, Math.min(quote.pointsMax, v)) });
                }}
                keyboardType="number-pad"
                accessibilityLabel="Loyalty points"
                className="h-11 w-24 rounded-lg px-3 text-right text-[15px]"
                style={{
                  backgroundColor: customerColors.surface.soft,
                  color: customerColors.charcoal.DEFAULT,
                  fontVariant: ['tabular-nums'],
                }}
              />
            </View>
          )}
        </View>
      )}

      <View className="px-4 pb-5">
        <View className="flex-row items-center justify-between">
          <Text className="text-[15px]" style={{ color: customerColors.charcoal.DEFAULT }}>
            Credits applied
          </Text>
          <Text
            className="text-[15px] font-semibold"
            style={{ color: customerColors.success.DEFAULT, fontVariant: ['tabular-nums'] }}
          >
            −{money(creditApplied)}
          </Text>
        </View>
        <Text className="mt-1 text-[13px]" style={{ color: customerColors.charcoal.soft }}>
          Fees &amp; taxes are always paid separately ({money(quote.nonRedeemable)})
        </Text>
      </View>
    </View>
  );
}
