import { useEffect, useRef, useState } from 'react';
import type { MutableRefObject, RefObject } from 'react';
import {
  View,
  Text,
  TextInput,
  Pressable,
  Alert,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useForm, Controller, type FieldErrors } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { router } from 'expo-router';
import { customerColors } from '@homechef/mobile-shared/theme';
import { api } from '../../lib/api';
import { useAuthStore } from '../../store/auth-store';
import { useCustomerOnboardingStore } from '../../store/onboarding-store';

const schema = z.object({
  firstName: z.string().min(2, 'First name must be at least 2 characters'),
  lastName: z.string().min(2, 'Last name must be at least 2 characters'),
  phone: z
    .string()
    .regex(/^[6-9]\d{9}$/, 'Enter a valid 10-digit Indian mobile number'),
});

type UserInfoForm = z.infer<typeof schema>;

// Android ripple tints — translucent tokens derived from existing colours,
// never a new literal colour (matches the ChefCard `withAlpha` convention).
const CTA_RIPPLE = `${customerColors.canvas}33`;
const GHOST_RIPPLE = `${customerColors.coral.DEFAULT}22`;

type UserInfoField = 'firstName' | 'lastName' | 'phone';

// 2px coral focus ring (falls back to the destructive border on error),
// matching the Input primitive's focus treatment (Task 1).
function fieldBorderStyle(hasError: boolean, isFocused: boolean) {
  if (hasError) {
    return { borderWidth: 1.5, borderColor: customerColors.destructive.DEFAULT };
  }
  if (isFocused) {
    return { borderWidth: 2, borderColor: customerColors.coral.DEFAULT };
  }
  return { borderWidth: 0, borderColor: 'transparent' };
}

