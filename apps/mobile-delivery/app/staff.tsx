import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  FlatList,
  Modal,
  RefreshControl,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Lock, UserPlus, X } from 'lucide-react-native';
import { useForm, Controller } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { api } from '../lib/api';

interface StaffMember {
  id: string;
  name: string;
  email: string;
  role: string;
  joinedAt: string;
}

const inviteSchema = z.object({
  email: z.string().email('Enter a valid email address'),
  role: z.string().min(1, 'Role is required'),
});
type InviteForm = z.infer<typeof inviteSchema>;

function useStaffList() {
  return useQuery<StaffMember[] | null>({
    queryKey: ['driver', 'staff'],
    queryFn: async () => {
      try {
        const r = await api.get('/delivery/staff');
        return r.data as StaffMember[];
      } catch (e: unknown) {
        if (
          e !== null &&
          typeof e === 'object' &&
          'response' in e &&
          (e as { response?: { status?: number } }).response?.status === 403
        ) {
          return null; // not-authorized, not an error
        }
        throw e;
      }
    },
  });
}

function useInviteStaff() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: InviteForm) =>
      api.post('/delivery/staff/invitations', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['driver', 'staff'] });
    },
  });
}

function StaffCard({ member }: { member: StaffMember }) {
  const joined = new Date(member.joinedAt).toLocaleDateString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
  return (
    <View className="bg-bone rounded-2xl p-4 mb-3 shadow-sm">
      <View className="flex-row items-center justify-between mb-1">
        <Text className="text-base font-semibold text-ink">{member.name}</Text>
        <View className="bg-herb-tint px-2 py-0.5 rounded-full">
          <Text className="text-xs font-medium text-herb">{member.role}</Text>
        </View>
      </View>
      <Text className="text-sm text-ink-muted mb-0.5">{member.email}</Text>
      <Text className="text-xs text-ink-muted">Joined {joined}</Text>
    </View>
  );
}

function InviteModal({
  visible,
  onClose,
}: {
  visible: boolean;
  onClose: () => void;
}) {
  const inviteMutation = useInviteStaff();
  const {
    control,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<InviteForm>({
    resolver: zodResolver(inviteSchema),
    defaultValues: { email: '', role: '' },
  });

  function handleInvite(data: InviteForm) {
    inviteMutation.mutate(data, {
      onSuccess: () => {
        Alert.alert('Success', `Invitation sent to ${data.email}`);
        reset();
        onClose();
      },
      onError: (err: unknown) => {
        const is403 =
          err !== null &&
          typeof err === 'object' &&
          'response' in err &&
          (err as { response?: { status?: number } }).response?.status === 403;
        Alert.alert(
          'Error',
          is403
            ? 'You do not have permission to invite staff.'
            : 'Failed to send invitation. Please try again.',
        );
      },
    });
  }

  return (
    <Modal visible={visible} animationType="slide" transparent presentationStyle="overFullScreen">
      <View className="flex-1 justify-end bg-ink/50">
        <View className="bg-bone rounded-t-3xl px-6 pt-6 pb-8">
          <View className="flex-row items-center justify-between mb-6">
            <Text className="text-xl font-semibold text-ink">Invite Staff Member</Text>
            <TouchableOpacity onPress={onClose} activeOpacity={0.7}>
              <X size={22} color="#7a7a76" />
            </TouchableOpacity>
          </View>

          <View className="mb-4">
            <Text className="text-sm font-medium text-ink-soft mb-1">Email</Text>
            <Controller
              control={control}
              name="email"
              render={({ field: { onChange, value } }) => (
                <TextInput
                  value={value}
                  onChangeText={onChange}
                  placeholder="colleague@example.com"
                  keyboardType="email-address"
                  autoCapitalize="none"
                  className="border border-mist-strong rounded-xl px-4 py-3 text-base text-ink"
                  placeholderTextColor="#7a7a76"
                />
              )}
            />
            {errors.email && (
              <Text className="text-xs text-paprika mt-1">{errors.email.message}</Text>
            )}
          </View>

          <View className="mb-6">
            <Text className="text-sm font-medium text-ink-soft mb-1">Role</Text>
            <Controller
              control={control}
              name="role"
              render={({ field: { onChange, value } }) => (
                <TextInput
                  value={value}
                  onChangeText={onChange}
                  placeholder="e.g. manager, coordinator"
                  autoCapitalize="none"
                  className="border border-mist-strong rounded-xl px-4 py-3 text-base text-ink"
                  placeholderTextColor="#7a7a76"
                />
              )}
            />
            {errors.role && (
              <Text className="text-xs text-paprika mt-1">{errors.role.message}</Text>
            )}
          </View>

          <TouchableOpacity
            onPress={handleSubmit(handleInvite)}
            disabled={inviteMutation.isPending}
            className={`py-4 rounded-2xl items-center ${
              inviteMutation.isPending ? 'bg-herb-soft' : 'bg-herb'
            }`}
            activeOpacity={0.8}
          >
            {inviteMutation.isPending ? (
              <ActivityIndicator color="white" />
            ) : (
              <Text className="text-paper font-semibold text-base">Send Invitation</Text>
            )}
          </TouchableOpacity>
        </View>
      </View>
    </Modal>
  );
}

export default function StaffScreen() {
  const [showInviteModal, setShowInviteModal] = useState(false);
  const {
    data: staff,
    isLoading,
    isError,
    refetch,
    isRefetching,
  } = useStaffList();

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
        <Text className="text-ink-muted text-base mb-4">Failed to load staff list</Text>
        <TouchableOpacity onPress={() => refetch()} className="bg-herb px-6 py-3 rounded-xl">
          <Text className="text-paper font-semibold">Retry</Text>
        </TouchableOpacity>
      </SafeAreaView>
    );
  }

  // 403 returned null — show non-error lock screen
  if (staff === null) {
    return (
      <SafeAreaView className="flex-1 bg-paper items-center justify-center px-8">
        <Lock size={48} color="#7a7a76" />
        <Text className="text-xl font-semibold text-ink mt-4 mb-2">Staff Management</Text>
        <Text className="text-sm text-ink-muted text-center leading-5">
          Staff management requires manager permissions. Contact your administrator to request
          access.
        </Text>
      </SafeAreaView>
    );
  }

  return (
    <>
      <SafeAreaView className="flex-1 bg-paper">
        <FlatList<StaffMember>
          data={staff}
          keyExtractor={(item) => item.id}
          contentContainerStyle={{ paddingHorizontal: 16, paddingBottom: 32 }}
          refreshControl={
            <RefreshControl refreshing={isRefetching} onRefresh={refetch} tintColor="#C2410C" />
          }
          ListHeaderComponent={
            <View className="flex-row items-center justify-between pt-4 pb-2">
              <Text className="font-display text-2xl font-semibold text-ink">Staff</Text>
              <TouchableOpacity
                onPress={() => setShowInviteModal(true)}
                className="flex-row items-center bg-herb px-3 py-2 rounded-xl"
                activeOpacity={0.8}
              >
                <UserPlus size={16} color="white" />
                <Text className="text-paper font-semibold text-sm ml-1">Invite</Text>
              </TouchableOpacity>
            </View>
          }
          ListEmptyComponent={
            <View className="items-center justify-center py-12">
              <Text className="text-base text-ink-muted">No staff members yet.</Text>
            </View>
          }
          renderItem={({ item }) => <StaffCard member={item} />}
        />
      </SafeAreaView>
      <InviteModal visible={showInviteModal} onClose={() => setShowInviteModal(false)} />
    </>
  );
}
