import {
  ActivityIndicator,
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
            <Pressable onPress={() => router.replace('/(tabs)')} style={styles.secondary}>
              <Text style={styles.secondaryText}>Go home</Text>
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

            {preview.joinable ? (
              <Pressable onPress={accept} disabled={join.isPending} style={styles.primary}>
                {join.isPending ? (
                  <ActivityIndicator color="#fff" />
                ) : (
                  <Text style={styles.primaryText}>Join the order</Text>
                )}
              </Pressable>
            ) : (
              <Text style={styles.closed}>This group order is no longer open to join.</Text>
            )}
            <Pressable onPress={() => router.replace('/(tabs)')} style={styles.secondary}>
              <Text style={styles.secondaryText}>Not now</Text>
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
    borderRadius: 12,
    backgroundColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: 40,
    marginTop: 12,
  },
  primaryText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: '#fff' },
  secondary: { paddingVertical: 10 },
  secondaryText: { fontFamily: 'Inter-Medium', fontSize: 15, color: customerColors.charcoal.soft },
});
