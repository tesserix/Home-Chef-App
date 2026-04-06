import { View, Text } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

export default function MenuScreen() {
  return (
    <SafeAreaView className="flex-1 bg-white">
      <View className="flex-1 items-center justify-center px-6">
        <Text className="text-xl font-semibold text-gray-700">Menu — coming in next plan</Text>
      </View>
    </SafeAreaView>
  );
}
