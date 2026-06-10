import { useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { Image } from 'expo-image';
import * as ImagePicker from 'expo-image-picker';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Camera, ChevronLeft, ImagePlus, Plus } from 'lucide-react-native';
import { multipartConfig, getServerErrorMessage } from '@homechef/mobile-shared/api';
import { theme } from '@homechef/mobile-shared/theme';
import { useToast } from '@homechef/mobile-shared/ui';
import { api } from '../lib/api';
import { useStates } from '../hooks/useLocations';

// ---- Data types -----------------------------------------------------------
// Matches the backend GET /chef/profile response.

interface ChefProfile {
  id: string;
  businessName: string;
  description: string;
  profileImage?: string;
  bannerImage?: string;
  cuisines: string[];
  specialties: string[];
  prepTime: string;
  minimumOrder: number;
  serviceRadius: number;
  acceptingOrders: boolean;
  kitchenPhotos: string[];
  addressLine1: string;
  addressLine2: string;
  city: string;
  state: string;
  postalCode: string;
  verified: boolean;
}

interface UpdateChefProfilePayload {
  businessName?: string;
  description?: string;
  cuisines?: string[];
  prepTime?: string;
  minimumOrder?: number;
  serviceRadius?: number;
  addressLine1?: string;
  addressLine2?: string;
  city?: string;
  state?: string;
  postalCode?: string;
}

// Preset lists — chip selectors instead of free-text input wherever the
// space is small enough that a chef shouldn't have to type. Keeps profile
// fast on a phone, eliminates spelling drift across kitchens.
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

const PREP_TIME_OPTIONS = ['15 min', '20 min', '30 min', '45 min', '60 min'] as const;

// Indian state list is now driven by the /api/v1/locations/countries/IN/states
// endpoint (see hooks/useLocations.ts) so we only maintain it in one place
// (the backend seeder). The hook is invoked inside ProfileScreen below.

function useChefProfile() {
  return useQuery<ChefProfile>({
    queryKey: ['chef', 'profile'],
    queryFn: () => api.get<ChefProfile>('/chef/profile').then((r) => r.data),
    staleTime: 60_000,
  });
}

function useUpdateProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: UpdateChefProfilePayload) =>
      api.put('/chef/profile', payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef', 'profile'] });
    },
  });
}

function useUploadProfileImage() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (uri: string) => {
      const formData = new FormData();
      const filename = uri.split('/').pop() ?? 'profile.jpg';
      const type = filename.endsWith('.png') ? 'image/png' : 'image/jpeg';
      formData.append('file', { uri, name: filename, type } as unknown as Blob);
      return api.post('/chef/profile-image', formData, multipartConfig());
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef', 'profile'] });
    },
  });
}

function useUploadBannerImage() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (uri: string) => {
      const formData = new FormData();
      const filename = uri.split('/').pop() ?? 'cover.jpg';
      const type = filename.endsWith('.png') ? 'image/png' : 'image/jpeg';
      formData.append('file', { uri, name: filename, type } as unknown as Blob);
      return api.post('/chef/banner-image', formData, multipartConfig());
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef', 'profile'] });
    },
  });
}

function useUploadKitchenPhoto() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (uri: string) => {
      const formData = new FormData();
      const filename = uri.split('/').pop() ?? 'kitchen.jpg';
      const type = filename.endsWith('.png') ? 'image/png' : 'image/jpeg';
      formData.append('file', { uri, name: filename, type } as unknown as Blob);
      return api.post('/chef/kitchen-photos', formData, multipartConfig());
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef', 'profile'] });
    },
  });
}

// ---- Helpers ------------------------------------------------------------

function deriveInitials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return 'C';
  if (parts.length === 1) return parts[0]!.slice(0, 2).toUpperCase();
  return (
    (parts[0]![0] ?? '') + (parts[parts.length - 1]![0] ?? '')
  ).toUpperCase();
}

function parseNumber(input: string): number {
  const n = Number(input.replace(/[^\d.]/g, ''));
  return Number.isFinite(n) && n >= 0 ? n : 0;
}

interface ChipProps {
  label: string;
  selected: boolean;
  onPress: () => void;
}

