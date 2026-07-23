// Dock — floating bottom navigation for the vendor app (replaces the flush
// custom tab bar). A detached rounded bar hovering above the home indicator:
// solid white (paper) surface, mist hairline border, soft shadow —
// deliberately NO blur/glass. Inactive tabs are icon-only; the active tab
// expands into a solid ink pill with a white label (the dock's single accent).
// No cart FAB — the vendor app has no cart.
//
// Rendered via expo-router's <Tabs tabBar={...}>. The root View is absolutely
// positioned so scenes get full screen height and content scrolls visibly
// through the gap beneath — screens pad their scroll bottom by
// useDockClearance() so the last row clears the dock.

import { Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import Animated, { Easing, FadeIn, useReducedMotion } from 'react-native-reanimated';
import {
  LayoutDashboard,
  ClipboardList,
  UtensilsCrossed,
  MoreHorizontal,
  type LucideIcon,
} from 'lucide-react-native';
import { colors } from '@homechef/mobile-shared/theme';
import { DOCK_BOTTOM_GAP, DOCK_HEIGHT, useDockClearance } from './dock-metrics';

// Screens keep importing clearance helpers from here too.
export { DOCK_BOTTOM_GAP, DOCK_HEIGHT, useDockClearance };

// Entrances use the app-standard ease-out-quart — no bounce, no overshoot.
const ENTRANCE_EASING = Easing.bezier(0.22, 1, 0.36, 1);

// Android ripple tint for the (icon-only) tab slots — translucent ink
// derived from the ink token, borderless so it reads as an icon-button
// ripple rather than filling the whole flex slot.
const TAB_RIPPLE = `${colors.ink.DEFAULT}14`;

const TAB_ICONS: Record<string, LucideIcon> = {
  index: LayoutDashboard,
  orders: ClipboardList,
  menu: UtensilsCrossed,
  more: MoreHorizontal,
};

// Short display labels for the active pill. Slots are equal-width, so the
// label must fit an even quarter of the bar — "Dashboard" (the screen title)
// is too long and would truncate, so the dock shows "Home". The full title is
// still used for the accessibility label.
const TAB_LABELS: Record<string, string> = {
  index: 'Home',
  orders: 'Orders',
  menu: 'Menu',
  more: 'More',
};

// Minimal structural types for the react-navigation tabBar props — the
// package isn't a direct dependency (expo-router wraps it), so we type the
// slice we consume.
interface DockRoute {
  key: string;
  name: string;
}
interface DockProps {
  state: { index: number; routes: DockRoute[] };
  descriptors: Record<string, { options: { title?: string } }>;
  navigation: {
    emit: (event: {
      type: 'tabPress';
      target: string;
      canPreventDefault: true;
    }) => { defaultPrevented: boolean };
    navigate: (name: string) => void;
  };
}

export function Dock({ state, descriptors, navigation }: DockProps) {
  const insets = useSafeAreaInsets();
  const reduceMotion = useReducedMotion();

  return (
    <View
      style={[styles.root, { bottom: insets.bottom + DOCK_BOTTOM_GAP }]}
      pointerEvents="box-none"
    >
      <View style={styles.bar} accessibilityRole="tablist">
        {state.routes.map((route, index) => {
          const { options } = descriptors[route.key] ?? { options: {} };
          const label = options?.title ?? route.name;
          const pillLabel = TAB_LABELS[route.name] ?? label;
          const isActive = state.index === index;
          const Icon = TAB_ICONS[route.name] ?? LayoutDashboard;

          const onPress = () => {
            const event = navigation.emit({
              type: 'tabPress',
              target: route.key,
              canPreventDefault: true,
            });
            if (!isActive && !event.defaultPrevented) {
              navigation.navigate(route.name);
            }
          };

          return (
            <Pressable
              key={route.key}
              onPress={onPress}
              accessibilityRole="tab"
              accessibilityState={{ selected: isActive }}
              accessibilityLabel={label}
              style={styles.slot}
              android_ripple={{ color: TAB_RIPPLE, borderless: true, radius: 28 }}
            >
              {({ pressed }) => (
                <View
                  style={
                    pressed && Platform.OS === 'ios' ? styles.slotPressedIOS : undefined
                  }
                >
                  {isActive ? (
                    <Animated.View
                      entering={
                        reduceMotion
                          ? undefined
                          : FadeIn.duration(250).easing(ENTRANCE_EASING)
                      }
                      style={styles.activePill}
                    >
                      <Icon size={20} color={colors.paper} />
                      <Text style={styles.activeLabel} numberOfLines={1}>
                        {pillLabel}
                      </Text>
                    </Animated.View>
                  ) : (
                    <Icon size={22} color={colors.ink.muted} />
                  )}
                </View>
              )}
            </Pressable>
          );
        })}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    position: 'absolute',
    left: 16,
    right: 16,
  },
  // Solid white pill — hairline border + soft shadow, no blur.
  bar: {
    flexDirection: 'row',
    alignItems: 'center',
    height: DOCK_HEIGHT,
    borderRadius: 28,
    backgroundColor: colors.paper,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: colors.mist.DEFAULT,
    paddingHorizontal: 8,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.1,
    shadowRadius: 16,
    elevation: 10,
  },
  // Every slot is an equal quarter of the bar — icons never move when the
  // active tab changes. ≥44px touch target comes from the bar height.
  slot: {
    flex: 1,
    height: '100%',
    alignItems: 'center',
    justifyContent: 'center',
  },
  // iOS-only pressed feedback (Android gets android_ripple on the Pressable
  // above — layering both would read as a double, janky press).
  slotPressedIOS: {
    opacity: 0.6,
  },
  activePill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: 14,
    paddingVertical: 9,
    borderRadius: 20,
    backgroundColor: colors.ink.DEFAULT,
  },
  activeLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: colors.paper,
  },
});
