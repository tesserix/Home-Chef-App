import React from 'react';
import { ActivityIndicator, FlatList, Image, Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { BookOpen, Clock, MessageCircle } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ScreenHeader } from '../components/ScreenHeader';
import { useChefBookFeed, REACTIONS, type Article } from '../hooks/useChefBook';

// ChefBook — chef-written culinary articles. Reads as a magazine rather than a
// social timeline: cover, title, excerpt, byline.

const RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

function ArticleCard({ article, onPress }: { article: Article; onPress: () => void }) {
  // Only summarise reactions the article actually received, so an unloved
  // article doesn't render a row of zeros.
  const present = REACTIONS.filter((r) => (article.reactionCounts?.[r.type] ?? 0) > 0);

  return (
    <Pressable
      onPress={onPress}
      android_ripple={{ color: RIPPLE }}
      accessibilityRole="button"
      accessibilityLabel={`${article.title} by ${article.chefName}, ${article.readingMinutes} minute read`}
      style={styles.card}
    >
      {article.cover ? (
        <Image source={{ uri: article.cover }} style={styles.cover} resizeMode="cover" />
      ) : null}
      <View style={styles.cardBody}>
        <View style={styles.byline}>
          {article.chefImage ? (
            <Image source={{ uri: article.chefImage }} style={styles.avatar} />
          ) : (
            <View style={[styles.avatar, styles.avatarFallback]} />
          )}
          <Text style={styles.chefName} numberOfLines={1}>
            {article.chefName}
          </Text>
        </View>

        <Text style={styles.title} numberOfLines={2}>
          {article.title}
        </Text>
        <Text style={styles.excerpt} numberOfLines={2}>
          {article.excerpt}
        </Text>

        <View style={styles.meta}>
          <Clock size={13} color={customerColors.charcoal.soft} />
          <Text style={styles.metaText}>{article.readingMinutes} min read</Text>

          {article.commentsCount > 0 ? (
            <>
              <MessageCircle size={13} color={customerColors.charcoal.soft} />
              <Text style={styles.metaText}>{article.commentsCount}</Text>
            </>
          ) : null}

          {present.length > 0 ? (
            <Text style={styles.metaText}>
              {present.map((r) => r.emoji).join('')} {article.reactionsTotal}
            </Text>
          ) : null}
        </View>
      </View>
    </Pressable>
  );
}

export default function ChefBookScreen() {
  const router = useRouter();
  const { data, isLoading, isError, refetch, isRefetching } = useChefBookFeed();
  const articles = data?.data ?? [];

  return (
    <SafeAreaView style={styles.screen} edges={['top']}>
      <ScreenHeader title="ChefBook" />

      {isLoading ? (
        <View style={styles.centre}>
          <ActivityIndicator color={customerColors.coral.DEFAULT} />
        </View>
      ) : isError ? (
        <View style={styles.centre}>
          <Text style={styles.muted}>ChefBook couldn&apos;t load just now.</Text>
          <Pressable onPress={() => refetch()} style={styles.retry} android_ripple={{ color: RIPPLE }}>
            <Text style={styles.retryText}>Try again</Text>
          </Pressable>
        </View>
      ) : (
        <FlatList
          data={articles}
          keyExtractor={(a) => a.id}
          contentContainerStyle={styles.list}
          refreshing={isRefetching}
          onRefresh={refetch}
          ListHeaderComponent={
            <Text style={styles.intro}>
              Recipes, methods and kitchen notes, written by the chefs who cook them.
            </Text>
          }
          ListEmptyComponent={
            <View style={styles.empty}>
              <BookOpen size={28} color={customerColors.charcoal.soft} />
              <Text style={styles.emptyText}>
                No articles yet. When chefs start writing, their recipes land here.
              </Text>
            </View>
          }
          renderItem={({ item }) => (
            <ArticleCard
              article={item}
              onPress={() => router.push(`/chefbook/${item.slug}` as never)}
            />
          )}
        />
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: customerColors.canvas },
  centre: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 12, padding: 24 },
  muted: { color: customerColors.charcoal.soft, fontSize: 14, textAlign: 'center' },
  retry: {
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: customerColors.hairline,
  },
  retryText: { color: customerColors.charcoal.DEFAULT, fontSize: 14, fontWeight: '600' },
  list: { padding: 16, gap: 14, paddingBottom: 32 },
  intro: { color: customerColors.charcoal.soft, fontSize: 14, marginBottom: 4 },
  card: {
    backgroundColor: customerColors.surface.DEFAULT,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    overflow: 'hidden',
  },
  cover: { width: '100%', aspectRatio: 16 / 9 },
  cardBody: { padding: 14, gap: 6 },
  byline: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  avatar: { width: 22, height: 22, borderRadius: 11 },
  avatarFallback: { backgroundColor: customerColors.hairline },
  chefName: { color: customerColors.charcoal.soft, fontSize: 13, flexShrink: 1 },
  title: { color: customerColors.charcoal.DEFAULT, fontSize: 17, fontWeight: '700' },
  excerpt: { color: customerColors.charcoal.soft, fontSize: 14, lineHeight: 20 },
  meta: { flexDirection: 'row', alignItems: 'center', gap: 6, marginTop: 2, flexWrap: 'wrap' },
  metaText: { color: customerColors.charcoal.soft, fontSize: 12, marginRight: 6 },
  empty: { alignItems: 'center', gap: 10, paddingVertical: 48 },
  emptyText: {
    color: customerColors.charcoal.soft,
    fontSize: 14,
    textAlign: 'center',
    paddingHorizontal: 24,
  },
});
