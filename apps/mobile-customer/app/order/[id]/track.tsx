import { useRef, useMemo, useEffect } from 'react';
import {
  View,
  Text,
  ActivityIndicator,
  Platform,
  Pressable,
  StyleSheet,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams, router, useIsFocused } from 'expo-router';
import BottomSheet, { BottomSheetScrollView } from '@gorhom/bottom-sheet';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useOrderTrackingWS } from '../../../hooks/useOrderTrackingWS';
import { useOrderTracking } from '../../../hooks/useOrderTracking';
import { DeliveryMap } from '../../../components/tracking/DeliveryMap';
import { OrderTimeline } from '../../../components/orders/OrderTimeline';
import { getStepIndex, getStatusLine } from '../../../lib/orderSteps';

// Progress dot row — coral dots for active/passed, hairline for future.
// Count of 4 mirrors the OrderTimeline step count so they stay in sync.
const PROGRESS_STEP_COUNT = 4;

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
  // 35% snapped by default — enough to show driver + ETA; drag up for timeline
  const snapPoints = useMemo(() => ['35%', '65%'], []);

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
        <ActivityIndicator
          size="large"
          color={customerColors.coral.DEFAULT}
        />
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

  const progressIndex = getStepIndex(
    tracking.status,
    tracking.fulfillmentType,
  );

  return (
    <View style={styles.container}>
      {/* Full-bleed map — driver position from WS (or polling when WS falls back) */}
      <DeliveryMap
        driverLat={effectiveDriverLat}
        driverLng={effectiveDriverLng}
        dropoffLat={tracking.delivery?.dropoffLatitude}
        dropoffLng={tracking.delivery?.dropoffLongitude}
        chefLat={tracking.chef?.latitude}
        chefLng={tracking.chef?.longitude}
      />

      {/* Circular floating back button — top-left, safe area aware.
          Spec §2.6: white, shadow[2], charcoal chevron.
          Shadow is on the outer View; inner View clips the shape.
          Using SafeAreaView edges top-only so it clears the status bar/notch. */}
      <SafeAreaView
        style={styles.backButtonContainer}
        edges={['top', 'left']}
        pointerEvents="box-none"
      >
        {/* Outer shadow layer — no overflow:hidden so shadow renders */}
        <View style={styles.backCircleShadow}>
          {/* iOS Pressable bug: visual styles on inner View */}
          <Pressable
            onPress={() => router.back()}
            accessibilityRole="button"
            accessibilityLabel="Go back"
            hitSlop={8}
            android_ripple={{ color: `${customerColors.charcoal.DEFAULT}1F`, borderless: true, radius: 21 }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.backCircleInner,
                  pressed && Platform.OS === 'ios' && styles.backCircleInnerPressed,
                ]}
              >
                <ChevronLeft
                  size={22}
                  color={customerColors.charcoal.DEFAULT}
                  accessibilityElementsHidden
                />
              </View>
            )}
          </Pressable>
        </View>
      </SafeAreaView>

      {/* Floating white status card — Airbnb-style bottom sheet.
          Spec §2.6: rounded-xl, shadow[2], driver name/avatar, ETA, progress dots coral.
          The BottomSheet handle indicator is styled hairline. */}
      <BottomSheet
        ref={bottomSheetRef}
        snapPoints={snapPoints}
        index={0}
        // White card background + rounded-xl — shadow from the sheet's outer view
        backgroundStyle={styles.sheetBackground}
        handleIndicatorStyle={styles.handleIndicator}
      >
        <BottomSheetScrollView
          contentContainerStyle={styles.sheetContent}
          showsVerticalScrollIndicator={false}
        >
          {/* Driver info row — name + ETA badge */}
          <View style={styles.driverRow}>
            {/* Driver avatar placeholder — charcoal-soft initial circle */}
            <View style={styles.driverAvatar}>
              <Text style={styles.driverAvatarText}>
                {tracking.chef?.name?.charAt(0).toUpperCase() ?? 'D'}
              </Text>
            </View>
            <View style={styles.driverInfo}>
              <Text style={styles.driverName}>
                {tracking.chef?.name ?? 'Your Chef'}
              </Text>
              <Text style={styles.orderNumberText}>
                Order #{tracking.orderNumber}
              </Text>
            </View>
            {/* ETA badge — tabular-nums, coral-tint bg */}
            {tracking.estimatedDeliveryTime ? (
              <View style={styles.etaBadge}>
                <Text style={styles.etaLabel}>ETA</Text>
                <Text style={styles.etaTime}>
                  {new Date(tracking.estimatedDeliveryTime).toLocaleTimeString(
                    'en-IN',
                    { hour: '2-digit', minute: '2-digit' },
                  )}
                </Text>
              </View>
            ) : null}
          </View>

          {/* Coral progress dots — one per step, filled coral up to current, hairline after */}
          <View style={styles.progressDots}>
            {Array.from({ length: PROGRESS_STEP_COUNT }).map((_, i) => (
              <View
                key={i}
                style={[
                  styles.progressDot,
                  i <= progressIndex
                    ? styles.progressDotActive
                    : styles.progressDotInactive,
                ]}
              />
            ))}
          </View>

          {/* Status label — current step text (pickup vs delivery aware) */}
          <Text style={styles.statusText}>
            {getStatusLine(tracking.status, tracking.fulfillmentType)}
          </Text>

          {/* Hairline divider before timeline */}
          <View style={styles.timelineDivider} />

          {/* Full timeline — visible when sheet is dragged to expanded snap point */}
          <OrderTimeline
            status={tracking.status}
            fulfillmentType={tracking.fulfillmentType}
            estimatedDeliveryTime={tracking.estimatedDeliveryTime}
          />
        </BottomSheetScrollView>
      </BottomSheet>
    </View>
  );
}

