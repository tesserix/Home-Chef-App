import { useState } from 'react';
import {
  ActivityIndicator,
  RefreshControl,
  ScrollView,
  Text,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

interface WeeklyHistory {
  week: string;
  amount: number;
  deliveries: number;
}

interface DriverEarnings {
  today: number;
  week: number;
  month: number;
  total: number;
  pending: number;
  lastPayout: { amount: number; date: string } | null;
  history: WeeklyHistory[];
}

type Period = 'today' | 'week' | 'month';

const PERIOD_LABELS: Record<Period, string> = {
  today: 'Today',
  week: 'Week',
  month: 'Month',
};

function useDriverEarnings() {
  return useQuery<DriverEarnings>({
    queryKey: ['driver', 'earnings'],
    queryFn: () => api.get('/delivery/earnings').then((r) => r.data as DriverEarnings),
  });
}

function WeekBar({ week, amount, maxAmount }: { week: string; amount: number; maxAmount: number }) {
  const barHeight = maxAmount > 0 ? Math.max((amount / maxAmount) * 80, 4) : 4;
  return (
    <View className="flex-1 items-center">
      <Text className="text-xs text-herb mb-1 font-medium">
        {amount > 0 ? `\u20B9${Math.round(amount)}` : ''}
      </Text>
      <View className="w-full items-center justify-end" style={{ height: 80 }}>
        <View
          className="w-5 bg-herb-soft rounded-t"
          style={{ height: barHeight }}
        />
      </View>
      <Text className="text-xs text-ink-muted mt-1" numberOfLines={1}>
        {week.slice(5)}
      </Text>
    </View>
  );
}

export default function DriverEarningsScreen() {
  const [period, setPeriod] = useState<Period>('week');
  const { data: earnings, isLoading, isError, refetch, isRefetching } = useDriverEarnings();

  const periodAmount = earnings?.[period] ?? 0;
  const history = earnings?.history ?? [];
  const maxAmount = history.reduce((m, h) => Math.max(m, h.amount), 0);

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-paper items-center justify-center">
        <ActivityIndicator size="large" color="#3e6b3c" />
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView className="flex-1 bg-paper items-center justify-center px-6">
        <Text className="text-ink-muted text-base mb-4">Failed to load earnings</Text>
        <TouchableOpacity
          onPress={() => refetch()}
          className="bg-herb px-6 py-3 rounded-xl"
        >
          <Text className="text-paper font-semibold">Retry</Text>
        </TouchableOpacity>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-paper">
      <ScrollView
        className="flex-1"
        refreshControl={
          <RefreshControl refreshing={isRefetching} onRefresh={refetch} tintColor="#3e6b3c" />
        }
        contentContainerStyle={{ paddingBottom: 32 }}
      >
        <View className="px-4 pt-4 pb-2">
          <Text className="font-display text-2xl font-semibold text-ink">Earnings</Text>
        </View>

        {/* Period selector */}
        <View className="mx-4 mb-4 flex-row bg-mist rounded-xl p-1">
          {(Object.keys(PERIOD_LABELS) as Period[]).map((p) => (
            <TouchableOpacity
              key={p}
              onPress={() => setPeriod(p)}
              className={`flex-1 py-2 rounded-lg items-center ${period === p ? 'bg-bone shadow-sm' : ''}`}
              activeOpacity={0.7}
            >
              <Text
                className={`text-sm font-medium ${period === p ? 'text-ink' : 'text-ink-muted'}`}
              >
                {PERIOD_LABELS[p]}
              </Text>
            </TouchableOpacity>
          ))}
        </View>

        {/* Period amount */}
        <View className="mx-4 bg-bone rounded-2xl p-6 shadow-sm mb-4 items-center">
          <Text className="text-sm text-ink-muted mb-1">{PERIOD_LABELS[period]} Earnings</Text>
          <Text className="font-display text-5xl font-semibold tabular-nums text-herb">
            &#8377;{periodAmount.toFixed(2)}
          </Text>
        </View>

        {/* Summary cards */}
        <View className="mx-4 flex-row gap-3 mb-4">
          <View className="flex-1 bg-bone rounded-2xl p-4 shadow-sm">
            <Text className="text-xs text-ink-muted mb-1">Total Earned</Text>
            <Text className="text-xl font-semibold text-ink">
              &#8377;{(earnings?.total ?? 0).toFixed(0)}
            </Text>
          </View>
          <View className="flex-1 bg-bone rounded-2xl p-4 shadow-sm">
            <Text className="text-xs text-ink-muted mb-1">Pending Payout</Text>
            <Text className="text-xl font-semibold text-amber">
              &#8377;{(earnings?.pending ?? 0).toFixed(0)}
            </Text>
          </View>
        </View>

        {/* Last payout */}
        {earnings?.lastPayout && (
          <View className="mx-4 bg-bone rounded-2xl p-4 shadow-sm mb-4">
            <Text className="text-sm text-ink-muted mb-1">Last Payout</Text>
            <View className="flex-row justify-between items-center">
              <Text className="text-base font-semibold text-ink">
                &#8377;{earnings.lastPayout.amount.toFixed(2)}
              </Text>
              <Text className="text-sm text-ink-muted">
                {new Date(earnings.lastPayout.date).toLocaleDateString('en-IN', {
                  day: 'numeric',
                  month: 'short',
                  year: 'numeric',
                })}
              </Text>
            </View>
          </View>
        )}

        {/* Weekly history bar chart */}
        {history.length > 0 && (
          <View className="mx-4 bg-bone rounded-2xl p-4 shadow-sm">
            <Text className="text-sm font-semibold text-ink-soft mb-4">Weekly History</Text>
            <View className="flex-row gap-1 items-end">
              {history.slice(-8).map((h) => (
                <WeekBar
                  key={h.week}
                  week={h.week}
                  amount={h.amount}
                  maxAmount={maxAmount}
                />
              ))}
            </View>
          </View>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}
