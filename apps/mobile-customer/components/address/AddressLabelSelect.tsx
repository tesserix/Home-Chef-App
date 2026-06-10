// Home / Work / Other selector for tagging a delivery address.
// Shared by the checkout "Add new address" form and the onboarding address step.
// Backend `Address.Label` is required (defaults to "Home" server-side); this lets
// the customer choose so a multi-address list is scannable instead of three
// identical "Home" rows.

import { Pressable, Text, View } from 'react-native';

export const ADDRESS_LABELS = ['Home', 'Work', 'Other'] as const;
export type AddressLabelValue = (typeof ADDRESS_LABELS)[number];

interface AddressLabelSelectProps {
  value: string;
  onChange: (label: AddressLabelValue) => void;
}

export function AddressLabelSelect({ value, onChange }: AddressLabelSelectProps) {
  return (
    <View>
      <Text className="text-sm font-medium text-charcoal-soft mb-2">Label this address</Text>
      <View className="flex-row gap-2">
        {ADDRESS_LABELS.map((label) => {
          const selected = value === label;
          return (
            <Pressable
              key={label}
              onPress={() => onChange(label)}
              accessibilityRole="radio"
              accessibilityState={{ checked: selected }}
              accessibilityLabel={`Label this address as ${label}`}
              className="flex-1"
            >
              {/* iOS Pressable inner-View pattern — visual styles on the inner View */}
              <View
                className={`rounded-xl py-2.5 items-center border ${
                  selected ? 'border-coral bg-coral-tint' : 'border-hairline bg-canvas'
                }`}
              >
                <Text
                  className={`text-sm font-medium ${
                    selected ? 'text-coral' : 'text-charcoal-soft'
                  }`}
                >
                  {label}
                </Text>
              </View>
            </Pressable>
          );
        })}
      </View>
    </View>
  );
}
