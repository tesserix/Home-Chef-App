// LoginScreen — uses NativeWind primitives for buttons so brand colors apply
// on React Native. The @tesserix/native Button reads CSS variables via
// `document.documentElement`, which is a web-only DOM API; in RN it always
// falls through to a hardcoded iOS-blue default. Inputs/headings stay on
// @tesserix/native because their fallback colors (white/black) are neutral.

import React, { useState } from 'react';
import { View, Pressable, Text as RNText } from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Input, Text, H1 } from '@tesserix/native';

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
      const msg = e instanceof Error ? e.message : 'Login failed. Please try again.';
      setError(msg);
    }
  };

  const runAsyncHandler = (handler: () => Promise<void>, fallbackMsg: string) => async () => {
    setError(null);
    try {
      await handler();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : fallbackMsg);
    }
  };

  return (
    <View className="flex-1 bg-bone px-6 pt-16">
      <H1 className="mb-2">{title}</H1>
      <Text size="base" color="#7a7a76" className="mb-8">
        Sign in to continue
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
        <View className="mb-4 items-end">
          <Pressable onPress={onNavigateToForgotPassword} className="py-2">
            <RNText className="text-herb text-sm">Forgot password?</RNText>
          </Pressable>
        </View>
      ) : null}

      <Pressable
        onPress={handleSubmit(onSubmit)}
        disabled={isSubmitting}
        className={`rounded-lg py-4 items-center w-full ${
          isSubmitting ? 'bg-herb-soft' : 'bg-herb active:bg-herb-soft'
        }`}
      >
        <RNText className="text-white text-base font-semibold">
          {isSubmitting ? 'Signing in…' : 'Sign in'}
        </RNText>
      </Pressable>

      {(onGoogleSignIn || onAppleSignIn || onBiometricLogin) ? (
        <>
          <View className="flex-row items-center my-6">
            <View className="flex-1 h-px bg-mist" />
            <Text size="sm" color="#7a7a76" className="mx-4">or</Text>
            <View className="flex-1 h-px bg-mist" />
          </View>

          {onGoogleSignIn ? (
            <Pressable
              onPress={runAsyncHandler(onGoogleSignIn, 'Google sign-in failed')}
              className="border border-herb rounded-lg py-4 items-center w-full mb-3 active:bg-herb-tint"
            >
              <RNText className="text-herb text-base font-semibold">Continue with Google</RNText>
            </Pressable>
          ) : null}

          {onAppleSignIn ? (
            <Pressable
              onPress={runAsyncHandler(onAppleSignIn, 'Apple sign-in failed')}
              className="bg-ink rounded-lg py-4 items-center w-full mb-3 active:bg-ink-soft"
            >
              <RNText className="text-white text-base font-semibold">Continue with Apple</RNText>
            </Pressable>
          ) : null}

          {onBiometricLogin ? (
            <Pressable
              onPress={runAsyncHandler(onBiometricLogin, 'Biometric auth failed')}
              className="border border-herb rounded-lg py-3 items-center w-full active:bg-herb-tint"
            >
              <RNText className="text-herb text-base font-semibold">Use Face ID / Touch ID</RNText>
            </Pressable>
          ) : null}
        </>
      ) : null}

      {onNavigateToRegister ? (
        <View className="mt-6 items-center">
          <Pressable onPress={onNavigateToRegister} className="py-2">
            <RNText className="text-herb text-sm">Don't have an account? Sign up</RNText>
          </Pressable>
        </View>
      ) : null}
    </View>
  );
}
