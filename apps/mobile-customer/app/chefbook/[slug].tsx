import React, { useState } from 'react';
import {
  ActivityIndicator,
  Image,
  Pressable,
  ScrollView,
  Share,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { Clock, Share2, Trash2 } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useToast } from '@homechef/mobile-shared/ui';
import { ScreenHeader } from '../../components/ScreenHeader';
import { useAuthStore } from '../../store/auth-store';
import { useProfile } from '../../hooks/useProfile';
import {
  useArticle,
  useReact,
  useAddComment,
  useDeleteComment,
  articleUrl,
  REACTIONS,
  type ArticleBlock,
  type ReactionType,
} from '../../hooks/useChefBook';

// The article reader. Blocks are rendered natively from the API's typed
// payload — the same payload the web reader renders — so the two stay in step
// without sharing a rendering layer.

const RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

function Block({ block }: { block: ArticleBlock }) {
  switch (block.type) {
    case 'heading':
      return <Text style={styles.heading}>{block.text}</Text>;
    case 'quote':
      return (
        <View style={styles.quoteWrap}>
          <Text style={styles.quote}>{block.text}</Text>
        </View>
      );
    case 'image':
      return (
        <View style={styles.figure}>
          <Image source={{ uri: block.url }} style={styles.blockImage} resizeMode="cover" />
          {block.caption ? <Text style={styles.caption}>{block.caption}</Text> : null}
        </View>
      );
    case 'list':
      return (
        <View style={styles.list}>
          {(block.items ?? []).map((item, i) => (
            <View key={i} style={styles.listRow}>
              <Text style={styles.bullet}>•</Text>
              <Text style={styles.listText}>{item}</Text>
            </View>
          ))}
        </View>
      );
    default:
      return <Text style={styles.paragraph}>{block.text}</Text>;
  }
}

