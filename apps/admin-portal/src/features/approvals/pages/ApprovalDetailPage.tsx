import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  ArrowLeft,
  Mail,
  Phone,
  ChefHat,
  FileText,
  UtensilsCrossed,
  IndianRupee,
  Loader2,
  CheckCircle,
  XCircle,
  AlertCircle,
  Download,
  Clock,
  User,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';

interface ApprovalRequest {
  id: string;
  type: string;
  status: string;
  priority: string;
  title: string;
  description: string;
  chefId?: string;
  partnerId?: string;
  entityType: string;
  entityId: string;
  adminNotes?: string;
  reviewedAt?: string;
  createdAt: string;
  submittedData?: Record<string, unknown>;
  documents?: ApprovalDocument[];
  chef?: {
    businessName: string;
    phone?: string;
    user?: { firstName: string; lastName: string; email: string };
  };
  partner?: {
    vehicleType: string;
    vehicleNumber: string;
    city: string;
    verificationStatus: string;
    user?: { firstName: string; lastName: string; email: string; phone: string };
  };
  submittedBy?: { firstName: string; lastName: string; email: string };
}

interface ApprovalDocument {
  id: string;
  fileName: string;
  type: string;
  fileSize: number;
  fileUrl?: string;
  contentType?: string;
  status?: string;
}

interface HistoryEntry {
  id: string;
  action: string;
  fromStatus: string;
  toStatus: string;
  notes?: string;
  adminName?: string;
  createdAt: string;
}

interface HistoryResponse {
  data: HistoryEntry[];
}

const typeConfig: Record<string, { icon: typeof ChefHat; label: string; style: string }> = {
  kitchen_onboarding: { icon: ChefHat, label: 'Kitchen Onboarding', style: 'bg-primary/10 text-primary' },
  document_verification: { icon: FileText, label: 'Document Verification', style: 'bg-info/10 text-info' },
  menu_item_new: { icon: UtensilsCrossed, label: 'New Menu Item', style: 'bg-success/10 text-success' },
  pricing_change: { icon: IndianRupee, label: 'Pricing Change', style: 'bg-warning/10 text-warning' },
  driver_onboarding: { icon: FileText, label: 'Driver Onboarding', style: 'bg-info/10 text-info' },
  driver_document: { icon: FileText, label: 'Driver Document', style: 'bg-info/10 text-info' },
};

const priorityStyles: Record<string, string> = {
  urgent: 'bg-destructive/10 text-destructive',
  high: 'bg-warning/10 text-warning',
  normal: 'bg-muted text-muted-foreground',
  low: 'bg-secondary text-muted-foreground',
};

const statusStyles: Record<string, string> = {
  pending: 'bg-warning/10 text-warning',
  approved: 'bg-success/10 text-success',
  rejected: 'bg-destructive/10 text-destructive',
  info_requested: 'bg-info/10 text-info',
};

const statusLabels: Record<string, string> = {
  pending: 'Pending',
  approved: 'Approved',
  rejected: 'Rejected',
  info_requested: 'Info Requested',
};

