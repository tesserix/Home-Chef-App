// Checkout screen — address selection, order summary, then Razorpay payment.
//
// Payment runs in-app via the WebView screen at app/payment/checkout.tsx
// (Razorpay Standard Checkout). This screen creates the order + Razorpay order,
// then hands off to that screen, which verifies the result server-side.

import { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  FlatList,
  Keyboard,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  Text,
  TextInput,
  View,
} from 'react-native';
import { Link, router, type Href } from 'expo-router';
import { Check, ChevronLeft, Clock, MapPin, Plus, Search } from 'lucide-react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useCartStore } from '../store/cart-store';
import { useCreateOrder } from '../hooks/useOrderCheckout';
import { useAddresses, useCreateAddress } from '../hooks/useAddresses';
import {
  useAddressAutocomplete,
  type AddressSuggestion,
} from '../hooks/useLocations';
import { api } from '../lib/api';
import { friendlyErrorMessage } from '../lib/errors';
import { AddressLabelSelect } from '../components/address/AddressLabelSelect';
import type { Address } from '../types/customer';

// ─── Address form schema ──────────────────────────────────────────────────────

const addressSchema = z.object({
  label: z.string().min(1),
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

  // ─── Address form ─────────────────────────────────────────────────────────

  const {
    control: addrControl,
    handleSubmit: handleAddrSubmit,
    reset: resetAddrForm,
    setValue: setAddrValue,
    formState: { errors: addrErrors, isSubmitting: addrSubmitting },
  } = useForm<AddressFormValues>({
    resolver: zodResolver(addressSchema),
    defaultValues: {
      label: 'Home',
      addressLine1: '',
      addressLine2: '',
      city: '',
      state: '',
      pincode: '',
      isDefault: false,
    },
  });

  // Address autocomplete (Photon/OpenStreetMap via backend) — fills the form
  // fields below, which stay editable. Same UX as the onboarding address step.
  const [addrQuery, setAddrQuery] = useState('');
  const [showAddrSuggestions, setShowAddrSuggestions] = useState(false);
  const { data: addrSuggestions = [], isFetching: addrSearching } =
    useAddressAutocomplete(addrQuery);

  function pickAddrSuggestion(s: AddressSuggestion): void {
    setAddrValue('addressLine1', s.line1 || s.description, { shouldValidate: true });
    if (s.city) setAddrValue('city', s.city, { shouldValidate: true });
    if (s.region) setAddrValue('state', s.region, { shouldValidate: true });
    if (s.postal) setAddrValue('pincode', s.postal, { shouldValidate: true });
    setAddrQuery('');
    setShowAddrSuggestions(false);
    Keyboard.dismiss();
  }

  async function onSaveAddress(values: AddressFormValues) {
    try {
      const result = await createAddress.mutateAsync({
        label: values.label,
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
        items: cartStore.items.map((i) => ({
          menuItemId: i.menuItemId,
          quantity: i.quantity,
          notes: i.instructions?.trim() || undefined,
        })),
        deliveryAddressId: selectedAddressId,
        specialInstructions: note.trim() || undefined,
      });

      const orderId = orderResult.data.id;

      // Step 2: Create Razorpay payment order (server-side)
      const paymentResp = await api.post<{ data: RazorpayPaymentData }>(
        `/v1/payments/order/${orderId}/create`,
        {}
      );
      const paymentData = paymentResp.data.data ?? (paymentResp.data as unknown as RazorpayPaymentData);

      // Step 3: Hand off to the in-app Razorpay checkout (WebView). That screen
      // opens the payment sheet, verifies the result server-side, and routes to
      // /payment/result. Clear our loading state so checkout is usable if the
      // user backs out of the payment sheet.
      setIsLoading(false);
      router.push({
        pathname: '/payment/checkout',
        params: {
          orderId,
          razorpayOrderId: paymentData.razorpayOrderId,
          razorpayKeyId: paymentData.razorpayKeyId,
          amount: String(paymentData.amount),
          currency: paymentData.currency ?? 'INR',
          name: paymentData.prefill?.name ?? '',
          email: paymentData.prefill?.email ?? '',
          phone: paymentData.prefill?.phone ?? '',
        },
      });
    } catch (err: unknown) {
      // Surface the real reason in a modal — the inline banner sits in the
      // scroll body, far from the sticky button, so a failed tap otherwise
      // reads as "nothing happened" (e.g. the delivery-zone coordinate gate).
      const message = friendlyErrorMessage(err, 'Order creation failed. Please try again.');
      setError(message);
      Alert.alert('Could not place order', message);
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
      className="flex-1 bg-canvas"
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      {/* Header */}
      <View className="flex-row items-center bg-canvas border-b border-hairline px-4 pt-14 pb-4 gap-3">
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          className="p-1"
        >
          <ChevronLeft size={24} color="#222222" />
        </Pressable>
        <Text className="text-xl font-semibold text-charcoal flex-1">Checkout</Text>
      </View>

      <ScrollView
        contentContainerStyle={{ paddingBottom: 120 }}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}
      >
        {/* ── Delivery Address ── */}
        <View className="mx-4 mt-4 bg-canvas rounded-2xl overflow-hidden border border-hairline">
          <View className="flex-row items-center px-4 pt-4 pb-2 gap-2">
            <MapPin size={18} color="#FF385C" />
            <Text className="text-base font-semibold text-charcoal">Delivery Address</Text>
          </View>

          {addressLoading ? (
            <View className="px-4 pb-4">
              <ActivityIndicator size="small" color="#FF385C" />
            </View>
          ) : (
            <>
              {addresses.map((addr) => (
                <Pressable
                  key={addr.id}
                  onPress={() => addr.id && setSelectedAddressId(addr.id)}
                  accessibilityRole="radio"
                  accessibilityState={{ checked: selectedAddressId === addr.id }}
                  accessibilityLabel={formatAddress(addr)}
                >
                  <View
                    className={`mx-4 mb-2 p-3 rounded-xl border ${
                      selectedAddressId === addr.id
                        ? 'border-coral bg-coral-tint'
                        : 'border-hairline bg-canvas'
                    }`}
                  >
                    <View className="flex-row items-start gap-3">
                      <View
                        className={`w-4 h-4 rounded-full border-2 mt-0.5 ${
                          selectedAddressId === addr.id
                            ? 'border-coral bg-coral'
                            : 'border-hairline bg-surface-soft'
                        }`}
                      />
                      <View className="flex-1">
                        {(addr.label || addr.isDefault) && (
                          <View className="flex-row items-center gap-2 mb-1">
                            {addr.label ? (
                              <View className="bg-surface-soft rounded-md px-2 py-0.5">
                                <Text className="text-[11px] font-semibold text-charcoal">
                                  {addr.label}
                                </Text>
                              </View>
                            ) : null}
                            {addr.isDefault ? (
                              <Text className="text-xs text-coral font-medium">Default</Text>
                            ) : null}
                          </View>
                        )}
                        <Text className="text-sm text-charcoal">{formatAddress(addr)}</Text>
                      </View>
                    </View>
                  </View>
                </Pressable>
              ))}

              {/* Add new address toggle */}
              <Pressable
                onPress={() => setShowAddressForm((prev: boolean) => !prev)}
                accessibilityRole="button"
                accessibilityLabel="Add new address"
              >
                <View className="flex-row items-center gap-2 px-4 py-3">
                  <Plus size={16} color="#FF385C" />
                  <Text className="text-sm text-coral font-medium">Add New Address</Text>
                </View>
              </Pressable>

              {showAddressForm && (
                <View className="px-4 pb-4 gap-3">
                  {/* Label selector — Home / Work / Other */}
                  <Controller
                    control={addrControl}
                    name="label"
                    render={({ field: { onChange, value } }) => (
                      <AddressLabelSelect value={value} onChange={onChange} />
                    )}
                  />
                  {/* Address autocomplete — fills the fields below (editable) */}
                  <View className="flex-row items-center bg-surface-soft rounded-xl px-3 gap-2">
                    <Search size={16} color={'#717171'} />
                    <TextInput
                      className="flex-1 h-11 text-sm text-charcoal"
                      placeholder="Search for your address"
                      placeholderTextColor="#717171"
                      value={addrQuery}
                      onChangeText={(t) => {
                        setAddrQuery(t);
                        setShowAddrSuggestions(true);
                      }}
                      autoCapitalize="words"
                      autoCorrect={false}
                      returnKeyType="search"
                      accessibilityLabel="Search for your address"
                    />
                    {addrSearching ? (
                      <ActivityIndicator size="small" color={'#FF385C'} />
                    ) : null}
                  </View>
                  {showAddrSuggestions && addrSuggestions.length > 0 && (
                    <View className="bg-surface border border-hairline rounded-xl overflow-hidden">
                      {addrSuggestions.map((s, i) => (
                        <Pressable
                          key={`${s.description}-${i}`}
                          onPress={() => pickAddrSuggestion(s)}
                          accessibilityRole="button"
                          accessibilityLabel={`Use address ${s.description}`}
                        >
                          {({ pressed }) => (
                            <View
                              className={`flex-row items-start gap-2 px-3 py-2.5 ${
                                pressed ? 'bg-surface-soft' : ''
                              } ${i < addrSuggestions.length - 1 ? 'border-b border-hairline' : ''}`}
                            >
                              <MapPin size={16} color={'#717171'} style={{ marginTop: 2 }} />
                              <Text className="flex-1 text-sm text-charcoal leading-5" numberOfLines={2}>
                                {s.description}
                              </Text>
                            </View>
                          )}
                        </Pressable>
                      ))}
                    </View>
                  )}
                  <Controller
                    control={addrControl}
                    name="addressLine1"
                    render={({ field: { onChange, value } }) => (
                      <View>
                        <TextInput
                          value={value}
                          onChangeText={onChange}
                          placeholder="Address line 1 *"
                          placeholderTextColor="#717171"
                          className="bg-surface-soft rounded-xl px-3 py-2.5 text-sm text-charcoal"
                          accessibilityLabel="Address line 1"
                        />
                        {addrErrors.addressLine1 && (
                          <Text className="text-xs text-destructive mt-1">
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
                        placeholder="Flat / House / Floor no. (optional)"
                        placeholderTextColor="#717171"
                        className="bg-surface-soft rounded-xl px-3 py-2.5 text-sm text-charcoal"
                        accessibilityLabel="Flat, house or floor number"
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
                            placeholderTextColor="#717171"
                            className="bg-surface-soft rounded-xl px-3 py-2.5 text-sm text-charcoal"
                            accessibilityLabel="City"
                          />
                          {addrErrors.city && (
                            <Text className="text-xs text-destructive mt-1">
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
                            placeholderTextColor="#717171"
                            className="bg-surface-soft rounded-xl px-3 py-2.5 text-sm text-charcoal"
                            accessibilityLabel="State"
                          />
                          {addrErrors.state && (
                            <Text className="text-xs text-destructive mt-1">
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
                          placeholderTextColor="#717171"
                          keyboardType="numeric"
                          maxLength={6}
                          className="bg-surface-soft rounded-xl px-3 py-2.5 text-sm text-charcoal"
                          accessibilityLabel="Pincode"
                        />
                        {addrErrors.pincode && (
                          <Text className="text-xs text-destructive mt-1">
                            {addrErrors.pincode.message}
                          </Text>
                        )}
                      </View>
                    )}
                  />
                  {/* Save address button — coral filled, iOS Pressable inner-View pattern */}
                  <Pressable
                    onPress={handleAddrSubmit(onSaveAddress)}
                    disabled={addrSubmitting}
                    accessibilityRole="button"
                    accessibilityLabel="Save address"
                  >
                    <View
                      className={`rounded-xl py-2.5 items-center ${
                        addrSubmitting ? 'bg-coral-tint' : 'bg-coral'
                      }`}
                    >
                      {addrSubmitting ? (
                        <ActivityIndicator size="small" color="#FF385C" />
                      ) : (
                        <Text className="text-canvas font-semibold text-sm">Save Address</Text>
                      )}
                    </View>
                  </Pressable>
                </View>
              )}
            </>
          )}
        </View>

        {/* ── Order Summary ── */}
        <View className="mx-4 mt-4 bg-canvas rounded-2xl overflow-hidden border border-hairline">
          <Text className="text-base font-semibold text-charcoal px-4 pt-4 pb-2">Order Summary</Text>
          <FlatList
            data={cartStore.items}
            keyExtractor={(item) => item.menuItemId}
            scrollEnabled={false}
            renderItem={({ item }) => (
              <View className="flex-row items-center px-4 py-2 gap-2">
                {/* Quantity chip — coral-tint bg, coral text */}
                <View className="w-6 h-6 rounded-full bg-coral-tint items-center justify-center">
                  <Text className="text-xs font-medium text-coral">{item.quantity}</Text>
                </View>
                <Text className="flex-1 text-sm text-charcoal">{item.name}</Text>
                <Text className="text-sm font-medium text-charcoal" style={{ fontVariant: ['tabular-nums'] }}>
                  ₹{(item.price * item.quantity).toFixed(2)}
                </Text>
              </View>
            )}
            ListEmptyComponent={
              <View className="px-4 py-4">
                <Text className="text-sm text-charcoal-soft">Your cart is empty.</Text>
              </View>
            }
          />
          {/* Price breakdown */}
          <View className="border-t border-hairline mx-4 mt-2 pt-3 pb-4 gap-2">
            <View className="flex-row justify-between">
              <Text className="text-sm text-charcoal-soft">Subtotal</Text>
              <Text className="text-sm text-charcoal" style={{ fontVariant: ['tabular-nums'] }}>
                ₹{subtotal.toFixed(2)}
              </Text>
            </View>
            <View className="flex-row justify-between">
              <Text className="text-sm text-charcoal-soft">Delivery fee</Text>
              {/* "Free" in success green per spec */}
              <Text className="text-sm text-success font-medium">Free</Text>
            </View>
            <View className="flex-row justify-between pt-1 border-t border-hairline">
              <Text className="text-base font-medium text-charcoal">Total</Text>
              <Text className="text-base font-medium text-charcoal" style={{ fontVariant: ['tabular-nums'] }}>
                ₹{total.toFixed(2)}
              </Text>
            </View>
          </View>
        </View>

        {/* ── Payment & delivery (concise; detail behind policy links) ── */}
        {/* CW-01d / RBI PA MD §8: keeps the required PA + refund-window disclosure
            as one line; full cancellation rules live behind the Refund Policy link. */}
        <View className="mx-4 mt-4 bg-canvas rounded-2xl border border-hairline p-4 gap-2.5">
          <View className="flex-row items-start gap-2">
            <Clock size={16} color="#FF385C" style={{ marginTop: 1 }} />
            <Text className="text-sm text-charcoal-soft flex-1 leading-5">
              Estimated delivery 30–45 min after the chef accepts.
            </Text>
          </View>
          <Text className="text-xs text-charcoal-soft leading-5">
            Payments secured by Razorpay (RBI-licensed). Refunds reach your original
            payment method within 7 working days. See{' '}
            {/* Terms/refund links: coral, no underline per spec */}
            <Link href={'/refund' as Href} className="text-coral">
              Refund Policy
            </Link>{' '}
            and{' '}
            <Link href={'/terms' as Href} className="text-coral">
              Terms
            </Link>
            .
          </Text>
        </View>

        {/* ── T&C + Refund consent (CW-01d) ── */}
        <View className="mx-4 mt-4">
          {/* iOS Pressable inner-View pattern — visual styles on the inner View */}
          <Pressable
            onPress={() => setAcceptedTerms((prev: boolean) => !prev)}
            accessibilityRole="checkbox"
            accessibilityState={{ checked: acceptedTerms }}
            accessibilityLabel="I agree to the Terms of Service and Refund Policy for this order"
          >
            <View className="flex-row items-start gap-3 bg-surface-soft rounded-2xl p-4">
              <View
                className={`w-5 h-5 rounded border-2 items-center justify-center mt-0.5 ${
                  acceptedTerms ? 'border-coral bg-coral' : 'border-hairline bg-canvas'
                }`}
              >
                {acceptedTerms && <Check size={14} color="#FFFFFF" />}
              </View>
              <Text className="flex-1 text-sm text-charcoal-soft leading-5">
                I agree to the{' '}
                <Link href={'/terms' as Href} className="text-coral">
                  Terms of Service
                </Link>{' '}
                and{' '}
                <Link href={'/refund' as Href} className="text-coral">
                  Refund Policy
                </Link>{' '}
                for this order.
              </Text>
            </View>
          </Pressable>
        </View>

        {/* ── Note (optional) ── */}
        <View className="mx-4 mt-4 bg-canvas rounded-2xl border border-hairline p-4">
          <Text className="text-sm font-medium text-charcoal-soft mb-2">Note to chef (optional)</Text>
          <TextInput
            value={note}
            onChangeText={setNote}
            placeholder="Any special instructions..."
            placeholderTextColor="#717171"
            multiline
            numberOfLines={2}
            maxLength={200}
            className="bg-surface-soft rounded-xl px-3 py-2.5 text-sm text-charcoal min-h-[60px]"
            accessibilityLabel="Note to chef"
            textAlignVertical="top"
          />
        </View>

        {/* ── Error ── */}
        {error && (
          <View className="mx-4 mt-4 bg-destructive-tint border border-destructive/30 rounded-xl px-4 py-3">
            <Text className="text-sm text-destructive">{error}</Text>
            <Pressable
              onPress={() => setError(null)}
              className="mt-1"
              accessibilityRole="button"
              accessibilityLabel="Dismiss error"
            >
              <Text className="text-xs text-destructive underline">Dismiss</Text>
            </Pressable>
          </View>
        )}
      </ScrollView>

      {/* ── Place Order button — sticky bottom, coral filled, radius 8, 52pt, tabular total ── */}
      <View
        style={{
          shadowColor: '#000',
          shadowOffset: { width: 0, height: -2 },
          shadowOpacity: 0.06,
          shadowRadius: 6,
          elevation: 4,
        }}
        className="absolute bottom-0 left-0 right-0 bg-canvas border-t border-hairline px-4 pt-3 pb-8"
      >
        {/* iOS Pressable inner-View pattern */}
        <Pressable
          onPress={handlePlaceOrder}
          disabled={!canPlaceOrder}
          accessibilityRole="button"
          accessibilityLabel="Place Order"
          accessibilityState={{ disabled: !canPlaceOrder }}
        >
          <View
            className={`w-full rounded-lg items-center justify-center flex-row gap-2 ${
              canPlaceOrder ? 'bg-coral' : 'bg-surface-soft'
            }`}
            style={{ minHeight: 52 }}
          >
            {isLoading ? (
              <ActivityIndicator size="small" color="#FFFFFF" />
            ) : (
              <Text
                className={`text-base font-semibold ${canPlaceOrder ? 'text-canvas' : 'text-charcoal-soft'}`}
                style={canPlaceOrder ? { fontVariant: ['tabular-nums'] } : undefined}
              >
                {isLoading ? 'Processing...' : `Place Order · ₹${total.toFixed(2)}`}
              </Text>
            )}
          </View>
        </Pressable>
      </View>
    </KeyboardAvoidingView>
  );
}
