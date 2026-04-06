import { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  ScrollView,
  Switch,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import { api } from '../lib/api';

interface NotificationPrefs {
  newOrderNotifications: boolean;
  payoutNotifications: boolean;
  reviewNotifications: boolean;
}

interface ChefSettings {
  notificationPrefs: NotificationPrefs;
  acceptingOrders: boolean;
}

function useChefSettings() {
  return useQuery<ChefSettings>({
    queryKey: ['chef', 'settings'],
    queryFn: () => api.get<ChefSettings>('/chef/settings').then((r) => r.data),
    staleTime: 60_000,
  });
}

function useUpdateSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: Partial<ChefSettings>) => api.put('/chef/settings', payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef', 'settings'] });
    },
  });
}

export default function SettingsScreen() {
  const { data, isLoading, isError } = useChefSettings();
  const updateMutation = useUpdateSettings();

  const [notificationPrefs, setNotificationPrefs] = useState<NotificationPrefs>({
    newOrderNotifications: true,
    payoutNotifications: true,
    reviewNotifications: true,
  });
  const [acceptingOrders, setAcceptingOrders] = useState(true);

  useEffect(() => {
    if (data) {
      setNotificationPrefs(data.notificationPrefs);
      setAcceptingOrders(data.acceptingOrders);
    }
  }, [data]);

  function handleNotificationToggle(key: keyof NotificationPrefs, value: boolean) {
    const updated = { ...notificationPrefs, [key]: value };
    setNotificationPrefs(updated);
    updateMutation.mutate({ notificationPrefs: updated });
  }

  function handleAcceptingOrdersToggle(value: boolean) {
    setAcceptingOrders(value);
    updateMutation.mutate({ acceptingOrders: value });
  }

  function handleDeleteAccount() {
    Alert.alert(
      'Delete Account',
      'To delete your account, please contact our support team. This action cannot be undone.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Contact Support',
          onPress: () => Alert.alert('Support', 'Please email support@homechef.app to delete your account.'),
        },
      ],
    );
  }

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50 items-center justify-center">
        <ActivityIndicator size="large" color="#FF6B35" />
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50 items-center justify-center px-6">
        <Text className="text-gray-500 text-base mb-4">Failed to load settings</Text>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      {/* Header */}
      <View className="flex-row items-center px-4 pt-2 pb-3 bg-white border-b border-gray-100">
        <TouchableOpacity onPress={() => router.back()} activeOpacity={0.7} className="mr-3">
          <ChevronLeft size={24} color="#374151" />
        </TouchableOpacity>
        <Text className="text-lg font-semibold text-gray-900">Settings</Text>
      </View>

      <ScrollView
        className="flex-1"
        contentContainerStyle={{ padding: 16, paddingBottom: 40 }}
        showsVerticalScrollIndicator={false}
      >
        {/* Notification Preferences */}
        <View className="bg-white rounded-2xl shadow-sm mb-4 overflow-hidden">
          <View className="px-4 py-3 border-b border-gray-100">
            <Text className="text-sm font-semibold text-gray-700">Notification Preferences</Text>
          </View>

          <View className="flex-row items-center justify-between px-4 py-4 border-b border-gray-100">
            <View className="flex-1 mr-4">
              <Text className="text-base text-gray-900">New Order Notifications</Text>
              <Text className="text-xs text-gray-400 mt-0.5">
                Get notified when a new order arrives
              </Text>
            </View>
            <Switch
              value={notificationPrefs.newOrderNotifications}
              onValueChange={(v) => handleNotificationToggle('newOrderNotifications', v)}
              trackColor={{ false: '#D1D5DB', true: '#FB923C' }}
              thumbColor={notificationPrefs.newOrderNotifications ? '#EA580C' : '#9CA3AF'}
              disabled={updateMutation.isPending}
            />
          </View>

          <View className="flex-row items-center justify-between px-4 py-4 border-b border-gray-100">
            <View className="flex-1 mr-4">
              <Text className="text-base text-gray-900">Payout Notifications</Text>
              <Text className="text-xs text-gray-400 mt-0.5">
                Get notified when payouts are processed
              </Text>
            </View>
            <Switch
              value={notificationPrefs.payoutNotifications}
              onValueChange={(v) => handleNotificationToggle('payoutNotifications', v)}
              trackColor={{ false: '#D1D5DB', true: '#FB923C' }}
              thumbColor={notificationPrefs.payoutNotifications ? '#EA580C' : '#9CA3AF'}
              disabled={updateMutation.isPending}
            />
          </View>

          <View className="flex-row items-center justify-between px-4 py-4">
            <View className="flex-1 mr-4">
              <Text className="text-base text-gray-900">Review Notifications</Text>
              <Text className="text-xs text-gray-400 mt-0.5">
                Get notified when customers leave reviews
              </Text>
            </View>
            <Switch
              value={notificationPrefs.reviewNotifications}
              onValueChange={(v) => handleNotificationToggle('reviewNotifications', v)}
              trackColor={{ false: '#D1D5DB', true: '#FB923C' }}
              thumbColor={notificationPrefs.reviewNotifications ? '#EA580C' : '#9CA3AF'}
              disabled={updateMutation.isPending}
            />
          </View>
        </View>

        {/* Availability */}
        <View className="bg-white rounded-2xl shadow-sm mb-4 overflow-hidden">
          <View className="px-4 py-3 border-b border-gray-100">
            <Text className="text-sm font-semibold text-gray-700">Availability</Text>
          </View>
          <View className="flex-row items-center justify-between px-4 py-4">
            <View className="flex-1 mr-4">
              <Text className="text-base text-gray-900">Accepting Orders</Text>
              <Text className="text-xs text-gray-400 mt-0.5">
                Toggle to start or pause accepting orders
              </Text>
            </View>
            <Switch
              value={acceptingOrders}
              onValueChange={handleAcceptingOrdersToggle}
              trackColor={{ false: '#D1D5DB', true: '#FB923C' }}
              thumbColor={acceptingOrders ? '#EA580C' : '#9CA3AF'}
              disabled={updateMutation.isPending}
            />
          </View>
        </View>

        {/* Account */}
        <View className="bg-white rounded-2xl shadow-sm overflow-hidden">
          <View className="px-4 py-3 border-b border-gray-100">
            <Text className="text-sm font-semibold text-gray-700">Account</Text>
          </View>

          <TouchableOpacity
            className="flex-row items-center justify-between px-4 py-4 border-b border-gray-100"
            activeOpacity={0.7}
            onPress={() => router.push('/(auth)/forgot-password' as never)}
          >
            <Text className="text-base text-gray-900">Change Password</Text>
            <Text className="text-gray-400 text-lg">›</Text>
          </TouchableOpacity>

          <TouchableOpacity
            className="px-4 py-4"
            activeOpacity={0.7}
            onPress={handleDeleteAccount}
          >
            <Text className="text-base text-red-500">Delete Account</Text>
          </TouchableOpacity>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