export default function ApprovalDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [adminNotes, setAdminNotes] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['admin-approval', id],
    queryFn: () => apiClient.get<ApprovalRequest>(`/admin/approvals/${id}`),
    enabled: !!id,
  });

  const { data: historyData } = useQuery({
    queryKey: ['admin-approval-history', id],
    queryFn: () => apiClient.get<HistoryResponse>(`/admin/approvals/${id}/history`),
    enabled: !!id,
  });

  const approval = data as unknown as ApprovalRequest | undefined;
  // History API returns array directly, not {data: [...]}
  const history = (Array.isArray(historyData) ? historyData : (historyData as unknown as HistoryResponse)?.data ?? []) as HistoryEntry[];

  const approveMutation = useMutation({
    mutationFn: () => apiClient.put(`/admin/approvals/${id}/approve`, { notes: adminNotes || undefined }),
    onSuccess: () => {
      toast.success('Approval request approved');
      queryClient.invalidateQueries({ queryKey: ['admin-approval', id] });
      queryClient.invalidateQueries({ queryKey: ['admin-approval-history', id] });
      queryClient.invalidateQueries({ queryKey: ['admin-approvals'] });
      queryClient.invalidateQueries({ queryKey: ['admin-approval-counts'] });
      setAdminNotes('');
    },
    onError: () => toast.error('Failed to approve request'),
  });

  const rejectMutation = useMutation({
    mutationFn: () => apiClient.put(`/admin/approvals/${id}/reject`, { notes: adminNotes || undefined }),
    onSuccess: () => {
      toast.success('Approval request rejected');
      queryClient.invalidateQueries({ queryKey: ['admin-approval', id] });
      queryClient.invalidateQueries({ queryKey: ['admin-approval-history', id] });
      queryClient.invalidateQueries({ queryKey: ['admin-approvals'] });
      queryClient.invalidateQueries({ queryKey: ['admin-approval-counts'] });
      setAdminNotes('');
    },
    onError: () => toast.error('Failed to reject request'),
  });

  const requestInfoMutation = useMutation({
    mutationFn: () => apiClient.put(`/admin/approvals/${id}/request-info`, { notes: adminNotes || undefined }),
    onSuccess: () => {
      toast.success('More information requested from chef');
      queryClient.invalidateQueries({ queryKey: ['admin-approval', id] });
      queryClient.invalidateQueries({ queryKey: ['admin-approval-history', id] });
      queryClient.invalidateQueries({ queryKey: ['admin-approvals'] });
      queryClient.invalidateQueries({ queryKey: ['admin-approval-counts'] });
      setAdminNotes('');
    },
    onError: () => toast.error('Failed to request information'),
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (!approval) {
    return (
      <div className="py-20 text-center">
        <p className="text-muted-foreground">Approval request not found</p>
        <button onClick={() => navigate('/approvals')} className="mt-4 text-sm text-primary hover:underline">
          Back to Reviews
        </button>
      </div>
    );
  }

  const typeInfo = typeConfig[approval.type];
  const TypeIcon = typeInfo?.icon || FileText;
  const isPending = approval.status === 'pending' || approval.status === 'info_requested';
  const isMutating = approveMutation.isPending || rejectMutation.isPending || requestInfoMutation.isPending;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button onClick={() => navigate('/approvals')}
          className="rounded-lg p-2 hover:bg-secondary transition-colors">
          <ArrowLeft className="h-5 w-5 text-muted-foreground" />
        </button>
        <div className="flex-1">
          <div className="flex items-center gap-3 flex-wrap">
            <h1 className="text-2xl font-bold text-foreground">{approval.title}</h1>
            <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${typeInfo?.style || 'bg-muted text-muted-foreground'}`}>
              <TypeIcon className="h-3 w-3" />
              {typeInfo?.label || approval.type}
            </span>
            <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${statusStyles[approval.status] || 'bg-muted text-muted-foreground'}`}>
              {statusLabels[approval.status] || approval.status}
            </span>
            <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${priorityStyles[approval.priority] || priorityStyles.normal}`}>
              {approval.priority}
            </span>
          </div>
          <p className="text-sm text-muted-foreground mt-1">{approval.description}</p>
        </div>
        {isPending && (
          <div className="flex items-center gap-2">
            <button
              onClick={() => approveMutation.mutate()}
              disabled={isMutating}
              className="inline-flex items-center gap-2 rounded-lg border border-success/30 px-4 py-2 text-sm font-medium text-success hover:bg-success/10 transition-colors disabled:opacity-50"
            >
              <CheckCircle className="h-4 w-4" />Approve
            </button>
            <button
              onClick={() => rejectMutation.mutate()}
              disabled={isMutating}
              className="inline-flex items-center gap-2 rounded-lg border border-destructive/30 px-4 py-2 text-sm font-medium text-destructive hover:bg-destructive/10 transition-colors disabled:opacity-50"
            >
              <XCircle className="h-4 w-4" />Reject
            </button>
            <button
              onClick={() => requestInfoMutation.mutate()}
              disabled={isMutating}
              className="inline-flex items-center gap-2 rounded-lg border border-warning/30 px-4 py-2 text-sm font-medium text-warning hover:bg-warning/10 transition-colors disabled:opacity-50"
            >
              <AlertCircle className="h-4 w-4" />Request Info
            </button>
          </div>
        )}
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Chef Info Card */}
          {approval.chef && (
            <div className="rounded-xl border border-border bg-card p-6 shadow-card">
              <h2 className="text-lg font-semibold text-foreground mb-4">Chef Information</h2>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <InfoRow icon={ChefHat} label="Business Name" value={approval.chef.businessName} />
                {approval.chef.user && (
                  <>
                    <InfoRow icon={User} label="Name" value={`${approval.chef.user.firstName} ${approval.chef.user.lastName}`} />
                    <InfoRow icon={Mail} label="Email" value={approval.chef.user.email} />
                  </>
                )}
                {approval.chef.phone && (
                  <InfoRow icon={Phone} label="Phone" value={approval.chef.phone} />
                )}
              </div>
            </div>
          )}

          {/* Submitted Data */}
          {approval.submittedData && (
            <div className="rounded-xl border border-border bg-card p-6 shadow-card">
              <h2 className="text-lg font-semibold text-foreground mb-4">Submitted Data</h2>
              <SubmittedDataView type={approval.type} data={
                typeof approval.submittedData === 'string'
                  ? (() => { try { return JSON.parse(approval.submittedData as unknown as string); } catch { return {}; } })()
                  : approval.submittedData
              } />
            </div>
          )}

          {/* Documents */}
          {approval.documents && approval.documents.length > 0 && (
            <div className="rounded-xl border border-border bg-card p-6 shadow-card">
              <h2 className="text-lg font-semibold text-foreground mb-4">Documents</h2>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                {approval.documents.map((doc) => (
                  <div key={doc.id} className="flex items-center gap-3 rounded-lg border border-border bg-secondary/20 p-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info/10">
                      <FileText className="h-5 w-5 text-info" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-foreground truncate">{doc.fileName || doc.type}</p>
                      <p className="text-xs text-muted-foreground">{doc.type}{doc.status ? ` - ${doc.status}` : ''}{doc.fileSize ? ` - ${formatFileSize(doc.fileSize)}` : ''}</p>
                    </div>
                    <a
                      href={doc.fileUrl || `/bff/api/v1/admin/approvals/${id}/documents/${doc.id}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="rounded-lg p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                      title="Download"
                    >
                      <Download className="h-4 w-4" />
                    </a>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Admin Notes */}
          {isPending && (
            <div className="rounded-xl border border-border bg-card p-6 shadow-card">
              <h2 className="text-lg font-semibold text-foreground mb-4">Admin Notes</h2>
              <textarea
                value={adminNotes}
                onChange={(e) => setAdminNotes(e.target.value)}
                placeholder="Add notes for this approval action (optional)..."
                rows={4}
                className="w-full rounded-lg border border-input bg-background px-4 py-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring resize-none"
              />
              <p className="mt-2 text-xs text-muted-foreground">
                These notes will be recorded in the history and visible to other admins.
              </p>
            </div>
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Details Card */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">Details</h2>
            <div className="space-y-3">
              <div className="flex items-center justify-between rounded-lg bg-muted/50 px-4 py-3">
                <span className="text-sm text-muted-foreground">Request ID</span>
                <code className="text-xs font-mono text-foreground">{approval.id?.slice(0, 8) || approval.id}...</code>
              </div>
              <div className="flex items-center justify-between rounded-lg bg-muted/50 px-4 py-3">
                <span className="text-sm text-muted-foreground">Submitted</span>
                <span className="text-sm text-foreground">
                  {new Date(approval.createdAt).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })}
                </span>
              </div>
              {approval.reviewedAt && (
                <div className="flex items-center justify-between rounded-lg bg-muted/50 px-4 py-3">
                  <span className="text-sm text-muted-foreground">Reviewed</span>
                  <span className="text-sm text-foreground">
                    {new Date(approval.reviewedAt).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })}
                  </span>
                </div>
              )}
              {approval.adminNotes && (
                <div className="rounded-lg bg-muted/50 px-4 py-3">
                  <span className="text-sm text-muted-foreground">Last Admin Notes</span>
                  <p className="text-sm text-foreground mt-1">{approval.adminNotes}</p>
                </div>
              )}
            </div>
          </div>

          {/* History Timeline */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-card">
            <h2 className="text-lg font-semibold text-foreground mb-4">History</h2>
            {history.length === 0 ? (
              <p className="text-sm text-muted-foreground">No history recorded yet.</p>
            ) : (
              <div className="space-y-4">
                {history.map((entry, index) => (
                  <div key={entry.id} className="relative flex gap-3">
                    {index < history.length - 1 && (
                      <div className="absolute left-[15px] top-8 h-[calc(100%-8px)] w-px bg-border" />
                    )}
                    <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10">
                      <Clock className="h-4 w-4 text-primary" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${statusStyles[entry.toStatus] || 'bg-muted text-muted-foreground'}`}>
                          {statusLabels[entry.toStatus] || entry.toStatus}
                        </span>
                        {entry.fromStatus && (
                          <span className="text-xs text-muted-foreground">
                            from {statusLabels[entry.fromStatus] || entry.fromStatus}
                          </span>
                        )}
                      </div>
                      {entry.adminName && (
                        <p className="text-xs text-muted-foreground mt-1">by {entry.adminName}</p>
                      )}
                      {entry.notes && (
                        <p className="text-xs text-foreground mt-1">{entry.notes}</p>
                      )}
                      <p className="text-xs text-muted-foreground mt-1">
                        {new Date(entry.createdAt).toLocaleString('en-IN', {
                          day: 'numeric', month: 'short', year: 'numeric',
                          hour: '2-digit', minute: '2-digit',
                        })}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function InfoRow({ icon: Icon, label, value }: { icon: typeof Mail; label: string; value: string }) {
  return (
    <div className="flex items-start gap-3 rounded-lg border border-border bg-secondary/20 p-3">
      <Icon className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
      <div className="flex-1 min-w-0">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-sm font-medium text-foreground truncate">{value}</p>
      </div>
    </div>
  );
}

function SubmittedDataView({ type, data }: { type: string; data: Record<string, unknown> }) {
  if (type === 'kitchen_onboarding') {
    const addr = data.kitchenAddress as Record<string, string> | undefined;
    const hours = data.operatingHours as Record<string, { open: string; close: string }> | undefined;
    const addressStr = addr ? [addr.line1, addr.line2, addr.city, addr.state, addr.postalCode].filter(Boolean).join(', ') : undefined;

    return (
      <div className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <DataField label="Business Name" value={data.businessName} />
          <DataField label="Full Name" value={data.fullName} />
          <DataField label="Email" value={data.email} />
          <DataField label="Phone" value={data.phone} />
          <DataField label="Description" value={data.description} />
          <DataField label="Kitchen Type" value={data.kitchenType} />
          <DataField label="Cuisines" value={Array.isArray(data.cuisines) ? (data.cuisines as string[]).join(', ') : data.cuisines} />
          <DataField label="Specialties" value={Array.isArray(data.specialties) && (data.specialties as string[]).length > 0 ? (data.specialties as string[]).join(', ') : undefined} />
          <DataField label="Prep Time" value={data.prepTime} />
          <DataField label="Years of Experience" value={data.yearsOfExperience} />
          <DataField label="Meals Per Day" value={data.mealsPerDay} />
          <DataField label="Service Radius" value={data.serviceRadius ? `${data.serviceRadius} km` : undefined} />
          <DataField label="Minimum Order" value={data.minimumOrder ? `₹${data.minimumOrder}` : undefined} />
          <DataField label="Delivery Fee" value={data.deliveryFee ? `₹${data.deliveryFee}` : undefined} />
        </div>

        {addressStr && (
          <div className="rounded-lg bg-muted/50 px-4 py-3">
            <p className="text-xs text-muted-foreground">Kitchen Address</p>
            <p className="text-sm font-medium text-foreground mt-0.5">{addressStr}</p>
          </div>
        )}

        {hours && Object.keys(hours).length > 0 && (
          <div className="rounded-lg bg-muted/50 px-4 py-3">
            <p className="text-xs text-muted-foreground mb-2">Operating Hours</p>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
              {['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'].map((day) => {
                const h = hours[day];
                return (
                  <div key={day} className="text-sm">
                    <span className="font-medium text-foreground capitalize">{day.slice(0, 3)}: </span>
                    <span className="text-muted-foreground">{h ? `${h.open} - ${h.close}` : 'Closed'}</span>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <DataField label="Terms Accepted" value={data.acceptedTerms ? 'Yes' : 'No'} />
          <DataField label="Hygiene Policy Accepted" value={data.acceptedHygienePolicy ? 'Yes' : 'No'} />
        </div>
      </div>
    );
  }

  if (type === 'document_verification') {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <DataField label="Document Type" value={data.documentType} />
        <DataField label="File Name" value={data.fileName} />
        <DataField label="Status" value={data.status} />
        <DataField label="Document Number" value={data.documentNumber} />
        {renderExtraFields(data, ['documentType', 'fileName', 'status', 'documentNumber'])}
      </div>
    );
  }

  if (type === 'menu_item_new' || type === 'menu_item_update') {
    return (
      <div className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <DataField label="Item Name" value={data.name || data.itemName} />
          <DataField label="Price" value={data.price ? `₹${data.price}` : undefined} />
          {(data.comparePrice as number) > 0 && <DataField label="Compare Price" value={`₹${data.comparePrice}`} />}
          <DataField label="Description" value={data.description} />
          <DataField label="Prep Time" value={data.prepTime ? `${data.prepTime} mins` : undefined} />
          <DataField label="Portion Size" value={data.portionSize} />
          <DataField label="Serves" value={data.serves} />
          <DataField label="Featured" value={data.isFeatured} />
          <DataField label="Dietary Tags" value={data.dietaryTags} />
          <DataField label="Allergens" value={data.allergens} />
        </div>
      </div>
    );
  }

  if (type === 'pricing_change') {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <DataField label="Item Name" value={data.itemName || data.name} />
        <DataField label="Current Price" value={data.currentPrice ? `₹${data.currentPrice}` : undefined} />
        <DataField label="New Price" value={data.newPrice ? `₹${data.newPrice}` : undefined} />
        <DataField label="Reason" value={data.reason} />
        {renderExtraFields(data, ['itemName', 'name', 'currentPrice', 'newPrice', 'reason'])}
      </div>
    );
  }

  // Fallback: render all fields
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
      {Object.entries(data).map(([key, value]) => (
        <DataField key={key} label={formatLabel(key)} value={value} />
      ))}
    </div>
  );
}

function DataField({ label, value }: { label: string; value: unknown }) {
  if (value === undefined || value === null || value === '') return null;
  let displayValue: string;
  if (Array.isArray(value)) {
    displayValue = value.length > 0 ? value.join(', ') : 'None';
  } else if (typeof value === 'boolean') {
    displayValue = value ? 'Yes' : 'No';
  } else if (typeof value === 'object') {
    // Format objects nicely instead of raw JSON
    const obj = value as Record<string, unknown>;
    const parts = Object.entries(obj)
      .filter(([, v]) => v !== undefined && v !== null && v !== '')
      .map(([k, v]) => `${formatLabel(k)}: ${v}`);
    displayValue = parts.length > 0 ? parts.join(' | ') : 'N/A';
  } else {
    displayValue = String(value);
  }
  return (
    <div className="rounded-lg bg-muted/50 px-4 py-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-sm font-medium text-foreground mt-0.5">{displayValue}</p>
    </div>
  );
}

function renderExtraFields(data: Record<string, unknown>, knownKeys: string[]) {
  const extra = Object.entries(data).filter(([key]) => !knownKeys.includes(key));
  if (extra.length === 0) return null;
  return (
    <>
      {extra.map(([key, value]) => (
        <DataField key={key} label={formatLabel(key)} value={value} />
      ))}
    </>
  );
}

function formatLabel(key: string): string {
  return key
    .replace(/([A-Z])/g, ' $1')
    .replace(/_/g, ' ')
    .replace(/^\w/, (c) => c.toUpperCase())
    .trim();
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
