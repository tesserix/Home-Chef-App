import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  Bell,
  CheckCircle,
  XCircle,
  AlertCircle,
  Info,
  ChefHat,
  ShoppingBag,
  FileText,
  Loader2,
  CheckCheck,
  Clock,
  Send,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';

interface Notification {
  id: string;
  type: string;
  title: string;
  message: string;
  data?: string;
  isRead: boolean;
  readAt?: string;
  createdAt: string;
}

interface ApprovalRequest {
  id: string;
  type: string;
  status: string;
  priority: string;
  title: string;
  description: string;
  adminNotes: string;
  reviewedAt?: string;
  createdAt: string;
}

interface NotificationsResponse {
  data: Notification[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
  };
}

const statusLabels: Record<string, string> = {
  pending: 'Pending Review',
  approved: 'Approved',
  rejected: 'Rejected',
  info_requested: 'Info Requested',
  cancelled: 'Cancelled',
};

const statusIcons: Record<string, typeof Bell> = {
  pending: Clock,
  approved: CheckCircle,
  rejected: XCircle,
  info_requested: AlertCircle,
  cancelled: XCircle,
};

const statusStyles: Record<string, string> = {
  pending: 'bg-warning/10 text-warning',
  approved: 'bg-success/10 text-success',
  rejected: 'bg-destructive/10 text-destructive',
  info_requested: 'bg-info/10 text-info',
  cancelled: 'bg-muted text-muted-foreground',
};

const typeIcons: Record<string, typeof Bell> = {
  approval_approved: CheckCircle,
  approval_rejected: XCircle,
  approval_info_requested: AlertCircle,
  chef_verified: CheckCircle,
  new_order: ShoppingBag,
  order_created: ShoppingBag,
  order_status: ShoppingBag,
  order_cancelled: XCircle,
  order_delivered: CheckCircle,
  system: Info,
  document_verified: FileText,
};

const typeStyles: Record<string, string> = {
  approval_approved: 'bg-success/10 text-success',
  approval_rejected: 'bg-destructive/10 text-destructive',
  approval_info_requested: 'bg-warning/10 text-warning',
  chef_verified: 'bg-success/10 text-success',
  new_order: 'bg-primary/10 text-primary',
  order_created: 'bg-primary/10 text-primary',
  order_status: 'bg-info/10 text-info',
  order_cancelled: 'bg-destructive/10 text-destructive',
  order_delivered: 'bg-success/10 text-success',
  system: 'bg-muted text-muted-foreground',
};

