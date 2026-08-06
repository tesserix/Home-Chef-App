import React, { useEffect, useMemo, useRef, useState } from 'react';
import { ActivityIndicator, Platform, Pressable, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Stack, useLocalSearchParams, useRouter } from 'expo-router';
import { Star } from 'lucide-react-native';
import { useFormDraft } from '@homechef/mobile-shared/hooks';
import { customerColors } from '@homechef/mobile-shared/theme';
import { KeyboardAwareScrollView, useAlert } from '@homechef/mobile-shared/ui';
import { useOrder } from '../../../hooks/useOrderHistory';
import { useCreateReview } from '../../../hooks/useCreateReview';
import { useOrderReview } from '../../../hooks/useOrderReview';
import { friendlyErrorMessage } from '../../../lib/errors';

// Android ripple tint for the star targets — translucent token, never a new
// literal colour.
const STAR_RIPPLE = `${customerColors.coral.DEFAULT}1F`;

function StarRow({
  label,
  value,
  onChange,
  required,
  readOnly,
}: {
  label: string;
  value: number;
  onChange?: (v: number) => void;
  required?: boolean;
  readOnly?: boolean;
}) {
  if (readOnly) {
    return (
      <View
        style={{ flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', paddingVertical: 8 }}
        accessibilityLabel={`${label}: ${value} out of 5`}
      >
        <Text style={{ flex: 1, fontFamily: 'Inter', fontSize: 15, color: customerColors.charcoal.DEFAULT }}>
          {label}
        </Text>
        <View style={{ flexDirection: 'row', gap: 4 }} accessibilityElementsHidden importantForAccessibility="no-hide-descendants">
          {[1, 2, 3, 4, 5].map((n) => (
            <Star
              key={n}
              size={18}
              color={customerColors.coral.DEFAULT}
              fill={n <= value ? customerColors.coral.DEFAULT : 'transparent'}
            />
          ))}
        </View>
      </View>
    );
  }
  return (
    <View
      style={{
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        paddingVertical: 10,
      }}
    >
      <Text style={{ flex: 1, fontFamily: 'Inter', fontSize: 15, color: customerColors.charcoal.DEFAULT }}>
        {label}
        {required ? <Text style={{ color: customerColors.destructive.DEFAULT }}> *</Text> : null}
      </Text>
      <View style={{ flexDirection: 'row' }} accessibilityRole="radiogroup" accessibilityLabel={label}>
        {[1, 2, 3, 4, 5].map((n) => (
          <Pressable
            key={n}
            onPress={() => onChange?.(n)}
            accessibilityRole="radio"
            accessibilityState={{ selected: value === n }}
            accessibilityLabel={`${n} star${n > 1 ? 's' : ''}`}
            // 44pt touch target (R5) — the glyph itself is 26px, so the box
            // reserves the extra space rather than relying on overlap-prone
            // hitSlop between adjacent stars.
            style={{ width: 44, height: 44, alignItems: 'center', justifyContent: 'center' }}
            android_ripple={{ color: STAR_RIPPLE, borderless: true, radius: 20 }}
          >
            {({ pressed }) => (
              <View style={pressed && Platform.OS === 'ios' ? { opacity: 0.6 } : undefined}>
                <Star
                  size={26}
                  color={customerColors.coral.DEFAULT}
                  fill={n <= value ? customerColors.coral.DEFAULT : 'transparent'}
                />
              </View>
            )}
          </Pressable>
        ))}
      </View>
    </View>
  );
}

export default function OrderReviewScreen() {
  const { showAlert } = useAlert();
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data, isLoading } = useOrder(id ?? '');
  const order = data?.data;
  const { data: existingReview, isLoading: reviewLoading } = useOrderReview(id ?? '');
  const createReview = useCreateReview();
  const { ready, draft, saveDraft, clearDraft } = useFormDraft<string>(
    `review-${id ?? 'unknown'}`,
  );

  const [overall, setOverall] = useState(0);
  const [food, setFood] = useState(0);
  const [delivery, setDelivery] = useState(0);
  const [value, setValue] = useState(0);
  const [packaging, setPackaging] = useState(0);
  const [hygiene, setHygiene] = useState(0);
  const [dishRatings, setDishRatings] = useState<Record<string, number>>({});
  const [title, setTitle] = useState('');
  const [comment, setComment] = useState('');

  // Persist the free-text comment (the typing-heavy field) so a background/kill
  // doesn't wipe it. Restored once the async load resolves; cleared on submit.
  const restored = useRef(false);
  useEffect(() => {
    if (!ready || restored.current || draft == null) return;
    restored.current = true;
    setComment(draft);
  }, [ready, draft]);
  useEffect(() => {
    if (!ready) return;
    saveDraft(comment);
  }, [ready, comment, saveDraft]);

  // One row per distinct dish in the order (#145).
  const dishes = useMemo(() => {
    const seen = new Map<string, string>();
    for (const it of order?.items ?? []) {
      if (!seen.has(it.menuItemId)) seen.set(it.menuItemId, it.name);
    }
    return Array.from(seen, ([menuItemId, name]) => ({ menuItemId, name }));
  }, [order]);

  // A submitted dish rating is stored by menu item id; the name only lives on
  // the order, so pair them back up for the read state.
  const reviewedDishes = useMemo(() => {
    const names = new Map(dishes.map((d) => [d.menuItemId, d.name]));
    return (existingReview?.dishRatings ?? []).map((r) => ({
      menuItemId: r.menuItemId,
      name: names.get(r.menuItemId) ?? 'Dish',
      rating: r.rating,
    }));
  }, [dishes, existingReview]);

  function handleSubmit() {
    if (overall < 1) {
      showAlert('Add a rating', 'Please give an overall rating before submitting.');
      return;
    }
    createReview.mutate(
      {
        orderId: id!,
        overallRating: overall,
        foodRating: food || undefined,
        deliveryRating: delivery || undefined,
        valueRating: value || undefined,
        packagingRating: packaging || undefined,
        hygieneRating: hygiene || undefined,
        title,
        comment,
        dishRatings: Object.entries(dishRatings).map(([menuItemId, rating]) => ({ menuItemId, rating })),
      },
      {
        onSuccess: () => {
          clearDraft();
          showAlert('Thanks!', 'Your review has been submitted.');
          router.back();
        },
        onError: (e) => showAlert('Could not submit', friendlyErrorMessage(e, 'Please try again.')),
      }
    );
  }

  const inputStyle = {
    backgroundColor: customerColors.surface.soft,
    borderRadius: 12,
    paddingHorizontal: 14,
    paddingVertical: 12,
    fontFamily: 'Inter',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  } as const;

  return (
    <SafeAreaView edges={['bottom']} style={{ flex: 1, backgroundColor: customerColors.canvas }}>
      <Stack.Screen options={{ title: existingReview ? 'Your review' : 'Leave a review' }} />
      {isLoading || reviewLoading ? (
        <ActivityIndicator style={{ marginTop: 32 }} color={customerColors.charcoal.soft} />
      ) : !order ? (
        <Text style={{ textAlign: 'center', marginTop: 32, fontFamily: 'Inter', color: customerColors.charcoal.soft }}>
          We couldn’t find that order.
        </Text>
      ) : order.status !== 'delivered' ? (
        <Text style={{ textAlign: 'center', marginTop: 32, fontFamily: 'Inter', color: customerColors.charcoal.soft }}>
          You can review this order once it’s delivered.
        </Text>
      ) : existingReview ? (
        // An order can carry one review (reviews.order_id is unique), so once it
        // exists the form would only ever 409 — show what was submitted instead.
        <KeyboardAwareScrollView contentContainerStyle={{ padding: 20, paddingBottom: 40 }}>
          <Text style={{ fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, marginBottom: 16 }}>
            You reviewed order #{order.orderNumber}.
          </Text>

          <View style={{ borderWidth: 1, borderColor: customerColors.hairline, borderRadius: 16, padding: 16 }}>
            <StarRow label="Overall" value={existingReview.overallRating} readOnly />
            {existingReview.foodRating ? (
              <StarRow label="Food quality" value={existingReview.foodRating} readOnly />
            ) : null}
            {existingReview.deliveryRating ? (
              <StarRow label="Delivery" value={existingReview.deliveryRating} readOnly />
            ) : null}
            {existingReview.valueRating ? (
              <StarRow label="Value for money" value={existingReview.valueRating} readOnly />
            ) : null}
            {existingReview.packagingRating ? (
              <StarRow label="Packaging" value={existingReview.packagingRating} readOnly />
            ) : null}
            {existingReview.hygieneRating ? (
              <StarRow label="Hygiene" value={existingReview.hygieneRating} readOnly />
            ) : null}
          </View>

          {reviewedDishes.length > 0 && (
            <View style={{ borderWidth: 1, borderColor: customerColors.hairline, borderRadius: 16, padding: 16, marginTop: 16 }}>
              <Text style={{ fontFamily: 'Inter', fontSize: 13, fontWeight: '600', color: customerColors.charcoal.DEFAULT, marginBottom: 4 }}>
                Your dish ratings
              </Text>
              {reviewedDishes.map((d) => (
                <StarRow key={d.menuItemId} label={d.name} value={d.rating} readOnly />
              ))}
            </View>
          )}

          {(existingReview.title || existingReview.comment) && (
            <View style={{ borderWidth: 1, borderColor: customerColors.hairline, borderRadius: 16, padding: 16, marginTop: 16, gap: 6 }}>
              {existingReview.title ? (
                <Text style={{ fontFamily: 'Inter-SemiBold', fontSize: 15, color: customerColors.charcoal.DEFAULT }}>
                  {existingReview.title}
                </Text>
              ) : null}
              {existingReview.comment ? (
                <Text style={{ fontFamily: 'Inter', fontSize: 15, lineHeight: 22, color: customerColors.charcoal.DEFAULT }}>
                  {existingReview.comment}
                </Text>
              ) : null}
            </View>
          )}

          {existingReview.chefResponse ? (
            <View style={{ backgroundColor: customerColors.surface.soft, borderRadius: 16, padding: 16, marginTop: 16, gap: 6 }}>
              <Text style={{ fontFamily: 'Inter-SemiBold', fontSize: 13, color: customerColors.charcoal.soft }}>
                Reply from the chef
              </Text>
              <Text style={{ fontFamily: 'Inter', fontSize: 15, lineHeight: 22, color: customerColors.charcoal.DEFAULT }}>
                {existingReview.chefResponse}
              </Text>
            </View>
          ) : null}
        </KeyboardAwareScrollView>
      ) : (
        <KeyboardAwareScrollView contentContainerStyle={{ padding: 20, paddingBottom: 40 }}>
          <Text style={{ fontFamily: 'Inter', fontSize: 13, color: customerColors.charcoal.soft, marginBottom: 16 }}>
            How was order #{order.orderNumber}?
          </Text>

          <View style={{ borderWidth: 1, borderColor: customerColors.hairline, borderRadius: 16, padding: 16 }}>
            <StarRow label="Overall" value={overall} onChange={setOverall} required />
            <StarRow label="Food quality" value={food} onChange={setFood} />
            <StarRow label="Delivery" value={delivery} onChange={setDelivery} />
            <StarRow label="Value for money" value={value} onChange={setValue} />
            <StarRow label="Packaging" value={packaging} onChange={setPackaging} />
            <StarRow label="Hygiene" value={hygiene} onChange={setHygiene} />
          </View>

          {dishes.length > 0 && (
            <View style={{ borderWidth: 1, borderColor: customerColors.hairline, borderRadius: 16, padding: 16, marginTop: 16 }}>
              <Text style={{ fontFamily: 'Inter', fontSize: 13, fontWeight: '600', color: customerColors.charcoal.DEFAULT, marginBottom: 4 }}>
                Rate the dishes
              </Text>
              {dishes.map((d) => (
                <StarRow
                  key={d.menuItemId}
                  label={d.name}
                  value={dishRatings[d.menuItemId] ?? 0}
                  onChange={(v) => setDishRatings((prev) => ({ ...prev, [d.menuItemId]: v }))}
                />
              ))}
            </View>
          )}

          <View style={{ marginTop: 16, gap: 12 }}>
            <TextInput
              value={title}
              onChangeText={setTitle}
              maxLength={120}
              placeholder="Title (optional)"
              accessibilityLabel="Review title"
              placeholderTextColor={customerColors.charcoal.soft}
              style={inputStyle}
            />
            <TextInput
              value={comment}
              onChangeText={setComment}
              maxLength={1000}
              placeholder="Tell others about your order (optional)"
              accessibilityLabel="Review details"
              placeholderTextColor={customerColors.charcoal.soft}
              multiline
              style={[inputStyle, { minHeight: 96, textAlignVertical: 'top' }]}
            />
          </View>

          <Pressable
            onPress={handleSubmit}
            disabled={createReview.isPending}
            accessibilityRole="button"
            accessibilityLabel="Submit review"
            style={{ marginTop: 24 }}
            android_ripple={
              overall < 1 || createReview.isPending
                ? undefined
                : { color: `${customerColors.canvas}33`, borderless: false }
            }
          >
            {({ pressed }) => (
              <View
                style={{
                  backgroundColor:
                    overall < 1
                      ? customerColors.surface.soft
                      : pressed && Platform.OS === 'ios'
                        ? customerColors.coral.pressed
                        : customerColors.coral.DEFAULT,
                  borderRadius: 8,
                  minHeight: 52,
                  paddingVertical: 16,
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                {createReview.isPending ? (
                  <ActivityIndicator color={customerColors.canvas} />
                ) : (
                  <Text
                    style={{
                      fontFamily: 'Inter-SemiBold',
                      fontSize: 16,
                      color: overall < 1 ? customerColors.charcoal.soft : customerColors.canvas,
                    }}
                  >
                    Submit review
                  </Text>
                )}
              </View>
            )}
          </Pressable>
        </KeyboardAwareScrollView>
      )}
    </SafeAreaView>
  );
}
