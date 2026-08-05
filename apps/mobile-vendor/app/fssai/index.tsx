// FSSAI filing request — the chef asks us to obtain their registration.
//
// One screen, three states, driven entirely by the server's view of the
// request: no request yet (the offer + form), a request that needs something
// from the chef (pay or upload), and a request we are working on (the tracker).
// The app never decides what comes next — `fssaiChefAction` reads the status
// the server set, so a payment the server has not seen can never show as done.

import React, { useCallback, useMemo, useState } from 'react';
import {
  ActivityIndicator,
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
import { multipartConfig } from '@homechef/mobile-shared/api';
import { api } from '../../lib/api';
import { formatMoney } from '../../lib/format';
import {
  buildCashfreeCheckoutUrl,
  fssaiChefAction,
  fssaiStepIndex,
  FSSAI_DOCUMENT_LABELS,
  FSSAI_TRACKER_STEPS,
  fssaiStatusLabel,
  isFssaiClosed,
  type FssaiDocumentKind,
  type FssaiQuote,
  type FssaiRequest,
} from '../../lib/fssai';
import {
  useConfirmFssaiPayment,
  useCreateFssaiRequest,
  useFssaiQuote,
  useFssaiRequest,
} from '../../hooks/useFssai';

const RETURN_URL = 'homechef-vendor://fssai';

export default function FssaiScreen() {
  const { show: showToast } = useToast();
  const quoteQuery = useFssaiQuote();
  const requestQuery = useFssaiRequest();
  const createRequest = useCreateFssaiRequest();
  const confirmPayment = useConfirmFssaiPayment();
  const [uploading, setUploading] = useState<FssaiDocumentKind | null>(null);
  const [termYears, setTermYears] = useState(1);
  const [form, setForm] = useState<FssaiForm>({
    kitchenName: '',
    applicantName: '',
    contactPhone: '',
    contactEmail: '',
    addressLine1: '',
    addressLine2: '',
    city: '',
    state: '',
    postalCode: '',
  });

  const request = requestQuery.data?.request ?? null;
  const enabled = quoteQuery.data?.enabled ?? requestQuery.data?.enabled ?? false;
  const terms = quoteQuery.data?.terms ?? [];
  const quote = useMemo(
    () => terms.find((t) => t.termYears === termYears) ?? quoteQuery.data?.quote,
    [terms, termYears, quoteQuery.data],
  );
  const action = fssaiChefAction(request);

  // Opens the hosted Cashfree page and, whatever the browser reports back, asks
  // the SERVER whether the money arrived. The browser's answer is a UX hint
  // only — Cashfree hands the client nothing it could prove a payment with.
  const pay = useCallback(
    async (requestId: string, paymentSessionId: string, env: string) => {
      try {
        await WebBrowser.openAuthSessionAsync(
          buildCashfreeCheckoutUrl({ paymentSessionId, env, returnUrl: RETURN_URL }),
          RETURN_URL,
        );
      } catch {
        // Even a browser error is not proof of failure — fall through and let
        // the server adjudicate rather than telling a chef who paid that they
        // did not.
      }
      try {
        await confirmPayment.mutateAsync(requestId);
        showToast({ message: 'Payment received. Now add your documents.', tone: 'success' });
      } catch {
        showToast({
          message: "We haven't seen the payment yet. If you paid, pull to refresh in a moment.",
          tone: 'error',
        });
      }
    },
    [confirmPayment, showToast],
  );

  async function handleStart() {
    const missing = Object.entries(form).find(
      ([k, v]) => k !== 'addressLine2' && v.trim() === '',
    );
    if (missing) {
      showToast({ message: 'Fill in every field so we can complete your form.', tone: 'error' });
      return;
    }
    try {
      const res = await createRequest.mutateAsync({ ...form, termYears });
      await pay(res.request.id, res.cashfreePaymentSessionId, res.cashfreeEnv);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Could not start the request';
      showToast({ message, tone: 'error' });
    }
  }

  async function handleUpload(kind: FssaiDocumentKind) {
    if (!request) return;
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      showToast({ message: 'Allow photo access to attach your documents.', tone: 'error' });
      return;
    }
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      quality: 0.85,
    });
    if (result.canceled || !result.assets[0]) return;

    setUploading(kind);
    try {
      const asset = result.assets[0];
      const formData = new FormData();
      const filename = asset.uri.split('/').pop() ?? `${kind}.jpg`;
      formData.append('file', { uri: asset.uri, name: filename, type: 'image/jpeg' } as unknown as Blob);
      formData.append('kind', kind);
      await api.post(
        `/chef/fssai/requests/${request.id}/upload`,
        formData,
        multipartConfig(),
      );
      await requestQuery.refetch();
      showToast({ message: `${FSSAI_DOCUMENT_LABELS[kind]} added.`, tone: 'success' });
    } catch {
      showToast({ message: 'Upload failed. Try again.', tone: 'error' });
    } finally {
      setUploading(null);
    }
  }

  if (quoteQuery.isLoading || requestQuery.isLoading) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <Header />
        <View style={styles.centered}>
          <ActivityIndicator color={theme.colors.ink.DEFAULT} />
        </View>
      </SafeAreaView>
    );
  }

  if (!enabled && !request) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <Header />
        <View style={styles.centered}>
          <Text style={styles.body}>
            We aren't taking FSSAI applications at the moment. You can still apply
            yourself on the government portal — see fe3dr.com/fssai for the steps.
          </Text>
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <Header />
      <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
        {request && !isFssaiClosed(request.status) ? (
          <ActiveRequest
            request={request}
            action={action}
            uploading={uploading}
            onUpload={handleUpload}
            onPay={() => {
              // Re-mint the session: a Cashfree order the chef abandoned may
              // have expired, so the safe move is to ask the server again.
              showToast({
                message: 'Reopening your payment…',
                tone: 'success',
              });
              void requestQuery.refetch();
            }}
          />
        ) : (
          <NewRequest
            quote={quote}
            terms={terms}
            termYears={termYears}
            setTermYears={setTermYears}
            form={form}
            setForm={setForm}
            onStart={handleStart}
            submitting={createRequest.isPending || confirmPayment.isPending}
            nonRefundableNotice={quoteQuery.data?.nonRefundableNotice}
            lastRequest={request && isFssaiClosed(request.status) ? request : null}
          />
        )}
      </ScrollView>
    </SafeAreaView>
  );
}

