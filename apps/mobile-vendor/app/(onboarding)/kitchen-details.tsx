// apps/mobile-vendor/app/(onboarding)/kitchen-details.tsx
// Step 2/6 — Kitchen name, cuisine multi-select, description, address.
//
// Address is India-only (the only country we serve today). State and city
// come from the backend reference data via useStates() / useCities(); the
// PIN field carries an autocomplete that resolves to a full (state, city,
// PIN) triple in one tap.

import { useState as useReactState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { router } from 'expo-router';
import * as Location from 'expo-location';
import { MapPin } from 'lucide-react-native';
import { Input, OnboardingScaffold } from '@homechef/mobile-shared/ui';
import { useToast } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { useVendorOnboardingStore } from '../../store/onboarding-store';
import { api } from '../../lib/api';
import {
  useStates,
  useCities,
  usePostcodeSearch,
  type State,
  type City,
  type PostcodeSearchResult,
} from '../../hooks/useLocations';

const CUISINE_OPTIONS = [
  'North Indian',
  'South Indian',
  'Chinese',
  'Continental',
  'Bakery',
  'Snacks',
  'Beverages',
  'Other',
] as const;

const schema = z.object({
  businessName: z.string().min(3, 'Business name must be at least 3 characters'),
  cuisines: z.array(z.string()).min(1, 'Select at least one cuisine type'),
  description: z
    .string()
    .min(50, 'Description must be at least 50 characters')
    .max(500, 'Description must be at most 500 characters'),
  addressLine1: z.string().min(3, 'Address line 1 is required'),
  addressLine2: z.string(),
  city: z.string().min(2, 'City is required'),
  state: z.string().min(2, 'State is required'),
  // India PIN codes are exactly 6 digits. Tightened from the legacy >=4
  // check now that we enforce India-only.
  postalCode: z
    .string()
    .regex(/^\d{6}$/, 'PIN code must be exactly 6 digits'),
});

type FormValues = z.infer<typeof schema>;

export default function KitchenDetailsScreen() {
  const { updateKitchenDetails, setStep } = useVendorOnboardingStore();

  const {
    control,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      businessName: '',
      cuisines: [],
      description: '',
      addressLine1: '',
      addressLine2: '',
      city: '',
      state: '',
      postalCode: '',
    },
  });

  const selectedCuisines = watch('cuisines');
  const descriptionValue = watch('description');
  const selectedStateName = watch('state');
  const selectedCityName = watch('city');
  const postalCodeValue = watch('postalCode');

  const states = useStates();
  // Resolve the local state code (e.g. "MH") from the selected state name
  // so we can ask the backend for that state's cities.
  const selectedStateCode = states.data?.find(
    (s) => s.name === selectedStateName,
  )?.code ?? null;
  const cities = useCities(selectedStateCode);

  // Postcode autocomplete is driven by an internal-only query string so
  // the suggestions panel can stay open while the form's postalCode value
  // is empty (until the user picks a result).
  const [postcodeQuery, setPostcodeQuery] = useReactState('');
  const postcodeSearch = usePostcodeSearch(postcodeQuery);
  const [showPostcodeSuggestions, setShowPostcodeSuggestions] = useReactState(false);

  // Geolocation auto-fill state — drives the spinner on the CTA.
  const [locating, setLocating] = useReactState(false);
  const { show: showToast } = useToast();

  function toggleCuisine(cuisine: string): void {
    const current = selectedCuisines ?? [];
    if (current.includes(cuisine)) {
      setValue('cuisines', current.filter((c) => c !== cuisine), { shouldValidate: true });
    } else {
      setValue('cuisines', [...current, cuisine], { shouldValidate: true });
    }
  }

  function pickState(s: State): void {
    setValue('state', s.name, { shouldValidate: true });
    // Clear the city when the state changes — the prior selection
    // belongs to a different state and would be misleading.
    setValue('city', '', { shouldValidate: false });
  }

  function pickCity(c: City): void {
    setValue('city', c.name, { shouldValidate: true });
  }

  function pickPostcodeSuggestion(item: {
    code: string;
    cityName: string;
    stateName: string;
  }): void {
    setValue('postalCode', item.code, { shouldValidate: true });
    setValue('city', item.cityName, { shouldValidate: true });
    setValue('state', item.stateName, { shouldValidate: true });
    setPostcodeQuery(item.code);
    setShowPostcodeSuggestions(false);
  }

  // GPS auto-fill. Asks for foreground location permission, reverse-
  // geocodes the device's current position, and best-effort fills every
  // address field so the chef rarely has to type any of them. Manual
  // entry stays available regardless — this is purely additive.
  //
  // We canonicalize state + city against /locations/postcodes/search so
  // the names match our seeded reference data (otherwise iOS might
  // return "Bengaluru" while we serve "Bengaluru", or "Karnataka" vs
  // "Karnatak"). When the canonicalize lookup misses we still fall
  // through to the raw reverse-geocode values rather than leave fields
  // blank — the chef can adjust either way.
  async function handleUseCurrentLocation(): Promise<void> {
    setLocating(true);
    try {
      const perm = await Location.requestForegroundPermissionsAsync();
      if (perm.status !== 'granted') {
        showToast({
          message: 'Location permission denied — enter your address manually.',
          tone: 'error',
        });
        return;
      }
      const position = await Location.getCurrentPositionAsync({
        accuracy: Location.Accuracy.Balanced,
      });
      const results = await Location.reverseGeocodeAsync({
        latitude: position.coords.latitude,
        longitude: position.coords.longitude,
      });
      const top = results[0];
      if (!top) {
        showToast({
          message: "We couldn't read your address from GPS — enter it manually.",
          tone: 'error',
        });
        return;
      }
      // Block non-India locations: the address form, state list, and PIN
      // validation are all India-only. Bailing here keeps the chef from
      // saving an unusable address.
      if (top.isoCountryCode && top.isoCountryCode !== 'IN') {
        showToast({
          message: `HomeChef is India-only today — detected ${top.country ?? top.isoCountryCode}.`,
          tone: 'error',
        });
        return;
      }

      // addressLine1 — best-effort assembly of building + street.
      const line1 = [top.name, top.street].filter(Boolean).join(', ');
      if (line1) setValue('addressLine1', line1, { shouldValidate: true });

      // Try the canonicalized path: PIN → /postcodes/search → matched
      // state/city/PIN that align with our seed data.
      let canonicalized = false;
      if (top.postalCode) {
        try {
          const r = await api.get<{ data: PostcodeSearchResult[] }>(
            `/locations/postcodes/search?q=${encodeURIComponent(top.postalCode)}`,
          );
          const hit = r.data.data.find((row) => row.code === top.postalCode);
          if (hit) {
            setValue('postalCode', hit.code, { shouldValidate: true });
            setValue('city', hit.cityName, { shouldValidate: true });
            setValue('state', hit.stateName, { shouldValidate: true });
            setPostcodeQuery(hit.code);
            canonicalized = true;
          }
        } catch {
          // Network hiccup — fall through to the raw reverse-geocode
          // values below.
        }
      }

      if (!canonicalized) {
        if (top.postalCode) {
          setValue('postalCode', top.postalCode, { shouldValidate: true });
          setPostcodeQuery(top.postalCode);
        }
        if (top.city) setValue('city', top.city, { shouldValidate: true });
        if (top.region) setValue('state', top.region, { shouldValidate: true });
      }

      showToast({
        message: 'Address filled from your location — adjust as needed.',
        tone: 'success',
      });
    } catch (err) {
      showToast({
        message: 'Location lookup failed — enter your address manually.',
        tone: 'error',
      });
    } finally {
      setLocating(false);
    }
  }

  function onSubmit(data: FormValues): void {
    updateKitchenDetails(data);
    setStep(3);
    router.push('/(onboarding)/operations');
  }

  function onInvalid(errs: typeof errors): void {
    const firstError = Object.values(errs)[0];
    if (firstError?.message) Alert.alert('Check your details', firstError.message);
  }

  return (
    <OnboardingScaffold
      step={2}
      total={6}
      title="Your kitchen"
      subtitle="Name, style, and address — what your customers will find."
      primaryLabel="Continue"
      onPrimary={handleSubmit(onSubmit, onInvalid)}
    >
      {/* Business name */}
      <Controller
        control={control}
        name="businessName"
        render={({ field: { onChange, onBlur, value } }) => (
          <Input
            label="Kitchen / business name"
            placeholder="e.g. Amma's Kitchen"
            onBlur={onBlur}
            onChangeText={onChange}
            value={value}
            autoCapitalize="words"
            error={errors.businessName?.message}
          />
        )}
      />

      {/* Cuisine chips — outlined pill pattern, ink border active */}
      <View style={styles.fieldGroup}>
        <Text style={styles.fieldLabel}>Cuisine types</Text>
        <View style={styles.chipRow}>
          {CUISINE_OPTIONS.map((cuisine) => {
            const selected = selectedCuisines?.includes(cuisine) ?? false;
            return (
              <View
                key={cuisine}
                style={[styles.chip, selected && styles.chipActive]}
              >
                <Text
                  style={[styles.chipLabel, selected && styles.chipLabelActive]}
                  onPress={() => toggleCuisine(cuisine)}
                  suppressHighlighting
                >
                  {cuisine}
                </Text>
              </View>
            );
          })}
        </View>
        {errors.cuisines ? (
          <Text style={styles.fieldError}>{errors.cuisines.message}</Text>
        ) : null}
      </View>

      {/* Description */}
      <View style={styles.fieldGroup}>
        <Controller
          control={control}
          name="description"
          render={({ field: { onChange, onBlur, value } }) => (
            <Input
              label="About your kitchen"
              placeholder="Describe your kitchen, specialties, and cooking style (min 50 characters)"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              multiline
              numberOfLines={4}
              inputStyle={styles.textArea}
              error={errors.description?.message}
            />
          )}
        />
        <Text style={styles.charCount}>{(descriptionValue ?? '').length} / 500</Text>
      </View>

      {/* Address divider */}
      <View style={styles.sectionDivider}>
        <View style={styles.hairline} />
        <Text style={styles.sectionLabel}>Kitchen address</Text>
        <View style={styles.hairline} />
      </View>

      {/* GPS auto-fill CTA — the fast path. One tap pulls state, city,
          PIN, and street from the device location; manual fields below
          stay editable as the always-available fallback. */}
      <Pressable
        onPress={handleUseCurrentLocation}
        disabled={locating}
        accessibilityRole="button"
        accessibilityLabel="Use my current location to fill the address"
      >
        {({ pressed }) => (
          <View
            style={[
              styles.locateCta,
              pressed && { opacity: 0.85 },
              locating && { opacity: 0.7 },
            ]}
          >
            {locating ? (
              <ActivityIndicator size="small" color={theme.colors.paper} />
            ) : (
              <MapPin size={18} color={theme.colors.paper} strokeWidth={2.2} />
            )}
            <Text style={styles.locateCtaLabel}>
              {locating ? 'Finding your address…' : 'Use my current location'}
            </Text>
          </View>
        )}
      </Pressable>
      <Text style={styles.locateCtaHelper}>
        Or fill the fields below manually.
      </Text>

      {/* Address fields */}
      <Controller
        control={control}
        name="addressLine1"
        render={({ field: { onChange, onBlur, value } }) => (
          <Input
            label="Address line 1"
            placeholder="House / building, street"
            onBlur={onBlur}
            onChangeText={onChange}
            value={value}
            autoCapitalize="words"
            error={errors.addressLine1?.message}
          />
        )}
      />

      <Controller
        control={control}
        name="addressLine2"
        render={({ field: { onChange, onBlur, value } }) => (
          <Input
            label="Address line 2"
            placeholder="Apartment, suite (optional)"
            onBlur={onBlur}
            onChangeText={onChange}
            value={value ?? ''}
            autoCapitalize="words"
          />
        )}
      />

      {/* State picker — horizontal chip strip from /locations/IN/states */}
      <View style={styles.fieldGroup}>
        <Text style={styles.fieldLabel}>State</Text>
        {states.isLoading ? (
          <View style={styles.pickerLoader}>
            <ActivityIndicator size="small" color={theme.colors.ink.muted} />
          </View>
        ) : (
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={styles.pickerStrip}
          >
            {(states.data ?? []).map((s) => {
              const selected = s.name === selectedStateName;
              return (
                <Pressable
                  key={s.id}
                  onPress={() => pickState(s)}
                  hitSlop={4}
                  accessibilityRole="button"
                  accessibilityState={{ selected }}
                >
                  {({ pressed }) => (
                    <View
                      style={[
                        styles.pickerChip,
                        selected && styles.pickerChipActive,
                        pressed && { opacity: 0.7 },
                      ]}
                    >
                      <Text
                        style={[
                          styles.pickerChipLabel,
                          selected && styles.pickerChipLabelActive,
                        ]}
                      >
                        {s.name}
                      </Text>
                    </View>
                  )}
                </Pressable>
              );
            })}
          </ScrollView>
        )}
        {errors.state ? (
          <Text style={styles.fieldError}>{errors.state.message}</Text>
        ) : null}
      </View>

      {/* City picker — horizontal chip strip; only meaningful once a
          state is selected, so we render a hint if not. Free-text entry
          stays available via the fallback Input below the strip. */}
      <View style={styles.fieldGroup}>
        <Text style={styles.fieldLabel}>City</Text>
        {!selectedStateCode ? (
          <Text style={styles.fieldHint}>Pick a state first.</Text>
        ) : cities.isLoading ? (
          <View style={styles.pickerLoader}>
            <ActivityIndicator size="small" color={theme.colors.ink.muted} />
          </View>
        ) : (cities.data?.length ?? 0) === 0 ? (
          <Text style={styles.fieldHint}>
            No seeded cities for this state yet — type your city below.
          </Text>
        ) : (
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={styles.pickerStrip}
          >
            {(cities.data ?? []).map((c) => {
              const selected = c.name === selectedCityName;
              return (
                <Pressable
                  key={c.id}
                  onPress={() => pickCity(c)}
                  hitSlop={4}
                >
                  {({ pressed }) => (
                    <View
                      style={[
                        styles.pickerChip,
                        selected && styles.pickerChipActive,
                        pressed && { opacity: 0.7 },
                      ]}
                    >
                      <Text
                        style={[
                          styles.pickerChipLabel,
                          selected && styles.pickerChipLabelActive,
                        ]}
                      >
                        {c.name}
                      </Text>
                    </View>
                  )}
                </Pressable>
              );
            })}
          </ScrollView>
        )}
        {/* Free-text fallback — kept so chefs in cities we haven't seeded
            yet aren't blocked from completing onboarding. */}
        <Controller
          control={control}
          name="city"
          render={({ field: { onChange, onBlur, value } }) => (
            <Input
              label=""
              placeholder="Or type city name"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              autoCapitalize="words"
              error={errors.city?.message}
            />
          )}
        />
      </View>

      {/* PIN code with autocomplete. Suggestions panel hangs underneath
          the input while the user is typing; tapping a row fills PIN,
          city, AND state in one go. */}
      <View style={styles.fieldGroup}>
        <Controller
          control={control}
          name="postalCode"
          render={({ field: { onChange, onBlur, value } }) => (
            <Input
              label="PIN code"
              placeholder="6-digit PIN or area name"
              onBlur={() => {
                onBlur();
                // Delay hiding so the Pressable above the keyboard has
                // a chance to register its tap.
                setTimeout(() => setShowPostcodeSuggestions(false), 150);
              }}
              onFocus={() => setShowPostcodeSuggestions(true)}
              onChangeText={(text) => {
                onChange(text);
                setPostcodeQuery(text);
                setShowPostcodeSuggestions(true);
              }}
              value={value}
              keyboardType="default"
              maxLength={40}
              error={errors.postalCode?.message}
              helper={value && !/^\d{6}$/.test(value) ? undefined : 'Tap a suggestion to autofill state + city.'}
            />
          )}
        />
        {showPostcodeSuggestions && postcodeQuery.trim().length >= 2 ? (
          <View style={styles.suggestionsPanel}>
            {postcodeSearch.isLoading ? (
              <View style={styles.suggestionLoader}>
                <ActivityIndicator size="small" color={theme.colors.ink.muted} />
              </View>
            ) : (postcodeSearch.data?.length ?? 0) === 0 ? (
              <Text style={styles.suggestionEmpty}>
                No match — type the 6-digit PIN manually.
              </Text>
            ) : (
              (postcodeSearch.data ?? []).map((item) => (
                <Pressable
                  key={item.code}
                  onPress={() => pickPostcodeSuggestion(item)}
                  accessibilityRole="button"
                >
                  {({ pressed }) => (
                    <View
                      style={[
                        styles.suggestionRow,
                        pressed && { backgroundColor: theme.colors.bone },
                      ]}
                    >
                      <Text style={styles.suggestionCode}>{item.code}</Text>
                      <Text style={styles.suggestionMeta} numberOfLines={1}>
                        {item.areaName} · {item.cityName}, {item.stateName}
                      </Text>
                    </View>
                  )}
                </Pressable>
              ))
            )}
          </View>
        ) : null}
        {postalCodeValue && !/^\d{6}$/.test(postalCodeValue) ? (
          <Text style={styles.fieldHint}>PIN code must be exactly 6 digits.</Text>
        ) : null}
      </View>

      {/* Spacer so last field clears sticky CTA */}
      <View style={styles.bottomSpacer} />
    </OnboardingScaffold>
  );
}

