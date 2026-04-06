import MapView, { Marker, PROVIDER_DEFAULT } from 'react-native-maps';
import { StyleSheet } from 'react-native';

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

  const initialRegion = hasDestination
    ? {
        latitude: destLat,
        longitude: destLng,
        latitudeDelta: 0.05,
        longitudeDelta: 0.05,
      }
    : {
        // India center fallback if no coords available at all
        latitude: 20.5937,
        longitude: 78.9629,
        latitudeDelta: 10,
        longitudeDelta: 10,
      };

  return (
    <MapView
      style={StyleSheet.absoluteFillObject}
      provider={PROVIDER_DEFAULT}
      initialRegion={initialRegion}
      showsUserLocation={false}
      showsMyLocationButton={false}
    >
      {/* Delivery destination marker (red) — customer dropoff from Delivery model */}
      {hasDestination && (
        <Marker
          coordinate={{ latitude: destLat, longitude: destLng }}
          title={hasDropoffCoords ? 'Delivery Address' : 'Chef Location'}
          pinColor="red"
        />
      )}
      {/* Driver location marker (blue) — only when driver has real coordinates */}
      {hasDriverLocation && (
        <Marker
          coordinate={{ latitude: driverLat!, longitude: driverLng! }}
          title="Driver"
          pinColor="blue"
        />
      )}
    </MapView>
  );
}
