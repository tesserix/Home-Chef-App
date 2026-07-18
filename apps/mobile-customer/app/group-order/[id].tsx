import { useMemo } from 'react';
import {
  ActivityIndicator,
  Alert,
  FlatList,
  Pressable,
  ScrollView,
  Share,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft, Plus, Share2, Trash2 } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import {
  useAddGroupItem,
  useCancelGroup,
  useGroupOrder,
  useLeaveGroup,
  useLockGroup,
  usePayGroupShare,
  useRemoveGroupItem,
  type GroupOrder,
  type GroupParticipant,
} from '../../hooks/useGroupOrder';
import { useChefMenu } from '../../hooks/useChefs';
import { useAddresses } from '../../hooks/useAddresses';
import { useConfirmGroupOrderReceived } from '../../hooks/useConfirmReceived';
import { canConfirmReceipt, payoutHoldMeta } from '../../lib/payout-hold';
import { friendlyErrorMessage } from '../../lib/errors';

const STATUS_LABEL: Record<string, string> = {
  open: 'Open — add items',
  locked: 'Awaiting payment',
  placed: 'Placed',
  confirmed: 'Confirmed',
  delivered: 'Delivered',
  cancelled: 'Cancelled',
  expired: 'Expired',
};

export default function GroupOrderHubScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data, isLoading } = useGroupOrder(id);
  const g = data?.groupOrder;
  const me = data?.me;

  const addItem = useAddGroupItem(id);
  const removeItem = useRemoveGroupItem(id);
  const lockGroup = useLockGroup(id);
  const cancelGroup = useCancelGroup(id);
  const leaveGroup = useLeaveGroup(id);
  const payShare = usePayGroupShare();
  const confirmGroup = useConfirmGroupOrderReceived();
  const { data: addressData } = useAddresses();
  const { data: menuData } = useChefMenu(g?.chefId ?? '');

  const isHost = me?.role === 'host';
  const menuItems = (menuData?.data ?? []).filter((m) => m.isAvailable);

  function share() {
    if (!data?.joinUrl) return;
    void Share.share({
      message: `Join my ${g?.type === 'office' ? 'office' : 'group'} order on Fe3dr: ${data.joinUrl}`,
    });
  }

  function lock() {
    const addr = addressData?.data?.find((a) => a.isDefault) ?? addressData?.data?.[0];
    const addrId = addr?.id;
    if (!addrId) {
      Alert.alert('Add a delivery address', 'Set a delivery address in your profile first.');
      return;
    }
    Alert.alert('Lock & collect payment?', 'No more items can be added after this. Everyone pays their share.', [
      { text: 'Back', style: 'cancel' },
      {
        text: 'Lock',
        onPress: () => lockGroup.mutate({ deliveryAddressId: addrId }),
      },
    ]);
  }

  function pay() {
    if (!id) return;
    payShare.mutate(id, {
      onSuccess: (d) => {
        router.push({
          pathname: '/payment/checkout',
          params: {
            kind: 'group',
            groupId: id,
            orderId: id,
            razorpayOrderId: d.razorpayOrderId,
            razorpayKeyId: d.razorpayKeyId,
            amount: String(d.amount),
            currency: d.currency,
          },
        });
      },
      onError: () => Alert.alert('Could not start payment', 'Please try again.'),
    });
  }

  // #649 — host confirms the group received its delivered order (escrow dual-
  // approval). Inert while the flags are off (the hold never reaches awaiting).
  function confirmReceived() {
    if (!id || !g) return;
    Alert.alert(
      'Confirm your order?',
      `Let us know your group received the order${
        g.chef?.businessName ? ` from ${g.chef.businessName}` : ''
      }. You can still report an issue if something's wrong.`,
      [
        { text: 'Not yet', style: 'cancel' },
        {
          text: 'Confirm received',
          onPress: () =>
            confirmGroup.mutate(id, {
              onSuccess: (res) => Alert.alert('Thanks!', res.message),
              onError: (err) =>
                Alert.alert(
                  'Something went wrong',
                  friendlyErrorMessage(err, 'Could not confirm right now. Please try again.'),
                ),
            }),
        },
      ],
    );
  }

  if (isLoading || !g || !me) {
    return (
      <SafeAreaView style={styles.centered}>
        <ActivityIndicator color={customerColors.coral.DEFAULT} />
      </SafeAreaView>
    );
  }

  // A freshly created group has no items yet, and the API serialises an empty
  // has-many as null — so default to [] to avoid a crash on .filter/.map/.length.
  const items = g.items ?? [];
  const participants = g.participants ?? [];
  const itemsByParticipant = (pid: string) => items.filter((it) => it.participantId === pid);
  const open = g.status === 'open';
  const locked = g.status === 'locked';
  const placed = ['placed', 'confirmed', 'delivered'].includes(g.status);
  // #649 — escrow: host-only confirm CTA when the delivered group awaits
  // confirmation; otherwise a confirmed/disputed pill (visible to all).
  const showConfirm = isHost && canConfirmReceipt(g);
  const holdMeta = payoutHoldMeta(g.payoutHoldStatus);

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable onPress={() => router.back()} hitSlop={8} accessibilityLabel="Go back">
          <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.title} numberOfLines={1}>
          {g.title || (g.type === 'office' ? 'Office order' : 'Group order')}
        </Text>
        {isHost && open ? (
          <Pressable onPress={share} hitSlop={8} accessibilityLabel="Share invite">
            <Share2 size={22} color={customerColors.coral.DEFAULT} />
          </Pressable>
        ) : (
          <View style={{ width: 22 }} />
        )}
      </View>

      <View style={styles.statusBar}>
        <Text style={styles.chefName}>{g.chef?.businessName ?? 'Chef'}</Text>
        <Text style={styles.statusText}>{STATUS_LABEL[g.status] ?? g.status}</Text>
      </View>

      {placed ? (
        <View style={styles.centered}>
          <Text style={styles.bigEmoji}>🎉</Text>
          <Text style={styles.placedTitle}>Order placed!</Text>
          <Text style={styles.placedBody}>Everyone paid — it's on its way to the chef.</Text>
          {g.orderId ? (
            <Pressable onPress={() => router.replace(`/order/${g.orderId}` as never)} style={styles.primaryBtn}>
              <Text style={styles.primaryBtnText}>View order</Text>
            </Pressable>
          ) : null}
          {/* #649 — escrow fulfilment confirmation (host only) */}
          {showConfirm ? (
            <Pressable
              onPress={confirmReceived}
              disabled={confirmGroup.isPending}
              style={[styles.primaryBtn, { marginTop: 12 }]}
              accessibilityRole="button"
              accessibilityLabel="Confirm your group received this order"
            >
              {confirmGroup.isPending ? (
                <ActivityIndicator color={customerColors.canvas} />
              ) : (
                <Text style={styles.primaryBtnText}>Confirm received</Text>
              )}
            </Pressable>
          ) : holdMeta.label ? (
            <View style={[styles.holdPill, { backgroundColor: holdMeta.bg }]}>
              <Text style={[styles.holdPillText, { color: holdMeta.color }]}>{holdMeta.label}</Text>
            </View>
          ) : null}
        </View>
      ) : (
        <ScrollView contentContainerStyle={styles.scroll}>
          {/* Shared cart — everyone's items grouped by participant */}
          <Text style={styles.sectionLabel}>Shared cart</Text>
          {participants.map((p) => (
            <ParticipantBlock
              key={p.id}
              participant={p}
              items={itemsByParticipant(p.id)}
              isMe={p.id === me.id}
              canEdit={open}
              currency={g.currency}
              onRemove={(itemId) => removeItem.mutate(itemId)}
              showShare={locked}
            />
          ))}

          {/* Add from the chef's menu (open only) */}
          {open ? (
            <>
              <Text style={[styles.sectionLabel, { marginTop: 20 }]}>Add your items</Text>
              <FlatList
                scrollEnabled={false}
                data={menuItems}
                keyExtractor={(m) => m.id}
                renderItem={({ item }) => (
                  <View style={styles.menuRow}>
                    <View style={{ flex: 1 }}>
                      <Text style={styles.menuName} numberOfLines={1}>{item.name}</Text>
                      <Text style={styles.menuPrice}>
                        ₹{item.price.toFixed(0)}
                        {item.soldOut
                          ? ' · Sold out'
                          : item.remainingToday != null && item.remainingToday > 0
                            ? ` · ${item.remainingToday} left`
                            : ''}
                      </Text>
                    </View>
                    {item.soldOut ? (
                      <View style={[styles.addBtn, styles.addBtnDisabled]}>
                        <Plus size={18} color="#fff" strokeWidth={2.5} />
                      </View>
                    ) : (
                      <Pressable
                        onPress={() => addItem.mutate({ menuItemId: item.id, quantity: 1 })}
                        style={styles.addBtn}
                        accessibilityLabel={`Add ${item.name}`}
                      >
                        <Plus size={18} color="#fff" strokeWidth={2.5} />
                      </Pressable>
                    )}
                  </View>
                )}
              />
            </>
          ) : null}
        </ScrollView>
      )}

      {/* Footer CTAs */}
      {!placed ? (
        <View style={styles.footer}>
          {locked && me.paymentStatus === 'pending' && me.shareAmount > 0 ? (
            <Pressable onPress={pay} disabled={payShare.isPending} style={styles.cta}>
              {payShare.isPending ? (
                <ActivityIndicator color="#fff" />
              ) : (
                <Text style={styles.ctaText}>Pay your share · ₹{me.shareAmount.toFixed(0)}</Text>
              )}
            </Pressable>
          ) : null}
          {locked && (me.paymentStatus === 'completed' || me.shareAmount === 0) ? (
            <Text style={styles.waiting}>You're paid — waiting on the rest of the group…</Text>
          ) : null}
          {open && isHost ? (
            <Pressable onPress={lock} disabled={lockGroup.isPending || items.length === 0} style={[styles.cta, items.length === 0 && styles.ctaDisabled]}>
              <Text style={styles.ctaText}>{items.length === 0 ? 'Add items to continue' : 'Lock & collect payment'}</Text>
            </Pressable>
          ) : null}
          {open && !isHost ? (
            <Pressable onPress={() => leaveGroup.mutate(undefined)} style={styles.secondaryBtn}>
              <Text style={styles.secondaryBtnText}>Leave group</Text>
            </Pressable>
          ) : null}
          {isHost && (open || locked) ? (
            <Pressable
              onPress={() =>
                Alert.alert('Cancel group order?', 'Everyone who paid is refunded to their wallet.', [
                  { text: 'Back', style: 'cancel' },
                  { text: 'Cancel order', style: 'destructive', onPress: () => cancelGroup.mutate(undefined) },
                ])
              }
              style={styles.cancelLink}
            >
              <Text style={styles.cancelLinkText}>Cancel group order</Text>
            </Pressable>
          ) : null}
        </View>
      ) : null}
    </SafeAreaView>
  );
}

