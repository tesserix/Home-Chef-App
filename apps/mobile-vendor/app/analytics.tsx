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
    <SafeAreaView className="flex-1 bg-gray-50">
      {/* Header */}
      <View className="flex-row items-center px-4 pt-2 pb-3 bg-white border-b border-gray-100">
        <TouchableOpacity onPress={() => router.back()} activeOpacity={0.7} className="mr-3">
          <ChevronLeft size={24} color="#374151" />
        </TouchableOpacity>
        <Text className="text-lg font-semibold text-gray-900">Analytics</Text>
      </View>

      {/* Period selector */}
      <View className="flex-row px-4 py-3 bg-white border-b border-gray-100 gap-2">
        {PERIODS.map((p) => (
          <TouchableOpacity
            key={p.value}
            onPress={() => setPeriod(p.value)}
            className={`flex-1 py-2 rounded-xl items-center ${
              period === p.value ? 'bg-orange-500' : 'bg-gray-100'
            }`}
            activeOpacity={0.7}
          >
            <Text
              className={`text-sm font-semibold ${
                period === p.value ? 'text-white' : 'text-gray-600'
              }`}
            >
              {p.label}
            </Text>
          </TouchableOpacity>
        ))}
      </View>

      {isLoading ? (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color="#FF6B35" />
        </View>
      ) : isError ? (
        <View className="flex-1 items-center justify-center px-6">
          <Text className="text-gray-500 text-base mb-4">Failed to load analytics</Text>
          <TouchableOpacity
            onPress={() => refetch()}
            className="bg-orange-500 px-6 py-3 rounded-xl"
          >
            <Text className="text-white font-semibold">Retry</Text>
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
            <View className="flex-1 bg-white rounded-2xl p-4 shadow-sm items-center">
              <Text className="text-xs text-gray-400 mb-1">Total Orders</Text>
              <Text className="text-xl font-bold text-gray-900">{data?.totalOrders ?? 0}</Text>
            </View>
            <View className="flex-1 bg-white rounded-2xl p-4 shadow-sm items-center">
              <Text className="text-xs text-gray-400 mb-1">Total Revenue</Text>
              <Text className="text-xl font-bold text-gray-900">
                ₹{(data?.totalRevenue ?? 0).toLocaleString('en-IN')}
              </Text>
            </View>
          </View>

          {/* Popular Items */}
          {data?.popularItems && data.popularItems.length > 0 && (
            <View className="bg-white rounded-2xl p-4 shadow-sm mb-4">
              <Text className="text-sm font-semibold text-gray-700 mb-3">Popular Items</Text>
              {data.popularItems.slice(0, 5).map((item, index) => (
                <View
                  key={item.name}
                  className={`flex-row items-center py-3 ${
                    index < data.popularItems.length - 1 ? 'border-b border-gray-100' : ''
                  }`}
                >
                  <View className="w-7 h-7 rounded-full bg-orange-100 items-center justify-center mr-3">
                    <Text className="text-xs font-bold text-orange-600">{index + 1}</Text>
                  </View>
                  <Text className="flex-1 text-sm text-gray-800 font-medium" numberOfLines={1}>
                    {item.name}
                  </Text>
                  <View className="items-end">
                    <Text className="text-xs text-gray-500">{item.orders} orders</Text>
                    <Text className="text-xs text-orange-600 font-medium">
                      ₹{item.revenue.toLocaleString('en-IN')}
                    </Text>
                  </View>
                </View>
              ))}
            </View>
          )}

          {/* Daily Revenue Chart */}
          {data?.dailyRevenue && data.dailyRevenue.length > 0 && (
            <View className="bg-white rounded-2xl p-4 shadow-sm">
              <Text className="text-sm font-semibold text-gray-700 mb-4">Revenue Trend</Text>
              <View className="gap-2">
                {data.dailyRevenue.map((entry) => (
                  <View key={entry.date} className="flex-row items-center gap-3">
                    <Text className="text-xs text-gray-400 w-16 text-right" numberOfLines={1}>
                      {entry.date}
                    </Text>
                    <View className="flex-1 h-5 bg-gray-100 rounded-full overflow-hidden">
                      <View
                        className="h-5 bg-orange-400 rounded-full"
                        style={{ width: `${(entry.revenue / maxRevenue) * 100}%` }}
                      />
                    </View>
                    <Text className="text-xs text-gray-700 w-16">
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
