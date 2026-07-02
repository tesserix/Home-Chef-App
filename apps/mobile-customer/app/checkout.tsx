// Checkout screen — address selection, order summary, then Razorpay payment.
//
// This screen creates the order, then calls startOrderPayment (lib/payment),
// which opens the NATIVE Razorpay checkout sheet (react-native-razorpay) and
// routes to /payment/result. No WebView, no visible web page load.

import { useEffect, useRef, useState } from 'react';
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
import { AlertTriangle, Check, ChevronLeft, Clock, MapPin, Plus, Search } from 'lucide-react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useCartStore } from '../store/cart-store';
import { useCreateOrder } from '../hooks/useOrderCheckout';
import { useChef } from '../hooks/useChefs';
import { useCustomerCoords } from '../hooks/useCustomerCoords';
import { useValidatePromo, promoErrorMessage, type PromoValidationResult } from '../hooks/usePromoCode';
import { useWinback } from '../hooks/useWinback';
import { useDeliverySlots, type DeliverySlot } from '../hooks/useDeliverySlots';
import { useDietaryCheck } from '../hooks/useDietaryConflicts';
import { useWallet } from '../hooks/useWallet';
import { useAddresses, useCreateAddress } from '../hooks/useAddresses';
import {
  useAddressAutocomplete,
  type AddressSuggestion,
} from '../hooks/useLocations';
import { startOrderPayment } from '../lib/payment';
import { friendlyErrorMessage } from '../lib/errors';
import { useFormDraft } from '@homechef/mobile-shared/hooks';
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

// Wallet-at-checkout (#141) is gated to match the API's WALLET_CHECKOUT_ENABLED;
// the toggle stays hidden until both the app build and the server enable it.
const WALLET_CHECKOUT_ENABLED = process.env.EXPO_PUBLIC_WALLET_CHECKOUT_ENABLED === 'true';

// slotDayLabel turns a "YYYY-MM-DD" slot date into a human label relative to the
// device's today ("Today" / "Tomorrow" / "Mon, 22 Jun") for the slot picker (#51).
function slotDayLabel(dateStr: string): string {
  const d = new Date(`${dateStr}T00:00:00`);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const diff = Math.round((d.getTime() - today.getTime()) / 86_400_000);
  if (diff <= 0) return 'Today';
  if (diff === 1) return 'Tomorrow';
  return d.toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' });
}

// ─── Component ────────────────────────────────────────────────────────────────

