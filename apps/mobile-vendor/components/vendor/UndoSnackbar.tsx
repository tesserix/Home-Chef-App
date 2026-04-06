import { useEffect, useRef } from 'react';
import { View, Text, Pressable, Animated } from 'react-native';
import type { PendingUndo } from '../../hooks/useVendorOrders';

interface UndoSnackbarProps {
  pendingUndo: PendingUndo | null;
  onUndo: () => void;
}

export function UndoSnackbar({ pendingUndo, onUndo }: UndoSnackbarProps) {
  const translateY = useRef(new Animated.Value(100)).current;

  useEffect(() => {
    if (pendingUndo) {
      Animated.spring(translateY, {
        toValue: 0,
        useNativeDriver: true,
        tension: 100,
        friction: 10,
      }).start();
    } else {
      Animated.timing(translateY, {
        toValue: 100,
        duration: 200,
        useNativeDriver: true,
      }).start();
    }
  }, [pendingUndo, translateY]);

  if (!pendingUndo) return null;

  const isAccepted = pendingUndo.action === 'accepted';
  const bgClass = isAccepted ? 'bg-green-600' : 'bg-red-600';
  const label = isAccepted ? 'Order accepted' : 'Order rejected';

  return (
    <Animated.View
      style={{ transform: [{ translateY }] }}
      className={`absolute bottom-4 left-4 right-4 z-50 flex-row items-center justify-between rounded-xl px-4 py-3 shadow-lg ${bgClass}`}
    >
      <Text className="flex-1 text-sm font-medium text-white">{label}</Text>
      <Pressable
        onPress={onUndo}
        className="ml-4 rounded-md bg-white/20 px-3 py-1.5 active:bg-white/30"
      >
        <Text className="text-sm font-bold text-white">UNDO</Text>
      </Pressable>
    </Animated.View>
  );
}
