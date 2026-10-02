// Bakery configurator editor for the menu-item form (#1065). Baker-facing:
// what the product is, how it is priced by weight, and the choices a customer
// must make (shape, flavour, egg, sugar) before the cake can be baked.
//
// Only rendered for a kitchen that sells bakery — either vertical or opt-in.
// The backend gates on the same rule and rejects a spec from anyone else.

import { Platform, Pressable, StyleSheet, Switch, Text, TextInput, View } from 'react-native';
import { Trash2 } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import {
  BAKERY_OCCASIONS,
  BAKERY_OPTION_KINDS,
  BAKERY_OPTION_KIND_LABELS,
  BAKERY_PRODUCT_TYPES,
  weightChoices,
  type BakeryOptionKind,
} from '@homechef/mobile-shared/bakery';
import { BAKERY_DIET_OPTIONS, BAKERY_ALLERGEN_OPTIONS } from '@homechef/mobile-shared/dietary';
import type { BakeryOptionInput, BakerySpecInput } from '../../hooks/useVendorMenu';
import { useKitchenMarket } from '../../hooks/useKitchenMarket';
import { currencySymbol } from '../../lib/format';

interface Props {
  spec: BakerySpecInput | null;
  setSpec: (s: BakerySpecInput | null) => void;
}

export const EMPTY_BAKERY_SPEC: BakerySpecInput = {
  productType: 'cake',
  pricePerKg: 0,
  minWeightKg: 0.5,
  maxWeightKg: 3,
  weightStepKg: 0.5,
  servesPerKg: 8,
  allowMessage: true,
  maxMessageChars: 40,
  allowReferencePhoto: false,
  leadTimeHours: 24,
  occasions: [],
  options: [],
};

const num = (s: string) => parseFloat(s.replace(/[^0-9.]/g, '')) || 0;
const int = (s: string) => parseInt(s.replace(/[^0-9]/g, ''), 10) || 0;

