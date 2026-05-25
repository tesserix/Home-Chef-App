import * as Location from 'expo-location';
import * as TaskManager from 'expo-task-manager';
import * as SecureStore from 'expo-secure-store';

export const LOCATION_TASK_NAME = 'homechef-delivery-tracking';

// The SecureStore key for the JWT access token — same key used by the auth store
const JWT_STORAGE_KEY = 'access_token';

async function updateDriverLocationAPI(
  latitude: number,
  longitude: number,
): Promise<void> {
  // Must use SecureStore + fetch — NO React context available in background
  const token = await SecureStore.getItemAsync(JWT_STORAGE_KEY);
  if (!token) return; // Driver not logged in — skip silently

  const apiBase =
    process.env.EXPO_PUBLIC_API_URL ?? 'https://api.homechef.app';
  try {
    const response = await fetch(`${apiBase}/api/v1/delivery/location`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-Auth-Token': token,
      },
      body: JSON.stringify({ latitude, longitude }),
    });
    if (!response.ok && response.status !== 429) {
      console.warn('[GPS] Location update failed:', response.status);
    }
    // 429 = rate limited — expected; log silently and continue
  } catch (err) {
    console.warn('[GPS] Location update error:', err);
  }
}

// MUST be defined at module level (not inside a component or hook)
// This file must be imported in _layout.tsx so defineTask runs at app startup
TaskManager.defineTask(
  LOCATION_TASK_NAME,
  async ({ data, error }: TaskManager.TaskManagerTaskBody) => {
    if (error) {
      console.error('[GPS] Background task error:', error);
      return;
    }
    if (!data) return;
    const { locations } = data as { locations: Location.LocationObject[] };
    const latest = locations[locations.length - 1];
    if (!latest) return;
    await updateDriverLocationAPI(
      latest.coords.latitude,
      latest.coords.longitude,
    );
  },
);

export async function startTracking(): Promise<void> {
  const isAlreadyRunning = await Location.hasStartedLocationUpdatesAsync(
    LOCATION_TASK_NAME,
  ).catch(() => false);
  if (isAlreadyRunning) return;

  await Location.startLocationUpdatesAsync(LOCATION_TASK_NAME, {
    accuracy: Location.Accuracy.Balanced,
    timeInterval: 15000, // 15 seconds between updates (D-02)
    distanceInterval: 30, // Minimum 30 metres movement (avoids updates while stationary)
    foregroundService: {
      // Android only — required for background on Android (D-05)
      notificationTitle: 'HomeChef Delivery',
      notificationBody: 'Tracking your location during delivery',
      notificationColor: '#C2410C',
    },
    pausesUpdatesAutomatically: false,
    showsBackgroundLocationIndicator: true, // iOS blue status bar indicator
  });
}

export async function stopTracking(): Promise<void> {
  const isRunning = await Location.hasStartedLocationUpdatesAsync(
    LOCATION_TASK_NAME,
  ).catch(() => false);
  if (isRunning) {
    await Location.stopLocationUpdatesAsync(LOCATION_TASK_NAME);
  }
}
