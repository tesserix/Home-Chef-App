// AddressSwitcher — compact "Delivering to <label> · <city>" pill for the
// Home screen header. Shows whichever address useCustomerCoords would pick
// (see hooks/useCustomerCoords.ts's pickActiveAddress) and opens
// AddressSwitcherSheet to view/change it.
//
// iOS Pressable inner-View pattern: no style array on Pressable itself —
// layout/background live on the inner View.

import { useRef } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { ChevronDown, MapPin } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { BottomSheetMethods } from '@gorhom/bottom-sheet/lib/typescript/types';
import { useActiveAddress } from '../../hooks/useCustomerCoords';
import { AddressSwitcherSheet } from './AddressSwitcherSheet';

export function AddressSwitcher() {
  const sheetRef = useRef<BottomSheetMethods>(null);
  const { address, addresses, isLoading } = useActiveAddress();

  function handleOpen() {
    sheetRef.current?.expand();
  }

  const hasAnyAddress = addresses.length > 0;
  const triggerLabel = isLoading
    ? 'Loading address…'
    : address
      ? `${address.label || 'Address'} · ${address.city}`
      : hasAnyAddress
        ? 'Select delivery address'
        : 'Add delivery address';

  const accessibilityLabel = address
    ? `Delivering to ${address.label || 'address'}, ${address.city}. Tap to change address.`
    : hasAnyAddress
      ? 'No address selected for delivery. Tap to choose one.'
      : 'No delivery address saved. Tap to add one.';

  return (
    <>
      <Pressable
        onPress={handleOpen}
        accessibilityRole="button"
        accessibilityLabel={accessibilityLabel}
      >
        <View style={styles.trigger}>
          <MapPin
            size={16}
            color={customerColors.coral.DEFAULT}
            accessibilityElementsHidden
          />
          <Text style={styles.triggerText} numberOfLines={1}>
            {triggerLabel}
          </Text>
          <ChevronDown
            size={16}
            color={customerColors.charcoal.soft}
            accessibilityElementsHidden
          />
        </View>
      </Pressable>

      {/* Mounted always (index -1 = hidden) so @gorhom can manage its own
          gesture/animation state, matching the FilterSheet pattern. */}
      <AddressSwitcherSheet ref={sheetRef} />
    </>
  );
}

const styles = StyleSheet.create({
  trigger: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    minHeight: 44,
    paddingHorizontal: 16,
    paddingTop: 12,
  },
  triggerText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    flexShrink: 1,
  },
});
