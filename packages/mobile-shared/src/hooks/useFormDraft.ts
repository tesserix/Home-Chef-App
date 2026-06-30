import { useCallback, useEffect, useRef, useState } from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';

const DRAFT_TTL_MS = 48 * 60 * 60 * 1000; // 48 hours

interface DraftEnvelope<T> {
  data: T;
  savedAt: number;
}

/**
 * Persists in-progress form data to AsyncStorage with a 48-hour TTL so a
 * background/kill/accidental-nav does not lose what the user typed.
 *
 * Mobile counterpart of the web `useDraftForm` hook. Load is async, so the
 * hook exposes a `ready` flag plus an already-resolved `draft` value instead
 * of a synchronous `loadDraft()` — restore the draft into form state once
 * `ready` flips true.
 *
 * Usage:
 *   const { ready, draft, saveDraft, clearDraft } = useFormDraft<FormValues>('menu-item-new');
 *
 *   useEffect(() => {
 *     if (ready && draft) reset(draft); // restore once load resolves
 *   }, [ready]);
 *
 *   // On change: saveDraft(values) — debounced internally (~500ms)
 *   // On submit: clearDraft()
 */
export function useFormDraft<T>(key: string): {
  ready: boolean;
  draft: T | null;
  saveDraft: (data: T) => void;
  clearDraft: () => void;
} {
  const storageKey = `draft:${key}`;
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [ready, setReady] = useState(false);
  const [draft, setDraft] = useState<T | null>(null);

  // Load any saved draft once on mount (or when the key changes).
  useEffect(() => {
    let active = true;
    setReady(false);
    setDraft(null);

    (async () => {
      try {
        const raw = await AsyncStorage.getItem(storageKey);
        if (!active) return;

        if (raw) {
          const envelope: DraftEnvelope<T> = JSON.parse(raw);
          const age = Date.now() - envelope.savedAt;

          if (age > DRAFT_TTL_MS) {
            await AsyncStorage.removeItem(storageKey).catch(() => {});
          } else {
            setDraft(envelope.data);
          }
        }
      } catch {
        // Corrupt/unreadable draft — drop it and continue with a clean form.
        await AsyncStorage.removeItem(storageKey).catch(() => {});
      } finally {
        if (active) setReady(true);
      }
    })();

    return () => {
      active = false;
    };
  }, [storageKey]);

  // Clean up the debounce timer on unmount.
  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  const saveDraft = useCallback(
    (data: T) => {
      // Debounce saves to avoid thrashing storage on every keystroke.
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => {
        const envelope: DraftEnvelope<T> = { data, savedAt: Date.now() };
        AsyncStorage.setItem(storageKey, JSON.stringify(envelope)).catch(() => {
          // Storage full or unavailable — silently ignore.
        });
      }, 500);
    },
    [storageKey],
  );

  const clearDraft = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    AsyncStorage.removeItem(storageKey).catch(() => {});
  }, [storageKey]);

  return { ready, draft, saveDraft, clearDraft };
}
