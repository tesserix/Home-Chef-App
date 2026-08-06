import { useEffect, useMemo, useState } from "react";
import { useNavigate, Link } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  MapPin,
  Clock,
  ChevronRight,
  Check,
  FileText,
  Shield,
  AlertTriangle,
  Store,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import { useCartStore } from "@/app/store/cart-store";
import { useAuth } from "@/app/providers/AuthProvider";
import { apiClient } from "@/shared/services/api-client";
import { useFormatPrice } from "@/shared/utils/format-price";
import { loadStripeJs } from "@/shared/utils/load-stripe";
import { openCashfreeCheckout } from "@/shared/utils/cashfree";
import { resolveCssVarColor } from "@/shared/utils/css-color";
import { Button } from "@/shared/components/ui";
import { earliestBakeryFulfillment } from "@homechef/mobile-shared/bakery";
import type { Order, Address } from "@/shared/types";
import { useDeliveryQuote, type CreditIntent } from "../hooks/useDeliveryQuote";
import { surgeReasonText } from "../lib/surge";
import {
  useFulfillmentTimes,
  groupFulfillmentTimes,
  type FulfillmentTime,
} from "../hooks/useFulfillmentTimes";
import { getFeeRowLabel } from "../lib/orderSteps";
import { CheckoutCredits } from "../components/CheckoutCredits";
import { AddressSearch } from "../components/AddressSearch";
import {
  suggestionCoords,
  type AddressSuggestion,
} from "../hooks/useAddressAutocomplete";

const addressSchema = z.object({
  label: z.string().min(1, "Label is required"),
  line1: z.string().min(5, "Address is required"),
  line2: z.string().optional(),
  city: z.string().min(2, "City is required"),
  state: z.string().min(2, "State is required"),
  postalCode: z.string().min(5, "Postal code is required"),
  deliveryInstructions: z.string().optional(),
});

type AddressFormData = z.infer<typeof addressSchema>;

// Tip presets, in the order currency. Whole amounts a customer would actually
// leave on a home-kitchen order — the previous 2/5/10 were a pre-INR holdover
// that offered a ₹2 tip on a ₹270 order. Kept in lockstep with
// apps/mobile-customer/app/checkout.tsx.
const TIP_PRESETS = [0, 20, 30, 50];
// A ceiling on the custom field: a fat-fingered 99999 tip is a support ticket,
// not a generous customer.
const MAX_TIP = 5000;

