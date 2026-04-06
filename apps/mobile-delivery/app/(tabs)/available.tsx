import {
  ActivityIndicator,
  FlatList,
  RefreshControl,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { DeliveryCard } from '../../components/driver/DeliveryCard';
import { useDriverDashboard, useToggleOnline } from '../../hooks/useDriverDashboard';
import { useAcceptDelivery, useAvailableDeliveries, type AvailableDelivery } from '../../hooks/useDriverDeliveries';

export default function AvailableScreen() {
  const { data: dashboard } = useDriverDashboard();
  const { data: available, refetch, isLoading, isRefetching } = useAvailableDeliveries();
  const acceptMutation = useAcceptDelivery();
  const toggleOnlineMutation = useToggleOnline();

  const isOnline = dashboard?.isOnline ?? false;
  const deliveries = available?.deliveries ?? [];

  function handleAccept(deliveryId: string) {
    acceptMutation.mutate(deliveryId);
  }

  function handleGoOnline() {
    toggleOnlineMutation.mutate(true);
  }

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50 items-center justify-center">
        <ActivityIndicator size="large" color="#FF6B35" />
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      {/* Header */}
      <View className="px-4 pt-4 pb-2">
        <Text className="text-2xl font-bold text-gray-900">Available</Text>
      </View>

      {/* Offline banner */}
      {!isOnline && (
        <View className="mx-4 mb-4 bg-gray-200 rounded-2xl px-4 py-4 items-center">
          <Text className="text-lg font-semibold text-gray-600 mb-1">You&apos;re Offline</Text>
          <Text className="text-sm text-gray-500 mb-4 text-center">
            Go online to see available deliveries near you
          </Text>
          <TouchableOpacity
            onPress={handleGoOnline}
            disabled={toggleOnlineMutation.isPending}
            className={`px-8 py-3 rounded-xl ${toggleOnlineMutation.isPending ? 'bg-orange-300' : 'bg-orange-500'}`}
            activeOpacity={0.8}
          >
            {toggleOnlineMutation.isPending ? (
              <ActivityIndicator size="small" color="white" />
            ) : (
              <Text className="text-white font-semibold text-base">
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
              tintColor="#FF6B35"
            />
          }
          ListEmptyComponent={
            <View className="items-center justify-center py-16">
              <Text className="text-4xl mb-4">&#128690;</Text>
              <Text className="text-base text-gray-500 text-center">
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
    </SafeAreaView>
  );
}
