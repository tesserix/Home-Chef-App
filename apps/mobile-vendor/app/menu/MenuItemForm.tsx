/**
 * MenuItemForm — shared form body for new.tsx and [itemId]/edit.tsx.
 *
 * Design language: v2 "canvas + cards" (UI-V2-SPEC) — bone canvas, white
 * group cards per labelled section, bone-filled inputs, ink-fill pill chips
 * for category/diet/prep-time selectors. Single-column.
 */
import { useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Animated,
  KeyboardAvoidingView,
  Linking,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Image } from 'expo-image';
import * as ImagePicker from 'expo-image-picker';
import { ChevronLeft, Plus } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { useToast } from '@homechef/mobile-shared/ui';
import { DIET_OPTIONS, ALLERGEN_OPTIONS } from '@homechef/mobile-shared/dietary';
import { DietIcon } from '../../components/vendor/DietIcon';
import { ModifierComboEditor } from '../../components/vendor/ModifierComboEditor';
import type {
  MenuItemImage,
  Category,
  ModifierGroupInput,
  ComboItemInput,
} from '../../hooks/useVendorMenu';

// ---- Constants ---------------------------------------------------------------

const PREP_TIME_OPTIONS = [5, 10, 15, 20, 30, 45, 60] as const;
type PrepTime = (typeof PREP_TIME_OPTIONS)[number];

// Keep menu photos light — reject anything over 5 MB after the picker's own
// compression. (Images are captured/picked at quality 0.6.)
const MAX_IMAGE_BYTES = 5 * 1024 * 1024;

// Diet tags the chef adds beyond the veg/non-veg toggle (#41). "vegetarian" is
// excluded — that's the DIET toggle's job.
const EXTRA_DIET_OPTIONS = DIET_OPTIONS.filter((o) => o.value !== 'vegetarian');

// ---- Public types ------------------------------------------------------------

export interface MenuItemFormValues {
  name: string;
  description: string;
  price: string;
  categoryId: string;
  isVeg: boolean;
  // Extra diet tags + declared allergens (#41).
  dietaryTags: string[];
  allergens: string[];
  // Add-ons / combos (#52).
  isCombo: boolean;
  modifierGroups: ModifierGroupInput[];
  comboItems: ComboItemInput[];
  preparationTime: number;
  // HSN/SAC — optional. Backend defaults to 996331 (restaurant
  // services) when empty. Most chefs leave this alone; surfaces as
  // an "Advanced" field so it doesn't add visual noise to the
  // common case.
  hsn: string;
  // Weekly-menu schedule: weekdays (0=Sun..6=Sat) this dish is offered. Empty =
  // available every day. Lets the chef set a fixed rotation instead of toggling
  // availability daily — today's dishes auto-show for customers.
  availableDays: number[];
}

// DAY_OPTIONS drives the weekly-menu day chips (0=Sun..6=Sat), matching the
// backend AvailableDays convention.
const DAY_OPTIONS: ReadonlyArray<{ value: number; label: string }> = [
  { value: 0, label: 'Sun' },
  { value: 1, label: 'Mon' },
  { value: 2, label: 'Tue' },
  { value: 3, label: 'Wed' },
  { value: 4, label: 'Thu' },
  { value: 5, label: 'Fri' },
  { value: 6, label: 'Sat' },
];

export interface MenuItemFormProps {
  /** "new" or "edit" — drives title, right-header action, and delete affordance */
  mode: 'new' | 'edit';
  /** Initial field values. For "new" pass the blank defaults. */
  initialValues: MenuItemFormValues;
  /** Existing uploaded photos (edit mode only). */
  existingPhotos?: MenuItemImage[];
  /** All available categories from the menu cache. */
  categories: Category[];
  /** The chef's other menu items, for the combo builder's item picker (#52). */
  menuItems?: { id: string; name: string }[];
  /** Called when the chef confirms deletion (edit mode only). */
  onDelete?: () => void;
  /** True while the delete mutation is in-flight. */
  isDeleting?: boolean;
  /** Called with the final validated values when the chef saves. */
  onSave: (values: MenuItemFormValues, localPhotoUris: string[]) => void;
  /** True while the save mutation is in-flight. */
  isSaving: boolean;
  /** Called when the chef taps the − (remove) button on an existing photo. */
  onRemoveExistingPhoto?: (imageId: string) => void;
  /** Called after each new photo is picked (uri). Parent uploads immediately if
   *  in edit mode, or queues for post-create in new mode. */
  onAddPhoto?: (uri: string) => void;
  /** True while a photo upload is in-flight (edit mode). */
  isUploadingPhoto?: boolean;
  /** Called when the chef taps a "+ New category" and confirms a name. */
  onCreateCategory?: (name: string) => Promise<Category>;
  /** Called when the chef confirms they want to back out. Parent owns the
   *  actual navigation. */
  onBack: () => void;
  /** Called on every field change with the current values so the parent can
   *  persist an in-progress draft (new mode only). Photo URIs are local file
   *  paths that may dangle across cold starts, so they're intentionally not
   *  part of the draft. */
  onDraftChange?: (values: MenuItemFormValues) => void;
}

