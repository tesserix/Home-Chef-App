import { StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { customerColors } from '@homechef/mobile-shared/theme';

import { ScreenTitle } from '../../components/shared/ScreenTitle';
import { MealPlanList } from '../../components/meal-plan/MealPlanList';
import { CalendarDays } from 'lucide-react-native';
import { GuestGate } from '../../components/GuestGate';
import { useIsGuest } from '../../hooks/useRequireAccount';

// Plans tab (#196): a permanent one-tap home for the customer's tiffin plans,
// previously buried under Profile → My meal plans. The list is the shared,
// self-contained MealPlanList — it owns its own query, loading / error / empty
// states, and dock clearance — so this tab, the standalone /meal-plans route,
// and the Orders segment stay identical.
export default function PlansScreen() {
  // App Review 5.1.1(iv): this tab is account-based, so a guest gets an
  // explanation and a way in rather than an error or an empty list.
  const isGuest = useIsGuest();
  if (isGuest) {
    return (
      <GuestGate
        icon={CalendarDays}
        title="Meal plans need an account"
        body="Sign in to book a weekly plan from a chef and manage it from here."
      />
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* "Plans" rather than "Meal plans" is a leftover from when this screen
          held two things — recurring tiffin subscriptions and one-off meal plans
          (#900). The subscriptions row went with the rest of the recurring UI in
          #1035; the title stays as-is because it matches the tab. */}
      <ScreenTitle title="Plans" />
      <MealPlanList />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
});
