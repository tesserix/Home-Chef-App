// Add-address screen — reached from AddressSwitcherSheet's "Add new address"
// row (components/address/AddressSwitcherSheet.tsx). Mirrors the onboarding
// address step's autocomplete + fields (app/(onboarding)/address.tsx) and
// checkout's inline add-address form (app/checkout.tsx), but is a standalone
// route: it creates the address via useCreateAddress and returns to the
// previous screen, rather than writing to the onboarding draft store or
// staying inline in checkout. Reuses AddressLabelSelect (components/address/)
// and useAddressAutocomplete — no new address-entry UI invented.

import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Keyboard,
  Platform,
  Pressable,
  ScrollView,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { router } from 'expo-router';
import { ChevronLeft, Search } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import {
  useAddressAutocomplete,
  type AddressSuggestion,
} from '../../hooks/useLocations';
import { useCreateAddress } from '../../hooks/useAddresses';
import { AddressLabelSelect } from '../../components/address/AddressLabelSelect';
import { friendlyErrorMessage } from '../../lib/errors';

const schema = z.object({
  label: z.string().min(1),
  addressLine2: z.string().optional(),
  addressLine1: z.string().min(5, 'Address must be at least 5 characters'),
  city: z.string().min(1, 'City is required'),
  state: z.string().min(1, 'State is required'),
  pincode: z.string().regex(/^\d{6}$/, 'Enter a valid 6-digit pincode'),
});

type AddressForm = z.infer<typeof schema>;