// ---- Inline sub-components --------------------------------------------------

interface FormFieldProps {
  label: string;
  error?: string;
  children: React.ReactNode;
}

function FormField({ label, error, children }: FormFieldProps) {
  return (
    <View style={fieldStyles.root}>
      <Text style={fieldStyles.label}>{label}</Text>
      {children}
      {error ? <Text style={fieldStyles.error}>{error}</Text> : null}
    </View>
  );
}

const fieldStyles = StyleSheet.create({
  // Fields live inside a padded white card — spacing only, no hairlines.
  root: {
    marginBottom: theme.spacing[3],
  },
  label: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.3,
    marginBottom: theme.spacing[1],
  },
  error: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.destructive.DEFAULT,
    marginTop: theme.spacing[1],
  },
});

// ---- Selector chips (UI-V2-SPEC §5) ------------------------------------------
// radius.full pills — selected = ink fill + paper text; unselected = paper bg
// + mist.strong border + ink.soft text. Matches the menu screen category chips.

const chipStyles = StyleSheet.create({
  root: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: theme.spacing[2],
    minHeight: 36,
    paddingHorizontal: theme.spacing[3],
    borderRadius: theme.radius.full,
    backgroundColor: theme.colors.paper,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
  },
  rootActive: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderColor: theme.colors.ink.DEFAULT,
  },
  label: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  labelActive: { color: theme.colors.paper },
  labelTabular: { fontVariant: ['tabular-nums'] },
});

// ---- Category chip ------------------------------------------------------------

interface CategoryTabProps {
  label: string;
  active: boolean;
  onPress: () => void;
}

function CategoryTab({ label, active, onPress }: CategoryTabProps) {
  return (
    <Pressable
      onPress={onPress}
      hitSlop={6}
      accessibilityRole="tab"
      accessibilityState={{ selected: active }}
    >
      {({ pressed }) => (
        <View
          style={[
            chipStyles.root,
            active && chipStyles.rootActive,
            pressed && { opacity: 0.7 },
          ]}
        >
          <Text style={[chipStyles.label, active && chipStyles.labelActive]}>
            {label}
          </Text>
        </View>
      )}
    </Pressable>
  );
}

// ---- Prep time chip -----------------------------------------------------------

interface PrepTabProps {
  value: number;
  active: boolean;
  onPress: () => void;
}

function PrepTab({ value, active, onPress }: PrepTabProps) {
  return (
    <Pressable
      onPress={onPress}
      hitSlop={6}
      accessibilityRole="radio"
      accessibilityState={{ checked: active }}
      accessibilityLabel={`${value} minutes`}
    >
      {({ pressed }) => (
        <View
          style={[
            chipStyles.root,
            active && chipStyles.rootActive,
            pressed && { opacity: 0.7 },
          ]}
        >
          <Text
            style={[
              chipStyles.label,
              chipStyles.labelTabular,
              active && chipStyles.labelActive,
            ]}
          >
            {value} min
          </Text>
        </View>
      )}
    </Pressable>
  );
}

// ---- Diet chip ------------------------------------------------------------------

interface DietTabProps {
  label: string;
  /** Identifies which option this tab represents. Drives the persistent
   *  green/red DietIcon — does NOT change based on active state. */
  optionIsVeg: boolean;
  active: boolean;
  onPress: () => void;
}

function DietTab({ label, optionIsVeg, active, onPress }: DietTabProps) {
  return (
    <Pressable
      onPress={onPress}
      hitSlop={6}
      accessibilityRole="radio"
      accessibilityState={{ checked: active }}
      accessibilityLabel={label}
    >
      {({ pressed }) => (
        <View
          style={[
            chipStyles.root,
            active && chipStyles.rootActive,
            pressed && { opacity: 0.7 },
          ]}
        >
          <DietIcon isVeg={optionIsVeg} size={12} />
          <Text style={[chipStyles.label, active && chipStyles.labelActive]}>
            {label}
          </Text>
        </View>
      )}
    </Pressable>
  );
}

// ---- Multi-select chip (diet tags + allergens, #41) -------------------------

interface MultiChipProps {
  label: string;
  active: boolean;
  onPress: () => void;
}

function MultiChip({ label, active, onPress }: MultiChipProps) {
  return (
    <Pressable
      onPress={onPress}
      hitSlop={6}
      accessibilityRole="checkbox"
      accessibilityState={{ checked: active }}
      accessibilityLabel={label}
    >
      {({ pressed }) => (
        <View style={[chipStyles.root, active && chipStyles.rootActive, pressed && { opacity: 0.7 }]}>
          <Text style={[chipStyles.label, active && chipStyles.labelActive]}>{label}</Text>
        </View>
      )}
    </Pressable>
  );
}

