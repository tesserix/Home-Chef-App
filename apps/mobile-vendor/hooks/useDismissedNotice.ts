import { useCallback, useEffect, useState } from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';

// Dismissing a dashboard card is per-STATE, not per-card.
//
// Keyed by what the card is currently saying, so clearing "we're preparing your
// form" hides that and nothing else: when the request moves to filed, the key
// changes and the chef sees the new state. A card dismissed by id alone would go
// quiet for the rest of the request's life, which is the opposite of the point.
const KEY_PREFIX = 'dismissed-notice:';

export function useDismissedNotice(noticeKey: string | null): {
  dismissed: boolean;
  dismiss: () => void;
} {
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    let live = true;
    if (!noticeKey) {
      setDismissed(false);
      return;
    }
    // Assume NOT dismissed until storage says otherwise: a card that flickers in
    // is better than one that silently never appears if the read fails.
    void AsyncStorage.getItem(KEY_PREFIX + noticeKey)
      .then((v) => {
        if (live) setDismissed(v === '1');
      })
      .catch(() => {
        if (live) setDismissed(false);
      });
    return () => {
      live = false;
    };
  }, [noticeKey]);

  const dismiss = useCallback(() => {
    if (!noticeKey) return;
    setDismissed(true); // optimistic — the card goes on the tap, not on the write
    void AsyncStorage.setItem(KEY_PREFIX + noticeKey, '1').catch(() => {});
  }, [noticeKey]);

  return { dismissed, dismiss };
}