export default function AddAddressScreen() {
  const createAddress = useCreateAddress();

  const {
    control,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<AddressForm>({
    resolver: zodResolver(schema),
    defaultValues: {
      label: 'Home',
      addressLine2: '',
      addressLine1: '',
      city: '',
      state: '',
      pincode: '',
    },
  });

  // Address autocomplete (Photon/OpenStreetMap via the backend) — a shortcut
  // that fills the fields below; they stay editable.
  const [addressQuery, setAddressQuery] = useState('');
  const [showSuggestions, setShowSuggestions] = useState(false);
  const { data: suggestions = [], isFetching: isSearching } =
    useAddressAutocomplete(addressQuery);

  // Geocoded coords from the picked suggestion. Cleared on manual edit of the
  // street line since the geocode would be stale — mirrors the onboarding
  // address step's behaviour.
  const [coords, setCoords] = useState<{ lat: number; lon: number } | null>(null);

  function pickSuggestion(s: AddressSuggestion): void {
    setValue('addressLine1', s.line1 || s.description, { shouldValidate: true });
    if (s.city) setValue('city', s.city, { shouldValidate: true });
    if (s.region) setValue('state', s.region, { shouldValidate: true });
    if (s.postal) setValue('pincode', s.postal, { shouldValidate: true });
    setCoords(
      typeof s.lat === 'number' && typeof s.lon === 'number'
        ? { lat: s.lat, lon: s.lon }
        : null,
    );
    setAddressQuery('');
    setShowSuggestions(false);
    Keyboard.dismiss();
  }

  async function onSubmit(data: AddressForm): Promise<void> {
    try {
      await createAddress.mutateAsync({
        label: data.label,
        addressLine1: data.addressLine1,
        addressLine2: data.addressLine2 ?? '',
        city: data.city,
        state: data.state,
        pincode: data.pincode,
        latitude: coords ? coords.lat : undefined,
        longitude: coords ? coords.lon : undefined,
        isDefault: false,
      });
      router.back();
    } catch (err) {
      Alert.alert('Could not save address', friendlyErrorMessage(err));
    }
  }

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      <View className="flex-row items-center border-b border-hairline px-4 pb-4 gap-3">
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
        >
          <View className="p-1">
            <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
          </View>
        </Pressable>
        <Text className="text-xl font-semibold text-charcoal flex-1">
          Add address
        </Text>
      </View>

      <KeyboardAvoidingView
        className="flex-1"
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      >
        <ScrollView
          className="flex-1"
          contentContainerStyle={{ padding: 24, paddingTop: 20 }}
          keyboardShouldPersistTaps="handled"
        >
          {/* ── Label selector — Home / Work / Other ── */}
          <View className="mb-5">
            <Controller
              control={control}
              name="label"
              render={({ field: { onChange, value } }) => (
                <AddressLabelSelect value={value} onChange={onChange} />
              )}
            />
          </View>

          {/* ── Address autocomplete (Photon/OpenStreetMap) ── */}
          <View className="flex-row items-center bg-surface-soft rounded-lg px-4 mb-2 gap-2">
            <Search size={18} color={customerColors.charcoal.soft} />
            <TextInput
              className="flex-1 h-12 text-base text-charcoal"
              placeholder="Search for your address"
              placeholderTextColor={customerColors.charcoal.soft}
              value={addressQuery}
              onChangeText={(t) => {
                setAddressQuery(t);
                setShowSuggestions(true);
              }}
              autoCapitalize="words"
              autoCorrect={false}
              returnKeyType="search"
              accessibilityLabel="Search for your address"
            />
            {isSearching ? (
              <ActivityIndicator size="small" color={customerColors.coral.DEFAULT} />
            ) : null}
          </View>

          {showSuggestions && suggestions.length > 0 ? (
            <View className="bg-surface border border-hairline rounded-lg mb-4 overflow-hidden">
              {suggestions.map((s, i) => (
                <Pressable
                  key={`${s.description}-${i}`}
                  onPress={() => pickSuggestion(s)}
                  accessibilityRole="button"
                  accessibilityLabel={`Use address ${s.description}`}
                >
                  <View
                    className={`px-4 py-3 ${
                      i < suggestions.length - 1 ? 'border-b border-hairline' : ''
                    }`}
                  >
                    <Text className="text-sm text-charcoal leading-5" numberOfLines={2}>
                      {s.description}
                    </Text>
                  </View>
                </Pressable>
              ))}
            </View>
          ) : null}

          {/* ── Flat / House / Floor (user-typed) ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">
            Flat / House / Floor no.
          </Text>
          <Controller
            control={control}
            name="addressLine2"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-4"
                placeholder="e.g. Flat 402, B-Wing"
                placeholderTextColor={customerColors.charcoal.soft}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                autoCapitalize="words"
                returnKeyType="next"
                accessibilityLabel="Flat, house or floor number"
              />
            )}
          />

          {/* ── Address Line 1 (autocompleted) ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">
            Building / Society / Area
          </Text>
          <Controller
            control={control}
            name="addressLine1"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                style={{
                  borderWidth: errors.addressLine1 ? 1.5 : 0,
                  borderColor: errors.addressLine1
                    ? customerColors.destructive.DEFAULT
                    : 'transparent',
                }}
                placeholder="House no., street, area"
                placeholderTextColor={customerColors.charcoal.soft}
                onBlur={onBlur}
                onChangeText={(t) => {
                  // Manual edit invalidates the picked-suggestion geocode.
                  onChange(t);
                  setCoords(null);
                }}
                value={value}
                autoCapitalize="words"
                returnKeyType="next"
                accessibilityLabel="Address line 1"
              />
            )}
          />
          {errors.addressLine1 ? (
            <Text className="text-xs text-destructive mb-3">
              {errors.addressLine1.message}
            </Text>
          ) : (
            <View className="mb-4" />
          )}

          {/* ── City ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">City</Text>
          <Controller
            control={control}
            name="city"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                style={{
                  borderWidth: errors.city ? 1.5 : 0,
                  borderColor: errors.city
                    ? customerColors.destructive.DEFAULT
                    : 'transparent',
                }}
                placeholder="Enter your city"
                placeholderTextColor={customerColors.charcoal.soft}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                autoCapitalize="words"
                returnKeyType="next"
                accessibilityLabel="City"
              />
            )}
          />
          {errors.city ? (
            <Text className="text-xs text-destructive mb-3">{errors.city.message}</Text>
          ) : (
            <View className="mb-4" />
          )}

          {/* ── State ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">State</Text>
          <Controller
            control={control}
            name="state"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                style={{
                  borderWidth: errors.state ? 1.5 : 0,
                  borderColor: errors.state
                    ? customerColors.destructive.DEFAULT
                    : 'transparent',
                }}
                placeholder="Enter your state"
                placeholderTextColor={customerColors.charcoal.soft}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                autoCapitalize="words"
                returnKeyType="next"
                accessibilityLabel="State"
              />
            )}
          />
          {errors.state ? (
            <Text className="text-xs text-destructive mb-3">{errors.state.message}</Text>
          ) : (
            <View className="mb-4" />
          )}

          {/* ── Pincode ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">Pincode</Text>
          <Controller
            control={control}
            name="pincode"
            render={({ field: { onChange, onBlur, value } }) => (
              <TextInput
                className="h-12 bg-surface-soft rounded-lg px-4 text-base text-charcoal mb-1"
                style={{
                  borderWidth: errors.pincode ? 1.5 : 0,
                  borderColor: errors.pincode
                    ? customerColors.destructive.DEFAULT
                    : 'transparent',
                }}
                placeholder="6-digit pincode"
                placeholderTextColor={customerColors.charcoal.soft}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                keyboardType="numeric"
                maxLength={6}
                returnKeyType="done"
                accessibilityLabel="Pincode"
              />
            )}
          />
          {errors.pincode ? (
            <Text className="text-xs text-destructive mb-3">{errors.pincode.message}</Text>
          ) : (
            <View className="mb-4" />
          )}

          {/* ── Primary CTA — iOS Pressable pattern: styles on the inner View ── */}
          <Pressable
            onPress={() => void handleSubmit(onSubmit)()}
            disabled={createAddress.isPending}
            accessibilityRole="button"
            accessibilityLabel="Save address"
          >
            <View
              className={`rounded-lg min-h-[52px] items-center justify-center mt-4 bg-coral ${
                createAddress.isPending ? 'opacity-70' : ''
              }`}
            >
              {createAddress.isPending ? (
                <ActivityIndicator size="small" color={customerColors.canvas} />
              ) : (
                <Text className="text-canvas font-semibold text-base">Save address</Text>
              )}
            </View>
          </Pressable>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