// ---- Photo thumb + Add tile -------------------------------------------------

interface PhotoThumbProps {
  uri: string;
  onRemove?: () => void;
}

function PhotoThumb({ uri, onRemove }: PhotoThumbProps) {
  return (
    <View style={photoStyles.thumbWrap}>
      <Image
        source={{ uri }}
        style={photoStyles.thumbImg}
        contentFit="cover"
      />
      {onRemove ? (
        <Pressable
          onPress={onRemove}
          hitSlop={{ top: 12, right: 12, bottom: 12, left: 12 }}
          accessibilityRole="button"
          accessibilityLabel="Remove photo"
          style={photoStyles.removeBtnWrap}
        >
          {({ pressed }) => (
            <View style={[photoStyles.removeBtn, pressed && { opacity: 0.75 }]}>
              <Text style={photoStyles.removeBtnText}>×</Text>
            </View>
          )}
        </Pressable>
      ) : null}
    </View>
  );
}

function PhotoAddTile({ onPress, uploading }: { onPress: () => void; uploading?: boolean }) {
  return (
    <Pressable
      onPress={onPress}
      disabled={uploading}
      accessibilityRole="button"
      accessibilityLabel="Add photo"
    >
      {({ pressed }) => (
        <View
          style={[
            photoStyles.addTile,
            (pressed || uploading) && { opacity: 0.65 },
          ]}
        >
          {uploading ? (
            <ActivityIndicator size="small" color={theme.colors.ink.muted} />
          ) : (
            <Plus size={20} color={theme.colors.ink.muted} strokeWidth={1.75} />
          )}
        </View>
      )}
    </Pressable>
  );
}

const photoStyles = StyleSheet.create({
  thumbWrap: { position: 'relative' },
  thumbImg: {
    width: 60,
    height: 60,
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.bone,
  },
  removeBtnWrap: {
    position: 'absolute',
    top: -6,
    right: -6,
  },
  removeBtn: {
    width: 20,
    height: 20,
    borderRadius: 10,
    backgroundColor: theme.colors.ink.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1.5,
    borderColor: theme.colors.paper,
  },
  removeBtnText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    lineHeight: 14,
    color: theme.colors.paper,
  },
  addTile: {
    width: 60,
    height: 60,
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderStyle: 'dashed',
    borderColor: theme.colors.mist.strong,
    backgroundColor: theme.colors.bone,
    alignItems: 'center',
    justifyContent: 'center',
  },
});

// ---- Main component ---------------------------------------------------------

