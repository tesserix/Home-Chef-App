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
import { ChevronLeft, Minus, Plus, Share2, Users } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import {
  useAddGroupItem,
  useCancelGroup,
  useGroupOrder,
  useLeaveGroup,
  useLockGroup,
  usePayGroupShare,
  useRemoveGroupItem,
  type GroupItem,
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
  // Running totals for the summary — item subtotals are known the moment items are
  // added (fees + tax are computed server-side at lock).
  const itemsSubtotal = items.reduce((sum, it) => sum + it.subtotal, 0);
  const totalUnits = items.reduce((sum, it) => sum + it.quantity, 0);
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
          {/* Group context — people + item counts at a glance */}
          <View style={styles.countRow}>
            <View style={styles.countChip}>
              <Users size={13} color={customerColors.charcoal.soft} />
              <Text style={styles.countChipText}>
                {participants.length} {participants.length === 1 ? 'person' : 'people'}
              </Text>
            </View>
            <View style={styles.countChip}>
              <Text style={styles.countChipText}>
                {totalUnits} {totalUnits === 1 ? 'item' : 'items'}
              </Text>
            </View>
          </View>

          {/* Shared cart — everyone's items grouped by participant */}
          <Text style={styles.sectionLabel}>Shared cart</Text>
          {items.length === 0 ? (
            <View style={styles.emptyCart}>
              <Text style={styles.emptyCartTitle}>No items yet</Text>
              <Text style={styles.emptyCartBody}>
                {isHost
                  ? 'Add dishes below, or share the invite so your group can add theirs.'
                  : 'Add your items from the menu below.'}
              </Text>
            </View>
          ) : (
            participants
              .filter((p) => p.id === me.id || itemsByParticipant(p.id).length > 0)
              .map((p) => (
                <ParticipantBlock
                  key={p.id}
                  participant={p}
                  items={itemsByParticipant(p.id)}
                  isMe={p.id === me.id}
                  canEdit={open}
                  onRemove={(itemId) => removeItem.mutate(itemId)}
                  onAdd={(menuItemId, notes) => addItem.mutate({ menuItemId, quantity: 1, notes })}
                  showShare={locked}
                />
              ))
          )}

          {/* Order summary — subtotal now, full fee/tax breakdown once locked */}
          {items.length > 0 ? (
            <OrderSummary g={g} itemsSubtotal={itemsSubtotal} locked={locked} myShare={me.shareAmount} />
          ) : null}

          {/* Add from the chef's menu (open only) */}
          {open ? (
            <>
              <Text style={[styles.sectionLabel, { marginTop: 20 }]}>Add items</Text>
              <View style={styles.menuCard}>
                <FlatList
                  scrollEnabled={false}
                  data={menuItems}
                  keyExtractor={(m) => m.id}
                  ItemSeparatorComponent={() => <View style={styles.menuSep} />}
                  renderItem={({ item }) => {
                    const inCart = items
                      .filter((it) => it.participantId === me.id && it.menuItemId === item.id)
                      .reduce((s, it) => s + it.quantity, 0);
                    return (
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
                        {inCart > 0 ? <Text style={styles.inCartBadge}>{inCart} in cart</Text> : null}
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
                    );
                  }}
                />
              </View>
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
              <Text style={styles.ctaText}>
                {items.length === 0 ? 'Add items to continue' : `Lock & collect · ₹${itemsSubtotal.toFixed(0)}`}
              </Text>
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

// ConsolidatedItem folds the separate per-add GroupItem rows for the same dish
// (same menu item + note) into one line with a total quantity, so a cart with
// four "Test Menu" adds reads as "Test Menu ×4" instead of four delete rows.
interface ConsolidatedItem {
  key: string;
  menuItemId: string;
  name: string;
  quantity: number;
  subtotal: number;
  notes?: string;
  itemIds: string[];
}

function consolidateItems(items: GroupItem[]): ConsolidatedItem[] {
  const byKey = new Map<string, ConsolidatedItem>();
  for (const it of items) {
    const key = `${it.menuItemId}|${it.notes ?? ''}`;
    const existing = byKey.get(key);
    if (existing) {
      existing.quantity += it.quantity;
      existing.subtotal += it.subtotal;
      existing.itemIds.push(it.id);
    } else {
      byKey.set(key, {
        key,
        menuItemId: it.menuItemId,
        name: it.name,
        quantity: it.quantity,
        subtotal: it.subtotal,
        notes: it.notes,
        itemIds: [it.id],
      });
    }
  }
  return [...byKey.values()];
}

function initial(name?: string): string {
  const n = (name ?? '').trim();
  return n ? n[0]!.toUpperCase() : 'G';
}

function ParticipantBlock({
  participant,
  items,
  isMe,
  canEdit,
  onRemove,
  onAdd,
  showShare,
}: {
  participant: GroupParticipant;
  items: GroupItem[];
  isMe: boolean;
  canEdit: boolean;
  onRemove: (itemId: string) => void;
  onAdd: (menuItemId: string, notes?: string) => void;
  showShare: boolean;
}) {
  const lines = consolidateItems(items);
  const subtotal = items.reduce((s, it) => s + it.subtotal, 0);
  const paid = participant.paymentStatus === 'completed';
  return (
    <View style={styles.pBlock}>
      <View style={styles.pHeader}>
        <View style={[styles.pAvatar, isMe && styles.pAvatarMe]}>
          <Text style={[styles.pAvatarText, isMe && styles.pAvatarTextMe]}>{initial(participant.displayName)}</Text>
        </View>
        <View style={{ flex: 1 }}>
          <Text style={styles.pName} numberOfLines={1}>
            {participant.displayName ?? 'Guest'}
            {isMe ? ' (you)' : ''}
            {participant.role === 'host' ? ' · host' : ''}
          </Text>
          {showShare ? (
            <Text style={[styles.pShareLine, paid && styles.pPaid]}>
              {paid ? '✓ Paid' : `Share · ₹${participant.shareAmount.toFixed(0)}`}
            </Text>
          ) : null}
        </View>
        <Text style={styles.pSubtotal}>₹{subtotal.toFixed(0)}</Text>
      </View>
      {lines.length === 0 ? (
        <Text style={styles.pEmpty}>No items yet</Text>
      ) : (
        lines.map((ci) => (
          <View key={ci.key} style={styles.itemRow}>
            <Text style={styles.itemName} numberOfLines={1}>{ci.name}</Text>
            {isMe && canEdit ? (
              <View style={styles.stepper}>
                <Pressable
                  onPress={() => onRemove(ci.itemIds[ci.itemIds.length - 1]!)}
                  hitSlop={6}
                  style={styles.stepperBtn}
                  accessibilityLabel={`Remove one ${ci.name}`}
                >
                  <Minus size={15} color={customerColors.charcoal.DEFAULT} strokeWidth={2.5} />
                </Pressable>
                <Text style={styles.stepperQty}>{ci.quantity}</Text>
                <Pressable
                  onPress={() => onAdd(ci.menuItemId, ci.notes)}
                  hitSlop={6}
                  style={styles.stepperBtn}
                  accessibilityLabel={`Add one ${ci.name}`}
                >
                  <Plus size={15} color={customerColors.charcoal.DEFAULT} strokeWidth={2.5} />
                </Pressable>
              </View>
            ) : (
              <Text style={styles.itemQty}>×{ci.quantity}</Text>
            )}
            <Text style={styles.itemPrice}>₹{ci.subtotal.toFixed(0)}</Text>
          </View>
        ))
      )}
    </View>
  );
}

// OrderSummary shows the running item subtotal while the group is open (fees + tax
// are only computed server-side at lock), and the full compliant breakdown —
// subtotal, delivery, platform fee, GST, total — once locked.
function OrderSummary({
  g,
  itemsSubtotal,
  locked,
  myShare,
}: {
  g: GroupOrder;
  itemsSubtotal: number;
  locked: boolean;
  myShare: number;
}) {
  const money = (n: number) => `₹${n.toFixed(0)}`;
  const gstLabel = g.taxName || 'GST';
  return (
    <View style={styles.summaryCard}>
      {!locked ? (
        <>
          <View style={styles.sumRow}>
            <Text style={styles.sumLabel}>Items subtotal</Text>
            <Text style={styles.sumValue}>{money(itemsSubtotal)}</Text>
          </View>
          <Text style={styles.sumNote}>
            Delivery, platform fee &amp; {gstLabel} are calculated when you lock the order.
          </Text>
        </>
      ) : (
        <>
          <View style={styles.sumRow}>
            <Text style={styles.sumLabel}>Subtotal</Text>
            <Text style={styles.sumValue}>{money(g.subtotal)}</Text>
          </View>
          {g.deliveryFee > 0 ? (
            <View style={styles.sumRow}>
              <Text style={styles.sumLabel}>Delivery</Text>
              <Text style={styles.sumValue}>{money(g.deliveryFee)}</Text>
            </View>
          ) : null}
          {g.serviceFee > 0 ? (
            <View style={styles.sumRow}>
              <Text style={styles.sumLabel}>Platform fee</Text>
              <Text style={styles.sumValue}>{money(g.serviceFee)}</Text>
            </View>
          ) : null}
          {g.tax > 0 ? (
            <View style={styles.sumRow}>
              <Text style={styles.sumLabel}>
                {gstLabel}
                {g.taxRate ? ` (${g.taxRate}%)` : ''}
              </Text>
              <Text style={styles.sumValue}>{money(g.tax)}</Text>
            </View>
          ) : null}
          <View style={styles.sumDivider} />
          <View style={styles.sumRow}>
            <Text style={styles.sumTotalLabel}>Total</Text>
            <Text style={styles.sumTotalValue}>{money(g.total)}</Text>
          </View>
          {myShare > 0 ? (
            <View style={styles.sumShareRow}>
              <Text style={styles.sumShareLabel}>Your share</Text>
              <Text style={styles.sumShareValue}>{money(myShare)}</Text>
            </View>
          ) : null}
        </>
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
  countRow: { flexDirection: 'row', gap: 8, marginBottom: 16 },
  countChip: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 5,
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderRadius: 999,
    backgroundColor: customerColors.surface.soft,
  },
  countChipText: { fontFamily: 'Inter-Medium', fontSize: 12.5, color: customerColors.charcoal.soft },
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    textTransform: 'uppercase',
    letterSpacing: 0.3,
    color: customerColors.charcoal.soft,
    marginBottom: 10,
  },
  emptyCart: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    borderRadius: 12,
    padding: 20,
    gap: 4,
    alignItems: 'center',
  },
  emptyCartTitle: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  emptyCartBody: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, textAlign: 'center', lineHeight: 18 },
  pBlock: {
    backgroundColor: customerColors.canvas,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    borderRadius: 12,
    padding: 12,
    marginBottom: 10,
  },
  pHeader: { flexDirection: 'row', alignItems: 'center', gap: 10, marginBottom: 8 },
  pAvatar: {
    width: 30,
    height: 30,
    borderRadius: 15,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
  },
  pAvatarMe: { backgroundColor: customerColors.coral.tint },
  pAvatarText: { fontFamily: 'Inter-SemiBold', fontSize: 13, color: customerColors.charcoal.soft },
  pAvatarTextMe: { color: customerColors.coral.DEFAULT },
  pName: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  pShareLine: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft, marginTop: 1 },
  pSubtotal: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT, fontVariant: ['tabular-nums'] },
  pPaid: { color: customerColors.success.DEFAULT, fontFamily: 'Inter-SemiBold' },
  pEmpty: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft },
  itemRow: { flexDirection: 'row', alignItems: 'center', gap: 10, paddingVertical: 5 },
  itemName: { flex: 1, fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  itemQty: { fontFamily: 'Inter-Medium', fontSize: 13, color: customerColors.charcoal.soft, fontVariant: ['tabular-nums'] },
  itemPrice: { fontFamily: 'Inter-Medium', fontSize: 14, color: customerColors.charcoal.DEFAULT, minWidth: 52, textAlign: 'right', fontVariant: ['tabular-nums'] },
  stepper: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    borderRadius: 999,
    paddingHorizontal: 4,
    paddingVertical: 2,
  },
  stepperBtn: { width: 24, height: 24, alignItems: 'center', justifyContent: 'center' },
  stepperQty: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.DEFAULT, minWidth: 14, textAlign: 'center', fontVariant: ['tabular-nums'] },
  summaryCard: {
    marginTop: 16,
    backgroundColor: customerColors.surface.soft,
    borderRadius: 12,
    padding: 14,
    gap: 8,
  },
  sumRow: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  sumLabel: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.soft },
  sumValue: { fontFamily: 'Inter-Medium', fontSize: 14, color: customerColors.charcoal.DEFAULT, fontVariant: ['tabular-nums'] },
  sumNote: { fontFamily: 'Inter', fontSize: 12, color: customerColors.charcoal.soft, lineHeight: 16 },
  sumDivider: { height: StyleSheet.hairlineWidth, backgroundColor: customerColors.hairline, marginVertical: 2 },
  sumTotalLabel: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  sumTotalValue: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: customerColors.charcoal.DEFAULT, fontVariant: ['tabular-nums'] },
  sumShareRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginTop: 6,
    paddingTop: 8,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
  },
  sumShareLabel: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.coral.DEFAULT },
  sumShareValue: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.coral.DEFAULT, fontVariant: ['tabular-nums'] },
  menuCard: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    borderRadius: 12,
    paddingHorizontal: 12,
  },
  menuSep: { height: StyleSheet.hairlineWidth, backgroundColor: customerColors.hairline },
  menuRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    paddingVertical: 12,
  },
  inCartBadge: { fontFamily: 'Inter-Medium', fontSize: 12, color: customerColors.coral.DEFAULT },
  menuName: { fontFamily: 'Inter-Medium', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  menuPrice: { fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, marginTop: 2, fontVariant: ['tabular-nums'] },
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
