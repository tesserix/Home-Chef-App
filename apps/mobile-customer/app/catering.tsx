// Catering screen — endpoints confirmed from apps/api/handlers/catering.go
// POST /v1/catering/requests  → create request
// GET  /v1/catering/requests  → list my requests

import React, { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  FlatList,
  SafeAreaView,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';
import { z } from 'zod';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  useCateringRequests,
  useCreateCateringRequest,
} from '../hooks/useCatering';
import type { CateringRequest } from '../hooks/useCatering';

// Threat model T-02-05-02: Zod validates required fields before POST
const cateringSchema = z.object({
  eventType: z.string().min(1, 'Event type is required'),
  eventDate: z
    .string()
    .regex(/^\d{4}-\d{2}-\d{2}$/, 'Date must be YYYY-MM-DD')
    .refine((d) => new Date(d) > new Date(), 'Event date must be in the future'),
  guestCount: z
    .number({ invalid_type_error: 'Guest count must be a number' })
    .int()
    .min(1, 'At least 1 guest required'),
  budget: z
    .number({ invalid_type_error: 'Budget must be a number' })
    .positive('Budget must be positive')
    .optional(),
  city: z.string().min(1, 'City is required'),
  state: z.string().min(1, 'State is required'),
  description: z.string().optional(),
});

type CateringFormValues = z.infer<typeof cateringSchema>;

const EVENT_TYPES = [
  'Wedding',
  'Birthday',
  'Corporate',
  'Anniversary',
  'Festival',
  'House Party',
  'Other',
];

type TabKey = 'request' | 'my-requests';

function statusLabel(status: CateringRequest['status']): {
  label: string;
  color: string;
} {
  switch (status) {
    case 'open':
      return { label: 'Open', color: '#4a73a3' };
    case 'quoted':
      return { label: 'Quoted', color: '#d1a64a' };
    case 'accepted':
      return { label: 'Accepted', color: '#C2410C' };
    case 'completed':
      return { label: 'Completed', color: '#7a7a76' };
    case 'cancelled':
      return { label: 'Cancelled', color: '#c95b3e' };
    default:
      return { label: status, color: '#7a7a76' };
  }
}

function RequestCard({ request }: { request: CateringRequest }) {
  const { label, color } = statusLabel(request.status);
  return (
    <View style={styles.requestCard}>
      <View style={styles.requestCardHeader}>
        <Text style={styles.requestEventType}>{request.eventType}</Text>
        <View style={[styles.statusDot, { backgroundColor: color }]}>
          <Text style={styles.statusDotText}>{label}</Text>
        </View>
      </View>
      <Text style={styles.requestMeta}>
        {request.guestCount} guests • {request.eventDate} • {request.city},{' '}
        {request.state}
      </Text>
      {request.budget != null && (
        <Text style={styles.requestBudget}>Budget: ₹{request.budget}</Text>
      )}
      {request.status === 'quoted' && (
        <Text style={styles.viewQuoteHint}>Quotes available — view details</Text>
      )}
    </View>
  );
}

