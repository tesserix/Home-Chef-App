import { useState } from 'react';
import {
  RefreshControl,
  ScrollView,
  Switch,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useDriverDashboard, useToggleOnline } from '../../hooks/useDriverDashboard';

type Period = 'today' | 'week' | 'month';

const PERIOD_LABELS: Record<Period, string> = {
  today: 'Today',
  week: 'Week',
  month: 'Month',
};

export default function DashboardScreen() {
  const [period, setPeriod] = useState<Period>('today');
  const { data: stats, isLoading, isError, refetch, isRefetching } = useDriverDashboard();
  const toggleOnlineMutation = useToggleOnline();

  const periodData = stats?.[period as keyof typeof stats] as { deliveries: number; earnings: number } | undefined;

  function handleToggleOnline(value: boolean) {
    toggleOnlineMutation.mutate(value);
  }

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50">
        <View className="px-4 pt-4 pb-2">
          <Text className="text-2xl font-bold text-gray-900">Dashboard</Text>
        </View>
        <View className="px-4 pt-2">
          <View className="h-16 bg-gray-200 rounded-2xl mb-3" />
          <View className="flex-row gap-3 mb-3">
            <View className="flex-1 h-24 bg-gray-200 rounded-2xl" />
            <View className="flex-1 h-24 bg-gray-200 rounded-2xl" />
          </View>
          <View className="flex-row gap-3">
            <View className="flex-1 h-24 bg-gray-200 rounded-2xl" />
            <View className="flex-1 h-24 bg-gray-200 rounded-2xl" />
          </View>
        </View>
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView className="flex-1 bg-gray-50 items-center justify-center px-6">
        <Text className="text-gray-500 text-base mb-4">Failed to load dashboard</Text>
        <TouchableOpacity
          onPress={() => refetch()}
          className="bg-orange-500 px-6 py-3 rounded-xl"
        >
          <Text className="text-white font-semibold">Retry</Text>
        </TouchableOpacity>
      </SafeAreaView>
    );
  }

  const isOnline = stats?.isOnline ?? false;

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
          <Text className="text-2xl font-bold text-gray-900">Dashboard</Text>
        </View>

        {/* Online / Offline Banner */}
        <View
          className={`mx-4 mb-4 rounded-2xl px-4 py-3 flex-row items-center justify-between ${
            isOnline ? 'bg-green-100' : 'bg-gray-200'
          }`}
        >
          <View>
            <Text
              className={`text-base font-semibold ${
                isOnline ? 'text-green-800' : 'text-gray-600'
              }`}
            >
              {isOnline ? 'You are Online' : 'You are Offline'}
            </Text>
            <Text
              className={`text-xs mt-0.5 ${isOnline ? 'text-green-600' : 'text-gray-500'}`}
            >
              {isOnline
                ? 'Receiving delivery requests'
                : 'Toggle to start receiving requests'}
            </Text>
          </View>
          <Switch
            value={isOnline}
            onValueChange={handleToggleOnline}
            trackColor={{ false: '#9CA3AF', true: '#22C55E' }}
            thumbColor="white"
            disabled={toggleOnlineMutation.isPending}
          />
        </View>

        {/* Period selector */}
        <View className="mx-4 mb-4 flex-row bg-gray-200 rounded-xl p-1">
          {(Object.keys(PERIOD_LABELS) as Period[]).map((p) => (
            <TouchableOpacity
              key={p}
              onPress={() => setPeriod(p)}
              className={`flex-1 py-2 rounded-lg items-center ${
                period === p ? 'bg-white shadow-sm' : ''
              }`}
              activeOpacity={0.7}
            >
              <Text
                className={`text-sm font-medium ${
                  period === p ? 'text-gray-900' : 'text-gray-500'
                }`}
              >
                {PERIOD_LABELS[p]}
              </Text>
            </TouchableOpacity>
          ))}
        </View>

        {/* Period stats cards */}
        <View className="mx-4 flex-row gap-3 mb-4">
          <View className="flex-1 bg-white rounded-2xl p-4 shadow-sm">
            <Text className="text-sm text-gray-500 mb-1">Deliveries</Text>
            <Text className="text-3xl font-bold text-gray-900">
              {periodData?.deliveries ?? 0}
            </Text>
          </View>
          <View className="flex-1 bg-white rounded-2xl p-4 shadow-sm">
            <Text className="text-sm text-gray-500 mb-1">Earnings</Text>
            <Text className="text-3xl font-bold text-orange-500">
              &#8377;{periodData?.earnings ?? 0}
            </Text>
          </View>
        </View>

        {/* Overall stats */}
        <View className="mx-4 flex-row gap-3 mb-8">
          <View className="flex-1 bg-white rounded-2xl p-4 shadow-sm">
            <Text className="text-sm text-gray-500 mb-1">Rating</Text>
            <Text className="text-2xl font-bold text-yellow-500">
              {stats?.rating?.toFixed(1) ?? '\u2014'} {'\u2b50'}
            </Text>
          </View>
          <View className="flex-1 bg-white rounded-2xl p-4 shadow-sm">
            <Text className="text-sm text-gray-500 mb-1">Total Deliveries</Text>
            <Text className="text-2xl font-bold text-gray-900">
              {stats?.totalDeliveries ?? 0}
            </Text>
          </View>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
