import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft, Plus, Trash2, Share2 } from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui';
import { openRazorpayCheckout } from '@/shared/utils/razorpay';
import type { MenuItem, Address } from '@/shared/types';

// Group / office orders (#46) — web parity. Shared cart hub: add your own items,
// host locks + collects, each participant pays their split share via Razorpay.

interface GroupParticipant {
  id: string;
  userId: string;
  role: 'host' | 'guest';
  displayName?: string;
  shareAmount: number;
  paymentStatus: 'pending' | 'completed' | 'refunded';
}
interface GroupItem {
  id: string;
  participantId: string;
  menuItemId: string;
  name: string;
  price: number;
  quantity: number;
  subtotal: number;
}
interface GroupOrder {
  id: string;
  hostId: string;
  chefId: string;
  status: 'open' | 'locked' | 'placed' | 'confirmed' | 'delivered' | 'cancelled' | 'expired';
  splitMode: 'split' | 'host';
  type: 'office' | 'personal';
  title?: string;
  currency: string;
  total: number;
  participants: GroupParticipant[];
  items: GroupItem[];
  chef?: { businessName?: string } | null;
  orderId?: string | null;
}
interface GroupDetail {
  groupOrder: GroupOrder;
  me: GroupParticipant;
  joinToken?: string;
}
interface GroupPayResponse {
  razorpayOrderId: string;
  razorpayKeyId: string;
  amount: number;
  currency: string;
}

const STATUS_LABEL: Record<string, string> = {
  open: 'Open — add items',
  locked: 'Awaiting payment',
  placed: 'Placed',
  confirmed: 'Confirmed',
  delivered: 'Delivered',
  cancelled: 'Cancelled',
  expired: 'Expired',
};

