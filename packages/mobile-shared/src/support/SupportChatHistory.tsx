// A customer's or chef's past support chats, with the state of each made
// obvious at a glance. Anything not yet resolved stays visually loud so an
// unanswered thread is never lost in a list of closed ones.
import { ActivityIndicator, Pressable, StyleSheet, Text, View } from "react-native";

import type { SupportPalette } from "./SupportChatView";
import type { SupportConversation } from "./types";

/** How a thread reads to the person who raised it. */
export type ChatState = "waiting" | "with_agent" | "assistant" | "resolved";

export function chatState(c: SupportConversation): ChatState {
  if (c.status === "closed") return "resolved";
  if (c.status === "pending") return "waiting";
  // Active with needs_human means a person owns it; otherwise the bot does.
  return c.needs_human ? "with_agent" : "assistant";
}

const STATE_LABEL: Record<ChatState, string> = {
  waiting: "Waiting for an agent",
  with_agent: "With our team",
  assistant: "Answered by Otto",
  resolved: "Resolved",
};

/**
 * Colour grade per state, derived from the app's palette so each app keeps
 * its own accent. Unresolved states borrow the accent/danger end of the
 * scale; resolved settles into muted neutral.
 */
function stateColors(state: ChatState, palette: SupportPalette): { fg: string; bg: string } {
  switch (state) {
    case "waiting":
      return { fg: palette.danger, bg: `${palette.danger}1A` };
    case "with_agent":
      return { fg: palette.primary, bg: `${palette.primary}1A` };
    case "assistant":
      return { fg: palette.textSecondary, bg: `${palette.border}66` };
    case "resolved":
      return { fg: palette.textSecondary, bg: `${palette.border}40` };
  }
}

function relative(iso?: string): string {
  if (!iso) return "";
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return "";
  const min = Math.floor((Date.now() - then) / 60000);
  if (min < 1) return "just now";
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  const day = Math.floor(hr / 24);
  if (day < 7) return `${day}d ago`;
  return new Date(iso).toLocaleDateString("en-IN", { day: "numeric", month: "short" });
}

export interface SupportChatHistoryProps {
  conversations: SupportConversation[];
  palette: SupportPalette;
  loading?: boolean;
  /** Opens a thread. Unresolved ones can be continued; resolved are read-only. */
  onOpen: (conversation: SupportConversation) => void;
  title?: string;
}

export function SupportChatHistory({
  conversations,
  palette,
  loading,
  onOpen,
  title = "Your recent chats",
}: SupportChatHistoryProps) {
  if (loading) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={palette.primary} />
      </View>
    );
  }
  if (conversations.length === 0) return null;

  const unresolved = conversations.filter((c) => chatState(c) !== "resolved").length;

  return (
    <View style={styles.wrap}>
      <View style={styles.headerRow}>
        <Text style={[styles.title, { color: palette.text }]}>{title}</Text>
        {unresolved > 0 ? (
          <Text style={[styles.unresolved, { color: palette.danger }]}>
            {unresolved} still open
          </Text>
        ) : null}
      </View>

      {conversations.map((c) => {
        const state = chatState(c);
        const tone = stateColors(state, palette);
        return (
          <Pressable
            key={c.id}
            onPress={() => onOpen(c)}
            accessibilityRole="button"
            accessibilityLabel={`${STATE_LABEL[state]}: ${c.subject || c.case_id || "support chat"}`}
            style={[styles.row, { backgroundColor: palette.surface, borderColor: palette.border }]}
          >
            <View style={styles.rowMain}>
              <Text style={[styles.subject, { color: palette.text }]} numberOfLines={1}>
                {c.subject || "Support chat"}
              </Text>
              <Text style={[styles.meta, { color: palette.textSecondary }]} numberOfLines={1}>
                {c.case_id}
                {c.last_message_at ? ` · ${relative(c.last_message_at as string)}` : ""}
              </Text>
            </View>
            <View style={[styles.badge, { backgroundColor: tone.bg }]}>
              <Text style={[styles.badgeText, { color: tone.fg }]}>{STATE_LABEL[state]}</Text>
            </View>
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: 8, paddingHorizontal: 16, paddingTop: 8, paddingBottom: 4 },
  loading: { paddingVertical: 20, alignItems: "center" },
  headerRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  title: { fontSize: 13, fontWeight: "600" },
  unresolved: { fontSize: 12, fontWeight: "600" },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    borderRadius: 12,
    borderWidth: StyleSheet.hairlineWidth,
    paddingHorizontal: 12,
    paddingVertical: 10,
  },
  rowMain: { flex: 1 },
  subject: { fontSize: 14, fontWeight: "600" },
  meta: { fontSize: 12, marginTop: 1 },
  badge: { borderRadius: 999, paddingHorizontal: 8, paddingVertical: 4 },
  badgeText: { fontSize: 11, fontWeight: "700" },
});
