// Shared layout for static legal/policy screens (Refund, Terms, Privacy).
// Header + scrollable, section-rendered body. headerShown is false app-wide,
// so each screen draws its own back header.

import { Pressable, ScrollView, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';

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
          className="p-1"
        >
          <ChevronLeft size={24} color="#222222" />
        </Pressable>
        <Text className="text-xl font-semibold text-charcoal flex-1" numberOfLines={1}>
          {title}
        </Text>
      </View>

      <ScrollView
        contentContainerStyle={{ padding: 20, paddingBottom: 48 }}
        showsVerticalScrollIndicator={false}
      >
        <Text className="text-xs text-charcoal-soft mb-4">Last updated: {lastUpdated}</Text>

        {intro ? (
          <Text className="text-sm text-charcoal-soft leading-6 mb-6">{intro}</Text>
        ) : null}

        {sections.map((section, i) => (
          <View key={`${section.heading}-${i}`} className="mb-6">
            <Text className="text-base font-semibold text-charcoal mb-2">{section.heading}</Text>
            {section.paragraphs.map((p, j) => {
              const isBullet = p.startsWith('• ');
              return (
                <Text
                  key={j}
                  className={`text-sm text-charcoal-soft leading-6 mb-2 ${isBullet ? 'pl-2' : ''}`}
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
