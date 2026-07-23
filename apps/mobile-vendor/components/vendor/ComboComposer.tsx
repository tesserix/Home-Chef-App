// ComboComposer — build a combo/thali's components by multi-selecting the chef's
// EXISTING menu items and/or adding custom add-ons (achar, curd) ONE AT A TIME.
// Replaces the daily-menu builder's comma-typed text field (#431): the chef's
// dishes are picked, not re-typed, and add-ons are added individually. Reuses the
// modal-item-picker + add-one-at-a-time interaction from ModifierComboEditor.
// Produces a list of component NAMES (the daily combo carries a name list).

import { useMemo, useState } from 'react';
import {
  Modal,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Plus, X } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';

interface Props {
  /** The chef's own à-la-carte dishes, to multi-select from. */
  menuItems: { id: string; name: string }[];
  /** Current component names. */
  value: string[];
  onChange: (names: string[]) => void;
}

export function ComboComposer({ menuItems, value, onChange }: Props) {
  const [picker, setPicker] = useState(false);
  const [custom, setCustom] = useState('');

  const has = (name: string) =>
    value.some((v) => v.toLowerCase() === name.trim().toLowerCase());

  const add = (name: string) => {
    const n = name.trim();
    if (!n || has(n)) return;
    onChange([...value, n]);
  };
  const remove = (name: string) => onChange(value.filter((v) => v !== name));

  const addCustom = () => {
    add(custom);
    setCustom('');
  };

  // Menu items not yet in the combo — the multi-select source.
  const available = useMemo(
    () => menuItems.filter((m) => !has(m.name)),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [menuItems, value],
  );

  return (
    <View style={styles.wrap}>
      {value.length === 0 ? (
        <Text style={styles.hint}>
          Add the dishes in this combo — pick from your menu, or add a custom item.
        </Text>
      ) : (
        value.map((name) => (
          <View key={name} style={styles.chipRow}>
            <Text style={styles.chipName} numberOfLines={1}>
              {name}
            </Text>
            <Pressable
              onPress={() => remove(name)}
              hitSlop={6}
              accessibilityRole="button"
              accessibilityLabel={`Remove ${name}`}
              android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
            >
              <X size={16} color={theme.colors.ink.muted} />
            </Pressable>
          </View>
        ))
      )}

      <Pressable
        onPress={() => setPicker(true)}
        hitSlop={6}
        accessibilityRole="button"
        accessibilityLabel="Add from your menu"
        android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
      >
        <Text style={styles.addLink}>+ Add from your menu</Text>
      </Pressable>

      {/* Custom add-on, one at a time (no comma-typing). */}
      <View style={styles.customRow}>
        <TextInput
          value={custom}
          onChangeText={setCustom}
          placeholder="Custom item (e.g. Achar, Curd)"
          placeholderTextColor={theme.colors.ink.muted}
          style={styles.customInput}
          onSubmitEditing={addCustom}
          returnKeyType="done"
          blurOnSubmit={false}
        />
        <Pressable
          onPress={addCustom}
          hitSlop={6}
          accessibilityRole="button"
          accessibilityLabel="Add custom item"
          style={styles.customAdd}
          android_ripple={{ color: `${theme.colors.paper}33`, borderless: false }}
        >
          <Plus size={18} color={theme.colors.paper} />
        </Pressable>
      </View>

      {/* Multi-select picker over the chef's menu — stays open so several dishes
          can be added in one go; each added dish drops out of the list. */}
      <Modal
        visible={picker}
        animationType="slide"
        presentationStyle="pageSheet"
        onRequestClose={() => setPicker(false)}
      >
        <SafeAreaView style={styles.modalRoot} edges={['top', 'left', 'right']}>
          <View style={styles.modalHeader}>
            <Text style={styles.modalTitle}>Add from your menu</Text>
            <Pressable
              onPress={() => setPicker(false)}
              hitSlop={8}
              accessibilityRole="button"
              accessibilityLabel="Done"
              android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
            >
              <X size={24} color={theme.colors.ink.DEFAULT} />
            </Pressable>
          </View>
          <ScrollView contentContainerStyle={{ padding: theme.spacing[4] }}>
            {available.length === 0 ? (
              <Text style={styles.hint}>
                No more menu items to add. Create dishes in your menu, or add a custom item above.
              </Text>
            ) : (
              available.map((m) => (
                <Pressable
                  key={m.id}
                  onPress={() => add(m.name)}
                  accessibilityRole="button"
                  accessibilityLabel={`Add ${m.name}`}
                  android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
                >
                  <View style={styles.pickRow}>
                    <Text style={styles.pickName}>{m.name}</Text>
                    <Plus size={18} color={theme.colors.ink.DEFAULT} />
                  </View>
                </Pressable>
              ))
            )}
          </ScrollView>
        </SafeAreaView>
      </Modal>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    gap: theme.spacing[2],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    paddingTop: theme.spacing[2],
  },
  hint: { fontFamily: 'Inter', fontSize: 12, lineHeight: 16, color: theme.colors.ink.muted },
  chipRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.DEFAULT,
    paddingVertical: theme.spacing[2],
    paddingHorizontal: theme.spacing[3],
    gap: theme.spacing[2],
  },
  chipName: { flex: 1, fontFamily: 'Inter', fontSize: 14, color: theme.colors.ink.DEFAULT },
  addLink: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: theme.colors.ink.DEFAULT },
  customRow: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[2] },
  customInput: {
    flex: 1,
    minHeight: 40,
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[3],
    fontFamily: 'Inter',
    fontSize: 14,
    color: theme.colors.ink.DEFAULT,
  },
  customAdd: {
    width: 40,
    height: 40,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
  },
  modalRoot: { flex: 1, backgroundColor: theme.colors.bone },
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
  pickName: { fontFamily: 'Inter', fontSize: 15, color: theme.colors.ink.DEFAULT },
});
