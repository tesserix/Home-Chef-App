import { View, Text } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

export default function ActiveScreen() {
  return (
    <SafeAreaView className="flex-1 bg-white items-center justify-center">
      <Text className="text-gray-500 text-base">Active delivery — coming soon</Text>
    </SafeAreaView>
  );
}
