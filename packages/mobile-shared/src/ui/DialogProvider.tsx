import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';

import { Dialog, type DialogAction } from './Dialog';

// DialogProvider — one mounted Dialog for the whole app, driven imperatively.
//
// useDialog() covers a screen that owns its own confirmation, but it cannot
// serve a HOOK: a hook has no render tree, so it has nowhere to mount
// dialog.element. useSkipDayFlow and friends call Alert.alert today for exactly
// that reason, and they are the sites a screen-local API can never reach.
//
// It also makes migration cheap. showAlert keeps Alert.alert's signature
// exactly, so converting a call site is an import swap — no visibility flag to
// hoist, no element to mount, no render tree to touch. Across ~195 call sites
// that difference is what separates a mechanical change from a rewrite.
//
//   const { showAlert } = useAlert();
//   showAlert('Cancel this plan?', "You'll be fully refunded.", [
//     { text: 'Keep plan', style: 'cancel' },
//     { text: 'Cancel plan', style: 'destructive', onPress: doCancel },
//   ]);

/** Mirrors React Native's AlertButton so call sites port unchanged. */
export interface AlertButton {
  text?: string;
  onPress?: () => void;
  style?: 'default' | 'cancel' | 'destructive';
}

interface AlertRequest {
  title: string;
  message?: string;
  buttons?: AlertButton[];
}

interface AlertContextValue {
  /** Same shape as Alert.alert. Omitting buttons yields a single OK. */
  showAlert: (title: string, message?: string, buttons?: AlertButton[]) => void;
}

const AlertContext = createContext<AlertContextValue | null>(null);

export function DialogProvider({
  children,
  accentColor,
}: {
  children: ReactNode;
  /** The app's accent — shared palette is persimmon, customer runs coral. */
  accentColor?: string;
}) {
  // A queue, not a single slot: a handler that opens another alert (the common
  // "did it work?" follow-up) would otherwise overwrite the one still on screen
  // and the second would never be seen.
  const [queue, setQueue] = useState<AlertRequest[]>([]);
  const current = queue[0];

  const showAlert = useCallback((title: string, message?: string, buttons?: AlertButton[]) => {
    setQueue((q) => [...q, { title, message, buttons }]);
  }, []);

  const dismiss = useCallback(() => setQueue((q) => q.slice(1)), []);

  const actions: DialogAction[] = useMemo(() => {
    const buttons = current?.buttons?.length
      ? current.buttons
      : [{ text: 'OK' } as AlertButton];
    return buttons.map((b) => ({
      label: b.text ?? 'OK',
      destructive: b.style === 'destructive',
      cancel: b.style === 'cancel',
      onPress: () => {
        // Advance the queue FIRST: handlers routinely navigate or raise the next
        // alert, and doing that under a still-mounted modal strands it.
        dismiss();
        b.onPress?.();
      },
    }));
  }, [current, dismiss]);

  const value = useMemo(() => ({ showAlert }), [showAlert]);

  return (
    <AlertContext.Provider value={value}>
      {children}
      <Dialog
        visible={current !== undefined}
        title={current?.title ?? ''}
        message={current?.message}
        actions={actions}
        accentColor={accentColor}
        onDismiss={dismiss}
      />
    </AlertContext.Provider>
  );
}

/**
 * Branded replacement for Alert.alert.
 *
 * Falls back to a no-op-safe error if used outside the provider, rather than
 * silently swallowing a confirmation the user needed to see.
 */
export function useAlert(): AlertContextValue {
  const ctx = useContext(AlertContext);
  if (!ctx) {
    throw new Error('useAlert must be used within <DialogProvider>');
  }
  return ctx;
}
