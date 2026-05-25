// LoginScreen — buttons use explicit StyleSheet objects so they render
// correctly on RN. NativeWind v5 preview doesn't reliably transform
// className on Pressable, and @tesserix/native Button reads DOM APIs that
// don't exist on RN (defaults to iOS blue). Inline styles are the
// pragmatic fix; tokens mirror tailwind.config.js paper/ink/herb scale.

import React, { useState } from 'react';
import { View, Pressable, Text as RNText, StyleSheet } from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Input, Text, H1 } from '@tesserix/native';

const BRAND = {
  bone: '#f3f2ee',
  ink: '#1a1a18',
  inkSoft: '#4a4a47',
  inkMuted: '#7a7a76',
  // Persimmon (warm editorial orange) — token names kept as `herb*` to avoid
  // wide-scope renames; render orange. See apps/mobile-*/tailwind.config.js.
  herb: '#C2410C',
  herbSoft: '#9A3412',
  herbTint: '#FFEDD5',
  mist: '#e6e5e0',
  paprika: '#c95b3e',
  paprikaTint: '#f3dcd2',
  white: '#ffffff',
} as const;

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: BRAND.bone, paddingHorizontal: 24, paddingTop: 64 },
  errorBanner: {
    backgroundColor: BRAND.paprikaTint,
    borderColor: BRAND.paprika,
    borderWidth: 1,
    borderRadius: 8,
    padding: 12,
    marginBottom: 16,
  },
  field: { marginBottom: 16 },
  fieldLast: { marginBottom: 24 },
  forgotRow: { marginBottom: 16, alignItems: 'flex-end' },
  ghostText: { color: BRAND.herb, fontSize: 14 },
  ctaPrimary: {
    backgroundColor: BRAND.herb,
    borderRadius: 8,
    paddingVertical: 16,
    alignItems: 'center',
    width: '100%',
  },
  ctaPrimaryDisabled: {
    backgroundColor: BRAND.herbSoft,
    borderRadius: 8,
    paddingVertical: 16,
    alignItems: 'center',
    width: '100%',
  },
  ctaOutline: {
    borderColor: BRAND.herb,
    borderWidth: 1,
    borderRadius: 8,
    paddingVertical: 16,
    alignItems: 'center',
    width: '100%',
    marginBottom: 12,
  },
  ctaDark: {
    backgroundColor: BRAND.ink,
    borderRadius: 8,
    paddingVertical: 16,
    alignItems: 'center',
    width: '100%',
    marginBottom: 12,
  },
  ctaPrimaryText: { color: BRAND.white, fontSize: 16, fontWeight: '600' },
  ctaOutlineText: { color: BRAND.herb, fontSize: 16, fontWeight: '600' },
  dividerRow: { flexDirection: 'row', alignItems: 'center', marginVertical: 24 },
  dividerLine: { flex: 1, height: 1, backgroundColor: BRAND.mist },
  dividerLabel: { marginHorizontal: 16 },
  signupRow: { marginTop: 24, alignItems: 'center' },
});

const loginSchema = z.object({
  email: z.string().email('Enter a valid email'),
  password: z.string().min(1, 'Password is required'),
});

type LoginFormData = z.infer<typeof loginSchema>;

interface LoginScreenProps {
  onLogin: (data: LoginFormData) => Promise<void>;
  onNavigateToRegister?: () => void;
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
      setError(e instanceof Error ? e.message : 'Login failed. Please try again.');
    }
  };

  const wrap = (handler: () => Promise<void>, fallback: string) => async () => {
    setError(null);
    try {
      await handler();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : fallback);
    }
  };

  return (
    <View style={styles.screen}>
      <H1 style={{ marginBottom: 8 }}>{title}</H1>
      <Text size="base" color={BRAND.inkMuted} style={{ marginBottom: 32 }}>
        Sign in to continue
      </Text>

      {error ? (
        <View style={styles.errorBanner}>
          <Text size="sm" color={BRAND.paprika}>{error}</Text>
        </View>
      ) : null}

      <Controller
        control={control}
        name="email"
        render={({ field: { onChange, onBlur, value } }) => (
          <View style={styles.field}>
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
          <View style={styles.fieldLast}>
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
        <View style={styles.forgotRow}>
          <Pressable onPress={onNavigateToForgotPassword} hitSlop={8}>
            <RNText style={styles.ghostText}>Forgot password?</RNText>
          </Pressable>
        </View>
      ) : null}

      <Pressable
        onPress={handleSubmit(onSubmit)}
        disabled={isSubmitting}
        style={isSubmitting ? styles.ctaPrimaryDisabled : styles.ctaPrimary}
      >
        <RNText style={styles.ctaPrimaryText}>
          {isSubmitting ? 'Signing in…' : 'Sign in'}
        </RNText>
      </Pressable>

      {(onGoogleSignIn || onAppleSignIn || onBiometricLogin) ? (
        <>
          <View style={styles.dividerRow}>
            <View style={styles.dividerLine} />
            <Text size="sm" color={BRAND.inkMuted} style={styles.dividerLabel}>or</Text>
            <View style={styles.dividerLine} />
          </View>

          {onGoogleSignIn ? (
            <Pressable onPress={wrap(onGoogleSignIn, 'Google sign-in failed')} style={styles.ctaOutline}>
              <RNText style={styles.ctaOutlineText}>Continue with Google</RNText>
            </Pressable>
          ) : null}

          {onAppleSignIn ? (
            <Pressable onPress={wrap(onAppleSignIn, 'Apple sign-in failed')} style={styles.ctaDark}>
              <RNText style={styles.ctaPrimaryText}>Continue with Apple</RNText>
            </Pressable>
          ) : null}

          {onBiometricLogin ? (
            <Pressable onPress={wrap(onBiometricLogin, 'Biometric auth failed')} style={styles.ctaOutline}>
              <RNText style={styles.ctaOutlineText}>Use Face ID / Touch ID</RNText>
            </Pressable>
          ) : null}
        </>
      ) : null}

      {onNavigateToRegister ? (
        <View style={styles.signupRow}>
          <Pressable onPress={onNavigateToRegister} hitSlop={8}>
            <RNText style={styles.ghostText}>Don't have an account? Sign up</RNText>
          </Pressable>
        </View>
      ) : null}
    </View>
  );
}
