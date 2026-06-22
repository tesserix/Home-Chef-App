// Admin UI kit — the small, shared building blocks every admin screen composes
// from. Built on StyleSheet + theme tokens (not NativeWind classes) so the
// look is deterministic regardless of tailwind class resolution. Ink-centric
// per .impeccable.md: one accent, hairlines over heavy borders, functional
// colour only.
import type { ReactNode } from 'react';
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { ChevronLeft, ChevronRight, Search, AlertCircle } from 'lucide-react-native';
import { router } from 'expo-router';
import { theme } from '@homechef/mobile-shared/theme';

const c = theme.colors;

type Tone = 'neutral' | 'success' | 'warning' | 'danger' | 'info';

const toneStyle: Record<Tone, { bg: string; fg: string }> = {
  neutral: { bg: c.bone, fg: c.ink.soft },
  success: { bg: c.success.tint, fg: c.success.soft },
  warning: { bg: c.amber.tint, fg: '#7a5a13' },
  danger: { bg: c.destructive.tint, fg: c.destructive.DEFAULT },
  info: { bg: c.info.tint, fg: c.info.DEFAULT },
};

/** Sticky page header: optional back button, title, subtitle, right slot. */
export function ScreenHeader({
  title,
  subtitle,
  back,
  right,
}: {
  title: string;
  subtitle?: string;
  back?: boolean;
  right?: ReactNode;
}) {
  return (
    <View style={styles.header}>
      <View style={styles.headerRow}>
        {back ? (
          <Pressable
            onPress={() => router.back()}
            hitSlop={12}
            style={styles.backBtn}
            accessibilityRole="button"
            accessibilityLabel="Go back"
          >
            <ChevronLeft size={24} color={c.ink.DEFAULT} />
          </Pressable>
        ) : null}
        <View style={{ flex: 1 }}>
          <Text style={styles.title} numberOfLines={1}>
            {title}
          </Text>
          {subtitle ? (
            <Text style={styles.subtitle} numberOfLines={2}>
              {subtitle}
            </Text>
          ) : null}
        </View>
        {right ? <View style={styles.headerRight}>{right}</View> : null}
      </View>
    </View>
  );
}

/** White card surface with a hairline + soft elevation. */
export function Card({
  children,
  onPress,
  style,
}: {
  children: ReactNode;
  onPress?: () => void;
  style?: object;
}) {
  if (onPress) {
    return (
      <Pressable
        onPress={onPress}
        style={({ pressed }) => [styles.card, pressed && styles.cardPressed, style]}
      >
        {children}
      </Pressable>
    );
  }
  return <View style={[styles.card, style]}>{children}</View>;
}

/** Metric tile: big number + label + optional delta. */
export function StatCard({
  label,
  value,
  delta,
  deltaTone = 'neutral',
}: {
  label: string;
  value: string;
  delta?: string;
  deltaTone?: Tone;
}) {
  return (
    <View style={styles.statCard}>
      <Text style={styles.statLabel} numberOfLines={1}>
        {label}
      </Text>
      <Text style={styles.statValue} numberOfLines={1} allowFontScaling={false}>
        {value}
      </Text>
      {delta ? (
        <Text style={[styles.statDelta, { color: toneStyle[deltaTone].fg }]}>{delta}</Text>
      ) : null}
    </View>
  );
}

/** Small status pill. */
export function Badge({ label, tone = 'neutral' }: { label: string; tone?: Tone }) {
  const t = toneStyle[tone];
  return (
    <View style={[styles.badge, { backgroundColor: t.bg }]}>
      <Text style={[styles.badgeText, { color: t.fg }]} numberOfLines={1}>
        {label}
      </Text>
    </View>
  );
}

