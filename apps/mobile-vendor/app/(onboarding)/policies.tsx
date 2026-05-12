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
    <View className="flex-1 bg-bone">
      <View className="px-6 pt-4">
        <View className="h-1.5 rounded-full bg-mist mb-6">
          <View className="h-1.5 rounded-full bg-herb" style={{ width: `${(5 / 6) * 100}%` }} />
        </View>

        <Text className="font-display text-2xl font-semibold text-ink mb-1">Policies</Text>
        <Text className="text-sm text-ink-muted mb-4">Review and accept terms</Text>
      </View>

      <ScrollView className="flex-1 px-6">
        <View className="bg-paper rounded-xl p-4 mb-5">
          <Text className="text-sm text-ink-soft leading-6">{VENDOR_TERMS_TEXT}</Text>
        </View>

        <TouchableOpacity
          className="flex-row items-center mb-6"
          onPress={() => setAcceptedTerms((prev: boolean) => !prev)}
        >
          <View
            className={`w-5 h-5 rounded border-2 mr-3 items-center justify-center ${
              acceptedTerms ? 'bg-herb border-herb' : 'border-mist-strong'
            }`}
          >
            {acceptedTerms && <Text className="text-paper text-xs font-medium">✓</Text>}
          </View>
          <Text className="text-sm text-ink-soft flex-1">
            I accept the terms and conditions
          </Text>
        </TouchableOpacity>

        <Text className="text-sm font-medium text-ink-soft mb-3">Cancellation Policy *</Text>
        {CANCELLATION_POLICY_OPTIONS.map((option) => (
          <TouchableOpacity
            key={option.value}
            className={`flex-row items-center border rounded-xl px-4 py-3 mb-3 ${
              cancellationPolicy === option.value
                ? 'bg-herb-tint border-herb'
                : 'border-mist'
            }`}
            onPress={() => setCancellationPolicy(option.value)}
          >
            <View
              className={`w-5 h-5 rounded-full border-2 mr-3 items-center justify-center ${
                cancellationPolicy === option.value
                  ? 'border-herb'
                  : 'border-mist-strong'
              }`}
            >
              {cancellationPolicy === option.value && (
                <View className="w-2.5 h-2.5 rounded-full bg-herb" />
              )}
            </View>
            <Text
              className={`text-sm flex-1 ${
                cancellationPolicy === option.value ? 'text-herb font-medium' : 'text-ink-soft'
              }`}
            >
              {option.label}
            </Text>
          </TouchableOpacity>
        ))}

        <TouchableOpacity
          className={`rounded-xl py-4 items-center mb-8 mt-2 ${
            acceptedTerms && cancellationPolicy ? 'bg-herb' : 'bg-mist-strong'
          }`}
          onPress={onNext}
          disabled={!acceptedTerms || !cancellationPolicy}
        >
          <Text
            className={`font-semibold text-base ${
              acceptedTerms && cancellationPolicy ? 'text-paper' : 'text-ink-muted'
            }`}
          >
            Next
          </Text>
        </TouchableOpacity>
      </ScrollView>
    </View>
  );
}
