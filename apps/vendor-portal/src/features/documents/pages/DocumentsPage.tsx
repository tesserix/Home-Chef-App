import { useRef, useState } from 'react';
import { toast } from 'sonner';
import { format } from 'date-fns';
import { AlertTriangle, FileText, Upload } from 'lucide-react';
import { Card } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import { Badge } from '@/shared/components/ui/Badge';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import {
  documentLabel,
  EXPECTED_DOCS,
  useAddDocument,
  useDocuments,
  useExpiringDocuments,
  useReplaceDocument,
  type ChefDocument,
} from '../hooks/useDocuments';
import { FssaiOfferCard } from '@/features/fssai/components/FssaiOfferCard';

// Compliance documents — the web twin of
// apps/mobile-vendor/app/documents/renew.tsx.
//
// The portal had no documents screen. A chef whose FSSAI licence was about to
// lapse could be warned on their phone but had no way to see or replace
// anything from the web — and a lapsed licence stops them trading.

export function DocumentsPage() {
  const { data: docs = [], isLoading } = useDocuments();
  const { data: expiring = [] } = useExpiringDocuments();

  // Anything rejected or expiring is what the chef actually has to act on, so
  // it leads. The rest is reference.
  const expiringIds = new Set(expiring.map((e) => e.id));
  const needsAction = docs.filter((d) => d.status === 'rejected' || expiringIds.has(d.id));
  const rest = docs.filter((d) => !needsAction.includes(d));
  // What onboarding wanted but never got.
  const missing = EXPECTED_DOCS.filter((e) => !docs.some((d) => d.type === e.type));

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">Documents</h1>
      <p className="mt-1 text-sm text-ink-soft">
        Upload anything onboarding still needs, or renew what is about to lapse. An expired
        licence stops new orders.
      </p>

      {isLoading ? (
        <Skeleton className="mt-8 h-40 w-full" />
      ) : (
        <>
          {missing.length > 0 && (
            <section className="mt-6">
              <h2 className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
                Still needed
              </h2>
              <div className="mt-2 flex flex-col gap-3">
                {missing.map((m) => (
                  <MissingDocumentCard key={m.type} type={m.type} why={m.why} />
                ))}
              </div>
              {/* A chef with no FSSAI licence cannot upload one. This is the
                  screen where they find that out. */}
              {missing.some((m) => m.type === 'fssai_license') && (
                <div className="mt-3">
                  <FssaiOfferCard />
                </div>
              )}
            </section>
          )}
          {needsAction.length > 0 && (
            <section className="mt-6">
              <h2 className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
                Needs your attention
              </h2>
              <div className="mt-2 flex flex-col gap-3">
                {needsAction.map((d) => (
                  <DocumentCard
                    key={d.id}
                    doc={d}
                    daysLeft={expiring.find((e) => e.id === d.id)?.daysUntilExpiry}
                  />
                ))}
              </div>
            </section>
          )}
          {rest.length > 0 && (
            <section className="mt-8">
              <h2 className="text-xs font-semibold uppercase tracking-wide text-ink-muted">
                On file
              </h2>
              <div className="mt-2 flex flex-col gap-3">
                {rest.map((d) => (
                  <DocumentCard key={d.id} doc={d} />
                ))}
              </div>
            </section>
          )}
        </>
      )}
    </div>
  );
}

