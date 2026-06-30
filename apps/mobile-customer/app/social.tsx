// Social feed screen — endpoint confirmed from apps/api/handlers/social.go
// GET  /v1/social/feed          → paginated PostResponse list
// POST /v1/social/posts/:id/like → toggle like

import React, { useState, useCallback } from 'react';
import {
  ActivityIndicator,
  FlatList,
  Pressable,
  RefreshControl,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Image } from 'expo-image';
import { Camera, Heart } from 'lucide-react-native';
import { useSocialFeed, useLikePost } from '../hooks/useSocial';
import type { SocialPost } from '../hooks/useSocial';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ScreenHeader } from '../components/ScreenHeader';

const PAGE_LIMIT = 20;

// ─── Post card ───────────────────────────────────────────────────────────────

function PostCard({ post }: { post: SocialPost }) {
  const likePost = useLikePost();
  const [optimisticLiked, setOptimisticLiked] = useState(post.isLiked);
  const [optimisticCount, setOptimisticCount] = useState(post.likesCount);

  function handleLike() {
    const wasLiked = optimisticLiked;
    setOptimisticLiked(!wasLiked);
    setOptimisticCount((prev) => (wasLiked ? prev - 1 : prev + 1));

    likePost.mutate(post.id, {
      onError: () => {
        // Revert on error
        setOptimisticLiked(wasLiked);
        setOptimisticCount(post.likesCount);
      },
    });
  }

  return (
    <View className="bg-canvas">
      {/* Author row */}
      <View className="flex-row items-center px-4 pt-4 pb-3 gap-3">
        {/* Avatar initials circle */}
        <View className="w-10 h-10 rounded-full bg-coral items-center justify-center">
          <Text className="text-base font-bold text-canvas font-display">
            {post.chefName.charAt(0).toUpperCase()}
          </Text>
        </View>
        <View className="flex-1">
          <Text className="text-sm font-semibold text-charcoal">
            {post.chefName}
          </Text>
          <Text className="text-xs text-charcoal-soft">
            {new Date(post.createdAt).toLocaleDateString('en-IN', {
              day: 'numeric',
              month: 'short',
            })}
          </Text>
        </View>
      </View>

      {/* Post image — full width, 4:3 ratio */}
      {post.images && post.images.length > 0 ? (
        <Image
          source={{ uri: post.images[0] }}
          style={{ width: '100%', aspectRatio: 4 / 3 }}
          contentFit="cover"
          transition={200}
          accessibilityLabel={`Photo by ${post.chefName}`}
        />
      ) : null}

      {/* Caption */}
      {post.content ? (
        <Text className="text-sm text-charcoal-soft leading-relaxed px-4 pt-3">
          {post.content}
        </Text>
      ) : null}

      {/* Hashtags in coral */}
      {post.hashtags && post.hashtags.length > 0 ? (
        <Text className="text-xs text-coral px-4 pt-1">
          {post.hashtags.map((t) => `#${t}`).join(' ')}
        </Text>
      ) : null}

      {/* Actions row */}
      <View className="flex-row items-center px-4 pt-3 pb-4 border-t border-hairline mt-3">
        {/* iOS Pressable pattern: visual styles on inner View */}
        <Pressable
          onPress={handleLike}
          accessibilityRole="button"
          accessibilityLabel={optimisticLiked ? 'Unlike post' : 'Like post'}
          accessibilityState={{ selected: optimisticLiked }}
        >
          <View className="flex-row items-center gap-1.5 py-1 px-2">
            <Heart
              size={18}
              color={optimisticLiked ? customerColors.coral.DEFAULT : customerColors.charcoal.soft}
              fill={optimisticLiked ? customerColors.coral.DEFAULT : 'transparent'}
            />
            <Text
              className={`text-sm font-medium ${optimisticLiked ? 'text-coral' : 'text-charcoal-soft'}`}
            >
              {optimisticCount}
            </Text>
          </View>
        </Pressable>
      </View>

      {/* Hairline separator between posts */}
      <View className="h-px bg-hairline" />
    </View>
  );
}

// ─── Empty state ─────────────────────────────────────────────────────────────

function EmptyState() {
  return (
    <View className="flex-1 items-center justify-center px-8 gap-4 pt-20">
      {/* Surface-soft icon circle */}
      <View className="w-20 h-20 rounded-full bg-surface-soft items-center justify-center">
        <Camera size={36} color={customerColors.charcoal.soft} />
      </View>
      <View className="items-center gap-2">
        <Text className="text-xl font-bold text-charcoal text-center font-display">
          No posts yet
        </Text>
        <Text className="text-sm text-charcoal-soft text-center leading-5">
          Chefs will share their latest creations here.
        </Text>
      </View>
    </View>
  );
}

// ─── Main screen ─────────────────────────────────────────────────────────────

export default function SocialScreen() {
  const [page, setPage] = useState(1);
  const [allPosts, setAllPosts] = useState<SocialPost[]>([]);
  const [isLoadingMore, setIsLoadingMore] = useState(false);

  const { data, isLoading, isRefetching, refetch } = useSocialFeed({
    page,
    limit: PAGE_LIMIT,
  });

  React.useEffect(() => {
    if (data?.data) {
      if (page === 1) {
        setAllPosts(data.data);
      } else {
        setAllPosts((prev) => {
          const existingIds = new Set(prev.map((p) => p.id));
          const newItems = data.data.filter((p) => !existingIds.has(p.id));
          return [...prev, ...newItems];
        });
        setIsLoadingMore(false);
      }
    }
  }, [data, page]);

  const handleRefresh = useCallback(() => {
    setPage(1);
    setAllPosts([]);
    void refetch();
  }, [refetch]);

  function handleLoadMore() {
    if (!data || isLoadingMore) return;
    if (allPosts.length < data.total) {
      setIsLoadingMore(true);
      setPage((prev) => prev + 1);
    }
  }

  // ── Loading ──────────────────────────────────────────────────────────────
  if (isLoading && page === 1) {
    return (
      <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color={customerColors.coral.DEFAULT} />
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>

      <ScreenHeader title="Social Feed" />

      <FlatList<SocialPost>
        data={allPosts}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => <PostCard post={item} />}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={handleRefresh}
            tintColor={customerColors.coral.DEFAULT}
          />
        }
        onEndReached={handleLoadMore}
        onEndReachedThreshold={0.3}
        ListFooterComponent={
          isLoadingMore ? (
            <View className="py-4 items-center">
              <ActivityIndicator size="small" color={customerColors.coral.DEFAULT} />
            </View>
          ) : null
        }
        ListEmptyComponent={<EmptyState />}
        contentContainerStyle={allPosts.length === 0 ? { flexGrow: 1 } : { paddingBottom: 24 }}
      />

    </SafeAreaView>
  );
}
