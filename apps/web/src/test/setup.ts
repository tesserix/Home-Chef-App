import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// zustand's persist middleware binds its storage backend when the store module
// is first imported, which happens before any test body runs — so this has to
// live in setup rather than in an individual test. Without it, importing any
// persisted store (cart, favourites) throws on first write and the test failure
// points at zustand internals rather than at the component under test.
{
  const mem = new Map<string, string>();
  Object.defineProperty(globalThis, 'localStorage', {
    value: {
      getItem: (k: string) => mem.get(k) ?? null,
      setItem: (k: string, v: string) => void mem.set(k, String(v)),
      removeItem: (k: string) => void mem.delete(k),
      clear: () => mem.clear(),
      key: (i: number) => Array.from(mem.keys())[i] ?? null,
      get length() {
        return mem.size;
      },
    },
    writable: true,
    configurable: true,
  });
}

// React Testing Library does not auto-clean under vitest's globals mode.
afterEach(() => {
  cleanup();
});
