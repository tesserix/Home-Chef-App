import { OpenPanel } from '@openpanel/web';

// Self-hosted OpenPanel at analytics.tesserix.app. The client ID is a public
// write key baked in at build time, so an unset value simply means no-op —
// local and preview builds stay out of the production dataset.
export function initAnalytics(): void {
  const clientId = import.meta.env.VITE_OPENPANEL_CLIENT_ID;

  if (!clientId) {
    return;
  }

  new OpenPanel({
    clientId,
    apiUrl: import.meta.env.VITE_OPENPANEL_API_URL,
    trackScreenViews: true,
    trackOutgoingLinks: true,
    trackAttributes: true,
  });
}
