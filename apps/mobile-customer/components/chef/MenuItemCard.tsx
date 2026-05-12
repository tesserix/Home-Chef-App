import { Alert, Pressable, Text, View } from 'react-native';
import { Image } from 'expo-image';
import * as Haptics from 'expo-haptics';
import { Plus } from 'lucide-react-native';
import { useCartStore } from '../../store/cart-store';
import type { MenuItem } from '../../types/customer';

interface MenuItemCardProps {
  item: MenuItem;
  chefId: string;
  chefName: string;
}

export function MenuItemCard({ item, chefId, chefName }: MenuItemCardProps) {
  const handleAdd = () => {
    const cartItem = {
      menuItemId: item.id,
      name: item.name,
      price: item.price,
      quantity: 1,
      imageUrl: item.imageUrl,
    };

    const result = useCartStore.getState().addItem(cartItem, { id: chefId, name: chefName });

    if (result === 'cross_chef_conflict') {
      Alert.alert(
        'Replace Cart?',
        'You have items from another chef. Replace cart?',
        [
          { text: 'Cancel', style: 'cancel' },
          {
            text: 'Replace',
            style: 'destructive',
            onPress: () => {
              useCartStore.getState().clearCart();
              useCartStore.getState().addItem(cartItem, { id: chefId, name: chefName });
              void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
            },
          },
        ]
      );
    } else {
      void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    }
  };

  return (
    <View className="flex-row items-center bg-bone rounded-xl p-3 mb-2 border border-mist">
      {item.imageUrl ? (
        <Image
          source={{ uri: item.imageUrl }}
          style={{ width: 80, height: 80, borderRadius: 10 }}
          contentFit="cover"
          placeholder={{ blurhash: 'L6PZfSi_.AyE_3t7t7R**0o#DgR4' }}
          transition={200}
        />
      ) : (
        <View style={{ width: 80, height: 80, borderRadius: 10 }} className="bg-mist" />
      )}

      <View className="flex-1 ml-3 gap-1">
        <Text className="text-sm font-semibold text-ink" numberOfLines={1}>
          {item.name}
        </Text>
        {item.description ? (
          <Text className="text-xs text-ink-muted" numberOfLines={2}>
            {item.description}
          </Text>
        ) : null}
        {item.dietaryTags && item.dietaryTags.length > 0 ? (
          <View className="flex-row flex-wrap gap-1">
            {item.dietaryTags.map((tag) => (
              <View key={tag} className="bg-herb-tint border border-herb/30 rounded-full px-2 py-0.5">
                <Text className="text-xs text-herb">{tag}</Text>
              </View>
            ))}
          </View>
        ) : null}
        <Text className="text-sm font-medium text-ink">₹{item.price.toFixed(2)}</Text>
      </View>

      <Pressable
        onPress={handleAdd}
        disabled={!item.isAvailable}
        className={`ml-3 w-9 h-9 rounded-full items-center justify-center ${item.isAvailable ? 'bg-herb active:bg-herb' : 'bg-mist'}`}
        accessibilityLabel={`Add ${item.name} to cart`}
        accessibilityRole="button"
      >
        <Plus size={18} color={item.isAvailable ? '#fafaf7' : '#7a7a76'} />
      </Pressable>
    </View>
  );
}
