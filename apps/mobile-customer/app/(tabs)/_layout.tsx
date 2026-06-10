import { StyleSheet, View } from 'react-native';
import { Tabs, router } from 'expo-router';
import { Home, ShoppingBag, Heart, User } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { CartBar } from '../../components/cart/CartBar';

// Matches tabBarStyle.height — the tab bar already spans to the screen bottom,
// so the cart bar sits flush on top of it (no extra safe-area inset, or a gap
// opens up between the bar and the tab bar).
const TAB_BAR_HEIGHT = 64;

export default function TabsLayout() {
  return (
    <View style={{ flex: 1 }}>
      <Tabs
        screenOptions={{
          headerShown: false,
          tabBarStyle: styles.tabBar,
          tabBarActiveTintColor: customerColors.coral.DEFAULT,
          tabBarInactiveTintColor: customerColors.charcoal.soft,
          tabBarLabelStyle: styles.tabLabel,
        }}
      >
      <Tabs.Screen
        name="index"
        options={{
          title: 'Home',
          tabBarIcon: ({ color }) => <Home size={22} color={color} />,
        }}
      />
      <Tabs.Screen
        name="orders"
        options={{
          title: 'Orders',
          tabBarIcon: ({ color }) => <ShoppingBag size={22} color={color} />,
        }}
      />
      <Tabs.Screen
        name="favorites"
        options={{
          title: 'Saved',
          tabBarIcon: ({ color }) => <Heart size={22} color={color} />,
        }}
      />
      <Tabs.Screen
        name="profile"
        options={{
          title: 'Profile',
          tabBarIcon: ({ color }) => <User size={22} color={color} />,
        }}
      />
      </Tabs>
      {/* Floating cart bar — shows on every tab when the cart has items,
          sitting just above the tab bar. Returns null when the cart is empty. */}
      <CartBar
        onPress={() => router.push('/checkout')}
        bottomOffset={TAB_BAR_HEIGHT}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  // White bar with a top shadow — floating over the canvas, no persimmon.
  // Shadow lives only on the bar; active = coral, inactive = charcoal-soft.
  tabBar: {
    backgroundColor: customerColors.canvas,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
    height: 64,
    paddingBottom: 8,
    paddingTop: 6,
    // Top elevation shadow (iOS)
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: -2 },
    shadowOpacity: 0.06,
    shadowRadius: 8,
    elevation: 8,
  },
  tabLabel: {
    fontFamily: 'Inter',
    fontSize: 11,
    letterSpacing: 0.1,
  },
});
