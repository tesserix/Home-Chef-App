import { useEffect, useState } from 'react';
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
import { router, useLocalSearchParams } from 'expo-router';
import { Image } from 'expo-image';
import * as ImagePicker from 'expo-image-picker';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Camera, Image as ImageIcon, ChevronLeft, Plus, Trash2 } from 'lucide-react-native';
import {
  useVendorMenu,
  useUpdateMenuItem,
  useUploadMenuPhoto,
} from '../../../hooks/useVendorMenu';
import { api } from '../../../lib/api';

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
  category: z.string().min(1, 'Please select a category'),
  isVeg: z.boolean(),
  preparationTime: z.number(),
});

type FormValues = z.infer<typeof schema>;

export default function EditMenuItemScreen() {
  const { itemId } = useLocalSearchParams<{ itemId: string }>();
  const { data: menuData } = useVendorMenu();
  const updateMutation = useUpdateMenuItem();
  const uploadMutation = useUploadMenuPhoto(itemId ?? '');

  const item = menuData?.items?.find((i) => i.id === itemId);
  const categories = menuData?.categories?.map((c) => c.name) ?? [];

  const [originalPrice, setOriginalPrice] = useState<string>('');
  const [showPriceChangeBanner, setShowPriceChangeBanner] = useState(false);

  const {
    control,
    handleSubmit,
    formState: { errors },
    watch,
    reset,
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      description: '',
      price: '',
      category: '',
      isVeg: true,
      preparationTime: 15,
    },
  });

  const selectedCategory = watch('category');
  const currentPrice = watch('price');

  useEffect(() => {
    if (item) {
      const priceStr = String(item.price);
      setOriginalPrice(priceStr);
      reset({
        name: item.name,
        description: item.description,
        price: priceStr,
        category: item.category,
        isVeg: item.isVeg,
        preparationTime: item.preparationTime,
      });
    }
  }, [item, reset]);

  useEffect(() => {
    if (originalPrice && currentPrice !== originalPrice) {
      setShowPriceChangeBanner(true);
    } else {
      setShowPriceChangeBanner(false);
    }
  }, [currentPrice, originalPrice]);

  async function handleAddPhoto() {
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      quality: 0.85,
      allowsEditing: true,
      aspect: [4, 3],
    });
    if (!result.canceled && result.assets[0]) {
      uploadMutation.mutate(result.assets[0].uri, {
        onError: () => Alert.alert('Error', 'Failed to upload photo. Please try again.'),
      });
    }
  }

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
      uploadMutation.mutate(result.assets[0].uri, {
        onError: () => Alert.alert('Error', 'Failed to upload photo. Please try again.'),
      });
    }
  }

  async function handleDeletePhoto(imageId: string) {
    Alert.alert('Delete Photo', 'Remove this photo?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Delete',
        style: 'destructive',
        onPress: async () => {
          try {
            await api.delete(`/chef/menu/items/${itemId}/images/${imageId}`);
          } catch {
            Alert.alert('Error', 'Failed to delete photo.');
          }
        },
      },
    ]);
  }

  async function onSubmit(values: FormValues) {
    if (!itemId) return;
    try {
      await updateMutation.mutateAsync({
        itemId,
        payload: {
          name: values.name,
          description: values.description,
          price: Number(values.price),
          category: values.category,
          isVeg: values.isVeg,
          preparationTime: values.preparationTime,
        },
      });
      router.back();
    } catch {
      Alert.alert('Error', 'Failed to update menu item. Please try again.');
    }
  }

  if (!item) {
    return (
      <SafeAreaView className="flex-1 bg-paper items-center justify-center">
        <ActivityIndicator size="large" color="#3e6b3c" />
      </SafeAreaView>
    );
  }

  const isSubmitting = updateMutation.isPending;
  const existingPhotos = item.images.slice(0, 5);

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
          <Text className="text-lg font-semibold text-ink">Edit Menu Item</Text>
        </View>

        <ScrollView
          className="flex-1"
          contentContainerStyle={{ padding: 16, paddingBottom: 40 }}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          {/* Photos */}
          <View className="bg-bone rounded-2xl p-4 mb-4 shadow-sm">
            <Text className="text-base font-semibold text-ink-soft mb-3">Photos</Text>
            <ScrollView horizontal showsHorizontalScrollIndicator={false}>
              <View className="flex-row gap-3">
                {existingPhotos.map((img) => (
                  <View key={img.id} className="relative">
                    <Image
                      source={{ uri: img.url }}
                      style={{ width: 90, height: 90, borderRadius: 12 }}
                      contentFit="cover"
                    />
                    <TouchableOpacity
                      onPress={() => handleDeletePhoto(img.id)}
                      className="absolute top-1 right-1 w-6 h-6 bg-paprika rounded-full items-center justify-center"
                    >
                      <Trash2 size={12} color="white" />
                    </TouchableOpacity>
                  </View>
                ))}
                {existingPhotos.length < 5 && (
                  <View className="gap-2">
                    <TouchableOpacity
                      onPress={handleTakePhoto}
                      className="w-[90px] h-[42px] bg-herb-tint border border-herb-tint rounded-xl items-center justify-center"
                      activeOpacity={0.7}
                    >
                      <Camera size={16} color="#3e6b3c" />
                    </TouchableOpacity>
                    <TouchableOpacity
                      onPress={handleAddPhoto}
                      className="w-[90px] h-[42px] bg-mist border border-mist rounded-xl items-center justify-center"
                      activeOpacity={0.7}
                    >
                      <Plus size={16} color="#7a7a76" />
                    </TouchableOpacity>
                  </View>
                )}
              </View>
            </ScrollView>
            {uploadMutation.isPending && (
              <View className="flex-row items-center gap-2 mt-2">
                <ActivityIndicator size="small" color="#3e6b3c" />
                <Text className="text-sm text-ink-muted">Uploading photo...</Text>
              </View>
            )}
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
                    multiline
                    numberOfLines={3}
                    textAlignVertical="top"
                    placeholderTextColor="#7a7a76"
                    className={`border rounded-xl px-4 py-3 text-base text-ink min-h-[80px] ${errors.description ? 'border-paprika' : 'border-mist'}`}
                  />
                )}
              />
              {errors.description && (
                <Text className="text-paprika text-xs mt-1">{errors.description.message}</Text>
              )}
            </View>

            {/* Price */}
            <View className="mb-1">
              <Text className="text-sm font-medium text-ink-soft mb-1">Price (₹) *</Text>
              <Controller
                control={control}
                name="price"
                render={({ field: { value, onChange, onBlur } }) => (
                  <TextInput
                    value={value}
                    onChangeText={onChange}
                    onBlur={onBlur}
                    keyboardType="decimal-pad"
                    placeholderTextColor="#7a7a76"
                    className={`border rounded-xl px-4 py-3 text-base text-ink ${errors.price ? 'border-paprika' : 'border-mist'}`}
                  />
                )}
              />
              {errors.price && (
                <Text className="text-paprika text-xs mt-1">{errors.price.message}</Text>
              )}
            </View>

            {/* Price change banner */}
            {showPriceChangeBanner && (
              <View className="bg-amber-tint border border-amber/30 rounded-xl px-4 py-3 mb-4">
                <Text className="text-amber text-sm">
                  Price changes are submitted for admin review and may take 24 hours to reflect.
                </Text>
              </View>
            )}

            {!showPriceChangeBanner && <View className="mb-4" />}

            {/* Category */}
            <View className="mb-4">
              <Text className="text-sm font-medium text-ink-soft mb-2">Category *</Text>
              <Controller
                control={control}
                name="category"
                render={({ field: { onChange } }) => (
                  <ScrollView horizontal showsHorizontalScrollIndicator={false}>
                    <View className="flex-row gap-2">
                      {categories.map((cat) => (
                        <TouchableOpacity
                          key={cat}
                          onPress={() => onChange(cat)}
                          className={`px-4 py-2 rounded-full border ${
                            selectedCategory === cat
                              ? 'bg-herb border-herb'
                              : 'bg-bone border-mist'
                          }`}
                          activeOpacity={0.7}
                        >
                          <Text
                            className={`text-sm font-medium ${
                              selectedCategory === cat ? 'text-paper' : 'text-ink-soft'
                            }`}
                          >
                            {cat}
                          </Text>
                        </TouchableOpacity>
                      ))}
                    </View>
                  </ScrollView>
                )}
              />
              {errors.category && (
                <Text className="text-paprika text-xs mt-1">{errors.category.message}</Text>
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
              <Text className="text-paper font-semibold text-base">Save Changes</Text>
            )}
          </TouchableOpacity>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
