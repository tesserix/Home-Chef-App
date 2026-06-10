import { useRef, useState } from 'react';
import {
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import { z } from 'zod';
import { theme } from '@homechef/mobile-shared/theme';
import { useToast } from '@homechef/mobile-shared/ui';
import { api } from '../../lib/api';
import { useReviews } from '../reviews';

// ----- Validation -----------------------------------------------------------

const replySchema = z.string().min(10, 'Reply must be at least 10 characters');

// ----- Hook -----------------------------------------------------------------

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

// ----- Helpers --------------------------------------------------------------

function relativeDate(iso: string): string {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return '';
  const days = Math.floor((Date.now() - t) / 86_400_000);
  if (days < 1) return 'Today';
  if (days < 7) return `${days}d ago`;
  return new Date(iso).toLocaleDateString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

// ----- Screen ---------------------------------------------------------------

export default function ReviewDetailScreen() {
  const { reviewId } = useLocalSearchParams<{ reviewId: string }>();
  const { data: reviewsData } = useReviews();
  const replyMutation = useReplyToReview(reviewId ?? '');
  const { show: showToast } = useToast();

  const [replyText, setReplyText] = useState('');
  const [inputFocused, setInputFocused] = useState(false);
  const [validationError, setValidationError] = useState('');
  // Input expands as content grows, but we cap it at 180 before scrolling
  const [inputHeight, setInputHeight] = useState(120);

  const inputRef = useRef<TextInput>(null);

  const review = reviewsData?.reviews?.find((r) => r.id === reviewId);
  const alreadyReplied = Boolean(review?.reply);

  function handleChangeText(text: string) {
    setReplyText(text);
    if (validationError) setValidationError('');
  }

  function handleSend() {
    const result = replySchema.safeParse(replyText.trim());
    if (!result.success) {
      setValidationError(result.error.issues[0]?.message ?? 'Reply is too short');
      return;
    }
    setValidationError('');

    replyMutation.mutate(replyText.trim(), {
      onSuccess: () => {
        showToast({ message: 'Reply sent', tone: 'success' });
        router.back();
      },
      onError: () => {
        showToast({
          message: 'Failed to send reply. Check your connection and try again.',
          tone: 'error',
        });
      },
    });
  }

  const sendDisabled = replyMutation.isPending || Boolean(validationError);

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <KeyboardAvoidingView
        style={styles.flex}
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        keyboardVerticalOffset={0}
      >
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
          <Text style={styles.commandTitle}>
            {alreadyReplied ? 'Your reply' : 'Reply to review'}
          </Text>
        </View>

        <ScrollView
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          {/* Original review — hairline-bordered section, not a card */}
          <View style={styles.reviewSection}>
            {review ? (
              <>
                {/* Header row: name left, date right */}
                <View style={styles.reviewHeader}>
                  <Text style={styles.customerName} numberOfLines={1}>
                    {review.customerName}
                  </Text>
                  <Text style={styles.reviewDate}>
                    {relativeDate(review.createdAt)}
                  </Text>
                </View>

                {/* Star display — compact "★ N" from reviews.tsx */}
                <View style={styles.starRow}>
                  <Text style={styles.starGlyph}>★</Text>
                  <Text style={styles.starCount}>{review.rating}</Text>
                  <Text style={styles.starOutOf}>/5</Text>
                </View>

                {/* Comment body */}
                <Text style={styles.reviewComment}>{review.comment}</Text>
              </>
            ) : (
              <ActivityIndicator
                size="small"
                color={theme.colors.ink.DEFAULT}
                style={styles.reviewLoader}
              />
            )}
          </View>

          {/* Section divider label */}
          <Text style={styles.sectionLabel}>
            {alreadyReplied ? 'YOUR REPLY' : 'YOUR REPLY'}
          </Text>

          {alreadyReplied ? (
            /* Read-only replied state — muted hairline block */
            <View style={styles.repliedBlock}>
              <Text style={styles.repliedText}>{review?.reply}</Text>
            </View>
          ) : (
            /* Reply input — borderless TextInput inside hairline container */
            <>
              <View
                style={[
                  styles.inputContainer,
                  inputFocused && styles.inputContainerFocused,
                  validationError && styles.inputContainerError,
                ]}
              >
                <TextInput
                  ref={inputRef}
                  value={replyText}
                  onChangeText={handleChangeText}
                  onFocus={() => setInputFocused(true)}
                  onBlur={() => setInputFocused(false)}
                  placeholder="Write a thoughtful reply…"
                  placeholderTextColor={theme.colors.ink.muted}
                  multiline
                  scrollEnabled={inputHeight >= 180}
                  textAlignVertical="top"
                  style={[styles.input, { height: Math.min(Math.max(inputHeight, 120), 180) }]}
                  onContentSizeChange={(e) =>
                    setInputHeight(e.nativeEvent.contentSize.height + 24)
                  }
                  accessibilityLabel="Reply text"
                  accessibilityHint="Write your response to this customer review"
                />
              </View>

              {/* Validation error stays inline for immediate typing feedback.
                  Network failures surface as a toast (see handleSend). */}
              {validationError ? (
                <Text style={styles.errorText}>{validationError}</Text>
              ) : null}
            </>
          )}
        </ScrollView>

        {/* Sticky Send footer — hidden when review is already replied */}
        {!alreadyReplied && (
          <SafeAreaView edges={['bottom']} style={styles.stickyFooter}>
            <Pressable
              onPress={handleSend}
              disabled={sendDisabled}
              accessibilityRole="button"
              accessibilityLabel="Send reply"
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.sendBtn,
                    sendDisabled && styles.sendBtnDisabled,
                    pressed && !sendDisabled && { opacity: 0.88 },
                  ]}
                >
                  {replyMutation.isPending ? (
                    <ActivityIndicator color={theme.colors.paper} />
                  ) : (
                    <Text style={styles.sendBtnLabel}>Send reply</Text>
                  )}
                </View>
              )}
            </Pressable>
          </SafeAreaView>
        )}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

