// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { OrderActionButton } from './OrderActionButton';
import { getChefOrderAction, type ChefOrderAction } from '@/features/orders/order-actions';
import type { OrderStatus } from '@/shared/types';

type AdvanceAction = Extract<ChefOrderAction, { kind: 'advance' }>;

function advanceActionFor(status: 'preparing' | 'ready', fulfillment: 'pickup' | 'delivery') {
  const action = getChefOrderAction(status, fulfillment);
  if (action?.kind !== 'advance') throw new Error('expected an advance action');
  return action as AdvanceAction;
}

let container: HTMLDivElement;
let root: Root;

beforeEach(() => {
  // React 19 logs a warning unless the test env opts in to act().
  (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
  container = document.createElement('div');
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
  vi.unstubAllGlobals();
});

function render(action: AdvanceAction, onAdvance: (nextStatus: OrderStatus) => void) {
  act(() => {
    root.render(
      <OrderActionButton
        orderId="o1"
        action={action}
        isPending={false}
        onAdvance={onAdvance}
      />
    );
  });
}

/** Drops a file into the hidden input and fires the change event React listens for. */
async function pickFile(file: File) {
  const input = container.querySelector('input[type="file"]') as HTMLInputElement;
  expect(input, 'photo-gated action must render a file input').toBeTruthy();
  Object.defineProperty(input, 'files', { value: [file], configurable: true });
  await act(async () => {
    input.dispatchEvent(new Event('change', { bubbles: true }));
  });
  return input;
}

describe('OrderActionButton — photo-gated transitions', () => {
  it('uploads the food-ready photo, then advances the order to ready', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ kind: 'ready', url: 'https://storage.googleapis.com/x/ready.jpg' }),
    });
    vi.stubGlobal('fetch', fetchMock);
    const onAdvance = vi.fn();

    render(advanceActionFor('preparing', 'delivery'), onAdvance);
    await pickFile(new File(['bytes'], 'dish.jpg', { type: 'image/jpeg' }));

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe('/bff/api/v1/chef/orders/o1/photos');
    expect(init.method).toBe('POST');
    expect(init.credentials).toBe('include');
    const body = init.body as FormData;
    expect(body.get('kind')).toBe('ready');
    expect((body.get('file') as File).name).toBe('dish.jpg');

    // The status only moves AFTER the upload resolves.
    expect(onAdvance).toHaveBeenCalledWith('ready', undefined);
  });

  it('carries the committed carrier through to the status update', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: async () => ({ kind: 'ready', url: 'u' }) })
    );
    const onAdvance = vi.fn();
    const action = getChefOrderAction('preparing', 'delivery', {
      offersSelfDelivery: true,
      riderDispatchAvailable: false,
    });
    if (action?.kind !== 'advance') throw new Error('expected an advance action');

    render(action, onAdvance);
    await pickFile(new File(['bytes'], 'dish.jpg', { type: 'image/jpeg' }));

    expect(onAdvance).toHaveBeenCalledWith('ready', 'chef_delivery');
  });

  it('sends kind=handover when completing a pickup order', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ kind: 'handover', url: 'https://storage.googleapis.com/x/h.jpg' }),
    });
    vi.stubGlobal('fetch', fetchMock);
    const onAdvance = vi.fn();

    render(advanceActionFor('ready', 'pickup'), onAdvance);
    await pickFile(new File(['bytes'], 'handover.png', { type: 'image/png' }));

    const body = (fetchMock.mock.calls[0] as [string, RequestInit])[1].body as FormData;
    expect(body.get('kind')).toBe('handover');
    expect(onAdvance).toHaveBeenCalledWith('delivered', undefined);
  });

  it('leaves the order where it was when the upload fails', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ error: 'Failed to upload photo' }),
    });
    vi.stubGlobal('fetch', fetchMock);
    const onAdvance = vi.fn();

    render(advanceActionFor('preparing', 'pickup'), onAdvance);
    await pickFile(new File(['bytes'], 'dish.jpg', { type: 'image/jpeg' }));

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(onAdvance).not.toHaveBeenCalled();
  });

  it('rejects a non-image before anything is uploaded', async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    const onAdvance = vi.fn();

    render(advanceActionFor('preparing', 'delivery'), onAdvance);
    await pickFile(new File(['%PDF'], 'invoice.pdf', { type: 'application/pdf' }));

    expect(fetchMock).not.toHaveBeenCalled();
    expect(onAdvance).not.toHaveBeenCalled();
  });

  it('clears the input so re-picking the same file after a failure still fires', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, json: async () => ({}) }));

    render(advanceActionFor('preparing', 'delivery'), vi.fn());
    const input = await pickFile(new File(['bytes'], 'dish.jpg', { type: 'image/jpeg' }));

    expect(input.value).toBe('');
  });

  it('advances immediately, with no upload, on a step that needs no photo', async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    const onAdvance = vi.fn();

    const action = getChefOrderAction('accepted', 'delivery');
    if (action?.kind !== 'advance') throw new Error('expected an advance action');
    render(action, onAdvance);

    expect(container.querySelector('input[type="file"]')).toBeNull();
    await act(async () => {
      container.querySelector('button')?.click();
    });
    expect(fetchMock).not.toHaveBeenCalled();
    expect(onAdvance).toHaveBeenCalledWith('preparing', undefined);
  });
});
