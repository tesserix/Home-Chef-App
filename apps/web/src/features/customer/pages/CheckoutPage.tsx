import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
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

const SAVED_ADDRESSES: Address[] = [
  {
    id: '1',
    userId: '1',
    label: 'Home',
    line1: '123 Main Street',
    line2: 'Apt 4B',
    city: 'San Francisco',
    state: 'CA',
    postalCode: '94102',
    country: 'USA',
    isDefault: true,
  },
  {
    id: '2',
    userId: '1',
    label: 'Work',
    line1: '456 Market Street',
    city: 'San Francisco',
    state: 'CA',
    postalCode: '94105',
    country: 'USA',
    isDefault: false,
  },
];

export default function CheckoutPage() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const cart = useCartStore();
  const fp = useFormatPrice();
  const [selectedAddress, setSelectedAddress] = useState<string>(SAVED_ADDRESSES[0]?.id || '');
  const [showNewAddress, setShowNewAddress] = useState(false);
  const [scheduledTime, setScheduledTime] = useState<string>('asap');
  const [tip, setTip] = useState<number>(0);
  const [specialInstructions, setSpecialInstructions] = useState('');
  const [isProcessing, setIsProcessing] = useState<boolean>(false);

  const subtotal = cart.getSubtotal();
  const deliveryFee = cart.chef?.deliveryFee || 0;
  const serviceFee = subtotal * 0.05;
  const tax = subtotal * 0.0875; // 8.75% tax
  const total = subtotal + deliveryFee + serviceFee + tax + tip;

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

    if (!window.Razorpay) {
      toast.error('Payment gateway is loading. Please try again.');
      return;
    }

    setIsProcessing(true);

    try {
      // Step 1: Create order
      const order = await apiClient.post<Order>('/orders', {
        items: cart.items,
        chefId: cart.chefId,
        deliveryAddressId: selectedAddress,
        paymentMethodId: 'razorpay',
        tip,
        specialInstructions: specialInstructions || undefined,
        scheduledFor: scheduledTime !== 'asap' ? scheduledTime : undefined,
      });

      // Step 2: Create Razorpay payment order
      const paymentData = await apiClient.post<{
        razorpayOrderId: string;
        razorpayKeyId: string;
        amount: number;
        currency: string;
      }>(`/payments/order/${order.id}/create`, {});

      // Step 3: Open Razorpay checkout
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
        theme: {
          color: '#C2410C',
        },
        handler: async (response) => {
          try {
            // Step 4: Verify payment
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
          ondismiss: () => {
            toast.error('Payment cancelled');
          },
        },
      };

      const rzp = new window.Razorpay(options);
      rzp.open();
    } catch {
      toast.error('Failed to initiate payment. Please try again.');
    } finally {
      setIsProcessing(false);
    }
  };

  const onAddressSubmit = (data: AddressFormData) => {
    // In a real app, this would save to the API
    const newAddress: Address = {
      id: `new_${Date.now()}`,
      userId: user?.id || '',
      ...data,
      country: 'USA',
      isDefault: false,
    };
    SAVED_ADDRESSES.push(newAddress);
    setSelectedAddress(newAddress.id);
    setShowNewAddress(false);
    reset();
    toast.success('Address saved');
  };

  if (cart.items.length === 0) {
    navigate('/cart');
    return null;
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="container-app max-w-4xl">
        <h1 className="text-2xl font-bold text-gray-900 md:text-3xl">Checkout</h1>

        <div className="mt-8 flex flex-col gap-8 lg:flex-row">
          {/* Main Form */}
          <div className="flex-1 space-y-6">
            {/* Delivery Address */}
            <section className="rounded-xl bg-white p-6 shadow-sm">
              <div className="flex items-center justify-between">
                <h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900">
                  <MapPin className="h-5 w-5 text-brand-500" />
                  Delivery Address
                </h2>
                <button
                  onClick={() => setShowNewAddress(!showNewAddress)}
                  className="text-sm text-brand-600 hover:text-brand-700"
                >
                  {showNewAddress ? 'Cancel' : 'Add New'}
                </button>
              </div>

              {showNewAddress ? (
                <form onSubmit={handleSubmit(onAddressSubmit)} className="mt-4 space-y-4">
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div>
                      <label className="block text-sm font-medium text-gray-700">
                        Label
                      </label>
                      <input
                        {...register('label')}
                        placeholder="Home, Work, etc."
                        className="input-base mt-1"
                      />
                      {errors.label && (
                        <p className="mt-1 text-xs text-red-600">{errors.label.message}</p>
                      )}
                    </div>
                    <div className="sm:col-span-2">
                      <label className="block text-sm font-medium text-gray-700">
                        Street Address
                      </label>
                      <input
                        {...register('line1')}
                        placeholder="123 Main Street"
                        className="input-base mt-1"
                      />
                      {errors.line1 && (
                        <p className="mt-1 text-xs text-red-600">{errors.line1.message}</p>
                      )}
                    </div>
                    <div className="sm:col-span-2">
                      <label className="block text-sm font-medium text-gray-700">
                        Apartment, suite, etc. (optional)
                      </label>
                      <input
                        {...register('line2')}
                        placeholder="Apt 4B"
                        className="input-base mt-1"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700">City</label>
                      <input {...register('city')} className="input-base mt-1" />
                      {errors.city && (
                        <p className="mt-1 text-xs text-red-600">{errors.city.message}</p>
                      )}
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="block text-sm font-medium text-gray-700">State</label>
                        <input {...register('state')} className="input-base mt-1" />
                        {errors.state && (
                          <p className="mt-1 text-xs text-red-600">{errors.state.message}</p>
                        )}
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-700">
                          Postal Code
                        </label>
                        <input {...register('postalCode')} className="input-base mt-1" />
                        {errors.postalCode && (
                          <p className="mt-1 text-xs text-red-600">{errors.postalCode.message}</p>
                        )}
                      </div>
                    </div>
                  </div>
                  <button type="submit" className="btn-primary">
                    Save Address
                  </button>
                </form>
              ) : (
                <div className="mt-4 space-y-3">
                  {SAVED_ADDRESSES.map((address) => (
                    <label
                      key={address.id}
                      className={`flex cursor-pointer items-start gap-3 rounded-lg border p-4 transition-colors ${
                        selectedAddress === address.id
                          ? 'border-brand-500 bg-brand-50'
                          : 'border-gray-200 hover:bg-gray-50'
                      }`}
                    >
                      <input
                        type="radio"
                        name="address"
                        value={address.id}
                        checked={selectedAddress === address.id}
                        onChange={(e) => setSelectedAddress(e.target.value)}
                        className="mt-1 h-4 w-4 text-brand-600 focus:ring-brand-500"
                      />
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-gray-900">{address.label}</span>
                          {address.isDefault && (
                            <span className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600">
                              Default
                            </span>
                          )}
                        </div>
                        <p className="mt-1 text-sm text-gray-600">
                          {address.line1}
                          {address.line2 && `, ${address.line2}`}
                        </p>
                        <p className="text-sm text-gray-600">
                          {address.city}, {address.state} {address.postalCode}
                        </p>
                      </div>
                      {selectedAddress === address.id && (
                        <Check className="h-5 w-5 text-brand-500" />
                      )}
                    </label>
                  ))}
                </div>
              )}
            </section>

            {/* Delivery Time */}
            <section className="rounded-xl bg-white p-6 shadow-sm">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900">
                <Clock className="h-5 w-5 text-brand-500" />
                Delivery Time
              </h2>

              <div className="mt-4 space-y-3">
                <label
                  className={`flex cursor-pointer items-center gap-3 rounded-lg border p-4 ${
                    scheduledTime === 'asap'
                      ? 'border-brand-500 bg-brand-50'
                      : 'border-gray-200 hover:bg-gray-50'
                  }`}
                >
                  <input
                    type="radio"
                    name="time"
                    value="asap"
                    checked={scheduledTime === 'asap'}
                    onChange={(e) => setScheduledTime(e.target.value)}
                    className="h-4 w-4 text-brand-600 focus:ring-brand-500"
                  />
                  <div>
                    <span className="font-medium text-gray-900">As soon as possible</span>
                    <p className="text-sm text-gray-500">Usually 30-45 minutes</p>
                  </div>
                </label>
                <label
                  className={`flex cursor-pointer items-center gap-3 rounded-lg border p-4 ${
                    scheduledTime !== 'asap'
                      ? 'border-brand-500 bg-brand-50'
                      : 'border-gray-200 hover:bg-gray-50'
                  }`}
                >
                  <input
                    type="radio"
                    name="time"
                    value="scheduled"
                    checked={scheduledTime !== 'asap'}
                    onChange={() => setScheduledTime('12:00')}
                    className="h-4 w-4 text-brand-600 focus:ring-brand-500"
                  />
                  <div className="flex-1">
                    <span className="font-medium text-gray-900">Schedule for later</span>
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
            <section className="rounded-xl bg-white p-6 shadow-sm">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900">
                <Shield className="h-5 w-5 text-brand-500" />
                Payment
              </h2>
              <div className="mt-4 flex items-center gap-3 rounded-lg border border-gray-200 bg-gray-50 p-4">
                <img
                  src="https://razorpay.com/assets/razorpay-glyph.svg"
                  alt="Razorpay"
                  className="h-6 w-6"
                />
                <div>
                  <p className="text-sm font-medium text-gray-900">Powered by Razorpay</p>
                  <p className="text-xs text-gray-500">
                    Pay securely via UPI, cards, net banking, or wallets
                  </p>
                </div>
              </div>
            </section>

            {/* Tip */}
            <section className="rounded-xl bg-white p-6 shadow-sm">
              <h2 className="text-lg font-semibold text-gray-900">Add a tip</h2>
              <p className="mt-1 text-sm text-gray-500">
                100% of your tip goes to the home chef
              </p>

              <div className="mt-4 flex flex-wrap gap-2">
                {[0, 2, 5, 10].map((amount) => (
                  <button
                    key={amount}
                    onClick={() => setTip(amount)}
                    className={`rounded-lg px-4 py-2 transition-colors ${
                      tip === amount
                        ? 'bg-brand-500 text-white'
                        : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
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
            <section className="rounded-xl bg-white p-6 shadow-sm">
              <h2 className="text-lg font-semibold text-gray-900">Special Instructions</h2>
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
            <div className="rounded-xl bg-white p-6 shadow-sm lg:sticky lg:top-24">
              <h3 className="text-lg font-semibold text-gray-900">Order Summary</h3>

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
                  <span className="font-medium text-gray-900">{cart.chef.businessName}</span>
                </div>
              )}

              {/* Items */}
              <div className="mt-4 space-y-2 border-b pb-4">
                {cart.items.map((item) => (
                  <div key={item.id} className="flex justify-between text-sm">
                    <span className="text-gray-600">
                      {item.quantity}x {item.name}
                    </span>
                    <span className="text-gray-900">{fp(item.price * item.quantity)}</span>
                  </div>
                ))}
              </div>

              {/* Breakdown */}
              <div className="mt-4 space-y-2 text-sm">
                <div className="flex justify-between text-gray-600">
                  <span>Subtotal</span>
                  <span>{fp(subtotal)}</span>
                </div>
                <div className="flex justify-between text-gray-600">
                  <span>Delivery fee</span>
                  <span>{fp(deliveryFee)}</span>
                </div>
                <div className="flex justify-between text-gray-600">
                  <span>Service fee</span>
                  <span>{fp(serviceFee)}</span>
                </div>
                <div className="flex justify-between text-gray-600">
                  <span>Tax</span>
                  <span>{fp(tax)}</span>
                </div>
                {tip > 0 && (
                  <div className="flex justify-between text-gray-600">
                    <span>Tip</span>
                    <span>{fp(tip)}</span>
                  </div>
                )}
              </div>

              <div className="mt-4 flex justify-between border-t pt-4 text-lg font-semibold">
                <span>Total</span>
                <span>{fp(total)}</span>
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
                    Place Order - {fp(total)}
                    <ChevronRight className="ml-2 h-5 w-5" />
                  </>
                )}
              </button>

              <p className="mt-4 text-center text-xs text-gray-500">
                By placing this order, you agree to our Terms of Service
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