export function MenuItemForm({
  mode,
  initialValues,
  existingPhotos = [],
  categories,
  menuItems,
  onDelete,
  isDeleting = false,
  onSave,
  isSaving,
  onRemoveExistingPhoto,
  onAddPhoto,
  isUploadingPhoto = false,
  onCreateCategory,
  onBack,
  onDraftChange,
}: MenuItemFormProps) {
  const { show: showToast } = useToast();

  // Form state — plain useState mirrors profile.tsx's EditableField pattern
  const [name, setName] = useState(initialValues.name);
  const [description, setDescription] = useState(initialValues.description);
  const [price, setPrice] = useState(initialValues.price);
  const [categoryId, setCategoryId] = useState(initialValues.categoryId);
  const [isVeg, setIsVeg] = useState(initialValues.isVeg);
  const [dietTags, setDietTags] = useState<string[]>(initialValues.dietaryTags ?? []);
  const [allergens, setAllergens] = useState<string[]>(initialValues.allergens ?? []);
  // Add-ons / combos (#52).
  const [modifierGroups, setModifierGroups] = useState<ModifierGroupInput[]>(initialValues.modifierGroups ?? []);
  const [comboItems, setComboItems] = useState<ComboItemInput[]>(initialValues.comboItems ?? []);
  const [isCombo, setIsCombo] = useState(initialValues.isCombo ?? false);
  const [preparationTime, setPreparationTime] = useState(initialValues.preparationTime);
  const [hsn, setHsn] = useState(initialValues.hsn);
  const [availableDays, setAvailableDays] = useState<number[]>(initialValues.availableDays ?? []);

  const toggleIn = (set: React.Dispatch<React.SetStateAction<string[]>>) => (value: string) =>
    set((prev) => (prev.includes(value) ? prev.filter((v) => v !== value) : [...prev, value]));

  // Local photo URIs (new-mode queuing or pre-upload preview)
  const [localPhotoUris, setLocalPhotoUris] = useState<string[]>([]);

  // Persist an in-progress draft on every field change (new mode only — the
  // parent only wires onDraftChange there). Text/selection fields only; photo
  // URIs are excluded (see prop doc).
  useEffect(() => {
    onDraftChange?.({
      name,
      description,
      price,
      categoryId,
      isVeg,
      dietaryTags: dietTags,
      allergens,
      isCombo,
      modifierGroups,
      comboItems,
      preparationTime,
      hsn,
      availableDays,
    });
  }, [
    name,
    description,
    price,
    categoryId,
    isVeg,
    dietTags,
    allergens,
    isCombo,
    modifierGroups,
    comboItems,
    preparationTime,
    hsn,
    availableDays,
    onDraftChange,
  ]);

  // Validation errors (surface on Save attempt)
  const [errors, setErrors] = useState<Partial<Record<keyof MenuItemFormValues, string>>>({});

  // New category inline input
  const [showNewCatInput, setShowNewCatInput] = useState(false);
  const [newCatName, setNewCatName] = useState('');
  const [isCreatingCategory, setIsCreatingCategory] = useState(false);

  // Dirty detection for sticky footer animation
  const isDirty =
    name !== initialValues.name ||
    description !== initialValues.description ||
    price !== initialValues.price ||
    categoryId !== initialValues.categoryId ||
    isVeg !== initialValues.isVeg ||
    dietTags.join('|') !== (initialValues.dietaryTags ?? []).join('|') ||
    allergens.join('|') !== (initialValues.allergens ?? []).join('|') ||
    isCombo !== (initialValues.isCombo ?? false) ||
    JSON.stringify(modifierGroups) !== JSON.stringify(initialValues.modifierGroups ?? []) ||
    JSON.stringify(comboItems) !== JSON.stringify(initialValues.comboItems ?? []) ||
    preparationTime !== initialValues.preparationTime ||
    hsn !== initialValues.hsn ||
    availableDays.join('|') !== (initialValues.availableDays ?? []).join('|') ||
    localPhotoUris.length > 0 ||
    mode === 'new'; // new form is always "ready to save"

  const footerTranslate = useRef(new Animated.Value(mode === 'new' ? 0 : 80)).current;

  // We use useRef to compare against the previous isDirty without triggering
  // an extra render on every keystroke.
  const prevDirty = useRef(mode === 'new' ? true : false);
  if (prevDirty.current !== isDirty) {
    prevDirty.current = isDirty;
    Animated.timing(footerTranslate, {
      toValue: isDirty ? 0 : 80,
      duration: theme.motion.duration.default,
      useNativeDriver: true,
    }).start();
  }

  // ---- Validation -----------------------------------------------------------

  function validate(): boolean {
    const next: Partial<Record<keyof MenuItemFormValues, string>> = {};
    if (name.trim().length < 3) next.name = 'Name must be at least 3 characters';
    if (description.trim().length < 20) next.description = 'Description must be at least 20 characters';
    const priceNum = Number(price);
    if (!price.trim() || isNaN(priceNum) || priceNum <= 0 || priceNum > 10_000) {
      next.price = 'Price must be between ₹1 and ₹10,000';
    }
    if (!categoryId) next.categoryId = 'Select a category';
    setErrors(next);
    return Object.keys(next).length === 0;
  }

  // ---- Handlers -------------------------------------------------------------

  function handleSave() {
    if (!validate()) return;
    onSave(
      {
        name: name.trim(),
        description: description.trim(),
        price,
        categoryId,
        isVeg,
        dietaryTags: dietTags,
        allergens,
        isCombo,
        // Drop blank groups/options so a half-filled row doesn't persist (#52).
        modifierGroups: modifierGroups
          .filter((g) => g.name.trim() !== '')
          .map((g) => ({ ...g, options: g.options.filter((o) => o.name.trim() !== '') }))
          .filter((g) => g.options.length > 0),
        comboItems: isCombo ? comboItems : [],
        preparationTime,
        hsn: hsn.trim(),
        availableDays: [...availableDays].sort((a, b) => a - b),
      },
      localPhotoUris,
    );
  }

  function handleDelete() {
    Alert.alert(
      'Delete item?',
      'This dish will be permanently removed from your menu.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: () => onDelete?.(),
        },
      ],
    );
  }

  // Ask for the right OS permission up front; if denied, point the chef at
  // Settings instead of silently doing nothing.
  async function ensurePermission(kind: 'camera' | 'library'): Promise<boolean> {
    const res =
      kind === 'camera'
        ? await ImagePicker.requestCameraPermissionsAsync()
        : await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (res.granted) return true;
    Alert.alert(
      kind === 'camera' ? 'Camera access needed' : 'Photo access needed',
      `Allow ${kind === 'camera' ? 'camera' : 'photo library'} access in Settings to ${
        kind === 'camera' ? 'take a photo of your dish' : 'choose a photo'
      }.`,
      [
        { text: 'Not now', style: 'cancel' },
        { text: 'Open Settings', onPress: () => void Linking.openSettings() },
      ],
    );
    return false;
  }

  function acceptImage(asset: ImagePicker.ImagePickerAsset): boolean {
    if (typeof asset.fileSize === 'number' && asset.fileSize > MAX_IMAGE_BYTES) {
      Alert.alert(
        'Photo too large',
        'Please use a photo under 5 MB — try cropping it or taking a new shot.',
      );
      return false;
    }
    return true;
  }

  function addPhoto(uri: string) {
    if (mode === 'edit') {
      // Upload immediately in edit mode; parent handles the API call.
      onAddPhoto?.(uri);
    } else {
      // Queue for after create in new mode.
      setLocalPhotoUris((prev) => [...prev, uri]);
    }
  }

  async function takePhoto() {
    if (!(await ensurePermission('camera'))) return;
    const result = await ImagePicker.launchCameraAsync({
      mediaTypes: ['images'],
      quality: 0.6,
      allowsEditing: true,
      aspect: [4, 3],
    });
    const asset = result.canceled ? null : result.assets[0];
    if (asset && acceptImage(asset)) addPhoto(asset.uri);
  }

  async function pickFromLibrary() {
    if (!(await ensurePermission('library'))) return;
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      quality: 0.6,
      allowsEditing: true,
      aspect: [4, 3],
    });
    const asset = result.canceled ? null : result.assets[0];
    if (asset && acceptImage(asset)) addPhoto(asset.uri);
  }

  // Let the chef take a live photo or pick from the library, with a clear
  // reminder to keep personal information out of the shot.
  function handleAddMedia() {
    Alert.alert(
      'Add a photo',
      "Show the dish only — please don't include people's faces, IDs, addresses, vehicle number plates, or any personal information.",
      [
        { text: 'Take Photo', onPress: () => void takePhoto() },
        { text: 'Choose from Library', onPress: () => void pickFromLibrary() },
        { text: 'Cancel', style: 'cancel' },
      ],
    );
  }

  async function handleCreateCategory() {
    const trimmed = newCatName.trim();
    if (trimmed.length < 2 || !onCreateCategory) return;
    setIsCreatingCategory(true);
    try {
      const created = await onCreateCategory(trimmed);
      setCategoryId(created.id);
      setNewCatName('');
      setShowNewCatInput(false);
      showToast({ message: `Category "${trimmed}" created.`, tone: 'success' });
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } } | null)?.response?.data?.error ??
        'Could not add category. Try again.';
      Alert.alert('Category error', msg);
    } finally {
      setIsCreatingCategory(false);
    }
  }

  const totalPhotos = existingPhotos.length + localPhotoUris.length;
  const canAddMore = totalPhotos < 5;

  // ---- Render ---------------------------------------------------------------

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <KeyboardAvoidingView
        style={styles.flex}
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        keyboardVerticalOffset={0}
      >
        {/* Command bar */}
        <View style={styles.commandBar}>
          <Pressable
            onPress={() => {
              // Dirty guard in new mode only — in edit the chef already
              // has server-saved state and can back out freely.
              if (
                mode === 'new' &&
                isDirty &&
                (name.trim() || description.trim() || price.trim())
              ) {
                Alert.alert(
                  'Discard changes?',
                  "Your new item hasn't been saved.",
                  [
                    { text: 'Keep editing', style: 'cancel' },
                    {
                      text: 'Discard',
                      style: 'destructive',
                      onPress: onBack,
                    },
                  ],
                );
                return;
              }
              onBack();
            }}
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

          <Text style={styles.commandTitle} numberOfLines={1}>
            {mode === 'new' ? 'New item' : 'Edit item'}
          </Text>

          {mode === 'edit' && onDelete ? (
            <Pressable
              onPress={handleDelete}
              disabled={isDeleting}
              hitSlop={8}
              accessibilityRole="button"
              accessibilityLabel="Delete this item"
              style={styles.rightSlot}
            >
              {({ pressed }) => (
                <Text
                  style={[
                    styles.deleteLabel,
                    (pressed || isDeleting) && { opacity: 0.6 },
                  ]}
                >
                  {isDeleting ? 'Deleting…' : 'Delete'}
                </Text>
              )}
            </Pressable>
          ) : (
            // Spacer to keep title centred
            <View style={styles.rightSlot} />
          )}
        </View>

        <ScrollView
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          {/* PHOTOS section */}
          <Text style={styles.sectionLabel}>PHOTOS</Text>
          <View style={styles.card}>
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerStyle={styles.photoStrip}
            >
              {existingPhotos.map((img) => (
                <PhotoThumb
                  key={img.id}
                  uri={img.url}
                  onRemove={onRemoveExistingPhoto ? () => onRemoveExistingPhoto(img.id) : undefined}
                />
              ))}
              {localPhotoUris.map((uri, idx) => (
                <PhotoThumb
                  key={`local-${idx}`}
                  uri={uri}
                  onRemove={() =>
                    setLocalPhotoUris((prev) => prev.filter((_, i) => i !== idx))
                  }
                />
              ))}
              {canAddMore && (
                <PhotoAddTile onPress={handleAddMedia} uploading={isUploadingPhoto} />
              )}
            </ScrollView>
            {isUploadingPhoto && (
              <Text style={styles.uploadingLabel}>Uploading…</Text>
            )}
            <Text style={styles.photoHint}>
              Up to 5 photos. First photo is shown to customers. Show the dish
              only — no faces, IDs, addresses, or other personal information.
            </Text>
          </View>

          {/* DETAILS section */}
          <Text style={styles.sectionLabel}>DETAILS</Text>
          <View style={styles.card}>
            {/* Name */}
            <FormField label="Item name" error={errors.name}>
              <TextInput
                value={name}
                onChangeText={(t) => {
                  setName(t);
                  if (errors.name) setErrors((e) => ({ ...e, name: undefined }));
                }}
                placeholder="e.g. Butter Chicken"
                placeholderTextColor={theme.colors.ink.muted}
                style={[styles.textInput, errors.name && styles.textInputError]}
                autoCapitalize="words"
                returnKeyType="next"
              />
            </FormField>

            {/* Description */}
            <FormField label="Description" error={errors.description}>
              <TextInput
                value={description}
                onChangeText={(t) => {
                  setDescription(t);
                  if (errors.description) setErrors((e) => ({ ...e, description: undefined }));
                }}
                placeholder="Tell customers what makes this dish special (min 20 chars)"
                placeholderTextColor={theme.colors.ink.muted}
                multiline
                numberOfLines={4}
                textAlignVertical="top"
                style={[styles.textInput, styles.textInputMultiline, errors.description && styles.textInputError]}
              />
            </FormField>

            {/* Price — inline ₹ prefix + tabular value. Matches body-size
                inputs above for consistent rhythm; the dashboard / detail
                hero treatments are where the Geist-Bold display weight
                lives, not here on an inline form field. */}
            <FormField label="Price" error={errors.price}>
              <View style={styles.priceRow}>
                <Text style={styles.pricePrefix}>₹</Text>
                <TextInput
                  value={price}
                  onChangeText={(t) => {
                    setPrice(t);
                    if (errors.price) setErrors((e) => ({ ...e, price: undefined }));
                  }}
                  placeholder="0"
                  placeholderTextColor={theme.colors.ink.muted}
                  keyboardType="decimal-pad"
                  style={[
                    styles.textInput,
                    styles.priceInput,
                    errors.price && styles.textInputError,
                  ]}
                />
              </View>
            </FormField>
          </View>

          {/* CATEGORY section */}
          <Text style={styles.sectionLabel}>CATEGORY</Text>
          <View style={styles.card}>
            {categories.length > 0 ? (
              <View style={styles.tabBarWrap}>
                <ScrollView
                  horizontal
                  showsHorizontalScrollIndicator={false}
                  contentContainerStyle={styles.tabBar}
                >
                  {categories.map((cat) => (
                    <CategoryTab
                      key={cat.id}
                      label={cat.name}
                      active={categoryId === cat.id}
                      onPress={() => {
                        setCategoryId(cat.id);
                        if (errors.categoryId) setErrors((e) => ({ ...e, categoryId: undefined }));
                      }}
                    />
                  ))}
                  <CategoryTab
                    label="+ New"
                    active={false}
                    onPress={() => setShowNewCatInput((v) => !v)}
                  />
                </ScrollView>
              </View>
            ) : (
              <Text style={styles.categoryHint}>
                No categories yet — name your first one below (e.g. Starters) to
                organize your menu.
              </Text>
            )}
            {errors.categoryId ? (
              <Text style={[fieldStyles.error, { marginTop: theme.spacing[2] }]}>
                {errors.categoryId}
              </Text>
            ) : null}
            {(showNewCatInput || categories.length === 0) && (
              <View style={styles.newCatRow}>
                <TextInput
                  value={newCatName}
                  onChangeText={setNewCatName}
                  placeholder="Category name (e.g. Starters)"
                  placeholderTextColor={theme.colors.ink.muted}
                  style={styles.newCatInput}
                  autoCapitalize="words"
                  maxLength={40}
                  autoFocus
                />
                <Pressable
                  onPress={handleCreateCategory}
                  disabled={isCreatingCategory || newCatName.trim().length < 2}
                  accessibilityRole="button"
                  accessibilityLabel="Add category"
                >
                  {({ pressed }) => (
                    <View
                      style={[
                        styles.newCatAdd,
                        (isCreatingCategory || newCatName.trim().length < 2) && styles.newCatAddDisabled,
                        pressed && { opacity: 0.8 },
                      ]}
                    >
                      {isCreatingCategory ? (
                        <ActivityIndicator size="small" color={theme.colors.paper} />
                      ) : (
                        <Text style={styles.newCatAddLabel}>Add</Text>
                      )}
                    </View>
                  )}
                </Pressable>
              </View>
            )}
          </View>

          {/* DIET section — two-tab segmented control instead of a switch
              with a label that flips between Veg/Non-veg. The earlier UX
              made it easy to misread which state was actually saved; with
              fixed labels + persistent DietIcon colour per tab the active
              choice is unambiguous. */}
          <Text style={styles.sectionLabel}>DIET</Text>
          <View style={styles.card}>
            <View style={styles.tabBarWrap}>
              <View style={styles.dietTabBar}>
                <DietTab
                  label="Vegetarian"
                  optionIsVeg
                  active={isVeg}
                  onPress={() => setIsVeg(true)}
                />
                <DietTab
                  label="Non-vegetarian"
                  optionIsVeg={false}
                  active={!isVeg}
                  onPress={() => setIsVeg(false)}
                />
              </View>
            </View>
          </View>

          {/* DIETARY TAGS section (#41) — extra diet suitability beyond veg/non-veg */}
          <Text style={styles.sectionLabel}>DIETARY TAGS</Text>
          <View style={styles.card}>
            <View style={styles.wrapChips}>
              {EXTRA_DIET_OPTIONS.map((opt) => (
                <MultiChip
                  key={opt.value}
                  label={opt.label}
                  active={dietTags.includes(opt.value)}
                  onPress={() => toggleIn(setDietTags)(opt.value)}
                />
              ))}
            </View>
          </View>

          {/* ALLERGENS section (#41) — declared for customer safety + warnings */}
          <Text style={styles.sectionLabel}>ALLERGENS</Text>
          <View style={styles.card}>
            <Text style={styles.allergenHint}>
              Declare every allergen this dish contains — customers who flag these are warned.
            </Text>
            <View style={styles.wrapChips}>
              {ALLERGEN_OPTIONS.map((opt) => (
                <MultiChip
                  key={opt.value}
                  label={opt.label}
                  active={allergens.includes(opt.value)}
                  onPress={() => toggleIn(setAllergens)(opt.value)}
                />
              ))}
            </View>
          </View>

          {/* ADD-ONS + COMBO sections (#52) */}
          <ModifierComboEditor
            groups={modifierGroups}
            setGroups={setModifierGroups}
            isCombo={isCombo}
            setIsCombo={setIsCombo}
            comboItems={comboItems}
            setComboItems={setComboItems}
            menuItems={menuItems ?? []}
          />

          {/* PREP TIME section — own header so it sits in the same
              rhythm as CATEGORY (caps label + hairline group + scrollable
              underline tab strip). */}
          <Text style={styles.sectionLabel}>PREP TIME</Text>
          <View style={styles.card}>
            <View style={styles.tabBarWrap}>
              <ScrollView
                horizontal
                showsHorizontalScrollIndicator={false}
                contentContainerStyle={styles.tabBar}
              >
                {PREP_TIME_OPTIONS.map((mins) => (
                  <PrepTab
                    key={mins}
                    value={mins}
                    active={preparationTime === mins}
                    onPress={() => setPreparationTime(mins as PrepTime)}
                  />
                ))}
              </ScrollView>
            </View>
          </View>

          {/* AVAILABLE DAYS — weekly-menu schedule. Pick the days this dish is
              offered; leave all off to serve it every day. Today's dishes then
              auto-show for customers without daily edits. */}
          <Text style={styles.sectionLabel}>AVAILABLE DAYS</Text>
          <View style={styles.card}>
            <Text style={styles.allergenHint}>
              {availableDays.length === 0
                ? 'Available every day. Pick specific days to set a weekly menu.'
                : 'Only shown to customers on the selected days.'}
            </Text>
            <View style={styles.wrapChips}>
              {DAY_OPTIONS.map((opt) => (
                <MultiChip
                  key={opt.value}
                  label={opt.label}
                  active={availableDays.includes(opt.value)}
                  onPress={() =>
                    setAvailableDays((prev) =>
                      prev.includes(opt.value)
                        ? prev.filter((d) => d !== opt.value)
                        : [...prev, opt.value],
                    )
                  }
                />
              ))}
            </View>
          </View>

          {/* HSN — advanced tax code, optional. Defaults to 996331
              (restaurant services SAC) when blank. Printed on the
              customer's GST invoice per Wave 3. */}
          <Text style={styles.sectionLabel}>HSN / SAC (OPTIONAL)</Text>
          <View style={styles.card}>
            <TextInput
              value={hsn}
              onChangeText={(v) => setHsn(v.replace(/[^0-9]/g, '').slice(0, 8))}
              placeholder="996331 (default — restaurant services)"
              placeholderTextColor={theme.colors.ink.muted}
              keyboardType="number-pad"
              maxLength={8}
              autoCorrect={false}
              style={[styles.textInput, styles.hsnInput]}
            />
          </View>

          {/* Bottom padding to clear sticky footer */}
          <View style={{ height: 100 }} />
        </ScrollView>

        {/* Sticky Save footer */}
        <Animated.View
          style={[styles.stickyFooter, { transform: [{ translateY: footerTranslate }] }]}
          pointerEvents={isDirty ? 'auto' : 'none'}
        >
          <SafeAreaView edges={['bottom']} style={styles.stickyFooterInner}>
            <Pressable
              onPress={handleSave}
              disabled={isSaving}
              accessibilityRole="button"
              accessibilityLabel={mode === 'new' ? 'Add item' : 'Save changes'}
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.saveBtn,
                    (pressed || isSaving) && { opacity: 0.85 },
                  ]}
                >
                  {isSaving ? (
                    <ActivityIndicator color={theme.colors.paper} />
                  ) : (
                    <Text style={styles.saveBtnLabel}>
                      {mode === 'new' ? 'Add item' : 'Save changes'}
                    </Text>
                  )}
                </View>
              )}
            </Pressable>
          </SafeAreaView>
        </Animated.View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

