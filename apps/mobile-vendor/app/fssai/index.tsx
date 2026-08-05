// FSSAI filing request — the chef asks us to obtain their registration.
//
// One screen, three states, driven entirely by the server's view of the
// request: no request yet (the offer + form), a draft still being assembled
// (documents, then pay), and a paid request we are working on (the tracker).
// The app never decides what comes next — `canPay` is the server's answer, so a
// payment can never be offered against documents it has not received.

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import * as ImagePicker from 'expo-image-picker';
import * as WebBrowser from 'expo-web-browser';
import { theme } from '@homechef/mobile-shared/theme';
import { useToast } from '@homechef/mobile-shared/ui';
import { formatMoney } from '../../lib/format';
import {
  buildCashfreeCheckoutUrl,
  fssaiStepIndex,
  FSSAI_DOCUMENT_HINTS,
  FSSAI_DOCUMENT_KINDS,
  FSSAI_DOCUMENT_LABELS,
  FSSAI_TRACKER_STEPS,
  fssaiStatusLabel,
  isFssaiClosed,
  isFssaiPaid,
  type FssaiDocument,
  type FssaiDocumentKind,
  type FssaiQuote,
  type FssaiRequest,
} from '../../lib/fssai';
import {
  useCancelFssaiRequest,
  useConfirmFssaiPayment,
  useCreateFssaiRequest,
  useFssaiQuote,
  useFssaiRequest,
  useRemoveFssaiDocument,
  useStartFssaiPayment,
  useUploadFssaiDocument,
} from '../../hooks/useFssai';

const RETURN_URL = 'homechef-vendor://fssai';

interface FssaiForm {
  kitchenName: string;
  applicantName: string;
  contactPhone: string;
  contactEmail: string;
  addressLine1: string;
  addressLine2: string;
  city: string;
  state: string;
  postalCode: string;
}

const EMPTY_FORM: FssaiForm = {
  kitchenName: '',
  applicantName: '',
  contactPhone: '',
  contactEmail: '',
  addressLine1: '',
  addressLine2: '',
  city: '',
  state: '',
  postalCode: '',
};

const FORM_FIELDS = [
  ['kitchenName', 'Kitchen name'],
  ['applicantName', 'Your full name'],
  ['contactPhone', 'Phone'],
  ['contactEmail', 'Email'],
  ['addressLine1', 'Address line 1'],
  ['addressLine2', 'Address line 2 (optional)'],
  ['city', 'City'],
  ['state', 'State'],
  ['postalCode', 'PIN code'],
] as const;

