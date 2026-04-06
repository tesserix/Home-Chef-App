import { View, Text, ScrollView, TouchableOpacity, Alert, ActivityIndicator } from 'react-native';
import { useState } from 'react';
import { router } from 'expo-router';
import { api } from '../../lib/api';
import { useVendorOnboardingStore } from '../../store/onboarding-store';

export default function ReviewScreen() {
  const store = useVendorOnboardingStore();
  const [submitting, setSubmitting] = useState(false);

  const { personalInfo, kitchenDetails, operations, documents, policies } = store;

  const openDays = Object.entries(operations.operatingHours)
    .filter(([, hours]) => !hours.closed)
    .map(([day]) => day.slice(0, 3))
    .join(', ');

  async function onSubmit(): Promise<void> {
    setSubmitting(true);
    try {
      await api.post('/chef/onboarding', {
        personalInfo,
        kitchenDetails,
        operations,
        policies,
      });
      store.reset();
      router.replace('/(onboarding)/pending');
    } catch (error: unknown) {
      const message =
        error instanceof Error ? error.message : 'Submission failed. Please try again.';
      Alert.alert('Submission Error', message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <ScrollView className="flex-1 bg-white">
      <View className="px-6 pt-4 pb-8">
        <View className="h-1.5 rounded-full bg-gray-200 mb-6">
          <View className="h-1.5 rounded-full bg-orange-500" style={{ width: '100%' }} />
        </View>

        <Text className="text-2xl font-bold text-gray-900 mb-1">Review Application</Text>
        <Text className="text-sm text-gray-500 mb-6">Confirm your details before submitting</Text>

        <View className="bg-gray-50 rounded-xl p-4 mb-4">
          <Text className="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-3">
            Personal Information
          </Text>
          <View className="mb-2">
            <Text className="text-xs text-gray-500">Full Name</Text>
            <Text className="text-sm text-gray-900 font-medium">{personalInfo.fullName}</Text>
          </View>
          <View className="mb-2">
            <Text className="text-xs text-gray-500">Phone</Text>
            <Text className="text-sm text-gray-900 font-medium">{personalInfo.phone}</Text>
          </View>
          <View>
            <Text className="text-xs text-gray-500">Email</Text>
            <Text className="text-sm text-gray-900 font-medium">{personalInfo.email}</Text>
          </View>
        </View>

        <View className="bg-gray-50 rounded-xl p-4 mb-4">
          <Text className="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-3">
            Kitchen Details
          </Text>
          <View className="mb-2">
            <Text className="text-xs text-gray-500">Business Name</Text>
            <Text className="text-sm text-gray-900 font-medium">{kitchenDetails.businessName}</Text>
          </View>
          <View className="mb-2">
            <Text className="text-xs text-gray-500">Cuisines</Text>
            <Text className="text-sm text-gray-900 font-medium">{kitchenDetails.cuisines.join(', ')}</Text>
          </View>
          <View>
            <Text className="text-xs text-gray-500">Description</Text>
            <Text className="text-sm text-gray-900" numberOfLines={3}>{kitchenDetails.description}</Text>
          </View>
        </View>

        <View className="bg-gray-50 rounded-xl p-4 mb-4">
          <Text className="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-3">
            Operations
          </Text>
          <View className="mb-2">
            <Text className="text-xs text-gray-500">Open Days</Text>
            <Text className="text-sm text-gray-900 font-medium capitalize">{openDays}</Text>
          </View>
          <View className="mb-2">
            <Text className="text-xs text-gray-500">Prep Time</Text>
            <Text className="text-sm text-gray-900 font-medium">{operations.prepTime}</Text>
          </View>
          <View>
            <Text className="text-xs text-gray-500">Service Radius</Text>
            <Text className="text-sm text-gray-900 font-medium">{operations.serviceRadius} km</Text>
          </View>
        </View>

        <View className="bg-gray-50 rounded-xl p-4 mb-4">
          <Text className="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-3">
            Documents
          </Text>
          <View className="mb-2">
            <Text className="text-xs text-gray-500">ID Proof</Text>
            <Text className="text-sm text-green-600 font-medium">
              {documents.idProofUri ? 'Uploaded' : 'Not uploaded'}
            </Text>
          </View>
          <View>
            <Text className="text-xs text-gray-500">FSSAI License</Text>
            <Text className="text-sm text-green-600 font-medium">
              {documents.fssaiUri ? 'Uploaded' : 'Not uploaded'}
            </Text>
          </View>
        </View>

        <View className="bg-gray-50 rounded-xl p-4 mb-6">
          <Text className="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-3">
            Policies
          </Text>
          <View className="mb-2">
            <Text className="text-xs text-gray-500">Terms Accepted</Text>
            <Text className="text-sm text-green-600 font-medium">Yes</Text>
          </View>
          <View>
            <Text className="text-xs text-gray-500">Cancellation Policy</Text>
            <Text className="text-sm text-gray-900 font-medium">{policies.cancellationPolicy}</Text>
          </View>
        </View>

        <TouchableOpacity
          className={`rounded-xl py-4 items-center ${submitting ? 'bg-orange-300' : 'bg-orange-500'}`}
          onPress={onSubmit}
          disabled={submitting}
        >
          {submitting ? (
            <ActivityIndicator color="white" />
          ) : (
            <Text className="text-white font-semibold text-base">Submit Application</Text>
          )}
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}
