import { Stack, usePathname } from 'expo-router';

function getStepTitle(pathname: string): string {
  if (pathname.includes('/personal')) return 'Step 1 of 6';
  if (pathname.includes('/vehicle')) return 'Step 2 of 6';
  if (pathname.includes('/documents')) return 'Step 3 of 6';
  if (pathname.includes('/payout')) return 'Step 4 of 6';
  if (pathname.includes('/subscription')) return 'Step 5 of 6';
  if (pathname.includes('/review')) return 'Step 6 of 6';
  return 'Driver Onboarding';
}

export default function OnboardingLayout() {
  const pathname = usePathname();
  const isFirstStep = pathname.includes('/personal');

  return (
    <Stack
      screenOptions={{
        headerShown: true,
        headerTitle: getStepTitle(pathname),
        headerTitleAlign: 'center',
        gestureEnabled: !isFirstStep,
      }}
    />
  );
}
