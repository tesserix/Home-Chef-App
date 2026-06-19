import { Pressable, StyleSheet, Text, View } from 'react-native';
import { CalendarDays, ChevronRight } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import type { MealPlan } from '../../hooks/useMealPlans';

interface Props {
  plan: MealPlan;
  onPress: () => void;
}

function customerName(plan: MealPlan): string {
  const c = plan.customer;
  const full = `${c?.firstName ?? ''} ${c?.lastName ?? ''}`.trim();
  return full.length > 0 ? full : 'A customer';
}

function dateRange(plan: MealPlan): string {
  const fmt = (iso: string) =>
    new Date(iso).toLocaleDateString(undefined, {
      day: 'numeric',
      month: 'short',
    });
  if (!plan.startDate || !plan.endDate) return '';
  return `${fmt(plan.startDate)} – ${fmt(plan.endDate)}`;
}

export function MealPlanRequestCard({ plan, onPress }: Props) {
  const days = plan.days ?? [];
  const veg = days.filter((d) => d.variant === 'veg').length;
  const nonVeg = days.length - veg;

  return (
    <Pressable onPress={onPress} accessibilityRole="button">
      {({ pressed }) => (
        <View style={[styles.card, pressed && styles.pressed]}>
          <View style={styles.header}>
            <Text style={styles.name} numberOfLines={1}>
              {customerName(plan)}
            </Text>
            <Text style={styles.total}>₹{plan.total.toFixed(0)}</Text>
          </View>

          <View style={styles.metaRow}>
            <CalendarDays
              size={14}
              color={theme.colors.ink.muted}
              strokeWidth={1.75}
            />
            <Text style={styles.meta}>
              {dateRange(plan)} · {days.length} day
              {days.length === 1 ? '' : 's'}
            </Text>
          </View>

          <View style={styles.footer}>
            <View style={styles.chips}>
              {veg > 0 ? (
                <View style={[styles.chip, styles.vegChip]}>
                  <Text style={[styles.chipText, { color: theme.colors.diet.veg }]}>
                    {veg} veg
                  </Text>
                </View>
              ) : null}
              {nonVeg > 0 ? (
                <View style={[styles.chip, styles.nonVegChip]}>
                  <Text
                    style={[styles.chipText, { color: theme.colors.diet.nonVeg }]}
                  >
                    {nonVeg} non-veg
                  </Text>
                </View>
              ) : null}
            </View>
            <View style={styles.cta}>
              <Text style={styles.ctaText}>Review</Text>
              <ChevronRight
                size={16}
                color={theme.colors.herb.DEFAULT}
                strokeWidth={2}
              />
            </View>
          </View>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    padding: theme.spacing[4],
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[3],
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    ...theme.shadow[1],
  },
  pressed: { backgroundColor: theme.colors.bone },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: theme.spacing[2],
  },
  name: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: theme.colors.ink.DEFAULT,
  },
  total: {
    fontFamily: 'Geist-Bold',
    fontSize: 18,
    color: theme.colors.ink.DEFAULT,
    marginLeft: theme.spacing[2],
  },
  metaRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
    marginBottom: theme.spacing[3],
  },
  meta: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: theme.colors.ink.soft,
  },
  footer: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  chips: { flexDirection: 'row', gap: theme.spacing[2] },
  chip: {
    paddingHorizontal: theme.spacing[2],
    paddingVertical: 2,
    borderRadius: theme.radius.sm,
  },
  vegChip: { backgroundColor: theme.colors.success.tint },
  nonVegChip: { backgroundColor: theme.colors.destructive.tint },
  chipText: { fontFamily: 'Inter-SemiBold', fontSize: 12 },
  cta: { flexDirection: 'row', alignItems: 'center', gap: 2 },
  ctaText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: theme.colors.herb.DEFAULT,
  },
});
