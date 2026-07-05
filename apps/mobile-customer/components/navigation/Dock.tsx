// Dock — floating bottom navigation for the customer app (replaces the
// classic edge-to-edge tab bar). A detached rounded bar hovering above the
// home indicator: solid canvas, hairline border, soft shadow — deliberately
// NO blur/glass. Inactive tabs are icon-only; the active tab expands into a
// coral-tint pill with its label (the dock's single accent). When the cart
// has items a solid-coral cart pill docks onto the right end as a dynamic
// fifth slot (see DockCartPill).
//
// Rendered via expo-router's <Tabs tabBar={...}>. The root View is
// absolutely positioned, so scenes get the full screen height and content
// scrolls visibly through the gap beneath the dock — screens must pad their
// scrollable bottom by useDockClearance() so the last row clears the dock.

import { Pressable, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import Animated, { Easing, FadeIn, useReducedMotion } from 'react-native-reanimated';
import { Heart, Home, ShoppingBag, User, type LucideIcon } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { DockCartPill } from './DockCartPill';

/** Dock bar height. */
export const DOCK_HEIGHT = 64;
/** Gap between the dock and the bottom safe-area inset. */
export const DOCK_BOTTOM_GAP = 12;

// Entrances use the app-standard ease-out-quart — no bounce, no overshoot.
const ENTRANCE_EASING = Easing.bezier(0.22, 1, 0.36, 1);

/**
 * Bottom padding a screen needs so scrollable content clears the floating
 * dock (dock height + gap + safe-area + a breathing row of space).
 */
export function useDockClearance(): number {
  const insets = useSafeAreaInsets();
  return insets.bottom + DOCK_BOTTOM_GAP + DOCK_HEIGHT + 12;
}

const TAB_ICONS: Record<string, LucideIcon> = {
  index: Home,
  orders: ShoppingBag,
  favorites: Heart,
  profile: User,
};

// Minimal structural types for the react-navigation tabBar props — the
// package isn't a direct dependency (expo-router wraps it), so we type the
// slice we consume instead of importing BottomTabBarProps.
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
          const isActive = state.index === index;
          const Icon = TAB_ICONS[route.name] ?? Home;

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
                  <Icon size={20} color={customerColors.coral.DEFAULT} />
                  <Text style={styles.activeLabel} numberOfLines={1}>
                    {label}
                  </Text>
                </Animated.View>
              ) : (
                <Icon size={22} color={customerColors.charcoal.soft} />
              )}
            </Pressable>
          );
        })}

        {/* Dynamic fifth slot — only when the cart has items. */}
        <DockCartPill />
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
  // Solid canvas pill — hairline border + soft shadow, no blur.
  bar: {
    flexDirection: 'row',
    alignItems: 'center',
    height: DOCK_HEIGHT,
    borderRadius: 28,
    backgroundColor: customerColors.canvas,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    paddingHorizontal: 8,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.1,
    shadowRadius: 16,
    elevation: 10,
  },
  // Each slot flexes equally; ≥44px touch target comes from the bar height.
  slot: {
    flex: 1,
    height: '100%',
    alignItems: 'center',
    justifyContent: 'center',
  },
  activePill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: 14,
    paddingVertical: 9,
    borderRadius: 20,
    backgroundColor: customerColors.coral.tint,
  },
  activeLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: customerColors.coral.DEFAULT,
  },
});
