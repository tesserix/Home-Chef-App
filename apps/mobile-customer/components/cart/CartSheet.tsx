import React, { forwardRef } from 'react';
import { FlatList, Pressable, Text, View } from 'react-native';
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

  return (
    <View className="flex-row items-center py-3 border-b border-mist gap-3">
      {item.imageUrl ? (
        <Image
          source={{ uri: item.imageUrl }}
          style={{ width: 56, height: 56, borderRadius: 8 }}
          contentFit="cover"
          placeholder={{ blurhash: 'L6PZfSi_.AyE_3t7t7R**0o#DgR4' }}
          transition={150}
        />
      ) : (
        <View style={{ width: 56, height: 56, borderRadius: 8 }} className="bg-mist" />
      )}

      <View className="flex-1 gap-0.5">
        <Text className="text-sm font-semibold text-ink" numberOfLines={1}>
          {item.name}
        </Text>
        <Text className="text-xs text-ink-muted">₹{item.price.toFixed(2)} each</Text>
      </View>

      <View className="flex-row items-center gap-2">
        <Pressable
          onPress={() => updateQty(item.menuItemId, item.quantity - 1)}
          className="w-7 h-7 rounded-full border border-mist-strong items-center justify-center active:bg-mist"
          accessibilityLabel="Decrease quantity"
        >
          <Minus size={14} color="#4a4a47" />
        </Pressable>
        <Text className="text-sm font-medium text-ink w-5 text-center">{item.quantity}</Text>
        <Pressable
          onPress={() => updateQty(item.menuItemId, item.quantity + 1)}
          className="w-7 h-7 rounded-full border border-mist-strong items-center justify-center active:bg-mist"
          accessibilityLabel="Increase quantity"
        >
          <Plus size={14} color="#4a4a47" />
        </Pressable>
        <Pressable
          onPress={() => removeItem(item.menuItemId)}
          className="w-7 h-7 rounded-full bg-paprika-tint items-center justify-center ml-1 active:bg-paprika-tint"
          accessibilityLabel={`Remove ${item.name}`}
        >
          <Trash2 size={14} color="#c95b3e" />
        </Pressable>
      </View>
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
          <Text className="text-lg font-medium text-ink">Your Cart</Text>
          {chefName ? (
            <Text className="text-sm text-ink-muted">{chefName}</Text>
          ) : null}
        </View>

        <FlatList
          data={items}
          keyExtractor={(item) => item.menuItemId}
          renderItem={({ item }) => <CartItemRow item={item} />}
          contentContainerStyle={{ paddingBottom: 120 }}
        />

        {items.length > 0 && (
          <View className="absolute bottom-0 left-0 right-0 px-4 pb-8 pt-3 bg-bone border-t border-mist">
            <View className="flex-row justify-between mb-3">
              <Text className="text-base font-semibold text-ink-soft">Subtotal</Text>
              <Text className="text-base font-medium text-ink">₹{total.toFixed(2)}</Text>
            </View>
            <Pressable
              onPress={handleCheckout}
              className="bg-herb rounded-xl py-4 items-center active:bg-herb"
              accessibilityLabel="Proceed to checkout"
              accessibilityRole="button"
            >
              <Text className="text-paper text-base font-medium">Proceed to Checkout</Text>
            </Pressable>
          </View>
        )}

        {items.length === 0 && (
          <View className="flex-1 items-center justify-center pb-20">
            <Text className="text-ink-muted text-base">Your cart is empty</Text>
          </View>
        )}
      </BottomSheetView>
    </BottomSheet>
  );
});

CartSheet.displayName = 'CartSheet';
