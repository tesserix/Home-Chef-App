import React, { useState } from 'react';
import {
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Image } from 'expo-image';
import { router } from 'expo-router';
import { ChevronLeft, Minus, Plus, Trash2, UtensilsCrossed } from 'lucide-react-native';
import { customerColors, customerTheme } from '@homechef/mobile-shared/theme';
import { useCartStore } from '../store/cart-store';
import type { CartItem } from '../types/customer';

// Android ripple tints — translucent colours derived from existing tokens
// (never a new literal colour), matching the ChefCard/MenuItemCard convention.
const BACK_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const STEPPER_RIPPLE = `${customerColors.coral.DEFAULT}22`;
const REMOVE_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const CTA_RIPPLE = `${customerColors.canvas}33`;

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
            style={{ width: 60, height: 60, borderRadius: 12 }}
            contentFit="cover"
            placeholder={{ blurhash: 'L6PZfSi_.AyE_3t7t7R**0o#DgR4' }}
            transition={150}
            accessible={false}
          />
        ) : (
          // Missing image — surface-soft placeholder + utensil glyph (R2):
          // required, not optional. No grey void.
          <View
            style={{ width: 60, height: 60, borderRadius: 12 }}
            className="bg-surface-soft items-center justify-center"
          >
            <UtensilsCrossed size={22} color={customerColors.charcoal.soft} strokeWidth={1.5} />
          </View>
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

        {/* Qty stepper — coral accent. 44pt touch target via hitSlop, with a
            visible pressed state on both platforms (spec R5 + §3.5). */}
        <View className="flex-row items-center gap-1.5">
          <Pressable
            onPress={() => updateQty(item.lineId, item.quantity - 1)}
            accessibilityRole="button"
            accessibilityLabel={`Decrease quantity of ${item.name}`}
            hitSlop={8}
            android_ripple={{ color: STEPPER_RIPPLE, borderless: true, radius: 18 }}
          >
            {({ pressed }) => (
              <View
                className={`w-8 h-8 rounded-full border border-hairline items-center justify-center bg-canvas ${
                  pressed && Platform.OS === 'ios' ? 'bg-surface-soft' : ''
                }`}
              >
                <Minus size={14} color={customerColors.charcoal.soft} />
              </View>
            )}
          </Pressable>

          <View className="w-7 h-7 rounded-full bg-coral-tint items-center justify-center">
            <Text
              className="text-xs font-semibold text-coral"
              style={{ fontVariant: ['tabular-nums'] }}
            >
              {item.quantity}
            </Text>
          </View>

          <Pressable
            onPress={() => updateQty(item.lineId, item.quantity + 1)}
            accessibilityRole="button"
            accessibilityLabel={`Increase quantity of ${item.name}`}
            hitSlop={8}
            android_ripple={{ color: STEPPER_RIPPLE, borderless: true, radius: 18 }}
          >
            {({ pressed }) => (
              <View
                className={`w-8 h-8 rounded-full border border-coral items-center justify-center bg-canvas ${
                  pressed && Platform.OS === 'ios' ? 'bg-coral-tint' : ''
                }`}
              >
                <Plus size={14} color={customerColors.coral.DEFAULT} />
              </View>
            )}
          </Pressable>

          <Pressable
            onPress={() => removeItem(item.lineId)}
            accessibilityRole="button"
            accessibilityLabel={`Remove ${item.name}`}
            hitSlop={8}
            android_ripple={{ color: REMOVE_RIPPLE, borderless: true, radius: 18 }}
          >
            {({ pressed }) => (
              <View
                className={`w-8 h-8 rounded-full bg-surface-soft items-center justify-center ml-1 ${
                  pressed && Platform.OS === 'ios' ? 'opacity-60' : ''
                }`}
              >
                <Trash2 size={14} color={customerColors.charcoal.soft} />
              </View>
            )}
          </Pressable>
        </View>
      </View>

      {/* Per-item special instructions — e.g. "no onions" on one dish. Plain
          "done" return key + blurOnSubmit so the keyboard dismisses cleanly
          (R9) instead of hunting for a dismiss affordance. */}
      <TextInput
        value={note}
        onChangeText={(t) => {
          setNote(t);
          setInstructions(item.lineId, t);
        }}
        placeholder="Add a note (e.g. no onions, less spicy)"
        placeholderTextColor={customerColors.charcoal.soft}
        maxLength={140}
        multiline
        textAlignVertical="top"
        returnKeyType="done"
        blurOnSubmit
        className="bg-surface-soft rounded-xl px-3 py-2.5 text-sm text-charcoal min-h-[60px]"
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
      {/* Header — back chevron + title hierarchy (canonical spec §4) */}
      <View className="flex-row items-center px-4 py-3 border-b border-hairline">
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={10}
          android_ripple={{ color: BACK_RIPPLE, borderless: true, radius: 20 }}
        >
          {({ pressed }) => (
            <View style={pressed && Platform.OS === 'ios' ? { opacity: 0.6 } : undefined}>
              <ChevronLeft size={26} color={customerColors.charcoal.DEFAULT} />
            </View>
          )}
        </Pressable>
        <View className="ml-2">
          <Text className="text-xl font-bold text-charcoal font-display">Your Cart</Text>
          {chefName ? <Text className="text-sm text-charcoal-soft">{chefName}</Text> : null}
        </View>
      </View>

      {items.length === 0 ? (
        <View className="flex-1 items-center justify-center px-6">
          <Text className="text-charcoal-soft text-base">Your cart is empty</Text>
        </View>
      ) : (
        <KeyboardAvoidingView
          className="flex-1"
          behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
          keyboardVerticalOffset={Platform.OS === 'ios' ? 8 : 0}
        >
          <FlatList
            data={items}
            keyExtractor={(item) => item.lineId}
            renderItem={({ item }) => <CartItemRow item={item} />}
            contentContainerStyle={{ paddingHorizontal: 16, paddingBottom: 24 }}
            keyboardShouldPersistTaps="handled"
          />

          {/* Sticky checkout footer — top hairline + shadow[2] (spec item 5) */}
          <View
            className="bg-canvas border-t border-hairline px-4 pt-3 pb-4"
            style={{
              shadowColor: customerTheme.shadow[2].shadowColor,
              shadowOffset: { width: 0, height: -2 },
              shadowOpacity: customerTheme.shadow[2].shadowOpacity,
              shadowRadius: customerTheme.shadow[2].shadowRadius,
              elevation: customerTheme.shadow[2].elevation,
            }}
          >
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
              android_ripple={{ color: CTA_RIPPLE, borderless: false }}
            >
              {({ pressed }) => (
                <View
                  className={`rounded-lg items-center justify-center ${
                    pressed && Platform.OS === 'ios' ? 'bg-coral-pressed' : 'bg-coral'
                  }`}
                  style={{ minHeight: 52 }}
                >
                  <Text className="text-canvas text-base font-semibold">Proceed to Checkout</Text>
                </View>
              )}
            </Pressable>
          </View>
        </KeyboardAvoidingView>
      )}
    </SafeAreaView>
  );
}
