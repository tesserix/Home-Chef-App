import { router } from 'expo-router';
import { ForgotPasswordScreen } from '@homechef/mobile-shared/screens';
import { forgotPassword } from '@homechef/mobile-shared/api';
import { api } from '../../lib/api';

export default function ForgotPasswordPage() {
  return (
    <ForgotPasswordScreen
      onForgotPassword={async ({ email }) => {
        await forgotPassword(api, { email });
        // Screen shows success message internally after resolve (T-03-06: generic response)
      }}
      onNavigateToLogin={() => router.back()}
    />
  );
}
