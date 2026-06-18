import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation } from '@tanstack/react-query';
import { Users } from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui';

// Group-order invite landing (#46) — web. Opened from the shared link
// (/group/:token). Previews the invite, then joins → the shared-cart hub.

interface GroupInvitePreview {
  title?: string;
  type: 'office' | 'personal';
  chefName?: string;
  status: string;
  joinable: boolean;
  hostName?: string;
}

export default function GroupInvitePage() {
  const { token } = useParams<{ token: string }>();
  const navigate = useNavigate();

  const { data: preview, isLoading, isError } = useQuery({
    queryKey: ['group-invite', token],
    queryFn: () => apiClient.get<GroupInvitePreview>(`/group-invites/${token}`),
    enabled: !!token,
  });

  const join = useMutation({
    mutationFn: () =>
      apiClient.post<{ groupOrderId?: string }>(`/group-invites/${token}/accept`, {}),
    onSuccess: (d) => {
      if (d.groupOrderId) navigate(`/group-orders/${d.groupOrderId}`);
    },
  });

  return (
    <div className="mx-auto flex min-h-[60vh] max-w-md flex-col items-center justify-center px-4 text-center">
      {isLoading ? (
        <p className="text-ink-muted">Loading…</p>
      ) : isError || !preview ? (
        <>
          <h1 className="text-xl font-semibold text-ink">Invite not found</h1>
          <p className="mt-1 text-ink-soft">This group order link is invalid or has expired.</p>
          <Button variant="outline" className="mt-4" onClick={() => navigate('/')}>Go home</Button>
        </>
      ) : (
        <>
          <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-herb-tint">
            <Users className="h-8 w-8 text-herb" aria-hidden="true" />
          </div>
          <h1 className="text-xl font-semibold text-ink">
            {preview.hostName ? `${preview.hostName} invited you` : "You're invited"}
          </h1>
          <p className="mt-2 text-ink-soft">
            Join {preview.type === 'office' ? 'this office order' : 'this group order'}
            {preview.title ? ` "${preview.title}"` : ''} from{' '}
            <span className="font-medium text-ink">{preview.chefName ?? 'a chef'}</span>. Add your
            own items and pay your share.
          </p>
          {preview.joinable ? (
            <Button variant="primary" className="mt-5" isLoading={join.isPending} onClick={() => join.mutate()}>
              Join the order
            </Button>
          ) : (
            <p className="mt-4 text-sm text-paprika">This group order is no longer open to join.</p>
          )}
          <Button variant="ghost" className="mt-2" onClick={() => navigate('/')}>Not now</Button>
        </>
      )}
    </div>
  );
}