export default function FssaiScreen() {
  const { show: showToast } = useToast();
  const quoteQuery = useFssaiQuote();
  const requestQuery = useFssaiRequest();
  const createRequest = useCreateFssaiRequest();
  const uploadDocument = useUploadFssaiDocument();
  const removeDocument = useRemoveFssaiDocument();
  const startPayment = useStartFssaiPayment();
  const confirmPayment = useConfirmFssaiPayment();
  const cancelRequest = useCancelFssaiRequest();

  const [uploading, setUploading] = useState<FssaiDocumentKind | null>(null);
  const [termYears, setTermYears] = useState(1);
  const [form, setForm] = useState<FssaiForm>(EMPTY_FORM);
  // Seed from onboarding once, and only while the chef has not typed: this must
  // never overwrite an edit they have already made, which is what a plain
  // `setForm(prefill)` on every fetch would do.
  const seeded = useRef(false);

  const request = requestQuery.data?.request ?? null;
  const enabled = quoteQuery.data?.enabled ?? requestQuery.data?.enabled ?? false;
  const terms = quoteQuery.data?.terms ?? [];
  // A live draft is priced by the row, not by the picker — the figure was
  // frozen when it was created and a later policy change must not restate it.
  const draft = request && !isFssaiPaid(request.status) ? request : null;
  const quote = useMemo(
    () => terms.find((t) => t.termYears === termYears) ?? quoteQuery.data?.quote,
    [terms, termYears, quoteQuery.data],
  );

  const prefill = requestQuery.data?.prefill;
  useEffect(() => {
    if (seeded.current || !prefill || draft) return;
    seeded.current = true;
    setForm((prev) => ({
      kitchenName: prev.kitchenName || prefill.kitchenName,
      applicantName: prev.applicantName || prefill.applicantName,
      contactPhone: prev.contactPhone || prefill.contactPhone,
      contactEmail: prev.contactEmail || prefill.contactEmail,
      addressLine1: prev.addressLine1 || prefill.addressLine1,
      addressLine2: prev.addressLine2 || prefill.addressLine2,
      city: prev.city || prefill.city,
      state: prev.state || prefill.state,
      postalCode: prev.postalCode || prefill.postalCode,
    }));
  }, [prefill, draft]);

  const reportError = useCallback(
    (err: unknown, fallback: string) => {
      showToast({ message: err instanceof Error ? err.message : fallback, tone: 'error' });
    },
    [showToast],
  );

  async function handleCreateDraft() {
    const missing = FORM_FIELDS.find(
      ([key]) => key !== 'addressLine2' && form[key].trim() === '',
    );
    if (missing) {
      showToast({ message: `${missing[1]} is needed to complete your form.`, tone: 'error' });
      return;
    }
    try {
      await createRequest.mutateAsync({ ...form, termYears });
      showToast({ message: 'Now add your documents.', tone: 'success' });
    } catch (err) {
      reportError(err, 'Could not start the request');
    }
  }

  async function handleUpload(kind: FssaiDocumentKind) {
    if (!draft) return;
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      showToast({ message: 'Allow photo access to attach your documents.', tone: 'error' });
      return;
    }
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      quality: 0.85,
    });
    const asset = result.assets?.[0];
    if (result.canceled || !asset) return;

    setUploading(kind);
    try {
      await uploadDocument.mutateAsync({
        requestId: draft.id,
        kind,
        uri: asset.uri,
        mimeType: asset.mimeType,
      });
      showToast({ message: `${FSSAI_DOCUMENT_LABELS[kind]} added.`, tone: 'success' });
    } catch (err) {
      reportError(err, 'Upload failed. Try again.');
    } finally {
      setUploading(null);
    }
  }

  async function handleRemove(kind: FssaiDocumentKind) {
    if (!draft) return;
    try {
      await removeDocument.mutateAsync({ requestId: draft.id, kind });
    } catch (err) {
      reportError(err, 'Could not remove the document');
    }
  }

  // Opens the hosted Cashfree page and, whatever the browser reports back, asks
  // the SERVER whether the money arrived. The browser's answer is a UX hint
  // only — Cashfree hands the client nothing it could prove a payment with.
  const handlePay = useCallback(async () => {
    if (!draft) return;
    try {
      const session = await startPayment.mutateAsync(draft.id);
      try {
        await WebBrowser.openAuthSessionAsync(
          buildCashfreeCheckoutUrl({
            paymentSessionId: session.cashfreePaymentSessionId,
            env: session.cashfreeEnv,
            returnUrl: RETURN_URL,
          }),
          RETURN_URL,
        );
      } catch {
        // A browser error is not proof of failure — fall through and let the
        // server adjudicate rather than telling a chef who paid that they did not.
      }
      const updated = await confirmPayment.mutateAsync(draft.id);
      const paid = isFssaiPaid(updated.status);
      showToast({
        message: paid
          ? 'Payment received. Your request is with our team.'
          : "We haven't seen the payment yet. Pull to refresh in a moment.",
        tone: paid ? 'success' : 'error',
      });
    } catch (err) {
      reportError(err, "We couldn't complete the payment. Nothing has been charged.");
    }
  }, [draft, startPayment, confirmPayment, showToast, reportError]);

  function handleCancel() {
    if (!draft) return;
    Alert.alert(
      'Discard this request?',
      'Nothing has been charged. Your details and documents will be removed.',
      [
        { text: 'Keep it', style: 'cancel' },
        {
          text: 'Discard',
          style: 'destructive',
          onPress: async () => {
            try {
              await cancelRequest.mutateAsync(draft.id);
              setForm(EMPTY_FORM);
              showToast({ message: 'Request discarded.', tone: 'success' });
            } catch (err) {
              reportError(err, 'Could not discard the request');
            }
          },
        },
      ],
    );
  }

  if (quoteQuery.isLoading || requestQuery.isLoading) {
    return (
      <Shell>
        <View style={styles.centered}>
          <ActivityIndicator color={theme.colors.ink.DEFAULT} />
        </View>
      </Shell>
    );
  }

  if (!enabled && !request) {
    return (
      <Shell>
        <View style={styles.centered}>
          <Text style={styles.body}>
            We aren't taking FSSAI applications at the moment. You can still apply
            yourself on the government portal — see fe3dr.com/fssai for the steps.
          </Text>
        </View>
      </Shell>
    );
  }

  return (
    <Shell>
      <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
        {request && isFssaiPaid(request.status) ? (
          <Tracker request={request} />
        ) : draft ? (
          <Draft
            request={draft}
            uploading={uploading}
            onUpload={handleUpload}
            onRemove={handleRemove}
            onPay={handlePay}
            onCancel={handleCancel}
            paying={startPayment.isPending || confirmPayment.isPending}
            notice={quoteQuery.data?.nonRefundableNotice}
          />
        ) : (
          <NewRequest
            quote={quote}
            terms={terms}
            termYears={termYears}
            setTermYears={setTermYears}
            form={form}
            setForm={setForm}
            onContinue={handleCreateDraft}
            submitting={createRequest.isPending}
            lastRequest={request && isFssaiClosed(request.status) ? request : null}
          />
        )}
      </ScrollView>
    </Shell>
  );
}

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={12}
        >
          <Text style={styles.back}>‹</Text>
        </Pressable>
        <Text style={styles.title}>FSSAI registration</Text>
      </View>
      {children}
    </SafeAreaView>
  );
}

