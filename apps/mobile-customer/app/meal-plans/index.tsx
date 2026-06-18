import {
  FlatList,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { CalendarDays, ChevronLeft, ChevronRight } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useMyMealPlans, type MealPlan } from '../../hooks/useMealPlans';
import { formatDateRange, mealPlanStatusMeta } from '../../lib/meal-plan';

// My tiffin plans (#196): every plan the customer has booked, with a status
// chip. Plans needing approval (chef trimmed the days) are flagged so the
// customer can open and approve/reject.
export default function MyMealPlansScreen() {
  const { data, isLoading, refetch, isRefetching } = useMyMealPlans();
  const plans = data?.data ?? [];

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable onPress={() => router.back()} hitSlop={8} accessibilityLabel="Go back">
          <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>My meal plans</Text>
        <View style={{ width: 24 }} />
      </View>

      <FlatList
        data={plans}
        keyExtractor={(p) => p.id}
        renderItem={({ item }) => <PlanRow plan={item} />}
        contentContainerStyle={styles.listContent}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={refetch}
            tintColor={customerColors.coral.DEFAULT}
          />
        }
        ListEmptyComponent={
          isLoading ? null : (
            <View style={styles.empty}>
              <CalendarDays size={40} color={customerColors.charcoal.soft} strokeWidth={1.5} />
              <Text style={styles.emptyTitle}>No meal plans yet</Text>
              <Text style={styles.emptyText}>
                Found a chef you love? Pre-book a week of meals from their
                profile.
              </Text>
            </View>
          )
        }
      />
    </SafeAreaView>
  );
}

function PlanRow({ plan }: { plan: MealPlan }) {
  const meta = mealPlanStatusMeta(plan.status);
  const days = plan.days ?? [];
  return (
    <Pressable
      onPress={() => router.push(`/meal-plans/${plan.id}` as never)}
      accessibilityRole="button"
    >
      {({ pressed }) => (
        <View style={[styles.card, pressed && styles.pressed]}>
          <View style={styles.cardTop}>
            <Text style={styles.planNo}>{plan.mealPlanNumber}</Text>
            <View style={[styles.chip, { backgroundColor: meta.bg }]}>
              <Text style={[styles.chipText, { color: meta.color }]}>
                {meta.label}
              </Text>
            </View>
          </View>
          <Text style={styles.chefName} numberOfLines={1}>
            {plan.chef?.businessName ?? 'Your chef'}
          </Text>
          <View style={styles.cardBottom}>
            <Text style={styles.meta}>
              {formatDateRange(plan.startDate, plan.endDate)} · {days.length} day
              {days.length === 1 ? '' : 's'} · ₹{plan.total.toFixed(0)}
            </Text>
            <ChevronRight size={18} color={customerColors.charcoal.soft} />
          </View>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  title: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 18,
    color: customerColors.charcoal.DEFAULT,
  },
  listContent: { padding: 16, paddingTop: 8, gap: 12 },
  card: {
    backgroundColor: customerColors.surface.DEFAULT,
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    padding: 16,
    gap: 6,
  },
  pressed: { backgroundColor: customerColors.surface.soft },
  cardTop: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  planNo: {
    fontFamily: 'Inter-Medium',
    fontSize: 13,
    color: customerColors.charcoal.soft,
  },
  chip: { paddingHorizontal: 10, paddingVertical: 4, borderRadius: 999 },
  chipText: { fontFamily: 'Inter-SemiBold', fontSize: 12 },
  chefName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 17,
    color: customerColors.charcoal.DEFAULT,
  },
  cardBottom: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginTop: 2,
  },
  meta: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft },
  empty: { alignItems: 'center', paddingHorizontal: 32, paddingTop: 80, gap: 8 },
  emptyTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 17,
    color: customerColors.charcoal.DEFAULT,
    marginTop: 8,
  },
  emptyText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
    lineHeight: 20,
  },
});
