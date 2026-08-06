import {
  FlatList,
  Platform,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { Image } from 'expo-image';
import { router } from 'expo-router';
import { AlertCircle, CalendarDays, ChefHat, ChevronRight, Wallet } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';

import {
  useMyMealPlans,
  useCustomerRefundChoices,
  type MealPlan,
} from '../../hooks/useMealPlans';
import { formatDateRange, mealPlanStatusMeta } from '../../lib/meal-plan';
import { useDockClearance } from '../navigation/Dock';
import { formatMoney } from '../../lib/format';
import { HAIRLINE } from '../../lib/hairline';

// MealPlanList — the shared list of every meal plan the customer has booked (#196),
// with a status chip and an "approval needed" flag. Headerless + self-contained (it
// owns the useMyMealPlans query, loading / error / empty states), so it drops into the
// standalone /meal-plans route, the Plans tab, and the Orders-tab segment identically.
//
// Photo-forward pass: this list was 100% text even though the chef's photo is
// already on the wire (MealPlan.chef.profileImage) and thrown away — the one
// place in the app where "photo-forward, chrome-light" was fully ignored. Each
// row now leads with the chef's face, same blurhash-placeholder + fixed-footprint
// treatment MealPlanDayList uses for dish photos, so a missing image degrades to
// a neutral avatar instead of a blank flash or layout shift. The plan number,
// previously the loudest line on the card (caps, first), is now a quiet
// single-line caption under the chef name — the same fix #1047-adjacent work
// just made on the Orders list — and card weight (elevated / default / quiet)
// mirrors OrderCard's three tiers so a live plan visibly outweighs a settled or
// cancelled one instead of every row carrying identical mass.

const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}0F`;
const CTA_RIPPLE = `${customerColors.canvas}33`;
// Same blurhash placeholder MealPlanDayList/ChefCard/MenuItemCard use — one
// photo placeholder token across the app, not a per-screen invention.
const PHOTO_BLURHASH = 'L6PZfSi_.AyE_3t7t7R**0o#DgR4';

// Card weight — mirrors OrderCard's three tiers (#orders-card-weight) so the
// Plans and Orders lists read as one system. 'elevated' = still in flight
// (needs the customer's eye), 'default' = completed (a settled receipt),
// 'quiet' = cancelled/expired (recede).
type CardWeight = 'elevated' | 'default' | 'quiet';

function getCardWeight(status: string): CardWeight {
  if (status === 'cancelled' || status === 'expired') return 'quiet';
  if (status === 'completed') return 'default';
  return 'elevated';
}

export function MealPlanList() {
  const { data, isLoading, isError, refetch, isRefetching } = useMyMealPlans();
  const plans = data?.data ?? [];
  // Refunds the chef has agreed but the customer hasn't yet routed (wallet vs original).
  // Surfaced as a banner above the list so the RBI medium choice isn't buried in a plan.
  const { data: refundData } = useCustomerRefundChoices();
  const pendingRefunds = refundData?.data?.length ?? 0;
  // The floating Dock overlays scene content; pad the list bottom so the last
  // plan clears it. Harmless (~dock height of extra space) on the one consumer
  // shown outside the tabs (the standalone /meal-plans route).
  const dockPad = useDockClearance();

  if (isLoading) {
    return (
      <View style={styles.listContent}>
        <SkeletonRow />
        <SkeletonRow />
        <SkeletonRow />
      </View>
    );
  }
  if (isError) return <ErrorState onRetry={() => void refetch()} />;

  return (
    <FlatList
      data={plans}
      keyExtractor={(p) => p.id}
      renderItem={({ item }) => <PlanRow plan={item} />}
      contentContainerStyle={[styles.listContent, { paddingBottom: dockPad }]}
      refreshControl={
        <RefreshControl
          refreshing={isRefetching}
          onRefresh={refetch}
          tintColor={customerColors.coral.DEFAULT}
        />
      }
      ListHeaderComponent={
        pendingRefunds > 0 ? <RefundChoiceBanner count={pendingRefunds} /> : null
      }
      ListEmptyComponent={
        <View style={styles.empty}>
          <View style={styles.emptyIconWrap}>
            <CalendarDays size={32} color={customerColors.charcoal.soft} strokeWidth={1.5} />
          </View>
          <Text style={styles.emptyTitle}>No meal plans yet</Text>
          <Text style={styles.emptyText}>
            Found a chef you love? Pre-book a week of meals from their profile.
          </Text>
        </View>
      }
    />
  );
}

// A tappable banner nudging the customer to route their agreed refunds (wallet vs original).
// Coral-tinted so it reads as an action, not an error.
function RefundChoiceBanner({ count }: { count: number }) {
  return (
    <Pressable
      onPress={() => router.push('/meal-plans/refund-choices' as never)}
      accessibilityRole="button"
      accessibilityLabel={`Choose where ${count} refund${count === 1 ? '' : 's'} go`}
      android_ripple={{ color: ROW_RIPPLE, borderless: false }}
    >
      {({ pressed }) => (
        <View style={[styles.banner, pressed && Platform.OS === 'ios' && styles.pressed]}>
          <View style={styles.bannerIcon}>
            <Wallet size={18} color={customerColors.coral.DEFAULT} />
          </View>
          <View style={{ flex: 1 }}>
            <Text style={styles.bannerTitle}>
              {count === 1 ? 'A refund is ready' : `${count} refunds are ready`}
            </Text>
            <Text style={styles.bannerText}>
              Choose your HomeChef wallet (instant) or your original payment method.
            </Text>
          </View>
          <ChevronRight size={18} color={customerColors.coral.DEFAULT} />
        </View>
      )}
    </Pressable>
  );
}

function SkeletonRow() {
  return (
    <View style={styles.cardOuter}>
      <View style={styles.cardInner}>
        <View style={styles.row}>
          <View style={[styles.avatar, styles.skeletonChip]} />
          <View style={styles.content}>
            <View style={styles.topRow}>
              <View style={{ flex: 1, gap: 6 }}>
                <View style={[styles.skeletonLine, { width: '60%', height: 15 }]} />
                <View style={[styles.skeletonLine, { width: '40%', height: 11 }]} />
              </View>
              <View style={[styles.skeletonChip, { width: 84, height: 22, borderRadius: 999 }]} />
            </View>
            <View style={styles.hairline} />
            <View style={[styles.skeletonLine, { width: '75%', height: 12 }]} />
          </View>
        </View>
      </View>
    </View>
  );
}

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
  const weight = getCardWeight(plan.status);
  const days = plan.days ?? [];
  const chefName = plan.chef?.businessName ?? 'Your chef';
  const chefImage = plan.chef?.profileImage;
  return (
    <Pressable
      onPress={() => router.push(`/meal-plans/${plan.id}` as never)}
      accessibilityRole="button"
      accessibilityLabel={`Meal plan from ${chefName}, ${meta.label}`}
      android_ripple={{ color: ROW_RIPPLE, borderless: false }}
    >
      {({ pressed }) => (
        <View
          style={[
            styles.cardOuter,
            weight === 'elevated' && styles.cardOuterElevated,
            weight === 'quiet' && styles.cardOuterQuiet,
            pressed && Platform.OS === 'ios' && styles.pressed,
          ]}
        >
          <View style={styles.cardInner}>
            <View style={styles.row}>
              {chefImage ? (
                <Image
                  source={{ uri: chefImage }}
                  style={styles.avatar}
                  contentFit="cover"
                  placeholder={{ blurhash: PHOTO_BLURHASH }}
                  transition={150}
                  accessibilityElementsHidden
                />
              ) : (
                // Missing chef photo — neutral avatar placeholder, same 56×56
                // footprint as a matched photo so no card ever shifts layout
                // depending on whether the chef has uploaded a picture.
                <View style={[styles.avatar, styles.avatarPlaceholder]}>
                  <ChefHat
                    size={22}
                    color={customerColors.charcoal.soft}
                    strokeWidth={1.5}
                    accessibilityElementsHidden
                  />
                </View>
              )}

              <View style={styles.content}>
                <View style={styles.topRow}>
                  <View style={styles.chefInfo}>
                    <Text
                      style={[styles.chefName, weight === 'quiet' && styles.chefNameQuiet]}
                      numberOfLines={1}
                    >
                      {chefName}
                    </Text>
                    {/* Plan number — was the loudest, first line on every card
                        (caps, top). Now a quiet single-line caption; the chef's
                        face and name lead instead. */}
                    <Text style={styles.planNo} numberOfLines={1} ellipsizeMode="middle">
                      {plan.mealPlanNumber}
                    </Text>
                  </View>
                  <View style={[styles.chip, { backgroundColor: meta.bg }]}>
                    <Text style={[styles.chipText, { color: meta.color }]}>{meta.label}</Text>
                  </View>
                </View>

                <View style={styles.hairline} />

                <View style={styles.cardBottom}>
                  <Text style={styles.meta}>
                    {/* MEALS, not days (#1040). `plan.days` is the array of booked
                        meals — a plan can book lunch AND dinner on one date — so the
                        count contradicted the date range beside it: "7 Aug – 9 Aug ·
                        5 days" for three calendar dates, "4 Aug – 4 Aug · 2 days" for
                        one. Meals is also what the customer is buying. */}
                    {formatDateRange(plan.startDate, plan.endDate)} · {days.length} meal
                    {days.length === 1 ? '' : 's'} · {formatMoney(plan.total)}
                  </Text>
                  <ChevronRight size={18} color={customerColors.charcoal.soft} />
                </View>
              </View>
            </View>
          </View>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  listContent: { padding: 16, paddingTop: 8, gap: 12 },
  banner: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    backgroundColor: customerColors.coral.tint,
    borderRadius: 12,
    borderWidth: HAIRLINE,
    borderColor: customerColors.coral.DEFAULT,
    padding: 14,
  },
  bannerIcon: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: customerColors.canvas,
    alignItems: 'center',
    justifyContent: 'center',
  },
  bannerTitle: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  bannerText: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    marginTop: 1,
    lineHeight: 18,
  },
  // Card weight system (mirrors OrderCard, #orders-card-weight). `canvas` and
  // `surface.DEFAULT` are both #FFFFFF, so every tier needs a real shadow or
  // it's invisible against the screen background — a hairline alone doesn't
  // read as a card. Base tier = 'default' (completed): shadow[1].
  cardOuter: {
    backgroundColor: customerColors.canvas,
    borderRadius: 16,
    borderWidth: HAIRLINE,
    borderColor: customerColors.hairline,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.04,
    shadowRadius: 3,
    elevation: 1,
  },
  // 'elevated' — a live plan still in flight (awaiting the chef/customer, or
  // actively running). shadow[2]: lifts it visibly above a settled or
  // cancelled row without introducing a new colour.
  cardOuterElevated: {
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.1,
    shadowRadius: 12,
    elevation: 4,
  },
  // 'quiet' — cancelled/expired. Flattened to the hairline only so it recedes
  // behind live and completed plans instead of matching their visual mass.
  cardOuterQuiet: {
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 0 },
    shadowOpacity: 0,
    shadowRadius: 0,
    elevation: 0,
  },
  pressed: { backgroundColor: customerColors.surface.soft },
  cardInner: { borderRadius: 16, overflow: 'hidden', padding: 16 },
  row: { flexDirection: 'row', gap: 12 },
  // Chef avatar — fixed 56×56 footprint whether or not the chef has a photo, so
  // a matched image and the placeholder never shift the row layout relative to
  // each other (same rule MealPlanDayList applies to dish thumbnails).
  avatar: {
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: customerColors.surface.soft,
  },
  avatarPlaceholder: { alignItems: 'center', justifyContent: 'center' },
  content: { flex: 1, justifyContent: 'center' },
  topRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'flex-start', gap: 10 },
  chefInfo: { flex: 1, gap: 2 },
  chefName: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: customerColors.charcoal.DEFAULT },
  // Cancelled/expired plans recede — the chef name drops to the secondary
  // colour instead of full charcoal, matching the flattened card shadow.
  chefNameQuiet: { color: customerColors.charcoal.soft },
  // Plan number — was the loudest, first line on the card (caps, top). Now a
  // quiet single-line caption; truncated rather than wrapped.
  planNo: {
    fontFamily: 'Inter',
    fontSize: 11,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
  chip: { paddingHorizontal: 10, paddingVertical: 4, borderRadius: 999 },
  chipText: { fontFamily: 'Inter-SemiBold', fontSize: 11, letterSpacing: 0.1 },
  hairline: { height: StyleSheet.hairlineWidth, backgroundColor: customerColors.hairline, marginVertical: 10 },
  cardBottom: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', gap: 8 },
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
  emptyTitle: { fontFamily: 'Inter-SemiBold', fontSize: 17, color: customerColors.charcoal.DEFAULT },
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
  skeletonLine: { borderRadius: 4, backgroundColor: customerColors.hairline },
  skeletonChip: { borderRadius: 9999, backgroundColor: customerColors.surface.soft },
});
