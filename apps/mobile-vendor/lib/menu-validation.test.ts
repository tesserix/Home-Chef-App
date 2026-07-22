import { describe, it, expect } from '@jest/globals';

import { validationSummary, firstInvalidField, type MenuItemErrors } from './menu-validation';

// Why this exists: tapping "Add item" with an unselected category did nothing
// at all — no toast, no scroll, no movement. The only signal was an inline
// "Select a category" message sitting off-screen further up a long form, so
// the button read as broken. Three save attempts were lost to it before the
// cause was found by dumping the view hierarchy.

describe('validationSummary', () => {
  it('is silent when the form is valid', () => {
    expect(validationSummary({})).toBeNull();
  });

  it('names the single problem so the chef knows what to fix', () => {
    expect(validationSummary({ categoryId: 'Select a category' })).toBe('Select a category');
  });

  it('leads with the first problem in form order, not object order', () => {
    // categoryId sits last on the form; name is first. Whatever order the
    // errors object happens to enumerate in, the chef should be sent to the
    // topmost field.
    const errors: MenuItemErrors = {
      categoryId: 'Select a category',
      name: 'Name must be at least 3 characters',
    };
    expect(validationSummary(errors)).toContain('Name must be at least 3 characters');
  });

  it('says how many fields need attention when there is more than one', () => {
    const msg = validationSummary({
      name: 'Name must be at least 3 characters',
      price: 'Price must be between ₹1 and ₹10,000',
      categoryId: 'Select a category',
    });
    expect(msg).toContain('3');
    expect(msg).toContain('Name must be at least 3 characters');
  });

  it('ignores keys explicitly cleared to undefined', () => {
    // Fields clear their own error as the chef types, leaving undefined behind.
    expect(validationSummary({ name: undefined, description: undefined })).toBeNull();
  });
});

describe('firstInvalidField', () => {
  it('reports the topmost invalid field for scrolling', () => {
    expect(firstInvalidField({ categoryId: 'x', price: 'y' })).toBe('price');
    expect(firstInvalidField({ categoryId: 'x' })).toBe('categoryId');
    expect(firstInvalidField({})).toBeNull();
  });
});
