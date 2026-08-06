// SelectField — a labelled row that shows the value already saved and opens a
// searchable sheet to change it.
//
// A horizontal chip strip cannot do that job for a long list: it shows the
// first two entries alphabetically and gives no clue what is currently set, so
// a saved value reads as an empty field.

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
import { Check, ChevronDown, Search, X } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { filterOptions } from '../lib/optionSearch';

interface SelectFieldProps {
  label: string;
  value: string;
  options: string[];
  onChange: (value: string) => void;
  placeholder?: string;
  /** Sheet title; defaults to the field label. */
  title?: string;
  searchPlaceholder?: string;
  loading?: boolean;
  /** Validation message shown under the trigger, as a text input would. */
  error?: string;
  hasBorderBottom?: boolean;
  /** Off inside a card that already pads its own contents. */
  hasPadding?: boolean;
}

export function SelectField({
  label,
  value,
  options,
  onChange,
  placeholder = 'Select',
  title,
  searchPlaceholder = 'Search',
  loading = false,
  error,
  hasBorderBottom = true,
  hasPadding = true,
}: SelectFieldProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const shown = useMemo(() => filterOptions(options, query), [options, query]);

  function close() {
    setOpen(false);
    setQuery('');
  }

  return (
    <View style={[hasPadding && styles.row, hasBorderBottom && styles.rowBorder]}>
      <Text style={styles.label}>{label}</Text>
      <Pressable
        onPress={() => setOpen(true)}
        disabled={loading}
        accessibilityRole="button"
        accessibilityLabel={value ? `${label}: ${value}. Change` : `${label}: ${placeholder}`}
        style={styles.trigger}
      >
        <Text style={[styles.triggerText, !value && styles.triggerPlaceholder]} numberOfLines={1}>
          {loading ? 'Loading…' : value || placeholder}
        </Text>
        <ChevronDown size={16} color={theme.colors.ink.muted} />
      </Pressable>
      {error ? <Text style={styles.error}>{error}</Text> : null}

      <Modal visible={open} animationType="slide" transparent onRequestClose={close}>
        <Pressable style={styles.backdrop} onPress={close} />
        <SafeAreaView edges={['bottom']} style={styles.sheet}>
          <View style={styles.sheetHead}>
            <Text style={styles.sheetTitle}>{title ?? label}</Text>
            <Pressable onPress={close} accessibilityRole="button" accessibilityLabel="Close" hitSlop={8}>
              <X size={20} color={theme.colors.ink.DEFAULT} />
            </Pressable>
          </View>

          <View style={styles.searchRow}>
            <Search size={16} color={theme.colors.ink.muted} />
            <TextInput
              value={query}
              onChangeText={setQuery}
              placeholder={searchPlaceholder}
              placeholderTextColor={theme.colors.ink.muted}
              autoCorrect={false}
              style={styles.searchInput}
            />
          </View>

          <ScrollView keyboardShouldPersistTaps="handled">
            {shown.length === 0 ? (
              <Text style={styles.empty}>No match for “{query.trim()}”.</Text>
            ) : (
              shown.map((option) => {
                const selected = option === value;
                return (
                  <Pressable
                    key={option}
                    onPress={() => {
                      onChange(option);
                      close();
                    }}
                    accessibilityRole="button"
                    accessibilityState={{ selected }}
                    style={[styles.option, selected && styles.optionSelected]}
                  >
                    <Text style={styles.optionLabel} numberOfLines={1}>
                      {option}
                    </Text>
                    {selected ? <Check size={16} color={theme.colors.ink.DEFAULT} /> : null}
                  </Pressable>
                );
              })
            )}
          </ScrollView>
        </SafeAreaView>
      </Modal>
    </View>
  );
}

const styles = StyleSheet.create({
  row: { paddingHorizontal: 16, paddingVertical: 10 },
  rowBorder: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  label: { fontSize: 12, color: theme.colors.ink.muted, marginBottom: 6 },
  trigger: {
    minHeight: 44,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    backgroundColor: theme.colors.bone,
    borderRadius: 12,
    paddingHorizontal: 12,
    paddingVertical: 10,
  },
  triggerText: { flex: 1, fontSize: 15, color: theme.colors.ink.DEFAULT },
  triggerPlaceholder: { color: theme.colors.ink.muted },
  error: { fontSize: 12, color: theme.colors.destructive.DEFAULT, marginTop: 6 },
  backdrop: { flex: 1, backgroundColor: 'rgba(0,0,0,0.4)' },
  sheet: {
    backgroundColor: theme.colors.paper,
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
    maxHeight: '75%',
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
  searchRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginHorizontal: 16,
    marginTop: 12,
    paddingHorizontal: 12,
    backgroundColor: theme.colors.bone,
    borderRadius: 12,
  },
  searchInput: {
    flex: 1,
    minHeight: 44,
    fontSize: 15,
    color: theme.colors.ink.DEFAULT,
  },
  option: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 12,
    paddingHorizontal: 16,
    paddingVertical: 14,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  optionSelected: { backgroundColor: theme.colors.bone },
  optionLabel: { flex: 1, fontSize: 15, color: theme.colors.ink.DEFAULT },
  empty: { padding: 24, textAlign: 'center', fontSize: 14, color: theme.colors.ink.muted },
});
