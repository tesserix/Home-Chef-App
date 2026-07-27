// GuestGate — what a browsing-without-an-account user sees on the tabs that
// cannot work without one (Orders, Plans, Saved, Profile).
//
// App Review guideline 5.1.1(iv) draws the line at "directly relevant to the
// core functionality": browsing chefs and menus is not account-based, an order
// history is. So those tabs stay reachable — hiding them would make the app
// look broken — and explain themselves instead of erroring or showing an empty
// list a guest cannot fill.

import { Pressable, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import type { LucideIcon } from 'lucide-react-native';

import { customerColors } from '@homechef/mobile-shared/theme';

interface GuestGateProps {
  /** Icon for the tab this is standing in for. */
  icon: LucideIcon;
  /** e.g. "Your orders". */
  title: string;
  /** One line on what an account gets them here. */
  body: string;
}

export function GuestGate({ icon: Icon, title, body }: GuestGateProps) {
  function goToSignIn(): void {
    // Guest mode is deliberately left alone — see the note in useRequireAccount.
    // The root gate no longer treats "not a guest" as "show me login", so
    // clearing it here would only bounce back through guest on the next render.
    router.push('/(auth)/login');
  }

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      <View className="flex-1 items-center justify-center px-8 gap-4">
        <View className="w-20 h-20 rounded-full bg-surface-soft items-center justify-center">
          <Icon size={32} color={customerColors.charcoal.soft} />
        </View>

        <View className="items-center gap-2">
          <Text className="text-xl font-bold text-charcoal text-center font-display">{title}</Text>
          <Text className="text-sm text-charcoal-soft text-center leading-5">{body}</Text>
        </View>

        <Pressable
          onPress={goToSignIn}
          accessibilityRole="button"
          accessibilityLabel="Sign in or create an account"
          className="mt-2 min-h-[48px] px-6 items-center justify-center rounded-lg bg-coral"
        >
          <Text className="text-base font-semibold text-white">Sign in or sign up</Text>
        </Pressable>

        <Pressable
          onPress={() => router.replace('/(tabs)')}
          accessibilityRole="button"
          accessibilityLabel="Keep browsing without an account"
          className="min-h-[44px] px-4 items-center justify-center"
        >
          <Text className="text-sm font-medium text-charcoal-soft">Keep browsing</Text>
        </Pressable>
      </View>
    </SafeAreaView>
  );
}
