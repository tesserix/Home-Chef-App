import { router } from 'expo-router';
import { ForgotPasswordScreen } from '@homechef/mobile-shared/screens';
import { sendPasswordResetEmail } from '@homechef/mobile-shared/auth';

export default function ForgotPasswordPage() {
  return (
    <ForgotPasswordScreen
      brand="Fe3dr · Vendor"
      onForgotPassword={async ({ email }) => {
        // Firebase handles password reset email delivery (GIP-backed).
        // Screen shows success message internally after resolve (T-03-06: generic response).
        await sendPasswordResetEmail(email);
      }}
      onNavigateToLogin={() => router.back()}
    />
  );
}
