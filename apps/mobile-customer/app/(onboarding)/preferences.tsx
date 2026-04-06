import { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { router, useLocalSearchParams } from 'expo-router';
import { useAuthStore } from '../../store/auth-store';
import { api } from '../../lib/api';

const CUISINE_OPTIONS = [
  'North Indian',
  'South Indian',
  'Chinese',
  'Continental',
  'Italian',
  'Healthy',
  'Street Food',
  'Desserts',
] as const;

export default function PreferencesScreen() {
  const [selected, setSelected] = useState<string[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const params = useLocalSearchParams<{
    firstName: string;
    lastName: string;
    phone: string;
    addressLine1: string;
    city: string;
    state: string;
    pincode: string;
  }>();

  const setOnboardingComplete = useAuthStore(
    (s) => s.setOnboardingComplete
  );

  const toggleChip = (cuisine: string) => {
    setSelected((prev) =>
      prev.includes(cuisine)
        ? prev.filter((c) => c !== cuisine)
        : [...prev, cuisine]
    );
  };

  const onFinish = async () => {
    setIsSubmitting(true);
    try {
      await api.post('/v1/customer/onboarding', {
        firstName: params.firstName,
        lastName: params.lastName,
        phone: params.phone,
        address: {
          addressLine1: params.addressLine1,
          city: params.city,
          state: params.state,
          pincode: params.pincode,
        },
        cuisinePreferences: selected,
      });

      await setOnboardingComplete(true);
      router.replace('/(tabs)');
    } catch (error: unknown) {
      const message =
        error instanceof Error ? error.message : 'Something went wrong. Please try again.';
      Alert.alert('Setup failed', message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <ScrollView
      style={styles.flex}
      contentContainerStyle={styles.container}
      keyboardShouldPersistTaps="handled"
    >
      {/* Progress */}
      <Text style={styles.stepLabel}>Step 3 of 3</Text>
      <View style={styles.progressBar}>
        <View style={[styles.progressFill, { width: '100%' }]} />
      </View>

      <Text style={styles.title}>What do you love to eat?</Text>
      <Text style={styles.subtitle}>
        Select your favourite cuisines to get personalised recommendations.
      </Text>

      {/* Cuisine chips */}
      <View style={styles.chipContainer}>
        {CUISINE_OPTIONS.map((cuisine) => {
          const isActive = selected.includes(cuisine);
          return (
            <TouchableOpacity
              key={cuisine}
              style={[styles.chip, isActive && styles.chipActive]}
              onPress={() => toggleChip(cuisine)}
              activeOpacity={0.8}
            >
              <Text
                style={[styles.chipText, isActive && styles.chipTextActive]}
              >
                {cuisine}
              </Text>
            </TouchableOpacity>
          );
        })}
      </View>

      <TouchableOpacity
        style={[styles.button, isSubmitting && styles.buttonDisabled]}
        onPress={onFinish}
        disabled={isSubmitting}
        activeOpacity={0.8}
      >
        {isSubmitting ? (
          <ActivityIndicator color="#FFFFFF" />
        ) : (
          <Text style={styles.buttonText}>Finish Setup</Text>
        )}
      </TouchableOpacity>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, backgroundColor: '#FFFFFF' },
  container: { padding: 24, paddingTop: 60 },
  stepLabel: { fontSize: 13, color: '#6B7280', marginBottom: 8 },
  progressBar: {
    height: 4,
    backgroundColor: '#E5E7EB',
    borderRadius: 2,
    marginBottom: 32,
  },
  progressFill: {
    height: 4,
    backgroundColor: '#FF6B35',
    borderRadius: 2,
  },
  title: { fontSize: 24, fontWeight: '700', color: '#111827', marginBottom: 8 },
  subtitle: { fontSize: 15, color: '#6B7280', marginBottom: 32 },
  chipContainer: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 10,
    marginBottom: 40,
  },
  chip: {
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderRadius: 20,
    borderWidth: 1.5,
    borderColor: '#D1D5DB',
    backgroundColor: '#F9FAFB',
  },
  chipActive: {
    backgroundColor: '#FFF3EE',
    borderColor: '#FF6B35',
  },
  chipText: { fontSize: 14, color: '#374151' },
  chipTextActive: { color: '#FF6B35', fontWeight: '600' },
  button: {
    height: 52,
    backgroundColor: '#FF6B35',
    borderRadius: 12,
    alignItems: 'center',
    justifyContent: 'center',
  },
  buttonDisabled: { opacity: 0.6 },
  buttonText: { fontSize: 16, fontWeight: '600', color: '#FFFFFF' },
});
