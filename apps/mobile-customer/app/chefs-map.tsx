import React from 'react';
import { ActivityIndicator, Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import MapView, { Marker, PROVIDER_DEFAULT } from 'react-native-maps';
import { ChevronLeft, MapPin } from 'lucide-react-native';
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

// Floating circular back button — spec §2.6: white, shadow[2], charcoal
// chevron. Replaces the flat ScreenHeader bar so the map renders full-bleed
// (mirrors the order-tracking screen's overlay treatment). Map logic/provider/
// marker data flow below is untouched — this is styling only.
function FloatingBackButton() {
  return (
    <SafeAreaView style={styles.backButtonContainer} edges={['top', 'left']} pointerEvents="box-none">
      <View style={styles.backCircleShadow}>
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={8}
          android_ripple={{ color: `${customerColors.charcoal.DEFAULT}1F`, borderless: true, radius: 21 }}
        >
          {({ pressed }) => (
            <View
              style={[styles.backCircleInner, pressed && Platform.OS === 'ios' && styles.backCircleInnerPressed]}
            >
              <ChevronLeft size={22} color={customerColors.charcoal.DEFAULT} accessibilityElementsHidden />
            </View>
          )}
        </Pressable>
      </View>
    </SafeAreaView>
  );
}

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
    <View style={styles.container}>
      {isLoading ? (
        <View style={styles.centered}>
          <ActivityIndicator color={customerColors.coral.DEFAULT} />
        </View>
      ) : located.length === 0 ? (
        <View style={styles.centered}>
          <View style={styles.emptyIconWrap}>
            <MapPin size={26} color={customerColors.charcoal.soft} />
          </View>
          <Text style={styles.emptyTitle}>No chefs on the map yet</Text>
          <Text style={styles.emptyBody}>
            No chefs with a mapped location yet. Check back soon.
          </Text>
        </View>
      ) : (
        // Map logic/provider/markers untouched — overlay styling only.
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

      <FloatingBackButton />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: customerColors.canvas },
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 32,
    gap: 6,
  },
  emptyIconWrap: {
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 4,
  },
  emptyTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 17,
    color: customerColors.charcoal.DEFAULT,
    textAlign: 'center',
  },
  emptyBody: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
    lineHeight: 18,
  },

  // ---- Floating back button — mirrors order/[id]/track.tsx §2.6 treatment ----
  backButtonContainer: {
    position: 'absolute',
    top: 0,
    left: 0,
    zIndex: 10,
    padding: 16,
  },
  backCircleShadow: {
    borderRadius: 9999,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.12,
    shadowRadius: 6,
    elevation: 4,
  },
  backCircleInner: {
    width: 42,
    height: 42,
    borderRadius: 9999,
    backgroundColor: customerColors.surface.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden',
  },
  backCircleInnerPressed: {
    backgroundColor: customerColors.surface.soft,
  },
});
