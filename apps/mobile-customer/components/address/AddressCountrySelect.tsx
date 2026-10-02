import { Pressable, Text, View } from "react-native";

const countries = [
  { code: "IN", name: "India" },
  { code: "AU", name: "Australia" },
  { code: "NZ", name: "New Zealand" },
] as const;

export function AddressCountrySelect({
  value,
  onChange,
}: {
  value: string;
  onChange: (country: "IN" | "AU" | "NZ") => void;
}) {
  return (
    <View className="mb-4">
      <Text className="text-sm font-medium text-charcoal mb-2">Country</Text>
      <View className="flex-row flex-wrap gap-2">
        {countries.map(({ code, name }) => (
          <Pressable
            key={code}
            onPress={() => onChange(code)}
            accessibilityRole="radio"
            accessibilityLabel={name}
            accessibilityState={{ checked: value === code }}
            className={`min-h-12 rounded-lg px-3 justify-center border ${value === code ? "border-charcoal bg-surface" : "border-hairline bg-surface-soft"}`}
          >
            <Text className="text-charcoal">{name}</Text>
          </Pressable>
        ))}
      </View>
    </View>
  );
}
