import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { Tabs } from 'expo-router';
import {
  LayoutDashboard,
  ClipboardList,
  UtensilsCrossed,
  MoreHorizontal,
} from 'lucide-react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import type { BottomTabBarProps } from '@react-navigation/bottom-tabs';
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
//   - Inner View carries `width: '100%'` so the absolute-positioned
//     persimmon top line spans the full column. (Default cross-axis
//     stretch should do this on its own, but explicit width belt-and-
//     suspenders against any inherited alignItems.)
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
              <View
                style={[
                  styles.topLine,
                  isFocused && styles.topLineActive,
                ]}
              />
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
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
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
  topLine: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    height: 2,
    backgroundColor: 'transparent',
  },
  topLineActive: {
    backgroundColor: theme.colors.herb.DEFAULT,
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
          tabBarIcon: ({
            color,
            focused,
            size,
          }: {
            color: string;
            focused: boolean;
            size: number;
          }) => (
            <LayoutDashboard
              size={size}
              color={color}
              strokeWidth={focused ? 2.4 : 2}
            />
          ),
        }}
      />
      <Tabs.Screen
        name="orders"
        options={{
          title: 'Orders',
          tabBarIcon: ({
            color,
            focused,
            size,
          }: {
            color: string;
            focused: boolean;
            size: number;
          }) => (
            <ClipboardList
              size={size}
              color={color}
              strokeWidth={focused ? 2.4 : 2}
            />
          ),
        }}
      />
      <Tabs.Screen
        name="menu"
        options={{
          title: 'Menu',
          tabBarIcon: ({
            color,
            focused,
            size,
          }: {
            color: string;
            focused: boolean;
            size: number;
          }) => (
            <UtensilsCrossed
              size={size}
              color={color}
              strokeWidth={focused ? 2.4 : 2}
            />
          ),
        }}
      />
      <Tabs.Screen
        name="more"
        options={{
          title: 'More',
          tabBarIcon: ({
            color,
            focused,
            size,
          }: {
            color: string;
            focused: boolean;
            size: number;
          }) => (
            <MoreHorizontal
              size={size}
              color={color}
              strokeWidth={focused ? 2.4 : 2}
            />
          ),
        }}
      />
    </Tabs>
  );
}
