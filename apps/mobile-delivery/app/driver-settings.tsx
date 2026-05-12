import { useState } from 'react';
import { Alert, ScrollView, Switch, Text, TouchableOpacity, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useMutation } from '@tanstack/react-query';
import Constants from 'expo-constants';
import { api } from '../lib/api';

interface NotificationSettings {
  newDelivery: boolean;
  earningsPayout: boolean;
}

interface AppSettings {
  defaultOnlineStatus: boolean;
}

function useUpdateSettings() {
  return useMutation({
    mutationFn: (settings: Partial<NotificationSettings & AppSettings>) =>
      api.put('/delivery/settings', settings),
  });
}

function SectionHeader({ title }: { title: string }) {
  return (
    <Text className="text-xs font-semibold text-ink-muted uppercase tracking-wider px-4 mb-1 mt-6">
      {title}
    </Text>
  );
}

function SettingRow({
  label,
  description,
  right,
}: {
  label: string;
  description?: string;
  right: React.ReactNode;
}) {
  return (
    <View className="flex-row items-center px-4 py-3 bg-bone border-b border-mist">
      <View className="flex-1 mr-3">
        <Text className="text-base text-ink">{label}</Text>
        {description && <Text className="text-xs text-ink-muted mt-0.5">{description}</Text>}
      </View>
      {right}
    </View>
  );
}

function ActionRow({
  label,
  description,
  onPress,
  destructive,
}: {
  label: string;
  description?: string;
  onPress: () => void;
  destructive?: boolean;
}) {
  return (
    <TouchableOpacity
      onPress={onPress}
      className="flex-row items-center px-4 py-3 bg-bone border-b border-mist"
      activeOpacity={0.7}
    >
      <View className="flex-1">
        <Text className={`text-base ${destructive ? 'text-paprika' : 'text-ink'}`}>
          {label}
        </Text>
        {description && <Text className="text-xs text-ink-muted mt-0.5">{description}</Text>}
      </View>
    </TouchableOpacity>
  );
}

export default function DriverSettingsScreen() {
  const [newDelivery, setNewDelivery] = useState(true);
  const [earningsPayout, setEarningsPayout] = useState(true);
  const [defaultOnline, setDefaultOnline] = useState(false);

  const updateMutation = useUpdateSettings();

  function handleToggleNewDelivery(value: boolean) {
    setNewDelivery(value);
    updateMutation.mutate({ newDelivery: value });
  }

  function handleToggleEarningsPayout(value: boolean) {
    setEarningsPayout(value);
    updateMutation.mutate({ earningsPayout: value });
  }

  function handleToggleDefaultOnline(value: boolean) {
    setDefaultOnline(value);
  }

  function handleChangePassword() {
    router.push('/(auth)/forgot-password');
  }

  function handleViewSubscription() {
    Alert.alert(
      'Subscription',
      'Visit the web portal to manage your subscription.',
      [{ text: 'OK' }],
    );
  }

  function handleDeleteAccount() {
    Alert.alert(
      'Delete Account',
      'Contact support at support@homechef.in to request account deletion.',
      [{ text: 'OK' }],
    );
  }

  const appVersion =
    (Constants.expoConfig?.version as string | undefined) ?? '1.0.0';

  return (
    <SafeAreaView className="flex-1 bg-paper">
      <ScrollView className="flex-1" contentContainerStyle={{ paddingBottom: 32 }}>
        <View className="px-4 pt-4 pb-2">
          <Text className="font-display text-2xl font-semibold text-ink">Settings</Text>
        </View>

        {/* Notifications */}
        <SectionHeader title="Notifications" />
        <SettingRow
          label="New Delivery Notifications"
          description="Get notified when a new delivery is available"
          right={
            <Switch
              value={newDelivery}
              onValueChange={handleToggleNewDelivery}
              trackColor={{ false: '#d4d3ce', true: '#558257' }}
              thumbColor="white"
            />
          }
        />
        <SettingRow
          label="Earnings Payout Notifications"
          description="Get notified when a payout is processed"
          right={
            <Switch
              value={earningsPayout}
              onValueChange={handleToggleEarningsPayout}
              trackColor={{ false: '#d4d3ce', true: '#558257' }}
              thumbColor="white"
            />
          }
        />

        {/* Availability */}
        <SectionHeader title="Availability" />
        <SettingRow
          label="Default Online Status"
          description="Automatically go online when app opens"
          right={
            <Switch
              value={defaultOnline}
              onValueChange={handleToggleDefaultOnline}
              trackColor={{ false: '#d4d3ce', true: '#3e6b3c' }}
              thumbColor="white"
            />
          }
        />

        {/* Account */}
        <SectionHeader title="Account" />
        <ActionRow
          label="Change Password"
          onPress={handleChangePassword}
        />
        <ActionRow
          label="View Subscription Plan"
          onPress={handleViewSubscription}
        />
        <ActionRow
          label="Delete Account"
          destructive
          onPress={handleDeleteAccount}
        />
        <View className="flex-row items-center px-4 py-3 bg-bone">
          <Text className="flex-1 text-base text-ink-muted">App Version</Text>
          <Text className="text-sm text-ink-muted">{appVersion}</Text>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
