// Checkout screen — address selection, order summary, Razorpay hosted checkout via expo-web-browser.
//
// D-04 decision: react-native-razorpay is incompatible with Expo managed workflow SDK 55.
// Using expo-web-browser + Razorpay hosted checkout URL as fallback.
// DO NOT import react-native-razorpay or razorpay-react-native-checkout.

import { useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  Text,
  TextInput,
  View,
} from 'react-native';
import { Link, router } from 'expo-router';
import * as Haptics from 'expo-haptics';
import * as WebBrowser from 'expo-web-browser';
import { Check, ChevronLeft, Clock, FileText, MapPin, Plus } from 'lucide-react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useCartStore } from '../store/cart-store';
import { useCreateOrder, useOrderStatus } from '../hooks/useOrderCheckout';
import { useAddresses, useCreateAddress } from '../hooks/useAddresses';
import { api } from '../lib/api';
import type { Address } from '../types/customer';

// ─── Address form schema ──────────────────────────────────────────────────────

const addressSchema = z.object({
  addressLine1: z.string().min(1, 'Address line 1 is required'),
  addressLine2: z.string().optional(),
  city: z.string().min(1, 'City is required'),
  state: z.string().min(1, 'State is required'),
  pincode: z
    .string()
    .regex(/^\d{6}$/, 'Pincode must be 6 digits'),
  isDefault: z.boolean().optional(),
});

type AddressFormValues = z.infer<typeof addressSchema>;

// ─── Razorpay payment data shape ─────────────────────────────────────────────

interface RazorpayPaymentData {
  razorpayOrderId: string;
  razorpayKeyId: string;
  amount: number;
  currency: string;
  orderNumber?: string;
  prefill?: {
    name?: string;
    email?: string;
    phone?: string;
  };
}

// ─── Component ────────────────────────────────────────────────────────────────

