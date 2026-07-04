import { Alert, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { type Href, router } from 'expo-router';
import {
  BadgeCheck,
  ShieldAlert,
  AlertTriangle,
  Users,
  Star,
  Wallet,
  CalendarRange,
  Truck,
  UserCog,
  BarChart3,
  Settings as SettingsIcon,
  LogOut,
  ChevronRight,
} from 'lucide-react-native';
import { Screen } from '@homechef/mobile-shared/ui';
import { useAuth } from '@homechef/mobile-shared/auth';
import { theme } from '@homechef/mobile-shared/theme';
import { useAuthStore } from '../../store/auth-store';
import { ScreenHeader, SectionTitle } from '../../components/kit';

const c = theme.colors;

type Item = { label: string; sub: string; href: Href; icon: typeof Users };

const OPERATIONS: Item[] = [
  { label: 'Approvals', sub: 'Kitchen & document review queue', href: '/approvals', icon: BadgeCheck },
  { label: 'Users', sub: 'Customers & accounts', href: '/users', icon: Users },
  { label: 'FSSAI Lockouts', sub: 'Expired licences & overrides', href: '/fssai', icon: ShieldAlert },
  { label: 'Reviews', sub: 'Moderate ratings', href: '/reviews', icon: Star },
  { label: 'Cancellations', sub: 'Disputes & vendor timeouts', href: '/cancellations', icon: AlertTriangle },
  { label: 'Wallets', sub: 'Customer credit & adjustments', href: '/wallets', icon: Wallet },
  { label: 'Meal Plans', sub: 'Subscriptions oversight', href: '/meal-plans', icon: CalendarRange },
  { label: 'Delivery', sub: 'Partners & deliveries', href: '/delivery', icon: Truck },
];

const PLATFORM: Item[] = [
  { label: 'Staff', sub: 'Internal team & roles', href: '/staff', icon: UserCog },
  { label: 'Analytics', sub: 'Performance overview', href: '/analytics', icon: BarChart3 },
  { label: 'Settings', sub: 'Platform configuration', href: '/settings', icon: SettingsIcon },
];

function Row({ item }: { item: Item }) {
  const Icon = item.icon;
  return (
    <Pressable
      onPress={() => router.push(item.href)}
      style={({ pressed }) => [styles.row, pressed && styles.pressed]}
    >
      <View style={styles.iconWrap}>
        <Icon size={20} color={c.ink.DEFAULT} strokeWidth={2} />
      </View>
      <View style={{ flex: 1 }}>
        <Text style={styles.label}>{item.label}</Text>
        <Text style={styles.sub}>{item.sub}</Text>
      </View>
      <ChevronRight size={18} color={c.ink.muted} />
    </Pressable>
  );
}

export default function MoreScreen() {
  const { signOut } = useAuth();
  const user = useAuthStore((s) => s.user);

  const onLogout = () => {
    Alert.alert('Log out', 'Sign out of the admin console?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Log out',
        style: 'destructive',
        onPress: async () => {
          try {
            await signOut();
          } catch {
            /* ignore */
          }
          useAuthStore.getState().logout();
          router.replace('/(auth)/login');
        },
      },
    ]);
  };

  return (
    <Screen>
      <ScreenHeader title="More" subtitle={user?.email ?? 'Admin tools'} />
      <ScrollView showsVerticalScrollIndicator={false} contentContainerStyle={{ paddingBottom: 32 }}>
        <SectionTitle>Operations</SectionTitle>
        <View style={styles.group}>
          {OPERATIONS.map((it) => (
            <Row key={it.label} item={it} />
          ))}
        </View>

        <SectionTitle>Platform</SectionTitle>
        <View style={styles.group}>
          {PLATFORM.map((it) => (
            <Row key={it.label} item={it} />
          ))}
        </View>

        <SectionTitle>Account</SectionTitle>
        <View style={styles.group}>
          <Pressable
            onPress={onLogout}
            style={({ pressed }) => [styles.row, pressed && styles.pressed]}
          >
            <View style={[styles.iconWrap, { backgroundColor: c.destructive.tint }]}>
              <LogOut size={20} color={c.destructive.DEFAULT} strokeWidth={2} />
            </View>
            <View style={{ flex: 1 }}>
              <Text style={[styles.label, { color: c.destructive.DEFAULT }]}>Log out</Text>
              <Text style={styles.sub}>End this admin session</Text>
            </View>
          </Pressable>
        </View>
      </ScrollView>
    </Screen>
  );
}

const styles = StyleSheet.create({
  group: {
    backgroundColor: c.paper,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: c.mist.DEFAULT,
    overflow: 'hidden',
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingVertical: 12,
    paddingHorizontal: 12,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: c.mist.DEFAULT,
  },
  pressed: { backgroundColor: c.bone },
  iconWrap: {
    width: 38,
    height: 38,
    borderRadius: 10,
    backgroundColor: c.bone,
    alignItems: 'center',
    justifyContent: 'center',
  },
  label: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: c.ink.DEFAULT },
  sub: { fontFamily: 'Inter', fontSize: 12, color: c.ink.muted, marginTop: 1 },
});
