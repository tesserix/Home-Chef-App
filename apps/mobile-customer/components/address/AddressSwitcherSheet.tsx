// AddressSwitcherSheet — lists the customer's saved delivery addresses and
// lets them switch which one is active (drives chef discovery + delivery-area
// checks via useCustomerCoords / useActiveAddress).
//
// Selecting a different address PUTs it back as the server default
// (useSetDefaultAddress), which invalidates both the address list and the
// chef/discovery queries so the home feed re-fetches against the new coords.
//
// Design: @gorhom/bottom-sheet at 60% snap, matching FilterSheet's chrome
// (radius-lg sheet, hairline separators, coral active state). iOS Pressable
// inner-View pattern throughout — never a function-style style array on
// Pressable.

import { forwardRef, useCallback } from 'react';
import { ActivityIndicator, Alert, Pressable, StyleSheet, Text, View } from 'react-native';
import { router } from 'expo-router';
import BottomSheet, { BottomSheetScrollView } from '@gorhom/bottom-sheet';
import type { BottomSheetMethods } from '@gorhom/bottom-sheet/lib/typescript/types';
import { Check, Plus } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useAddresses, useSetDefaultAddress } from '../../hooks/useAddresses';
import { hasUsableCoords, pickActiveAddress } from '../../hooks/useCustomerCoords';
import { friendlyErrorMessage } from '../../lib/errors';
import type { Address } from '../../types/customer';

function formatShortLine(addr: Address): string {
  const parts = [addr.city];
  if (addr.pincode) parts.push(addr.pincode);
  return parts.filter(Boolean).join(' ');
}

export const AddressSwitcherSheet = forwardRef<BottomSheetMethods>((_props, ref) => {
  const { data, isLoading } = useAddresses();
  const addresses = data?.data ?? [];
  const activeAddress = pickActiveAddress(addresses);
  const setDefault = useSetDefaultAddress();

  const handleClose = useCallback(() => {
    if (ref && 'current' in ref && ref.current) {
      ref.current.close();
    }
  }, [ref]);

  async function handleSelect(addr: Address) {
    if (addr.id === activeAddress?.id || setDefault.isPending) return;
    try {
      await setDefault.mutateAsync(addr);
      handleClose();
    } catch (err) {
      Alert.alert('Could not switch address', friendlyErrorMessage(err));
    }
  }

  function handleAddAddress() {
    handleClose();
    router.push('/address/add');
  }

  return (
    <BottomSheet
      ref={ref}
      index={-1}
      snapPoints={['60%']}
      enablePanDownToClose
      backgroundStyle={styles.sheetBackground}
      handleIndicatorStyle={styles.handleIndicator}
    >
      <BottomSheetScrollView
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
      >
        {/* ── Sheet header ── */}
        <View style={styles.sheetHeader}>
          <Text style={styles.sheetTitle}>Delivery address</Text>
          <Pressable
            onPress={handleClose}
            accessibilityRole="button"
            accessibilityLabel="Close"
          >
            <View style={styles.doneButton}>
              <Text style={styles.doneLabel}>Done</Text>
            </View>
          </Pressable>
        </View>

        {isLoading ? (
          <View style={styles.loadingState}>
            <ActivityIndicator size="small" color={customerColors.coral.DEFAULT} />
          </View>
        ) : addresses.length === 0 ? (
          <View style={styles.emptyState}>
            <Text style={styles.emptyTitle}>No saved addresses yet</Text>
            <Text style={styles.emptyBody}>
              Add an address so chefs near you show up first.
            </Text>
          </View>
        ) : (
          <View style={styles.list} accessibilityRole="radiogroup">
            {addresses.map((addr) => {
              const isActive = addr.id === activeAddress?.id;
              const isMissingCoords = !hasUsableCoords(addr);
              return (
                <Pressable
                  key={addr.id}
                  onPress={() => handleSelect(addr)}
                  disabled={setDefault.isPending}
                  accessibilityRole="radio"
                  accessibilityState={{ checked: isActive, disabled: setDefault.isPending }}
                  accessibilityLabel={`${addr.label || 'Address'}, ${formatShortLine(addr)}${
                    isActive ? ', currently active' : ''
                  }`}
                >
                  <View style={[styles.row, isActive && styles.rowActive]}>
                    <View style={[styles.radio, isActive && styles.radioActive]}>
                      {isActive ? (
                        <Check size={12} color={customerColors.canvas} />
                      ) : null}
                    </View>
                    <View style={styles.rowBody}>
                      <View style={styles.rowHeaderLine}>
                        <Text style={styles.rowLabel}>{addr.label || 'Address'}</Text>
                        {addr.isDefault ? (
                          <Text style={styles.defaultTag}>Default</Text>
                        ) : null}
                      </View>
                      <Text style={styles.rowLine} numberOfLines={1}>
                        {addr.addressLine1}, {formatShortLine(addr)}
                      </Text>
                      {isMissingCoords ? (
                        <Text style={styles.rowWarning}>
                          No location saved — won&apos;t affect nearby chefs
                        </Text>
                      ) : null}
                    </View>
                  </View>
                </Pressable>
              );
            })}
          </View>
        )}

        {/* ── Add new address ── */}
        <Pressable
          onPress={handleAddAddress}
          accessibilityRole="button"
          accessibilityLabel="Add a new address"
        >
          <View style={styles.addRow}>
            <Plus size={16} color={customerColors.coral.DEFAULT} />
            <Text style={styles.addLabel}>Add new address</Text>
          </View>
        </Pressable>

        <View style={{ height: 32 }} />
      </BottomSheetScrollView>
    </BottomSheet>
  );
});

