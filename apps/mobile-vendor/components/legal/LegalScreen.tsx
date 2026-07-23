// Template legal content — have counsel review before launch (see COUNSEL-REVIEW.md).
//
// Shared layout for static legal/policy screens in the vendor app (Privacy,
// Terms, Chef Agreement, EULA). Mirrors the customer LegalScreen prop contract
// but renders with the vendor StyleSheet + theme system — the vendor app has no
// NativeWind className support. headerShown is false app-wide, so this draws its
// own back command bar (copied from settings.tsx).

import { Platform, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';

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
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* Command bar — back chevron left, Geist-Bold 28pt title */}
      <View style={styles.commandBar}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityLabel="Go back"
          accessibilityRole="button"
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
        >
          {({ pressed }) => (
            <View
              style={[styles.backBtn, pressed && Platform.OS === 'ios' && { opacity: 0.6 }]}
            >
              <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} strokeWidth={1.75} />
            </View>
          )}
        </Pressable>
        <Text style={styles.commandTitle} numberOfLines={1}>
          {title}
        </Text>
      </View>

      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
      >
        <Text style={styles.lastUpdated}>Last updated: {lastUpdated}</Text>

        {intro ? <Text style={styles.intro}>{intro}</Text> : null}

        {sections.map((section, i) => (
          <View key={`${section.heading}-${i}`} style={styles.section}>
            <Text style={styles.heading}>{section.heading}</Text>
            {section.paragraphs.map((p, j) => {
              const isBullet = p.startsWith('• ');
              return (
                <Text
                  key={j}
                  style={[styles.paragraph, isBullet && styles.bullet]}
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

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: theme.colors.bone,
  },

  // Command bar — matches settings/more geometry
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
    gap: theme.spacing[2],
  },
  backBtn: {
    marginRight: theme.spacing[1],
  },
  commandTitle: {
    flex: 1,
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },

  scroll: { flex: 1 },
  scrollContent: {
    paddingHorizontal: theme.spacing[5],
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[10],
  },

  lastUpdated: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[4],
  },
  intro: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    lineHeight: 22,
    color: theme.colors.ink.soft,
    marginBottom: theme.spacing[6],
  },

  section: {
    marginBottom: theme.spacing[6],
  },
  heading: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    marginBottom: theme.spacing[2],
  },
  paragraph: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    lineHeight: 22,
    color: theme.colors.ink.soft,
    marginBottom: theme.spacing[2],
  },
  bullet: {
    paddingLeft: theme.spacing[2],
  },
});
