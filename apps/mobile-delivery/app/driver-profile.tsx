import { useState } from 'react';
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
import * as ImagePicker from 'expo-image-picker';
import { Image } from 'expo-image';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Camera, User } from 'lucide-react-native';
import { api } from '../lib/api';

interface DriverProfile {
  id: string;
  name: string;
  email: string;
  phone: string;
  city: string;
  vehicleType: string;
  vehicleNumber: string;
  rating: number;
  totalDeliveries: number;
  isOnline: boolean;
  isVerified: boolean;
  profileImageUrl?: string;
}

interface UpdateProfilePayload {
  name: string;
  phone: string;
  city: string;
}

function useDriverProfile() {
  return useQuery<DriverProfile>({
    queryKey: ['driver', 'profile'],
    queryFn: () => api.get('/delivery/profile').then((r) => r.data as DriverProfile),
  });
}

function useUpdateProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: UpdateProfilePayload) =>
      api.put('/delivery/profile', payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['driver', 'profile'] });
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
      return api.post('/delivery/profile-image', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['driver', 'profile'] });
    },
  });
}

function StatBadge({ label, value }: { label: string; value: string }) {
  return (
    <View className="flex-1 items-center">
      <Text className="text-xl font-semibold text-ink">{value}</Text>
      <Text className="text-xs text-ink-muted mt-0.5">{label}</Text>
    </View>
  );
}

