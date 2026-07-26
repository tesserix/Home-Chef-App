import { router } from 'expo-router';

import { ForgotPasswordScreen } from '@homechef/mobile-shared/screens';
import { requestPasswordReset } from '@homechef/mobile-shared/auth';
import { customerColors } from '@homechef/mobile-shared/theme';

const API_URL = process.env.EXPO_PUBLIC_API_URL ?? '';

export default function ForgotPasswordPage() {
  return (
    <ForgotPasswordScreen
      brand="Fe3dr"
      accent={customerColors.coral.DEFAULT}
      // THE SPEC §2 AA micro-adjustment: link text uses coral-pressed
      // (#E00B41), not the coral fill (#FF385C), which fails AA at link/body
      // text size. Fills (CTA, focus rings) stay coral via `accent` above.
      linkColor={customerColors.coral.pressed}
      onForgotPassword={async ({ email }) => {
        // Routed through our own API rather than Firebase: Firebase delivers
        // from an unauthenticated firebaseapp.com address that Gmail files as
        // spam, branded with the GCP project name. Ours sends from the
        // verified platform sender, with a single-use link that expires in 15
        // minutes. The "customer" tenant is required — accounts are tenant-scoped,
        // so the wrong one silently finds nothing.
        await requestPasswordReset(API_URL, email, 'customer');
      }}
      onNavigateToLogin={() => router.back()}
    />
  );
}
