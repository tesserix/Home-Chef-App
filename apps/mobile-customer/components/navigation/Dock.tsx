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

import { Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import Animated, { Easing, FadeIn, useReducedMotion } from 'react-native-reanimated';
import { CalendarCheck, Heart, Home, ShoppingBag, User, type LucideIcon } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { CartFab } from './DockCartPill';
import { DOCK_BOTTOM_GAP, DOCK_HEIGHT, useDockClearance } from './dock-metrics';

// Geometry lives in ./dock-metrics (shared with CartFab, no import cycle).
// Screens keep importing useDockClearance from here.
export { DOCK_BOTTOM_GAP, DOCK_HEIGHT, useDockClearance };

// Entrances use the app-standard ease-out-quart — no bounce, no overshoot.
const ENTRANCE_EASING = Easing.bezier(0.22, 1, 0.36, 1);

// Android ripple tint for the (icon-only) tab slots — translucent ink
// derived from the charcoal token, borderless so it reads as an icon-button
// ripple rather than filling the whole flex slot.
const TAB_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

const TAB_ICONS: Record<string, LucideIcon> = {
  index: Home,
  orders: ShoppingBag,
  plans: CalendarCheck,
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
                      <Icon size={20} color={customerColors.coral.DEFAULT} />
                      <Text style={styles.activeLabel} numberOfLines={1}>
                        {label}
                      </Text>
                    </Animated.View>
                  ) : (
                    <Icon size={22} color={customerColors.charcoal.soft} />
                  )}
                </View>
              )}
            </Pressable>
          );
        })}

      </View>

      {/* Floating cart — hovers above the dock's right end when the cart has
          items (an action layered over navigation, not a fifth tab). */}
      <CartFab />
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
    backgroundColor: customerColors.coral.tint,
  },
  activeLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: customerColors.coral.DEFAULT,
  },
});
