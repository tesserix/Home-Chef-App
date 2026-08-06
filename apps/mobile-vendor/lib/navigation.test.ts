import { describe, it, expect, jest } from '@jest/globals';
import { leaveTo } from './navigation';

// A screen reached from a push notification or a deep link has no back stack,
// so router.back() logs "GO_BACK was not handled" and the chef stays on the
// form they just saved — reading as a failed save, and inviting a second one.
describe('leaveTo', () => {
  const router = (canGoBack: boolean) => ({
    canGoBack: () => canGoBack,
    back: jest.fn(),
    replace: jest.fn(),
  });

  it('goes back when there is somewhere to go back to', () => {
    const r = router(true);
    leaveTo(r, '/(tabs)/menu');
    expect(r.back).toHaveBeenCalled();
    expect(r.replace).not.toHaveBeenCalled();
  });

  it('replaces with the fallback when the stack is empty', () => {
    const r = router(false);
    leaveTo(r, '/(tabs)/menu');
    expect(r.replace).toHaveBeenCalledWith('/(tabs)/menu');
    expect(r.back).not.toHaveBeenCalled();
  });
});
