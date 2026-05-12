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
import { stopTracking } from '../../lib/background-location';

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
        onPress: async () => {
          // Stop background GPS before clearing auth so we don't keep
          // posting location with a stale token (or worse, post under
          // a freshly logged-in different driver's session).
          try {
            await stopTracking();
          } catch {
            // best-effort; proceed with logout regardless
          }
          logout();
          router.replace('/(auth)/login');
        },
      },
    ]);
  };

  const navItems: NavItem[] = [
    {
      icon: <User2 size={20} color="#7a7a76" />,
      label: 'Profile',
      onPress: () => router.push('/driver-profile'),
    },
    {
      icon: <DollarSign size={20} color="#7a7a76" />,
      label: 'Earnings',
      onPress: () => router.push('/driver-earnings'),
    },
    {
      icon: <Clock size={20} color="#7a7a76" />,
      label: 'History',
      onPress: () => router.push('/driver-history'),
    },
    {
      icon: <Users2 size={20} color="#7a7a76" />,
      label: 'Fleet',
      onPress: () => router.push('/fleet'),
    },
    {
      icon: <UserCog size={20} color="#7a7a76" />,
      label: 'Staff',
      onPress: () => router.push('/staff'),
    },
    {
      icon: <Settings size={20} color="#7a7a76" />,
      label: 'Settings',
      onPress: () => router.push('/driver-settings'),
    },
  ];

  return (
    <SafeAreaView className="flex-1 bg-bone">
      <ScrollView className="flex-1">
        <View className="px-6 pt-6 pb-2">
          <Text className="font-display text-2xl font-semibold text-ink">More</Text>
        </View>

        <View className="mt-4">
          {navItems.map((item, index) => (
            <TouchableOpacity
              key={item.label}
              onPress={item.onPress}
              className={`flex-row items-center px-6 py-4 ${
                index < navItems.length - 1 ? 'border-b border-mist' : ''
              }`}
            >
              <View className="w-8 items-center">{item.icon}</View>
              <Text className="flex-1 ml-3 text-base text-ink">{item.label}</Text>
              <ChevronRight size={18} color="#7a7a76" />
            </TouchableOpacity>
          ))}
        </View>

        {/* Logout */}
        <View className="mt-4 border-t border-mist">
          <TouchableOpacity
            onPress={handleLogout}
            className="flex-row items-center px-6 py-4"
          >
            <View className="w-8 items-center">
              <LogOut size={20} color="#c95b3e" />
            </View>
            <Text className="flex-1 ml-3 text-base text-paprika">Logout</Text>
          </TouchableOpacity>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
