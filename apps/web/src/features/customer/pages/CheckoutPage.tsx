import { useEffect, useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  MapPin,
  Clock,
  ChevronRight,
  Check,
  FileText,
  Shield,
  AlertTriangle,
} from 'lucide-react';
import { toast } from 'sonner';
import { useCartStore } from '@/app/store/cart-store';
import { useAuth } from '@/app/providers/AuthProvider';
import { apiClient } from '@/shared/services/api-client';
import { useFormatPrice } from '@/shared/utils/format-price';
import { loadStripeJs } from '@/shared/utils/load-stripe';
import { resolveCssVarColor } from '@/shared/utils/css-color';
import { Button } from '@/shared/components/ui';
import type { Order, Address } from '@/shared/types';

const addressSchema = z.object({
  label: z.string().min(1, 'Label is required'),
  line1: z.string().min(5, 'Address is required'),
  line2: z.string().optional(),
  city: z.string().min(2, 'City is required'),
  state: z.string().min(2, 'State is required'),
  postalCode: z.string().min(5, 'Postal code is required'),
  deliveryInstructions: z.string().optional(),
});

type AddressFormData = z.infer<typeof addressSchema>;

// Scheduled delivery slots (#51) — mirrors the API GET /chefs/:id/delivery-slots
// response (services.SlotAvailability).
interface DeliverySlot {
  date: string; // "YYYY-MM-DD" IST
  slot: 'lunch' | 'dinner';
  label: string;
  window: string; // "12:00–14:00"
  remaining: number | null; // null = unlimited
  available: boolean;
}
interface DeliverySlotsResponse {
  slotsEnabled: boolean;
  slots: DeliverySlot[];
}

// Dietary conflict check (#41) — mirrors POST /dietary/check.
interface DietaryWarning {
  menuItemId: string;
  name: string;
  conflicts: { type: string; label: string; detail: string }[];
}
interface DietaryCheckResult {
  hasConflicts: boolean;
  warnings: DietaryWarning[];
}

// slotDayLabel turns a "YYYY-MM-DD" slot date into a label relative to today
// ("Today" / "Tomorrow" / "Mon, 22 Jun").
function slotDayLabel(dateStr: string): string {
  const d = new Date(`${dateStr}T00:00:00`);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const diff = Math.round((d.getTime() - today.getTime()) / 86_400_000);
  if (diff <= 0) return 'Today';
  if (diff === 1) return 'Tomorrow';
  return d.toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' });
}

// Saved addresses come from /api/v1/addresses (see useQuery below). The old
// hardcoded list shipped mock ids "1" and "2" which the backend rejected at
// order creation with "invalid UUID length" because deliveryAddressId is a
// UUID column. Fetching the real records means the selected id is a valid
// UUID that the API can actually look up.

