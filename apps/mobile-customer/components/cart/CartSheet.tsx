import React, { forwardRef, useState } from 'react';
import { FlatList, Pressable, Text, TextInput, View } from 'react-native';
import BottomSheet, { BottomSheetView } from '@gorhom/bottom-sheet';
import type { BottomSheetMethods } from '@gorhom/bottom-sheet/lib/typescript/types';
import { Image } from 'expo-image';
import { Minus, Plus, Trash2 } from 'lucide-react-native';
import { router } from 'expo-router';
import { useCartStore } from '../../store/cart-store';
import type { CartItem } from '../../types/customer';

interface CartItemRowProps {
  item: CartItem;
}

function CartItemRow({ item }: CartItemRowProps) {
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
        />
      ) : (
        // Missing image → surface-soft placeholder per spec
        <View style={{ width: 56, height: 56, borderRadius: 8 }} className="bg-surface-soft" />
      )}

      <View className="flex-1 gap-0.5">
        <Text className="text-sm font-semibold text-charcoal" numberOfLines={1}>
          {item.name}
        </Text>
        <Text className="text-xs text-charcoal-soft" style={{ fontVariant: ['tabular-nums'] }}>
          ₹{item.price.toFixed(2)} each
        </Text>
      </View>

      {/* Qty stepper — coral accent, iOS Pressable inner-View pattern */}
      <View className="flex-row items-center gap-2">
        <Pressable
          onPress={() => updateQty(item.menuItemId, item.quantity - 1)}
          accessibilityLabel="Decrease quantity"
        >
          <View className="w-7 h-7 rounded-full border border-hairline items-center justify-center bg-canvas">
            <Minus size={14} color="#717171" />
          </View>
        </Pressable>

        {/* Quantity display — coral-tint chip */}
        <View className="w-6 h-6 rounded-full bg-coral-tint items-center justify-center">
          <Text className="text-xs font-semibold text-coral">{item.quantity}</Text>
        </View>

        <Pressable
          onPress={() => updateQty(item.menuItemId, item.quantity + 1)}
          accessibilityLabel="Increase quantity"
        >
          <View className="w-7 h-7 rounded-full border border-coral items-center justify-center bg-canvas">
            <Plus size={14} color="#FF385C" />
          </View>
        </Pressable>

        <Pressable
          onPress={() => removeItem(item.menuItemId)}
          accessibilityLabel={`Remove ${item.name}`}
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
          setInstructions(item.menuItemId, t);
        }}
        placeholder="Add a note (e.g. no onions)"
        placeholderTextColor="#717171"
        maxLength={140}
        className="bg-surface-soft rounded-lg px-3 py-2 text-xs text-charcoal"
        accessibilityLabel={`Special instructions for ${item.name}`}
      />
    </View>
  );
}

export const CartSheet = forwardRef<BottomSheetMethods>((_: unknown, ref: React.ForwardedRef<BottomSheetMethods>) => {
  const items = useCartStore((s) => s.items);
  const total = useCartStore((s) => s.total());
  const chefName = useCartStore((s) => s.chefName);

  const snapPoints = ['60%', '90%'];

  const handleCheckout = () => {
    if (ref && 'current' in ref && ref.current) {
      ref.current.close();
    }
    router.push('/checkout');
  };

  return (
    // @ts-expect-error — @gorhom/bottom-sheet ref types not yet updated for React 19
    <BottomSheet
      ref={ref}
      index={-1}
      snapPoints={snapPoints}
      enablePanDownToClose
      backgroundStyle={{ borderRadius: 24 }}
    >
      {/* @ts-expect-error — BottomSheetView children prop not compatible with React 19 JSX types */}
      <BottomSheetView style={{ flex: 1, paddingHorizontal: 16 }}>
        <View className="pb-2">
          <Text className="text-lg font-semibold text-charcoal">Your Cart</Text>
          {chefName ? (
            <Text className="text-sm text-charcoal-soft">{chefName}</Text>
          ) : null}
        </View>

        <FlatList
          data={items}
          keyExtractor={(item) => item.menuItemId}
          renderItem={({ item }) => <CartItemRow item={item} />}
          contentContainerStyle={{ paddingBottom: 120 }}
        />

        {items.length > 0 && (
          // Sticky bottom bar — white bg, top hairline, shadow via outer View
          <View
            style={{
              position: 'absolute',
              bottom: 0,
              left: 0,
              right: 0,
              shadowColor: '#000',
              shadowOffset: { width: 0, height: -2 },
              shadowOpacity: 0.06,
              shadowRadius: 6,
              elevation: 4,
            }}
          >
            <View className="bg-canvas border-t border-hairline px-4 pb-8 pt-3">
              <View className="flex-row justify-between mb-3">
                <Text className="text-base font-semibold text-charcoal-soft">Subtotal</Text>
                <Text
                  className="text-base font-medium text-charcoal"
                  style={{ fontVariant: ['tabular-nums'] }}
                >
                  ₹{total.toFixed(2)}
                </Text>
              </View>
              {/* Coral primary CTA — iOS Pressable inner-View pattern */}
              <Pressable
                onPress={handleCheckout}
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
          </View>
        )}

        {items.length === 0 && (
          <View className="flex-1 items-center justify-center pb-20">
            <Text className="text-charcoal-soft text-base">Your cart is empty</Text>
          </View>
        )}
      </BottomSheetView>
    </BottomSheet>
  );
});

CartSheet.displayName = 'CartSheet';
