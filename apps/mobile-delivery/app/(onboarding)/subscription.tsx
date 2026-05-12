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
    <View className="border border-mist rounded-xl p-4 mb-4">
      <View className="h-5 bg-mist rounded w-1/2 mb-2" />
      <View className="h-4 bg-mist rounded w-1/4 mb-3" />
      <View className="h-3 bg-mist rounded w-full mb-2" />
      <View className="h-3 bg-mist rounded w-4/5" />
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
    <SafeAreaView className="flex-1 bg-bone" edges={['bottom']}>
      <ScrollView className="flex-1 px-6" keyboardShouldPersistTaps="handled">
        {/* Progress bar */}
        <View className="mt-4 mb-6 h-1 bg-mist rounded-full">
          <View className="h-1 bg-herb rounded-full" style={{ width: '83.33%' }} />
        </View>

        <Text className="font-display text-2xl font-semibold text-ink mb-2">Choose Your Plan</Text>
        <Text className="text-ink-muted mb-6">
          Select a subscription plan that works best for you
        </Text>

        {isLoading ? (
          <>
            <PlanCardSkeleton />
            <PlanCardSkeleton />
          </>
        ) : isError ? (
          <View className="items-center py-12">
            <Text className="text-ink-soft mb-4">Failed to load subscription plans.</Text>
            <TouchableOpacity
              onPress={() => refetch()}
              className="bg-herb px-6 py-3 rounded-lg"
            >
              <Text className="text-paper font-semibold">Retry</Text>
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
                  isSelected ? 'border-herb bg-herb-tint' : 'border-mist bg-bone'
                } ${plan.recommended ? 'border-2' : ''}`}
              >
                <View className="flex-row justify-between items-start mb-1">
                  <Text className="text-lg font-medium text-ink">{plan.name}</Text>
                  <View className="flex-row items-center gap-2">
                    {plan.recommended && (
                      <View className="bg-herb-tint px-2 py-1 rounded">
                        <Text className="text-herb text-xs font-semibold">Recommended</Text>
                      </View>
                    )}
                    {isSelected && (
                      <View className="w-6 h-6 bg-herb rounded-full items-center justify-center">
                        <Text className="text-paper text-xs font-medium">✓</Text>
                      </View>
                    )}
                  </View>
                </View>

                <Text className="font-display text-2xl font-semibold text-herb mb-1">
                  ₹{plan.price}
                  <Text className="text-base font-normal text-ink-muted">/month</Text>
                </Text>

                <Text className="text-ink-soft text-sm mb-3">{plan.description}</Text>

                {plan.maxDeliveries !== undefined && (
                  <Text className="text-sm text-ink-muted mb-2">
                    Up to {plan.maxDeliveries} deliveries/month
                  </Text>
                )}

                <View className="gap-1">
                  {plan.features.map((feature: string, idx: number) => (
                    <View key={idx} className="flex-row items-center gap-2">
                      <Text className="text-herb">•</Text>
                      <Text className="text-ink-soft text-sm">{feature}</Text>
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
      <View className="px-6 py-4 border-t border-mist">
        <TouchableOpacity
          onPress={handleSelectPlan}
          disabled={!selectedPlanId || isSubmitting || isLoading}
          className={`w-full py-4 rounded-xl items-center ${
            selectedPlanId && !isSubmitting && !isLoading ? 'bg-herb' : 'bg-mist-strong'
          }`}
        >
          {isSubmitting ? (
            <ActivityIndicator color="white" />
          ) : (
            <Text className="text-paper font-semibold text-base">Select Plan</Text>
          )}
        </TouchableOpacity>
      </View>
    </SafeAreaView>
  );
}
