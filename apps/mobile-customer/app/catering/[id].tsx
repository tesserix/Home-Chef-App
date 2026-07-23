// Catering request detail (#55) — request summary + chef quotes. The customer
// reviews quotes, accepts/declines, and pays the deposit to confirm. Mirrors the
// web CateringQuotesPage detail panel.

import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Pressable,
  ScrollView,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { Calendar, Check, ChevronLeft, MapPin, Users } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import {
  useAcceptCateringQuote,
  useCancelCateringRequest,
  useCateringRequest,
  useCreateCateringDeposit,
  useDeclineCateringQuote,
  type CateringQuote,
  type CateringRequest,
} from '../../hooks/useCatering';
import { friendlyErrorMessage } from '../../lib/errors';

// Android ripple tints — translucent tokens, never a new literal colour.
const ICON_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const CANVAS_RIPPLE = `${customerColors.canvas}33`;
const GHOST_RIPPLE = `${customerColors.charcoal.DEFAULT}0F`;

function money(n: number): string {
  return `₹${Math.round(n).toLocaleString('en-IN')}`;
}

function fmtDate(d?: string): string {
  if (!d) return '';
  return new Date(d).toLocaleDateString(undefined, {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

// ─── Status banner ─────────────────────────────────────────────────────────────

function StatusBanner({ request }: { request: CateringRequest }) {
  const map: Record<string, { label: string; bg: string; text: string }> = {
    open: { label: 'Awaiting quotes', bg: customerColors.surface.soft, text: customerColors.charcoal.DEFAULT },
    quoted: { label: 'Quotes received', bg: customerColors.coral.tint, text: customerColors.coral.pressed },
    accepted: { label: 'Quote accepted', bg: customerColors.coral.tint, text: customerColors.coral.pressed },
    confirmed: { label: 'Booking confirmed', bg: customerColors.success.tint, text: customerColors.success.DEFAULT },
    completed: { label: 'Completed', bg: customerColors.success.tint, text: customerColors.success.DEFAULT },
    cancelled: { label: 'Cancelled', bg: customerColors.surface.soft, text: customerColors.charcoal.soft },
  };
  const c = map[request.status] ?? map.open;
  return (
    <View className="self-start rounded-full px-3 py-1" style={{ backgroundColor: c.bg }}>
      <Text className="text-xs font-semibold" style={{ color: c.text }}>
        {c.label}
      </Text>
    </View>
  );
}

// ─── Quote card ─────────────────────────────────────────────────────────────────

function QuoteCard({
  quote,
  request,
  onAccept,
  onDecline,
  busy,
}: {
  quote: CateringQuote;
  request: CateringRequest;
  onAccept: () => void;
  onDecline: () => void;
  busy: boolean;
}) {
  const isAccepted = quote.status === 'accepted';
  const actionable = quote.status === 'pending' && (request.status === 'open' || request.status === 'quoted');
  const includes = [
    quote.includesSetup && 'Setup',
    quote.includesServing && 'Serving',
    quote.includesCleanup && 'Cleanup',
    quote.includesEquipment && 'Equipment',
  ].filter(Boolean) as string[];

  return (
    <View
      className={`bg-canvas rounded-xl border p-4 mb-3 ${isAccepted ? 'border-coral' : 'border-hairline'}`}
    >
      {/* Chef + total */}
      <View className="flex-row items-start justify-between">
        <View className="flex-1 mr-3">
          <Text className="text-base font-bold text-charcoal font-display">
            {quote.chef?.businessName ?? 'Chef'}
          </Text>
          {quote.chef?.rating != null && quote.chef.rating > 0 ? (
            <Text className="text-xs text-charcoal-soft mt-0.5">
              ★ {quote.chef.rating.toFixed(1)} ({quote.chef.totalReviews ?? 0})
            </Text>
          ) : null}
        </View>
        <View className="items-end">
          <Text className="text-lg font-bold text-charcoal" style={{ fontVariant: ['tabular-nums'] }}>
            {money(quote.totalPrice)}
          </Text>
          <Text className="text-xs text-charcoal-soft" style={{ fontVariant: ['tabular-nums'] }}>
            {money(quote.pricePerPerson)}/person
          </Text>
        </View>
      </View>

      {/* Proposed menu */}
      {quote.proposedMenu ? (
        <Text className="text-sm text-charcoal-soft mt-3 leading-5">{quote.proposedMenu}</Text>
      ) : null}

      {/* Menu items */}
      {quote.menuItems?.length ? (
        <View className="flex-row flex-wrap gap-1.5 mt-2">
          {quote.menuItems.map((m, i) => (
            <View key={`${m}-${i}`} className="bg-surface-soft rounded-full px-2.5 py-1">
              <Text className="text-xs text-charcoal-soft">{m}</Text>
            </View>
          ))}
        </View>
      ) : null}

      {/* Includes */}
      {includes.length ? (
        <Text className="text-xs text-charcoal-soft mt-2">Includes: {includes.join(' · ')}</Text>
      ) : null}

      {/* Deposit + validity */}
      <Text className="text-xs text-charcoal-soft mt-2" style={{ fontVariant: ['tabular-nums'] }}>
        Deposit to confirm: {money(quote.depositAmount)}
        {quote.validUntil ? ` · valid till ${fmtDate(quote.validUntil)}` : ''}
      </Text>

      {quote.notes ? (
        <Text className="text-xs text-charcoal-soft mt-1 italic">“{quote.notes}”</Text>
      ) : null}

      {/* Actions */}
      {isAccepted ? (
        <View className="flex-row items-center gap-1.5 mt-3">
          <Check size={16} color={customerColors.coral.DEFAULT} />
          <Text className="text-sm font-semibold text-coral">Accepted</Text>
        </View>
      ) : actionable ? (
        <View className="flex-row gap-2 mt-3">
          <Pressable
            className="flex-1"
            onPress={onAccept}
            disabled={busy}
            accessibilityRole="button"
            accessibilityLabel="Accept quote"
            android_ripple={busy ? undefined : { color: CANVAS_RIPPLE, borderless: false }}
          >
            <View className={`rounded-lg min-h-[44px] items-center justify-center bg-coral ${busy ? 'opacity-60' : ''}`}>
              <Text className="text-canvas font-semibold text-sm">Accept</Text>
            </View>
          </Pressable>
          <Pressable
            className="flex-1"
            onPress={onDecline}
            disabled={busy}
            accessibilityRole="button"
            accessibilityLabel="Decline quote"
            android_ripple={busy ? undefined : { color: GHOST_RIPPLE, borderless: false }}
          >
            <View className="rounded-lg min-h-[44px] items-center justify-center border border-hairline bg-canvas">
              <Text className="text-charcoal-soft font-semibold text-sm">Decline</Text>
            </View>
          </Pressable>
        </View>
      ) : quote.status === 'rejected' ? (
        <Text className="text-xs text-charcoal-soft mt-3">Declined</Text>
      ) : null}
    </View>
  );
}

// ─── Screen ───────────────────────────────────────────────────────────────────

export default function CateringDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data, isLoading } = useCateringRequest(id);
  const accept = useAcceptCateringQuote(id);
  const decline = useDeclineCateringQuote(id);
  const cancel = useCancelCateringRequest();
  const createDeposit = useCreateCateringDeposit();
  const [paying, setPaying] = useState(false);

  const request = data?.data;
  const quotes = data?.quotes ?? [];
  const busy = accept.isPending || decline.isPending;

  function confirmAccept(quote: CateringQuote) {
    Alert.alert(
      'Accept this quote?',
      `${quote.chef?.businessName ?? 'This chef'} · ${money(quote.totalPrice)}. You'll then pay a ${money(quote.depositAmount)} deposit to confirm.`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Accept',
          onPress: () =>
            accept.mutate(quote.id, {
              onError: (err) => Alert.alert('Could not accept', friendlyErrorMessage(err, 'Please try again.')),
            }),
        },
      ],
    );
  }

  function payDeposit() {
    if (!request) return;
    setPaying(true);
    createDeposit.mutate(request.id, {
      onSuccess: (d) => {
        setPaying(false);
        router.push({
          pathname: '/payment/checkout',
          params: {
            kind: 'catering',
            cateringId: request.id,
            orderId: request.id,
            razorpayOrderId: d.razorpayOrderId,
            razorpayKeyId: d.razorpayKeyId,
            amount: String(d.amount),
            currency: d.currency,
          },
        });
      },
      onError: (err) => {
        setPaying(false);
        Alert.alert('Deposit unavailable', friendlyErrorMessage(err, 'Deposits aren’t available yet. Please try again later.'));
      },
    });
  }

  function confirmCancel() {
    if (!request) return;
    Alert.alert('Cancel this request?', 'Chefs will no longer be able to quote.', [
      { text: 'Keep it', style: 'cancel' },
      {
        text: 'Cancel request',
        style: 'destructive',
        onPress: () =>
          cancel.mutate(request.id, {
            onError: (err) => Alert.alert('Could not cancel', friendlyErrorMessage(err, 'Please try again.')),
          }),
      },
    ]);
  }

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      {/* Header */}
      <View className="flex-row items-center px-4 pt-3 pb-2 gap-3">
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: ICON_RIPPLE, borderless: true }}
        >
          <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text className="text-xl font-bold text-charcoal font-display flex-1">Catering request</Text>
      </View>

      {isLoading || !request ? (
        <View className="px-4 pt-1">
          <View className="bg-canvas rounded-xl border border-hairline p-4">
            <View className="h-5 rounded bg-hairline" style={{ width: '50%' }} />
            <View className="h-3 rounded bg-hairline mt-3" style={{ width: '70%' }} />
            <View className="h-3 rounded bg-hairline mt-2" style={{ width: '40%' }} />
            <View className="h-3 rounded bg-hairline mt-2" style={{ width: '80%' }} />
          </View>
        </View>
      ) : (
        <ScrollView contentContainerStyle={{ padding: 16, paddingBottom: 40 }}>
          {/* Summary */}
          <View className="bg-canvas rounded-xl border border-hairline p-4">
            <View className="flex-row items-center justify-between mb-2">
              <Text className="text-lg font-bold text-charcoal font-display flex-1 mr-2">
                {request.eventType}
              </Text>
              <StatusBanner request={request} />
            </View>
            <View className="gap-1.5 mt-1">
              <View className="flex-row items-center gap-2">
                <Calendar size={15} color={customerColors.charcoal.soft} />
                <Text className="text-sm text-charcoal-soft">
                  {fmtDate(request.eventDate)}
                  {request.eventTime ? ` · ${request.eventTime}` : ''}
                </Text>
              </View>
              <View className="flex-row items-center gap-2">
                <Users size={15} color={customerColors.charcoal.soft} />
                <Text className="text-sm text-charcoal-soft">{request.guestCount} guests</Text>
              </View>
              <View className="flex-row items-center gap-2">
                <MapPin size={15} color={customerColors.charcoal.soft} />
                <Text className="text-sm text-charcoal-soft flex-1">
                  {[request.venueName, request.addressLine1, request.city, request.state]
                    .filter(Boolean)
                    .join(', ')}
                </Text>
              </View>
            </View>
            {request.cuisineTypes?.length ? (
              <Text className="text-xs text-charcoal-soft mt-2">
                Cuisine: {request.cuisineTypes.join(', ')}
              </Text>
            ) : null}
            {request.dietaryNeeds?.length ? (
              <Text className="text-xs text-charcoal-soft mt-1">
                Dietary: {request.dietaryNeeds.join(', ')}
              </Text>
            ) : null}
            {request.menuStyle ? (
              <Text className="text-xs text-charcoal-soft mt-1">Menu style: {request.menuStyle}</Text>
            ) : null}
            {request.budget != null && request.budget > 0 ? (
              <Text className="text-sm text-charcoal mt-2" style={{ fontVariant: ['tabular-nums'] }}>
                Budget: {money(request.budget)}
              </Text>
            ) : null}
          </View>

          {/* Deposit CTA — accepted, awaiting deposit */}
          {request.status === 'accepted' && request.depositStatus === 'pending' ? (
            <View className="bg-coral-tint rounded-xl p-4 mt-4">
              <Text className="text-sm font-semibold text-charcoal">Confirm your booking</Text>
              <Text className="text-sm text-charcoal-soft mt-1 tabular-nums">
                Pay a {money(request.depositAmount ?? 0)} deposit to lock in your chef. The balance is
                settled after the event.
              </Text>
              <Pressable
                onPress={payDeposit}
                disabled={paying || createDeposit.isPending}
                accessibilityRole="button"
                accessibilityLabel="Pay deposit"
                android_ripple={paying || createDeposit.isPending ? undefined : { color: CANVAS_RIPPLE, borderless: false }}
              >
                <View className={`mt-3 rounded-lg min-h-[48px] items-center justify-center bg-coral ${paying || createDeposit.isPending ? 'opacity-60' : ''}`}>
                  {paying || createDeposit.isPending ? (
                    <ActivityIndicator size="small" color={customerColors.canvas} />
                  ) : (
                    <Text className="text-canvas font-semibold text-base tabular-nums">
                      Pay deposit · {money(request.depositAmount ?? 0)}
                    </Text>
                  )}
                </View>
              </Pressable>
            </View>
          ) : null}

          {/* Confirmed banner */}
          {request.status === 'confirmed' || request.status === 'completed' ? (
            <View className="bg-success-tint rounded-xl p-4 mt-4 flex-row items-center gap-2">
              <Check size={18} color={customerColors.success.DEFAULT} />
              <Text className="text-sm font-medium text-charcoal flex-1">
                {request.status === 'completed'
                  ? 'This event is complete. Thanks for using Fe3dr catering!'
                  : 'Booking confirmed — your chef has been notified.'}
              </Text>
            </View>
          ) : null}

          {/* Quotes */}
          <Text className="text-base font-bold text-charcoal font-display mt-6 mb-2">
            {quotes.length > 0 ? `Quotes (${quotes.length})` : 'Quotes'}
          </Text>
          {quotes.length === 0 ? (
            <Text className="text-sm text-charcoal-soft">
              No quotes yet. Chefs in your area will send proposals soon.
            </Text>
          ) : (
            quotes.map((q) => (
              <QuoteCard
                key={q.id}
                quote={q}
                request={request}
                busy={busy}
                onAccept={() => confirmAccept(q)}
                onDecline={() =>
                  decline.mutate(q.id, {
                    onError: (err) => Alert.alert('Could not decline', friendlyErrorMessage(err, 'Please try again.')),
                  })
                }
              />
            ))
          )}

          {/* Cancel request */}
          {['open', 'quoted', 'accepted'].includes(request.status) ? (
            <Pressable onPress={confirmCancel} disabled={cancel.isPending} accessibilityRole="button" accessibilityLabel="Cancel request" className="mt-6">
              <Text className="text-center text-sm text-charcoal-soft underline">Cancel this request</Text>
            </Pressable>
          ) : null}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