const styles = StyleSheet.create({
  fieldGroup: {
    gap: theme.spacing[1],
  },

  fieldLabel: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.soft,
    marginBottom: theme.spacing[2],
  },

  fieldError: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.destructive.DEFAULT,
    marginTop: theme.spacing[1],
  },

  fieldHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[1],
  },

  // Cuisine chips — outlined pill with ink-active state, 3–4 per row.
  chipRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: theme.spacing[2],
  },

  chip: {
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    borderRadius: theme.radius.full,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[1],
    minHeight: theme.touchTarget.vendor,
    alignItems: 'center',
    justifyContent: 'center',
  },

  chipActive: {
    borderColor: theme.colors.ink.DEFAULT,
    backgroundColor: theme.colors.ink.DEFAULT,
  },

  chipLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },

  chipLabelActive: {
    fontFamily: 'Inter-Medium',
    color: theme.colors.paper,
  },

  textArea: {
    minHeight: 100,
    textAlignVertical: 'top',
  },

  charCount: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    textAlign: 'right',
    marginTop: theme.spacing[1],
  },

  // Inline divider with centred label between form sections.
  sectionDivider: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    marginVertical: theme.spacing[3],
  },

  hairline: {
    flex: 1,
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
  },

  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.soft,
    letterSpacing: 0.4,
  },

  // GPS auto-fill CTA — ink-filled to read as a primary affordance
  // (this is the fast path we want chefs to take), but smaller than
  // the screen-level Continue button so it doesn't fight for attention.
  locateCta: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: theme.spacing[2],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    minHeight: theme.touchTarget.vendor,
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.ink.DEFAULT,
    marginBottom: theme.spacing[1],
  },
  locateCtaLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
    letterSpacing: 0.2,
  },
  locateCtaHelper: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    textAlign: 'center',
    marginBottom: theme.spacing[3],
  },

  // Picker strip — horizontally scrollable chip row used for State + City.
  pickerStrip: {
    gap: theme.spacing[2],
    paddingRight: theme.spacing[4],
    paddingBottom: theme.spacing[1],
  },
  pickerChip: {
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    borderRadius: 999,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    backgroundColor: theme.colors.paper,
    minHeight: 36,
    justifyContent: 'center',
  },
  pickerChipActive: {
    borderColor: theme.colors.ink.DEFAULT,
    backgroundColor: theme.colors.ink.DEFAULT,
  },
  pickerChipLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    letterSpacing: 0.1,
  },
  pickerChipLabelActive: {
    color: theme.colors.paper,
  },
  pickerLoader: {
    paddingVertical: theme.spacing[3],
    alignItems: 'flex-start',
  },

  // PIN autocomplete suggestion panel — appears under the input while
  // the user types. Capped via the API limit (20) so this can't grow
  // arbitrarily tall.
  suggestionsPanel: {
    marginTop: theme.spacing[2],
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.paper,
    overflow: 'hidden',
  },
  suggestionRow: {
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
  },
  suggestionCode: {
    fontFamily: 'Geist-Bold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  suggestionMeta: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  suggestionLoader: {
    paddingVertical: theme.spacing[4],
    alignItems: 'center',
  },
  suggestionEmpty: {
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
  },

  bottomSpacer: {
    height: theme.spacing[2],
  },
});