// Outlined-pill chip — ink border + ink fill when selected, mist hairline
// border + ink-soft label when unselected. Matches the onboarding
// kitchen-details cuisine row.
function Chip({ label, selected, onPress }: ChipProps) {
  return (
    <Pressable
      onPress={onPress}
      hitSlop={4}
      accessibilityRole="button"
      accessibilityState={{ selected }}
      accessibilityLabel={label}
    >
      {({ pressed }) => (
        <View
          style={[
            chipStyles.root,
            selected && chipStyles.rootSelected,
            pressed && { opacity: 0.7 },
          ]}
        >
          <Text
            style={[
              chipStyles.label,
              selected && chipStyles.labelSelected,
            ]}
          >
            {label}
          </Text>
        </View>
      )}
    </Pressable>
  );
}

const chipStyles = StyleSheet.create({
  root: {
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    borderRadius: theme.radius.full,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    backgroundColor: theme.colors.paper,
    minHeight: 36,
    justifyContent: 'center',
  },
  rootSelected: {
    borderColor: theme.colors.ink.DEFAULT,
    backgroundColor: theme.colors.ink.DEFAULT,
  },
  label: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    letterSpacing: 0.1,
  },
  labelSelected: {
    color: theme.colors.paper,
  },
});

// ---- Sub-components -----------------------------------------------------

interface EditableFieldProps {
  label: string;
  value: string;
  onChangeText: (t: string) => void;
  multiline?: boolean;
  keyboardType?: 'default' | 'phone-pad' | 'decimal-pad';
  placeholder?: string;
  hasBorderBottom?: boolean;
}

function EditableField({
  label,
  value,
  onChangeText,
  multiline = false,
  keyboardType = 'default',
  placeholder,
  hasBorderBottom = true,
}: EditableFieldProps) {
  return (
    <>
      <View style={styles.editRow}>
        <Text style={styles.editLabel}>{label}</Text>
        <TextInput
          value={value}
          onChangeText={onChangeText}
          multiline={multiline}
          numberOfLines={multiline ? 4 : 1}
          textAlignVertical={multiline ? 'top' : 'center'}
          keyboardType={keyboardType}
          placeholder={placeholder}
          placeholderTextColor={theme.colors.ink.muted}
          style={[styles.editInput, multiline && styles.editInputMultiline]}
        />
      </View>
      {/* Inset hairline — skipped on the last row of a group card */}
      {hasBorderBottom ? <View style={styles.separator} /> : null}
    </>
  );
}

// ---- Screen -------------------------------------------------------------