// Scheduled delivery slots (#51) — mirrors the API GET /chefs/:id/delivery-slots
// response (services.SlotAvailability).
interface DeliverySlot {
  date: string; // "YYYY-MM-DD" IST
  slot: "lunch" | "dinner";
  label: string;
  window: string; // "12:00–14:00"
  remaining: number | null; // null = unlimited
  available: boolean;
  scheduledFor: string; // the slot's instant, used to gate bakery lead time
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

// "Tomorrow, 4:30 pm" — the earliest a bake with a lead time can be had (#1065).
function leadTimeLabel(d: Date): string {
  const day = slotDayLabel(
    `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(
      d.getDate(),
    ).padStart(2, "0")}`,
  );
  const time = d.toLocaleTimeString(undefined, {
    hour: "numeric",
    minute: "2-digit",
  });
  return `${day}, ${time}`;
}

// slotDayLabel turns a "YYYY-MM-DD" slot date into a label relative to today
// ("Today" / "Tomorrow" / "Mon, 22 Jun").
function slotDayLabel(dateStr: string): string {
  const d = new Date(`${dateStr}T00:00:00`);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const diff = Math.round((d.getTime() - today.getTime()) / 86_400_000);
  if (diff <= 0) return "Today";
  if (diff === 1) return "Tomorrow";
  return d.toLocaleDateString(undefined, {
    weekday: "short",
    day: "numeric",
    month: "short",
  });
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
    queryKey: ["addresses"],
    queryFn: () => apiClient.get<Address[]>("/addresses"),
  });

  // Fulfilment mode. The customer chooses delivery vs pickup and nothing else —
  // WHO carries a delivery order (the chef themselves vs a 3PL rider) is the
  // chef's call at Mark Ready, resolved server-side, so `chef_delivery` is never
  // sent from here. Mirrors apps/mobile-customer/app/checkout.tsx.
  const [fulfillment, setFulfillment] = useState<"delivery" | "pickup">(
    "delivery",
  );
  const isPickup = fulfillment === "pickup";

  const [selectedAddress, setSelectedAddress] = useState<string>("");
  // Default to the user's default address (or the first one) as soon as
  // the list loads. Resets if the previously-selected id disappears.
  useEffect(() => {
    if (!savedAddresses.length) {
      if (selectedAddress) setSelectedAddress("");
      return;
    }
    const exists = savedAddresses.some((a) => a.id === selectedAddress);
    if (!exists) {
      const preferred =
        savedAddresses.find((a) => a.isDefault) ?? savedAddresses[0];
      if (preferred) setSelectedAddress(preferred.id);
    }
  }, [savedAddresses, selectedAddress]);
  const [showNewAddress, setShowNewAddress] = useState(false);
  // Scheduled delivery slot (#51) — null = ASAP. The picker below only offers
  // slots when the chef has enabled them.
  const [selectedSlot, setSelectedSlot] = useState<{
    slot: string;
    date: string;
  } | null>(null);
  const { data: slotsData } = useQuery({
    queryKey: ["delivery-slots", cart.chefId],
    queryFn: () =>
      apiClient.get<DeliverySlotsResponse>(
        `/chefs/${cart.chefId}/delivery-slots`,
      ),
    enabled: Boolean(cart.chefId),
    staleTime: 60_000,
  });
  const availableSlots = (slotsData?.slots ?? []).filter((s) => s.available);
  // Windowed (restaurant-style) chefs keep the #51 slot picker; everyone else —
  // the home-tiffin default — gets the suggested-time handshake (#709) below.
  const useSlotPicker =
    Boolean(slotsData?.slotsEnabled) && availableSlots.length > 0;

  // Home-tiffin suggested time (#709): the customer PROPOSES a preferred time and
  // the chef confirms or counters at accept. null = "as soon as ready", which
  // stays the default.
  const [requestedTime, setRequestedTime] = useState<FulfillmentTime | null>(
    null,
  );
  const { data: fulfillmentTimesData } = useFulfillmentTimes(
    cart.chefId ?? undefined,
  );
  const fulfillmentTimeGroups = useMemo(
    () => groupFulfillmentTimes(fulfillmentTimesData?.times ?? []),
    [fulfillmentTimesData],
  );
  // Bakery lead time (#1065). The server rejects an order placed sooner than a
  // bake needs, so the customer is told here rather than at submit: ASAP is
  // hidden, too-soon times are disabled, and placing the order is blocked.
  const bakeryLeadHours = cart.items.reduce(
    (max, i) => Math.max(max, i.bakeryLeadTimeHours ?? 0),
    0,
  );
  const bakeryEarliest =
    bakeryLeadHours > 0
      ? earliestBakeryFulfillment(new Date(), bakeryLeadHours)
      : null;
  const chosenFulfillmentAt = useSlotPicker
    ? (() => {
        const s = availableSlots.find(
          (x) => x.slot === selectedSlot?.slot && x.date === selectedSlot?.date,
        );
        return s ? new Date(s.scheduledFor) : null;
      })()
    : requestedTime
      ? new Date(requestedTime.at)
      : null;
  const bakeryTimeTooSoon =
    bakeryEarliest !== null &&
    (chosenFulfillmentAt === null ||
      chosenFulfillmentAt.getTime() < bakeryEarliest.getTime());
  const bakeryLeadNotice =
    bakeryEarliest === null
      ? null
      : `This order includes a bake that needs ${bakeryLeadHours}h notice — choose a time from ${leadTimeLabel(bakeryEarliest)} onwards.`;

  // Dietary & allergen conflict warning (#41) — server-checks the cart's items
  // against the customer's saved profile. Non-blocking.
  const cartItemIds = cart.items.map((i) => i.menuItemId);
  const { data: dietaryCheck } = useQuery({
    queryKey: ["dietary-check", cartItemIds],
    queryFn: () =>
      apiClient.post<DietaryCheckResult>("/dietary/check", {
        menuItemIds: cartItemIds,
      }),
    enabled: cartItemIds.length > 0,
    staleTime: 60_000,
  });
  const dietaryWarnings = dietaryCheck?.warnings ?? [];

  const [tip, setTip] = useState<number>(0);
  const [customTip, setCustomTip] = useState("");
  // Credit intent. Both rails default ON so the customer always spends the credit
  // they hold; undefined amounts mean "auto" — the server applies as much as its
  // ceilings allow. Touching either control pins both.
  const [credit, setCredit] = useState<CreditIntent>({
    useWallet: true,
    useLoyalty: true,
  });
  const [specialInstructions, setSpecialInstructions] = useState("");
  const [isProcessing, setIsProcessing] = useState<boolean>(false);
  // CW-01d: explicit T&C + Refund Policy consent is required per order
  // (not just at signup) for RBI PA disclosure compliance. The Place Order
  // CTA stays disabled until this is checked.
  const [acceptedTerms, setAcceptedTerms] = useState<boolean>(false);

  // The currency of the amounts on this page is the chef's settlement
  // currency — that's what the backend will charge the customer in. Falls
  // back to INR so pre-multi-gateway chef profiles keep rendering.
  const orderCurrency =
    (cart.chef as { currency?: string } | null)?.currency || "INR";

  const selectedAddressObj = savedAddresses.find(
    (a) => a.id === selectedAddress,
  );
  const taxCountry = selectedAddressObj?.country || "IN";

  const subtotal = cart.getSubtotal();
  // Applied promo discount (#39), clamped to the subtotal. Mirrors the server,
  // which taxes the post-discount base; server is authoritative at order time.
  // Declared before the quote because the credit ceiling is computed on the
  // DISCOUNTED food value, so the server needs it.
  const discount = cart.promoCode ? Math.min(cart.promoDiscount, subtotal) : 0;

  // Fees, tax AND the wallet/loyalty allocation all come from the one endpoint
  // CreateOrder itself prices against. The page used to invent its own delivery
  // fee (the chef's flat column) and service fee (a hardcoded 5%), so the total
  // it showed was not the total it charged — and it had no access to the credit
  // block at all, which is why wallet and points were unspendable here.
  const { data: quote } = useDeliveryQuote(cart.chefId ?? undefined, {
    latitude: selectedAddressObj?.latitude,
    longitude: selectedAddressObj?.longitude,
    city: selectedAddressObj?.city,
    state: selectedAddressObj?.state,
    country: taxCountry,
    subtotal,
    discount,
    tip,
    fulfillment,
    credit,
  });

  // The aggregator named in the RBI PA disclosure below. Driven by the
  // SERVER-resolved gateway, never by a hardcoded name — the platform can route
  // an order to a different gateway than the chef's stored one, and this block
  // is a regulatory disclosure that has to say who actually processes the money.
  // Unknown resolves to neutral wording rather than guessing.
  const gatewayName =
    quote?.paymentProvider === "cashfree"
      ? "Cashfree"
      : quote?.paymentProvider === "razorpay"
        ? "Razorpay"
        : quote?.paymentProvider === "stripe"
          ? "Stripe"
          : null;
  // What the chef actually offers. Both come from the quote the page already
  // fetches, so there is no second round-trip. offersDelivery is the computed
  // capability CreateOrder gates on (chef self-delivers OR a 3PL provider is
  // live) — defaulting it to true keeps an older API working.
  const offersPickup = quote?.offersPickup ?? false;
  const offersDelivery = quote?.offersDelivery ?? true;
  const fulfillmentModes: Array<"delivery" | "pickup"> = [
    ...(offersDelivery ? (["delivery"] as const) : []),
    ...(offersPickup ? (["pickup"] as const) : []),
  ];

  // Snap the selection to something the chef actually offers. This is the guard
  // that matters with 3PL dark and a non-self-delivering chef: without it the page
  // posts a delivery order the server rejects, and the customer only finds out at
  // payment. Runs on the quote, so it corrects as soon as capabilities are known.
  useEffect(() => {
    if (fulfillmentModes.length === 0) return;
    if (!fulfillmentModes.includes(fulfillment)) {
      setFulfillment(fulfillmentModes[0] as "delivery" | "pickup");
    }
    // fulfillmentModes is rebuilt every render; depend on its inputs instead.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [offersDelivery, offersPickup, fulfillment]);

  // Pickup is always free — the customer carries it. Delivery is the quoted fee.
  const deliveryFee = isPickup ? 0 : (quote?.deliveryFee ?? 0);
  const selfDeliveryBreakdown = quote?.selfDeliveryBreakdown;
  // Why delivery costs more today, in words. null when conditions are normal.
  const surgeReason = surgeReasonText(quote?.surge);
  // What switching to pickup would save. Only real when delivery actually costs
  // something; 0 means show no incentive rather than a fake one.
  const pickupSaving = quote?.pickupSaving ?? 0;
  // Fee, tax rows and total all come from the quote, priced by the SAME function
  // CreateOrder charges with (models/pricing.go). The page redoing that
  // arithmetic on rounded inputs is how checkout previewed a total a paise off
  // the one the receipt then printed.
  const platformFee = quote?.platformFee ?? 0;
  const taxLines = quote?.taxLines ?? [];
  const total = quote?.total ?? 0;

  // Wallet + loyalty credit. Every figure comes from the server quote; the page
  // does no money arithmetic of its own here. `payable` is what the gateway will
  // be asked for, and it is what the CTA must say.
  const creditQuote = quote?.credit;
  const walletApplied =
    creditQuote && credit.useWallet ? creditQuote.walletApplied : 0;
  const loyaltyApplied =
    creditQuote && credit.useLoyalty ? creditQuote.pointsValue : 0;
  const creditApplied = walletApplied + loyaltyApplied;
  const payable = creditQuote ? creditQuote.payable : total;

  // Serviceability (#709). CreateOrder HARD-rejects a delivery order it cannot
  // range-check: no coordinates on the address is a 422 `delivery_location_
  // required`, and beyond the kitchen's radius is a 422 `outside_delivery_range`.
  // Web had neither guard AND no way to put coordinates on an address, so every
  // web order 422'd and the customer saw only "Failed to initiate payment".
  //
  // Neither guard applies to pickup: there is no drop address to range-check
  // because the customer comes to the kitchen. Gating them on !isPickup is what
  // lets pickup stay orderable for a customer whose only saved address is
  // unpinned or out of range — which is exactly when pickup is most useful.
  const addressNeedsLocation =
    !isPickup &&
    Boolean(selectedAddressObj) &&
    !(selectedAddressObj?.latitude && selectedAddressObj?.longitude);
  const deliveryOutOfRange =
    !isPickup && quote?.rangeKnown === true && quote?.deliverable === false;

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
    setValue,
  } = useForm<AddressFormData>({
    resolver: zodResolver(addressSchema),
  });

  // Coordinates of the picked suggestion, persisted with the new address. Without
  // them the address is unorderable — see addressNeedsLocation above.
  const [newAddressCoords, setNewAddressCoords] = useState<{
    lat: number;
    lon: number;
  } | null>(null);

  // Fill the form from a geocoder suggestion. The fields stay editable — the
  // suggestion supplies the coordinates and a sane starting point, not the last
  // word on the flat number.
  const applySuggestion = (s: AddressSuggestion) => {
    setValue("line1", s.line1 || s.description, { shouldValidate: true });
    if (s.city) setValue("city", s.city, { shouldValidate: true });
    if (s.region) setValue("state", s.region, { shouldValidate: true });
    if (s.postal) setValue("postalCode", s.postal, { shouldValidate: true });
    setNewAddressCoords(suggestionCoords(s));
  };

  // Pin an EXISTING saved address that predates the geocoder. Every address the
  // web created before this shipped has no coordinates, so without a repair path
  // those customers would have to re-add addresses they already saved.
  const [isPinning, setIsPinning] = useState(false);
  const pinExistingAddress = async (s: AddressSuggestion) => {
    const coords = suggestionCoords(s);
    if (!coords || !selectedAddressObj) {
      toast.error(
        "That suggestion has no location — try a nearby landmark or the area name.",
      );
      return;
    }
    setIsPinning(true);
    try {
      // UpdateAddress replaces the record wholesale, so every field has to ride
      // along or the untouched ones would be blanked.
      await apiClient.put(`/addresses/${selectedAddressObj.id}`, {
        label: selectedAddressObj.label,
        line1: selectedAddressObj.line1,
        line2: selectedAddressObj.line2,
        city: selectedAddressObj.city,
        state: selectedAddressObj.state,
        postalCode: selectedAddressObj.postalCode,
        country: selectedAddressObj.country || "IN",
        latitude: coords.lat,
        longitude: coords.lon,
        isDefault: selectedAddressObj.isDefault,
      });
      await queryClient.invalidateQueries({ queryKey: ["addresses"] });
      toast.success("Delivery location saved");
    } catch {
      toast.error("Could not save that location. Please try again.");
    } finally {
      setIsPinning(false);
    }
  };

  const handlePlaceOrder = async () => {
    if (!cart.chefId) {
      toast.error("Your cart is empty");
      return;
    }
    // A pickup order has no drop address — the customer collects from the kitchen.
    if (!isPickup && !selectedAddress) {
      toast.error("Please select a delivery address");
      return;
    }
    if (addressNeedsLocation) {
      toast.error(
        "Set this address's delivery location so we can confirm it's within the kitchen's range.",
      );
      return;
    }
    if (deliveryOutOfRange) {
      toast.error(
        `This address is ${(quote?.distanceKm ?? 0).toFixed(1)} km from the kitchen — beyond its ${(
          quote?.maxRadiusKm ?? 10
        ).toFixed(0)} km delivery range.`,
      );
      return;
    }
    if (!acceptedTerms) {
      toast.error(
        "Please accept the Terms of Service and Refund Policy to continue",
      );
      return;
    }
    if (bakeryTimeTooSoon && bakeryLeadNotice) {
      toast.error(bakeryLeadNotice);
      return;
    }

    setIsProcessing(true);

    try {
      // Step 1: Create order. The backend decides which gateway to use
      // based on the chef's PaymentProvider setting.
      const order = await apiClient.post<Order>("/orders", {
        // Map cart lines to the API item shape, including selected add-ons (#232).
        items: cart.items.map((i) => ({
          menuItemId: i.menuItemId,
          quantity: i.quantity,
          notes: i.notes || undefined,
          modifierOptionIds: i.modifiers?.map((m) => m.optionId),
          // The bake as configured (#1065) — the server re-prices it from the
          // spec and snapshots it onto the line.
          bakery: i.bakery,
        })),
        chefId: cart.chefId,
        // Pickup carries no delivery address. Sending one anyway would make the
        // server range-check a drop it will never make.
        deliveryAddressId: isPickup ? undefined : selectedAddress,
        fulfillmentType: fulfillment,
        tip,
        specialInstructions: specialInstructions || undefined,
        deliverySlot: selectedSlot?.slot,
        deliveryDate: selectedSlot?.date,
        // The customer's suggested fulfilment time (#709) — a proposal, not a
        // promise. The chef confirms or counters when they accept.
        requestedFulfillmentAt: requestedTime?.at,
        // Applied promo (#39) — server re-validates + recomputes the discount.
        promoCode: cart.promoCode || undefined,
        // The surge this checkout was quoted at. Traffic and weather move while the
        // customer picks an address and pays; without this the server would re-read
        // them and could charge a delivery fee this page never displayed.
        surgePin: quote?.surgePin || undefined,
      });

      // Step 2: Ask the backend to prepare a payment. The response shape
      // varies by provider — `provider: "razorpay"` returns a Razorpay
      // order id + key id; `provider: "stripe"` returns a PaymentIntent
      // clientSecret + publishable key.
      type RazorpayPayment = {
        provider: "razorpay";
        paid?: boolean;
        razorpayOrderId: string;
        razorpayKeyId: string;
        amount: number;
        currency: string;
      };
      type StripePayment = {
        provider: "stripe";
        paid?: boolean;
        stripePaymentIntentId: string;
        clientSecret: string;
        publishableKey: string;
        amount: number;
        currency: string;
      };
      // Cashfree returns a payment_session_id the SDK opens checkout with, plus
      // the environment — there is no publishable key, and no client-side
      // signature comes back, so the server's verify call is the only authority.
      type CashfreePayment = {
        provider: "cashfree";
        paid?: boolean;
        cashfreePaymentSessionId: string;
        cashfreeOrderId: string;
        cashfreeEnv?: string;
        amount: number;
        currency: string;
      };
      // Credit covered the whole total — the server has already marked the order
      // paid and there is nothing for a gateway to collect.
      type WalletPayment = { provider: "wallet"; paid?: boolean };

      // The client sends INTENT, never a computed payable: the server re-runs the
      // whole allocation from the live balance and the real order, and its answer
      // is what is charged. Posting an amount is what would let the screen show
      // one figure while the gateway took another.
      const paymentData = await apiClient.post<
        RazorpayPayment | CashfreePayment | StripePayment | WalletPayment
      >(`/payments/order/${order.id}/create`, credit);

      if (paymentData.provider === "wallet" || paymentData.paid) {
        cart.clearCart();
        toast.success("Paid with your credits!");
        navigate(`/orders/${order.id}`);
        return;
      }

      if (paymentData.provider === "stripe") {
        await confirmStripePayment(order.id, paymentData);
      } else if (paymentData.provider === "cashfree") {
        await confirmCashfreePayment(order, paymentData);
      } else {
        await confirmRazorpayPayment(order, paymentData);
      }
    } catch (e: unknown) {
      // Surface a promo failure specifically (e.g. the code was exhausted between
      // applying it and checkout) and drop it so the retry isn't blocked (#39).
      const raw = (e as { error?: unknown })?.error;
      const msg =
        typeof raw === "string"
          ? raw
          : raw && typeof raw === "object" && "message" in raw
            ? String((raw as { message?: unknown }).message ?? "")
            : e instanceof Error
              ? e.message
              : "";
      if (/promo/i.test(msg)) {
        cart.clearPromo();
        toast.error(
          msg || "That promo code is no longer available. Please try again.",
        );
      } else {
        // This block also catches order CREATION failures, which have real,
        // actionable reasons — an out-of-range address, a kitchen that stopped
        // accepting orders, an item that sold out. Flattening all of them to
        // "Failed to initiate payment" told the customer to retry the one thing
        // that would fail identically. Show what the server actually said.
        toast.error(msg || "Failed to initiate payment. Please try again.");
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
    },
  ) => {
    if (!window.Razorpay) {
      toast.error("Payment gateway is loading. Please try again.");
      return;
    }
    const options: RazorpayOptions = {
      key: paymentData.razorpayKeyId,
      amount: paymentData.amount,
      currency: paymentData.currency,
      name: "Fe3dr",
      description: `Order from ${cart.chef?.businessName || "Home Chef"}`,
      order_id: paymentData.razorpayOrderId,
      prefill: {
        name: user?.name || "",
        email: user?.email || "",
      },
      // Resolve --herb at runtime so a theme change ripples to Razorpay's
      // hosted checkout. Falls back to a static herb-equivalent hex if the
      // CSS var is unavailable (SSR / older browsers without oklch).
      theme: { color: resolveCssVarColor("--herb", "#3e6b3c") },
      handler: async (response) => {
        try {
          await apiClient.post(`/payments/order/${order.id}/verify`, {
            razorpayPaymentId: response.razorpay_payment_id,
            razorpayOrderId: response.razorpay_order_id,
            razorpaySignature: response.razorpay_signature,
          });
          cart.clearCart();
          toast.success("Payment successful!");
          navigate(`/orders/${order.id}`);
        } catch {
          toast.error("Payment verification failed. Please contact support.");
        }
      },
      modal: {
        ondismiss: () => toast.error("Payment cancelled"),
      },
    };
    new window.Razorpay(options).open();
  };

  // Open the Cashfree modal, then let the SERVER decide whether it was paid.
  //
  // Unlike the Razorpay handler above there is no (payment_id, order_id,
  // signature) triple to hand back — Cashfree's SDK returns no client-verifiable
  // proof. So the verify call posts only the order id, and the server reads the
  // captured payment from Cashfree itself. That is why this runs on ANY non-error
  // close rather than on a "success" callback: the customer may well have paid on
  // a sheet that closed untidily, and the only way to know is to ask the gateway.
  //
  // A verify failure is therefore not automatically a payment failure. The order
  // page is authoritative (it polls the real paymentStatus, which the webhook also
  // drives), so we send the customer there rather than claiming it failed.
  const confirmCashfreePayment = async (
    order: Order,
    paymentData: {
      cashfreePaymentSessionId: string;
      cashfreeOrderId: string;
      cashfreeEnv?: string;
      amount: number;
      currency: string;
    },
  ) => {
    await openCashfreeCheckout({
      data: paymentData,
      onSettled: async () => {
        try {
          await apiClient.post(`/payments/order/${order.id}/verify`, {
            cashfreeOrderId: paymentData.cashfreeOrderId,
          });
          cart.clearCart();
          toast.success("Payment successful!");
        } catch {
          // Not "verification failed" — the payment may still be settling, and
          // the webhook completes it server-side either way.
          toast.message("Confirming your payment…");
        }
        navigate(`/orders/${order.id}`);
      },
      onDismiss: () => toast.error("Payment cancelled"),
    });
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
    },
  ) => {
    const stripe = await loadStripeJs(paymentData.publishableKey);
    if (!stripe) {
      toast.error("Stripe failed to load");
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
      toast.error(error.message || "Payment failed");
      return;
    }
    // If confirmPayment doesn't redirect (rare — happens for sync
    // confirmations like some non-3DS card paths), verify inline.
    try {
      await apiClient.post(`/payments/order/${orderId}/verify`, {
        stripePaymentIntentId: paymentData.stripePaymentIntentId,
      });
      cart.clearCart();
      toast.success("Payment successful!");
      navigate(`/orders/${orderId}`);
    } catch {
      toast.error("Payment verification failed. Please contact support.");
    }
  };

  const onAddressSubmit = async (data: AddressFormData) => {
    try {
      const created = await apiClient.post<Address>("/addresses", {
        ...data,
        country: "IN",
        // The coordinates are the point of the search box above: an address
        // without them can never be range-checked, so it can never be ordered to.
        latitude: newAddressCoords?.lat,
        longitude: newAddressCoords?.lon,
        isDefault: false,
      });
      // Refresh the list + select the brand-new one.
      await queryClient.invalidateQueries({ queryKey: ["addresses"] });
      setSelectedAddress(created.id);
      setShowNewAddress(false);
      reset();
      setNewAddressCoords(null);
      toast.success("Address saved");
    } catch {
      toast.error("Failed to save address");
    }
  };

  if (cart.items.length === 0) {
    navigate("/cart");
    return null;
  }

  return (
    <div className="min-h-screen bg-paper py-8">
      <div className="container-app max-w-4xl">
        <h1 className="font-display text-2xl font-semibold text-ink md:text-3xl">
          Checkout
        </h1>

        <div className="mt-8 flex flex-col gap-8 lg:flex-row">
          {/* Main Form */}
          <div className="flex-1 space-y-6">
            {/* Fulfilment choice — shown only when the chef offers more than one
                mode. A single-mode chef gets no pointless toggle. */}
            {fulfillmentModes.length > 1 && (
              <section className="rounded-xl bg-bone p-6 shadow-1">
                <h2 className="text-lg font-semibold text-ink">
                  How would you like it?
                </h2>
                <div
                  role="radiogroup"
                  aria-label="Fulfilment method"
                  className="mt-4 grid grid-cols-2 gap-3"
                >
                  {fulfillmentModes.map((mode) => {
                    const selected = fulfillment === mode;
                    const isPickupMode = mode === "pickup";
                    return (
                      <button
                        type="button"
                        key={mode}
                        role="radio"
                        aria-checked={selected}
                        onClick={() => setFulfillment(mode)}
                        className={`flex min-h-11 items-center gap-3 rounded-lg border p-4 text-left transition-colors ${
                          selected
                            ? "border-herb bg-herb-tint"
                            : "border-mist hover:bg-paper"
                        }`}
                      >
                        {isPickupMode ? (
                          <Store
                            className={`h-5 w-5 flex-shrink-0 ${selected ? "text-herb" : "text-ink-muted"}`}
                            aria-hidden="true"
                          />
                        ) : (
                          <MapPin
                            className={`h-5 w-5 flex-shrink-0 ${selected ? "text-herb" : "text-ink-muted"}`}
                            aria-hidden="true"
                          />
                        )}
                        <span>
                          <span className="block font-medium text-ink">
                            {isPickupMode ? "Pickup" : "Delivery"}
                          </span>
                          <span className="block text-xs text-ink-muted">
                            {isPickupMode
                              ? "Collect from the kitchen — no delivery fee"
                              : "Brought to your address"}
                          </span>
                        </span>
                      </button>
                    );
                  })}
                </div>

                {/* Distance to the kitchen. This is the number that makes the
                    delivery-vs-pickup choice a real decision rather than a guess —
                    it is only knowable once the drop address has coordinates. */}
                {!isPickup && quote?.rangeKnown && (
                  <p className="mt-3 text-sm text-ink-soft tabular-nums">
                    {quote.distanceKm.toFixed(1)} km from{" "}
                    {cart.chef?.businessName ?? "the kitchen"}
                    {quote.maxRadiusKm > 0
                      ? ` · delivers up to ${quote.maxRadiusKm.toFixed(0)} km`
                      : ""}
                  </p>
                )}
                {isPickup && quote?.rangeKnown && quote.distanceKm > 0 && (
                  <p className="mt-3 text-sm text-ink-soft tabular-nums">
                    The kitchen is {quote.distanceKm.toFixed(1)} km from your
                    saved address. You will get the exact address and map pin
                    once the chef accepts.
                  </p>
                )}
              </section>
            )}

            {/* Pickup incentive — only when the customer is on delivery, pickup is
                actually offered, and switching would genuinely save money. */}
            {!isPickup && offersPickup && pickupSaving > 0 && (
              <button
                type="button"
                onClick={() => setFulfillment("pickup")}
                className="flex w-full items-center justify-between gap-3 rounded-xl bg-herb-tint p-4 text-left transition-colors hover:bg-herb-tint/70"
              >
                <span>
                  <span className="block text-sm font-semibold text-herb tabular-nums">
                    Pick up &amp; save{" "}
                    {fp(pickupSaving, { currency: orderCurrency })}
                  </span>
                  <span className="block text-xs text-ink-soft">
                    Collect from the kitchen — no delivery fee.
                  </span>
                </span>
                <span className="text-sm font-semibold text-herb">
                  Switch →
                </span>
              </button>
            )}

            {/* Delivery Address — a pickup order has no drop address. */}
            {!isPickup && (
              <section className="rounded-xl bg-bone p-6 shadow-1">
                <div className="flex items-center justify-between">
                  <h2 className="flex items-center gap-2 text-lg font-semibold text-ink">
                    <MapPin className="h-5 w-5 text-herb" aria-hidden="true" />
                    Delivery Address
                  </h2>
                  <button
                    type="button"
                    onClick={() => setShowNewAddress(!showNewAddress)}
                    className="text-sm text-herb hover:text-herb"
                  >
                    {showNewAddress ? "Cancel" : "Add New"}
                  </button>
                </div>

                {showNewAddress ? (
                  <form
                    onSubmit={handleSubmit(onAddressSubmit)}
                    className="mt-4 space-y-4"
                  >
                    {/* Search first, then refine. This is what supplies the
                      coordinates the order is range-checked against. */}
                    <AddressSearch
                      label="Find your address"
                      hint="Pick your area from the list, then add your flat or house number below."
                      onPick={applySuggestion}
                    />
                    {!newAddressCoords && (
                      <p className="flex items-start gap-2 text-xs text-ink-muted">
                        <AlertTriangle
                          className="mt-0.5 h-3.5 w-3.5 flex-shrink-0"
                          aria-hidden="true"
                        />
                        Choose a suggestion above so we can check the kitchen
                        delivers to you.
                      </p>
                    )}
                    <div className="grid gap-4 sm:grid-cols-2">
                      <div>
                        <label
                          htmlFor="addr-label"
                          className="block text-sm font-medium text-ink-soft"
                        >
                          Label
                        </label>
                        <input
                          id="addr-label"
                          {...register("label")}
                          aria-invalid={!!errors.label || undefined}
                          aria-describedby={
                            errors.label ? "addr-label-err" : undefined
                          }
                          placeholder="Home, Work, etc."
                          className="input-base mt-1"
                        />
                        {errors.label && (
                          <p
                            id="addr-label-err"
                            role="alert"
                            className="mt-1 text-xs text-paprika"
                          >
                            {errors.label.message}
                          </p>
                        )}
                      </div>
                      <div className="sm:col-span-2">
                        <label
                          htmlFor="addr-line1"
                          className="block text-sm font-medium text-ink-soft"
                        >
                          Street Address
                        </label>
                        <input
                          id="addr-line1"
                          {...register("line1")}
                          aria-invalid={!!errors.line1 || undefined}
                          aria-describedby={
                            errors.line1 ? "addr-line1-err" : undefined
                          }
                          placeholder="123 Main Street"
                          className="input-base mt-1"
                        />
                        {errors.line1 && (
                          <p
                            id="addr-line1-err"
                            role="alert"
                            className="mt-1 text-xs text-paprika"
                          >
                            {errors.line1.message}
                          </p>
                        )}
                      </div>
                      <div className="sm:col-span-2">
                        <label
                          htmlFor="addr-line2"
                          className="block text-sm font-medium text-ink-soft"
                        >
                          Apartment, suite, etc. (optional)
                        </label>
                        <input
                          id="addr-line2"
                          {...register("line2")}
                          placeholder="Apt 4B"
                          className="input-base mt-1"
                        />
                      </div>
                      <div>
                        <label
                          htmlFor="addr-city"
                          className="block text-sm font-medium text-ink-soft"
                        >
                          City
                        </label>
                        <input
                          id="addr-city"
                          {...register("city")}
                          aria-invalid={!!errors.city || undefined}
                          aria-describedby={
                            errors.city ? "addr-city-err" : undefined
                          }
                          className="input-base mt-1"
                        />
                        {errors.city && (
                          <p
                            id="addr-city-err"
                            role="alert"
                            className="mt-1 text-xs text-paprika"
                          >
                            {errors.city.message}
                          </p>
                        )}
                      </div>
                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <label
                            htmlFor="addr-state"
                            className="block text-sm font-medium text-ink-soft"
                          >
                            State
                          </label>
                          <input
                            id="addr-state"
                            {...register("state")}
                            aria-invalid={!!errors.state || undefined}
                            aria-describedby={
                              errors.state ? "addr-state-err" : undefined
                            }
                            className="input-base mt-1"
                          />
                          {errors.state && (
                            <p
                              id="addr-state-err"
                              role="alert"
                              className="mt-1 text-xs text-paprika"
                            >
                              {errors.state.message}
                            </p>
                          )}
                        </div>
                        <div>
                          <label
                            htmlFor="addr-postal"
                            className="block text-sm font-medium text-ink-soft"
                          >
                            Postal Code
                          </label>
                          <input
                            id="addr-postal"
                            {...register("postalCode")}
                            aria-invalid={!!errors.postalCode || undefined}
                            aria-describedby={
                              errors.postalCode ? "addr-postal-err" : undefined
                            }
                            className="input-base mt-1"
                          />
                          {errors.postalCode && (
                            <p
                              id="addr-postal-err"
                              role="alert"
                              className="mt-1 text-xs text-paprika"
                            >
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
                            ? "border-herb bg-herb-tint"
                            : "border-mist hover:bg-paper"
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
                            <span className="font-medium text-ink">
                              {address.label}
                            </span>
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
                          <Check
                            className="h-5 w-5 text-herb"
                            aria-hidden="true"
                          />
                        )}
                      </label>
                    ))}
                  </div>
                )}

                {/* Repair path for addresses saved before the web had a geocoder.
                  Every one of them has no coordinates, so without this the
                  customer would have to re-add an address they already have. */}
                {!showNewAddress && addressNeedsLocation && (
                  <div className="mt-4 rounded-lg border border-amber/40 bg-amber-tint p-4">
                    <p className="flex items-start gap-2 text-sm font-medium text-ink">
                      <AlertTriangle
                        className="mt-0.5 h-4 w-4 flex-shrink-0"
                        aria-hidden="true"
                      />
                      This address has no delivery location saved
                    </p>
                    <p className="mt-1 text-sm text-ink-soft">
                      We need it to confirm the kitchen delivers to you. Search
                      for your area below — the address itself stays exactly as
                      it is.
                    </p>
                    <div className="mt-3">
                      <AddressSearch
                        label="Set delivery location"
                        placeholder="Search your street, area or a nearby landmark"
                        onPick={pinExistingAddress}
                      />
                    </div>
                    {isPinning && (
                      <p className="mt-2 text-xs text-ink-muted">
                        Saving location…
                      </p>
                    )}
                  </div>
                )}

                {/* Out of range: the server would reject this order anyway, so say
                  so here rather than letting the customer discover it at payment. */}
                {!addressNeedsLocation && deliveryOutOfRange && (
                  <div className="mt-4 rounded-lg border border-paprika/30 bg-paprika-tint p-4">
                    <p className="flex items-start gap-2 text-sm font-medium text-paprika">
                      <AlertTriangle
                        className="mt-0.5 h-4 w-4 flex-shrink-0"
                        aria-hidden="true"
                      />
                      Outside this kitchen&apos;s delivery range
                    </p>
                    <p className="mt-1 text-sm text-paprika tabular-nums">
                      This address is {(quote?.distanceKm ?? 0).toFixed(1)} km
                      away — beyond the {(quote?.maxRadiusKm ?? 10).toFixed(0)}{" "}
                      km this kitchen delivers. Pick an address closer to the
                      kitchen.
                    </p>
                  </div>
                )}
              </section>
            )}

            {/* Pickup: where the food is collected from. The exact street address
                and map pin are deliberately NOT shown here — they are revealed on
                the order page once the chef accepts (chefTrackCoords in
                handlers/orders.go returns the exact kitchen only for pickup). It
                is a home kitchen, so the address is private until there is a real
                order behind the request. */}
            {isPickup && (
              <section className="rounded-xl bg-bone p-6 shadow-1">
                <h2 className="flex items-center gap-2 text-lg font-semibold text-ink">
                  <Store className="h-5 w-5 text-herb" aria-hidden="true" />
                  Collect from
                </h2>
                <p className="mt-2 text-sm text-ink">
                  {cart.chef?.businessName ?? "The kitchen"}
                </p>
                <p className="mt-1 text-sm text-ink-soft">
                  You&apos;ll get the full address and a map pin on your order
                  page as soon as the chef accepts. No delivery fee —
                  you&apos;re collecting this yourself.
                </p>
              </section>
            )}

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
                      • {w.name} —{" "}
                      {w.conflicts.map((cf) => cf.detail).join(", ")}
                    </li>
                  ))}
                </ul>
                <p className="mt-2 text-xs text-ink-muted">
                  You can still place this order. Review your items or update
                  your dietary profile in your account.
                </p>
              </section>
            )}

            {/* Timing. Two different pickers, and which one you get depends on the
                chef, not the fulfilment mode:
                  · a windowed (restaurant-style) chef keeps the #51 slot picker;
                  · everyone else — the home-tiffin default — gets the suggested
                    time handshake (#709), where the customer PROPOSES a time and
                    the chef confirms or counters at accept.
                The wording changes with the mode (collect vs delivered) but the
                suggestions themselves don't: the food is ready when the chef cooks
                it, and the mode only changes who carries it. */}
            <section className="rounded-xl bg-bone p-6 shadow-1">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-ink">
                <Clock className="h-5 w-5 text-herb" aria-hidden="true" />
                {isPickup ? "Pickup time" : "Delivery time"}
              </h2>

              {/* A bake can't be made to order in the next half hour (#1065). */}
              {bakeryLeadNotice && (
                <p className="mt-2 rounded-lg bg-herb-tint px-3 py-2 text-sm text-ink">
                  {bakeryLeadNotice}
                </p>
              )}

              {useSlotPicker ? (
                <div className="mt-4 space-y-3">
                  {/* ASAP (default) — not offered when a bake needs notice. */}
                  <label
                    className={`${
                      bakeryEarliest !== null ? "hidden" : "flex"
                    } cursor-pointer items-center gap-3 rounded-lg border p-4 ${
                      selectedSlot === null
                        ? "border-herb bg-herb-tint"
                        : "border-mist hover:bg-paper"
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
                      <span className="font-medium text-ink">
                        As soon as possible
                      </span>
                      <p className="text-sm text-ink-muted">
                        {isPickup
                          ? "Estimated 30-45 minutes after the chef accepts. We’ll tell you the moment it’s ready to collect."
                          : "Estimated 30-45 minutes after the chef accepts your order. Actual time depends on chef preparation and the route."}
                      </p>
                    </div>
                  </label>

                  {/* Scheduled slots (#51) */}
                  {availableSlots.map((s) => {
                    const sel =
                      selectedSlot?.slot === s.slot &&
                      selectedSlot?.date === s.date;
                    // Too soon for a bake that needs notice (#1065).
                    const tooSoon =
                      bakeryEarliest !== null &&
                      new Date(s.scheduledFor).getTime() <
                        bakeryEarliest.getTime();
                    return (
                      <label
                        key={`${s.date}-${s.slot}`}
                        className={`flex items-center gap-3 rounded-lg border p-4 ${
                          tooSoon
                            ? "cursor-not-allowed border-mist opacity-40"
                            : sel
                              ? "cursor-pointer border-herb bg-herb-tint"
                              : "cursor-pointer border-mist hover:bg-paper"
                        }`}
                      >
                        <input
                          type="radio"
                          name="time"
                          checked={sel}
                          disabled={tooSoon}
                          onChange={() =>
                            setSelectedSlot({ slot: s.slot, date: s.date })
                          }
                          className="h-4 w-4 text-herb focus-visible:ring-herb"
                        />
                        <div className="flex-1">
                          <span className="font-medium text-ink">
                            {slotDayLabel(s.date)} · {s.label}
                          </span>
                          <p className="text-sm text-ink-muted tabular-nums">
                            {s.window}
                            {s.remaining != null
                              ? ` · ${s.remaining} left`
                              : ""}
                          </p>
                        </div>
                      </label>
                    );
                  })}
                </div>
              ) : (
                <>
                  <p className="mt-1 text-sm text-ink-muted">
                    {isPickup
                      ? "When will you come to collect? It’s a home kitchen — the chef confirms once they accept."
                      : "Suggest when you’d like it. It’s a home kitchen, not a restaurant — the chef confirms or proposes a time when they accept."}
                  </p>

                  <div className="mt-4 space-y-3">
                    {/* As soon as ready — the default, and deliberately the widest
                        option so it reads as the primary choice. */}
                    <label
                      className={`${
                        bakeryEarliest !== null ? "hidden" : "flex"
                      } cursor-pointer items-center gap-3 rounded-lg border p-4 ${
                        requestedTime === null
                          ? "border-herb bg-herb-tint"
                          : "border-mist hover:bg-paper"
                      }`}
                    >
                      <input
                        type="radio"
                        name="time"
                        checked={requestedTime === null}
                        onChange={() => setRequestedTime(null)}
                        className="h-4 w-4 text-herb focus-visible:ring-herb"
                      />
                      <div>
                        <span className="font-medium text-ink">
                          As soon as ready
                        </span>
                        <p className="text-sm text-ink-muted">
                          Chef decides when to start
                        </p>
                      </div>
                    </label>

                    {fulfillmentTimeGroups.map((group) => (
                      <div key={group.key}>
                        <p className="mb-2 text-xs font-medium uppercase tracking-wide text-ink-muted">
                          {group.key}
                        </p>
                        <div className="flex flex-wrap gap-2">
                          {group.times.map((t) => {
                            const sel = requestedTime?.at === t.at;
                            // Too soon for a bake that needs notice (#1065).
                            const tooSoon =
                              bakeryEarliest !== null &&
                              new Date(t.at).getTime() <
                                bakeryEarliest.getTime();
                            return (
                              <button
                                type="button"
                                key={t.at}
                                disabled={tooSoon}
                                onClick={() => setRequestedTime(t)}
                                aria-pressed={sel}
                                aria-label={`${isPickup ? "Pickup" : "Delivery"} around ${t.label}, ${t.day} ${t.meal}`}
                                className={`min-h-11 rounded-lg border px-4 py-2 text-sm tabular-nums transition-colors ${
                                  tooSoon
                                    ? "border-mist text-ink-soft opacity-40"
                                    : sel
                                      ? "border-herb bg-herb-tint font-medium text-herb"
                                      : "border-mist text-ink-soft hover:bg-paper"
                                }`}
                              >
                                {t.label}
                              </button>
                            );
                          })}
                        </div>
                      </div>
                    ))}

                    {fulfillmentTimeGroups.length === 0 && (
                      <p className="text-sm text-ink-muted">
                        {isPickup
                          ? "No specific pickup times to suggest right now — your order will be ready to collect as soon as the chef finishes."
                          : "No specific times to suggest right now — your order will be sent as soon as it’s ready."}
                      </p>
                    )}
                  </div>
                </>
              )}
            </section>

            {/* Payment */}
            <section className="rounded-xl bg-bone p-6 shadow-1">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-ink">
                <Shield className="h-5 w-5 text-herb" aria-hidden="true" />
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
                  <p className="text-sm font-medium text-ink">
                    {gatewayName
                      ? `Powered by ${gatewayName}`
                      : "Secure payment"}
                  </p>
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
                  <div className="mb-1 font-medium text-ink">
                    Payment &amp; refund summary
                  </div>
                  <p>
                    Payments are processed by{" "}
                    {gatewayName
                      ? `${gatewayName} (RBI-licensed payment aggregator)`
                      : "an RBI-licensed payment aggregator"}
                    . Tesserix Pty Ltd (operator of Fe3dr) facilitates the
                    transaction; order proceeds go to your chef minus the
                    platform commission.
                  </p>
                </div>
                <div className="space-y-2">
                  <div className="flex items-start gap-2">
                    <Clock
                      className="mt-0.5 h-4 w-4 flex-shrink-0 text-herb"
                      aria-hidden="true"
                    />
                    <p>
                      Refunds return to your original payment method within{" "}
                      <strong>7 working days</strong> per RBI Payment Aggregator
                      Master Direction §8.
                    </p>
                  </div>
                  <div className="flex items-start gap-2">
                    <FileText
                      className="mt-0.5 h-4 w-4 flex-shrink-0 text-herb"
                      aria-hidden="true"
                    />
                    <p>
                      See our{" "}
                      <Link to="/refund" className="text-herb hover:underline">
                        Refund Policy
                      </Link>{" "}
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
                {TIP_PRESETS.map((amount) => {
                  const selected = tip === amount && !customTip;
                  return (
                    <button
                      type="button"
                      key={amount}
                      onClick={() => {
                        setCustomTip("");
                        setTip(amount);
                      }}
                      aria-pressed={selected}
                      className={`rounded-lg px-4 py-2 tabular-nums transition-colors ${
                        selected
                          ? "bg-herb text-paper"
                          : "bg-mist text-ink-soft hover:bg-mist-strong"
                      }`}
                    >
                      {amount === 0
                        ? "No tip"
                        : fp(amount, { currency: orderCurrency })}
                    </button>
                  );
                })}
                <input
                  type="text"
                  inputMode="numeric"
                  placeholder="Custom"
                  value={customTip}
                  onChange={(e) => {
                    // Digits only — a tip is whole rupees, and the server floors
                    // a negative one anyway.
                    const digits = e.target.value.replace(/[^0-9]/g, "");
                    setCustomTip(digits);
                    setTip(Math.min(MAX_TIP, Number(digits) || 0));
                  }}
                  maxLength={5}
                  aria-label="Custom tip amount"
                  className={`w-24 rounded-lg border px-3 py-2 text-center tabular-nums ${
                    customTip ? "border-herb bg-herb-tint" : "border-mist"
                  }`}
                />
              </div>
            </section>

            {/* Pay with your credits — wallet + loyalty. Renders only when at
                least one rail is live and has something to spend; the server
                decides that, so a stale build can't disagree with the API about
                whether the feature exists. */}
            {creditQuote && (
              <CheckoutCredits
                quote={creditQuote}
                useWallet={credit.useWallet}
                useLoyalty={credit.useLoyalty}
                currency={orderCurrency}
                onChange={setCredit}
              />
            )}

            {/* Special Instructions */}
            <section className="rounded-xl bg-bone p-6 shadow-1">
              <h2 className="text-lg font-semibold text-ink">
                Special Instructions
              </h2>
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
                      onError={(e) => {
                        e.currentTarget.style.display = "none";
                      }}
                    />
                  )}
                  <span className="font-medium text-ink">
                    {cart.chef.businessName}
                  </span>
                </div>
              )}

              {/* Items. Each line carries a remove control, and the block a way
                  back to the cart.
                  
                  Checkout used to be a one-way door: the summary listed what you
                  were about to pay for but offered no way to change it, and the
                  header cart link is hidden on this route — so a customer who
                  changed their mind had nowhere to go but the browser's back
                  button. Removing the last line empties the cart, which sends
                  them back to it rather than leaving a checkout for nothing. */}
              <div className="mt-4 space-y-2 border-b pb-4">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-medium uppercase tracking-wide text-ink-muted">
                    Items
                  </span>
                  <Link
                    to="/cart"
                    className="text-xs font-medium text-primary hover:underline"
                  >
                    Edit cart
                  </Link>
                </div>
                {cart.items.map((item) => (
                  <div
                    key={item.id}
                    className="flex items-center justify-between gap-2 text-sm"
                  >
                    <span className="min-w-0 flex-1 text-ink-soft">
                      {item.quantity}x {item.name}
                    </span>
                    <span className="text-ink">
                      {fp(item.price * item.quantity, {
                        currency: orderCurrency,
                      })}
                    </span>
                    <button
                      type="button"
                      onClick={() => {
                        cart.removeItem(item.id);
                        toast.success(`Removed ${item.name}`);
                      }}
                      aria-label={`Remove ${item.name}`}
                      className="shrink-0 rounded p-1 text-ink-muted hover:text-destructive"
                    >
                      <Trash2 className="h-4 w-4" aria-hidden="true" />
                    </button>
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
                  <span>{getFeeRowLabel(fulfillment)}</span>
                  {/* Free reads as a benefit, not a zero — and pickup is always
                      free, so rendering "₹0.00" there is just noise. */}
                  {deliveryFee === 0 ? (
                    <span className="font-medium text-herb">Free</span>
                  ) : (
                    <span className="tabular-nums">
                      {fp(deliveryFee, { currency: orderCurrency })}
                    </span>
                  )}
                </div>
                {/* Why the fee is what it is. The free-zone line explains a ₹0
                    delivery, and the surge line names the actual conditions —
                    a bare multiplier explains nothing to a hungry customer. */}
                {!isPickup && selfDeliveryBreakdown?.withinFreeZone ? (
                  <p className="text-xs text-herb">
                    Within the chef's free-delivery radius — no delivery charge.
                  </p>
                ) : null}
                {!isPickup && surgeReason && deliveryFee > 0 ? (
                  <p className="text-xs text-ink-soft">{surgeReason}</p>
                ) : null}
                <div className="flex justify-between text-ink-soft">
                  <span>Platform fee</span>
                  <span>{fp(platformFee, { currency: orderCurrency })}</span>
                </div>
                {discount > 0 && (
                  <div className="flex justify-between text-herb">
                    <span>
                      Promo{cart.promoCode ? ` (${cart.promoCode})` : ""}
                    </span>
                    <span>−{fp(discount, { currency: orderCurrency })}</span>
                  </div>
                )}
                {/* Tax rows exactly as the server split them — the same lines
                    this order's receipt and invoice PDF will carry.
                    TODO(CW-01e): backend to attach the HSN/SAC code per CGST
                    Act 2017 §31. */}
                {taxLines.map((row) => (
                  <div
                    key={row.code}
                    className="flex justify-between text-ink-soft"
                  >
                    <span>{row.label}</span>
                    <span className="tabular-nums">
                      {fp(row.amount, { currency: orderCurrency })}
                    </span>
                  </div>
                ))}
                {tip > 0 && (
                  <div className="flex justify-between text-ink-soft">
                    <span>Tip for the chef</span>
                    <span className="tabular-nums">
                      {fp(tip, { currency: orderCurrency })}
                    </span>
                  </div>
                )}
                {walletApplied > 0 && (
                  <div className="flex justify-between text-herb">
                    <span>Wallet credit</span>
                    <span className="tabular-nums">
                      −{fp(walletApplied, { currency: orderCurrency })}
                    </span>
                  </div>
                )}
                {loyaltyApplied > 0 && (
                  <div className="flex justify-between text-herb">
                    <span>Loyalty points</span>
                    <span className="tabular-nums">
                      −{fp(loyaltyApplied, { currency: orderCurrency })}
                    </span>
                  </div>
                )}
              </div>

              <div className="mt-4 flex justify-between border-t pt-4 text-lg font-semibold">
                <span>{creditApplied > 0 ? "To pay" : "Total"}</span>
                <span className="tabular-nums">
                  {fp(payable, { currency: orderCurrency })}
                </span>
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
                  I agree to the{" "}
                  <Link to="/terms" className="text-herb hover:underline">
                    Terms of Service
                  </Link>{" "}
                  and{" "}
                  <Link to="/refund" className="text-herb hover:underline">
                    Refund Policy
                  </Link>{" "}
                  for this order.
                </span>
              </label>

              <Button
                variant="primary"
                size="lg"
                fullWidth
                isLoading={isProcessing}
                onClick={handlePlaceOrder}
                disabled={
                  isProcessing ||
                  // Pickup needs no address — requiring one would make the CTA
                  // permanently dead for a customer who has never saved one.
                  (!isPickup && !selectedAddress) ||
                  !acceptedTerms ||
                  addressNeedsLocation ||
                  deliveryOutOfRange ||
                  // A bake ordered sooner than its lead time is rejected by the
                  // server — block it here, where the fix is one tap away (#1065).
                  bakeryTimeTooSoon
                }
                rightIcon={
                  !isProcessing ? (
                    <ChevronRight aria-hidden="true" className="h-5 w-5" />
                  ) : undefined
                }
                className="mt-4"
              >
                {isProcessing
                  ? "Placing Order..."
                  : `Place Order - ${fp(payable, { currency: orderCurrency })}`}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
