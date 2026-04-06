import { useRef, useMemo, useEffect } from 'react';
import { View, Text, ActivityIndicator, StyleSheet } from 'react-native';
import { useLocalSearchParams, router, useIsFocused } from 'expo-router';
import BottomSheet, { BottomSheetScrollView } from '@gorhom/bottom-sheet';
import { useOrderTracking } from '../../../hooks/useOrderTracking';
import { DeliveryMap } from '../../../components/tracking/DeliveryMap';
import { OrderTimeline } from '../../../components/orders/OrderTimeline';

export default function TrackOrderScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const isFocused = useIsFocused(); // T-02-04-02: guard polling with focus

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
        <ActivityIndicator size="large" color="#F97316" />
      </View>
    );
  }

  return (
    <View style={styles.container}>
      {/* Full-screen map — occupies entire screen behind bottom sheet */}
      <DeliveryMap
        driverLat={tracking.delivery?.currentLatitude}
        driverLng={tracking.delivery?.currentLongitude}
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
    backgroundColor: '#F9FAFB',
  },
  sheetBackground: {
    backgroundColor: '#FFFFFF',
    borderRadius: 16,
  },
  handleIndicator: {
    backgroundColor: '#D1D5DB',
  },
  sheetContent: {
    paddingHorizontal: 16,
    paddingBottom: 24,
    paddingTop: 4,
  },
  orderNumber: {
    fontSize: 16,
    fontWeight: '700',
    color: '#111827',
    marginBottom: 4,
  },
  chefName: {
    fontSize: 14,
    color: '#6B7280',
    marginBottom: 4,
  },
});
