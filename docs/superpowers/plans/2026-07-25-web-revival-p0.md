# Web Revival P0 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the revived web surfaces correct and safe to expose to users — fix the money-affecting cancellation path, add the legally-required data-rights screens, and restore build validation for both web apps.

**Architecture:** `apps/web` (React 19 + Vite SPA) and `apps/vendor-portal` talk to the same Go API (`api.fe3dr.com`) the mobile apps use. The mobile hooks in `apps/mobile-customer/hooks/` are the reference implementations — this plan ports their behaviour to web using web's own `apiClient`, which differs from mobile's axios client in two ways that cause silent bugs if ignored (see Global Constraints).

**Tech Stack:** React 19, TypeScript 5.7, Vite 6, TanStack React Query v5, Zustand, Tailwind 4, Vitest + jsdom, Go 1.26 API.

## Global Constraints

- **Web `apiClient` returns the parsed body directly.** `apiClient.get<T>(url)` resolves to `T`. Do **not** write `.then(r => r.data)` — that is mobile's axios idiom and yields `undefined` on web.
- **Web endpoints take no `/v1` prefix.** `VITE_API_URL` already ends in `/api/v1` (`apps/web/src/shared/services/api-client.ts`). Mobile hooks show `/v1/...` because their baseURL ends at `/api`. Copying the mobile path verbatim 404s.
- **Import alias:** `@/` maps to `apps/web/src/`.
- **TS strictness:** `noUnusedLocals`, `noUnusedParameters`, `noUncheckedIndexedAccess` are all on. Prefix deliberately-unused params with `_`.
- **Toasts:** use `toast` from `sonner`, already a dependency.
- **No new colour, radius, or motion values.** Use existing Tailwind tokens. Source of truth `.impeccable.md`.
- **Never call `POST /orders/{id}/cancel`.** It is the legacy direct-cancel endpoint that bypasses the refund engine. See Task 2.
- **Commit after every task.** Conventional commits, no AI/assistant references in messages.
- **Verify with `pnpm --filter @homechef/web typecheck` and `pnpm --filter @homechef/web test`** before every commit.

## Open decisions — resolve before any deploy task

These block deployment, not implementation. Tasks 1-4 are safe to complete first.

1. **The `homechef-web` image slot is shared.** `.github/workflows/homechef-web-build.yml` warns that a `deploy=true` dispatch "OVERWRITES the live web-landing serving fe3dr.com". One slot cannot serve both the marketing landing and the customer SPA. Someone must choose: give the slot back to `apps/web` and move the landing to its own ArgoCD app/chart, put the SPA on a separate host, or serve the landing at `/` with the SPA under a path.
2. **auth-bff must be redeployed.** Commit `46257856` restored the `web` and `vendor-portal` product-registry entries, but `homechef-products.yaml` is baked into the image (`apps/auth-bff/Dockerfile:19`). Until auth-bff is rebuilt and synced, **login stays broken on both hosts** and Tasks 2-3 cannot be verified against production. Consider mounting the registry from a ConfigMap so host changes stop requiring an image rebuild.

---

### Task 1: Test harness for `apps/web`

`apps/web` has `vitest` and `jsdom` in devDependencies and a `test` script, but no vitest config block and no testing-library. `pnpm --filter @homechef/web test` currently exits 1 with "No test files found". Every later task is TDD, so the harness comes first.

**Files:**
- Modify: `apps/web/package.json` (add testing-library devDependencies)
- Modify: `apps/web/vite.config.ts` (add the `test` block)
- Create: `apps/web/src/test/setup.ts`
- Create: `apps/web/src/test/renderWithProviders.tsx`
- Test: `apps/web/src/shared/services/api-client.test.ts`

**Interfaces:**
- Consumes: nothing.
- Produces: `renderWithProviders(ui: React.ReactElement): RenderResult` — wraps a component in a fresh `QueryClientProvider` and `MemoryRouter`. Every later task's component tests use it.

- [ ] **Step 1: Add the test dependencies**

```bash
cd apps/web
pnpm add -D @testing-library/react@^16.1.0 @testing-library/jest-dom@^6.6.3 @testing-library/user-event@^14.5.2
```

