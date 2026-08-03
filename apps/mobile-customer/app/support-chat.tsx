// Chat with Tesserix support (Phase 5). Renders the shared SupportChatView
// wired to the HomeChef API support-chat proxy, which bridges to otto's
// homechef tenant. A Tesserix admin answers from the platform inbox; the
// conversation carries the signed-in customer's identity so they skip OTP.
import { useEffect, useMemo, useRef } from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useRouter } from "expo-router";
import { ChevronLeft } from "lucide-react-native";

import {
  createSupportClient,
  secureStoreKV,
  SupportChatView,
  useSupportChat,
  type IntakeReason,
  type SupportPalette,
  type SupportTicketContext,
} from "@homechef/mobile-shared/support";
import { customerColors } from "@homechef/mobile-shared/theme";
import { secureGet, secureSet } from "@homechef/mobile-shared/utils";
import { refreshSession } from "@homechef/mobile-shared/auth";
import { useToast } from "@homechef/mobile-shared/ui";
import { api } from "../lib/api";
import { useAuthStore } from "../store/auth-store";

// Matches otto's TenantReasons["homechef"] whitelist + the web widget
// (apps/web/src/features/support/OttoChat.tsx). No DOB — HomeChef identifies
// orders by the signed-in account. general_question skips the summary field.
const HOMECHEF_REASONS: IntakeReason[] = [
  { value: "general_question", label: "Ask a quick question", requiresStatus: false },
  { value: "order_tracking", label: "Order tracking / ETA" },
  { value: "delivery_issue", label: "Delivery problem" },
  { value: "refund", label: "Refund request" },
  { value: "chef_question", label: "Question about a chef or dish" },
  { value: "account_issue", label: "Account / login issue" },
  { value: "other", label: "Something else" },
];

const SESSION_KEY = "otto_homechef_support_session";

export default function SupportChatScreen() {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const { show: showToast } = useToast();
  const creatingTicket = useRef(false);

  // Converts the chat into a durable support ticket (visible to the admin
  // Support desk). The API converges on one ticket per conversation, so a
  // second tap — or the backend escalation hook racing us — returns the
  // already-created ticket instead of a duplicate.
  const createTicket = async (ctx: SupportTicketContext) => {
    if (creatingTicket.current) return;
    creatingTicket.current = true;
    try {
      const res = await api.post<{ ticketNumber: string }>("/support/tickets", {
        category: "other",
        subject: ctx.subject || "Support chat follow-up",
        description: ctx.transcript || "(no messages yet)",
        conversationId: ctx.conversationId,
      });
      showToast({
        message: `Ticket ${res.data.ticketNumber} created — our team will follow up.`,
        tone: "success",
      });
    } catch {
      showToast({
        message: "Couldn't create the ticket. Please try again in a moment.",
        tone: "error",
      });
    } finally {
      creatingTicket.current = false;
    }
  };

  const client = useMemo(
    () =>
      createSupportClient({
        baseUrl: process.env.EXPO_PUBLIC_API_URL!,
        basePath: "/v1/support/chat",
        getToken: async () => useAuthStore.getState().accessToken,
        refreshToken: refreshSession,
        onUnauthorized: () => {
          // Refresh already failed inside the client; tear the session down so
          // the layout auth guard routes back to login.
          useAuthStore.getState().logout();
        },
        loadSessionToken: () => secureGet(SESSION_KEY),
        saveSessionToken: (t) => secureSet(SESSION_KEY, t),
      }),
    [],
  );

  const chat = useSupportChat({ client, storage: secureStoreKV });

  // Past chats (resolved and not) so nothing raised is lost.
  useEffect(() => {
    void chat.refreshHistory();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Outgoing bubbles are neutral charcoal-on-soft, not coral — every message
  // the customer sends would otherwise repeat the accent. Coral stays reserved
  // for the single Send/submit action (`primary`).
  const palette = useMemo<SupportPalette>(
    () => ({
      background: customerColors.canvas,
      surface: customerColors.surface.soft,
      bubbleOwn: customerColors.charcoal.DEFAULT,
      textOnOwn: customerColors.canvas,
      text: customerColors.charcoal.DEFAULT,
      textSecondary: customerColors.charcoal.soft,
      border: customerColors.hairline,
      primary: customerColors.coral.DEFAULT,
      onPrimary: customerColors.canvas,
      danger: customerColors.destructive.DEFAULT,
    }),
    [],
  );

  const defaults = useMemo(
    () => ({
      name: [user?.firstName, user?.lastName].filter(Boolean).join(" ") || undefined,
      email: user?.email ?? undefined,
    }),
    [user?.firstName, user?.lastName, user?.email],
  );

  return (
    <SafeAreaView style={styles.fill} edges={["top", "left", "right", "bottom"]}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={8}
          style={styles.back}
        >
          <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>Chat with support</Text>
        <View style={styles.back} />
      </View>
      <View style={styles.fill}>
        <SupportChatView
          chat={chat}
          palette={palette}
          reasons={HOMECHEF_REASONS}
          defaults={defaults}
          onCreateTicket={createTicket}
          introTitle="How can we help?"
          introSubtitle="Message the Fe3dr support team — we'll get back to you here."
          composerPlaceholder="Type a message…"
          statusPlaceholder="e.g. Order #ORD-2041 stuck on 'preparing'"
        />
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1, backgroundColor: customerColors.canvas },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: 8,
    paddingVertical: 8,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  back: { width: 40, height: 40, alignItems: "center", justifyContent: "center" },
  title: { fontSize: 17, fontWeight: "600", color: customerColors.charcoal.DEFAULT },
});
