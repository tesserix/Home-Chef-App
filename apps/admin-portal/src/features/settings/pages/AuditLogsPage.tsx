import { Fragment, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';
import { ArrowLeft, FileClock } from 'lucide-react';
import { Button } from '@/shared/components/ui/Button';

interface AuditUser {
  firstName?: string;
  lastName?: string;
  email?: string;
}

interface AuditLog {
  id: string;
  userId: string | null;
  action: string;
  entityType: string;
  entityId: string;
  oldValue?: string;
  newValue?: string;
  ipAddress?: string;
  userAgent?: string;
  createdAt: string;
  user?: AuditUser;
}

interface AuditResponse {
  logs: AuditLog[];
  total: number;
  page: number;
  limit: number;
}

export default function AuditLogsPage() {
  const [page, setPage] = useState(1);
  const [action, setAction] = useState('');
  const [entityType, setEntityType] = useState('');
  const [from, setFrom] = useState('');
  const [to, setTo] = useState('');
  const [expanded, setExpanded] = useState<string | null>(null);

  const queryKey = useMemo(
    () => ['audit-logs', { page, action, entityType, from, to }],
    [page, action, entityType, from, to],
  );

  const { data, isLoading } = useQuery({
    queryKey,
    queryFn: () =>
      apiClient.get<AuditResponse>('/admin/audit-logs', {
        page: String(page),
        limit: '50',
        ...(action ? { action } : {}),
        ...(entityType ? { entityType } : {}),
        ...(from ? { from } : {}),
        ...(to ? { to } : {}),
      }),
  });

  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.limit)) : 1;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Link
          to="/settings"
          className="rounded-lg p-2 text-muted-foreground hover:bg-secondary hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="page-title">Audit logs</h1>
          <p className="page-description">
            Admin actions recorded with actor, IP, user agent, and before/after values.
          </p>
        </div>
      </div>

      <div className="rounded-xl border border-border bg-card p-6 shadow-card">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <FileClock className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h3 className="font-semibold text-foreground">Filters</h3>
            <p className="text-sm text-muted-foreground">
              Narrow by action prefix, entity type, or date range.
            </p>
          </div>
        </div>
        <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="Action (prefix)">
            <input
              value={action}
              onChange={(e) => {
                setAction(e.target.value);
                setPage(1);
              }}
              placeholder="e.g. chef.verify"
              className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
            />
          </Field>
          <Field label="Entity type">
            <input
              value={entityType}
              onChange={(e) => {
                setEntityType(e.target.value);
                setPage(1);
              }}
              placeholder="e.g. user, chef"
              className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
            />
          </Field>
          <Field label="From">
            <input
              type="date"
              value={from}
              onChange={(e) => {
                setFrom(e.target.value);
                setPage(1);
              }}
              className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
            />
          </Field>
          <Field label="To">
            <input
              type="date"
              value={to}
              onChange={(e) => {
                setTo(e.target.value);
                setPage(1);
              }}
              className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-herb focus:outline-none focus:ring-2 focus:ring-herb/40"
            />
          </Field>
        </div>
      </div>

      <div className="rounded-xl border border-border bg-card shadow-card">
        {isLoading ? (
          <p className="p-6 text-sm text-muted-foreground">Loading...</p>
        ) : (data?.logs ?? []).length === 0 ? (
          <p className="p-6 text-sm text-muted-foreground">
            No audit events match these filters.
          </p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-left text-xs uppercase tracking-wide text-muted-foreground">
                <th className="py-2 px-4 font-medium">When</th>
                <th className="py-2 px-4 font-medium">Actor</th>
                <th className="py-2 px-4 font-medium">Action</th>
                <th className="py-2 px-4 font-medium">Entity</th>
                <th className="py-2 px-4 font-medium">IP</th>
              </tr>
            </thead>
            <tbody>
              {(data?.logs ?? []).map((log) => {
                const open = expanded === log.id;
                const actor = log.user
                  ? `${log.user.firstName ?? ''} ${log.user.lastName ?? ''}`.trim() ||
                    log.user.email ||
                    (log.userId ?? '—')
                  : log.userId ?? '—';
                return (
                  <Fragment key={log.id}>
                    <tr
                      onClick={() => setExpanded(open ? null : log.id)}
                      className="cursor-pointer border-b border-border hover:bg-secondary/30"
                    >
                      <td className="py-2 px-4 text-xs text-muted-foreground">
                        {new Date(log.createdAt).toLocaleString()}
                      </td>
                      <td className="py-2 px-4 text-xs text-foreground">{actor}</td>
                      <td className="py-2 px-4">
                        <code className="rounded bg-muted px-1.5 py-0.5 text-xs">{log.action}</code>
                      </td>
                      <td className="py-2 px-4 text-xs text-muted-foreground">
                        {log.entityType}
                        {log.entityId ? ` · ${log.entityId.slice(0, 8)}` : ''}
                      </td>
                      <td className="py-2 px-4 text-xs text-muted-foreground">
                        {log.ipAddress || '—'}
                      </td>
                    </tr>
                    {open && (log.oldValue || log.newValue || log.userAgent) ? (
                      <tr className="border-b border-border bg-secondary/20">
                        <td colSpan={5} className="py-3 px-4">
                          {log.userAgent && (
                            <p className="mb-2 text-xs text-muted-foreground">
                              <span className="font-medium">User agent:</span> {log.userAgent}
                            </p>
                          )}
                          <div className="grid gap-3 md:grid-cols-2">
                            {log.oldValue && (
                              <div>
                                <p className="mb-1 text-xs font-medium text-muted-foreground">Before</p>
                                <pre className="max-h-40 overflow-auto rounded bg-muted px-2 py-1.5 text-xs">
                                  {formatJSON(log.oldValue)}
                                </pre>
                              </div>
                            )}
                            {log.newValue && (
                              <div>
                                <p className="mb-1 text-xs font-medium text-muted-foreground">After</p>
                                <pre className="max-h-40 overflow-auto rounded bg-muted px-2 py-1.5 text-xs">
                                  {formatJSON(log.newValue)}
                                </pre>
                              </div>
                            )}
                          </div>
                        </td>
                      </tr>
                    ) : null}
                  </Fragment>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {data && data.total > data.limit && (
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <span>
            Page {data.page} of {totalPages} · {data.total} total
          </span>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            >
              Next
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="block text-xs font-medium text-muted-foreground mb-1">{label}</label>
      {children}
    </div>
  );
}

function formatJSON(raw: string): string {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}