interface NewRequestProps {
  quote?: FssaiQuote;
  terms: FssaiQuote[];
  termYears: number;
  setTermYears: (n: number) => void;
  form: FssaiForm;
  setForm: React.Dispatch<React.SetStateAction<FssaiForm>>;
  onContinue: () => void;
  submitting: boolean;
  lastRequest: FssaiRequest | null;
}

function NewRequest({
  quote,
  terms,
  termYears,
  setTermYears,
  form,
  setForm,
  onContinue,
  submitting,
  lastRequest,
}: NewRequestProps) {
  return (
    <>
      {lastRequest?.registrationNo ? (
        <View style={styles.card}>
          <Text style={styles.cardTitle}>Your last registration</Text>
          <Text style={styles.body}>
            {lastRequest.registrationNo} — obtained{' '}
            {new Date(lastRequest.issuedAt ?? '').toLocaleDateString('en-IN')}
          </Text>
        </View>
      ) : null}

      <Text style={styles.lead}>
        Every home kitchen needs an FSSAI registration by law. You can apply
        yourself on the government portal for free, or we can complete the whole
        application for you.
      </Text>

      <Text style={styles.sectionLabel}>HOW LONG FOR?</Text>
      <View style={styles.termRow}>
        {terms.map((t) => (
          <Pressable
            key={t.termYears}
            onPress={() => setTermYears(t.termYears)}
            accessibilityRole="button"
            accessibilityState={{ selected: termYears === t.termYears }}
            accessibilityLabel={`${t.termYears} year${t.termYears === 1 ? '' : 's'}, ${formatMoney(t.total)}`}
            style={[styles.termChip, termYears === t.termYears && styles.termChipOn]}
          >
            <Text
              style={[styles.termChipText, termYears === t.termYears && styles.termChipTextOn]}
            >
              {t.termYears}y
            </Text>
          </Pressable>
        ))}
      </View>

      {quote ? <QuoteCard quote={quote} /> : null}

      <Text style={styles.sectionLabel}>YOUR DETAILS</Text>
      {FORM_FIELDS.map(([key, label]) => (
        <TextInput
          key={key}
          style={styles.input}
          placeholder={label}
          placeholderTextColor={theme.colors.ink.soft}
          value={form[key]}
          onChangeText={(v) => setForm((prev) => ({ ...prev, [key]: v }))}
          accessibilityLabel={label}
          autoCapitalize={key === 'contactEmail' ? 'none' : 'words'}
          keyboardType={
            key === 'contactPhone' || key === 'postalCode'
              ? 'number-pad'
              : key === 'contactEmail'
                ? 'email-address'
                : 'default'
          }
        />
      ))}

      <Text style={styles.fine}>
        Next you'll add two documents. Nothing is charged until they are in and
        you confirm the amount.
      </Text>

      <PrimaryButton
        label={submitting ? 'Saving…' : 'Continue to documents'}
        onPress={onContinue}
        disabled={submitting}
      />
    </>
  );
}

function QuoteCard({ quote }: { quote: FssaiQuote }) {
  return (
    <View style={styles.card}>
      <Text style={styles.cardTitle}>What you pay</Text>
      {/* Every line is the server's. The app renders them and never re-adds the
          total — the figure a chef agrees to must be the figure charged. */}
      <Row label="FSSAI registration fee" value={quote.governmentFee} />
      <Row label={`GST on it (${quote.gstPercent}%)`} value={quote.governmentTax} />
      <Row label="Our filing fee" value={quote.serviceFee} />
      <Row label={`GST on our fee (${quote.gstPercent}%)`} value={quote.serviceTax} />
      <Row label="Total" value={quote.total} emphasis />
      <Text style={styles.fine}>
        The registration is issued in your name and stays yours. We pay FSSAI on
        your behalf out of this amount.
      </Text>
    </View>
  );
}

