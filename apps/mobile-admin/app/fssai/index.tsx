import { useState } from 'react';
import {
  Alert,
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { Screen, Button } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import {
  useFSSAILocked,
  useOverrideFSSAILock,
  useClearFSSAIOverride,
} from '../../hooks/useAdminFssai';
import type { FSSAILockedChef } from '../../lib/admin-types';
import {
  Badge,
  Card,
  Empty,
  ErrorState,
  Field,
  FilterChips,
  LoadingState,
  ScreenHeader,
  SectionTitle,
} from '../../components/kit';
import { formatDate, errorMessage } from '../../lib/format';

const c = theme.colors;
const DAY_OPTIONS = [
  { key: '7', label: '7 days' },
  { key: '14', label: '14 days' },
  { key: '30', label: '30 days' },
];

export default function FSSAIScreen() {
  const q = useFSSAILocked();
  const override = useOverrideFSSAILock();
  const clear = useClearFSSAIOverride();
  const [target, setTarget] = useState<FSSAILockedChef | null>(null);
  const [reason, setReason] = useState('');
  const [days, setDays] = useState('7');

  const closeModal = () => {
    setTarget(null);
    setReason('');
    setDays('7');
  };

  const submitOverride = () => {
    if (!target || reason.trim().length < 10) {
      Alert.alert('Reason too short', 'Please enter at least 10 characters.');
      return;
    }
    override.mutate(
      { chefId: target.chefId, reason: reason.trim(), days: Number(days) },
      {
        onError: (e) => Alert.alert('Failed', errorMessage(e)),
        onSuccess: () => {
          closeModal();
          Alert.alert('Override granted', `Lock lifted for ${days} days.`);
        },
      }
    );
  };

  const doClear = (ch: FSSAILockedChef) => {
    Alert.alert('Clear override', `Re-lock ${ch.businessName}?`, [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Clear',
        style: 'destructive',
        onPress: () =>
          clear.mutate(ch.chefId, { onError: (e) => Alert.alert('Failed', errorMessage(e)) }),
      },
    ]);
  };

  return (
    <Screen>
      <ScreenHeader
        title="FSSAI Lockouts"
        back
        subtitle={
          q.data
            ? `${q.data.lockedCount} locked · ${q.data.overriddenCount} overridden`
            : 'Expired licences'
        }
      />
      {q.isLoading ? (
        <LoadingState label="Loading…" />
      ) : q.isError || !q.data ? (
        <ErrorState message={errorMessage(q.error)} onRetry={() => q.refetch()} />
      ) : (
        <ScrollView showsVerticalScrollIndicator={false} contentContainerStyle={{ paddingBottom: 40 }}>
          {q.data.missingExpiryCount > 0 ? (
            <Card style={{ marginBottom: 4 }}>
              <Text style={{ fontFamily: 'Inter', fontSize: 13, color: c.ink.soft }}>
                {q.data.missingExpiryCount} chef(s) have no FSSAI expiry on record.
              </Text>
            </Card>
          ) : null}

          <SectionTitle>Locked ({q.data.lockedCount})</SectionTitle>
          {q.data.locked.length === 0 ? (
            <Empty title="None locked" />
          ) : (
            q.data.locked.map((ch) => (
              <Card key={ch.chefId} style={{ marginBottom: 10 }}>
                <View style={{ flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text style={{ fontFamily: 'Inter-SemiBold', fontSize: 15, color: c.ink.DEFAULT, flex: 1 }}>
                    {ch.businessName}
                  </Text>
                  <Badge label={`${ch.daysSinceExpiry}d expired`} tone="danger" />
                </View>
                <Field label="FSSAI expiry" value={ch.fssaiExpiry ? formatDate(ch.fssaiExpiry) : 'Unknown'} />
                <Button
                  label="Grant temporary override"
                  variant="secondary"
                  onPress={() => setTarget(ch)}
                />
              </Card>
            ))
          )}

          <SectionTitle>Overridden ({q.data.overriddenCount})</SectionTitle>
          {q.data.overridden.length === 0 ? (
            <Empty title="No active overrides" />
          ) : (
            q.data.overridden.map((ch) => (
              <Card key={ch.chefId} style={{ marginBottom: 10 }}>
                <View style={{ flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text style={{ fontFamily: 'Inter-SemiBold', fontSize: 15, color: c.ink.DEFAULT, flex: 1 }}>
                    {ch.businessName}
                  </Text>
                  <Badge label="Override active" tone="info" />
                </View>
                <Field label="Override until" value={ch.overrideUntil ? formatDate(ch.overrideUntil) : '—'} />
                {ch.overrideReason ? <Field label="Reason" value={ch.overrideReason} /> : null}
                <Button label="Clear override" variant="destructive" onPress={() => doClear(ch)} />
              </Card>
            ))
          )}
        </ScrollView>
      )}

      <Modal visible={!!target} transparent animationType="fade" onRequestClose={closeModal}>
        <KeyboardAvoidingView
          behavior={Platform.OS === 'ios' ? 'padding' : undefined}
          style={styles.backdrop}
        >
          <Pressable style={StyleSheet.absoluteFill} onPress={closeModal} />
          <View style={styles.sheet}>
            <Text style={styles.title}>Override FSSAI lock</Text>
            <Text style={styles.msg}>{target?.businessName}</Text>
            <Text style={styles.fieldLabel}>Duration</Text>
            <FilterChips options={DAY_OPTIONS} value={days} onChange={setDays} />
            <Text style={[styles.fieldLabel, { marginTop: 8 }]}>Reason (min 10 chars)</Text>
            <TextInput
              value={reason}
              onChangeText={setReason}
              placeholder="Why is this override justified?"
              placeholderTextColor={c.ink.muted}
              multiline
              style={styles.input}
            />
            <View style={{ flexDirection: 'row', gap: 10, marginTop: 8 }}>
              <Pressable onPress={closeModal} style={[styles.btn, styles.ghost]}>
                <Text style={styles.ghostText}>Cancel</Text>
              </Pressable>
              <Pressable onPress={submitOverride} style={[styles.btn, styles.primary]} disabled={override.isPending}>
                <Text style={styles.primaryText}>{override.isPending ? 'Saving…' : 'Grant override'}</Text>
              </Pressable>
            </View>
          </View>
        </KeyboardAvoidingView>
      </Modal>
    </Screen>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: 'flex-end', backgroundColor: 'rgba(14,14,12,0.45)' },
  sheet: {
    backgroundColor: c.paper,
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
    padding: 20,
    paddingBottom: 32,
    gap: 8,
  },
  title: { fontFamily: 'Geist', fontSize: 20, color: c.ink.DEFAULT },
  msg: { fontFamily: 'Inter', fontSize: 13, color: c.ink.soft, marginBottom: 4 },
  fieldLabel: { fontFamily: 'Inter-Medium', fontSize: 12, color: c.ink.muted },
  input: {
    backgroundColor: c.bone,
    borderRadius: 10,
    padding: 12,
    minHeight: 80,
    textAlignVertical: 'top',
    fontFamily: 'Inter',
    fontSize: 15,
    color: c.ink.DEFAULT,
  },
  btn: { flex: 1, height: 48, borderRadius: 12, alignItems: 'center', justifyContent: 'center' },
  ghost: { backgroundColor: c.bone },
  ghostText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: c.ink.DEFAULT },
  primary: { backgroundColor: c.ink.DEFAULT },
  primaryText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: c.paper },
});
