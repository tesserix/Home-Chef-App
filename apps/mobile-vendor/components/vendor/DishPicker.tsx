// DishPicker — choose a dish from the menu the chef already built, instead of
// typing it again.
//
// The mobile twin of apps/vendor-portal/.../DishPicker.tsx: same two modes,
// native sheet instead of a dropdown. Styling follows the vendor app's own
// convention — StyleSheet + theme.colors.ink/paper/mist, no NativeWind classes
// and no coral: this app retires persimmon as an accent and lets ink carry
// actions (see the theme tokens).
//
// PICK is the default because the answer is nearly always already on the menu,
// and picking carries price, portion, serves, tags and allergens across. TYPE
// stays available because a tiffin often includes something never sold à la
// carte; refusing that would force junk items onto the public menu.

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
import { Check, ChevronDown, Pencil, X } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { useVendorMenu, type MenuItem } from '../../hooks/useVendorMenu';
import { useKitchenMarket } from '../../hooks/useKitchenMarket';
import { currencySymbol } from '../../lib/format';

export interface DishPrefill {
  menuItemId: string;
  name: string;
  price: number;
  portionSize?: string;
  serves: number;
  dietaryTags?: string[];
  allergens?: string[];
}

interface DishPickerProps {
  variant: 'veg' | 'nonveg';
  name: string;
  menuItemId?: string | null;
  onPick: (prefill: DishPrefill) => void;
  onTypeName: (name: string) => void;
  placeholder?: string;
}

/**
 * Options for a variant. An item that never declared isVeg stays in BOTH lists
 * rather than vanishing — hiding a dish the chef can see on their Menu screen
 * reads as a bug, and the cell's own variant is what customers are shown.
 */
function optionsForVariant(items: MenuItem[], variant: 'veg' | 'nonveg'): MenuItem[] {
  return items.filter((i) =>
    i.isVeg === undefined ? true : variant === 'veg' ? i.isVeg : !i.isVeg,
  );
}

function summarise(
  o: { price: number; portionSize?: string; serves?: number },
  symbol: string,
): string {
  const bits = [`${symbol}${o.price}`];
  if (o.portionSize) bits.push(o.portionSize);
  if ((o.serves ?? 1) > 1) bits.push(`serves ${o.serves}`);
  return bits.join(' · ');
}

export function DishPicker({
  variant,
  name,
  menuItemId,
  onPick,
  onTypeName,
  placeholder = 'Choose a dish',
}: DishPickerProps) {
  const { data } = useVendorMenu();
  const symbol = currencySymbol(useKitchenMarket()?.currency);
  const options = useMemo(
    () => optionsForVariant(data?.items ?? [], variant),
    [data, variant],
  );
  // Open in whichever mode matches what is saved, so nothing is reinterpreted.
  const [typing, setTyping] = useState(() => !menuItemId && !!name);
  const [open, setOpen] = useState(false);

  const picked = options.find((o) => o.id === menuItemId);

  if (typing) {
    return (
      <View>
        <TextInput
          value={name}
          onChangeText={onTypeName}
          placeholder="Dish name"
          placeholderTextColor={theme.colors.ink.muted}
          style={styles.input}
        />
        {options.length > 0 ? (
          <Pressable onPress={() => setTyping(false)} accessibilityRole="button" hitSlop={6}>
            <Text style={styles.switchLink}>Choose from my menu instead</Text>
          </Pressable>
        ) : null}
      </View>
    );
  }

  return (
    <View>
      <Pressable
        onPress={() => setOpen(true)}
        accessibilityRole="button"
        accessibilityLabel={name ? `Dish: ${name}. Change` : placeholder}
        style={styles.trigger}
      >
        <View style={styles.triggerText}>
          {name ? (
            <>
              <Text style={styles.triggerName} numberOfLines={1}>
                {name}
              </Text>
              <Text style={styles.triggerMeta} numberOfLines={1}>
                {picked ? summarise(picked, symbol) : 'Custom dish'}
              </Text>
            </>
          ) : (
            <Text style={styles.triggerPlaceholder}>{placeholder}</Text>
          )}
        </View>
        <ChevronDown size={16} color={theme.colors.ink.muted} />
      </Pressable>

      <Modal visible={open} animationType="slide" transparent onRequestClose={() => setOpen(false)}>
        <Pressable style={styles.backdrop} onPress={() => setOpen(false)} />
        <SafeAreaView edges={['bottom']} style={styles.sheet}>
          <View style={styles.sheetHead}>
            <Text style={styles.sheetTitle}>Choose a dish</Text>
            <Pressable
              onPress={() => setOpen(false)}
              accessibilityRole="button"
              accessibilityLabel="Close"
              hitSlop={8}
            >
              <X size={20} color={theme.colors.ink.DEFAULT} />
            </Pressable>
          </View>

          <ScrollView>
            {options.length === 0 ? (
              <Text style={styles.empty}>
                Nothing on your menu yet — type the dish instead.
              </Text>
            ) : (
              options.map((o) => {
                const selected = o.id === menuItemId;
                return (
                  <Pressable
                    key={o.id}
                    onPress={() => {
                      onPick({
                        menuItemId: o.id,
                        name: o.name,
                        price: o.price,
                        portionSize: o.portionSize,
                        serves: o.serves || 1,
                        dietaryTags: o.dietaryTags,
                        allergens: o.allergens,
                      });
                      setOpen(false);
                    }}
                    accessibilityRole="button"
                    accessibilityState={{ selected }}
                    style={[styles.option, selected && styles.optionSelected]}
                  >
                    <View style={styles.optionText}>
                      <Text style={styles.optionName} numberOfLines={1}>
                        {o.name}
                      </Text>
                      <Text style={styles.optionMeta} numberOfLines={1}>
                        {summarise(o, symbol)}
                        {/* Still pickable when off-menu: a plan is a schedule,
                            and the chef may re-enable it before serving day. */}
                        {o.isAvailable === false ? ' · currently off menu' : ''}
                      </Text>
                    </View>
                    {selected ? <Check size={16} color={theme.colors.ink.DEFAULT} /> : null}
                  </Pressable>
                );
              })
            )}

            <Pressable
              onPress={() => {
                setOpen(false);
                setTyping(true);
              }}
              accessibilityRole="button"
              style={styles.typeRow}
            >
              <Pencil size={16} color={theme.colors.ink.DEFAULT} />
              <Text style={styles.typeRowText}>Type a dish that isn&apos;t on my menu</Text>
            </Pressable>
          </ScrollView>
        </SafeAreaView>
      </Modal>
    </View>
  );
}

