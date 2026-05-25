import { ActivityIndicator, ScrollView, Text, TouchableOpacity, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { useQuery } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import { api } from '../../../lib/api';

interface Partner {
  id: string;
  name: string;
  phone: string;
  status: 'active' | 'inactive';
  todayDeliveries: number;
  rating: number;
}

function usePartnerDetail(id: string) {
  return useQuery<Partner | null>({
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
    select: (data) => {
      if (!data || !Array.isArray(data)) return null;
      return data.find((p) => p.id === id) ?? null;
    },
  });
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <View className="flex-row justify-between items-center py-3 border-b border-mist">
      <Text className="text-sm text-ink-muted">{label}</Text>
      <Text className="text-sm font-medium text-ink">{value}</Text>
    </View>
  );
}

export default function PartnerDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data: partner, isLoading, isError, refetch } = usePartnerDetail(id ?? '');

  function handleBack() {
    if (router.canGoBack()) {
      router.back();
    } else {
      router.replace('/fleet');
    }
  }

  return (
    <SafeAreaView className="flex-1 bg-paper">
      <View className="flex-row items-center px-4 pt-4 pb-2">
        <TouchableOpacity accessibilityLabel="Go back" accessibilityRole="button" onPress={handleBack} className="mr-3 p-3" activeOpacity={0.7}>
          <ChevronLeft size={24} color="#4a4a47" />
        </TouchableOpacity>
        <Text className="text-xl font-semibold text-ink">Partner Detail</Text>
      </View>

      {isLoading && (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color="#C2410C" />
        </View>
      )}

      {isError && (
        <View className="flex-1 items-center justify-center px-6">
          <Text className="text-ink-muted text-base mb-4">Failed to load partner detail</Text>
          <TouchableOpacity
            onPress={() => refetch()}
            className="bg-herb px-6 py-3 rounded-xl"
          >
            <Text className="text-paper font-semibold">Retry</Text>
          </TouchableOpacity>
        </View>
      )}

      {!isLoading && !isError && !partner && (
        <View className="flex-1 items-center justify-center px-6">
          <Text className="text-ink-muted text-base">Partner not found.</Text>
        </View>
      )}

      {partner && (
        <ScrollView className="flex-1 px-4" contentContainerStyle={{ paddingBottom: 32 }}>
          {/* Partner header card */}
          <View className="bg-bone rounded-2xl p-5 mt-4 shadow-sm items-center">
            <View className="w-16 h-16 rounded-full bg-herb-tint items-center justify-center mb-3">
              <Text className="font-display text-2xl font-semibold text-herb">
                {partner.name.charAt(0).toUpperCase()}
              </Text>
            </View>
            <Text className="text-xl font-semibold text-ink mb-1">{partner.name}</Text>
            <View
              className={`px-3 py-1 rounded-full ${
                partner.status === 'active' ? 'bg-herb-tint' : 'bg-mist'
              }`}
            >
              <Text
                className={`text-sm font-medium ${
                  partner.status === 'active' ? 'text-herb' : 'text-ink-muted'
                }`}
              >
                {partner.status === 'active' ? 'Active' : 'Inactive'}
              </Text>
            </View>
          </View>

          {/* Detail rows */}
          <View className="bg-bone rounded-2xl px-4 mt-3 shadow-sm">
            <DetailRow label="Phone" value={partner.phone} />
            <DetailRow label="Rating" value={`\u2605 ${partner.rating.toFixed(1)}`} />
            <DetailRow label="Today's Deliveries" value={String(partner.todayDeliveries)} />
          </View>
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