interface DraftProps {
  request: FssaiRequest;
  uploading: FssaiDocumentKind | null;
  onUpload: (kind: FssaiDocumentKind) => void;
  onRemove: (kind: FssaiDocumentKind) => void;
  onPay: () => void;
  onCancel: () => void;
  paying: boolean;
  notice?: string;
}

function Draft({
  request,
  uploading,
  onUpload,
  onRemove,
  onPay,
  onCancel,
  paying,
  notice,
}: DraftProps) {
  const byKind = new Map<FssaiDocumentKind, FssaiDocument>(
    request.documents.map((d) => [d.kind, d]),
  );

  return (
    <>
      <View style={styles.card}>
        <Text style={styles.cardTitle}>{request.kitchenName}</Text>
        <Text style={styles.body}>
          {request.termYears} year{request.termYears === 1 ? '' : 's'} ·{' '}
          {formatMoney(request.feeTotal)} · not sent yet
        </Text>
      </View>

      <Text style={styles.sectionLabel}>YOUR DOCUMENTS</Text>
      <Text style={styles.body}>
        FSSAI needs a photo of you and a government photo ID. Add an address proof
        only if your kitchen is somewhere other than the address on that ID.
      </Text>
      {FSSAI_DOCUMENT_KINDS.map((kind) => {
        const doc = byKind.get(kind);
        const optional = kind === 'address_proof';
        return (
          <View key={kind} style={styles.docRow}>
            <Pressable
              onPress={() => onUpload(kind)}
              disabled={uploading !== null}
              accessibilityRole="button"
              accessibilityLabel={`${doc ? 'Replace' : 'Add'} ${FSSAI_DOCUMENT_LABELS[kind]}`}
              style={styles.docMain}
            >
              <Text style={styles.docLabel}>
                {FSSAI_DOCUMENT_LABELS[kind]}
                {optional ? ' (only if needed)' : ''}
              </Text>
              <Text style={styles.fine}>{doc?.fileName || FSSAI_DOCUMENT_HINTS[kind]}</Text>
            </Pressable>
            <View style={styles.docActions}>
              <Text style={styles.docState}>
                {uploading === kind ? 'Uploading…' : doc ? 'Added ✓' : 'Add'}
              </Text>
              {doc && optional ? (
                <Pressable
                  onPress={() => onRemove(kind)}
                  accessibilityRole="button"
                  accessibilityLabel={`Remove ${FSSAI_DOCUMENT_LABELS[kind]}`}
                  hitSlop={8}
                >
                  <Text style={styles.docRemove}>Remove</Text>
                </Pressable>
              ) : null}
            </View>
          </View>
        );
      })}

      {notice ? <Text style={styles.notice}>{notice}</Text> : null}

      <PrimaryButton
        label={
          paying
            ? 'Opening payment…'
            : request.canPay
              ? `Pay ${formatMoney(request.feeTotal)} and apply`
              : 'Add your photo and photo ID to continue'
        }
        onPress={onPay}
        disabled={!request.canPay || paying}
      />

      <Pressable
        onPress={onCancel}
        accessibilityRole="button"
        accessibilityLabel="Discard this request"
        style={styles.secondary}
      >
        <Text style={styles.secondaryText}>Discard this request</Text>
      </Pressable>
    </>
  );
}

