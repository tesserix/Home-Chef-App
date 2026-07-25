import { describe, expect, it } from 'vitest';
import { friendlyErrorMessage } from './errors';

// apiClient throws the parsed error body with the HTTP status attached. The
// Go handlers mostly return a flat `{ error: "message" }`, but a few
// endpoints nest it as `{ error: { message: "message" } }` — both must
// resolve to the actual server message, not a blanket fallback, since a
// cancellation's 502 ("cancelled, but the refund could not be issued") means
// something very different from its 409 ("already requested").
describe('friendlyErrorMessage', () => {
  it('reads the flat { error: string } shape the API handlers actually return', () => {
    expect(
      friendlyErrorMessage({
        error: 'Cancelled, but the refund could not be issued — support will follow up',
        status: 502,
      }),
    ).toBe('Cancelled, but the refund could not be issued — support will follow up');
  });

  it('reads the nested { error: { message } } shape some endpoints use', () => {
    expect(
      friendlyErrorMessage({ error: { message: 'A cancellation request already exists' }, status: 409 }),
    ).toBe('A cancellation request already exists');
  });

  it('falls back to a plain Error message when there is no API body', () => {
    expect(friendlyErrorMessage(new Error('network down'))).toBe('network down');
  });

  it('falls back to the provided default when nothing usable is present', () => {
    expect(friendlyErrorMessage({}, 'Could not request cancellation')).toBe(
      'Could not request cancellation',
    );
    expect(friendlyErrorMessage(null, 'Could not request cancellation')).toBe(
      'Could not request cancellation',
    );
  });

  it('uses the generic fallback when none is provided', () => {
    expect(friendlyErrorMessage(undefined)).toBe('Something went wrong. Please try again.');
  });
});
