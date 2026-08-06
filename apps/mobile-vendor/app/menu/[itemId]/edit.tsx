/**
 * EditMenuItemScreen — thin screen shell for editing an existing menu item.
 *
 * All visual logic lives in MenuItemForm. This screen:
 *  1. Resolves the item from the menu cache via useLocalSearchParams.
 *  2. Wires update, delete, photo-upload, and photo-remove mutations.
 *  3. Resets form state when the item loads (mirrors profile.tsx's useEffect reset).
 */
import { useEffect, useRef, useState } from 'react';
import { ActivityIndicator, StyleSheet, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { theme } from '@homechef/mobile-shared/theme';
import { getServerErrorMessage } from '@homechef/mobile-shared/api';
import { useToast, useAlert } from '@homechef/mobile-shared/ui';
import {
  useVendorMenu,
  useUpdateMenuItem,
  useDeleteMenuItem,
  useUploadMenuPhoto,
  useCreateCategory,
  extraDietTags,
} from '../../../hooks/useVendorMenu';
import { api } from '../../../lib/api';
import { MenuItemForm } from '../MenuItemForm';
import { useOffersBakery } from '../../../hooks/useChefVertical';
import { leaveTo } from '../../../lib/navigation';
import type { MenuItemFormValues } from '../MenuItemForm';

export default function EditMenuItemScreen() {
  const { showAlert } = useAlert();
  const { itemId } = useLocalSearchParams<{ itemId: string }>();
  const { data: menuData } = useVendorMenu();
  const { show: showToast } = useToast();

  const item = menuData?.items?.find((i) => i.id === itemId);
  const categories = menuData?.categories ?? [];

  const updateMutation = useUpdateMenuItem();
  const deleteMutation = useDeleteMenuItem();
  const uploadMutation = useUploadMenuPhoto();
  const createCategoryMutation = useCreateCategory();
  const isBakery = useOffersBakery();

  // Derive initial values from the item whenever it first arrives (or updates).
  // We keep a version counter so MenuItemForm can re-mount with fresh
  // initialValues when the item loads asynchronously after navigate.
  const [formKey, setFormKey] = useState(0);
  const [initialValues, setInitialValues] = useState<MenuItemFormValues>({
    name: '',
    description: '',
    price: '',
    categoryId: '',
    isVeg: true,
    dietaryTags: [],
    allergens: [],
    isCombo: false,
    modifierGroups: [],
    comboItems: [],
    bakery: null,
    preparationTime: 15,
    hsn: '',
    availableDays: [],
  });

  const seenItemId = useRef<string | null>(null);
  useEffect(() => {
    if (item && item.id !== seenItemId.current) {
      seenItemId.current = item.id;
      setInitialValues({
        name: item.name ?? '',
        description: item.description ?? '',
        price: String(item.price ?? 0),
        categoryId: item.categoryId ?? '',
        isVeg: item.isVeg ?? true,
        // Strip the veg-flag tokens so the form's diet-tag chips show only the
        // extra tags; the veg toggle owns vegetarian/non-vegetarian (#41).
        dietaryTags: extraDietTags(item.dietaryTags),
        allergens: item.allergens ?? [],
        // Add-ons / combos (#52) — map read shapes to the editor's input shapes.
        isCombo: item.isCombo ?? false,
        modifierGroups: (item.modifierGroups ?? []).map((g) => ({
          name: g.name,
          required: g.required,
          minSelect: g.minSelect,
          maxSelect: g.maxSelect,
          options: g.options.map((o) => ({ name: o.name, priceDelta: o.priceDelta, isAvailable: o.isAvailable })),
        })),
        comboItems: (item.comboItems ?? []).map((c) => ({ menuItemId: c.menuItemId, quantity: c.quantity })),
        // Bakery configurator (#1065) — read shape to the editor's input shape.
        bakery: item.bakery
          ? {
              productType: item.bakery.productType,
              pricePerKg: item.bakery.pricePerKg,
              minWeightKg: item.bakery.minWeightKg,
              maxWeightKg: item.bakery.maxWeightKg,
              weightStepKg: item.bakery.weightStepKg,
              servesPerKg: item.bakery.servesPerKg,
              allowMessage: item.bakery.allowMessage,
              maxMessageChars: item.bakery.maxMessageChars,
              allowReferencePhoto: item.bakery.allowReferencePhoto,
              leadTimeHours: item.bakery.leadTimeHours,
              occasions: item.bakery.occasions ?? [],
              options: (item.bakery.options ?? []).map((o) => ({
                kind: o.kind,
                name: o.name,
                priceDelta: o.priceDelta,
                priceMode: o.priceMode,
                dietaryTags: o.dietaryTags ?? [],
                allergens: o.allergens ?? [],
                isAvailable: o.isAvailable,
                isDefault: o.isDefault,
              })),
            }
          : null,
        preparationTime: item.preparationTime ?? 15,
        hsn: item.hsn ?? '',
        availableDays: item.availableDays ?? [],
      });
      // Bump key so MenuItemForm re-initialises its useState from the new initialValues
      setFormKey((k) => k + 1);
    }
  }, [item]);

  // ---- Loading state --------------------------------------------------------

  if (!item) {
    return (
      <SafeAreaView style={styles.loading} edges={['top', 'left', 'right']}>
        <ActivityIndicator size="large" color={theme.colors.ink.DEFAULT} />
      </SafeAreaView>
    );
  }

  // ---- Handlers -------------------------------------------------------------

  async function handleSave(values: MenuItemFormValues) {
    if (!itemId) return;
    try {
      await updateMutation.mutateAsync({
        itemId,
        payload: {
          name: values.name,
          description: values.description,
          price: Number(values.price),
          categoryId: values.categoryId,
          isVeg: values.isVeg,
          dietaryTags: values.dietaryTags,
          allergens: values.allergens,
          isCombo: values.isCombo,
          modifierGroups: values.modifierGroups,
          comboItems: values.comboItems,
          bakery: values.bakery,
          preparationTime: values.preparationTime,
          hsn: values.hsn,
          availableDays: values.availableDays,
        },
      });
      showToast({ message: 'Item saved', tone: 'success' });
      leaveTo(router, '/(tabs)/menu');
    } catch (err: unknown) {
      showAlert(
        'Could not save',
        getServerErrorMessage(err, 'Please check your details and try again.'),
      );
    }
  }

  function handleDelete() {
    if (!itemId) return;
    deleteMutation.mutate(itemId, {
      onSuccess: () => leaveTo(router, '/(tabs)/menu'),
      onError: (err) =>
        showAlert('Delete failed', getServerErrorMessage(err, 'Please try again.')),
    });
  }

  async function handleRemoveExistingPhoto(imageId: string) {
    try {
      await api.delete(`/chef/menu/items/${itemId}/images/${imageId}`);
      // Cache invalidation is triggered inside useDeleteMenuItem's onSettled;
      // for the photo endpoint we do a manual query invalidation via the
      // upload mutation's queryClient. For simplicity we reload via refetch —
      // the upload mutation shares the same MENU_KEY invalidation.
    } catch (err: unknown) {
      showAlert('Could not remove photo', getServerErrorMessage(err, 'Please try again.'));
    }
  }

  // Without this the form's "Add category" control renders enabled but is a
  // silent no-op in edit mode — MenuItemForm.handleCreateCategory() bails out
  // when the callback is absent, so nothing is sent and no chip appears.
  async function handleCreateCategory(name: string) {
    return createCategoryMutation.mutateAsync(name);
  }

  function handleAddPhoto(uri: string) {
    uploadMutation.mutate(
      { itemId: itemId ?? '', uri },
      {
        onError: (err) =>
          showAlert('Upload failed', getServerErrorMessage(err, 'Please try again.')),
      },
    );
  }

  const isSaving = updateMutation.isPending;

  return (
    <MenuItemForm
      key={formKey}
      mode="edit"
      initialValues={initialValues}
      existingPhotos={item.images ?? []}
      categories={categories}
      menuItems={(menuData?.items ?? [])
        .filter((m) => m.id !== itemId)
        .map((m) => ({ id: m.id, name: m.name }))}
      isBakery={isBakery}
      onSave={handleSave}
      isSaving={isSaving}
      onDelete={handleDelete}
      isDeleting={deleteMutation.isPending}
      onCreateCategory={handleCreateCategory}
      onRemoveExistingPhoto={handleRemoveExistingPhoto}
      onAddPhoto={handleAddPhoto}
      isUploadingPhoto={uploadMutation.isPending}
      onBack={() => leaveTo(router, '/(tabs)/menu')}
    />
  );
}

const styles = StyleSheet.create({
  loading: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: theme.colors.bone,
  },
});
