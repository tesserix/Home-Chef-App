/**
 * PayoutScreen — chef payout details (bank transfer or UPI).
 *
 * Backend exposes GET /chef/payout and POST /chef/payout. GET returns the
 * currently selected method plus masked values for already-saved fields.
 * POST validates the chosen method's required inputs and stores sensitive
 * fields in GCP Secret Manager (so server-side reads are masked too).
 *
 * UX: a method toggle (Bank / UPI) drives which fields render. Saving
 * confirms via toast and pops back to Earnings.
 */
import { useEffect, useRef, useState } from 'react';
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
import { router } from 'expo-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import { getServerErrorMessage } from '@homechef/mobile-shared/api';
import { theme } from '@homechef/mobile-shared/theme';
import { useToast } from '@homechef/mobile-shared/ui';
import { api } from '../lib/api';

// ---- Data types -----------------------------------------------------------

type PayoutMethod = 'bank_transfer' | 'upi';

interface PayoutDetailsResponse {
  payoutMethod: string;
  bankAccountName: string;
  bankAccountNumber: string; // already masked by backend
  bankIFSC: string;
  upiId: string; // already masked
  razorpayConnected: boolean;
  stripeConnected: boolean;
  paymentProvider: string;
  payoutCountry: string;
}

interface SavePayoutPayload {
  payoutMethod: PayoutMethod;
  bankAccountNumber?: string;
  bankIFSC?: string;
  bankAccountName?: string;
  upiId?: string;
}

function usePayoutDetails() {
  return useQuery<PayoutDetailsResponse>({
    queryKey: ['chef', 'payout'],
    queryFn: () =>
      api.get<PayoutDetailsResponse>('/chef/payout').then((r) => r.data),
    staleTime: 30_000,
  });
}

function useSavePayout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: SavePayoutPayload) => api.post('/chef/payout', payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef', 'payout'] });
    },
  });
}

// ---- Sub-components -------------------------------------------------------

interface MethodTabProps {
  label: string;
  active: boolean;
  onPress: () => void;
}

// Segment of the iOS-style segmented control (spec §5): mist track,
// active segment = paper pill with shadow[1]. Visual styles live on the
// inner View (iOS Pressable inner-View pattern).
function MethodTab({ label, active, onPress }: MethodTabProps) {
  return (
    <Pressable
      onPress={onPress}
      hitSlop={6}
      accessibilityRole="radio"
      accessibilityState={{ checked: active }}
      style={tabStyles.pressable}
    >
      {({ pressed }) => (
        <View
          style={[
            tabStyles.segment,
            active && tabStyles.segmentActive,
            pressed && !active && { opacity: 0.7 },
          ]}
        >
          <Text style={[tabStyles.label, active && tabStyles.labelActive]}>
            {label}
          </Text>
        </View>
      )}
    </Pressable>
  );
}

const tabStyles = StyleSheet.create({
  pressable: { flex: 1 },
  segment: {
    minHeight: 34,
    borderRadius: 9,
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: theme.spacing[2],
  },
  segmentActive: {
    backgroundColor: theme.colors.paper,
    ...theme.shadow[1],
  },
  label: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
  },
  labelActive: { color: theme.colors.ink.DEFAULT },
});

interface FieldProps {
  label: string;
  value: string;
  onChangeText: (t: string) => void;
  placeholder?: string;
  autoCapitalize?: 'none' | 'words' | 'characters';
  keyboardType?: 'default' | 'number-pad' | 'email-address';
  caption?: string;
  hasBorderBottom?: boolean;
}

function Field({
  label,
  value,
  onChangeText,
  placeholder,
  autoCapitalize = 'none',
  keyboardType = 'default',
  caption,
  hasBorderBottom = true,
}: FieldProps) {
  return (
    <>
      <View style={styles.editRow}>
        <Text style={styles.editLabel}>{label}</Text>
        <TextInput
          value={value}
          onChangeText={onChangeText}
          placeholder={placeholder}
          placeholderTextColor={theme.colors.ink.muted}
          autoCapitalize={autoCapitalize}
          autoCorrect={false}
          keyboardType={keyboardType}
          style={styles.editInput}
        />
        {caption ? <Text style={styles.editCaption}>{caption}</Text> : null}
      </View>
      {/* Inset hairline — skipped on the last row of a group card */}
      {hasBorderBottom ? <View style={styles.separator} /> : null}
    </>
  );
}