AddressSwitcherSheet.displayName = 'AddressSwitcherSheet';

const styles = StyleSheet.create({
  sheetBackground: {
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
    backgroundColor: customerColors.canvas,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: -4 },
    shadowOpacity: 0.08,
    shadowRadius: 16,
    elevation: 8,
  },
  handleIndicator: {
    backgroundColor: customerColors.hairline,
    width: 36,
    height: 4,
  },
  scrollContent: {
    paddingTop: 8,
    paddingBottom: 0,
  },

  sheetHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 20,
    paddingTop: 4,
    paddingBottom: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
    marginBottom: 8,
  },
  sheetTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 17,
    color: customerColors.charcoal.DEFAULT,
    letterSpacing: -0.2,
  },
  doneButton: {
    paddingHorizontal: 12,
    paddingVertical: 6,
    minHeight: 36,
    justifyContent: 'center',
  },
  doneLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.coral.DEFAULT,
  },

  loadingState: {
    paddingVertical: 32,
    alignItems: 'center',
  },
  emptyState: {
    paddingHorizontal: 20,
    paddingVertical: 24,
    gap: 4,
  },
  emptyTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },
  emptyBody: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
  },

  list: {
    paddingHorizontal: 16,
    gap: 8,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 12,
    padding: 12,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.canvas,
    minHeight: 44,
  },
  rowActive: {
    borderColor: customerColors.coral.DEFAULT,
    backgroundColor: customerColors.coral.tint,
  },
  radio: {
    width: 20,
    height: 20,
    borderRadius: 10,
    borderWidth: 2,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 1,
  },
  radioActive: {
    borderColor: customerColors.coral.DEFAULT,
    backgroundColor: customerColors.coral.DEFAULT,
  },
  rowBody: {
    flex: 1,
    gap: 2,
  },
  rowHeaderLine: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  rowLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
  },
  defaultTag: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    color: customerColors.coral.DEFAULT,
  },
  rowLine: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
  },
  rowWarning: {
    fontFamily: 'Inter',
    fontSize: 11,
    color: customerColors.charcoal.soft,
    marginTop: 2,
  },

  addRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    paddingHorizontal: 20,
    paddingVertical: 14,
    marginTop: 8,
    minHeight: 44,
  },
  addLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.coral.DEFAULT,
  },
});
