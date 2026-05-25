import React, { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  SafeAreaView,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';
import { useRouter } from 'expo-router';
import { z } from 'zod';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useProfile, useUpdateProfile } from '../../hooks/useProfile';
import { useAuthStore } from '../../store/auth-store';

// Threat model T-02-05-01: Zod validates profile fields before PATCH
const profileSchema = z.object({
  firstName: z.string().min(1, 'First name is required').max(50),
  lastName: z.string().min(1, 'Last name is required').max(50),
  phone: z
    .string()
    .regex(/^\+?[0-9]{7,15}$/, 'Invalid phone number')
    .optional()
    .or(z.literal('')),
});

type ProfileFormValues = z.infer<typeof profileSchema>;

const CUISINE_OPTIONS = [
  'North Indian',
  'South Indian',
  'Chinese',
  'Continental',
  'Italian',
  'Healthy',
  'Desserts',
  'Street Food',
];

export default function ProfileScreen() {
  const router = useRouter();
  const { data, isLoading } = useProfile();
  const updateProfile = useUpdateProfile();
  const logout = useAuthStore((s) => s.logout);

  const profile = data?.data;

  const [cuisinePrefs, setCuisinePrefs] = useState<string[]>([]);

  const {
    control,
    handleSubmit,
    reset,
    formState: { errors, isDirty },
  } = useForm<ProfileFormValues>({
    resolver: zodResolver(profileSchema),
    defaultValues: { firstName: '', lastName: '', phone: '' },
  });

  // Populate form when profile loads
  useEffect(() => {
    if (profile) {
      reset({
        firstName: profile.firstName ?? '',
        lastName: profile.lastName ?? '',
        phone: profile.phone ?? '',
      });
      setCuisinePrefs(profile.cuisinePreferences ?? []);
    }
  }, [profile, reset]);

  function toggleCuisine(cuisine: string) {
    setCuisinePrefs((prev) =>
      prev.includes(cuisine)
        ? prev.filter((c) => c !== cuisine)
        : [...prev, cuisine],
    );
  }

  function onSavePersonalInfo(values: ProfileFormValues) {
    updateProfile.mutate(
      {
        firstName: values.firstName,
        lastName: values.lastName,
        phone: values.phone ?? undefined,
      },
      {
        onSuccess: () => Alert.alert('Saved', 'Profile updated successfully.'),
        onError: () =>
          Alert.alert('Error', 'Could not update profile. Please try again.'),
      },
    );
  }

  function saveCuisinePrefs() {
    updateProfile.mutate(
      { cuisinePreferences: cuisinePrefs },
      {
        onSuccess: () =>
          Alert.alert('Saved', 'Cuisine preferences updated.'),
        onError: () =>
          Alert.alert('Error', 'Could not save preferences.'),
      },
    );
  }

  function handleLogout() {
    Alert.alert('Log out', 'Are you sure you want to log out?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Log Out',
        style: 'destructive',
        onPress: () => {
          useAuthStore.getState().logout();
          router.replace('/(auth)/login');
        },
      },
    ]);
  }

  if (isLoading) {
    return (
      <SafeAreaView style={styles.container}>
        <View style={styles.centered}>
          <ActivityIndicator size="large" color="#C2410C" />
        </View>
      </SafeAreaView>
    );
  }

  // Initials for avatar placeholder
  const initials = [profile?.firstName?.[0], profile?.lastName?.[0]]
    .filter(Boolean)
    .join('')
    .toUpperCase() || '?';

  return (
    <SafeAreaView style={styles.container}>
      <ScrollView contentContainerStyle={styles.scrollContent}>
        {/* Avatar */}
        <View style={styles.avatarSection}>
          <View style={styles.avatar}>
            <Text style={styles.avatarInitials}>{initials}</Text>
          </View>
          <Text style={styles.emailDisplay}>{profile?.email ?? ''}</Text>
        </View>

        {/* Section 1 — Personal Info */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Personal Info</Text>

          <Text style={styles.label}>First Name</Text>
          <Controller
            control={control}
            name="firstName"
            render={({ field: { onChange, value, onBlur } }) => (
              <TextInput
                style={[styles.input, errors.firstName && styles.inputError]}
                value={value}
                onChangeText={onChange}
                onBlur={onBlur}
                placeholder="First name"
                autoCapitalize="words"
              />
            )}
          />
          {errors.firstName && (
            <Text style={styles.errorText}>{errors.firstName.message}</Text>
          )}

          <Text style={styles.label}>Last Name</Text>
          <Controller
            control={control}
            name="lastName"
            render={({ field: { onChange, value, onBlur } }) => (
              <TextInput
                style={[styles.input, errors.lastName && styles.inputError]}
                value={value}
                onChangeText={onChange}
                onBlur={onBlur}
                placeholder="Last name"
                autoCapitalize="words"
              />
            )}
          />
          {errors.lastName && (
            <Text style={styles.errorText}>{errors.lastName.message}</Text>
          )}

          <Text style={styles.label}>Phone</Text>
          <Controller
            control={control}
            name="phone"
            render={({ field: { onChange, value, onBlur } }) => (
              <TextInput
                style={[styles.input, errors.phone && styles.inputError]}
                value={value ?? ''}
                onChangeText={onChange}
                onBlur={onBlur}
                placeholder="+91 9876543210"
                keyboardType="phone-pad"
              />
            )}
          />
          {errors.phone && (
            <Text style={styles.errorText}>{errors.phone.message}</Text>
          )}

          {isDirty && (
            <TouchableOpacity
              style={styles.saveButton}
              onPress={() => void handleSubmit(onSavePersonalInfo)()}
              disabled={updateProfile.isPending}
            >
              {updateProfile.isPending ? (
                <ActivityIndicator size="small" color="#fafaf7" />
              ) : (
                <Text style={styles.saveButtonText}>Save Changes</Text>
              )}
            </TouchableOpacity>
          )}
        </View>

        {/* Section 2 — Food Preferences */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Food Preferences</Text>
          <View style={styles.chipGrid}>
            {CUISINE_OPTIONS.map((cuisine) => {
              const isSelected = cuisinePrefs.includes(cuisine);
              return (
                <TouchableOpacity
                  key={cuisine}
                  onPress={() => toggleCuisine(cuisine)}
                  style={[styles.chip, isSelected && styles.chipSelected]}
                >
                  <Text
                    style={[
                      styles.chipText,
                      isSelected && styles.chipTextSelected,
                    ]}
                  >
                    {cuisine}
                  </Text>
                </TouchableOpacity>
              );
            })}
          </View>
          <TouchableOpacity
            style={styles.saveButton}
            onPress={saveCuisinePrefs}
            disabled={updateProfile.isPending}
          >
            <Text style={styles.saveButtonText}>Save Preferences</Text>
          </TouchableOpacity>
        </View>

        {/* Section 3 — More */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>More</Text>
          <TouchableOpacity
            style={styles.moreRow}
            onPress={() => router.push('/social')}
          >
            <Text style={styles.moreRowIcon}>📱</Text>
            <Text style={styles.moreRowLabel}>Social Feed</Text>
            <Text style={styles.moreRowArrow}>›</Text>
          </TouchableOpacity>
          <View style={styles.divider} />
          <TouchableOpacity
            style={styles.moreRow}
            onPress={() => router.push('/catering')}
          >
            <Text style={styles.moreRowIcon}>🍽️</Text>
            <Text style={styles.moreRowLabel}>Catering</Text>
            <Text style={styles.moreRowArrow}>›</Text>
          </TouchableOpacity>
        </View>

        {/* Logout */}
        <TouchableOpacity
          style={styles.logoutButton}
          onPress={handleLogout}
        >
          <Text style={styles.logoutText}>Log Out</Text>
        </TouchableOpacity>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fafaf7',
  },
  centered: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  scrollContent: {
    paddingBottom: 40,
  },
  avatarSection: {
    alignItems: 'center',
    paddingTop: 24,
    paddingBottom: 16,
    backgroundColor: '#fafaf7',
    borderBottomWidth: 1,
    borderBottomColor: '#e6e5e0',
  },
  avatar: {
    width: 72,
    height: 72,
    borderRadius: 36,
    backgroundColor: '#C2410C',
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 8,
  },
  avatarInitials: {
    fontSize: 28,
    fontWeight: '700',
    color: '#fafaf7',
  },
  emailDisplay: {
    fontSize: 14,
    color: '#7a7a76',
  },
  section: {
    backgroundColor: '#fafaf7',
    marginTop: 12,
    paddingHorizontal: 16,
    paddingVertical: 16,
  },
  sectionTitle: {
    fontSize: 17,
    fontWeight: '700',
    color: '#1a1a18',
    marginBottom: 16,
  },
  label: {
    fontSize: 13,
    color: '#4a4a47',
    fontWeight: '500',
    marginBottom: 6,
    marginTop: 12,
  },
  input: {
    borderWidth: 1,
    borderColor: '#d4d3ce',
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 15,
    color: '#1a1a18',
    backgroundColor: '#fafaf7',
  },
  inputError: {
    borderColor: '#c95b3e',
  },
  errorText: {
    fontSize: 12,
    color: '#c95b3e',
    marginTop: 4,
  },
  saveButton: {
    marginTop: 16,
    backgroundColor: '#C2410C',
    paddingVertical: 12,
    borderRadius: 10,
    alignItems: 'center',
  },
  saveButtonText: {
    color: '#fafaf7',
    fontSize: 15,
    fontWeight: '600',
  },
  chipGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
    marginBottom: 8,
  },
  chip: {
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: 20,
    borderWidth: 1,
    borderColor: '#d4d3ce',
    backgroundColor: '#fafaf7',
  },
  chipSelected: {
    backgroundColor: '#FFEDD5',
    borderColor: '#C2410C',
  },
  chipText: {
    fontSize: 13,
    color: '#7a7a76',
  },
  chipTextSelected: {
    color: '#C2410C',
    fontWeight: '600',
  },
  moreRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 12,
    gap: 12,
  },
  moreRowIcon: {
    fontSize: 20,
  },
  moreRowLabel: {
    flex: 1,
    fontSize: 15,
    color: '#4a4a47',
    fontWeight: '500',
  },
  moreRowArrow: {
    fontSize: 20,
    color: '#7a7a76',
  },
  divider: {
    height: 1,
    backgroundColor: '#e6e5e0',
  },
  logoutButton: {
    margin: 16,
    marginTop: 24,
    paddingVertical: 14,
    borderRadius: 12,
    borderWidth: 1.5,
    borderColor: '#c95b3e',
    alignItems: 'center',
  },
  logoutText: {
    fontSize: 15,
    fontWeight: '600',
    color: '#c95b3e',
  },
});