/** Tappable list row: title, subtitle, optional right value + chevron. */
export function ListItem({
  title,
  subtitle,
  meta,
  right,
  badge,
  onPress,
  chevron = true,
}: {
  title: string;
  subtitle?: string;
  meta?: string;
  right?: string;
  badge?: ReactNode;
  onPress?: () => void;
  chevron?: boolean;
}) {
  const body = (
    <>
      <View style={{ flex: 1 }}>
        <View style={styles.rowTitleLine}>
          <Text style={styles.rowTitle} numberOfLines={1}>
            {title}
          </Text>
          {badge}
        </View>
        {subtitle ? (
          <Text style={styles.rowSubtitle} numberOfLines={1}>
            {subtitle}
          </Text>
        ) : null}
        {meta ? (
          <Text style={styles.rowMeta} numberOfLines={1}>
            {meta}
          </Text>
        ) : null}
      </View>
      {right ? <Text style={styles.rowRight}>{right}</Text> : null}
      {chevron && onPress ? <ChevronRight size={18} color={c.ink.muted} /> : null}
    </>
  );
  if (onPress) {
    return (
      <Pressable
        onPress={onPress}
        style={({ pressed }) => [styles.row, pressed && styles.cardPressed]}
      >
        {body}
      </Pressable>
    );
  }
  return <View style={styles.row}>{body}</View>;
}

/** Label/value pair for detail screens. */
export function Field({ label, value }: { label: string; value: ReactNode }) {
  return (
    <View style={styles.field}>
      <Text style={styles.fieldLabel}>{label}</Text>
      {typeof value === 'string' || typeof value === 'number' ? (
        <Text style={styles.fieldValue}>{String(value)}</Text>
      ) : (
        value
      )}
    </View>
  );
}

export function SectionTitle({ children }: { children: ReactNode }) {
  return <Text style={styles.sectionTitle}>{children}</Text>;
}

/** Debounce-free search input (parent owns the value). */
export function SearchBar({
  value,
  onChangeText,
  placeholder = 'Search…',
}: {
  value: string;
  onChangeText: (t: string) => void;
  placeholder?: string;
}) {
  return (
    <View style={styles.search}>
      <Search size={18} color={c.ink.muted} />
      <TextInput
        value={value}
        onChangeText={onChangeText}
        placeholder={placeholder}
        placeholderTextColor={c.ink.muted}
        style={styles.searchInput}
        autoCapitalize="none"
        autoCorrect={false}
        returnKeyType="search"
        clearButtonMode="while-editing"
      />
    </View>
  );
}

/** Full-screen centered spinner. */
export function LoadingState({ label }: { label?: string }) {
  return (
    <View style={styles.center}>
      <ActivityIndicator color={c.ink.DEFAULT} />
      {label ? <Text style={styles.centerText}>{label}</Text> : null}
    </View>
  );
}

/** Inline skeleton list while a query loads. */
export function LoadingList({ rows = 6 }: { rows?: number }) {
  return (
    <View>
      {Array.from({ length: rows }).map((_, i) => (
        <View key={i} style={styles.skelRow}>
          <View style={styles.skelLineWide} />
          <View style={styles.skelLineNarrow} />
        </View>
      ))}
    </View>
  );
}

/** Error panel with retry. */
export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <View style={styles.center}>
      <AlertCircle size={28} color={c.destructive.DEFAULT} />
      <Text style={styles.errorTitle}>Couldn’t load</Text>
      <Text style={styles.centerText}>{message}</Text>
      {onRetry ? (
        <Pressable onPress={onRetry} style={styles.retryBtn}>
          <Text style={styles.retryText}>Retry</Text>
        </Pressable>
      ) : null}
    </View>
  );
}

/** Empty list message. */
export function Empty({ title, body }: { title: string; body?: string }) {
  return (
    <View style={styles.center}>
      <Text style={styles.emptyTitle}>{title}</Text>
      {body ? <Text style={styles.centerText}>{body}</Text> : null}
    </View>
  );
}

