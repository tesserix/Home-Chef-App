// Social feed screen — endpoint confirmed from apps/api/handlers/social.go
// GET  /v1/social/feed          → paginated PostResponse list
// POST /v1/social/posts/:id/like → toggle like

import React, { useState, useCallback } from 'react';
import {
  ActivityIndicator,
  FlatList,
  RefreshControl,
  SafeAreaView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { Image } from 'expo-image';
import { Heart } from 'lucide-react-native';
import { useSocialFeed, useLikePost } from '../hooks/useSocial';
import type { SocialPost } from '../hooks/useSocial';

const PAGE_LIMIT = 20;

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
    <View style={styles.postCard}>
      {/* Author row */}
      <View style={styles.authorRow}>
        <View style={styles.authorAvatar}>
          <Text style={styles.authorInitial}>
            {post.chefName.charAt(0).toUpperCase()}
          </Text>
        </View>
        <View style={styles.authorInfo}>
          <Text style={styles.authorName}>{post.chefName}</Text>
          <Text style={styles.postDate}>
            {new Date(post.createdAt).toLocaleDateString('en-IN', {
              day: 'numeric',
              month: 'short',
            })}
          </Text>
        </View>
      </View>

      {/* Content */}
      <Text style={styles.postContent}>{post.content}</Text>

      {/* Images */}
      {post.images && post.images.length > 0 && (
        <Image
          source={{ uri: post.images[0] }}
          style={styles.postImage}
          contentFit="cover"
          transition={200}
        />
      )}

      {/* Hashtags */}
      {post.hashtags && post.hashtags.length > 0 && (
        <Text style={styles.hashtags}>
          {post.hashtags.map((t) => `#${t}`).join(' ')}
        </Text>
      )}

      {/* Actions */}
      <View style={styles.actionsRow}>
        <TouchableOpacity
          onPress={handleLike}
          style={styles.likeButton}
          activeOpacity={0.7}
        >
          <Heart
            size={18}
            color={optimisticLiked ? '#c95b3e' : '#7a7a76'}
            fill={optimisticLiked ? '#c95b3e' : 'transparent'}
          />
          <Text
            style={[
              styles.likeCount,
              optimisticLiked && styles.likeCountActive,
            ]}
          >
            {optimisticCount}
          </Text>
        </TouchableOpacity>
      </View>
    </View>
  );
}

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

  if (isLoading && page === 1) {
    return (
      <SafeAreaView style={styles.container}>
        <View style={styles.centered}>
          <ActivityIndicator size="large" color="#C2410C" />
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.title}>Social Feed</Text>
      </View>

      <FlatList
        data={allPosts}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => <PostCard post={item} />}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={handleRefresh}
            tintColor="#C2410C"
          />
        }
        onEndReached={handleLoadMore}
        onEndReachedThreshold={0.3}
        ListFooterComponent={
          isLoadingMore ? (
            <ActivityIndicator
              size="small"
              color="#C2410C"
              style={styles.footerLoader}
            />
          ) : null
        }
        ListEmptyComponent={
          <View style={styles.emptyState}>
            <Text style={styles.emptyIcon}>📸</Text>
            <Text style={styles.emptyTitle}>No posts yet</Text>
            <Text style={styles.emptySubtitle}>
              Chefs will share their latest creations here.
            </Text>
          </View>
        }
        contentContainerStyle={
          allPosts.length === 0 ? styles.emptyContent : styles.listContent
        }
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fafaf7',
  },
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  header: {
    paddingHorizontal: 16,
    paddingTop: 8,
    paddingBottom: 4,
  },
  title: {
    fontSize: 24,
    fontWeight: '700',
    color: '#1a1a18',
  },
  listContent: {
    paddingBottom: 24,
  },
  emptyContent: {
    flex: 1,
  },
  emptyState: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingTop: 80,
  },
  emptyIcon: {
    fontSize: 48,
    marginBottom: 16,
  },
  emptyTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#4a4a47',
    marginBottom: 8,
  },
  emptySubtitle: {
    fontSize: 14,
    color: '#7a7a76',
    textAlign: 'center',
    paddingHorizontal: 40,
  },
  footerLoader: {
    paddingVertical: 16,
  },
  // Post card
  postCard: {
    backgroundColor: '#fafaf7',
    marginHorizontal: 16,
    marginVertical: 6,
    borderRadius: 12,
    padding: 16,
    shadowColor: '#1a1a18',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.06,
    shadowRadius: 4,
    elevation: 2,
  },
  authorRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 12,
    gap: 10,
  },
  authorAvatar: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: '#C2410C',
    alignItems: 'center',
    justifyContent: 'center',
  },
  authorInitial: {
    fontSize: 16,
    fontWeight: '700',
    color: '#fafaf7',
  },
  authorInfo: {
    flex: 1,
  },
  authorName: {
    fontSize: 14,
    fontWeight: '600',
    color: '#1a1a18',
  },
  postDate: {
    fontSize: 12,
    color: '#7a7a76',
  },
  postContent: {
    fontSize: 14,
    color: '#4a4a47',
    lineHeight: 22,
    marginBottom: 12,
  },
  postImage: {
    width: '100%',
    height: 220,
    borderRadius: 8,
    marginBottom: 10,
  },
  hashtags: {
    fontSize: 13,
    color: '#C2410C',
    marginBottom: 10,
  },
  actionsRow: {
    flexDirection: 'row',
    alignItems: 'center',
    borderTopWidth: 1,
    borderTopColor: '#e6e5e0',
    paddingTop: 10,
    marginTop: 4,
  },
  likeButton: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingVertical: 4,
    paddingHorizontal: 8,
  },
  likeCount: {
    fontSize: 13,
    color: '#7a7a76',
    fontWeight: '500',
  },
  likeCountActive: {
    color: '#c95b3e',
  },
});
