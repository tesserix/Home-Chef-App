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
      icon: <User size={20} color="#7a7a76" />,
    },
    {
      label: 'Earnings',
      route: '/earnings',
      icon: <DollarSign size={20} color="#7a7a76" />,
    },
    {
      label: 'Analytics',
      route: '/analytics',
      icon: <BarChart2 size={20} color="#7a7a76" />,
    },
    {
      label: 'Reviews',
      route: '/reviews',
      icon: <Star size={20} color="#7a7a76" />,
    },
    {
      label: 'Settings',
      route: '/settings',
      icon: <Settings size={20} color="#7a7a76" />,
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
    <SafeAreaView className="flex-1 bg-paper">
      {/* User info header */}
      <View className="bg-bone px-4 pt-4 pb-4 border-b border-mist">
        <View className="flex-row items-center gap-3">
          <View className="w-12 h-12 rounded-full bg-herb items-center justify-center">
            <Text className="text-paper font-semibold text-base">{initials}</Text>
          </View>
          <View>
            <Text className="text-base font-semibold text-ink">My Account</Text>
            <Text className="text-sm text-ink-muted" numberOfLines={1}>
              {email}
            </Text>
          </View>
        </View>
      </View>

      <ScrollView className="flex-1" showsVerticalScrollIndicator={false}>
        <View className="bg-bone mx-4 mt-4 rounded-2xl shadow-sm overflow-hidden">
          {NAV_ITEMS.map((item, index) => (
            <TouchableOpacity
              key={item.route}
              className={`flex-row items-center px-4 py-4 ${
                index < NAV_ITEMS.length - 1 ? 'border-b border-mist' : ''
              }`}
              onPress={() => router.push(item.route as never)}
              activeOpacity={0.7}
            >
              <View className="w-8 h-8 rounded-lg bg-mist items-center justify-center mr-3">
                {item.icon}
              </View>
              <Text className="flex-1 text-base text-ink font-medium">{item.label}</Text>
              <ChevronRight size={18} color="#7a7a76" />
            </TouchableOpacity>
          ))}
        </View>

        {/* Logout */}
        <View className="bg-bone mx-4 mt-4 rounded-2xl shadow-sm overflow-hidden mb-8">
          <TouchableOpacity
            className="flex-row items-center px-4 py-4"
            onPress={handleLogout}
            activeOpacity={0.7}
          >
            <View className="w-8 h-8 rounded-lg bg-paprika-tint items-center justify-center mr-3">
              <LogOut size={20} color="#c95b3e" />
            </View>
            <Text className="flex-1 text-base text-paprika font-medium">Logout</Text>
          </TouchableOpacity>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
