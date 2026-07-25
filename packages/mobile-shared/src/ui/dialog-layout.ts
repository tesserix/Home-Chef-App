// The action-layout rules for <Dialog>, kept apart from the component so they
// can be tested without a renderer. A three-action dialog silently overflowing
// its card is exactly the kind of regression a visual pass misses, and nothing
// here needs React Native to decide.

export interface DialogAction {
  label: string;
  onPress?: () => void;
  /** Renders in the destructive colour. For the action that cannot be undone. */
  destructive?: boolean;
  /** The quiet, dismissing choice. */
  cancel?: boolean;
}

/** Labels beyond this, summed, cannot share one row on a phone. */
const ROW_LABEL_BUDGET = 24;

export interface DialogLayout {
  /** Actions run down the card rather than across it. */
  stacked: boolean;
  /** Render order for the chosen layout. */
  ordered: DialogAction[];
  /** The action to paint in the accent, if any. */
  primary: DialogAction | undefined;
}

/** Decide how a dialog's actions lay out. */
export function resolveDialogLayout(actions: DialogAction[]): DialogLayout {
  // Two short labels sit side by side; three, or two long ones, do not — they
  // collide or truncate on a phone. Stack in that case, which is what both
  // platform dialogs do at roughly the same threshold.
  const labelChars = actions.reduce((n, a) => n + a.label.length, 0);
  const stacked = actions.length > 2 || labelChars > ROW_LABEL_BUDGET;

  const cancels = actions.filter((a) => a.cancel);
  const rest = actions.filter((a) => !a.cancel);

  // Row: the way out reads first, so the eye passes it before the irreversible
  // choice — the ordering the platform alerts use. Stacked: it sits last,
  // nearest the thumb, which is the convention for a vertical list and puts the
  // safe target in the easiest place to reach.
  const ordered = stacked ? [...rest, ...cancels] : [...cancels, ...rest];

  // Accent the confirming action only in the classic one-way-out dialog. When
  // the actions are peer choices ("personal group" / "office"), accenting an
  // arbitrary one of them invents a recommendation the caller never made — and
  // .impeccable.md allows exactly one accent competing for the eye.
  const primary = rest.length === 1 && !rest[0]!.destructive ? rest[0] : undefined;

  return { stacked, ordered, primary };
}
