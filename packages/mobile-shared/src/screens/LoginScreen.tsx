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
  /** Greeting copy override. Defaults to "Welcome back". */
  title?: string;
  /** One-line supporting copy under the title. Pass app-specific wording —
   *  e.g. vendor: "Sign in to keep your kitchen running". */
  subtitle?: string;
  /** Optional brand wordmark. When provided, renders above the title — the
   *  only persimmon-coloured element above the fold. */
  brand?: string;
  /** Optional accent colour for the primary CTA + links. The customer app
   *  passes its Airbnb coral; vendor/driver omit it and keep the ink palette. */
  accent?: string;
}

/**
 * <LoginScreen> — first impression. Per .impeccable.md the brand is
 * "confident, appetizing, quietly modern" — this screen reads as
 * confident through restraint: generous whitespace, one accent, no
 * decorative chrome.
 *
 * Layout decisions:
 *   - Geist display headline; Inter body — the brand voice lands here
 *     before any colour does.
 *   - Biometric is the "fast lane" — surfaced *above* the email/password
 *     form for returning users. Email/password is the explicit path.
 *   - Social (Google / Apple) lives below an "or" hairline divider so
 *     the visual weight matches their conceptual weight — fallback for
 *     users who don't have an account yet.
 *   - The error banner slides in (250ms ease-out-quart) and lives in the
 *     same column flow as everything else; no overlay.
 */
export function LoginScreen({
  onLogin,
  onNavigateToRegister,
  onNavigateToForgotPassword,
  onGoogleSignIn,
  onAppleSignIn,
  onBiometricLogin,
  title = 'Welcome back',
  subtitle = 'Sign in to continue',
  brand,
  accent,
}: LoginScreenProps) {
  const [error, setError] = useState<string | null>(null);
  const errorOpacity = useRef(new Animated.Value(0)).current;
  const errorTranslate = useRef(new Animated.Value(-8)).current;

  const {
    control,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: '', password: '' },
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

  const onSubmit = async (data: LoginFormData) => {
    setError(null);
    try {
      await onLogin(data);
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
        name="password"
        render={({ field: { onChange, onBlur, value } }) => (
          <Input
            label="Password"
            placeholder="••••••••"
            secureTextEntry
            passwordPeek
            autoComplete="password"
            onBlur={onBlur}
            onChangeText={onChange}
            value={value}
            error={errors.password?.message}
          />
        )}
      />

      {onNavigateToForgotPassword ? (
        <View style={styles.forgotRow}>
          <Pressable onPress={onNavigateToForgotPassword} hitSlop={8}>
            <Text style={[styles.linkText, accent ? { color: accent } : null]}>
              Forgot password?
            </Text>
          </Pressable>
        </View>
      ) : null}

      <View style={styles.primaryActions}>
        <Button
          label={isSubmitting ? 'Signing in…' : 'Sign in'}
          onPress={handleSubmit(onSubmit)}
          loading={isSubmitting}
          disabled={isSubmitting}
          accentColor={accent}
        />
        {onBiometricLogin ? (
          <Button
            label="Use Face ID / Touch ID"
            variant="ghost"
            onPress={wrap(onBiometricLogin)}
          />
        ) : null}
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

      {onNavigateToRegister ? (
        <View style={styles.signupRow}>
          <Pressable onPress={onNavigateToRegister} hitSlop={8}>
            <Text style={styles.signupPrompt}>
              Don't have an account?{' '}
              <Text style={[styles.signupCTA, accent ? { color: accent } : null]}>
                Sign up
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
  subtitle: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.soft,
    marginBottom: theme.spacing[8],
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

  forgotRow: {
    alignItems: 'flex-end',
    marginBottom: theme.spacing[5],
  },
  linkText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.DEFAULT,
    textDecorationLine: 'underline',
  },

  primaryActions: {
    gap: theme.spacing[2],
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

  signupRow: {
    alignItems: 'center',
    marginTop: theme.spacing[2],
  },
  signupPrompt: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
  },
  signupCTA: {
    fontFamily: 'Inter-SemiBold',
    color: theme.colors.ink.DEFAULT,
    textDecorationLine: 'underline',
  },
});
