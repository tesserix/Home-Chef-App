// packages/mobile-shared/src/screens/RegisterScreen.tsx

import React, { useEffect, useRef, useState } from 'react';
import {
  Animated,
  Easing,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Screen } from '../ui/Screen';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { theme } from '../theme/tokens';
import { resolveAuthErrorMessage } from '../auth/bff-session';
import { SocialIconButton, GoogleGlyph, AppleGlyph } from './_socialIcons';

const registerSchema = z.object({
  firstName: z.string().min(1, 'First name is required'),
  lastName: z.string().min(1, 'Last name is required'),
  email: z.string().email('Enter a valid email'),
  // Phone stays optional at sign-up (onboarding makes it required), but when a
  // value IS entered it must match the same 10-digit Indian-mobile rule the
  // onboarding step enforces — otherwise a malformed number (e.g. 9 digits)
  // sails through here and only gets rejected later on the onboarding screen.
  phone: z
    .string()
    .optional()
    .refine((v) => !v || /^[6-9]\d{9}$/.test(v), {
      message: 'Enter a valid 10-digit Indian mobile number',
    }),
  password: z.string().min(8, 'Minimum 8 characters'),
});

type RegisterFormData = z.infer<typeof registerSchema>;

interface RegisterScreenProps {
  onRegister: (data: RegisterFormData) => Promise<void>;
  onNavigateToLogin?: () => void;
  /** When provided, the social icon row appears below the form under an
   *  "or continue with" hairline divider — same pattern as LoginScreen.
   *  The OAuth handler is expected to create the account and route the
   *  user into the app on success. */
  onGoogleSignIn?: () => Promise<void>;
  /** Same pattern as `onGoogleSignIn`, iOS only — callers should gate by
   *  `Platform.OS === 'ios'`. */
  onAppleSignIn?: () => Promise<void>;
  /** Optional brand wordmark. When provided, renders above the title. */
  brand?: string;
  title?: string;
  subtitle?: string;
  /** Optional accent colour for the primary CTA + links. Customer passes its
   *  Airbnb coral; vendor/driver omit it and keep the ink palette. */
  accent?: string;
}

