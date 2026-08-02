// packages/mobile-shared/src/screens/RegisterScreen.tsx

import React, { useEffect, useRef, useState } from 'react';
import {
  AccessibilityInfo,
  Animated,
  Easing,
  Keyboard,
  KeyboardAvoidingView,
  type LayoutChangeEvent,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { useAlert } from '../ui/DialogProvider';
import { theme } from '../theme/tokens';
import { resolveAuthErrorMessage } from '../auth/bff-session';
import { SocialIconButton, GoogleGlyph, AppleGlyph } from './_socialIcons';
import { KitchenBackdrop } from './_kitchenBackdrop';
import {
  getPhoneRule,
  sanitizePhoneInput,
  DEFAULT_PHONE_COUNTRY,
  type PhoneRule,
} from '../validation/phone';
import { isStrongPassword, passwordCheckResults } from '../validation/password';

// The form shape is stable regardless of country; only the phone rule inside the
// schema changes. Kept explicit so the onRegister prop type doesn't shift.
interface RegisterFormData {
  firstName: string;
  lastName: string;
  email: string;
  phone?: string;
  password: string;
  confirmPassword: string;
}

/** The persistable slice of the form — never includes passwords. */
export type RegisterDraft = Pick<
  RegisterFormData,
  'firstName' | 'lastName' | 'email' | 'phone'
>;

// Country-aware so a future non-India market just supplies a different phone
// rule (length + pattern) — the rest of the form is unchanged.
function makeRegisterSchema(rule: PhoneRule) {
  return z
    .object({
      firstName: z.string().min(1, 'First name is required'),
      lastName: z.string().min(1, 'Last name is required'),
      email: z.string().email('Enter a valid email'),
      // Phone stays optional at sign-up (onboarding makes it required), but when a
      // value IS entered it must be a complete national number for the country —
      // the input already hard-caps the digit count, this catches "too short".
      phone: z
        .string()
        .optional()
        .refine((v) => !v || rule.pattern.test(v), {
          message: `Enter a valid ${rule.length}-digit mobile number`,
        }),
      password: z
        .string()
        .refine(isStrongPassword, 'Meet all the password requirements below'),
      confirmPassword: z.string().min(1, 'Re-enter your password to confirm'),
    })
    .refine((d) => d.password === d.confirmPassword, {
      message: 'Passwords do not match',
      path: ['confirmPassword'],
    });
}

const EMAIL_LIKE = /^\S+@\S+\.\S+$/;
const STEP1_FIELDS = ['firstName', 'lastName', 'email', 'phone'] as const;

interface RegisterScreenProps {
  onRegister: (data: RegisterFormData) => Promise<void>;
  onNavigateToLogin?: () => void;
  /** When provided, the social icon row appears on step 1 under an
   *  "or continue with" hairline divider — same pattern as LoginScreen.
   *  The OAuth handler is expected to create the account and route the
   *  user into the app on success. */
  onGoogleSignIn?: () => Promise<void>;
  /** Same pattern as `onGoogleSignIn`, iOS only — callers should gate by
   *  `Platform.OS === 'ios'`. */
  onAppleSignIn?: () => Promise<void>;
  /** Optional brand wordmark. Renders in the top bar on step 1. */
  brand?: string;
  title?: string;
  subtitle?: string;
  /** Step 2 headline — the "you're nearly in" moment. */
  step2Title?: string;
  step2Subtitle?: string;
  /** Optional accent colour for the primary CTA, progress fill and Input focus
   *  rings. Customer passes its Airbnb coral; vendor omits it (ink). */
  accent?: string;
  /** Optional colour override for the "Sign in" text link only — distinct
   *  from `accent` (CTA fill + Input focus ring). THE SPEC's AA
   *  micro-adjustment: the customer's coral fill (#FF385C) reads ~3.9:1 at
   *  link/body text size (fails AA), so the customer wrapper passes
   *  `coral-pressed` (#E00B41, ~4.9:1) here. Also drops the underline when
   *  set. Defaults to `accent` when omitted, so vendor/driver are unaffected. */
  linkColor?: string;
  /** ISO country for the phone rule (digit length + format). Defaults to India;
   *  a future market passes its own code. */
  phoneCountry?: string;
  /** Renders a ✕ in the top bar. Called AFTER the user confirms discarding —
   *  the wrapper clears any persisted draft and navigates away. */
  onCancel?: () => void;
  /** Pre-fill from a persisted draft (name/email/phone — never passwords).
   *  Applied only while the form is untouched, so async store hydration
   *  can't clobber typing. */
  initialValues?: Partial<RegisterDraft>;
  /** Called with the step-1 values when the user advances to step 2 — the
   *  wrapper persists them so a killed app resumes where it left off. */
  onDraftSave?: (draft: RegisterDraft) => void;
  /** Whisper-quiet decorative sketch layer behind the form. 'kitchen' scatters
   *  faint pan/cloche/steam line glyphs — the vendor app's "this is about
   *  cooking" cue. Omit for a plain surface. */
  backdrop?: 'kitchen';
  /** Quiet "why sign up" card on step 1 — a ₹ badge + one earning-focused
   *  line. No invented numbers: copy states what the platform actually does
   *  (own menu/prices, bank payouts). */
  incentive?: { title: string; body: string };
}

export function RegisterScreen({
  onRegister,
  onNavigateToLogin,
  onGoogleSignIn,
  onAppleSignIn,
  title = 'Create account',
  subtitle = 'A few details to get you cooking',
  step2Title = 'Almost there',
  step2Subtitle = "Set a password and you're in.",
  brand,
  accent,
  linkColor,
  phoneCountry = DEFAULT_PHONE_COUNTRY,
  onCancel,
  initialValues,
  onDraftSave,
  backdrop,
  incentive,
}: RegisterScreenProps) {
  const resolvedLinkColor = linkColor ?? accent;
  const accentColor = accent ?? theme.colors.ink.DEFAULT;
  const { showAlert } = useAlert();
  const [step, setStep] = useState<1 | 2>(1);
  const [error, setError] = useState<string | null>(null);
  const scrollRef = useRef<ScrollView>(null);

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

  const phoneRule = getPhoneRule(phoneCountry);
  const registerSchema = React.useMemo(() => makeRegisterSchema(phoneRule), [phoneRule]);

  const {
    control,
    handleSubmit,
    watch,
    trigger,
    getValues,
    reset,
    formState: { errors, isSubmitting, isDirty },
  } = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
    defaultValues: { firstName: '', lastName: '', email: '', phone: '', password: '', confirmPassword: '' },
  });

  // Resume a persisted draft, but only while the form is untouched — store
  // hydration is async and must never clobber what the user is typing.
  useEffect(() => {
    if (!initialValues || isDirty) return;
    const hasDraft = Object.values(initialValues).some((v) => !!v);
    if (!hasDraft) return;
    reset({
      firstName: initialValues.firstName ?? '',
      lastName: initialValues.lastName ?? '',
      email: initialValues.email ?? '',
      phone: initialValues.phone ?? '',
      password: '',
      confirmPassword: '',
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialValues]);

  const formValues = watch();
  const passwordValue = formValues.password ?? '';
  const passwordChecks = passwordCheckResults(passwordValue);

  // Continuous journey progress: step 1 fills the first half of the track as
  // its fields become valid, step 2 fills the rest (#697: a completion
  // indicator that shows how much is LEFT, not just where you are).
  const step1Done =
    Number(!!formValues.firstName?.trim()) +
    Number(!!formValues.lastName?.trim()) +
    Number(EMAIL_LIKE.test(formValues.email ?? ''));
  const step2Done =
    Number(isStrongPassword(passwordValue)) +
    Number(!!formValues.confirmPassword && formValues.confirmPassword === passwordValue);
  const fraction =
    step === 1 ? (step1Done / 3) * 0.5 : 0.5 + (step2Done / 2) * 0.5;
  const complete = fraction >= 1;

  // The fill is a full-width bar slid left by translateX (transform-only —
  // width never animates, per .impeccable.md §Motion), native-driver safe.
  const [trackWidth, setTrackWidth] = useState(0);
  const fillX = useRef(new Animated.Value(0)).current;
  const onTrackLayout = (e: LayoutChangeEvent) => {
    const w = e.nativeEvent.layout.width;
    setTrackWidth(w);
    // First paint at the correct position — no fill-then-drain flash.
    fillX.setValue(-w * (1 - fraction));
  };
  useEffect(() => {
    if (trackWidth === 0) return;
    const target = -trackWidth * (1 - fraction);
    if (reduceMotion) {
      fillX.setValue(target);
      return;
    }
    Animated.timing(fillX, {
      toValue: target,
      duration: theme.motion.duration.default,
      easing: Easing.bezier(...theme.motion.easing.state),
      useNativeDriver: true,
    }).start();
  }, [fraction, trackWidth, reduceMotion, fillX]);

  // Step transition — the incoming step fades in and settles from the side
  // it came from. Entrance easing, opacity/transform only.
  const stepAnim = useRef(new Animated.Value(1)).current;
  const stepDir = useRef(1);
  const goToStep = (next: 1 | 2) => {
    stepDir.current = next > step ? 1 : -1;
    Keyboard.dismiss();
    setError(null);
    setStep(next);
    scrollRef.current?.scrollTo({ y: 0, animated: false });
    if (reduceMotion) {
      stepAnim.setValue(1);
      return;
    }
    stepAnim.setValue(0);
    Animated.timing(stepAnim, {
      toValue: 1,
      duration: theme.motion.duration.default,
      easing: Easing.bezier(...theme.motion.easing.entrance),
      useNativeDriver: true,
    }).start();
  };

  const handleContinue = async () => {
    const ok = await trigger([...STEP1_FIELDS]);
    if (!ok) return;
    const v = getValues();
    onDraftSave?.({
      firstName: v.firstName,
      lastName: v.lastName,
      email: v.email,
      phone: v.phone,
    });
    goToStep(2);
  };

  const handleCancel = () => {
    showAlert(
      'Cancel sign-up?',
      "This clears everything you've entered so far.",
      [
        { text: 'Keep going', style: 'cancel' },
        {
          text: 'Discard',
          style: 'destructive',
          onPress: () => {
            reset();
            onCancel?.();
          },
        },
      ],
    );
  };

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

  const stepStyle = {
    opacity: stepAnim,
    transform: [
      {
        translateX: stepAnim.interpolate({
          inputRange: [0, 1],
          outputRange: [24 * stepDir.current, 0],
        }),
      },
    ],
  };

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {backdrop === 'kitchen' ? <KitchenBackdrop /> : null}
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
        style={styles.kav}
      >
        {/* Top bar: back / brand · step counter · cancel */}
        <View style={styles.topBar}>
          <View style={styles.topBarSide}>
            {step === 2 ? (
              <Pressable
                onPress={() => goToStep(1)}
                hitSlop={12}
                accessibilityRole="button"
                accessibilityLabel="Back to your details"
              >
                {({ pressed }) => (
                  <Text style={[styles.backLink, pressed && { opacity: 0.6 }]}>← Back</Text>
                )}
              </Pressable>
            ) : brand ? (
              <Text style={styles.brand}>{brand}</Text>
            ) : null}
          </View>
          <View style={styles.topBarRight}>
            <Text style={styles.stepLabel}>Step {step} of 2</Text>
            {onCancel ? (
              <Pressable
                onPress={handleCancel}
                hitSlop={12}
                accessibilityRole="button"
                accessibilityLabel="Cancel sign-up"
              >
                {({ pressed }) => (
                  <Text style={[styles.cancelGlyph, pressed && { opacity: 0.6 }]}>✕</Text>
                )}
              </Pressable>
            ) : null}
          </View>
        </View>

        {/* Journey progress — one continuous track across both steps. */}
        <View
          style={styles.progressRow}
          accessibilityRole="progressbar"
          accessibilityValue={{ min: 0, max: 100, now: Math.round(fraction * 100) }}
        >
          <View style={styles.track} onLayout={onTrackLayout}>
            {trackWidth > 0 ? (
              <Animated.View
                style={[
                  styles.trackFill,
                  {
                    backgroundColor: complete ? theme.colors.success.DEFAULT : accentColor,
                    transform: [{ translateX: fillX }],
                  },
                ]}
              />
            ) : null}
            <View style={styles.tickRow} pointerEvents="none">
              <View style={styles.tick} />
            </View>
          </View>
        </View>

        <ScrollView
          ref={scrollRef}
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          <Animated.View style={stepStyle}>
            <Text style={styles.title}>{step === 1 ? title : step2Title}</Text>
            <Text style={styles.subtitle}>{step === 1 ? subtitle : step2Subtitle}</Text>

            {error ? (
              <View style={styles.errorBanner}>
                <Text style={styles.errorText}>{error}</Text>
              </View>
            ) : null}

            {step === 1 ? (
              <>
                {incentive ? (
                  <View style={styles.incentiveCard}>
                    <View style={styles.incentiveBadge}>
                      <Text style={styles.incentiveRupee}>₹</Text>
                    </View>
                    <View style={styles.incentiveCopy}>
                      <Text style={styles.incentiveTitle}>{incentive.title}</Text>
                      <Text style={styles.incentiveBody}>{incentive.body}</Text>
                    </View>
                  </View>
                ) : null}

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
                          accentColor={accent}
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
                          accentColor={accent}
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
                      accentColor={accent}
                    />
                  )}
                />

                <Controller
                  control={control}
                  name="phone"
                  render={({ field: { onChange, onBlur, value } }) => (
                    <Input
                      label="Phone (optional)"
                      placeholder={phoneRule.example}
                      keyboardType="number-pad"
                      autoComplete="tel"
                      // Hard guard: strip non-digits and cap at the country's national
                      // length, so the user physically cannot type an extra digit.
                      // maxLength backs this up at the native TextInput level.
                      maxLength={phoneRule.length}
                      onBlur={onBlur}
                      onChangeText={(text) => onChange(sanitizePhoneInput(text, phoneCountry))}
                      value={value ?? ''}
                      error={errors.phone?.message}
                      helper={`${phoneRule.dialCode} · ${phoneRule.length}-digit mobile. Only for order issues — never shared.`}
                      accentColor={accent}
                    />
                  )}
                />

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
                    <Pressable
                      onPress={onNavigateToLogin}
                      hitSlop={8}
                      accessibilityRole="link"
                      accessibilityLabel="Already have an account? Sign in"
                    >
                      <Text style={styles.signinPrompt}>
                        Already have an account?{' '}
                        <Text
                          style={[
                            styles.signinCTA,
                            resolvedLinkColor
                              ? { color: resolvedLinkColor, textDecorationLine: 'none' }
                              : null,
                          ]}
                        >
                          Sign in
                        </Text>
                      </Text>
                    </Pressable>
                  </View>
                ) : null}
              </>
            ) : (
              <>
                <Controller
                  control={control}
                  name="password"
                  render={({ field: { onChange, onBlur, value } }) => (
                    <Input
                      label="Password"
                      placeholder="Create a strong password"
                      secureTextEntry
                      passwordPeek
                      autoComplete="new-password"
                      onBlur={onBlur}
                      onChangeText={onChange}
                      value={value}
                      error={errors.password?.message}
                      accentColor={accent}
                    />
                  )}
                />

                {/* Live password-requirement checklist — turns green as each rule is met.
                    Only appears once the user starts typing, so it guides rather than nags. */}
                {passwordValue.length > 0 ? (
                  <View style={styles.pwChecklist}>
                    {passwordChecks.map((c) => (
                      <View key={c.id} style={styles.pwCheckRow}>
                        <Text
                          style={[styles.pwCheckMark, c.met ? styles.pwCheckMarkMet : null]}
                          accessibilityLabel={c.met ? 'met' : 'not met'}
                        >
                          {c.met ? '✓' : '○'}
                        </Text>
                        <Text style={[styles.pwCheckLabel, c.met ? styles.pwCheckLabelMet : null]}>
                          {c.label}
                        </Text>
                      </View>
                    ))}
                  </View>
                ) : null}

                <Controller
                  control={control}
                  name="confirmPassword"
                  render={({ field: { onChange, onBlur, value } }) => (
                    <Input
                      label="Confirm password"
                      placeholder="Re-enter your password"
                      secureTextEntry
                      passwordPeek
                      autoComplete="new-password"
                      onBlur={onBlur}
                      onChangeText={onChange}
                      value={value}
                      error={errors.confirmPassword?.message}
                      accentColor={accent}
                    />
                  )}
                />
              </>
            )}
          </Animated.View>
        </ScrollView>

        {/* Sticky CTA — always visible, lifts above the keyboard on iOS. */}
        <View style={styles.ctaWrap}>
          {step === 1 ? (
            <Button label="Continue" onPress={handleContinue} accentColor={accent} />
          ) : (
            <Button
              label={isSubmitting ? 'Creating account…' : 'Create account'}
              onPress={handleSubmit(onSubmit)}
              loading={isSubmitting}
              disabled={isSubmitting}
              accentColor={accent}
            />
          )}
        </View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },
  kav: { flex: 1 },

  topBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[6],
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[3],
  },
  topBarSide: { flexShrink: 1 },
  topBarRight: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[4],
  },
  brand: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    letterSpacing: 1.5,
    textTransform: 'uppercase',
  },
  backLink: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  stepLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.5,
    fontVariant: ['tabular-nums'],
  },
  cancelGlyph: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: theme.colors.ink.soft,
    paddingHorizontal: theme.spacing[1],
  },

  progressRow: {
    paddingHorizontal: theme.spacing[6],
    paddingBottom: theme.spacing[5],
  },
  track: {
    height: 4,
    borderRadius: 2,
    backgroundColor: theme.colors.mist.DEFAULT,
    overflow: 'hidden',
    justifyContent: 'center',
  },
  trackFill: {
    position: 'absolute',
    top: 0,
    bottom: 0,
    left: 0,
    right: 0,
    borderRadius: 2,
  },
  // Midpoint tick keeps the "2 discrete steps" reading on the continuous track.
  tickRow: {
    flexDirection: 'row',
    justifyContent: 'center',
    alignItems: 'center',
    height: '100%',
  },
  tick: {
    width: 1,
    height: '100%',
    backgroundColor: theme.colors.paper,
    opacity: 0.9,
  },

  scroll: { flex: 1 },
  scrollContent: {
    paddingHorizontal: theme.spacing[6],
    paddingBottom: theme.spacing[8],
  },

  title: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.display.size,
    lineHeight:
      theme.typography.size.display.size *
      theme.typography.size.display.lineHeight,
    letterSpacing: theme.typography.size.display.letterSpacing,
    color: theme.colors.ink.DEFAULT,
    marginTop: theme.spacing[2],
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

  errorBanner: {
    backgroundColor: theme.colors.destructive.tint,
    borderLeftWidth: 3,
    borderLeftColor: theme.colors.destructive.DEFAULT,
    borderRadius: theme.radius.sm,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
    marginBottom: theme.spacing[4],
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

  incentiveCard: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    marginBottom: theme.spacing[5],
  },
  incentiveBadge: {
    width: 36,
    height: 36,
    borderRadius: theme.radius.full,
    backgroundColor: theme.colors.success.tint,
    alignItems: 'center',
    justifyContent: 'center',
  },
  incentiveRupee: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: theme.colors.success.soft,
  },
  incentiveCopy: { flex: 1 },
  incentiveTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    marginBottom: 2,
  },
  incentiveBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.label.size,
    lineHeight: theme.typography.size.label.size * 1.4,
    color: theme.colors.ink.soft,
  },

  pwChecklist: {
    marginBottom: theme.spacing[4],
    gap: theme.spacing[1],
    paddingLeft: theme.spacing[1],
  },
  pwCheckRow: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[2] },
  pwCheckMark: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    width: 16,
    textAlign: 'center',
    color: theme.colors.ink.muted,
  },
  pwCheckMarkMet: { color: theme.colors.success.DEFAULT },
  pwCheckLabel: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.muted },
  pwCheckLabelMet: { color: theme.colors.ink.soft },

  divider: {
    flexDirection: 'row',
    alignItems: 'center',
    marginTop: theme.spacing[2],
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

  ctaWrap: {
    paddingHorizontal: theme.spacing[6],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[4],
    borderTopWidth: 1,
    borderTopColor: theme.colors.mist.DEFAULT,
    backgroundColor: theme.colors.paper,
  },
});