- [ ] **Step 2: Add the vitest config block**

In `apps/web/vite.config.ts`, add a `test` property to the `defineConfig` object, as a sibling of `server` and `build`:

```ts
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    // The SPA has no tests outside src/; keep the glob tight so vitest does
    // not try to run anything under dist/ or node_modules/.
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
  },
```

Vite's `defineConfig` needs the vitest types for this to typecheck. Add this as the **first line** of the file:

```ts
/// <reference types="vitest" />
```

- [ ] **Step 3: Create the setup file**

Create `apps/web/src/test/setup.ts`:

```ts
import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// React Testing Library does not auto-clean under vitest's globals mode.
afterEach(() => {
  cleanup();
});
```

- [ ] **Step 4: Create the render helper**

Create `apps/web/src/test/renderWithProviders.tsx`:

```tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, type RenderResult } from '@testing-library/react';
import type { ReactElement, ReactNode } from 'react';
import { MemoryRouter } from 'react-router-dom';

/**
 * Renders a component with the providers every page in this app assumes.
 *
 * A fresh QueryClient per call keeps tests isolated — a shared client leaks
 * cached queries between them. Retries are off so a rejected query surfaces
 * immediately instead of after three backoffs.
 */
export function renderWithProviders(ui: ReactElement): RenderResult {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>{children}</MemoryRouter>
      </QueryClientProvider>
    );
  }

  return render(ui, { wrapper: Wrapper });
}
```

- [ ] **Step 5: Write a test that proves the harness works**

Create `apps/web/src/shared/services/api-client.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { renderWithProviders } from '@/test/renderWithProviders';

describe('test harness', () => {
  it('renders a component through the providers', () => {
    const { getByText } = renderWithProviders(<span>harness ok</span>);
    expect(getByText('harness ok')).toBeInTheDocument();
  });
});
```

Because this file contains JSX, rename it to `api-client.test.tsx`.

- [ ] **Step 6: Run the tests**

Run: `pnpm --filter @homechef/web test -- --run`
Expected: 1 test passes, exit 0.

- [ ] **Step 7: Typecheck**

Run: `pnpm --filter @homechef/web typecheck`
Expected: exit 0.

- [ ] **Step 8: Commit**

```bash
git add apps/web/package.json apps/web/vite.config.ts apps/web/src/test pnpm-lock.yaml
git add apps/web/src/shared/services/api-client.test.tsx
git commit -m "test(web): add vitest harness and provider render helper"
```

---

### Task 2: Move web order cancellation onto the cancel-request flow

`apps/web/src/features/customer/pages/OrderDetailPage.tsx:140-149` posts to `/orders/{id}/cancel`. That is the legacy endpoint: it sets `status=cancelled` and returns, with no policy check and no refund calculation. Mobile uses `/orders/{id}/cancel-request`, which runs the cancellation policy, the refund calculator, vendor confirmation, and the dispute path. A customer cancelling on web today skips the entire refund engine.

Reference implementation: `apps/mobile-customer/hooks/useCancellation.ts`.

**Files:**
- Create: `apps/web/src/features/customer/hooks/useCancellation.ts`
- Create: `apps/web/src/features/customer/hooks/useCancellation.test.tsx`
- Modify: `apps/web/src/features/customer/pages/OrderDetailPage.tsx:140-149` (replace `cancelMutation`)

**Interfaces:**
- Consumes: `renderWithProviders` from Task 1; `apiClient` from `@/shared/services/api-client`.
- Produces:
  - `interface CancellationRequest { id, orderId, status, vendorReason?, refundDestination?, refundTotalPaise, refundExecuted, vendorRespondBy? }`
  - `useCancellationRequest(orderId: string | undefined)` → `UseQueryResult<CancellationRequest | null>`
  - `useRequestCancellation()` → mutation taking `{ orderId: string; reason?: string }`
  - `useDisputeCancellation()` → mutation taking `{ orderId: string; reason?: string }`
  - `orderCancellable(status: string): boolean`

- [ ] **Step 1: Write the failing test**

