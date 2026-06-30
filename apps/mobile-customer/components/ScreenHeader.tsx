// Shared sub-screen header: a back affordance + centered title.
//
// headerShown is false app-wide (see app/_layout.tsx), so every pushed screen
// must draw its own header — otherwise there is no way back (the Wallet screen
// shipped with exactly this gap). Use this on every screen that is *pushed*
// onto the stack (Wallet, Rewards, Social, Catering, …) so the back control,
// title typography, and spacing stay identical across the app.

import type { ReactNode } from 'react';
import { Pressable, Text, View } from 'react-native';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';

interface ScreenHeaderProps {
  title: string;
  /** Optional element pinned to the right (an action). Absent → symmetric spacer. */
  right?: ReactNode;
  /** Hairline divider beneath the header. Default true. */
  border?: boolean;
  /** Override the back action. Default: pop the stack, or fall back to the tabs. */
  onBack?: () => void;
}

export function ScreenHeader({ title, right, border = true, onBack }: ScreenHeaderProps) {
  const handleBack =
    onBack ??
    (() => {
      // A deep link can land on this screen with nothing to pop back to.
      if (router.canGoBack()) router.back();
      else router.replace('/(tabs)');
    });

  return (
    <View
      className={`flex-row items-center justify-between bg-canvas px-4 py-3 ${
        border ? 'border-b border-hairline' : ''
      }`}
    >
      <Pressable
        onPress={handleBack}
        hitSlop={10}
        accessibilityRole="button"
        accessibilityLabel="Go back"
        className="-ml-1 p-1"
      >
        <ChevronLeft size={26} color={customerColors.charcoal.DEFAULT} />
      </Pressable>

      <Text
        className="flex-1 px-2 text-center text-xl font-bold text-charcoal font-display"
        numberOfLines={1}
      >
        {title}
      </Text>

      {/* Right slot — fixed 26px width keeps the title optically centered. */}
      <View style={{ width: 26 }} className="items-end">
        {right}
      </View>
    </View>
  );
}
