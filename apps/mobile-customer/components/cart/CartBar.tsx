import { Pressable, Text, View } from 'react-native';
import { useCartStore } from '../../store/cart-store';

interface CartBarProps {
  onPress: () => void;
}

export function CartBar({ onPress }: CartBarProps) {
  const items = useCartStore((s) => s.items);
  const total = useCartStore((s) => s.total());

  if (items.length === 0) {
    return null;
  }

  const itemCount = items.reduce((sum, i) => sum + i.quantity, 0);

  return (
    // Outer View carries the shadow; inner View clips to rounded corners (iOS shadow + overflow bug).
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
      <Pressable
        onPress={onPress}
        accessibilityLabel={`View cart — ${itemCount} items, ₹${total.toFixed(2)}`}
        accessibilityRole="button"
      >
        <View className="bg-canvas flex-row items-center px-5 gap-3 border-t border-hairline" style={{ height: 72 }}>
          {/* Item count + total — left side */}
          <View className="flex-1 flex-row items-center gap-3">
            {/* Count badge — coral-tint bg, coral text */}
            <View className="w-8 h-8 bg-coral-tint rounded-full items-center justify-center">
              <Text className="text-coral text-sm font-semibold">{itemCount}</Text>
            </View>
            <View>
              <Text className="text-charcoal text-sm font-semibold">
                {itemCount === 1 ? '1 item' : `${itemCount} items`}
              </Text>
              {/* Tabular figures for price */}
              <Text className="text-charcoal-soft text-xs" style={{ fontVariant: ['tabular-nums'] }}>
                ₹{total.toFixed(2)}
              </Text>
            </View>
          </View>

          {/* Coral CTA — right side */}
          <View className="bg-coral rounded-lg px-5 items-center justify-center" style={{ height: 44 }}>
            <Text className="text-canvas text-sm font-semibold">View cart</Text>
          </View>
        </View>
      </Pressable>
    </View>
  );
}