Create `apps/web/src/features/customer/hooks/useCancellation.test.tsx`:

```tsx
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/shared/services/api-client';
import {
  orderCancellable,
  useRequestCancellation,
} from '@/features/customer/hooks/useCancellation';

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe('useRequestCancellation', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('posts to the cancel-request endpoint, never the legacy direct cancel', async () => {
    const post = vi.spyOn(apiClient, 'post').mockResolvedValue({});

    const { result } = renderHook(() => useRequestCancellation(), { wrapper });
    result.current.mutate({ orderId: 'order-1', reason: 'changed my mind' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(post).toHaveBeenCalledWith('/orders/order-1/cancel-request', {
      reason: 'changed my mind',
    });
    // The legacy endpoint bypasses the refund engine — it must never be hit.
    expect(post).not.toHaveBeenCalledWith(
      '/orders/order-1/cancel',
      expect.anything(),
    );
  });
});

describe('orderCancellable', () => {
  it('allows the stages the API still accepts a request for', () => {
    expect(orderCancellable('pending')).toBe(true);
    expect(orderCancellable('accepted')).toBe(true);
    expect(orderCancellable('preparing')).toBe(true);
  });

  it('rejects terminal and in-transit stages', () => {
    expect(orderCancellable('delivered')).toBe(false);
    expect(orderCancellable('cancelled')).toBe(false);
    expect(orderCancellable('out_for_delivery')).toBe(false);
  });
});
```

- [ ] **Step 2: Run it to make sure it fails**

Run: `pnpm --filter @homechef/web test -- --run src/features/customer/hooks/useCancellation.test.tsx`
Expected: FAIL — cannot resolve `@/features/customer/hooks/useCancellation`.

- [ ] **Step 3: Write the hook**

Create `apps/web/src/features/customer/hooks/useCancellation.ts`:

```ts
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Customer cancellation-with-arbitration (#475/#478). Requests a cancel, shows
// the vendor's decision + refund, and disputes it. This is the same API the
// vendor surfaces and the mobile app use — the legacy POST /orders/:id/cancel
// skips the policy and refund calculation entirely and must not be used.

export interface CancellationRequest {
  id: string;
  orderId: string;
  /** pending_vendor | auto_refunded | approved | disputed | admin_review | resolved */
  status: string;
  vendorReason?: string;
  refundDestination?: RefundDestination;
  refundTotalPaise: number;
  refundExecuted: boolean;
  vendorRespondBy?: string | null;
}

/**
 * Where a refund landed, as reported by the server. Not a customer choice —
 * the server derives it from the order's payment and refunds to the original
 * method. 'wallet' appears only on legacy rows, or where there is no gateway
 * payment to refund against.
 */
export type RefundDestination = 'wallet' | 'original';

/** The cancellation request for an order (null when none). Polls while pending. */
export function useCancellationRequest(orderId: string | undefined) {
  return useQuery<CancellationRequest | null>({
    queryKey: ['order', orderId, 'cancel-request'],
    queryFn: () =>
      apiClient
        .get<{ request: CancellationRequest }>(`/orders/${orderId}/cancel-request`)
        .then((r) => r.request)
        .catch(() => null), // 404 = no request yet
    enabled: Boolean(orderId),
    retry: false,
    refetchInterval: (query) =>
      query.state.data?.status === 'pending_vendor' ? 20000 : false,
  });
}

export function useRequestCancellation() {
  const queryClient = useQueryClient();
  return useMutation({
    // No refundDestination: the server decides (original payment method).
    mutationFn: (vars: { orderId: string; reason?: string }) =>
      apiClient.post(`/orders/${vars.orderId}/cancel-request`, {
        reason: vars.reason,
      }),
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({ queryKey: ['order', vars.orderId] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
    },
  });
}

export function useDisputeCancellation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (vars: { orderId: string; reason?: string }) =>
      apiClient.post(`/orders/${vars.orderId}/cancel-request/dispute`, {
        reason: vars.reason,
      }),
    onSuccess: (_data, vars) =>
      queryClient.invalidateQueries({
        queryKey: ['order', vars.orderId, 'cancel-request'],
      }),
  });
}

/** Whether an order is at a stage the customer can still ask to cancel. */
export function orderCancellable(status: string): boolean {
  return ['pending', 'accepted', 'preparing'].includes(status);
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `pnpm --filter @homechef/web test -- --run src/features/customer/hooks/useCancellation.test.tsx`
Expected: PASS, 3 tests.

- [ ] **Step 5: Point the page at the new hook**

In `apps/web/src/features/customer/pages/OrderDetailPage.tsx`, replace the `cancelMutation` definition (currently lines 140-149) with the hook. Add to the imports at the top of the file:

```ts
import { useRequestCancellation } from '@/features/customer/hooks/useCancellation';
```

Replace the whole `const cancelMutation = useMutation({ ... });` block with:

```ts
  // Requests a cancellation through the policy + refund engine. The old
  // POST /orders/:id/cancel skipped both.
  const cancelMutation = useRequestCancellation();
