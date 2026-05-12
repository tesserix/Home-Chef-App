import { View, Text } from 'react-native';

interface DashboardStatsCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  color?: string;
}

export function DashboardStatsCard({ title, value, subtitle }: DashboardStatsCardProps) {
  return (
    <View className="flex-1 rounded-2xl border border-mist bg-bone p-4 shadow-sm">
      <Text className="text-xs font-medium uppercase tracking-wide text-ink-muted">{title}</Text>
      <Text className="mt-2 font-display text-3xl font-semibold tabular-nums text-herb">{value}</Text>
      {subtitle ? (
        <Text className="mt-1 text-xs text-ink-muted">{subtitle}</Text>
      ) : null}
    </View>
  );
}
