import React, { useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Pressable,
  ScrollView,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Stack, useLocalSearchParams, useRouter } from 'expo-router';
import { Star } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useOrder } from '../../../hooks/useOrderHistory';
import { useCreateReview } from '../../../hooks/useCreateReview';

function StarRow({
  label,
  value,
  onChange,
  required,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
  required?: boolean;
}) {
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
      <View style={{ flexDirection: 'row', gap: 4 }} accessibilityRole="radiogroup" accessibilityLabel={label}>
        {[1, 2, 3, 4, 5].map((n) => (
          <Pressable
            key={n}
            onPress={() => onChange(n)}
            hitSlop={6}
            accessibilityRole="radio"
            accessibilityState={{ selected: value === n }}
            accessibilityLabel={`${n} star${n > 1 ? 's' : ''}`}
          >
            <Star
              size={26}
              color={customerColors.coral.DEFAULT}
              fill={n <= value ? customerColors.coral.DEFAULT : 'transparent'}
            />
          </Pressable>
        ))}
      </View>
    </View>
  );
}

export default function OrderReviewScreen() {
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data, isLoading } = useOrder(id ?? '');
  const order = data?.data;
  const createReview = useCreateReview();

  const [overall, setOverall] = useState(0);
  const [food, setFood] = useState(0);
  const [delivery, setDelivery] = useState(0);
  const [value, setValue] = useState(0);
  const [packaging, setPackaging] = useState(0);
  const [hygiene, setHygiene] = useState(0);
  const [dishRatings, setDishRatings] = useState<Record<string, number>>({});
  const [title, setTitle] = useState('');
  const [comment, setComment] = useState('');

  // One row per distinct dish in the order (#145).
  const dishes = useMemo(() => {
    const seen = new Map<string, string>();
    for (const it of order?.items ?? []) {
      if (!seen.has(it.menuItemId)) seen.set(it.menuItemId, it.name);
    }
    return Array.from(seen, ([menuItemId, name]) => ({ menuItemId, name }));
  }, [order]);

  function handleSubmit() {
    if (overall < 1) {
      Alert.alert('Add a rating', 'Please give an overall rating before submitting.');
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
          Alert.alert('Thanks!', 'Your review has been submitted.');
          router.back();
        },
        onError: (e) => Alert.alert('Could not submit', e.message || 'Please try again.'),
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
      <Stack.Screen options={{ title: 'Leave a review' }} />
      {isLoading ? (
        <ActivityIndicator style={{ marginTop: 32 }} color={customerColors.charcoal.soft} />
      ) : !order ? (
        <Text style={{ textAlign: 'center', marginTop: 32, fontFamily: 'Inter', color: customerColors.charcoal.soft }}>
          We couldn’t find that order.
        </Text>
      ) : order.status !== 'delivered' ? (
        <Text style={{ textAlign: 'center', marginTop: 32, fontFamily: 'Inter', color: customerColors.charcoal.soft }}>
          You can review this order once it’s delivered.
        </Text>
      ) : (
        <ScrollView contentContainerStyle={{ padding: 20, paddingBottom: 40 }} keyboardShouldPersistTaps="handled">
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
              placeholderTextColor={customerColors.charcoal.soft}
              style={inputStyle}
            />
            <TextInput
              value={comment}
              onChangeText={setComment}
              maxLength={1000}
              placeholder="Tell others about your order (optional)"
              placeholderTextColor={customerColors.charcoal.soft}
              multiline
              style={[inputStyle, { minHeight: 96, textAlignVertical: 'top' }]}
            />
          </View>

          <Pressable onPress={handleSubmit} disabled={createReview.isPending} style={{ marginTop: 24 }}>
            <View
              style={{
                backgroundColor: overall < 1 ? customerColors.surface.soft : customerColors.coral.DEFAULT,
                borderRadius: 999,
                paddingVertical: 16,
                alignItems: 'center',
              }}
            >
              {createReview.isPending ? (
                <ActivityIndicator color={customerColors.canvas} />
              ) : (
                <Text
                  style={{
                    fontFamily: 'Inter',
                    fontSize: 16,
                    fontWeight: '600',
                    color: overall < 1 ? customerColors.charcoal.soft : customerColors.canvas,
                  }}
                >
                  Submit review
                </Text>
              )}
            </View>
          </Pressable>
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
