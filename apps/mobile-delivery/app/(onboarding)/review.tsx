import { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { router } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { api } from '../../lib/api';
import { useDriverOnboardingStore } from '../../store/onboarding-store';

interface SummaryRowProps {
  label: string;
  value: string;
}

function SummaryRow({ label, value }: SummaryRowProps) {
  return (
    <View className="flex-row justify-between py-3 border-b border-mist">
      <Text className="text-ink-muted text-sm">{label}</Text>
      <Text className="text-ink text-sm font-medium flex-1 text-right ml-4">{value}</Text>
    </View>
  );
}

function maskAccountNumber(account: string): string {
  if (!account || account.length < 4) return '****';
  return `****${account.slice(-4)}`;
}

export default function ReviewScreen() {
  const [termsAccepted, setTermsAccepted] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { personalInfo, vehicleDetails, documents, payoutDetails, subscriptionInfo, reset } =
    useDriverOnboardingStore();

  const docsUploaded =
    [documents.drivingLicenseUri, documents.idProofUri].filter(Boolean).length;
  const totalDocsRequired = 2;

  const payoutSummary =
    payoutDetails.payoutMethod === 'bank'
      ? `Bank Account: ${maskAccountNumber(payoutDetails.bankAccountNumber)}`
      : `UPI: ${payoutDetails.upiId}`;

  const handleSubmit = async () => {
    if (!termsAccepted) {
      Alert.alert('Terms Required', 'Please accept the Terms of Service and Privacy Policy.');
      return;
    }
    setIsSubmitting(true);
    try {
      await api.post('/driver/onboarding/submit', { termsAccepted: true });
      reset();
      router.replace('/(onboarding)/pending');
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : 'Failed to submit application. Please try again.';
      Alert.alert('Submission Error', message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <SafeAreaView className="flex-1 bg-bone" edges={['bottom']}>
      <ScrollView className="flex-1 px-6" keyboardShouldPersistTaps="handled">
        {/* Progress bar */}
        <View className="mt-4 mb-6 h-1 bg-mist rounded-full">
          <View className="h-1 bg-herb rounded-full w-full" />
        </View>

        <Text className="font-display text-2xl font-semibold text-ink mb-2">Review Your Application</Text>
        <Text className="text-ink-muted mb-6">
          Please review your details before submitting
        </Text>

        {/* Personal Info Section */}
        <View className="bg-paper rounded-xl px-4 mb-4">
          <Text className="text-base font-semibold text-ink pt-4 pb-2">Personal Info</Text>
          <SummaryRow label="City" value={personalInfo.city || '—'} />
          <SummaryRow
            label="Vehicle Type"
            value={personalInfo.vehicleType ? personalInfo.vehicleType.charAt(0).toUpperCase() + personalInfo.vehicleType.slice(1) : '—'}
          />
          <SummaryRow label="Emergency Contact" value={personalInfo.emergencyContact || '—'} />
          <SummaryRow label="Emergency Phone" value={personalInfo.emergencyPhone || '—'} />
          {personalInfo.dateOfBirth && (
            <SummaryRow label="Date of Birth" value={personalInfo.dateOfBirth} />
          )}
        </View>

        {/* Vehicle Details Section */}
        <View className="bg-paper rounded-xl px-4 mb-4">
          <Text className="text-base font-semibold text-ink pt-4 pb-2">Vehicle Details</Text>
          <SummaryRow label="Make & Model" value={`${vehicleDetails.vehicleMake} ${vehicleDetails.vehicleModel}`} />
          <SummaryRow label="Year & Color" value={`${vehicleDetails.vehicleYear} — ${vehicleDetails.vehicleColor}`} />
          <SummaryRow label="Registration" value={vehicleDetails.vehicleNumber || '—'} />
          <SummaryRow label="License Number" value={vehicleDetails.licenseNumber || '—'} />
        </View>

        {/* Documents Section */}
        <View className="bg-paper rounded-xl px-4 mb-4">
          <Text className="text-base font-semibold text-ink pt-4 pb-2">Documents</Text>
          <SummaryRow
            label="Documents Uploaded"
            value={`${docsUploaded}/${totalDocsRequired} required${documents.vehicleRcUri ? ' + RC' : ''}`}
          />
        </View>

        {/* Payout Section */}
        <View className="bg-paper rounded-xl px-4 mb-4">
          <Text className="text-base font-semibold text-ink pt-4 pb-2">Payout Method</Text>
          <SummaryRow label="Method" value={payoutSummary} />
        </View>

        {/* Subscription Section */}
        <View className="bg-paper rounded-xl px-4 mb-6">
          <Text className="text-base font-semibold text-ink pt-4 pb-2">Subscription Plan</Text>
          <SummaryRow label="Selected Plan" value={subscriptionInfo.planName || '—'} />
        </View>

        {/* Terms Checkbox */}
        <TouchableOpacity
          onPress={() => setTermsAccepted((prev: boolean) => !prev)}
          className="flex-row items-start gap-3 mb-8"
        >
          <View
            className={`w-5 h-5 mt-0.5 rounded border-2 items-center justify-center ${
              termsAccepted ? 'bg-herb border-herb' : 'border-mist-strong bg-bone'
            }`}
          >
            {termsAccepted && <Text className="text-paper text-xs font-medium">✓</Text>}
          </View>
          <Text className="flex-1 text-ink-soft text-sm leading-5">
            I accept the{' '}
            <Text className="text-herb font-medium">Terms of Service</Text>
            {' '}and{' '}
            <Text className="text-herb font-medium">Privacy Policy</Text>
          </Text>
        </TouchableOpacity>
      </ScrollView>

      {/* Submit Button */}
      <View className="px-6 py-4 border-t border-mist">
        <TouchableOpacity
          onPress={handleSubmit}
          disabled={!termsAccepted || isSubmitting}
          className={`w-full py-4 rounded-xl items-center ${
            termsAccepted && !isSubmitting ? 'bg-herb' : 'bg-mist-strong'
          }`}
        >
          {isSubmitting ? (
            <ActivityIndicator color="white" />
          ) : (
            <Text className="text-paper font-semibold text-base">Submit Application</Text>
          )}
        </TouchableOpacity>
      </View>
    </SafeAreaView>
  );
}
