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
        fullName: personalInfo.fullName,
        phone: personalInfo.phone,
        email: personalInfo.email,
        businessName: kitchenDetails.businessName,
        description: kitchenDetails.description,
        cuisines: kitchenDetails.cuisines,
        kitchenAddress: {
          line1: kitchenDetails.addressLine1,
          line2: kitchenDetails.addressLine2,
          city: kitchenDetails.city,
          state: kitchenDetails.state,
          postalCode: kitchenDetails.postalCode,
        },
        prepTime: operations.prepTime,
        serviceRadius: operations.serviceRadius,
        operatingHours: operations.operatingHours,
        acceptedTerms: policies.acceptedTerms,
      });
      store.reset();
      router.replace('/(onboarding)/pending');
    } catch (error: unknown) {
      const serverError = (error as { response?: { data?: { error?: string } } } | null)?.response?.data?.error;
      const fallback = error instanceof Error ? error.message : 'Submission failed. Please try again.';
      Alert.alert('Submission Error', serverError ?? fallback);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <ScrollView className="flex-1 bg-bone">
      <View className="px-6 pt-4 pb-8">
        <View className="h-1.5 rounded-full bg-mist mb-6">
          <View className="h-1.5 rounded-full bg-herb" style={{ width: '100%' }} />
        </View>

        <Text className="font-display text-2xl font-semibold text-ink mb-1">Review Application</Text>
        <Text className="text-sm text-ink-muted mb-6">Confirm your details before submitting</Text>

        <View className="bg-paper rounded-xl p-4 mb-4">
          <Text className="text-xs font-semibold text-ink-muted uppercase tracking-wide mb-3">
            Personal Information
          </Text>
          <View className="mb-2">
            <Text className="text-xs text-ink-muted">Full Name</Text>
            <Text className="text-sm text-ink font-medium">{personalInfo.fullName}</Text>
          </View>
          <View className="mb-2">
            <Text className="text-xs text-ink-muted">Phone</Text>
            <Text className="text-sm text-ink font-medium">{personalInfo.phone}</Text>
          </View>
          <View>
            <Text className="text-xs text-ink-muted">Email</Text>
            <Text className="text-sm text-ink font-medium">{personalInfo.email}</Text>
          </View>
        </View>

        <View className="bg-paper rounded-xl p-4 mb-4">
          <Text className="text-xs font-semibold text-ink-muted uppercase tracking-wide mb-3">
            Kitchen Details
          </Text>
          <View className="mb-2">
            <Text className="text-xs text-ink-muted">Business Name</Text>
            <Text className="text-sm text-ink font-medium">{kitchenDetails.businessName}</Text>
          </View>
          <View className="mb-2">
            <Text className="text-xs text-ink-muted">Cuisines</Text>
            <Text className="text-sm text-ink font-medium">{kitchenDetails.cuisines.join(', ')}</Text>
          </View>
          <View className="mb-2">
            <Text className="text-xs text-ink-muted">Description</Text>
            <Text className="text-sm text-ink" numberOfLines={3}>{kitchenDetails.description}</Text>
          </View>
          <View>
            <Text className="text-xs text-ink-muted">Address</Text>
            <Text className="text-sm text-ink font-medium">
              {[
                kitchenDetails.addressLine1,
                kitchenDetails.addressLine2,
                kitchenDetails.city,
                kitchenDetails.state,
                kitchenDetails.postalCode,
              ].filter(Boolean).join(', ')}
            </Text>
          </View>
        </View>

        <View className="bg-paper rounded-xl p-4 mb-4">
          <Text className="text-xs font-semibold text-ink-muted uppercase tracking-wide mb-3">
            Operations
          </Text>
          <View className="mb-2">
            <Text className="text-xs text-ink-muted">Open Days</Text>
            <Text className="text-sm text-ink font-medium capitalize">{openDays}</Text>
          </View>
          <View className="mb-2">
            <Text className="text-xs text-ink-muted">Prep Time</Text>
            <Text className="text-sm text-ink font-medium">{operations.prepTime}</Text>
          </View>
          <View>
            <Text className="text-xs text-ink-muted">Service Radius</Text>
            <Text className="text-sm text-ink font-medium">{operations.serviceRadius} km</Text>
          </View>
        </View>

        <View className="bg-paper rounded-xl p-4 mb-4">
          <Text className="text-xs font-semibold text-ink-muted uppercase tracking-wide mb-3">
            Documents
          </Text>
          <View className="mb-2">
            <Text className="text-xs text-ink-muted">ID Proof</Text>
            <Text className="text-sm text-herb font-medium">
              {documents.idProofUri ? 'Uploaded' : 'Not uploaded'}
            </Text>
          </View>
          <View>
            <Text className="text-xs text-ink-muted">FSSAI License</Text>
            <Text className="text-sm text-herb font-medium">
              {documents.fssaiUri ? 'Uploaded' : 'Not uploaded'}
            </Text>
          </View>
        </View>

        <View className="bg-paper rounded-xl p-4 mb-6">
          <Text className="text-xs font-semibold text-ink-muted uppercase tracking-wide mb-3">
            Policies
          </Text>
          <View className="mb-2">
            <Text className="text-xs text-ink-muted">Terms Accepted</Text>
            <Text className="text-sm text-herb font-medium">Yes</Text>
          </View>
          <View>
            <Text className="text-xs text-ink-muted">Cancellation Policy</Text>
            <Text className="text-sm text-ink font-medium">{policies.cancellationPolicy}</Text>
          </View>
        </View>

        <TouchableOpacity
          className={`rounded-xl py-4 items-center ${submitting ? 'bg-herb-soft' : 'bg-herb'}`}
          onPress={onSubmit}
          disabled={submitting}
        >
          {submitting ? (
            <ActivityIndicator color="white" />
          ) : (
            <Text className="text-paper font-semibold text-base">Submit Application</Text>
          )}
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}
