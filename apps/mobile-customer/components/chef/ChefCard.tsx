import { Pressable, Text, View } from 'react-native';
import { Image } from 'expo-image';
import { router } from 'expo-router';
import { Star } from 'lucide-react-native';
import type { Chef } from '../../types/customer';

interface ChefCardProps {
  chef: Chef;
}

export function ChefCard({ chef }: ChefCardProps) {
  const handlePress = () => {
    router.push(`/chef/${chef.id}`);
  };

  return (
    <Pressable
      className="flex-1 bg-bone rounded-2xl overflow-hidden border border-mist shadow-sm"
      onPress={handlePress}
      accessibilityLabel={`View ${chef.name}`}
      accessibilityRole="button"
    >
      <View className="relative">
        <Image
          source={chef.imageUrl ? { uri: chef.imageUrl } : null}
          style={{ width: '100%', height: 160 }}
          contentFit="cover"
          placeholder={{ blurhash: 'L6PZfSi_.AyE_3t7t7R**0o#DgR4' }}
          transition={200}
        />
        <View
          className={`absolute top-2 right-2 px-2 py-0.5 rounded-full ${chef.isOpen ? 'bg-herb' : 'bg-ink-muted'}`}
        >
          <Text className="text-paper text-xs font-semibold">
            {chef.isOpen ? 'Open' : 'Closed'}
          </Text>
        </View>
      </View>

      <View className="p-3 gap-1">
        <Text className="text-base font-medium text-ink" numberOfLines={1}>
          {chef.name}
        </Text>
        <Text className="text-xs text-ink-muted" numberOfLines={1}>
          {chef.cuisine}
        </Text>

        <View className="flex-row items-center gap-1 mt-1">
          <Star size={12} color="#d1a64a" fill="#d1a64a" />
          <Text className="text-xs font-semibold text-ink-soft">
            {chef.rating.toFixed(1)}
          </Text>
          <Text className="text-xs text-ink-muted">
            ({chef.reviewCount})
          </Text>
        </View>

        <View className="flex-row items-center justify-between mt-1">
          {chef.deliveryTime != null && (
            <Text className="text-xs text-ink-muted">{chef.deliveryTime}</Text>
          )}
          {chef.minimumOrder != null && (
            <Text className="text-xs text-ink-muted">Min ₹{chef.minimumOrder}</Text>
          )}
        </View>
      </View>
    </Pressable>
  );
}