export default function CheckoutPage() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const cart = useCartStore();
  const fp = useFormatPrice();
  const queryClient = useQueryClient();
  const { data: savedAddresses = [] } = useQuery({
    queryKey: ['addresses'],
    queryFn: () => apiClient.get<Address[]>('/addresses'),
  });

  const [selectedAddress, setSelectedAddress] = useState<string>('');
  // Default to the user's default address (or the first one) as soon as
  // the list loads. Resets if the previously-selected id disappears.
  useEffect(() => {
    if (!savedAddresses.length) {
      if (selectedAddress) setSelectedAddress('');
      return;
    }
    const exists = savedAddresses.some((a) => a.id === selectedAddress);
    if (!exists) {
      const preferred = savedAddresses.find((a) => a.isDefault) ?? savedAddresses[0];
      if (preferred) setSelectedAddress(preferred.id);
    }
  }, [savedAddresses, selectedAddress]);
  const [showNewAddress, setShowNewAddress] = useState(false);
  // Scheduled delivery slot (#51) — null = ASAP. The picker below only offers
  // slots when the chef has enabled them.
  const [selectedSlot, setSelectedSlot] = useState<{ slot: string; date: string } | null>(null);
  const { data: slotsData } = useQuery({
    queryKey: ['delivery-slots', cart.chefId],
    queryFn: () =>
      apiClient.get<DeliverySlotsResponse>(`/chefs/${cart.chefId}/delivery-slots`),
    enabled: Boolean(cart.chefId),
    staleTime: 60_000,
  });
  const availableSlots = (slotsData?.slots ?? []).filter((s) => s.available);
  // Dietary & allergen conflict warning (#41) — server-checks the cart's items
  // against the customer's saved profile. Non-blocking.
  const cartItemIds = cart.items.map((i) => i.menuItemId);
  const { data: dietaryCheck } = useQuery({
    queryKey: ['dietary-check', cartItemIds],
    queryFn: () =>
      apiClient.post<DietaryCheckResult>('/dietary/check', { menuItemIds: cartItemIds }),
    enabled: cartItemIds.length > 0,
    staleTime: 60_000,
  });
  const dietaryWarnings = dietaryCheck?.warnings ?? [];

  const [tip, setTip] = useState<number>(0);
  const [specialInstructions, setSpecialInstructions] = useState('');
  const [isProcessing, setIsProcessing] = useState<boolean>(false);
  // CW-01d: explicit T&C + Refund Policy consent is required per order
  // (not just at signup) for RBI PA disclosure compliance. The Place Order
  // CTA stays disabled until this is checked.
  const [acceptedTerms, setAcceptedTerms] = useState<boolean>(false);

  // The currency of the amounts on this page is the chef's settlement
  // currency — that's what the backend will charge the customer in. Falls
  // back to INR so pre-multi-gateway chef profiles keep rendering.
  const orderCurrency = (cart.chef as { currency?: string } | null)?.currency || 'INR';

  // Tax rate preview comes from the public /tax-rates/lookup endpoint and
  // is driven by the delivery address the customer selects. The real tax
  // applied at order creation is recomputed server-side — this is just to
  // keep the displayed total honest between address pick and checkout.
  const selectedAddressObj = savedAddresses.find((a) => a.id === selectedAddress);
  const taxCountry = (selectedAddressObj as { country?: string } | undefined)?.country || 'IN';
  const taxRegion = (selectedAddressObj as { state?: string } | undefined)?.state || '';
  const { data: taxRule } = useQuery({
    queryKey: ['tax-rate', taxCountry, taxRegion],
    queryFn: () =>
      apiClient.get<{ rate: number; taxName: string; inclusive: boolean }>(
        `/tax-rates/lookup?country=${encodeURIComponent(taxCountry)}&region=${encodeURIComponent(taxRegion)}`
      ),
    enabled: Boolean(taxCountry),
  });

  const subtotal = cart.getSubtotal();
  const deliveryFee = cart.chef?.deliveryFee || 0;
  const serviceFee = subtotal * 0.05;
  // Applied promo discount (#39), clamped to the subtotal. Mirrors the server,
  // which taxes the post-discount base; server is authoritative at order time.
  const discount = cart.promoCode ? Math.min(cart.promoDiscount, subtotal) : 0;
  const rate = taxRule?.rate ?? 0;
  const isInclusive = taxRule?.inclusive ?? false;
  const taxBase = subtotal + deliveryFee + serviceFee - discount;
  const tax = isInclusive
    ? taxBase - taxBase / (1 + rate / 100)
    : taxBase * (rate / 100);
  const total = isInclusive
    ? subtotal + deliveryFee + serviceFee - discount + tip
    : subtotal + deliveryFee + serviceFee - discount + tax + tip;

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<AddressFormData>({
    resolver: zodResolver(addressSchema),
  });

  const handlePlaceOrder = async () => {
    if (!cart.chefId || !selectedAddress) {
      toast.error('Please select a delivery address');
      return;
    }
    if (!acceptedTerms) {
      toast.error('Please accept the Terms of Service and Refund Policy to continue');
      return;
    }

    setIsProcessing(true);

    try {
      // Step 1: Create order. The backend decides which gateway to use
      // based on the chef's PaymentProvider setting.
      const order = await apiClient.post<Order>('/orders', {
        // Map cart lines to the API item shape, including selected add-ons (#232).
        items: cart.items.map((i) => ({
          menuItemId: i.menuItemId,
          quantity: i.quantity,
          notes: i.notes || undefined,
          modifierOptionIds: i.modifiers?.map((m) => m.optionId),
        })),
        chefId: cart.chefId,
        deliveryAddressId: selectedAddress,
        tip,
        specialInstructions: specialInstructions || undefined,
        deliverySlot: selectedSlot?.slot,
        deliveryDate: selectedSlot?.date,
        // Applied promo (#39) — server re-validates + recomputes the discount.
        promoCode: cart.promoCode || undefined,
      });

      // Step 2: Ask the backend to prepare a payment. The response shape
      // varies by provider — `provider: "razorpay"` returns a Razorpay
      // order id + key id; `provider: "stripe"` returns a PaymentIntent
      // clientSecret + publishable key.
      type RazorpayPayment = {
        provider: 'razorpay';
        razorpayOrderId: string;
        razorpayKeyId: string;
        amount: number;
        currency: string;
      };
      type StripePayment = {
        provider: 'stripe';
        stripePaymentIntentId: string;
        clientSecret: string;
        publishableKey: string;
        amount: number;
        currency: string;
      };
      const paymentData = await apiClient.post<RazorpayPayment | StripePayment>(
        `/payments/order/${order.id}/create`,
        {}
      );

      if (paymentData.provider === 'stripe') {
        await confirmStripePayment(order.id, paymentData);
      } else {
        await confirmRazorpayPayment(order, paymentData);
      }
    } catch (e: unknown) {
      // Surface a promo failure specifically (e.g. the code was exhausted between
      // applying it and checkout) and drop it so the retry isn't blocked (#39).
      const raw = (e as { error?: unknown })?.error;
      const msg =
        typeof raw === 'string'
          ? raw
          : raw && typeof raw === 'object' && 'message' in raw
            ? String((raw as { message?: unknown }).message ?? '')
            : e instanceof Error
              ? e.message
              : '';
      if (/promo/i.test(msg)) {
        cart.clearPromo();
        toast.error(msg || 'That promo code is no longer available. Please try again.');
      } else {
        toast.error('Failed to initiate payment. Please try again.');
      }
    } finally {
      setIsProcessing(false);
    }
  };

  const confirmRazorpayPayment = async (
    order: Order,
    paymentData: {
      razorpayOrderId: string;
      razorpayKeyId: string;
      amount: number;
      currency: string;
    }
  ) => {
    if (!window.Razorpay) {
      toast.error('Payment gateway is loading. Please try again.');
      return;
    }
    const options: RazorpayOptions = {
      key: paymentData.razorpayKeyId,
      amount: paymentData.amount,
      currency: paymentData.currency,
      name: 'Fe3dr',
      description: `Order from ${cart.chef?.businessName || 'Home Chef'}`,
      order_id: paymentData.razorpayOrderId,
      prefill: {
        name: user?.name || '',
        email: user?.email || '',
      },
      // Resolve --herb at runtime so a theme change ripples to Razorpay's
      // hosted checkout. Falls back to a static herb-equivalent hex if the
      // CSS var is unavailable (SSR / older browsers without oklch).
      theme: { color: resolveCssVarColor('--herb', '#3e6b3c') },
      handler: async (response) => {
        try {
          await apiClient.post(`/payments/order/${order.id}/verify`, {
            razorpayPaymentId: response.razorpay_payment_id,
            razorpayOrderId: response.razorpay_order_id,
            razorpaySignature: response.razorpay_signature,
          });
          cart.clearCart();
          toast.success('Payment successful!');
          navigate(`/orders/${order.id}`);
        } catch {
          toast.error('Payment verification failed. Please contact support.');
        }
      },
      modal: {
        ondismiss: () => toast.error('Payment cancelled'),
      },
    };
    new window.Razorpay(options).open();
  };

  // Launch the Stripe hosted-redirect flow. We use the Checkout redirect
  // (simpler + no extra React integration needed) by loading Stripe.js on
  // demand and calling stripe.confirmPayment with the client secret.
  const confirmStripePayment = async (
    orderId: string,
    paymentData: {
      stripePaymentIntentId: string;
      clientSecret: string;
      publishableKey: string;
      amount: number;
      currency: string;
    }
  ) => {
    const stripe = await loadStripeJs(paymentData.publishableKey);
    if (!stripe) {
      toast.error('Stripe failed to load');
      return;
    }
    // Customer confirms via Stripe-hosted form. `return_url` is where
    // Stripe sends them after 3DS / wallet confirmation — we land back
    // on the order page and VerifyPayment is triggered there too.
    const returnUrl = `${window.location.origin}/orders/${orderId}?stripe_pi=${paymentData.stripePaymentIntentId}`;
    const { error } = await stripe.confirmPayment({
      clientSecret: paymentData.clientSecret,
      confirmParams: { return_url: returnUrl },
    });
    if (error) {
      toast.error(error.message || 'Payment failed');
      return;
    }
    // If confirmPayment doesn't redirect (rare — happens for sync
    // confirmations like some non-3DS card paths), verify inline.
    try {
      await apiClient.post(`/payments/order/${orderId}/verify`, {
        stripePaymentIntentId: paymentData.stripePaymentIntentId,
      });
      cart.clearCart();
      toast.success('Payment successful!');
      navigate(`/orders/${orderId}`);
    } catch {
      toast.error('Payment verification failed. Please contact support.');
    }
  };

  const onAddressSubmit = async (data: AddressFormData) => {
    try {
      const created = await apiClient.post<Address>('/addresses', {
        ...data,
        country: 'IN',
        isDefault: false,
      });
      // Refresh the list + select the brand-new one.
      await queryClient.invalidateQueries({ queryKey: ['addresses'] });
      setSelectedAddress(created.id);
      setShowNewAddress(false);
      reset();
      toast.success('Address saved');
    } catch {
      toast.error('Failed to save address');
    }
  };

  if (cart.items.length === 0) {
    navigate('/cart');
    return null;
  }

  return (
    <div className="min-h-screen bg-paper py-8">
      <div className="container-app max-w-4xl">
        <h1 className="font-display text-2xl font-semibold text-ink md:text-3xl">Checkout</h1>

        <div className="mt-8 flex flex-col gap-8 lg:flex-row">
          {/* Main Form */}
          <div className="flex-1 space-y-6">
            {/* Delivery Address */}
            <section className="rounded-xl bg-bone p-6 shadow-1">
              <div className="flex items-center justify-between">
                <h2 className="flex items-center gap-2 text-lg font-semibold text-ink">
                  <MapPin className="h-5 w-5 text-herb"  aria-hidden="true" />
                  Delivery Address
                </h2>
                <button type="button"
                  onClick={() => setShowNewAddress(!showNewAddress)}
                  className="text-sm text-herb hover:text-herb"
                >
                  {showNewAddress ? 'Cancel' : 'Add New'}
                </button>
              </div>

              {showNewAddress ? (
                <form onSubmit={handleSubmit(onAddressSubmit)} className="mt-4 space-y-4">
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div>
                      <label htmlFor="addr-label" className="block text-sm font-medium text-ink-soft">
                        Label
                      </label>
                      <input
                        id="addr-label"
                        {...register('label')}
                        aria-invalid={!!errors.label || undefined}
                        aria-describedby={errors.label ? 'addr-label-err' : undefined}
                        placeholder="Home, Work, etc."
                        className="input-base mt-1"
                      />
                      {errors.label && (
                        <p id="addr-label-err" role="alert" className="mt-1 text-xs text-paprika">
                          {errors.label.message}
                        </p>
                      )}
                    </div>
                    <div className="sm:col-span-2">
                      <label htmlFor="addr-line1" className="block text-sm font-medium text-ink-soft">
                        Street Address
                      </label>
                      <input
                        id="addr-line1"
                        {...register('line1')}
                        aria-invalid={!!errors.line1 || undefined}
                        aria-describedby={errors.line1 ? 'addr-line1-err' : undefined}
                        placeholder="123 Main Street"
                        className="input-base mt-1"
                      />
                      {errors.line1 && (
                        <p id="addr-line1-err" role="alert" className="mt-1 text-xs text-paprika">
                          {errors.line1.message}
                        </p>
                      )}
                    </div>
                    <div className="sm:col-span-2">
                      <label htmlFor="addr-line2" className="block text-sm font-medium text-ink-soft">
                        Apartment, suite, etc. (optional)
                      </label>
                      <input
                        id="addr-line2"
                        {...register('line2')}
                        placeholder="Apt 4B"
                        className="input-base mt-1"
                      />
                    </div>
                    <div>
                      <label htmlFor="addr-city" className="block text-sm font-medium text-ink-soft">City</label>
                      <input
                        id="addr-city"
                        {...register('city')}
                        aria-invalid={!!errors.city || undefined}
                        aria-describedby={errors.city ? 'addr-city-err' : undefined}
                        className="input-base mt-1"
                      />
                      {errors.city && (
                        <p id="addr-city-err" role="alert" className="mt-1 text-xs text-paprika">
                          {errors.city.message}
                        </p>
                      )}
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label htmlFor="addr-state" className="block text-sm font-medium text-ink-soft">State</label>
                        <input
                          id="addr-state"
                          {...register('state')}
                          aria-invalid={!!errors.state || undefined}
                          aria-describedby={errors.state ? 'addr-state-err' : undefined}
                          className="input-base mt-1"
                        />
                        {errors.state && (
                          <p id="addr-state-err" role="alert" className="mt-1 text-xs text-paprika">
                            {errors.state.message}
                          </p>
                        )}
                      </div>
                      <div>
                        <label htmlFor="addr-postal" className="block text-sm font-medium text-ink-soft">
                          Postal Code
                        </label>
                        <input
                          id="addr-postal"
                          {...register('postalCode')}
                          aria-invalid={!!errors.postalCode || undefined}
                          aria-describedby={errors.postalCode ? 'addr-postal-err' : undefined}
                          className="input-base mt-1"
                        />
                        {errors.postalCode && (
                          <p id="addr-postal-err" role="alert" className="mt-1 text-xs text-paprika">
                            {errors.postalCode.message}
                          </p>
                        )}
                      </div>
                    </div>
                  </div>
                  <Button type="submit" variant="primary">
                    Save Address
                  </Button>
                </form>
              ) : savedAddresses.length === 0 ? (
                <div className="mt-4 rounded-lg border border-dashed border-mist-strong p-6 text-center text-sm text-ink-soft">
                  You don't have any saved addresses yet. Add one to continue.
                </div>
              ) : (
                <div className="mt-4 space-y-3">
                  {savedAddresses.map((address) => (
                    <label
                      key={address.id}
                      className={`flex cursor-pointer items-start gap-3 rounded-lg border p-4 transition-colors ${
                        selectedAddress === address.id
                          ? 'border-herb bg-herb-tint'
                          : 'border-mist hover:bg-paper'
                      }`}
                    >
                      <input
                        type="radio"
                        name="address"
                        value={address.id}
                        checked={selectedAddress === address.id}
                        onChange={(e) => setSelectedAddress(e.target.value)}
                        className="mt-1 h-4 w-4 text-herb focus-visible:ring-herb"
                      />
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-ink">{address.label}</span>
                          {address.isDefault && (
                            <span className="rounded bg-mist px-2 py-0.5 text-xs text-ink-soft">
                              Default
                            </span>
                          )}
                        </div>
                        <p className="mt-1 text-sm text-ink-soft">
                          {address.line1}
                          {address.line2 && `, ${address.line2}`}
                        </p>
                        <p className="text-sm text-ink-soft">
                          {address.city}, {address.state} {address.postalCode}
                        </p>
                      </div>
                      {selectedAddress === address.id && (
                        <Check className="h-5 w-5 text-herb"  aria-hidden="true" />
                      )}
                    </label>
                  ))}
                </div>
              )}
            </section>

            {/* Dietary / allergen conflict warning (#41) — non-blocking */}
            {dietaryWarnings.length > 0 && (
              <section className="rounded-xl border border-paprika/30 bg-paprika-tint p-6 shadow-1">
                <h2 className="flex items-center gap-2 text-lg font-semibold text-paprika">
                  <AlertTriangle className="h-5 w-5" aria-hidden="true" />
                  Check your order
                </h2>
                <p className="mt-1 text-sm text-paprika">
                  Some items may not match your dietary profile:
                </p>
                <ul className="mt-2 space-y-1">
                  {dietaryWarnings.map((w) => (
                    <li key={w.menuItemId} className="text-sm text-paprika">
                      • {w.name} — {w.conflicts.map((cf) => cf.detail).join(', ')}
                    </li>
                  ))}
                </ul>
                <p className="mt-2 text-xs text-ink-muted">
                  You can still place this order. Review your items or update your dietary profile in
                  your account.
                </p>
              </section>
            )}

            {/* Delivery Time */}
            <section className="rounded-xl bg-bone p-6 shadow-1">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-ink">
                <Clock className="h-5 w-5 text-herb"  aria-hidden="true" />
                Delivery Time
              </h2>

              <div className="mt-4 space-y-3">
                {/* ASAP (default) */}
                <label
                  className={`flex cursor-pointer items-center gap-3 rounded-lg border p-4 ${
                    selectedSlot === null
                      ? 'border-herb bg-herb-tint'
                      : 'border-mist hover:bg-paper'
                  }`}
                >
                  <input
                    type="radio"
                    name="time"
                    checked={selectedSlot === null}
                    onChange={() => setSelectedSlot(null)}
                    className="h-4 w-4 text-herb focus-visible:ring-herb"
                  />
                  <div>
                    <span className="font-medium text-ink">As soon as possible</span>
                    <p className="text-sm text-ink-muted">
                      Estimated 30-45 minutes after the chef accepts your order.
                      Actual time depends on chef preparation and driver route.
                    </p>
                  </div>
                </label>

                {/* Scheduled slots (#51) — only when the chef offers them */}
                {slotsData?.slotsEnabled &&
                  availableSlots.map((s) => {
                    const sel =
                      selectedSlot?.slot === s.slot && selectedSlot?.date === s.date;
                    return (
                      <label
                        key={`${s.date}-${s.slot}`}
                        className={`flex cursor-pointer items-center gap-3 rounded-lg border p-4 ${
                          sel ? 'border-herb bg-herb-tint' : 'border-mist hover:bg-paper'
                        }`}
                      >
                        <input
                          type="radio"
                          name="time"
                          checked={sel}
                          onChange={() => setSelectedSlot({ slot: s.slot, date: s.date })}
                          className="h-4 w-4 text-herb focus-visible:ring-herb"
                        />
                        <div className="flex-1">
                          <span className="font-medium text-ink">
                            {slotDayLabel(s.date)} · {s.label}
                          </span>
                          <p className="text-sm text-ink-muted tabular-nums">
                            {s.window}
                            {s.remaining != null ? ` · ${s.remaining} left` : ''}
                          </p>
                        </div>
                      </label>
                    );
                  })}

                {slotsData?.slotsEnabled && availableSlots.length === 0 && (
                  <p className="text-sm text-ink-muted">
                    No delivery windows are open right now — your order will be delivered ASAP.
                  </p>
                )}
              </div>
            </section>

            {/* Payment */}
            <section className="rounded-xl bg-bone p-6 shadow-1">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-ink">
                <Shield className="h-5 w-5 text-herb"  aria-hidden="true" />
                Payment
              </h2>
              <div className="mt-4 flex items-center gap-3 rounded-lg border border-mist bg-paper p-4">
                <img
                  src="https://razorpay.com/assets/razorpay-glyph.svg"
                  alt="Razorpay"
                  width={24}
                  height={24}
                  loading="lazy"
                  decoding="async"
                  className="h-6 w-6 shrink-0"
                />
                <div>
                  <p className="text-sm font-medium text-ink">Powered by Razorpay</p>
                  <p className="text-xs text-ink-muted">
                    Pay securely via UPI, cards, net banking, or wallets
                  </p>
                </div>
              </div>

              {/* CW-01d: RBI Payment Aggregator disclosure block. Per RBI PA
                  Master Direction §8, refund timeline and merchant-of-record
                  must be disclosed at the point of payment. */}
              <div className="mt-4 space-y-3 rounded-md border border-mist bg-paper p-4 text-sm text-ink-soft">
                <div>
                  <div className="mb-1 font-medium text-ink">Payment &amp; refund summary</div>
                  <p>
                    Payments are processed by Razorpay (RBI-licensed payment aggregator).
                    Tesserix Pty Ltd (operator of Fe3dr) facilitates the transaction;
                    order proceeds go to your chef minus the platform commission.
                  </p>
                </div>
                <div className="space-y-2">
                  <div className="flex items-start gap-2">
                    <Clock className="mt-0.5 h-4 w-4 flex-shrink-0 text-herb" aria-hidden="true" />
                    <p>
                      Refunds return to your original payment method within{' '}
                      <strong>7 working days</strong> per RBI Payment Aggregator Master Direction §8.
                    </p>
                  </div>
                  <div className="flex items-start gap-2">
                    <FileText className="mt-0.5 h-4 w-4 flex-shrink-0 text-herb" aria-hidden="true" />
                    <p>
                      See our{' '}
                      <Link to="/refund" className="text-herb hover:underline">
                        Refund Policy
                      </Link>{' '}
                      for cancellation rules by order stage.
                    </p>
                  </div>
                </div>
              </div>
            </section>

            {/* Tip */}
            <section className="rounded-xl bg-bone p-6 shadow-1">
              <h2 className="text-lg font-semibold text-ink">Add a tip</h2>
              <p className="mt-1 text-sm text-ink-muted">
                100% of your tip goes to the home chef
              </p>

              <div className="mt-4 flex flex-wrap gap-2">
                {[0, 2, 5, 10].map((amount) => (
                  <button type="button"
                    key={amount}
                    onClick={() => setTip(amount)}
                    className={`rounded-lg px-4 py-2 transition-colors ${
                      tip === amount
                        ? 'bg-herb text-paper'
                        : 'bg-mist text-ink-soft hover:bg-mist'
                    }`}
                  >
                    {amount === 0 ? 'No tip' : fp(amount)}
                  </button>
                ))}
                <input
                  type="number"
                  placeholder="Custom"
                  min="0"
                  value={tip > 10 ? tip : ''}
                  onChange={(e) => setTip(Number(e.target.value) || 0)}
                  className="w-24 rounded-lg border px-3 py-2 text-center"
                />
              </div>
            </section>

            {/* Special Instructions */}
            <section className="rounded-xl bg-bone p-6 shadow-1">
              <h2 className="text-lg font-semibold text-ink">Special Instructions</h2>
              <textarea
                value={specialInstructions}
                onChange={(e) => setSpecialInstructions(e.target.value)}
                placeholder="Any special requests or delivery instructions..."
                rows={3}
                className="input-base mt-4"
              />
            </section>
          </div>

          {/* Order Summary */}
          <div className="lg:w-80">
            <div className="rounded-xl bg-bone p-6 shadow-1 lg:sticky lg:top-24">
              <h3 className="text-lg font-semibold text-ink">Order Summary</h3>

              {/* Chef */}
              {cart.chef && (
                <div className="mt-4 flex items-center gap-3 border-b pb-4">
                  {cart.chef.profileImage && (
                    <img
                      src={cart.chef.profileImage}
                      alt=""
                      width={40}
                      height={40}
                      loading="lazy"
                      decoding="async"
                      className="h-10 w-10 rounded-lg object-cover shrink-0"
                      onError={(e) => { e.currentTarget.style.display = 'none'; }}
                    />
                  )}
                  <span className="font-medium text-ink">{cart.chef.businessName}</span>
                </div>
              )}

              {/* Items */}
              <div className="mt-4 space-y-2 border-b pb-4">
                {cart.items.map((item) => (
                  <div key={item.id} className="flex justify-between text-sm">
                    <span className="text-ink-soft">
                      {item.quantity}x {item.name}
                    </span>
                    <span className="text-ink">{fp(item.price * item.quantity, { currency: orderCurrency })}</span>
                  </div>
                ))}
              </div>

              {/* Breakdown */}
              <div className="mt-4 space-y-2 text-sm">
                <div className="flex justify-between text-ink-soft">
                  <span>Subtotal</span>
                  <span>{fp(subtotal, { currency: orderCurrency })}</span>
                </div>
                <div className="flex justify-between text-ink-soft">
                  <span>Delivery fee</span>
                  <span>{fp(deliveryFee, { currency: orderCurrency })}</span>
                </div>
                <div className="flex justify-between text-ink-soft">
                  <span>Service fee</span>
                  <span>{fp(serviceFee, { currency: orderCurrency })}</span>
                </div>
                {discount > 0 && (
                  <div className="flex justify-between text-herb">
                    <span>Promo{cart.promoCode ? ` (${cart.promoCode})` : ''}</span>
                    <span>−{fp(discount, { currency: orderCurrency })}</span>
                  </div>
                )}
                {/* CW-01d / LEG-COREUX-031: Clarify GST line for IN orders.
                    TODO(CW-01e): backend to split out GST line with HSN/SAC
                    code per CGST Act 2017 §31 — currently we render whatever
                    label the backend returns and fall back to "GST" for IN
                    orders so customers see the legally-required name. */}
                <div className="flex justify-between text-ink-soft">
                  <span>
                    {taxRule?.taxName || (taxCountry === 'IN' ? 'GST' : 'Tax')}
                    {rate > 0 ? ` (${rate}%${isInclusive ? ' incl.' : ''})` : ''}
                  </span>
                  <span>{fp(tax, { currency: orderCurrency })}</span>
                </div>
                {tip > 0 && (
                  <div className="flex justify-between text-ink-soft">
                    <span>Tip</span>
                    <span>{fp(tip, { currency: orderCurrency })}</span>
                  </div>
                )}
              </div>

              <div className="mt-4 flex justify-between border-t pt-4 text-lg font-semibold">
                <span>Total</span>
                <span>{fp(total, { currency: orderCurrency })}</span>
              </div>

              {/* CW-01d: explicit per-order T&C + Refund Policy consent.
                  Place Order CTA stays disabled until checked. */}
              <label
                htmlFor="checkout-accept-terms"
                className="mt-6 flex items-start gap-2 text-sm text-ink-soft"
              >
                <input
                  id="checkout-accept-terms"
                  type="checkbox"
                  required
                  checked={acceptedTerms}
                  onChange={(e) => setAcceptedTerms(e.target.checked)}
                  aria-describedby="checkout-accept-terms-help"
                  className="mt-0.5 h-4 w-4 flex-shrink-0 text-herb focus-visible:ring-herb"
                />
                <span id="checkout-accept-terms-help">
                  I agree to the{' '}
                  <Link to="/terms" className="text-herb hover:underline">
                    Terms of Service
                  </Link>{' '}
                  and{' '}
                  <Link to="/refund" className="text-herb hover:underline">
                    Refund Policy
                  </Link>{' '}
                  for this order.
                </span>
              </label>

              <Button
                variant="primary"
                size="lg"
                fullWidth
                isLoading={isProcessing}
                onClick={handlePlaceOrder}
                disabled={isProcessing || !selectedAddress || !acceptedTerms}
                rightIcon={!isProcessing ? <ChevronRight aria-hidden="true" className="h-5 w-5" /> : undefined}
                className="mt-4"
              >
                {isProcessing ? 'Placing Order...' : `Place Order - ${fp(total, { currency: orderCurrency })}`}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
