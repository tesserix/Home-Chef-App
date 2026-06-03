import { StyleSheet } from 'react-native';
import { Tabs } from 'expo-router';
import {
  LayoutDashboard,
  ClipboardList,
  UtensilsCrossed,
  MoreHorizontal,
} from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';

export default function VendorTabsLayout() {
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarStyle: {
          height: 64,
          paddingBottom: 8,
          paddingTop: 6,
          backgroundColor: theme.colors.paper,
          borderTopWidth: StyleSheet.hairlineWidth,
          borderTopColor: theme.colors.mist.DEFAULT,
          elevation: 0,
        },
        tabBarActiveTintColor: theme.colors.ink.DEFAULT,
        tabBarInactiveTintColor: theme.colors.ink.muted,
        tabBarLabelStyle: {
          fontFamily: 'Inter-SemiBold',
          fontSize: 11,
          letterSpacing: 0,
          marginTop: 2,
          textAlign: 'center',
          includeFontPadding: false,
        },
      }}
    >
      <Tabs.Screen
        name="index"
        options={{
          title: 'Dashboard',
          tabBarIcon: ({ color }: { color: string }) => (
            <LayoutDashboard size={22} color={color} strokeWidth={2.2} />
          ),
        }}
      />
      <Tabs.Screen
        name="orders"
        options={{
          title: 'Orders',
          tabBarIcon: ({ color }: { color: string }) => (
            <ClipboardList size={22} color={color} strokeWidth={2.2} />
          ),
        }}
      />
      <Tabs.Screen
        name="menu"
        options={{
          title: 'Menu',
          tabBarIcon: ({ color }: { color: string }) => (
            <UtensilsCrossed size={22} color={color} strokeWidth={2.2} />
          ),
        }}
      />
      <Tabs.Screen
        name="more"
        options={{
          title: 'More',
          tabBarIcon: ({ color }: { color: string }) => (
            <MoreHorizontal size={22} color={color} strokeWidth={2.2} />
          ),
        }}
      />
    </Tabs>
  );
}
