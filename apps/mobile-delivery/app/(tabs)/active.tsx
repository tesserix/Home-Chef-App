import { useState, useEffect } from 'react';
import {
  ActivityIndicator,
  Linking,
  Platform,
  RefreshControl,
  ScrollView,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { SlideToConfirm } from '../../components/driver/SlideToConfirm';
import { StatusStepIndicator } from '../../components/driver/StatusStepIndicator';
import { useCurrentDelivery, useUpdateDeliveryStatus } from '../../hooks/useDriverDeliveries';
import { stopTracking } from '../../lib/background-location';
import { useDeliveryStore } from '../../store/delivery-store';

const STATUS_LABELS: Record<string, string> = {
  assigned: 'Head to Kitchen',
  at_pickup: 'Waiting for Order',
  picked_up: 'Order Picked Up',
  in_transit: 'On the Way',
  at_dropoff: 'At Dropoff Location',
  delivered: 'Delivery Complete',
  cancelled: 'Delivery Cancelled',
};

const STATUS_ACTIONS: Record<string, { label: string; nextStatus: string }> = {
  assigned: { label: 'Arrived at Kitchen', nextStatus: 'at_pickup' },
  at_pickup: { label: 'Picked Up Order', nextStatus: 'picked_up' },
  picked_up: { label: 'Start Delivery', nextStatus: 'in_transit' },
  in_transit: { label: 'Arrived at Dropoff', nextStatus: 'at_dropoff' },
  at_dropoff: { label: 'Mark as Delivered', nextStatus: 'delivered' },
};

function openNavigation(lat: number, lng: number, label: string) {
  const encoded = encodeURIComponent(label);
  const url = Platform.select({
    ios: `maps://?q=${encoded}&ll=${lat},${lng}`,
    android: `geo:${lat},${lng}?q=${lat},${lng}(${encoded})`,
  })!;
  Linking.canOpenURL(url).then((supported) => {
    if (supported) {
      Linking.openURL(url);
    } else {
      Linking.openURL(`https://maps.google.com/?q=${lat},${lng}`);
    }
  });
}

export default function ActiveScreen() {
  const { data: currentDelivery, isLoading, refetch, isRefetching } = useCurrentDelivery();
  const statusMutation = useUpdateDeliveryStatus();
  const [orderExpanded, setOrderExpanded] = useState(false);

  // Stop background GPS tracking when delivery reaches a terminal status
  useEffect(() => {
    const status = currentDelivery?.status;
    if (status === 'delivered' || status === 'cancelled') {
      stopTracking().then(() => {
        useDeliveryStore.getState().setTrackingLocation(false, null);
      }).catch((err: unknown) => {
        console.warn('[GPS] Failed to stop tracking:', err);
      });
    }
  }, [currentDelivery?.status]);

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50 items-center justify-center">
        <ActivityIndicator size="large" color="#FF6B35" />
      </SafeAreaView>
    );
  }

  if (!currentDelivery) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50 items-center justify-center px-6">
        <Text className="text-5xl mb-4">&#128692;</Text>
        <Text className="text-xl font-bold text-gray-800 mb-2">No Active Delivery</Text>
        <Text className="text-sm text-gray-500 text-center">
          Accept a delivery from the Available tab to get started.
        </Text>
      </SafeAreaView>
    );
  }

  const delivery = currentDelivery;
  const action = STATUS_ACTIONS[delivery.status];
  const showPickupCard =
    delivery.status === 'assigned' || delivery.status === 'at_pickup';

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      <ScrollView
        className="flex-1"
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={refetch}
            tintColor="#FF6B35"
          />
        }
      >
        {/* Header */}
        <View className="px-4 pt-4 pb-2">
          <Text className="text-2xl font-bold text-gray-900">Active Delivery</Text>
          <Text className="text-sm text-gray-500 mt-1">
            Order #{delivery.orderId.slice(-6).toUpperCase()}
          </Text>
        </View>

        {/* Section 1: Status Step Indicator */}
        <View className="mx-4 bg-white rounded-2xl px-4 py-3 mb-4 shadow-sm">
          <StatusStepIndicator currentStatus={delivery.status} />
          <Text className="text-sm font-semibold text-gray-700 text-center mt-2">
            {STATUS_LABELS[delivery.status] ?? delivery.status}
          </Text>
        </View>

        {/* Cancelled banner */}
        {delivery.status === 'cancelled' && (
          <View className="mx-4 mb-4 bg-red-100 rounded-2xl px-4 py-4">
            <Text className="text-base font-bold text-red-700 text-center">
              Delivery Cancelled
            </Text>
            <Text className="text-sm text-red-500 text-center mt-1">
              This delivery has been cancelled. Check the Available tab for new requests.
            </Text>
          </View>
        )}

        {/* Section 2: Pickup or Dropoff card */}
        {delivery.status !== 'cancelled' && delivery.status !== 'delivered' && (
          <View
            className={`mx-4 mb-4 rounded-2xl overflow-hidden shadow-sm`}
          >
            {/* Card header */}
            <View
              className={`px-4 py-3 ${showPickupCard ? 'bg-orange-500' : 'bg-blue-500'}`}
            >
              <Text className="text-white font-bold text-base">
                {showPickupCard ? 'Pickup Location' : 'Dropoff Location'}
              </Text>
            </View>

            <View className="bg-white px-4 py-4">
              {showPickupCard ? (
                <>
                  <Text className="text-base font-semibold text-gray-800 mb-1">
                    {delivery.pickup.chefName}
                  </Text>
                  <TouchableOpacity
                    onPress={() =>
                      Linking.openURL(`tel:${delivery.pickup.chefPhone}`)
                    }
                  >
                    <Text className="text-sm text-blue-500 mb-2">
                      {delivery.pickup.chefPhone}
                    </Text>
                  </TouchableOpacity>
                  <Text className="text-sm text-gray-600 mb-3">
                    {delivery.pickup.address}
                  </Text>
                  {delivery.pickup.instructions ? (
                    <Text className="text-xs text-gray-400 mb-3">
                      Note: {delivery.pickup.instructions}
                    </Text>
                  ) : null}
                </>
              ) : (
                <>
                  <Text className="text-base font-semibold text-gray-800 mb-1">
                    {delivery.dropoff.customerName}
                  </Text>
                  <TouchableOpacity
                    onPress={() =>
                      Linking.openURL(`tel:${delivery.dropoff.customerPhone}`)
                    }
                  >
                    <Text className="text-sm text-blue-500 mb-2">
                      {delivery.dropoff.customerPhone}
                    </Text>
                  </TouchableOpacity>
                  <Text className="text-sm text-gray-600 mb-3">
                    {delivery.dropoff.address}
                  </Text>
                  {delivery.dropoff.instructions ? (
                    <Text className="text-xs text-gray-400 mb-3">
                      Note: {delivery.dropoff.instructions}
                    </Text>
                  ) : null}
                </>
              )}

              {/* Navigate button */}
              <TouchableOpacity
                onPress={() => {
                  if (showPickupCard) {
                    openNavigation(
                      delivery.pickup.lat,
                      delivery.pickup.lng,
                      `Pickup: ${delivery.pickup.chefName}`,
                    );
                  } else {
                    openNavigation(
                      delivery.dropoff.lat,
                      delivery.dropoff.lng,
                      `Dropoff: ${delivery.dropoff.customerName}`,
                    );
                  }
                }}
                className={`w-full py-3 rounded-xl items-center ${
                  showPickupCard ? 'bg-orange-500' : 'bg-blue-500'
                }`}
                activeOpacity={0.8}
              >
                <Text className="text-white font-semibold text-base">
                  Navigate
                </Text>
              </TouchableOpacity>
            </View>
          </View>
        )}

        {/* Section 3: Order Summary (collapsible) */}
        <View className="mx-4 mb-4 bg-white rounded-2xl shadow-sm overflow-hidden">
          <TouchableOpacity
            onPress={() => setOrderExpanded((prev: boolean) => !prev)}
            className="px-4 py-3 flex-row items-center justify-between"
            activeOpacity={0.7}
          >
            <Text className="text-base font-semibold text-gray-800">
              Order Summary ({delivery.order.items.length}{' '}
              {delivery.order.items.length === 1 ? 'item' : 'items'})
            </Text>
            <Text className="text-gray-400">{orderExpanded ? '▲' : '▼'}</Text>
          </TouchableOpacity>
          {orderExpanded && (
            <View className="px-4 pb-4">
              {delivery.order.items.map((item, idx) => (
                <View key={String(idx)} className="flex-row justify-between py-1">
                  <Text className="text-sm text-gray-700">{item.name}</Text>
                  <Text className="text-sm text-gray-500">x{item.quantity}</Text>
                </View>
              ))}
              <View className="border-t border-gray-100 mt-2 pt-2 flex-row justify-between">
                <Text className="text-sm font-semibold text-gray-700">Total</Text>
                <Text className="text-sm font-bold text-orange-500">
                  &#8377;{delivery.order.total}
                </Text>
              </View>
              {delivery.order.specialInstructions ? (
                <Text className="text-xs text-gray-400 mt-2">
                  Special instructions: {delivery.order.specialInstructions}
                </Text>
              ) : null}
            </View>
          )}
        </View>

        {/* Section 4: Status Update Slider / Terminal states */}
        <View className="mx-4 mb-8">
          {action && delivery.status !== 'cancelled' && (
            <SlideToConfirm
              label={action.label}
              onConfirm={() =>
                statusMutation.mutate({
                  id: delivery.id,
                  status: action.nextStatus,
                })
              }
              disabled={statusMutation.isPending}
            />
          )}

          {delivery.status === 'delivered' && (
            <View className="bg-green-100 rounded-2xl px-4 py-4 items-center">
              <Text className="text-lg font-bold text-green-700 mb-1">
                Delivery Complete
              </Text>
              <Text className="text-sm text-green-600">
                You earned &#8377;{delivery.payout} for this delivery
              </Text>
            </View>
          )}
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
