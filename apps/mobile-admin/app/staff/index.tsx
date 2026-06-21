import { useState } from 'react';
import {
  Alert,
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { Plus } from 'lucide-react-native';
import { FlatList, RefreshControl } from 'react-native';
import { Screen } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import {
  useAdminStaff,
  useStaffRoles,
  useInviteStaff,
  useDeactivateStaff,
  useReactivateStaff,
} from '../../hooks/useAdminStaff';
import type { StaffMember } from '../../lib/admin-types';
import {
  Badge,
  Empty,
  ErrorState,
  FilterChips,
  ListItem,
  LoadingList,
  ScreenHeader,
} from '../../components/kit';
import { titleCase, errorMessage } from '../../lib/format';

const c = theme.colors;
const FALLBACK_ROLES = ['support', 'fleet_manager', 'delivery_ops', 'admin'];

export default function StaffScreen() {
  const q = useAdminStaff();
  const roles = useStaffRoles();
  const invite = useInviteStaff();
  const deactivate = useDeactivateStaff();
  const reactivate = useReactivateStaff();

  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState('');
  const [role, setRole] = useState('support');
  const [title, setTitle] = useState('');

  const roleOptions = (roles.data?.length
    ? roles.data.map((r) => ({ key: r.key, label: r.label ?? titleCase(r.key) }))
    : FALLBACK_ROLES.map((r) => ({ key: r, label: titleCase(r) })));

  const staff = q.data?.data ?? [];

  const reset = () => {
    setOpen(false);
    setEmail('');
    setRole('support');
    setTitle('');
  };

  const submitInvite = () => {
    if (!email.trim() || !email.includes('@')) {
      Alert.alert('Invalid email', 'Enter a valid email address.');
      return;
    }
    invite.mutate(
      { email: email.trim(), staffRole: role, title: title.trim() || undefined },
      {
        onError: (e) => Alert.alert('Failed', errorMessage(e)),
        onSuccess: () => {
          reset();
          Alert.alert('Invitation sent', `Invited ${email.trim()}.`);
        },
      }
    );
  };

  const onRowPress = (m: StaffMember) => {
    const name =
      m.user?.email || `${m.user?.firstName ?? ''} ${m.user?.lastName ?? ''}`.trim() || m.id;
    Alert.alert(name, titleCase(m.staffRole), [
      m.isActive
        ? {
            text: 'Deactivate',
            style: 'destructive',
            onPress: () =>
              deactivate.mutate(m.id, { onError: (e) => Alert.alert('Failed', errorMessage(e)) }),
          }
        : {
            text: 'Reactivate',
            onPress: () =>
              reactivate.mutate(m.id, { onError: (e) => Alert.alert('Failed', errorMessage(e)) }),
          },
      { text: 'Cancel', style: 'cancel' },
    ]);
  };

  return (
    <Screen>
      <ScreenHeader
        title="Staff"
        back
        subtitle={q.data ? `${q.data.pagination.total} team members` : 'Internal team'}
        right={
          <Pressable
            onPress={() => setOpen(true)}
            style={{ width: 38, height: 38, borderRadius: 10, backgroundColor: c.ink.DEFAULT, alignItems: 'center', justifyContent: 'center' }}
            accessibilityLabel="Invite staff"
          >
            <Plus size={20} color={c.paper} />
          </Pressable>
        }
      />

      {q.isLoading ? (
        <LoadingList />
      ) : q.isError ? (
        <ErrorState message={errorMessage(q.error)} onRetry={() => q.refetch()} />
      ) : (
        <FlatList
          data={staff}
          keyExtractor={(m) => m.id}
          showsVerticalScrollIndicator={false}
          contentContainerStyle={{ paddingBottom: 24 }}
          refreshControl={
            <RefreshControl refreshing={q.isRefetching} onRefresh={() => q.refetch()} tintColor="#0E0E0C" />
          }
          ListEmptyComponent={<Empty title="No staff" body="Invite your first team member." />}
          renderItem={({ item }) => (
            <ListItem
              title={
                item.user?.email ||
                `${item.user?.firstName ?? ''} ${item.user?.lastName ?? ''}`.trim() ||
                'Staff member'
              }
              subtitle={`${titleCase(item.staffRole)}${item.title ? ` · ${item.title}` : ''}`}
              meta={item.department ? titleCase(item.department) : undefined}
              badge={
                <Badge label={item.isActive ? 'Active' : 'Inactive'} tone={item.isActive ? 'success' : 'neutral'} />
              }
              onPress={() => onRowPress(item)}
            />
          )}
        />
      )}

      <Modal visible={open} transparent animationType="fade" onRequestClose={reset}>
        <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : undefined} style={styles.backdrop}>
          <Pressable style={StyleSheet.absoluteFill} onPress={reset} />
          <View style={styles.sheet}>
            <Text style={styles.title}>Invite staff</Text>
            <Text style={styles.label}>Email</Text>
            <TextInput
              value={email}
              onChangeText={setEmail}
              placeholder="name@fe3dr.com"
              placeholderTextColor={c.ink.muted}
              autoCapitalize="none"
              keyboardType="email-address"
              style={styles.input}
            />
            <Text style={styles.label}>Role</Text>
            <FilterChips options={roleOptions} value={role} onChange={setRole} />
            <Text style={styles.label}>Title (optional)</Text>
            <TextInput
              value={title}
              onChangeText={setTitle}
              placeholder="e.g. Support Lead"
              placeholderTextColor={c.ink.muted}
              style={styles.input}
            />
            <View style={{ flexDirection: 'row', gap: 10, marginTop: 12 }}>
              <Pressable onPress={reset} style={[styles.btn, styles.ghost]}>
                <Text style={styles.ghostText}>Cancel</Text>
              </Pressable>
              <Pressable onPress={submitInvite} style={[styles.btn, styles.primary]} disabled={invite.isPending}>
                <Text style={styles.primaryText}>{invite.isPending ? 'Sending…' : 'Send invite'}</Text>
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
    gap: 6,
  },
  title: { fontFamily: 'Geist', fontSize: 20, color: c.ink.DEFAULT, marginBottom: 6 },
  label: { fontFamily: 'Inter-Medium', fontSize: 12, color: c.ink.muted, marginTop: 6 },
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
