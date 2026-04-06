import { Stack } from 'expo-router';

export default function OnboardingLayout() {
  return (
    <Stack screenOptions={{ headerShown: false }}>
      <Stack.Screen name="user-info" options={{ gestureEnabled: false }} />
      <Stack.Screen name="address" />
      <Stack.Screen name="preferences" />
    </Stack>
  );
}
