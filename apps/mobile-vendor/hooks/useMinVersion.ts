import { useQuery } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { AppState } from 'react-native';
import { appPlatform, appVersion, semverLess } from '../lib/app-version';

interface MinVersionResponse {
  minVersion: string;
  latestVersion: string;
  storeUrl: string;
  platform: string;
  app: string;
}

const POLL_INTERVAL_MS = 30 * 60 * 1000; // 30 min while foregrounded
const API_BASE_URL = process.env.EXPO_PUBLIC_API_URL ?? '';

async function fetchMinVersion(): Promise<MinVersionResponse | null> {
  if (!API_BASE_URL || !appPlatform) return null;
  const url = `${API_BASE_URL}/mobile/min-version?platform=${appPlatform}&app=vendor`;
  // Bare fetch (NOT the shared axios client): the endpoint is public
  // and we don't want the 401/426 interceptors firing on a missing or
  // out-of-date token. We also want a clean failure mode if the
  // backend hasn't shipped the endpoint yet — `null` instead of an
  // upgrade wall.
  try {
    const res = await fetch(url);
    if (!res.ok) return null;
    return (await res.json()) as MinVersionResponse;
  } catch {
    return null;
  }
}

export interface UseMinVersionResult {
  /** True when the running app version is below the backend's minimum */
  upgradeRequired: boolean;
  minVersion: string | null;
  storeUrl: string | null;
  /** Current app version (semver + build, e.g. "1.0.3+12") for display */
  currentVersion: string;
}

// Polls /mobile/min-version every 30 min while the app is in the
// foreground, plus once on each foreground transition. Returns enough
// state for the root layout to short-circuit routing to the upgrade
// wall, plus the metadata that wall needs to render.
export function useMinVersion(): UseMinVersionResult {
  const [foregroundTick, setForegroundTick] = useState(0);

  useEffect(() => {
    const sub = AppState.addEventListener('change', (next) => {
      if (next === 'active') setForegroundTick((t) => t + 1);
    });
    return () => sub.remove();
  }, []);

  const { data } = useQuery({
    queryKey: ['mobile', 'min-version', foregroundTick],
    queryFn: fetchMinVersion,
    refetchInterval: POLL_INTERVAL_MS,
    refetchOnWindowFocus: false,
    retry: 1,
    staleTime: 5 * 60 * 1000,
  });

  const minVersion = data?.minVersion ?? null;
  const upgradeRequired = !!minVersion && semverLess(appVersion, minVersion);

  return {
    upgradeRequired,
    minVersion,
    storeUrl: data?.storeUrl ?? null,
    currentVersion: appVersion,
  };
}
