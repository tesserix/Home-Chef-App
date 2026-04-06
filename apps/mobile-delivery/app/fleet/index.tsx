import {
  ActivityIndicator,
  FlatList,
  RefreshControl,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useQuery } from '@tanstack/react-query';
import { Lock } from 'lucide-react-native';
import { api } from '../../lib/api';

interface FleetOverview {
  totalDrivers: number;
  onlineDrivers: number;
  todayDeliveries: number;
  todayEarnings: number;
}

interface Partner {
  id: string;
  name: string;
  phone: string;
  status: 'active' | 'inactive';
  todayDeliveries: number;
  rating: number;
}

function useFleetOverview() {
  return useQuery<FleetOverview | null>({
    queryKey: ['driver', 'fleet', 'overview'],
    queryFn: async () => {
      try {
        const r = await api.get('/delivery/staff/fleet/overview');
        return r.data as FleetOverview;
      } catch (e: unknown) {
        if (
          e !== null &&
          typeof e === 'object' &&
          'response' in e &&
          (e as { response?: { status?: number } }).response?.status === 403
        ) {
          return null; // not-authorized, not an error
        }
        throw e;
      }
    },
  });
}

function useFleetPartners(enabled: boolean) {
  return useQuery<Partner[] | null>({
    queryKey: ['driver', 'fleet', 'partners'],
    queryFn: async () => {
      try {
        const r = await api.get('/delivery/staff/fleet/partners');
        return r.data as Partner[];
      } catch (e: unknown) {
        if (
          e !== null &&
          typeof e === 'object' &&
          'response' in e &&
          (e as { response?: { status?: number } }).response?.status === 403
        ) {
          return null;
        }
        throw e;
      }
    },
    enabled,
  });
}

function StatCard({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <View className="flex-1 bg-white rounded-2xl p-4 shadow-sm">
      <Text className="text-xs text-gray-500 mb-1">{label}</Text>
      <Text className={`text-xl font-bold ${accent ? 'text-orange-500' : 'text-gray-900'}`}>
        {value}
      </Text>
    </View>
  );
}

function PartnerCard({ partner }: { partner: Partner }) {
  const isActive = partner.status === 'active';
  return (
    <TouchableOpacity
      onPress={() => router.push(`/fleet/partner/${partner.id}`)}
      className="bg-white rounded-2xl p-4 mb-3 shadow-sm flex-row items-center"
      activeOpacity={0.75}
    >
      <View className="flex-1">
        <View className="flex-row items-center mb-1">
          <Text className="text-base font-semibold text-gray-900 mr-2">{partner.name}</Text>
          <View
            className={`px-2 py-0.5 rounded-full ${isActive ? 'bg-green-100' : 'bg-gray-100'}`}
          >
            <Text
              className={`text-xs font-medium ${isActive ? 'text-green-700' : 'text-gray-500'}`}
            >
              {isActive ? 'Active' : 'Inactive'}
            </Text>
          </View>
        </View>
        <View className="flex-row gap-4">
          <Text className="text-sm text-gray-500">
            Today: {partner.todayDeliveries} deliveries
          </Text>
          <Text className="text-sm text-yellow-500">
            &#9733; {partner.rating.toFixed(1)}
          </Text>
        </View>
      </View>
    </TouchableOpacity>
  );
}

export default function FleetScreen() {
  const {
    data: fleet,
    isLoading: fleetLoading,
    isError: fleetError,
    refetch: refetchFleet,
    isRefetching,
  } = useFleetOverview();

  const fleetLoaded = fleet !== null && fleet !== undefined && !fleetLoading;
  const { data: partners } = useFleetPartners(fleetLoaded);

  if (fleetLoading) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50 items-center justify-center">
        <ActivityIndicator size="large" color="#FF6B35" />
      </SafeAreaView>
    );
  }

  if (fleetError) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50 items-center justify-center px-6">
        <Text className="text-gray-500 text-base mb-4">Failed to load fleet data</Text>
        <TouchableOpacity
          onPress={() => refetchFleet()}
          className="bg-orange-500 px-6 py-3 rounded-xl"
        >
          <Text className="text-white font-semibold">Retry</Text>
        </TouchableOpacity>
      </SafeAreaView>
    );
  }

  // 403 returned null — show non-error lock screen for fleet managers only
  if (fleet === null) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50 items-center justify-center px-8">
        <Lock size={48} color="#9CA3AF" />
        <Text className="text-xl font-bold text-gray-900 mt-4 mb-2">Fleet Management</Text>
        <Text className="text-sm text-gray-500 text-center leading-5">
          Fleet management is available for fleet managers only. Contact your administrator to
          request access.
        </Text>
      </SafeAreaView>
    );
  }

  const partnerList = partners ?? [];

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      <FlatList<Partner>
        data={partnerList}
        keyExtractor={(item) => item.id}
        contentContainerStyle={{ paddingHorizontal: 16, paddingBottom: 32 }}
        refreshControl={
          <RefreshControl
            refreshing={isRefetching}
            onRefresh={refetchFleet}
            tintColor="#FF6B35"
          />
        }
        ListHeaderComponent={
          <View>
            <View className="pt-4 pb-2">
              <Text className="text-2xl font-bold text-gray-900">Fleet Overview</Text>
            </View>

            {/* Stats grid */}
            <View className="flex-row gap-3 mb-3">
              <StatCard label="Total Drivers" value={String(fleet.totalDrivers)} />
              <StatCard label="Online Now" value={String(fleet.onlineDrivers)} />
            </View>
            <View className="flex-row gap-3 mb-4">
              <StatCard label="Today's Deliveries" value={String(fleet.todayDeliveries)} />
              <StatCard
                label="Today's Earnings"
                value={`\u20B9${fleet.todayEarnings.toFixed(0)}`}
                accent
              />
            </View>

            {partnerList.length > 0 && (
              <Text className="text-base font-semibold text-gray-700 mb-3">Partners</Text>
            )}
          </View>
        }
        ListEmptyComponent={
          <View className="items-center justify-center py-12">
            <Text className="text-base text-gray-400">No partners found.</Text>
          </View>
        }
        renderItem={({ item }) => <PartnerCard partner={item} />}
      />
    </SafeAreaView>
  );
}
