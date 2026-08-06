// The shared auth package spoke @react-native-firebase's namespaced API —
// auth().signInWithEmailAndPassword(), auth().onAuthStateChanged(). Every one of
// those calls logs a deprecation warning at runtime (v22 moves to the Firebase
// Web modular SDK shape), and they drowned the console: 16 warnings on a cold
// start, hiding real errors behind them. The namespaced default export must not
// be called at all.

import { describe, it, expect, vi, beforeEach } from 'vitest';

// Hoisted: vi.mock's factory runs before module-level consts initialise.
const {
  namespaced,
  mockGetAuth,
  mockSignIn,
  mockSignOut,
  mockOnAuthStateChanged,
  mockSendPasswordResetEmail,
} = vi.hoisted(() => ({
  namespaced: vi.fn(() => {
    throw new Error('namespaced auth() called — use the modular API');
  }),
  mockGetAuth: vi.fn(() => ({ tenantId: 'tenant-1', currentUser: null })),
  mockSignIn: vi.fn(async () => ({ user: { displayName: 'Anita' } })),
  mockSignOut: vi.fn(async () => {}),
  mockOnAuthStateChanged: vi.fn(() => () => {}),
  mockSendPasswordResetEmail: vi.fn(async () => {}),
}));

vi.mock('@react-native-firebase/auth', () => ({
  default: Object.assign(namespaced, {
    GoogleAuthProvider: { credential: vi.fn() },
    AppleAuthProvider: { credential: vi.fn() },
  }),
  getAuth: () => mockGetAuth(),
  signInWithEmailAndPassword: (...a: unknown[]) => mockSignIn(...(a as [])),
  createUserWithEmailAndPassword: (...a: unknown[]) => mockSignIn(...(a as [])),
  signInWithCredential: (...a: unknown[]) => mockSignIn(...(a as [])),
  signOut: (...a: unknown[]) => mockSignOut(...(a as [])),
  onAuthStateChanged: (...a: unknown[]) => mockOnAuthStateChanged(...(a as [])),
  sendPasswordResetEmail: (...a: unknown[]) => mockSendPasswordResetEmail(...(a as [])),
  verifyPhoneNumber: vi.fn(),
  updateProfile: vi.fn(async () => {}),
  getIdToken: vi.fn(async () => 'id-token'),
  GoogleAuthProvider: { credential: vi.fn() },
  AppleAuthProvider: { credential: vi.fn() },
}));

vi.mock('expo-constants', () => ({
  default: { expoConfig: { ios: { bundleIdentifier: 'com.tesserix.homechef.customer' } } },
}));

import {
  signInWithEmail,
  registerWithEmail,
  signOut,
  sendPasswordResetEmail,
  getLinkedProviderIds,
} from '../auth/sign-in';

beforeEach(() => {
  namespaced.mockClear();
  mockSignIn.mockClear();
  mockSignOut.mockClear();
  mockSendPasswordResetEmail.mockClear();
});

describe('shared auth speaks the modular Firebase API', () => {
  it('signs in through signInWithEmailAndPassword(auth, …)', async () => {
    await signInWithEmail('anita@example.com', 'hunter2');

    expect(namespaced).not.toHaveBeenCalled();
    expect(mockSignIn).toHaveBeenCalledWith(
      expect.objectContaining({ tenantId: 'tenant-1' }),
      'anita@example.com',
      'hunter2',
    );
  });

  it('registers through the modular call', async () => {
    await registerWithEmail('anita@example.com', 'hunter2');

    expect(namespaced).not.toHaveBeenCalled();
    expect(mockSignIn).toHaveBeenCalled();
  });

  it('signs out through signOut(auth)', async () => {
    await signOut();

    expect(namespaced).not.toHaveBeenCalled();
    expect(mockSignOut).toHaveBeenCalledWith(expect.objectContaining({ tenantId: 'tenant-1' }));
  });

  it('sends the reset email through the modular call', async () => {
    await sendPasswordResetEmail('anita@example.com');

    expect(namespaced).not.toHaveBeenCalled();
    expect(mockSendPasswordResetEmail).toHaveBeenCalledWith(
      expect.objectContaining({ tenantId: 'tenant-1' }),
      'anita@example.com',
    );
  });

  it('reads linked providers off the modular auth instance', () => {
    expect(getLinkedProviderIds()).toEqual([]);
    expect(namespaced).not.toHaveBeenCalled();
  });
});