export default function ProfileScreen() {
  const { data, isLoading, isError, refetch, isRefetching } = useChefProfile();
  const updateMutation = useUpdateProfile();
  const uploadProfileImageMutation = useUploadProfileImage();
  const uploadBannerImageMutation = useUploadBannerImage();
  const uploadKitchenPhotoMutation = useUploadKitchenPhoto();
  const { show: showToast } = useToast();
  // Indian states from the reference-data API. Falls back to an empty
  // array until the request resolves — the chip strip just renders zero
  // chips during that brief window, no layout pop.
  const statesQuery = useStates();
  const indianStates = statesQuery.data ?? [];

  // Once a save succeeds, flip this flag so handleBack doesn't re-prompt
  // before the query invalidation refreshes `data`. Cleared in the data
  // useEffect when fresh server values arrive.
  const savedRef = useRef(false);

  // Profile is always-editable inline (no separate Edit mode) — closer to
  // iOS Settings + Notes than to a CMS dashboard. The save button stays
  // visible at the bottom and back prompts if dirty.
  const [businessName, setBusinessName] = useState('');
  const [description, setDescription] = useState('');
  const [cuisines, setCuisines] = useState<string[]>([]);
  const [prepTime, setPrepTime] = useState('');
  const [minimumOrder, setMinimumOrder] = useState('');
  const [serviceRadius, setServiceRadius] = useState('');
  const [addressLine1, setAddressLine1] = useState('');
  const [addressLine2, setAddressLine2] = useState('');
  const [city, setCity] = useState('');
  const [stateName, setStateName] = useState('');
  const [postalCode, setPostalCode] = useState('');

  // Dirty against last-known server values — drives the disabled state of
  // the always-visible save button and the back-discard prompt.
  const isDirty =
    !savedRef.current &&
    data !== undefined &&
    (businessName.trim() !== (data.businessName ?? '') ||
      description.trim() !== (data.description ?? '') ||
      JSON.stringify([...cuisines].sort()) !==
        JSON.stringify([...(data.cuisines ?? [])].sort()) ||
      prepTime.trim() !== (data.prepTime ?? '') ||
      parseNumber(minimumOrder) !== (data.minimumOrder ?? 0) ||
      parseNumber(serviceRadius) !== (data.serviceRadius ?? 0) ||
      addressLine1.trim() !== (data.addressLine1 ?? '') ||
      addressLine2.trim() !== (data.addressLine2 ?? '') ||
      city.trim() !== (data.city ?? '') ||
      stateName.trim() !== (data.state ?? '') ||
      postalCode.trim() !== (data.postalCode ?? ''));

  // Sync local form state when data loads (including after a successful save
  // which invalidates the query and re-fetches). Clear savedRef so that
  // subsequent edits are tracked correctly against the fresh server values.
  useEffect(() => {
    if (data) {
      setBusinessName(data.businessName ?? '');
      setDescription(data.description ?? '');
      setCuisines(data.cuisines ?? []);
      setPrepTime(data.prepTime ?? '');
      setMinimumOrder(
        data.minimumOrder ? String(data.minimumOrder) : '',
      );
      setServiceRadius(
        data.serviceRadius ? String(data.serviceRadius) : '',
      );
      setAddressLine1(data.addressLine1 ?? '');
      setAddressLine2(data.addressLine2 ?? '');
      setCity(data.city ?? '');
      setStateName(data.state ?? '');
      setPostalCode(data.postalCode ?? '');
      savedRef.current = false;
    }
  }, [data]);

  function toggleCuisine(name: string) {
    setCuisines((prev) =>
      prev.includes(name) ? prev.filter((c) => c !== name) : [...prev, name],
    );
  }

  function handleSave() {
    if (!businessName.trim()) {
      Alert.alert(
        'Business name required',
        'Enter the name customers will see on the storefront.',
      );
      return;
    }
    const payload: UpdateChefProfilePayload = {
      businessName: businessName.trim(),
      description: description.trim(),
      cuisines,
      prepTime: prepTime.trim(),
      minimumOrder: parseNumber(minimumOrder),
      serviceRadius: parseNumber(serviceRadius),
      addressLine1: addressLine1.trim(),
      addressLine2: addressLine2.trim(),
      city: city.trim(),
      state: stateName.trim(),
      postalCode: postalCode.trim(),
    };
    updateMutation.mutate(payload, {
      onSuccess: () => {
        // Mark clean immediately so isDirty evaluates false. The query
        // invalidation will re-fetch and reset savedRef via the data
        // useEffect, but that happens asynchronously — the toast and any
        // subsequent back press must not see stale dirty state.
        savedRef.current = true;
        showToast({ message: 'Profile saved', tone: 'success' });
      },
      onError: (err) =>
        Alert.alert('Save failed', getServerErrorMessage(err, 'Please try again.')),
    });
  }

  // Back press: if there are unsaved edits, give the chef a choice. The
  // "Save" option saves AND navigates back — no modal stacking. Discard
  // rolls form state to the last server values. Stay-on-page = cancel.
  function handleBack() {
    if (!isDirty) {
      router.back();
      return;
    }
    Alert.alert(
      'Save changes?',
      'You have unsaved profile edits. Save them before going back?',
      [
        { text: 'Keep editing', style: 'cancel' },
        {
          text: 'Discard',
          style: 'destructive',
          onPress: () => {
            router.back();
          },
        },
        {
          text: 'Save',
          onPress: () => {
            if (!businessName.trim()) {
              Alert.alert(
                'Business name required',
                'Enter the name customers will see on the storefront.',
              );
              return;
            }
            const payload: UpdateChefProfilePayload = {
              businessName: businessName.trim(),
              description: description.trim(),
              cuisines,
              prepTime: prepTime.trim(),
              minimumOrder: parseNumber(minimumOrder),
              serviceRadius: parseNumber(serviceRadius),
              addressLine1: addressLine1.trim(),
              addressLine2: addressLine2.trim(),
              city: city.trim(),
              state: stateName.trim(),
              postalCode: postalCode.trim(),
            };
            updateMutation.mutate(payload, {
              onSuccess: () => {
                savedRef.current = true;
                showToast({ message: 'Profile saved', tone: 'success' });
                router.back();
              },
              onError: (err) =>
                Alert.alert('Save failed', getServerErrorMessage(err, 'Please try again.')),
            });
          },
        },
      ],
    );
  }

  async function handlePickProfileImage() {
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      allowsEditing: true,
      aspect: [1, 1],
      quality: 0.85,
    });
    if (!result.canceled && result.assets[0]) {
      uploadProfileImageMutation.mutate(result.assets[0].uri, {
        onError: (err) =>
          Alert.alert('Upload failed', getServerErrorMessage(err, 'Failed to upload photo.')),
      });
    }
  }

  async function handlePickBannerImage() {
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      allowsEditing: true,
      aspect: [16, 9],
      quality: 0.85,
    });
    if (!result.canceled && result.assets[0]) {
      uploadBannerImageMutation.mutate(result.assets[0].uri, {
        onError: (err) =>
          Alert.alert('Upload failed', getServerErrorMessage(err, 'Failed to upload cover.')),
        onSuccess: () => showToast({ message: 'Cover photo updated.', tone: 'success' }),
      });
    }
  }

  async function handleAddKitchenPhoto() {
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      allowsEditing: true,
      aspect: [4, 3],
      quality: 0.85,
    });
    if (!result.canceled && result.assets[0]) {
      uploadKitchenPhotoMutation.mutate(result.assets[0].uri, {
        onError: (err) =>
          Alert.alert('Upload failed', getServerErrorMessage(err, 'Failed to upload photo.')),
        onSuccess: () => showToast({ message: 'Kitchen photo added.', tone: 'success' }),
      });
    }
  }

  // ---- Loading & error states -------------------------------------------

  if (isLoading) {
    return (
      <SafeAreaView style={styles.centeredFill} edges={['top', 'left', 'right']}>
        <ActivityIndicator size="large" color={theme.colors.ink.DEFAULT} />
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView style={styles.centeredFill} edges={['top', 'left', 'right']}>
        <Text style={styles.errorBody}>Failed to load profile</Text>
        <Pressable
          onPress={() => refetch()}
          style={({ pressed }) => [styles.errorBtn, pressed && { opacity: 0.85 }]}
        >
          <Text style={styles.errorBtnLabel}>Retry</Text>
        </Pressable>
      </SafeAreaView>
    );
  }

  const initials = deriveInitials(data?.businessName ?? 'Chef');

  // ---- Render ------------------------------------------------------------

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        keyboardVerticalOffset={0}
      >
        {/* Command bar: back left, title center, no right action (the
            sticky Save footer below replaces the Edit/Cancel toggle).
            Back uses handleBack so dirty edits prompt before navigation. */}
        <View style={styles.commandBar}>
          <Pressable
            onPress={handleBack}
            hitSlop={8}
            accessibilityRole="button"
            accessibilityLabel="Go back"
          >
            {({ pressed }) => (
              <View style={[styles.backBtn, pressed && { opacity: 0.6 }]}>
                <ChevronLeft size={22} color={theme.colors.ink.DEFAULT} strokeWidth={2} />
              </View>
            )}
          </Pressable>
          <Text style={styles.commandTitle}>Profile</Text>
          <View style={styles.commandSpacer} />
        </View>

        <ScrollView
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
          refreshControl={
            <RefreshControl
              refreshing={isRefetching}
              onRefresh={refetch}
              tintColor={theme.colors.ink.DEFAULT}
            />
          }
          showsVerticalScrollIndicator={false}
          keyboardShouldPersistTaps="handled"
        >
          {/* Cover photo — the hero customers see on your listing */}
          <Pressable
            onPress={handlePickBannerImage}
            disabled={uploadBannerImageMutation.isPending}
            accessibilityRole="button"
            accessibilityLabel="Change cover photo"
          >
            {({ pressed }) => (
              // Layout lives on this inner View — a Pressable with a
              // function `style` returning an array drops height/bg/border
              // on iOS, which collapsed (or blew out) the cover box.
              <View style={[styles.coverWrapper, pressed && { opacity: 0.9 }]}>
                {data?.bannerImage ? (
                  <Image
                    source={{ uri: data.bannerImage }}
                    style={styles.coverImage}
                    contentFit="cover"
                  />
                ) : (
                  <View style={styles.coverPlaceholder}>
                    <ImagePlus
                      size={22}
                      color={theme.colors.ink.muted}
                      strokeWidth={1.75}
                    />
                    <Text style={styles.coverPlaceholderText}>Add a cover photo</Text>
                    <Text style={styles.coverPlaceholderHint}>
                      Shown to customers on your listing
                    </Text>
                  </View>
                )}
                <View style={styles.coverBadge}>
                  {uploadBannerImageMutation.isPending ? (
                    <ActivityIndicator
                      size="small"
                      color={theme.colors.ink.DEFAULT}
                    />
                  ) : (
                    <Camera size={14} color={theme.colors.ink.DEFAULT} strokeWidth={2} />
                  )}
                </View>
              </View>
            )}
          </Pressable>

          {/* Identity block */}
          <View style={styles.identityBlock}>
            <Pressable
              onPress={handlePickProfileImage}
              disabled={uploadProfileImageMutation.isPending}
              style={({ pressed }) => [
                styles.avatarWrapper,
                pressed && { opacity: 0.85 },
              ]}
              accessibilityRole="button"
              accessibilityLabel="Change profile photo"
            >
              {data?.profileImage ? (
                <Image
                  source={{ uri: data.profileImage }}
                  style={styles.avatarImage}
                  contentFit="cover"
                />
              ) : (
                <View style={styles.avatarFallback}>
                  <Text style={styles.avatarInitials}>{initials}</Text>
                </View>
              )}
              <View style={styles.avatarBadge}>
                {uploadProfileImageMutation.isPending ? (
                  <ActivityIndicator size="small" color={theme.colors.ink.DEFAULT} />
                ) : (
                  <Camera size={14} color={theme.colors.ink.DEFAULT} strokeWidth={2} />
                )}
              </View>
            </Pressable>

            <View style={styles.identityText}>
              <Text style={styles.identityName} numberOfLines={1}>
                {data?.businessName || 'Your kitchen'}
              </Text>
              <Text style={styles.identityCaption} numberOfLines={1}>
                {data?.verified ? 'Verified chef' : 'Pending verification'}
              </Text>
            </View>
          </View>

          {/* BUSINESS section — name + description, what customers see */}
          <Text style={styles.sectionLabel}>BUSINESS</Text>
          <View style={styles.hairlineGroup}>
            <EditableField
              label="Business name"
              value={businessName}
              onChangeText={setBusinessName}
              placeholder="The name customers see"
            />
            <EditableField
              label="Description"
              value={description}
              onChangeText={setDescription}
              multiline
              placeholder="One or two sentences about your kitchen"
              hasBorderBottom={false}
            />
          </View>

          {/* KITCHEN section — cuisines + prep time as preset pills,
              minimum order + service radius as small numeric inputs. Less
              typing, less spelling drift. */}
          <Text style={styles.sectionLabel}>KITCHEN</Text>
          <View style={styles.hairlineGroup}>
            <View style={styles.chipFieldRow}>
              <Text style={styles.chipFieldLabel}>Cuisines</Text>
              <View style={styles.chipWrap}>
                {CUISINE_OPTIONS.map((cuisine) => (
                  <Chip
                    key={cuisine}
                    label={cuisine}
                    selected={cuisines.includes(cuisine)}
                    onPress={() => toggleCuisine(cuisine)}
                  />
                ))}
              </View>
            </View>
            <View style={styles.chipFieldRow}>
              <Text style={styles.chipFieldLabel}>Prep time</Text>
              <View style={styles.chipWrap}>
                {PREP_TIME_OPTIONS.map((opt) => (
                  <Chip
                    key={opt}
                    label={opt}
                    selected={prepTime === opt}
                    onPress={() => setPrepTime(opt)}
                  />
                ))}
              </View>
            </View>
            <EditableField
              label="Minimum order (₹)"
              value={minimumOrder}
              onChangeText={setMinimumOrder}
              keyboardType="decimal-pad"
              placeholder="0"
            />
            <EditableField
              label="Service radius (km)"
              value={serviceRadius}
              onChangeText={setServiceRadius}
              keyboardType="decimal-pad"
              placeholder="10"
              hasBorderBottom={false}
            />
          </View>

          {/* ADDRESS section — editable post-onboarding now that the
              backend's UpdateChefProfileRequest accepts the address
              pointer fields. State is a horizontal scrollable chip strip
              to avoid free-text spelling drift. */}
          <Text style={styles.sectionLabel}>ADDRESS</Text>
          <View style={styles.hairlineGroup}>
            <EditableField
              label="Address line 1"
              value={addressLine1}
              onChangeText={setAddressLine1}
              placeholder="Street, building, area"
            />
            <EditableField
              label="Address line 2 (optional)"
              value={addressLine2}
              onChangeText={setAddressLine2}
              placeholder="Landmark, floor, apartment"
            />
            <EditableField
              label="City"
              value={city}
              onChangeText={setCity}
              placeholder="e.g. Bengaluru"
            />
            <View style={styles.chipFieldRow}>
              <Text style={styles.chipFieldLabel}>State</Text>
              <ScrollView
                horizontal
                showsHorizontalScrollIndicator={false}
                contentContainerStyle={styles.chipStateRow}
              >
                {indianStates.map((s) => (
                  <Chip
                    key={s.id}
                    label={s.name}
                    selected={stateName === s.name}
                    onPress={() => setStateName(s.name)}
                  />
                ))}
              </ScrollView>
            </View>
            <EditableField
              label="Postal code"
              value={postalCode}
              onChangeText={setPostalCode}
              keyboardType="decimal-pad"
              placeholder="6-digit PIN"
              hasBorderBottom={false}
            />
          </View>

          {/* KITCHEN PHOTOS — horizontal strip inside its own white card */}
          <Text style={styles.sectionLabel}>KITCHEN PHOTOS</Text>
          <View style={styles.photosCard}>
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerStyle={styles.photoStrip}
            >
            {(data?.kitchenPhotos ?? []).slice(0, 5).map((url, idx) => (
              <Image
                key={`kp-${idx}-${url}`}
                source={{ uri: url }}
                style={styles.photoThumb}
                contentFit="cover"
              />
            ))}
            {(data?.kitchenPhotos?.length ?? 0) < 5 && (
              <Pressable
                onPress={handleAddKitchenPhoto}
                disabled={uploadKitchenPhotoMutation.isPending}
                style={({ pressed }) => [
                  styles.photoAddSlot,
                  pressed && { opacity: 0.7 },
                ]}
                accessibilityRole="button"
                accessibilityLabel="Add kitchen photo"
              >
                {uploadKitchenPhotoMutation.isPending ? (
                  <ActivityIndicator size="small" color={theme.colors.ink.soft} />
                ) : (
                  <Plus size={22} color={theme.colors.ink.muted} strokeWidth={1.75} />
                )}
              </Pressable>
            )}
            </ScrollView>
          </View>
        </ScrollView>

        {/* Always-visible save footer. Disabled state when nothing
            changed yet — the chef can see the button is there waiting,
            no hunting for an Edit toggle. */}
        <View style={styles.stickyFooter}>
          <SafeAreaView edges={['bottom']} style={styles.stickyFooterInner}>
            <Pressable
              onPress={handleSave}
              disabled={updateMutation.isPending || !isDirty}
              style={({ pressed }) => [
                styles.saveBtn,
                !isDirty && styles.saveBtnDisabled,
                (pressed || updateMutation.isPending) && { opacity: 0.85 },
              ]}
              accessibilityRole="button"
              accessibilityState={{ disabled: !isDirty }}
            >
              {updateMutation.isPending ? (
                <ActivityIndicator color={theme.colors.paper} />
              ) : (
                <Text
                  style={[
                    styles.saveBtnLabel,
                    !isDirty && styles.saveBtnLabelDisabled,
                  ]}
                >
                  {isDirty ? 'Save changes' : 'No changes to save'}
                </Text>
              )}
            </Pressable>
          </SafeAreaView>
        </View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

