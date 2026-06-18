// Catering screen — endpoints confirmed from apps/api/handlers/catering.go
// POST /v1/catering/requests  → create request
// GET  /v1/catering/requests  → list my requests

import React, { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  FlatList,
  Pressable,
  ScrollView,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { z } from 'zod';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { UtensilsCrossed } from 'lucide-react-native';
import {
  useCateringRequests,
  useCreateCateringRequest,
} from '../hooks/useCatering';
import type { CateringRequest } from '../hooks/useCatering';
import { friendlyErrorMessage } from '../lib/errors';
import { customerColors } from '@homechef/mobile-shared/theme';

// Threat model T-02-05-02: Zod validates required fields before POST
const cateringSchema = z.object({
  eventType: z.string().min(1, 'Event type is required'),
  eventDate: z
    .string()
    .regex(/^\d{4}-\d{2}-\d{2}$/, 'Date must be YYYY-MM-DD')
    .refine((d) => new Date(d) > new Date(), 'Event date must be in the future'),
  guestCount: z
    .number({ error: 'Guest count must be a number' })
    .int()
    .min(1, 'At least 1 guest required'),
  budget: z
    .number({ error: 'Budget must be a number' })
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

// ─── Status chip ─────────────────────────────────────────────────────────────

type StatusConfig = { label: string; bg: string; text: string };

function getStatusConfig(status: CateringRequest['status']): StatusConfig {
  switch (status) {
    case 'open':
      return { label: 'Open', bg: customerColors.surface.soft, text: customerColors.charcoal.DEFAULT };
    case 'quoted':
      // A quote awaits the customer's action — coral attention state.
      return { label: 'Quoted', bg: customerColors.coral.tint, text: customerColors.coral.pressed };
    case 'accepted':
      return { label: 'Accepted', bg: customerColors.coral.tint, text: customerColors.coral.pressed };
    case 'completed':
      return { label: 'Completed', bg: customerColors.success.tint, text: customerColors.success.DEFAULT };
    case 'cancelled':
      return { label: 'Cancelled', bg: customerColors.surface.soft, text: customerColors.charcoal.soft };
    default:
      return { label: status, bg: customerColors.surface.soft, text: customerColors.charcoal.soft };
  }
}

// ─── Request card ─────────────────────────────────────────────────────────────

function RequestCard({ request }: { request: CateringRequest }) {
  const { label, bg, text } = getStatusConfig(request.status);

  return (
    /* Shadow on outer View, overflow+radius on inner clip View */
    <View
      style={{
        shadowColor: customerColors.charcoal.DEFAULT,
        shadowOffset: { width: 0, height: 1 },
        shadowOpacity: 0.06,
        shadowRadius: 4,
        elevation: 2,
        marginBottom: 12,
      }}
    >
      <View className="bg-canvas rounded-xl overflow-hidden">
        <View className="p-4">
          {/* Header row: event type + status chip */}
          <View className="flex-row items-center justify-between mb-2">
            <Text className="text-base font-bold text-charcoal font-display flex-1 mr-2">
              {request.eventType}
            </Text>
            <View
              className="rounded-full px-3 py-1"
              style={{ backgroundColor: bg }}
            >
              <Text className="text-xs font-semibold" style={{ color: text }}>
                {label}
              </Text>
            </View>
          </View>

          {/* Meta info */}
          <Text className="text-sm text-charcoal-soft">
            {request.guestCount} guests · {request.eventDate} · {request.city},{' '}
            {request.state}
          </Text>

          {/* Budget */}
          {request.budget != null ? (
            <Text className="text-sm text-charcoal mt-1">
              Budget: ₹{request.budget}
            </Text>
          ) : null}

          {/* Quoted hint in coral */}
          {request.status === 'quoted' ? (
            <Text className="text-xs text-coral font-medium mt-2">
              Quotes available — view details
            </Text>
          ) : null}
        </View>
      </View>
    </View>
  );
}

// ─── Empty state ─────────────────────────────────────────────────────────────

function EmptyState() {
  return (
    <View className="flex-1 items-center justify-center px-8 gap-4 pt-20">
      <View className="w-20 h-20 rounded-full bg-surface-soft items-center justify-center">
        <UtensilsCrossed size={36} color={customerColors.charcoal.soft} />
      </View>
      <View className="items-center gap-2">
        <Text className="text-xl font-bold text-charcoal text-center font-display">
          No requests yet
        </Text>
        <Text className="text-sm text-charcoal-soft text-center leading-5">
          Submit a catering request to get quotes from chefs.
        </Text>
      </View>
    </View>
  );
}

// ─── Request form (label + input field helper) ────────────────────────────────

function FieldLabel({ children }: { children: string }) {
  return (
    <Text className="text-xs font-semibold text-charcoal-soft uppercase tracking-wide mb-1.5 mt-4">
      {children}
    </Text>
  );
}

interface StyledInputProps {
  hasError: boolean;
  children: React.ReactNode;
}

function InputWrap({ hasError, children }: StyledInputProps) {
  return (
    <View
      className={`bg-surface-soft rounded-lg px-3 py-3 ${hasError ? 'border border-coral-pressed' : ''}`}
    >
      {children}
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
        onError: (err) => {
          Alert.alert(
            'Error',
            friendlyErrorMessage(err, 'Could not submit request. Please try again.'),
          );
        },
      },
    );
  }

  return (
    <ScrollView
      contentContainerStyle={{ paddingHorizontal: 16, paddingBottom: 40 }}
      keyboardShouldPersistTaps="handled"
    >
      {/* Event Type chips — horizontal scroll */}
      <FieldLabel>Event Type *</FieldLabel>
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={{ gap: 8, paddingBottom: 4 }}
      >
        {EVENT_TYPES.map((t) => {
          const isSelected = selectedEventType === t;
          return (
            /* iOS Pressable pattern: visual styles on inner View */
            <Pressable
              key={t}
              onPress={() => selectEventType(t)}
              accessibilityRole="button"
              accessibilityLabel={t}
              accessibilityState={{ selected: isSelected }}
            >
              <View
                className={`px-4 py-2 rounded-full border ${
                  isSelected
                    ? 'bg-coral-tint border-coral'
                    : 'bg-canvas border-hairline'
                }`}
              >
                <Text
                  className={`text-sm font-medium ${
                    isSelected ? 'text-coral font-semibold' : 'text-charcoal-soft'
                  }`}
                >
                  {t}
                </Text>
              </View>
            </Pressable>
          );
        })}
      </ScrollView>
      {errors.eventType ? (
        <Text className="text-xs text-coral-pressed mt-1">{errors.eventType.message}</Text>
      ) : null}

      {/* Event Date */}
      <FieldLabel>Event Date * (YYYY-MM-DD)</FieldLabel>
      <Controller
        control={control}
        name="eventDate"
        render={({ field: { onChange, value, onBlur } }) => (
          <InputWrap hasError={!!errors.eventDate}>
            <TextInput
              className="text-base text-charcoal"
              value={value}
              onChangeText={onChange}
              onBlur={onBlur}
              placeholder="2026-06-15"
              placeholderTextColor={customerColors.charcoal.soft}
              keyboardType="numbers-and-punctuation"
              accessibilityLabel="Event date"
            />
          </InputWrap>
        )}
      />
      {errors.eventDate ? (
        <Text className="text-xs text-coral-pressed mt-1">{errors.eventDate.message}</Text>
      ) : null}

      {/* Guest Count */}
      <FieldLabel>Guest Count *</FieldLabel>
      <Controller
        control={control}
        name="guestCount"
        render={({ field: { onChange, value, onBlur } }) => (
          <InputWrap hasError={!!errors.guestCount}>
            <TextInput
              className="text-base text-charcoal"
              value={value !== undefined ? String(value) : ''}
              onChangeText={(v) => onChange(v === '' ? undefined : Number(v))}
              onBlur={onBlur}
              placeholder="50"
              placeholderTextColor={customerColors.charcoal.soft}
              keyboardType="number-pad"
              accessibilityLabel="Guest count"
            />
          </InputWrap>
        )}
      />
      {errors.guestCount ? (
        <Text className="text-xs text-coral-pressed mt-1">{errors.guestCount.message}</Text>
      ) : null}

      {/* Budget */}
      <FieldLabel>Budget (₹)</FieldLabel>
      <Controller
        control={control}
        name="budget"
        render={({ field: { onChange, value, onBlur } }) => (
          <InputWrap hasError={!!errors.budget}>
            <TextInput
              className="text-base text-charcoal"
              value={value !== undefined ? String(value) : ''}
              onChangeText={(v) => onChange(v === '' ? undefined : Number(v))}
              onBlur={onBlur}
              placeholder="25000"
              placeholderTextColor={customerColors.charcoal.soft}
              keyboardType="number-pad"
              accessibilityLabel="Budget in rupees"
            />
          </InputWrap>
        )}
      />
      {errors.budget ? (
        <Text className="text-xs text-coral-pressed mt-1">{errors.budget.message}</Text>
      ) : null}

      {/* City */}
      <FieldLabel>City *</FieldLabel>
      <Controller
        control={control}
        name="city"
        render={({ field: { onChange, value, onBlur } }) => (
          <InputWrap hasError={!!errors.city}>
            <TextInput
              className="text-base text-charcoal"
              value={value}
              onChangeText={onChange}
              onBlur={onBlur}
              placeholder="Mumbai"
              placeholderTextColor={customerColors.charcoal.soft}
              autoCapitalize="words"
              accessibilityLabel="City"
            />
          </InputWrap>
        )}
      />
      {errors.city ? (
        <Text className="text-xs text-coral-pressed mt-1">{errors.city.message}</Text>
      ) : null}

      {/* State */}
      <FieldLabel>State *</FieldLabel>
      <Controller
        control={control}
        name="state"
        render={({ field: { onChange, value, onBlur } }) => (
          <InputWrap hasError={!!errors.state}>
            <TextInput
              className="text-base text-charcoal"
              value={value}
              onChangeText={onChange}
              onBlur={onBlur}
              placeholder="Maharashtra"
              placeholderTextColor={customerColors.charcoal.soft}
              autoCapitalize="words"
              accessibilityLabel="State"
            />
          </InputWrap>
        )}
      />
      {errors.state ? (
        <Text className="text-xs text-coral-pressed mt-1">{errors.state.message}</Text>
      ) : null}

      {/* Additional Details */}
      <FieldLabel>Additional Details</FieldLabel>
      <Controller
        control={control}
        name="description"
        render={({ field: { onChange, value, onBlur } }) => (
          <View className="bg-surface-soft rounded-lg px-3 py-3">
            <TextInput
              className="text-base text-charcoal"
              style={{ height: 100, textAlignVertical: 'top' }}
              value={value ?? ''}
              onChangeText={onChange}
              onBlur={onBlur}
              placeholder="Any specific requirements, dietary restrictions, menu preferences..."
              placeholderTextColor={customerColors.charcoal.soft}
              multiline
              numberOfLines={4}
              accessibilityLabel="Additional details"
            />
          </View>
        )}
      />

      {/* Submit CTA — coral primary, radius 8, min-height 52 */}
      <Pressable
        onPress={() => void handleSubmit(onSubmit)()}
        disabled={createRequest.isPending}
        accessibilityRole="button"
        accessibilityLabel="Submit catering request"
      >
        {({ pressed }) => (
          <View
            className={`mt-6 rounded-lg min-h-[52px] items-center justify-center bg-coral ${
              createRequest.isPending ? 'opacity-60' : pressed ? 'opacity-90' : ''
            }`}
          >
            {createRequest.isPending ? (
              <ActivityIndicator size="small" color={customerColors.canvas} />
            ) : (
              <Text className="text-canvas font-semibold text-base">
                Submit Request
              </Text>
            )}
          </View>
        )}
      </Pressable>
    </ScrollView>
  );
}

// ─── Main screen ─────────────────────────────────────────────────────────────

export default function CateringScreen() {
  const [activeTab, setActiveTab] = useState<TabKey>('request');
  const { data, isLoading, refetch } = useCateringRequests();

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>

      {/* ── Geist-Bold header ── */}
      <View className="px-4 pt-3 pb-2">
        <Text className="text-2xl font-bold text-charcoal tracking-tight font-display">
          Catering
        </Text>
      </View>

      {/* ── Segmented tab bar ── */}
      <View className="mx-4 my-3 bg-surface-soft rounded-lg p-1 flex-row">
        {/* Tab: Request Catering */}
        {/* iOS Pressable pattern: visual styles on inner View */}
        <Pressable
          className="flex-1"
          onPress={() => setActiveTab('request')}
          accessibilityRole="tab"
          accessibilityLabel="Request Catering"
          accessibilityState={{ selected: activeTab === 'request' }}
        >
          <View
            className={`flex-1 py-2 items-center rounded-md ${
              activeTab === 'request'
                ? 'bg-canvas'
                : 'bg-transparent'
            }`}
            style={
              activeTab === 'request'
                ? {
                    shadowColor: customerColors.charcoal.DEFAULT,
                    shadowOffset: { width: 0, height: 1 },
                    shadowOpacity: 0.06,
                    shadowRadius: 2,
                    elevation: 1,
                  }
                : undefined
            }
          >
            <Text
              className={`text-sm font-medium ${
                activeTab === 'request' ? 'text-charcoal font-bold' : 'text-charcoal-soft'
              }`}
            >
              Request Catering
            </Text>
          </View>
        </Pressable>

        {/* Tab: My Requests */}
        <Pressable
          className="flex-1"
          onPress={() => {
            setActiveTab('my-requests');
            void refetch();
          }}
          accessibilityRole="tab"
          accessibilityLabel="My Requests"
          accessibilityState={{ selected: activeTab === 'my-requests' }}
        >
          <View
            className={`flex-1 py-2 items-center rounded-md ${
              activeTab === 'my-requests'
                ? 'bg-canvas'
                : 'bg-transparent'
            }`}
            style={
              activeTab === 'my-requests'
                ? {
                    shadowColor: customerColors.charcoal.DEFAULT,
                    shadowOffset: { width: 0, height: 1 },
                    shadowOpacity: 0.06,
                    shadowRadius: 2,
                    elevation: 1,
                  }
                : undefined
            }
          >
            <Text
              className={`text-sm font-medium ${
                activeTab === 'my-requests' ? 'text-charcoal font-bold' : 'text-charcoal-soft'
              }`}
            >
              My Requests
            </Text>
          </View>
        </Pressable>
      </View>

      {/* ── Content ── */}
      {activeTab === 'request' ? (
        <RequestForm onSuccess={() => setActiveTab('my-requests')} />
      ) : isLoading ? (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color={customerColors.coral.DEFAULT} />
        </View>
      ) : (
        <FlatList<CateringRequest>
          data={data?.data ?? []}
          keyExtractor={(item) => item.id}
          renderItem={({ item }) => <RequestCard request={item} />}
          contentContainerStyle={
            (data?.data ?? []).length === 0
              ? { flexGrow: 1 }
              : { padding: 16, paddingBottom: 32 }
          }
          ListEmptyComponent={<EmptyState />}
        />
      )}

    </SafeAreaView>
  );
}
