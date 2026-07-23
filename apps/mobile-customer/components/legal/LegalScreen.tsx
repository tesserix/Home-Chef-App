// Shared layout for static legal/policy screens (Refund, Terms, Privacy,
// EULA). Header + scrollable, section-rendered body. headerShown is false
// app-wide, so each screen draws its own back header.
//
// Typography rhythm only (design-sweep Task 7) — Inter 15/1.5 body copy,
// consistent section spacing. No visual experimentation, no copy changes.

import { Platform, Pressable, ScrollView, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';

// Android ripple tint — translucent token, never a new literal colour.
const ICON_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

export interface LegalSection {
  heading: string;
  /** Each entry is a paragraph. Prefix with "• " to render as a bullet line. */
  paragraphs: string[];
}

interface LegalScreenProps {
  title: string;
  lastUpdated: string;
  intro?: string;
  sections: LegalSection[];
}

export function LegalScreen({ title, lastUpdated, intro, sections }: LegalScreenProps) {
  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      {/* Header */}
      <View className="flex-row items-center bg-canvas border-b border-hairline px-4 pb-4 gap-3">
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={8}
          android_ripple={{ color: ICON_RIPPLE, borderless: true }}
        >
          {({ pressed }) => (
            <View
              className={`p-1 rounded-full ${pressed && Platform.OS === 'ios' ? 'bg-surface-soft' : ''}`}
            >
              <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text className="text-xl font-semibold text-charcoal flex-1" numberOfLines={1}>
          {title}
        </Text>
      </View>

      <ScrollView
        contentContainerStyle={{ padding: 20, paddingBottom: 48 }}
        showsVerticalScrollIndicator={false}
      >
        <Text className="text-xs text-charcoal-soft mb-5">Last updated: {lastUpdated}</Text>

        {/* Body copy: Inter 15/1.5 (text-[15px] leading-[22.5px]) for a
            consistent, comfortable reading rhythm across the legal set. */}
        {intro ? (
          <Text className="text-[15px] leading-[22.5px] text-charcoal-soft mb-7">{intro}</Text>
        ) : null}

        {sections.map((section, i) => (
          <View key={`${section.heading}-${i}`} className="mb-7">
            <Text className="text-base font-semibold text-charcoal mb-2.5">{section.heading}</Text>
            {section.paragraphs.map((p, j) => {
              const isBullet = p.startsWith('• ');
              return (
                <Text
                  key={j}
                  className={`text-[15px] leading-[22.5px] text-charcoal-soft mb-2 ${isBullet ? 'pl-2' : ''}`}
                >
                  {p}
                </Text>
              );
            })}
          </View>
        ))}
      </ScrollView>
    </SafeAreaView>
  );
}
