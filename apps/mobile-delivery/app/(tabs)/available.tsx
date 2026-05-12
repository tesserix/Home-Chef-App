import { useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  RefreshControl,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import * as Haptics from 'expo-haptics';
import { SafeAreaView } from 'react-native-safe-area-context';
import * as Location from 'expo-location';
import { DeliveryCard } from '../../components/driver/DeliveryCard';
import { LocationRationaleModal } from '../../components/LocationRationaleModal';
import { useDriverDashboard, useToggleOnline } from '../../hooks/useDriverDashboard';
import { useAcceptDelivery, useAvailableDeliveries, type AvailableDelivery } from '../../hooks/useDriverDeliveries';
import { startTracking } from '../../lib/background-location';
import { useDeliveryStore } from '../../store/delivery-store';

export default function AvailableScreen() {
  const { data: dashboard } = useDriverDashboard();
  const { data: available, refetch, isLoading, isRefetching } = useAvailableDeliveries();
  const acceptMutation = useAcceptDelivery();
  const toggleOnlineMutation = useToggleOnline();
  const [showRationale, setShowRationale] = useState(false);
  const [pendingDeliveryAccept, setPendingDeliveryAccept] = useState<string | null>(null);

  const isOnline = dashboard?.isOnline ?? false;
  const deliveries = available?.deliveries ?? [];

  async function requestGPSAndStartTracking(deliveryId: string) {
    const { status: foreground } = await Location.getForegroundPermissionsAsync();
    const { status: background } = await Location.getBackgroundPermissionsAsync();

    if (background === 'granted') {
      await startTracking();
      useDeliveryStore.getState().setTrackingLocation(true, deliveryId);
      return;
    }

    if (foreground !== 'granted') {
      const { status } = await Location.requestForegroundPermissionsAsync();
      if (status !== 'granted') {
        console.warn('[GPS] Foreground location denied — delivery proceeds without tracking');
        return;
      }
    }

    // Show rationale modal before requesting background permission (D-03)
    setPendingDeliveryAccept(deliveryId);
    setShowRationale(true);
  }

  async function handleRationaleAllow() {
    setShowRationale(false);
    const { status } = await Location.requestBackgroundPermissionsAsync();
    if (pendingDeliveryAccept) {
      await startTracking();
      useDeliveryStore.getState().setTrackingLocation(true, pendingDeliveryAccept);
    }
    if (status !== 'granted') {
      console.warn('[GPS] Background location denied — tracking foreground only');
    }
    setPendingDeliveryAccept(null);
  }

  function handleRationaleDeny() {
    setShowRationale(false);
    setPendingDeliveryAccept(null);
    // Delivery proceeds without background tracking — acceptable per D-03
  }

  function handleAccept(deliveryId: string) {
    acceptMutation.mutate(deliveryId, {
      onSuccess: () => {
        void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
        requestGPSAndStartTracking(deliveryId).catch((err: unknown) => {
          console.warn('[GPS] Failed to start tracking after accept:', err);
        });
      },
    });
  }

  function handleGoOnline() {
    toggleOnlineMutation.mutate(true);
  }

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-paper">
        <View className="px-4 pt-4 pb-2">
          <Text className="font-display text-2xl font-semibold text-ink">Available</Text>
        </View>
        <View className="px-4 pt-2">
          <View className="h-28 bg-mist rounded-2xl mb-3" />
          <View className="h-28 bg-mist rounded-2xl mb-3" />
          <View className="h-28 bg-mist rounded-2xl mb-3" />
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-paper">
      {/* Header */}
      <View className="px-4 pt-4 pb-2">
        <Text className="font-display text-2xl font-semibold text-ink">Available</Text>
      </View>

      {/* Offline banner */}
      {!isOnline && (
        <View className="mx-4 mb-4 bg-mist rounded-2xl px-4 py-4 items-center">
          <Text className="text-lg font-semibold text-ink-soft mb-1">You&apos;re Offline</Text>
          <Text className="text-sm text-ink-muted mb-4 text-center">
            Go online to see available deliveries near you
          </Text>
          <TouchableOpacity
            onPress={handleGoOnline}
            disabled={toggleOnlineMutation.isPending}
            className={`px-8 py-3 rounded-xl ${toggleOnlineMutation.isPending ? 'bg-herb-soft' : 'bg-herb'}`}
            activeOpacity={0.8}
          >
            {toggleOnlineMutation.isPending ? (
              <ActivityIndicator size="small" color="white" />
            ) : (
              <Text className="text-paper font-semibold text-base">
                Go Online to Accept Deliveries
              </Text>
            )}
          </TouchableOpacity>
        </View>
      )}

      {/* Deliveries list — only rendered when online */}
      {isOnline && (
        <FlatList<AvailableDelivery>
          data={deliveries}
          keyExtractor={(item) => item.id}
          contentContainerStyle={{ paddingHorizontal: 16, paddingBottom: 24 }}
          refreshControl={
            <RefreshControl
              refreshing={isRefetching}
              onRefresh={refetch}
              tintColor="#3e6b3c"
            />
          }
          ListEmptyComponent={
            <View className="items-center justify-center py-16">
              <Text className="text-4xl mb-4">&#128690;</Text>
              <Text className="text-base text-ink-muted text-center">
                No deliveries available nearby.{'\n'}Pull to refresh.
              </Text>
            </View>
          }
          renderItem={({ item }) => (
            <DeliveryCard
              delivery={item}
              onAccept={() => handleAccept(item.id)}
              isLoading={acceptMutation.isPending && acceptMutation.variables === item.id}
            />
          )}
        />
      )}

      {/* Background location rationale modal — shown before system permission prompt */}
      <LocationRationaleModal
        visible={showRationale}
        onAllow={handleRationaleAllow}
        onDeny={handleRationaleDeny}
      />
    </SafeAreaView>
  );
}
