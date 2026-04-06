import { Stack } from 'expo-router';

export default function OnboardingLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: true,
        headerBackVisible: false,
        gestureEnabled: false,
      }}
    >
      <Stack.Screen
        name="personal-info"
        options={{ title: 'Step 1 of 6', gestureEnabled: false }}
      />
      <Stack.Screen
        name="kitchen-details"
        options={{ title: 'Step 2 of 6', gestureEnabled: false }}
      />
      <Stack.Screen
        name="operations"
        options={{ title: 'Step 3 of 6', gestureEnabled: false }}
      />
      <Stack.Screen
        name="documents"
        options={{ title: 'Step 4 of 6', gestureEnabled: false }}
      />
      <Stack.Screen
        name="policies"
        options={{ title: 'Step 5 of 6', gestureEnabled: false }}
      />
      <Stack.Screen
        name="review"
        options={{ title: 'Step 6 of 6', gestureEnabled: false }}
      />
      <Stack.Screen
        name="pending"
        options={{ title: 'Application Status', gestureEnabled: false }}
      />
    </Stack>
  );
}