export default function ArticleScreen() {
  const { slug } = useLocalSearchParams<{ slug: string }>();
  const router = useRouter();
  const { show: showToast } = useToast();
  const { data, isLoading, isError } = useArticle(slug);
  const react = useReact(slug);
  const addComment = useAddComment(slug);
  const removeComment = useDeleteComment(slug);
  const [comment, setComment] = useState('');

  const isAuthenticated = useAuthStore((s) => !!s.accessToken);
  // The auth store only rehydrates the token after a cold start, never the user
  // object, so "is this my comment" must come from the profile query instead.
  const { data: profile } = useProfile({ enabled: isAuthenticated });
  const currentUserId = profile?.userId;
  const article = data?.data;

  if (isLoading) {
    return (
      <SafeAreaView style={styles.screen} edges={['top']}>
        <ScreenHeader title="" />
        <View style={styles.centre}>
          <ActivityIndicator color={customerColors.coral.DEFAULT} />
        </View>
      </SafeAreaView>
    );
  }

  if (isError || !article) {
    return (
      <SafeAreaView style={styles.screen} edges={['top']}>
        <ScreenHeader title="ChefBook" />
        <View style={styles.centre}>
          <Text style={styles.muted}>That article isn&apos;t available.</Text>
          <Pressable onPress={() => router.back()} style={styles.retry} android_ripple={{ color: RIPPLE }}>
            <Text style={styles.retryText}>Go back</Text>
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  const onReact = (type: ReactionType) => {
    if (!isAuthenticated) {
      showToast({ message: 'Sign in to react', tone: 'error' });
      return;
    }
    // Tapping the active reaction clears it — the API reads '' as "remove".
    react.mutate({ id: article.id, reaction: article.viewerReaction === type ? '' : type });
  };

  const onShare = async () => {
    // The OS sheet is the only route to Instagram and the like from a phone,
    // so there's no in-app list of networks to maintain here.
    try {
      await Share.share({
        message: `${article.title} — ${articleUrl(article.slug)}`,
        url: articleUrl(article.slug),
        title: article.title,
      });
    } catch {
      // Dismissing the sheet isn't an error worth reporting back.
    }
  };

  const submitComment = () => {
    const body = comment.trim();
    if (!body) return;
    addComment.mutate(
      { id: article.id, body },
      {
        onSuccess: () => {
          setComment('');
          showToast({ message: 'Comment posted', tone: 'success' });
        },
        onError: () => showToast({ message: 'Could not post your comment', tone: 'error' }),
      },
    );
  };

  return (
    <SafeAreaView style={styles.screen} edges={['top']}>
      <ScreenHeader
        title="ChefBook"
        right={
          <Pressable onPress={onShare} accessibilityRole="button" accessibilityLabel="Share article" hitSlop={8}>
            <Share2 size={20} color={customerColors.charcoal.DEFAULT} />
          </Pressable>
        }
      />

      <ScrollView contentContainerStyle={styles.body} keyboardShouldPersistTaps="handled">
        <Text style={styles.title}>{article.title}</Text>

        <View style={styles.byline}>
          {article.chefImage ? (
            <Image source={{ uri: article.chefImage }} style={styles.avatar} />
          ) : (
            <View style={[styles.avatar, styles.avatarFallback]} />
          )}
          <Text style={styles.chefName}>{article.chefName}</Text>
          <Clock size={13} color={customerColors.charcoal.soft} />
          <Text style={styles.metaText}>{article.readingMinutes} min read</Text>
        </View>

        {article.cover ? (
          <Image source={{ uri: article.cover }} style={styles.cover} resizeMode="cover" />
        ) : null}

        {(article.blocks ?? []).map((b, i) => (
          <Block key={i} block={b} />
        ))}

        {article.tags && article.tags.length > 0 ? (
          <View style={styles.tags}>
            {article.tags.map((t) => (
              <View key={t} style={styles.tag}>
                <Text style={styles.tagText}>#{t}</Text>
              </View>
            ))}
          </View>
        ) : null}

        {/* Reactions */}
        <View style={styles.reactions}>
          {REACTIONS.map((r) => {
            const active = article.viewerReaction === r.type;
            const count = article.reactionCounts?.[r.type] ?? 0;
            return (
              <Pressable
                key={r.type}
                onPress={() => onReact(r.type)}
                disabled={react.isPending}
                accessibilityRole="button"
                accessibilityState={{ selected: active }}
                accessibilityLabel={`${r.label}${count > 0 ? `, ${count}` : ''}`}
                android_ripple={{ color: RIPPLE }}
                style={[styles.reaction, active && styles.reactionActive]}
              >
                <Text style={styles.reactionEmoji}>{r.emoji}</Text>
                <Text style={[styles.reactionLabel, active && styles.reactionLabelActive]}>
                  {r.label}
                  {count > 0 ? ` ${count}` : ''}
                </Text>
              </Pressable>
            );
          })}
        </View>

        {/* Comments */}
        <View style={styles.commentsSection}>
          <Text style={styles.commentsTitle}>
            Comments{article.commentsCount > 0 ? ` (${article.commentsCount})` : ''}
          </Text>

          {isAuthenticated ? (
            <View style={styles.composer}>
              <TextInput
                value={comment}
                onChangeText={setComment}
                placeholder="Ask the chef something, or say how yours turned out."
                placeholderTextColor={customerColors.charcoal.soft}
                multiline
                maxLength={1000}
                style={styles.input}
              />
              <Pressable
                onPress={submitComment}
                disabled={!comment.trim() || addComment.isPending}
                accessibilityRole="button"
                android_ripple={{ color: RIPPLE }}
                style={[styles.postBtn, (!comment.trim() || addComment.isPending) && styles.postBtnDisabled]}
              >
                <Text style={styles.postBtnText}>Post</Text>
              </Pressable>
            </View>
          ) : (
            <Text style={styles.muted}>Sign in to join the conversation.</Text>
          )}

          {(article.comments ?? []).map((c) => (
            <View key={c.id} style={styles.comment}>
              {c.userAvatar ? (
                <Image source={{ uri: c.userAvatar }} style={styles.commentAvatar} />
              ) : (
                <View style={[styles.commentAvatar, styles.avatarFallback]} />
              )}
              <View style={styles.commentBody}>
                <Text style={styles.commentName}>{c.userName || 'Someone'}</Text>
                <Text style={styles.commentText}>{c.body}</Text>
              </View>
              {currentUserId === c.userId ? (
                <Pressable
                  onPress={() => removeComment.mutate({ id: article.id, commentId: c.id })}
                  accessibilityRole="button"
                  accessibilityLabel="Delete your comment"
                  hitSlop={8}
                >
                  <Trash2 size={16} color={customerColors.charcoal.soft} />
                </Pressable>
              ) : null}
            </View>
          ))}

          {(article.comments ?? []).length === 0 ? (
            <Text style={styles.muted}>No comments yet.</Text>
          ) : null}
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: customerColors.canvas },
  centre: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 12, padding: 24 },
  muted: { color: customerColors.charcoal.soft, fontSize: 14 },
  retry: {
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: customerColors.hairline,
  },
  retryText: { color: customerColors.charcoal.DEFAULT, fontSize: 14, fontWeight: '600' },

  body: { padding: 16, paddingBottom: 40 },
  title: { color: customerColors.charcoal.DEFAULT, fontSize: 24, fontWeight: '700', lineHeight: 31 },
  byline: { flexDirection: 'row', alignItems: 'center', gap: 8, marginTop: 10, flexWrap: 'wrap' },
  avatar: { width: 28, height: 28, borderRadius: 14 },
  avatarFallback: { backgroundColor: customerColors.hairline },
  chefName: { color: customerColors.charcoal.DEFAULT, fontSize: 14, fontWeight: '600' },
  metaText: { color: customerColors.charcoal.soft, fontSize: 13 },
  cover: { width: '100%', aspectRatio: 16 / 9, borderRadius: 12, marginTop: 14 },

  heading: { color: customerColors.charcoal.DEFAULT, fontSize: 18, fontWeight: '700', marginTop: 22 },
  paragraph: { color: customerColors.charcoal.DEFAULT, fontSize: 16, lineHeight: 25, marginTop: 14 },
  quoteWrap: {
    borderLeftWidth: 2,
    borderLeftColor: customerColors.coral.DEFAULT,
    paddingLeft: 12,
    marginTop: 18,
  },
  quote: { color: customerColors.charcoal.soft, fontSize: 16, fontStyle: 'italic', lineHeight: 24 },
  figure: { marginTop: 18 },
  blockImage: { width: '100%', aspectRatio: 4 / 3, borderRadius: 10 },
  caption: { color: customerColors.charcoal.soft, fontSize: 13, textAlign: 'center', marginTop: 6 },
  list: { marginTop: 14, gap: 6 },
  listRow: { flexDirection: 'row', gap: 8 },
  bullet: { color: customerColors.charcoal.soft, fontSize: 16, lineHeight: 24 },
  listText: { color: customerColors.charcoal.DEFAULT, fontSize: 16, lineHeight: 24, flex: 1 },

  tags: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, marginTop: 22 },
  tag: {
    borderWidth: 1,
    borderColor: customerColors.hairline,
    borderRadius: 999,
    paddingHorizontal: 12,
    paddingVertical: 5,
  },
  tagText: { color: customerColors.charcoal.soft, fontSize: 13 },

  reactions: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
    marginTop: 24,
    paddingTop: 18,
    borderTopWidth: 1,
    borderTopColor: customerColors.hairline,
  },
  reaction: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    borderRadius: 999,
    paddingHorizontal: 12,
    paddingVertical: 8,
    minHeight: 44,
  },
  reactionActive: { borderColor: customerColors.coral.DEFAULT, backgroundColor: customerColors.coral.tint },
  reactionEmoji: { fontSize: 15 },
  reactionLabel: { color: customerColors.charcoal.soft, fontSize: 13 },
  reactionLabelActive: { color: customerColors.coral.pressed, fontWeight: '600' },

  commentsSection: {
    marginTop: 24,
    paddingTop: 18,
    borderTopWidth: 1,
    borderTopColor: customerColors.hairline,
    gap: 12,
  },
  commentsTitle: { color: customerColors.charcoal.DEFAULT, fontSize: 16, fontWeight: '600' },
  composer: { gap: 8 },
  input: {
    borderWidth: 1,
    borderColor: customerColors.hairline,
    borderRadius: 10,
    padding: 12,
    minHeight: 80,
    color: customerColors.charcoal.DEFAULT,
    fontSize: 15,
    textAlignVertical: 'top',
    backgroundColor: customerColors.surface.soft,
  },
  postBtn: {
    alignSelf: 'flex-end',
    backgroundColor: customerColors.charcoal.DEFAULT,
    borderRadius: 8,
    paddingHorizontal: 18,
    paddingVertical: 11,
    minHeight: 44,
    justifyContent: 'center',
  },
  postBtnDisabled: { opacity: 0.5 },
  postBtnText: { color: customerColors.canvas, fontSize: 14, fontWeight: '600' },

  comment: { flexDirection: 'row', gap: 10, alignItems: 'flex-start' },
  commentAvatar: { width: 32, height: 32, borderRadius: 16 },
  commentBody: { flex: 1, gap: 2 },
  commentName: { color: customerColors.charcoal.DEFAULT, fontSize: 14, fontWeight: '600' },
  commentText: { color: customerColors.charcoal.soft, fontSize: 14, lineHeight: 20 },
});
