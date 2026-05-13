import { OttoWidget, type ReasonOption } from '@tesserix/otto-widget';

import '@tesserix/otto-widget/styles/otto.css';

import { useAuth } from '@/app/providers/AuthProvider';

// HomeChef support chat — thin wrapper around @tesserix/otto-widget.
// Pulls signed-in identity from the existing AuthProvider so logged-in
// customers skip OTP. The `reasons` and `tenantId` props tell the
// backend which per-product SLM + MCP knowledge base to use.
//
// /api/otto/* is proxied to support-platform-otto via the Istio gateway
// (see nginx.conf for the SPA routing rules); the WebSocket leaves
// Next.js entirely and connects directly to otto at
// /api/v1/storefront/otto/conversations/:id/ws.

// Food-delivery shape — order tracking, delivery problems, chef
// questions. DOB is unnecessary because HomeChef identifies orders by
// the signed-in account, not by birthday lookup.
const HOMECHEF_REASONS: readonly ReasonOption[] = [
  { value: 'order_tracking', label: 'Order tracking / ETA' },
  { value: 'delivery_issue', label: 'Delivery problem' },
  { value: 'refund', label: 'Refund request' },
  { value: 'chef_question', label: 'Question about a chef or dish' },
  { value: 'account_issue', label: 'Account / login issue' },
  { value: 'other', label: 'Something else' },
];

export function OttoChat() {
  const { user } = useAuth();

  const displayName =
    user?.firstName || user?.lastName
      ? `${user.firstName ?? ''} ${user.lastName ?? ''}`.trim()
      : undefined;

  return (
    <OttoWidget
      apiBaseUrl="/api/otto"
      buildWsUrl={buildConversationWsUrl}
      productName="HomeChef Support"
      tenantId="homechef"
      reasons={HOMECHEF_REASONS}
      customerName={displayName}
      customerEmail={user?.email ?? undefined}
    />
  );
}

function buildConversationWsUrl(id: string): string {
  if (typeof window === 'undefined') return '';
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}/api/v1/storefront/otto/conversations/${encodeURIComponent(id)}/ws`;
}