export function BakerySpecEditor({ spec, setSpec }: Props) {
  const symbol = currencySymbol(useKitchenMarket()?.currency);
  const on = spec !== null;
  const patch = (p: Partial<BakerySpecInput>): void => {
    if (spec) setSpec({ ...spec, ...p });
  };

  const addOption = (kind: BakeryOptionKind): void => {
    if (!spec) return;
    patch({
      options: [
        ...spec.options,
        { kind, name: '', priceDelta: 0, priceMode: 'flat', dietaryTags: [], allergens: [], isAvailable: true },
      ],
    });
  };
  const patchOption = (index: number, p: Partial<BakeryOptionInput>): void => {
    if (!spec) return;
    patch({ options: spec.options.map((o, i) => (i === index ? { ...o, ...p } : o)) });
  };
  const removeOption = (index: number): void => {
    if (!spec) return;
    patch({ options: spec.options.filter((_, i) => i !== index) });
  };

  const toggleTag = (index: number, field: 'dietaryTags' | 'allergens', value: string): void => {
    const current = spec?.options[index]?.[field] ?? [];
    patchOption(index, {
      [field]: current.includes(value) ? current.filter((v) => v !== value) : [...current, value],
    });
  };

  const toggleOccasion = (value: string): void => {
    if (!spec) return;
    patch({
      occasions: spec.occasions.includes(value)
        ? spec.occasions.filter((o) => o !== value)
        : [...spec.occasions, value],
    });
  };

  const sizes = spec ? weightChoices(spec) : [];

  return (
    <>
      <Text style={styles.sectionLabel}>BAKERY</Text>
      <View style={styles.card}>
        <View style={styles.toggleRow}>
          <Text style={styles.toggleLabel}>Customers configure this bake</Text>
          <Switch
            value={on}
            onValueChange={(v) => setSpec(v ? { ...EMPTY_BAKERY_SPEC } : null)}
            trackColor={{ true: theme.colors.herb.DEFAULT }}
          />
        </View>
        <Text style={styles.hint}>
          Turn this on for a cake sold by weight or with a choice of shape, flavour and egg preference. Leave it
          off for something sold as-is, like a loaf.
        </Text>

        {spec ? (
          <>
            <Text style={styles.fieldLabel}>What is it?</Text>
            <View style={styles.chipWrap}>
              {BAKERY_PRODUCT_TYPES.map((p) => (
                <Chip
                  key={p.value}
                  label={p.label}
                  selected={spec.productType === p.value}
                  onPress={() => patch({ productType: p.value })}
                />
              ))}
            </View>

            <View style={styles.hairline} />

            <Text style={styles.fieldLabel}>Priced by weight</Text>
            <Text style={styles.hint}>
              Set a price per kg to sell by size. Leave it at 0 and the item price above applies as-is.
            </Text>
            <View style={styles.gridRow}>
              <Field
                label="Per kg"
                prefix={symbol}
                value={spec.pricePerKg ? String(spec.pricePerKg) : ''}
                onChangeText={(t) => patch({ pricePerKg: num(t) })}
              />
              <Field
                label="Serves per kg"
                value={spec.servesPerKg ? String(spec.servesPerKg) : ''}
                onChangeText={(t) => patch({ servesPerKg: int(t) })}
              />
            </View>
            {spec.pricePerKg > 0 ? (
              <>
                <View style={styles.gridRow}>
                  <Field
                    label="Smallest"
                    suffix="kg"
                    value={spec.minWeightKg ? String(spec.minWeightKg) : ''}
                    onChangeText={(t) => patch({ minWeightKg: num(t) })}
                  />
                  <Field
                    label="Largest"
                    suffix="kg"
                    value={spec.maxWeightKg ? String(spec.maxWeightKg) : ''}
                    onChangeText={(t) => patch({ maxWeightKg: num(t) })}
                  />
                  <Field
                    label="Steps of"
                    suffix="kg"
                    value={spec.weightStepKg ? String(spec.weightStepKg) : ''}
                    onChangeText={(t) => patch({ weightStepKg: num(t) })}
                  />
                </View>
                <Text style={styles.hint}>
                  {sizes.length > 0
                    ? `Customers see ${sizes.map((s) => `${s} kg`).join(', ')} — from ${symbol}${Math.round(
                        spec.pricePerKg * (sizes[0] ?? 0),
                      )}.`
                    : 'Set a smallest and largest size to show a size picker.'}
                </Text>
              </>
            ) : null}

            <View style={styles.hairline} />

            <Text style={styles.fieldLabel}>Choices</Text>
            <Text style={styles.hint}>
              Every group you add here becomes a required pick — a cake with no flavour chosen is not something
              you can start baking.
            </Text>
            {BAKERY_OPTION_KINDS.map((kind) => {
              const rows = spec.options
                .map((o, i) => ({ o, i }))
                .filter(({ o }) => o.kind === kind);
              return (
                <View key={kind} style={styles.group}>
                  <View style={styles.rowBetween}>
                    <Text style={styles.groupTitle}>{BAKERY_OPTION_KIND_LABELS[kind]}</Text>
                    <Pressable
                      onPress={() => addOption(kind)}
                      hitSlop={8}
                      accessibilityRole="button"
                      accessibilityLabel={`Add a ${BAKERY_OPTION_KIND_LABELS[kind]} choice`}
                    >
                      {({ pressed }) => (
                        <Text style={[styles.addLink, pressed && Platform.OS === 'ios' && { opacity: 0.6 }]}>
                          + Add
                        </Text>
                      )}
                    </Pressable>
                  </View>
                  {rows.map(({ o, i }) => (
                    <View key={i} style={styles.optionBlock}>
                      <View style={styles.optionRow}>
                        <TextInput
                          style={[styles.input, { flex: 1 }]}
                          placeholder={placeholderFor(kind)}
                          placeholderTextColor={theme.colors.ink.muted}
                          value={o.name}
                          onChangeText={(t) => patchOption(i, { name: t })}
                        />
                        <View style={[styles.priceWrap, styles.optionPrice]}>
                          <Text style={styles.pricePrefix}>{symbol}</Text>
                          <TextInput
                            style={styles.priceInput}
                            placeholder="0"
                            placeholderTextColor={theme.colors.ink.muted}
                            keyboardType="numbers-and-punctuation"
                            value={o.priceDelta ? String(o.priceDelta) : ''}
                            onChangeText={(t) => patchOption(i, { priceDelta: num(t) })}
                          />
                        </View>
                        <Pressable
                          onPress={() => removeOption(i)}
                          hitSlop={6}
                          accessibilityRole="button"
                          accessibilityLabel={`Remove ${o.name || 'choice'}`}
                        >
                          {({ pressed }) => (
                            <View style={pressed && Platform.OS === 'ios' && { opacity: 0.6 }}>
                              <Trash2 size={16} color={theme.colors.destructive.DEFAULT} />
                            </View>
                          )}
                        </Pressable>
                      </View>
                      {spec.pricePerKg > 0 ? (
                        <View style={styles.chipWrap}>
                          <Chip
                            label="Flat add-on"
                            selected={o.priceMode !== 'per_kg'}
                            onPress={() => patchOption(i, { priceMode: 'flat' })}
                          />
                          <Chip
                            label="Per kg"
                            selected={o.priceMode === 'per_kg'}
                            onPress={() => patchOption(i, { priceMode: 'per_kg' })}
                          />
                        </View>
                      ) : null}
                      <View style={styles.chipWrap}>
                        {BAKERY_DIET_OPTIONS.map((d) => (
                          <Chip
                            key={d.value}
                            label={d.label}
                            small
                            selected={(o.dietaryTags ?? []).includes(d.value)}
                            onPress={() => toggleTag(i, 'dietaryTags', d.value)}
                          />
                        ))}
                      </View>
                      <View style={styles.chipWrap}>
                        {BAKERY_ALLERGEN_OPTIONS.map((a) => (
                          <Chip
                            key={a.value}
                            label={a.label}
                            small
                            tone="warn"
                            selected={(o.allergens ?? []).includes(a.value)}
                            onPress={() => toggleTag(i, 'allergens', a.value)}
                          />
                        ))}
                      </View>
                    </View>
                  ))}
                </View>
              );
            })}

            <View style={styles.hairline} />

            <View style={styles.toggleRow}>
              <Text style={styles.toggleLabel}>Message on the bake</Text>
              <Switch
                value={spec.allowMessage}
                onValueChange={(v) => patch({ allowMessage: v })}
                trackColor={{ true: theme.colors.herb.DEFAULT }}
              />
            </View>
            {spec.allowMessage ? (
              <Field
                label="Character limit"
                value={spec.maxMessageChars ? String(spec.maxMessageChars) : ''}
                onChangeText={(t) => patch({ maxMessageChars: int(t) })}
              />
            ) : null}
            <View style={styles.toggleRow}>
              <Text style={styles.toggleLabel}>Accept a reference photo</Text>
              <Switch
                value={spec.allowReferencePhoto}
                onValueChange={(v) => patch({ allowReferencePhoto: v })}
                trackColor={{ true: theme.colors.herb.DEFAULT }}
              />
            </View>

            <Field
              label="Notice you need"
              suffix="hours"
              value={spec.leadTimeHours ? String(spec.leadTimeHours) : ''}
              onChangeText={(t) => patch({ leadTimeHours: int(t) })}
            />
            <Text style={styles.hint}>
              Customers can only pick a slot this far ahead — an order for this item can never arrive as “as soon
              as possible”.
            </Text>

            <View style={styles.hairline} />

            <Text style={styles.fieldLabel}>Occasions</Text>
            <Text style={styles.hint}>Leave empty to show this for every occasion.</Text>
            <View style={styles.chipWrap}>
              {BAKERY_OCCASIONS.map((o) => (
                <Chip
                  key={o.value}
                  label={o.label}
                  selected={spec.occasions.includes(o.value)}
                  onPress={() => toggleOccasion(o.value)}
                />
              ))}
            </View>
          </>
        ) : null}
      </View>
    </>
  );
}

