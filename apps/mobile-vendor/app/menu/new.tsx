import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { Image } from 'expo-image';
import * as ImagePicker from 'expo-image-picker';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Camera, Image as ImageIcon, ChevronLeft } from 'lucide-react-native';
import { useVendorMenu, useCreateMenuItem, useUploadMenuPhoto, useCreateCategory } from '../../hooks/useVendorMenu';

const PREP_TIME_OPTIONS = [5, 10, 15, 30, 45, 60];

const schema = z.object({
  name: z.string().min(3, 'Name must be at least 3 characters'),
  description: z.string().min(20, 'Description must be at least 20 characters'),
  price: z
    .string()
    .min(1, 'Price is required')
    .refine((v) => !isNaN(Number(v)) && Number(v) > 0 && Number(v) <= 10000, {
      message: 'Price must be between ₹1 and ₹10,000',
    }),
  categoryId: z.string().min(1, 'Please select a category'),
  isVeg: z.boolean(),
  preparationTime: z.number(),
});

type FormValues = z.infer<typeof schema>;

export default function NewMenuItemScreen() {
  const { data: menuData } = useVendorMenu();
  const createMutation = useCreateMenuItem();
  const createCategoryMutation = useCreateCategory();
  const [photoUri, setPhotoUri] = useState<string | null>(null);
  const [uploadItemId, setUploadItemId] = useState<string | null>(null);
  const uploadMutation = useUploadMenuPhoto(uploadItemId ?? '');

  const {
    control,
    handleSubmit,
    formState: { errors },
    watch,
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      description: '',
      price: '',
      categoryId: '',
      isVeg: true,
      preparationTime: 15,
    },
  });

  const selectedCategoryId = watch('categoryId');
  const [newCategoryName, setNewCategoryName] = useState('');
  const [showNewCategoryInput, setShowNewCategoryInput] = useState(false);

  async function handleTakePhoto() {
    const permission = await ImagePicker.requestCameraPermissionsAsync();
    if (!permission.granted) {
      Alert.alert('Permission Required', 'Camera permission is needed to take photos.');
      return;
    }
    const result = await ImagePicker.launchCameraAsync({
      mediaTypes: ['images'],
      quality: 0.85,
      allowsEditing: true,
      aspect: [4, 3],
    });
    if (!result.canceled && result.assets[0]) {
      setPhotoUri(result.assets[0].uri);
    }
  }

  async function handleChooseFromGallery() {
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      quality: 0.85,
      allowsEditing: true,
      aspect: [4, 3],
    });
    if (!result.canceled && result.assets[0]) {
      setPhotoUri(result.assets[0].uri);
    }
  }

  async function onSubmit(values: FormValues) {
    try {
      const result = await createMutation.mutateAsync({
        name: values.name,
        description: values.description,
        price: Number(values.price),
        categoryId: values.categoryId,
        isVeg: values.isVeg,
        preparationTime: values.preparationTime,
      });

      const newItemId = result.item.id;

      if (photoUri && newItemId) {
        setUploadItemId(newItemId);
        const formData = new FormData();
        formData.append('file', {
          uri: photoUri,
          name: 'menu-photo.jpg',
          type: 'image/jpeg',
        } as unknown as Blob);
        await uploadMutation.mutateAsync(photoUri);
      }

      router.back();
    } catch (error: unknown) {
      const serverError = (error as { response?: { data?: { error?: string } } } | null)
        ?.response?.data?.error;
      Alert.alert('Error', serverError ?? 'Failed to create menu item. Please try again.');
    }
  }

  const categories = menuData?.categories ?? [];
  const isSubmitting = createMutation.isPending || uploadMutation.isPending;

  return (
    <SafeAreaView className="flex-1 bg-paper">
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        className="flex-1"
      >
        {/* Header */}
        <View className="flex-row items-center px-4 pt-2 pb-3 bg-bone border-b border-mist">
          <TouchableOpacity accessibilityLabel="Go back" accessibilityRole="button" onPress={() => router.back()} activeOpacity={0.7} className="mr-3">
            <ChevronLeft size={24} color="#4a4a47" />
          </TouchableOpacity>
          <Text className="text-lg font-semibold text-ink">Add Menu Item</Text>
        </View>

        <ScrollView
          className="flex-1"
          contentContainerStyle={{ padding: 16, paddingBottom: 40 }}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          {/* Photo section */}
          <View className="bg-bone rounded-2xl p-4 mb-4 shadow-sm">
            <Text className="text-base font-semibold text-ink-soft mb-3">Food Photo</Text>
            {photoUri ? (
              <View className="rounded-xl overflow-hidden mb-3">
                <Image
                  source={{ uri: photoUri }}
                  style={{ width: '100%', height: 180 }}
                  contentFit="cover"
                />
              </View>
            ) : (
              <View className="w-full h-36 bg-mist rounded-xl items-center justify-center mb-3 border-2 border-dashed border-mist-strong">
                <ImageIcon size={32} color="#7a7a76" />
                <Text className="text-ink-muted text-sm mt-2">No photo selected</Text>
              </View>
            )}
            <View className="flex-row gap-3">
              <TouchableOpacity
                onPress={handleTakePhoto}
                className="flex-1 flex-row items-center justify-center gap-2 border border-herb rounded-xl py-3"
                activeOpacity={0.7}
              >
                <Camera size={18} color="#C2410C" />
                <Text className="text-herb font-medium text-sm">Take Photo</Text>
              </TouchableOpacity>
              <TouchableOpacity
                onPress={handleChooseFromGallery}
                className="flex-1 flex-row items-center justify-center gap-2 border border-mist-strong rounded-xl py-3"
                activeOpacity={0.7}
              >
                <ImageIcon size={18} color="#7a7a76" />
                <Text className="text-ink-soft font-medium text-sm">Gallery</Text>
              </TouchableOpacity>
            </View>
          </View>

          {/* Item details */}
          <View className="bg-bone rounded-2xl p-4 mb-4 shadow-sm">
            <Text className="text-base font-semibold text-ink-soft mb-4">Item Details</Text>

            {/* Name */}
            <View className="mb-4">
              <Text className="text-sm font-medium text-ink-soft mb-1">Item Name *</Text>
              <Controller
                control={control}
                name="name"
                render={({ field: { value, onChange, onBlur } }) => (
                  <TextInput
                    value={value}
                    onChangeText={onChange}
                    onBlur={onBlur}
                    placeholder="e.g. Butter Chicken"
                    placeholderTextColor="#7a7a76"
                    className={`border rounded-xl px-4 py-3 text-base text-ink ${errors.name ? 'border-paprika' : 'border-mist'}`}
                  />
                )}
              />
              {errors.name && (
                <Text className="text-paprika text-xs mt-1">{errors.name.message}</Text>
              )}
            </View>

            {/* Description */}
            <View className="mb-4">
              <Text className="text-sm font-medium text-ink-soft mb-1">Description *</Text>
              <Controller
                control={control}
                name="description"
                render={({ field: { value, onChange, onBlur } }) => (
                  <TextInput
                    value={value}
                    onChangeText={onChange}
                    onBlur={onBlur}
                    placeholder="Describe your dish (at least 20 characters)"
                    placeholderTextColor="#7a7a76"
                    multiline
                    numberOfLines={3}
                    textAlignVertical="top"
                    className={`border rounded-xl px-4 py-3 text-base text-ink min-h-[80px] ${errors.description ? 'border-paprika' : 'border-mist'}`}
                  />
                )}
              />
              {errors.description && (
                <Text className="text-paprika text-xs mt-1">{errors.description.message}</Text>
              )}
            </View>

            {/* Price */}
            <View className="mb-4">
              <Text className="text-sm font-medium text-ink-soft mb-1">Price (₹) *</Text>
              <Controller
                control={control}
                name="price"
                render={({ field: { value, onChange, onBlur } }) => (
                  <TextInput
                    value={value}
                    onChangeText={onChange}
                    onBlur={onBlur}
                    placeholder="0.00"
                    placeholderTextColor="#7a7a76"
                    keyboardType="decimal-pad"
                    className={`border rounded-xl px-4 py-3 text-base text-ink ${errors.price ? 'border-paprika' : 'border-mist'}`}
                  />
                )}
              />
              {errors.price && (
                <Text className="text-paprika text-xs mt-1">{errors.price.message}</Text>
              )}
            </View>

            {/* Category */}
            <View className="mb-4">
              <Text className="text-sm font-medium text-ink-soft mb-2">Category *</Text>
              <Controller
                control={control}
                name="categoryId"
                render={({ field: { onChange } }) => (
                  <>
                    <ScrollView
                      horizontal
                      showsHorizontalScrollIndicator={false}
                      style={{ flexGrow: 0, flexShrink: 0 }}
                      contentContainerStyle={{ alignItems: 'flex-start' }}
                    >
                      <View className="flex-row gap-2">
                        {categories.map((cat) => (
                          <TouchableOpacity
                            key={cat.id}
                            onPress={() => onChange(cat.id)}
                            className={`px-4 py-2 rounded-full border ${
                              selectedCategoryId === cat.id
                                ? 'bg-herb border-herb'
                                : 'bg-bone border-mist'
                            }`}
                            activeOpacity={0.7}
                          >
                            <Text
                              className={`text-sm font-medium ${
                                selectedCategoryId === cat.id ? 'text-paper' : 'text-ink-soft'
                              }`}
                            >
                              {cat.name}
                            </Text>
                          </TouchableOpacity>
                        ))}
                        <TouchableOpacity
                          onPress={() => setShowNewCategoryInput((v) => !v)}
                          className="px-4 py-2 rounded-full border border-dashed border-herb bg-bone"
                          activeOpacity={0.7}
                        >
                          <Text className="text-sm font-medium text-herb">+ New</Text>
                        </TouchableOpacity>
                      </View>
                    </ScrollView>

                    {showNewCategoryInput && (
                      <View className="flex-row gap-2 mt-3">
                        <TextInput
                          value={newCategoryName}
                          onChangeText={setNewCategoryName}
                          placeholder="Category name (e.g. Starters)"
                          placeholderTextColor="#7a7a76"
                          className="flex-1 border border-mist rounded-xl px-4 py-3 text-base text-ink"
                          autoCapitalize="words"
                          maxLength={40}
                        />
                        <TouchableOpacity
                          disabled={
                            createCategoryMutation.isPending || newCategoryName.trim().length < 2
                          }
                          onPress={async () => {
                            const name = newCategoryName.trim();
                            if (name.length < 2) return;
                            try {
                              const created = await createCategoryMutation.mutateAsync(name);
                              onChange(created.id);
                              setNewCategoryName('');
                              setShowNewCategoryInput(false);
                            } catch (error: unknown) {
                              const serverError = (error as { response?: { data?: { error?: string } } } | null)
                                ?.response?.data?.error;
                              Alert.alert(
                                'Could not add category',
                                serverError ?? 'Please try again.',
                              );
                            }
                          }}
                          className={`px-4 py-3 rounded-xl ${
                            createCategoryMutation.isPending || newCategoryName.trim().length < 2
                              ? 'bg-mist'
                              : 'bg-herb'
                          }`}
                        >
                          {createCategoryMutation.isPending ? (
                            <ActivityIndicator color="#fff" />
                          ) : (
                            <Text className="text-paper font-semibold">Add</Text>
                          )}
                        </TouchableOpacity>
                      </View>
                    )}
                  </>
                )}
              />
              {errors.categoryId && (
                <Text className="text-paprika text-xs mt-1">{errors.categoryId.message}</Text>
              )}
            </View>

            {/* Veg / Non-Veg */}
            <View className="mb-4">
              <Text className="text-sm font-medium text-ink-soft mb-2">Type</Text>
              <Controller
                control={control}
                name="isVeg"
                render={({ field: { value, onChange } }) => (
                  <View className="flex-row gap-3">
                    <TouchableOpacity
                      onPress={() => onChange(true)}
                      className={`flex-1 py-3 rounded-xl border items-center ${
                        value ? 'bg-herb border-herb' : 'bg-bone border-mist'
                      }`}
                      activeOpacity={0.7}
                    >
                      <Text
                        className={`font-medium text-sm ${value ? 'text-paper' : 'text-ink-soft'}`}
                      >
                        Veg
                      </Text>
                    </TouchableOpacity>
                    <TouchableOpacity
                      onPress={() => onChange(false)}
                      className={`flex-1 py-3 rounded-xl border items-center ${
                        !value ? 'bg-paprika border-paprika' : 'bg-bone border-mist'
                      }`}
                      activeOpacity={0.7}
                    >
                      <Text
                        className={`font-medium text-sm ${!value ? 'text-paper' : 'text-ink-soft'}`}
                      >
                        Non-Veg
                      </Text>
                    </TouchableOpacity>
                  </View>
                )}
              />
            </View>

            {/* Preparation Time */}
            <View>
              <Text className="text-sm font-medium text-ink-soft mb-2">Preparation Time</Text>
              <Controller
                control={control}
                name="preparationTime"
                render={({ field: { value, onChange } }) => (
                  <ScrollView horizontal showsHorizontalScrollIndicator={false}>
                    <View className="flex-row gap-2">
                      {PREP_TIME_OPTIONS.map((mins) => (
                        <TouchableOpacity
                          key={mins}
                          onPress={() => onChange(mins)}
                          className={`px-4 py-2 rounded-full border ${
                            value === mins
                              ? 'bg-herb border-herb'
                              : 'bg-bone border-mist'
                          }`}
                          activeOpacity={0.7}
                        >
                          <Text
                            className={`text-sm font-medium ${
                              value === mins ? 'text-paper' : 'text-ink-soft'
                            }`}
                          >
                            {mins} min
                          </Text>
                        </TouchableOpacity>
                      ))}
                    </View>
                  </ScrollView>
                )}
              />
            </View>
          </View>

          {/* Submit */}
          <TouchableOpacity
            onPress={handleSubmit(onSubmit)}
            disabled={isSubmitting}
            className={`py-4 rounded-2xl items-center ${isSubmitting ? 'bg-herb-soft' : 'bg-herb'}`}
            activeOpacity={0.85}
          >
            {isSubmitting ? (
              <ActivityIndicator color="white" />
            ) : (
              <Text className="text-paper font-semibold text-base">Add Item</Text>
            )}
          </TouchableOpacity>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
