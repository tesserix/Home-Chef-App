// Reusable chef-review list — the single source for rendering a chef's public
// reviews. Used inline by the Reviews tab on the chef detail screen AND by the
// standalone /chef/reviews/[id] route (kept for deep links), so the row layout
// and loading/error/empty states never drift between the two surfaces.
//
// Renders plain Views (not a FlatList) so it can live inside a parent
// ScrollView without nested-VirtualizedList warnings; review lists are small.

import { useRef } from 'react';
import { ActivityIndicator, Pressable, StyleSheet, Text, View } from 'react-native';
import Animated, { Easing, FadeInDown, useReducedMotion } from 'react-native-reanimated';
import { Image } from 'expo-image';
import { MoreHorizontal, Star } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { EmptyState, ReportSheet, type SheetHandle } from '@homechef/mobile-shared/ui';
import { useChefReviews, type ChefReview } from '../../hooks/useChefs';
import { useReportContent, useBlockUser } from '../../hooks/useModeration';
import { useAuthStore } from '../../store/auth-store';
import { useProfile } from '../../hooks/useProfile';

// Entrance easing — ease-out-quart, matches the app-wide motion spec (§3.5).
const ENTRANCE_EASING = Easing.bezier(0.22, 1, 0.36, 1);

// Relative date — "Today" / "3 days ago" / "2 weeks ago" — reads calmer than
// an absolute date in a review feed (spec: "relative dates charcoal-soft").
function formatRelativeDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const diffDays = Math.floor((Date.now() - d.getTime()) / (1000 * 60 * 60 * 24));
  if (diffDays <= 0) return 'Today';
  if (diffDays === 1) return 'Yesterday';
  if (diffDays < 7) return `${diffDays} days ago`;
  if (diffDays < 30) {
    const weeks = Math.floor(diffDays / 7);
    return weeks === 1 ? '1 week ago' : `${weeks} weeks ago`;
  }
  if (diffDays < 365) {
    const months = Math.floor(diffDays / 30);
    return months === 1 ? '1 month ago' : `${months} months ago`;
  }
  const years = Math.floor(diffDays / 365);
  return years === 1 ? '1 year ago' : `${years} years ago`;
}

interface ReviewerAvatarProps {
  name: string;
  avatarUrl?: string;
}

// Reviewer identity — photo when the API has one, otherwise a letter avatar
// (R2 exception: letter avatars stay fine for *people*, unlike photo surfaces).
function ReviewerAvatar({ name, avatarUrl }: ReviewerAvatarProps) {
  const initial = (name || 'C').trim().charAt(0).toUpperCase() || 'C';
  if (avatarUrl) {
    return (
      // surface-soft backgroundColor (on the Image itself) + blurhash
      // placeholder — no blank flash before the 150ms fade-in (R2).
      <Image
        source={{ uri: avatarUrl }}
        style={styles.avatarPhoto}
        contentFit="cover"
        placeholder={{ blurhash: 'L6PZfSi_.AyE_3t7t7R**0o#DgR4' }}
        transition={150}
        accessibilityElementsHidden
      />
    );
  }
  return (
    <View style={styles.avatarLetter} accessibilityElementsHidden>
      <Text style={styles.avatarLetterText}>{initial}</Text>
    </View>
  );
}

interface ReviewRowProps {
  review: ChefReview;
  /** The signed-in customer's USER id, or undefined for a guest. */
  viewerId?: string;
}

function ReviewRow({ review, viewerId }: ReviewRowProps) {
  // App Review 1.2 — reviews are user-generated content, so each row carries a
  // reachable report action and, where the reviewer's id is known, a block.
  const reportSheetRef = useRef<SheetHandle>(null);
  const reportContent = useReportContent();
  const blockUser = useBlockUser();

  // Moderation is for OTHER people's content (#1047). The overflow rendered the
  // generic report/block sheet unconditionally, so on your own review the app
  // offered to report you to its own moderation queue and to "Block Priya S."
  // — the signed-in customer, blocking herself, almost certainly into a broken
  // self-referential block row. Your own review gets a quiet ownership marker
  // instead. (Edit/Delete belong here too, but /v1/reviews is POST +
  // GET-by-order only — no update or delete endpoint exists yet.)
  const isOwnReview = Boolean(viewerId && review.customerId && review.customerId === viewerId);

  return (
    <View style={styles.card}>
      <View style={styles.cardHeader}>
        <ReviewerAvatar name={review.customerName} avatarUrl={review.customerAvatar} />
        <View style={styles.headerTextCol}>
          <Text style={styles.customerName} numberOfLines={1}>
            {review.customerName || 'Customer'}
          </Text>
          <Text style={styles.date}>{formatRelativeDate(review.createdAt)}</Text>
        </View>
        {isOwnReview ? (
          <Text style={styles.ownBadge}>Your review</Text>
        ) : (
          <Pressable
            onPress={() => reportSheetRef.current?.present()}
            accessibilityRole="button"
            accessibilityLabel="Report or block this reviewer"
            hitSlop={8}
            style={styles.reportButton}
          >
            <MoreHorizontal size={18} color={customerColors.charcoal.soft} />
          </Pressable>
        )}
      </View>
      <View style={styles.starRow}>
        <Text style={styles.star}>★</Text>
        <Text style={styles.ratingValue}>{review.overallRating}</Text>
        <Text style={styles.ratingOutOf}>/5</Text>
      </View>
      {review.title ? <Text style={styles.title}>{review.title}</Text> : null}
      {review.comment ? (
        <Text style={styles.comment}>{review.comment}</Text>
      ) : null}
      {review.chefResponse ? (
        <View style={styles.replyBlock}>
          <Text style={styles.replyLabel}>Chef’s reply</Text>
          <Text style={styles.replyText}>{review.chefResponse}</Text>
        </View>
      ) : null}

      {isOwnReview ? null : (
      <ReportSheet
        ref={reportSheetRef}
        subject="this review"
        onSubmit={async (reason, details) => {
          await reportContent.mutateAsync({
            targetType: 'review',
            targetId: review.id,
            reason,
            details,
          });
        }}
        onBlock={
          review.customerId
            ? async () => {
                await blockUser.mutateAsync({ userId: review.customerId as string });
              }
            : undefined
        }
        blockLabel={`Block ${review.customerName || 'this reviewer'}`}
      />
      )}
    </View>
  );
}

