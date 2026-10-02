import { afterEach, expect, it, vi } from 'vitest';

afterEach(() => { delete window.Stripe; document.head.querySelectorAll('script[src="https://js.stripe.com/v3/"]').forEach((script) => script.remove()); vi.resetModules(); });

it('initializes each requested platform key after a shared script load', async () => {
  const { loadStripeJs } = await import('./load-stripe');
  const first = loadStripeJs('pk_test_au');
  const second = loadStripeJs('pk_test_nz');
  const constructor = vi.fn();
  window.Stripe = constructor;
  document.head.querySelector('script')!.dispatchEvent(new Event('load'));
  await Promise.all([first, second]);
  expect(constructor.mock.calls).toEqual([['pk_test_au'], ['pk_test_nz']]);
});

it('can retry after a script network failure', async () => {
  const { loadStripeJs } = await import('./load-stripe');
  const failed = loadStripeJs('pk_test_au');
  const oldScript = document.head.querySelector('script')!;
  oldScript.dispatchEvent(new Event('error'));
  expect(await failed).toBeNull();
  const retry = loadStripeJs('pk_test_au');
  const newScript = document.head.querySelector('script')!;
  expect(newScript).not.toBe(oldScript);
  const constructor = vi.fn();
  window.Stripe = constructor;
  newScript.dispatchEvent(new Event('load'));
  await retry;
  expect(constructor).toHaveBeenCalledWith('pk_test_au');
});
