import { View, Text, TouchableOpacity, ActivityIndicator } from 'react-native';
import { useEffect } from 'react';
import { router } from 'expo-router';
import { useQuery } from '@tanstack/react-query';
import { SafeAreaView } from 'react-native-safe-area-context';
import { api } from '../../lib/api';
import { useAuthStore } from '../../store/auth-store';

interface OnboardingStatusResponse {
  status: 'not_started' | 'in_progress' | 'pending_review' | 'submitted' | 'verified' | 'rejected';
  completed: boolean;
  step: number;
  chefId: string | null;
  profile: { rejectionReason?: string } | null;
}

export default function PendingScreen() {
  const { logout } = useAuthStore();

  const { data, isLoading } = useQuery({
    queryKey: ['chef', 'onboarding', 'status'],
    queryFn: () => api.get<OnboardingStatusResponse>('/chef/onboarding/status'),
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
  });

  const status = data?.data?.status;

  useEffect(() => {
    if (status === 'verified') {
      router.replace('/(tabs)');
    }
  }, [status]);

  function handleLogout(): void {
    logout();
    router.replace('/(auth)/login');
  }

  function handleReapply(): void {
    router.replace('/(onboarding)/personal-info');
  }

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-bone items-center justify-center">
        <ActivityIndicator color="#C2410C" size="large" />
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-bone">
      <View className="flex-row justify-end px-6 pt-2">
        <TouchableOpacity onPress={handleLogout} className="py-2 px-4">
          <Text className="text-sm text-ink-muted font-medium">Logout</Text>
        </TouchableOpacity>
      </View>

      <View className="flex-1 items-center justify-center px-8">
        {status === 'rejected' ? (
          <>
            <View className="w-20 h-20 rounded-full bg-paprika-tint items-center justify-center mb-6">
              <Text className="text-4xl">✗</Text>
            </View>
            <Text className="font-display text-2xl font-semibold text-ink text-center mb-3">
              Application Not Approved
            </Text>
            {data?.data?.profile?.rejectionReason ? (
              <Text className="text-sm text-ink-soft text-center mb-2">
                Reason: {data.data.profile.rejectionReason}
              </Text>
            ) : null}
            <Text className="text-sm text-ink-muted text-center mb-8">
              Please review the feedback and resubmit your application.
            </Text>
            <TouchableOpacity
              className="bg-herb rounded-xl py-4 px-8 items-center w-full"
              onPress={handleReapply}
            >
              <Text className="text-paper font-semibold text-base">Reapply</Text>
            </TouchableOpacity>
          </>
        ) : (
          <>
            <View className="w-20 h-20 rounded-full bg-herb-tint items-center justify-center mb-6">
              <Text className="text-4xl">✓</Text>
            </View>
            <Text className="font-display text-2xl font-semibold text-ink text-center mb-3">
              Application Submitted!
            </Text>
            <Text className="text-sm text-ink-muted text-center mb-6">
              Our team will review your application within 24-48 hours. We will notify you once the review is complete.
            </Text>
            <View className="bg-herb-tint rounded-xl px-4 py-3 w-full">
              <Text className="text-xs text-herb text-center">
                Status: {status === 'pending_review' ? 'Pending Review' : 'Submitted'}
              </Text>
            </View>
          </>
        )}
      </View>
    </SafeAreaView>
  );
}