export interface ChefReviewListProps {
  chefId: string;
  /**
   * Whether rows should play their entrance stagger on this mount. Defaults
   * to true (the standalone `/chef/reviews/[id]` route always wants it on
   * its one-and-only mount). The chef-detail Reviews tab remounts this list
   * every time the tab is revealed (conditional render) — pass `false` after
   * the first reveal so revisiting the tab doesn't replay the stagger.
   */
  animateOnMount?: boolean;
}

export function ChefReviewList({ chefId, animateOnMount = true }: ChefReviewListProps) {
  const { data, isLoading, isError } = useChefReviews(chefId);
  const reviews = data?.data ?? [];

  // Who is looking (#1047), resolved ONCE for the whole list rather than per row.
  //
  // This must come from the profile, not `useAuthStore().user`: the store only
  // populates `user` in `setAuthResponse`, i.e. immediately after a fresh login.
  // `hydrateFromStorage` — the cold-launch path every returning customer takes —
  // restores the token and `isAuthenticated` but leaves `user` null. Keying
  // ownership off the store therefore silently never matched, which is exactly
  // how this shipped broken the first time.
  //
  // `profile.userId`, not `profile.id`: the latter is the CustomerProfile row id
  // (the hook's own doc comment warns about it), while reviews carry the
  // reviewer's USER id. Skipped entirely for a guest, who owns nothing here.
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const { data: profile } = useProfile({ enabled: isAuthenticated });
  const viewerId = profile?.userId;
  const reduceMotion = useReducedMotion();
  const shouldAnimate = animateOnMount && !reduceMotion;

  if (isLoading) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator color={customerColors.coral.DEFAULT} />
      </View>
    );
  }

  if (isError) {
    return (
      <View style={styles.centered}>
        <Text style={styles.emptyText}>Couldn’t load reviews. Try again.</Text>
      </View>
    );
  }

  if (reviews.length === 0) {
    // R8 — branded empty state, calm sentence, no raw "No data" string.
    return (
      <EmptyState
        icon={<Star size={26} color={customerColors.charcoal.soft} strokeWidth={1.5} />}
        title="No reviews yet"
        body="Once customers review this kitchen, their feedback shows up here."
        accentColor={customerColors.coral.DEFAULT}
      />
    );
  }

  return (
    <View style={styles.list}>
      {reviews.map((review, index) => (
        <Animated.View
          key={review.id}
          entering={
            shouldAnimate
              ? // §3.5: stagger steps 40-60ms, max 3 steps.
                FadeInDown.delay(Math.min(index, 2) * 60)
                  .duration(250)
                  .easing(ENTRANCE_EASING)
              : undefined
          }
        >
          <ReviewRow review={review} viewerId={viewerId} />
        </Animated.View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  list: {
    // Rows sit flat on white, separated by hairline — spec §1: separation by
    // hairline, not card-soup.
  },
  centered: {
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 40,
    paddingHorizontal: 32,
  },
  emptyText: {
    fontFamily: 'Inter',
    fontSize: 15,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
  },
  card: {
    backgroundColor: customerColors.canvas,
    paddingVertical: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  cardHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
  },
  avatarPhoto: {
    width: 36,
    height: 36,
    borderRadius: 18,
    // Neutral fill behind the photo so there's no blank flash before the
    // blurhash placeholder/fade resolves (R2).
    backgroundColor: customerColors.surface.soft,
  },
  avatarLetter: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
  },
  avatarLetterText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
  },
  headerTextCol: {
    flex: 1,
  },
  reportButton: {
    minHeight: 44,
    minWidth: 44,
    alignItems: 'center',
    justifyContent: 'center',
  },
  // Ownership marker where the moderation overflow would sit — a label, not a
  // control, so it needs no touch target.
  ownBadge: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    color: customerColors.charcoal.soft,
  },
  customerName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },
  date: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    marginTop: 1,
  },
  starRow: { flexDirection: 'row', alignItems: 'center', marginTop: 8 },
  star: { fontSize: 14, color: customerColors.charcoal.DEFAULT },
  ratingValue: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    marginLeft: 4,
    fontVariant: ['tabular-nums'],
  },
  ratingOutOf: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
  },
  title: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    marginTop: 8,
  },
  comment: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    lineHeight: 21,
    marginTop: 6,
  },
  replyBlock: {
    marginTop: 12,
    paddingTop: 12,
    paddingLeft: 12,
    borderLeftWidth: 2,
    borderLeftColor: customerColors.coral.DEFAULT,
  },
  replyLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    color: customerColors.coral.pressed,
    marginBottom: 2,
  },
  replyText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    lineHeight: 21,
  },
});
