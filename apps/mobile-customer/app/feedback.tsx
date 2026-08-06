// Send feedback / share an idea. What a customer writes here is filed straight
// onto the product backlog as a labelled issue, so nothing is lost in a mailbox.

import { useState } from 'react';
import { ActivityIndicator, Pressable, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { Lightbulb, MessageSquare } from 'lucide-react-native';

import { customerColors } from '@homechef/mobile-shared/theme';
import { KeyboardAwareScrollView, useAlert } from '@homechef/mobile-shared/ui';
import {
  FEEDBACK_MESSAGE_MAX,
  validateFeedback,
  type FeedbackKind,
} from '@homechef/mobile-shared/feedback';
import { ScreenHeader } from '../components/ScreenHeader';
import { useSendFeedback } from '../hooks/useSendFeedback';

const KINDS: Array<{ value: FeedbackKind; label: string; hint: string; icon: typeof MessageSquare }> = [
  {
    value: 'feedback',
    label: 'Feedback',
    hint: 'Something felt off, confusing or broken.',
    icon: MessageSquare,
  },
  {
    value: 'idea',
    label: 'Idea',
    hint: 'Something you wish the app could do.',
    icon: Lightbulb,
  },
];

export default function FeedbackScreen() {
  const router = useRouter();
  const { showAlert } = useAlert();
  const send = useSendFeedback();

  const [kind, setKind] = useState<FeedbackKind>('feedback');
  const [title, setTitle] = useState('');
  const [message, setMessage] = useState('');

  const error = validateFeedback({ title, message });
  const remaining = FEEDBACK_MESSAGE_MAX - message.trim().length;

  function onSubmit() {
    if (error) {
      showAlert('Almost there', error);
      return;
    }
    send.mutate(
      { kind, title, message },
      {
        onSuccess: (receipt) => {
          showAlert(
            kind === 'idea' ? 'Idea received' : 'Thank you',
            `We've logged this for the team as #${receipt.reference}. If we build it, you'll see it in an update.`,
          );
          router.back();
        },
        onError: () =>
          showAlert(
            "Couldn't send that",
            'Your message could not be sent right now. Please check your connection and try again.',
          ),
      },
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-canvas" edges={['top', 'left', 'right']}>
      <ScreenHeader title="Send feedback" />

      <KeyboardAwareScrollView contentContainerStyle={{ padding: 16, paddingBottom: 48, gap: 20 }}>
        <Text className="text-sm text-charcoal-soft leading-5">
          Tell us what's working, what isn't, or what you'd like to see. A real person reads every
          one of these.
        </Text>

        <View className="gap-2">
          <Text className="text-xs font-semibold tracking-wide text-charcoal-soft">
            WHAT IS THIS?
          </Text>
          <View className="flex-row gap-3">
            {KINDS.map(({ value, label, hint, icon: Icon }) => {
              const selected = kind === value;
              return (
                <Pressable
                  key={value}
                  onPress={() => setKind(value)}
                  accessibilityRole="radio"
                  accessibilityState={{ selected }}
                  accessibilityLabel={label}
                  // No border at all — selection is a FILL. Any border on a
                  // rounded shape is resolved to whole dp by Android and renders
                  // rough at the corners, which is visible even when the border
                  // colour matches the fill. Two fills also keep both cards on
                  // an identical box model, so selecting one cannot shift its
                  // size by a pixel.
                  className={`flex-1 min-h-[88px] rounded-lg px-3 py-3 gap-1 ${
                    selected ? 'bg-coral-tint' : 'bg-surface-soft'
                  }`}
                >
                  <Icon
                    size={18}
                    color={selected ? customerColors.coral.DEFAULT : customerColors.charcoal.soft}
                  />
                  <Text
                    className={`text-sm font-semibold ${selected ? 'text-coral' : 'text-charcoal'}`}
                  >
                    {label}
                  </Text>
                  <Text className="text-xs text-charcoal-soft leading-4">{hint}</Text>
                </Pressable>
              );
            })}
          </View>
        </View>

        <View className="gap-2">
          <Text className="text-xs font-semibold tracking-wide text-charcoal-soft">TITLE</Text>
          <TextInput
            value={title}
            onChangeText={setTitle}
            placeholder={kind === 'idea' ? 'Dark mode' : 'The receipt is hard to read'}
            placeholderTextColor={customerColors.charcoal.soft}
            maxLength={200}
            accessibilityLabel="Title"
            className="min-h-[48px] rounded-lg border border-hairline bg-white px-3 text-base text-charcoal"
          />
        </View>

        <View className="gap-2">
          <Text className="text-xs font-semibold tracking-wide text-charcoal-soft">DETAILS</Text>
          <TextInput
            value={message}
            onChangeText={setMessage}
            placeholder="What happened, or what would you like to see?"
            placeholderTextColor={customerColors.charcoal.soft}
            multiline
            textAlignVertical="top"
            maxLength={FEEDBACK_MESSAGE_MAX}
            accessibilityLabel="Details"
            className="min-h-[140px] rounded-lg border border-hairline bg-white px-3 py-3 text-base text-charcoal"
          />
          <Text className="text-xs text-charcoal-soft">
            {remaining < 200 ? `${remaining} characters left` : 'A sentence or two is plenty.'}
          </Text>
        </View>

        <Pressable
          onPress={onSubmit}
          disabled={send.isPending}
          accessibilityRole="button"
          accessibilityLabel="Send"
          accessibilityState={{ disabled: send.isPending }}
          className={`min-h-[48px] flex-row items-center justify-center rounded-lg bg-coral ${
            send.isPending ? 'opacity-60' : ''
          }`}
        >
          {send.isPending ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text className="text-base font-semibold text-white">Send</Text>
          )}
        </Pressable>

        <Text className="text-xs text-charcoal-soft leading-4">
          We log your app version and device platform with this so we can reproduce issues. Your
          email address is never published.
        </Text>
      </KeyboardAwareScrollView>
    </SafeAreaView>
  );
}