export default function CheckoutScreen() {
  const cartStore = useCartStore();
  const createOrder = useCreateOrder();
  const coords = useCustomerCoords();
  const { data: chefData } = useChef(cartStore.chefId ?? '', coords ?? undefined);
  const offersPickup = !!chefData?.data?.offersPickup;
  // Whether the customer can pick "Delivery" at all. Server-computed = the chef
  // self-delivers OR a 3PL provider is live. With 3PL dark and a non-self-
  // delivering chef this is false, so we offer pickup only (no unfulfillable
  // delivery order). Default true so an older API without the field still works.
  const offersDelivery = chefData?.data?.offersDelivery ?? true;
  // Delivery-area reach: even if the chef offers delivery, a self-delivering
  // chef the customer is outside the radius of comes back deliverableToYou=false
  // — the app must offer pickup only. Undefined (no coords) keeps delivery
  // available; the server order guard is the final backstop.
  const deliverableHere = chefData?.data?.deliverableToYou !== false;
  const deliveryAvailable = offersDelivery && deliverableHere;
  const createAddress = useCreateAddress();
  const { data: addressData, isLoading: addressLoading } = useAddresses();
  const { data: wallet } = useWallet();
  const validatePromo = useValidatePromo();

  // Promo code (#39) — validated server-side; we keep the previewed result and
  // pass the code to CreateOrder, which re-validates and computes the real discount.
  const [promoInput, setPromoInput] = useState('');
  const [appliedPromo, setAppliedPromo] = useState<PromoValidationResult | null>(null);

  // Auto-prefill the promo field with the customer's active win-back code (#42)
  // so the offer is one tap from applied. Only fills an empty field.
  const { data: winback } = useWinback();
  useEffect(() => {
    if (winback?.code) {
      setPromoInput((cur) => (cur ? cur : winback.code));
    }
  }, [winback?.code]);
  const [promoError, setPromoError] = useState<string | null>(null);

  const addresses = addressData?.data ?? [];

  const [fulfillment, setFulfillment] = useState<
    'delivery' | 'pickup' | 'chef_delivery'
  >('delivery');
  // The customer only chooses delivery vs pickup. WHO delivers (the chef
  // themselves vs a 3PL rider) is the chef's decision, resolved server-side
  // from the chef's "I deliver myself" setting — never a customer choice.
  // Delivery only appears when it's actually fulfillable AND reaches the
  // customer (deliveryAvailable = offersDelivery && within the chef's range).
  const fulfillmentModes: Array<'delivery' | 'pickup' | 'chef_delivery'> = [
    ...(deliveryAvailable ? (['delivery'] as const) : []),
    ...(offersPickup ? (['pickup'] as const) : []),
  ];
  const fulfillmentLabel: Record<typeof fulfillment, string> = {
    delivery: 'Delivery',
    chef_delivery: 'Chef delivery',
    pickup: 'Pickup',
  };
  // Once the chef loads, snap the selection to an available mode. Guards the case
  // where the chef can't deliver (3PL dark + no self-delivery) so we never post an
  // unfulfillable delivery order — the customer sees pickup pre-selected instead.
  useEffect(() => {
    const available: Array<'delivery' | 'pickup'> = [
      ...(deliveryAvailable ? (['delivery'] as const) : []),
      ...(offersPickup ? (['pickup'] as const) : []),
    ];
    if (
      available.length > 0 &&
      !available.includes(fulfillment as 'delivery' | 'pickup')
    ) {
      setFulfillment(available[0]);
    }
  }, [deliveryAvailable, offersPickup, fulfillment]);
  const [selectedAddressId, setSelectedAddressId] = useState<string>('');
  const [applyWallet, setApplyWallet] = useState(false);
  const [note, setNote] = useState('');
  // Persist the optional note-to-chef so it survives a background/kill (the
  // cart itself is already persisted by cart-store). Cleared once the order is
  // created, which consumes the note as specialInstructions.
  const {
    ready: noteReady,
    draft: noteDraft,
    saveDraft: saveNoteDraft,
    clearDraft: clearNoteDraft,
  } = useFormDraft<string>('checkout-note');
  const noteRestored = useRef(false);
  useEffect(() => {
    if (!noteReady || noteRestored.current || noteDraft == null) return;
    noteRestored.current = true;
    setNote(noteDraft);
  }, [noteReady, noteDraft]);
  useEffect(() => {
    if (!noteReady) return;
    saveNoteDraft(note);
  }, [noteReady, note, saveNoteDraft]);
  // Scheduled delivery slot (#51) — null = ASAP. Only shown when the chef
  // offers slots; the chosen slot+date ride along on the create-order payload.
  const { data: slotsData } = useDeliverySlots(cartStore.chefId ?? undefined);
  const [selectedSlot, setSelectedSlot] = useState<{ slot: string; date: string } | null>(null);
  const availableSlots = (slotsData?.slots ?? []).filter((s) => s.available);

  // Dietary & allergen conflict warning (#41) — checks the cart's items against
  // the customer's saved profile server-side. Non-blocking.
  const cartItemIds = cartStore.items.map((i) => i.menuItemId);
  const { data: dietaryCheck } = useDietaryCheck(cartItemIds);
  const dietaryWarnings = dietaryCheck?.warnings ?? [];
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
    const needsAddress = fulfillment !== 'pickup';
    if ((needsAddress && !selectedAddressId) || cartStore.items.length === 0 || !cartStore.chefId) return;
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
          // Selected add-on option ids for this line (#232).
          modifierOptionIds: i.modifiers?.map((m) => m.optionId),
        })),
        deliveryAddressId: fulfillment === 'pickup' ? undefined : selectedAddressId,
        fulfillmentType: fulfillment,
        specialInstructions: note.trim() || undefined,
        deliverySlot: selectedSlot?.slot,
        deliveryDate: selectedSlot?.date,
        promoCode: appliedPromo?.code,
      });

      const orderId = orderResult.data.id;
      // Order created — the note is now persisted server-side; drop the backup.
      clearNoteDraft();

      // Steps 2–3: create the Razorpay payment and open the NATIVE checkout
      // sheet. startOrderPayment handles the full-wallet (already-paid) shortcut
      // and routes to /payment/result, which polls the authoritative payment
      // status. Clear loading first so the screen stays usable if the user
      // dismisses the sheet.
      setIsLoading(false);
      await startOrderPayment(orderId, { walletAmount: walletApplied });
    } catch (err: unknown) {
      // Surface the real reason in a modal — the inline banner sits in the
      // scroll body, far from the sticky button, so a failed tap otherwise
      // reads as "nothing happened" (e.g. the delivery-zone coordinate gate).
      const message = friendlyErrorMessage(err, 'Order creation failed. Please try again.');
      // If the promo was rejected at order time (e.g. exhausted since applying),
      // drop it so the retry isn't blocked by a dead code (#39).
      if (/promo/i.test(message)) {
        removePromo();
      }
      setError(message);
      Alert.alert('Could not place order', message);
      setIsLoading(false);
    }
  }

  // ─── Helpers ──────────────────────────────────────────────────────────────

  const canPlaceOrder =
    cartStore.items.length > 0 &&
    (fulfillment === 'pickup' || !!selectedAddressId) &&
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
  // Promo discount (#39) — server-validated preview, clamped to the subtotal.
  const discount = appliedPromo ? Math.min(appliedPromo.discount, subtotal) : 0;
  const total = Math.max(0, subtotal + deliveryFee - discount);

  // Wallet store-credit applied at checkout (#141). Apply as much as the balance
  // and total allow; the remaining payable is what the gateway charges.
  const walletBalance = wallet?.balance ?? 0;
  const walletAvailable = WALLET_CHECKOUT_ENABLED && walletBalance > 0;
  const walletApplied = applyWallet ? Math.min(walletBalance, total) : 0;
  const payable = Math.max(0, total - walletApplied);

  async function applyPromo() {
    const code = promoInput.trim();
    if (!code || !cartStore.chefId) return;
    setPromoError(null);
    try {
      const result = await validatePromo.mutateAsync({ code, orderTotal: subtotal, chefId: cartStore.chefId });
      setAppliedPromo(result);
    } catch (err) {
      setAppliedPromo(null);
      setPromoError(promoErrorMessage(err));
    }
  }

  function removePromo() {
    setAppliedPromo(null);
    setPromoInput('');
    setPromoError(null);
  }

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
        {/* ── Fulfillment mode selector (shown when the chef offers more than
            just 3PL delivery — i.e. pickup and/or self-delivery) ── */}
        {fulfillmentModes.length > 1 ? (
          <View className="mx-4 mt-4 flex-row gap-2" accessibilityRole="radiogroup">
            {fulfillmentModes.map((mode) => (
              <Pressable
                key={mode}
                onPress={() => setFulfillment(mode)}
                accessibilityRole="radio"
                accessibilityState={{ checked: fulfillment === mode }}
                className={`flex-1 min-h-[44px] items-center justify-center rounded-lg border px-2 ${
                  fulfillment === mode
                    ? 'border-coral bg-coral-tint'
                    : 'border-hairline bg-canvas'
                }`}
              >
                <Text
                  className={`text-sm font-semibold ${
                    fulfillment === mode ? 'text-coral-pressed' : 'text-charcoal-soft'
                  }`}
                  numberOfLines={1}
                >
                  {fulfillmentLabel[mode]}
                </Text>
              </Pressable>
            ))}
          </View>
        ) : null}

        {/* ── Delivery Address ── */}
        {fulfillment !== 'pickup' ? (
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
        ) : null}

        {/* ── Order Summary ── */}
        <View className="mx-4 mt-4 bg-canvas rounded-2xl overflow-hidden border border-hairline">
          <Text className="text-base font-semibold text-charcoal px-4 pt-4 pb-2">Order Summary</Text>
          <FlatList
            data={cartStore.items}
            keyExtractor={(item) => item.lineId}
            scrollEnabled={false}
            renderItem={({ item }) => (
              <View className="flex-row items-start px-4 py-2 gap-2">
                {/* Quantity chip — coral-tint bg, coral text */}
                <View className="w-6 h-6 rounded-full bg-coral-tint items-center justify-center mt-0.5">
                  <Text className="text-xs font-medium text-coral">{item.quantity}</Text>
                </View>
                <View className="flex-1">
                  <Text className="text-sm text-charcoal">{item.name}</Text>
                  {/* Selected add-ons (#232) */}
                  {item.modifiers && item.modifiers.length > 0 ? (
                    <Text className="text-xs text-charcoal-soft">
                      {item.modifiers.map((m) => m.optionName).join(', ')}
                    </Text>
                  ) : null}
                </View>
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

            {/* Promo code (#39) */}
            {appliedPromo ? (
              <View className="flex-row items-center justify-between">
                <View className="flex-row items-center gap-2">
                  <View className="rounded bg-coral/10 px-2 py-0.5">
                    <Text className="text-xs font-semibold text-coral">{appliedPromo.code}</Text>
                  </View>
                  <Pressable onPress={removePromo} accessibilityRole="button" accessibilityLabel="Remove promo code">
                    <Text className="text-xs text-charcoal-soft underline">Remove</Text>
                  </Pressable>
                </View>
                <Text className="text-sm text-success font-medium" style={{ fontVariant: ['tabular-nums'] }}>
                  −₹{discount.toFixed(2)}
                </Text>
              </View>
            ) : (
              <View className="gap-1">
                <View className="flex-row items-center gap-2">
                  <TextInput
                    value={promoInput}
                    onChangeText={(t) => setPromoInput(t.toUpperCase())}
                    placeholder="Promo code"
                    // #717171 (charcoal-soft) meets AA on canvas; #9CA3AF did not.
                    placeholderTextColor="#717171"
                    autoCapitalize="characters"
                    autoCorrect={false}
                    accessibilityLabel="Promo code"
                    className="flex-1 rounded-lg border border-hairline bg-canvas px-3 py-2 text-sm text-charcoal"
                  />
                  <Pressable
                    onPress={applyPromo}
                    disabled={!promoInput.trim() || validatePromo.isPending}
                    accessibilityRole="button"
                    accessibilityLabel="Apply promo code"
                    className={`rounded-lg px-4 py-2 ${
                      !promoInput.trim() || validatePromo.isPending ? 'bg-hairline' : 'bg-coral'
                    }`}
                  >
                    <Text className={`text-sm font-semibold ${!promoInput.trim() || validatePromo.isPending ? 'text-charcoal-soft' : 'text-canvas'}`}>
                      {validatePromo.isPending ? '…' : 'Apply'}
                    </Text>
                  </Pressable>
                </View>
                {promoError ? <Text className="text-xs text-paprika">{promoError}</Text> : null}
              </View>
            )}

            {/* Wallet store-credit toggle (#141) — only when the feature is live
                and the customer has a balance. */}
            {walletAvailable && (
              <Pressable
                onPress={() => setApplyWallet((v) => !v)}
                accessibilityRole="switch"
                accessibilityState={{ checked: applyWallet }}
                accessibilityLabel="Apply wallet credit"
                className="flex-row items-center justify-between"
              >
                <View className="flex-row items-center gap-2">
                  <View
                    className={`h-5 w-5 items-center justify-center rounded border ${
                      applyWallet ? 'border-coral bg-coral' : 'border-hairline bg-canvas'
                    }`}
                  >
                    {applyWallet && <Check size={14} color="#FFFFFF" />}
                  </View>
                  <Text className="text-sm text-charcoal">
                    Use wallet credit (₹{walletBalance.toFixed(2)})
                  </Text>
                </View>
                {applyWallet && (
                  <Text className="text-sm text-success font-medium" style={{ fontVariant: ['tabular-nums'] }}>
                    −₹{walletApplied.toFixed(2)}
                  </Text>
                )}
              </Pressable>
            )}

            <View className="flex-row justify-between pt-1 border-t border-hairline">
              <Text className="text-base font-medium text-charcoal">
                {walletApplied > 0 ? 'To pay' : 'Total'}
              </Text>
              <Text className="text-base font-medium text-charcoal" style={{ fontVariant: ['tabular-nums'] }}>
                ₹{payable.toFixed(2)}
              </Text>
            </View>
          </View>
        </View>

        {/* ── Dietary / allergen conflict warning (#41) ── */}
        {dietaryWarnings.length > 0 && (
          <View className="mx-4 mt-4 bg-destructive-tint border border-destructive/30 rounded-2xl p-4">
            <View className="flex-row items-center gap-2 mb-1">
              <AlertTriangle size={16} color="#B22B0E" />
              <Text className="text-sm font-semibold text-destructive">Check your order</Text>
            </View>
            <Text className="text-sm text-destructive leading-5 mb-2">
              Some items may not match your dietary profile:
            </Text>
            {dietaryWarnings.map((w) => (
              <Text key={w.menuItemId} className="text-sm text-destructive leading-5">
                • {w.name} — {w.conflicts.map((cf) => cf.detail).join(', ')}
              </Text>
            ))}
            <Text className="text-xs text-charcoal-soft mt-2 leading-5">
              You can still place this order. Review your items or update your dietary profile in
              your account.
            </Text>
          </View>
        )}

        {/* ── Delivery time / scheduled slot (#51) ── */}
        {slotsData?.slotsEnabled && (
          <View className="mx-4 mt-4 bg-canvas rounded-2xl border border-hairline p-4">
            <Text className="text-sm font-medium text-charcoal-soft mb-3">Delivery time</Text>
            <View className="flex-row flex-wrap gap-2">
              {/* ASAP (default) */}
              <Pressable
                onPress={() => setSelectedSlot(null)}
                accessibilityRole="radio"
                accessibilityState={{ selected: selectedSlot === null }}
              >
                <View
                  className={`px-3 py-2 rounded-xl border ${
                    selectedSlot === null ? 'border-coral bg-coral-tint' : 'border-hairline bg-surface-soft'
                  }`}
                >
                  <Text className={`text-sm font-medium ${selectedSlot === null ? 'text-coral' : 'text-charcoal'}`}>
                    ASAP
                  </Text>
                  <Text className="text-xs text-charcoal-soft">After chef accepts</Text>
                </View>
              </Pressable>

              {availableSlots.map((s: DeliverySlot) => {
                const sel = selectedSlot?.slot === s.slot && selectedSlot?.date === s.date;
                return (
                  <Pressable
                    key={`${s.date}-${s.slot}`}
                    onPress={() => setSelectedSlot({ slot: s.slot, date: s.date })}
                    accessibilityRole="radio"
                    accessibilityState={{ selected: sel }}
                    accessibilityLabel={`${slotDayLabel(s.date)} ${s.label} ${s.window}`}
                  >
                    <View
                      className={`px-3 py-2 rounded-xl border ${
                        sel ? 'border-coral bg-coral-tint' : 'border-hairline bg-surface-soft'
                      }`}
                    >
                      <Text className={`text-sm font-medium ${sel ? 'text-coral' : 'text-charcoal'}`}>
                        {slotDayLabel(s.date)} · {s.label}
                      </Text>
                      <Text className="text-xs text-charcoal-soft" style={{ fontVariant: ['tabular-nums'] }}>
                        {s.window}
                        {s.remaining != null ? ` · ${s.remaining} left` : ''}
                      </Text>
                    </View>
                  </Pressable>
                );
              })}
            </View>
            {availableSlots.length === 0 && (
              <Text className="text-xs text-charcoal-soft mt-1">
                No delivery windows are open right now — your order will be delivered ASAP.
              </Text>
            )}
          </View>
        )}

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
                {isLoading ? 'Processing...' : `Place Order · ₹${payable.toFixed(2)}`}
              </Text>
            )}
          </View>
        </Pressable>
      </View>
    </KeyboardAvoidingView>
  );
}
