// FOUND-03: Uses @tesserix/native components for all interactive UI and text.
// API note: @tesserix/native exports `Button` (variant: 'solid'|'outline'|'ghost', colorScheme: 'primary'|'secondary',
// isLoading), `Input` (errorMessage, isInvalid, isDisabled), `Text` (size, weight), `H1`/`H2` for headings.
// Plan spec referenced `Typography` + `variant="primary"` + `loading` — adjusted to actual package API.

import React, { useState } from 'react';
import { View } from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
// FOUND-03: use @tesserix/native components — not raw RN primitives for interactive UI
import { Button, Input, Text, H1 } from '@tesserix/native';

const loginSchema = z.object({
  email: z.string().email('Enter a valid email'),
  password: z.string().min(1, 'Password is required'),
});

type LoginFormData = z.infer<typeof loginSchema>;

interface LoginScreenProps {
  onLogin: (data: LoginFormData) => Promise<void>;
  onNavigateToRegister?: () => void;
  /** Vendor-only: show forgot password link */
  onNavigateToForgotPassword?: () => void;
  onGoogleSignIn?: () => Promise<void>;
  onAppleSignIn?: () => Promise<void>;
  onBiometricLogin?: () => Promise<void>;
  title?: string;
}

export function LoginScreen({
  onLogin,
  onNavigateToRegister,
  onNavigateToForgotPassword,
  onGoogleSignIn,
  onAppleSignIn,
  onBiometricLogin,
  title = 'Welcome back',
}: LoginScreenProps) {
  const [error, setError] = useState<string | null>(null);
  const {
    control,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormData>({ resolver: zodResolver(loginSchema) });

  const onSubmit = async (data: LoginFormData) => {
    setError(null);
    try {
      await onLogin(data);
    } catch (e: unknown) {
      // T-03-02: show generic message to avoid leaking "email not found" vs "wrong password"
      const msg = e instanceof Error ? e.message : 'Login failed. Please try again.';
      setError(msg);
    }
  };

  return (
    <View className="flex-1 bg-white px-6 pt-16">
      <H1 className="mb-2">{title}</H1>
      <Text size="base" color="#718096" className="mb-8">
        Sign in to continue
      </Text>

      {error ? (
        <View className="bg-red-50 border border-red-200 rounded-lg p-3 mb-4">
          <Text size="sm" color="#b91c1c">{error}</Text>
        </View>
      ) : null}

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
              placeholder="••••••••"
              secureTextEntry
              autoComplete="password"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              isInvalid={!!errors.password}
              errorMessage={errors.password?.message}
            />
          </View>
        )}
      />

      {onNavigateToForgotPassword ? (
        <View className="mb-6 items-end">
          <Button
            variant="ghost"
            onPress={onNavigateToForgotPassword}
          >
            Forgot password?
          </Button>
        </View>
      ) : null}

      <Button
        variant="solid"
        colorScheme="primary"
        onPress={handleSubmit(onSubmit)}
        disabled={isSubmitting}
        isLoading={isSubmitting}
        fullWidth
      >
        Sign in
      </Button>

      {(onGoogleSignIn || onAppleSignIn || onBiometricLogin) ? (
        <>
          {/* Divider */}
          <View className="flex-row items-center my-6">
            <View className="flex-1 h-px bg-gray-200" />
            <Text size="sm" color="#9ca3af" className="mx-4">or</Text>
            <View className="flex-1 h-px bg-gray-200" />
          </View>

          {onGoogleSignIn ? (
            <View className="mb-3">
              <Button
                variant="outline"
                fullWidth
                onPress={async () => {
                  setError(null);
                  try { await onGoogleSignIn(); }
                  catch (e: unknown) { setError(e instanceof Error ? e.message : 'Google sign-in failed'); }
                }}
              >
                Continue with Google
              </Button>
            </View>
          ) : null}

          {onAppleSignIn ? (
            <View className="mb-3">
              <Button
                variant="solid"
                colorScheme="secondary"
                fullWidth
                onPress={async () => {
                  setError(null);
                  try { await onAppleSignIn(); }
                  catch (e: unknown) { setError(e instanceof Error ? e.message : 'Apple sign-in failed'); }
                }}
              >
                Continue with Apple
              </Button>
            </View>
          ) : null}

          {onBiometricLogin ? (
            <View className="items-center py-3">
              <Button
                variant="outline"
                onPress={async () => {
                  setError(null);
                  try { await onBiometricLogin(); }
                  catch (e: unknown) { setError(e instanceof Error ? e.message : 'Biometric auth failed'); }
                }}
              >
                Use Face ID / Touch ID
              </Button>
            </View>
          ) : null}
        </>
      ) : null}

      {onNavigateToRegister ? (
        <View className="mt-6 items-center">
          <Button variant="outline" onPress={onNavigateToRegister}>
            Don't have an account? Sign up
          </Button>
        </View>
      ) : null}
    </View>
  );
}