export default function GroupOrderPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const qc = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['group-order', id],
    queryFn: () => apiClient.get<GroupDetail>(`/group-orders/${id}`),
    enabled: !!id,
    refetchInterval: 8000,
  });
  const g = data?.groupOrder;
  const me = data?.me;

  const { data: menu } = useQuery({
    queryKey: ['chef', g?.chefId, 'menu'],
    queryFn: () => apiClient.get<{ items: MenuItem[] }>(`/chefs/${g!.chefId}/menu`),
    enabled: !!g?.chefId,
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ['group-order', id] });
  const addItem = useMutation({
    mutationFn: (menuItemId: string) => apiClient.post(`/group-orders/${id}/items`, { menuItemId, quantity: 1 }),
    onSuccess: invalidate,
    onError: () => toast.error('Could not add item'),
  });
  const removeItem = useMutation({
    mutationFn: (itemId: string) => apiClient.delete(`/group-orders/${id}/items/${itemId}`),
    onSuccess: invalidate,
  });
  const lock = useMutation({
    mutationFn: (deliveryAddressId: string) => apiClient.post(`/group-orders/${id}/lock`, { deliveryAddressId }),
    onSuccess: invalidate,
    onError: () => toast.error('Could not lock — check items are in stock'),
  });
  const cancel = useMutation({
    mutationFn: () => apiClient.post(`/group-orders/${id}/cancel`, {}),
    onSuccess: () => { invalidate(); toast.success('Group order cancelled'); },
  });
  const leave = useMutation({
    mutationFn: () => apiClient.post(`/group-orders/${id}/leave`, {}),
    onSuccess: () => { toast.success('You left the group'); navigate('/orders'); },
  });

  function shareInvite() {
    if (!data?.joinToken) return;
    const url = `${window.location.origin}/group/${data.joinToken}`;
    void navigator.clipboard?.writeText(url);
    toast.success('Invite link copied');
  }

  async function onLock() {
    try {
      const addrs = await apiClient.get<Address[]>('/addresses');
      const def = addrs.find((a) => a.isDefault) ?? addrs[0];
      if (!def) {
        toast.error('Add a delivery address in your profile first');
        return;
      }
      lock.mutate(def.id);
    } catch {
      toast.error('Could not load your address');
    }
  }

  async function payShare() {
    try {
      const pd = await apiClient.post<GroupPayResponse>(`/group-orders/${id}/pay`, {});
      openRazorpayCheckout({
        data: pd,
        description: 'Your group order share',
        onVerified: async (resp) => {
          await apiClient.post(`/group-orders/${id}/pay/verify`, {
            razorpayPaymentId: resp.razorpay_payment_id,
            razorpayOrderId: resp.razorpay_order_id,
          });
          invalidate();
          toast.success('Share paid!');
        },
        onDismiss: () => toast.error('Payment cancelled'),
      });
    } catch {
      toast.error('Could not start payment');
    }
  }

  if (isLoading || !g || !me) {
    return <div className="px-4 py-10 text-center text-ink-muted">Loading…</div>;
  }

  const isHost = me.role === 'host';
  const open = g.status === 'open';
  const locked = g.status === 'locked';
  const placed = ['placed', 'confirmed', 'delivered'].includes(g.status);
  const menuItems = (menu?.items ?? []).filter((m) => m.isAvailable);
  const itemsOf = (pid: string) => g.items.filter((it) => it.participantId === pid);

  return (
    <div className="mx-auto max-w-2xl px-4 py-6">
      <Link to="/orders" className="mb-4 inline-flex items-center text-sm text-ink-muted hover:text-ink">
        <ChevronLeft className="h-4 w-4" aria-hidden="true" /> My orders
      </Link>

      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-ink">
            {g.title || (g.type === 'office' ? 'Office order' : 'Group order')}
          </h1>
          <p className="text-sm text-ink-soft">
            {g.chef?.businessName ?? 'Chef'} · {STATUS_LABEL[g.status] ?? g.status}
          </p>
        </div>
        {isHost && open && (
          <Button variant="outline" size="sm" leftIcon={<Share2 className="h-4 w-4" aria-hidden="true" />} onClick={shareInvite}>
            Invite
          </Button>
        )}
      </div>

      {placed ? (
        <div className="mt-8 rounded-xl border border-mist bg-paper p-6 text-center">
          <p className="text-lg font-semibold text-ink">Order placed 🎉</p>
          <p className="mt-1 text-ink-soft">Everyone paid — it's on its way to the chef.</p>
          {g.orderId && (
            <Button asChild variant="primary" className="mt-4">
              <Link to={`/orders/${g.orderId}`}>View order</Link>
            </Button>
          )}
        </div>
      ) : (
        <>
          {/* Shared cart */}
          <h2 className="mt-6 text-sm font-semibold uppercase tracking-wide text-ink-muted">Shared cart</h2>
          <div className="mt-2 space-y-3">
            {g.participants.map((p) => (
              <div key={p.id} className="rounded-xl border border-mist bg-bone p-4">
                <div className="flex items-center justify-between">
                  <span className="font-medium text-ink">
                    {p.displayName ?? 'Guest'}{p.id === me.id ? ' (you)' : ''}{p.role === 'host' ? ' · host' : ''}
                  </span>
                  {locked && (
                    <span className={`text-sm font-semibold tabular-nums ${p.paymentStatus === 'completed' ? 'text-herb' : 'text-ink'}`}>
                      {p.paymentStatus === 'completed' ? '✓ paid' : `₹${p.shareAmount.toFixed(0)}`}
                    </span>
                  )}
                </div>
                {itemsOf(p.id).length === 0 ? (
                  <p className="mt-1 text-sm text-ink-muted">No items yet</p>
                ) : (
                  <ul className="mt-2 space-y-1">
                    {itemsOf(p.id).map((it) => (
                      <li key={it.id} className="flex items-center justify-between text-sm">
                        <span className="text-ink-soft">{it.quantity}× {it.name}</span>
                        <span className="flex items-center gap-2">
                          <span className="tabular-nums text-ink">₹{it.subtotal.toFixed(0)}</span>
                          {p.id === me.id && open && (
                            <button type="button" onClick={() => removeItem.mutate(it.id)} aria-label="Remove">
                              <Trash2 className="h-4 w-4 text-paprika" aria-hidden="true" />
                            </button>
                          )}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            ))}
          </div>

          {/* Add items (open only) */}
          {open && (
            <>
              <h2 className="mt-6 text-sm font-semibold uppercase tracking-wide text-ink-muted">Add your items</h2>
              <div className="mt-2 divide-y divide-mist rounded-xl border border-mist bg-paper">
                {menuItems.map((it) => (
                  <div key={it.id} className="flex items-center justify-between px-4 py-3">
                    <div>
                      <p className="font-medium text-ink">{it.name}</p>
                      <p className="text-sm text-ink-muted tabular-nums">
                        ₹{it.price.toFixed(0)}{it.soldOut ? ' · Sold out' : it.remainingToday != null && it.remainingToday > 0 ? ` · ${it.remainingToday} left` : ''}
                      </p>
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={it.soldOut || addItem.isPending}
                      leftIcon={<Plus className="h-4 w-4" aria-hidden="true" />}
                      onClick={() => addItem.mutate(it.id)}
                    >
                      Add
                    </Button>
                  </div>
                ))}
              </div>
            </>
          )}

          {/* Actions */}
          <div className="mt-6 space-y-3">
            {locked && me.paymentStatus === 'pending' && me.shareAmount > 0 && (
              <Button variant="primary" fullWidth onClick={payShare}>
                Pay your share · ₹{me.shareAmount.toFixed(0)}
              </Button>
            )}
            {locked && (me.paymentStatus === 'completed' || me.shareAmount === 0) && (
              <p className="text-center text-sm text-ink-soft">You're paid — waiting on the rest of the group…</p>
            )}
            {open && isHost && (
              <Button variant="primary" fullWidth disabled={g.items.length === 0 || lock.isPending} onClick={onLock}>
                {g.items.length === 0 ? 'Add items to continue' : 'Lock & collect payment'}
              </Button>
            )}
            {open && !isHost && (
              <Button variant="outline" fullWidth onClick={() => leave.mutate()}>Leave group</Button>
            )}
            {isHost && (open || locked) && (
              <button type="button" onClick={() => cancel.mutate()} className="block w-full text-center text-sm text-paprika">
                Cancel group order
              </button>
            )}
          </div>
        </>
      )}
    </div>
  );
}
