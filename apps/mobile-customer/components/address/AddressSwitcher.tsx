// AddressSwitcher — compact "Delivering to <label> · <city>" pill for the
// Home screen header. Shows whichever address useCustomerCoords would pick
// (see hooks/useCustomerCoords.ts's pickActiveAddress) and opens
// AddressSwitcherSheet to view/change it.
//
// iOS Pressable inner-View pattern: no style array on Pressable itself —
// layout/background live on the inner View.

import { useEffect } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { ChevronDown, MapPin } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useActiveAddress } from '../../hooks/useCustomerCoords';
import { useIsGuest } from '../../hooks/useRequireAccount';
import { useDeviceLocation } from '../../hooks/useDeviceLocation';

interface AddressSwitcherProps {
  /** Opens the address sheet. The sheet is mounted at the screen root (a
      full-screen sibling of the FlatList) so @gorhom can overlay the whole
      screen — mounting it inside this header trapped it in the header's layout
      box and caused the sheet to render over the search bar/tabs. */
  onOpen: () => void;
}

export function AddressSwitcher({ onOpen }: AddressSwitcherProps) {
  const { address, addresses, isLoading } = useActiveAddress();
  const isGuest = useIsGuest();
  const { location: deviceLocation, status: locationStatus, request } = useDeviceLocation();

  // Ask once, the first time a guest lands on Home. Deliberately here rather
  // than on app start: the customer sees the kitchens first and the OS prompt
  // second, so "why does this want my location" already has an obvious answer.
  // 'idle' means neither cached nor asked — a denial parks it on 'denied' and
  // it is never asked again.
  useEffect(() => {
    if (isGuest && locationStatus === 'idle' && !deviceLocation) {
      void request();
    }
  }, [isGuest, locationStatus, deviceLocation, request]);

  const hasAnyAddress = addresses.length > 0;
  // A guest has no saved addresses to load, so "Loading address…" would be a
  // spinner that never resolves. Show wherever the phone says it is, and fall
  // back to inviting them to pick an area when the OS won't say.
  const guestLabel =
    locationStatus === 'locating'
      ? 'Finding you…'
      : deviceLocation?.label
        ? deviceLocation.label
        : 'Set your location';
  const triggerLabel = isGuest
    ? guestLabel
    : isLoading
      ? 'Loading address…'
      : address
        ? `${address.label || 'Address'} · ${address.city}`
        : hasAnyAddress
          ? 'Select delivery address'
          : 'Add delivery address';

  const accessibilityLabel = isGuest
    ? deviceLocation?.label
      ? `Showing kitchens near ${deviceLocation.label}. Tap to change your area.`
      : 'No location set. Tap to choose your area.'
    : address
      ? `Delivering to ${address.label || 'address'}, ${address.city}. Tap to change address.`
      : hasAnyAddress
        ? 'No address selected for delivery. Tap to choose one.'
        : 'No delivery address saved. Tap to add one.';

  return (
    <Pressable
      onPress={onOpen}
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
