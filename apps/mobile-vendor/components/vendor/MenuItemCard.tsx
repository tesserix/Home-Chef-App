import { Alert, Switch, Text, TouchableOpacity, View } from 'react-native';
import { Image } from 'expo-image';
import { Edit2, Trash2 } from 'lucide-react-native';
import type { MenuItem } from '../../hooks/useVendorMenu';
import { useToggleAvailability } from '../../hooks/useVendorMenu';

interface MenuItemCardProps {
  item: MenuItem;
  onEdit: () => void;
  onDelete: () => void;
}

export function MenuItemCard({ item, onEdit, onDelete }: MenuItemCardProps) {
  const toggleMutation = useToggleAvailability();

  function handleDelete() {
    Alert.alert('Delete Item', 'Are you sure you want to delete this menu item?', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Delete', onPress: onDelete, style: 'destructive' },
    ]);
  }

  return (
    <View className="bg-white rounded-2xl shadow-sm p-3 mb-2 flex-row">
      {/* Item image */}
      <View className="w-20 h-20 rounded-xl overflow-hidden bg-gray-100 mr-3 flex-shrink-0">
        {item.images[0]?.url ? (
          <Image
            source={{ uri: item.images[0].url }}
            style={{ width: 80, height: 80 }}
            contentFit="cover"
          />
        ) : (
          <View className="w-20 h-20 bg-gray-200 rounded-xl" />
        )}
      </View>

      {/* Item details */}
      <View className="flex-1">
        <View className="flex-row items-start justify-between mb-1">
          <Text className="text-base font-semibold text-gray-900 flex-1 mr-2" numberOfLines={1}>
            {item.name}
          </Text>
          {/* Veg / non-veg indicator */}
          <View
            className={`w-4 h-4 rounded-full border-2 ${item.isVeg ? 'bg-green-500 border-green-600' : 'bg-red-500 border-red-600'}`}
          />
        </View>

        <Text className="text-orange-600 font-semibold text-sm mb-1">
          ₹{item.price.toFixed(2)}
        </Text>

        {/* Category badge */}
        <View className="self-start bg-orange-50 px-2 py-0.5 rounded-full mb-2">
          <Text className="text-xs text-orange-600">{item.category}</Text>
        </View>

        {/* Bottom row: availability toggle + action buttons */}
        <View className="flex-row items-center justify-between">
          <View className="flex-row items-center gap-1.5">
            <Switch
              value={item.isAvailable}
              onValueChange={(value) =>
                toggleMutation.mutate({ itemId: item.id, isAvailable: value })
              }
              trackColor={{ false: '#D1D5DB', true: '#FB923C' }}
              thumbColor={item.isAvailable ? '#EA580C' : '#9CA3AF'}
              disabled={toggleMutation.isPending}
            />
            <Text className="text-xs text-gray-500">
              {item.isAvailable ? 'Available' : 'Unavailable'}
            </Text>
          </View>

          <View className="flex-row gap-2">
            <TouchableOpacity
              onPress={onEdit}
              className="p-2 rounded-lg bg-blue-50"
              activeOpacity={0.7}
            >
              <Edit2 size={16} color="#3B82F6" />
            </TouchableOpacity>
            <TouchableOpacity
              onPress={handleDelete}
              className="p-2 rounded-lg bg-red-50"
              activeOpacity={0.7}
            >
              <Trash2 size={16} color="#EF4444" />
            </TouchableOpacity>
          </View>
        </View>
      </View>
    </View>
  );
}
