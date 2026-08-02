import { Stack } from 'expo-router';

// OnboardingScaffold owns every piece of wizard chrome (step counter,
// progress track, back, cancel) — a native header on top of it double-prints
// "Step N of M" and wastes a full row of vertical space.
export default function OnboardingLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        gestureEnabled: false,
      }}
    />
  );
}
