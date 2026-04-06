import { View, Text, TouchableOpacity, ScrollView, Alert } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import {
  User2,
  DollarSign,
  Clock,
  Users2,
  UserCog,
  Settings,
  LogOut,
  ChevronRight,
} from 'lucide-react-native';
import { useAuthStore } from '../../store/auth-store';

interface NavItem {
  icon: React.ReactNode;
  label: string;
  onPress: () => void;
}

export default function MoreScreen() {
  const { logout } = useAuthStore();

  const handleLogout = () => {
    Alert.alert('Logout', 'Are you sure you want to logout?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Logout',
        style: 'destructive',
        onPress: () => {
          logout();
          router.replace('/(auth)/login');
        },
      },
    ]);
  };

  const navItems: NavItem[] = [
    {
      icon: <User2 size={20} color="#6B7280" />,
      label: 'Profile',
      onPress: () => router.push('/driver-profile'),
    },
    {
      icon: <DollarSign size={20} color="#6B7280" />,
      label: 'Earnings',
      onPress: () => router.push('/driver-earnings'),
    },
    {
      icon: <Clock size={20} color="#6B7280" />,
      label: 'History',
      onPress: () => router.push('/driver-history'),
    },
    {
      icon: <Users2 size={20} color="#6B7280" />,
      label: 'Fleet',
      onPress: () => router.push('/fleet'),
    },
    {
      icon: <UserCog size={20} color="#6B7280" />,
      label: 'Staff',
      onPress: () => router.push('/staff'),
    },
    {
      icon: <Settings size={20} color="#6B7280" />,
      label: 'Settings',
      onPress: () => router.push('/driver-settings'),
    },
  ];

  return (
    <SafeAreaView className="flex-1 bg-white">
      <ScrollView className="flex-1">
        <View className="px-6 pt-6 pb-2">
          <Text className="text-2xl font-bold text-gray-900">More</Text>
        </View>

        <View className="mt-4">
          {navItems.map((item, index) => (
            <TouchableOpacity
              key={item.label}
              onPress={item.onPress}
              className={`flex-row items-center px-6 py-4 ${
                index < navItems.length - 1 ? 'border-b border-gray-100' : ''
              }`}
            >
              <View className="w-8 items-center">{item.icon}</View>
              <Text className="flex-1 ml-3 text-base text-gray-800">{item.label}</Text>
              <ChevronRight size={18} color="#9CA3AF" />
            </TouchableOpacity>
          ))}
        </View>

        {/* Logout */}
        <View className="mt-4 border-t border-gray-100">
          <TouchableOpacity
            onPress={handleLogout}
            className="flex-row items-center px-6 py-4"
          >
            <View className="w-8 items-center">
              <LogOut size={20} color="#EF4444" />
            </View>
            <Text className="flex-1 ml-3 text-base text-red-500">Logout</Text>
          </TouchableOpacity>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
