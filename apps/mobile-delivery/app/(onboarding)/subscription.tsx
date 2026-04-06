import { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { useQuery } from '@tanstack/react-query';
import { AxiosResponse } from 'axios';
import { router } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { api } from '../../lib/api';
import { useDriverOnboardingStore } from '../../store/onboarding-store';

interface SubscriptionPlan {
  id: string;
  name: string;
  price: number;
  description: string;
  features: string[];
  maxDeliveries?: number;
  recommended?: boolean;
}

interface PlansResponse {
  data: SubscriptionPlan[];
}

function PlanCardSkeleton() {
  return (
    <View className="border border-gray-200 rounded-xl p-4 mb-4">
      <View className="h-5 bg-gray-200 rounded w-1/2 mb-2" />
      <View className="h-4 bg-gray-200 rounded w-1/4 mb-3" />
      <View className="h-3 bg-gray-200 rounded w-full mb-2" />
      <View className="h-3 bg-gray-200 rounded w-4/5" />
    </View>
  );
}

export default function SubscriptionScreen() {
  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(
    useDriverOnboardingStore.getState().subscriptionInfo.selectedPlanId
  );
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { updateSubscriptionInfo, setStep } = useDriverOnboardingStore();

  const {
    data: plansResponse,
    isLoading,
    isError,
    refetch,
  } = useQuery<AxiosResponse<PlansResponse>>({
    queryKey: ['driver', 'subscription', 'plans'],
    queryFn: () => api.get<PlansResponse>('/driver/subscription/plans'),
    staleTime: 5 * 60 * 1000,
  });

  const plans: SubscriptionPlan[] = plansResponse?.data?.data ?? [];

  const handleSelectPlan = async () => {
    if (!selectedPlanId) {
      Alert.alert('Select a Plan', 'Please select a subscription plan to continue.');
      return;
    }
    setIsSubmitting(true);
    try {
      await api.post('/driver/subscription/plan', { planId: selectedPlanId });
      const selectedPlan = plans.find((p: SubscriptionPlan) => p.id === selectedPlanId);
      updateSubscriptionInfo({
        selectedPlanId,
        planName: selectedPlan?.name ?? '',
      });
      setStep(6);
      router.push('/(onboarding)/review');
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : 'Failed to select plan. Please try again.';
      Alert.alert('Error', message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <SafeAreaView className="flex-1 bg-white" edges={['bottom']}>
      <ScrollView className="flex-1 px-6" keyboardShouldPersistTaps="handled">
        {/* Progress bar */}
        <View className="mt-4 mb-6 h-1 bg-gray-200 rounded-full">
          <View className="h-1 bg-orange-500 rounded-full" style={{ width: '83.33%' }} />
        </View>

        <Text className="text-2xl font-bold text-gray-900 mb-2">Choose Your Plan</Text>
        <Text className="text-gray-500 mb-6">
          Select a subscription plan that works best for you
        </Text>

        {isLoading ? (
          <>
            <PlanCardSkeleton />
            <PlanCardSkeleton />
          </>
        ) : isError ? (
          <View className="items-center py-12">
            <Text className="text-gray-600 mb-4">Failed to load subscription plans.</Text>
            <TouchableOpacity
              onPress={() => refetch()}
              className="bg-orange-500 px-6 py-3 rounded-lg"
            >
              <Text className="text-white font-semibold">Retry</Text>
            </TouchableOpacity>
          </View>
        ) : (
          plans.map((plan: SubscriptionPlan) => {
            const isSelected = selectedPlanId === plan.id;
            return (
              <TouchableOpacity
                key={plan.id}
                onPress={() => setSelectedPlanId(plan.id)}
                className={`border rounded-xl p-4 mb-4 ${
                  isSelected ? 'border-orange-500 bg-orange-50' : 'border-gray-200 bg-white'
                } ${plan.recommended ? 'border-2' : ''}`}
              >
                <View className="flex-row justify-between items-start mb-1">
                  <Text className="text-lg font-bold text-gray-900">{plan.name}</Text>
                  <View className="flex-row items-center gap-2">
                    {plan.recommended && (
                      <View className="bg-orange-100 px-2 py-1 rounded">
                        <Text className="text-orange-600 text-xs font-semibold">Recommended</Text>
                      </View>
                    )}
                    {isSelected && (
                      <View className="w-6 h-6 bg-orange-500 rounded-full items-center justify-center">
                        <Text className="text-white text-xs font-bold">✓</Text>
                      </View>
                    )}
                  </View>
                </View>

                <Text className="text-2xl font-bold text-orange-500 mb-1">
                  ₹{plan.price}
                  <Text className="text-base font-normal text-gray-500">/month</Text>
                </Text>

                <Text className="text-gray-600 text-sm mb-3">{plan.description}</Text>

                {plan.maxDeliveries !== undefined && (
                  <Text className="text-sm text-gray-500 mb-2">
                    Up to {plan.maxDeliveries} deliveries/month
                  </Text>
                )}

                <View className="gap-1">
                  {plan.features.map((feature: string, idx: number) => (
                    <View key={idx} className="flex-row items-center gap-2">
                      <Text className="text-orange-500">•</Text>
                      <Text className="text-gray-700 text-sm">{feature}</Text>
                    </View>
                  ))}
                </View>
              </TouchableOpacity>
            );
          })
        )}

        <View className="mb-8" />
      </ScrollView>

      {/* Select Plan Button */}
      <View className="px-6 py-4 border-t border-gray-100">
        <TouchableOpacity
          onPress={handleSelectPlan}
          disabled={!selectedPlanId || isSubmitting || isLoading}
          className={`w-full py-4 rounded-xl items-center ${
            selectedPlanId && !isSubmitting && !isLoading ? 'bg-orange-500' : 'bg-gray-300'
          }`}
        >
          {isSubmitting ? (
            <ActivityIndicator color="white" />
          ) : (
            <Text className="text-white font-semibold text-base">Select Plan</Text>
          )}
        </TouchableOpacity>
      </View>
    </SafeAreaView>
  );
}
