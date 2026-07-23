// packages/mobile-shared/src/screens/ForgotPasswordScreen.tsx

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

const forgotPasswordSchema = z.object({
  email: z.string().email('Enter a valid email'),
});

type ForgotPasswordFormData = z.infer<typeof forgotPasswordSchema>;

interface ForgotPasswordScreenProps {
  onForgotPassword: (data: ForgotPasswordFormData) => Promise<void>;
  onNavigateToLogin?: () => void;
  /** Optional brand wordmark. When provided, renders above the title. */
  brand?: string;
  title?: string;
  subtitle?: string;
  /** Optional accent colour for the primary CTA + Input focus ring. Customer
   *  passes its Airbnb coral; vendor/driver omit it and keep the ink palette. */
  accent?: string;
  /** Optional colour override for the "Back to sign in" text link only —
   *  distinct from `accent` (CTA fill + Input focus ring). THE SPEC's AA
   *  micro-adjustment: the customer's coral fill (#FF385C) reads ~3.9:1 at
   *  link/body text size (fails AA), so the customer wrapper passes
   *  `coral-pressed` (#E00B41, ~4.9:1) here. Also drops the underline when
   *  set. Defaults to `accent` when omitted, so vendor/driver are unaffected. */
  linkColor?: string;
}

export function ForgotPasswordScreen({
  onForgotPassword,
  onNavigateToLogin,
  title = 'Reset password',
  subtitle = "Enter your email and we'll send you a reset link",
  brand,
  accent,
  linkColor,
}: ForgotPasswordScreenProps) {
  const resolvedLinkColor = linkColor ?? accent;
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
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
  } = useForm<ForgotPasswordFormData>({
    resolver: zodResolver(forgotPasswordSchema),
    defaultValues: { email: '' },
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

  const onSubmit = async (data: ForgotPasswordFormData) => {
    setError(null);
    try {
      await onForgotPassword(data);
      setSuccess(true);
    } catch (e: unknown) {
      setError(resolveAuthErrorMessage(e));
    }
  };

  // The Go API returns a generic success regardless of whether the email
  // exists — anti-enumeration. The success panel makes no claim about an
  // account: it says "if there's one, a link is on the way."
  if (success) {
    return (
      <Screen paddingX={theme.spacing[6]}>
        <View style={styles.successWrap}>
          {/* Envelope glyph — drawn from Views to avoid svg dep */}
          <View style={styles.envelopeIcon}>
            <View style={styles.envelopeBody}>
              <View style={styles.envelopeFlapLeft} />
              <View style={styles.envelopeFlapRight} />
            </View>
          </View>

          <Text style={styles.successTitle}>Check your inbox</Text>
          <Text style={styles.successBody}>
            If an account exists for that email, we've sent a reset link. It can take a couple of minutes.
          </Text>

          {onNavigateToLogin ? (
            <View style={styles.successBackRow}>
              <Button
                label="Back to sign in"
                variant="ghost"
                onPress={onNavigateToLogin}
              />
            </View>
          ) : null}
        </View>
      </Screen>
    );
  }

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
            accentColor={accent}
          />
        )}
      />

      <View style={styles.primaryAction}>
        <Button
          label={isSubmitting ? 'Sending…' : 'Send reset link'}
          onPress={handleSubmit(onSubmit)}
          loading={isSubmitting}
          disabled={isSubmitting}
          accentColor={accent}
        />
      </View>

      {onNavigateToLogin ? (
        <View style={styles.backRow}>
          <Pressable
            onPress={onNavigateToLogin}
            hitSlop={8}
            accessibilityRole="link"
            accessibilityLabel="Back to sign in"
          >
            <Text
              style={[
                styles.backText,
                resolvedLinkColor
                  ? { color: resolvedLinkColor, textDecorationLine: 'none' }
                  : null,
              ]}
            >
              Back to sign in
            </Text>
          </Pressable>
        </View>
      ) : null}

      <View style={styles.bottomGap} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  topGap: { height: theme.spacing[8] },
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

  primaryAction: {
    marginTop: theme.spacing[2],
    marginBottom: theme.spacing[5],
  },

  backRow: { alignItems: 'center' },
  backText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    textDecorationLine: 'underline',
  },

  // ---- Success state ----------------------------------------------------------
  // Not using EmptyState — that component reads as "nothing here yet" (list
  // pages, search zero state). A reset-email confirmation is a deliberate
  // outcome screen, so it gets its own inline treatment that reads as
  // positive completion rather than absence.

  successWrap: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
    gap: theme.spacing[4],
  },

  // Envelope drawn from Views — no SVG dep, matches the AppleGlyph
  // approach used in the social row. 40×30pt target so it reads at a
  // glance without dominating.
  envelopeIcon: {
    marginBottom: theme.spacing[2],
  },
  envelopeBody: {
    width: 40,
    height: 28,
    borderRadius: theme.radius.sm,
    borderWidth: 1.5,
    borderColor: theme.colors.ink.DEFAULT,
    overflow: 'hidden',
    position: 'relative',
  },
  envelopeFlapLeft: {
    position: 'absolute',
    top: 0,
    left: 0,
    width: 22,
    height: 18,
    borderBottomRightRadius: 12,
    borderRightWidth: 1.5,
    borderBottomWidth: 1.5,
    borderColor: theme.colors.ink.DEFAULT,
    transform: [{ rotate: '-8deg' }, { translateX: -4 }, { translateY: -4 }],
  },
  envelopeFlapRight: {
    position: 'absolute',
    top: 0,
    right: 0,
    width: 22,
    height: 18,
    borderBottomLeftRadius: 12,
    borderLeftWidth: 1.5,
    borderBottomWidth: 1.5,
    borderColor: theme.colors.ink.DEFAULT,
    transform: [{ rotate: '8deg' }, { translateX: 4 }, { translateY: -4 }],
  },

  successTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.h1.size,
    lineHeight:
      theme.typography.size.h1.size * theme.typography.size.h1.lineHeight,
    letterSpacing: theme.typography.size.h1.letterSpacing,
    color: theme.colors.ink.DEFAULT,
    textAlign: 'center',
  },
  successBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    lineHeight:
      theme.typography.size.body.size * theme.typography.size.body.lineHeight,
    color: theme.colors.ink.soft,
    textAlign: 'center',
  },
  successBackRow: {
    marginTop: theme.spacing[2],
    alignSelf: 'stretch',
  },
});
