import React, { useEffect, useRef, useState } from 'react';
import {
  AccessibilityInfo,
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
  /** Email/password submit — legacy path; omit when using hosted sign-in. */
  onLogin?: (data: LoginFormData) => Promise<void>;
  /**
   * Hosted (Zitadel) sign-in: opens the system browser and resolves once
   * the session exists. When provided, the in-app credential form and
   * social buttons are hidden — the hosted page owns all of that.
   */
  onHostedSignIn?: () => Promise<void>;
  /** CTA label for the hosted button. Defaults to "Sign in". */
  hostedCtaLabel?: string;
  /** Copy for the bottom account-switch row (defaults to sign-up copy). */
  registerPrompt?: string;
  registerCta?: string;
  onNavigateToRegister?: () => void;
  onNavigateToForgotPassword?: () => void;
  onGoogleSignIn?: () => Promise<void>;
  onAppleSignIn?: () => Promise<void>;
  onBiometricLogin?: () => Promise<void>;
  /** Greeting copy override. Defaults to "Welcome back". */
  title?: string;
  /** One-line supporting copy under the title. Pass app-specific wording —
   *  e.g. the brand tagline "For those who love to eat — and those who love
   *  to cook." */
  subtitle?: string;
  /** Optional brand wordmark. When provided, renders above the title — the
   *  only persimmon-coloured element above the fold. */
  brand?: string;
  /** Optional accent colour for the primary CTA + focus rings. The customer
   *  app passes its Airbnb coral; vendor/driver omit it and keep the ink
   *  palette. */
  accent?: string;
  /** Optional colour override for text LINKS only ("Forgot password?",
   *  "Sign up") — distinct from `accent` (CTA fill + Input focus ring).
   *  THE SPEC's AA micro-adjustment: the customer's coral fill (#FF385C)
   *  reads ~3.9:1 at link/body text size (fails AA), so the customer wrapper
   *  passes `coral-pressed` (#E00B41, ~4.9:1) here while `accent` stays
   *  coral for fills. Also drops the underline when set, matching the
   *  customer spec's "coral, no underline" link style. Defaults to `accent`
   *  when omitted, so vendor/driver are unaffected. */
  linkColor?: string;
  /**
   * Optional "browse without an account" escape hatch.
   *
   * App Review guideline 5.1.1(iv): an app may only require an account for
   * features that genuinely need one. The customer app passes this so people
   * can look at chefs and menus before committing; the vendor app does not,
   * because every screen behind it is tied to a specific kitchen.
   */
  onContinueAsGuest?: () => void;
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
  onHostedSignIn,
  hostedCtaLabel = 'Sign in',
  registerPrompt = "Don't have an account?",
  registerCta = 'Sign up',
  onNavigateToRegister,
  onNavigateToForgotPassword,
  onGoogleSignIn,
  onAppleSignIn,
  onBiometricLogin,
  title = 'Welcome back',
  subtitle = 'Sign in to continue',
  brand,
  accent,
  linkColor,
  onContinueAsGuest,
}: LoginScreenProps) {
  const resolvedLinkColor = linkColor ?? accent;
  const [error, setError] = useState<string | null>(null);
  const [hostedSubmitting, setHostedSubmitting] = useState(false);
  const errorOpacity = useRef(new Animated.Value(0)).current;
  const errorTranslate = useRef(new Animated.Value(-8)).current;

  // No Reanimated dependency on this screen, so Reduce Motion is read the
  // same way the shared UI primitives (Skeleton/SheetBase/Toast) do — via
  // AccessibilityInfo.
  const [reduceMotion, setReduceMotion] = useState(false);
  useEffect(() => {
    let mounted = true;
    AccessibilityInfo.isReduceMotionEnabled()
      .then((enabled) => {
        if (mounted) setReduceMotion(enabled);
      })
      .catch(() => {});
    const subscription = AccessibilityInfo.addEventListener('reduceMotionChanged', (enabled) => {
      setReduceMotion(enabled);
    });
    return () => {
      mounted = false;
      subscription.remove();
    };
  }, []);

  const {
    control,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: '', password: '' },
  });

  useEffect(() => {
    if (reduceMotion) {
      errorOpacity.setValue(error ? 1 : 0);
      errorTranslate.setValue(error ? 0 : -8);
      return;
    }
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
  }, [error, errorOpacity, errorTranslate, reduceMotion]);

  const onSubmit = async (data: LoginFormData) => {
    if (!onLogin) return;
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

  const hostedPress = async () => {
    if (!onHostedSignIn || hostedSubmitting) return;
    setError(null);
    setHostedSubmitting(true);
    try {
      await onHostedSignIn();
    } catch (e: unknown) {
      setError(resolveAuthErrorMessage(e));
    } finally {
      setHostedSubmitting(false);
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

      {onHostedSignIn ? (
        <View style={styles.primaryActions}>
          <Button
            label={hostedSubmitting ? 'Opening…' : hostedCtaLabel}
            onPress={hostedPress}
            loading={hostedSubmitting}
            disabled={hostedSubmitting}
            accentColor={accent}
          />
          {onBiometricLogin ? (
            <Button
              label="Use Face ID / Touch ID"
              variant="ghost"
              onPress={wrap(onBiometricLogin)}
            />
          ) : null}
          <Text style={styles.hostedHint}>
            You'll sign in securely in your browser and come right back.
          </Text>
        </View>
      ) : null}

      {!onHostedSignIn ? (
        <>
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
            accentColor={accent}
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
            accentColor={accent}
          />
        )}
      />

      {onNavigateToForgotPassword ? (
        <View style={styles.forgotRow}>
          <Pressable
            onPress={onNavigateToForgotPassword}
            hitSlop={8}
            accessibilityRole="link"
            accessibilityLabel="Forgot password?"
          >
            <Text
              style={[
                styles.linkText,
                resolvedLinkColor
                  ? { color: resolvedLinkColor, textDecorationLine: 'none' }
                  : null,
              ]}
            >
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

        </>
      ) : null}

      {onContinueAsGuest ? (
        <View style={styles.guestRow}>
          <Pressable
            onPress={onContinueAsGuest}
            hitSlop={8}
            accessibilityRole="button"
            accessibilityLabel="Browse without an account"
            style={styles.guestButton}
          >
            <Text
              style={[
                styles.guestText,
                resolvedLinkColor ? { color: resolvedLinkColor } : null,
              ]}
            >
              Browse without an account
            </Text>
          </Pressable>
        </View>
      ) : null}

      {onNavigateToRegister ? (
        <View style={styles.signupRow}>
          <Pressable
            onPress={onNavigateToRegister}
            hitSlop={8}
            accessibilityRole="link"
            accessibilityLabel={`${registerPrompt} ${registerCta}`}
          >
            <Text style={styles.signupPrompt}>
              {registerPrompt}{' '}
              <Text
                style={[
                  styles.signupCTA,
                  resolvedLinkColor
                    ? { color: resolvedLinkColor, textDecorationLine: 'none' }
                    : null,
                ]}
              >
                {registerCta}
              </Text>
            </Text>
          </Pressable>
        </View>
      ) : null}

      {/* Operator attribution. Sits on the sign-in screen because that is the
          first place a person decides whether to trust the app with money, and
          because the name on their bank statement will be Zivana's, not the
          brand's. Kept small and muted — disclosure, not marketing. */}
      <Text style={styles.operatorLine}>
        Powered by Zivana Innovations LLP{'\n'}
        part of Tesserix Pty Ltd · ACN 694 070 865 · ABN 59 694 070 865
      </Text>

      <View style={styles.bottomGap} />
    </Screen>
  );
}


const styles = StyleSheet.create({
  topGap: { height: theme.spacing[6] },
  bottomGap: { height: theme.spacing[8] },

  operatorLine: {
    marginTop: theme.spacing[6],
    textAlign: 'center',
    fontFamily: 'Inter',
    fontSize: 11,
    lineHeight: 16,
    color: theme.colors.ink.muted,
  },

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
  hostedHint: {
    marginTop: theme.spacing[2],
    textAlign: 'center',
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
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

  guestRow: {
    alignItems: 'center',
    marginTop: theme.spacing[4],
  },
  guestButton: {
    minHeight: 44, // touch-target floor — this is a real action, not fine print
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[4],
  },
  guestText: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
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
