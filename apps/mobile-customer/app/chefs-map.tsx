import React from 'react';
import { ActivityIndicator, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Stack, router } from 'expo-router';
import MapView, { Marker, PROVIDER_DEFAULT } from 'react-native-maps';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useChefs } from '../hooks/useChefs';

// Fallback when no chef has coordinates yet (Bengaluru centre). The map recenters
// on the first located chef when one exists.
const DEFAULT_REGION = {
  latitude: 12.9716,
  longitude: 77.5946,
  latitudeDelta: 0.12,
  longitudeDelta: 0.12,
};

export default function ChefsMapScreen() {
  const { data, isLoading } = useChefs({ limit: 50 });
  const located = (data?.data ?? []).filter(
    (c) => typeof c.latitude === 'number' && typeof c.longitude === 'number' && (c.latitude !== 0 || c.longitude !== 0)
  );

  const first = located[0];
  const region =
    first && first.latitude != null && first.longitude != null
      ? { latitude: first.latitude, longitude: first.longitude, latitudeDelta: 0.12, longitudeDelta: 0.12 }
      : DEFAULT_REGION;

  return (
    <SafeAreaView edges={['bottom']} style={{ flex: 1, backgroundColor: customerColors.canvas }}>
      <Stack.Screen options={{ title: 'Chefs near you' }} />
      {isLoading ? (
        <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center' }}>
          <ActivityIndicator color={customerColors.charcoal.soft} />
        </View>
      ) : located.length === 0 ? (
        <Text style={{ textAlign: 'center', marginTop: 32, color: customerColors.charcoal.soft, fontFamily: 'Inter' }}>
          No chefs with a mapped location yet.
        </Text>
      ) : (
        <MapView style={StyleSheet.absoluteFill} provider={PROVIDER_DEFAULT} initialRegion={region}>
          {located.map((c) => (
            <Marker
              key={c.id}
              coordinate={{ latitude: c.latitude as number, longitude: c.longitude as number }}
              title={c.name}
              description={c.cuisine}
              onCalloutPress={() => router.push(`/chef/${c.id}`)}
            />
          ))}
        </MapView>
      )}
    </SafeAreaView>
  );
}
