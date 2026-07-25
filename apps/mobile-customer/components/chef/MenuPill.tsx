import { Platform, Pressable, Text, View } from 'react-native';
import { Menu as MenuIcon } from 'lucide-react-native';

import { customerColors } from '@homechef/mobile-shared/theme';

// MenuPill — the always-reachable way into the category sheet.
//
// It floats rather than sitting in the scroll body precisely because the problem
// it solves is "I am deep in a long menu and want to get somewhere else": a
// control that scrolls away is useless at the moment it is needed.

export interface MenuPillProps {
  onPress: () => void;
  /** Lifted above the cart bar when one is showing, so neither covers the other. */
  bottomOffset?: number;
}

export function MenuPill({ onPress, bottomOffset = 24 }: MenuPillProps) {
  return (
    <View
      className="absolute left-0 right-0 items-center"
      style={{ bottom: bottomOffset }}
      pointerEvents="box-none"
    >
      <Pressable
        onPress={onPress}
        accessibilityRole="button"
        accessibilityLabel="Open menu categories"
        android_ripple={{ color: `${customerColors.canvas}33`, borderless: false }}
      >
        {({ pressed }) => (
          <View
            className="min-h-[48px] flex-row items-center gap-2 rounded-full px-6"
            style={{
              backgroundColor: customerColors.charcoal.DEFAULT,
              opacity: pressed && Platform.OS === 'ios' ? 0.85 : 1,
              shadowColor: customerColors.charcoal.DEFAULT,
              shadowOffset: { width: 0, height: 3 },
              shadowOpacity: 0.22,
              shadowRadius: 10,
              elevation: 6,
              justifyContent: 'center',
            }}
          >
            <MenuIcon size={20} color={customerColors.canvas} />
            <Text
              className="text-[16px] font-semibold"
              style={{ color: customerColors.canvas }}
            >
              Menu
            </Text>
          </View>
        )}
      </Pressable>
    </View>
  );
}