/** A horizontally scrollable filter chip row. */
export function FilterChips<T extends string>({
  options,
  value,
  onChange,
}: {
  options: { key: T; label: string }[];
  value: T;
  onChange: (v: T) => void;
}) {
  return (
    <ScrollView
      horizontal
      showsHorizontalScrollIndicator={false}
      contentContainerStyle={styles.chips}
    >
      {options.map((o) => {
        const active = o.key === value;
        return (
          <Pressable
            key={o.key}
            onPress={() => onChange(o.key)}
            style={[styles.chip, active && styles.chipActive]}
          >
            <Text style={[styles.chipText, active && styles.chipTextActive]}>{o.label}</Text>
          </Pressable>
        );
      })}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  header: { paddingTop: 8, paddingBottom: 12 },
  headerRow: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  backBtn: { marginLeft: -6, padding: 2 },
  headerRight: { marginLeft: 8 },
  title: {
    fontFamily: 'Geist',
    fontSize: 26,
    letterSpacing: -0.3,
    color: c.ink.DEFAULT,
  },
  subtitle: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: c.ink.muted,
    marginTop: 2,
  },
  card: {
    backgroundColor: c.paper,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: c.mist.DEFAULT,
    padding: 14,
  },
  cardPressed: { opacity: 0.6 },
  statCard: {
    flex: 1,
    backgroundColor: c.bone,
    borderRadius: 12,
    padding: 14,
    minHeight: 84,
    justifyContent: 'center',
  },
  statLabel: { fontFamily: 'Inter', fontSize: 12, color: c.ink.muted },
  statValue: {
    fontFamily: 'Geist-Bold',
    fontSize: 24,
    color: c.ink.DEFAULT,
    marginTop: 4,
  },
  statDelta: { fontFamily: 'Inter-Medium', fontSize: 12, marginTop: 2 },
  badge: {
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: 999,
    alignSelf: 'flex-start',
  },
  badgeText: { fontFamily: 'Inter-SemiBold', fontSize: 11, letterSpacing: 0.2 },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    paddingVertical: 14,
    borderBottomWidth: 1,
    borderBottomColor: c.mist.DEFAULT,
  },
  rowTitleLine: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  rowTitle: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: c.ink.DEFAULT, flexShrink: 1 },
  rowSubtitle: { fontFamily: 'Inter', fontSize: 13, color: c.ink.soft, marginTop: 2 },
  rowMeta: { fontFamily: 'Inter', fontSize: 12, color: c.ink.muted, marginTop: 2 },
  rowRight: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: c.ink.DEFAULT },
  field: {
    paddingVertical: 10,
    borderBottomWidth: 1,
    borderBottomColor: c.mist.DEFAULT,
  },
  fieldLabel: { fontFamily: 'Inter', fontSize: 12, color: c.ink.muted, marginBottom: 3 },
  fieldValue: { fontFamily: 'Inter-Medium', fontSize: 15, color: c.ink.DEFAULT },
  sectionTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    letterSpacing: 0.3,
    color: c.ink.muted,
    textTransform: 'uppercase',
    marginTop: 20,
    marginBottom: 8,
  },
  search: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    backgroundColor: c.bone,
    borderRadius: 10,
    paddingHorizontal: 12,
    height: 44,
  },
  searchInput: { flex: 1, fontFamily: 'Inter', fontSize: 15, color: c.ink.DEFAULT, padding: 0 },
  center: { alignItems: 'center', justifyContent: 'center', paddingVertical: 48, gap: 8 },
  centerText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: c.ink.muted,
    textAlign: 'center',
    paddingHorizontal: 24,
  },
  errorTitle: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: c.ink.DEFAULT, marginTop: 4 },
  emptyTitle: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: c.ink.DEFAULT },
  retryBtn: {
    marginTop: 8,
    backgroundColor: c.ink.DEFAULT,
    paddingHorizontal: 18,
    paddingVertical: 10,
    borderRadius: 10,
  },
  retryText: { color: c.paper, fontFamily: 'Inter-SemiBold', fontSize: 14 },
  skelRow: { paddingVertical: 16, borderBottomWidth: 1, borderBottomColor: c.mist.DEFAULT, gap: 8 },
  skelLineWide: { height: 14, width: '70%', borderRadius: 6, backgroundColor: c.bone },
  skelLineNarrow: { height: 12, width: '40%', borderRadius: 6, backgroundColor: c.bone },
  chips: { gap: 8, paddingVertical: 4 },
  chip: {
    paddingHorizontal: 14,
    height: 34,
    borderRadius: 999,
    backgroundColor: c.bone,
    alignItems: 'center',
    justifyContent: 'center',
  },
  chipActive: { backgroundColor: c.ink.DEFAULT },
  chipText: { fontFamily: 'Inter-Medium', fontSize: 13, color: c.ink.soft },
  chipTextActive: { color: c.paper },
});

export { toneStyle };
export type { Tone };
