import { View, Text, Pressable, ScrollView } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useAuthStore } from '../../store/auth-store';

interface NavItem {
  label: string;
  route: string;
  icon: string;
}

const NAV_ITEMS: NavItem[] = [
  { label: 'Profile', route: '/profile', icon: '👤' },
  { label: 'Earnings', route: '/earnings', icon: '💰' },
  { label: 'Analytics', route: '/analytics', icon: '📊' },
  { label: 'Reviews', route: '/reviews', icon: '⭐' },
  { label: 'Settings', route: '/settings', icon: '⚙️' },
];

export default function MoreScreen() {
  const { logout } = useAuthStore();

  function handleLogout(): void {
    logout();
    router.replace('/(auth)/login');
  }

  return (
    <SafeAreaView className="flex-1 bg-white">
      <View className="px-6 pt-4 pb-2">
        <Text className="text-2xl font-bold text-gray-900">More</Text>
      </View>

      <ScrollView className="flex-1">
        <View className="px-6 py-2">
          {NAV_ITEMS.map((item) => (
            <Pressable
              key={item.route}
              className="flex-row items-center py-4 border-b border-gray-100"
              onPress={() => router.push(item.route as never)}
            >
              <Text className="text-xl mr-4">{item.icon}</Text>
              <Text className="flex-1 text-base text-gray-800 font-medium">{item.label}</Text>
              <Text className="text-gray-400 text-lg">›</Text>
            </Pressable>
          ))}

          <Pressable
            className="flex-row items-center py-4 mt-4"
            onPress={handleLogout}
          >
            <Text className="text-xl mr-4">🚪</Text>
            <Text className="flex-1 text-base text-red-600 font-medium">Logout</Text>
          </Pressable>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
