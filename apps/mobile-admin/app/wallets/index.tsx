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
import { useLocalSearchParams } from 'expo-router';
import { Screen, Button } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { useCustomerWallet, useAdjustWallet } from '../../hooks/useAdminWallet';
import {
  Badge,
  Card,
  Empty,
  ErrorState,
  Field,
  FilterChips,
  LoadingState,
  ScreenHeader,
  SearchBar,
  SectionTitle,
} from '../../components/kit';
import { formatINR, formatDateTime, titleCase, errorMessage } from '../../lib/format';

const c = theme.colors;

export default function WalletsScreen() {
  const params = useLocalSearchParams<{ userId?: string }>();
  const [userId, setUserId] = useState(params.userId ?? '');
  const [active, setActive] = useState(params.userId ?? '');
  const q = useCustomerWallet(active);
  const adjust = useAdjustWallet();

  const [open, setOpen] = useState(false);
  const [amount, setAmount] = useState('');
  const [reason, setReason] = useState('');
  const [type, setType] = useState<'credit' | 'debit'>('credit');

  const reset = () => {
    setOpen(false);
    setAmount('');
    setReason('');
    setType('credit');
  };

  const submit = () => {
    const amt = Number(amount);
    if (!Number.isFinite(amt) || amt <= 0) {
      Alert.alert('Invalid amount', 'Enter an amount greater than zero.');
      return;
    }
    if (reason.trim().length === 0) {
      Alert.alert('Reason required', 'Add a reason for the adjustment.');
      return;
    }
    adjust.mutate(
      { userId: active, amount: amt, reason: reason.trim(), type },
      {
        onError: (e) => Alert.alert('Failed', errorMessage(e)),
        onSuccess: () => {
          reset();
          Alert.alert('Done', 'Wallet adjusted.');
        },
      }
    );
  };

  return (
    <Screen>
      <ScreenHeader title="Wallets" back subtitle="Customer store credit" />
      <View style={{ flexDirection: 'row', gap: 8, marginBottom: 8 }}>
        <View style={{ flex: 1 }}>
          <SearchBar value={userId} onChangeText={setUserId} placeholder="Customer user ID…" />
        </View>
        <Pressable
          onPress={() => setActive(userId.trim())}
          style={{ paddingHorizontal: 16, borderRadius: 10, backgroundColor: c.ink.DEFAULT, alignItems: 'center', justifyContent: 'center' }}
        >
          <Text style={{ color: c.paper, fontFamily: 'Inter-SemiBold', fontSize: 14 }}>Load</Text>
        </Pressable>
      </View>

      {!active ? (
        <Empty title="Look up a wallet" body="Enter a customer user ID, or open a wallet from the Users list." />
      ) : q.isLoading ? (
        <LoadingState label="Loading wallet…" />
      ) : q.isError || !q.data ? (
        <ErrorState message={errorMessage(q.error)} onRetry={() => q.refetch()} />
      ) : (
        <ScrollView showsVerticalScrollIndicator={false} contentContainerStyle={{ paddingBottom: 40 }}>
          <Card>
            <Text style={{ fontFamily: 'Inter', fontSize: 13, color: c.ink.muted }}>Balance</Text>
            <Text style={{ fontFamily: 'Geist-Bold', fontSize: 32, color: c.ink.DEFAULT, marginTop: 2 }}>
              {formatINR(q.data.balance)}
            </Text>
            <View style={{ marginTop: 12 }}>
              <Button label="Adjust balance" onPress={() => setOpen(true)} />
            </View>
          </Card>

          <SectionTitle>Transactions ({q.data.count})</SectionTitle>
          {q.data.transactions.length === 0 ? (
            <Empty title="No transactions" />
          ) : (
            <Card style={{ paddingVertical: 4 }}>
              {q.data.transactions.map((t) => (
                <View
                  key={t.id}
                  style={{ paddingVertical: 10, borderBottomWidth: 1, borderBottomColor: c.mist.DEFAULT }}
                >
                  <View style={{ flexDirection: 'row', justifyContent: 'space-between' }}>
                    <Text style={{ fontFamily: 'Inter-Medium', fontSize: 14, color: c.ink.DEFAULT }}>
                      {titleCase(t.source)}
                    </Text>
                    <Text
                      style={{
                        fontFamily: 'Inter-SemiBold',
                        fontSize: 14,
                        color: t.type === 'credit' ? c.success.soft : c.destructive.DEFAULT,
                      }}
                    >
                      {t.type === 'credit' ? '+' : '-'}
                      {formatINR(t.amount)}
                    </Text>
                  </View>
                  <Text style={{ fontFamily: 'Inter', fontSize: 12, color: c.ink.muted, marginTop: 2 }}>
                    {t.reason} · {formatDateTime(t.createdAt)}
                  </Text>
                </View>
              ))}
            </Card>
          )}
        </ScrollView>
      )}

      <Modal visible={open} transparent animationType="fade" onRequestClose={reset}>
        <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : undefined} style={styles.backdrop}>
          <Pressable style={StyleSheet.absoluteFill} onPress={reset} />
          <View style={styles.sheet}>
            <Text style={styles.title}>Adjust wallet</Text>
            <FilterChips
              options={[
                { key: 'credit', label: 'Credit (+)' },
                { key: 'debit', label: 'Debit (−)' },
              ]}
              value={type}
              onChange={(v) => setType(v as 'credit' | 'debit')}
            />
            <Text style={styles.label}>Amount (₹)</Text>
            <TextInput
              value={amount}
              onChangeText={setAmount}
              placeholder="0.00"
              placeholderTextColor={c.ink.muted}
              keyboardType="numeric"
              style={styles.input}
            />
            <Text style={styles.label}>Reason</Text>
            <TextInput
              value={reason}
              onChangeText={setReason}
              placeholder="Why this adjustment?"
              placeholderTextColor={c.ink.muted}
              style={styles.input}
            />
            <View style={{ flexDirection: 'row', gap: 10, marginTop: 10 }}>
              <Pressable onPress={reset} style={[styles.btn, styles.ghost]}>
                <Text style={styles.ghostText}>Cancel</Text>
              </Pressable>
              <Pressable onPress={submit} style={[styles.btn, styles.primary]} disabled={adjust.isPending}>
                <Text style={styles.primaryText}>{adjust.isPending ? 'Saving…' : 'Apply'}</Text>
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
  title: { fontFamily: 'Geist', fontSize: 20, color: c.ink.DEFAULT, marginBottom: 4 },
  label: { fontFamily: 'Inter-Medium', fontSize: 12, color: c.ink.muted, marginTop: 4 },
  input: {
    backgroundColor: c.bone,
    borderRadius: 10,
    padding: 12,
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