function ParticipantBlock({
  participant,
  items,
  isMe,
  canEdit,
  currency,
  onRemove,
  showShare,
}: {
  participant: GroupParticipant;
  items: GroupOrder['items'];
  isMe: boolean;
  canEdit: boolean;
  currency: string;
  onRemove: (itemId: string) => void;
  showShare: boolean;
}) {
  return (
    <View style={styles.pBlock}>
      <View style={styles.pHeader}>
        <Text style={styles.pName}>
          {participant.displayName ?? 'Guest'}
          {isMe ? ' (you)' : ''}
          {participant.role === 'host' ? ' · host' : ''}
        </Text>
        {showShare ? (
          <Text style={[styles.pShare, participant.paymentStatus === 'completed' && styles.pPaid]}>
            {participant.paymentStatus === 'completed' ? '✓ paid' : `₹${participant.shareAmount.toFixed(0)}`}
          </Text>
        ) : null}
      </View>
      {items.length === 0 ? (
        <Text style={styles.pEmpty}>No items yet</Text>
      ) : (
        items.map((it) => (
          <View key={it.id} style={styles.itemRow}>
            <Text style={styles.itemName} numberOfLines={1}>
              {it.quantity}× {it.name}
            </Text>
            <Text style={styles.itemPrice}>₹{it.subtotal.toFixed(0)}</Text>
            {isMe && canEdit ? (
              <Pressable onPress={() => onRemove(it.id)} hitSlop={8} accessibilityLabel="Remove item">
                <Trash2 size={16} color={customerColors.destructive.DEFAULT} />
              </Pressable>
            ) : null}
          </View>
        ))
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, gap: 8 },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  title: { flex: 1, textAlign: 'center', fontFamily: 'Inter-SemiBold', fontSize: 17, color: customerColors.charcoal.DEFAULT },
  statusBar: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingBottom: 12,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  chefName: { fontFamily: 'Inter-Medium', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  statusText: { fontFamily: 'Inter', fontSize: 13, color: customerColors.coral.DEFAULT },
  scroll: { padding: 16, paddingBottom: 24 },
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    textTransform: 'uppercase',
    letterSpacing: 0.3,
    color: customerColors.charcoal.soft,
    marginBottom: 10,
  },
  pBlock: {
    backgroundColor: customerColors.surface.soft,
    borderRadius: 12,
    padding: 12,
    marginBottom: 10,
  },
  pHeader: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 },
  pName: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  pShare: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  pPaid: { color: customerColors.success.DEFAULT },
  pEmpty: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft },
  itemRow: { flexDirection: 'row', alignItems: 'center', gap: 10, paddingVertical: 4 },
  itemName: { flex: 1, fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  itemPrice: { fontFamily: 'Inter-Medium', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  menuRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 10,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  menuName: { fontFamily: 'Inter-Medium', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  menuPrice: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, marginTop: 2 },
  addBtn: {
    width: 32,
    height: 32,
    borderRadius: 16,
    backgroundColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  addBtnDisabled: { backgroundColor: customerColors.charcoal.soft, opacity: 0.5 },
  footer: {
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 16,
    backgroundColor: customerColors.canvas,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
    gap: 10,
  },
  cta: {
    height: 52,
    borderRadius: 12,
    backgroundColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  ctaDisabled: { backgroundColor: customerColors.charcoal.soft, opacity: 0.5 },
  ctaText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: '#fff' },
  waiting: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft, textAlign: 'center' },
  secondaryBtn: {
    height: 48,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    alignItems: 'center',
    justifyContent: 'center',
  },
  secondaryBtnText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  cancelLink: { alignItems: 'center', paddingVertical: 4 },
  cancelLinkText: { fontFamily: 'Inter', fontSize: 13, color: customerColors.destructive.DEFAULT },
  primaryBtn: {
    height: 52,
    borderRadius: 12,
    backgroundColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 32,
    marginTop: 12,
  },
  primaryBtnText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: '#fff' },
  // #649 — confirmed / disputed escrow pill shown once the CTA is no longer actionable.
  holdPill: {
    marginTop: 12,
    borderRadius: 8,
    minHeight: 44,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 16,
  },
  holdPillText: { fontFamily: 'Inter-SemiBold', fontSize: 15 },
  bigEmoji: { fontSize: 48 },
  placedTitle: { fontFamily: 'Inter-SemiBold', fontSize: 20, color: customerColors.charcoal.DEFAULT },
  placedBody: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft, textAlign: 'center' },
});