export default function DriverProfileScreen() {
  const { data: profile, isLoading, isError, refetch, isRefetching } = useDriverProfile();
  const updateMutation = useUpdateProfile();
  const uploadImageMutation = useUploadProfileImage();

  const [name, setName] = useState('');
  const [phone, setPhone] = useState('');
  const [city, setCity] = useState('');
  const [isEditing, setIsEditing] = useState(false);

  function initEditFields() {
    setName(profile?.name ?? '');
    setPhone(profile?.phone ?? '');
    setCity(profile?.city ?? '');
    setIsEditing(true);
  }

  function handleSave() {
    if (!name.trim() || !phone.trim() || !city.trim()) {
      Alert.alert('Validation', 'Name, phone, and city are required.');
      return;
    }
    updateMutation.mutate(
      { name: name.trim(), phone: phone.trim(), city: city.trim() },
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

  async function handlePickImage() {
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ImagePicker.MediaTypeOptions.Images,
      allowsEditing: true,
      aspect: [1, 1],
      quality: 0.8,
    });
    if (!result.canceled && result.assets[0]) {
      const asset = result.assets[0];
      uploadImageMutation.mutate(asset.uri, {
        onError: () => Alert.alert('Error', 'Failed to upload photo. Please try again.'),
      });
    }
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
      <ScrollView
        className="flex-1"
        refreshControl={
          <RefreshControl refreshing={isRefetching} onRefresh={refetch} tintColor="#3e6b3c" />
        }
        contentContainerStyle={{ paddingBottom: 32 }}
      >
        <View className="px-4 pt-4 pb-2">
          <Text className="font-display text-2xl font-semibold text-ink">Profile</Text>
        </View>

        {/* Avatar */}
        <View className="items-center mt-4 mb-6">
          <TouchableOpacity
            onPress={handlePickImage}
            disabled={uploadImageMutation.isPending}
            activeOpacity={0.8}
          >
            <View className="w-24 h-24 rounded-full overflow-hidden bg-herb-tint items-center justify-center border-2 border-herb-tint">
              {profile?.profileImageUrl ? (
                <Image
                  source={{ uri: profile.profileImageUrl }}
                  style={{ width: 96, height: 96 }}
                  contentFit="cover"
                />
              ) : (
                <User size={40} color="#558257" />
              )}
            </View>
            <View className="absolute bottom-0 right-0 w-8 h-8 rounded-full bg-herb items-center justify-center border-2 border-bone">
              {uploadImageMutation.isPending ? (
                <ActivityIndicator size="small" color="white" />
              ) : (
                <Camera size={14} color="white" />
              )}
            </View>
          </TouchableOpacity>
        </View>

        {/* Stats row */}
        <View className="mx-4 bg-bone rounded-2xl p-4 shadow-sm flex-row mb-4">
          <StatBadge label="Rating" value={`${(profile?.rating ?? 0).toFixed(1)} \u2B50`} />
          <View className="w-px bg-mist mx-2" />
          <StatBadge label="Deliveries" value={String(profile?.totalDeliveries ?? 0)} />
          <View className="w-px bg-mist mx-2" />
          <StatBadge
            label="Status"
            value={profile?.isVerified ? '\u2714 Verified' : 'Pending'}
          />
        </View>

        {/* Editable fields */}
        <View className="mx-4 bg-bone rounded-2xl p-4 shadow-sm mb-4">
          <View className="flex-row items-center justify-between mb-4">
            <Text className="text-base font-semibold text-ink-soft">Personal Info</Text>
            {!isEditing && (
              <TouchableOpacity onPress={initEditFields} activeOpacity={0.7}>
                <Text className="text-sm text-herb font-medium">Edit</Text>
              </TouchableOpacity>
            )}
          </View>

          <View className="mb-3">
            <Text className="text-xs text-ink-muted mb-1">Full Name</Text>
            {isEditing ? (
              <TextInput
                value={name}
                onChangeText={setName}
                className="border border-mist-strong rounded-xl px-4 py-3 text-base text-ink"
                placeholderTextColor="#7a7a76"
              />
            ) : (
              <Text className="text-base text-ink">{profile?.name ?? ''}</Text>
            )}
          </View>

          <View className="mb-3">
            <Text className="text-xs text-ink-muted mb-1">Phone</Text>
            {isEditing ? (
              <TextInput
                value={phone}
                onChangeText={setPhone}
                keyboardType="phone-pad"
                className="border border-mist-strong rounded-xl px-4 py-3 text-base text-ink"
                placeholderTextColor="#7a7a76"
              />
            ) : (
              <Text className="text-base text-ink">{profile?.phone ?? ''}</Text>
            )}
          </View>

          <View>
            <Text className="text-xs text-ink-muted mb-1">City</Text>
            {isEditing ? (
              <TextInput
                value={city}
                onChangeText={setCity}
                className="border border-mist-strong rounded-xl px-4 py-3 text-base text-ink"
                placeholderTextColor="#7a7a76"
              />
            ) : (
              <Text className="text-base text-ink">{profile?.city ?? ''}</Text>
            )}
          </View>
        </View>

        {/* Read-only vehicle info */}
        <View className="mx-4 bg-bone rounded-2xl p-4 shadow-sm mb-4">
          <Text className="text-base font-semibold text-ink-soft mb-4">Vehicle Details</Text>
          <View className="mb-3">
            <Text className="text-xs text-ink-muted mb-1">Vehicle Type</Text>
            <Text className="text-base text-ink capitalize">
              {profile?.vehicleType ?? '\u2014'}
            </Text>
          </View>
          <View>
            <Text className="text-xs text-ink-muted mb-1">Registration Number</Text>
            <Text className="text-base text-ink uppercase">
              {profile?.vehicleNumber ?? '\u2014'}
            </Text>
          </View>
        </View>

        {/* Account info */}
        <View className="mx-4 bg-bone rounded-2xl p-4 shadow-sm">
          <Text className="text-base font-semibold text-ink-soft mb-3">Account</Text>
          <Text className="text-xs text-ink-muted mb-1">Email</Text>
          <Text className="text-base text-ink">{profile?.email ?? ''}</Text>
        </View>

        {/* Save button when editing */}
        {isEditing && (
          <View className="mx-4 mt-4 flex-row gap-3">
            <TouchableOpacity
              onPress={() => setIsEditing(false)}
              className="flex-1 py-4 rounded-2xl items-center border border-mist-strong"
              activeOpacity={0.8}
            >
              <Text className="text-ink-soft font-semibold">Cancel</Text>
            </TouchableOpacity>
            <TouchableOpacity
              onPress={handleSave}
              disabled={updateMutation.isPending}
              className={`flex-1 py-4 rounded-2xl items-center ${
                updateMutation.isPending ? 'bg-herb-soft' : 'bg-herb'
              }`}
              activeOpacity={0.8}
            >
              {updateMutation.isPending ? (
                <ActivityIndicator color="white" />
              ) : (
                <Text className="text-paper font-semibold">Save Changes</Text>
              )}
            </TouchableOpacity>
          </View>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}
