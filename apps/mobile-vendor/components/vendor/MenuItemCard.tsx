import { Alert, Switch, Text, TouchableOpacity, View } from 'react-native';
import { Image } from 'expo-image';
import { Edit2, Trash2 } from 'lucide-react-native';
import type { MenuItem } from '../../hooks/useVendorMenu';
import { useToggleAvailability } from '../../hooks/useVendorMenu';

interface MenuItemCardProps {
  item: MenuItem;
  categoryName?: string;
  onEdit: () => void;
  onDelete: () => void;
}

export function MenuItemCard({ item, categoryName, onEdit, onDelete }: MenuItemCardProps) {
  const toggleMutation = useToggleAvailability();
  const images = item.images ?? [];
  const priceLabel = Number.isFinite(item.price) ? Number(item.price).toFixed(2) : '0.00';

  function handleDelete() {
    Alert.alert('Delete Item', 'Are you sure you want to delete this menu item?', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Delete', onPress: onDelete, style: 'destructive' },
    ]);
  }

  return (
    <View className="bg-bone rounded-2xl shadow-sm p-3 mb-2 flex-row">
      {/* Item image */}
      <View className="w-20 h-20 rounded-xl overflow-hidden bg-mist mr-3 flex-shrink-0">
        {images[0]?.url ? (
          <Image
            source={{ uri: images[0].url }}
            style={{ width: 80, height: 80 }}
            contentFit="cover"
          />
        ) : (
          <View className="w-20 h-20 bg-mist rounded-xl" />
        )}
      </View>

      {/* Item details */}
      <View className="flex-1">
        <View className="flex-row items-start justify-between mb-1">
          <Text className="text-base font-semibold text-ink flex-1 mr-2" numberOfLines={1}>
            {item.name}
          </Text>
          {/* Veg / non-veg indicator */}
          <View
            className={`w-4 h-4 rounded-full border-2 ${item.isVeg ? 'bg-herb border-herb' : 'bg-paprika border-paprika'}`}
          />
        </View>

        <Text className="text-herb font-semibold text-sm mb-1">
          ₹{priceLabel}
        </Text>

        {categoryName ? (
          <View className="self-start bg-herb-tint px-2 py-0.5 rounded-full mb-2">
            <Text className="text-xs text-herb">{categoryName}</Text>
          </View>
        ) : (
          <View className="mb-2" />
        )}

        {/* Bottom row: availability toggle + action buttons */}
        <View className="flex-row items-center justify-between">
          <View className="flex-row items-center gap-1.5">
            <Switch
              value={item.isAvailable}
              onValueChange={(value) =>
                toggleMutation.mutate({ itemId: item.id, isAvailable: value })
              }
              trackColor={{ false: '#d4d3ce', true: '#9A3412' }}
              thumbColor={item.isAvailable ? '#C2410C' : '#7a7a76'}
              disabled={toggleMutation.isPending}
            />
            <Text className="text-xs text-ink-muted">
              {item.isAvailable ? 'Available' : 'Unavailable'}
            </Text>
          </View>

          <View className="flex-row gap-2">
            <TouchableOpacity
              onPress={onEdit}
              className="p-2 rounded-lg bg-info/10"
              activeOpacity={0.7}
            >
              <Edit2 size={16} color="#4a73a3" />
            </TouchableOpacity>
            <TouchableOpacity
              onPress={handleDelete}
              className="p-2 rounded-lg bg-paprika-tint"
              activeOpacity={0.7}
            >
              <Trash2 size={16} color="#c95b3e" />
            </TouchableOpacity>
          </View>
        </View>
      </View>
    </View>
  );
}
