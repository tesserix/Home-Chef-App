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
      <View className="flex-1 justify-center items-center bg-ink/60 px-6">
        <View className="bg-bone rounded-2xl p-6 w-full max-w-sm">
          <Text className="text-xl font-semibold text-ink mb-3">
            Background Location Needed
          </Text>
          <Text className="text-ink-soft mb-2">
            To keep customers updated on their delivery, HomeChef Delivery
            needs to track your location while you are on an active delivery —
            even when the app is in the background.
          </Text>
          <Text className="text-ink-muted text-sm mb-6">
            Location tracking stops automatically when the delivery is
            completed. Battery usage is minimised by only tracking every 15
            seconds.
          </Text>
          <TouchableOpacity
            className="bg-herb rounded-xl py-3 items-center mb-3"
            onPress={onAllow}
            activeOpacity={0.8}
          >
            <Text className="text-paper font-semibold text-base">
              Allow Background Location
            </Text>
          </TouchableOpacity>
          <TouchableOpacity className="py-3 items-center" onPress={onDeny}>
            <Text className="text-ink-muted text-base">Not Now</Text>
          </TouchableOpacity>
        </View>
      </View>
    </Modal>
  );
}
