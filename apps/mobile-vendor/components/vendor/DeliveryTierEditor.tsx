import { Pressable, StyleSheet, Text, TextInput, View } from 'react-native';
import { Plus, X } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';

import {
  type DeliveryFeeCap,
  type TierRow,
  bandCeiling,
  validateTierRows,
} from '../../lib/deliveryTiers';
import { currencySymbol } from '../../lib/format';

// The chef publishes one ladder — "up to 5 km ₹75, up to 10 km ₹150" — and it is
// the fee the customer pays. Shared by the profile screen and onboarding so a
// chef sets it in exactly the same place, with the same rules, either way.

interface DeliveryTierEditorProps {
  rows: TierRow[];
  cap: DeliveryFeeCap;
  onChange: (rows: TierRow[]) => void;
  currency?: string;
}

export function DeliveryTierEditor({ rows, cap, onChange, currency }: DeliveryTierEditorProps) {
  const error = validateTierRows(rows, cap, currency);
  const symbol = currencySymbol(currency);
  const canAdd = rows.length < cap.maxBands;

  function update(index: number, patch: Partial<TierRow>) {
    onChange(rows.map((row, i) => (i === index ? { ...row, ...patch } : row)));
  }

  return (
    <View style={styles.container}>
      <View style={styles.card}>
        {rows.map((row, i) => {
          const km = parseFloat(row.km);
          const ceiling = Number.isFinite(km) && km > 0 ? bandCeiling(cap, km) : null;
          return (
            <View key={i} style={[styles.row, i > 0 && styles.rowDivider]}>
              <View style={styles.field}>
                <Text style={styles.prefix}>Up to</Text>
                <TextInput
                  value={row.km}
                  onChangeText={(km) => update(i, { km })}
                  keyboardType="decimal-pad"
                  placeholder="5"
                  placeholderTextColor={theme.colors.ink.muted}
                  style={styles.input}
                  accessibilityLabel={`Band ${i + 1} distance in km`}
                />
                <Text style={styles.suffix}>km</Text>
              </View>
              <View style={styles.field}>
                <Text style={styles.prefix}>{symbol}</Text>
                <TextInput
                  value={row.fee}
                  onChangeText={(fee) => update(i, { fee })}
                  keyboardType="decimal-pad"
                  placeholder={ceiling != null ? String(Math.round(ceiling)) : '75'}
                  placeholderTextColor={theme.colors.ink.muted}
                  style={styles.input}
                  accessibilityLabel={`Band ${i + 1} fee in ${symbol}`}
                />
              </View>
              <Pressable
                onPress={() => onChange(rows.filter((_, j) => j !== i))}
                hitSlop={10}
                accessibilityRole="button"
                accessibilityLabel={`Remove band ${i + 1}`}
                style={styles.remove}
              >
                <X size={16} color={theme.colors.ink.muted} />
              </Pressable>
            </View>
          );
        })}

        {rows.length === 0 ? (
          <Text style={styles.empty}>
            No bands yet — add one to set your own delivery price.
          </Text>
        ) : null}
      </View>

      {canAdd ? (
        <Pressable
          onPress={() => onChange([...rows, { km: '', fee: '' }])}
          accessibilityRole="button"
          accessibilityLabel="Add a distance band"
          style={styles.add}
        >
          <Plus size={16} color={theme.colors.herb.DEFAULT} />
          <Text style={styles.addLabel}>Add a band</Text>
        </Pressable>
      ) : null}

      {error ? (
        <Text style={styles.error}>{error}</Text>
      ) : (
        <Text style={styles.hint}>
          Customers pay exactly this at checkout — you're never asked for more later.
          The platform allows up to {symbol}
          {Math.round(bandCeiling(cap, 5))} for 5 km and {symbol}
          {Math.round(bandCeiling(cap, 10))} for 10 km.
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { gap: theme.spacing[2] },
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    minHeight: 48,
  },
  rowDivider: { borderTopWidth: 1, borderTopColor: theme.colors.mist.DEFAULT },
  field: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
  },
  prefix: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  suffix: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  input: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    paddingVertical: 6,
    fontVariant: ['tabular-nums'],
  },
  remove: { padding: theme.spacing[1] },
  empty: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    padding: theme.spacing[3],
  },
  add: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
    minHeight: 44,
    paddingHorizontal: theme.spacing[1],
  },
  addLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.herb.DEFAULT,
  },
  hint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.soft,
    lineHeight: 16,
  },
  error: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.destructive.DEFAULT,
    lineHeight: 16,
  },
});
