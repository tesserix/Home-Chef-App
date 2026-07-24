import { StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { customerColors } from '@homechef/mobile-shared/theme';

import { ScreenTitle } from '../../components/shared/ScreenTitle';
import { MealPlanList } from '../../components/meal-plan/MealPlanList';

// Plans tab (#196): a permanent one-tap home for the customer's tiffin plans,
// previously buried under Profile → My meal plans. The list is the shared,
// self-contained MealPlanList — it owns its own query, loading / error / empty
// states, and dock clearance — so this tab, the standalone /meal-plans route,
// and the Orders segment stay identical.
export default function PlansScreen() {
  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <ScreenTitle title="Meal plans" />
      <MealPlanList />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
});