// ---- Styles -----------------------------------------------------------------

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  flex: { flex: 1 },

  // Command bar
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
  },
  backBtn: {
    minWidth: 44,
    minHeight: 44,
    justifyContent: 'center',
  },
  commandTitle: {
    flex: 1,
    textAlign: 'center',
    fontFamily: 'Geist-Bold',
    fontSize: 22,
    lineHeight: 26,
    letterSpacing: -0.2,
    color: theme.colors.ink.DEFAULT,
  },
  rightSlot: {
    minWidth: 44,
    minHeight: 44,
    justifyContent: 'center',
    alignItems: 'flex-end',
  },
  deleteLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.destructive.DEFAULT,
    letterSpacing: 0.1,
  },

  // Scroll
  scroll: { flex: 1 },
  scrollContent: { paddingBottom: theme.spacing[6] },

  // Section label
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[2],
  },

  // Empty-state hint shown in the CATEGORY card when the chef has no
  // categories yet — points them at the inline "create your first one" field.
  categoryHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    lineHeight: 19,
    color: theme.colors.ink.soft,
  },

  // White group card on the bone canvas (UI-V2-SPEC §1)
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    ...theme.shadow[1],
    padding: theme.spacing[4],
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[4],
  },

  // Text inputs — bone fill inside the white cards, no border.
  textInput: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    minHeight: 44,
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
  },
  textInputMultiline: {
    minHeight: 88,
    lineHeight: 22,
  },
  textInputError: {
    color: theme.colors.destructive.DEFAULT,
  },
  hsnInput: {
    fontFamily: 'Inter-SemiBold',
  },

  // Price input — ₹ prefix + value share one bone-filled field so it
  // reads as a single cohesive input instead of a floating numeral.
  priceRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[3],
    minHeight: 44,
  },
  pricePrefix: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.soft,
    fontVariant: ['tabular-nums'],
  },
  priceInput: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontVariant: ['tabular-nums'],
    backgroundColor: 'transparent',
    paddingHorizontal: 0,
  },

  // Photo strip
  photoStrip: {
    gap: theme.spacing[3],
    flexDirection: 'row',
    alignItems: 'center',
  },
  uploadingLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[2],
  },
  photoHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[2],
    lineHeight: 16,
  },

  // Category chip strip — chips carry their own pill chrome; the wrap is
  // just structure now (card provides padding, no hairlines).
  tabBarWrap: {},
  tabBar: {
    gap: theme.spacing[2],
    alignItems: 'center',
  },

  // New category inline row
  newCatRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    marginTop: theme.spacing[3],
  },
  newCatInput: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    minHeight: 44,
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
  },
  newCatAdd: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    minHeight: 44,
    justifyContent: 'center',
    alignItems: 'center',
  },
  newCatAddDisabled: {
    backgroundColor: theme.colors.mist.DEFAULT,
  },
  newCatAddLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.paper,
  },

  // Diet two-chip row — both options stay visible without horizontal scroll.
  dietTabBar: {
    flexDirection: 'row',
    gap: theme.spacing[2],
  },

  // Wrap-flow chips for the multi-select diet-tag + allergen sections (#41).
  wrapChips: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: theme.spacing[2],
  },
  allergenHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    lineHeight: 16,
    color: theme.colors.ink.soft,
    marginBottom: theme.spacing[3],
  },

  // Sticky footer — white bar lifted off the canvas with a top shadow
  // (UI-V2-SPEC §6-style elevation, no hairline).
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
    justifyContent: 'center',
    minHeight: 52,
  },
  saveBtnLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
    letterSpacing: 0.2,
  },
});
