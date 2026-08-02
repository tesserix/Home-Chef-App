import { OttoWidget, type ReasonOption } from '@tesserix/otto-widget';

import '@tesserix/otto-widget/styles/otto.css';

import { useAuth } from '@/app/providers/AuthProvider';

// Chef-facing support chat — the vendor twin of apps/web's OttoChat.
//
// tenantId is `homechef-vendor`, NOT `homechef`. The two are separate otto
// tenants with their own RAG namespace and their own system prompt: they were
// one aliased tenant until 2026-08-02, and sharing it meant chefs were given
// customer-app navigation ("open the Orders tab") for kitchen questions. Do not
// collapse them back.
//
// /api/otto/* is rewritten to otto's storefront surface by nginx, which injects
// the tenant + internal-auth headers server-side so neither reaches the browser
// bundle.

// Kitchen-side concerns. Deliberately different from the customer list: a chef
// asks about payouts and their menu, never about tracking their own delivery.
const VENDOR_REASONS: readonly ReasonOption[] = [
  { value: 'general_question', label: 'Ask a quick question', requiresStatus: false },
  { value: 'payout_issue', label: 'Payouts / earnings' },
  { value: 'order_issue', label: 'Problem with an order' },
  { value: 'menu_help', label: 'Menu, dishes or capacity' },
  { value: 'documents', label: 'Documents / verification' },
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
      productName="Fe3dr Chef Support"
      tenantId="homechef-vendor"
      reasons={VENDOR_REASONS}
      statusPlaceholder="e.g. payout for last week hasn't landed"
      customerId={user?.id ?? undefined}
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
