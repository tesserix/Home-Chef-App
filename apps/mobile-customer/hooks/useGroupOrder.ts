import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Group / office orders (#46): shared cart + split pay + single delivery.

export type GroupType = 'office' | 'personal';
export type GroupSplitMode = 'split' | 'host';
export type GroupStatus =
  | 'open'
  | 'locked'
  | 'placed'
  | 'confirmed'
  | 'delivered'
  | 'cancelled'
  | 'expired';

export interface GroupParticipant {
  id: string;
  userId: string;
  role: 'host' | 'guest';
  displayName?: string;
  shareAmount: number;
  paymentStatus: 'pending' | 'completed' | 'refunded';
}

export interface GroupItem {
  id: string;
  participantId: string;
  menuItemId: string;
  name: string;
  price: number;
  quantity: number;
  subtotal: number;
  notes?: string;
}

export interface GroupOrder {
  id: string;
  hostId: string;
  chefId: string;
  type: GroupType;
  splitMode: GroupSplitMode;
  title?: string;
  companyName?: string;
  status: GroupStatus;
  orderId?: string | null;
  currency: string;
  subtotal: number;
  deliveryFee: number;
  serviceFee: number;
  tax: number;
  total: number;
  participants: GroupParticipant[];
  items: GroupItem[];
  chef?: { businessName?: string } | null;
}

export interface GroupOrderDetail {
  groupOrder: GroupOrder;
  me: GroupParticipant;
  joinToken?: string;
  joinUrl?: string;
}

export interface GroupInvitePreview {
  title?: string;
  type: GroupType;
  chefName?: string;
  chefId: string;
  status: GroupStatus;
  joinable: boolean;
  hostName?: string;
}

export function useCreateGroupOrder() {
  return useMutation({
    mutationFn: (body: {
      chefId: string;
      type: GroupType;
      splitMode: GroupSplitMode;
      title?: string;
      companyName?: string;
    }) =>
      api
        .post<{ groupOrder: GroupOrder; joinToken: string; joinUrl: string }>(
          '/v1/group-orders',
          body,
        )
        .then((r) => r.data),
  });
}

export function useGroupOrder(id: string | undefined) {
  return useQuery<GroupOrderDetail>({
    queryKey: ['group-order', id],
    queryFn: () =>
      api.get<GroupOrderDetail>(`/v1/group-orders/${id}`).then((r) => r.data),
    enabled: Boolean(id),
    refetchInterval: 8000, // keep the shared cart fresh as others add items / pay
  });
}

export function useGroupInvitePreview(token: string | undefined) {
  return useQuery<GroupInvitePreview>({
    queryKey: ['group-invite', token],
    queryFn: () =>
      api.get<GroupInvitePreview>(`/v1/group-invites/${token}`).then((r) => r.data),
    enabled: Boolean(token),
  });
}

export function useJoinGroup() {
  return useMutation({
    mutationFn: (token: string) =>
      api
        .post<{ groupOrderId?: string; joined?: boolean; groupOrder?: GroupOrder; alreadyJoined?: boolean }>(
          `/v1/group-invites/${token}/accept`,
          {},
        )
        .then((r) => r.data),
  });
}

function useGroupMutation<TVars>(
  fn: (vars: TVars) => Promise<unknown>,
  groupId: string | undefined,
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['group-order', groupId] }),
  });
}

export function useAddGroupItem(groupId: string | undefined) {
  return useGroupMutation(
    (vars: { menuItemId: string; quantity: number; notes?: string }) =>
      api.post(`/v1/group-orders/${groupId}/items`, vars).then((r) => r.data),
    groupId,
  );
}

export function useRemoveGroupItem(groupId: string | undefined) {
  return useGroupMutation(
    (itemId: string) =>
      api.delete(`/v1/group-orders/${groupId}/items/${itemId}`).then((r) => r.data),
    groupId,
  );
}

export function useLockGroup(groupId: string | undefined) {
  return useGroupMutation(
    (vars: { deliveryAddressId: string; deliveryInstructions?: string }) =>
      api.post(`/v1/group-orders/${groupId}/lock`, vars).then((r) => r.data),
    groupId,
  );
}

export function useCancelGroup(groupId: string | undefined) {
  return useGroupMutation(
    () => api.post(`/v1/group-orders/${groupId}/cancel`, {}).then((r) => r.data),
    groupId,
  );
}

export function useLeaveGroup(groupId: string | undefined) {
  return useGroupMutation(
    () => api.post(`/v1/group-orders/${groupId}/leave`, {}).then((r) => r.data),
    groupId,
  );
}

export interface GroupPayResponse {
  razorpayOrderId: string;
  razorpayKeyId: string;
  amount: number;
  currency: string;
}

export function usePayGroupShare() {
  return useMutation({
    mutationFn: (groupId: string) =>
      api
        .post<GroupPayResponse>(`/v1/group-orders/${groupId}/pay`, {})
        .then((r) => r.data),
  });
}
