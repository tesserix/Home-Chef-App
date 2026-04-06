import React from 'react';
import { Modal, View, Text, TouchableOpacity } from 'react-native';

interface Props {
  visible: boolean;
  onAllow: () => void;
  onDeny: () => void;
}

export function LocationRationaleModal({ visible, onAllow, onDeny }: Props) {
  return (
    <Modal visible={visible} transparent animationType="fade">
      <View className="flex-1 justify-center items-center bg-black/60 px-6">
        <View className="bg-white rounded-2xl p-6 w-full max-w-sm">
          <Text className="text-xl font-bold text-gray-900 mb-3">
            Background Location Needed
          </Text>
          <Text className="text-gray-600 mb-2">
            To keep customers updated on their delivery, HomeChef Delivery
            needs to track your location while you are on an active delivery —
            even when the app is in the background.
          </Text>
          <Text className="text-gray-500 text-sm mb-6">
            Location tracking stops automatically when the delivery is
            completed. Battery usage is minimised by only tracking every 15
            seconds.
          </Text>
          <TouchableOpacity
            className="bg-orange-500 rounded-xl py-3 items-center mb-3"
            onPress={onAllow}
            activeOpacity={0.8}
          >
            <Text className="text-white font-semibold text-base">
              Allow Background Location
            </Text>
          </TouchableOpacity>
          <TouchableOpacity className="py-3 items-center" onPress={onDeny}>
            <Text className="text-gray-500 text-base">Not Now</Text>
          </TouchableOpacity>
        </View>
      </View>
    </Modal>
  );
}
