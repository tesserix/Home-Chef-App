/**
 * NewMenuItemScreen — thin screen shell for creating a menu item.
 *
 * All visual logic lives in MenuItemForm. This screen:
 *  1. Wires the create + photo-upload mutations.
 *  2. Handles navigation (back + post-create).
 *  3. Provides the ChevronLeft back Pressable (form's backBtn fires router.back).
 */
import { Alert } from 'react-native';
import { router } from 'expo-router';
import { getServerErrorMessage } from '@homechef/mobile-shared/api';
import { useToast } from '@homechef/mobile-shared/ui';
import {
  useVendorMenu,
  useCreateMenuItem,
  useUploadMenuPhoto,
  useCreateCategory,
} from '../../hooks/useVendorMenu';
import { MenuItemForm } from './MenuItemForm';
import type { MenuItemFormValues } from './MenuItemForm';
import { useState } from 'react';

const BLANK: MenuItemFormValues = {
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
  preparationTime: 15,
  hsn: '',
};

export default function NewMenuItemScreen() {
  const { data: menuData } = useVendorMenu();
  const createMutation = useCreateMenuItem();
  const createCategoryMutation = useCreateCategory();
  const { show: showToast } = useToast();

  // The newly-created item's ID is needed for photo upload. We hold it in a
  // ref so we can construct the upload hook once at the top of the component
  // without violating the rules of hooks.
  const [newItemId, setNewItemId] = useState<string>('');
  const uploadMutation = useUploadMenuPhoto(newItemId);

  const categories = menuData?.categories ?? [];
  const isSaving = createMutation.isPending || uploadMutation.isPending;

  async function handleSave(values: MenuItemFormValues, localPhotoUris: string[]) {
    try {
      const result = await createMutation.mutateAsync({
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
        preparationTime: values.preparationTime,
        hsn: values.hsn || undefined,
      });

      const createdId = result.item.id;
      setNewItemId(createdId);

      // Upload queued photos sequentially
      for (const uri of localPhotoUris) {
        try {
          await uploadMutation.mutateAsync(uri);
        } catch {
          // Photo upload failure is non-blocking — item already created.
          // A later session allows adding photos from the edit screen.
        }
      }

      showToast({ message: `${values.name} added to menu`, tone: 'success' });
      router.back();
    } catch (err: unknown) {
      Alert.alert(
        'Could not add item',
        getServerErrorMessage(err, 'Please check your details and try again.'),
      );
    }
  }

  async function handleCreateCategory(name: string) {
    return createCategoryMutation.mutateAsync(name);
  }

  return (
    <MenuItemForm
      mode="new"
      initialValues={BLANK}
      categories={categories}
      menuItems={(menuData?.items ?? []).map((m) => ({ id: m.id, name: m.name }))}
      onSave={handleSave}
      isSaving={isSaving}
      onCreateCategory={handleCreateCategory}
      onBack={() => router.back()}
    />
  );
}
