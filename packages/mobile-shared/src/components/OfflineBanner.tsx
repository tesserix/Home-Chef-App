import React, { useEffect, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import NetInfo from '@react-native-community/netinfo';
import { colors, spacing } from '../theme/tokens';

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
    <View style={styles.banner}>
      <Text style={styles.label}>You are offline — showing cached data</Text>
    </View>
  );
}

// StyleSheet, not NativeWind classes: this package has no NativeWind types, so
// className silently did nothing here and the banner rendered unstyled.
const styles = StyleSheet.create({
  banner: {
    backgroundColor: colors.herb.DEFAULT,
    paddingHorizontal: spacing[4],
    paddingVertical: spacing[2],
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
  },
  label: {
    color: colors.paper,
    fontSize: 14,
    fontWeight: '500',
  },
});
