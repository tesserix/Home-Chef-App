import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { useToast } from '@homechef/mobile-shared/ui';
import {
  CATEGORY_LABEL,
  useAddMessage,
  useCloseTicket,
  useTicket,
  type SupportMessage,
} from '../../hooks/useSupport';
import { TicketStatusChip } from '../../components/vendor/TicketStatusChip';
import { TicketStatusStepper } from '../../components/vendor/TicketStatusStepper';

function formatTime(iso: string): string {
  try {
    return new Date(iso).toLocaleString('en-IN', {
      day: 'numeric',
      month: 'short',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return iso;
  }
}

// One message bubble. The chef's own messages (sender === reporter) align
// right with an ink fill; support replies align left on a bone surface.
function MessageBubble({
  message,
  isMine,
}: {
  message: SupportMessage;
  isMine: boolean;
}) {
  return (
    <View
      style={[
        styles.bubbleRow,
        isMine ? styles.bubbleRowMine : styles.bubbleRowTheirs,
      ]}
    >
      <View
        style={[
          styles.bubble,
          isMine ? styles.bubbleMine : styles.bubbleTheirs,
        ]}
      >
        {!isMine ? (
          <Text style={styles.bubbleSender}>
            {message.senderName?.trim() || 'Support'}
          </Text>
        ) : null}
        <Text style={[styles.bubbleText, isMine && styles.bubbleTextMine]}>
          {message.content}
        </Text>
        <Text style={[styles.bubbleTime, isMine && styles.bubbleTimeMine]}>
          {formatTime(message.createdAt)}
        </Text>
      </View>
    </View>
  );
}

export default function TicketDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const ticketId = typeof id === 'string' ? id : '';
  const { data: ticket, isLoading, isError } = useTicket(ticketId);
  const addMessage = useAddMessage(ticketId);
  const closeTicket = useCloseTicket(ticketId);
  const { show: showToast } = useToast();

  const [reply, setReply] = useState('');

  async function handleSend(): Promise<void> {
    const text = reply.trim();
    if (text.length === 0) return;
    try {
      await addMessage.mutateAsync(text);
      setReply('');
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } } | null)?.response
          ?.data?.error ??
        (err instanceof Error ? err.message : 'Reply failed.');
      showToast({ message: msg, tone: 'error' });
    }
  }

  function confirmClose(): void {
    Alert.alert('Close this ticket?', 'You can always open a new one later.', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Close ticket',
        style: 'destructive',
        onPress: async () => {
          try {
            await closeTicket.mutateAsync();
            showToast({ message: 'Ticket closed.', tone: 'success' });
          } catch (err: unknown) {
            const msg =
              (err as { response?: { data?: { error?: string } } } | null)
                ?.response?.data?.error ??
              (err instanceof Error ? err.message : 'Could not close ticket.');
            showToast({ message: msg, tone: 'error' });
          }
        },
      },
    ]);
  }

  if (isLoading) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <View style={styles.center}>
          <ActivityIndicator size="large" color={theme.colors.ink.DEFAULT} />
        </View>
      </SafeAreaView>
    );
  }

  if (isError || !ticket) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <View style={styles.commandBar}>
          <Pressable
            onPress={() => router.back()}
            hitSlop={8}
            accessibilityRole="button"
          >
            <ChevronLeft
              size={26}
              color={theme.colors.ink.DEFAULT}
              strokeWidth={1.75}
            />
          </Pressable>
        </View>
        <View style={styles.center}>
          <Text style={styles.muted}>This ticket couldn't be loaded.</Text>
        </View>
      </SafeAreaView>
    );
  }

  const messages = ticket.messages ?? [];
  const isClosed = ticket.status === 'closed';
  const showClose = ticket.status !== 'closed' && ticket.status !== 'resolved';

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      >
        <View style={styles.commandBar}>
          <Pressable
            onPress={() => router.back()}
            hitSlop={8}
            accessibilityRole="button"
            accessibilityLabel="Back"
          >
            <ChevronLeft
              size={26}
              color={theme.colors.ink.DEFAULT}
              strokeWidth={1.75}
            />
          </Pressable>
          <Text style={styles.commandTitle} numberOfLines={1}>
            {ticket.ticketNumber ?? 'Ticket'}
          </Text>
        </View>

        <ScrollView
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          <View style={styles.headerRow}>
            <TicketStatusChip status={ticket.status} />
            {ticket.category ? (
              <Text style={styles.category}>
                {CATEGORY_LABEL[ticket.category] ?? ticket.category}
              </Text>
            ) : null}
          </View>
          <Text style={styles.subject}>{ticket.subject}</Text>

          <View style={styles.stepperWrap}>
            <TicketStatusStepper status={ticket.status} />
          </View>

          {ticket.description ? (
            <View style={styles.descBlock}>
              <Text style={styles.descLabel}>YOUR REPORT</Text>
              <Text style={styles.descBody}>{ticket.description}</Text>
            </View>
          ) : null}

          <Text style={styles.transcriptLabel}>CONVERSATION</Text>
          {messages.length === 0 ? (
            <Text style={styles.noMessages}>
              No replies yet. Our team will respond here.
            </Text>
          ) : (
            messages.map((m) => (
              <MessageBubble
                key={m.id}
                message={m}
                isMine={m.senderId === ticket.reporterId}
              />
            ))
          )}

          {showClose ? (
            <Pressable
              onPress={confirmClose}
              disabled={closeTicket.isPending}
              accessibilityRole="button"
              accessibilityLabel="Close ticket"
              style={styles.closeWrap}
            >
              <Text style={styles.closeLabel}>
                {closeTicket.isPending ? 'Closing…' : 'Close ticket'}
              </Text>
            </Pressable>
          ) : null}
        </ScrollView>

        {/* Reply composer — hidden once the ticket is closed (the API rejects
            messages on a closed ticket). */}
        {!isClosed ? (
          <View style={styles.composer}>
            <TextInput
              value={reply}
              onChangeText={setReply}
              placeholder="Write a reply…"
              placeholderTextColor={theme.colors.ink.muted}
              style={styles.replyInput}
              multiline
              editable={!addMessage.isPending}
              maxLength={2000}
            />
            <Pressable
              onPress={handleSend}
              disabled={addMessage.isPending || reply.trim().length === 0}
              accessibilityRole="button"
              accessibilityLabel="Send reply"
            >
              <View
                style={[
                  styles.sendBtn,
                  (addMessage.isPending || reply.trim().length === 0) && {
                    opacity: 0.4,
                  },
                ]}
              >
                <Text style={styles.sendLabel}>
                  {addMessage.isPending ? '…' : 'Send'}
                </Text>
              </View>
            </Pressable>
          </View>
        ) : null}
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
    gap: theme.spacing[2],
  },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 24,
    lineHeight: 28,
    letterSpacing: -0.2,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  scroll: { flex: 1 },
  scrollContent: {
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[8],
  },

  headerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    marginBottom: theme.spacing[2],
  },
  category: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
  },
  subject: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.h2.size,
    color: theme.colors.ink.DEFAULT,
    lineHeight: 26,
    marginBottom: theme.spacing[5],
  },

  stepperWrap: {
    paddingVertical: theme.spacing[2],
    marginBottom: theme.spacing[5],
  },

  descBlock: {
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
    marginBottom: theme.spacing[6],
    gap: 6,
  },
  descLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    letterSpacing: 1.2,
    color: theme.colors.ink.muted,
  },
  descBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    lineHeight: 21,
  },

  transcriptLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[3],
  },
  noMessages: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    fontStyle: 'italic',
  },

  bubbleRow: {
    flexDirection: 'row',
    marginBottom: theme.spacing[2],
  },
  bubbleRowMine: { justifyContent: 'flex-end' },
  bubbleRowTheirs: { justifyContent: 'flex-start' },
  bubble: {
    maxWidth: '82%',
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    gap: 4,
  },
  bubbleMine: { backgroundColor: theme.colors.ink.DEFAULT },
  bubbleTheirs: {
    backgroundColor: theme.colors.bone,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
  },
  bubbleSender: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.soft,
  },
  bubbleText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    lineHeight: 20,
  },
  bubbleTextMine: { color: theme.colors.paper },
  bubbleTime: {
    fontFamily: 'Inter',
    fontSize: 10,
    color: theme.colors.ink.muted,
  },
  bubbleTimeMine: { color: theme.colors.mist.DEFAULT },

  closeWrap: {
    alignSelf: 'center',
    paddingVertical: theme.spacing[4],
    marginTop: theme.spacing[4],
  },
  closeLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.destructive.DEFAULT,
    textDecorationLine: 'underline',
  },

  composer: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    gap: theme.spacing[2],
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[3],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    backgroundColor: theme.colors.paper,
  },
  replyInput: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    maxHeight: 120,
    minHeight: theme.touchTarget.vendor,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.strong,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: 12,
    paddingTop: 10,
    paddingBottom: 10,
  },
  sendBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[4],
    height: theme.touchTarget.vendor,
    alignItems: 'center',
    justifyContent: 'center',
  },
  sendLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
  },

  center: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
  },
  muted: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.muted,
    textAlign: 'center',
  },
});
