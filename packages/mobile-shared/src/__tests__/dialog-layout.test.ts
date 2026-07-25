import { describe, expect, it } from 'vitest';

import { resolveDialogLayout, type DialogAction } from '../ui/dialog-layout';

const labels = (actions: DialogAction[]) => actions.map((a) => a.label);

describe('resolveDialogLayout', () => {
  describe('row vs stacked', () => {
    it('keeps two short actions on one row', () => {
      const { stacked } = resolveDialogLayout([
        { label: 'Cancel', cancel: true },
        { label: 'Delete' },
      ]);
      expect(stacked).toBe(false);
    });

    it('stacks three actions', () => {
      // The group-order picker: three labels can never share a phone row.
      const { stacked } = resolveDialogLayout([
        { label: 'Cancel', cancel: true },
        { label: 'Office / corporate' },
        { label: 'Personal group' },
      ]);
      expect(stacked).toBe(true);
    });

    it('stacks two actions whose labels are too long to share a row', () => {
      const { stacked } = resolveDialogLayout([
        { label: 'Keep my subscription', cancel: true },
        { label: 'Cancel subscription', destructive: true },
      ]);
      expect(stacked).toBe(true);
    });

    it('keeps a lone OK on one row', () => {
      const { stacked, ordered } = resolveDialogLayout([{ label: 'OK' }]);
      expect(stacked).toBe(false);
      expect(labels(ordered)).toEqual(['OK']);
    });
  });

  describe('ordering', () => {
    it('puts the cancel first in a row, so the eye passes it first', () => {
      const { ordered } = resolveDialogLayout([
        { label: 'Delete' },
        { label: 'Cancel', cancel: true },
      ]);
      expect(labels(ordered)).toEqual(['Cancel', 'Delete']);
    });

    it('puts the cancel last when stacked, nearest the thumb', () => {
      const { ordered } = resolveDialogLayout([
        { label: 'Cancel', cancel: true },
        { label: 'Office / corporate' },
        { label: 'Personal group' },
      ]);
      expect(labels(ordered)).toEqual(['Office / corporate', 'Personal group', 'Cancel']);
    });

    it('preserves the caller order of the non-cancel actions', () => {
      const { ordered } = resolveDialogLayout([
        { label: 'Full refund' },
        { label: 'Half refund' },
        { label: 'No refund' },
        { label: 'Decline', cancel: true },
      ]);
      expect(labels(ordered)).toEqual([
        'Full refund',
        'Half refund',
        'No refund',
        'Decline',
      ]);
    });

    it('never drops or duplicates an action', () => {
      const actions: DialogAction[] = [
        { label: 'A', cancel: true },
        { label: 'B' },
        { label: 'C', destructive: true },
      ];
      const { ordered } = resolveDialogLayout(actions);
      expect(ordered).toHaveLength(3);
      expect(new Set(labels(ordered))).toEqual(new Set(['A', 'B', 'C']));
    });
  });

  describe('accent', () => {
    it('accents the single confirming action', () => {
      const { primary } = resolveDialogLayout([
        { label: 'Cancel', cancel: true },
        { label: 'Save' },
      ]);
      expect(primary?.label).toBe('Save');
    });

    it('accents nothing when the actions are peer choices', () => {
      // Painting one of two equal options in the brand accent would invent a
      // recommendation the caller never made.
      const { primary } = resolveDialogLayout([
        { label: 'Cancel', cancel: true },
        { label: 'Office / corporate' },
        { label: 'Personal group' },
      ]);
      expect(primary).toBeUndefined();
    });

    it('leaves a destructive action to its own colour', () => {
      const { primary } = resolveDialogLayout([
        { label: 'Keep', cancel: true },
        { label: 'Delete', destructive: true },
      ]);
      expect(primary).toBeUndefined();
    });
  });
});