const styles = StyleSheet.create({
  // Full-bleed container — map fills all available space
  container: {
    flex: 1,
    backgroundColor: customerColors.canvas,
  },
  loading: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.canvas,
  },

  // ---- Floating back button ------------------------------------------------
  // Positioned absolute top-left; SafeAreaView handles notch inset
  backButtonContainer: {
    position: 'absolute',
    top: 0,
    left: 0,
    // Explicit zIndex so it floats above the map
    zIndex: 10,
    padding: 16,
  },
  // Shadow wrapper — no overflow:hidden so iOS shadow renders
  backCircleShadow: {
    borderRadius: 9999,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.12,
    shadowRadius: 6,
    elevation: 4,
  },
  // Inner clip view — overflow:hidden for circular shape without killing shadow
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

  // ---- Bottom sheet floating card -----------------------------------------
  // White card background, rounded-xl top corners — matches Airbnb spec §2.6
  sheetBackground: {
    backgroundColor: customerColors.surface.DEFAULT,
    borderRadius: 20,
    // shadow[2] applied here so it lifts over the map
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: -2 },
    shadowOpacity: 0.08,
    shadowRadius: 12,
    elevation: 8,
  },
  // Hairline drag handle — charcoal-soft, subtle
  handleIndicator: {
    backgroundColor: customerColors.hairline,
    width: 36,
  },
  sheetContent: {
    paddingHorizontal: 20,
    paddingBottom: 32,
    paddingTop: 4,
  },

  // ---- Driver info row ----------------------------------------------------
  driverRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    marginBottom: 16,
    marginTop: 4,
  },
  // Circular avatar with charcoal-soft bg + initial letter
  driverAvatar: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
  },
  driverAvatarText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.charcoal.soft,
  },
  driverInfo: {
    flex: 1,
  },
  driverName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    marginBottom: 2,
  },
  orderNumberText: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
  // ETA badge — coral-tint bg, coral-pressed text, tabular time
  etaBadge: {
    backgroundColor: customerColors.coral.tint,
    borderRadius: 10,
    paddingHorizontal: 12,
    paddingVertical: 6,
    alignItems: 'center',
    minWidth: 60,
  },
  etaLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 9,
    color: customerColors.coral.pressed,
    letterSpacing: 0.8,
    marginBottom: 1,
  },
  etaTime: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.coral.pressed,
    fontVariant: ['tabular-nums'],
  },

  // ---- Coral progress dots ------------------------------------------------
  // Row of dots — coral filled up to current step, hairline outline after.
  progressDots: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginBottom: 10,
  },
  progressDot: {
    flex: 1,
    height: 4,
    borderRadius: 2,
  },
  progressDotActive: {
    backgroundColor: customerColors.coral.DEFAULT,
  },
  progressDotInactive: {
    backgroundColor: customerColors.hairline,
  },

  // Status label — brief human-readable current step
  statusText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    marginBottom: 16,
  },

  // Hairline before the full timeline (visible when sheet is expanded)
  timelineDivider: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: customerColors.hairline,
    marginBottom: 4,
  },
});
