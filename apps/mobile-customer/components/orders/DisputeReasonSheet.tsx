import { forwardRef, useState } from 'react';
import { Platform, Pressable, StyleSheet, Text, TextInput, View } from 'react-native';
import { SheetBase, type SheetHandle } from '@homechef/mobile-shared/ui';
import { customerColors } from '@homechef/mobile-shared/theme';

// Preset reasons for the dispute-the-refund flow (#876). Not the content-
// moderation taxonomy ReportSheet/REPORT_REASONS uses — that union has no
// refund/cancellation target, so this is a small local list instead, mirroring
// ReportSheet's structural pattern (SheetBase + required reason chips +
// optional free-text) with dispute-appropriate copy.
const DISPUTE_REASONS: { value: string; label: string }[] = [
  { value: 'amount_too_low', label: 'The refund amount is too low' },
  { value: 'disagree_with_decision', label: "I disagree with the chef's decision" },
  { value: 'refund_not_arrived', label: "The refund hasn't arrived" },
  { value: 'other', label: 'Something else' },
];

const CHIP_RIPPLE = `${customerColors.coral.DEFAULT}1F`;
const BTN_RIPPLE = `${customerColors.canvas}33`;

interface DisputeReasonSheetProps {
  /** Called with a single, always-non-empty combined reason string on submit. */
  onSubmit: (reason: string) => void;
}

/**
 * DisputeReasonSheet — collects a required preset reason plus optional
 * free-text detail before a refund dispute is raised, then hands a single
 * combined reason string back to the caller. Owns no network call and no
 * async/pending state: the mutation, its pending state, and its error/success
 * handling live in CancellationSection.tsx.
 */
export const DisputeReasonSheet = forwardRef<SheetHandle, DisputeReasonSheetProps>(
  function DisputeReasonSheet({ onSubmit }, ref) {
    const [reason, setReason] = useState<string | null>(null);
    const [details, setDetails] = useState('');

    function reset() {
      setReason(null);
      setDetails('');
    }

    function handleSubmit() {
      if (!reason) return;
      const preset = DISPUTE_REASONS.find((r) => r.value === reason);
      const label = preset?.label ?? reason;
      const trimmedDetails = details.trim();
      const combined = trimmedDetails ? `${label} — ${trimmedDetails}` : label;
      onSubmit(combined);
      reset();
      if (ref && 'current' in ref && ref.current) ref.current.dismiss();
    }

    return (
      <SheetBase ref={ref}>
        <Text style={styles.title}>Dispute the refund</Text>
        <Text style={styles.body}>Tell us why — this goes straight to our team for review.</Text>

        <View style={styles.reasons}>
          {DISPUTE_REASONS.map((r) => {
            const selected = reason === r.value;
            return (
              <Pressable
                key={r.value}
                onPress={() => setReason(r.value)}
                accessibilityRole="radio"
                accessibilityState={{ selected }}
                accessibilityLabel={r.label}
                android_ripple={{ color: CHIP_RIPPLE, borderless: false }}
              >
                {({ pressed }) => (
                  <View
                    style={[
                      styles.reason,
                      selected && styles.reasonSelected,
                      pressed && Platform.OS === 'ios' && styles.reasonPressed,
                    ]}
                  >
                    <Text style={[styles.reasonText, selected && styles.reasonTextSelected]}>
                      {r.label}
                    </Text>
                  </View>
                )}
              </Pressable>
            );
          })}
        </View>

        <TextInput
          style={styles.details}
          value={details}
          onChangeText={setDetails}
          placeholder="Add details (optional)"
          placeholderTextColor={customerColors.charcoal.soft}
          multiline
          maxLength={500}
          accessibilityLabel="Additional details"
        />

        <Pressable
          onPress={handleSubmit}
          disabled={!reason}
          accessibilityRole="button"
          accessibilityLabel="Submit dispute"
          android_ripple={reason ? { color: BTN_RIPPLE, borderless: false } : undefined}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.submit,
                !reason && styles.submitDisabled,
                pressed && Platform.OS === 'ios' && !!reason && styles.submitPressed,
              ]}
            >
              <Text style={styles.submitText}>Submit dispute</Text>
            </View>
          )}
        </Pressable>
      </SheetBase>
    );
  },
);

const styles = StyleSheet.create({
  title: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 18,
    color: customerColors.charcoal.DEFAULT,
    marginBottom: 6,
  },
  body: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    marginBottom: 16,
  },
  reasons: { gap: 8 },
  reason: {
    minHeight: 48,
    justifyContent: 'center',
    paddingHorizontal: 14,
    borderRadius: 8,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.canvas,
  },
  reasonSelected: {
    borderColor: customerColors.coral.DEFAULT,
    backgroundColor: customerColors.coral.tint,
  },
  reasonPressed: { backgroundColor: customerColors.surface.soft },
  reasonText: { fontFamily: 'Inter', fontSize: 15, color: customerColors.charcoal.DEFAULT },
  reasonTextSelected: { fontFamily: 'Inter-SemiBold', color: customerColors.coral.DEFAULT },
  details: {
    marginTop: 16,
    minHeight: 88,
    borderRadius: 8,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    padding: 12,
    fontFamily: 'Inter',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    textAlignVertical: 'top',
  },
  submit: {
    marginTop: 16,
    minHeight: 48,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.coral.DEFAULT,
  },
  submitDisabled: { opacity: 0.45 },
  submitPressed: { backgroundColor: customerColors.coral.pressed },
  submitText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: customerColors.canvas },
});
