// In-page tab bar for the chef detail screen — Airbnb underline style. The
// active tab label is charcoal with a 2px coral underline (the one accent);
// inactive labels are charcoal-soft. Purely presentational: the screen owns
// which tabs exist (Weekly plan is gated by TIFFIN_ENABLED) and the active key.

import { Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';

// Android ripple tint — translucent charcoal derived from the token.
const TAB_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

export type ChefDetailTabKey = 'menu' | 'weekly' | 'reviews';

export interface ChefDetailTab {
  key: ChefDetailTabKey;
  label: string;
}

export interface ChefDetailTabsProps {
  tabs: ChefDetailTab[];
  activeTab: ChefDetailTabKey;
  onChange: (tab: ChefDetailTabKey) => void;
}

export function ChefDetailTabs({ tabs, activeTab, onChange }: ChefDetailTabsProps) {
  return (
    <View style={styles.row}>
      {tabs.map((tab) => {
        const isActive = tab.key === activeTab;
        return (
          <Pressable
            key={tab.key}
            onPress={() => onChange(tab.key)}
            accessibilityRole="tab"
            accessibilityLabel={tab.label}
            accessibilityState={{ selected: isActive }}
            android_ripple={{ color: TAB_RIPPLE }}
          >
            {({ pressed }) => (
              // Visual + layout styles on the inner View — iOS drops
              // flex/bg/padding returned from a Pressable style function.
              <View style={[styles.tab, pressed && Platform.OS === 'ios' && styles.tabPressed]}>
                <Text style={[styles.label, isActive && styles.labelActive]}>
                  {tab.label}
                </Text>
                {isActive ? <View style={styles.underline} /> : null}
              </View>
            )}
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    paddingHorizontal: 20,
    gap: 24,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  // ≥44pt touch target.
  tab: {
    minHeight: 44,
    paddingTop: 12,
    paddingBottom: 12,
    justifyContent: 'center',
    position: 'relative',
  },
  // Opacity-only press feedback (no scale/bounce; reduced-motion safe).
  tabPressed: {
    opacity: 0.7,
  },
  label: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    letterSpacing: 0.1,
    color: customerColors.charcoal.soft,
  },
  labelActive: {
    color: customerColors.charcoal.DEFAULT,
  },
  // 2px coral underline on the active tab (Airbnb category-bar style).
  underline: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    height: 2,
    borderRadius: 1,
    backgroundColor: customerColors.coral.DEFAULT,
  },
});