function RequestForm({ onSuccess }: { onSuccess: () => void }) {
  const createRequest = useCreateCateringRequest();
  const [selectedEventType, setSelectedEventType] = useState('');

  const {
    control,
    handleSubmit,
    setValue,
    reset,
    formState: { errors },
  } = useForm<CateringFormValues>({
    resolver: zodResolver(cateringSchema),
    defaultValues: {
      eventType: '',
      eventDate: '',
      guestCount: undefined,
      budget: undefined,
      city: '',
      state: '',
      description: '',
    },
  });

  function selectEventType(t: string) {
    setSelectedEventType(t);
    setValue('eventType', t, { shouldValidate: true });
  }

  function onSubmit(values: CateringFormValues) {
    createRequest.mutate(
      {
        eventType: values.eventType,
        eventDate: values.eventDate,
        guestCount: values.guestCount,
        budget: values.budget,
        city: values.city,
        state: values.state,
        description: values.description,
      },
      {
        onSuccess: () => {
          Alert.alert(
            'Request Submitted!',
            'Chefs will review and send quotes.',
          );
          reset();
          setSelectedEventType('');
          onSuccess();
        },
        onError: (_err) => {
          Alert.alert('Error', 'Could not submit request. Please try again.');
        },
      },
    );
  }

  return (
    <ScrollView contentContainerStyle={styles.formScroll}>
      {/* Event Type */}
      <Text style={styles.formLabel}>Event Type *</Text>
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={styles.eventTypeRow}
      >
        {EVENT_TYPES.map((t) => (
          <TouchableOpacity
            key={t}
            onPress={() => selectEventType(t)}
            style={[
              styles.eventTypeChip,
              selectedEventType === t && styles.eventTypeChipSelected,
            ]}
          >
            <Text
              style={[
                styles.eventTypeChipText,
                selectedEventType === t && styles.eventTypeChipTextSelected,
              ]}
            >
              {t}
            </Text>
          </TouchableOpacity>
        ))}
      </ScrollView>
      {errors.eventType && (
        <Text style={styles.errorText}>{errors.eventType.message}</Text>
      )}

      {/* Event Date */}
      <Text style={styles.formLabel}>Event Date * (YYYY-MM-DD)</Text>
      <Controller
        control={control}
        name="eventDate"
        render={({ field: { onChange, value, onBlur } }) => (
          <TextInput
            style={[styles.input, errors.eventDate && styles.inputError]}
            value={value}
            onChangeText={onChange}
            onBlur={onBlur}
            placeholder="2026-06-15"
            keyboardType="numbers-and-punctuation"
          />
        )}
      />
      {errors.eventDate && (
        <Text style={styles.errorText}>{errors.eventDate.message}</Text>
      )}

      {/* Guest Count */}
      <Text style={styles.formLabel}>Guest Count *</Text>
      <Controller
        control={control}
        name="guestCount"
        render={({ field: { onChange, value, onBlur } }) => (
          <TextInput
            style={[styles.input, errors.guestCount && styles.inputError]}
            value={value !== undefined ? String(value) : ''}
            onChangeText={(v) => onChange(v === '' ? undefined : Number(v))}
            onBlur={onBlur}
            placeholder="50"
            keyboardType="number-pad"
          />
        )}
      />
      {errors.guestCount && (
        <Text style={styles.errorText}>{errors.guestCount.message}</Text>
      )}

      {/* Budget */}
      <Text style={styles.formLabel}>Budget (₹)</Text>
      <Controller
        control={control}
        name="budget"
        render={({ field: { onChange, value, onBlur } }) => (
          <TextInput
            style={[styles.input, errors.budget && styles.inputError]}
            value={value !== undefined ? String(value) : ''}
            onChangeText={(v) => onChange(v === '' ? undefined : Number(v))}
            onBlur={onBlur}
            placeholder="25000"
            keyboardType="number-pad"
          />
        )}
      />
      {errors.budget && (
        <Text style={styles.errorText}>{errors.budget.message}</Text>
      )}

      {/* City */}
      <Text style={styles.formLabel}>City *</Text>
      <Controller
        control={control}
        name="city"
        render={({ field: { onChange, value, onBlur } }) => (
          <TextInput
            style={[styles.input, errors.city && styles.inputError]}
            value={value}
            onChangeText={onChange}
            onBlur={onBlur}
            placeholder="Mumbai"
            autoCapitalize="words"
          />
        )}
      />
      {errors.city && (
        <Text style={styles.errorText}>{errors.city.message}</Text>
      )}

      {/* State */}
      <Text style={styles.formLabel}>State *</Text>
      <Controller
        control={control}
        name="state"
        render={({ field: { onChange, value, onBlur } }) => (
          <TextInput
            style={[styles.input, errors.state && styles.inputError]}
            value={value}
            onChangeText={onChange}
            onBlur={onBlur}
            placeholder="Maharashtra"
            autoCapitalize="words"
          />
        )}
      />
      {errors.state && (
        <Text style={styles.errorText}>{errors.state.message}</Text>
      )}

      {/* Description */}
      <Text style={styles.formLabel}>Additional Details</Text>
      <Controller
        control={control}
        name="description"
        render={({ field: { onChange, value, onBlur } }) => (
          <TextInput
            style={[styles.input, styles.textArea]}
            value={value ?? ''}
            onChangeText={onChange}
            onBlur={onBlur}
            placeholder="Any specific requirements, dietary restrictions, menu preferences..."
            multiline
            numberOfLines={4}
          />
        )}
      />

      <TouchableOpacity
        style={[
          styles.submitButton,
          createRequest.isPending && styles.submitButtonDisabled,
        ]}
        onPress={() => void handleSubmit(onSubmit)()}
        disabled={createRequest.isPending}
      >
        {createRequest.isPending ? (
          <ActivityIndicator size="small" color="#fafaf7" />
        ) : (
          <Text style={styles.submitButtonText}>Submit Request</Text>
        )}
      </TouchableOpacity>
    </ScrollView>
  );
}

