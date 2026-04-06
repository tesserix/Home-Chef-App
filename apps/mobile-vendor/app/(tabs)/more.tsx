import { Alert, ScrollView, Text, TouchableOpacity, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import {
  BarChart2,
  ChevronRight,
  DollarSign,
  LogOut,
  Settings,
  Star,
  User,
} from 'lucide-react-native';
import { useAuthStore } from '../../store/auth-store';

interface NavItem {
  label: string;
  route: string;
  icon: React.ReactNode;
}

export default function MoreScreen() {
  const { logout, user } = useAuthStore();

  const email = user?.email ?? '';
  const initials = email
    .split('@')[0]
    ?.slice(0, 2)
    .toUpperCase() ?? '??';

  const NAV_ITEMS: NavItem[] = [
    {
      label: 'Profile',
      route: '/profile',
      icon: <User size={20} color="#6B7280" />,
    },
    {
      label: 'Earnings',
      route: '/earnings',
      icon: <DollarSign size={20} color="#6B7280" />,
    },
    {
      label: 'Analytics',
      route: '/analytics',
      icon: <BarChart2 size={20} color="#6B7280" />,
    },
    {
      label: 'Reviews',
      route: '/reviews',
      icon: <Star size={20} color="#6B7280" />,
    },
    {
      label: 'Settings',
      route: '/settings',
      icon: <Settings size={20} color="#6B7280" />,
    },
  ];

  function handleLogout() {
    Alert.alert('Logout', 'Are you sure you want to log out?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Logout',
        style: 'destructive',
        onPress: () => {
          logout();
          router.replace('/(auth)/login' as never);
        },
      },
    ]);
  }

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      {/* User info header */}
      <View className="bg-white px-4 pt-4 pb-4 border-b border-gray-100">
        <View className="flex-row items-center gap-3">
          <View className="w-12 h-12 rounded-full bg-orange-500 items-center justify-center">
            <Text className="text-white font-bold text-base">{initials}</Text>
          </View>
          <View>
            <Text className="text-base font-semibold text-gray-900">My Account</Text>
            <Text className="text-sm text-gray-400" numberOfLines={1}>
              {email}
            </Text>
          </View>
        </View>
      </View>

      <ScrollView className="flex-1" showsVerticalScrollIndicator={false}>
        <View className="bg-white mx-4 mt-4 rounded-2xl shadow-sm overflow-hidden">
          {NAV_ITEMS.map((item, index) => (
            <TouchableOpacity
              key={item.route}
              className={`flex-row items-center px-4 py-4 ${
                index < NAV_ITEMS.length - 1 ? 'border-b border-gray-100' : ''
              }`}
              onPress={() => router.push(item.route as never)}
              activeOpacity={0.7}
            >
              <View className="w-8 h-8 rounded-lg bg-gray-100 items-center justify-center mr-3">
                {item.icon}
              </View>
              <Text className="flex-1 text-base text-gray-800 font-medium">{item.label}</Text>
              <ChevronRight size={18} color="#9CA3AF" />
            </TouchableOpacity>
          ))}
        </View>

        {/* Logout */}
        <View className="bg-white mx-4 mt-4 rounded-2xl shadow-sm overflow-hidden mb-8">
          <TouchableOpacity
            className="flex-row items-center px-4 py-4"
            onPress={handleLogout}
            activeOpacity={0.7}
          >
            <View className="w-8 h-8 rounded-lg bg-red-50 items-center justify-center mr-3">
              <LogOut size={20} color="#EF4444" />
            </View>
            <Text className="flex-1 text-base text-red-500 font-medium">Logout</Text>
          </TouchableOpacity>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
