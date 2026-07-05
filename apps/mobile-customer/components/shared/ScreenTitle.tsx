// ScreenTitle — the one large-title treatment for top-level tab screens
// (Orders / Saved / Profile). Geist display face, consistent size/tracking/
// gutters, so every tab reads like one hand designed it. Home has its own
// header (address + search) and does not use this.

import { StyleSheet, Text, View } from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';

interface ScreenTitleProps {
  title: string;
}

export function ScreenTitle({ title }: ScreenTitleProps) {
  return (
    <View style={styles.wrapper}>
      <Text style={styles.title} accessibilityRole="header">
        {title}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  wrapper: {
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 8,
  },
  title: {
    fontFamily: 'Geist-Bold',
    fontSize: 27,
    color: customerColors.charcoal.DEFAULT,
    letterSpacing: -0.3,
  },
});
