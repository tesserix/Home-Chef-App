import { useEffect, useState } from 'react';
import { Platform, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft, Check } from 'lucide-react-native';
import { useTranslation } from 'react-i18next';
import { theme } from '@homechef/mobile-shared/theme';
import {
  getStoredLocalePreference,
  setLocale,
  type AppLocale,
} from '../lib/i18n';

type Choice = AppLocale | 'system';

interface Option {
  value: Choice;
  labelKey: string;
}

const OPTIONS: Option[] = [
  { value: 'system', labelKey: 'language.system' },
  { value: 'en', labelKey: 'language.english' },
  { value: 'hi', labelKey: 'language.hindi' },
];

export default function LanguageScreen() {
  const { t } = useTranslation();
  const [selected, setSelected] = useState<Choice>('system');

  useEffect(() => {
    getStoredLocalePreference().then(setSelected);
  }, []);

  async function choose(value: Choice): Promise<void> {
    setSelected(value);
    await setLocale(value); // triggers i18next re-render app-wide
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.commandBar}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel={t('common.close')}
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.backButton,
                pressed && Platform.OS === 'ios' && { opacity: 0.6 },
              ]}
            >
              <ChevronLeft size={22} color={theme.colors.ink.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text style={styles.commandTitle}>{t('language.title')}</Text>
      </View>

      <ScrollView contentContainerStyle={styles.scrollContent}>
        <Text style={styles.subtitle}>{t('language.subtitle')}</Text>
        <View style={styles.card}>
          <View style={styles.cardInner}>
            {OPTIONS.map((opt, index) => {
              const active = selected === opt.value;
              return (
                <View key={opt.value}>
                  <Pressable
                    onPress={() => choose(opt.value)}
                    accessibilityRole="radio"
                    accessibilityState={{ selected: active }}
                    accessibilityLabel={t(opt.labelKey)}
                    android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
                  >
                    {({ pressed }) => (
                      <View
                        style={[
                          styles.row,
                          pressed && Platform.OS === 'ios' && styles.rowPressed,
                        ]}
                      >
                        <Text style={styles.rowLabel}>{t(opt.labelKey)}</Text>
                        {active ? (
                          <Check
                            size={20}
                            color={theme.colors.ink.DEFAULT}
                            strokeWidth={2}
                          />
                        ) : null}
                      </View>
                    )}
                  </Pressable>
                  {index < OPTIONS.length - 1 ? (
                    <View style={styles.separator} />
                  ) : null}
                </View>
              );
            })}
          </View>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
    gap: theme.spacing[2],
  },
  backButton: {
    minWidth: 44,
    minHeight: 44,
    alignItems: 'flex-start',
    justifyContent: 'center',
  },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },
  scrollContent: { paddingBottom: theme.spacing[10] },
  subtitle: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[3],
  },
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    marginHorizontal: theme.spacing[4],
    ...theme.shadow[1],
  },
  cardInner: { borderRadius: theme.radius.lg, overflow: 'hidden' },
  rowPressed: { backgroundColor: theme.colors.bone },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    minHeight: 56,
  },
  rowLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  separator: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginLeft: theme.spacing[4],
  },
});
