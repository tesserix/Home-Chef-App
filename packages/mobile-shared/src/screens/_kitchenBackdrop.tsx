// packages/mobile-shared/src/screens/_kitchenBackdrop.tsx
//
// A whisper-quiet kitchen sketch layer for the auth/onboarding surfaces:
// oversized line-icon glyphs (pan, cloche, steam) in ink at 4–5% opacity,
// bleeding off the screen edges. Decorative only — hidden from screen
// readers and untouchable (pointerEvents none). Keep opacities ≤ 0.05 so
// it reads as paper grain, not artwork (.impeccable.md: chrome-light,
// nothing loud).

import React from 'react';
import { StyleSheet, View } from 'react-native';
import Svg, { Circle, Path } from 'react-native-svg';
import { theme } from '../theme/tokens';

const STROKE = theme.colors.ink.DEFAULT;

function Pan({ size }: { size: number }) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Circle cx={9.5} cy={12} r={6.5} stroke={STROKE} strokeWidth={1.1} />
      <Circle cx={9.5} cy={12} r={4} stroke={STROKE} strokeWidth={0.7} />
      <Path d="M16.2 12h5.6" stroke={STROKE} strokeWidth={1.1} strokeLinecap="round" />
    </Svg>
  );
}

function Cloche({ size }: { size: number }) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      <Path d="M4 16a8 8 0 0 1 16 0" stroke={STROKE} strokeWidth={1.1} />
      <Path d="M2.5 16.2h19" stroke={STROKE} strokeWidth={1.1} strokeLinecap="round" />
      <Circle cx={12} cy={6.8} r={1} stroke={STROKE} strokeWidth={0.9} />
    </Svg>
  );
}

function Steam({ size }: { size: number }) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill="none">
      {[7, 12, 17].map((x) => (
        <Path
          key={x}
          d={`M${x} 5c-1.4 1.8-1.4 3.6 0 5.4s1.4 3.6 0 5.4`}
          stroke={STROKE}
          strokeWidth={1}
          strokeLinecap="round"
        />
      ))}
    </Svg>
  );
}

export function KitchenBackdrop() {
  return (
    <View
      style={StyleSheet.absoluteFill}
      pointerEvents="none"
      accessibilityElementsHidden
      importantForAccessibility="no-hide-descendants"
    >
      <View style={[styles.item, styles.pan]}>
        <Pan size={150} />
      </View>
      <View style={[styles.item, styles.cloche]}>
        <Cloche size={180} />
      </View>
      <View style={[styles.item, styles.steam]}>
        <Steam size={120} />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  item: { position: 'absolute' },
  pan: { top: 36, right: -30, opacity: 0.05, transform: [{ rotate: '18deg' }] },
  cloche: { top: 300, left: -48, opacity: 0.04, transform: [{ rotate: '-10deg' }] },
  steam: { bottom: 140, right: -14, opacity: 0.04, transform: [{ rotate: '6deg' }] },
});
