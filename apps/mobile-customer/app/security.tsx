import { SecuritySettingsScreen } from '@homechef/mobile-shared/mfa';
import { customerColors } from '@homechef/mobile-shared/theme';

import { api } from '../lib/api';

// Security settings. The screen itself lives in mobile-shared so all four apps
// present the same two-factor setup rather than four near-copies that drift.
//
// onVerifyPhone is deliberately not passed yet: the phone factor needs this
// app's Firebase phone-verification handshake wired first, and the shared
// screen hides the phone option entirely when it is absent. Better a working
// email-only screen than a phone button that dead-ends.
export default function SecurityScreen() {
  return <SecuritySettingsScreen api={api} accentColor={customerColors.coral.DEFAULT} />;
}
