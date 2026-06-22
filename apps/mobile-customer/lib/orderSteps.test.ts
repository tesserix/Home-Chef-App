import {
  getStepLabels,
  getStepIndex,
  getStatusLine,
  getChipLabel,
  isPickupFulfillment,
} from './orderSteps';

describe('orderSteps', () => {
  describe('getStepLabels', () => {
    it('uses delivery steps for delivery / chef_delivery / undefined', () => {
      const expected = ['Confirmed', 'Preparing', 'On the way', 'Delivered'];
      expect(getStepLabels('delivery')).toEqual(expected);
      expect(getStepLabels('chef_delivery')).toEqual(expected);
      expect(getStepLabels(undefined)).toEqual(expected);
    });

    it('uses pickup steps for pickup', () => {
      expect(getStepLabels('pickup')).toEqual([
        'Confirmed',
        'Preparing',
        'Ready for pickup',
        'Collected',
      ]);
    });
  });

  describe('getStepIndex', () => {
    it('keeps delivery at "Preparing" (step 1) when the food is ready', () => {
      expect(getStepIndex('ready', 'delivery')).toBe(1);
    });

    it('advances pickup to "Ready for pickup" (step 2) when ready', () => {
      expect(getStepIndex('ready', 'pickup')).toBe(2);
    });

    it('maps the terminal status to the last step for both modes', () => {
      expect(getStepIndex('delivered', 'delivery')).toBe(3);
      expect(getStepIndex('delivered', 'pickup')).toBe(3);
    });

    it('returns -1 for pre-confirm / terminal-cancelled statuses', () => {
      expect(getStepIndex('pending', 'pickup')).toBe(-1);
      expect(getStepIndex('cancelled', 'delivery')).toBe(-1);
    });
  });

  describe('getStatusLine', () => {
    it('never mentions a driver for pickup', () => {
      expect(getStatusLine('ready', 'pickup')).toBe(
        'Ready for pickup — collect from the chef',
      );
      expect(getStatusLine('delivered', 'pickup')).toBe('Collected');
    });

    it('uses delivery wording for delivery', () => {
      expect(getStatusLine('ready', 'delivery')).toBe(
        'Almost ready — waiting for your driver',
      );
      expect(getStatusLine('delivered', 'delivery')).toBe('Delivered');
    });
  });

  describe('getChipLabel', () => {
    it('relabels the terminal status as Collected for pickup', () => {
      expect(getChipLabel('delivered', 'pickup')).toBe('Collected');
      expect(getChipLabel('delivered', 'delivery')).toBe('Delivered');
    });
  });

  describe('isPickupFulfillment', () => {
    it('is true only for pickup', () => {
      expect(isPickupFulfillment('pickup')).toBe(true);
      expect(isPickupFulfillment('delivery')).toBe(false);
      expect(isPickupFulfillment(undefined)).toBe(false);
    });
  });
});
