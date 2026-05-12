import { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  ScrollView,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { router } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { api } from '../../lib/api';
import { useDriverOnboardingStore } from '../../store/onboarding-store';

const ifscRegex = /^[A-Z]{4}0[A-Z0-9]{6}$/;
const upiRegex = /@/;

const payoutSchema = z
  .object({
    payoutMethod: z.enum(['bank', 'upi']),
    bankAccountNumber: z.string().optional(),
    confirmAccountNumber: z.string().optional(),
    bankIFSC: z.string().optional(),
    bankName: z.string().optional(),
    upiId: z.string().optional(),
  })
  .superRefine((data, ctx) => {
    if (data.payoutMethod === 'bank') {
      if (!data.bankAccountNumber || data.bankAccountNumber.length === 0) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Account number is required', path: ['bankAccountNumber'] });
      }
      if (!data.confirmAccountNumber || data.confirmAccountNumber.length === 0) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Please confirm account number', path: ['confirmAccountNumber'] });
      } else if (data.bankAccountNumber !== data.confirmAccountNumber) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Account numbers do not match', path: ['confirmAccountNumber'] });
      }
      if (!data.bankIFSC || !ifscRegex.test(data.bankIFSC)) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Enter a valid IFSC code (e.g. HDFC0001234)', path: ['bankIFSC'] });
      }
      if (!data.bankName || data.bankName.length === 0) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Bank name is required', path: ['bankName'] });
      }
    } else if (data.payoutMethod === 'upi') {
      if (!data.upiId || !upiRegex.test(data.upiId)) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Enter a valid UPI ID (must contain @)', path: ['upiId'] });
      }
    }
  });

type PayoutFormData = z.infer<typeof payoutSchema>;

