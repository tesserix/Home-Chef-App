import type { ReactNode } from 'react';
import { ScrollView, StatusBar, StyleSheet, View } from 'react-native';
import {
  SafeAreaView,
  type Edge,
  type SafeAreaViewProps,
} from 'react-native-safe-area-context';
import { theme } from '../theme/tokens';

interface ScreenProps {
  children: ReactNode;
  /** Add ScrollView wrapper. Default false — most screens scroll their own
   * sub-region (a FlatList, a form ScrollView) and don't want the whole
   * surface scrolling. */
  scroll?: boolean;
  /** Override the safe-area edges. Default ['top', 'left', 'right'] —
   * bottom is usually owned by a tab bar or sticky CTA. */
  edges?: ReadonlyArray<Edge>;
  /** Override background. Default `theme.colors.paper`. */
  background?: string;
  /** Inner horizontal padding. Default `theme.spacing[4]` (16). */
  paddingX?: number;
  /** Optional StatusBar style override. Default 'dark' (matches paper). */
  statusBar?: 'dark' | 'light';
  testID?: string;
}

/**
 * <Screen> — the surface every top-level screen uses.
 *
 * Centralises:
 *   - SafeAreaView edges (top by default; bottom owned by the tab bar)
 *   - Paper background (per .impeccable.md; never pure #fff)
 *   - StatusBar style harmonised with the surface
 *   - Optional ScrollView so the screen author doesn't reinvent inset math
 *
 * Use this everywhere instead of bare SafeAreaView. When a screen
 * needs to break the rules (a full-bleed photo, a tinted status bar)
 * use bare RN components and document why.
 */
export function Screen({
  children,
  scroll = false,
  edges,
  background = theme.colors.paper,
  paddingX = theme.spacing[4],
  statusBar = 'dark',
  testID,
}: ScreenProps) {
  const safeAreaProps: SafeAreaViewProps = {
    edges: (edges ?? ['top', 'left', 'right']) as SafeAreaViewProps['edges'],
    style: [styles.root, { backgroundColor: background }],
  };

  const content = (
    <View style={{ flex: 1, paddingHorizontal: paddingX }}>{children}</View>
  );

  return (
    <SafeAreaView {...safeAreaProps} testID={testID}>
      <StatusBar
        barStyle={statusBar === 'dark' ? 'dark-content' : 'light-content'}
        backgroundColor={background}
      />
      {scroll ? (
        <ScrollView
          contentContainerStyle={{ flexGrow: 1 }}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          {content}
        </ScrollView>
      ) : (
        content
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
});
