// Bulk subscription prep view (#50) — "tomorrow you owe N lunches / M dinners".
// Rolls up the chef's confirmed meal-plan days for a day into a per-dish manifest
// + a packing list, with mark-prepared (which the customer then sees live).

import { useState } from 'react';
import {
  ActivityIndicator,
  Platform,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { Check, ChevronLeft, Inbox } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import {
  usePrepManifest,
  useMarkPrepared,
  type PrepManifestLine,
  type PrepPackingRow,
} from '../../hooks/useMealPlans';

function ymd(offset: number): string {
  const d = new Date();
  d.setDate(d.getDate() + offset);
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${d.getFullYear()}-${m}-${day}`;
}

const DAYS = [
  { label: 'Today', value: ymd(0) },
  { label: 'Tomorrow', value: ymd(1) },
  { label: 'In 2 days', value: ymd(2) },
];

function VariantDot({ variant }: { variant: 'veg' | 'nonveg' }) {
  return (
    <View
      style={[
        styles.dot,
        { backgroundColor: variant === 'veg' ? theme.colors.diet.veg : theme.colors.diet.nonVeg },
      ]}
    />
  );
}

export default function PrepScreen() {
  const [date, setDate] = useState(DAYS[1]!.value); // default tomorrow
  const { data, isLoading, isError, refetch, isRefetching } = usePrepManifest(date);
  const mark = useMarkPrepared();

  const manifest = data?.manifest ?? [];
  const packing = data?.packingList ?? [];
  const totals = data?.totals;
  const lunch = manifest.filter((m) => m.slot === 'lunch');
  const dinner = manifest.filter((m) => m.slot === 'dinner');

  function markDish(line: PrepManifestLine) {
    if (line.prepared >= line.total) return;
    mark.mutate({ date, slot: line.slot, variant: line.variant, dishName: line.dishName });
  }
  function markDay(row: PrepPackingRow) {
    if (row.status === 'prepared') return;
    mark.mutate({ dayIds: [row.dayId] });
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
        >
          {({ pressed }) => (
            <View style={pressed && Platform.OS === 'ios' && { opacity: 0.6 }}>
              <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text style={styles.title}>Prep</Text>
        <View style={{ width: 24 }} />
      </View>

      {/* Day selector */}
      <View style={styles.tabBar}>
        {DAYS.map((d) => {
          const sel = date === d.value;
          return (
            <Pressable
              key={d.value}
              style={styles.tabItem}
              onPress={() => setDate(d.value)}
              accessibilityRole="tab"
              accessibilityState={{ selected: sel }}
              accessibilityLabel={d.label}
              android_ripple={{
                color: sel ? `${theme.colors.ink.DEFAULT}14` : `${theme.colors.ink.DEFAULT}0d`,
                borderless: false,
              }}
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.tabInner,
                    sel && styles.tabInnerSel,
                    pressed && Platform.OS === 'ios' && { opacity: 0.7 },
                  ]}
                >
                  <Text style={[styles.tabText, sel && styles.tabTextSel]}>{d.label}</Text>
                </View>
              )}
            </Pressable>
          );
        })}
      </View>

      {isLoading ? (
        <View style={styles.centered}>
          <ActivityIndicator color={theme.colors.ink.DEFAULT} />
        </View>
      ) : isError ? (
        <View style={styles.empty}>
          <Text style={styles.emptyTitle}>Couldn't load the prep list</Text>
          <Text style={styles.emptyText}>Check your connection and try again.</Text>
          <Pressable
            onPress={() => refetch()}
            accessibilityRole="button"
            accessibilityLabel="Retry loading prep list"
            android_ripple={{ color: `${theme.colors.paper}33`, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.markBtn,
                  pressed && Platform.OS === 'ios' && { opacity: 0.85 },
                  { marginTop: theme.spacing[2] },
                ]}
              >
                <Text style={styles.markBtnText}>Retry</Text>
              </View>
            )}
          </Pressable>
        </View>
      ) : (
        <ScrollView
          contentContainerStyle={styles.scroll}
          refreshControl={<RefreshControl refreshing={isRefetching} onRefresh={refetch} tintColor={theme.colors.ink.muted} />}
        >
          {/* Totals headline */}
          {totals && totals.total > 0 ? (
            <View style={styles.totalsCard}>
              <Text style={styles.totalsText}>
                You owe <Text style={styles.totalsStrong}>{totals.lunch} lunch</Text> ·{' '}
                <Text style={styles.totalsStrong}>{totals.dinner} dinner</Text>
              </Text>
              <Text style={styles.totalsSub} accessibilityLabel={`${totals.prepared} of ${totals.total} prepared`}>
                {totals.prepared}/{totals.total} prepared
              </Text>
            </View>
          ) : null}

          {manifest.length === 0 ? (
            <View style={styles.empty}>
              <Inbox size={40} color={theme.colors.ink.muted} strokeWidth={1.5} />
              <Text style={styles.emptyTitle}>Nothing to prep</Text>
              <Text style={styles.emptyText}>
                No confirmed subscription meals for this day yet.
              </Text>
            </View>
          ) : (
            <>
              <ManifestSection title="Lunch" lines={lunch} onMark={markDish} busy={mark.isPending} />
              <ManifestSection title="Dinner" lines={dinner} onMark={markDish} busy={mark.isPending} />

              {/* Packing list */}
              <Text style={styles.sectionLabel}>Packing list</Text>
              <View style={styles.card}>
                {packing.map((row, i) => (
                  <View key={row.dayId} style={[styles.packRow, i < packing.length - 1 && styles.divider]}>
                    <VariantDot variant={row.variant} />
                    <View style={{ flex: 1 }}>
                      <Text style={styles.packName} numberOfLines={1}>
                        {row.dishName || 'Dish'}
                      </Text>
                      <Text style={styles.packSub} numberOfLines={1}>
                        {row.customerName || 'Customer'} · {row.slot} · {row.planNumber}
                      </Text>
                    </View>
                    {row.status === 'prepared' ? (
                      <View style={styles.donePill}>
                        <Check size={13} color={theme.colors.success.soft} />
                        <Text style={styles.donePillText}>Prepared</Text>
                      </View>
                    ) : (
                      <Pressable
                        onPress={() => markDay(row)}
                        disabled={mark.isPending}
                        hitSlop={6}
                        accessibilityRole="button"
                        accessibilityLabel={`Mark ${row.dishName || 'dish'} for ${row.customerName || 'customer'} prepared`}
                        android_ripple={{ color: `${theme.colors.paper}33`, borderless: false }}
                      >
                        {({ pressed }) => (
                          <View
                            style={[
                              styles.markBtn,
                              pressed && Platform.OS === 'ios' && { opacity: 0.85 },
                            ]}
                          >
                            <Text style={styles.markBtnText}>Mark</Text>
                          </View>
                        )}
                      </Pressable>
                    )}
                  </View>
                ))}
              </View>
            </>
          )}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

function ManifestSection({
  title,
  lines,
  onMark,
  busy,
}: {
  title: string;
  lines: PrepManifestLine[];
  onMark: (line: PrepManifestLine) => void;
  busy: boolean;
}) {
  if (lines.length === 0) return null;
  return (
    <>
      <Text style={styles.sectionLabel}>{title}</Text>
      <View style={styles.card}>
        {lines.map((line, i) => {
          const done = line.prepared >= line.total;
          return (
            <View key={`${line.variant}-${line.dishName}`} style={[styles.dishRow, i < lines.length - 1 && styles.divider]}>
              <VariantDot variant={line.variant} />
              <View style={{ flex: 1 }}>
                <Text style={styles.dishName} numberOfLines={1}>{line.dishName || 'Dish'}</Text>
                <Text style={styles.dishSub}>
                  {line.total} owed{line.prepared > 0 ? ` · ${line.prepared} prepared` : ''}
                </Text>
              </View>
              <Text style={styles.countBadge}>×{line.total}</Text>
              {done ? (
                <View style={styles.donePill}>
                  <Check size={13} color={theme.colors.success.soft} />
                  <Text style={styles.donePillText}>Done</Text>
                </View>
              ) : (
                <Pressable
                  onPress={() => onMark(line)}
                  disabled={busy}
                  hitSlop={6}
                  accessibilityRole="button"
                  accessibilityLabel={`Mark ${line.dishName || 'dish'} prepared`}
                  android_ripple={{ color: `${theme.colors.paper}33`, borderless: false }}
                >
                  {({ pressed }) => (
                    <View
                      style={[
                        styles.markBtn,
                        pressed && Platform.OS === 'ios' && { opacity: 0.85 },
                      ]}
                    >
                      <Text style={styles.markBtnText}>Mark prepared</Text>
                    </View>
                  )}
                </Pressable>
              )}
            </View>
          );
        })}
      </View>
    </>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  title: { fontFamily: 'Geist-Bold', fontSize: 20, color: theme.colors.ink.DEFAULT },
  scroll: { padding: theme.spacing[4], gap: theme.spacing[3], paddingBottom: theme.spacing[10] },

  tabBar: {
    flexDirection: 'row',
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[2],
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.md,
    padding: 4,
    gap: 4,
  },
  tabItem: { flex: 1 },
  tabInner: { paddingVertical: theme.spacing[2], alignItems: 'center', borderRadius: theme.radius.DEFAULT },
  tabInnerSel: { backgroundColor: theme.colors.paper, ...theme.shadow[1] },
  tabText: { fontFamily: 'Inter', fontSize: 14, color: theme.colors.ink.soft },
  tabTextSel: { fontFamily: 'Inter-SemiBold', color: theme.colors.ink.DEFAULT },

  totalsCard: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    padding: theme.spacing[4],
    ...theme.shadow[1],
  },
  totalsText: { fontFamily: 'Inter', fontSize: 16, color: theme.colors.ink.DEFAULT },
  totalsStrong: { fontFamily: 'Inter-SemiBold', color: theme.colors.ink.DEFAULT },
  totalsSub: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: theme.colors.ink.soft,
    marginTop: 4,
    fontVariant: ['tabular-nums'],
  },

  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    letterSpacing: 0.3,
    textTransform: 'uppercase',
    color: theme.colors.ink.soft,
    marginTop: theme.spacing[2],
  },
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[4],
    ...theme.shadow[1],
  },
  divider: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  dot: { width: 10, height: 10, borderRadius: 5 },

  dishRow: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[3], paddingVertical: theme.spacing[3] },
  dishName: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.ink.DEFAULT },
  dishSub: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.muted, marginTop: 2 },
  countBadge: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },

  packRow: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[3], paddingVertical: theme.spacing[3] },
  packName: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: theme.colors.ink.DEFAULT },
  packSub: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.muted, marginTop: 2 },

  markBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.full,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
  },
  markBtnText: { fontFamily: 'Inter-SemiBold', fontSize: 12, color: theme.colors.paper },
  donePill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    backgroundColor: theme.colors.success.tint,
    borderRadius: theme.radius.full,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
  },
  donePillText: { fontFamily: 'Inter-SemiBold', fontSize: 12, color: theme.colors.success.soft },

  empty: { alignItems: 'center', paddingHorizontal: theme.spacing[8], paddingTop: theme.spacing[10], gap: theme.spacing[2] },
  emptyTitle: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: theme.colors.ink.DEFAULT, marginTop: theme.spacing[2] },
  emptyText: { fontFamily: 'Inter', fontSize: 14, color: theme.colors.ink.muted, textAlign: 'center', lineHeight: 20 },
});
