import { describe, it, expect } from 'vitest';
import { formatRating } from './format-rating';

// The API returns an unrounded average — 7 five-star reviews and one four-star
// prints as 4.857142857142857 next to the chef's name, which reads as broken.
describe('formatRating', () => {
  it('rounds a raw average to one decimal', () => {
    expect(formatRating(4.857142857142857)).toBe('4.9');
  });

  it('keeps a whole number readable as a rating', () => {
    expect(formatRating(5)).toBe('5.0');
  });

  it('has nothing to show for a chef with no reviews yet', () => {
    expect(formatRating(0)).toBe('—');
    expect(formatRating(null)).toBe('—');
    expect(formatRating(undefined)).toBe('—');
    expect(formatRating(Number.NaN)).toBe('—');
  });
});
