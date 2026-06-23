import MapView, { Circle, Marker, PROVIDER_DEFAULT } from 'react-native-maps';
import { useEffect, useRef } from 'react';
import { StyleSheet } from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';

// Radius (m) of the chef "pickup area" circle. The backend sends an APPROXIMATE
// chef location (deterministically offset ~300m from the real kitchen), and we
// draw this soft circle around it — so the customer sees the rough area their
// food comes from, never the exact address. Off-platform contact stays blocked.
const CHEF_AREA_RADIUS_M = 300;

interface DeliveryMapProps {
  driverLat?: number | null;
  driverLng?: number | null;
  // Customer's own delivery destination (Delivery.DropoffLatitude/Longitude).
  // Safe to show precisely — it's the customer's own address.
  dropoffLat?: number | null;
  dropoffLng?: number | null;
  // Chef pickup — APPROXIMATE coords from the backend. Rendered as a soft area
  // circle (never a precise pin) so the exact kitchen address is never revealed.
  chefLat?: number | null;
  chefLng?: number | null;
}

export function DeliveryMap({
  driverLat,
  driverLng,
  dropoffLat,
  dropoffLng,
  chefLat,
  chefLng,
}: DeliveryMapProps) {
  // Guard per Pitfall 6: driver may have 0,0 before first location update
  const hasDriverLocation =
    driverLat != null &&
    driverLng != null &&
    driverLat !== 0 &&
    driverLng !== 0;

  // Customer's own delivery destination — shown as a precise pin (their address).
  const hasDropoffCoords =
    dropoffLat != null &&
    dropoffLng != null &&
    dropoffLat !== 0 &&
    dropoffLng !== 0;
  // Chef pickup area — shown as a soft circle (approximate), never a pin.
  const hasChef =
    chefLat != null && chefLng != null && chefLat !== 0 && chefLng !== 0;

  // Frame all real points (dropoff, chef area, live driver) with padding so the
  // map zooms to the delivery area instead of a country-wide view.
  const points: { lat: number; lng: number }[] = [];
  if (hasDropoffCoords) points.push({ lat: dropoffLat!, lng: dropoffLng! });
  if (hasChef) points.push({ lat: chefLat!, lng: chefLng! });
  if (hasDriverLocation) points.push({ lat: driverLat!, lng: driverLng! });

  const mapRef = useRef<MapView>(null);

  let initialRegion;
  if (points.length > 0) {
    const lats = points.map((p) => p.lat);
    const lngs = points.map((p) => p.lng);
    const minLat = Math.min(...lats);
    const maxLat = Math.max(...lats);
    const minLng = Math.min(...lngs);
    const maxLng = Math.max(...lngs);
    // 60% padding around the bounding box; min delta keeps a single point from
    // zooming in to street level.
    initialRegion = {
      latitude: (minLat + maxLat) / 2,
      longitude: (minLng + maxLng) / 2,
      latitudeDelta: Math.max((maxLat - minLat) * 1.6, 0.02),
      longitudeDelta: Math.max((maxLng - minLng) * 1.6, 0.02),
    };
  } else {
    // India center fallback if no coords available at all
    initialRegion = {
      latitude: 20.5937,
      longitude: 78.9629,
      latitudeDelta: 10,
      longitudeDelta: 10,
    };
  }

  // Coords load async (and the driver moves), so re-center the map whenever the
  // points change — initialRegion alone is mount-only and would stay on the
  // country-wide fallback if it rendered before the coords arrived.
  useEffect(() => {
    if (points.length > 0) {
      mapRef.current?.animateToRegion(initialRegion, 400);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dropoffLat, dropoffLng, chefLat, chefLng, driverLat, driverLng]);

  return (
    <MapView
      ref={mapRef}
      style={StyleSheet.absoluteFill}
      provider={PROVIDER_DEFAULT}
      initialRegion={initialRegion}
      showsUserLocation={false}
      showsMyLocationButton={false}
    >
      {/* Chef pickup AREA — a soft translucent coral wash around the approximate
          chef location: a gentle "rough area", never a solid disc or a hard
          edge. The stroke is set explicitly (low-opacity coral) so iOS MapKit
          can't fall back to a heavy default black ring. Privacy: the customer
          sees the rough area their food comes from, never the exact address. */}
      {hasChef && (
        <Circle
          center={{ latitude: chefLat!, longitude: chefLng! }}
          radius={CHEF_AREA_RADIUS_M}
          fillColor="rgba(255, 56, 92, 0.10)"
          strokeColor="rgba(255, 56, 92, 0.35)"
          strokeWidth={1.5}
        />
      )}
      {/* Destination marker — charcoal pin at the customer's own delivery address */}
      {hasDropoffCoords && (
        <Marker
          coordinate={{ latitude: dropoffLat!, longitude: dropoffLng! }}
          title="Delivery Address"
          // Charcoal destination marker — the "home base" / delivery target
          pinColor={customerColors.charcoal.DEFAULT}
        />
      )}
      {/* Driver location marker — coral accent so the driver stands out on the map */}
      {hasDriverLocation && (
        <Marker
          coordinate={{ latitude: driverLat!, longitude: driverLng! }}
          title="Driver"
          // Coral accent: the driver is the active moving element, primary interest
          pinColor={customerColors.coral.DEFAULT}
        />
      )}
    </MapView>
  );
}
