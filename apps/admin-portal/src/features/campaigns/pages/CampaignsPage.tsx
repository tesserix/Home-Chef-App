import { useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { Megaphone, Loader2, Plus, Send, Clock, Beaker, X, BarChart3 } from 'lucide-react';
import { Button } from '@/shared/components/ui/Button';
import {
  useCampaigns,
  useCreateCampaign,
  useUpdateCampaign,
  usePreviewSegment,
  useCampaignAction,
  useCampaignMetrics,
  parseSegment,
  type Campaign,
  type CampaignInput,
  type SegmentCriteria,
  type SegmentPreview,
} from '@/features/campaigns/hooks/useCampaigns';

const STATUS_STYLE: Record<Campaign['status'], string> = {
  draft: 'bg-muted text-muted-foreground',
  scheduled: 'bg-info/10 text-info',
  queued: 'bg-info/10 text-info',
  sending: 'bg-warning/10 text-warning',
  sent: 'bg-success/10 text-success',
  cancelled: 'bg-muted text-muted-foreground',
};

const EMPTY: CampaignInput = {
  name: '',
  sendPush: true,
  sendEmail: false,
  pushTitle: '',
  pushBody: '',
  emailSubject: '',
  emailHtml: '',
  segment: { recency: '', subscription: '' },
};

const field =
  'h-9 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring';
const label = 'mb-1 block text-xs font-medium text-muted-foreground';

export default function CampaignsPage() {
  const { data, isLoading } = useCampaigns();
  const campaigns = (data as unknown as { data: Campaign[] } | undefined)?.data ?? [];
  const [editing, setEditing] = useState<Campaign | 'new' | null>(null);

  return (
    <div className="space-y-6">
      <div className="page-header flex items-start justify-between">
        <div>
          <h1 className="page-title">Marketing Campaigns</h1>
          <p className="page-description">
            Segmented push + email campaigns over owned channels — lapsed win-back, zone launches,
            announcements. Consent &amp; opt-out are always honored.
          </p>
        </div>
        {!editing && (
          <Button variant="primary" onClick={() => setEditing('new')}>
            <Plus className="h-4 w-4" /> New campaign
          </Button>
        )}
      </div>

      {editing ? (
        <CampaignEditor campaign={editing === 'new' ? null : editing} onClose={() => setEditing(null)} />
      ) : isLoading ? (
        <div className="flex items-center justify-center rounded-xl border border-border bg-card py-10 text-muted-foreground">
          <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading…
        </div>
      ) : campaigns.length === 0 ? (
        <div className="rounded-xl border border-border bg-card p-8 text-center text-muted-foreground">
          No campaigns yet. Create your first to reach customers on push &amp; email.
        </div>
      ) : (
        <div className="space-y-3">
          {campaigns.map((c) => (
            <CampaignRow key={c.id} campaign={c} onEdit={() => setEditing(c)} />
          ))}
        </div>
      )}
    </div>
  );
}

function CampaignRow({ campaign, onEdit }: { campaign: Campaign; onEdit: () => void }) {
  const action = useCampaignAction();
  const [showMetrics, setShowMetrics] = useState(false);
  const editable = campaign.status === 'draft' || campaign.status === 'scheduled';

  const run = (a: 'send' | 'cancel') => {
    if (a === 'send' && !window.confirm(`Send "${campaign.name}" now to its full audience?`)) return;
    if (a === 'cancel' && !window.confirm('Cancel this campaign?')) return;
    action.mutate(
      { id: campaign.id, action: a },
      {
        onSuccess: () => toast.success(a === 'send' ? 'Campaign queued for sending' : 'Campaign cancelled'),
        onError: () => toast.error('Action failed'),
      },
    );
  };

  const channels = [campaign.sendPush && 'Push', campaign.sendEmail && 'Email'].filter(Boolean).join(' + ');

  return (
    <div className="rounded-xl border border-border bg-card p-5">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="truncate font-semibold text-foreground">{campaign.name}</span>
            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_STYLE[campaign.status]}`}>
              {campaign.status}
            </span>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            {channels || 'No channels'}
            {campaign.recipients > 0 && ` · ${campaign.recipients.toLocaleString('en-IN')} recipients`}
            {campaign.scheduledAt && campaign.status === 'scheduled' && ` · scheduled ${new Date(campaign.scheduledAt).toLocaleString()}`}
            {campaign.sentAt && ` · sent ${new Date(campaign.sentAt).toLocaleString()}`}
          </p>
        </div>
        <div className="flex flex-shrink-0 items-center gap-2">
          {campaign.status === 'sent' && (
            <Button variant="ghost" size="sm" onClick={() => setShowMetrics((s) => !s)}>
              <BarChart3 className="h-4 w-4" /> Metrics
            </Button>
          )}
          {editable && (
            <>
              <Button variant="ghost" size="sm" onClick={onEdit}>Edit</Button>
              <Button variant="primary" size="sm" isLoading={action.isPending} onClick={() => run('send')}>
                <Send className="h-4 w-4" /> Send now
              </Button>
              <Button variant="ghost" size="sm" onClick={() => run('cancel')}>Cancel</Button>
            </>
          )}
        </div>
      </div>
      {showMetrics && <CampaignMetricsPanel id={campaign.id} />}
    </div>
  );
}

function CampaignMetricsPanel({ id }: { id: string }) {
  const { data, isLoading } = useCampaignMetrics(id);
  if (isLoading || !data) {
    return <div className="mt-4 text-sm text-muted-foreground">Loading metrics…</div>;
  }
  const cell = (lbl: string, v: number) => (
    <div className="rounded-lg bg-muted/50 p-3">
      <p className="text-xs text-muted-foreground">{lbl}</p>
      <p className="mt-0.5 text-lg font-semibold tabular-nums text-foreground">{v.toLocaleString('en-IN')}</p>
    </div>
  );
  return (
    <div className="mt-4 grid grid-cols-2 gap-3 border-t border-border pt-4 sm:grid-cols-4">
      {cell('Recipients', data.recipients)}
      {cell('Push sent', data.push.sent)}
      {cell('Email sent', data.email.sent)}
      {cell('Email opened', data.email.opened)}
    </div>
  );
}

function CampaignEditor({ campaign, onClose }: { campaign: Campaign | null; onClose: () => void }) {
  const create = useCreateCampaign();
  const update = useUpdateCampaign();
  const preview = usePreviewSegment();
  const action = useCampaignAction();

  const [form, setForm] = useState<CampaignInput>(() =>
    campaign
      ? {
          name: campaign.name,
          sendPush: campaign.sendPush,
          sendEmail: campaign.sendEmail,
          pushTitle: campaign.pushTitle,
          pushBody: campaign.pushBody,
          emailSubject: campaign.emailSubject,
          emailHtml: campaign.emailHtml,
          segment: parseSegment(campaign.segment),
        }
      : EMPTY,
  );
  const [scheduledAt, setScheduledAt] = useState('');
  const [audience, setAudience] = useState<SegmentPreview | null>(null);

  // Snapshot the form as first mounted so we can detect unsaved edits and
  // confirm before discarding them on close.
  const initialFormRef = useRef(form);
  const isDirty = JSON.stringify(form) !== JSON.stringify(initialFormRef.current) || scheduledAt !== '';
  const requestClose = () => {
    if (isDirty && !window.confirm('Discard unsaved changes to this campaign?')) return;
    onClose();
  };

  // Refresh the audience preview when the segment or channels change.
  useEffect(() => {
    const t = setTimeout(() => {
      preview.mutate(
        { segment: form.segment, sendPush: form.sendPush, sendEmail: form.sendEmail },
        { onSuccess: setAudience },
      );
    }, 400);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(form.segment), form.sendPush, form.sendEmail]);

  const setSeg = (patch: Partial<SegmentCriteria>) => setForm((f) => ({ ...f, segment: { ...f.segment, ...patch } }));

  const persist = async (): Promise<string | null> => {
    try {
      if (campaign) {
        await update.mutateAsync({ id: campaign.id, body: form });
        return campaign.id;
      }
      const created = await create.mutateAsync(form);
      return created.id;
    } catch (e: unknown) {
      const msg = e && typeof e === 'object' && 'error' in e ? String((e as { error?: unknown }).error) : 'Could not save';
      toast.error(msg);
      return null;
    }
  };

  const saveDraft = async () => {
    if ((await persist()) !== null) {
      toast.success('Campaign saved');
      onClose();
    }
  };

  const sendNow = async () => {
    const id = await persist();
    if (!id) return;
    if (!window.confirm('Send this campaign now to its full reachable audience?')) return;
    action.mutate(
      { id, action: 'send' },
      { onSuccess: () => { toast.success('Campaign queued for sending'); onClose(); }, onError: () => toast.error('Send failed') },
    );
  };

  const schedule = async () => {
    if (!scheduledAt) { toast.error('Pick a date & time'); return; }
    const id = await persist();
    if (!id) return;
    action.mutate(
      { id, action: 'schedule', body: { scheduledAt: new Date(scheduledAt).toISOString() } },
      { onSuccess: () => { toast.success('Campaign scheduled'); onClose(); }, onError: () => toast.error('Schedule failed') },
    );
  };

  const testSend = async () => {
    const id = await persist();
    if (!id) return;
    action.mutate(
      { id, action: 'test' },
      { onSuccess: () => toast.success('Test sent to your account'), onError: () => toast.error('Test send failed') },
    );
  };

  const saving = create.isPending || update.isPending;

  return (
    <div className="space-y-5 rounded-xl border border-border bg-card p-5">
      <div className="flex items-center justify-between">
        <h3 className="flex items-center gap-2 text-sm font-semibold text-foreground">
          <Megaphone className="h-5 w-5 text-primary" /> {campaign ? 'Edit campaign' : 'New campaign'}
        </h3>
        <button type="button" aria-label="Close" onClick={requestClose} className="rounded-lg p-1 hover:bg-muted">
          <X className="h-5 w-5 text-muted-foreground" />
        </button>
      </div>

      <div>
        <label className={label}>Campaign name</label>
        <input className={field} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="Diwali win-back" />
      </div>

      {/* Audience / segment builder */}
      <div className="rounded-lg border border-border p-4">
        <div className="mb-3 flex items-center justify-between">
          <span className="text-sm font-medium text-foreground">Audience</span>
          {audience && (
            <span className="text-xs text-muted-foreground">
              <span className="font-semibold tabular-nums text-foreground">{audience.matched.toLocaleString('en-IN')}</span> matched ·{' '}
              {form.sendPush && <>{audience.reachablePush.toLocaleString('en-IN')} push </>}
              {form.sendEmail && <>{audience.reachableEmail.toLocaleString('en-IN')} email </>}
              reachable
            </span>
          )}
        </div>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <div>
            <label className={label}>Order recency</label>
            <select className={field} value={form.segment.recency ?? ''} onChange={(e) => setSeg({ recency: e.target.value as SegmentCriteria['recency'] })}>
              <option value="">Any</option>
              <option value="active">Active (ordered recently)</option>
              <option value="lapsed">Lapsed (no recent order)</option>
            </select>
          </div>
          <div>
            <label className={label}>Recency window (days)</label>
            <input className={field} type="number" min={1} value={form.segment.recencyDays ?? 30} onChange={(e) => setSeg({ recencyDays: Number(e.target.value) })} />
          </div>
          <div>
            <label className={label}>Subscription</label>
            <select className={field} value={form.segment.subscription ?? ''} onChange={(e) => setSeg({ subscription: e.target.value as SegmentCriteria['subscription'] })}>
              <option value="">Any</option>
              <option value="active">Active subscribers</option>
              <option value="paused">Paused subscribers</option>
              <option value="none">No subscription</option>
            </select>
          </div>
          <div>
            <label className={label}>Cities / zones (comma-separated)</label>
            <input className={field} value={(form.segment.cities ?? []).join(', ')} onChange={(e) => setSeg({ cities: e.target.value.split(',').map((s) => s.trim()).filter(Boolean) })} placeholder="Bengaluru, Mumbai" />
          </div>
          <div>
            <label className={label}>New customers within (days, 0 = off)</label>
            <input className={field} type="number" min={0} value={form.segment.newWithinDays ?? 0} onChange={(e) => setSeg({ newWithinDays: Number(e.target.value) })} />
          </div>
        </div>
      </div>

      {/* Channels + content */}
      <div className="rounded-lg border border-border p-4">
        <div className="mb-3 flex gap-4">
          <label className="flex items-center gap-2 text-sm text-foreground">
            <input type="checkbox" checked={form.sendPush} onChange={(e) => setForm({ ...form, sendPush: e.target.checked })} /> Push
          </label>
          <label className="flex items-center gap-2 text-sm text-foreground">
            <input type="checkbox" checked={form.sendEmail} onChange={(e) => setForm({ ...form, sendEmail: e.target.checked })} /> Email
          </label>
        </div>
        {form.sendPush && (
          <div className="mb-4 grid gap-3">
            <div>
              <label className={label}>Push title</label>
              <input className={field} maxLength={120} value={form.pushTitle} onChange={(e) => setForm({ ...form, pushTitle: e.target.value })} />
            </div>
            <div>
              <label className={label}>Push body</label>
              <input className={field} value={form.pushBody} onChange={(e) => setForm({ ...form, pushBody: e.target.value })} />
            </div>
          </div>
        )}
        {form.sendEmail && (
          <div className="grid gap-3">
            <div>
              <label className={label}>Email subject</label>
              <input className={field} value={form.emailSubject} onChange={(e) => setForm({ ...form, emailSubject: e.target.value })} />
            </div>
            <div>
              <label className={label}>Email body (HTML)</label>
              <textarea className={`${field} h-32 py-2`} value={form.emailHtml} onChange={(e) => setForm({ ...form, emailHtml: e.target.value })} placeholder="<h1>…</h1>" />
              <p className="mt-1 text-[11px] text-muted-foreground">An unsubscribe link + open-tracking pixel are appended automatically.</p>
            </div>
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <input className={field} style={{ width: 'auto' }} type="datetime-local" value={scheduledAt} onChange={(e) => setScheduledAt(e.target.value)} />
          <Button variant="secondary" isLoading={saving || action.isPending} onClick={schedule}>
            <Clock className="h-4 w-4" /> Schedule
          </Button>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="ghost" isLoading={saving || action.isPending} onClick={testSend}>
            <Beaker className="h-4 w-4" /> Test to me
          </Button>
          <Button variant="ghost" isLoading={saving} onClick={saveDraft}>Save draft</Button>
          <Button variant="primary" isLoading={saving || action.isPending} onClick={sendNow}>
            <Send className="h-4 w-4" /> Send now
          </Button>
        </div>
      </div>
    </div>
  );
}
