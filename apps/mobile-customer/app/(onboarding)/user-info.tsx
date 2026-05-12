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
import { router } from 'expo-router';

const schema = z.object({
  firstName: z.string().min(2, 'First name must be at least 2 characters'),
  lastName: z.string().min(2, 'Last name must be at least 2 characters'),
  phone: z
    .string()
    .regex(/^[6-9]\d{9}$/, 'Enter a valid 10-digit Indian mobile number'),
});

type UserInfoForm = z.infer<typeof schema>;

export default function UserInfoScreen() {
  const {
    control,
    handleSubmit,
    formState: { errors },
  } = useForm<UserInfoForm>({
    resolver: zodResolver(schema),
    defaultValues: { firstName: '', lastName: '', phone: '' },
  });

  const onSubmit = (data: UserInfoForm) => {
    router.push({
      pathname: '/(onboarding)/address',
      params: {
        firstName: data.firstName,
        lastName: data.lastName,
        phone: data.phone,
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
        <Text style={styles.stepLabel}>Step 1 of 3</Text>
        <View style={styles.progressBar}>
          <View style={[styles.progressFill, { width: '33%' }]} />
        </View>

        <Text style={styles.title}>Tell us about yourself</Text>
        <Text style={styles.subtitle}>
          We need a few details to set up your account.
        </Text>

        {/* First Name */}
        <Text style={styles.label}>First Name</Text>
        <Controller
          control={control}
          name="firstName"
          render={({ field: { onChange, onBlur, value } }) => (
            <TextInput
              style={[styles.input, errors.firstName && styles.inputError]}
              placeholder="Enter your first name"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              autoCapitalize="words"
              returnKeyType="next"
            />
          )}
        />
        {errors.firstName && (
          <Text style={styles.errorText}>{errors.firstName.message}</Text>
        )}

        {/* Last Name */}
        <Text style={styles.label}>Last Name</Text>
        <Controller
          control={control}
          name="lastName"
          render={({ field: { onChange, onBlur, value } }) => (
            <TextInput
              style={[styles.input, errors.lastName && styles.inputError]}
              placeholder="Enter your last name"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              autoCapitalize="words"
              returnKeyType="next"
            />
          )}
        />
        {errors.lastName && (
          <Text style={styles.errorText}>{errors.lastName.message}</Text>
        )}

        {/* Phone */}
        <Text style={styles.label}>Phone Number</Text>
        <Controller
          control={control}
          name="phone"
          render={({ field: { onChange, onBlur, value } }) => (
            <TextInput
              style={[styles.input, errors.phone && styles.inputError]}
              placeholder="10-digit mobile number"
              onBlur={onBlur}
              onChangeText={onChange}
              value={value}
              keyboardType="phone-pad"
              maxLength={10}
              returnKeyType="done"
            />
          )}
        />
        {errors.phone && (
          <Text style={styles.errorText}>{errors.phone.message}</Text>
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
  flex: { flex: 1, backgroundColor: '#fafaf7' },
  container: { padding: 24, paddingTop: 60 },
  stepLabel: { fontSize: 13, color: '#7a7a76', marginBottom: 8 },
  progressBar: {
    height: 4,
    backgroundColor: '#e6e5e0',
    borderRadius: 2,
    marginBottom: 32,
  },
  progressFill: {
    height: 4,
    backgroundColor: '#3e6b3c',
    borderRadius: 2,
  },
  title: { fontSize: 24, fontWeight: '700', color: '#1a1a18', marginBottom: 8 },
  subtitle: { fontSize: 15, color: '#7a7a76', marginBottom: 32 },
  label: { fontSize: 14, fontWeight: '500', color: '#4a4a47', marginBottom: 6 },
  input: {
    height: 48,
    borderWidth: 1,
    borderColor: '#d4d3ce',
    borderRadius: 10,
    paddingHorizontal: 14,
    fontSize: 16,
    color: '#1a1a18',
    marginBottom: 4,
    backgroundColor: '#fafaf7',
  },
  inputError: { borderColor: '#c95b3e' },
  errorText: { fontSize: 12, color: '#c95b3e', marginBottom: 12 },
  button: {
    height: 52,
    backgroundColor: '#3e6b3c',
    borderRadius: 12,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 32,
  },
  buttonText: { fontSize: 16, fontWeight: '600', color: '#fafaf7' },
});
