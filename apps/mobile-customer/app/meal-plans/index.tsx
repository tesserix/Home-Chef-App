import {
  FlatList,
  Platform,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { AlertCircle, CalendarDays, ChevronLeft, ChevronRight } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useMyMealPlans, type MealPlan } from '../../hooks/useMealPlans';
import { formatDateRange, mealPlanStatusMeta } from '../../lib/meal-plan';

// Android ripple tints — translucent tokens, never a new literal colour.
const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}0F`;
const ICON_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const CTA_RIPPLE = `${customerColors.canvas}33`;

// My tiffin plans (#196): every plan the customer has booked, with a status
// chip. Plans needing approval (chef trimmed the days) are flagged so the
// customer can open and approve/reject.
export default function MyMealPlansScreen() {
  const { data, isLoading, isError, refetch, isRefetching } = useMyMealPlans();
  const plans = data?.data ?? [];

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: ICON_RIPPLE, borderless: true }}
        >
          <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>My meal plans</Text>
        <View style={{ width: 24 }} />
      </View>

      {isLoading ? (
        <View style={styles.listContent}>
          <SkeletonRow />
          <SkeletonRow />
          <SkeletonRow />
        </View>
      ) : isError ? (
        <ErrorState onRetry={() => void refetch()} />
      ) : (
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
            <View style={styles.empty}>
              <View style={styles.emptyIconWrap}>
                <CalendarDays size={32} color={customerColors.charcoal.soft} strokeWidth={1.5} />
              </View>
              <Text style={styles.emptyTitle}>No meal plans yet</Text>
              <Text style={styles.emptyText}>
                Found a chef you love? Pre-book a week of meals from their
                profile.
              </Text>
            </View>
          }
        />
      )}
    </SafeAreaView>
  );
}

// ─── Loading skeleton — matches PlanRow proportions ──────────────────────────

function SkeletonRow() {
  return (
    <View style={styles.card}>
      <View style={styles.cardTop}>
        <View style={[styles.skeletonLine, { width: '30%', height: 12 }]} />
        <View style={[styles.skeletonChip, { width: 84, height: 22 }]} />
      </View>
      <View style={[styles.skeletonLine, { width: '55%', height: 16, marginTop: 8 }]} />
      <View style={styles.cardBottom}>
        <View style={[styles.skeletonLine, { width: '65%', height: 12 }]} />
      </View>
    </View>
  );
}

// ─── Error state (R8) ─────────────────────────────────────────────────────────

function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <View style={styles.empty}>
      <View style={styles.emptyIconWrap}>
        <AlertCircle size={32} color={customerColors.charcoal.soft} />
      </View>
      <Text style={styles.emptyTitle}>Something went wrong</Text>
      <Text style={styles.emptyText}>We could not load your meal plans. Please try again.</Text>
      <Pressable
        onPress={onRetry}
        accessibilityRole="button"
        accessibilityLabel="Retry loading meal plans"
        android_ripple={{ color: CTA_RIPPLE, borderless: false }}
      >
        {({ pressed }) => (
          <View style={[styles.retryCta, pressed && Platform.OS === 'ios' && styles.retryCtaPressed]}>
            <Text style={styles.retryCtaText}>Try again</Text>
          </View>
        )}
      </Pressable>
    </View>
  );
}

function PlanRow({ plan }: { plan: MealPlan }) {
  const meta = mealPlanStatusMeta(plan.status);
  const days = plan.days ?? [];
  return (
    <Pressable
      onPress={() => router.push(`/meal-plans/${plan.id}` as never)}
      accessibilityRole="button"
      accessibilityLabel={`Meal plan ${plan.mealPlanNumber}, ${meta.label}`}
      android_ripple={{ color: ROW_RIPPLE, borderless: false }}
    >
      {({ pressed }) => (
        <View style={[styles.card, pressed && Platform.OS === 'ios' && styles.pressed]}>
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
    backgroundColor: customerColors.canvas,
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
    fontVariant: ['tabular-nums'],
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
  meta: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
    flex: 1,
  },
  empty: { alignItems: 'center', paddingHorizontal: 32, paddingTop: 80, gap: 4 },
  emptyIconWrap: {
    width: 64,
    height: 64,
    borderRadius: 32,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 8,
  },
  emptyTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 17,
    color: customerColors.charcoal.DEFAULT,
  },
  emptyText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
    lineHeight: 20,
  },
  retryCta: {
    marginTop: 16,
    minHeight: 44,
    justifyContent: 'center',
    paddingHorizontal: 24,
    borderRadius: 8,
    backgroundColor: customerColors.coral.DEFAULT,
  },
  retryCtaPressed: { backgroundColor: customerColors.coral.pressed },
  retryCtaText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.canvas },
  // Skeleton block geometry — static hairline rects, no shimmer (matches the
  // convention used on (tabs)/orders.tsx).
  skeletonLine: { borderRadius: 4, backgroundColor: customerColors.hairline },
  skeletonChip: { borderRadius: 9999, backgroundColor: customerColors.surface.soft },
});