```

Then find the call site that invokes it (search for `cancelMutation.mutate`) and change the argument from the bare reason to the hook's variables shape:

```ts
  cancelMutation.mutate(
    { orderId: id!, reason: cancelReason },
    {
      onSuccess: () => {
        toast.success('Cancellation requested — the kitchen will confirm shortly');
        setShowCancelDialog(false);
      },
      onError: () => {
        toast.error('Could not request cancellation');
      },
    },
  );
```

Update the confirmation copy in the cancel dialog (around line 576) — the action is no longer immediate:

```tsx
Ask the kitchen to cancel this order? They confirm it, and any refund is
worked out from the cancellation policy.
```

- [ ] **Step 6: Verify nothing still calls the legacy endpoint**

Run: `grep -rn "orders/\${id}/cancel\`\|orders/\${orderId}/cancel\`" apps/web/src`
Expected: no matches (the only cancel paths are `/cancel-request` and `/cancel-request/dispute`).

- [ ] **Step 7: Typecheck and run the whole suite**

Run: `pnpm --filter @homechef/web typecheck && pnpm --filter @homechef/web test -- --run`
Expected: both exit 0.

- [ ] **Step 8: Commit**

```bash
git add apps/web/src/features/customer/hooks/useCancellation.ts \
        apps/web/src/features/customer/hooks/useCancellation.test.tsx \
        apps/web/src/features/customer/pages/OrderDetailPage.tsx
