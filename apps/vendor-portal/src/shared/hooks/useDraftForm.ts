import { useEffect, useCallback, useRef } from 'react';

const DRAFT_TTL_MS = 48 * 60 * 60 * 1000; // 48 hours

interface DraftEnvelope<T> {
  data: T;
  savedAt: number;
}

/**
 * Persists form data to localStorage with a 48-hour TTL.
 *
 * Usage:
 *   const { loadDraft, saveDraft, clearDraft } = useDraftForm<FormValues>('menu-item-new');
 *
 *   // On mount: loadDraft() returns saved data or null
 *   // On change: call saveDraft(formValues) — debounced internally
 *   // On submit: call clearDraft()
 */
export function useDraftForm<T>(key: string) {
  const storageKey = `draft:${key}`;
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Clean up debounce timer on unmount
  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  const loadDraft = useCallback((): T | null => {
    try {
      const raw = localStorage.getItem(storageKey);
      if (!raw) return null;

      const envelope: DraftEnvelope<T> = JSON.parse(raw);
      const age = Date.now() - envelope.savedAt;

      if (age > DRAFT_TTL_MS) {
        localStorage.removeItem(storageKey);
        return null;
      }

      return envelope.data;
    } catch {
      localStorage.removeItem(storageKey);
      return null;
    }
  }, [storageKey]);

  const saveDraft = useCallback(
    (data: T) => {
      // Debounce saves to avoid thrashing localStorage on every keystroke
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => {
        try {
          const envelope: DraftEnvelope<T> = { data, savedAt: Date.now() };
          localStorage.setItem(storageKey, JSON.stringify(envelope));
        } catch {
          // localStorage full or unavailable — silently ignore
        }
      }, 500);
    },
    [storageKey]
  );

  const clearDraft = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    localStorage.removeItem(storageKey);
  }, [storageKey]);

  return { loadDraft, saveDraft, clearDraft };
}

/**
 * Zustand persist storage adapter with 48-hour TTL.
 * Use as the `storage` option in zustand/persist.
 *
 * Example:
 *   persist(storeCreator, {
 *     name: 'vendor-onboarding',
 *     storage: createTTLStorage(48 * 60 * 60 * 1000),
 *   })
 */
export function createTTLStorage(ttlMs: number = DRAFT_TTL_MS) {
  return {
    getItem: (name: string) => {
      try {
        const raw = localStorage.getItem(name);
        if (!raw) return null;

        const parsed = JSON.parse(raw);

        // Check TTL
        if (parsed._savedAt) {
          const age = Date.now() - parsed._savedAt;
          if (age > ttlMs) {
            localStorage.removeItem(name);
            return null;
          }
        }

        return parsed;
      } catch {
        return null;
      }
    },
    setItem: (name: string, value: unknown) => {
      try {
        const withTimestamp = { ...(value as Record<string, unknown>), _savedAt: Date.now() };
        localStorage.setItem(name, JSON.stringify(withTimestamp));
      } catch {
        // silently ignore
      }
    },
    removeItem: (name: string) => {
      localStorage.removeItem(name);
    },
  };
}
