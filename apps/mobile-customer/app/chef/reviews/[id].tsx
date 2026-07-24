// Customer-facing list of a chef's reviews. The chef detail screen now embeds
// the same list in its Reviews tab; this standalone route stays for deep links.
// Read-only; data from the public GET /chefs/:id/reviews via ChefReviewList.

import { Platform, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ChefReviewList } from '../../../components/chef/ChefReviewList';

// Android ripple tint — translucent charcoal derived from the token.
const BACK_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

export default function ChefReviewsScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={10}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          android_ripple={{ color: BACK_RIPPLE, borderless: true, radius: 20 }}
        >
          {({ pressed }) => (
            <View style={pressed && Platform.OS === 'ios' && styles.backPressed}>
              <ChevronLeft size={26} color={customerColors.charcoal.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text style={styles.headerTitle}>Reviews</Text>
        <View style={{ width: 26 }} />
      </View>

      <ScrollView
        contentContainerStyle={styles.listContent}
        showsVerticalScrollIndicator={false}
      >
        <ChefReviewList chefId={id ?? ''} />
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  // Page canvas is white per surface model §1 — NOT the grey surface-soft.
  root: { flex: 1, backgroundColor: customerColors.canvas },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  backPressed: {
    opacity: 0.6,
  },
  headerTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 20,
    color: customerColors.charcoal.DEFAULT,
  },
  listContent: { padding: 16 },
});
