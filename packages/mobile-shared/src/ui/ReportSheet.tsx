import { forwardRef, useState } from 'react';
import { ActivityIndicator, Pressable, StyleSheet, Text, TextInput, View } from 'react-native';
import { theme } from '../theme/tokens';
import { SheetBase, type SheetHandle } from './SheetBase';

export type { SheetHandle };

/**
 * Reasons a user can pick. Mirrors models.ReportReason on the API — adding one
 * here without adding it there produces a 400, so the two lists move together.
 *
 * Kept short on purpose: a long list produces worse signal, because reporters
 * pick the first plausible option rather than the accurate one.
 */
export const REPORT_REASONS = [
  { value: 'harassment', label: 'Harassment or bullying' },
  { value: 'hate_speech', label: 'Hate speech' },
  { value: 'sexual_content', label: 'Nudity or sexual content' },
  { value: 'violence', label: 'Violence or threats' },
  { value: 'food_safety', label: 'Food safety concern' },
  { value: 'spam', label: 'Spam or scam' },
  { value: 'misinformation', label: 'False information' },
  { value: 'illegal', label: 'Illegal activity' },
  { value: 'other', label: 'Something else' },
] as const;

export type ReportReason = (typeof REPORT_REASONS)[number]['value'];

/** Content kinds the API accepts. Mirrors models.ReportableType. */
export type ReportTargetType =
  | 'review'
  | 'social_post'
  | 'post_comment'
  | 'message'
  | 'user'
  | 'chef'
  | 'menu_item';

interface ReportSheetProps {
  /** What is being reported. Shown to the user, e.g. "this review". */
  subject: string;
  /**
   * Submits the report. Should resolve on success and throw on failure — the
   * sheet renders its own error state rather than closing over a broken submit.
   */
  onSubmit: (reason: ReportReason, details: string) => Promise<void>;
  /**
   * Optional block action, offered alongside reporting.
   *
   * App Review guideline 1.2 asks for reporting AND blocking. Offering the
   * block here, at the moment the user is already upset enough to report, is
   * the only place they reliably find it — a settings-screen-only block is
   * technically compliant and practically useless.
   */
  onBlock?: () => Promise<void>;
  /** Name shown on the block button, e.g. "Block Anita's kitchen". */
  blockLabel?: string;
}

/**
 * <ReportSheet> — the report-content flow, shared by every UGC surface.
 *
 * One implementation because there are four surfaces (reviews, posts, comments,
 * order messages) and a reporting flow that differs per surface is a reporting
 * flow that rots on three of them.
 */
export const ReportSheet = forwardRef<SheetHandle, ReportSheetProps>(function ReportSheet(
  { subject, onSubmit, onBlock, blockLabel },
  ref,
) {
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);

  async function handleSubmit(): Promise<void> {
    if (!reason || busy) return;
    setBusy(true);
    setError(null);
    try {
      await onSubmit(reason, details.trim());
      setDone(true);
    } catch {
      setError('Could not send your report. Please check your connection and try again.');
    } finally {
      setBusy(false);
    }
  }

  async function handleBlock(): Promise<void> {
    if (!onBlock || busy) return;
    setBusy(true);
    setError(null);
    try {
      await onBlock();
      setDone(true);
    } catch {
      setError('Could not block this account. Please try again.');
    } finally {
      setBusy(false);
    }
  }

  // Acknowledgement state. Apple's reviewers look for confirmation that a
  // report went somewhere — a sheet that just closes reads as a no-op.
  if (done) {
    return (
      <SheetBase ref={ref}>
        <Text style={styles.title}>Thanks for telling us</Text>
        <Text style={styles.body}>
          Our team reviews every report within 24 hours. If it breaks our rules we take it down.
        </Text>
      </SheetBase>
    );
  }

  return (
    <SheetBase ref={ref}>
      <Text style={styles.title}>{`Report ${subject}`}</Text>
      <Text style={styles.body}>Why are you reporting this?</Text>

      <View style={styles.reasons}>
        {REPORT_REASONS.map((r) => {
          const selected = reason === r.value;
          return (
            <Pressable
              key={r.value}
              onPress={() => setReason(r.value)}
              accessibilityRole="radio"
              accessibilityState={{ selected }}
              accessibilityLabel={r.label}
              style={[styles.reason, selected && styles.reasonSelected]}
            >
              <Text style={[styles.reasonText, selected && styles.reasonTextSelected]}>
                {r.label}
              </Text>
            </Pressable>
          );
        })}
      </View>

      <TextInput
        style={styles.details}
        value={details}
        onChangeText={setDetails}
        placeholder="Anything else we should know? (optional)"
        placeholderTextColor={theme.colors.ink.muted}
        multiline
        maxLength={2000}
        accessibilityLabel="Additional details"
      />

      {error && <Text style={styles.error}>{error}</Text>}

      <Pressable
        onPress={handleSubmit}
        disabled={!reason || busy}
        accessibilityRole="button"
        accessibilityLabel="Submit report"
        style={[styles.submit, (!reason || busy) && styles.submitDisabled]}
      >
        {busy ? (
          <ActivityIndicator color="#fff" />
        ) : (
          <Text style={styles.submitText}>Submit report</Text>
        )}
      </Pressable>

      {onBlock && (
        <Pressable
          onPress={handleBlock}
          disabled={busy}
          accessibilityRole="button"
          accessibilityLabel={blockLabel ?? 'Block this account'}
          style={styles.block}
        >
          <Text style={styles.blockText}>{blockLabel ?? 'Block this account'}</Text>
        </Pressable>
      )}
    </SheetBase>
  );
});

const styles = StyleSheet.create({
  title: {
    fontSize: 20,
    fontWeight: '600',
    color: theme.colors.ink.DEFAULT,
    marginBottom: 6,
  },
  body: {
    fontSize: 15,
    color: theme.colors.ink.muted,
    marginBottom: 12,
  },
  reasons: { gap: 8 },
  reason: {
    minHeight: 48, // 44px floor plus padding — customer-app touch target
    justifyContent: 'center',
    paddingHorizontal: 14,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
  },
  reasonSelected: {
    borderColor: theme.colors.herb.DEFAULT,
    backgroundColor: theme.colors.herb.tint,
  },
  reasonText: { fontSize: 15, color: theme.colors.ink.DEFAULT },
  reasonTextSelected: { color: theme.colors.herb.DEFAULT, fontWeight: '600' },
  details: {
    marginTop: 16,
    minHeight: 88,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    padding: 12,
    fontSize: 15,
    color: theme.colors.ink.DEFAULT,
    textAlignVertical: 'top',
  },
  error: { marginTop: 12, fontSize: 14, color: theme.colors.destructive.DEFAULT },
  submit: {
    marginTop: 16,
    minHeight: 48,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: theme.colors.herb.DEFAULT,
  },
  submitDisabled: { opacity: 0.45 },
  submitText: { fontSize: 16, fontWeight: '600', color: '#fff' },
  block: {
    marginTop: 10,
    minHeight: 48,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1,
    borderColor: theme.colors.destructive.DEFAULT,
  },
  blockText: { fontSize: 16, fontWeight: '600', color: theme.colors.destructive.DEFAULT },
});