// ----- Styles ---------------------------------------------------------------

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: theme.colors.paper,
  },
  flex: {
    flex: 1,
  },

  // Zone A — Command bar (matches reviews.tsx exactly)
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
    fontSize: 22,
    lineHeight: 26,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },

  // Scroll
  scroll: { flex: 1 },
  scrollContent: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[2],
    paddingBottom: 120, // room for sticky footer
  },

  // Original review — hairline-bordered section
  reviewSection: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[4],
    marginBottom: theme.spacing[6],
  },
  reviewLoader: {
    marginVertical: theme.spacing[4],
  },
  reviewHeader: {
    flexDirection: 'row',
    alignItems: 'baseline',
    justifyContent: 'space-between',
    gap: theme.spacing[2],
    marginBottom: theme.spacing[2],
  },
  customerName: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  reviewDate: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    flexShrink: 0,
  },

  // Compact "★ N /5" — matches reviews.tsx starBlock language
  starRow: {
    flexDirection: 'row',
    alignItems: 'baseline',
    gap: 3,
    marginBottom: theme.spacing[3],
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
  starOutOf: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    lineHeight: 16,
  },
  reviewComment: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    lineHeight: 21,
  },

  // Section label — matches profile.tsx sectionLabel
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[2],
  },

  // Reply input container — hairline border, persimmon focus ring
  inputContainer: {
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
  },
  inputContainerFocused: {
    borderColor: theme.colors.ink.DEFAULT, // persimmon focus ring — only usage on screen
  },
  inputContainerError: {
    borderColor: theme.colors.destructive.DEFAULT,
  },
  input: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    lineHeight: 22,
    // No border — container provides containment
  },

  // Inline error — validation + network, destructive color
  errorText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.destructive.DEFAULT,
    marginTop: theme.spacing[2],
    lineHeight: 16,
  },

  // Already-replied read-only block
  repliedBlock: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[4],
    backgroundColor: theme.colors.bone,
  },
  repliedText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    lineHeight: 21,
  },

  // Sticky Send footer — ink fill, matches profile.tsx saveBtn
  stickyFooter: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    backgroundColor: theme.colors.paper,
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
  },
  sendBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingVertical: theme.spacing[4],
    alignItems: 'center',
    minHeight: 52,
    justifyContent: 'center',
  },
  sendBtnDisabled: {
    backgroundColor: theme.colors.mist.strong,
  },
  sendBtnLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
    letterSpacing: 0.2,
  },
});
