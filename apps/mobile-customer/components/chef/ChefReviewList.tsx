// Reusable chef-review list — the single source for rendering a chef's public
// reviews. Used inline by the Reviews tab on the chef detail screen AND by the
// standalone /chef/reviews/[id] route (kept for deep links), so the row layout
// and loading/error/empty states never drift between the two surfaces.
//
// Renders plain Views (not a FlatList) so it can live inside a parent
// ScrollView without nested-VirtualizedList warnings; review lists are small.

import { ActivityIndicator, StyleSheet, Text, View } from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useChefReviews, type ChefReview } from '../../hooks/useChefs';

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleDateString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

interface ReviewRowProps {
  review: ChefReview;
}

function ReviewRow({ review }: ReviewRowProps) {
  return (
    <View style={styles.card}>
      <View style={styles.cardHeader}>
        <Text style={styles.customerName} numberOfLines={1}>
          {review.customerName || 'Customer'}
        </Text>
        <Text style={styles.date}>{formatDate(review.createdAt)}</Text>
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
    return (
      <View style={styles.centered}>
        <Text style={styles.emptyText}>No reviews yet.</Text>
      </View>
    );
  }

  return (
    <View style={styles.list}>
      {reviews.map((review) => (
        <ReviewRow key={review.id} review={review} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  list: {
    gap: 12,
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
    borderRadius: 12,
    padding: 16,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
  },
  cardHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 12,
  },
  customerName: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },
  date: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
  },
  starRow: { flexDirection: 'row', alignItems: 'center', marginTop: 6 },
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
