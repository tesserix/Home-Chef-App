// Reusable chef-review list — the single source for rendering a chef's public
// reviews. Used inline by the Reviews tab on the chef detail screen AND by the
// standalone /chef/reviews/[id] route (kept for deep links), so the row layout
// and loading/error/empty states never drift between the two surfaces.
//
// Renders plain Views (not a FlatList) so it can live inside a parent
// ScrollView without nested-VirtualizedList warnings; review lists are small.

import { ActivityIndicator, StyleSheet, Text, View } from 'react-native';
import Animated, { Easing, FadeInDown, useReducedMotion } from 'react-native-reanimated';
import { Image } from 'expo-image';
import { Star } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { EmptyState } from '@homechef/mobile-shared/ui';
import { useChefReviews, type ChefReview } from '../../hooks/useChefs';

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
      <Image
        source={{ uri: avatarUrl }}
        style={styles.avatarPhoto}
        contentFit="cover"
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
}

function ReviewRow({ review }: ReviewRowProps) {
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
    </View>
  );
}

export interface ChefReviewListProps {
  chefId: string;
}

export function ChefReviewList({ chefId }: ChefReviewListProps) {
  const { data, isLoading, isError } = useChefReviews(chefId);
  const reviews = data?.data ?? [];
  const reduceMotion = useReducedMotion();

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
            reduceMotion
              ? undefined
              // §3.5: stagger steps 40-60ms, max 3 steps.
              : FadeInDown.delay(Math.min(index, 2) * 60)
                  .duration(250)
                  .easing(ENTRANCE_EASING)
          }
        >
          <ReviewRow review={review} />
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
