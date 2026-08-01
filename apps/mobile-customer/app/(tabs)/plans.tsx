import { StyleSheet } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { customerColors } from '@homechef/mobile-shared/theme';

import { ScreenTitle } from '../../components/shared/ScreenTitle';
import { MealPlanList } from '../../components/meal-plan/MealPlanList';
import { SubscriptionsSummary } from '../../components/meal-plan/SubscriptionsSummary';
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
      {/* "Plans", not "Meal plans": this screen now holds two different things —
          recurring tiffin subscriptions and one-off meal plans — and the tab it
          sits under is called Plans. A subscriber reading "Meal plans" above
          their tiffin row is being told they're in the wrong place (#900). */}
      <ScreenTitle title="Plans" />
      <SubscriptionsSummary />
      <MealPlanList />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
});
