/**
 * MenuItemForm — shared form body for new.tsx and [itemId]/edit.tsx.
 *
 * Design language: v2 "canvas + cards" (UI-V2-SPEC) — bone canvas, white
 * group cards per labelled section, bone-filled inputs, ink-fill pill chips
 * for category/diet/prep-time selectors. Single-column.
 */
import { useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Animated,
  KeyboardAvoidingView,
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
import { DietIcon } from '../../components/vendor/DietIcon';
import type { MenuItemImage, Category } from '../../hooks/useVendorMenu';

// ---- Constants ---------------------------------------------------------------

const PREP_TIME_OPTIONS = [5, 10, 15, 20, 30, 45, 60] as const;
type PrepTime = (typeof PREP_TIME_OPTIONS)[number];

// ---- Public types ------------------------------------------------------------

export interface MenuItemFormValues {
  name: string;
  description: string;
  price: string;
  categoryId: string;
  isVeg: boolean;
  preparationTime: number;
  // HSN/SAC — optional. Backend defaults to 996331 (restaurant
  // services) when empty. Most chefs leave this alone; surfaces as
  // an "Advanced" field so it doesn't add visual noise to the
  // common case.
  hsn: string;
}

export interface MenuItemFormProps {
  /** "new" or "edit" — drives title, right-header action, and delete affordance */
  mode: 'new' | 'edit';
  /** Initial field values. For "new" pass the blank defaults. */
  initialValues: MenuItemFormValues;
  /** Existing uploaded photos (edit mode only). */
  existingPhotos?: MenuItemImage[];
  /** All available categories from the menu cache. */
  categories: Category[];
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
  onDelete,
  isDeleting = false,
  onSave,
  isSaving,
  onRemoveExistingPhoto,
  onAddPhoto,
  isUploadingPhoto = false,
  onCreateCategory,
  onBack,
}: MenuItemFormProps) {
  const { show: showToast } = useToast();

  // Form state — plain useState mirrors profile.tsx's EditableField pattern
  const [name, setName] = useState(initialValues.name);
  const [description, setDescription] = useState(initialValues.description);
  const [price, setPrice] = useState(initialValues.price);
  const [categoryId, setCategoryId] = useState(initialValues.categoryId);
  const [isVeg, setIsVeg] = useState(initialValues.isVeg);
  const [preparationTime, setPreparationTime] = useState(initialValues.preparationTime);
  const [hsn, setHsn] = useState(initialValues.hsn);

  // Local photo URIs (new-mode queuing or pre-upload preview)
  const [localPhotoUris, setLocalPhotoUris] = useState<string[]>([]);

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
    preparationTime !== initialValues.preparationTime ||
    hsn !== initialValues.hsn ||
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
      { name: name.trim(), description: description.trim(), price, categoryId, isVeg, preparationTime, hsn: hsn.trim() },
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

  async function handlePickPhoto() {
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      quality: 0.85,
      allowsEditing: true,
      aspect: [4, 3],
    });
    if (!result.canceled && result.assets[0]) {
      const uri = result.assets[0].uri;
      if (mode === 'edit') {
        // Upload immediately in edit mode; parent handles the API call
        onAddPhoto?.(uri);
      } else {
        // Queue for after create in new mode
        setLocalPhotoUris((prev) => [...prev, uri]);
      }
    }
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
                <PhotoAddTile onPress={handlePickPhoto} uploading={isUploadingPhoto} />
              )}
            </ScrollView>
            {isUploadingPhoto && (
              <Text style={styles.uploadingLabel}>Uploading…</Text>
            )}
            <Text style={styles.photoHint}>
              Up to 5 photos. First photo is shown to customers.
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
            {errors.categoryId ? (
              <Text style={[fieldStyles.error, { marginTop: theme.spacing[2] }]}>
                {errors.categoryId}
              </Text>
            ) : null}
            {showNewCatInput && (
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