export default function PayoutScreen() {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { payoutDetails, updatePayoutDetails, setStep } = useDriverOnboardingStore();
  const [selectedMethod, setSelectedMethod] = useState<'bank' | 'upi'>(
    payoutDetails.payoutMethod
  );

  const {
    control,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<PayoutFormData>({
    resolver: zodResolver(payoutSchema),
    defaultValues: {
      payoutMethod: payoutDetails.payoutMethod,
      bankAccountNumber: payoutDetails.bankAccountNumber,
      confirmAccountNumber: payoutDetails.bankAccountNumber,
      bankIFSC: payoutDetails.bankIFSC,
      bankName: '',
      upiId: payoutDetails.upiId,
    },
  });

  const onSelectMethod = (method: 'bank' | 'upi') => {
    setSelectedMethod(method);
    setValue('payoutMethod', method);
  };

  const onSubmit = async (data: PayoutFormData) => {
    setIsSubmitting(true);
    try {
      const payload =
        data.payoutMethod === 'bank'
          ? {
              payoutMethod: 'bank' as const,
              bankAccountNumber: data.bankAccountNumber,
              bankIFSC: data.bankIFSC,
            }
          : {
              payoutMethod: 'upi' as const,
              upiId: data.upiId,
            };

      await api.post('/driver/onboarding/payout', payload);
      updatePayoutDetails({
        payoutMethod: data.payoutMethod,
        bankAccountNumber: data.payoutMethod === 'bank' ? (data.bankAccountNumber ?? '') : '',
        bankIFSC: data.payoutMethod === 'bank' ? (data.bankIFSC ?? '') : '',
        upiId: data.payoutMethod === 'upi' ? (data.upiId ?? '') : '',
      });
      setStep(5);
      router.push('/(onboarding)/subscription');
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : 'Failed to save payout details. Please try again.';
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
          <View className="h-1 bg-herb rounded-full" style={{ width: '66.67%' }} />
        </View>

        <Text className="font-display text-2xl font-semibold text-ink mb-2">Payout Details</Text>
        <Text className="text-ink-muted mb-6">
          Choose how you would like to receive your earnings
        </Text>

        {/* Method selector */}
        <View className="flex-row mb-6 border border-mist rounded-xl overflow-hidden">
          <TouchableOpacity
            onPress={() => onSelectMethod('bank')}
            className={`flex-1 py-3 items-center ${selectedMethod === 'bank' ? 'bg-herb' : 'bg-bone'}`}
          >
            <Text
              className={`font-semibold ${selectedMethod === 'bank' ? 'text-paper' : 'text-ink-soft'}`}
            >
              Bank Account
            </Text>
          </TouchableOpacity>
          <TouchableOpacity
            onPress={() => onSelectMethod('upi')}
            className={`flex-1 py-3 items-center ${selectedMethod === 'upi' ? 'bg-herb' : 'bg-bone'}`}
          >
            <Text
              className={`font-semibold ${selectedMethod === 'upi' ? 'text-paper' : 'text-ink-soft'}`}
            >
              UPI
            </Text>
          </TouchableOpacity>
        </View>

        {/* Hidden payoutMethod controller */}
        <Controller
          control={control}
          name="payoutMethod"
          render={() => <View />}
        />

        {selectedMethod === 'bank' ? (
          <>
            {/* Account Number */}
            <View className="mb-4">
              <Text className="text-sm font-medium text-ink-soft mb-1">
                Account Number <Text className="text-paprika">*</Text>
              </Text>
              <Controller
                control={control}
                name="bankAccountNumber"
                render={({ field: { onChange, onBlur, value } }) => (
                  <TextInput
                    className={`border rounded-lg px-4 py-3 text-ink ${errors.bankAccountNumber ? 'border-paprika' : 'border-mist-strong'}`}
                    placeholder="Enter account number"
                    keyboardType="numeric"
                    secureTextEntry
                    onBlur={onBlur}
                    onChangeText={onChange}
                    value={value ?? ''}
                  />
                )}
              />
              {errors.bankAccountNumber && (
                <Text className="text-paprika text-sm mt-1">
                  {errors.bankAccountNumber.message}
                </Text>
              )}
            </View>

            {/* Confirm Account Number */}
            <View className="mb-4">
              <Text className="text-sm font-medium text-ink-soft mb-1">
                Confirm Account Number <Text className="text-paprika">*</Text>
              </Text>
              <Controller
                control={control}
                name="confirmAccountNumber"
                render={({ field: { onChange, onBlur, value } }) => (
                  <TextInput
                    className={`border rounded-lg px-4 py-3 text-ink ${errors.confirmAccountNumber ? 'border-paprika' : 'border-mist-strong'}`}
                    placeholder="Re-enter account number"
                    keyboardType="numeric"
                    onBlur={onBlur}
                    onChangeText={onChange}
                    value={value ?? ''}
                  />
                )}
              />
              {errors.confirmAccountNumber && (
                <Text className="text-paprika text-sm mt-1">
                  {errors.confirmAccountNumber.message}
                </Text>
              )}
            </View>

            {/* IFSC */}
            <View className="mb-4">
              <Text className="text-sm font-medium text-ink-soft mb-1">
                IFSC Code <Text className="text-paprika">*</Text>
              </Text>
              <Controller
                control={control}
                name="bankIFSC"
                render={({ field: { onChange, onBlur, value } }) => (
                  <TextInput
                    className={`border rounded-lg px-4 py-3 text-ink ${errors.bankIFSC ? 'border-paprika' : 'border-mist-strong'}`}
                    placeholder="e.g. HDFC0001234"
                    autoCapitalize="characters"
                    onBlur={onBlur}
                    onChangeText={onChange}
                    value={value ?? ''}
                  />
                )}
              />
              {errors.bankIFSC && (
                <Text className="text-paprika text-sm mt-1">{errors.bankIFSC.message}</Text>
              )}
            </View>

            {/* Bank Name */}
            <View className="mb-4">
              <Text className="text-sm font-medium text-ink-soft mb-1">
                Bank Name <Text className="text-paprika">*</Text>
              </Text>
              <Controller
                control={control}
                name="bankName"
                render={({ field: { onChange, onBlur, value } }) => (
                  <TextInput
                    className={`border rounded-lg px-4 py-3 text-ink ${errors.bankName ? 'border-paprika' : 'border-mist-strong'}`}
                    placeholder="e.g. HDFC Bank"
                    onBlur={onBlur}
                    onChangeText={onChange}
                    value={value ?? ''}
                  />
                )}
              />
              {errors.bankName && (
                <Text className="text-paprika text-sm mt-1">{errors.bankName.message}</Text>
              )}
            </View>
          </>
        ) : (
          <View className="mb-4">
            <Text className="text-sm font-medium text-ink-soft mb-1">
              UPI ID <Text className="text-paprika">*</Text>
            </Text>
            <Controller
              control={control}
              name="upiId"
              render={({ field: { onChange, onBlur, value } }) => (
                <TextInput
                  className={`border rounded-lg px-4 py-3 text-ink ${errors.upiId ? 'border-paprika' : 'border-mist-strong'}`}
                  placeholder="e.g. yourname@upi"
                  autoCapitalize="none"
                  keyboardType="email-address"
                  onBlur={onBlur}
                  onChangeText={onChange}
                  value={value ?? ''}
                />
              )}
            />
            {errors.upiId && (
              <Text className="text-paprika text-sm mt-1">{errors.upiId.message}</Text>
            )}
          </View>
        )}

        {/* Security note */}
        <View className="bg-herb-tint border border-herb-tint rounded-lg px-4 py-3 mb-8">
          <Text className="text-herb text-sm">
            Your payout details are encrypted and never stored on this device. They are used solely for processing your earnings.
          </Text>
        </View>
      </ScrollView>

      {/* Next Button */}
      <View className="px-6 py-4 border-t border-mist">
        <TouchableOpacity
          onPress={handleSubmit(onSubmit)}
          disabled={isSubmitting}
          className={`w-full py-4 rounded-xl items-center ${isSubmitting ? 'bg-herb-soft' : 'bg-herb'}`}
        >
          {isSubmitting ? (
            <ActivityIndicator color="white" />
          ) : (
            <Text className="text-paper font-semibold text-base">Next</Text>
          )}
        </TouchableOpacity>
      </View>
    </SafeAreaView>
  );
}
