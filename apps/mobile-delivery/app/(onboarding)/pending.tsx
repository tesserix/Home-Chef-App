import { useEffect } from 'react';
import { View, Text, TouchableOpacity, ActivityIndicator, Alert } from 'react-native';
import { useQuery } from '@tanstack/react-query';
import { AxiosResponse } from 'axios';
import { router } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { api } from '../../lib/api';
import { useAuthStore } from '../../store/auth-store';

interface DriverOnboardingStatus {
  step: number;
  status: 'not_started' | 'in_progress' | 'pending_review' | 'approved' | 'rejected';
  onboardingComplete: boolean;
  verificationStatus: string;
  profile: {
    city?: string;
    rejectionReason?: string;
  } | null;
}

export default function PendingScreen() {
  const { logout } = useAuthStore();

  const { data, isLoading } = useQuery<AxiosResponse<DriverOnboardingStatus>>({
    queryKey: ['driver', 'onboarding', 'status'],
    queryFn: () =>
      api.get<DriverOnboardingStatus>('/driver/onboarding/status'),
    // Poll every 30 seconds (foreground only)
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
    staleTime: 0,
  });

  const status = data?.data;

  useEffect(() => {
    if (!status) return;
    if (status.onboardingComplete || status.verificationStatus === 'approved') {
      router.replace('/(tabs)');
    }
  }, [status]);

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

  const handleReapply = () => {
    router.replace('/(onboarding)/personal');
  };

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-bone items-center justify-center">
        <ActivityIndicator size="large" color="#3e6b3c" />
      </SafeAreaView>
    );
  }

  const isRejected = status?.status === 'rejected';
  const rejectionReason = status?.profile?.rejectionReason;

  return (
    <SafeAreaView className="flex-1 bg-bone">
      {/* Logout button */}
      <View className="flex-row justify-end px-6 pt-4">
        <TouchableOpacity onPress={handleLogout}>
          <Text className="text-ink-muted font-medium">Logout</Text>
        </TouchableOpacity>
      </View>

      <View className="flex-1 items-center justify-center px-8">
        {isRejected ? (
          <>
            {/* Rejected state */}
            <View className="w-20 h-20 bg-paprika-tint rounded-full items-center justify-center mb-6">
              <Text className="text-4xl">✗</Text>
            </View>
            <Text className="font-display text-2xl font-semibold text-ink text-center mb-3">
              Application Not Approved
            </Text>
            <Text className="text-ink-muted text-center mb-4">
              Unfortunately your application was not approved at this time.
            </Text>
            {rejectionReason ? (
              <View className="bg-paprika-tint border border-paprika/30 rounded-xl px-4 py-3 w-full mb-6">
                <Text className="text-paprika text-sm font-medium mb-1">Reason:</Text>
                <Text className="text-paprika text-sm">{rejectionReason}</Text>
              </View>
            ) : null}
            <TouchableOpacity
              onPress={handleReapply}
              className="w-full bg-herb py-4 rounded-xl items-center"
            >
              <Text className="text-paper font-semibold text-base">Reapply</Text>
            </TouchableOpacity>
          </>
        ) : (
          <>
            {/* Pending review state */}
            <View className="w-20 h-20 bg-herb-tint rounded-full items-center justify-center mb-6">
              <ActivityIndicator size="large" color="#3e6b3c" />
            </View>
            <Text className="font-display text-2xl font-semibold text-ink text-center mb-3">
              Application Under Review
            </Text>
            <Text className="text-ink-muted text-center mb-4">
              Your application has been submitted and is being reviewed by our team.
            </Text>
            <View className="bg-herb-tint border border-herb-tint rounded-xl px-4 py-3 w-full mb-4">
              <Text className="text-herb text-sm text-center">
                Estimated review time: 24–48 hours
              </Text>
            </View>
            <Text className="text-ink-muted text-sm text-center">
              We'll notify you once your application is approved. This page checks for updates automatically every 30 seconds.
            </Text>
          </>
        )}
      </View>
    </SafeAreaView>
  );
}
