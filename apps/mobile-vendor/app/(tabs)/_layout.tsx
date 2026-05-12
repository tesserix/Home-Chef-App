import { Tabs } from 'expo-router';
import { LayoutDashboard, ClipboardList, UtensilsCrossed, MoreHorizontal } from 'lucide-react-native';

export default function VendorTabsLayout() {
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarStyle: { height: 64, paddingBottom: 8 },
        tabBarActiveTintColor: '#3e6b3c',
        tabBarInactiveTintColor: '#7a7a76',
      }}
    >
      <Tabs.Screen
        name="index"
        options={{
          title: 'Dashboard',
          tabBarIcon: ({ color }: { color: string }) => (
            <LayoutDashboard size={22} color={color} />
          ),
        }}
      />
      <Tabs.Screen
        name="orders"
        options={{
          title: 'Orders',
          tabBarIcon: ({ color }: { color: string }) => (
            <ClipboardList size={22} color={color} />
          ),
        }}
      />
      <Tabs.Screen
        name="menu"
        options={{
          title: 'Menu',
          tabBarIcon: ({ color }: { color: string }) => (
            <UtensilsCrossed size={22} color={color} />
          ),
        }}
      />
      <Tabs.Screen
        name="more"
        options={{
          title: 'More',
          tabBarIcon: ({ color }: { color: string }) => (
            <MoreHorizontal size={22} color={color} />
          ),
        }}
      />
    </Tabs>
  );
}
