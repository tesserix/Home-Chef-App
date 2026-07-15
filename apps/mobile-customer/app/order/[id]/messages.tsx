// In-app messaging (#53) — admin-mediated, order-scoped chat. The customer's
// messages are reviewed by Fe3dr support before reaching the chef (no direct
// chef↔customer channel); relayed replies arrive here. Polls for new messages.

import { useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams } from 'expo-router';
import { Send, ShieldCheck } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ScreenHeader } from '../../../components/ScreenHeader';
import { useOrderMessages, useSendMessage, type Message } from '../../../hooks/useMessaging';

function timeLabel(iso: string): string {
  try {
    return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  } catch {
    return '';
  }
}

export default function MessagesScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const orderId = id ?? '';
  const { data: messages = [], isLoading } = useOrderMessages(orderId);
  const send = useSendMessage(orderId);
  const [text, setText] = useState('');
  const scrollRef = useRef<ScrollView>(null);

  const onSend = () => {
    const content = text.trim();
    if (!content || send.isPending) return;
    // Clear the input only AFTER the send succeeds. Clearing up-front lost the
    // customer's message on any failure (network / 500 / PII block) with no error.
    send.mutate(content, {
      onSuccess: () => {
        setText('');
        setTimeout(() => scrollRef.current?.scrollToEnd({ animated: true }), 50);
      },
      onError: () => {
        Alert.alert(
          'Message not sent',
          'Something went wrong sending your message. Please check your connection and try again.',
        );
      },
    });
  };

  return (
    <SafeAreaView edges={['top', 'left', 'right', 'bottom']} className="flex-1 bg-canvas">
      <ScreenHeader title="Messages" />
      <KeyboardAvoidingView
        className="flex-1"
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
        keyboardVerticalOffset={Platform.OS === 'ios' ? 90 : 0}
      >
        {/* Mediation notice */}
        <View className="flex-row items-center gap-2 border-b border-hairline bg-surface-soft px-4 py-2.5">
          <ShieldCheck size={15} color={customerColors.charcoal.soft} />
          <Text className="flex-1 text-xs text-charcoal-soft">
            Messages are reviewed by Fe3dr support before reaching the chef. Please don't share phone
            numbers or emails.
          </Text>
        </View>

        {isLoading ? (
          <View className="flex-1 items-center justify-center">
            <ActivityIndicator color={customerColors.charcoal.soft} />
          </View>
        ) : (
          <ScrollView
            ref={scrollRef}
            className="flex-1"
            contentContainerStyle={{ padding: 16, gap: 8 }}
            onContentSizeChange={() => scrollRef.current?.scrollToEnd({ animated: false })}
          >
            {messages.length === 0 ? (
              <Text className="mt-8 text-center text-charcoal-soft">
                No messages yet. Send a note about your order — allergies, timing, anything.
              </Text>
            ) : (
              messages.map((m: Message) => {
                const mine = m.senderRole === 'customer';
                return (
                  <View key={m.id} className={mine ? 'items-end' : 'items-start'}>
                    <View
                      className={`max-w-[80%] rounded-2xl px-3.5 py-2.5 ${mine ? 'bg-coral' : 'bg-surface-soft'}`}
                    >
                      {!mine && (
                        <Text className="mb-0.5 text-[11px] font-semibold text-charcoal-soft">
                          {m.senderRole === 'chef' ? 'Chef (via support)' : 'Fe3dr support'}
                        </Text>
                      )}
                      <Text className={mine ? 'text-white' : 'text-charcoal'}>{m.content}</Text>
                    </View>
                    <Text className="mt-0.5 px-1 text-[10px] text-charcoal-soft">
                      {timeLabel(m.createdAt)}
                      {mine && m.relayStatus === 'pending' ? ' · under review' : ''}
                    </Text>
                  </View>
                );
              })
            )}
          </ScrollView>
        )}

        {/* Composer */}
        <View className="flex-row items-end gap-2 border-t border-hairline bg-canvas px-3 py-2">
          <TextInput
            className="max-h-28 flex-1 rounded-2xl bg-surface-soft px-4 py-2.5 text-base text-charcoal"
            placeholder="Message…"
            accessibilityLabel="Message"
            placeholderTextColor={customerColors.charcoal.soft}
            value={text}
            onChangeText={setText}
            multiline
          />
          <Pressable
            onPress={onSend}
            disabled={!text.trim() || send.isPending}
            accessibilityRole="button"
            accessibilityLabel="Send message"
            className={`h-11 w-11 items-center justify-center rounded-full ${text.trim() ? 'bg-coral' : 'bg-surface-soft'}`}
          >
            {send.isPending ? (
              <ActivityIndicator color="#FFFFFF" size="small" />
            ) : (
              <Send size={18} color={text.trim() ? '#FFFFFF' : customerColors.charcoal.soft} />
            )}
          </Pressable>
        </View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}
