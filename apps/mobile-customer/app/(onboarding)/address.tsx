import { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  Pressable,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  ActivityIndicator,
  Keyboard,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { router } from 'expo-router';
import { Search, MapPin } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import {
  useAddressAutocomplete,
  type AddressSuggestion,
} from '../../hooks/useLocations';
import { AddressLabelSelect } from '../../components/address/AddressLabelSelect';
import { useCustomerOnboardingStore } from '../../store/onboarding-store';

const schema = z.object({
  label: z.string().min(1),
  addressLine2: z.string().optional(), // flat / house / floor — user-typed
  addressLine1: z.string().min(5, 'Address must be at least 5 characters'),
  city: z.string().min(1, 'City is required'),
  state: z.string().min(1, 'State is required'),
  pincode: z.string().regex(/^\d{6}$/, 'Enter a valid 6-digit pincode'),
});

type AddressForm = z.infer<typeof schema>;

export default function AddressScreen() {
  const draft = useCustomerOnboardingStore();

  const {
    control,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<AddressForm>({
    resolver: zodResolver(schema),
    // Seed from the saved draft so navigating back to this step (or resuming
    // after the app was backgrounded) shows the entered address, not a blank.
    defaultValues: {
      label: draft.label || 'Home',
      addressLine2: draft.addressLine2,
      addressLine1: draft.addressLine1,
      city: draft.city,
      state: draft.state,
      pincode: draft.pincode,
    },
  });

  // Address autocomplete (Photon/OpenStreetMap via the backend). The search
  // box is a shortcut — it fills the fields below, which stay editable.
  const [addressQuery, setAddressQuery] = useState('');
  const [showSuggestions, setShowSuggestions] = useState(false);
  const { data: suggestions = [], isFetching: isSearching } =
    useAddressAutocomplete(addressQuery);

  // Geocoded coordinates from the picked suggestion. Persisted on the address
  // so the server can run delivery-zone checks + live 3PL quotes. Cleared when
  // the user manually edits the street line (the geocode would be stale), in
  // which case the server falls back to a flat delivery fee.
  const [coords, setCoords] = useState<{ lat: number; lon: number } | null>(
    draft.latitude != null && draft.longitude != null
      ? { lat: draft.latitude, lon: draft.longitude }
      : null,
  );

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

  const onSubmit = (data: AddressForm) => {
    draft.update({
      label: data.label,
      addressLine1: data.addressLine1,
      addressLine2: data.addressLine2 ?? '',
      city: data.city,
      state: data.state,
      pincode: data.pincode,
      latitude: coords ? coords.lat : null,
      longitude: coords ? coords.lon : null,
    });
    router.push('/(onboarding)/preferences');
  };

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      <KeyboardAvoidingView
        className="flex-1"
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      >
        <ScrollView
          className="flex-1"
          contentContainerStyle={{ padding: 24, paddingTop: 40 }}
          keyboardShouldPersistTaps="handled"
        >
          {/* ── Step progress ── */}
          <Text className="text-[13px] text-charcoal-soft mb-2">
            Step 2 of 3
          </Text>
          <View className="h-1 bg-hairline rounded-full mb-8 overflow-hidden">
            <View className="h-1 bg-coral rounded-full" style={{ width: '66%' }} />
          </View>

          {/* ── Heading ── */}
          <Text className="text-[26px] font-bold text-charcoal tracking-tight font-display mb-2">
            Your delivery address
          </Text>
          <Text className="text-[15px] text-charcoal-soft mb-6">
            Where should we deliver your orders?
          </Text>

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
          {/* A search shortcut that fills the fields below; they stay editable. */}
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
                  {({ pressed }) => (
                    <View
                      className={`flex-row items-start gap-3 px-4 py-3 ${
                        pressed ? 'bg-surface-soft' : ''
                      } ${i < suggestions.length - 1 ? 'border-b border-hairline' : ''}`}
                    >
                      <MapPin
                        size={18}
                        color={customerColors.charcoal.soft}
                        style={{ marginTop: 2 }}
                      />
                      <Text
                        className="flex-1 text-sm text-charcoal leading-5"
                        numberOfLines={2}
                      >
                        {s.description}
                      </Text>
                    </View>
                  )}
                </Pressable>
              ))}
            </View>
          ) : null}

          {/* ── Flat / House / Floor (user-typed; autocomplete won't fill this) ── */}
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

          {/* ── Address Line 1 (building / society / area — autocompleted) ── */}
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
                  // (pickSuggestion uses setValue, which doesn't fire this.)
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
            <Text className="text-xs text-destructive mb-3">
              {errors.city.message}
            </Text>
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
            <Text className="text-xs text-destructive mb-3">
              {errors.state.message}
            </Text>
          ) : (
            <View className="mb-4" />
          )}

          {/* ── Pincode ── */}
          <Text className="text-sm font-medium text-charcoal mb-1">
            Pincode
          </Text>
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
            <Text className="text-xs text-destructive mb-3">
              {errors.pincode.message}
            </Text>
          ) : (
            <View className="mb-4" />
          )}

          {/* ── Primary CTA ── */}
          {/* iOS Pressable pattern: visual styles on inner View */}
          <Pressable
            onPress={() => void handleSubmit(onSubmit)()}
            accessibilityRole="button"
            accessibilityLabel="Continue to preferences"
          >
            {({ pressed }) => (
              <View
                className={`rounded-lg min-h-[52px] items-center justify-center mt-4 ${
                  pressed ? 'opacity-90' : ''
                } bg-coral`}
              >
                <Text className="text-canvas font-semibold text-base">
                  Continue
                </Text>
              </View>
            )}
          </Pressable>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