export function RegisterScreen({
  onRegister,
  onNavigateToLogin,
  onGoogleSignIn,
  onAppleSignIn,
  title = 'Create account',
  subtitle = 'A few details to get you cooking',
  brand,
  accent,
}: RegisterScreenProps) {
  const [error, setError] = useState<string | null>(null);
  const errorOpacity = useRef(new Animated.Value(0)).current;
  const errorTranslate = useRef(new Animated.Value(-8)).current;

  const {
    control,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
    defaultValues: { firstName: '', lastName: '', email: '', phone: '', password: '' },
  });

  useEffect(() => {
    Animated.parallel([
      Animated.timing(errorOpacity, {
        toValue: error ? 1 : 0,
        duration: theme.motion.duration.default,
        easing: Easing.bezier(...theme.motion.easing.entrance),
        useNativeDriver: true,
      }),
      Animated.timing(errorTranslate, {
        toValue: error ? 0 : -8,
        duration: theme.motion.duration.default,
        easing: Easing.bezier(...theme.motion.easing.entrance),
        useNativeDriver: true,
      }),
    ]).start();
  }, [error, errorOpacity, errorTranslate]);

  const onSubmit = async (data: RegisterFormData) => {
    setError(null);
    try {
      await onRegister(data);
    } catch (e: unknown) {
      setError(resolveAuthErrorMessage(e));
    }
  };

  const wrap = (handler: () => Promise<void>) => async () => {
    setError(null);
    try {
      await handler();
    } catch (e: unknown) {
      setError(resolveAuthErrorMessage(e));
    }
  };

  return (
    <Screen scroll paddingX={theme.spacing[6]}>
      <View style={styles.topGap} />

      {brand ? <Text style={styles.brand}>{brand}</Text> : null}
      <Text style={styles.title}>{title}</Text>
      <Text style={styles.subtitle}>{subtitle}</Text>

      <Animated.View
        style={[
          styles.errorBannerWrap,
          {
            opacity: errorOpacity,
            transform: [{ translateY: errorTranslate }],
            // Reserve no space when error is null, so layout doesn't jump.
            height: error ? undefined : 0,
            marginBottom: error ? theme.spacing[4] : 0,
          },
        ]}
        pointerEvents={error ? 'auto' : 'none'}
      >
        {error ? (
          <View style={styles.errorBanner}>
            <Text style={styles.errorText}>{error}</Text>
          </View>
        ) : null}
      </Animated.View>

      <View style={styles.nameRow}>
        <View style={styles.nameField}>
          <Controller
            control={control}
            name="firstName"
            render={({ field: { onChange, onBlur, value } }) => (
              <Input
                label="First name"
                autoCapitalize="words"
                autoComplete="given-name"
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                error={errors.firstName?.message}
              />
            )}
          />
        </View>
        <View style={styles.nameField}>
          <Controller
            control={control}
            name="lastName"
            render={({ field: { onChange, onBlur, value } }) => (
              <Input
                label="Last name"
                autoCapitalize="words"
                autoComplete="family-name"
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                error={errors.lastName?.message}
              />
            )}
          />
        </View>
      </View>

      <Controller
        control={control}
        name="email"
        render={({ field: { onChange, onBlur, value } }) => (
          <Input
            label="Email"
            placeholder="you@example.com"
            keyboardType="email-address"
            autoCapitalize="none"
            autoComplete="email"
            onBlur={onBlur}
            onChangeText={onChange}
            value={value}
            error={errors.email?.message}
          />
        )}
      />

      <Controller
        control={control}
        name="phone"
        render={({ field: { onChange, onBlur, value } }) => (
          <Input
            label="Phone (optional)"
            placeholder="9876543210"
            keyboardType="phone-pad"
            autoComplete="tel"
            onBlur={onBlur}
            onChangeText={onChange}
            value={value ?? ''}
            error={errors.phone?.message}
            helper="We use this only for order issues — never shared."
          />
        )}
      />

      <Controller
        control={control}
        name="password"
        render={({ field: { onChange, onBlur, value } }) => (
          <Input
            label="Password"
            placeholder="At least 8 characters"
            secureTextEntry
            passwordPeek
            autoComplete="new-password"
            onBlur={onBlur}
            onChangeText={onChange}
            value={value}
            error={errors.password?.message}
            helper="At least 8 characters. Mix letters and numbers."
          />
        )}
      />

      <View style={styles.primaryActions}>
        <Button
          label={isSubmitting ? 'Creating account…' : 'Create account'}
          onPress={handleSubmit(onSubmit)}
          loading={isSubmitting}
          disabled={isSubmitting}
          accentColor={accent}
        />
      </View>

      {onGoogleSignIn || onAppleSignIn ? (
        <>
          <View style={styles.divider}>
            <View style={styles.dividerLine} />
            <Text style={styles.dividerLabel}>or continue with</Text>
            <View style={styles.dividerLine} />
          </View>

          <View style={styles.socialIconRow}>
            {onGoogleSignIn ? (
              <SocialIconButton
                label="Continue with Google"
                onPress={wrap(onGoogleSignIn)}
                icon={<GoogleGlyph />}
              />
            ) : null}
            {onAppleSignIn ? (
              <SocialIconButton
                label="Continue with Apple"
                onPress={wrap(onAppleSignIn)}
                icon={<AppleGlyph />}
              />
            ) : null}
          </View>
        </>
      ) : null}

      {onNavigateToLogin ? (
        <View style={styles.signinRow}>
          <Pressable onPress={onNavigateToLogin} hitSlop={8}>
            <Text style={styles.signinPrompt}>
              Already have an account?{' '}
              <Text style={[styles.signinCTA, accent ? { color: accent } : null]}>
                Sign in
              </Text>
            </Text>
          </Pressable>
        </View>
      ) : null}

      <View style={styles.bottomGap} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  topGap: { height: theme.spacing[6] },
  bottomGap: { height: theme.spacing[8] },

  brand: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    letterSpacing: 1.5,
    textTransform: 'uppercase',
    marginBottom: theme.spacing[6],
  },
  title: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.display.size,
    lineHeight:
      theme.typography.size.display.size *
      theme.typography.size.display.lineHeight,
    letterSpacing: theme.typography.size.display.letterSpacing,
    color: theme.colors.ink.DEFAULT,
    marginBottom: theme.spacing[1],
  },
  // ink.soft (not ink.muted) — subtitle is secondary, not tertiary.
  // Matches LoginScreen's subtitle style exactly.
  subtitle: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.soft,
    marginBottom: theme.spacing[6],
  },

  errorBannerWrap: { overflow: 'hidden' },
  errorBanner: {
    backgroundColor: theme.colors.destructive.tint,
    borderLeftWidth: 3,
    borderLeftColor: theme.colors.destructive.DEFAULT,
    borderRadius: theme.radius.sm,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
  },
  errorText: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.destructive.DEFAULT,
  },

  nameRow: {
    flexDirection: 'row',
    gap: theme.spacing[3],
  },
  nameField: { flex: 1 },

  primaryActions: {
    marginTop: theme.spacing[2],
    marginBottom: theme.spacing[6],
  },

  divider: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: theme.spacing[4],
  },
  dividerLine: {
    flex: 1,
    height: 1,
    backgroundColor: theme.colors.mist.DEFAULT,
  },
  dividerLabel: {
    marginHorizontal: theme.spacing[3],
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    letterSpacing: 1,
    textTransform: 'uppercase',
  },

  socialIconRow: {
    flexDirection: 'row',
    gap: theme.spacing[3],
    justifyContent: 'center',
    marginBottom: theme.spacing[6],
  },

  signinRow: {
    alignItems: 'center',
    marginTop: theme.spacing[2],
  },
  signinPrompt: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
  },
  signinCTA: {
    fontFamily: 'Inter-SemiBold',
    color: theme.colors.ink.DEFAULT,
    textDecorationLine: 'underline',
  },
});
