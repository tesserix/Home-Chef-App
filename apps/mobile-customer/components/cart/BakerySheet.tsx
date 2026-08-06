// Cake configurator (#1065) — opened from a bakery item. The customer picks a
// size, shape, flavour, egg and sugar preference, optionally a message and an
// occasion, and sees the live price.
//
// Pricing runs through the same rules the server enforces
// (@homechef/mobile-shared/bakery), so the price shown here is the price
// charged. A rejection from those rules is shown as the reason the CTA is off.

import { useMemo, useState } from 'react';
import { Modal, Platform, Pressable, ScrollView, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Check, Minus, Plus, X } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import {
  BAKERY_OCCASIONS,
  BAKERY_OPTION_KINDS,
  BAKERY_OPTION_KIND_LABELS,
  bakerySummary,
  priceBakeryLine,
  weightChoices,
} from '@homechef/mobile-shared/bakery';
import type { CartBakeryConfig, MenuItem } from '../../types/customer';
import { formatMoney } from '../../lib/format';

const CLOSE_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const OPTION_RIPPLE = `${customerColors.coral.DEFAULT}14`;
const STEPPER_RIPPLE = `${customerColors.coral.DEFAULT}22`;
const CTA_RIPPLE = `${customerColors.canvas}40`;

interface BakerySheetProps {
  item: MenuItem;
  visible: boolean;
  onClose: () => void;
  onConfirm: (
    config: CartBakeryConfig,
    summary: string,
    unitPrice: number,
    quantity: number,
  ) => void;
}