git commit -m "fix(web): request cancellations through the policy and refund engine"
```

---

### Task 3: Customer data-rights on web

`/customer/me/export`, `/customer/me/deletion-eligibility`, `/customer/me/delete`, `/customer/me/deactivate` and `/customer/me/reactivate` are called by the mobile app and by nothing on web. These are DPDP Act 2023 access/erasure rights and Google Play's account-deletion policy. `fe3dr.com/account-deletion/` currently tells users to do it in the app — acceptable while web is a marketing page, not acceptable once web is a signed-in ordering surface.

Reference implementation: `apps/mobile-customer/hooks/useDataPrivacy.ts` and screen `apps/mobile-customer/app/data-privacy.tsx`.

**Files:**
- Create: `apps/web/src/features/customer/hooks/useDataPrivacy.ts`
- Create: `apps/web/src/features/customer/hooks/useDataPrivacy.test.tsx`
- Create: `apps/web/src/features/customer/pages/DataPrivacyPage.tsx`
- Modify: `apps/web/src/app/routes/index.tsx` (add the lazy import and the `data-privacy` route)

**Interfaces:**
- Consumes: `apiClient`; `renderWithProviders` from Task 1.
- Produces:
  - `interface DeletionBlocker { code: 'active_orders' | 'mealplan_escrow' | 'wallet_balance' | 'pending_payout' | 'active_delivery'; label: string; count?: number; amount?: number }`
  - `interface DeletionEligibility { deletable: boolean; blockers: DeletionBlocker[]; retentionDays: number }`
  - `interface DeleteAccountResult { status: 'deleted' | 'already_deleted'; deletedAt: string; purgeAfter: string; notice?: string }`
  - `useExportMyData()`, `useDeletionEligibility()`, `useDeleteAccount()`, `useDeactivateAccount()`, `useReactivateAccount()`

- [ ] **Step 1: Write the failing test**

Create `apps/web/src/features/customer/hooks/useDataPrivacy.test.tsx`:

```tsx
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/shared/services/api-client';
import {
  useDeleteAccount,
  useDeletionEligibility,
} from '@/features/customer/hooks/useDataPrivacy';

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe('useDeletionEligibility', () => {
  beforeEach(() => vi.restoreAllMocks());

  it('reads eligibility without the /v1 prefix', async () => {
    const get = vi.spyOn(apiClient, 'get').mockResolvedValue({
      deletable: false,
      blockers: [{ code: 'wallet_balance', label: 'Wallet credit left', amount: 250 }],
      retentionDays: 180,
    });

    const { result } = renderHook(() => useDeletionEligibility(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(get).toHaveBeenCalledWith('/customer/me/deletion-eligibility');
    expect(result.current.data?.blockers[0]?.code).toBe('wallet_balance');
  });
});

describe('useDeleteAccount', () => {
  beforeEach(() => vi.restoreAllMocks());

  it('sends the typed confirmation email as the body', async () => {
    const post = vi.spyOn(apiClient, 'post').mockResolvedValue({
      status: 'deleted',
      deletedAt: '2026-07-25T00:00:00Z',
      purgeAfter: '2027-01-21T00:00:00Z',
    });

    const { result } = renderHook(() => useDeleteAccount(), { wrapper });
    result.current.mutate('customer@demo.com');
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(post).toHaveBeenCalledWith('/customer/me/delete', {
      confirmEmail: 'customer@demo.com',
    });
  });
});
```

- [ ] **Step 2: Run it to make sure it fails**

Run: `pnpm --filter @homechef/web test -- --run src/features/customer/hooks/useDataPrivacy.test.tsx`
Expected: FAIL — module not found.

- [ ] **Step 3: Confirm the request bodies against the API before writing the hook**

The exact field names matter. Read the Go handlers and match them:

Run: `grep -rn "ConfirmEmail\|confirmEmail" apps/api/handlers/account_lifecycle.go | head`

Use whatever JSON tag the handler binds. If it is not `confirmEmail`, use the real one in both the hook and the test above.

- [ ] **Step 4: Write the hook**

Create `apps/web/src/features/customer/hooks/useDataPrivacy.ts`:

```ts
import { useMutation, useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// DPDP Act 2023 data-subject actions plus the account lifecycle
// (deactivate / delete / restore) required by Google Play's account-deletion
// policy. Mirrors apps/mobile-customer/hooks/useDataPrivacy.ts.

/** One reason the account cannot be deleted yet. Codes are stable. */
export interface DeletionBlocker {
  code:
    | 'active_orders'
    | 'mealplan_escrow'
    | 'wallet_balance'
    | 'pending_payout'
    | 'active_delivery';
  label: string;
  count?: number;
  amount?: number;
}

export interface DeletionEligibility {
  deletable: boolean;
  blockers: DeletionBlocker[];
  retentionDays: number;
}

export interface DeleteAccountResult {
  status: 'deleted' | 'already_deleted';
  deletedAt: string;
  /** When the account is erased for good — restore is possible until then. */
  purgeAfter: string;
  notice?: string;
}

export function useExportMyData() {
  return useMutation<unknown, unknown, void>({
    mutationFn: () => apiClient.get('/customer/me/export'),
  });
}

/**
 * Previews whether deletion would succeed, so the page can show "finish these
 * first" up front rather than after the user types their email and presses a
 * button that then fails.
 */
export function useDeletionEligibility() {
  return useQuery<DeletionEligibility>({
    queryKey: ['deletion-eligibility'],
    queryFn: () => apiClient.get<DeletionEligibility>('/customer/me/deletion-eligibility'),
    staleTime: 30_000,
  });
}

export function useDeleteAccount() {
  return useMutation<DeleteAccountResult, unknown, string>({
    mutationFn: (confirmEmail: string) =>
      apiClient.post<DeleteAccountResult>('/customer/me/delete', { confirmEmail }),
  });
}

export function useDeactivateAccount() {
  return useMutation<unknown, unknown, void>({
    mutationFn: () => apiClient.post('/customer/me/deactivate'),
  });
}

export function useReactivateAccount() {
  return useMutation<unknown, unknown, void>({
    mutationFn: () => apiClient.post('/customer/me/reactivate'),
  });
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `pnpm --filter @homechef/web test -- --run src/features/customer/hooks/useDataPrivacy.test.tsx`
Expected: PASS, 2 tests.

- [ ] **Step 6: Build the page**

Create `apps/web/src/features/customer/pages/DataPrivacyPage.tsx`:

```tsx
import { useState } from 'react';
import { toast } from 'sonner';
import { useAuth } from '@/app/providers/AuthProvider';
import {
  useDeactivateAccount,
  useDeleteAccount,
  useDeletionEligibility,
  useExportMyData,
} from '@/features/customer/hooks/useDataPrivacy';
import { Button, Card } from '@/shared/components/ui';

/**
 * DPDP Act 2023 access and erasure rights, plus the reversible pause.
 *
 * Deletion eligibility is fetched on mount rather than checked on submit, so
 * an account with money or work in flight says so up front instead of failing
 * after the user has typed their email.
 */
export default function DataPrivacyPage() {
  const { user } = useAuth();
  const [confirmEmail, setConfirmEmail] = useState('');
  const [purgeAfter, setPurgeAfter] = useState<string | null>(null);

  const eligibility = useDeletionEligibility();
  const exportData = useExportMyData();
  const deactivate = useDeactivateAccount();
  const deleteAccount = useDeleteAccount();

  const accountEmail = user?.email ?? '';
  const emailMatches =
    confirmEmail.trim().toLowerCase() === accountEmail.toLowerCase() &&
    accountEmail.length > 0;
  const blockers = eligibility.data?.blockers ?? [];
  const deletable = eligibility.data?.deletable === true;

  function handleExport() {
    exportData.mutate(undefined, {
      onSuccess: (data) => {
        // Hand the user a file rather than dumping JSON on screen — this is
        // their record to keep.
        const blob = new Blob([JSON.stringify(data, null, 2)], {
          type: 'application/json',
        });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = 'fe3dr-my-data.json';
        link.click();
        URL.revokeObjectURL(url);
      },
      onError: () => toast.error('Could not prepare your data — try again'),
    });
  }

  function handleDeactivate() {
    deactivate.mutate(undefined, {
      onSuccess: () => toast.success('Account paused. Sign in again to resume.'),
      onError: () => toast.error('Could not pause your account — try again'),
    });
  }

  function handleDelete() {
    deleteAccount.mutate(confirmEmail.trim(), {
      onSuccess: (result) => setPurgeAfter(result.purgeAfter),
      onError: () => toast.error('Could not delete your account — try again'),
    });
  }

  if (purgeAfter) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-8">
        <h1 className="text-2xl font-semibold text-foreground">Account deleted</h1>
        <p className="mt-3 text-muted-foreground">
          Your sign-in no longer works. We keep your data until{' '}
          {new Date(purgeAfter).toLocaleDateString()} — sign up again with the
          same email before then and you can restore it. After that it is erased
          for good.
        </p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      <h1 className="text-2xl font-semibold text-foreground">Your data</h1>

      <Card className="mt-8 p-6">
        <h2 className="text-lg font-medium text-foreground">Download my data</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          A machine-readable copy of your profile, orders and related records.
        </p>
        <Button
          className="mt-4"
          onClick={handleExport}
          disabled={exportData.isPending}
        >
          {exportData.isPending ? 'Preparing…' : 'Download my data'}
        </Button>
      </Card>

      <Card className="mt-4 p-6">
        <h2 className="text-lg font-medium text-foreground">Pause my account</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          Hides your profile and stops notifications. Nothing is deleted, and
          there is no time limit — sign in again whenever you want to come back.
        </p>
        <Button
          variant="outline"
          className="mt-4"
          onClick={handleDeactivate}
          disabled={deactivate.isPending}
        >
          {deactivate.isPending ? 'Pausing…' : 'Pause my account'}
        </Button>
      </Card>

      <Card className="mt-4 p-6">
        <h2 className="text-lg font-medium text-foreground">Delete my account</h2>

        {eligibility.isPending ? (
          <p className="mt-2 text-sm text-muted-foreground">Checking…</p>
        ) : deletable ? (
          <>
            <p className="mt-2 text-sm text-muted-foreground">
              Your sign-in stops working straight away. We keep your data for{' '}
              {eligibility.data?.retentionDays ?? 180} days in case you change
              your mind, then erase it permanently.
            </p>
            <label
              htmlFor="confirm-email"
              className="mt-4 block text-sm font-medium text-foreground"
            >
              Type {accountEmail} to confirm
            </label>
            <input
              id="confirm-email"
              type="email"
              autoComplete="off"
              value={confirmEmail}
              onChange={(e) => setConfirmEmail(e.target.value)}
              className="mt-2 w-full rounded-lg border border-input px-3 py-2 text-foreground"
            />
            <Button
              variant="destructive"
              className="mt-4"
              onClick={handleDelete}
              disabled={!emailMatches || deleteAccount.isPending}
            >
              {deleteAccount.isPending ? 'Deleting…' : 'Delete my account'}
            </Button>
          </>
        ) : (
          <>
            <p className="mt-2 text-sm text-muted-foreground">
              Finish these first — deleting now would strand a payment or an
              order someone is waiting on.
            </p>
            <ul className="mt-3 space-y-2">
              {blockers.map((blocker) => (
                <li key={blocker.code} className="text-sm text-foreground">
                  {blocker.label}
                </li>
              ))}
            </ul>
            <Button variant="destructive" className="mt-4" disabled>
              Delete my account
            </Button>
          </>
        )}
      </Card>
    </div>
  );
}
```

Check the named exports against `apps/web/src/shared/components/ui` before running — if `Card` or a `variant` is not exported there, use the plain element the neighbouring pages use rather than inventing a component.

- [ ] **Step 7: Register the route**

In `apps/web/src/app/routes/index.tsx`, add the lazy import beside the other customer pages:

```ts
const DataPrivacyPage = lazyWithRetry(() => import('@/features/customer/pages/DataPrivacyPage'));
```

and add the route inside the same authenticated block that holds `profile` and `wallet`:

```tsx
<Route path="data-privacy" element={<DataPrivacyPage />} />
```

- [ ] **Step 8: Link it from the profile page**

In `apps/web/src/features/customer/pages/ProfilePage.tsx`, add a link to `/data-privacy` labelled "Your data" in the same list as the existing account links, so the page is reachable without typing a URL.

- [ ] **Step 9: Typecheck and run the whole suite**

Run: `pnpm --filter @homechef/web typecheck && pnpm --filter @homechef/web test -- --run`
Expected: both exit 0.

- [ ] **Step 10: Commit**

```bash
git add apps/web/src/features/customer/hooks/useDataPrivacy.ts \
        apps/web/src/features/customer/hooks/useDataPrivacy.test.tsx \
        apps/web/src/features/customer/pages/DataPrivacyPage.tsx \
        apps/web/src/features/customer/pages/ProfilePage.tsx \
        apps/web/src/app/routes/index.tsx
git commit -m "feat(web): data export, pause and account deletion for customers"
```

---

### Task 4: Restore build validation for both web apps

`homechef-web-build.yml` is manual-dispatch only, so nothing catches a broken `apps/web` build on a PR — which is exactly how it rotted into the four TypeScript errors fixed in `a51110d1`. `vendor-portal` has no workflow at all; `homechef-vendor-portal-build.yml` was deleted when the portal was sunset.

This task restores **build validation only**. It must not enable image push or deploy — the `homechef-web` slot conflict in "Open decisions" is unresolved, and a `deploy=true` run would overwrite the live landing page.

**Files:**
- Modify: `.github/workflows/homechef-web-build.yml`
- Create: `.github/workflows/homechef-vendor-portal-build.yml`

**Interfaces:**
- Consumes: nothing.
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Re-enable the PR trigger on the web workflow, without deploy**

In `.github/workflows/homechef-web-build.yml`, uncomment the `pull_request` trigger only, and leave `push` commented out. Replace the commented `pull_request` block with a live one:

```yaml
  pull_request:
    branches: [main]
    paths:
      - 'apps/web/**'
      - 'packages/**'
      - 'pnpm-lock.yaml'
      - '.github/workflows/homechef-web-build.yml'
```

Update the workflow `name` from `HomeChef Web - CI Build (paused)` to `HomeChef Web - CI Build`, and replace the PAUSED comment block with:

```yaml
# Build validation for apps/web. Runs on PRs that touch the app so it cannot
# rot again (it accumulated four TS errors while nothing built it).
#
# Deploy stays manual: `inputs.deploy` gates the image push, and the
# homechef-web slot currently serves the web-landing image, so an accidental
# deploy would overwrite the live fe3dr.com landing. Resolve that slot conflict
# before wiring push-to-main deploys.
```

Confirm the push step is still gated — it must read `push: ${{ inputs.deploy == true }}`, which is false for any `pull_request` run.

- [ ] **Step 2: Verify the web workflow parses**

Run: `gh workflow view homechef-web-build.yml --repo tesserix/Home-Chef-App` if the repo is reachable, otherwise validate locally:

```bash
python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/homechef-web-build.yml')); print('ok')"
```

Expected: `ok`.

- [ ] **Step 3: Create the vendor-portal workflow**

Create `.github/workflows/homechef-vendor-portal-build.yml` by copying `homechef-web-build.yml` and changing exactly these things:

- `name: HomeChef Vendor Portal - CI Build`
- `IMAGE_NAME: tesserix/home-chef-app/homechef-vendor-portal`
- every `apps/web/**` path filter → `apps/vendor-portal/**`
- the workflow-file path filter → `.github/workflows/homechef-vendor-portal-build.yml`
- the build command target → `@homechef/vendor-portal`
- the Docker `file:` → `apps/vendor-portal/Dockerfile`
- `org.opencontainers.image.title=homechef-vendor-portal`

Keep the same `workflow_dispatch` + `pull_request` trigger shape and the same `push: ${{ inputs.deploy == true }}` gate.

- [ ] **Step 4: Verify the vendor workflow parses**

```bash
python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/homechef-vendor-portal-build.yml')); print('ok')"
```

Expected: `ok`.

- [ ] **Step 5: Prove both apps actually build**

Run: `pnpm --filter @homechef/web build && pnpm --filter @homechef/vendor-portal build`
Expected: both exit 0.

- [ ] **Step 6: Commit**

```bash
git add .github/workflows/homechef-web-build.yml \
        .github/workflows/homechef-vendor-portal-build.yml
git commit -m "ci: validate web and vendor-portal builds on pull requests"
```

---

## Known gap against the P0 list

`docs/web-mobile-parity.md` §10 P0 item 2 says data-rights must land on "web
**and** portal". Task 3 covers the customer web only. The chef equivalents
(`/chef/me/export`, `/chef/me/delete`, `/chef/me/deletion-eligibility`,
`/chef/me/deactivate`, `/chef/me/reactivate`) are in Plan 3 with the rest of
the vendor-portal work, because they share its page shell and auth context.
If the portal ships to chefs before Plan 3 completes, pull that task forward —
the Play policy applies to the chef app's account too.

## Follow-on plans

This plan deliberately stops at P0. The remaining parity work is two further plans, each independently testable:

- **Plan 2 — Customer web feature parity.** Meal plans / tiffin (6 endpoints, 5 screens), order tracking, chef chat, report-issue, confirm-received, delivery quote + fulfilment slot at checkout, dish search, catering deposits. Source: `docs/web-mobile-parity.md` §4.2.
- **Plan 3 — Vendor portal parity.** The 28 endpoints in `docs/web-mobile-parity.md` §6: availability pause/resume, per-item cancel, delivery-failed, catering quote/complete, support tickets, media uploads, notification preferences, chef account lifecycle.
