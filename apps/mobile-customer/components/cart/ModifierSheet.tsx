// Add-on modifier picker (#232) — opened from a menu item that has modifier
// groups. The customer picks options (radio for single-choice, checkbox for
// multi), sees the live price, and confirms; the parent adds the configured
// line to the cart. Self-contained RN Modal for reliability.

import { useMemo, useState } from 'react';
import { Modal, Platform, Pressable, ScrollView, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Check, Minus, Plus, X } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { MenuItem, SelectedModifier } from '../../types/customer';

// Android ripple tints — translucent colours derived from existing tokens
// (never a new literal colour), matching the ChefCard `withAlpha` convention.
const CLOSE_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const OPTION_RIPPLE = `${customerColors.coral.DEFAULT}14`;
const STEPPER_RIPPLE = `${customerColors.coral.DEFAULT}22`;
const CTA_RIPPLE = `${customerColors.canvas}40`;

interface ModifierSheetProps {
  item: MenuItem;
  visible: boolean;
  onClose: () => void;
  onConfirm: (modifiers: SelectedModifier[], unitPrice: number, quantity: number) => void;
}

export function ModifierSheet({ item, visible, onClose, onConfirm }: ModifierSheetProps) {
  const groups = item.modifierGroups ?? [];
  const [selected, setSelected] = useState<Record<string, string[]>>({});
  const [qty, setQty] = useState(1);

  function toggle(groupId: string, optionId: string, single: boolean) {
    setSelected((prev) => {
      const cur = prev[groupId] ?? [];
      if (single) return { ...prev, [groupId]: cur.includes(optionId) ? [] : [optionId] };
      return {
        ...prev,
        [groupId]: cur.includes(optionId) ? cur.filter((id) => id !== optionId) : [...cur, optionId],
      };
    });
  }

  // Build the selection + delta, and whether every required group is satisfied.
  const { modifiers, delta, valid } = useMemo(() => {
    const mods: SelectedModifier[] = [];
    let d = 0;
    let ok = true;
    for (const g of groups) {
      const picks = selected[g.id] ?? [];
      const minSel = g.required ? Math.max(1, g.minSelect) : g.minSelect;
      if (picks.length < minSel) ok = false;
      if (g.maxSelect > 0 && picks.length > g.maxSelect) ok = false;
      for (const oid of picks) {
        const o = g.options.find((x) => x.id === oid);
        if (o) {
          d += o.priceDelta;
          mods.push({ groupId: g.id, groupName: g.name, optionId: o.id, optionName: o.name, priceDelta: o.priceDelta });
        }
      }
    }
    return { modifiers: mods, delta: d, valid: ok };
  }, [groups, selected]);

  const unitPrice = item.price + delta;

  function confirm() {
    if (!valid) return;
    onConfirm(modifiers, unitPrice, qty);
    setSelected({});
    setQty(1);
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
          {groups.map((g) => {
            const single = g.maxSelect === 1;
            const picks = selected[g.id] ?? [];
            return (
              <View key={g.id} className="mb-5">
                <View className="flex-row items-center justify-between mb-2">
                  <Text className="text-base font-semibold text-charcoal">{g.name}</Text>
                  <Text className="text-xs text-charcoal-soft">
                    {g.required ? 'Required' : 'Optional'}
                    {g.maxSelect > 1 ? ` · up to ${g.maxSelect}` : ''}
                  </Text>
                </View>
                {g.options.map((o) => {
                  const on = picks.includes(o.id);
                  const disabled = !o.isAvailable;
                  return (
                    <Pressable
                      key={o.id}
                      onPress={() => !disabled && toggle(g.id, o.id, single)}
                      disabled={disabled}
                      accessibilityRole={single ? 'radio' : 'checkbox'}
                      accessibilityState={{ selected: on, disabled }}
                      accessibilityLabel={
                        o.priceDelta !== 0
                          ? `${o.name}, ${o.priceDelta > 0 ? '+' : ''}₹${o.priceDelta.toFixed(0)}`
                          : `${o.name}, free`
                      }
                      android_ripple={disabled ? undefined : { color: OPTION_RIPPLE }}
                    >
                      {({ pressed }) => (
                        <View
                          className={`flex-row items-center gap-3 rounded-xl border px-4 py-3 mb-2 ${
                            on ? 'border-coral bg-coral-tint' : 'border-hairline bg-canvas'
                          } ${disabled ? 'opacity-40' : ''} ${
                            !disabled && pressed && Platform.OS === 'ios' ? 'opacity-70' : ''
                          }`}
                        >
                          <View
                            className={`w-5 h-5 items-center justify-center ${single ? 'rounded-full' : 'rounded'} border ${
                              on ? 'border-coral bg-coral' : 'border-hairline'
                            }`}
                          >
                            {on ? <Check size={13} color={customerColors.canvas} /> : null}
                          </View>
                          <Text className="flex-1 text-sm text-charcoal">{o.name}</Text>
                          {o.priceDelta !== 0 ? (
                            <Text className="text-sm text-charcoal-soft" style={{ fontVariant: ['tabular-nums'] }}>
                              {o.priceDelta > 0 ? '+' : ''}₹{o.priceDelta.toFixed(0)}
                            </Text>
                          ) : (
                            <Text className="text-xs text-charcoal-soft">Free</Text>
                          )}
                        </View>
                      )}
                    </Pressable>
                  );
                })}
              </View>
            );
          })}

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
              <Text className="min-w-[28px] text-center text-charcoal font-semibold" style={{ fontVariant: ['tabular-nums'] }}>
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

        {/* Add CTA */}
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
                  Add {qty} · ₹{(unitPrice * qty).toFixed(0)}
                </Text>
              </View>
            )}
          </Pressable>
          {!valid ? (
            <Text className="text-center text-xs text-charcoal-soft mt-1">Choose the required options to continue</Text>
          ) : null}
        </View>
      </SafeAreaView>
    </Modal>
  );
}
