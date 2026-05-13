import { OttoWidget } from '@tesserix/otto-widget';

import '@tesserix/otto-widget/styles/otto.css';

import { useAuth } from '@/app/providers/AuthProvider';

// HomeChef support chat — thin wrapper around the canonical Otto widget
// published as @tesserix/otto-widget from slm-support-platform. Pulls the
// signed-in customer identity from HomeChef's AuthProvider so the widget
// can prefill name/email and skip the OTP step for logged-in users.
//
// REST traffic goes to /api/otto, which apps/web/nginx.conf proxies to
// support-platform-otto.support-platform.svc.cluster.local:8089 in prod.
// The WebSocket path leaves nginx and goes straight to otto through the
// Istio gateway (matching the mark8ly storefront pattern).
//
// This replaces a previous bespoke 200-line polling component. The
// canonical widget brings WebSocket transport, intake reasons,
// OTP for anonymous users, and reconnect logic. Brand/design-token
// alignment (herb / paper / Inter) is a follow-up on the widget itself —
// styling there belongs in @tesserix/otto-widget so every product picks
// it up rather than diverging again.
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
