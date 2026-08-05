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
import * as Clipboard from 'expo-clipboard';
import {
  Bell,
  BookOpen,
  Check,
  ChevronDown,
  ChevronLeft,
  Copy,
  Heart,
  Share2,
} from 'lucide-react-native';
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
import {
  KITCHEN_SUBJECT,
  resolveShareSubject,
  shareSubjectOptions,
} from '../lib/share-subject';

interface PromoteProfile {
  businessName?: string;
  slug?: string;
  likeCount?: number;
  subscriberCount?: number;
  articleReactionCount?: number;
}

export default function PromoteScreen() {
  const { show: showToast } = useToast();
  const [pending, setPending] = useState<SocialNetwork | null>(null);
  // ChefBook sends the post the chef tapped Share on, so they land here with it
  // already chosen rather than picking it out of the list again.
  const { articleId } = useLocalSearchParams<{ articleId?: string }>();
  const [subject, setSubject] = useState<string>(articleId ?? KITCHEN_SUBJECT);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [targetsOpen, setTargetsOpen] = useState(false);

  const { data, isLoading } = useQuery<PromoteProfile>({
    // Same key the Profile screen uses, so an edit there refreshes the name and
    // slug shown here instead of leaving a stale copy behind.
    queryKey: ['chef', 'profile'],
    queryFn: () => api.get<PromoteProfile>('/chef/profile').then((r) => r.data),
    // Likes and subscribers move while the chef is elsewhere in the app; the
    // shared cache was serving counts from whenever Profile last loaded.
    refetchOnMount: 'always',
  });
  const { data: articleData } = useMyArticles();

  const businessName = data?.businessName ?? '';
  const slug = data?.slug ?? '';
  const url = slug ? chefPublicUrl(slug) : '';

  const options = useMemo(
    () => shareSubjectOptions(articleData?.data ?? [], businessName),
    [articleData, businessName],
  );
  const selected = resolveShareSubject(options, subject);
  const isKitchen = !selected || selected.id === KITCHEN_SUBJECT;

  const onShare = useCallback(
    async (network: SocialNetwork) => {
      if (!url) return;
      setPending(network);
      try {
        const content =
          selected && selected.id !== KITCHEN_SUBJECT
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

  const onCopy = useCallback(async () => {
    if (!url) return;
    await Clipboard.setStringAsync(url);
    showToast({ message: 'Link copied', tone: 'success' });
  }, [url, showToast]);

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
              <View style={styles.linkRow}>
                <Text style={styles.linkUrl} numberOfLines={1}>
                  {url}
                </Text>
                <Pressable
                  onPress={() => void onCopy()}
                  hitSlop={8}
                  style={styles.iconButton}
                  accessibilityRole="button"
                  accessibilityLabel="Copy your page link"
                  android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
                >
                  <Copy size={18} strokeWidth={2} color={theme.colors.ink.soft} />
                </Pressable>
                <Pressable
                  onPress={() => setTargetsOpen((o) => !o)}
                  hitSlop={8}
                  style={styles.iconButton}
                  accessibilityRole="button"
                  accessibilityState={{ expanded: targetsOpen }}
                  accessibilityLabel={targetsOpen ? 'Hide share options' : 'Show share options'}
                  android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
                >
                  <Share2
                    size={18}
                    strokeWidth={2}
                    color={targetsOpen ? theme.colors.herb.DEFAULT : theme.colors.ink.soft}
                  />
                </Pressable>
              </View>
            </View>
          ) : (
            // No slug means the profile has no business name yet — sharing a
            // broken link would be worse than saying so.
            <Text style={styles.explainer}>
              Finish your profile to get a shareable page for your kitchen.
            </Text>
          )}

          {url && options.length > 1 ? (
            <View style={styles.subjects}>
              <Text style={styles.sectionLabel}>What to share</Text>
              <Pressable
                onPress={() => setPickerOpen((o) => !o)}
                accessibilityRole="button"
                accessibilityState={{ expanded: pickerOpen }}
                accessibilityLabel={`Sharing ${selected?.title ?? 'your kitchen'}. Tap to change.`}
                android_ripple={{ color: `${theme.colors.ink.DEFAULT}14` }}
              >
                <View style={styles.pickerRow}>
                  <Text style={styles.pickerValue} numberOfLines={1}>
                    {selected?.title ?? 'Your kitchen'}
                  </Text>
                  <ChevronDown
                    size={18}
                    strokeWidth={2}
                    color={theme.colors.ink.soft}
                    style={pickerOpen ? styles.chevronOpen : undefined}
                  />
                </View>
              </Pressable>
              {pickerOpen
                ? options.map((option) => {
                    const active = selected?.id === option.id;
                    return (
                      <Pressable
                        key={option.id}
                        onPress={() => {
                          setSubject(option.id);
                          setPickerOpen(false);
                        }}
                        accessibilityRole="radio"
                        accessibilityState={{ selected: active }}
                        accessibilityLabel={`Share ${option.title}`}
                        android_ripple={{ color: `${theme.colors.ink.DEFAULT}14` }}
                      >
                        <View style={styles.optionRow}>
                          <Text
                            style={[styles.subjectLabel, active && styles.subjectLabelActive]}
                            numberOfLines={1}
                          >
                            {option.title}
                          </Text>
                          {active ? (
                            <Check size={18} strokeWidth={2.5} color={theme.colors.herb.DEFAULT} />
                          ) : null}
                        </View>
                      </Pressable>
                    );
                  })
                : null}
            </View>
          ) : null}

          {url && targetsOpen ? (
            <View style={styles.targets}>
              {SHARE_TARGETS.map((target) => (
                <Pressable
                  key={target.id}
                  onPress={() => void onShare(target.id)}
                  disabled={pending !== null}
                  accessibilityRole="button"
                  accessibilityLabel={`Share ${isKitchen ? businessName : selected!.title} on ${target.label}`}
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
  linkRow: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[2] },
  linkUrl: { flex: 1, fontFamily: 'Inter', fontSize: 14, color: theme.colors.ink.DEFAULT },
  iconButton: { width: 44, height: 44, alignItems: 'center', justifyContent: 'center' },

  subjects: { gap: theme.spacing[2] },
  sectionLabel: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.soft },
  pickerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    minHeight: 44,
    paddingHorizontal: theme.spacing[3],
    borderRadius: 8,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.ink.soft,
  },
  pickerValue: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: 15,
    fontWeight: '500',
    color: theme.colors.ink.DEFAULT,
  },
  chevronOpen: { transform: [{ rotate: '180deg' }] },
  optionRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    minHeight: 44,
    paddingHorizontal: theme.spacing[3],
  },
  subjectLabel: { flex: 1, fontFamily: 'Inter', fontSize: 15, color: theme.colors.ink.soft },
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
