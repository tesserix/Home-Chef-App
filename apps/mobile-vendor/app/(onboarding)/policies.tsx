import { View, Text, ScrollView, TouchableOpacity, Alert } from 'react-native';
import { useState } from 'react';
import { router } from 'expo-router';
import { useVendorOnboardingStore } from '../../store/onboarding-store';
import {
  VENDOR_TERMS_TEXT,
  CANCELLATION_POLICY_OPTIONS,
  type CancellationPolicy,
} from '../../constants/terms';

export default function PoliciesScreen() {
  const { policies, updatePolicies, setStep } = useVendorOnboardingStore();

  const [acceptedTerms, setAcceptedTerms] = useState<boolean>(policies.acceptedTerms);
  const [cancellationPolicy, setCancellationPolicy] = useState<CancellationPolicy | ''>(
    (policies.cancellationPolicy as CancellationPolicy) || '',
  );

  function onNext(): void {
    if (!acceptedTerms) {
      Alert.alert('Terms Required', 'Please accept the terms and conditions to continue.');
      return;
    }
    if (!cancellationPolicy) {
      Alert.alert('Policy Required', 'Please select a cancellation policy.');
      return;
    }
    updatePolicies({ acceptedTerms, cancellationPolicy });
    setStep(6);
    router.push('/(onboarding)/review');
  }

  return (
    <View className="flex-1 bg-white">
      <View className="px-6 pt-4">
        <View className="h-1.5 rounded-full bg-gray-200 mb-6">
          <View className="h-1.5 rounded-full bg-orange-500" style={{ width: `${(5 / 6) * 100}%` }} />
        </View>

        <Text className="text-2xl font-bold text-gray-900 mb-1">Policies</Text>
        <Text className="text-sm text-gray-500 mb-4">Review and accept terms</Text>
      </View>

      <ScrollView className="flex-1 px-6">
        <View className="bg-gray-50 rounded-xl p-4 mb-5">
          <Text className="text-sm text-gray-700 leading-6">{VENDOR_TERMS_TEXT}</Text>
        </View>

        <TouchableOpacity
          className="flex-row items-center mb-6"
          onPress={() => setAcceptedTerms((prev: boolean) => !prev)}
        >
          <View
            className={`w-5 h-5 rounded border-2 mr-3 items-center justify-center ${
              acceptedTerms ? 'bg-orange-500 border-orange-500' : 'border-gray-400'
            }`}
          >
            {acceptedTerms && <Text className="text-white text-xs font-bold">✓</Text>}
          </View>
          <Text className="text-sm text-gray-700 flex-1">
            I accept the terms and conditions
          </Text>
        </TouchableOpacity>

        <Text className="text-sm font-medium text-gray-700 mb-3">Cancellation Policy *</Text>
        {CANCELLATION_POLICY_OPTIONS.map((option) => (
          <TouchableOpacity
            key={option.value}
            className={`flex-row items-center border rounded-xl px-4 py-3 mb-3 ${
              cancellationPolicy === option.value
                ? 'bg-orange-50 border-orange-500'
                : 'border-gray-200'
            }`}
            onPress={() => setCancellationPolicy(option.value)}
          >
            <View
              className={`w-5 h-5 rounded-full border-2 mr-3 items-center justify-center ${
                cancellationPolicy === option.value
                  ? 'border-orange-500'
                  : 'border-gray-400'
              }`}
            >
              {cancellationPolicy === option.value && (
                <View className="w-2.5 h-2.5 rounded-full bg-orange-500" />
              )}
            </View>
            <Text
              className={`text-sm flex-1 ${
                cancellationPolicy === option.value ? 'text-orange-700 font-medium' : 'text-gray-700'
              }`}
            >
              {option.label}
            </Text>
          </TouchableOpacity>
        ))}

        <TouchableOpacity
          className={`rounded-xl py-4 items-center mb-8 mt-2 ${
            acceptedTerms && cancellationPolicy ? 'bg-orange-500' : 'bg-gray-300'
          }`}
          onPress={onNext}
          disabled={!acceptedTerms || !cancellationPolicy}
        >
          <Text
            className={`font-semibold text-base ${
              acceptedTerms && cancellationPolicy ? 'text-white' : 'text-gray-500'
            }`}
          >
            Next
          </Text>
        </TouchableOpacity>
      </ScrollView>
    </View>
  );
}
