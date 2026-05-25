import { useRef, useMemo, useEffect } from 'react';
import { View, Text, ActivityIndicator, StyleSheet } from 'react-native';
import { useLocalSearchParams, router, useIsFocused } from 'expo-router';
import BottomSheet, { BottomSheetScrollView } from '@gorhom/bottom-sheet';
import { useOrderTrackingWS } from '../../../hooks/useOrderTrackingWS';
import { useOrderTracking } from '../../../hooks/useOrderTracking';
import { DeliveryMap } from '../../../components/tracking/DeliveryMap';
import { OrderTimeline } from '../../../components/orders/OrderTimeline';

export default function TrackOrderScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const isFocused = useIsFocused(); // T-02-04-02: guard polling with focus

  // WS hook: real-time driver location with polling fallback after 3 failures
  const { driverLocation, isPollingFallback } = useOrderTrackingWS(
    id ?? '',
    isFocused,
  );

  // REST polling: provides order metadata (status, chef, timeline) and serves
  // as the driver-location source when WS fallback is active
  const { data, isLoading } = useOrderTracking(id ?? '', isFocused);
  const tracking = data?.data;

  const bottomSheetRef = useRef<BottomSheet>(null);
  const snapPoints = useMemo(() => ['30%', '60%'], []);

  // Navigate to order detail when delivery is complete
  useEffect(() => {
    if (tracking?.status === 'delivered') {
      const timer = setTimeout(() => {
        router.replace(`/order/${id}`);
      }, 2000);
      return () => clearTimeout(timer);
    }
    return undefined;
  }, [tracking?.status, id]);

  if (isLoading || !tracking) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator size="large" color="#C2410C" />
      </View>
    );
  }

  // Driver coords: prefer real-time WS location; fall back to polling coords
  // isPollingFallback flag indicates WS failed — polling data is already live
  const effectiveDriverLat =
    driverLocation != null && !isPollingFallback
      ? driverLocation.latitude
      : tracking.delivery?.currentLatitude;
  const effectiveDriverLng =
    driverLocation != null && !isPollingFallback
      ? driverLocation.longitude
      : tracking.delivery?.currentLongitude;

  return (
    <View style={styles.container}>
      {/* Full-screen map — driver position from WS (or polling when WS falls back) */}
      <DeliveryMap
        driverLat={effectiveDriverLat}
        driverLng={effectiveDriverLng}
        dropoffLat={tracking.delivery?.dropoffLatitude}
        dropoffLng={tracking.delivery?.dropoffLongitude}
        chefLat={tracking.chef?.latitude}
        chefLng={tracking.chef?.longitude}
      />

      {/* Bottom sheet overlaid on top of map */}
      <BottomSheet
        ref={bottomSheetRef}
        snapPoints={snapPoints}
        index={0}
        backgroundStyle={styles.sheetBackground}
        handleIndicatorStyle={styles.handleIndicator}
      >
        <BottomSheetScrollView contentContainerStyle={styles.sheetContent}>
          <Text style={styles.orderNumber}>Order #{tracking.orderNumber}</Text>
          {tracking.chef?.name ? (
            <Text style={styles.chefName}>{tracking.chef.name}</Text>
          ) : null}
          <OrderTimeline
            status={tracking.status}
            estimatedDeliveryTime={tracking.estimatedDeliveryTime}
          />
        </BottomSheetScrollView>
      </BottomSheet>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  loading: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#fafaf7',
  },
  sheetBackground: {
    backgroundColor: '#fafaf7',
    borderRadius: 16,
  },
  handleIndicator: {
    backgroundColor: '#d4d3ce',
  },
  sheetContent: {
    paddingHorizontal: 16,
    paddingBottom: 24,
    paddingTop: 4,
  },
  orderNumber: {
    fontSize: 16,
    fontWeight: '700',
    color: '#1a1a18',
    marginBottom: 4,
  },
  chefName: {
    fontSize: 14,
    color: '#7a7a76',
    marginBottom: 4,
  },
});