/**
 * The one-line answer to "what am I actually selling here?".
 *
 * A bare "140" in a box says nothing — not the currency, not how much food, not
 * what it works out to per head. This states the whole thing in the order a chef
 * reasons about it: total, portion, how many it feeds, and the per-person figure
 * derived rather than typed, so the two can never disagree.
 */
export function cellSummary(
  price: number,
  portionSize: string,
  serves: number,
  symbol = '₹',
): string {
  const bits = [`${symbol}${Math.round(price)}`];
  if (portionSize.trim()) bits.push(portionSize.trim());
  bits.push(serves > 1 ? `feeds ${serves}` : 'single portion');
  let out = bits.join(' · ');
  // Only worth showing when it differs from the total — "₹140 per person" under
  // "₹140" is noise.
  if (serves > 1 && price > 0) out += `  →  ${symbol}${Math.round(price / serves)} per person`;
  return out;
}

interface PriceSummaryProps {
  price: number;
  portionSize: string;
  serves: number;
}

export function PriceSummary({ price, portionSize, serves }: PriceSummaryProps) {
  const symbol = currencySymbol(useKitchenMarket()?.currency);
  if (!price) return null;
  return <Text style={styles.summary}>{cellSummary(price, portionSize, serves, symbol)}</Text>;
}

interface PortionFieldsProps {
  portionSize: string;
  serves: number;
  onChange: (patch: { portionSize?: string; serves?: number }) => void;
}

/**
 * How much food, and for how many. A price with no portion tells a tiffin
 * customer nothing about value.
 */
export function PortionFields({ portionSize, serves, onChange }: PortionFieldsProps) {
  return (
    <View style={styles.portionRow}>
      <View style={styles.portionCol}>
        <Text style={styles.portionLabel}>Portion</Text>
        <TextInput
          value={portionSize}
          onChangeText={(t) => onChange({ portionSize: t })}
          placeholder="e.g. 500 ml"
          placeholderTextColor={theme.colors.ink.muted}
          style={styles.input}
        />
      </View>
      <View style={styles.servesCol}>
        <Text style={styles.portionLabel}>Serves</Text>
        <TextInput
          value={String(serves || 1)}
          onChangeText={(t) => {
            const n = Number(t.replace(/[^0-9]/g, ''));
            onChange({ serves: n > 0 ? Math.min(n, 50) : 1 });
          }}
          keyboardType="number-pad"
          accessibilityLabel="How many people this serves"
          style={styles.input}
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  input: {
    minHeight: 44,
    backgroundColor: theme.colors.bone,
    borderRadius: 12,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 14,
    color: theme.colors.ink.DEFAULT,
  },
  switchLink: {
    marginTop: 4,
    fontSize: 12,
    fontWeight: '600',
    color: theme.colors.ink.DEFAULT,
  },
  trigger: {
    minHeight: 44,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    backgroundColor: theme.colors.bone,
    borderRadius: 12,
    paddingHorizontal: 12,
    paddingVertical: 8,
  },
  triggerText: { flex: 1 },
  triggerName: { fontSize: 14, color: theme.colors.ink.DEFAULT },
  triggerMeta: { fontSize: 11, color: theme.colors.ink.muted },
  triggerPlaceholder: { fontSize: 14, color: theme.colors.ink.muted },
  backdrop: { flex: 1, backgroundColor: 'rgba(0,0,0,0.4)' },
  sheet: {
    backgroundColor: theme.colors.paper,
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
    maxHeight: '70%',
  },
  sheetHead: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  sheetTitle: { fontSize: 16, fontWeight: '600', color: theme.colors.ink.DEFAULT },
  empty: { paddingHorizontal: 16, paddingVertical: 16, fontSize: 14, color: theme.colors.ink.soft },
  option: {
    minHeight: 44,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  optionSelected: { backgroundColor: theme.colors.bone },
  optionText: { flex: 1 },
  optionName: { fontSize: 14, color: theme.colors.ink.DEFAULT },
  optionMeta: { fontSize: 11, color: theme.colors.ink.muted },
  typeRow: {
    minHeight: 44,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    paddingHorizontal: 16,
    paddingVertical: 14,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
  },
  typeRowText: { fontSize: 14, fontWeight: '600', color: theme.colors.ink.DEFAULT },
  summary: {
    marginTop: 6,
    fontSize: 12,
    color: theme.colors.ink.soft,
  },
  portionRow: { flexDirection: 'row', gap: 8, marginTop: 8 },
  portionCol: { flex: 1 },
  servesCol: { width: 92 },
  portionLabel: {
    fontSize: 11,
    fontWeight: '500',
    color: theme.colors.ink.muted,
    marginBottom: 4,
  },
});
