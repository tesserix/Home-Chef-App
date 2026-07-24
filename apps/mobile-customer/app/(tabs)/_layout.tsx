import { Tabs } from 'expo-router';
import { Dock } from '../../components/navigation/Dock';

// Tab navigation renders through the floating Dock (components/navigation/
// Dock.tsx) — a detached rounded bar above the home indicator, not an
// edge-to-edge tab bar. The dock is absolutely positioned so scenes get the
// full screen height; screens pad their scroll bottom by useDockClearance().
// The cart lives IN the dock as a dynamic fifth slot (DockCartPill), so the
// old full-width CartBar is gone from the tab layer.

export default function TabsLayout() {
  return (
    <Tabs
      tabBar={(props) => <Dock {...props} />}
      screenOptions={{ headerShown: false }}
    >
      <Tabs.Screen name="index" options={{ title: 'Home' }} />
      <Tabs.Screen name="orders" options={{ title: 'Orders' }} />
      <Tabs.Screen name="plans" options={{ title: 'Plans' }} />
      <Tabs.Screen name="favorites" options={{ title: 'Saved' }} />
      <Tabs.Screen name="profile" options={{ title: 'Profile' }} />
    </Tabs>
  );
}
