import { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  RefreshControl,
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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Camera, Plus, User, ChevronLeft } from 'lucide-react-native';
import { api } from '../lib/api';

interface ChefProfile {
  id: string;
  displayName: string;
  bio: string;
  phone: string;
  kitchenName: string;
  cuisineTypes: string[];
  profileImageUrl?: string;
  kitchenPhotos: { id: string; url: string }[];
}

interface UpdateProfilePayload {
  displayName: string;
  bio: string;
  phone: string;
}

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
    mutationFn: (payload: UpdateProfilePayload) => api.put('/chef/profile', payload),
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
      return api.post('/chef/profile-image', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
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
      return api.post('/chef/kitchen-photos', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef', 'profile'] });
    },
  });
}

export default function ProfileScreen() {
  const { data, isLoading, isError, refetch, isRefetching } = useChefProfile();
  const updateMutation = useUpdateProfile();
  const uploadProfileImageMutation = useUploadProfileImage();
  const uploadKitchenPhotoMutation = useUploadKitchenPhoto();

  const [displayName, setDisplayName] = useState('');
  const [bio, setBio] = useState('');
  const [phone, setPhone] = useState('');
  const [isEditing, setIsEditing] = useState(false);

  useEffect(() => {
    if (data) {
      setDisplayName(data.displayName);
      setBio(data.bio);
      setPhone(data.phone);
    }
  }, [data]);

  async function handlePickProfileImage() {
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      allowsEditing: true,
      aspect: [1, 1],
      quality: 0.85,
    });
    if (!result.canceled && result.assets[0]) {
      uploadProfileImageMutation.mutate(result.assets[0].uri, {
        onError: () => Alert.alert('Error', 'Failed to upload profile photo.'),
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
        onError: () => Alert.alert('Error', 'Failed to upload kitchen photo.'),
      });
    }
  }

  function handleSave() {
    if (!displayName.trim()) {
      Alert.alert('Validation', 'Display name is required.');
      return;
    }
    updateMutation.mutate(
      { displayName: displayName.trim(), bio: bio.trim(), phone: phone.trim() },
      {
        onSuccess: () => {
          setIsEditing(false);
          Alert.alert('Success', 'Profile updated successfully.');
        },
        onError: () => {
          Alert.alert('Error', 'Failed to update profile. Please try again.');
        },
      },
    );
  }

  if (isLoading) {
    return (
      <SafeAreaView className="flex-1 bg-paper items-center justify-center">
        <ActivityIndicator size="large" color="#3e6b3c" />
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView className="flex-1 bg-paper items-center justify-center px-6">
        <Text className="text-ink-muted text-base mb-4">Failed to load profile</Text>
        <TouchableOpacity
          onPress={() => refetch()}
          className="bg-herb px-6 py-3 rounded-xl"
        >
          <Text className="text-paper font-semibold">Retry</Text>
        </TouchableOpacity>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-paper">
      {/* Header */}
      <View className="flex-row items-center px-4 pt-2 pb-3 bg-bone border-b border-mist">
        <TouchableOpacity accessibilityLabel="Go back" accessibilityRole="button" onPress={() => router.back()} activeOpacity={0.7} className="mr-3">
          <ChevronLeft size={24} color="#4a4a47" />
        </TouchableOpacity>
        <Text className="text-lg font-semibold text-ink">Profile</Text>
      </View>

      <ScrollView
        className="flex-1"
        contentContainerStyle={{ padding: 16, paddingBottom: 40 }}
        refreshControl={
          <RefreshControl refreshing={isRefetching} onRefresh={refetch} tintColor="#3e6b3c" />
        }
        showsVerticalScrollIndicator={false}
      >
        {/* Profile photo */}
        <View className="items-center mb-6">
          <TouchableOpacity
            onPress={handlePickProfileImage}
            disabled={uploadProfileImageMutation.isPending}
            activeOpacity={0.8}
          >
            <View className="w-24 h-24 rounded-full overflow-hidden bg-herb-tint items-center justify-center border-2 border-herb-tint">
              {data?.profileImageUrl ? (
                <Image
                  source={{ uri: data.profileImageUrl }}
                  style={{ width: 96, height: 96 }}
                  contentFit="cover"
                />
              ) : (
                <User size={40} color="#558257" />
              )}
            </View>
            <View className="absolute bottom-0 right-0 w-8 h-8 rounded-full bg-herb items-center justify-center border-2 border-bone">
              {uploadProfileImageMutation.isPending ? (
                <ActivityIndicator size="small" color="white" />
              ) : (
                <Camera size={14} color="white" />
              )}
            </View>
          </TouchableOpacity>
        </View>

        {/* Basic info */}
        <View className="bg-bone rounded-2xl shadow-sm p-4 mb-4">
          <View className="flex-row items-center justify-between mb-4">
            <Text className="text-base font-semibold text-ink-soft">Personal Info</Text>
            {!isEditing && (
              <TouchableOpacity
                onPress={() => setIsEditing(true)}
                activeOpacity={0.7}
              >
                <Text className="text-sm text-herb font-medium">Edit</Text>
              </TouchableOpacity>
            )}
          </View>

          <View className="mb-3">
            <Text className="text-xs text-ink-muted mb-1">Display Name</Text>
            {isEditing ? (
              <TextInput
                value={displayName}
                onChangeText={setDisplayName}
                className="border border-mist rounded-xl px-4 py-3 text-base text-ink"
                placeholderTextColor="#7a7a76"
              />
            ) : (
              <Text className="text-base text-ink">{data?.displayName ?? ''}</Text>
            )}
          </View>

          <View className="mb-3">
            <Text className="text-xs text-ink-muted mb-1">Bio</Text>
            {isEditing ? (
              <TextInput
                value={bio}
                onChangeText={setBio}
                multiline
                numberOfLines={3}
                textAlignVertical="top"
                className="border border-mist rounded-xl px-4 py-3 text-base text-ink min-h-[80px]"
                placeholderTextColor="#7a7a76"
              />
            ) : (
              <Text className="text-base text-ink">{data?.bio ?? ''}</Text>
            )}
          </View>

          <View>
            <Text className="text-xs text-ink-muted mb-1">Phone</Text>
            {isEditing ? (
              <TextInput
                value={phone}
                onChangeText={setPhone}
                keyboardType="phone-pad"
                className="border border-mist rounded-xl px-4 py-3 text-base text-ink"
                placeholderTextColor="#7a7a76"
              />
            ) : (
              <Text className="text-base text-ink">{data?.phone ?? ''}</Text>
            )}
          </View>

          {isEditing && (
            <View className="flex-row gap-3 mt-4">
              <TouchableOpacity
                onPress={() => setIsEditing(false)}
                className="flex-1 py-3 rounded-xl items-center border border-mist-strong"
                activeOpacity={0.8}
              >
                <Text className="text-ink-soft font-semibold">Cancel</Text>
              </TouchableOpacity>
              <TouchableOpacity
                onPress={handleSave}
                disabled={updateMutation.isPending}
                className={`flex-1 py-3 rounded-xl items-center ${
                  updateMutation.isPending ? 'bg-herb-soft' : 'bg-herb'
                }`}
                activeOpacity={0.8}
              >
                {updateMutation.isPending ? (
                  <ActivityIndicator color="white" />
                ) : (
                  <Text className="text-paper font-semibold">Save</Text>
                )}
              </TouchableOpacity>
            </View>
          )}
        </View>

        {/* Kitchen info (read-only) */}
        <View className="bg-bone rounded-2xl shadow-sm p-4 mb-4">
          <Text className="text-base font-semibold text-ink-soft mb-4">Kitchen Info</Text>

          <View className="mb-3">
            <Text className="text-xs text-ink-muted mb-1">Kitchen Name</Text>
            <Text className="text-base text-ink">{data?.kitchenName ?? '—'}</Text>
          </View>

          <View>
            <Text className="text-xs text-ink-muted mb-1">Cuisine Types</Text>
            <View className="flex-row flex-wrap gap-2 mt-1">
              {data?.cuisineTypes?.map((cuisine) => (
                <View key={cuisine} className="bg-herb-tint px-3 py-1 rounded-full">
                  <Text className="text-xs text-herb">{cuisine}</Text>
                </View>
              ))}
            </View>
          </View>
        </View>

        {/* Kitchen photos */}
        <View className="bg-bone rounded-2xl shadow-sm p-4">
          <Text className="text-base font-semibold text-ink-soft mb-3">Kitchen Photos</Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false}>
            <View className="flex-row gap-3">
              {data?.kitchenPhotos?.slice(0, 5).map((photo) => (
                <Image
                  key={photo.id}
                  source={{ uri: photo.url }}
                  style={{ width: 90, height: 90, borderRadius: 12 }}
                  contentFit="cover"
                />
              ))}
              {(data?.kitchenPhotos?.length ?? 0) < 5 && (
                <TouchableOpacity
                  onPress={handleAddKitchenPhoto}
                  disabled={uploadKitchenPhotoMutation.isPending}
                  className="w-[90px] h-[90px] bg-mist border-2 border-dashed border-mist-strong rounded-xl items-center justify-center"
                  activeOpacity={0.7}
                >
                  {uploadKitchenPhotoMutation.isPending ? (
                    <ActivityIndicator size="small" color="#3e6b3c" />
                  ) : (
                    <Plus size={24} color="#7a7a76" />
                  )}
                </TouchableOpacity>
              )}
            </View>
          </ScrollView>
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