/** A document onboarding asked for that the chef never uploaded. */
function MissingDocumentCard({ type, why }: { type: string; why: string }) {
  const add = useAddDocument();
  const fileRef = useRef<HTMLInputElement>(null);
  const [expiry, setExpiry] = useState('');
  const carriesExpiry = type === 'fssai_license';

  function onPick(file?: File | null) {
    if (!file) return;
    add.mutate(
      { type, file, expiryDate: expiry || undefined },
      {
        onSuccess: () => toast.success('Uploaded — it goes to admins for verification.'),
        onError: () => toast.error('Could not upload that file. Please try again.'),
      },
    );
  }

  return (
    <Card className="p-4">
      <div className="flex items-start gap-3">
        <FileText className="mt-0.5 h-5 w-5 shrink-0 text-ink-muted" aria-hidden="true" />
        <div className="min-w-0">
          <p className="font-medium text-foreground">{documentLabel(type)}</p>
          <p className="text-xs text-ink-muted">{why}</p>
        </div>
      </div>

      <div className="mt-3 flex flex-wrap items-end gap-2">
        {carriesExpiry && (
          <div>
            <label htmlFor={`add-expiry-${type}`} className="block text-xs font-medium text-ink-soft">
              Expiry date
            </label>
            <input
              id={`add-expiry-${type}`}
              type="date"
              value={expiry}
              onChange={(e) => setExpiry(e.target.value)}
              className="input-base mt-1 w-44 tabular-nums"
            />
          </div>
        )}
        <input
          ref={fileRef}
          type="file"
          accept="image/*,application/pdf"
          className="hidden"
          onChange={(e) => onPick(e.target.files?.[0])}
        />
        <Button size="sm" disabled={add.isPending} onClick={() => fileRef.current?.click()}>
          <Upload className="mr-1 h-4 w-4" aria-hidden="true" />
          {add.isPending ? 'Uploading…' : 'Upload'}
        </Button>
      </div>
    </Card>
  );
}

const STATUS_META = {
  verified: { label: 'Verified', variant: 'success' as const },
  pending: { label: 'In review', variant: 'warning' as const },
  rejected: { label: 'Rejected', variant: 'error' as const },
};

function DocumentCard({ doc, daysLeft }: { doc: ChefDocument; daysLeft?: number }) {
  const replace = useReplaceDocument();
  const fileRef = useRef<HTMLInputElement>(null);
  const [expiry, setExpiry] = useState(doc.expiryDate ? doc.expiryDate.slice(0, 10) : '');
  const meta = STATUS_META[doc.status] ?? STATUS_META.pending;

  function onPick(file?: File | null) {
    if (!file) return;
    replace.mutate(
      { docId: doc.id, file, expiryDate: expiry || undefined },
      {
        onSuccess: () => toast.success('Uploaded — it goes back for verification.'),
        onError: () => toast.error('Could not upload that file. Please try again.'),
      },
    );
  }

  return (
    <Card className="p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3">
          <FileText className="mt-0.5 h-5 w-5 shrink-0 text-ink-muted" aria-hidden="true" />
          <div className="min-w-0">
            <p className="font-medium text-foreground">{documentLabel(doc.type)}</p>
            <p className="truncate text-xs text-ink-muted" title={doc.fileName}>
              {doc.fileName}
            </p>
            {doc.expiryDate && (
              <p className="text-xs text-ink-muted tabular-nums">
                Expires {format(new Date(doc.expiryDate), 'd MMM yyyy')}
              </p>
            )}
          </div>
        </div>
        <Badge variant={meta.variant}>{meta.label}</Badge>
      </div>

      {daysLeft != null && (
        <p className="mt-2 flex items-center gap-1.5 text-xs font-medium text-amber">
          <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />
          Expires in {daysLeft} day{daysLeft === 1 ? '' : 's'} — renew it now to keep trading.
        </p>
      )}

      {doc.status === 'rejected' && doc.rejectionReason && (
        <p className="mt-2 rounded-lg border border-mist bg-paper p-2 text-xs text-ink-soft">
          <span className="font-medium text-foreground">Why it was rejected:</span>{' '}
          {doc.rejectionReason}
        </p>
      )}

      <div className="mt-3 flex flex-wrap items-end gap-2">
        <div>
          <label
            htmlFor={`expiry-${doc.id}`}
            className="block text-xs font-medium text-ink-soft"
          >
            New expiry (if it has one)
          </label>
          <input
            id={`expiry-${doc.id}`}
            type="date"
            value={expiry}
            onChange={(e) => setExpiry(e.target.value)}
            className="input-base mt-1 w-44 tabular-nums"
          />
        </div>
        <input
          ref={fileRef}
          type="file"
          accept="image/*,application/pdf"
          className="hidden"
          onChange={(e) => onPick(e.target.files?.[0])}
        />
        <Button
          size="sm"
          variant="secondary"
          disabled={replace.isPending}
          onClick={() => fileRef.current?.click()}
        >
          <Upload className="mr-1 h-4 w-4" aria-hidden="true" />
          {replace.isPending ? 'Uploading…' : 'Replace'}
        </Button>
      </div>
    </Card>
  );
}

export default DocumentsPage;
