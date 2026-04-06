import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StyleSheet,
} from 'react-native';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { router, useLocalSearchParams } from 'expo-router';

const schema = z.object({
  addressLine1: z.string().min(5, 'Address must be at least 5 characters'),
  city: z.string().min(1, 'City is required'),
  state: z.string().min(1, 'State is required'),
  pincode: z.string().regex(/^\d{6}$/, 'Enter a valid 6-digit pincode'),
});

type AddressForm = z.infer<typeof schema>;

export default function AddressScreen() {
  const params = useLocalSearchParams<{
    firstName: string;
    lastName: string;
    phone: string;
  }>();

  const {
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<AddressForm>({
    resolver: zodResolver(schema),
    defaultValues: { addressLine1: '', city: '', state: '', pincode: '' },
  });

  const onSubmit = (data: AddressForm) => {
    router.push({
      pathname: '/(onboarding)/preferences',
      params: {
        firstName: params.firstName,
        lastName: params.lastName,
        phone: params.phone,
        addressLine1: data.addressLine1,
        city: data.city,
        state: data.state,
        pincode: data.pincode,
      },
    });
  };

  return (
    <KeyboardAvoidingView
      style={styles.flex}
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
    >
      <ScrollView
        contentContainerStyle={styles.container}
        keyboardShouldPersistTaps="handled"
      >
        {/* Progress */}
        <Text style={styles.stepLabel}>Step 2 of 3</Text>
        <View style={styles.progressBar}>
          <View style={[styles.progressFill, { width: '66%' }]} />
        </View>

        <Text style={styles.title}>Your delivery address</Text>
        <Text style={styles.subtitle}>
          Where should we deliver your orders?
        </Text>

        {/* Address Line 1 */}
        <Text style={styles.label}>Address</Text>
        <Controller
          control={control}
          name="addressLine1"
          render={({ field: { onChange, onBlur, value } }) => (
            <TextInput
              style={[styles.input, errors.addressLine1 && styles.inputError]}
              placeholder="House no., street, area"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              autoCapitalize="words"
              returnKeyType="next"
            />
          )}
        />
        {errors.addressLine1 && (
          <Text style={styles.errorText}>{errors.addressLine1.message}</Text>
        )}

        {/* City */}
        <Text style={styles.label}>City</Text>
        <Controller
          control={control}
          name="city"
          render={({ field: { onChange, onBlur, value } }) => (
            <TextInput
              style={[styles.input, errors.city && styles.inputError]}
              placeholder="Enter your city"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              autoCapitalize="words"
              returnKeyType="next"
            />
          )}
        />
        {errors.city && (
          <Text style={styles.errorText}>{errors.city.message}</Text>
        )}

        {/* State */}
        <Text style={styles.label}>State</Text>
        <Controller
          control={control}
          name="state"
          render={({ field: { onChange, onBlur, value } }) => (
            <TextInput
              style={[styles.input, errors.state && styles.inputError]}
              placeholder="Enter your state"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              autoCapitalize="words"
              returnKeyType="next"
            />
          )}
        />
        {errors.state && (
          <Text style={styles.errorText}>{errors.state.message}</Text>
        )}

        {/* Pincode */}
        <Text style={styles.label}>Pincode</Text>
        <Controller
          control={control}
          name="pincode"
          render={({ field: { onChange, onBlur, value } }) => (
            <TextInput
              style={[styles.input, errors.pincode && styles.inputError]}
              placeholder="6-digit pincode"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              keyboardType="numeric"
              maxLength={6}
              returnKeyType="done"
            />
          )}
        />
        {errors.pincode && (
          <Text style={styles.errorText}>{errors.pincode.message}</Text>
        )}

        <TouchableOpacity
          style={styles.button}
          onPress={handleSubmit(onSubmit)}
          activeOpacity={0.8}
        >
          <Text style={styles.buttonText}>Continue</Text>
        </TouchableOpacity>
      </ScrollView>
    </KeyboardAvoidingView>
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
  label: { fontSize: 14, fontWeight: '500', color: '#374151', marginBottom: 6 },
  input: {
    height: 48,
    borderWidth: 1,
    borderColor: '#D1D5DB',
    borderRadius: 10,
    paddingHorizontal: 14,
    fontSize: 16,
    color: '#111827',
    marginBottom: 4,
    backgroundColor: '#F9FAFB',
  },
  inputError: { borderColor: '#EF4444' },
  errorText: { fontSize: 12, color: '#EF4444', marginBottom: 12 },
  button: {
    height: 52,
    backgroundColor: '#FF6B35',
    borderRadius: 12,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 32,
  },
  buttonText: { fontSize: 16, fontWeight: '600', color: '#FFFFFF' },
});
