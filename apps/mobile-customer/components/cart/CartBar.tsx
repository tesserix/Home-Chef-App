import { Pressable, Text, View } from 'react-native';
import { ShoppingCart } from 'lucide-react-native';
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
    <Pressable
      onPress={onPress}
      className="absolute bottom-0 left-0 right-0 h-[72px] bg-herb flex-row items-center px-5 gap-3 active:bg-herb"
      accessibilityLabel={`View cart — ${itemCount} items, ₹${total.toFixed(2)}`}
      accessibilityRole="button"
    >
      <View className="w-8 h-8 bg-herb-soft rounded-full items-center justify-center">
        <Text className="text-paper text-sm font-medium">{itemCount}</Text>
      </View>
      <ShoppingCart size={20} color="#fafaf7" />
      <Text className="flex-1 text-paper text-base font-semibold">View Cart</Text>
      <Text className="text-paper text-base font-medium">₹{total.toFixed(2)}</Text>
    </Pressable>
  );
}