// ---- Styles -------------------------------------------------------------

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  centeredFill: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
    backgroundColor: theme.colors.bone,
  },

  // Command bar
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
  },
  backBtn: {
    minWidth: 48,
    minHeight: 44,
    justifyContent: 'center',
  },
  commandTitle: {
    flex: 1,
    textAlign: 'center',
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },
  commandSpacer: { width: 48 },

  // Cover photo — wide hero card at the top of the profile.
  coverWrapper: {
    height: 150,
    borderRadius: theme.radius.lg,
    backgroundColor: theme.colors.bone,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    overflow: 'hidden',
    marginBottom: theme.spacing[4],
    justifyContent: 'center',
  },
  coverImage: {
    // Pin to the wrapper's bounds so the photo can never push the 160px
    // box taller — `contentFit="cover"` then crops to fill. (A percentage
    // height on expo-image falls back to the image's intrinsic size and
    // blows out the layout.)
    ...StyleSheet.absoluteFillObject,
  },
  coverPlaceholder: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: theme.spacing[1],
  },
  coverPlaceholderText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  coverPlaceholderHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
  },
  coverBadge: {
    position: 'absolute',
    right: theme.spacing[3],
    bottom: theme.spacing[3],
    width: 32,
    height: 32,
    borderRadius: 16,
    backgroundColor: theme.colors.paper,
    alignItems: 'center',
    justifyContent: 'center',
    ...theme.shadow[1],
  },

  scroll: { flex: 1 },
  scrollContent: {
    paddingBottom: 120,
  },

  // Identity block — its own white card on the bone canvas (spec §1)
  identityBlock: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[4],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[5],
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[6],
    ...theme.shadow[1],
  },
  avatarWrapper: {
    width: 72,
    height: 72,
    borderRadius: theme.radius.full,
    overflow: 'visible',
  },
  avatarImage: {
    width: 72,
    height: 72,
    borderRadius: theme.radius.full,
  },
  avatarFallback: {
    width: 72,
    height: 72,
    borderRadius: theme.radius.full,
    backgroundColor: theme.colors.ink.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  avatarInitials: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 20,
    color: theme.colors.paper,
    letterSpacing: 0.5,
  },
  avatarBadge: {
    position: 'absolute',
    bottom: 0,
    right: 0,
    width: 24,
    height: 24,
    borderRadius: 12,
    backgroundColor: theme.colors.bone,
    borderWidth: 1.5,
    borderColor: theme.colors.paper,
    alignItems: 'center',
    justifyContent: 'center',
  },
  identityText: { flex: 1 },
  identityName: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  identityCaption: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    marginTop: 2,
  },

  // Section label
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[2],
  },

  // Section group card — white surface on the bone canvas (spec §1).
  // Name kept from the v1 hairline layout to avoid call-site churn.
  hairlineGroup: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[6],
    paddingBottom: theme.spacing[1],
    ...theme.shadow[1],
  },
  // Inset hairline separator — aligned to row text, not edge-to-edge
  separator: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginLeft: theme.spacing[4],
  },

  // Editable field
  editRow: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[2],
  },
  editLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.3,
    marginBottom: theme.spacing[1],
  },
  // Bone input box inside the white group card (spec §1 surface flip)
  editInput: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    minHeight: 44,
    paddingVertical: theme.spacing[2],
    paddingHorizontal: theme.spacing[3],
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.md,
  },
  editInputMultiline: {
    minHeight: 88,
    lineHeight: 22,
    paddingTop: theme.spacing[3],
  },

  // Kitchen photo strip — white card wrapping the horizontal scroll
  photosCard: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[6],
    ...theme.shadow[1],
  },
  photoStrip: {
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[4],
    gap: theme.spacing[3],
    flexDirection: 'row',
    alignItems: 'center',
  },
  photoThumb: {
    width: 90,
    height: 90,
    borderRadius: theme.radius.DEFAULT,
  },
  photoAddSlot: {
    width: 90,
    height: 90,
    borderRadius: theme.radius.DEFAULT,
    borderWidth: 1,
    borderStyle: 'dashed',
    borderColor: theme.colors.mist.strong,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: theme.colors.paper,
  },

  // Sticky Save footer — paper surface lifted off the canvas with a
  // top shadow instead of a hairline (spec §6-style top elevation)
  stickyFooter: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    backgroundColor: theme.colors.paper,
    shadowColor: theme.colors.ink.DEFAULT,
    shadowOffset: { width: 0, height: -2 },
    shadowOpacity: 0.06,
    shadowRadius: 12,
    elevation: 8,
  },
  stickyFooterInner: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
  },
  saveBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingVertical: theme.spacing[4],
    alignItems: 'center',
    minHeight: 52,
    justifyContent: 'center',
  },
  saveBtnDisabled: {
    backgroundColor: theme.colors.mist.DEFAULT,
  },
  saveBtnLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
    letterSpacing: 0.2,
  },
  saveBtnLabelDisabled: {
    color: theme.colors.ink.muted,
  },

  // Chip field — label above a wrapping flex row of Chip pills
  chipFieldRow: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
  },
  chipFieldLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.3,
    marginBottom: theme.spacing[2],
  },
  chipWrap: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: theme.spacing[2],
  },
  chipStateRow: {
    gap: theme.spacing[2],
    paddingRight: theme.spacing[4],
  },

  // Error state
  errorBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[4],
  },
  errorBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[6],
    paddingVertical: theme.spacing[3],
  },
  errorBtnLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
  },
});
