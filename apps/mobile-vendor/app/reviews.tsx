import { useMemo, useState } from 'react';
import {
  FlatList,
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

interface BackendReviewsSummary {
  averageRating: number;
  totalReviews: number;
  distribution?: Record<string, number>;
}

// ----- Hook ------------------------------------------------------------------

export function useReviews() {
  return useQuery<ReviewsResponse>({
    queryKey: ['chef', 'reviews'],
    queryFn: async () => {
      // Two parallel calls — list (paginated) + summary (aggregate
      // across ALL reviews). Summary failure is non-fatal: we fall
      // back to a client-side computation over the first page so
      // older backends without /reviews/summary still render the tab.
      const [listRes, summaryRes] = await Promise.all([
        api.get<BackendReviewsEnvelope>('/chef/reviews'),
        api
          .get<BackendReviewsSummary>('/chef/reviews/summary')
          .catch(() => null),
      ]);

      const items = (listRes.data?.data ?? []).map(adaptReview);

      if (summaryRes && typeof summaryRes.data?.averageRating === 'number') {
        return {
          reviews: items,
          averageRating: summaryRes.data.averageRating,
          totalReviews: summaryRes.data.totalReviews ?? items.length,
        };
      }

      const totalReviews = listRes.data?.pagination?.total ?? items.length;
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
  { key: '2-', label: 'Low (1–2★)' },
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

// Horizontal chip variant of the v2 segmented control (UI-V2-SPEC §5) —
// five segments with "(N)" counts are too tight for a fixed track, so the
// filters render as a scrolling chip row: active = ink fill + paper text,
// inactive = paper bg + mist.strong border.
function FilterTab({ item, active, onPress }: FilterTabProps) {
  return (
    <Pressable
      onPress={onPress}
      hitSlop={6}
      accessibilityRole="tab"
      accessibilityState={{ selected: active }}
      android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
    >
      {({ pressed }) => (
        // Inner-View pattern — visual styles on the View, never a
        // function-style array on the Pressable (iOS drops them).
        <View
          style={[
            filterTabStyles.chip,
            active ? filterTabStyles.chipActive : filterTabStyles.chipInactive,
            pressed && Platform.OS === 'ios' && { opacity: 0.85 },
          ]}
        >
          <Text
            style={[
              filterTabStyles.label,
              active && filterTabStyles.labelActive,
            ]}
          >
            {item.label}
          </Text>
        </View>
      )}
    </Pressable>
  );
}

interface ReviewRowProps {
  review: Review;
  first: boolean;
  last: boolean;
}

// One segment of the white "group card" (UI-V2-SPEC §1/§9). First-in-list
// gets top radii, last gets bottom radii; shadow lives on the outer shell
// so the inner overflow-hidden clip (needed for the pressed-state bone
// fill to respect the radius) doesn't cut it off.
function ReviewRow({ review, first, last }: ReviewRowProps) {
  return (
    <View
      style={[
        segmentStyles.shell,
        first && segmentStyles.top,
        last && segmentStyles.bottom,
      ]}
    >
      <View
        style={[
          segmentStyles.clip,
          first && segmentStyles.top,
          last && segmentStyles.bottom,
        ]}
      >
        <Pressable
          onPress={() => router.push({ pathname: '/review/[reviewId]', params: { reviewId: review.id } })}
          accessibilityRole="button"
          accessibilityLabel={`Review by ${review.customerName}, ${review.rating} stars. Tap to reply.`}
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
        >
          {({ pressed }) => (
            <View
              style={[
                rowStyles.root,
                pressed && Platform.OS === 'ios' && rowStyles.rootPressed,
              ]}
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
        {!last && <View style={segmentStyles.separator} />}
      </View>
    </View>
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

  const reviews = data?.reviews ?? [];

  const filterCounts = useMemo<Record<StarFilter, number>>(
    () => ({
      all: reviews.length,
      '5': reviews.filter((r) => r.rating === 5).length,
      '4': reviews.filter((r) => r.rating === 4).length,
      '3': reviews.filter((r) => r.rating === 3).length,
      '2-': reviews.filter((r) => r.rating <= 2).length,
    }),
    [reviews],
  );

  const filteredReviews = useMemo(() => {
    return reviews.filter((r) => passesFilter(r, activeFilter));
  }, [reviews, activeFilter]);

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* Zone A — Command bar */}
      <View style={styles.commandBar}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.backBtn,
                pressed && Platform.OS === 'ios' && { opacity: 0.6 },
              ]}
            >
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

      {/* Hero — average rating on a white paper card (UI-V2-SPEC §1) */}
      {!isLoading && !isError && (data?.totalReviews ?? 0) > 0 && (
        <View style={styles.summaryCard}>
          <View style={styles.summaryValueRow}>
            <Text style={styles.summaryRating}>
              {(data?.averageRating ?? 0).toFixed(1)}
            </Text>
            <Text style={styles.summaryStarGlyph}>★</Text>
          </View>
          <Text style={styles.summaryCount}>
            {data?.totalReviews ?? 0}{' '}
            {(data?.totalReviews ?? 0) === 1 ? 'review' : 'reviews'}
          </Text>
        </View>
      )}

      {/* Filter chips — horizontal chip-row variant of the v2 segmented
          control (five segments with counts are too tight for a track) */}
      {!isLoading && !isError && (
        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          style={styles.filterBar}
          contentContainerStyle={styles.filterBarContent}
        >
          {FILTER_LABELS.map((item) => (
            <FilterTab
              key={item.key}
              item={{ ...item, label: `${item.label} (${filterCounts[item.key]})` }}
              active={activeFilter === item.key}
              onPress={() => setActiveFilter(item.key)}
            />
          ))}
        </ScrollView>
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
            accessibilityRole="button"
            android_ripple={{ color: `${theme.colors.paper}30`, borderless: false }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.retryButton,
                  pressed && Platform.OS === 'ios' && { opacity: 0.85 },
                ]}
              >
                <Text style={styles.retryButtonLabel}>Retry</Text>
              </View>
            )}
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
          renderItem={({ item, index }) => (
            <ReviewRow
              review={item}
              first={index === 0}
              last={index === filteredReviews.length - 1}
            />
          )}
          ListEmptyComponent={<EmptyState filter={activeFilter} totalReviews={reviews.length} />}
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
  root: { flex: 1, backgroundColor: theme.colors.bone },

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

  // Hero — average rating on a white paper card (UI-V2-SPEC §1)
  summaryCard: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[3],
    ...theme.shadow[1],
  },
  summaryValueRow: {
    flexDirection: 'row',
    alignItems: 'baseline',
    gap: theme.spacing[1],
  },
  summaryRating: {
    fontFamily: 'Geist-Bold',
    fontSize: 32,
    lineHeight: 36,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
    letterSpacing: -0.4,
  },
  summaryStarGlyph: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.h2.size,
    color: theme.colors.amber.DEFAULT,
  },
  summaryCount: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.2,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[1],
  },

  // Filter chip row — horizontal scroll, no edge-to-edge hairline
  filterBar: {
    flexGrow: 0,
  },
  filterBarContent: {
    flexDirection: 'row',
    gap: theme.spacing[2],
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[3],
  },

  // List
  listContent: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[1],
    paddingBottom: theme.spacing[10],
  },

  // Skeleton stack — card-shaped while loading so the layout doesn't jump
  skeletonStack: {
    marginHorizontal: theme.spacing[4],
    marginTop: theme.spacing[2],
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    overflow: 'hidden',
    ...theme.shadow[1],
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
  chip: {
    minHeight: 36,
    paddingHorizontal: theme.spacing[3],
    borderRadius: theme.radius.full,
    alignItems: 'center',
    justifyContent: 'center',
  },
  chipActive: {
    backgroundColor: theme.colors.ink.DEFAULT,
  },
  chipInactive: {
    backgroundColor: theme.colors.paper,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
  },
  label: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.1,
  },
  labelActive: {
    color: theme.colors.paper,
  },
});

// Group-card segment shell (UI-V2-SPEC §1) — mirrors orders.tsx HistoryRow.
const segmentStyles = StyleSheet.create({
  shell: {
    backgroundColor: theme.colors.paper,
    ...theme.shadow[1],
  },
  top: {
    borderTopLeftRadius: theme.radius.lg,
    borderTopRightRadius: theme.radius.lg,
  },
  bottom: {
    borderBottomLeftRadius: theme.radius.lg,
    borderBottomRightRadius: theme.radius.lg,
  },
  clip: {
    overflow: 'hidden',
    backgroundColor: theme.colors.paper,
  },
  separator: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginLeft: theme.spacing[4], // inset — aligned to row content
  },
});

const rowStyles = StyleSheet.create({
  // Row inside the group card — separators come from the segment shell.
  root: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    minHeight: 56,
    paddingVertical: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
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

  // Trailing replied chip — UI-V2-SPEC §2 style: mist bg, ink.soft text
  repliedPill: {
    backgroundColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.full,
    paddingHorizontal: theme.spacing[2],
    paddingVertical: 3,
    flexShrink: 0,
  },
  repliedPillLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.soft,
    letterSpacing: 0.2,
  },
});
