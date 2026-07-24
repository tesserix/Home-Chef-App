import { router } from 'expo-router';
import { ForgotPasswordScreen } from '@homechef/mobile-shared/screens';
import { sendPasswordResetEmail } from '@homechef/mobile-shared/auth';
import { customerColors } from '@homechef/mobile-shared/theme';

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
        // Firebase (GIP-backed) sends the reset email. The screen shows a
        // generic "check your inbox" success internally (anti-enumeration).
        await sendPasswordResetEmail(email);
      }}
      onNavigateToLogin={() => router.back()}
    />
  );
}