export default function NotificationsPage() {
  const queryClient = useQueryClient();

  // Fetch approval requests directly for this chef's kitchen
  // This works regardless of which user account the notifications were sent to
  const { data: approvalData, isLoading: approvalsLoading } = useQuery({
    queryKey: ['chef-admin-requests'],
    queryFn: () => apiClient.get<{ data: ApprovalRequest[] }>('/chef/admin-requests'),
  });

  // Also fetch notifications (for non-approval notifications)
  const { data, isLoading: notifsLoading } = useQuery({
    queryKey: ['notifications'],
    queryFn: () => apiClient.get<NotificationsResponse>('/notifications', { limit: 50 }),
  });

  const isLoading = approvalsLoading && notifsLoading;

  const markReadMutation = useMutation({
    mutationFn: (id: string) => apiClient.put(`/notifications/${id}/read`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      queryClient.invalidateQueries({ queryKey: ['notification-count'] });
    },
  });

  const markAllReadMutation = useMutation({
    mutationFn: () => apiClient.put('/notifications/read-all'),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
      queryClient.invalidateQueries({ queryKey: ['notification-count'] });
    },
  });

  // Parse approval requests (primary data source for admin requests)
  const rawApproval = approvalData as unknown;
  const approvals: ApprovalRequest[] = Array.isArray(rawApproval)
    ? rawApproval
    : (rawApproval as { data: ApprovalRequest[] })?.data ?? [];

  // Parse notifications (secondary - for other notifications)
  const rawData = data as unknown;
  const notifications: Notification[] = Array.isArray(rawData)
    ? rawData
    : (rawData as NotificationsResponse)?.data ?? [];
  const hasUnread = notifications.some((n) => !n.isRead);
  const hasContent = approvals.length > 0 || notifications.length > 0;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="page-title">Admin Requests</h1>
          <p className="page-description">
            Requests and updates from the admin team
          </p>
        </div>
        {hasUnread && (
          <button
            onClick={() => markAllReadMutation.mutate()}
            className="inline-flex items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-secondary transition-colors"
          >
            <CheckCheck className="h-4 w-4" />
            Mark all read
          </button>
        )}
      </div>

      {/* Approval Requests Section */}
      {approvals.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-lg font-semibold text-foreground">Kitchen Review Status</h2>
          {approvals.map((req) => {
            const StatusIcon = statusIcons[req.status] || Clock;
            const style = statusStyles[req.status] || statusStyles.pending;
            return (
              <div key={req.id} className="rounded-xl border border-border bg-card p-4 shadow-card">
                <div className="flex items-start gap-4">
                  <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${style}`}>
                    <StatusIcon className="h-5 w-5" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <h3 className="text-sm font-semibold text-foreground">{req.title}</h3>
                        <div className="flex items-center gap-2 mt-1">
                          <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${style}`}>
                            {statusLabels[req.status] || req.status}
                          </span>
                          <span className="text-xs text-muted-foreground">{formatTimeAgo(req.createdAt)}</span>
                        </div>
                      </div>
                    </div>
                    <p className="mt-1 text-sm text-muted-foreground">{req.description}</p>

                    {req.status === 'pending' && (
                      <p className="mt-1.5 text-xs text-muted-foreground/80 italic">
                        Typically reviewed within 24-48 hours
                      </p>
                    )}

                    {req.adminNotes && (
                      <div className="mt-3 rounded-lg border border-warning/20 bg-warning/5 p-3">
                        <div className="flex items-center gap-1.5 mb-1">
                          <AlertCircle className="h-3.5 w-3.5 text-warning" />
                          <span className="text-xs font-medium text-warning">Admin Notes</span>
                        </div>
                        <p className="text-sm text-foreground">{req.adminNotes}</p>
                      </div>
                    )}

                    {req.status !== 'approved' && req.status !== 'cancelled' && (
                      <div className="mt-3 space-y-3">
                        {(req.status === 'info_requested' || req.status === 'rejected') && (
                          <div className="flex items-center gap-2">
                            <a href="/profile"
                              className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors">
                              <ChefHat className="h-3.5 w-3.5" />
                              Update Profile
                            </a>
                          </div>
                        )}
                        <RespondForm requestId={req.id} />
                      </div>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Notifications Section */}
      {notifications.length > 0 && (
        <h2 className="text-lg font-semibold text-foreground">Notifications</h2>
      )}

      {!hasContent ? (
        <div className="rounded-xl border border-border bg-card p-20 text-center shadow-card">
          <Bell className="mx-auto h-12 w-12 text-muted-foreground/30" />
          <p className="mt-4 text-lg font-medium text-foreground">No admin requests yet</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Admin requests for document uploads, profile updates, and review feedback will appear here.
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {notifications.map((notif) => {
            const Icon = typeIcons[notif.type] || Bell;
            const style = typeStyles[notif.type] || 'bg-muted text-muted-foreground';
            let extraData: Record<string, unknown> = {};
            try {
              if (notif.data) extraData = JSON.parse(notif.data);
            } catch { /* ignore */ }

            return (
              <div
                key={notif.id}
                className={`rounded-xl border bg-card p-4 shadow-card transition-colors ${
                  notif.isRead ? 'border-border opacity-70' : 'border-primary/20 bg-primary/[0.02]'
                }`}
              >
                <div className="flex items-start gap-4">
                  <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${style}`}>
                    <Icon className="h-5 w-5" />
                  </div>

                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <h3 className={`text-sm font-semibold ${notif.isRead ? 'text-foreground' : 'text-foreground'}`}>
                          {notif.title}
                        </h3>
                        <p className="mt-0.5 text-sm text-muted-foreground">{notif.message}</p>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <span className="text-xs text-muted-foreground whitespace-nowrap">
                          {formatTimeAgo(notif.createdAt)}
                        </span>
                        {!notif.isRead && (
                          <span className="h-2 w-2 rounded-full bg-primary shrink-0" />
                        )}
                      </div>
                    </div>

                    {/* Admin notes from approval requests */}
                    {(extraData.notes as string) && (
                      <div className="mt-3 rounded-lg border border-warning/20 bg-warning/5 p-3">
                        <div className="flex items-center gap-1.5 mb-1">
                          <AlertCircle className="h-3.5 w-3.5 text-warning" />
                          <span className="text-xs font-medium text-warning">Admin Notes</span>
                        </div>
                        <p className="text-sm text-foreground">{String(extraData.notes)}</p>
                      </div>
                    )}

                    {/* Action buttons */}
                    <div className="mt-3 flex items-center gap-2">
                      {!notif.isRead && (
                        <button
                          onClick={() => markReadMutation.mutate(notif.id)}
                          className="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-secondary transition-colors"
                        >
                          <CheckCircle className="h-3.5 w-3.5" />
                          Mark as read
                        </button>
                      )}

                      {notif.type === 'approval_info_requested' && (
                        <a
                          href="/profile"
                          className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
                        >
                          <ChefHat className="h-3.5 w-3.5" />
                          Update Details
                        </a>
                      )}

                      {notif.type === 'approval_approved' && (
                        <a
                          href="/dashboard"
                          className="inline-flex items-center gap-1.5 rounded-lg bg-success px-3 py-1.5 text-xs font-medium text-success-foreground hover:bg-success/90 transition-colors"
                        >
                          <CheckCircle className="h-3.5 w-3.5" />
                          Go to Dashboard
                        </a>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function RespondForm({ requestId }: { requestId: string }) {
  const [response, setResponse] = useState('');
  const [sending, setSending] = useState(false);
  const queryClient = useQueryClient();

  const handleSend = async () => {
    if (!response.trim()) {
      toast.error('Please enter a response');
      return;
    }
    setSending(true);
    try {
      await apiClient.put(`/chef/admin-requests/${requestId}/respond`, { response: response.trim() });
      toast.success('Response sent to admin');
      setResponse('');
      queryClient.invalidateQueries({ queryKey: ['chef-admin-requests'] });
    } catch {
      toast.error('Failed to send response');
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="rounded-lg border border-border bg-muted/30 p-3">
      <textarea
        value={response}
        onChange={(e) => setResponse(e.target.value)}
        placeholder="Type your response to the admin... (e.g., I've uploaded the FSSAI license, please review)"
        rows={2}
        className="w-full rounded-lg border border-input bg-card px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring resize-none"
      />
      <div className="mt-2 flex justify-end">
        <button
          onClick={handleSend}
          disabled={sending || !response.trim()}
          className="inline-flex items-center gap-1.5 rounded-lg bg-success px-4 py-1.5 text-xs font-medium text-success-foreground hover:bg-success/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {sending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Send className="h-3.5 w-3.5" />}
          Respond & Send
        </button>
      </div>
    </div>
  );
}

function formatTimeAgo(dateStr: string): string {
  const diff = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
  if (diff < 60) return 'just now';
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 604800) return `${Math.floor(diff / 86400)}d ago`;
  return new Date(dateStr).toLocaleDateString('en-IN', { day: 'numeric', month: 'short' });
}
