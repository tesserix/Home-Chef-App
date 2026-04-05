import { useEffect } from 'react';
import { Platform } from 'react-native';
import { router } from 'expo-router';
import { GoogleSignin } from '@react-native-google-signin/google-signin';
import * as AppleAuthentication from 'expo-apple-authentication';
import { LoginScreen } from '@homechef/mobile-shared/screens';
import { loginWithEmail, oauthLogin } from '@homechef/mobile-shared/api';
import { getRawFCMToken, registerDeviceToken, authenticateWithBiometrics } from '@homechef/mobile-shared/hooks';
import { useAuthStore } from '../../store/auth-store';
import { api } from '../../lib/api';

export default function LoginPage() {
  const { setAuthResponse, biometricsEnabled, accessToken } = useAuthStore();

  useEffect(() => {
    GoogleSignin.configure({
      webClientId: process.env.EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID!,
    });
  }, []);

  const handleGoogleSignIn = async () => {
    await GoogleSignin.hasPlayServices({ showPlayServicesUpdateDialog: true });
    const { data } = await GoogleSignin.signIn();
    if (!data?.idToken) throw new Error('Google sign-in failed: no ID token');
    const response = await oauthLogin(api, { provider: 'google', token: data.idToken });
    await setAuthResponse(response);
    try {
      const fcmToken = await getRawFCMToken();
      if (fcmToken) await registerDeviceToken(api, fcmToken);
    } catch { /* non-fatal */ }
    router.replace('/(tabs)');
  };

  const handleAppleSignIn = async () => {
    const cred = await AppleAuthentication.signInAsync({
      requestedScopes: [
        AppleAuthentication.AppleAuthenticationScope.FULL_NAME,
        AppleAuthentication.AppleAuthenticationScope.EMAIL,
      ],
    });
    if (!cred.identityToken) throw new Error('Apple sign-in failed: no identity token');
    const response = await oauthLogin(api, { provider: 'apple', token: cred.identityToken });
    await setAuthResponse(response);
    try {
      const fcmToken = await getRawFCMToken();
      if (fcmToken) await registerDeviceToken(api, fcmToken);
    } catch { /* non-fatal */ }
    router.replace('/(tabs)');
  };

  const handleBiometricLogin = async () => {
    const success = await authenticateWithBiometrics();
    if (!success) throw new Error('Biometric authentication failed');
    // Token is already in secure store from previous login — just confirm it's still valid
    const { accessToken: token } = useAuthStore.getState();
    if (!token) throw new Error('No saved session found. Please log in with email.');
    // Auth guard in _layout.tsx will detect isAuthenticated=true and redirect
    router.replace('/(tabs)');
  };

  return (
    <LoginScreen
      title="Welcome back"
      onLogin={async ({ email, password }) => {
        const response = await loginWithEmail(api, { email, password });
        await setAuthResponse(response);
        // Register FCM token after auth (D-09: raw FCM token)
        try {
          const fcmToken = await getRawFCMToken();
          if (fcmToken) await registerDeviceToken(api, fcmToken);
        } catch {
          // Non-fatal: push registration failure should not block login
        }
        router.replace('/(tabs)');
      }}
      onNavigateToRegister={() => router.push('/(auth)/register')}
      onGoogleSignIn={handleGoogleSignIn}
      onAppleSignIn={Platform.OS === 'ios' ? handleAppleSignIn : undefined}
      onBiometricLogin={biometricsEnabled ? handleBiometricLogin : undefined}
    />
  );
}
