// Customer-facing list of a chef's reviews. Reached by tapping the rating on
// the chef detail screen. Read-only; data from the public GET /chefs/:id/reviews.

import {
  ActivityIndicator,
  FlatList,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useChefReviews, type ChefReview } from '../../../hooks/useChefs';

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleDateString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

function ReviewRow({ review }: { review: ChefReview }) {
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

export default function ChefReviewsScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data, isLoading, isError } = useChefReviews(id ?? '');
  const reviews = data?.data ?? [];

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={10}
          accessibilityRole="button"
          accessibilityLabel="Go back"
        >
          <ChevronLeft size={26} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.headerTitle}>Reviews</Text>
        <View style={{ width: 26 }} />
      </View>

      {isLoading ? (
        <View style={styles.centered}>
          <ActivityIndicator color={customerColors.coral.DEFAULT} />
        </View>
      ) : isError ? (
        <View style={styles.centered}>
          <Text style={styles.emptyText}>Couldn’t load reviews. Try again.</Text>
        </View>
      ) : reviews.length === 0 ? (
        <View style={styles.centered}>
          <Text style={styles.emptyText}>No reviews yet.</Text>
        </View>
      ) : (
        <FlatList
          data={reviews}
          keyExtractor={(r) => r.id}
          renderItem={({ item }) => <ReviewRow review={item} />}
          contentContainerStyle={styles.listContent}
          showsVerticalScrollIndicator={false}
        />
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.surface.soft },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  headerTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 20,
    color: customerColors.charcoal.DEFAULT,
  },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 32 },
  emptyText: {
    fontFamily: 'Inter',
    fontSize: 15,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
  },
  listContent: { padding: 16, gap: 12 },
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
