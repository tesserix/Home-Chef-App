import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { ArrowLeft, Database, Download, FileClock, Loader2, Users as UsersIcon, ShoppingCart, BarChart3 } from 'lucide-react';

// Mirrors the admin api-client URL building so downloads hit the same base.
const BFF_URL = (() => {
  const env = import.meta.env.VITE_BFF_URL;
  if (env) return env as string;
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `${window.location.origin}/bff`;
  }
  return '/bff';
})();

// Download the CSV via fetch (carries Bearer token for JWT sessions and
// cookies for BFF sessions), then trigger a browser save via Blob URL.
// A plain <a href> would 401 for JWT-logged-in admins since it can't attach
// an Authorization header.
async function fetchAndDownload(path: string, filename: string) {
  const { useAuthStore } = await import('@/app/store/auth-store');
  const { accessToken } = useAuthStore.getState();
  const headers: Record<string, string> = {};
  if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`;

  const res = await fetch(`${BFF_URL}${path}`, {
    credentials: 'include',
    headers,
  });
  if (!res.ok) {
    throw new Error(`Download failed (${res.status})`);
  }
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  try {
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
  } finally {
    URL.revokeObjectURL(url);
  }
}

type ExportKind = 'users' | 'orders' | 'revenue';

export default function DataExportsPage() {
  const today = useMemo(() => new Date().toISOString().slice(0, 10), []);
  const thirtyDaysAgo = useMemo(() => {
    const d = new Date();
    d.setDate(d.getDate() - 30);
    return d.toISOString().slice(0, 10);
  }, []);

  const [from, setFrom] = useState(thirtyDaysAgo);
  const [to, setTo] = useState(today);
  const [downloading, setDownloading] = useState<ExportKind | null>(null);
  const [error, setError] = useState<string | null>(null);

  const buildPath = (kind: ExportKind, withRange: boolean) => {
    const params = withRange ? `?from=${from}&to=${to}` : '';
    return `/api/v1/admin/exports/${kind}.csv${params}`;
  };

  const download = async (kind: ExportKind, withRange: boolean, filename: string) => {
    setDownloading(kind);
    setError(null);
    try {
      await fetchAndDownload(buildPath(kind, withRange), `${filename}.csv`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Download failed');
    } finally {
      setDownloading(null);
    }
  };

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
          <h1 className="page-title">Data &amp; Exports</h1>
          <p className="page-description">
            Download platform data as CSV. Files stream directly from the API.
          </p>
        </div>
      </div>

      <div className="rounded-xl border border-border bg-card p-6 shadow-card">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <Database className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h3 className="font-semibold text-foreground">Date range</h3>
            <p className="text-sm text-muted-foreground">
              Applies to orders and revenue exports.
            </p>
          </div>
        </div>
        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          <DateField label="From" value={from} onChange={setFrom} />
          <DateField label="To" value={to} onChange={setTo} />
        </div>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700">
          {error}
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        <ExportCard
          icon={<UsersIcon className="h-5 w-5 text-primary" />}
          title="User data"
          description="All user accounts (id, email, role, created_at)"
          loading={downloading === 'users'}
          onDownload={() => download('users', false, 'homechef-users')}
        />
        <ExportCard
          icon={<ShoppingCart className="h-5 w-5 text-primary" />}
          title="Orders"
          description="Order rows filtered by the date range above"
          loading={downloading === 'orders'}
          onDownload={() => download('orders', true, 'homechef-orders')}
        />
        <ExportCard
          icon={<BarChart3 className="h-5 w-5 text-primary" />}
          title="Revenue"
          description="Per-day revenue rollup — paid orders only"
          loading={downloading === 'revenue'}
          onDownload={() => download('revenue', true, 'homechef-revenue')}
        />
      </div>

      <Link
        to="/settings/audit-logs"
        className="flex items-center justify-between gap-3 rounded-xl border border-border bg-card p-6 shadow-card transition-colors hover:border-primary/40"
      >
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <FileClock className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h3 className="font-semibold text-foreground">Audit logs</h3>
            <p className="text-sm text-muted-foreground">
              Who did what, when, and from where across the admin surface.
            </p>
          </div>
        </div>
        <span className="text-xs text-primary">View →</span>
      </Link>
    </div>
  );
}

function DateField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <div>
      <label className="block text-xs font-medium text-muted-foreground mb-1">{label}</label>
      <input
        type="date"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-lg border border-border bg-secondary/30 px-3 py-2 text-sm focus:border-primary focus:outline-none"
      />
    </div>
  );
}

function ExportCard({
  icon,
  title,
  description,
  loading,
  onDownload,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  loading: boolean;
  onDownload: () => void;
}) {
  return (
    <div className="rounded-xl border border-border bg-card p-6 shadow-card">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
          {icon}
        </div>
        <div>
          <h3 className="font-semibold text-foreground">{title}</h3>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
      </div>
      <button
        onClick={onDownload}
        disabled={loading}
        className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
      >
        {loading ? (
          <>
            <Loader2 className="h-4 w-4 animate-spin" />
            Preparing...
          </>
        ) : (
          <>
            <Download className="h-4 w-4" />
            Download CSV
          </>
        )}
      </button>
    </div>
  );
}
