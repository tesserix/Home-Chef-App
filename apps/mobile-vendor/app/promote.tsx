// "Promote your kitchen" — the chef's own audience, and one tap to share their
// public page to the networks they already use.
//
// The counts sit above the share buttons on purpose: likes and subscribers move
// the kitchen up in customer search, so the number is the reason to share, not
// a vanity stat bolted on afterwards.

import { useCallback, useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useQuery } from '@tanstack/react-query';
import { router, useLocalSearchParams } from 'expo-router';
import { Bell, BookOpen, ChevronLeft, Heart, Share2 } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { useToast } from '@homechef/mobile-shared/ui';
import { api } from '../lib/api';
import { useMyArticles } from '../hooks/useChefBook';
import {
  SHARE_TARGETS,
  articleShare,
  chefPublicUrl,
  kitchenShare,
  shareToNetwork,
  type SocialNetwork,
} from '../lib/social-share';

interface PromoteProfile {
  businessName?: string;
  slug?: string;
  likeCount?: number;
  subscriberCount?: number;
  articleReactionCount?: number;
}

// What the share buttons act on. 'kitchen' is the default; anything else is a
// published article id.
type ShareSubject = 'kitchen' | string;

export default function PromoteScreen() {
  const { show: showToast } = useToast();
  const [pending, setPending] = useState<SocialNetwork | null>(null);
  // ChefBook sends the post the chef tapped Share on, so they land here with it
  // already chosen rather than picking it out of the list again.
  const { articleId } = useLocalSearchParams<{ articleId?: string }>();
  const [subject, setSubject] = useState<ShareSubject>(articleId ?? 'kitchen');

  const { data, isLoading } = useQuery<PromoteProfile>({
    // Same key the Profile screen uses, so an edit there refreshes the name and
    // slug shown here instead of leaving a stale copy behind.
    queryKey: ['chef', 'profile'],
    queryFn: () => api.get<PromoteProfile>('/chef/profile').then((r) => r.data),
  });
  const { data: articleData } = useMyArticles();

  const businessName = data?.businessName ?? '';
  const slug = data?.slug ?? '';
  const url = slug ? chefPublicUrl(slug) : '';

  // Only published posts. Sharing a draft would send readers to something that
  // isn't there yet.
  const posts = useMemo(
    () => (articleData?.data ?? []).filter((a) => a.status === 'published'),
    [articleData],
  );

  const selected = posts.find((p) => p.id === subject);

  const onShare = useCallback(
    async (network: SocialNetwork) => {
      if (!url) return;
      setPending(network);
      try {
        const content = selected
          ? articleShare(businessName, selected.title, url)
          : kitchenShare(businessName, url);
        const ok = await shareToNetwork(network, content);
        if (!ok) {
          showToast({ message: 'Could not open that app. Try another.', tone: 'error' });
        }
      } finally {
        setPending(null);
      }
    },
    [businessName, url, selected, showToast],
  );

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={12}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
        >
          <ChevronLeft size={26} color={theme.colors.ink.DEFAULT} strokeWidth={1.75} />
        </Pressable>
        <Text style={styles.title}>Promote your kitchen</Text>
      </View>

      {isLoading ? (
        <View style={styles.centered}>
          <ActivityIndicator color={theme.colors.ink.DEFAULT} />
        </View>
      ) : (
        <ScrollView contentContainerStyle={styles.content} showsVerticalScrollIndicator={false}>
          <View style={styles.statsRow}>
            <View style={styles.stat}>
              <Heart size={18} strokeWidth={2} color={theme.colors.ink.soft} />
              <Text style={styles.statValue}>{data?.likeCount ?? 0}</Text>
              <Text style={styles.statLabel}>Likes</Text>
            </View>
            <View style={styles.statDivider} />
            <View style={styles.stat}>
              <Bell size={18} strokeWidth={2} color={theme.colors.ink.soft} />
              <Text style={styles.statValue}>{data?.subscriberCount ?? 0}</Text>
              <Text style={styles.statLabel}>Subscribers</Text>
            </View>
            <View style={styles.statDivider} />
            <View style={styles.stat}>
              <BookOpen size={18} strokeWidth={2} color={theme.colors.ink.soft} />
              <Text style={styles.statValue}>{data?.articleReactionCount ?? 0}</Text>
              <Text style={styles.statLabel}>Post reactions</Text>
            </View>
          </View>

          <Text style={styles.explainer}>
            Subscribers get told when you publish a menu, drop a price, open your kitchen or
            post to ChefBook. Likes, subscribers and reactions to your posts all lift your
            kitchen in customer search.
          </Text>

          {url ? (
            <View style={styles.linkBox}>
              <Text style={styles.linkLabel}>Your public page</Text>
              <Text style={styles.linkUrl} numberOfLines={1}>
                {url}
              </Text>
            </View>
          ) : (
            // No slug means the profile has no business name yet — sharing a
            // broken link would be worse than saying so.
            <Text style={styles.explainer}>
              Finish your profile to get a shareable page for your kitchen.
            </Text>
          )}

          {url && posts.length > 0 ? (
            <View style={styles.subjects}>
              <Text style={styles.sectionLabel}>What to share</Text>
              {[{ id: 'kitchen', title: 'Your kitchen' }, ...posts].map((option) => {
                const active = subject === option.id;
                return (
                  <Pressable
                    key={option.id}
                    onPress={() => setSubject(option.id)}
                    accessibilityRole="radio"
                    accessibilityState={{ selected: active }}
                    accessibilityLabel={`Share ${option.title}`}
                    android_ripple={{ color: `${theme.colors.ink.DEFAULT}14` }}
                  >
                    <View style={[styles.subjectRow, active && styles.subjectRowActive]}>
                      <Text
                        style={[styles.subjectLabel, active && styles.subjectLabelActive]}
                        numberOfLines={1}
                      >
                        {option.title}
                      </Text>
                    </View>
                  </Pressable>
                );
              })}
            </View>
          ) : null}

          {url ? (
            <View style={styles.targets}>
              {SHARE_TARGETS.map((target) => (
                <Pressable
                  key={target.id}
                  onPress={() => void onShare(target.id)}
                  disabled={pending !== null}
                  accessibilityRole="button"
                  accessibilityLabel={`Share ${selected ? selected.title : businessName} on ${target.label}`}
                  android_ripple={{ color: `${theme.colors.ink.DEFAULT}14` }}
                >
                  {({ pressed }) => (
                    <View
                      style={[
                        styles.targetRow,
                        pressed && Platform.OS === 'ios' && styles.targetRowPressed,
                      ]}
                    >
                      <Share2 size={18} strokeWidth={2} color={theme.colors.ink.soft} />
                      <View style={styles.targetText}>
                        <Text style={styles.targetLabel}>{target.label}</Text>
                        <Text style={styles.targetHint}>{target.hint}</Text>
                      </View>
                      {pending === target.id ? (
                        <ActivityIndicator size="small" color={theme.colors.ink.soft} />
                      ) : null}
                    </View>
                  )}
                </Pressable>
              ))}
            </View>
          ) : null}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  title: { fontFamily: 'Geist-Bold', fontSize: 20, color: theme.colors.ink.DEFAULT },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  content: { padding: theme.spacing[4], gap: theme.spacing[4], paddingBottom: theme.spacing[8] },

  statsRow: {
    flexDirection: 'row',
    alignItems: 'center',
    borderRadius: 8,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.ink.soft,
    paddingVertical: theme.spacing[4],
  },
  stat: { flex: 1, alignItems: 'center', gap: 4 },
  statDivider: { width: StyleSheet.hairlineWidth, alignSelf: 'stretch', backgroundColor: theme.colors.ink.soft },
  statValue: {
    fontFamily: 'Geist-Bold',
    fontSize: 24,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  statLabel: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.soft },

  explainer: { fontFamily: 'Inter', fontSize: 14, lineHeight: 20, color: theme.colors.ink.soft },

  linkBox: {
    borderRadius: 8,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.ink.soft,
    padding: theme.spacing[3],
    gap: 2,
  },
  linkLabel: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.soft },
  linkUrl: { fontFamily: 'Inter', fontSize: 14, color: theme.colors.ink.DEFAULT },

  subjects: { gap: theme.spacing[2] },
  sectionLabel: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.soft },
  subjectRow: {
    minHeight: 44,
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[3],
    borderRadius: 8,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.ink.soft,
  },
  subjectRowActive: { borderColor: theme.colors.ink.DEFAULT, borderWidth: 1.5 },
  subjectLabel: { fontFamily: 'Inter', fontSize: 15, color: theme.colors.ink.soft },
  subjectLabelActive: { color: theme.colors.ink.DEFAULT, fontWeight: '500' },

  targets: { gap: theme.spacing[2] },
  targetRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    minHeight: 56,
    paddingHorizontal: theme.spacing[3],
    borderRadius: 8,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.ink.soft,
  },
  targetRowPressed: { opacity: 0.6 },
  targetText: { flex: 1 },
  targetLabel: { fontFamily: 'Inter', fontSize: 15, fontWeight: '500', color: theme.colors.ink.DEFAULT },
  targetHint: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.soft },
});
