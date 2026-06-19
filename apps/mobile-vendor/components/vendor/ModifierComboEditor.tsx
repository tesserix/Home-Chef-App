// Add-on modifier-groups editor + combo builder for the menu-item form (#52).
// Chef-facing: define choice groups (single/multi, required, price deltas) and
// optionally bundle other dishes into a combo at a fixed price.

import { useState } from 'react';
import { Modal, Pressable, ScrollView, StyleSheet, Switch, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Plus, Trash2, X } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import type { ModifierGroupInput, ComboItemInput } from '../../hooks/useVendorMenu';

interface Props {
  groups: ModifierGroupInput[];
  setGroups: (g: ModifierGroupInput[]) => void;
  isCombo: boolean;
  setIsCombo: (v: boolean) => void;
  comboItems: ComboItemInput[];
  setComboItems: (c: ComboItemInput[]) => void;
  /** The chef's other menu items, for the combo picker. */
  menuItems: { id: string; name: string }[];
}

const num = (s: string) => parseFloat(s.replace(/[^0-9.-]/g, '')) || 0;

export function ModifierComboEditor({ groups, setGroups, isCombo, setIsCombo, comboItems, setComboItems, menuItems }: Props) {
  const [picker, setPicker] = useState(false);
  const nameById = (id: string) => menuItems.find((m) => m.id === id)?.name ?? 'Item';

  // ── Group helpers ──
  const addGroup = () =>
    setGroups([...groups, { name: '', required: false, minSelect: 0, maxSelect: 1, options: [{ name: '', priceDelta: 0 }] }]);
  const patchGroup = (gi: number, patch: Partial<ModifierGroupInput>) =>
    setGroups(groups.map((g, i) => (i === gi ? { ...g, ...patch } : g)));
  const removeGroup = (gi: number) => setGroups(groups.filter((_, i) => i !== gi));
  const addOption = (gi: number) =>
    patchGroup(gi, { options: [...groups[gi]!.options, { name: '', priceDelta: 0 }] });
  const patchOption = (gi: number, oi: number, patch: Partial<ModifierGroupInput['options'][number]>) =>
    patchGroup(gi, { options: groups[gi]!.options.map((o, i) => (i === oi ? { ...o, ...patch } : o)) });
  const removeOption = (gi: number, oi: number) =>
    patchGroup(gi, { options: groups[gi]!.options.filter((_, i) => i !== oi) });

  // ── Combo helpers ──
  const addComboItem = (id: string) => {
    if (comboItems.some((c) => c.menuItemId === id)) return;
    setComboItems([...comboItems, { menuItemId: id, quantity: 1 }]);
    setPicker(false);
  };
  const patchComboQty = (id: string, q: number) =>
    setComboItems(comboItems.map((c) => (c.menuItemId === id ? { ...c, quantity: Math.max(1, q) } : c)));
  const removeComboItem = (id: string) => setComboItems(comboItems.filter((c) => c.menuItemId !== id));

  return (
    <>
      {/* ADD-ONS */}
      <Text style={styles.sectionLabel}>ADD-ONS</Text>
      <View style={styles.card}>
        <Text style={styles.hint}>
          Offer choices like “Spice level” (single) or “Extras” (multiple). Customers pick at checkout.
        </Text>
        {groups.map((g, gi) => {
          const single = g.maxSelect === 1;
          return (
            <View key={gi} style={styles.group}>
              <View style={styles.rowBetween}>
                <TextInput
                  style={[styles.input, { flex: 1 }]}
                  placeholder="Group name (e.g. Spice level)"
                  placeholderTextColor={theme.colors.ink.muted}
                  value={g.name}
                  onChangeText={(t) => patchGroup(gi, { name: t })}
                />
                <Pressable onPress={() => removeGroup(gi)} hitSlop={8} accessibilityLabel="Remove group">
                  <Trash2 size={18} color={theme.colors.destructive.DEFAULT} />
                </Pressable>
              </View>
              <View style={styles.toggleRow}>
                <Text style={styles.toggleLabel}>Required</Text>
                <Switch value={g.required} onValueChange={(v) => patchGroup(gi, { required: v })} trackColor={{ true: theme.colors.herb.DEFAULT }} />
              </View>
              <View style={styles.toggleRow}>
                <Text style={styles.toggleLabel}>Single choice</Text>
                <Switch
                  value={single}
                  onValueChange={(v) => patchGroup(gi, { maxSelect: v ? 1 : 0 })}
                  trackColor={{ true: theme.colors.herb.DEFAULT }}
                />
              </View>
              {g.options.map((o, oi) => (
                <View key={oi} style={styles.optionRow}>
                  <TextInput
                    style={[styles.input, { flex: 1 }]}
                    placeholder="Option (e.g. Extra roti)"
                    placeholderTextColor={theme.colors.ink.muted}
                    value={o.name}
                    onChangeText={(t) => patchOption(gi, oi, { name: t })}
                  />
                  <View style={styles.priceWrap}>
                    <Text style={styles.pricePrefix}>₹</Text>
                    <TextInput
                      style={styles.priceInput}
                      placeholder="0"
                      placeholderTextColor={theme.colors.ink.muted}
                      keyboardType="numbers-and-punctuation"
                      value={o.priceDelta ? String(o.priceDelta) : ''}
                      onChangeText={(t) => patchOption(gi, oi, { priceDelta: num(t) })}
                    />
                  </View>
                  <Pressable onPress={() => removeOption(gi, oi)} hitSlop={6} accessibilityLabel="Remove option">
                    <X size={16} color={theme.colors.ink.muted} />
                  </Pressable>
                </View>
              ))}
              <Pressable onPress={() => addOption(gi)} hitSlop={6}>
                <Text style={styles.addLink}>+ Add option</Text>
              </Pressable>
            </View>
          );
        })}
        <Pressable onPress={addGroup} hitSlop={6}>
          <Text style={styles.addLink}>+ Add a group</Text>
        </Pressable>
      </View>

      {/* COMBO */}
      <Text style={styles.sectionLabel}>COMBO / BUNDLE</Text>
      <View style={styles.card}>
        <View style={styles.toggleRow}>
          <View style={{ flex: 1 }}>
            <Text style={styles.toggleLabel}>This item is a combo</Text>
            <Text style={styles.hint}>Bundle several dishes; the price above is the bundle price.</Text>
          </View>
          <Switch value={isCombo} onValueChange={setIsCombo} trackColor={{ true: theme.colors.herb.DEFAULT }} />
        </View>
        {isCombo ? (
          <>
            {comboItems.map((c) => (
              <View key={c.menuItemId} style={styles.optionRow}>
                <Text style={[styles.toggleLabel, { flex: 1 }]} numberOfLines={1}>
                  {nameById(c.menuItemId)}
                </Text>
                <View style={styles.qtyRow}>
                  <Pressable onPress={() => patchComboQty(c.menuItemId, c.quantity - 1)} hitSlop={6} accessibilityLabel="Decrease">
                    <Text style={styles.qtyBtn}>−</Text>
                  </Pressable>
                  <Text style={styles.qtyVal}>{c.quantity}</Text>
                  <Pressable onPress={() => patchComboQty(c.menuItemId, c.quantity + 1)} hitSlop={6} accessibilityLabel="Increase">
                    <Text style={styles.qtyBtn}>+</Text>
                  </Pressable>
                </View>
                <Pressable onPress={() => removeComboItem(c.menuItemId)} hitSlop={6} accessibilityLabel="Remove">
                  <X size={16} color={theme.colors.ink.muted} />
                </Pressable>
              </View>
            ))}
            <Pressable onPress={() => setPicker(true)} hitSlop={6}>
              <Text style={styles.addLink}>+ Add an item</Text>
            </Pressable>
          </>
        ) : null}
      </View>

      {/* Item picker modal */}
      <Modal visible={picker} animationType="slide" presentationStyle="pageSheet" onRequestClose={() => setPicker(false)}>
        <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
          <View style={styles.modalHeader}>
            <Text style={styles.modalTitle}>Add to combo</Text>
            <Pressable onPress={() => setPicker(false)} hitSlop={8} accessibilityLabel="Close">
              <X size={24} color={theme.colors.ink.DEFAULT} />
            </Pressable>
          </View>
          <ScrollView contentContainerStyle={{ padding: theme.spacing[4] }}>
            {menuItems.length === 0 ? (
              <Text style={styles.hint}>Add more dishes to your menu first, then bundle them here.</Text>
            ) : (
              menuItems.map((m) => {
                const picked = comboItems.some((c) => c.menuItemId === m.id);
                return (
                  <Pressable key={m.id} onPress={() => !picked && addComboItem(m.id)} disabled={picked}>
                    <View style={[styles.pickRow, picked && { opacity: 0.4 }]}>
                      <Text style={styles.toggleLabel}>{m.name}</Text>
                      {picked ? <Text style={styles.hint}>Added</Text> : <Plus size={18} color={theme.colors.herb.DEFAULT} />}
                    </View>
                  </Pressable>
                );
              })
            )}
          </ScrollView>
        </SafeAreaView>
      </Modal>
    </>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[2],
  },
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    ...theme.shadow[1],
    padding: theme.spacing[4],
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[4],
    gap: theme.spacing[2],
  },
  hint: { fontFamily: 'Inter', fontSize: 12, lineHeight: 16, color: theme.colors.ink.muted },
  group: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.md,
    padding: theme.spacing[3],
    gap: theme.spacing[2],
    marginTop: theme.spacing[2],
  },
  rowBetween: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[2] },
  toggleRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: theme.spacing[3] },
  toggleLabel: { fontFamily: 'Inter', fontSize: 15, color: theme.colors.ink.DEFAULT },
  optionRow: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[2] },
  input: {
    minHeight: 40,
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[3],
    fontFamily: 'Inter',
    fontSize: 14,
    color: theme.colors.ink.DEFAULT,
  },
  priceWrap: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[2],
    width: 72,
  },
  pricePrefix: { fontFamily: 'Inter', fontSize: 14, color: theme.colors.ink.soft },
  priceInput: { flex: 1, minHeight: 40, fontFamily: 'Inter-SemiBold', fontSize: 14, color: theme.colors.ink.DEFAULT, textAlign: 'center' },
  addLink: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: theme.colors.herb.DEFAULT, marginTop: theme.spacing[2] },
  qtyRow: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[2] },
  qtyBtn: { fontFamily: 'Inter-SemiBold', fontSize: 18, color: theme.colors.herb.DEFAULT, width: 24, textAlign: 'center' },
  qtyVal: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.ink.DEFAULT, minWidth: 20, textAlign: 'center' },
  modalHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  modalTitle: { fontFamily: 'Geist-Bold', fontSize: 20, color: theme.colors.ink.DEFAULT },
  pickRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: theme.spacing[3],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
});
