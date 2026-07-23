// Checkout screen — address selection, order summary, then Razorpay payment.
//
// This screen creates the order, then calls startOrderPayment (lib/payment),
// which opens the NATIVE Razorpay checkout sheet (react-native-razorpay) and
// routes to /payment/result. No WebView, no visible web page load.

import { useEffect, useMemo, useRef, useState, type RefObject } from 'react';
import {
  AccessibilityInfo,
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
  findNodeHandle,
} from 'react-native';
import Animated, {
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withTiming,
} from 'react-native-reanimated';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Link, router, type Href } from 'expo-router';
import * as Haptics from 'expo-haptics';
import {
  AlertTriangle,
  Check,
  ChevronDown,
  ChevronLeft,
  Clock,
  MapPin,
  Plus,
  Search,
} from 'lucide-react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { customerColors, customerTheme } from '@homechef/mobile-shared/theme';
import { useCartStore } from '../store/cart-store';
import { useCreateOrder } from '../hooks/useOrderCheckout';
import { useChef } from '../hooks/useChefs';
import { useCustomerCoords } from '../hooks/useCustomerCoords';
import { useDeliveryQuote } from '../hooks/useDeliveryQuote';
import { useValidatePromo, promoErrorMessage, type PromoValidationResult } from '../hooks/usePromoCode';
import { useWinback } from '../hooks/useWinback';
import { useDeliverySlots, type DeliverySlot } from '../hooks/useDeliverySlots';
import { useFulfillmentTimes } from '../hooks/useFulfillmentTimes';
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

// Android ripple tints — translucent colours derived from existing tokens
// (never a new literal colour), matching the ChefCard/MenuItemCard convention.
const CORAL_RIPPLE = `${customerColors.coral.DEFAULT}22`;
const CHARCOAL_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const CANVAS_RIPPLE = `${customerColors.canvas}33`;

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

// R13 (zero-flicker) — component-layer stale-while-revalidating. The delivery
// quote / slots / fulfillment-times queries below don't set
// `placeholderData`/`keepPreviousData` on the hook itself (that's a hook-config
// change, out of this task's scope — see the task report), so switching query
// keys (e.g. picking a different address re-keys the quote) would otherwise
// blank an already-rendered section for a frame. Remembering the last defined
// value and rendering that while the new value loads gets the same effect
// entirely at the component layer, with zero change to the hooks.
function useStaleValue<T>(value: T | undefined): T | undefined {
  const ref = useRef<T | undefined>(undefined);
  if (value !== undefined) {
    ref.current = value;
  }
  return value ?? ref.current;
}

// ─── Component ────────────────────────────────────────────────────────────────

