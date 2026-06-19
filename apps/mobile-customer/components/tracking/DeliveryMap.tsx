import MapView, { Marker, PROVIDER_DEFAULT } from 'react-native-maps';
import { useEffect, useRef } from 'react';
import { StyleSheet } from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';

interface DeliveryMapProps {
  driverLat?: number | null;
  driverLng?: number | null;
  // Destination: use Delivery.DropoffLatitude / DropoffLongitude from backend Delivery model.
  // NOTE: Backend gap — DeliveryResponse.ToResponse() omits these fields for now.
  // Falls back to chef coordinates when dropoff coords are 0 or missing.
  dropoffLat?: number | null;
  dropoffLng?: number | null;
  // Fallback: chef location (used when dropoff coords are 0 or missing)
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

  // Destination: prefer dropoff coords from Delivery model; fall back to chef location
  const hasDropoffCoords =
    dropoffLat != null &&
    dropoffLng != null &&
    dropoffLat !== 0 &&
    dropoffLng !== 0;
  const destLat = hasDropoffCoords ? dropoffLat! : (chefLat ?? 0);
  const destLng = hasDropoffCoords ? dropoffLng! : (chefLng ?? 0);
  const hasDestination = destLat !== 0 && destLng !== 0;
  const hasChef = chefLat != null && chefLng != null && chefLat !== 0 && chefLng !== 0;

  // Frame all real points (destination, chef pickup, live driver) with padding
  // so the map zooms to the delivery area instead of a country-wide view.
  const points: { lat: number; lng: number }[] = [];
  if (hasDestination) points.push({ lat: destLat, lng: destLng });
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
  }, [destLat, destLng, chefLat, chefLng, driverLat, driverLng]);

  return (
    <MapView
      ref={mapRef}
      style={StyleSheet.absoluteFill}
      provider={PROVIDER_DEFAULT}
      initialRegion={initialRegion}
      showsUserLocation={false}
      showsMyLocationButton={false}
    >
      {/* Destination marker — charcoal pin (customer dropoff / chef fallback) */}
      {hasDestination && (
        <Marker
          coordinate={{ latitude: destLat, longitude: destLng }}
          title={hasDropoffCoords ? 'Delivery Address' : 'Chef Location'}
          // Charcoal destination marker — the "home base" / delivery target
          pinColor={customerColors.charcoal.DEFAULT}
        />
      )}
      {/* Chef pickup marker — shown alongside the dropoff when both are known */}
      {hasChef && hasDropoffCoords && (
        <Marker
          coordinate={{ latitude: chefLat!, longitude: chefLng! }}
          title="Chef"
          pinColor={customerColors.charcoal.soft}
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
