import { useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { toast } from 'sonner';
import { BadgeCheck, Upload } from 'lucide-react';
import { Card } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import { formatCurrency } from '@/shared/utils/format';
import {
  buildCashfreeCheckoutUrl,
  fssaiStatusLabel,
  fssaiStepIndex,
  FSSAI_DOCUMENT_HINTS,
  FSSAI_DOCUMENT_KINDS,
  FSSAI_DOCUMENT_LABELS,
  FSSAI_TRACKER_STEPS,
  isFssaiClosed,
  isFssaiPaid,
  useCancelFssaiRequest,
  useConfirmFssaiPayment,
  useCreateFssaiRequest,
  useFssaiQuote,
  useFssaiRequest,
  useRemoveFssaiDocument,
  useStartFssaiPayment,
  useUploadFssaiDocument,
  type FssaiDocumentKind,
  type FssaiQuote,
  type FssaiRequest,
} from '../hooks/useFssai';

// FSSAI filing request — the web twin of apps/mobile-vendor/app/fssai/index.tsx.
//
// Same three states in the same order: the offer and form, a draft gathering
// documents, then the tracker. The chef pays only once the documents are in, so
// `canPay` is the server's answer and the page never infers it.

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

type FormKey = (typeof FORM_FIELDS)[number][0];
type FssaiForm = Record<FormKey, string>;

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

// Cashfree returns here after checkout. The flag is only a hint that the chef
// came back — the server still adjudicates whether the money arrived.
const RETURN_FLAG = 'fssaiReturn';

function errorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}

