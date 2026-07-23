import { router } from 'expo-router';
import { ForgotPasswordScreen } from '@homechef/mobile-shared/screens';
import { sendPasswordResetEmail } from '@homechef/mobile-shared/auth';
import { customerColors } from '@homechef/mobile-shared/theme';

export default function ForgotPasswordPage() {
  return (
    <ForgotPasswordScreen
      brand="Fe3dr"
      accent={customerColors.coral.DEFAULT}
      onForgotPassword={async ({ email }) => {
        // Firebase (GIP-backed) sends the reset email. The screen shows a
        // generic "check your inbox" success internally (anti-enumeration).
        await sendPasswordResetEmail(email);
      }}
      onNavigateToLogin={() => router.back()}
    />
  );
}
