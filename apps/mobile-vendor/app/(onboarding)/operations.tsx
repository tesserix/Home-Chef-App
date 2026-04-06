import { View, Text, TextInput, TouchableOpacity, ScrollView, Switch, Alert } from 'react-native';
import { useState } from 'react';
import { router } from 'expo-router';
import { useVendorOnboardingStore } from '../../store/onboarding-store';

type DayHours = { open: string; close: string; closed: boolean };
type HoursMap = Record<string, DayHours>;

const DAYS = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'] as const;
type Day = (typeof DAYS)[number];

const DAY_LABELS: Record<Day, string> = {
  monday: 'Monday',
  tuesday: 'Tuesday',
  wednesday: 'Wednesday',
  thursday: 'Thursday',
  friday: 'Friday',
  saturday: 'Saturday',
  sunday: 'Sunday',
};

const PREP_TIME_OPTIONS = ['15min', '30min', '45min', '60min', '90min'] as const;

export default function OperationsScreen() {
  const { operations, updateOperations, setStep } = useVendorOnboardingStore();

  const [hours, setHours] = useState<HoursMap>(operations.operatingHours);
  const [prepTime, setPrepTime] = useState(operations.prepTime);
  const [serviceRadius, setServiceRadius] = useState(String(operations.serviceRadius));

  function updateDay(day: Day, field: 'open' | 'close', value: string): void {
    setHours((prev: HoursMap) => ({
      ...prev,
      [day]: { ...prev[day], [field]: value },
    }));
  }

  function toggleDay(day: Day, closed: boolean): void {
    setHours((prev: HoursMap) => ({
      ...prev,
      [day]: { ...prev[day], closed },
    }));
  }

  function onNext(): void {
    const radius = parseInt(serviceRadius, 10);
    if (isNaN(radius) || radius < 1 || radius > 50) {
      Alert.alert('Validation Error', 'Service radius must be between 1 and 50 km');
      return;
    }
    updateOperations({ operatingHours: hours, prepTime, serviceRadius: radius });
    setStep(4);
    router.push('/(onboarding)/documents');
  }

  return (
    <ScrollView className="flex-1 bg-white">
      <View className="px-6 pt-4 pb-8">
        <View className="h-1.5 rounded-full bg-gray-200 mb-6">
          <View className="h-1.5 rounded-full bg-orange-500" style={{ width: `${(3 / 6) * 100}%` }} />
        </View>

        <Text className="text-2xl font-bold text-gray-900 mb-1">Operations</Text>
        <Text className="text-sm text-gray-500 mb-6">Set your working hours and service details</Text>

        <Text className="text-sm font-medium text-gray-700 mb-3">Operating Hours</Text>
        {DAYS.map((day) => {
          const dayData = hours[day] ?? { open: '09:00', close: '21:00', closed: false };
          return (
            <View key={day} className="flex-row items-center mb-3">
              <View className="w-24">
                <Text className="text-sm text-gray-700 font-medium">{DAY_LABELS[day].slice(0, 3)}</Text>
              </View>
              <Switch
                value={!dayData.closed}
                onValueChange={(val) => toggleDay(day, !val)}
                trackColor={{ false: '#D1D5DB', true: '#FED7AA' }}
                thumbColor={!dayData.closed ? '#FF6B35' : '#9CA3AF'}
              />
              {!dayData.closed ? (
                <View className="flex-row items-center ml-3 gap-2">
                  <TextInput
                    className="border border-gray-300 rounded px-2 py-1 text-sm text-gray-900 w-16 text-center"
                    value={dayData.open}
                    onChangeText={(val) => updateDay(day, 'open', val)}
                    placeholder="09:00"
                    maxLength={5}
                  />
                  <Text className="text-gray-400">–</Text>
                  <TextInput
                    className="border border-gray-300 rounded px-2 py-1 text-sm text-gray-900 w-16 text-center"
                    value={dayData.close}
                    onChangeText={(val) => updateDay(day, 'close', val)}
                    placeholder="21:00"
                    maxLength={5}
                  />
                </View>
              ) : (
                <Text className="ml-3 text-sm text-gray-400">Closed</Text>
              )}
            </View>
          );
        })}

        <Text className="text-sm font-medium text-gray-700 mt-4 mb-2">Prep Time</Text>
        <View className="flex-row flex-wrap gap-2 mb-4">
          {PREP_TIME_OPTIONS.map((option) => (
            <TouchableOpacity
              key={option}
              onPress={() => setPrepTime(option)}
              className={`px-4 py-2 rounded-full border ${
                prepTime === option
                  ? 'bg-orange-500 border-orange-500'
                  : 'bg-white border-gray-300'
              }`}
            >
              <Text className={`text-sm ${prepTime === option ? 'text-white font-medium' : 'text-gray-700'}`}>
                {option}
              </Text>
            </TouchableOpacity>
          ))}
        </View>

        <Text className="text-sm font-medium text-gray-700 mb-1">Service Radius (km)</Text>
        <TextInput
          className="border border-gray-300 rounded-lg px-4 py-3 text-base text-gray-900 mb-6"
          value={serviceRadius}
          onChangeText={setServiceRadius}
          keyboardType="number-pad"
          placeholder="1–50 km"
          maxLength={2}
        />

        <TouchableOpacity
          className="bg-orange-500 rounded-xl py-4 items-center"
          onPress={onNext}
        >
          <Text className="text-white font-semibold text-base">Next</Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}
