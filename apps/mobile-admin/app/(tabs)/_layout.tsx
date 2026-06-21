import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { Tabs } from 'expo-router';
import {
  LayoutDashboard,
  ChefHat,
  ClipboardList,
  MoreHorizontal,
} from 'lucide-react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import type { BottomTabBarProps } from 'expo-router/tabs';
import { theme } from '@homechef/mobile-shared/theme';

// Custom tab bar — same approach as the vendor app (TouchableOpacity over
// Pressable to avoid the iOS flex:1 quirk; full-width centered labels).
function CustomTabBar({ state, descriptors, navigation }: BottomTabBarProps) {
  const insets = useSafeAreaInsets();

  return (
    <View style={[styles.bar, { paddingBottom: Math.max(insets.bottom, 8) }]}>
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
            accessibilityLabel={options.tabBarAccessibilityLabel ?? `${label} tab`}
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
                style={[styles.label, isFocused && styles.labelActive]}
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
    shadowColor: theme.colors.ink.DEFAULT,
    shadowOffset: { width: 0, height: -2 },
    shadowOpacity: 0.06,
    shadowRadius: 12,
    elevation: 8,
  },
  itemTouchable: { flex: 1 },
  itemInner: {
    width: '100%',
    alignItems: 'center',
    justifyContent: 'flex-start',
    paddingTop: 10,
    paddingBottom: 6,
    minHeight: 56,
  },
  label: {
    alignSelf: 'stretch',
    textAlign: 'center',
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    letterSpacing: 0.1,
    marginTop: 4,
    color: theme.colors.ink.muted,
    includeFontPadding: false,
  },
  labelActive: { color: theme.colors.ink.DEFAULT },
});

export default function AdminTabsLayout() {
  return (
    <Tabs
      tabBar={(props) => <CustomTabBar {...props} />}
      screenOptions={{ headerShown: false }}
    >
      <Tabs.Screen
        name="index"
        options={{
          title: 'Dashboard',
          tabBarIcon: ({ color, focused, size }) => (
            <LayoutDashboard size={size} color={String(color)} strokeWidth={focused ? 2.4 : 2} />
          ),
        }}
      />
      <Tabs.Screen
        name="chefs"
        options={{
          title: 'Chefs',
          tabBarIcon: ({ color, focused, size }) => (
            <ChefHat size={size} color={String(color)} strokeWidth={focused ? 2.4 : 2} />
          ),
        }}
      />
      <Tabs.Screen
        name="orders"
        options={{
          title: 'Orders',
          tabBarIcon: ({ color, focused, size }) => (
            <ClipboardList size={size} color={String(color)} strokeWidth={focused ? 2.4 : 2} />
          ),
        }}
      />
      <Tabs.Screen
        name="more"
        options={{
          title: 'More',
          tabBarIcon: ({ color, focused, size }) => (
            <MoreHorizontal size={size} color={String(color)} strokeWidth={focused ? 2.4 : 2} />
          ),
        }}
      />
    </Tabs>
  );
}
