import React, { useEffect, useState } from 'react';
import { Text, View } from 'react-native';
import NetInfo from '@react-native-community/netinfo';

/**
 * OfflineBanner — non-blocking connectivity indicator.
 *
 * Listens to OS-level network state via NetInfo and renders a thin banner
 * when the device loses connectivity. Returns null when online, so it has
 * zero layout impact during normal operation.
 *
 * Render this above your root Stack/Tabs in each app's _layout.tsx.
 */
export function OfflineBanner() {
  const [isOffline, setIsOffline] = useState(false);

  useEffect(() => {
    const unsubscribe = NetInfo.addEventListener((state) => {
      setIsOffline(!state.isConnected);
    });
    return unsubscribe;
  }, []);

  if (!isOffline) return null;

  return (
    <View className="bg-herb px-4 py-2 flex-row items-center justify-center">
      <Text className="text-paper text-sm font-medium">
        You are offline — showing cached data
      </Text>
    </View>
  );
}
