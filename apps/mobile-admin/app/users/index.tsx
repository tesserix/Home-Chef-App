import { useState } from 'react';
import { Alert, FlatList, RefreshControl, View } from 'react-native';
import { router } from 'expo-router';
import { Screen } from '@homechef/mobile-shared/ui';
import {
  useAdminUsers,
  useActivateUser,
  useSuspendUser,
} from '../../hooks/useAdminUsers';
import type { UserWithStats } from '../../lib/admin-types';
import {
  Badge,
  Empty,
  ErrorState,
  FilterChips,
  ListItem,
  LoadingList,
  ScreenHeader,
  SearchBar,
} from '../../components/kit';
import { formatINR, titleCase, errorMessage } from '../../lib/format';

const ROLE_OPTIONS = [
  { key: '', label: 'All' },
  { key: 'customer', label: 'Customers' },
  { key: 'chef', label: 'Chefs' },
  { key: 'delivery', label: 'Drivers' },
  { key: 'admin', label: 'Admins' },
];

export default function UsersScreen() {
  const [search, setSearch] = useState('');
  const [role, setRole] = useState('');
  const q = useAdminUsers({ search, role });
  const suspend = useSuspendUser();
  const activate = useActivateUser();
  const users = q.data?.data ?? [];

  const onRowPress = (u: UserWithStats) => {
    const name = `${u.firstName} ${u.lastName}`.trim() || u.email;
    Alert.alert(name, u.email, [
      {
        text: 'View wallet',
        onPress: () => router.push({ pathname: '/wallets', params: { userId: u.id } }),
      },
      u.isActive
        ? {
            text: 'Suspend',
            style: 'destructive',
            onPress: () =>
              suspend.mutate(u.id, {
                onError: (e) => Alert.alert('Failed', errorMessage(e)),
              }),
          }
        : {
            text: 'Activate',
            onPress: () =>
              activate.mutate(u.id, {
                onError: (e) => Alert.alert('Failed', errorMessage(e)),
              }),
          },
      { text: 'Cancel', style: 'cancel' },
    ]);
  };

  return (
    <Screen>
      <ScreenHeader
        title="Users"
        back
        subtitle={q.data ? `${q.data.pagination.total} registered` : 'All accounts'}
      />
      <View style={{ gap: 10, marginBottom: 4 }}>
        <SearchBar value={search} onChangeText={setSearch} placeholder="Search name or email…" />
        <FilterChips options={ROLE_OPTIONS} value={role} onChange={setRole} />
      </View>

      {q.isLoading ? (
        <LoadingList />
      ) : q.isError ? (
        <ErrorState message={errorMessage(q.error)} onRetry={() => q.refetch()} />
      ) : (
        <FlatList
          data={users}
          keyExtractor={(u) => u.id}
          showsVerticalScrollIndicator={false}
          contentContainerStyle={{ paddingBottom: 24 }}
          refreshControl={
            <RefreshControl refreshing={q.isRefetching} onRefresh={() => q.refetch()} tintColor="#0E0E0C" />
          }
          ListEmptyComponent={<Empty title="No users found" />}
          renderItem={({ item }) => {
            const name = `${item.firstName} ${item.lastName}`.trim() || item.email;
            return (
              <ListItem
                title={name}
                subtitle={`${item.email} · ${titleCase(item.role)}`}
                meta={`${item.totalOrders} orders · ${formatINR(item.totalSpent)} spent`}
                badge={
                  <Badge
                    label={item.isActive ? 'Active' : 'Suspended'}
                    tone={item.isActive ? 'success' : 'danger'}
                  />
                }
                chevron
                onPress={() => onRowPress(item)}
              />
            );
          }}
        />
      )}
    </Screen>
  );
}
