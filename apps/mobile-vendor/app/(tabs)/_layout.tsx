import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { Tabs } from 'expo-router';
import {
  LayoutDashboard,
  ClipboardList,
  UtensilsCrossed,
  MoreHorizontal,
} from 'lucide-react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
// SDK 56: expo-router no longer re-exports react-navigation at the bare
// `@react-navigation/bottom-tabs` specifier, and the root-hoisted copy's
// `BottomTabBarProps` is a structurally different type from the one
// expo-router's <Tabs> actually passes to `tabBar`. Import the type from
// expo-router's own public `tabs` subpath so CustomTabBar matches exactly.
import type { BottomTabBarProps } from 'expo-router/tabs';
import { theme } from '@homechef/mobile-shared/theme';

// Custom tab bar — replaces React Navigation's default BottomTabBar so we
// can fully control layout.
//
// Implementation notes:
//   - `TouchableOpacity` instead of `Pressable`. iOS Pressable drops
//     `flex: 1` whether the style is supplied as an array OR as an
//     object returned from the function — see
//     feedback_ios_pressable_array_style.md. Without flex honored, all
//     four tabs collapse against the left edge. TouchableOpacity has no
//     such quirk.
//   - Inner View carries `width: '100%'` so the column fills its flex
//     slot. (Default cross-axis stretch should do this on its own, but
//     explicit width belt-and-suspenders against any inherited
//     alignItems.)
//   - `allowFontScaling={false}` + `numberOfLines={1}` on the label so
//     iOS Dynamic Type at 130%+ can't widen the text past the column
//     and re-introduce truncation.

function CustomTabBar({ state, descriptors, navigation }: BottomTabBarProps) {
  const insets = useSafeAreaInsets();

  return (
    <View
      style={[
        styles.bar,
        { paddingBottom: Math.max(insets.bottom, 8) },
      ]}
    >
      {state.routes.map((route, index) => {
        const { options } = descriptors[route.key];
        const isFocused = state.index === index;
        const label =
          (options.tabBarLabel as string | undefined) ??
          options.title ??
          route.name;
        const tintColor = isFocused
          ? theme.colors.ink.DEFAULT
          : theme.colors.ink.muted;

        function onPress(): void {
          const event = navigation.emit({
            type: 'tabPress',
            target: route.key,
            canPreventDefault: true,
          });
          if (!isFocused && !event.defaultPrevented) {
            navigation.navigate(route.name);
          }
        }

        function onLongPress(): void {
          navigation.emit({ type: 'tabLongPress', target: route.key });
        }

        return (
          <TouchableOpacity
            key={route.key}
            onPress={onPress}
            onLongPress={onLongPress}
            activeOpacity={0.7}
            style={styles.itemTouchable}
            accessibilityRole="tab"
            accessibilityState={isFocused ? { selected: true } : {}}
            accessibilityLabel={
              options.tabBarAccessibilityLabel ?? `${label} tab`
            }
            testID={options.tabBarButtonTestID}
          >
            <View style={styles.itemInner}>
              {options.tabBarIcon
                ? options.tabBarIcon({
                    focused: isFocused,
                    color: tintColor,
                    size: 24,
                  })
                : null}
              <Text
                style={[
                  styles.label,
                  isFocused && styles.labelActive,
                ]}
                allowFontScaling={false}
                numberOfLines={1}
              >
                {label}
              </Text>
            </View>
          </TouchableOpacity>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  bar: {
    flexDirection: 'row',
    backgroundColor: theme.colors.paper,
    // Top elevation instead of a hairline border — spec §6.
    shadowColor: theme.colors.ink.DEFAULT,
    shadowOffset: { width: 0, height: -2 },
    shadowOpacity: 0.06,
    shadowRadius: 12,
    elevation: 8,
  },
  itemTouchable: {
    flex: 1,
  },
  itemInner: {
    width: '100%',
    alignItems: 'center',
    justifyContent: 'flex-start',
    paddingTop: 10,
    paddingBottom: 6,
    minHeight: 56,
  },
  label: {
    // `alignSelf: 'stretch'` overrides the parent itemInner's
    // `alignItems: 'center'` for THIS child only — so the Text occupies
    // the full column width (TouchableOpacity is flex:1 → 25% of bar)
    // instead of being measured at content width and clipped by
    // numberOfLines={1}. `textAlign: 'center'` then centers the
    // characters within that full-width box.
    alignSelf: 'stretch',
    textAlign: 'center',
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    letterSpacing: 0.1,
    marginTop: 4,
    color: theme.colors.ink.muted,
    includeFontPadding: false,
  },
  labelActive: {
    color: theme.colors.ink.DEFAULT,
  },
});

export default function VendorTabsLayout() {
  return (
    <Tabs
      tabBar={(props) => <CustomTabBar {...props} />}
      screenOptions={{
        headerShown: false,
      }}
    >
      <Tabs.Screen
        name="index"
        options={{
          title: 'Dashboard',
          tabBarIcon: ({ color, focused, size }) => (
            <LayoutDashboard
              size={size}
              color={String(color)}
              strokeWidth={focused ? 2.4 : 2}
            />
          ),
        }}
      />
      <Tabs.Screen
        name="orders"
        options={{
          title: 'Orders',
          tabBarIcon: ({ color, focused, size }) => (
            <ClipboardList
              size={size}
              color={String(color)}
              strokeWidth={focused ? 2.4 : 2}
            />
          ),
        }}
      />
      <Tabs.Screen
        name="menu"
        options={{
          title: 'Menu',
          tabBarIcon: ({ color, focused, size }) => (
            <UtensilsCrossed
              size={size}
              color={String(color)}
              strokeWidth={focused ? 2.4 : 2}
            />
          ),
        }}
      />
      <Tabs.Screen
        name="more"
        options={{
          title: 'More',
          tabBarIcon: ({ color, focused, size }) => (
            <MoreHorizontal
              size={size}
              color={String(color)}
              strokeWidth={focused ? 2.4 : 2}
            />
          ),
        }}
      />
    </Tabs>
  );
}
