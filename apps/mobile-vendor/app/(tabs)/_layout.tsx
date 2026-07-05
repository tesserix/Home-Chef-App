import { Tabs } from 'expo-router';
import {
  LayoutDashboard,
  ClipboardList,
  UtensilsCrossed,
  MoreHorizontal,
} from 'lucide-react-native';
import { Dock } from '../../components/navigation/Dock';

// The floating dock (components/navigation/Dock) is the app's tab bar. It
// renders its own icons from the route name, so the `tabBarIcon` options
// below are unused by the custom `tabBar` — they're kept for screen
// registration parity and cost nothing (Dock ignores them).
export default function VendorTabsLayout() {
  return (
    <Tabs
      tabBar={(props) => <Dock {...props} />}
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
