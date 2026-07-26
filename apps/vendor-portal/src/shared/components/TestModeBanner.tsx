import { useQuery } from '@tanstack/react-query';

import { apiClient } from '@/shared/services/api-client';

interface DashboardModeInfo {
  mode?: 'live' | 'test';
  testSessionNo?: number;
}

/**
 * Tells a chef that the platform team is currently running tests on their
 * kitchen.
 *
 * While a kitchen is in test mode every figure in this portal belongs to a
 * sandbox — orders, earnings, menu. Without this banner a chef would read fake
 * earnings as real, or panic that their real orders had vanished. It is
 * deliberately un-dismissable and shown on EVERY page rather than just the
 * dashboard, because the confusion is worst on the screens that show money.
 *
 * Deliberately NOT shown in the admin console: an admin looks at live and test
 * kitchens side by side, so a page-wide "this is sandbox data" claim would be
 * false there. Admin marks test kitchens per row instead.
 *
 * Renders nothing for a live kitchen, and nothing on error — a failed request
 * must never bury the portal under a scary banner.
 */
export function TestModeBanner() {
  const { data } = useQuery<DashboardModeInfo>({
    queryKey: ['chef-mode'],
    queryFn: () => apiClient.get('/chef/dashboard'),
    // The chef's mode changes rarely, but they should learn about a flip
    // without needing to reload the whole app.
    refetchInterval: 60_000,
    retry: false,
  });

  if (data?.mode !== 'test') return null;

  return (
    <div
      role="status"
      className="border-b border-amber-300 bg-amber-100 px-4 py-3 text-amber-950 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-100"
    >
      <p className="text-sm font-semibold tracking-wide">TEST MODE</p>
      <p className="mt-0.5 text-sm leading-relaxed">
        The Fe3dr team is running tests on your kitchen to investigate an issue. Orders and
        earnings shown here are not real, and customers cannot order from you right now. Your real
        orders and menu are safe and return as soon as testing finishes.
      </p>
      {data.testSessionNo ? (
        <p className="mt-1 text-xs font-semibold text-amber-900 dark:text-amber-200">
          Reference: test session {data.testSessionNo}
        </p>
      ) : null}
    </div>
  );
}
