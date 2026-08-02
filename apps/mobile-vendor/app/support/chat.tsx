// Live chat with Tesserix support for chefs. Same shared SupportChatView the
// customer app uses, but the API proxy re-scopes chef-role callers to otto's
// homechef-vendor tenant, so these threads queue in the vendor lane of the
// admin inbox and route to vendor-specific answers. "Create a support ticket"
// converts the chat into a durable ticket and lands on its detail screen.
import { useMemo, useRef } from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import * as SecureStore from 'expo-secure-store';

import {
  createSupportClient,
  secureStoreKV,
  SupportChatView,
  useSupportChat,
  type IntakeReason,
  type SupportPalette,
  type SupportTicketContext,
} from '@homechef/mobile-shared/support';
import { theme } from '@homechef/mobile-shared/theme';
import { refreshSession } from '@homechef/mobile-shared/auth';
import { useToast } from '@homechef/mobile-shared/ui';
import { useAuthStore } from '../../store/auth-store';
import { useCreateTicket } from '../../hooks/useSupport';

// Matches otto's TenantReasons["homechef-vendor"] whitelist — chef-side
// topics, not customer ones.
const VENDOR_REASONS: IntakeReason[] = [
  { value: 'general_question', label: 'Ask a quick question', requiresStatus: false },
  { value: 'payout_issue', label: 'Payouts & earnings' },
  { value: 'order_management', label: 'Managing orders' },
  { value: 'menu_help', label: 'Menu & pricing' },
  { value: 'verification_docs', label: 'Verification & documents' },
  { value: 'account_issue', label: 'Account / login issue' },
  { value: 'other', label: 'Something else' },
];

const SESSION_KEY = 'otto_homechef_vendor_support_session';

export default function VendorSupportChatScreen() {
  const user = useAuthStore((s) => s.user);
  const { show: showToast } = useToast();
  const createTicket = useCreateTicket();
  const creating = useRef(false);

  const client = useMemo(
    () =>
      createSupportClient({
        baseUrl: process.env.EXPO_PUBLIC_API_URL!,
        // The vendor base URL already ends in /v1 (unlike the customer app's
        // …/api), so this path must not repeat it.
        basePath: '/support/chat',
        getToken: async () => useAuthStore.getState().accessToken,
        refreshToken: refreshSession,
        onUnauthorized: () => {
          useAuthStore.getState().logout();
        },
        loadSessionToken: () => SecureStore.getItemAsync(SESSION_KEY),
        saveSessionToken: (t) => SecureStore.setItemAsync(SESSION_KEY, t),
      }),
    [],
  );

  const chat = useSupportChat({ client, storage: secureStoreKV });

  // Vendor app carries actions in ink (persimmon retired as accent here);
  // own bubbles stay ink so the single Send action reads as THE action.
  const palette = useMemo<SupportPalette>(
    () => ({
      background: theme.colors.paper,
      surface: theme.colors.bone,
      bubbleOwn: theme.colors.ink.DEFAULT,
      textOnOwn: theme.colors.paper,
      text: theme.colors.ink.DEFAULT,
      textSecondary: theme.colors.ink.soft,
      border: theme.colors.mist.DEFAULT,
      primary: theme.colors.ink.DEFAULT,
      onPrimary: theme.colors.paper,
      danger: theme.colors.destructive.DEFAULT,
    }),
    [],
  );

  const defaults = useMemo(
    () => ({
      name: [user?.firstName, user?.lastName].filter(Boolean).join(' ') || undefined,
      email: user?.email ?? undefined,
    }),
    [user?.firstName, user?.lastName, user?.email],
  );

  const onCreateTicket = (ctx: SupportTicketContext) => {
    if (creating.current) return;
    creating.current = true;
    createTicket.mutate(
      {
        category: 'other',
        subject: ctx.subject || 'Support chat follow-up',
        description: ctx.transcript || '(no messages yet)',
        conversationId: ctx.conversationId,
      },
      {
        onSuccess: (ticket) => {
          showToast({ message: `Ticket ${ticket.ticketNumber} created`, tone: 'success' });
          router.push(`/support/${ticket.id}`);
        },
        onError: () => {
          showToast({ message: "Couldn't create the ticket. Please try again.", tone: 'error' });
        },
        onSettled: () => {
          creating.current = false;
        },
      },
    );
  };

  return (
    <SafeAreaView style={styles.fill} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={8}
          style={styles.back}
        >
          <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>Chat with support</Text>
        <View style={styles.back} />
      </View>
      <View style={styles.fill}>
        <SupportChatView
          chat={chat}
          palette={palette}
          reasons={VENDOR_REASONS}
          defaults={defaults}
          onCreateTicket={onCreateTicket}
          introTitle="How can we help your kitchen?"
          introSubtitle="Message the Fe3dr team — payouts, orders, menu, verification."
          composerPlaceholder="Type a message…"
          statusPlaceholder="e.g. Payout for last week not received"
        />
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1, backgroundColor: theme.colors.paper },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 8,
    paddingVertical: 8,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  back: { width: 40, height: 40, alignItems: 'center', justifyContent: 'center' },
  title: { fontSize: 17, fontWeight: '600', color: theme.colors.ink.DEFAULT },
});
