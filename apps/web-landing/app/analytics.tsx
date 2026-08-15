'use client';

import { useEffect, useState } from 'react';
import { OpenPanelComponent } from '@openpanel/nextjs';

interface AnalyticsConfig {
  clientId: string | null;
  apiUrl: string | null;
  scriptUrl: string | null;
}

// Self-hosted OpenPanel at analytics.tesserix.app. The landing is a static
// export, so the config the Helm chart injects into the pod is read from the
// nginx-rendered /analytics-config.json at runtime. No client ID means no-op.
export function Analytics() {
  const [config, setConfig] = useState<AnalyticsConfig | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetch('/analytics-config.json')
      .then((response) => (response.ok ? response.json() : null))
      .then((loaded) => {
        if (!cancelled) setConfig(loaded);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  if (!config?.clientId) {
    return null;
  }

  return (
    <OpenPanelComponent
      clientId={config.clientId}
      apiUrl={config.apiUrl ?? undefined}
      scriptUrl={config.scriptUrl ?? undefined}
      trackScreenViews
      trackOutgoingLinks
      trackAttributes
    />
  );
}
