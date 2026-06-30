import { useEffect, useRef, useState } from 'react';
import {
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
import { router } from 'expo-router';
import { ChevronLeft } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { useFormDraft } from '@homechef/mobile-shared/hooks';
import { useToast } from '@homechef/mobile-shared/ui';
import {
  useCreateTicket,
  type TicketCategory,
  type TicketPriority,
} from '../../hooks/useSupport';

// Vendor-curated categories. Two of them ("Feature request", "Something
// else") map to the same backend category (`other`) but stay distinct chips
// so the chef picks the words that match their intent. Identity is the chip
// `id`, not the backend value.
const CATEGORY_CHIPS: {
  id: string;
  label: string;
  category: TicketCategory;
}[] = [
  { id: 'technical', label: 'App problem', category: 'technical' },
  { id: 'payment', label: 'Payments & payouts', category: 'payment_issue' },
  { id: 'account', label: 'Account & verification', category: 'account_issue' },
  { id: 'order', label: 'An order', category: 'order_issue' },
  { id: 'feature', label: 'Feature request', category: 'other' },
  { id: 'other', label: 'Something else', category: 'other' },
];

const PRIORITY_CHIPS: { value: TicketPriority; label: string }[] = [
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Normal' },
  { value: 'high', label: 'High' },
  { value: 'urgent', label: 'Urgent' },
];

interface SupportTicketDraft {
  categoryId: string | null;
  subject: string;
  description: string;
  priority: TicketPriority;
}

export default function NewTicketScreen() {
  const { show: showToast } = useToast();
  const create = useCreateTicket();
  const { ready, draft, saveDraft, clearDraft } =
    useFormDraft<SupportTicketDraft>('support-ticket-draft');

  const [categoryId, setCategoryId] = useState<string | null>(null);
  const [subject, setSubject] = useState('');
  const [description, setDescription] = useState('');
  const [priority, setPriority] = useState<TicketPriority>('medium');

  // Restore a saved draft once, when the async load resolves.
  const restored = useRef(false);
  useEffect(() => {
    if (!ready || restored.current || !draft) return;
    restored.current = true;
    setCategoryId(draft.categoryId);
    setSubject(draft.subject);
    setDescription(draft.description);
    setPriority(draft.priority);
  }, [ready, draft]);

  // Persist on every change (debounced in the hook). Guard until the initial
  // load resolves so we never overwrite a saved draft with the empty defaults.
  useEffect(() => {
    if (!ready) return;
    saveDraft({ categoryId, subject, description, priority });
  }, [ready, categoryId, subject, description, priority, saveDraft]);

  // Back-button dirty guard — a half-written ticket shouldn't vanish on a
  // stray tap. Discarding clears the persisted draft too.
  function handleBack(): void {
    const dirty =
      !!categoryId || subject.trim().length > 0 || description.trim().length > 0;
    if (!dirty) {
      router.back();
      return;
    }
    Alert.alert('Discard ticket?', "Your draft hasn't been sent.", [
      { text: 'Keep editing', style: 'cancel' },
      {
        text: 'Discard',
        style: 'destructive',
        onPress: () => {
          clearDraft();
          router.back();
        },
      },
    ]);
  }

  const selectedCategory = CATEGORY_CHIPS.find((c) => c.id === categoryId);
  const canSubmit =
    !!selectedCategory &&
    subject.trim().length > 0 &&
    description.trim().length > 0 &&
    !create.isPending;

  async function handleSubmit(): Promise<void> {
    if (!selectedCategory) {
      showToast({ message: 'Pick what this is about.', tone: 'error' });
      return;
    }
    if (subject.trim().length === 0) {
      showToast({ message: 'Add a short subject.', tone: 'error' });
      return;
    }
    if (description.trim().length === 0) {
      showToast({ message: 'Describe the issue.', tone: 'error' });
      return;
    }
    try {
      const ticket = await create.mutateAsync({
        category: selectedCategory.category,
        subject: subject.trim(),
        description: description.trim(),
        priority,
      });
      clearDraft();
      // Land on the new ticket. `replace` so Back from the detail returns to
      // the list, not this now-stale form.
      router.replace(`/support/${ticket.id}`);
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } } | null)?.response
          ?.data?.error ??
        (err instanceof Error ? err.message : "Couldn't create ticket.");
      showToast({ message: msg, tone: 'error' });
    }
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      >
        <View style={styles.commandBar}>
          <Pressable
            onPress={handleBack}
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
          <Text style={styles.commandTitle}>New ticket</Text>
        </View>

        <ScrollView
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          <Text style={styles.sectionLabel}>WHAT'S THIS ABOUT?</Text>
          <View style={styles.chipWrap}>
            {CATEGORY_CHIPS.map((chip) => {
              const selected = chip.id === categoryId;
              return (
                <Pressable
                  key={chip.id}
                  onPress={() => setCategoryId(chip.id)}
                  accessibilityRole="button"
                  accessibilityState={{ selected }}
                >
                  <View style={[styles.chip, selected && styles.chipSelected]}>
                    <Text
                      style={[
                        styles.chipText,
                        selected && styles.chipTextSelected,
                      ]}
                    >
                      {chip.label}
                    </Text>
                  </View>
                </Pressable>
              );
            })}
          </View>

          <Text style={styles.sectionLabel}>SUBJECT</Text>
          <TextInput
            value={subject}
            onChangeText={setSubject}
            placeholder="A one-line summary"
            placeholderTextColor={theme.colors.ink.muted}
            style={styles.input}
            editable={!create.isPending}
            maxLength={120}
            returnKeyType="next"
          />

          <Text style={styles.sectionLabel}>DESCRIPTION</Text>
          <TextInput
            value={description}
            onChangeText={setDescription}
            placeholder="Tell us what's happening, and what you expected instead…"
            placeholderTextColor={theme.colors.ink.muted}
            style={styles.textarea}
            multiline
            textAlignVertical="top"
            editable={!create.isPending}
            maxLength={2000}
          />

          <Text style={styles.sectionLabel}>PRIORITY</Text>
          <View style={styles.chipWrap}>
            {PRIORITY_CHIPS.map((p) => {
              const selected = p.value === priority;
              return (
                <Pressable
                  key={p.value}
                  onPress={() => setPriority(p.value)}
                  accessibilityRole="button"
                  accessibilityState={{ selected }}
                >
                  <View style={[styles.chip, selected && styles.chipSelected]}>
                    <Text
                      style={[
                        styles.chipText,
                        selected && styles.chipTextSelected,
                      ]}
                    >
                      {p.label}
                    </Text>
                  </View>
                </Pressable>
              );
            })}
          </View>

          <Pressable
            onPress={handleSubmit}
            disabled={!canSubmit}
            accessibilityRole="button"
            accessibilityLabel="Submit ticket"
          >
            <View style={[styles.submitBtn, !canSubmit && { opacity: 0.4 }]}>
              <Text style={styles.submitLabel}>
                {create.isPending ? 'Sending…' : 'Submit ticket'}
              </Text>
            </View>
          </Pressable>
        </ScrollView>
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
  },
  scroll: { flex: 1 },
  scrollContent: {
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[12],
  },

  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    paddingTop: theme.spacing[5],
    paddingBottom: theme.spacing[3],
  },

  chipWrap: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: theme.spacing[2],
  },
  chip: {
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    borderRadius: theme.radius.full,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    backgroundColor: theme.colors.paper,
    minHeight: theme.touchTarget.vendor,
    justifyContent: 'center',
  },
  chipSelected: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderColor: theme.colors.ink.DEFAULT,
  },
  chipText: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  chipTextSelected: { color: theme.colors.paper },

  input: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.strong,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: 12,
    paddingVertical: 14,
    backgroundColor: theme.colors.paper,
  },
  textarea: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    minHeight: 140,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.strong,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: 12,
    paddingTop: 12,
    paddingBottom: 12,
    backgroundColor: theme.colors.paper,
    lineHeight: 22,
  },

  submitBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    paddingVertical: 16,
    alignItems: 'center',
    marginTop: theme.spacing[8],
  },
  submitLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
    letterSpacing: 0.3,
  },
});
