import React, { useState } from 'react';
import { FlatList, Pressable, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Image } from 'expo-image';
import { router } from 'expo-router';
import { ChevronLeft, Minus, Plus, Trash2 } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useCartStore } from '../store/cart-store';
import type { CartItem } from '../types/customer';

// Full-screen cart. Replaces the old @gorhom/bottom-sheet CartSheet, which
// silently failed to open (gorhom v5 + reanimated v4 incompatibility) — so the
// "View cart" bar appeared to do nothing and the cart was unreachable from the
// chef screen. A plain route is robust and reads as "take me to the cart".
function CartItemRow({ item }: { item: CartItem }) {
  const updateQty = useCartStore((s) => s.updateQty);
  const removeItem = useCartStore((s) => s.removeItem);
  const setInstructions = useCartStore((s) => s.setInstructions);
  // Local state keeps typing smooth; the store trims only for persistence.
  const [note, setNote] = useState(item.instructions ?? '');

  return (
    <View className="py-3 border-b border-hairline gap-2">
      <View className="flex-row items-center gap-3">
        {item.imageUrl ? (
          <Image
            source={{ uri: item.imageUrl }}
            style={{ width: 56, height: 56, borderRadius: 8 }}
            contentFit="cover"
            placeholder={{ blurhash: 'L6PZfSi_.AyE_3t7t7R**0o#DgR4' }}
            transition={150}
            accessible={false}
          />
        ) : (
          <View style={{ width: 56, height: 56, borderRadius: 8 }} className="bg-surface-soft" />
        )}

        <View className="flex-1 gap-0.5">
          <Text className="text-sm font-semibold text-charcoal" numberOfLines={1}>
            {item.name}
          </Text>
          <Text className="text-xs text-charcoal-soft" style={{ fontVariant: ['tabular-nums'] }}>
            ₹{item.price.toFixed(2)} each
          </Text>
          {item.modifiers && item.modifiers.length > 0 ? (
            <Text className="text-xs text-charcoal-soft" numberOfLines={2}>
              {item.modifiers.map((m) => m.optionName).join(', ')}
            </Text>
          ) : null}
        </View>

        {/* Qty stepper — coral accent */}
        <View className="flex-row items-center gap-2">
          <Pressable
            onPress={() => updateQty(item.lineId, item.quantity - 1)}
            accessibilityRole="button"
            accessibilityLabel={`Decrease quantity of ${item.name}`}
            hitSlop={10}
          >
            <View className="w-7 h-7 rounded-full border border-hairline items-center justify-center bg-canvas">
              <Minus size={14} color="#717171" />
            </View>
          </Pressable>

          <View className="w-6 h-6 rounded-full bg-coral-tint items-center justify-center">
            <Text className="text-xs font-semibold text-coral">{item.quantity}</Text>
          </View>

          <Pressable
            onPress={() => updateQty(item.lineId, item.quantity + 1)}
            accessibilityRole="button"
            accessibilityLabel={`Increase quantity of ${item.name}`}
            hitSlop={10}
          >
            <View className="w-7 h-7 rounded-full border border-coral items-center justify-center bg-canvas">
              <Plus size={14} color="#FF385C" />
            </View>
          </Pressable>

          <Pressable
            onPress={() => removeItem(item.lineId)}
            accessibilityRole="button"
            accessibilityLabel={`Remove ${item.name}`}
            hitSlop={10}
          >
            <View className="w-7 h-7 rounded-full bg-surface-soft items-center justify-center ml-1">
              <Trash2 size={14} color="#717171" />
            </View>
          </Pressable>
        </View>
      </View>

      {/* Per-item special instructions — e.g. "no onions" on one dish */}
      <TextInput
        value={note}
        onChangeText={(t) => {
          setNote(t);
          setInstructions(item.lineId, t);
        }}
        placeholder="Add a note (e.g. no onions, less spicy)"
        placeholderTextColor="#717171"
        maxLength={140}
        multiline
        textAlignVertical="top"
        className="bg-surface-soft rounded-lg px-3 py-2.5 text-sm text-charcoal min-h-[60px]"
        accessibilityLabel={`Special instructions for ${item.name}`}
      />
    </View>
  );
}

export default function CartScreen() {
  const items = useCartStore((s) => s.items);
  const total = useCartStore((s) => s.total());
  const chefName = useCartStore((s) => s.chefName);

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'bottom']}>
      {/* Header */}
      <View className="flex-row items-center px-4 py-3 border-b border-hairline">
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={10}
        >
          <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <View className="ml-2">
          <Text className="text-lg font-semibold text-charcoal">Your Cart</Text>
          {chefName ? <Text className="text-sm text-charcoal-soft">{chefName}</Text> : null}
        </View>
      </View>

      {items.length === 0 ? (
        <View className="flex-1 items-center justify-center px-6">
          <Text className="text-charcoal-soft text-base">Your cart is empty</Text>
        </View>
      ) : (
        <>
          <FlatList
            data={items}
            keyExtractor={(item) => item.lineId}
            renderItem={({ item }) => <CartItemRow item={item} />}
            contentContainerStyle={{ paddingHorizontal: 16, paddingBottom: 24 }}
          />

          {/* Sticky checkout footer */}
          <View className="bg-canvas border-t border-hairline px-4 pt-3 pb-4">
            <View className="flex-row justify-between mb-3">
              <Text className="text-base font-semibold text-charcoal-soft">Subtotal</Text>
              <Text
                className="text-base font-medium text-charcoal"
                style={{ fontVariant: ['tabular-nums'] }}
              >
                ₹{total.toFixed(2)}
              </Text>
            </View>
            <Pressable
              onPress={() => router.push('/checkout')}
              accessibilityLabel="Proceed to checkout"
              accessibilityRole="button"
            >
              <View
                className="bg-coral rounded-lg items-center justify-center"
                style={{ minHeight: 52 }}
              >
                <Text className="text-canvas text-base font-semibold">Proceed to Checkout</Text>
              </View>
            </Pressable>
          </View>
        </>
      )}
    </SafeAreaView>
  );
}