function placeholderFor(kind: BakeryOptionKind): string {
  switch (kind) {
    case 'shape':
      return 'Round, Heart, Square';
    case 'tier':
      return 'Single tier, Two tiers';
    case 'flavour':
      return 'Belgian chocolate, Red velvet';
    case 'sponge':
      return 'Vanilla, Chocolate, Whole wheat';
    case 'frosting':
      return 'Fresh cream, Fondant';
    case 'egg':
      return 'Eggless, With egg';
    case 'sweetness':
      return 'Regular, Sugar-free';
  }
}

function Field({
  label,
  value,
  onChangeText,
  prefix,
  suffix,
}: {
  label: string;
  value: string;
  onChangeText: (t: string) => void;
  prefix?: string;
  suffix?: string;
}) {
  return (
    <View style={styles.field}>
      <Text style={styles.smallLabel}>{label}</Text>
      <View style={styles.priceWrap}>
        {prefix ? <Text style={styles.pricePrefix}>{prefix}</Text> : null}
        <TextInput
          style={styles.priceInput}
          placeholder="0"
          placeholderTextColor={theme.colors.ink.muted}
          keyboardType="numbers-and-punctuation"
          value={value}
          onChangeText={onChangeText}
          accessibilityLabel={label}
        />
        {suffix ? <Text style={styles.pricePrefix}>{suffix}</Text> : null}
      </View>
    </View>
  );
}

