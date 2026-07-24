// apps/mobile-vendor/app/(onboarding)/kitchen-details.tsx
// Step 2/6 — Kitchen name, cuisine multi-select, description, address.
//
// Address is India-only (the only country we serve today). State and city
// come from the backend reference data via useStates() / useCities(); the
// PIN field carries an autocomplete that resolves to a full (state, city,
// PIN) triple in one tap.

import { useRef, useState as useReactState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Platform,
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
import { useTranslation } from 'react-i18next';
import * as Location from 'expo-location';
import { MapPin, UtensilsCrossed, FileText, Navigation } from 'lucide-react-native';
import { Input, OnboardingScaffold } from '@homechef/mobile-shared/ui';
import { useToast } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { useVendorOnboardingStore } from '../../store/onboarding-store';
import { api } from '../../lib/api';
import {
  useStates,
  useCities,
  usePostcodeSearch,
  useAddressAutocomplete,
  type State,
  type City,
  type PostcodeSearchResult,
  type AddressSuggestion,
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
  businessName: z.string().min(3, 'onboarding.errBusinessNameMin'),
  cuisines: z.array(z.string()).min(1, 'onboarding.errCuisineMin'),
  description: z
    .string()
    .min(50, 'onboarding.errDescriptionMin')
    .max(500, 'onboarding.errDescriptionMax'),
  addressLine1: z.string().min(3, 'onboarding.errAddressLine1'),
  addressLine2: z.string(),
  city: z.string().min(2, 'onboarding.errCity'),
  state: z.string().min(2, 'onboarding.errState'),
  // India PIN codes are exactly 6 digits. Tightened from the legacy >=4
  // check now that we enforce India-only.
  postalCode: z
    .string()
    .regex(/^\d{6}$/, 'onboarding.errPin'),
});

type FormValues = z.infer<typeof schema>;

