import { View, Text } from 'react-native';

interface DashboardStatsCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  color?: string;
}

export function DashboardStatsCard({ title, value, subtitle }: DashboardStatsCardProps) {
  return (
    <View className="flex-1 rounded-2xl border border-gray-100 bg-white p-4 shadow-sm">
      <Text className="text-xs font-medium uppercase tracking-wide text-gray-500">{title}</Text>
      <Text className="mt-2 text-3xl font-bold text-orange-500">{value}</Text>
      {subtitle ? (
        <Text className="mt-1 text-xs text-gray-400">{subtitle}</Text>
      ) : null}
    </View>
  );
}
