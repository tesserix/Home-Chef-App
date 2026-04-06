import {
  ActivityIndicator,
  FlatList,
  RefreshControl,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useQuery } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import { api } from '../lib/api';

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

export function useReviews() {
  return useQuery<ReviewsResponse>({
    queryKey: ['chef', 'reviews'],
    queryFn: () => api.get<ReviewsResponse>('/chef/reviews').then((r) => r.data),
    staleTime: 60_000,
  });
}

function StarRating({ rating }: { rating: number }) {
  return (
    <View className="flex-row">
      {[1, 2, 3, 4, 5].map((star) => (
        <Text key={star} className={`text-base ${star <= rating ? 'text-amber-400' : 'text-gray-200'}`}>
          ★
        </Text>
      ))}
    </View>
  );
}

function ReviewCard({ review }: { review: Review }) {
  return (
    <View className="bg-white rounded-2xl shadow-sm p-4 mb-3">
      <View className="flex-row items-start justify-between mb-2">
        <View className="flex-1">
          <Text className="text-base font-semibold text-gray-900">{review.customerName}</Text>
          <StarRating rating={review.rating} />
        </View>
        <Text className="text-xs text-gray-400">
          {new Date(review.createdAt).toLocaleDateString('en-IN', {
            day: 'numeric',
            month: 'short',
            year: 'numeric',
          })}
        </Text>
      </View>

      <Text className="text-sm text-gray-700 mb-3">{review.comment}</Text>

      {review.reply ? (
        <View className="bg-orange-50 border-l-4 border-orange-400 pl-3 py-2 rounded-r-xl">
          <Text className="text-xs font-semibold text-orange-600 mb-0.5">Your Reply</Text>
          <Text className="text-sm text-gray-700">{review.reply}</Text>
        </View>
      ) : (
        <TouchableOpacity
          onPress={() => router.push(`/review/${review.id}` as never)}
          className="self-start bg-orange-50 border border-orange-200 px-4 py-2 rounded-xl"
          activeOpacity={0.7}
        >
          <Text className="text-sm text-orange-600 font-medium">Reply</Text>
        </TouchableOpacity>
      )}
    </View>
  );
}

export default function ReviewsScreen() {
  const { data, isLoading, isError, refetch, isRefetching } = useReviews();

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50 items-center justify-center">
        <ActivityIndicator size="large" color="#FF6B35" />
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50 items-center justify-center px-6">
        <Text className="text-gray-500 text-base mb-4">Failed to load reviews</Text>
        <TouchableOpacity
          onPress={() => refetch()}
          className="bg-orange-500 px-6 py-3 rounded-xl"
        >
          <Text className="text-white font-semibold">Retry</Text>
        </TouchableOpacity>
      </SafeAreaView>
    );
  }

  const avgRating = data?.averageRating ?? 0;
  const fullStars = Math.floor(avgRating);
  const halfStar = avgRating - fullStars >= 0.5;

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      {/* Header */}
      <View className="flex-row items-center px-4 pt-2 pb-3 bg-white border-b border-gray-100">
        <TouchableOpacity onPress={() => router.back()} activeOpacity={0.7} className="mr-3">
          <ChevronLeft size={24} color="#374151" />
        </TouchableOpacity>
        <Text className="text-lg font-semibold text-gray-900">Customer Reviews</Text>
      </View>

      {/* Rating header */}
      <View className="bg-white px-4 py-4 border-b border-gray-100">
        <View className="flex-row items-center gap-3">
          <Text className="text-4xl font-bold text-gray-900">{avgRating.toFixed(1)}</Text>
          <View>
            <View className="flex-row">
              {[1, 2, 3, 4, 5].map((star) => (
                <Text
                  key={star}
                  className={`text-xl ${
                    star <= fullStars
                      ? 'text-amber-400'
                      : star === fullStars + 1 && halfStar
                        ? 'text-amber-300'
                        : 'text-gray-200'
                  }`}
                >
                  ★
                </Text>
              ))}
            </View>
            <Text className="text-sm text-gray-400 mt-0.5">
              {data?.totalReviews ?? 0} reviews
            </Text>
          </View>
        </View>
      </View>

      <FlatList
        data={data?.reviews ?? []}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => <ReviewCard review={item} />}
        contentContainerStyle={{ padding: 16, paddingBottom: 32 }}
        refreshControl={
          <RefreshControl refreshing={isRefetching} onRefresh={refetch} tintColor="#FF6B35" />
        }
        ListEmptyComponent={
          <View className="items-center justify-center py-16">
            <Text className="text-gray-400 text-base text-center">
              No reviews yet.{'\n'}Your first review will appear here.
            </Text>
          </View>
        }
        showsVerticalScrollIndicator={false}
      />
    </SafeAreaView>
  );
}
