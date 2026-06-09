import { useMemo, useState } from 'react';
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
import { useQuery } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import { Skeleton } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { api } from '../lib/api';

// ----- Data types ------------------------------------------------------------

export interface Review {
  id: string;
  customerName: string;
  rating: number;
  comment: string;
  createdAt: string;
  reply?: string;
}

interface ReviewsResponse {
  reviews: Review[];
  averageRating: number;
  totalReviews: number;
}

// Wire-level shape from the backend (chefs.go:715). The screen consumes
// the legacy ReviewsResponse shape above, so we adapt here. Field names
// here mirror models.ReviewResponse in the Go backend (overallRating,
// chefResponse) — NOT the screen's Review interface.
interface BackendReview {
  id: string;
  orderId?: string;
  overallRating: number;
  comment: string;
  customerName: string;
  customerAvatar?: string;
  createdAt: string;
  chefResponse?: string;
  chefRespondedAt?: string | null;
}

interface BackendReviewsEnvelope {
  data: BackendReview[];
  pagination?: { page: number; limit: number; total: number };
}

function adaptReview(b: BackendReview): Review {
  return {
    id: b.id,
    customerName: b.customerName || 'Customer',
    rating: b.overallRating,
    comment: b.comment,
    createdAt: b.createdAt,
    reply: b.chefResponse || undefined,
  };
}

// ----- Hook ------------------------------------------------------------------

export function useReviews() {
  return useQuery<ReviewsResponse>({
    queryKey: ['chef', 'reviews'],
    queryFn: async () => {
      const res = await api.get<BackendReviewsEnvelope>('/chef/reviews');
      const items = (res.data?.data ?? []).map(adaptReview);
      const totalReviews = res.data?.pagination?.total ?? items.length;
      // averageRating isn't returned by the backend yet; compute it from
      // the page we received. Good-enough for the v1 list — when the
      // backend grows a `GET /chef/reviews/summary` we can swap to that.
      const averageRating =
        items.length === 0
          ? 0
          : items.reduce((sum, r) => sum + (r.rating || 0), 0) / items.length;
      return { reviews: items, averageRating, totalReviews };
    },
    staleTime: 60_000,
  });
}

// ----- Helpers ---------------------------------------------------------------

type StarFilter = 'all' | '5' | '4' | '3' | '2-';

const FILTER_LABELS: { key: StarFilter; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: '5', label: '5★' },
  { key: '4', label: '4★' },
  { key: '3', label: '3★' },
  { key: '2-', label: '≤2★' },
];

function passesFilter(review: Review, filter: StarFilter): boolean {
  if (filter === 'all') return true;
  if (filter === '5') return review.rating === 5;
  if (filter === '4') return review.rating === 4;
  if (filter === '3') return review.rating === 3;
  if (filter === '2-') return review.rating <= 2;
  return true;
}