export default function CheckoutScreen() {
  const cartStore = useCartStore();
  const createOrder = useCreateOrder();
  const createAddress = useCreateAddress();
  const { data: addressData, isLoading: addressLoading } = useAddresses();

  const addresses = addressData?.data ?? [];

  const [selectedAddressId, setSelectedAddressId] = useState<string>('');
  const [note, setNote] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pollingOrderId, setPollingOrderId] = useState<string | null>(null);
  const [showAddressForm, setShowAddressForm] = useState(false);
  // CW-01d: explicit per-order T&C + Refund Policy consent for RBI PA disclosure.
  const [acceptedTerms, setAcceptedTerms] = useState(false);

  // Pre-select default address when addresses load
  useEffect(() => {
    if (addresses.length > 0 && !selectedAddressId) {
      const defaultAddr = addresses.find((a) => a.isDefault) ?? addresses[0];
      if (defaultAddr?.id) {
        setSelectedAddressId(defaultAddr.id);
      }
    }
  }, [addresses]);

  // ─── Polling ──────────────────────────────────────────────────────────────

  const { data: polledOrder } = useOrderStatus(pollingOrderId ?? '', !!pollingOrderId);

  useEffect(() => {
    if (!polledOrder) return;
    const status = polledOrder.data.status;
    if (status !== 'pending') {
      setPollingOrderId(null);
      setIsLoading(false);

      if (
        status === 'confirmed' ||
        status === 'preparing' ||
        status === 'ready' ||
        status === 'picked_up' ||
        status === 'delivered'
      ) {
        useCartStore.getState().clearCart();
        Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
        router.replace(`/order/${polledOrder.data.id}`);
      } else if (status === 'cancelled') {
        setError('Payment was not completed. Please try again.');
      }
    }
  }, [polledOrder]);

  // Hard timeout: stop polling after 60s
  useEffect(() => {
    if (!pollingOrderId) return;
    const timeout = setTimeout(() => {
      setPollingOrderId(null);
      setIsLoading(false);
      setError('Payment confirmation timed out. Check your order history to confirm status.');
    }, 60000);
    return () => clearTimeout(timeout);
  }, [pollingOrderId]);

  // Clean up polling on unmount
  const pollingOrderIdRef = useRef(pollingOrderId);
  pollingOrderIdRef.current = pollingOrderId;
  useEffect(() => {
    return () => {
      pollingOrderIdRef.current = null;
    };
  }, []);

  // ─── Address form ─────────────────────────────────────────────────────────

  const {
    control: addrControl,
    handleSubmit: handleAddrSubmit,
    reset: resetAddrForm,
    formState: { errors: addrErrors, isSubmitting: addrSubmitting },
  } = useForm<AddressFormValues>({
    resolver: zodResolver(addressSchema),
    defaultValues: {
      addressLine1: '',
      addressLine2: '',
      city: '',
      state: '',
      pincode: '',
      isDefault: false,
    },
  });

  async function onSaveAddress(values: AddressFormValues) {
    try {
      const result = await createAddress.mutateAsync({
        addressLine1: values.addressLine1,
        addressLine2: values.addressLine2,
        city: values.city,
        state: values.state,
        pincode: values.pincode,
        isDefault: values.isDefault,
      });
      if (result.data.id) {
        setSelectedAddressId(result.data.id);
      }
      resetAddrForm();
      setShowAddressForm(false);
    } catch {
      // Error displayed inline by mutation state if needed
    }
  }

  // ─── Place Order flow ──────────────────────────────────────────────────────

  async function handlePlaceOrder() {
    if (!selectedAddressId || cartStore.items.length === 0 || !cartStore.chefId) return;
    if (!acceptedTerms) {
      setError('Please accept the Terms of Service and Refund Policy to continue.');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      // Step 1: Create the order on the server
      const orderResult = await createOrder.mutateAsync({
        chefId: cartStore.chefId,
        items: cartStore.items.map((i) => ({ menuItemId: i.menuItemId, quantity: i.quantity })),
        deliveryAddressId: selectedAddressId,
        note: note.trim() || undefined,
      });

      const orderId = orderResult.data.id;

      // Step 2: Create Razorpay payment order (server-side)
      const paymentResp = await api.post<{ data: RazorpayPaymentData }>(
        `/v1/payments/order/${orderId}/create`,
        {}
      );
      const paymentData = paymentResp.data.data ?? (paymentResp.data as unknown as RazorpayPaymentData);

      // Step 3: Build Razorpay hosted checkout URL (D-04 fallback — expo-web-browser)
      const checkoutUrl = [
        'https://api.razorpay.com/v1/checkout/embedded',
        `?key_id=${paymentData.razorpayKeyId}`,
        `&order_id=${paymentData.razorpayOrderId}`,
        `&amount=${paymentData.amount}`,
        `&currency=${paymentData.currency ?? 'INR'}`,
        `&name=Fe3dr`,
        `&prefill[name]=${encodeURIComponent(paymentData.prefill?.name ?? '')}`,
        `&prefill[email]=${encodeURIComponent(paymentData.prefill?.email ?? '')}`,
        `&prefill[contact]=${encodeURIComponent(paymentData.prefill?.phone ?? '')}`,
        `&callback_url=${encodeURIComponent('homechef-customer://payment/result')}`,
      ].join('');

      // Step 4: Open hosted checkout in in-app browser
      await WebBrowser.openBrowserAsync(checkoutUrl);

      // Step 5: Start polling — webhook will update status server-side
      setPollingOrderId(orderId);
    } catch {
      setError('Order creation failed. Please try again.');
      setIsLoading(false);
    }
  }

  // ─── Helpers ──────────────────────────────────────────────────────────────

  const canPlaceOrder =
    cartStore.items.length > 0 &&
    !!selectedAddressId &&
    !!cartStore.chefId &&
    !isLoading &&
    acceptedTerms;

  function formatAddress(addr: Address): string {
    const parts = [addr.addressLine1];
    if (addr.addressLine2) parts.push(addr.addressLine2);
    parts.push(`${addr.city}, ${addr.state} ${addr.pincode}`);
    return parts.join(', ');
  }

  const subtotal = cartStore.total();
  const deliveryFee = 0; // free for v1
  const total = subtotal + deliveryFee;

  // ─── Render ───────────────────────────────────────────────────────────────

  return (
    <KeyboardAvoidingView
      className="flex-1 bg-paper"
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      {/* Header */}
      <View className="flex-row items-center bg-bone border-b border-mist px-4 pt-14 pb-4 gap-3">
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          className="p-1"
        >
          <ChevronLeft size={24} color="#4a4a47" />
        </Pressable>
        <Text className="text-xl font-semibold text-ink flex-1">Checkout</Text>
      </View>

      <ScrollView
        contentContainerStyle={{ paddingBottom: 120 }}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}
      >
        {/* ── Delivery Address ── */}
        <View className="mx-4 mt-4 bg-bone rounded-2xl overflow-hidden">
          <View className="flex-row items-center px-4 pt-4 pb-2 gap-2">
            <MapPin size={18} color="#C2410C" />
            <Text className="text-base font-semibold text-ink">Delivery Address</Text>
          </View>

          {addressLoading ? (
            <View className="px-4 pb-4">
              <ActivityIndicator size="small" color="#C2410C" />
            </View>
          ) : (
            <>
              {addresses.map((addr) => (
                <Pressable
                  key={addr.id}
                  onPress={() => addr.id && setSelectedAddressId(addr.id)}
                  className={`mx-4 mb-2 p-3 rounded-xl border ${
                    selectedAddressId === addr.id
                      ? 'border-herb bg-herb-tint'
                      : 'border-mist bg-paper'
                  }`}
                  accessibilityRole="radio"
                  accessibilityState={{ checked: selectedAddressId === addr.id }}
                  accessibilityLabel={formatAddress(addr)}
                >
                  <View className="flex-row items-start gap-3">
                    <View
                      className={`w-4 h-4 rounded-full border-2 mt-0.5 ${
                        selectedAddressId === addr.id
                          ? 'border-herb bg-herb'
                          : 'border-mist-strong bg-bone'
                      }`}
                    />
                    <View className="flex-1">
                      <Text className="text-sm text-ink">{formatAddress(addr)}</Text>
                      {addr.isDefault && (
                        <Text className="text-xs text-herb font-medium mt-0.5">Default</Text>
                      )}
                    </View>
                  </View>
                </Pressable>
              ))}

              {/* Add new address toggle */}
              <Pressable
                onPress={() => setShowAddressForm((prev: boolean) => !prev)}
                className="flex-row items-center gap-2 px-4 py-3"
                accessibilityRole="button"
                accessibilityLabel="Add new address"
              >
                <Plus size={16} color="#C2410C" />
                <Text className="text-sm text-herb font-medium">Add New Address</Text>
              </Pressable>

              {showAddressForm && (
                <View className="px-4 pb-4 gap-3">
                  <Controller
                    control={addrControl}
                    name="addressLine1"
                    render={({ field: { onChange, value } }) => (
                      <View>
                        <TextInput
                          value={value}
                          onChangeText={onChange}
                          placeholder="Address line 1 *"
                          placeholderTextColor="#7a7a76"
                          className="bg-mist rounded-xl px-3 py-2.5 text-sm text-ink"
                          accessibilityLabel="Address line 1"
                        />
                        {addrErrors.addressLine1 && (
                          <Text className="text-xs text-paprika mt-1">
                            {addrErrors.addressLine1.message}
                          </Text>
                        )}
                      </View>
                    )}
                  />
                  <Controller
                    control={addrControl}
                    name="addressLine2"
                    render={({ field: { onChange, value } }) => (
                      <TextInput
                        value={value}
                        onChangeText={onChange}
                        placeholder="Address line 2 (optional)"
                        placeholderTextColor="#7a7a76"
                        className="bg-mist rounded-xl px-3 py-2.5 text-sm text-ink"
                        accessibilityLabel="Address line 2"
                      />
                    )}
                  />
                  <View className="flex-row gap-2">
                    <Controller
                      control={addrControl}
                      name="city"
                      render={({ field: { onChange, value } }) => (
                        <View className="flex-1">
                          <TextInput
                            value={value}
                            onChangeText={onChange}
                            placeholder="City *"
                            placeholderTextColor="#7a7a76"
                            className="bg-mist rounded-xl px-3 py-2.5 text-sm text-ink"
                            accessibilityLabel="City"
                          />
                          {addrErrors.city && (
                            <Text className="text-xs text-paprika mt-1">
                              {addrErrors.city.message}
                            </Text>
                          )}
                        </View>
                      )}
                    />
                    <Controller
                      control={addrControl}
                      name="state"
                      render={({ field: { onChange, value } }) => (
                        <View className="flex-1">
                          <TextInput
                            value={value}
                            onChangeText={onChange}
                            placeholder="State *"
                            placeholderTextColor="#7a7a76"
                            className="bg-mist rounded-xl px-3 py-2.5 text-sm text-ink"
                            accessibilityLabel="State"
                          />
                          {addrErrors.state && (
                            <Text className="text-xs text-paprika mt-1">
                              {addrErrors.state.message}
                            </Text>
                          )}
                        </View>
                      )}
                    />
                  </View>
                  <Controller
                    control={addrControl}
                    name="pincode"
                    render={({ field: { onChange, value } }) => (
                      <View>
                        <TextInput
                          value={value}
                          onChangeText={onChange}
                          placeholder="Pincode *"
                          placeholderTextColor="#7a7a76"
                          keyboardType="numeric"
                          maxLength={6}
                          className="bg-mist rounded-xl px-3 py-2.5 text-sm text-ink"
                          accessibilityLabel="Pincode"
                        />
                        {addrErrors.pincode && (
                          <Text className="text-xs text-paprika mt-1">
                            {addrErrors.pincode.message}
                          </Text>
                        )}
                      </View>
                    )}
                  />
                  <Pressable
                    onPress={handleAddrSubmit(onSaveAddress)}
                    disabled={addrSubmitting}
                    className={`rounded-xl py-2.5 items-center ${
                      addrSubmitting ? 'bg-herb-soft' : 'bg-herb'
                    }`}
                    accessibilityRole="button"
                    accessibilityLabel="Save address"
                  >
                    {addrSubmitting ? (
                      <ActivityIndicator size="small" color="#fafaf7" />
                    ) : (
                      <Text className="text-paper font-semibold text-sm">Save Address</Text>
                    )}
                  </Pressable>
                </View>
              )}
            </>
          )}
        </View>

        {/* ── Order Summary ── */}
        <View className="mx-4 mt-4 bg-bone rounded-2xl overflow-hidden">
          <Text className="text-base font-semibold text-ink px-4 pt-4 pb-2">Order Summary</Text>
          <FlatList
            data={cartStore.items}
            keyExtractor={(item) => item.menuItemId}
            scrollEnabled={false}
            renderItem={({ item }) => (
              <View className="flex-row items-center px-4 py-2 gap-2">
                <View className="w-6 h-6 rounded-full bg-herb-tint items-center justify-center">
                  <Text className="text-xs font-medium text-herb">{item.quantity}</Text>
                </View>
                <Text className="flex-1 text-sm text-ink">{item.name}</Text>
                <Text className="text-sm font-medium text-ink">
                  ₹{(item.price * item.quantity).toFixed(2)}
                </Text>
              </View>
            )}
            ListEmptyComponent={
              <View className="px-4 py-4">
                <Text className="text-sm text-ink-muted">Your cart is empty.</Text>
              </View>
            }
          />
          <View className="border-t border-mist mx-4 mt-2 pt-3 pb-4 gap-2">
            <View className="flex-row justify-between">
              <Text className="text-sm text-ink-muted">Subtotal</Text>
              <Text className="text-sm text-ink">₹{subtotal.toFixed(2)}</Text>
            </View>
            <View className="flex-row justify-between">
              <Text className="text-sm text-ink-muted">Delivery fee</Text>
              <Text className="text-sm text-herb font-medium">Free</Text>
            </View>
            <View className="flex-row justify-between pt-1 border-t border-mist">
              <Text className="text-base font-medium text-ink">Total</Text>
              <Text className="text-base font-medium text-ink">₹{total.toFixed(2)}</Text>
            </View>
          </View>
        </View>

        {/* ── Payment & refund summary (CW-01d / RBI PA MD §8) ── */}
        <View className="mx-4 mt-4 bg-bone rounded-2xl p-4 gap-3">
          <View>
            <Text className="text-sm font-semibold text-ink mb-1">Payment & refund summary</Text>
            <Text className="text-sm text-ink-soft leading-5">
              Payments are processed by Razorpay (RBI-licensed payment aggregator).
              Tesserix Pty Ltd (operator of Fe3dr) facilitates the transaction;
              order proceeds go to your chef minus the platform commission.
            </Text>
          </View>
          <View className="flex-row items-start gap-2">
            <Clock size={16} color="#C2410C" style={{ marginTop: 2 }} />
            <Text className="text-sm text-ink-soft flex-1 leading-5">
              Refunds return to your original payment method within{' '}
              <Text className="font-semibold text-ink">7 working days</Text> per RBI Payment
              Aggregator Master Direction §8.
            </Text>
          </View>
          <View className="flex-row items-start gap-2">
            <FileText size={16} color="#C2410C" style={{ marginTop: 2 }} />
            <Text className="text-sm text-ink-soft flex-1 leading-5">
              See our{' '}
              <Link href="/refund" className="text-herb underline">
                Refund Policy
              </Link>{' '}
              for cancellation rules by order stage.
            </Text>
          </View>
          <View className="flex-row items-start gap-2">
            <Clock size={16} color="#C2410C" style={{ marginTop: 2 }} />
            <Text className="text-sm text-ink-soft flex-1 leading-5">
              Estimated delivery: 30–45 minutes after the chef accepts your order.
              Actual time depends on the chef's preparation and your driver's route.
            </Text>
          </View>
        </View>

        {/* ── T&C + Refund consent (CW-01d) ── */}
        <View className="mx-4 mt-4">
          <Pressable
            onPress={() => setAcceptedTerms((prev: boolean) => !prev)}
            accessibilityRole="checkbox"
            accessibilityState={{ checked: acceptedTerms }}
            accessibilityLabel="I agree to the Terms of Service and Refund Policy for this order"
            className="flex-row items-start gap-3 bg-bone rounded-2xl p-4"
          >
            <View
              className={`w-5 h-5 rounded border-2 items-center justify-center mt-0.5 ${
                acceptedTerms ? 'border-herb bg-herb' : 'border-mist-strong bg-paper'
              }`}
            >
              {acceptedTerms && <Check size={14} color="#fafaf7" />}
            </View>
            <Text className="flex-1 text-sm text-ink-soft leading-5">
              I agree to the{' '}
              <Link href="/terms" className="text-herb underline">
                Terms of Service
              </Link>{' '}
              and{' '}
              <Link href="/refund" className="text-herb underline">
                Refund Policy
              </Link>{' '}
              for this order.
            </Text>
          </Pressable>
        </View>

        {/* ── Note (optional) ── */}
        <View className="mx-4 mt-4 bg-bone rounded-2xl p-4">
          <Text className="text-sm font-medium text-ink-soft mb-2">Note to chef (optional)</Text>
          <TextInput
            value={note}
            onChangeText={setNote}
            placeholder="Any special instructions..."
            placeholderTextColor="#7a7a76"
            multiline
            numberOfLines={2}
            maxLength={200}
            className="bg-paper rounded-xl px-3 py-2.5 text-sm text-ink min-h-[60px]"
            accessibilityLabel="Note to chef"
            textAlignVertical="top"
          />
        </View>

        {/* ── Error ── */}
        {error && (
          <View className="mx-4 mt-4 bg-paprika-tint border border-paprika/30 rounded-xl px-4 py-3">
            <Text className="text-sm text-paprika">{error}</Text>
            <Pressable
              onPress={() => setError(null)}
              className="mt-1"
              accessibilityRole="button"
              accessibilityLabel="Dismiss error"
            >
              <Text className="text-xs text-paprika underline">Dismiss</Text>
            </Pressable>
          </View>
        )}
      </ScrollView>

      {/* ── Place Order button (fixed bottom) ── */}
      <View className="absolute bottom-0 left-0 right-0 bg-bone border-t border-mist px-4 pt-3 pb-8">
        <Pressable
          onPress={handlePlaceOrder}
          disabled={!canPlaceOrder}
          className={`w-full rounded-2xl py-4 items-center justify-center flex-row gap-2 ${
            canPlaceOrder ? 'bg-herb active:bg-herb' : 'bg-mist-strong'
          }`}
          accessibilityRole="button"
          accessibilityLabel="Place Order"
          accessibilityState={{ disabled: !canPlaceOrder }}
        >
          {isLoading ? (
            <ActivityIndicator size="small" color="#fafaf7" />
          ) : (
            <Text className={`text-base font-medium ${canPlaceOrder ? 'text-paper' : 'text-ink-muted'}`}>
              {isLoading ? 'Processing...' : `Place Order · ₹${total.toFixed(2)}`}
            </Text>
          )}
        </Pressable>
      </View>
    </KeyboardAvoidingView>
  );
}