function Header() {
  return (
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
  );
}

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

interface NewRequestProps {
  quote?: FssaiQuote;
  terms: FssaiQuote[];
  termYears: number;
  setTermYears: (n: number) => void;
  form: FssaiForm;
  setForm: React.Dispatch<React.SetStateAction<FssaiForm>>;
  onStart: () => void;
  submitting: boolean;
  nonRefundableNotice?: string;
  lastRequest: FssaiRequest | null;
}

function NewRequest({
  quote,
  terms,
  termYears,
  setTermYears,
  form,
  setForm,
  onStart,
  submitting,
  nonRefundableNotice,
  lastRequest,
}: NewRequestProps) {
  return (
    <>
      {lastRequest?.registrationNo ? (
        <View style={styles.card}>
          <Text style={styles.cardTitle}>Your last registration</Text>
          <Text style={styles.body}>
            {lastRequest.registrationNo} — obtained {new Date(lastRequest.issuedAt ?? '').toLocaleDateString('en-IN')}
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
            accessibilityLabel={`${t.termYears} year${t.termYears === 1 ? '' : 's'}, ${formatMoney(t.total)}`}
            style={[styles.termChip, termYears === t.termYears && styles.termChipOn]}
          >
            <Text style={[styles.termChipText, termYears === t.termYears && styles.termChipTextOn]}>
              {t.termYears}y
            </Text>
          </Pressable>
        ))}
      </View>

      {quote ? (
        <View style={styles.card}>
          <Text style={styles.cardTitle}>What you pay</Text>
          {/* Every line is the server's. The app renders them and never re-adds
              the total — the figure a chef agrees to must be the figure charged. */}
          <Row label="FSSAI registration fee" value={quote.governmentFee} />
          <Row label={`GST on it (${quote.gstPercent}%)`} value={quote.governmentTax} />
          <Row label="Our filing fee" value={quote.serviceFee} />
          <Row label={`GST on our fee (${quote.gstPercent}%)`} value={quote.serviceTax} />
          <Row label="Total" value={quote.total} emphasis />
          <Text style={styles.fine}>
            The registration is issued in your name and stays yours. We pay FSSAI
            on your behalf out of this amount.
          </Text>
        </View>
      ) : null}

      <Text style={styles.sectionLabel}>YOUR DETAILS</Text>
      {(
        [
          ['kitchenName', 'Kitchen name'],
          ['applicantName', 'Your full name'],
          ['contactPhone', 'Phone'],
          ['contactEmail', 'Email'],
          ['addressLine1', 'Address line 1'],
          ['addressLine2', 'Address line 2 (optional)'],
          ['city', 'City'],
          ['state', 'State'],
          ['postalCode', 'PIN code'],
        ] as const
      ).map(([key, label]) => (
        <TextInput
          key={key}
          style={styles.input}
          placeholder={label}
          placeholderTextColor={theme.colors.ink.soft}
          value={form[key]}
          onChangeText={(v) => setForm({ ...form, [key]: v })}
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

      {nonRefundableNotice ? <Text style={styles.notice}>{nonRefundableNotice}</Text> : null}

      <Pressable
        onPress={onStart}
        disabled={submitting}
        accessibilityRole="button"
        accessibilityLabel={quote ? `Pay ${formatMoney(quote.total)} and apply` : 'Pay and apply'}
        style={[styles.cta, submitting && { opacity: 0.6 }]}
      >
        <Text style={styles.ctaText}>
          {submitting ? 'Opening payment…' : quote ? `Pay ${formatMoney(quote.total)} and apply` : 'Continue'}
        </Text>
      </Pressable>
    </>
  );
}

interface ActiveRequestProps {
  request: FssaiRequest;
  action: 'pay' | 'upload' | null;
  uploading: FssaiDocumentKind | null;
  onUpload: (kind: FssaiDocumentKind) => void;
  onPay: () => void;
}

function ActiveRequest({ request, action, uploading, onUpload, onPay }: ActiveRequestProps) {
  const step = fssaiStepIndex(request.status);
  const have = new Set(request.documents.map((d) => d.kind));

  return (
    <>
      <View style={styles.card}>
        <Text style={styles.cardTitle}>{fssaiStatusLabel(request.status)}</Text>
        <Text style={styles.body}>
          {request.kitchenName} · {request.termYears} year
          {request.termYears === 1 ? '' : 's'} · {formatMoney(request.feeTotal)}
          {request.paidAt ? ' paid' : ''}
        </Text>
      </View>

      {action === 'pay' ? (
        <View style={styles.card}>
          <Text style={styles.cardTitle}>Finish your payment</Text>
          <Text style={styles.body}>
            We haven't received the payment for this request yet.
          </Text>
          <Pressable onPress={onPay} accessibilityRole="button" style={styles.cta}>
            <Text style={styles.ctaText}>Refresh</Text>
          </Pressable>
        </View>
      ) : null}

      {action === 'upload' ? (
        <>
          <Text style={styles.sectionLabel}>YOUR DOCUMENTS</Text>
          <Text style={styles.body}>
            FSSAI needs a photo of you and a government photo ID. Add an address
            proof only if your kitchen is somewhere other than the address on
            that ID.
          </Text>
          {(['photo', 'identity', 'address_proof'] as FssaiDocumentKind[]).map((kind) => (
            <Pressable
              key={kind}
              onPress={() => onUpload(kind)}
              disabled={uploading !== null}
              accessibilityRole="button"
              accessibilityLabel={`Add ${FSSAI_DOCUMENT_LABELS[kind]}`}
              style={styles.docRow}
            >
              <Text style={styles.docLabel}>
                {FSSAI_DOCUMENT_LABELS[kind]}
                {kind === 'address_proof' ? ' (only if needed)' : ''}
              </Text>
              <Text style={styles.docState}>
                {uploading === kind ? 'Uploading…' : have.has(kind) ? 'Added ✓' : 'Add'}
              </Text>
            </Pressable>
          ))}
        </>
      ) : null}

      {step >= 0 ? (
        <>
          <Text style={styles.sectionLabel}>PROGRESS</Text>
          <View style={styles.card}>
            {FSSAI_TRACKER_STEPS.map((s, i) => (
              <View key={s.key} style={styles.stepRow}>
                <Text style={[styles.stepDot, i <= step && styles.stepDotOn]}>
                  {i <= step ? '●' : '○'}
                </Text>
                <Text style={[styles.stepLabel, i <= step && styles.stepLabelOn]}>{s.label}</Text>
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
  rowLabel: { flex: 1, fontFamily: 'Inter', fontSize: theme.typography.size.bodySm.size, color: theme.colors.ink.soft },
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
  ctaText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
  },
  docRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.DEFAULT,
    padding: theme.spacing[4],
  },
  docLabel: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  docState: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
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
