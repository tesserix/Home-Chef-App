// FOUND-03: Uses @tesserix/native components for all interactive UI and text.
// API note: @tesserix/native Button uses `isLoading` not `loading`, Input uses `errorMessage` not `error`.

import React, { useState } from 'react';
import { View } from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
// FOUND-03: use @tesserix/native components — not raw RN primitives for interactive UI
import { Button, Input, Text, H1 } from '@tesserix/native';

const registerSchema = z.object({
  email: z.string().email('Enter a valid email'),
  password: z.string().min(8, 'Minimum 8 characters'),
  firstName: z.string().min(1, 'First name is required'),
  lastName: z.string().min(1, 'Last name is required'),
  phone: z.string().optional(),
});

type RegisterFormData = z.infer<typeof registerSchema>;

interface RegisterScreenProps {
  onRegister: (data: RegisterFormData) => Promise<void>;
  onNavigateToLogin?: () => void;
  title?: string;
}

export function RegisterScreen({
  onRegister,
  onNavigateToLogin,
  title = 'Create account',
}: RegisterScreenProps) {
  const [error, setError] = useState<string | null>(null);
  const {
    control,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<RegisterFormData>({ resolver: zodResolver(registerSchema) });

  const onSubmit = async (data: RegisterFormData) => {
    setError(null);
    try {
      await onRegister(data);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Registration failed. Please try again.';
      setError(msg);
    }
  };

  return (
    <View className="flex-1 bg-white px-6 pt-16">
      <H1 className="mb-2">{title}</H1>
      <Text size="base" color="#718096" className="mb-8">
        Fill in your details to get started
      </Text>

      {error ? (
        <View className="bg-red-50 border border-red-200 rounded-lg p-3 mb-4">
          <Text size="sm" color="#b91c1c">{error}</Text>
        </View>
      ) : null}

      <Controller
        control={control}
        name="firstName"
        render={({ field: { onChange, onBlur, value } }) => (
          <View className="mb-4">
            <Input
              label="First name"
              placeholder="First name"
              autoCapitalize="words"
              autoComplete="given-name"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              isInvalid={!!errors.firstName}
              errorMessage={errors.firstName?.message}
            />
          </View>
        )}
      />

      <Controller
        control={control}
        name="lastName"
        render={({ field: { onChange, onBlur, value } }) => (
          <View className="mb-4">
            <Input
              label="Last name"
              placeholder="Last name"
              autoCapitalize="words"
              autoComplete="family-name"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              isInvalid={!!errors.lastName}
              errorMessage={errors.lastName?.message}
            />
          </View>
        )}
      />

      <Controller
        control={control}
        name="email"
        render={({ field: { onChange, onBlur, value } }) => (
          <View className="mb-4">
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

      <Controller
        control={control}
        name="password"
        render={({ field: { onChange, onBlur, value } }) => (
          <View className="mb-6">
            <Input
              label="Password"
              placeholder="Minimum 8 characters"
              secureTextEntry
              autoComplete="new-password"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              isInvalid={!!errors.password}
              errorMessage={errors.password?.message}
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
        Create account
      </Button>

      {onNavigateToLogin ? (
        <View className="mt-6 items-center">
          <Button variant="outline" onPress={onNavigateToLogin}>
            Already have an account? Sign in
          </Button>
        </View>
      ) : null}
    </View>
  );
}
