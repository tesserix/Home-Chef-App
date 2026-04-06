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
import { router } from 'expo-router';
import * as Haptics from 'expo-haptics';
import * as WebBrowser from 'expo-web-browser';
import { ChevronLeft, MapPin, Plus } from 'lucide-react-native';
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
    !isLoading;

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
      className="flex-1 bg-gray-50"
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      {/* Header */}
      <View className="flex-row items-center bg-white border-b border-gray-100 px-4 pt-14 pb-4 gap-3">
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          className="p-1"
        >
          <ChevronLeft size={24} color="#374151" />
        </Pressable>
        <Text className="text-xl font-bold text-gray-900 flex-1">Checkout</Text>
      </View>

      <ScrollView
        contentContainerStyle={{ paddingBottom: 120 }}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}
      >
        {/* ── Delivery Address ── */}
        <View className="mx-4 mt-4 bg-white rounded-2xl overflow-hidden">
          <View className="flex-row items-center px-4 pt-4 pb-2 gap-2">
            <MapPin size={18} color="#F97316" />
            <Text className="text-base font-semibold text-gray-900">Delivery Address</Text>
          </View>

          {addressLoading ? (
            <View className="px-4 pb-4">
              <ActivityIndicator size="small" color="#F97316" />
            </View>
          ) : (
            <>
              {addresses.map((addr) => (
                <Pressable
                  key={addr.id}
                  onPress={() => addr.id && setSelectedAddressId(addr.id)}
                  className={`mx-4 mb-2 p-3 rounded-xl border ${
                    selectedAddressId === addr.id
                      ? 'border-orange-400 bg-orange-50'
                      : 'border-gray-200 bg-gray-50'
                  }`}
                  accessibilityRole="radio"
                  accessibilityState={{ checked: selectedAddressId === addr.id }}
                  accessibilityLabel={formatAddress(addr)}
                >
                  <View className="flex-row items-start gap-3">
                    <View
                      className={`w-4 h-4 rounded-full border-2 mt-0.5 ${
                        selectedAddressId === addr.id
                          ? 'border-orange-500 bg-orange-500'
                          : 'border-gray-400 bg-white'
                      }`}
                    />
                    <View className="flex-1">
                      <Text className="text-sm text-gray-800">{formatAddress(addr)}</Text>
                      {addr.isDefault && (
                        <Text className="text-xs text-orange-500 font-medium mt-0.5">Default</Text>
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
                <Plus size={16} color="#F97316" />
                <Text className="text-sm text-orange-500 font-medium">Add New Address</Text>
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
                          placeholderTextColor="#9CA3AF"
                          className="bg-gray-100 rounded-xl px-3 py-2.5 text-sm text-gray-900"
                          accessibilityLabel="Address line 1"
                        />
                        {addrErrors.addressLine1 && (
                          <Text className="text-xs text-red-500 mt-1">
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
                        placeholderTextColor="#9CA3AF"
                        className="bg-gray-100 rounded-xl px-3 py-2.5 text-sm text-gray-900"
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
                            placeholderTextColor="#9CA3AF"
                            className="bg-gray-100 rounded-xl px-3 py-2.5 text-sm text-gray-900"
                            accessibilityLabel="City"
                          />
                          {addrErrors.city && (
                            <Text className="text-xs text-red-500 mt-1">
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
                            placeholderTextColor="#9CA3AF"
                            className="bg-gray-100 rounded-xl px-3 py-2.5 text-sm text-gray-900"
                            accessibilityLabel="State"
                          />
                          {addrErrors.state && (
                            <Text className="text-xs text-red-500 mt-1">
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
                          placeholderTextColor="#9CA3AF"
                          keyboardType="numeric"
                          maxLength={6}
                          className="bg-gray-100 rounded-xl px-3 py-2.5 text-sm text-gray-900"
                          accessibilityLabel="Pincode"
                        />
                        {addrErrors.pincode && (
                          <Text className="text-xs text-red-500 mt-1">
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
                      addrSubmitting ? 'bg-orange-300' : 'bg-orange-500'
                    }`}
                    accessibilityRole="button"
                    accessibilityLabel="Save address"
                  >
                    {addrSubmitting ? (
                      <ActivityIndicator size="small" color="#fff" />
                    ) : (
                      <Text className="text-white font-semibold text-sm">Save Address</Text>
                    )}
                  </Pressable>
                </View>
              )}
            </>
          )}
        </View>

        {/* ── Order Summary ── */}
        <View className="mx-4 mt-4 bg-white rounded-2xl overflow-hidden">
          <Text className="text-base font-semibold text-gray-900 px-4 pt-4 pb-2">Order Summary</Text>
          <FlatList
            data={cartStore.items}
            keyExtractor={(item) => item.menuItemId}
            scrollEnabled={false}
            renderItem={({ item }) => (
              <View className="flex-row items-center px-4 py-2 gap-2">
                <View className="w-6 h-6 rounded-full bg-orange-100 items-center justify-center">
                  <Text className="text-xs font-bold text-orange-600">{item.quantity}</Text>
                </View>
                <Text className="flex-1 text-sm text-gray-800">{item.name}</Text>
                <Text className="text-sm font-medium text-gray-900">
                  ₹{(item.price * item.quantity).toFixed(2)}
                </Text>
              </View>
            )}
            ListEmptyComponent={
              <View className="px-4 py-4">
                <Text className="text-sm text-gray-400">Your cart is empty.</Text>
              </View>
            }
          />
          <View className="border-t border-gray-100 mx-4 mt-2 pt-3 pb-4 gap-2">
            <View className="flex-row justify-between">
              <Text className="text-sm text-gray-500">Subtotal</Text>
              <Text className="text-sm text-gray-800">₹{subtotal.toFixed(2)}</Text>
            </View>
            <View className="flex-row justify-between">
              <Text className="text-sm text-gray-500">Delivery fee</Text>
              <Text className="text-sm text-green-600 font-medium">Free</Text>
            </View>
            <View className="flex-row justify-between pt-1 border-t border-gray-100">
              <Text className="text-base font-bold text-gray-900">Total</Text>
              <Text className="text-base font-bold text-gray-900">₹{total.toFixed(2)}</Text>
            </View>
          </View>
        </View>

        {/* ── Note (optional) ── */}
        <View className="mx-4 mt-4 bg-white rounded-2xl p-4">
          <Text className="text-sm font-medium text-gray-700 mb-2">Note to chef (optional)</Text>
          <TextInput
            value={note}
            onChangeText={setNote}
            placeholder="Any special instructions..."
            placeholderTextColor="#9CA3AF"
            multiline
            numberOfLines={2}
            maxLength={200}
            className="bg-gray-50 rounded-xl px-3 py-2.5 text-sm text-gray-900 min-h-[60px]"
            accessibilityLabel="Note to chef"
            textAlignVertical="top"
          />
        </View>

        {/* ── Error ── */}
        {error && (
          <View className="mx-4 mt-4 bg-red-50 border border-red-200 rounded-xl px-4 py-3">
            <Text className="text-sm text-red-600">{error}</Text>
            <Pressable
              onPress={() => setError(null)}
              className="mt-1"
              accessibilityRole="button"
              accessibilityLabel="Dismiss error"
            >
              <Text className="text-xs text-red-400 underline">Dismiss</Text>
            </Pressable>
          </View>
        )}
      </ScrollView>

      {/* ── Place Order button (fixed bottom) ── */}
      <View className="absolute bottom-0 left-0 right-0 bg-white border-t border-gray-100 px-4 pt-3 pb-8">
        <Pressable
          onPress={handlePlaceOrder}
          disabled={!canPlaceOrder}
          className={`w-full rounded-2xl py-4 items-center justify-center flex-row gap-2 ${
            canPlaceOrder ? 'bg-orange-500 active:bg-orange-600' : 'bg-gray-300'
          }`}
          accessibilityRole="button"
          accessibilityLabel="Place Order"
          accessibilityState={{ disabled: !canPlaceOrder }}
        >
          {isLoading ? (
            <ActivityIndicator size="small" color="#fff" />
          ) : (
            <Text className={`text-base font-bold ${canPlaceOrder ? 'text-white' : 'text-gray-400'}`}>
              {isLoading ? 'Processing...' : `Place Order · ₹${total.toFixed(2)}`}
            </Text>
          )}
        </Pressable>
      </View>
    </KeyboardAvoidingView>
  );
}