export default function CateringScreen() {
  const [activeTab, setActiveTab] = useState<TabKey>('request');
  const { data, isLoading, refetch } = useCateringRequests();

  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.title}>Catering</Text>
      </View>

      {/* Tab switcher */}
      <View style={styles.tabBar}>
        <TouchableOpacity
          style={[styles.tab, activeTab === 'request' && styles.tabActive]}
          onPress={() => setActiveTab('request')}
        >
          <Text
            style={[
              styles.tabText,
              activeTab === 'request' && styles.tabTextActive,
            ]}
          >
            Request Catering
          </Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={[styles.tab, activeTab === 'my-requests' && styles.tabActive]}
          onPress={() => {
            setActiveTab('my-requests');
            void refetch();
          }}
        >
          <Text
            style={[
              styles.tabText,
              activeTab === 'my-requests' && styles.tabTextActive,
            ]}
          >
            My Requests
          </Text>
        </TouchableOpacity>
      </View>

      {activeTab === 'request' ? (
        <RequestForm onSuccess={() => setActiveTab('my-requests')} />
      ) : isLoading ? (
        <View style={styles.centered}>
          <ActivityIndicator size="large" color="#C2410C" />
        </View>
      ) : (
        <FlatList
          data={data?.data ?? []}
          keyExtractor={(item) => item.id}
          renderItem={({ item }) => <RequestCard request={item} />}
          contentContainerStyle={styles.requestList}
          ListEmptyComponent={
            <View style={styles.emptyState}>
              <Text style={styles.emptyIcon}>🍽️</Text>
              <Text style={styles.emptyTitle}>No requests yet</Text>
              <Text style={styles.emptySubtitle}>
                Submit a catering request to get quotes from chefs.
              </Text>
            </View>
          }
        />
      )}
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
  header: {
    paddingHorizontal: 16,
    paddingTop: 8,
    paddingBottom: 4,
  },
  title: {
    fontSize: 24,
    fontWeight: '700',
    color: '#1a1a18',
  },
  tabBar: {
    flexDirection: 'row',
    marginHorizontal: 16,
    marginVertical: 12,
    backgroundColor: '#e6e5e0',
    borderRadius: 10,
    padding: 3,
  },
  tab: {
    flex: 1,
    paddingVertical: 9,
    alignItems: 'center',
    borderRadius: 8,
  },
  tabActive: {
    backgroundColor: '#fafaf7',
    shadowColor: '#1a1a18',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.06,
    shadowRadius: 2,
    elevation: 1,
  },
  tabText: {
    fontSize: 13,
    fontWeight: '500',
    color: '#7a7a76',
  },
  tabTextActive: {
    color: '#1a1a18',
    fontWeight: '700',
  },
  // Form
  formScroll: {
    paddingHorizontal: 16,
    paddingBottom: 40,
  },
  formLabel: {
    fontSize: 13,
    fontWeight: '600',
    color: '#4a4a47',
    marginTop: 14,
    marginBottom: 6,
  },
  eventTypeRow: {
    gap: 8,
    paddingBottom: 4,
  },
  eventTypeChip: {
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: 20,
    borderWidth: 1,
    borderColor: '#d4d3ce',
    backgroundColor: '#fafaf7',
    marginRight: 8,
  },
  eventTypeChipSelected: {
    backgroundColor: '#FFEDD5',
    borderColor: '#C2410C',
  },
  eventTypeChipText: {
    fontSize: 13,
    color: '#7a7a76',
  },
  eventTypeChipTextSelected: {
    color: '#C2410C',
    fontWeight: '600',
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
  textArea: {
    height: 100,
    textAlignVertical: 'top',
  },
  errorText: {
    fontSize: 12,
    color: '#c95b3e',
    marginTop: 4,
  },
  submitButton: {
    marginTop: 24,
    backgroundColor: '#C2410C',
    paddingVertical: 14,
    borderRadius: 12,
    alignItems: 'center',
  },
  submitButtonDisabled: {
    opacity: 0.6,
  },
  submitButtonText: {
    color: '#fafaf7',
    fontSize: 16,
    fontWeight: '700',
  },
  // Request list
  requestList: {
    padding: 16,
  },
  requestCard: {
    backgroundColor: '#fafaf7',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    shadowColor: '#1a1a18',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.06,
    shadowRadius: 4,
    elevation: 2,
  },
  requestCardHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 8,
  },
  requestEventType: {
    fontSize: 16,
    fontWeight: '700',
    color: '#1a1a18',
  },
  statusDot: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 20,
  },
  statusDotText: {
    fontSize: 12,
    fontWeight: '600',
    color: '#fafaf7',
  },
  requestMeta: {
    fontSize: 13,
    color: '#7a7a76',
    marginBottom: 4,
  },
  requestBudget: {
    fontSize: 13,
    color: '#4a4a47',
    marginTop: 2,
  },
  viewQuoteHint: {
    fontSize: 12,
    color: '#C2410C',
    fontWeight: '500',
    marginTop: 6,
  },
  emptyState: {
    alignItems: 'center',
    justifyContent: 'center',
    paddingTop: 60,
  },
  emptyIcon: {
    fontSize: 48,
    marginBottom: 16,
  },
  emptyTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#4a4a47',
    marginBottom: 8,
  },
  emptySubtitle: {
    fontSize: 14,
    color: '#7a7a76',
    textAlign: 'center',
    paddingHorizontal: 40,
  },
});
