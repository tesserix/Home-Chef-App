// My tiffin subscriptions (#283): list + manage (pause / resume / skip / cancel)
// and a quick adherence + fulfillment glance. Orders are placed automatically by
// the platform at the chef's cutoff — no daily tapping.

import { ActivityIndicator, Alert, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import {
  useMealSubscriptions,
  useMealSubAction,
  useMealFulfillments,
  type MealSubscription,
} from '../hooks/useMealSubscription';

const STATUS_LABEL: Record<MealSubscription['status'], string> = {
  trialing: 'Trial',
  active: 'Active',
  paused: 'Paused',
  past_due: 'Payment due',
  cancelled: 'Cancelled',
};

function money(n: number): string {
  return `₹${Math.round(n).toLocaleString('en-IN')}`;
}

export default function SubscriptionsScreen() {
  const { data, isLoading } = useMealSubscriptions();
  const subs = data?.data ?? [];

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable onPress={() => router.back()} hitSlop={10} accessibilityLabel="Go back">
          <ChevronLeft size={26} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.headerTitle}>My subscriptions</Text>
        <View style={{ width: 26 }} />
      </View>

      {isLoading ? (
        <View style={styles.centered}><ActivityIndicator color={customerColors.coral.DEFAULT} /></View>
      ) : subs.length === 0 ? (
        <View style={styles.centered}><Text style={styles.muted}>No tiffin subscriptions yet.</Text></View>
      ) : (
        <ScrollView contentContainerStyle={styles.scroll}>
          {subs.map((s) => <SubCard key={s.id} sub={s} />)}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

function SubCard({ sub }: { sub: MealSubscription }) {
  const action = useMealSubAction();
  const { data: fulfil } = useMealFulfillments(sub.id);
  const adherence = fulfil?.adherence;
  const active = sub.status === 'active';
  const paused = sub.status === 'paused';
  const terminal = sub.status === 'cancelled';

  function run(a: 'pause' | 'resume' | 'cancel') {
    const confirm = a === 'cancel';
    const go = () => action.mutate({ id: sub.id, action: a });
    if (confirm) {
      Alert.alert('Cancel subscription?', 'You can resubscribe anytime.', [
        { text: 'Keep', style: 'cancel' },
        { text: 'Cancel it', style: 'destructive', onPress: go },
      ]);
    } else {
      go();
    }
  }

  return (
    <View style={styles.card}>
      <View style={styles.cardTop}>
        <Text style={styles.cardTitle}>{sub.slots.map((x) => (x === 'lunch' ? 'Lunch' : 'Dinner')).join(' + ')} · {sub.variant === 'veg' ? 'Veg' : 'Non-veg'}</Text>
        <View style={[styles.badge, active && styles.badgeActive, paused && styles.badgePaused]}>
          <Text style={[styles.badgeText, active && styles.badgeTextActive]}>{STATUS_LABEL[sub.status]}</Text>
        </View>
      </View>
      <Text style={styles.sub}>{sub.days.length} days/week · {sub.cadence === 'monthly' ? 'Monthly' : 'Weekly'} · {money(sub.cycleAmount)}</Text>
      {sub.creditBalance > 0 ? <Text style={styles.credit}>{money(sub.creditBalance)} credit applies to your next cycle</Text> : null}
      {adherence ? (
        <Text style={styles.adherence}>
          {adherence.delivered} delivered · {adherence.skipped} skipped · {adherence.missed} missed
        </Text>
      ) : null}

      {!terminal && (
        <View style={styles.actions}>
          {active && <ActionBtn label="Pause" onPress={() => run('pause')} />}
          {paused && <ActionBtn label="Resume" onPress={() => run('resume')} />}
          <ActionBtn label="Cancel" danger onPress={() => run('cancel')} />
        </View>
      )}
    </View>
  );
}

function ActionBtn({ label, onPress, danger }: { label: string; onPress: () => void; danger?: boolean }) {
  return (
    <Pressable onPress={onPress} accessibilityRole="button">
      <View style={[styles.actionBtn, danger && styles.actionBtnDanger]}>
        <Text style={[styles.actionText, danger && styles.actionTextDanger]}>{label}</Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.surface.soft },
  header: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', paddingHorizontal: 16, paddingVertical: 12 },
  headerTitle: { fontFamily: 'Geist-Bold', fontSize: 20, color: customerColors.charcoal.DEFAULT },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  scroll: { padding: 16, gap: 12 },
  muted: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft },
  card: { backgroundColor: customerColors.canvas, borderRadius: 12, padding: 16, gap: 6 },
  cardTop: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  cardTitle: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT, flex: 1 },
  sub: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft },
  credit: { fontFamily: 'Inter-SemiBold', fontSize: 12, color: customerColors.coral.pressed },
  adherence: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft },
  badge: { borderRadius: 9999, paddingHorizontal: 10, paddingVertical: 3, backgroundColor: customerColors.surface.soft },
  badgeActive: { backgroundColor: customerColors.coral.tint },
  badgePaused: { backgroundColor: customerColors.surface.soft },
  badgeText: { fontFamily: 'Inter-SemiBold', fontSize: 11, color: customerColors.charcoal.soft },
  badgeTextActive: { color: customerColors.coral.pressed },
  actions: { flexDirection: 'row', gap: 8, marginTop: 8 },
  actionBtn: { borderWidth: 1, borderColor: customerColors.hairline, borderRadius: 8, paddingHorizontal: 16, paddingVertical: 8 },
  actionBtnDanger: { borderColor: customerColors.coral.DEFAULT },
  actionText: { fontFamily: 'Inter-SemiBold', fontSize: 13, color: customerColors.charcoal.DEFAULT },
  actionTextDanger: { color: customerColors.coral.pressed },
});
