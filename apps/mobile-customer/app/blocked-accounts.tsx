// Blocked accounts — review and undo blocks.
//
// App Review guideline 1.2 asks for the ability to block abusive users. Blocking
// happens inline on the content itself (the ⋯ menu on a post or review), which
// is where people actually reach for it; this screen exists so the action is
// reversible and auditable, which a reviewer will check.

import { ActivityIndicator, Alert, Pressable, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { ScrollView } from 'react-native';
import { ShieldOff } from 'lucide-react-native';

import { customerColors } from '@homechef/mobile-shared/theme';
import { EmptyState } from '@homechef/mobile-shared/ui';
import { ScreenHeader } from '../components/ScreenHeader';
import { useBlockedAccounts, useUnblockUser } from '../hooks/useModeration';

export default function BlockedAccountsScreen() {
  const { data: blocks, isLoading, isError, refetch } = useBlockedAccounts();
  const unblock = useUnblockUser();

  function confirmUnblock(userId: string, name: string): void {
    Alert.alert(
      `Unblock ${name}?`,
      'Their posts and reviews will show up again, and they will be able to message you about orders.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Unblock',
          onPress: () =>
            unblock.mutate(userId, {
              onError: () =>
                Alert.alert('Could not unblock', 'Please check your connection and try again.'),
            }),
        },
      ],
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      <ScreenHeader title="Blocked accounts" />

      {isLoading ? (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator color={customerColors.charcoal.DEFAULT} />
        </View>
      ) : isError ? (
        <View className="flex-1 items-center justify-center px-8 gap-3">
          <Text className="text-base text-charcoal text-center">
            Could not load your blocked accounts.
          </Text>
          <Pressable
            onPress={() => refetch()}
            accessibilityRole="button"
            accessibilityLabel="Try again"
            className="min-h-[48px] px-5 items-center justify-center rounded-lg bg-coral"
          >
            <Text className="text-base font-semibold text-white">Try again</Text>
          </Pressable>
        </View>
      ) : !blocks || blocks.length === 0 ? (
        <EmptyState
          icon={<ShieldOff size={36} color={customerColors.charcoal.soft} />}
          title="No blocked accounts"
          body="When you block someone, they show up here so you can undo it."
        />
      ) : (
        <ScrollView contentContainerStyle={{ paddingBottom: 40 }}>
          <Text className="px-4 pt-3 pb-2 text-sm text-charcoal-soft">
            You will not see posts or reviews from these accounts, and they cannot message you.
          </Text>
          {blocks.map((b) => (
            <View
              key={b.userId}
              className="flex-row items-center gap-3 px-4 py-3 border-b border-hairline"
            >
              <View className="flex-1">
                <Text className="text-base font-medium text-charcoal" numberOfLines={1}>
                  {b.name}
                </Text>
                <Text className="text-xs text-charcoal-soft">
                  Blocked {new Date(b.createdAt).toLocaleDateString('en-IN', {
                    day: 'numeric',
                    month: 'short',
                    year: 'numeric',
                  })}
                </Text>
              </View>
              <Pressable
                onPress={() => confirmUnblock(b.userId, b.name)}
                disabled={unblock.isPending}
                accessibilityRole="button"
                accessibilityLabel={`Unblock ${b.name}`}
                className="min-h-[44px] px-4 items-center justify-center rounded-lg border border-hairline"
              >
                <Text className="text-sm font-semibold text-charcoal">Unblock</Text>
              </Pressable>
            </View>
          ))}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
