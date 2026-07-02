// Add-on modifier picker (#232) — opened from a menu item that has modifier
// groups. The customer picks options (radio for single-choice, checkbox for
// multi), sees the live price, and confirms; the parent adds the configured
// line to the cart. Self-contained RN Modal for reliability.

import { useMemo, useState } from 'react';
import { Modal, Pressable, ScrollView, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Check, Minus, Plus, X } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { MenuItem, SelectedModifier } from '../../types/customer';

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
          <Pressable onPress={onClose} hitSlop={8} accessibilityRole="button" accessibilityLabel="Close">
            <X size={24} color={customerColors.charcoal.DEFAULT} />
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
                    >
                      <View
                        className={`flex-row items-center gap-3 rounded-xl border px-4 py-3 mb-2 ${
                          on ? 'border-coral bg-coral-tint' : 'border-hairline bg-canvas'
                        } ${disabled ? 'opacity-40' : ''}`}
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
              <Pressable onPress={() => setQty((q) => Math.max(1, q - 1))} className="w-10 h-10 items-center justify-center" accessibilityRole="button" accessibilityLabel="Decrease quantity" hitSlop={6}>
                <Minus size={16} color={customerColors.coral.DEFAULT} strokeWidth={2.5} />
              </Pressable>
              <Text className="min-w-[28px] text-center text-charcoal font-semibold" style={{ fontVariant: ['tabular-nums'] }}>
                {qty}
              </Text>
              <Pressable onPress={() => setQty((q) => q + 1)} className="w-10 h-10 items-center justify-center" accessibilityRole="button" accessibilityLabel="Increase quantity" hitSlop={6}>
                <Plus size={16} color={customerColors.coral.DEFAULT} strokeWidth={2.5} />
              </Pressable>
            </View>
          </View>
        </ScrollView>

        {/* Add CTA */}
        <View className="px-4 pb-2 pt-2 border-t border-hairline">
          <Pressable onPress={confirm} disabled={!valid} accessibilityRole="button" accessibilityLabel="Add to cart">
            <View className={`rounded-lg min-h-[52px] items-center justify-center bg-coral ${!valid ? 'opacity-50' : ''}`}>
              <Text className="text-canvas font-semibold text-base">
                Add {qty} · ₹{(unitPrice * qty).toFixed(0)}
              </Text>
            </View>
          </Pressable>
          {!valid ? (
            <Text className="text-center text-xs text-charcoal-soft mt-1">Choose the required options to continue</Text>
          ) : null}
        </View>
      </SafeAreaView>
    </Modal>
  );
}