export default function CheckoutScreen() {
  const insets = useSafeAreaInsets();
  const reduceMotion = useReducedMotion();
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
  // Show the Delivery choice whenever the chef offers delivery — the toggle must
  // NOT disappear just because the customer's DEVICE location is out of range
  // (that hid delivery entirely and left only pickup, with no explanation). The
  // actual range check runs against the SELECTED delivery address via the quote
  // (deliveryOutOfRange below): if that address is beyond the kitchen's radius we
  // show an "outside delivery range" banner and block Place Order, and the server
  // 422 is the final backstop. This keeps the choice visible and the reason clear.
  const deliveryAvailable = offersDelivery;
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
  // themselves vs a 3PL rider) is the chef's decision, resolved server-side.
  // Delivery appears whenever the chef offers it; if the SELECTED address is
  // beyond the kitchen's range, the out-of-range banner + Place-Order block
  // (deliveryOutOfRange) handle it — the choice itself stays visible.
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
  const { data: slotsDataFresh } = useDeliverySlots(cartStore.chefId ?? undefined);
  const slotsData = useStaleValue(slotsDataFresh);
  const [selectedSlot, setSelectedSlot] = useState<{ slot: string; date: string } | null>(null);
  const availableSlots = (slotsData?.slots ?? []).filter((s) => s.available);
  // Windowed (restaurant-style) chefs keep the #51 slot picker; everyone else —
  // the home-tiffin default — uses the suggested-time handshake (#709).
  const useSlotPicker = !!slotsData?.slotsEnabled && availableSlots.length > 0;

  // Home-tiffin suggested time (#709): the customer PROPOSES a preferred time and
  // the chef confirms/proposes at accept. null = "as soon as ready" (stays the
  // default hero option — R14).
  const [requestedTime, setRequestedTime] = useState<Date | null>(null);
  // Realistic proposable times come from the server, derived from the CHEF's meal
  // windows + open hours + prep headroom — NOT "now + 1h" (which proposed 9am for a
  // 6am order). Same list for delivery and pickup.
  const { data: fulfillmentTimesDataFresh } = useFulfillmentTimes(cartStore.chefId ?? undefined);
  const fulfillmentTimesData = useStaleValue(fulfillmentTimesDataFresh);
  const fulfillmentTimes = fulfillmentTimesData?.times ?? [];
  // Group suggested times by "day · meal" so the picker reads as a few labelled
  // clusters (Today · Lunch, Today · Dinner …) instead of a wall of chips.
  const fulfillmentTimeGroups = useMemo(() => {
    const groups: { key: string; times: typeof fulfillmentTimes }[] = [];
    for (const t of fulfillmentTimesData?.times ?? []) {
      const key = `${t.day} · ${t.meal}`;
      const existing = groups.find((g) => g.key === key);
      if (existing) existing.times.push(t);
      else groups.push({ key, times: [t] });
    }
    return groups;
  }, [fulfillmentTimesData]);

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
  // Fee breakdown disclosure (R14) — collapsed by default; the itemised
  // self-delivery estimate is secondary detail, not core price transparency
  // (subtotal/delivery/tax/total stay always visible — #fee-transparency).
  const [feeBreakdownOpen, setFeeBreakdownOpen] = useState(false);
  const feeChevronRotation = useSharedValue(0);
  const feeChevronStyle = useAnimatedStyle(() => ({
    transform: [{ rotate: `${feeChevronRotation.value}deg` }],
  }));
  function toggleFeeBreakdown() {
    const next = !feeBreakdownOpen;
    setFeeBreakdownOpen(next);
    feeChevronRotation.value = reduceMotion
      ? next
        ? 180
        : 0
      : withTiming(next ? 180 : 0, { duration: 150 });
  }

  // R14 — scroll the first invalid/incomplete section into view + move
  // accessibility focus to it, instead of a disabled-looking tap doing nothing.
  const scrollRef = useRef<ScrollView>(null);
  const addressSectionRef = useRef<View>(null);
  const termsSectionRef = useRef<View>(null);
  const addressSectionY = useRef(0);
  const termsSectionY = useRef(0);
  function focusSection(sectionRef: RefObject<View | null>, y: number) {
    scrollRef.current?.scrollTo({ y: Math.max(0, y - 16), animated: true });
    const node = sectionRef.current ? findNodeHandle(sectionRef.current) : null;
    if (node) {
      // Slight delay so the scroll settles before VoiceOver/TalkBack grabs focus.
      setTimeout(() => AccessibilityInfo.setAccessibilityFocus(node), 300);
    }
  }

  // Pre-select default address when addresses load. Also covers the
  // only-address case: with exactly one saved address, `addresses[0]` IS that
  // address, so it auto-selects the same way (verified — do not remove).
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

  // Address autocomplete (Mappls + Photon via backend) — fills the form fields
  // below, which stay editable. Same UX as the onboarding address step.
  const [addrQuery, setAddrQuery] = useState('');
  const [showAddrSuggestions, setShowAddrSuggestions] = useState(false);
  // Coordinates of the picked suggestion. WITHOUT these the saved address has no
  // point, so the delivery quote can't measure the distance from the chef and the
  // km / fee never calculate — the onboarding form persists them, so must this.
  const [addrCoords, setAddrCoords] = useState<{ lat: number; lon: number } | null>(null);
  const { data: addrSuggestions = [], isFetching: addrSearching } =
    useAddressAutocomplete(addrQuery);

  function pickAddrSuggestion(s: AddressSuggestion): void {
    setAddrValue('addressLine1', s.line1 || s.description, { shouldValidate: true });
    if (s.city) setAddrValue('city', s.city, { shouldValidate: true });
    if (s.region) setAddrValue('state', s.region, { shouldValidate: true });
    if (s.postal) setAddrValue('pincode', s.postal, { shouldValidate: true });
    setAddrCoords(
      typeof s.lat === 'number' && typeof s.lon === 'number' && (s.lat !== 0 || s.lon !== 0)
        ? { lat: s.lat, lon: s.lon }
        : null,
    );
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
        // Persist the picked drop point so delivery distance/fee can be computed.
        latitude: addrCoords?.lat,
        longitude: addrCoords?.lon,
        isDefault: values.isDefault,
      });
      if (result.data.id) {
        setSelectedAddressId(result.data.id);
      }
      resetAddrForm();
      setAddrCoords(null);
      setShowAddressForm(false);
    } catch {
      // Error displayed inline by mutation state if needed
    }
  }

  // ─── Place Order flow ──────────────────────────────────────────────────────

  async function handlePlaceOrder() {
    const needsAddress = fulfillment !== 'pickup';
    // R14 — guide the customer to what's missing instead of a silent no-op tap:
    // scroll the relevant section into view and move accessibility focus to it.
    if (needsAddress && !selectedAddressId) {
      setError('Please choose a delivery address to continue.');
      focusSection(addressSectionRef, addressSectionY.current);
      return;
    }
    if (cartStore.items.length === 0 || !cartStore.chefId) return;
    if (!acceptedTerms) {
      setError('Please accept the Terms of Service and Refund Policy to continue.');
      focusSection(termsSectionRef, termsSectionY.current);
      return;
    }
    // Guard re-entry: the button is disabled via canPlaceOrder, but a second tap
    // can still land during the async create→pay round-trip before the disabled
    // state applies. Without this a double-tap creates a duplicate order + charge.
    if (isLoading) return;

    setIsLoading(true);
    setError(null);

    // R7 — light haptic the instant the customer commits to paying, fired
    // before the async round-trip starts. Never let a haptics failure (e.g. on
    // a simulator) interrupt the order flow.
    try {
      void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    } catch {
      // no-op — haptics are non-critical
    }

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
        // Home-tiffin suggested time (#709) — the chef confirms/proposes at accept.
        requestedFulfillmentAt: requestedTime?.toISOString(),
        promoCode: appliedPromo?.code,
      });

      const orderId = orderResult.data.id;
      // Order created — the note is now persisted server-side; drop the backup.
      clearNoteDraft();

      // Steps 2–3: create the Razorpay payment and open the NATIVE checkout
      // sheet. startOrderPayment handles the full-wallet (already-paid) shortcut
      // and routes to /payment/result, which polls the authoritative payment
      // status. Keep the button DISABLED through payment (cleared in finally) so a
      // second tap can't create a duplicate order during the create→pay round-trip
      // before the native sheet appears.
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
    } finally {
      // Re-enable only after the whole create→pay flow settles (dismiss/cancel/
      // error). On success startOrderPayment has already routed to /payment/result.
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

  // The delivery fee the SERVER will charge, previewed so checkout shows the real
  // number and pickup's saving (#pickup-incentive). Keyed to the selected drop
  // address so the distance fee re-quotes as it changes. This replaces the old
  // `deliveryFee = 0 // free for v1`, which both hid the fee and could show a
  // "Free" total the server then charged a fee on.
  const selectedAddr = addresses.find((a) => a.id === selectedAddressId);
  const { data: quoteFresh } = useDeliveryQuote(cartStore.chefId ?? undefined, {
    latitude: selectedAddr?.latitude,
    longitude: selectedAddr?.longitude,
    city: selectedAddr?.city,
    state: selectedAddr?.state,
    subtotal,
  });
  const quote = useStaleValue(quoteFresh);
  // Out-of-range guard (#709): the kitchen's delivery has a hard radius (the chef's
  // own range, else the platform's 10 km default). When the selected address is
  // beyond it, delivery isn't possible — block placing and tell the customer why,
  // so they aren't confused by a server rejection. Pickup is always allowed.
  const deliveryOutOfRange =
    fulfillment !== 'pickup' && quote?.rangeKnown === true && quote?.deliverable === false;
  // A delivery order needs a LOCATED address so the 10 km range can be checked —
  // the server hard-blocks a delivery order with no coordinates (#709). Mirror
  // that here: an address without lat/lng can't be delivered to; require the map.
  const deliveryNeedsLocation =
    fulfillment !== 'pickup' &&
    !!selectedAddressId &&
    !(selectedAddr?.latitude && selectedAddr?.longitude);

  // Pickup is always free; delivery is the quoted fee (0 until the quote lands or
  // when delivery is genuinely free).
  const deliveryFee = fulfillment === 'pickup' ? 0 : quote?.deliveryFee ?? 0;
  // What the customer would save by switching to pickup — only real when delivery
  // actually costs something. Drives the incentive nudge; 0 shows nothing.
  const pickupSaving = quote?.pickupSaving ?? 0;
  // Promo discount (#39) — server-validated preview, clamped to the subtotal.
  const discount = appliedPromo ? Math.min(appliedPromo.discount, subtotal) : 0;
  // Platform (service) fee + tax — shown so the total is honest, computed the SAME
  // way CreateOrder does (#fee-transparency). serviceFee is a % of subtotal; tax is
  // the rate on (subtotal+delivery+service−discount), backed out when inclusive.
  const serviceFee = quote?.serviceFee ?? 0;
  const taxRate = quote?.taxRatePercent ?? 0;
  const taxInclusive = quote?.taxInclusive ?? false;
  const taxName = quote?.taxName || 'Tax';
  const taxBase = Math.max(0, subtotal + deliveryFee + serviceFee - discount);
  const tax = taxInclusive ? taxBase - taxBase / (1 + taxRate / 100) : taxBase * (taxRate / 100);
  const total = taxInclusive
    ? Math.max(0, subtotal + deliveryFee + serviceFee - discount)
    : Math.max(0, subtotal + deliveryFee + serviceFee + tax - discount);
  // The Place Order button is live only when the order is placeable AND not an
  // out-of-range delivery (which the server would reject anyway).
  const placeEnabled = canPlaceOrder && !deliveryOutOfRange && !deliveryNeedsLocation;
  // The RN `disabled` prop only hard-blocks the conditions that have no
  // "fix by scrolling" path (loading, empty cart, out-of-range/needs-location —
  // both already surfaced by an always-visible banner). Missing address / unchecked
  // terms stay tappable so handlePlaceOrder's own guards can scroll-to-fix (R14);
  // they never let an order through — same gating as before, just reachable by tap.
  const hardBlocked =
    isLoading ||
    cartStore.items.length === 0 ||
    !cartStore.chefId ||
    deliveryOutOfRange ||
    deliveryNeedsLocation;

  // Wallet store-credit applied at checkout (#141). Apply as much as the balance
  // and total allow; the remaining payable is what the gateway charges.
  const walletBalance = wallet?.balance ?? 0;
  const walletAvailable = WALLET_CHECKOUT_ENABLED && walletBalance > 0;
  const walletApplied = applyWallet ? Math.min(walletBalance, total) : 0;
  const payable = Math.max(0, total - walletApplied);
  // Extracted to a plain local so it stays narrowed to non-undefined inside the
  // fee-breakdown disclosure's nested Pressable render-prop closures below — TS
  // doesn't retain narrowing of an object PROPERTY like `selfDeliveryBreakdown`
  // across a closure boundary, even though `quote` itself is const.
  const selfDeliveryBreakdown = quote?.selfDeliveryBreakdown;

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
      <View
        className="flex-row items-center bg-canvas border-b border-hairline px-4 pb-4 gap-3"
        style={{ paddingTop: insets.top + 12 }}
      >
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={8}
          android_ripple={{ color: CHARCOAL_RIPPLE, borderless: true, radius: 20 }}
        >
          {({ pressed }) => (
            <View
              className="p-1"
              style={pressed && Platform.OS === 'ios' ? { opacity: 0.6 } : undefined}
            >
              <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text className="text-xl font-bold text-charcoal font-display flex-1">Checkout</Text>
      </View>

      <ScrollView
        ref={scrollRef}
        contentContainerStyle={{ paddingBottom: 120 + insets.bottom }}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}
      >
        {/* ── Fulfilment segmented control (shown when the chef offers more than
            just 3PL delivery — i.e. pickup and/or self-delivery) ── */}
        {fulfillmentModes.length > 1 ? (
          <View
            className="mx-4 mt-4 flex-row bg-surface-soft rounded-lg p-1"
            style={{ minHeight: 44 }}
            accessibilityRole="radiogroup"
          >
            {fulfillmentModes.map((mode) => (
              <Pressable
                key={mode}
                onPress={() => setFulfillment(mode)}
                accessibilityRole="radio"
                accessibilityState={{ checked: fulfillment === mode }}
                accessibilityLabel={fulfillmentLabel[mode]}
                className="flex-1"
                android_ripple={{ color: CORAL_RIPPLE, borderless: false }}
              >
                {({ pressed }) => (
                  <View
                    className={`flex-1 items-center justify-center rounded-md ${
                      fulfillment === mode ? 'bg-coral' : 'bg-transparent'
                    } ${pressed && Platform.OS === 'ios' && fulfillment !== mode ? 'opacity-60' : ''}`}
                    style={{ minHeight: 36 }}
                  >
                    <Text
                      className={`text-sm font-semibold ${
                        fulfillment === mode ? 'text-canvas' : 'text-charcoal-soft'
                      }`}
                      numberOfLines={1}
                    >
                      {fulfillmentLabel[mode]}
                    </Text>
                  </View>
                )}
              </Pressable>
            ))}
          </View>
        ) : null}

        {/* Out-of-range notice (#709). The selected address is beyond the kitchen's
            delivery radius — delivery can't be placed. Tell the customer the exact
            distance + cap and point them to pickup or a closer address. */}
        {deliveryNeedsLocation ? (
          <View className="mx-4 mt-3 rounded-xl border border-coral/40 bg-coral-tint px-4 py-3">
            <View className="flex-row items-start gap-2">
              <AlertTriangle size={16} color={customerColors.coral.pressed} style={{ marginTop: 1 }} />
              <View className="flex-1">
                <Text className="text-sm font-semibold text-coral-pressed">
                  Address location needed
                </Text>
                <Text className="text-xs text-charcoal-soft mt-0.5 leading-4">
                  This address has no map location, so we can't confirm it's within the kitchen's
                  delivery range. Edit it and pick it on the map,{offersPickup ? ' or switch to pickup.' : ' please.'}
                </Text>
                {offersPickup ? (
                  <Pressable
                    onPress={() => setFulfillment('pickup')}
                    accessibilityRole="button"
                    accessibilityLabel="Switch to pickup"
                    android_ripple={{ color: CANVAS_RIPPLE, borderless: false }}
                  >
                    {({ pressed }) => (
                      <View
                        className={`mt-2 self-start rounded-lg bg-coral px-3 py-1.5 ${
                          pressed && Platform.OS === 'ios' ? 'bg-coral-pressed' : ''
                        }`}
                      >
                        <Text className="text-xs font-semibold text-canvas">Switch to pickup</Text>
                      </View>
                    )}
                  </Pressable>
                ) : null}
              </View>
            </View>
          </View>
        ) : null}

        {deliveryOutOfRange ? (
          <View className="mx-4 mt-3 rounded-xl border border-coral/40 bg-coral-tint px-4 py-3">
            <View className="flex-row items-start gap-2">
              <AlertTriangle size={16} color={customerColors.coral.pressed} style={{ marginTop: 1 }} />
              <View className="flex-1">
                <Text className="text-sm font-semibold text-coral-pressed">
                  Outside delivery range
                </Text>
                <Text className="text-xs text-charcoal-soft mt-0.5 leading-4">
                  This address is {(quote?.distanceKm ?? 0).toFixed(1)} km from the kitchen —
                  beyond the {(quote?.maxRadiusKm ?? 10).toFixed(0)} km delivery range.
                  {offersPickup ? ' Switch to pickup, or' : ' Please'} choose an address closer to
                  the kitchen.
                </Text>
                {offersPickup ? (
                  <Pressable
                    onPress={() => setFulfillment('pickup')}
                    accessibilityRole="button"
                    accessibilityLabel="Switch to pickup"
                    android_ripple={{ color: CANVAS_RIPPLE, borderless: false }}
                  >
                    {({ pressed }) => (
                      <View
                        className={`mt-2 self-start rounded-lg bg-coral px-3 py-1.5 ${
                          pressed && Platform.OS === 'ios' ? 'bg-coral-pressed' : ''
                        }`}
                      >
                        <Text className="text-xs font-semibold text-canvas">Switch to pickup</Text>
                      </View>
                    )}
                  </Pressable>
                ) : null}
              </View>
            </View>
          </View>
        ) : null}

        {/* Pickup incentive (#pickup-incentive). Shown only when the customer is
            on delivery, pickup is actually available, and it would genuinely save
            money — a real, one-tap saving, never a fabricated one. */}
        {fulfillment !== 'pickup' &&
        offersPickup &&
        !deliveryOutOfRange &&
        pickupSaving > 0 ? (
          <Pressable
            onPress={() => setFulfillment('pickup')}
            accessibilityRole="button"
            accessibilityLabel={`Switch to pickup and save ₹${pickupSaving.toFixed(0)}`}
            android_ripple={{ color: CORAL_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                className={`mx-4 mt-3 flex-row items-center justify-between rounded-lg bg-coral-tint px-4 py-3 ${
                  pressed && Platform.OS === 'ios' ? 'opacity-70' : ''
                }`}
              >
                <View className="flex-1 pr-3">
                  <Text className="text-sm font-semibold text-coral-pressed">
                    Pick up & save ₹{pickupSaving.toFixed(0)}
                  </Text>
                  <Text className="text-xs text-charcoal-soft mt-0.5">
                    Collect from the kitchen — no delivery fee.
                  </Text>
                </View>
                <Text className="text-sm font-semibold text-coral-pressed">Switch →</Text>
              </View>
            )}
          </Pressable>
        ) : null}

        {/* ── Delivery Address ── */}
        {fulfillment !== 'pickup' ? (
        <View
          ref={addressSectionRef}
          onLayout={(e) => {
            addressSectionY.current = e.nativeEvent.layout.y;
          }}
          className="bg-canvas border-t border-hairline"
        >
          <View className="flex-row items-center px-4 pt-4 pb-2 gap-2">
            <MapPin size={18} color={customerColors.coral.DEFAULT} />
            <Text className="text-base font-semibold text-charcoal">Delivery Address</Text>
          </View>

          {addressLoading ? (
            <View className="px-4 pb-4">
              <ActivityIndicator size="small" color={customerColors.coral.DEFAULT} />
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
                  android_ripple={{ color: CORAL_RIPPLE, borderless: false }}
                >
                  {({ pressed }) => (
                    <View
                      className={`mx-4 mb-2 p-3 rounded-xl border ${
                        selectedAddressId === addr.id
                          ? 'border-coral bg-coral-tint'
                          : 'border-hairline bg-canvas'
                      } ${pressed && Platform.OS === 'ios' && selectedAddressId !== addr.id ? 'bg-surface-soft' : ''}`}
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
                  )}
                </Pressable>
              ))}

              {/* Add new address toggle */}
              <Pressable
                onPress={() => setShowAddressForm((prev: boolean) => !prev)}
                accessibilityRole="button"
                accessibilityLabel="Add new address"
                android_ripple={{ color: CORAL_RIPPLE, borderless: false }}
              >
                {({ pressed }) => (
                  <View
                    className={`flex-row items-center gap-2 px-4 py-3 ${
                      pressed && Platform.OS === 'ios' ? 'opacity-70' : ''
                    }`}
                  >
                    <Plus size={16} color={customerColors.coral.DEFAULT} />
                    <Text className="text-sm text-coral font-medium">Add New Address</Text>
                  </View>
                )}
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
                    <Search size={16} color={customerColors.charcoal.soft} />
                    <TextInput
                      className="flex-1 h-11 text-sm text-charcoal"
                      placeholder="Search for your address"
                      placeholderTextColor={customerColors.charcoal.soft}
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
                      <ActivityIndicator size="small" color={customerColors.coral.DEFAULT} />
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
                          android_ripple={{ color: CHARCOAL_RIPPLE, borderless: false }}
                        >
                          {({ pressed }) => (
                            <View
                              className={`flex-row items-start gap-2 px-3 py-2.5 ${
                                pressed && Platform.OS === 'ios' ? 'bg-surface-soft' : ''
                              } ${i < addrSuggestions.length - 1 ? 'border-b border-hairline' : ''}`}
                            >
                              <MapPin size={16} color={customerColors.charcoal.soft} style={{ marginTop: 2 }} />
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
                          placeholderTextColor={customerColors.charcoal.soft}
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
                        placeholderTextColor={customerColors.charcoal.soft}
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
                            placeholderTextColor={customerColors.charcoal.soft}
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
                            placeholderTextColor={customerColors.charcoal.soft}
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
                          placeholderTextColor={customerColors.charcoal.soft}
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
                    android_ripple={{ color: CANVAS_RIPPLE, borderless: false }}
                  >
                    {({ pressed }) => (
                      <View
                        className={`rounded-xl py-2.5 items-center ${
                          addrSubmitting ? 'bg-coral-tint' : pressed && Platform.OS === 'ios' ? 'bg-coral-pressed' : 'bg-coral'
                        }`}
                        style={{ minHeight: 44, justifyContent: 'center' }}
                      >
                        {addrSubmitting ? (
                          <ActivityIndicator size="small" color={customerColors.coral.DEFAULT} />
                        ) : (
                          <Text className="text-canvas font-semibold text-sm">Save Address</Text>
                        )}
                      </View>
                    )}
                  </Pressable>
                </View>
              )}
            </>
          )}
        </View>
        ) : null}

        {/* ── Order Summary ── */}
        <View className="bg-canvas border-t border-hairline">
          <Text className="text-base font-semibold text-charcoal px-4 pt-4 pb-2">Order Summary</Text>
          <FlatList
            data={cartStore.items}
            keyExtractor={(item) => item.lineId}
            scrollEnabled={false}
            renderItem={({ item }) => (
              <View className="flex-row items-start px-4 py-2 gap-2">
                {/* Quantity chip — coral-tint bg, coral text */}
                <View className="w-6 h-6 rounded-full bg-coral-tint items-center justify-center mt-0.5">
                  <Text
                    className="text-xs font-medium text-coral"
                    style={{ fontVariant: ['tabular-nums'] }}
                  >
                    {item.quantity}
                  </Text>
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
            contentContainerStyle={{ paddingBottom: 8 }}
          />
        </View>

        {/* ── Price Details (fees) ── */}
        <View className="bg-canvas border-t border-hairline">
          <Text className="text-base font-semibold text-charcoal px-4 pt-4 pb-2">Price Details</Text>
          <View className="mx-4 pb-4 gap-2">
            <View className="flex-row justify-between">
              <Text className="text-sm text-charcoal-soft">Subtotal</Text>
              <Text
                className="text-sm text-charcoal"
                style={{ fontVariant: ['tabular-nums'], minWidth: 64, textAlign: 'right' }}
              >
                ₹{subtotal.toFixed(2)}
              </Text>
            </View>
            <View className="flex-row justify-between">
              <Text className="text-sm text-charcoal-soft">
                {fulfillment === 'pickup' ? 'Pickup' : 'Delivery fee'}
              </Text>
              {/* Free (green) when the fee is 0 — pickup, or a chef who delivers
                  free; otherwise the real amount the server will charge. */}
              {deliveryFee > 0 ? (
                <Text className="text-sm text-charcoal font-medium" style={{ fontVariant: ['tabular-nums'] }}>
                  ₹{deliveryFee.toFixed(2)}
                </Text>
              ) : (
                <Text className="text-sm text-success font-medium">Free</Text>
              )}
            </View>

            {/* Platform (service) fee — shown so the total is honest (#fee-transparency). */}
            {serviceFee > 0 ? (
              <View className="flex-row justify-between">
                <Text className="text-sm text-charcoal-soft">Platform fee</Text>
                <Text
                  className="text-sm text-charcoal font-medium"
                  style={{ fontVariant: ['tabular-nums'] }}
                >
                  ₹{serviceFee.toFixed(2)}
                </Text>
              </View>
            ) : null}

            {/* Tax — GST-compliant: an Indian intra-state supply shows CGST+SGST,
                inter-state shows IGST; other countries keep a single tax line. */}
            {tax > 0
              ? (quote?.taxCountry === 'IN'
                  ? quote?.taxIntraState
                    ? [
                        { label: `CGST (${(taxRate / 2).toFixed(2).replace(/\.?0+$/, '')}%)`, amt: tax / 2 },
                        { label: `SGST (${(taxRate / 2).toFixed(2).replace(/\.?0+$/, '')}%)`, amt: tax - tax / 2 },
                      ]
                    : [{ label: `IGST (${taxRate.toFixed(2).replace(/\.?0+$/, '')}%)`, amt: tax }]
                  : [{ label: taxInclusive ? `${taxName} (incl.)` : taxName, amt: tax }]
                ).map((row) => (
                  <View key={row.label} className="flex-row justify-between">
                    <Text className="text-sm text-charcoal-soft">{row.label}</Text>
                    <Text
                      className="text-sm text-charcoal font-medium"
                      style={{ fontVariant: ['tabular-nums'] }}
                    >
                      ₹{row.amt.toFixed(2)}
                    </Text>
                  </View>
                ))
              : null}

            {/* Self-delivery estimate (#702), collapsed behind a disclosure (R14):
                itemised base+distance+surge detail is secondary — the header row
                always shows the ceiling fee, the itemisation is opt-in. It's a
                ceiling the chef can only bring DOWN at accept — never a surprise
                increase — so it's still labelled as an estimate, not the charge. */}
            {fulfillment !== 'pickup' &&
            !deliveryOutOfRange &&
            quote?.offersSelfDelivery &&
            selfDeliveryBreakdown ? (
              <View className="rounded-xl border border-hairline bg-surface-soft px-3.5 py-3 gap-2">
                <Pressable
                  onPress={toggleFeeBreakdown}
                  accessibilityRole="button"
                  accessibilityLabel="Delivery fee breakdown"
                  accessibilityState={{ expanded: feeBreakdownOpen }}
                  android_ripple={{ color: CHARCOAL_RIPPLE, borderless: false }}
                >
                  {({ pressed }) => (
                    <View
                      className={`flex-row items-center justify-between ${
                        pressed && Platform.OS === 'ios' ? 'opacity-70' : ''
                      }`}
                      style={{ minHeight: 28 }}
                    >
                      <Text className="text-xs font-semibold text-charcoal">
                        Delivery fee breakdown
                      </Text>
                      <View className="flex-row items-center gap-1.5">
                        {selfDeliveryBreakdown.fee > 0 ? (
                          <Text
                            className="text-xs font-semibold text-charcoal"
                            style={{ fontVariant: ['tabular-nums'] }}
                          >
                            up to ₹{selfDeliveryBreakdown.fee.toFixed(2)}
                          </Text>
                        ) : (
                          <Text className="text-xs font-semibold text-success">Free</Text>
                        )}
                        <Animated.View style={feeChevronStyle}>
                          <ChevronDown size={14} color={customerColors.charcoal.soft} />
                        </Animated.View>
                      </View>
                    </View>
                  )}
                </Pressable>

                {feeBreakdownOpen ? (
                  <View className="gap-1.5">
                    <View className="flex-row items-center justify-between">
                      <Text className="text-xs text-charcoal-soft">Base fee</Text>
                      <Text
                        className="text-xs text-charcoal-soft"
                        style={{ fontVariant: ['tabular-nums'] }}
                      >
                        ₹{selfDeliveryBreakdown.baseFee.toFixed(2)}
                      </Text>
                    </View>
                    {selfDeliveryBreakdown.distanceKnown &&
                    selfDeliveryBreakdown.distanceComponent > 0 ? (
                      <View className="flex-row items-start justify-between gap-3">
                        <Text className="text-xs text-charcoal-soft flex-1 leading-4">
                          Distance ·{' '}
                          {selfDeliveryBreakdown.billableKm.toFixed(1)} km beyond{' '}
                          {selfDeliveryBreakdown.freeRadiusKm.toFixed(0)} km free
                        </Text>
                        <Text
                          className="text-xs text-charcoal-soft"
                          style={{ fontVariant: ['tabular-nums'] }}
                        >
                          ₹{selfDeliveryBreakdown.distanceComponent.toFixed(2)}
                        </Text>
                      </View>
                    ) : null}
                    {selfDeliveryBreakdown.surgeMultiplier > 1 &&
                    selfDeliveryBreakdown.distanceComponent > 0 ? (
                      <Text className="text-xs text-charcoal-soft leading-4">
                        Includes ×{selfDeliveryBreakdown.surgeMultiplier.toFixed(2)} surge
                        (fuel/peak conditions)
                      </Text>
                    ) : null}
                    {selfDeliveryBreakdown.withinFreeZone ? (
                      <Text className="text-xs text-success leading-4">
                        Within the chef's free-delivery radius
                      </Text>
                    ) : null}
                    {selfDeliveryBreakdown.capped ? (
                      <Text className="text-xs text-charcoal-soft leading-4">
                        Capped at the chef's ₹{selfDeliveryBreakdown.maxFee.toFixed(0)} maximum
                      </Text>
                    ) : null}
                    <Text className="text-xs text-charcoal-soft leading-4 pt-2 border-t border-hairline">
                      Estimate — the chef sets the final fee at accept and can only lower it.
                    </Text>
                  </View>
                ) : null}
              </View>
            ) : null}

            {/* Promo code (#39) — ghost-style Apply button per spec §3 */}
            {appliedPromo ? (
              <View className="flex-row items-center justify-between">
                <View className="flex-row items-center gap-2">
                  <View className="rounded bg-coral/10 px-2 py-0.5">
                    <Text className="text-xs font-semibold text-coral">{appliedPromo.code}</Text>
                  </View>
                  <Pressable
                    onPress={removePromo}
                    accessibilityRole="button"
                    accessibilityLabel="Remove promo code"
                    hitSlop={6}
                  >
                    {({ pressed }) => (
                      <Text
                        className="text-xs text-charcoal-soft underline"
                        style={pressed ? { opacity: 0.6 } : undefined}
                      >
                        Remove
                      </Text>
                    )}
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
                    // charcoal-soft meets AA on canvas; the old placeholder grey did not.
                    placeholderTextColor={customerColors.charcoal.soft}
                    autoCapitalize="characters"
                    autoCorrect={false}
                    accessibilityLabel="Promo code"
                    className="flex-1 rounded-lg border border-hairline bg-canvas px-3 py-2 text-sm text-charcoal"
                  />
                  {/* Ghost/outline button per spec §3: white bg, hairline border,
                      charcoal text — not a solid coral fill. */}
                  <Pressable
                    onPress={applyPromo}
                    disabled={!promoInput.trim() || validatePromo.isPending}
                    accessibilityRole="button"
                    accessibilityLabel="Apply promo code"
                    android_ripple={{ color: CHARCOAL_RIPPLE, borderless: false }}
                  >
                    {({ pressed }) => (
                      <View
                        className={`rounded-lg border px-4 items-center justify-center ${
                          !promoInput.trim() || validatePromo.isPending
                            ? 'border-hairline'
                            : 'border-charcoal'
                        } ${pressed && Platform.OS === 'ios' ? 'bg-surface-soft' : 'bg-canvas'}`}
                        style={{ minHeight: 40 }}
                      >
                        <Text
                          className={`text-sm font-semibold ${
                            !promoInput.trim() || validatePromo.isPending
                              ? 'text-charcoal-soft'
                              : 'text-charcoal'
                          }`}
                        >
                          {validatePromo.isPending ? '…' : 'Apply'}
                        </Text>
                      </View>
                    )}
                  </Pressable>
                </View>
                {promoError ? <Text className="text-xs text-destructive">{promoError}</Text> : null}
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
                android_ripple={{ color: CHARCOAL_RIPPLE, borderless: false }}
              >
                {({ pressed }) => (
                  <View
                    className={`flex-row items-center justify-between ${
                      pressed && Platform.OS === 'ios' ? 'opacity-70' : ''
                    }`}
                    style={{ minHeight: 44 }}
                  >
                    <View className="flex-row items-center gap-2">
                      <View
                        className={`h-5 w-5 items-center justify-center rounded border ${
                          applyWallet ? 'border-coral bg-coral' : 'border-hairline bg-canvas'
                        }`}
                      >
                        {applyWallet && <Check size={14} color={customerColors.canvas} />}
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
                  </View>
                )}
              </Pressable>
            )}

            <View className="flex-row justify-between pt-1 border-t border-hairline">
              <Text className="text-base font-medium text-charcoal">
                {walletApplied > 0 ? 'To pay' : 'Total'}
              </Text>
              <Text
                className="text-base font-medium text-charcoal"
                style={{ fontVariant: ['tabular-nums'], minWidth: 72, textAlign: 'right' }}
              >
                ₹{payable.toFixed(2)}
              </Text>
            </View>
          </View>
        </View>

        {/* ── Dietary / allergen conflict warning (#41) ── */}
        {dietaryWarnings.length > 0 && (
          <View className="mx-4 mt-4 bg-destructive-tint border border-destructive/30 rounded-2xl p-4">
            <View className="flex-row items-center gap-2 mb-1">
              <AlertTriangle size={16} color={customerColors.destructive.DEFAULT} />
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

        {/* ── Preferred time ──
            Window chefs (#51) keep the slot picker; the home-tiffin default is
            the suggested-time handshake (#709): the customer proposes a time and
            the chef confirms or proposes a different one at accept. */}
        {useSlotPicker ? (
          <View className="bg-canvas border-t border-hairline p-4">
            <Text className="text-sm font-medium text-charcoal-soft mb-3">Delivery time</Text>
            <View className="flex-row flex-wrap gap-2">
              {/* ASAP (default) — selected chip = coral fill + white text per spec */}
              <Pressable
                onPress={() => setSelectedSlot(null)}
                accessibilityRole="radio"
                accessibilityState={{ selected: selectedSlot === null }}
                android_ripple={{ color: CORAL_RIPPLE, borderless: false }}
              >
                {({ pressed }) => (
                  <View
                    className={`px-3 py-2 rounded-xl border justify-center ${
                      selectedSlot === null
                        ? 'border-coral bg-coral'
                        : 'border-hairline bg-surface-soft'
                    } ${pressed && Platform.OS === 'ios' && selectedSlot !== null ? 'bg-hairline' : ''}`}
                    style={{ minHeight: 44 }}
                  >
                    <Text
                      className={`text-sm font-medium ${
                        selectedSlot === null ? 'text-canvas' : 'text-charcoal'
                      }`}
                    >
                      ASAP
                    </Text>
                    <Text
                      className={`text-xs ${selectedSlot === null ? 'text-canvas/80' : 'text-charcoal-soft'}`}
                    >
                      After chef accepts
                    </Text>
                  </View>
                )}
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
                    android_ripple={{ color: CORAL_RIPPLE, borderless: false }}
                  >
                    {({ pressed }) => (
                      <View
                        className={`px-3 py-2 rounded-xl border justify-center ${
                          sel ? 'border-coral bg-coral' : 'border-hairline bg-surface-soft'
                        } ${pressed && Platform.OS === 'ios' && !sel ? 'bg-hairline' : ''}`}
                        style={{ minHeight: 44 }}
                      >
                        <Text className={`text-sm font-medium ${sel ? 'text-canvas' : 'text-charcoal'}`}>
                          {slotDayLabel(s.date)} · {s.label}
                        </Text>
                        <Text
                          className={`text-xs ${sel ? 'text-canvas/80' : 'text-charcoal-soft'}`}
                          style={{ fontVariant: ['tabular-nums'] }}
                        >
                          {s.window}
                          {s.remaining != null ? ` · ${s.remaining} left` : ''}
                        </Text>
                      </View>
                    )}
                  </Pressable>
                );
              })}
            </View>
          </View>
        ) : (
          <View className="bg-canvas border-t border-hairline p-4">
            <Text className="text-sm font-medium text-charcoal mb-0.5">
              {fulfillment === 'pickup' ? 'Preferred pickup time' : 'Preferred delivery time'}
            </Text>
            <Text className="text-xs text-charcoal-soft mb-3 leading-4">
              {fulfillment === 'pickup'
                ? "When will you come to collect? It's a home kitchen — the chef confirms once they accept."
                : "Suggest when you'd like it. It's a home kitchen, not a restaurant — the chef confirms or proposes a time when they accept."}
            </Text>
            <View className="gap-3">
              {/* As soon as ready (default) — recommended, full-width so it reads
                  as the primary choice above the specific-time clusters (R14: the
                  default hero option). */}
              <Pressable
                onPress={() => setRequestedTime(null)}
                accessibilityRole="radio"
                accessibilityState={{ selected: requestedTime === null }}
                accessibilityLabel="As soon as ready"
                android_ripple={{ color: CORAL_RIPPLE, borderless: false }}
              >
                {({ pressed }) => (
                  <View
                    className={`flex-row items-center justify-between px-3.5 py-3 rounded-xl border ${
                      requestedTime === null ? 'border-coral bg-coral-tint' : 'border-hairline bg-surface-soft'
                    } ${pressed && Platform.OS === 'ios' && requestedTime !== null ? 'bg-hairline' : ''}`}
                  >
                    <View className="flex-1">
                      <Text className={`text-sm font-semibold ${requestedTime === null ? 'text-coral' : 'text-charcoal'}`}>
                        As soon as ready
                      </Text>
                      <Text className="text-xs text-charcoal-soft mt-0.5">Chef decides when to start</Text>
                    </View>
                    <View
                      className={`h-5 w-5 items-center justify-center rounded-full border ${
                        requestedTime === null ? 'border-coral bg-coral' : 'border-hairline bg-canvas'
                      }`}
                    >
                      {requestedTime === null ? <Check size={13} color={customerColors.canvas} /> : null}
                    </View>
                  </View>
                )}
              </Pressable>

              {/* Specific times, grouped into scannable day·meal clusters. Chips
                  show just the clock label — the cluster header carries day+meal. */}
              {fulfillmentTimeGroups.map((group) => (
                <View key={group.key} className="gap-2">
                  <Text className="text-xs font-semibold text-charcoal-soft">{group.key}</Text>
                  <View className="flex-row flex-wrap gap-2">
                    {group.times.map((t) => {
                      const sel = requestedTime?.toISOString() === new Date(t.at).toISOString();
                      return (
                        <Pressable
                          key={t.at}
                          onPress={() => setRequestedTime(new Date(t.at))}
                          accessibilityRole="radio"
                          accessibilityState={{ selected: sel }}
                          accessibilityLabel={`${fulfillment === 'pickup' ? 'Pickup' : 'Delivery'} around ${t.label}, ${t.day} ${t.meal}`}
                          android_ripple={{ color: CORAL_RIPPLE, borderless: false }}
                        >
                          {({ pressed }) => (
                            <View
                              className={`min-w-[72px] items-center justify-center px-3 py-2 rounded-xl border ${
                                sel ? 'border-coral bg-coral' : 'border-hairline bg-surface-soft'
                              } ${pressed && Platform.OS === 'ios' && !sel ? 'bg-hairline' : ''}`}
                              style={{ minHeight: 44 }}
                            >
                              <Text
                                className={`text-sm font-medium ${sel ? 'text-canvas' : 'text-charcoal'}`}
                                style={{ fontVariant: ['tabular-nums'] }}
                              >
                                {t.label}
                              </Text>
                            </View>
                          )}
                        </Pressable>
                      );
                    })}
                  </View>
                </View>
              ))}
            </View>
          </View>
        )}

        {/* ── Payment & delivery (concise; detail behind policy links) ── */}
        {/* CW-01d / RBI PA MD §8: keeps the required PA + refund-window disclosure
            as one line; full cancellation rules live behind the Refund Policy link. */}
        <View className="bg-canvas border-t border-hairline p-4 gap-2.5">
          <View className="flex-row items-start gap-2">
            <Clock size={16} color={customerColors.coral.DEFAULT} style={{ marginTop: 1 }} />
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
        <View
          ref={termsSectionRef}
          onLayout={(e) => {
            termsSectionY.current = e.nativeEvent.layout.y;
          }}
          className="mx-4 mt-4"
        >
          {/* iOS Pressable inner-View pattern — visual styles on the inner View.
              44pt row (R5), visual checkbox, label preserved verbatim. */}
          <Pressable
            onPress={() => setAcceptedTerms((prev: boolean) => !prev)}
            accessibilityRole="checkbox"
            accessibilityState={{ checked: acceptedTerms }}
            accessibilityLabel="I agree to the Terms of Service and Refund Policy for this order"
            android_ripple={{ color: CHARCOAL_RIPPLE, borderless: false }}
          >
            {({ pressed }) => (
              <View
                className={`flex-row items-start gap-3 bg-surface-soft rounded-2xl p-4 ${
                  pressed && Platform.OS === 'ios' ? 'bg-hairline' : ''
                }`}
                style={{ minHeight: 44 }}
              >
                <View
                  className={`w-5 h-5 rounded border-2 items-center justify-center mt-0.5 ${
                    acceptedTerms ? 'border-coral bg-coral' : 'border-hairline bg-canvas'
                  }`}
                >
                  {acceptedTerms && <Check size={14} color={customerColors.canvas} />}
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
            )}
          </Pressable>
        </View>

        {/* ── Note (optional) ── */}
        <View className="bg-canvas border-t border-hairline p-4">
          <Text className="text-sm font-medium text-charcoal-soft mb-2">Note to chef (optional)</Text>
          <TextInput
            value={note}
            onChangeText={setNote}
            placeholder="Any special instructions..."
            placeholderTextColor={customerColors.charcoal.soft}
            multiline
            numberOfLines={2}
            maxLength={200}
            returnKeyType="done"
            blurOnSubmit
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
              hitSlop={6}
            >
              {({ pressed }) => (
                <Text
                  className="text-xs text-destructive underline"
                  style={pressed ? { opacity: 0.6 } : undefined}
                >
                  Dismiss
                </Text>
              )}
            </Pressable>
          </View>
        )}
      </ScrollView>

      {/* ── Place Order button — sticky bottom, coral filled, radius 8, 52pt,
          tabular total. Distinct disabled state: reduced-opacity coral fill +
          charcoal-soft label, never fully invisible-looking (spec §2.5). ── */}
      <View
        style={{
          shadowColor: customerTheme.shadow[2].shadowColor,
          shadowOffset: { width: 0, height: -2 },
          shadowOpacity: customerTheme.shadow[2].shadowOpacity,
          shadowRadius: customerTheme.shadow[2].shadowRadius,
          elevation: customerTheme.shadow[2].elevation,
          paddingBottom: insets.bottom > 0 ? insets.bottom + 8 : 16,
        }}
        className="absolute bottom-0 left-0 right-0 bg-canvas border-t border-hairline px-4 pt-3"
      >
        {/* iOS Pressable inner-View pattern */}
        <Pressable
          onPress={handlePlaceOrder}
          disabled={hardBlocked}
          accessibilityRole="button"
          accessibilityLabel="Place Order"
          accessibilityState={{ disabled: !placeEnabled }}
          android_ripple={{ color: CANVAS_RIPPLE, borderless: false }}
        >
          {({ pressed }) => (
            <View
              className={`w-full rounded-lg items-center justify-center flex-row gap-2 bg-coral ${
                placeEnabled && pressed && Platform.OS === 'ios' ? 'bg-coral-pressed' : ''
              }`}
              style={{ minHeight: 52, opacity: placeEnabled ? 1 : 0.4 }}
            >
              {isLoading ? (
                <ActivityIndicator size="small" color={customerColors.canvas} />
              ) : (
                <Text
                  className={`text-base font-semibold ${placeEnabled ? 'text-canvas' : 'text-charcoal-soft'}`}
                  style={{ fontVariant: ['tabular-nums'], minWidth: 96, textAlign: 'center' }}
                >
                  {isLoading ? 'Processing...' : `Place Order · ₹${payable.toFixed(2)}`}
                </Text>
              )}
            </View>
          )}
        </Pressable>
      </View>
    </KeyboardAvoidingView>
  );
}
