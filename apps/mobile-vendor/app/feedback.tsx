// Send feedback / share an idea. What a chef writes here is filed straight onto
// the product backlog as a labelled issue, so nothing is lost in a mailbox.

import { useState } from 'react';
import {
  ActivityIndicator,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft, Lightbulb, MessageSquare } from 'lucide-react-native';
import { useTranslation } from 'react-i18next';

import { theme } from '@homechef/mobile-shared/theme';
import { useAlert } from '@homechef/mobile-shared/ui';
import {
  FEEDBACK_MESSAGE_MAX,
  validateFeedback,
  type FeedbackKind,
} from '@homechef/mobile-shared/feedback';
import { useSendFeedback } from '../hooks/useSendFeedback';

const KINDS: Array<{ value: FeedbackKind; labelKey: string; hintKey: string; Icon: typeof MessageSquare }> = [
  { value: 'feedback', labelKey: 'feedback.kindFeedback', hintKey: 'feedback.kindFeedbackHint', Icon: MessageSquare },
  { value: 'idea', labelKey: 'feedback.kindIdea', hintKey: 'feedback.kindIdeaHint', Icon: Lightbulb },
];

export default function FeedbackScreen() {
  const { t } = useTranslation();
  const { showAlert } = useAlert();
  const send = useSendFeedback();

  const [kind, setKind] = useState<FeedbackKind>('feedback');
  const [title, setTitle] = useState('');
  const [message, setMessage] = useState('');

  const error = validateFeedback({ title, message });

  function onSubmit(): void {
    if (error) {
      showAlert(t('feedback.almostThere'), error);
      return;
    }
    send.mutate(
      { kind, title, message },
      {
        onSuccess: (receipt) => {
          showAlert(t('feedback.sentTitle'), t('feedback.sentBody', { reference: receipt.reference }));
          router.back();
        },
        onError: () => showAlert(t('feedback.failedTitle'), t('feedback.failedBody')),
      },
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.commandBar}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel={t('common.close')}
          android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
        >
          {({ pressed }) => (
            <View style={[styles.backButton, pressed && Platform.OS === 'ios' && { opacity: 0.6 }]}>
              <ChevronLeft size={22} color={theme.colors.ink.DEFAULT} />
            </View>
          )}
        </Pressable>
        <Text style={styles.commandTitle}>{t('feedback.title')}</Text>
      </View>

      <ScrollView contentContainerStyle={styles.scrollContent} keyboardShouldPersistTaps="handled">
        <Text style={styles.subtitle}>{t('feedback.subtitle')}</Text>

        <Text style={styles.sectionLabel}>{t('feedback.kindLabel').toUpperCase()}</Text>
        <View style={styles.kindRow}>
          {KINDS.map(({ value, labelKey, hintKey, Icon }) => {
            const active = kind === value;
            return (
              <Pressable
                key={value}
                onPress={() => setKind(value)}
                accessibilityRole="radio"
                accessibilityState={{ selected: active }}
                accessibilityLabel={t(labelKey)}
                style={[styles.kindCard, active && styles.kindCardActive]}
              >
                <Icon size={18} color={active ? theme.colors.herb.DEFAULT : theme.colors.ink.muted} />
                <Text style={[styles.kindLabel, active && styles.kindLabelActive]}>{t(labelKey)}</Text>
                <Text style={styles.kindHint}>{t(hintKey)}</Text>
              </Pressable>
            );
          })}
        </View>

        <Text style={styles.sectionLabel}>{t('feedback.titleLabel').toUpperCase()}</Text>
        <TextInput
          value={title}
          onChangeText={setTitle}
          placeholder={t('feedback.titlePlaceholder')}
          placeholderTextColor={theme.colors.ink.muted}
          maxLength={200}
          accessibilityLabel={t('feedback.titleLabel')}
          style={styles.input}
        />

        <Text style={styles.sectionLabel}>{t('feedback.detailsLabel').toUpperCase()}</Text>
        <TextInput
          value={message}
          onChangeText={setMessage}
          placeholder={t('feedback.detailsPlaceholder')}
          placeholderTextColor={theme.colors.ink.muted}
          multiline
          textAlignVertical="top"
          maxLength={FEEDBACK_MESSAGE_MAX}
          accessibilityLabel={t('feedback.detailsLabel')}
          style={[styles.input, styles.textarea]}
        />

        <Pressable
          onPress={onSubmit}
          disabled={send.isPending}
          accessibilityRole="button"
          accessibilityLabel={t('feedback.send')}
          accessibilityState={{ disabled: send.isPending }}
          style={[styles.submit, send.isPending && styles.submitBusy]}
        >
          {send.isPending ? (
            <ActivityIndicator color={theme.colors.paper} />
          ) : (
            <Text style={styles.submitLabel}>{t('feedback.send')}</Text>
          )}
        </Pressable>

        <Text style={styles.privacy}>{t('feedback.privacyNote')}</Text>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    gap: theme.spacing[2],
  },
  backButton: { minWidth: 44, minHeight: 44, alignItems: 'flex-start', justifyContent: 'center' },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },
  scrollContent: { paddingHorizontal: theme.spacing[4], paddingBottom: theme.spacing[10] },
  subtitle: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    lineHeight: 20,
    color: theme.colors.ink.muted,
    paddingBottom: theme.spacing[4],
  },
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.6,
    color: theme.colors.ink.muted,
    paddingBottom: theme.spacing[2],
  },
  kindRow: { flexDirection: 'row', gap: theme.spacing[3], paddingBottom: theme.spacing[4] },
  kindCard: {
    flex: 1,
    minHeight: 92,
    gap: theme.spacing[1],
    padding: theme.spacing[3],
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    backgroundColor: theme.colors.paper,
  },
  kindCardActive: { borderColor: theme.colors.herb.DEFAULT },
  kindLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  kindLabelActive: { color: theme.colors.herb.DEFAULT },
  kindHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    lineHeight: 16,
    color: theme.colors.ink.muted,
  },
  input: {
    minHeight: 48,
    borderRadius: theme.radius.md,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    backgroundColor: theme.colors.paper,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    marginBottom: theme.spacing[4],
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  textarea: { minHeight: 140, paddingTop: theme.spacing[3] },
  submit: {
    minHeight: 48,
    borderRadius: theme.radius.md,
    backgroundColor: theme.colors.herb.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },
  submitBusy: { opacity: 0.6 },
  submitLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
  },
  privacy: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    lineHeight: 16,
    color: theme.colors.ink.muted,
    paddingTop: theme.spacing[3],
  },
});