function Chip({
  label,
  selected,
  onPress,
  small,
  tone,
}: {
  label: string;
  selected: boolean;
  onPress: () => void;
  small?: boolean;
  tone?: 'warn';
}) {
  const activeBg = tone === 'warn' ? theme.colors.amber.tint : theme.colors.herb.tint;
  const activeBorder = tone === 'warn' ? theme.colors.amber.DEFAULT : theme.colors.herb.DEFAULT;
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityState={{ selected }}
      accessibilityLabel={label}
    >
      {({ pressed }) => (
        <View
          style={[
            styles.chip,
            small && styles.chipSmall,
            selected && { backgroundColor: activeBg, borderColor: activeBorder },
            pressed && Platform.OS === 'ios' && { opacity: 0.7 },
          ]}
        >
          <Text style={[styles.chipLabel, small && styles.chipLabelSmall, selected && styles.chipLabelActive]}>
            {label}
          </Text>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
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
  fieldLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    marginTop: theme.spacing[1],
  },
  hairline: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginVertical: theme.spacing[2],
  },
  group: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.md,
    padding: theme.spacing[3],
    gap: theme.spacing[2],
  },
  groupTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  optionBlock: { gap: theme.spacing[2] },
  // Bounded, or the price box eats the row and pushes the delete off-screen.
  optionPrice: { width: 92, flexGrow: 0, flexShrink: 0 },
  optionRow: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[2] },
  rowBetween: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    minHeight: 44,
  },
  toggleLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    flex: 1,
    paddingRight: theme.spacing[2],
  },
  gridRow: { flexDirection: 'row', gap: theme.spacing[2] },
  field: { flex: 1, gap: 4 },
  smallLabel: { fontFamily: 'Inter', fontSize: 11, color: theme.colors.ink.muted },
  input: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.strong,
    borderRadius: theme.radius.sm,
    paddingHorizontal: theme.spacing[3],
    minHeight: 44,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  priceWrap: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.strong,
    borderRadius: theme.radius.sm,
    paddingHorizontal: theme.spacing[2],
    minHeight: 44,
  },
  pricePrefix: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.muted },
  priceInput: {
    flex: 1,
    minWidth: 44,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  chipWrap: { flexDirection: 'row', flexWrap: 'wrap', gap: theme.spacing[2] },
  chip: {
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    borderRadius: 999,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    minHeight: 36,
    justifyContent: 'center',
  },
  chipSmall: { paddingVertical: 4, minHeight: 28 },
  chipLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  chipLabelSmall: { fontSize: 11 },
  chipLabelActive: { color: theme.colors.ink.DEFAULT },
  addLink: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.herb.DEFAULT,
  },
});
