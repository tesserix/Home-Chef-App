import {
  ActivityIndicator,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { Users } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useGroupInvitePreview, useJoinGroup } from '../../hooks/useGroupOrder';

// Android ripple tints — translucent tokens, never a new literal colour.
const CANVAS_RIPPLE = `${customerColors.canvas}33`;
const GHOST_RIPPLE = `${customerColors.charcoal.DEFAULT}0F`;

// Deep-link landing for a group-order invite: homechef-customer://group/<token>
// (and https://fe3dr.com/group/<token>). Previews the invite, then joins.
export default function GroupInviteScreen() {
  const { code } = useLocalSearchParams<{ code: string }>();
  const { data: preview, isLoading, isError } = useGroupInvitePreview(code);
  const join = useJoinGroup();

  function accept() {
    if (!code) return;
    join.mutate(code, {
      onSuccess: (d) => {
        const groupId = d.groupOrderId ?? d.groupOrder?.id;
        if (groupId) router.replace(`/group-order/${groupId}` as never);
      },
    });
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right', 'bottom']}>
      <View style={styles.centered}>
        {isLoading ? (
          <ActivityIndicator color={customerColors.coral.DEFAULT} />
        ) : isError || !preview ? (
          <>
            <Text style={styles.title}>Invite not found</Text>
            <Text style={styles.body}>This group order link is invalid or has expired.</Text>
            <Pressable
              onPress={() => router.replace('/(tabs)')}
              accessibilityRole="button"
              accessibilityLabel="Go home"
              android_ripple={{ color: GHOST_RIPPLE, borderless: false }}
            >
              {({ pressed }) => (
                <View style={[styles.secondary, pressed && Platform.OS === 'ios' && styles.secondaryPressed]}>
                  <Text style={styles.secondaryText}>Go home</Text>
                </View>
              )}
            </Pressable>
          </>
        ) : (
          <>
            <View style={styles.iconCircle}>
              <Users size={36} color={customerColors.coral.DEFAULT} strokeWidth={1.75} />
            </View>
            <Text style={styles.title}>
              {preview.hostName ? `${preview.hostName} invited you` : "You're invited"}
            </Text>
            <Text style={styles.body}>
              Join {preview.type === 'office' ? 'this office order' : 'this group order'}
              {preview.title ? ` "${preview.title}"` : ''} from{' '}
              <Text style={styles.bold}>{preview.chefName ?? 'a chef'}</Text>. Add your own items and
              pay your share.
            </Text>

            {/* Sticky-style primary CTA per spec §2.5 — coral filled, radius 8, 52pt. */}
            {preview.joinable ? (
              <Pressable
                onPress={accept}
                disabled={join.isPending}
                accessibilityRole="button"
                accessibilityLabel="Join the order"
                android_ripple={join.isPending ? undefined : { color: CANVAS_RIPPLE, borderless: false }}
              >
                {({ pressed }) => (
                  <View
                    style={[
                      styles.primary,
                      pressed && Platform.OS === 'ios' && !join.isPending && styles.primaryPressed,
                    ]}
                  >
                    {join.isPending ? (
                      <ActivityIndicator color={customerColors.canvas} />
                    ) : (
                      <Text style={styles.primaryText}>Join the order</Text>
                    )}
                  </View>
                )}
              </Pressable>
            ) : (
              <Text style={styles.closed}>This group order is no longer open to join.</Text>
            )}
            <Pressable
              onPress={() => router.replace('/(tabs)')}
              accessibilityRole="button"
              accessibilityLabel="Not now"
              android_ripple={{ color: GHOST_RIPPLE, borderless: false }}
            >
              {({ pressed }) => (
                <View style={[styles.secondary, pressed && Platform.OS === 'ios' && styles.secondaryPressed]}>
                  <Text style={styles.secondaryText}>Not now</Text>
                </View>
              )}
            </Pressable>
          </>
        )}
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.canvas },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 32, gap: 12 },
  iconCircle: {
    width: 72,
    height: 72,
    borderRadius: 36,
    backgroundColor: customerColors.coral.tint,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 8,
  },
  title: { fontFamily: 'Inter-SemiBold', fontSize: 20, color: customerColors.charcoal.DEFAULT, textAlign: 'center' },
  body: { fontFamily: 'Inter', fontSize: 15, color: customerColors.charcoal.soft, textAlign: 'center', lineHeight: 22 },
  bold: { fontFamily: 'Inter-SemiBold', color: customerColors.charcoal.DEFAULT },
  closed: { fontFamily: 'Inter', fontSize: 14, color: customerColors.destructive.DEFAULT, textAlign: 'center', marginTop: 8 },
  primary: {
    height: 52,
    borderRadius: 8,
    backgroundColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 40,
    marginTop: 12,
    minWidth: 200,
  },
  primaryPressed: { backgroundColor: customerColors.coral.pressed },
  primaryText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: customerColors.canvas },
  secondary: { paddingVertical: 10, paddingHorizontal: 16, borderRadius: 8 },
  secondaryPressed: { backgroundColor: customerColors.surface.soft },
  secondaryText: { fontFamily: 'Inter-Medium', fontSize: 15, color: customerColors.charcoal.soft },
});
