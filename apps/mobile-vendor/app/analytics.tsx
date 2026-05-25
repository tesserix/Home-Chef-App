import { useState } from 'react';
import {
  ActivityIndicator,
  ScrollView,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useQuery } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import { api } from '../lib/api';

type Period = 'week' | 'month' | 'year';

interface PopularItem {
  name: string;
  orders: number;
  revenue: number;
}

interface DailyRevenue {
  date: string;
  revenue: number;
}

interface AnalyticsResponse {
  period: string;
  totalOrders: number;
  totalRevenue: number;
  popularItems: PopularItem[];
  dailyRevenue: DailyRevenue[];
}

function useAnalytics(period: Period) {
  return useQuery<AnalyticsResponse>({
    queryKey: ['chef', 'analytics', period],
    queryFn: () =>
      api.get<AnalyticsResponse>(`/chef/analytics?period=${period}`).then((r) => r.data),
    staleTime: 60_000,
  });
}

const PERIODS: { label: string; value: Period }[] = [
  { label: 'Week', value: 'week' },
  { label: 'Month', value: 'month' },
  { label: 'Year', value: 'year' },
];

export default function AnalyticsScreen() {
  const [period, setPeriod] = useState<Period>('week');
  const { data, isLoading, isError, refetch } = useAnalytics(period);

  const maxRevenue = Math.max(...(data?.dailyRevenue?.map((d) => d.revenue) ?? [1]), 1);

  return (
    <SafeAreaView className="flex-1 bg-paper">
      {/* Header */}
      <View className="flex-row items-center px-4 pt-2 pb-3 bg-bone border-b border-mist">
        <TouchableOpacity accessibilityLabel="Go back" accessibilityRole="button" onPress={() => router.back()} activeOpacity={0.7} className="mr-3">
          <ChevronLeft size={24} color="#4a4a47" />
        </TouchableOpacity>
        <Text className="text-lg font-semibold text-ink">Analytics</Text>
      </View>

      {/* Period selector */}
      <View className="flex-row px-4 py-3 bg-bone border-b border-mist gap-2">
        {PERIODS.map((p) => (
          <TouchableOpacity
            key={p.value}
            onPress={() => setPeriod(p.value)}
            className={`flex-1 py-2 rounded-xl items-center ${
              period === p.value ? 'bg-herb' : 'bg-mist'
            }`}
            activeOpacity={0.7}
          >
            <Text
              className={`text-sm font-semibold ${
                period === p.value ? 'text-paper' : 'text-ink-soft'
              }`}
            >
              {p.label}
            </Text>
          </TouchableOpacity>
        ))}
      </View>

      {isLoading ? (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color="#C2410C" />
        </View>
      ) : isError ? (
        <View className="flex-1 items-center justify-center px-6">
          <Text className="text-ink-muted text-base mb-4">Failed to load analytics</Text>
          <TouchableOpacity
            onPress={() => refetch()}
            className="bg-herb px-6 py-3 rounded-xl"
          >
            <Text className="text-paper font-semibold">Retry</Text>
          </TouchableOpacity>
        </View>
      ) : (
        <ScrollView
          className="flex-1"
          contentContainerStyle={{ padding: 16, paddingBottom: 32 }}
          showsVerticalScrollIndicator={false}
        >
          {/* Summary */}
          <View className="flex-row gap-3 mb-4">
            <View className="flex-1 bg-bone rounded-2xl p-4 shadow-sm items-center">
              <Text className="text-xs text-ink-muted mb-1">Total Orders</Text>
              <Text className="text-xl font-semibold text-ink">{data?.totalOrders ?? 0}</Text>
            </View>
            <View className="flex-1 bg-bone rounded-2xl p-4 shadow-sm items-center">
              <Text className="text-xs text-ink-muted mb-1">Total Revenue</Text>
              <Text className="text-xl font-semibold text-ink">
                ₹{(data?.totalRevenue ?? 0).toLocaleString('en-IN')}
              </Text>
            </View>
          </View>

          {/* Popular Items */}
          {data?.popularItems && data.popularItems.length > 0 && (
            <View className="bg-bone rounded-2xl p-4 shadow-sm mb-4">
              <Text className="text-sm font-semibold text-ink-soft mb-3">Popular Items</Text>
              {data.popularItems.slice(0, 5).map((item, index) => (
                <View
                  key={item.name}
                  className={`flex-row items-center py-3 ${
                    index < data.popularItems.length - 1 ? 'border-b border-mist' : ''
                  }`}
                >
                  <View className="w-7 h-7 rounded-full bg-herb-tint items-center justify-center mr-3">
                    <Text className="text-xs font-medium text-herb">{index + 1}</Text>
                  </View>
                  <Text className="flex-1 text-sm text-ink font-medium" numberOfLines={1}>
                    {item.name}
                  </Text>
                  <View className="items-end">
                    <Text className="text-xs text-ink-muted">{item.orders} orders</Text>
                    <Text className="text-xs text-herb font-medium">
                      ₹{item.revenue.toLocaleString('en-IN')}
                    </Text>
                  </View>
                </View>
              ))}
            </View>
          )}

          {/* Daily Revenue Chart */}
          {data?.dailyRevenue && data.dailyRevenue.length > 0 && (
            <View className="bg-bone rounded-2xl p-4 shadow-sm">
              <Text className="text-sm font-semibold text-ink-soft mb-4">Revenue Trend</Text>
              <View className="gap-2">
                {data.dailyRevenue.map((entry) => (
                  <View key={entry.date} className="flex-row items-center gap-3">
                    <Text className="text-xs text-ink-muted w-16 text-right" numberOfLines={1}>
                      {entry.date}
                    </Text>
                    <View className="flex-1 h-5 bg-mist rounded-full overflow-hidden">
                      <View
                        className="h-5 bg-herb-soft rounded-full"
                        style={{ width: `${(entry.revenue / maxRevenue) * 100}%` }}
                      />
                    </View>
                    <Text className="text-xs text-ink-soft w-16">
                      ₹{entry.revenue.toLocaleString('en-IN')}
                    </Text>
                  </View>
                ))}
              </View>
            </View>
          )}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
