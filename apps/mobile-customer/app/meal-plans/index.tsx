import { Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';

import { MealPlanList } from '../../components/meal-plan/MealPlanList';

// Android ripple tint — translucent token, never a new literal colour.
const ICON_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

// My tiffin plans (#196): the standalone route (reached from Profile → My meal
// plans and the booking-confirmation redirect). The list itself is the shared
// MealPlanList, so this route, the Plans tab, and the Orders segment stay identical.
export default function MyMealPlansScreen() {
  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: ICON_RIPPLE, borderless: true }}
        >
          <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>My meal plans</Text>
        <View style={{ width: 24 }} />
      </View>

      <MealPlanList />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  title: { fontFamily: 'Inter-SemiBold', fontSize: 18, color: customerColors.charcoal.DEFAULT },
});
