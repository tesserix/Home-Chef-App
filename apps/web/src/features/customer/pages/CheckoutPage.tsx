import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  MapPin,
  Clock,
  ChevronRight,
  Check,
  Loader2,
  Shield,
} from 'lucide-react';
import { toast } from 'sonner';
import { useCartStore } from '@/app/store/cart-store';
import { useAuth } from '@/app/providers/AuthProvider';
import { apiClient } from '@/shared/services/api-client';
import { useFormatPrice } from '@/shared/utils/format-price';
import { loadStripeJs } from '@/shared/utils/load-stripe';
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
  const [scheduledTime, setScheduledTime] = useState<string>('asap');
  const [tip, setTip] = useState<number>(0);
  const [specialInstructions, setSpecialInstructions] = useState('');
  const [isProcessing, setIsProcessing] = useState<boolean>(false);

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
  const rate = taxRule?.rate ?? 0;
  const isInclusive = taxRule?.inclusive ?? false;
  const taxBase = subtotal + deliveryFee + serviceFee;
  const tax = isInclusive
    ? taxBase - taxBase / (1 + rate / 100)
    : taxBase * (rate / 100);
  const total = isInclusive
    ? subtotal + deliveryFee + serviceFee + tip
    : subtotal + deliveryFee + serviceFee + tax + tip;

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

    setIsProcessing(true);

    try {
      // Step 1: Create order. The backend decides which gateway to use
      // based on the chef's PaymentProvider setting.
      const order = await apiClient.post<Order>('/orders', {
        items: cart.items,
        chefId: cart.chefId,
        deliveryAddressId: selectedAddress,
        tip,
        specialInstructions: specialInstructions || undefined,
        scheduledFor: scheduledTime !== 'asap' ? scheduledTime : undefined,
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
    } catch {
      toast.error('Failed to initiate payment. Please try again.');
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
      theme: { color: '#3e6b3c' },
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
            <section className="rounded-xl bg-bone p-6 shadow-sm">
              <div className="flex items-center justify-between">
                <h2 className="flex items-center gap-2 text-lg font-semibold text-ink">
                  <MapPin className="h-5 w-5 text-herb" />
                  Delivery Address
                </h2>
                <button
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
                      <label className="block text-sm font-medium text-ink-soft">
                        Label
                      </label>
                      <input
                        {...register('label')}
                        placeholder="Home, Work, etc."
                        className="input-base mt-1"
                      />
                      {errors.label && (
                        <p className="mt-1 text-xs text-paprika">{errors.label.message}</p>
                      )}
                    </div>
                    <div className="sm:col-span-2">
                      <label className="block text-sm font-medium text-ink-soft">
                        Street Address
                      </label>
                      <input
                        {...register('line1')}
                        placeholder="123 Main Street"
                        className="input-base mt-1"
                      />
                      {errors.line1 && (
                        <p className="mt-1 text-xs text-paprika">{errors.line1.message}</p>
                      )}
                    </div>
                    <div className="sm:col-span-2">
                      <label className="block text-sm font-medium text-ink-soft">
                        Apartment, suite, etc. (optional)
                      </label>
                      <input
                        {...register('line2')}
                        placeholder="Apt 4B"
                        className="input-base mt-1"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-ink-soft">City</label>
                      <input {...register('city')} className="input-base mt-1" />
                      {errors.city && (
                        <p className="mt-1 text-xs text-paprika">{errors.city.message}</p>
                      )}
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="block text-sm font-medium text-ink-soft">State</label>
                        <input {...register('state')} className="input-base mt-1" />
                        {errors.state && (
                          <p className="mt-1 text-xs text-paprika">{errors.state.message}</p>
                        )}
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-ink-soft">
                          Postal Code
                        </label>
                        <input {...register('postalCode')} className="input-base mt-1" />
                        {errors.postalCode && (
                          <p className="mt-1 text-xs text-paprika">{errors.postalCode.message}</p>
                        )}
                      </div>
                    </div>
                  </div>
                  <button type="submit" className="btn-primary">
                    Save Address
                  </button>
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
                        className="mt-1 h-4 w-4 text-herb focus:ring-herb"
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
                        <Check className="h-5 w-5 text-herb" />
                      )}
                    </label>
                  ))}
                </div>
              )}
            </section>

            {/* Delivery Time */}
            <section className="rounded-xl bg-bone p-6 shadow-sm">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-ink">
                <Clock className="h-5 w-5 text-herb" />
                Delivery Time
              </h2>

              <div className="mt-4 space-y-3">
                <label
                  className={`flex cursor-pointer items-center gap-3 rounded-lg border p-4 ${
                    scheduledTime === 'asap'
                      ? 'border-herb bg-herb-tint'
                      : 'border-mist hover:bg-paper'
                  }`}
                >
                  <input
                    type="radio"
                    name="time"
                    value="asap"
                    checked={scheduledTime === 'asap'}
                    onChange={(e) => setScheduledTime(e.target.value)}
                    className="h-4 w-4 text-herb focus:ring-herb"
                  />
                  <div>
                    <span className="font-medium text-ink">As soon as possible</span>
                    <p className="text-sm text-ink-muted">Usually 30-45 minutes</p>
                  </div>
                </label>
                <label
                  className={`flex cursor-pointer items-center gap-3 rounded-lg border p-4 ${
                    scheduledTime !== 'asap'
                      ? 'border-herb bg-herb-tint'
                      : 'border-mist hover:bg-paper'
                  }`}
                >
                  <input
                    type="radio"
                    name="time"
                    value="scheduled"
                    checked={scheduledTime !== 'asap'}
                    onChange={() => setScheduledTime('12:00')}
                    className="h-4 w-4 text-herb focus:ring-herb"
                  />
                  <div className="flex-1">
                    <span className="font-medium text-ink">Schedule for later</span>
                    {scheduledTime !== 'asap' && (
                      <select
                        value={scheduledTime}
                        onChange={(e) => setScheduledTime(e.target.value)}
                        className="input-base mt-2"
                      >
                        <option value="12:00">12:00 PM</option>
                        <option value="12:30">12:30 PM</option>
                        <option value="13:00">1:00 PM</option>
                        <option value="13:30">1:30 PM</option>
                        <option value="18:00">6:00 PM</option>
                        <option value="18:30">6:30 PM</option>
                        <option value="19:00">7:00 PM</option>
                        <option value="19:30">7:30 PM</option>
                      </select>
                    )}
                  </div>
                </label>
              </div>
            </section>

            {/* Payment */}
            <section className="rounded-xl bg-bone p-6 shadow-sm">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-ink">
                <Shield className="h-5 w-5 text-herb" />
                Payment
              </h2>
              <div className="mt-4 flex items-center gap-3 rounded-lg border border-mist bg-paper p-4">
                <img
                  src="https://razorpay.com/assets/razorpay-glyph.svg"
                  alt="Razorpay"
                  className="h-6 w-6"
                />
                <div>
                  <p className="text-sm font-medium text-ink">Powered by Razorpay</p>
                  <p className="text-xs text-ink-muted">
                    Pay securely via UPI, cards, net banking, or wallets
                  </p>
                </div>
              </div>
            </section>

            {/* Tip */}
            <section className="rounded-xl bg-bone p-6 shadow-sm">
              <h2 className="text-lg font-semibold text-ink">Add a tip</h2>
              <p className="mt-1 text-sm text-ink-muted">
                100% of your tip goes to the home chef
              </p>

              <div className="mt-4 flex flex-wrap gap-2">
                {[0, 2, 5, 10].map((amount) => (
                  <button
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
            <section className="rounded-xl bg-bone p-6 shadow-sm">
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
            <div className="rounded-xl bg-bone p-6 shadow-sm lg:sticky lg:top-24">
              <h3 className="text-lg font-semibold text-ink">Order Summary</h3>

              {/* Chef */}
              {cart.chef && (
                <div className="mt-4 flex items-center gap-3 border-b pb-4">
                  {cart.chef.profileImage && (
                    <img
                      src={cart.chef.profileImage}
                      alt={cart.chef.businessName}
                      className="h-10 w-10 rounded-lg object-cover"
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
                <div className="flex justify-between text-ink-soft">
                  <span>
                    {taxRule?.taxName || 'Tax'}
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

              <button
                onClick={handlePlaceOrder}
                disabled={isProcessing || !selectedAddress}
                className="btn-primary mt-6 w-full py-4 disabled:opacity-50"
              >
                {isProcessing ? (
                  <>
                    <Loader2 className="h-5 w-5 animate-spin" />
                    Placing Order...
                  </>
                ) : (
                  <>
                    Place Order - {fp(total, { currency: orderCurrency })}
                    <ChevronRight className="ml-2 h-5 w-5" />
                  </>
                )}
              </button>

              <p className="mt-4 text-center text-xs text-ink-muted">
                By placing this order, you agree to our Terms of Service
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