export default function UserInfoScreen() {
  // Prefill from whatever the user already gave at sign-up (email signup
  // captures name + phone into the auth store; social sign-up leaves them
  // blank). Saves re-typing details we already have.
  const user = useAuthStore((s) => s.user);
  const draft = useCustomerOnboardingStore();
  const {
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<UserInfoForm>({
    resolver: zodResolver(schema),
    // Prefer the saved draft (so resuming / coming back shows what was typed),
    // then fall back to whatever sign-up captured into the auth store.
    defaultValues: {
      firstName: draft.firstName || (user?.firstName ?? ''),
      lastName: draft.lastName || (user?.lastName ?? ''),
      phone: draft.phone || (user?.phone ?? ''),
    },
  });

  const email = user?.email ?? '';
  const [focusedField, setFocusedField] = useState<UserInfoField | null>(null);

  // R14 — scroll to + focus the first invalid field on a failed submit.
  const scrollRef = useRef<ScrollView>(null);
  const firstNameY = useRef(0);
  const lastNameY = useRef(0);
  const phoneY = useRef(0);
  const firstNameRef = useRef<TextInput>(null);
  const lastNameRef = useRef<TextInput>(null);
  const phoneRef = useRef<TextInput>(null);

  function onInvalid(errs: FieldErrors<UserInfoForm>) {
    const order: { key: keyof UserInfoForm; y: MutableRefObject<number>; input: RefObject<TextInput | null> }[] = [
      { key: 'firstName', y: firstNameY, input: firstNameRef },
      { key: 'lastName', y: lastNameY, input: lastNameRef },
      { key: 'phone', y: phoneY, input: phoneRef },
    ];
    const first = order.find((f) => errs[f.key]);
    if (!first) return;
    scrollRef.current?.scrollTo({ y: Math.max(0, first.y.current - 16), animated: true });
    setTimeout(() => first.input.current?.focus(), 320);
  }
  const [emailVerified, setEmailVerified] = useState(draft.emailVerified);
  const [otpSent, setOtpSent] = useState(false);
  const [code, setCode] = useState('');
  const [codeFocused, setCodeFocused] = useState(false);
  const [sending, setSending] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [cooldown, setCooldown] = useState(0);

  useEffect(() => {
    if (cooldown <= 0) return;
    const id = setInterval(() => setCooldown((s) => (s > 0 ? s - 1 : 0)), 1000);
    return () => clearInterval(id);
  }, [cooldown]);

  const apiError = (err: unknown, fallback: string): string => {
    const e = err as { response?: { data?: { error?: string } } };
    return e?.response?.data?.error || fallback;
  };

  const sendCode = async (): Promise<void> => {
    if (sending || cooldown > 0 || !email) return;
    setSending(true);
    try {
      await api.post('/v1/account/email/otp/request', { email });
      setOtpSent(true);
      setCooldown(60);
    } catch (err) {
      Alert.alert('Verify email', apiError(err, "Couldn't send the code. Please try again."));
    } finally {
      setSending(false);
    }
  };

  const confirmCode = async (): Promise<void> => {
    if (verifying || code.length !== 6) return;
    setVerifying(true);
    try {
      await api.post('/v1/account/email/otp/verify', { email, code });
      setEmailVerified(true);
      draft.update({ emailVerified: true });
    } catch (err) {
      Alert.alert('Verify email', apiError(err, 'That code is incorrect or expired.'));
    } finally {
      setVerifying(false);
    }
  };

  const onSubmit = (data: UserInfoForm) => {
    if (!emailVerified) {
      Alert.alert('Verify email', 'Please verify your email before continuing.');
      return;
    }
    draft.update({
      firstName: data.firstName,
      lastName: data.lastName,
      phone: data.phone,
      emailVerified: true,
    });
    router.push('/(onboarding)/address');
  };

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      <KeyboardAvoidingView
        className="flex-1"
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      >
        <ScrollView
          ref={scrollRef}
          className="flex-1"
          contentContainerStyle={{ padding: 24, paddingTop: 40 }}
          keyboardShouldPersistTaps="handled"
        >
          {/* ── Step progress ── */}
          <Text className="text-[13px] text-charcoal-soft mb-2">
            Step 1 of 3
          </Text>
          <View className="h-1 bg-hairline rounded-full mb-8 overflow-hidden">
            <View className="h-1 bg-coral rounded-full" style={{ width: '33%' }} />
          </View>

          {/* ── Heading ── */}
          <Text className="text-[26px] font-bold text-charcoal tracking-tight font-display mb-2">
            Tell us about yourself
          </Text>
          <Text className="text-[15px] text-charcoal-soft mb-8">
            We need a few details to set up your account.
          </Text>

          {/* ── First Name ── */}
          <View onLayout={(e) => { firstNameY.current = e.nativeEvent.layout.y; }}>
            <Text className="text-sm font-medium text-charcoal mb-1">
              First Name
            </Text>
            <Controller
              control={control}
              name="firstName"
              render={({ field: { onChange, onBlur, value } }) => (
                <TextInput
                  ref={firstNameRef}
                  className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                  style={fieldBorderStyle(Boolean(errors.firstName), focusedField === 'firstName')}
                  placeholder="Enter your first name"
                  placeholderTextColor={customerColors.charcoal.soft}
                  onFocus={() => setFocusedField('firstName')}
                  onBlur={() => {
                    setFocusedField(null);
                    onBlur();
                  }}
                  onChangeText={onChange}
                  value={value}
                  autoCapitalize="words"
                  returnKeyType="next"
                  accessibilityLabel="First name"
                />
              )}
            />
            {errors.firstName ? (
              <Text className="text-xs text-destructive mb-3">
                {errors.firstName.message}
              </Text>
            ) : (
              <View className="mb-4" />
            )}
          </View>

          {/* ── Last Name ── */}
          <View onLayout={(e) => { lastNameY.current = e.nativeEvent.layout.y; }}>
            <Text className="text-sm font-medium text-charcoal mb-1">
              Last Name
            </Text>
            <Controller
              control={control}
              name="lastName"
              render={({ field: { onChange, onBlur, value } }) => (
                <TextInput
                  ref={lastNameRef}
                  className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                  style={fieldBorderStyle(Boolean(errors.lastName), focusedField === 'lastName')}
                  placeholder="Enter your last name"
                  placeholderTextColor={customerColors.charcoal.soft}
                  onFocus={() => setFocusedField('lastName')}
                  onBlur={() => {
                    setFocusedField(null);
                    onBlur();
                  }}
                  onChangeText={onChange}
                  value={value}
                  autoCapitalize="words"
                  returnKeyType="next"
                  accessibilityLabel="Last name"
                />
              )}
            />
            {errors.lastName ? (
              <Text className="text-xs text-destructive mb-3">
                {errors.lastName.message}
              </Text>
            ) : (
              <View className="mb-4" />
            )}
          </View>

          {/* ── Phone ── */}
          <View onLayout={(e) => { phoneY.current = e.nativeEvent.layout.y; }}>
            <Text className="text-sm font-medium text-charcoal mb-1">
              Phone Number
            </Text>
            <Controller
              control={control}
              name="phone"
              render={({ field: { onChange, onBlur, value } }) => (
                <TextInput
                  ref={phoneRef}
                  className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                  style={fieldBorderStyle(Boolean(errors.phone), focusedField === 'phone')}
                  placeholder="10-digit mobile number"
                  placeholderTextColor={customerColors.charcoal.soft}
                  onFocus={() => setFocusedField('phone')}
                  onBlur={() => {
                    setFocusedField(null);
                    onBlur();
                  }}
                  onChangeText={onChange}
                  value={value}
                  keyboardType="phone-pad"
                  maxLength={10}
                  returnKeyType="done"
                  accessibilityLabel="Phone number"
                />
              )}
            />
            {errors.phone ? (
              <Text className="text-xs text-destructive mb-3">
                {errors.phone.message}
              </Text>
            ) : (
              <View className="mb-4" />
            )}
          </View>

          {/* ── Email verification ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">Email</Text>
          <View className="h-12 bg-surface-soft rounded-lg px-4 flex-row items-center justify-between mb-1">
            <Text className="text-base text-charcoal-soft flex-1" numberOfLines={1}>
              {email}
            </Text>
            {emailVerified && (
              <Text className="text-xs font-semibold text-coral ml-2">Verified ✓</Text>
            )}
          </View>

          {!emailVerified &&
            (!otpSent ? (
              <Pressable
                onPress={() => void sendCode()}
                disabled={sending || !email}
                accessibilityRole="button"
                accessibilityLabel="Send verification code"
                android_ripple={{ color: GHOST_RIPPLE, borderless: false }}
              >
                {({ pressed }) => (
                  <View
                    className={`rounded-lg min-h-[44px] items-center justify-center mt-2 mb-4 border border-coral ${
                      pressed ? 'opacity-90' : ''
                    }`}
                  >
                    <Text className="text-coral font-semibold text-sm">
                      {sending ? 'Sending…' : 'Send verification code'}
                    </Text>
                  </View>
                )}
              </Pressable>
            ) : (
              <View className="mt-2 mb-4">
                <Text className="text-[13px] text-charcoal-soft mb-2">
                  Enter the 6-digit code we emailed you.
                </Text>
                <TextInput
                  className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal tracking-[8px] mb-2"
                  style={fieldBorderStyle(false, codeFocused)}
                  placeholder="000000"
                  placeholderTextColor={customerColors.charcoal.soft}
                  value={code}
                  onChangeText={(v) => setCode(v.replace(/\D/g, '').slice(0, 6))}
                  onFocus={() => setCodeFocused(true)}
                  onBlur={() => setCodeFocused(false)}
                  keyboardType="number-pad"
                  maxLength={6}
                  accessibilityLabel="Verification code"
                />
                <Pressable
                  onPress={() => void confirmCode()}
                  disabled={verifying || code.length !== 6}
                  accessibilityRole="button"
                  accessibilityLabel="Verify code"
                  android_ripple={{ color: CTA_RIPPLE, borderless: false }}
                >
                  {({ pressed }) => (
                    <View
                      className={`rounded-lg min-h-[44px] items-center justify-center bg-coral ${
                        pressed || verifying || code.length !== 6 ? 'opacity-90' : ''
                      }`}
                    >
                      <Text className="text-canvas font-semibold text-sm">
                        {verifying ? 'Verifying…' : 'Verify code'}
                      </Text>
                    </View>
                  )}
                </Pressable>
                <Pressable
                  onPress={() => void sendCode()}
                  disabled={cooldown > 0 || sending}
                  accessibilityRole="button"
                  accessibilityLabel={cooldown > 0 ? `Resend code in ${cooldown} seconds` : 'Resend code'}
                  hitSlop={8}
                  className="mt-2 items-center min-h-[44px] justify-center"
                  android_ripple={{ color: GHOST_RIPPLE, borderless: false }}
                >
                  {({ pressed }) => (
                    <Text
                      className={`text-[13px] text-charcoal-soft ${
                        pressed && Platform.OS === 'ios' ? 'opacity-70' : ''
                      }`}
                    >
                      {cooldown > 0 ? `Resend in ${cooldown}s` : 'Resend code'}
                    </Text>
                  )}
                </Pressable>
              </View>
            ))}

          {/* ── Primary CTA ── */}
          {/* iOS Pressable pattern: visual styles on inner View */}
          <Pressable
            onPress={() => void handleSubmit(onSubmit, onInvalid)()}
            accessibilityRole="button"
            accessibilityLabel="Continue to address"
            android_ripple={{ color: CTA_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                className={`rounded-lg min-h-[52px] items-center justify-center mt-4 ${
                  pressed ? 'opacity-90' : ''
                } bg-coral`}
              >
                <Text className="text-canvas font-semibold text-base">
                  Continue
                </Text>
              </View>
            )}
          </Pressable>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