export function BakerySheet({ item, visible, onClose, onConfirm }: BakerySheetProps) {
  const spec = item.bakery;
  const sizes = useMemo(() => (spec ? weightChoices(spec) : []), [spec]);

  const [weightKg, setWeightKg] = useState<number>(sizes[0] ?? 0);
  const [picked, setPicked] = useState<Record<string, string>>(() => defaultPicks(item));
  const [message, setMessage] = useState('');
  const [occasion, setOccasion] = useState('');
  const [qty, setQty] = useState(1);

  const optionIds = useMemo(() => Object.values(picked).filter(Boolean), [picked]);

  const config: CartBakeryConfig = useMemo(
    () => ({
      weightKg: weightKg || undefined,
      optionIds,
      messageOnCake: message.trim() || undefined,
      occasion: occasion || undefined,
    }),
    [weightKg, optionIds, message, occasion],
  );

  // The server's own rules, run locally. An error is the reason we can't add.
  const priced = useMemo(() => {
    if (!spec) return null;
    try {
      const { price, snapshot } = priceBakeryLine(spec, item.price, config);
      return { price, summary: bakerySummary(snapshot), serves: snapshot.serves, error: null as string | null };
    } catch (err) {
      return { price: 0, summary: '', serves: 0, error: err instanceof Error ? err.message : 'Choose your options' };
    }
  }, [spec, item.price, config]);

  if (!spec || !priced) return null;

  const valid = priced.error === null;

  function confirm() {
    if (!valid || !priced) return;
    onConfirm(config, priced.summary, priced.price, qty);
  }

  return (
    <Modal visible={visible} animationType="slide" presentationStyle="pageSheet" onRequestClose={onClose}>
      <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
        <View className="flex-row items-center justify-between px-4 pt-3 pb-2">
          <Text className="text-lg font-bold text-charcoal font-display flex-1" numberOfLines={1}>
            {item.name}
          </Text>
          <Pressable
            onPress={onClose}
            hitSlop={10}
            accessibilityRole="button"
            accessibilityLabel="Close"
            android_ripple={{ color: CLOSE_RIPPLE, borderless: true, radius: 20 }}
          >
            {({ pressed }) => (
              <View className={pressed && Platform.OS === 'ios' ? 'opacity-60' : ''}>
                <X size={24} color={customerColors.charcoal.DEFAULT} />
              </View>
            )}
          </Pressable>
        </View>

        <ScrollView contentContainerStyle={{ padding: 16, paddingBottom: 32 }}>
          {/* Size — the price driver, so it comes first. */}
          {sizes.length > 0 ? (
            <View className="mb-5">
              <View className="flex-row items-center justify-between mb-2">
                <Text className="text-base font-semibold text-charcoal">Size</Text>
                <Text className="text-xs text-charcoal-soft">
                  {priced.serves > 0 ? `Serves about ${priced.serves}` : `${formatMoney(spec.pricePerKg)} per kg`}
                </Text>
              </View>
              <View className="flex-row flex-wrap gap-2">
                {sizes.map((w) => {
                  const on = weightKg === w;
                  return (
                    <Pressable
                      key={w}
                      onPress={() => setWeightKg(w)}
                      accessibilityRole="radio"
                      accessibilityState={{ selected: on }}
                      accessibilityLabel={`${w} kilograms`}
                      android_ripple={{ color: OPTION_RIPPLE }}
                    >
                      {({ pressed }) => (
                        <View
                          className={`rounded-xl border px-4 min-h-[44px] justify-center ${
                            on ? 'border-coral bg-coral-tint' : 'border-hairline bg-canvas'
                          } ${pressed && Platform.OS === 'ios' ? 'opacity-70' : ''}`}
                        >
                          <Text className="text-sm text-charcoal tabular-nums">{w} kg</Text>
                        </View>
                      )}
                    </Pressable>
                  );
                })}
              </View>
            </View>
          ) : null}

          {/* One required pick per kind the baker offers. */}
          {BAKERY_OPTION_KINDS.map((kind) => {
            const options = spec.options.filter((o) => o.kind === kind && o.isAvailable);
            if (options.length === 0) return null;
            return (
              <View key={kind} className="mb-5">
                <View className="flex-row items-center justify-between mb-2">
                  <Text className="text-base font-semibold text-charcoal">
                    {BAKERY_OPTION_KIND_LABELS[kind]}
                  </Text>
                  <Text className="text-xs text-charcoal-soft">Required</Text>
                </View>
                {options.map((o) => {
                  const on = picked[kind] === o.id;
                  const delta = o.priceMode === 'per_kg' ? o.priceDelta * (weightKg || 1) : o.priceDelta;
                  return (
                    <Pressable
                      key={o.id}
                      onPress={() => setPicked((prev) => ({ ...prev, [kind]: o.id }))}
                      accessibilityRole="radio"
                      accessibilityState={{ selected: on }}
                      accessibilityLabel={
                        delta !== 0 ? `${o.name}, ${delta > 0 ? '+' : ''}${formatMoney(delta)}` : `${o.name}, no extra charge`
                      }
                      android_ripple={{ color: OPTION_RIPPLE }}
                    >
                      {({ pressed }) => (
                        <View
                          className={`flex-row items-center gap-3 rounded-xl border px-4 py-3 mb-2 ${
                            on ? 'border-coral bg-coral-tint' : 'border-hairline bg-canvas'
                          } ${pressed && Platform.OS === 'ios' ? 'opacity-70' : ''}`}
                        >
                          <View
                            className={`w-5 h-5 items-center justify-center rounded-full border ${
                              on ? 'border-coral bg-coral' : 'border-hairline'
                            }`}
                          >
                            {on ? <Check size={13} color={customerColors.canvas} /> : null}
                          </View>
                          <View className="flex-1">
                            <Text className="text-sm text-charcoal">{o.name}</Text>
                            {(o.dietaryTags ?? []).length > 0 || (o.allergens ?? []).length > 0 ? (
                              <Text className="text-xs text-charcoal-soft mt-0.5">
                                {[...(o.dietaryTags ?? []), ...(o.allergens ?? []).map((a) => `contains ${a}`)].join(' · ')}
                              </Text>
                            ) : null}
                          </View>
                          {delta !== 0 ? (
                            <Text className="text-sm text-charcoal-soft" style={{ fontVariant: ['tabular-nums'] }}>
                              {delta > 0 ? '+' : ''}
                              {formatMoney(delta)}
                            </Text>
                          ) : null}
                        </View>
                      )}
                    </Pressable>
                  );
                })}
              </View>
            );
          })}

          {/* Message on the bake */}
          {spec.allowMessage ? (
            <View className="mb-5">
              <View className="flex-row items-center justify-between mb-2">
                <Text className="text-base font-semibold text-charcoal">Message on it</Text>
                <Text className="text-xs text-charcoal-soft tabular-nums">
                  {message.length}/{spec.maxMessageChars || 40}
                </Text>
              </View>
              <TextInput
                className="rounded-xl border border-hairline bg-canvas px-4 min-h-[48px] text-sm text-charcoal"
                placeholder="Happy Birthday Aarav"
                placeholderTextColor={customerColors.charcoal.soft}
                value={message}
                maxLength={spec.maxMessageChars || 40}
                onChangeText={setMessage}
                accessibilityLabel="Message on the bake"
              />
            </View>
          ) : null}

          {/* Occasion — what the bake is for, so the baker can pack for it. */}
          {(spec.occasions?.length ?? 0) > 0 ? (
            <View className="mb-5">
              <Text className="text-base font-semibold text-charcoal mb-2">What's the occasion?</Text>
              <View className="flex-row flex-wrap gap-2">
                {BAKERY_OCCASIONS.filter((o) => spec.occasions?.includes(o.value)).map((o) => {
                  const on = occasion === o.value;
                  return (
                    <Pressable
                      key={o.value}
                      onPress={() => setOccasion(on ? '' : o.value)}
                      accessibilityRole="button"
                      accessibilityState={{ selected: on }}
                      accessibilityLabel={o.label}
                      android_ripple={{ color: OPTION_RIPPLE }}
                    >
                      {({ pressed }) => (
                        <View
                          className={`rounded-full border px-4 min-h-[40px] justify-center ${
                            on ? 'border-coral bg-coral-tint' : 'border-hairline bg-canvas'
                          } ${pressed && Platform.OS === 'ios' ? 'opacity-70' : ''}`}
                        >
                          <Text className="text-sm text-charcoal">{o.label}</Text>
                        </View>
                      )}
                    </Pressable>
                  );
                })}
              </View>
            </View>
          ) : null}

          {spec.leadTimeHours > 0 ? (
            <Text className="text-xs text-charcoal-soft mb-4">
              This bake needs {spec.leadTimeHours} hours' notice — you'll pick a date and time at checkout.
            </Text>
          ) : null}

          {/* Quantity */}
          <View className="flex-row items-center justify-between mt-2">
            <Text className="text-base font-semibold text-charcoal">Quantity</Text>
            <View className="flex-row items-center border border-coral rounded-lg overflow-hidden">
              <Pressable
                onPress={() => setQty((q) => Math.max(1, q - 1))}
                accessibilityRole="button"
                accessibilityLabel="Decrease quantity"
                android_ripple={{ color: STEPPER_RIPPLE }}
              >
                {({ pressed }) => (
                  <View
                    className={`w-11 h-11 items-center justify-center ${
                      pressed && Platform.OS === 'ios' ? 'opacity-60' : ''
                    }`}
                  >
                    <Minus size={16} color={customerColors.coral.DEFAULT} strokeWidth={2.5} />
                  </View>
                )}
              </Pressable>
              <Text
                className="min-w-[28px] text-center text-charcoal font-semibold"
                style={{ fontVariant: ['tabular-nums'] }}
              >
                {qty}
              </Text>
              <Pressable
                onPress={() => setQty((q) => q + 1)}
                accessibilityRole="button"
                accessibilityLabel="Increase quantity"
                android_ripple={{ color: STEPPER_RIPPLE }}
              >
                {({ pressed }) => (
                  <View
                    className={`w-11 h-11 items-center justify-center ${
                      pressed && Platform.OS === 'ios' ? 'opacity-60' : ''
                    }`}
                  >
                    <Plus size={16} color={customerColors.coral.DEFAULT} strokeWidth={2.5} />
                  </View>
                )}
              </Pressable>
            </View>
          </View>
        </ScrollView>

        <View className="px-4 pb-2 pt-2 border-t border-hairline">
          <Pressable
            onPress={confirm}
            disabled={!valid}
            accessibilityRole="button"
            accessibilityLabel="Add to cart"
            android_ripple={valid ? { color: CTA_RIPPLE } : undefined}
          >
            {({ pressed }) => (
              <View
                className={`rounded-lg min-h-[52px] items-center justify-center bg-coral ${
                  !valid ? 'opacity-50' : pressed && Platform.OS === 'ios' ? 'opacity-80' : ''
                }`}
              >
                <Text className="text-canvas font-semibold text-base tabular-nums">
                  Add {qty} · {formatMoney(priced.price * qty)}
                </Text>
              </View>
            )}
          </Pressable>
          <Text className="text-center text-xs text-charcoal-soft mt-1" numberOfLines={2}>
            {priced.error ?? priced.summary}
          </Text>
        </View>
      </SafeAreaView>
    </Modal>
  );
}

// Pre-select the baker's defaults so a customer who wants the standard cake can
// add it in one tap.
function defaultPicks(item: MenuItem): Record<string, string> {
  const out: Record<string, string> = {};
  for (const o of item.bakery?.options ?? []) {
    if (o.isDefault && o.isAvailable && !out[o.kind]) out[o.kind] = o.id;
  }
  return out;
}