function Tracker({ request }: { request: FssaiRequest }) {
  const step = fssaiStepIndex(request.status);
  return (
    <>
      <View style={styles.card}>
        <Text style={styles.cardTitle}>{fssaiStatusLabel(request.status)}</Text>
        <Text style={styles.body}>
          {request.kitchenName} · {request.termYears} year
          {request.termYears === 1 ? '' : 's'} · {formatMoney(request.feeTotal)} paid
        </Text>
      </View>

      {step >= 0 ? (
        <>
          <Text style={styles.sectionLabel}>PROGRESS</Text>
          <View style={styles.card}>
            {FSSAI_TRACKER_STEPS.map((s, i) => (
              <View key={s.key} style={styles.stepRow}>
                <Text style={[styles.stepDot, i <= step && styles.stepDotOn]}>
                  {i <= step ? '●' : '○'}
                </Text>
                <Text style={[styles.stepLabel, i <= step && styles.stepLabelOn]}>
                  {s.label}
                </Text>
              </View>
            ))}
            {request.applicationRef ? (
              <Text style={styles.fine}>
                FoSCoS reference {request.applicationRef} — you can track this on
                the government portal yourself.
              </Text>
            ) : null}
            {request.registrationNo ? (
              <Text style={styles.fine}>Registration {request.registrationNo}</Text>
            ) : null}
          </View>
        </>
      ) : null}

      {request.rejectedReason ? (
        <View style={styles.card}>
          <Text style={styles.cardTitle}>We couldn't proceed</Text>
          <Text style={styles.body}>{request.rejectedReason}</Text>
        </View>
      ) : null}
    </>
  );
}

function PrimaryButton({
  label,
  onPress,
  disabled,
}: {
  label: string;
  onPress: () => void;
  disabled?: boolean;
}) {
  return (
    <Pressable
      onPress={onPress}
      disabled={disabled}
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!disabled }}
      style={[styles.cta, disabled && styles.ctaOff]}
    >
      <Text style={styles.ctaText}>{label}</Text>
    </Pressable>
  );
}

function Row({ label, value, emphasis }: { label: string; value: number; emphasis?: boolean }) {
  return (
    <View style={styles.row}>
      <Text style={[styles.rowLabel, emphasis && styles.rowStrong]}>{label}</Text>
      <Text style={[styles.rowValue, emphasis && styles.rowStrong]}>{formatMoney(value)}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  back: { fontSize: 28, lineHeight: 30, color: theme.colors.ink.DEFAULT },
  title: {
    fontFamily: 'Geist-Bold',
    fontSize: 20,
    color: theme.colors.ink.DEFAULT,
  },
  scroll: { padding: theme.spacing[4], gap: theme.spacing[3], paddingBottom: theme.spacing[10] },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: theme.spacing[6] },
  lead: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    lineHeight: 22,
    color: theme.colors.ink.soft,
  },
  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.6,
    color: theme.colors.ink.soft,
    marginTop: theme.spacing[2],
  },
  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    gap: theme.spacing[2],
    ...theme.shadow[1],
  },
  cardTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  body: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    lineHeight: 20,
    color: theme.colors.ink.soft,
  },
  fine: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    lineHeight: 18,
    color: theme.colors.ink.soft,
  },
  notice: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    lineHeight: 18,
    color: theme.colors.ink.DEFAULT,
    backgroundColor: theme.colors.amber.tint,
    borderRadius: theme.radius.DEFAULT,
    padding: theme.spacing[3],
  },
  row: { flexDirection: 'row', justifyContent: 'space-between', gap: theme.spacing[3] },
  rowLabel: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  rowValue: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  rowStrong: { fontFamily: 'Inter-SemiBold', color: theme.colors.ink.DEFAULT },
  input: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
  },
  termRow: { flexDirection: 'row', gap: theme.spacing[2] },
  termChip: {
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[2],
    borderRadius: theme.radius.full,
    backgroundColor: theme.colors.mist.DEFAULT,
  },
  termChipOn: { backgroundColor: theme.colors.ink.DEFAULT },
  termChipText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  termChipTextOn: { color: theme.colors.paper },
  cta: {
    marginTop: theme.spacing[3],
    borderRadius: theme.radius.DEFAULT,
    backgroundColor: theme.colors.ink.DEFAULT,
    paddingVertical: theme.spacing[4],
    alignItems: 'center',
  },
  ctaOff: { opacity: 0.45 },
  ctaText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
  },
  secondary: { alignItems: 'center', paddingVertical: theme.spacing[3] },
  secondaryText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    textDecorationLine: 'underline',
  },
  docRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.DEFAULT,
    padding: theme.spacing[4],
  },
  docMain: { flex: 1, gap: theme.spacing[1] },
  docActions: { alignItems: 'flex-end', gap: theme.spacing[1] },
  docLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  docState: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  docRemove: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.soft,
    textDecorationLine: 'underline',
  },
  stepRow: { flexDirection: 'row', alignItems: 'center', gap: theme.spacing[3] },
  stepDot: { fontSize: 12, color: theme.colors.ink.soft },
  stepDotOn: { color: theme.colors.ink.DEFAULT },
  stepLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  stepLabelOn: { fontFamily: 'Inter-SemiBold', color: theme.colors.ink.DEFAULT },
});