export default function KitchenDetailsScreen() {
  const { t } = useTranslation();
  const { kitchenDetails, updateKitchenDetails, setStep } = useVendorOnboardingStore();

  const {
    control,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    // Seed from the persisted draft so editing this step from Review (or
    // resuming after backgrounding) shows the saved values, not a blank form.
    defaultValues: {
      businessName: kitchenDetails.businessName,
      cuisines: kitchenDetails.cuisines,
      description: kitchenDetails.description,
      addressLine1: kitchenDetails.addressLine1,
      addressLine2: kitchenDetails.addressLine2,
      city: kitchenDetails.city,
      state: kitchenDetails.state,
      postalCode: kitchenDetails.postalCode,
    },
  });

  const selectedCuisines = watch('cuisines');
  const descriptionValue = watch('description');
  const selectedStateName = watch('state');
  const selectedCityName = watch('city');

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
  // Photon-backed worldwide autocomplete — runs in parallel with the
  // seeded /postcodes/search so the panel surfaces both kinds of hits
  // simultaneously.
  const addressAutocomplete = useAddressAutocomplete(postcodeQuery);
  const [showPostcodeSuggestions, setShowPostcodeSuggestions] = useReactState(false);

  // Geolocation auto-fill state — drives the spinner on the CTA.
  const [locating, setLocating] = useReactState(false);
  const { show: showToast } = useToast();

  // R14 — scroll the section with the first invalid field into view.
  const scrollRef = useRef<ScrollView>(null);
  const identityFieldY = useRef(0);
  const addressFieldY = useRef(0);
  const IDENTITY_FIELDS = new Set(['businessName', 'cuisines', 'description']);

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

  // pickAddressSuggestion handles a tap on a Photon-backed row from
  // /locations/autocomplete. The Photon result already includes
  // line1/city/region/postal, but we canonicalize the PIN against our
  // seeded /postcodes/search whenever possible so state + city names
  // match the chip-strip pickers (which only show seeded values).
  async function pickAddressSuggestion(item: AddressSuggestion): Promise<void> {
    if (item.line1) setValue('addressLine1', item.line1, { shouldValidate: true });
    let canonicalized = false;
    if (item.postal) {
      try {
        const r = await api.get<{ data: PostcodeSearchResult[] }>(
          `/locations/postcodes/search?q=${encodeURIComponent(item.postal)}`,
        );
        const hit = r.data.data.find((row) => row.code === item.postal);
        if (hit) {
          setValue('postalCode', hit.code, { shouldValidate: true });
          setValue('city', hit.cityName, { shouldValidate: true });
          setValue('state', hit.stateName, { shouldValidate: true });
          setPostcodeQuery(hit.code);
          canonicalized = true;
        }
      } catch {
        // Network blip — fall through to raw Photon values.
      }
    }
    if (!canonicalized) {
      if (item.postal) {
        setValue('postalCode', item.postal, { shouldValidate: true });
        setPostcodeQuery(item.postal);
      }
      if (item.city) setValue('city', item.city, { shouldValidate: true });
      if (item.region) setValue('state', item.region, { shouldValidate: true });
    }
    setShowPostcodeSuggestions(false);
    showToast({ message: t('onboarding.addressFilled'), tone: 'success' });
  }

  // GPS auto-fill. Asks for foreground location permission, reverse-
  // geocodes the device's current position, and best-effort fills every
  // address field so the chef rarely has to type any of them. Manual
  // entry stays available regardless — this is purely additive.
  async function handleUseCurrentLocation(): Promise<void> {
    setLocating(true);
    try {
      const perm = await Location.requestForegroundPermissionsAsync();
      if (perm.status !== 'granted') {
        showToast({
          message: t('onboarding.locationDenied'),
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
          message: t('onboarding.locationReadFailed'),
          tone: 'error',
        });
        return;
      }
      // Block non-India locations: the address form, state list, and PIN
      // validation are all India-only.
      if (top.isoCountryCode && top.isoCountryCode !== 'IN') {
        showToast({
          message: t('onboarding.indiaOnly', {
            country: top.country ?? top.isoCountryCode,
          }),
          tone: 'error',
        });
        return;
      }

      const line1 = [top.name, top.street].filter(Boolean).join(', ');
      if (line1) setValue('addressLine1', line1, { shouldValidate: true });

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
          // Network hiccup — fall through.
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
        message: t('onboarding.addressFilledLocation'),
        tone: 'success',
      });
    } catch (_err) {
      showToast({
        message: t('onboarding.locationLookupFailed'),
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
    const firstKey = Object.keys(errs)[0];
    const y = firstKey && IDENTITY_FIELDS.has(firstKey) ? identityFieldY.current : addressFieldY.current;
    scrollRef.current?.scrollTo({ y: Math.max(0, y - 16), animated: true });
    const firstError = Object.values(errs)[0];
    if (firstError?.message) Alert.alert(t('onboarding.checkDetails'), t(firstError.message));
  }

  return (
    <OnboardingScaffold
      step={2}
      total={6}
      stepName={t('onboarding.stepKitchen')}
      title={t('onboarding.kitchenTitle')}
      subtitle={t('onboarding.kitchenSubtitle')}
      primaryLabel={t('onboarding.continue')}
      onPrimary={handleSubmit(onSubmit, onInvalid)}
      scrollRef={scrollRef}
    >
      {/* ── IDENTITY ──────────────────────────────────────────── */}
      <View style={styles.sectionLabel}>
        <UtensilsCrossed size={12} color={theme.colors.ink.muted} strokeWidth={2} />
        <Text style={styles.sectionLabelText}>{t('onboarding.kitchenIdentity')}</Text>
      </View>

      <View
        style={styles.fieldCard}
        onLayout={(e) => {
          identityFieldY.current = e.nativeEvent.layout.y;
        }}
      >
        {/* Business name */}
        <Controller
          control={control}
          name="businessName"
          render={({ field: { onChange, onBlur, value } }) => (
            <Input
              label={t('onboarding.businessName')}
              placeholder={t('onboarding.businessNamePlaceholder')}
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              autoCapitalize="words"
              error={errors.businessName?.message ? t(errors.businessName.message) : undefined}
            />
          )}
        />

        <View style={styles.innerHairline} />

        {/* Cuisine chips — outlined pill pattern, ink border active */}
        <View>
          <Text style={styles.fieldLabel}>{t('onboarding.cuisineTypes')}</Text>
          <View style={styles.chipRow}>
            {CUISINE_OPTIONS.map((cuisine) => {
              const selected = selectedCuisines?.includes(cuisine) ?? false;
              return (
                <Pressable
                  key={cuisine}
                  onPress={() => toggleCuisine(cuisine)}
                  accessibilityRole="button"
                  accessibilityLabel={cuisine}
                  accessibilityState={{ selected }}
                  android_ripple={{
                    color: selected
                      ? `${theme.colors.paper}30`
                      : `${theme.colors.ink.DEFAULT}14`,
                    borderless: false,
                  }}
                >
                  {({ pressed }) => (
                    <View
                      style={[
                        styles.chip,
                        selected && styles.chipActive,
                        pressed && Platform.OS === 'ios' && { opacity: 0.75 },
                      ]}
                    >
                      <Text
                        style={[styles.chipLabel, selected && styles.chipLabelActive]}
                      >
                        {cuisine}
                      </Text>
                    </View>
                  )}
                </Pressable>
              );
            })}
          </View>
          {errors.cuisines?.message ? (
            <Text style={styles.fieldError}>{t(errors.cuisines.message)}</Text>
          ) : null}
        </View>

        <View style={styles.innerHairline} />

        {/* Description */}
        <View>
          <Controller
            control={control}
            name="description"
            render={({ field: { onChange, onBlur, value } }) => (
              <Input
                label={t('onboarding.aboutKitchen')}
                placeholder={t('onboarding.aboutKitchenPlaceholder')}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                multiline
                numberOfLines={4}
                inputStyle={styles.textArea}
                error={errors.description?.message ? t(errors.description.message) : undefined}
              />
            )}
          />
          <Text style={styles.charCount}>{(descriptionValue ?? '').length} / 500</Text>
        </View>
      </View>

      {/* ── ADDRESS ───────────────────────────────────────────── */}
      <View style={styles.hairline} />

      <View style={styles.sectionLabel}>
        <MapPin size={12} color={theme.colors.ink.muted} strokeWidth={2} />
        <Text style={styles.sectionLabelText}>{t('onboarding.kitchenAddress')}</Text>
      </View>

      {/* GPS fast-path — ink-filled primary affordance */}
      <Pressable
        onPress={handleUseCurrentLocation}
        disabled={locating}
        accessibilityRole="button"
        accessibilityLabel={t('onboarding.useCurrentLocation')}
        android_ripple={
          locating ? undefined : { color: `${theme.colors.paper}30`, borderless: false }
        }
      >
        {({ pressed }) => (
          <View
            style={[
              styles.locateCta,
              pressed && Platform.OS === 'ios' && { opacity: 0.85 },
              locating && { opacity: 0.7 },
            ]}
          >
            {locating ? (
              <ActivityIndicator size="small" color={theme.colors.paper} />
            ) : (
              <Navigation size={16} color={theme.colors.paper} strokeWidth={2.2} />
            )}
            <Text style={styles.locateCtaLabel}>
              {locating ? t('onboarding.findingAddress') : t('onboarding.useCurrentLocation')}
            </Text>
          </View>
        )}
      </Pressable>

      {/* Search field */}
      <View style={styles.searchWrap}>
        <Input
          label=""
          placeholder={t('onboarding.searchAddressPlaceholder')}
          value={postcodeQuery}
          onChangeText={(text) => {
            setPostcodeQuery(text);
            setShowPostcodeSuggestions(true);
          }}
          onFocus={() => setShowPostcodeSuggestions(true)}
          onBlur={() => {
            setTimeout(() => setShowPostcodeSuggestions(false), 150);
          }}
          autoCapitalize="words"
          maxLength={60}
        />
        {showPostcodeSuggestions && postcodeQuery.trim().length >= 2 ? (
          <View style={styles.suggestionsPanel}>
            {postcodeSearch.isLoading || addressAutocomplete.isLoading ? (
              <View style={styles.suggestionLoader}>
                <ActivityIndicator size="small" color={theme.colors.ink.muted} />
              </View>
            ) : (
              <>
                {(postcodeSearch.data ?? []).map((item) => (
                  <Pressable
                    key={`pin-${item.code}`}
                    onPress={() => pickPostcodeSuggestion(item)}
                    accessibilityRole="button"
                    accessibilityLabel={`${item.code}, ${item.areaName}, ${item.cityName}, ${item.stateName}`}
                    android_ripple={{ color: `${theme.colors.ink.DEFAULT}0F`, borderless: false }}
                  >
                    {({ pressed }) => (
                      <View
                        style={[
                          styles.suggestionRow,
                          pressed && Platform.OS === 'ios' && { backgroundColor: theme.colors.bone },
                        ]}
                      >
                        <Text style={styles.suggestionCode}>{item.code}</Text>
                        <Text style={styles.suggestionMeta} numberOfLines={1}>
                          {item.areaName} · {item.cityName}, {item.stateName}
                        </Text>
                      </View>
                    )}
                  </Pressable>
                ))}

                {(addressAutocomplete.data ?? []).map((item, idx) => (
                  <Pressable
                    key={`osm-${idx}-${item.description}`}
                    onPress={() => { void pickAddressSuggestion(item); }}
                    accessibilityRole="button"
                    accessibilityLabel={item.description}
                    android_ripple={{ color: `${theme.colors.ink.DEFAULT}0F`, borderless: false }}
                  >
                    {({ pressed }) => (
                      <View
                        style={[
                          styles.suggestionRow,
                          pressed && Platform.OS === 'ios' && { backgroundColor: theme.colors.bone },
                        ]}
                      >
                        <MapPin size={16} color={theme.colors.ink.DEFAULT} strokeWidth={2.2} />
                        <Text style={styles.suggestionMeta} numberOfLines={2}>
                          {item.description}
                        </Text>
                      </View>
                    )}
                  </Pressable>
                ))}

                {(postcodeSearch.data?.length ?? 0) === 0 &&
                (addressAutocomplete.data?.length ?? 0) === 0 ? (
                  <Text style={styles.suggestionEmpty}>
                    {postcodeQuery.trim().length < 3
                      ? t('onboarding.typeMinChars')
                      : t('onboarding.noMatch')}
                  </Text>
                ) : null}
              </>
            )}
          </View>
        ) : null}
      </View>

      {/* Manual address fields grouped in a card */}
      <View
        style={styles.fieldCard}
        onLayout={(e) => {
          addressFieldY.current = e.nativeEvent.layout.y;
        }}
      >
        <Controller
          control={control}
          name="addressLine1"
          render={({ field: { onChange, onBlur, value } }) => (
            <Input
              label={t('onboarding.addressLine1')}
              placeholder={t('onboarding.addressLine1Placeholder')}
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              autoCapitalize="words"
              error={errors.addressLine1?.message ? t(errors.addressLine1.message) : undefined}
            />
          )}
        />

        <View style={styles.innerHairline} />

        <Controller
          control={control}
          name="addressLine2"
          render={({ field: { onChange, onBlur, value } }) => (
            <Input
              label={t('onboarding.addressLine2')}
              placeholder={t('onboarding.addressLine2Placeholder')}
              onBlur={onBlur}
              onChangeText={onChange}
              value={value ?? ''}
              autoCapitalize="words"
            />
          )}
        />

        <View style={styles.innerHairline} />

        {/* State — Input + quick-pick chip strip */}
        <View>
          <Controller
            control={control}
            name="state"
            render={({ field: { onChange, onBlur, value } }) => (
              <Input
                label={t('onboarding.state')}
                placeholder={t('onboarding.statePlaceholder')}
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                autoCapitalize="words"
                error={errors.state?.message ? t(errors.state.message) : undefined}
              />
            )}
          />
          {states.isLoading ? null : (
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
                    accessibilityLabel={s.name}
                    accessibilityState={{ selected }}
                    android_ripple={{
                      color: selected
                        ? `${theme.colors.paper}30`
                        : `${theme.colors.ink.DEFAULT}14`,
                      borderless: false,
                    }}
                  >
                    {({ pressed }) => (
                      <View
                        style={[
                          styles.pickerChip,
                          selected && styles.pickerChipActive,
                          pressed && Platform.OS === 'ios' && { opacity: 0.7 },
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
        </View>

        <View style={styles.innerHairline} />

        {/* City — same pattern as State */}
        <View>
          <Controller
            control={control}
            name="city"
            render={({ field: { onChange, onBlur, value } }) => (
              <Input
                label={t('onboarding.city')}
                placeholder={
                  selectedStateCode
                    ? t('onboarding.cityPlaceholderState')
                    : t('onboarding.cityPlaceholderNoState')
                }
                onBlur={onBlur}
                onChangeText={onChange}
                value={value}
                autoCapitalize="words"
                error={errors.city?.message ? t(errors.city.message) : undefined}
              />
            )}
          />
          {selectedStateCode && !cities.isLoading && (cities.data?.length ?? 0) > 0 ? (
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
                    accessibilityRole="button"
                    accessibilityLabel={c.name}
                    accessibilityState={{ selected }}
                    android_ripple={{
                      color: selected
                        ? `${theme.colors.paper}30`
                        : `${theme.colors.ink.DEFAULT}14`,
                      borderless: false,
                    }}
                  >
                    {({ pressed }) => (
                      <View
                        style={[
                          styles.pickerChip,
                          selected && styles.pickerChipActive,
                          pressed && Platform.OS === 'ios' && { opacity: 0.7 },
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
          ) : null}
        </View>

        <View style={styles.innerHairline} />

        {/* PIN code */}
        <Controller
          control={control}
          name="postalCode"
          render={({ field: { onChange, onBlur, value } }) => (
            <Input
              label={t('onboarding.pinCode')}
              placeholder={t('onboarding.pinPlaceholder')}
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              keyboardType="number-pad"
              maxLength={6}
              error={errors.postalCode?.message ? t(errors.postalCode.message) : undefined}
            />
          )}
        />
      </View>

      <View style={styles.bottomSpacer} />
    </OnboardingScaffold>
  );
}

const styles = StyleSheet.create({
  // Section label row: small icon + caps text
  sectionLabel: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
    marginBottom: theme.spacing[2],
    marginTop: theme.spacing[1],
  },
  sectionLabelText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    letterSpacing: 1.2,
    color: theme.colors.ink.muted,
  },

  // Hairline separator between sections
  hairline: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginVertical: theme.spacing[4],
  },

  // Inner hairline between fields inside a card
  innerHairline: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginVertical: theme.spacing[3],
  },

  // Grouped card — hairline border, no shadow
  fieldCard: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    padding: theme.spacing[4],
    backgroundColor: theme.colors.paper,
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

  // Cuisine chips
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

  // GPS CTA — ink-filled primary affordance
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
    marginBottom: theme.spacing[2],
  },
  locateCtaLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
    letterSpacing: 0.2,
  },

  searchWrap: {
    marginBottom: theme.spacing[3],
  },

  // Picker strip — horizontally scrollable chip row for State + City
  pickerStrip: {
    gap: theme.spacing[2],
    paddingRight: theme.spacing[4],
    paddingBottom: theme.spacing[1],
    paddingTop: theme.spacing[2],
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

  // Address suggestions panel
  suggestionsPanel: {
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.paper,
    overflow: 'hidden',
    marginTop: theme.spacing[1],
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
    height: theme.spacing[4],
  },
});
