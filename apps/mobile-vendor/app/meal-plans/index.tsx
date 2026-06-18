import { FlatList, Pressable, RefreshControl, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { CalendarDays, ChevronLeft, Inbox, UtensilsCrossed } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { Skeleton } from '@homechef/mobile-shared/ui';
import { useChefMealPlanRequests } from '../../hooks/useMealPlans';
import { MealPlanRequestCard } from '../../components/vendor/MealPlanRequestCard';

// Vendor tiffin hub (#195): pending meal-plan requests the chef must respond to
// (24h window), plus the entry point to edit the weekly menu the plans book from.
export default function MealPlansScreen() {
  const { data, isLoading, refetch, isRefetching } = useChefMealPlanRequests();
  const requests = data?.data ?? [];

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Go back"
        >
          <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>Tiffin plans</Text>
        <View style={{ width: 24 }} />
      </View>

      {/* Weekly-menu editor entry */}
      <Pressable
        onPress={() => router.push('/meal-plans/weekly-menu' as never)}
        accessibilityRole="button"
      >
        {({ pressed }) => (
          <View style={[styles.menuCta, pressed && styles.pressed]}>
            <View style={styles.menuIcon}>
              <UtensilsCrossed
                size={20}
                color={theme.colors.herb.DEFAULT}
                strokeWidth={1.75}
              />
            </View>
            <View style={{ flex: 1 }}>
              <Text style={styles.menuCtaTitle}>Weekly menu</Text>
              <Text style={styles.menuCtaCaption}>
                Set the dishes customers can pre-book, per day
              </Text>
            </View>
            <CalendarDays size={18} color={theme.colors.ink.muted} />
          </View>
        )}
      </Pressable>

      <Text style={styles.sectionLabel}>Pending requests</Text>

      {isLoading ? (
        <View style={{ paddingHorizontal: theme.spacing[4], gap: theme.spacing[3] }}>
          <Skeleton style={styles.skeleton} />
          <Skeleton style={styles.skeleton} />
        </View>
      ) : (
        <FlatList
          data={requests}
          keyExtractor={(p) => p.id}
          renderItem={({ item }) => (
            <MealPlanRequestCard
              plan={item}
              onPress={() => router.push(`/meal-plans/${item.id}` as never)}
            />
          )}
          contentContainerStyle={styles.listContent}
          refreshControl={
            <RefreshControl
              refreshing={isRefetching}
              onRefresh={refetch}
              tintColor={theme.colors.ink.muted}
            />
          }
          ListEmptyComponent={
            <View style={styles.empty}>
              <Inbox size={40} color={theme.colors.ink.muted} strokeWidth={1.5} />
              <Text style={styles.emptyTitle}>No pending requests</Text>
              <Text style={styles.emptyText}>
                When a customer pre-books a plan, it appears here for you to
                accept or adjust.
              </Text>
            </View>
          }
        />
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  title: {
    fontFamily: 'Geist-Bold',
    fontSize: 20,
    color: theme.colors.ink.DEFAULT,
  },
  menuCta: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    padding: theme.spacing[4],
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[4],
    ...theme.shadow[1],
  },
  pressed: { backgroundColor: theme.colors.bone },
  menuIcon: {
    width: 36,
    height: 36,
    borderRadius: theme.radius.full,
    backgroundColor: theme.colors.herb.tint,
    alignItems: 'center',
    justifyContent: 'center',
  },
  menuCtaTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: theme.colors.ink.DEFAULT,
  },
  menuCtaCaption: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: theme.colors.ink.muted,
    marginTop: 2,
  },
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    letterSpacing: 0.3,
    textTransform: 'uppercase',
    color: theme.colors.ink.soft,
    paddingHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[3],
  },
  listContent: { paddingBottom: theme.spacing[10] },
  skeleton: { height: 110, borderRadius: theme.radius.md },
  empty: {
    alignItems: 'center',
    paddingHorizontal: theme.spacing[8],
    paddingTop: theme.spacing[10],
    gap: theme.spacing[2],
  },
  emptyTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: theme.colors.ink.DEFAULT,
    marginTop: theme.spacing[2],
  },
  emptyText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: theme.colors.ink.muted,
    textAlign: 'center',
    lineHeight: 20,
  },
});
