import { useCallback, useState } from 'react';

import { Dialog, type DialogAction } from './Dialog';

// useDialog — an imperative wrapper so migrating off Alert.alert is a small,
// mechanical edit rather than restructuring a screen around new state.
//
// Alert.alert is a function call in the middle of a handler. A declarative
// <Dialog visible={...}> would force every call site to hoist its own visible
// flag, its own copy, and its own action list into component state — which is
// how a 20-line handler becomes 60 and how a mechanical migration turns into a
// rewrite. This keeps the call shape:
//
//   const dialog = useDialog();
//   ...
//   dialog.confirm({
//     title: 'Cancel this plan?',
//     message: "You haven't been served yet, so you'll be fully refunded.",
//     actions: [
//       { label: 'Keep plan', cancel: true },
//       { label: 'Cancel plan', destructive: true, onPress: doCancel },
//     ],
//   });
//   ...
//   return <>{...}{dialog.element}</>;

export interface ConfirmOptions {
  title: string;
  message?: string;
  actions: DialogAction[];
  /** Accent for the primary action — the app's own, since the shared palette
   *  and the customer palette differ. */
  accentColor?: string;
  /** Set false for a decision the user must actually make: no backdrop tap,
   *  no Android back. Defaults to dismissible. */
  dismissible?: boolean;
}

export function useDialog() {
  const [options, setOptions] = useState<ConfirmOptions | null>(null);

  const close = useCallback(() => setOptions(null), []);

  const confirm = useCallback((next: ConfirmOptions) => setOptions(next), []);

  // Every action closes the dialog FIRST, then runs its handler. Handlers
  // routinely navigate or open another dialog, and leaving this one mounted
  // while that happens strands a modal over the new screen.
  const actions: DialogAction[] = (options?.actions ?? []).map((a) => ({
    ...a,
    onPress: () => {
      close();
      a.onPress?.();
    },
  }));

  const element = (
    <Dialog
      visible={options !== null}
      title={options?.title ?? ''}
      message={options?.message}
      actions={actions}
      accentColor={options?.accentColor}
      onDismiss={options?.dismissible === false ? undefined : close}
    />
  );

  return { confirm, close, element };
}