function relativeDate(iso: string): string {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return '';
  const mins = Math.max(0, Math.floor((Date.now() - t) / 60_000));
  if (mins < 60) return mins < 1 ? 'just now' : `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  if (days < 7) return `${days}d ago`;
  return new Date(iso).toLocaleDateString('en-IN', {
    day: 'numeric',
    month: 'short',
  });
}

// ----- Sub-components --------------------------------------------------------

interface FilterTabProps {
  item: { key: StarFilter; label: string };
  active: boolean;
  onPress: () => void;
}

function FilterTab({ item, active, onPress }: FilterTabProps) {
  return (
    <Pressable
      onPress={onPress}
      hitSlop={6}
      accessibilityRole="tab"
      accessibilityState={{ selected: active }}
      style={filterTabStyles.root}
    >
      <Text
        style={[
          filterTabStyles.label,
          active && filterTabStyles.labelActive,
        ]}
      >
        {item.label}
      </Text>
      <View
        style={[
          filterTabStyles.indicator,
          active && filterTabStyles.indicatorActive,
        ]}
      />
    </Pressable>
  );
}

interface ReviewRowProps {
  review: Review;
}

function ReviewRow({ review }: ReviewRowProps) {
  return (
    <Pressable
      onPress={() => router.push(`/review/${review.id}` as `${string}`)}
      accessibilityRole="button"
      accessibilityLabel={`Review by ${review.customerName}, ${review.rating} stars. Tap to reply.`}
    >
      {({ pressed }) => (
        <View
          style={[rowStyles.root, pressed && rowStyles.rootPressed]}
        >
          {/* Leading — star count */}
          <View style={rowStyles.starBlock}>
            <Text style={rowStyles.starGlyph}>★</Text>
            <Text style={rowStyles.starCount}>{review.rating}</Text>
          </View>

          {/* Centre — name + comment snippet */}
          <View style={rowStyles.body}>
            <View style={rowStyles.nameRow}>
              <Text style={rowStyles.customerName} numberOfLines={1}>
                {review.customerName}
              </Text>
              <Text style={rowStyles.date}>
                {relativeDate(review.createdAt)}
              </Text>
            </View>
            <Text style={rowStyles.snippet} numberOfLines={1}>
              {review.comment}
            </Text>
          </View>

          {/* Trailing — Replied pill (only when reply exists) */}
          {review.reply ? (
            <View style={rowStyles.repliedPill}>
              <Text style={rowStyles.repliedPillLabel}>Replied</Text>
            </View>
          ) : null}
        </View>
      )}
    </Pressable>
  );
}

// ----- Skeleton rows ---------------------------------------------------------

function ReviewRowSkeleton() {
  return (
    <View style={rowStyles.root}>
      <Skeleton width={28} height={28} style={{ borderRadius: theme.radius.sm }} />
      <View style={[rowStyles.body, { gap: 6 }]}>
        <Skeleton width={120} height={12} />
        <Skeleton width={200} height={11} />
      </View>
    </View>
  );
}

// ----- Screen ----------------------------------------------------------------

export default function ReviewsScreen() {
  const { data, isLoading, isError, refetch, isRefetching } = useReviews();
  const [activeFilter, setActiveFilter] = useState<StarFilter>('all');

  const filteredReviews = useMemo(() => {
    const reviews = data?.reviews ?? [];
    return reviews.filter((r) => passesFilter(r, activeFilter));
  }, [data?.reviews, activeFilter]);

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* Zone A — Command bar */}
      <View style={styles.commandBar}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Go back"
        >
          {({ pressed }) => (
            <View style={[styles.backBtn, pressed && { opacity: 0.6 }]}>
              <ChevronLeft
                size={22}
                color={theme.colors.ink.DEFAULT}
                strokeWidth={2}
              />
            </View>
          )}
        </Pressable>
        <Text style={styles.commandTitle}>Reviews</Text>
      </View>

      {/* Summary row — inline below command bar, above filter tabs */}
      {!isLoading && !isError && (data?.totalReviews ?? 0) > 0 && (
        <View style={styles.summaryRow}>
          <Text style={styles.summaryRating}>
            {(data?.averageRating ?? 0).toFixed(1)}
          </Text>
          <Text style={styles.summaryStarGlyph}>★</Text>
          <Text style={styles.summaryDivider}>·</Text>
          <Text style={styles.summaryCount}>
            {data?.totalReviews ?? 0}{' '}
            {(data?.totalReviews ?? 0) === 1 ? 'review' : 'reviews'}
          </Text>
        </View>
      )}

      {/* Filter tab row — bare-text underline pattern from orders.tsx */}
      {!isLoading && !isError && (
        <View style={styles.filterBar}>
          {FILTER_LABELS.map((item) => (
            <FilterTab
              key={item.key}
              item={item}
              active={activeFilter === item.key}
              onPress={() => setActiveFilter(item.key)}
            />
          ))}
        </View>
      )}

      {/* List */}
      {isLoading ? (
        <View style={styles.skeletonStack}>
          {[0, 1, 2, 3, 4].map((i) => (
            <ReviewRowSkeleton key={i} />
          ))}
        </View>
      ) : isError ? (
        <View style={styles.errorBlock}>
          <Text style={styles.emptyHeadline}>Couldn't load reviews</Text>
          <Text style={styles.emptyBody}>
            Check your connection and try again.
          </Text>
          <Pressable
            onPress={() => refetch()}
            style={({ pressed }) => [
              styles.retryButton,
              pressed && { opacity: 0.85 },
            ]}
          >
            <Text style={styles.retryButtonLabel}>Retry</Text>
          </Pressable>
        </View>
      ) : (
        <FlatList
          data={filteredReviews}
          keyExtractor={(item) => item.id}
          contentContainerStyle={styles.listContent}
          showsVerticalScrollIndicator={false}
          refreshControl={
            <RefreshControl
              refreshing={isRefetching}
              onRefresh={refetch}
              tintColor={theme.colors.ink.DEFAULT}
            />
          }
          renderItem={({ item }) => <ReviewRow review={item} />}
          ListEmptyComponent={<EmptyState filter={activeFilter} totalReviews={data?.totalReviews ?? 0} />}
        />
      )}
    </SafeAreaView>
  );
}

// ----- Empty state -----------------------------------------------------------

interface EmptyStateProps {
  filter: StarFilter;
  totalReviews: number;
}

function EmptyState({ filter, totalReviews }: EmptyStateProps) {
  const isFiltered = filter !== 'all' && totalReviews > 0;
  return (
    <View style={styles.emptyBlock}>
      <Text style={styles.emptyHeadline}>
        {isFiltered ? 'No reviews here' : 'No reviews yet'}
      </Text>
      <Text style={styles.emptyBody}>
        {isFiltered
          ? 'No reviews match this star filter. Try All to see everything.'
          : 'When customers review their order, you’ll see them here. Your first review is on its way.'}
      </Text>
    </View>
  );
}

// ----- Styles ----------------------------------------------------------------

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },

  // Zone A — Command bar (matches orders.tsx exactly)
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
    paddingHorizontal: theme.spacing[3],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
  },
  backBtn: {
    width: 36,
    height: 36,
    alignItems: 'center',
    justifyContent: 'center',
  },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },

  // Summary row — sits between command bar and filter bar
  summaryRow: {
    flexDirection: 'row',
    alignItems: 'baseline',
    gap: theme.spacing[1],
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[3],
  },
  summaryRating: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
    letterSpacing: -0.1,
  },
  summaryStarGlyph: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.amber.DEFAULT,
  },
  summaryDivider: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.mist.strong,
    marginHorizontal: theme.spacing[1],
  },
  summaryCount: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },

  // Filter bar — matches orders.tsx tabBar pattern
  filterBar: {
    flexDirection: 'row',
    paddingHorizontal: theme.spacing[4],
    gap: theme.spacing[5],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },

  // List
  listContent: {
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[10],
  },

  // Skeleton stack
  skeletonStack: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[2],
  },

  // Empty + error states (matches orders.tsx emptyBlock)
  emptyBlock: {
    paddingHorizontal: 0,
    paddingTop: theme.spacing[10],
  },
  emptyHeadline: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.h2.size,
    color: theme.colors.ink.DEFAULT,
    letterSpacing: -0.2,
    marginBottom: theme.spacing[2],
  },
  emptyBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    lineHeight: 20,
    maxWidth: 300,
  },
  errorBlock: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[10],
  },
  retryButton: {
    marginTop: theme.spacing[4],
    alignSelf: 'flex-start',
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[5],
    paddingVertical: theme.spacing[3],
    minHeight: 44,
    justifyContent: 'center',
  },
  retryButtonLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.paper,
  },
});

const filterTabStyles = StyleSheet.create({
  root: {
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[2],
  },
  label: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.1,
    paddingBottom: 6,
  },
  labelActive: {
    color: theme.colors.ink.DEFAULT,
  },
  indicator: {
    height: 2,
    backgroundColor: 'transparent',
    borderRadius: 1,
  },
  indicatorActive: {
    backgroundColor: theme.colors.ink.DEFAULT,
  },
});

const rowStyles = StyleSheet.create({
  root: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    minHeight: 56,
    paddingVertical: theme.spacing[3],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  rootPressed: {
    backgroundColor: theme.colors.bone,
  },

  // Leading star block — amber glyph + digit, fixed width so all rows align
  starBlock: {
    flexDirection: 'row',
    alignItems: 'baseline',
    gap: 2,
    width: 28,
    flexShrink: 0,
  },
  starGlyph: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.amber.DEFAULT,
    lineHeight: 16,
  },
  starCount: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
    lineHeight: 16,
  },

  // Centre block
  body: {
    flex: 1,
    gap: 2,
  },
  nameRow: {
    flexDirection: 'row',
    alignItems: 'baseline',
    justifyContent: 'space-between',
    gap: theme.spacing[2],
  },
  customerName: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  date: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    flexShrink: 0,
  },
  snippet: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.soft,
    lineHeight: 16,
  },

  // Trailing replied pill — muted, no persimmon
  repliedPill: {
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.sm,
    paddingHorizontal: 6,
    paddingVertical: 2,
    flexShrink: 0,
  },
  repliedPillLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    color: theme.colors.ink.muted,
    letterSpacing: 0.3,
  },
});
