import { useState } from 'react';
import { ActivityIndicator, Alert, Pressable, StyleSheet, Text, TextInput, View } from 'react-native';
import { ChevronDown, ChevronUp, MessageSquare, Send, ShieldCheck } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { useOrderMessages, useSendOrderMessage } from '../../hooks/useOrderMessaging';

// Chef-side mediated message thread for one order (#53). Mirrors the vendor
// portal's OrderMessageThread. Collapsed by default and lazy: the query only
// runs once expanded, so an order screen nobody taps through costs no polling.
//
// The chef never gets the customer's number — support relays both directions —
// so this card is the only sanctioned channel, and the notice says so.

const MAX_LENGTH = 1000;

function fmtTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleTimeString('en-IN', { hour: 'numeric', minute: '2-digit' });
}

export function OrderMessageThread({ orderId }: { orderId: string }) {
  const [open, setOpen] = useState(false);
  const [text, setText] = useState('');
  const { data: messages = [], isLoading } = useOrderMessages(orderId, open);
  const send = useSendOrderMessage(orderId);

  function onSend(): void {
    const content = text.trim();
    if (!content || send.isPending) return;
    send.mutate(content, {
      onSuccess: (res) => {
        setText('');
        if (res.piiDetected) {
          Alert.alert(
            'Contact details removed',
            'Your message included personal information, so it was redacted before going to support for review.',
          );
        }
      },
      onError: () => Alert.alert('Could not send', 'Please try again.'),
    });
  }

  return (
    <View style={styles.card}>
      <Pressable
        onPress={() => setOpen((v) => !v)}
        accessibilityRole="button"
        accessibilityState={{ expanded: open }}
        accessibilityLabel={open ? 'Hide messages' : 'Show messages'}
      >
        {({ pressed }) => (
          <View style={[styles.headerRow, pressed && { opacity: 0.7 }]}>
            <View style={styles.headerLeft}>
              <MessageSquare size={16} color={theme.colors.ink.soft} />
              <Text style={styles.title}>Messages</Text>
              {messages.length > 0 ? (
                <Text style={styles.count}>{messages.length}</Text>
              ) : null}
            </View>
            {open ? (
              <ChevronUp size={16} color={theme.colors.ink.muted} />
            ) : (
              <ChevronDown size={16} color={theme.colors.ink.muted} />
            )}
          </View>
        )}
      </Pressable>

      {open ? (
        <>
          <View style={styles.notice}>
            <ShieldCheck size={13} color={theme.colors.ink.soft} />
            <Text style={styles.noticeText}>
              Support reviews your reply before the customer sees it. Don&apos;t share your phone
              number or address.
            </Text>
          </View>

          {isLoading ? (
            <View style={styles.loading}>
              <ActivityIndicator size="small" color={theme.colors.ink.muted} />
            </View>
          ) : messages.length === 0 ? (
            <Text style={styles.emptyText}>
              No messages yet. If the customer asks something about this order, it appears here.
            </Text>
          ) : (
            <View style={styles.thread}>
              {messages.map((m) => {
                const mine = m.senderRole === 'chef';
                return (
                  <View
                    key={m.id}
                    style={[styles.bubbleRow, mine ? styles.bubbleRowMine : styles.bubbleRowTheirs]}
                  >
                    <View style={[styles.bubble, mine ? styles.bubbleMine : styles.bubbleTheirs]}>
                      {!mine ? (
                        <Text style={styles.bubbleSender}>
                          {m.senderRole === 'customer' ? 'Customer · via support' : 'Support'}
                        </Text>
                      ) : null}
                      <Text style={mine ? styles.bubbleTextMine : styles.bubbleTextTheirs}>
                        {m.content}
                      </Text>
                      <Text style={mine ? styles.bubbleMetaMine : styles.bubbleMetaTheirs}>
                        {fmtTime(m.createdAt)}
                        {mine && m.relayStatus === 'pending' ? ' · under review' : ''}
                        {mine && m.relayStatus === 'blocked' ? ' · not delivered' : ''}
                      </Text>
                    </View>
                  </View>
                );
              })}
            </View>
          )}

          <View style={styles.composer}>
            <TextInput
              style={styles.input}
              value={text}
              onChangeText={setText}
              placeholder="Reply…"
              placeholderTextColor={theme.colors.ink.muted}
              multiline
              maxLength={MAX_LENGTH}
              accessibilityLabel="Message to the customer"
            />
            <Pressable
              onPress={onSend}
              disabled={!text.trim() || send.isPending}
              accessibilityRole="button"
              accessibilityLabel="Send message"
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.sendBtn,
                    (!text.trim() || send.isPending) && styles.sendBtnDisabled,
                    pressed && { opacity: 0.85 },
                  ]}
                >
                  {send.isPending ? (
                    <ActivityIndicator size="small" color={theme.colors.paper} />
                  ) : (
                    <Send size={16} color={theme.colors.paper} />
                  )}
                </View>
              )}
            </Pressable>
          </View>
        </>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    gap: theme.spacing[2],
    ...theme.shadow[1],
  },
  headerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 8,
    minHeight: 28,
  },
  headerLeft: { flexDirection: 'row', alignItems: 'center', gap: 8, flexShrink: 1 },
  title: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: theme.colors.ink.DEFAULT },
  count: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: theme.colors.ink.muted,
    fontVariant: ['tabular-nums'],
  },
  notice: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 6,
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
  },
  noticeText: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: 12,
    lineHeight: 17,
    color: theme.colors.ink.soft,
  },
  loading: { paddingVertical: theme.spacing[4], alignItems: 'center' },
  emptyText: { fontFamily: 'Inter', fontSize: 13, lineHeight: 18, color: theme.colors.ink.muted },
  thread: { gap: theme.spacing[2] },
  bubbleRow: { flexDirection: 'row' },
  bubbleRowMine: { justifyContent: 'flex-end' },
  bubbleRowTheirs: { justifyContent: 'flex-start' },
  bubble: {
    maxWidth: '85%',
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    gap: 2,
  },
  bubbleMine: { backgroundColor: theme.colors.ink.DEFAULT },
  bubbleTheirs: { backgroundColor: theme.colors.bone },
  bubbleSender: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    color: theme.colors.ink.muted,
  },
  bubbleTextMine: {
    fontFamily: 'Inter',
    fontSize: 14,
    lineHeight: 19,
    color: theme.colors.paper,
  },
  bubbleTextTheirs: {
    fontFamily: 'Inter',
    fontSize: 14,
    lineHeight: 19,
    color: theme.colors.ink.DEFAULT,
  },
  bubbleMetaMine: { fontFamily: 'Inter', fontSize: 10, color: theme.colors.mist.strong },
  bubbleMetaTheirs: { fontFamily: 'Inter', fontSize: 10, color: theme.colors.ink.muted },
  composer: { flexDirection: 'row', alignItems: 'flex-end', gap: 8, paddingTop: theme.spacing[1] },
  input: {
    flex: 1,
    minHeight: 44,
    maxHeight: 110,
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    fontFamily: 'Inter',
    fontSize: 14,
    color: theme.colors.ink.DEFAULT,
  },
  sendBtn: {
    width: 44,
    height: 44,
    borderRadius: theme.radius.full,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: theme.colors.ink.DEFAULT,
  },
  sendBtnDisabled: { backgroundColor: theme.colors.mist.strong },
});