// ---- Screen ---------------------------------------------------------------

export default function PayoutScreen() {
  const { data, isLoading, isError, refetch } = usePayoutDetails();
  const saveMutation = useSavePayout();
  const { show: showToast } = useToast();

  const [method, setMethod] = useState<PayoutMethod>('bank_transfer');
  const [bankAccountName, setBankAccountName] = useState('');
  const [bankAccountNumber, setBankAccountNumber] = useState('');
  const [bankIFSC, setBankIFSC] = useState('');
  const [upiId, setUpiId] = useState('');

  // Track whether a successful save has already cleared the dirty state so
  // the back handler doesn't re-prompt after the toast confirms success.
  const savedRef = useRef(false);

  useEffect(() => {
    if (!data) return;
    if (data.payoutMethod === 'upi') setMethod('upi');
    else setMethod('bank_transfer');
    // Pre-fill the non-sensitive fields so the chef sees what's saved
    // and doesn't have to retype to make a small change. The masked
    // account number / UPI ID intentionally stay blank — those are
    // re-entered when the chef wants to update them. The current
    // method banner above the form shows what's currently on file.
    if (data.bankAccountName) setBankAccountName(data.bankAccountName);
    if (data.bankIFSC) setBankIFSC(data.bankIFSC);
    // A data refresh after save means the server accepted our payload —
    // reset the saved flag so we compare against the fresh server values.
    savedRef.current = false;
  }, [data]);

  // Dirty against the un-pre-filled sensitive fields (the chef typed them
  // in this session) plus a possible method change. If anything is in
  // flight, back prompts to save or discard.
  const hasUnsavedSensitive =
    (method === 'bank_transfer' && bankAccountNumber.trim() !== '') ||
    (method === 'upi' && upiId.trim() !== '');
  const methodChanged = data && data.payoutMethod !== method;
  const isDirty = !savedRef.current && (hasUnsavedSensitive || Boolean(methodChanged));

  function popBack(): void {
    if (router.canGoBack()) router.back();
    else router.replace('/(tabs)');
  }

  function handleBack(): void {
    if (!isDirty) {
      popBack();
      return;
    }
    Alert.alert(
      'Save changes?',
      'You have unsaved payout edits. Save them before going back?',
      [
        { text: 'Keep editing', style: 'cancel' },
        { text: 'Discard', style: 'destructive', onPress: popBack },
        { text: 'Save', onPress: handleSave },
      ],
    );
  }

  function handleSave(): void {
    if (method === 'bank_transfer') {
      if (!bankAccountName.trim() || !bankAccountNumber.trim() || !bankIFSC.trim()) {
        Alert.alert(
          'Bank details required',
          'Enter account name, number, and IFSC to save bank transfer payout.',
        );
        return;
      }
    } else {
      if (!upiId.trim()) {
        Alert.alert('UPI ID required', 'Enter your UPI ID (e.g. name@bank).');
        return;
      }
    }

    const payload: SavePayoutPayload = { payoutMethod: method };
    if (method === 'bank_transfer') {
      payload.bankAccountName = bankAccountName.trim();
      payload.bankAccountNumber = bankAccountNumber.trim();
      payload.bankIFSC = bankIFSC.trim().toUpperCase();
    } else {
      payload.upiId = upiId.trim();
    }

    saveMutation.mutate(payload, {
      onSuccess: () => {
        // Mark as saved BEFORE showing feedback so that if the chef
        // immediately taps Back the dirty guard evaluates to false and
        // does NOT re-prompt. The query invalidation (in useSavePayout's
        // onSuccess) will reset savedRef via the data useEffect above.
        savedRef.current = true;
        showToast({ message: 'Payout details saved', tone: 'success' });
        popBack();
      },
      onError: (err) =>
        Alert.alert('Save failed', getServerErrorMessage(err, 'Please try again.')),
    });
  }

  // ---- Loading / error -----------------------------------------------------

  if (isLoading) {
    return (
      <SafeAreaView style={styles.centered} edges={['top', 'left', 'right']}>
        <CommandBar onBack={handleBack} />
        <View style={styles.centeredFill}>
          <ActivityIndicator size="large" color={theme.colors.ink.DEFAULT} />
        </View>
      </SafeAreaView>
    );
  }

  if (isError) {
    return (
      <SafeAreaView style={styles.centered} edges={['top', 'left', 'right']}>
        <CommandBar onBack={handleBack} />
        <View style={styles.centeredFill}>
          <Text style={styles.errorBody}>Failed to load payout details</Text>
          <Pressable
            onPress={() => refetch()}
            style={({ pressed }) => [
              styles.errorBtn,
              pressed && { opacity: 0.85 },
            ]}
          >
            <Text style={styles.errorBtnLabel}>Retry</Text>
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  // ---- Render --------------------------------------------------------------

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        keyboardVerticalOffset={0}
      >
        <CommandBar onBack={handleBack} />

        <ScrollView
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
          showsVerticalScrollIndicator={false}
          keyboardShouldPersistTaps="handled"
        >
          {/* Currently-saved method summary */}
          {data?.payoutMethod ? (
            <View style={styles.currentBanner}>
              <Text style={styles.currentBannerLabel}>
                Currently using{' '}
                {data.payoutMethod === 'upi' ? 'UPI' : 'Bank transfer'}
              </Text>
              {data.payoutMethod === 'bank_transfer' && data.bankAccountNumber ? (
                <Text style={styles.currentBannerSub}>
                  {data.bankAccountName} · {data.bankAccountNumber}
                </Text>
              ) : null}
              {data.payoutMethod === 'upi' && data.upiId ? (
                <Text style={styles.currentBannerSub}>{data.upiId}</Text>
              ) : null}
            </View>
          ) : null}

          {/* Method segmented control (spec §5) */}
          <Text style={styles.sectionLabel}>PAYOUT METHOD</Text>
          <View style={styles.segmentTrack}>
            <MethodTab
              label="Bank transfer"
              active={method === 'bank_transfer'}
              onPress={() => setMethod('bank_transfer')}
            />
            <MethodTab
              label="UPI"
              active={method === 'upi'}
              onPress={() => setMethod('upi')}
            />
          </View>

          {/* Method-specific fields */}
          {method === 'bank_transfer' ? (
            <>
              <Text style={styles.sectionLabel}>BANK ACCOUNT</Text>
              <View style={styles.hairlineGroup}>
                <Field
                  label="Account holder name"
                  value={bankAccountName}
                  onChangeText={setBankAccountName}
                  placeholder="Name as on bank records"
                  autoCapitalize="words"
                />
                <Field
                  label="Account number"
                  value={bankAccountNumber}
                  onChangeText={setBankAccountNumber}
                  placeholder="11–18 digit account number"
                  keyboardType="number-pad"
                />
                {data && (
                  <Text style={styles.reentryHelper}>(re-enter required)</Text>
                )}
                <Field
                  label="IFSC code"
                  value={bankIFSC}
                  onChangeText={setBankIFSC}
                  placeholder="HDFC0001234"
                  autoCapitalize="characters"
                  caption="11 characters · uppercase letters and numbers"
                  hasBorderBottom={false}
                />
              </View>
            </>
          ) : (
            <>
              <Text style={styles.sectionLabel}>UPI</Text>
              <View style={styles.hairlineGroup}>
                <Field
                  label="UPI ID"
                  value={upiId}
                  onChangeText={setUpiId}
                  placeholder="yourname@bank"
                  keyboardType="email-address"
                  caption="Example: 9876543210@upi or name@hdfcbank"
                  hasBorderBottom={false}
                />
                {data && (
                  <Text style={styles.reentryHelper}>(re-enter required)</Text>
                )}
              </View>
            </>
          )}

          <View style={styles.helperBlock}>
            <Text style={styles.helperHeadline}>How payouts work</Text>
            <Text style={styles.helperBody}>
              Earnings settle into this account weekly. Sensitive details are
              encrypted at rest and never shown back in full once saved.
            </Text>
          </View>
        </ScrollView>

        {/* Sticky Save footer */}
        <View style={styles.stickyFooter}>
          <SafeAreaView edges={['bottom']} style={styles.stickyFooterInner}>
            <Pressable
              onPress={handleSave}
              disabled={saveMutation.isPending}
              style={({ pressed }) => [
                styles.saveBtn,
                (pressed || saveMutation.isPending) && { opacity: 0.85 },
              ]}
              accessibilityRole="button"
            >
              {saveMutation.isPending ? (
                <ActivityIndicator color={theme.colors.paper} />
              ) : (
                <Text style={styles.saveBtnLabel}>
                  {isDirty ? 'Save changes' : 'Save payout details'}
                </Text>
              )}
            </Pressable>
          </SafeAreaView>
        </View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

// ---- Command bar ---------------------------------------------------------

interface CommandBarProps {
  onBack: () => void;
}

function CommandBar({ onBack }: CommandBarProps) {
  return (
    <View style={styles.commandBar}>
      <Pressable
        onPress={onBack}
        hitSlop={8}
        accessibilityRole="button"
        accessibilityLabel="Go back"
      >
        {({ pressed }) => (
          <View style={[styles.backBtn, pressed && { opacity: 0.6 }]}>
            <ChevronLeft size={22} color={theme.colors.ink.DEFAULT} strokeWidth={2} />
          </View>
        )}
      </Pressable>
      <Text style={styles.commandTitle}>Payout</Text>
      <View style={styles.commandSpacer} />
    </View>
  );
}

// ---- Styles --------------------------------------------------------------

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  centered: { flex: 1, backgroundColor: theme.colors.bone },
  centeredFill: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
  },

  // Command bar
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
    gap: theme.spacing[2],
  },
  backBtn: {
    width: 32,
    height: 32,
    justifyContent: 'center',
    alignItems: 'flex-start',
  },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
    flex: 1,
  },
  commandSpacer: { width: 32 },

  // Scroll
  scroll: { flex: 1 },
  scrollContent: { paddingBottom: 120 },

  // Current method banner — white card on the bone canvas (spec §1)
  currentBanner: {
    marginHorizontal: theme.spacing[4],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    borderRadius: theme.radius.lg,
    backgroundColor: theme.colors.paper,
    marginBottom: theme.spacing[4],
    ...theme.shadow[1],
  },
  currentBannerLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  currentBannerSub: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    marginTop: 2,
  },

  // Section label
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    marginTop: theme.spacing[2],
    marginBottom: theme.spacing[2],
  },

  // Section group card — white surface on the bone canvas (spec §1).
  // Name kept from the v1 hairline layout to avoid call-site churn.
  hairlineGroup: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[4],
    paddingBottom: theme.spacing[1],
    ...theme.shadow[1],
  },
  // Inset hairline separator — aligned to row text, not edge-to-edge
  separator: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginLeft: theme.spacing[4],
  },

  // Segmented control track (spec §5)
  segmentTrack: {
    flexDirection: 'row',
    backgroundColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.md,
    padding: 3,
    minHeight: 40,
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[4],
  },

  // Editable field
  editRow: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
  },
  editLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.3,
    marginBottom: theme.spacing[1],
  },
  // Bone input box inside the white group card (spec §1 surface flip)
  editInput: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    minHeight: 44,
    paddingVertical: theme.spacing[2],
    paddingHorizontal: theme.spacing[3],
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.md,
  },
  editCaption: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: theme.spacing[1],
  },

  // Re-entry helper — displayed under masked account fields when a saved
  // method already exists, to explain why the field appears empty.
  reentryHelper: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    paddingHorizontal: theme.spacing[4],
    paddingTop: 4,
    paddingBottom: theme.spacing[2],
  },

  // Helper block
  helperBlock: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[8],
  },
  helperHeadline: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    marginBottom: theme.spacing[1],
  },
  helperBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    lineHeight: 20,
  },

  // Sticky footer — paper surface lifted off the canvas with a top
  // shadow instead of a hairline (spec §6-style top elevation)
  stickyFooter: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    backgroundColor: theme.colors.paper,
    shadowColor: theme.colors.ink.DEFAULT,
    shadowOffset: { width: 0, height: -2 },
    shadowOpacity: 0.06,
    shadowRadius: 12,
    elevation: 8,
  },
  stickyFooterInner: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
  },
  saveBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingVertical: theme.spacing[4],
    alignItems: 'center',
    minHeight: 52,
    justifyContent: 'center',
  },
  saveBtnLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
    letterSpacing: 0.2,
  },

  // Error
  errorBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[4],
    textAlign: 'center',
  },
  errorBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[6],
    paddingVertical: theme.spacing[3],
  },
  errorBtnLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
  },
});
