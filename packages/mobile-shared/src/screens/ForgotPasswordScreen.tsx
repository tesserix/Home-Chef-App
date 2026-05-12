// FOUND-03: Uses @tesserix/native components for all interactive UI and text.
// API note: @tesserix/native Button uses `isLoading` not `loading`, Input uses `errorMessage` not `error`.
// T-03-06: Go API returns generic success regardless of email existence — prevents enumeration.

import React, { useState } from 'react';
import { View } from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
// FOUND-03: use @tesserix/native components — not raw RN primitives for interactive UI
import { Button, Input, Text, H1 } from '@tesserix/native';

const forgotPasswordSchema = z.object({
  email: z.string().email('Enter a valid email'),
});

type ForgotPasswordFormData = z.infer<typeof forgotPasswordSchema>;

interface ForgotPasswordScreenProps {
  onForgotPassword: (data: ForgotPasswordFormData) => Promise<void>;
  onNavigateToLogin?: () => void;
  title?: string;
}

export function ForgotPasswordScreen({
  onForgotPassword,
  onNavigateToLogin,
  title = 'Reset password',
}: ForgotPasswordScreenProps) {
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const {
    control,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ForgotPasswordFormData>({ resolver: zodResolver(forgotPasswordSchema) });

  const onSubmit = async (data: ForgotPasswordFormData) => {
    setError(null);
    try {
      await onForgotPassword(data);
      setSuccess(true);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Something went wrong. Please try again.';
      setError(msg);
    }
  };

  if (success) {
    return (
      <View className="flex-1 bg-bone px-6 pt-16">
        <H1 className="mb-4">Check your inbox</H1>
        <Text size="base" color="#4a4a47" className="mb-8">
          Check your email for a reset link
        </Text>
        {onNavigateToLogin ? (
          <Button variant="outline" onPress={onNavigateToLogin}>
            Back to sign in
          </Button>
        ) : null}
      </View>
    );
  }

  return (
    <View className="flex-1 bg-bone px-6 pt-16">
      <H1 className="mb-2">{title}</H1>
      <Text size="base" color="#7a7a76" className="mb-8">
        Enter your email address and we'll send you a reset link
      </Text>

      {error ? (
        <View className="bg-paprika-tint border border-paprika/30 rounded-lg p-3 mb-4">
          <Text size="sm" color="#c95b3e">{error}</Text>
        </View>
      ) : null}

      <Controller
        control={control}
        name="email"
        render={({ field: { onChange, onBlur, value } }) => (
          <View className="mb-6">
            <Input
              label="Email"
              placeholder="you@example.com"
              keyboardType="email-address"
              autoCapitalize="none"
              autoComplete="email"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              isInvalid={!!errors.email}
              errorMessage={errors.email?.message}
            />
          </View>
        )}
      />

      <Button
        variant="solid"
        colorScheme="primary"
        onPress={handleSubmit(onSubmit)}
        disabled={isSubmitting}
        isLoading={isSubmitting}
        fullWidth
      >
        Send reset link
      </Button>

      {onNavigateToLogin ? (
        <View className="mt-6 items-center">
          <Button variant="outline" onPress={onNavigateToLogin}>
            Back to sign in
          </Button>
        </View>
      ) : null}
    </View>
  );
}
