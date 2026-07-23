// Legal — consolidated index for the reference legal documents (Terms,
// Privacy, Refund, EULA). Profile links here with a single "Legal" row
// instead of four separate ones; "Your Data" stays top-level in Profile
// because it's a DPDP action center (export/delete), not reading material.

import { Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, type Href } from 'expo-router';
import {
  ChevronRight,
  FileText,
  Receipt,
  ScrollText,
  Shield,
  type LucideIcon,
} from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ScreenHeader } from '../components/ScreenHeader';

const DOCS: { label: string; route: Href; icon: LucideIcon }[] = [
  { label: 'Terms of Service', route: '/terms', icon: FileText },
  { label: 'Privacy Policy', route: '/privacy', icon: Shield },
  { label: 'Refund Policy', route: '/refund', icon: Receipt },
  { label: 'End User Licence', route: '/eula', icon: ScrollText },
];

// Android ripple tint — translucent token, never a new literal colour.
const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

export default function LegalScreen() {
  return (
    <SafeAreaView
      edges={['top', 'left', 'right']}
      style={{ flex: 1, backgroundColor: customerColors.canvas }}
    >
      <ScreenHeader title="Legal" />
      <View style={styles.card}>
        {DOCS.map((doc, i) => {
          const Icon = doc.icon;
          return (
            <View key={doc.label}>
              {i > 0 && <View style={styles.divider} />}
              <Pressable
                onPress={() => router.push(doc.route)}
                accessibilityRole="button"
                accessibilityLabel={doc.label}
                android_ripple={{ color: ROW_RIPPLE, borderless: false }}
              >
                {({ pressed }) => (
                  <View
                    style={[
                      styles.row,
                      pressed && Platform.OS === 'ios' && styles.rowPressed,
                    ]}
                  >
                    <View style={styles.iconCircle}>
                      <Icon size={18} color={customerColors.charcoal.soft} />
                    </View>
                    <Text style={styles.label}>{doc.label}</Text>
                    <ChevronRight
                      size={16}
                      color={customerColors.charcoal.soft}
                    />
                  </View>
                )}
              </Pressable>
            </View>
          );
        })}
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  card: {
    marginHorizontal: 16,
    marginTop: 8,
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    overflow: 'hidden',
    backgroundColor: customerColors.canvas,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 12,
    minHeight: 52,
    backgroundColor: customerColors.canvas,
  },
  rowPressed: {
    backgroundColor: customerColors.surface.soft,
  },
  iconCircle: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 12,
  },
  label: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: 16,
    color: customerColors.charcoal.DEFAULT,
  },
  divider: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: customerColors.hairline,
    marginLeft: 64,
  },
});
