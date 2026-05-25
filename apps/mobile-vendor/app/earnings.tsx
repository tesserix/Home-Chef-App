import {
  ActivityIndicator,
  RefreshControl,
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

interface BankAccount {
  bankName: string;
  accountNumber: string;
  ifsc: string;
}

interface LastPayout {
  amount: number;
  date: string;
}

interface EarningsHistoryEntry {
  week: string;
  amount: number;
}

interface PayoutResponse {
  bankAccount: BankAccount | null;
  upiId: string | null;
  totalEarnings: number;
  pendingPayout: number;
  lastPayout: LastPayout | null;
  earningsHistory: EarningsHistoryEntry[];
}

function useEarnings() {
  return useQuery<PayoutResponse>({
    queryKey: ['chef', 'payout'],
    queryFn: () => api.get<PayoutResponse>('/chef/payout').then((r) => r.data),
    staleTime: 60_000,
  });
}

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <View className="flex-1 bg-bone rounded-2xl p-4 shadow-sm items-center">
      <Text className="text-xs text-ink-muted mb-1">{label}</Text>
      <Text className="text-lg font-medium text-ink">{value}</Text>
    </View>
  );
}

function maskAccount(accountNumber: string): string {
  if (accountNumber.length <= 4) return accountNumber;
  return '****' + accountNumber.slice(-4);
}

export default function EarningsScreen() {
  const { data, isLoading, isError, refetch, isRefetching } = useEarnings();

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-paper items-center justify-center">
        <ActivityIndicator size="large" color="#C2410C" />
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

  const maxAmount = Math.max(
    ...(data?.earningsHistory?.map((e) => e.amount) ?? [1]),
    1,
  );

  return (
    <SafeAreaView className="flex-1 bg-paper">
      {/* Header */}
      <View className="flex-row items-center px-4 pt-2 pb-3 bg-bone border-b border-mist">
        <TouchableOpacity accessibilityLabel="Go back" accessibilityRole="button" onPress={() => router.back()} activeOpacity={0.7} className="mr-3">
          <ChevronLeft size={24} color="#4a4a47" />
        </TouchableOpacity>
        <Text className="text-lg font-semibold text-ink">Earnings</Text>
      </View>

      <ScrollView
        className="flex-1"
        contentContainerStyle={{ padding: 16, paddingBottom: 32 }}
        refreshControl={
          <RefreshControl refreshing={isRefetching} onRefresh={refetch} tintColor="#C2410C" />
        }
        showsVerticalScrollIndicator={false}
      >
        {/* Summary cards */}
        <View className="flex-row gap-3 mb-4">
          <SummaryCard
            label="Total Earnings"
            value={`₹${(data?.totalEarnings ?? 0).toLocaleString('en-IN')}`}
          />
          <SummaryCard
            label="Pending Payout"
            value={`₹${(data?.pendingPayout ?? 0).toLocaleString('en-IN')}`}
          />
        </View>

        {data?.lastPayout && (
          <View className="bg-bone rounded-2xl p-4 shadow-sm mb-4">
            <Text className="text-sm font-semibold text-ink-soft mb-2">Last Payout</Text>
            <View className="flex-row justify-between">
              <Text className="text-base text-ink font-semibold">
                ₹{data.lastPayout.amount.toLocaleString('en-IN')}
              </Text>
              <Text className="text-sm text-ink-muted">
                {new Date(data.lastPayout.date).toLocaleDateString('en-IN', {
                  day: 'numeric',
                  month: 'short',
                  year: 'numeric',
                })}
              </Text>
            </View>
          </View>
        )}

        {/* Payout details */}
        <View className="bg-bone rounded-2xl p-4 shadow-sm mb-4">
          <Text className="text-sm font-semibold text-ink-soft mb-3">Payout Account</Text>
          {data?.bankAccount ? (
            <View>
              <Text className="text-xs text-ink-muted mb-0.5">Bank</Text>
              <Text className="text-base text-ink mb-2">{data.bankAccount.bankName}</Text>
              <Text className="text-xs text-ink-muted mb-0.5">Account Number</Text>
              <Text className="text-base text-ink mb-2">
                {maskAccount(data.bankAccount.accountNumber)}
              </Text>
              <Text className="text-xs text-ink-muted mb-0.5">IFSC</Text>
              <Text className="text-base text-ink">{data.bankAccount.ifsc}</Text>
            </View>
          ) : data?.upiId ? (
            <View>
              <Text className="text-xs text-ink-muted mb-0.5">UPI ID</Text>
              <Text className="text-base text-ink">{data.upiId}</Text>
            </View>
          ) : (
            <Text className="text-sm text-ink-muted">No payout account configured</Text>
          )}
        </View>

        {/* Weekly earnings bar chart */}
        {data?.earningsHistory && data.earningsHistory.length > 0 && (
          <View className="bg-bone rounded-2xl p-4 shadow-sm">
            <Text className="text-sm font-semibold text-ink-soft mb-4">Weekly Earnings</Text>
            <View className="gap-3">
              {data.earningsHistory.map((entry) => (
                <View key={entry.week} className="flex-row items-center gap-3">
                  <Text className="text-xs text-ink-muted w-16 text-right">{entry.week}</Text>
                  <View className="flex-1 h-6 bg-mist rounded-full overflow-hidden">
                    <View
                      className="h-6 bg-herb-soft rounded-full"
                      style={{ width: `${(entry.amount / maxAmount) * 100}%` }}
                    />
                  </View>
                  <Text className="text-xs text-ink-soft w-16">
                    ₹{entry.amount.toLocaleString('en-IN')}
                  </Text>
                </View>
              ))}
            </View>
          </View>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}
