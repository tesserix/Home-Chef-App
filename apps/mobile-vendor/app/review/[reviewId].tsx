import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import { z } from 'zod';
import { api } from '../../lib/api';
import { useReviews } from '../reviews';

const replySchema = z.string().min(10, 'Reply must be at least 10 characters');

function useReplyToReview(reviewId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (reply: string) =>
      api.post(`/chef/reviews/${reviewId}/reply`, { reply }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef', 'reviews'] });
    },
  });
}

export default function ReviewDetailScreen() {
  const { reviewId } = useLocalSearchParams<{ reviewId: string }>();
  const { data: reviewsData } = useReviews();
  const replyMutation = useReplyToReview(reviewId ?? '');
  const [replyText, setReplyText] = useState('');
  const [validationError, setValidationError] = useState('');

  const review = reviewsData?.reviews?.find((r) => r.id === reviewId);

  function handleSendReply() {
    const result = replySchema.safeParse(replyText.trim());
    if (!result.success) {
      setValidationError(result.error.issues[0]?.message ?? 'Reply is too short');
      return;
    }
    setValidationError('');

    replyMutation.mutate(replyText.trim(), {
      onSuccess: () => {
        Alert.alert('Reply Sent', 'Your reply has been posted.', [
          { text: 'OK', onPress: () => router.back() },
        ]);
      },
      onError: () => {
        Alert.alert('Error', 'Failed to send reply. Please try again.');
      },
    });
  }

  return (
    <SafeAreaView className="flex-1 bg-paper">
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        className="flex-1"
      >
        {/* Header */}
        <View className="flex-row items-center px-4 pt-2 pb-3 bg-bone border-b border-mist">
          <TouchableOpacity accessibilityLabel="Go back" accessibilityRole="button" onPress={() => router.back()} activeOpacity={0.7} className="mr-3">
            <ChevronLeft size={24} color="#4a4a47" />
          </TouchableOpacity>
          <Text className="text-lg font-semibold text-ink">Reply to Review</Text>
        </View>

        <ScrollView
          className="flex-1"
          contentContainerStyle={{ padding: 16, paddingBottom: 40 }}
          keyboardShouldPersistTaps="handled"
        >
          {/* Original review */}
          {review ? (
            <View className="bg-bone rounded-2xl shadow-sm p-4 mb-4">
              <View className="flex-row items-start justify-between mb-2">
                <Text className="text-base font-semibold text-ink">
                  {review.customerName}
                </Text>
                <Text className="text-xs text-ink-muted">
                  {new Date(review.createdAt).toLocaleDateString('en-IN', {
                    day: 'numeric',
                    month: 'short',
                    year: 'numeric',
                  })}
                </Text>
              </View>

              {/* Star rating */}
              <View className="flex-row mb-2">
                {[1, 2, 3, 4, 5].map((star) => (
                  <Text
                    key={star}
                    className={`text-base ${star <= review.rating ? 'text-amber' : 'text-mist-strong'}`}
                  >
                    ★
                  </Text>
                ))}
              </View>

              <Text className="text-sm text-ink-soft">{review.comment}</Text>
            </View>
          ) : (
            <View className="bg-bone rounded-2xl shadow-sm p-4 mb-4 items-center">
              <ActivityIndicator size="small" color="#C2410C" />
            </View>
          )}

          {/* Reply form */}
          <View className="bg-bone rounded-2xl shadow-sm p-4 mb-4">
            <Text className="text-sm font-semibold text-ink-soft mb-3">Your Reply</Text>
            <TextInput
              value={replyText}
              onChangeText={(text) => {
                setReplyText(text);
                if (validationError) setValidationError('');
              }}
              placeholder="Write a thoughtful reply to this review..."
              placeholderTextColor="#7a7a76"
              multiline
              numberOfLines={5}
              textAlignVertical="top"
              className={`border rounded-xl px-4 py-3 text-base text-ink min-h-[120px] ${
                validationError ? 'border-paprika' : 'border-mist'
              }`}
            />
            {validationError ? (
              <Text className="text-paprika text-xs mt-1">{validationError}</Text>
            ) : null}
          </View>

          {/* Send button */}
          <TouchableOpacity
            onPress={handleSendReply}
            disabled={replyMutation.isPending}
            className={`py-4 rounded-2xl items-center ${
              replyMutation.isPending ? 'bg-herb-soft' : 'bg-herb'
            }`}
            activeOpacity={0.85}
          >
            {replyMutation.isPending ? (
              <ActivityIndicator color="white" />
            ) : (
              <Text className="text-paper font-semibold text-base">Send Reply</Text>
            )}
          </TouchableOpacity>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