export function FssaiPage() {
  const quoteQuery = useFssaiQuote();
  const requestQuery = useFssaiRequest();
  const createRequest = useCreateFssaiRequest();
  const startPayment = useStartFssaiPayment();
  const confirmPayment = useConfirmFssaiPayment();
  const cancelRequest = useCancelFssaiRequest();

  const [searchParams, setSearchParams] = useSearchParams();
  const [termYears, setTermYears] = useState(1);
  const [form, setForm] = useState<FssaiForm>(EMPTY_FORM);

  const request = requestQuery.data?.request ?? null;
  const enabled = quoteQuery.data?.enabled ?? requestQuery.data?.enabled ?? false;
  const terms = useMemo(() => quoteQuery.data?.terms ?? [], [quoteQuery.data]);
  const draft = request && !isFssaiPaid(request.status) ? request : null;
  const quote = useMemo(
    () => terms.find((t) => t.termYears === termYears) ?? quoteQuery.data?.quote,
    [terms, termYears, quoteQuery.data],
  );

  // Seed from onboarding once, and never over an edit the chef has already
  // made — which is what assigning the prefill on every fetch would do.
  const seeded = useRef(false);
  const prefill = requestQuery.data?.prefill;
  useEffect(() => {
    if (seeded.current || !prefill || draft) return;
    seeded.current = true;
    setForm((prev) => {
      const next = { ...prev };
      (Object.keys(EMPTY_FORM) as FormKey[]).forEach((k) => {
        next[k] = prev[k] || prefill[k] || '';
      });
      return next;
    });
  }, [prefill, draft]);

  // Coming back from the hosted checkout: ask the SERVER whether the capture
  // landed. Runs once — the flag is cleared immediately so a refresh or a
  // re-render cannot fire a second confirm.
  const confirmed = useRef(false);
  useEffect(() => {
    if (!searchParams.get(RETURN_FLAG) || confirmed.current || !draft) return;
    confirmed.current = true;
    setSearchParams({}, { replace: true });
    confirmPayment
      .mutateAsync(draft.id)
      .then((updated) => {
        toast[isFssaiPaid(updated.status) ? 'success' : 'error'](
          isFssaiPaid(updated.status)
            ? 'Payment received. Your request is with our team.'
            : "We haven't seen the payment yet. Refresh in a moment.",
        );
      })
      .catch((err: unknown) => toast.error(errorMessage(err, 'Could not confirm the payment')));
  }, [searchParams, draft, confirmPayment, setSearchParams]);

  async function handleCreateDraft() {
    const missing = FORM_FIELDS.find(
      ([key]) => key !== 'addressLine2' && form[key].trim() === '',
    );
    if (missing) {
      toast.error(`${missing[1]} is needed to complete your form.`);
      return;
    }
    try {
      await createRequest.mutateAsync({ ...form, termYears });
      toast.success('Now add your documents.');
    } catch (err) {
      toast.error(errorMessage(err, 'Could not start the request'));
    }
  }

  async function handlePay() {
    if (!draft) return;
    try {
      const session = await startPayment.mutateAsync(draft.id);
      const returnUrl = `${window.location.origin}${window.location.pathname}?${RETURN_FLAG}=1`;
      window.location.assign(
        buildCashfreeCheckoutUrl({
          paymentSessionId: session.cashfreePaymentSessionId,
          env: session.cashfreeEnv,
          returnUrl,
        }),
      );
    } catch (err) {
      toast.error(errorMessage(err, "We couldn't start the payment. Nothing has been charged."));
    }
  }

  async function handleCancel() {
    if (!draft) return;
    if (!window.confirm('Discard this request? Nothing has been charged.')) return;
    try {
      await cancelRequest.mutateAsync(draft.id);
      setForm(EMPTY_FORM);
      toast.success('Request discarded.');
    } catch (err) {
      toast.error(errorMessage(err, 'Could not discard the request'));
    }
  }

  if (quoteQuery.isLoading || requestQuery.isLoading) {
    return (
      <div className="mx-auto max-w-2xl space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">
        FSSAI registration
      </h1>
      <p className="mt-1 text-sm text-ink-soft">
        Every home kitchen needs an FSSAI registration by law. You can apply yourself on the
        government portal for free, or we can complete the whole application for you.
      </p>

      <div className="mt-6 space-y-4">
        {!enabled && !request ? (
          <Card className="p-5 text-sm text-ink-soft">
            We aren't taking FSSAI applications at the moment. You can still apply yourself —
            see fe3dr.com/fssai for the steps.
          </Card>
        ) : request && isFssaiPaid(request.status) ? (
          <Tracker request={request} />
        ) : draft ? (
          <DraftSection
            request={draft}
            notice={quoteQuery.data?.nonRefundableNotice}
            onPay={handlePay}
            onCancel={handleCancel}
            paying={startPayment.isPending || confirmPayment.isPending}
          />
        ) : (
          <NewRequestSection
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
      </div>
    </div>
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

function NewRequestSection({
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
        <Card className="p-5">
          <h2 className="text-sm font-semibold text-foreground">Your last registration</h2>
          <p className="mt-1 text-sm text-ink-soft">{lastRequest.registrationNo}</p>
        </Card>
      ) : null}

      <Card className="p-5">
        <h2 className="text-sm font-semibold text-foreground">How long for?</h2>
        <div className="mt-3 flex flex-wrap gap-2">
          {terms.map((t) => (
            <button
              key={t.termYears}
              type="button"
              onClick={() => setTermYears(t.termYears)}
              aria-pressed={termYears === t.termYears}
              className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
                termYears === t.termYears
                  ? 'bg-foreground text-background'
                  : 'bg-muted text-foreground hover:bg-muted/80'
              }`}
            >
              {t.termYears} year{t.termYears === 1 ? '' : 's'}
            </button>
          ))}
        </div>
      </Card>

      {quote ? <QuoteCard quote={quote} /> : null}

      <Card className="p-5">
        <h2 className="text-sm font-semibold text-foreground">Your details</h2>
        <div className="mt-3 grid gap-3 sm:grid-cols-2">
          {FORM_FIELDS.map(([key, label]) => (
            <label key={key} className="flex flex-col gap-1 text-sm">
              <span className="text-ink-soft">{label}</span>
              <input
                className="rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
                value={form[key]}
                onChange={(e) => setForm((prev) => ({ ...prev, [key]: e.target.value }))}
                type={key === 'contactEmail' ? 'email' : 'text'}
                inputMode={key === 'contactPhone' || key === 'postalCode' ? 'numeric' : 'text'}
              />
            </label>
          ))}
        </div>
        <p className="mt-3 text-xs text-ink-soft">
          Next you'll add two documents. Nothing is charged until they are in and you confirm
          the amount.
        </p>
        <Button className="mt-4 w-full" onClick={onContinue} disabled={submitting}>
          {submitting ? 'Saving…' : 'Continue to documents'}
        </Button>
      </Card>
    </>
  );
}

function QuoteCard({ quote }: { quote: FssaiQuote }) {
  return (
    <Card className="p-5">
      <h2 className="text-sm font-semibold text-foreground">What you pay</h2>
      {/* Every line is the server's. Rendered, never re-added — the figure a
          chef agrees to must be the figure charged. */}
      <dl className="mt-3 space-y-2 text-sm">
        <QuoteRow label="FSSAI registration fee" value={quote.governmentFee} />
        <QuoteRow label={`GST on it (${quote.gstPercent}%)`} value={quote.governmentTax} />
        <QuoteRow label="Our filing fee" value={quote.serviceFee} />
        <QuoteRow label={`GST on our fee (${quote.gstPercent}%)`} value={quote.serviceTax} />
        <QuoteRow label="Total" value={quote.total} emphasis />
      </dl>
      <p className="mt-3 text-xs text-ink-soft">
        The registration is issued in your name and stays yours. We pay FSSAI on your behalf out
        of this amount.
      </p>
    </Card>
  );
}

function QuoteRow({
  label,
  value,
  emphasis,
}: {
  label: string;
  value: number;
  emphasis?: boolean;
}) {
  return (
    <div className={`flex justify-between ${emphasis ? 'font-semibold text-foreground' : ''}`}>
      <dt className={emphasis ? '' : 'text-ink-soft'}>{label}</dt>
      <dd className="tabular-nums">{formatCurrency(value)}</dd>
    </div>
  );
}

interface DraftProps {
  request: FssaiRequest;
  notice?: string;
  onPay: () => void;
  onCancel: () => void;
  paying: boolean;
}

function DraftSection({ request, notice, onPay, onCancel, paying }: DraftProps) {
  return (
    <>
      <Card className="p-5">
        <h2 className="text-sm font-semibold text-foreground">{request.kitchenName}</h2>
        <p className="mt-1 text-sm text-ink-soft">
          {request.termYears} year{request.termYears === 1 ? '' : 's'} ·{' '}
          {formatCurrency(request.feeTotal)} · not sent yet
        </p>
      </Card>

      <Card className="p-5">
        <h2 className="text-sm font-semibold text-foreground">Your documents</h2>
        <p className="mt-1 text-sm text-ink-soft">
          FSSAI needs a photo of you and a government photo ID. Add an address proof only if your
          kitchen is somewhere other than the address on that ID.
        </p>
        <div className="mt-4 space-y-2">
          {FSSAI_DOCUMENT_KINDS.map((kind) => (
            <DocumentRow key={kind} requestId={request.id} kind={kind} request={request} />
          ))}
        </div>
      </Card>

      {notice ? (
        <p className="rounded-md bg-amber-50 p-3 text-xs text-foreground dark:bg-amber-950/30">
          {notice}
        </p>
      ) : null}

      <Button className="w-full" onClick={onPay} disabled={!request.canPay || paying}>
        {paying
          ? 'Opening payment…'
          : request.canPay
            ? `Pay ${formatCurrency(request.feeTotal)} and apply`
            : 'Add your photo and photo ID to continue'}
      </Button>

      <button
        type="button"
        onClick={onCancel}
        className="w-full text-center text-sm text-ink-soft underline"
      >
        Discard this request
      </button>
    </>
  );
}

function DocumentRow({
  requestId,
  kind,
  request,
}: {
  requestId: string;
  kind: FssaiDocumentKind;
  request: FssaiRequest;
}) {
  const upload = useUploadFssaiDocument();
  const remove = useRemoveFssaiDocument();
  const inputRef = useRef<HTMLInputElement>(null);
  const doc = request.documents.find((d) => d.kind === kind);
  const optional = kind === 'address_proof';

  async function handleFile(file: File | undefined) {
    if (!file) return;
    try {
      await upload.mutateAsync({ requestId, kind, file });
      toast.success(`${FSSAI_DOCUMENT_LABELS[kind]} added.`);
    } catch (err) {
      toast.error(errorMessage(err, 'Upload failed. Try again.'));
    }
  }

  return (
    <div className="flex items-center gap-3 rounded-md border border-border p-3">
      <div className="min-w-0 flex-1">
        <p className="text-sm text-foreground">
          {FSSAI_DOCUMENT_LABELS[kind]}
          {optional ? ' (only if needed)' : ''}
        </p>
        <p className="truncate text-xs text-ink-soft">
          {doc?.fileName || FSSAI_DOCUMENT_HINTS[kind]}
        </p>
      </div>
      {doc && optional ? (
        <button
          type="button"
          onClick={() => remove.mutate({ requestId, kind })}
          className="text-xs text-ink-soft underline"
        >
          Remove
        </button>
      ) : null}
      <input
        ref={inputRef}
        type="file"
        accept={optional ? 'image/jpeg,image/png,image/webp,application/pdf' : 'image/jpeg,image/png,image/webp'}
        className="hidden"
        onChange={(e) => void handleFile(e.target.files?.[0])}
      />
      <Button
        variant="outline"
        size="sm"
        onClick={() => inputRef.current?.click()}
        disabled={upload.isPending}
      >
        <Upload className="mr-1 h-3.5 w-3.5" />
        {upload.isPending ? 'Uploading…' : doc ? 'Replace' : 'Add'}
      </Button>
    </div>
  );
}

function Tracker({ request }: { request: FssaiRequest }) {
  const step = fssaiStepIndex(request.status);
  return (
    <>
      <Card className="p-5">
        <div className="flex items-center gap-2">
          <BadgeCheck className="h-4 w-4 text-ink-soft" />
          <h2 className="text-sm font-semibold text-foreground">
            {fssaiStatusLabel(request.status)}
          </h2>
        </div>
        <p className="mt-1 text-sm text-ink-soft">
          {request.kitchenName} · {request.termYears} year
          {request.termYears === 1 ? '' : 's'} · {formatCurrency(request.feeTotal)} paid
        </p>
      </Card>

      {step >= 0 ? (
        <Card className="p-5">
          <h2 className="text-sm font-semibold text-foreground">Progress</h2>
          <ol className="mt-3 space-y-2">
            {FSSAI_TRACKER_STEPS.map((s, i) => (
              <li key={s.key} className="flex items-center gap-3 text-sm">
                <span className={i <= step ? 'text-foreground' : 'text-ink-soft'}>
                  {i <= step ? '●' : '○'}
                </span>
                <span className={i <= step ? 'font-medium text-foreground' : 'text-ink-soft'}>
                  {s.label}
                </span>
              </li>
            ))}
          </ol>
          {request.applicationRef ? (
            <p className="mt-3 text-xs text-ink-soft">
              FoSCoS reference {request.applicationRef} — you can track this on the government
              portal yourself.
            </p>
          ) : null}
          {request.registrationNo ? (
            <p className="mt-1 text-xs text-ink-soft">Registration {request.registrationNo}</p>
          ) : null}
        </Card>
      ) : null}

      {request.rejectedReason ? (
        <Card className="p-5">
          <h2 className="text-sm font-semibold text-foreground">We couldn't proceed</h2>
          <p className="mt-1 text-sm text-ink-soft">{request.rejectedReason}</p>
        </Card>
      ) : null}
    </>
  );
}

export default FssaiPage;
