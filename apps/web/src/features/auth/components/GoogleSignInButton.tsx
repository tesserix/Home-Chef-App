import { useEffect, useRef, useState, type ReactElement } from 'react';
import { mountGoogleButton } from '@/lib/gip/google-gsi';
import {
  signInWithGoogleCredential,
  GIPError,
} from '@/lib/gip/customer-signin';
import {
  postExchange,
  toSessionUser,
} from '@/features/auth/services/auth-service';
import type { AuthSession } from '@/features/auth/services/auth-service';
import { useAuthStore } from '@/store/auth-store';

interface GoogleSignInButtonProps {
  onSuccess: (session: AuthSession) => void;
  onError?: (message: string) => void;
  /** Width in pixels for the rendered GSI button. Default 280. */
  width?: number;
  /** GSI button label. Default "continue_with". */
  buttonText?: 'signin_with' | 'signup_with' | 'continue_with' | 'signin';
}

/**
 * Renders the official Google Identity Services button inline. On click,
 * Google's FedCM modal collects the user's consent, returns a Google
 * credential JWT, which we exchange (via Identity Toolkit signInWithIdp)
 * for a GIP id_token in the configured tenant pool, then POST to the
 * BFF /auth/exchange endpoint to mint the session cookie.
 *
 * Browser-only. Avoids both signInWithPopup (COOP-incompatible) and the
 * default Firebase auth handler (which exposes <projectId>.firebaseapp.com).
 */
export function GoogleSignInButton({
  onSuccess,
  onError,
  width,
  buttonText,
}: GoogleSignInButtonProps): ReactElement {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [status, setStatus] = useState<'idle' | 'ready' | 'exchanging'>(
    'idle',
  );

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    let cancelled = false;
    let teardown: (() => void) | null = null;

    void (async () => {
      try {
        const cleanup = await mountGoogleButton({
          buttonContainer: container,
          buttonText,
          width,
          tryOneTap: true,
          onCredential: (googleCred) => {
            if (cancelled) return;
            void handleCredential(googleCred);
          },
          onError: (err) => {
            if (cancelled) return;
            onError?.(err.message);
          },
        });
        if (cancelled) {
          cleanup();
          return;
        }
        teardown = cleanup;
        setStatus('ready');
      } catch (err) {
        if (cancelled) return;
        onError?.(err instanceof Error ? err.message : 'Sign-in unavailable');
      }
    })();

    async function handleCredential(googleCred: string): Promise<void> {
      setStatus('exchanging');
      try {
        const gip = await signInWithGoogleCredential(googleCred);
        if (gip.kind === 'needConfirmation') {
          onError?.(
            'This email already has an account. Sign in with your password to link Google.',
          );
          setStatus('ready');
          return;
        }
        const session = await postExchange(gip.idToken);
        // Sync the global auth store before navigating so the header,
        // route guards, and protected pages see the user as signed in.
        // Mirrors the wiring done in AuthProvider.login() for other providers.
        useAuthStore.getState().setApiAuth(toSessionUser(session), '', '');
        onSuccess(session);
      } catch (err) {
        if (err instanceof GIPError) {
          onError?.(
            err.code === 'config_missing'
              ? 'Google sign-in is not available right now.'
              : 'Google sign-in failed. Please try again.',
          );
        } else {
          onError?.(
            err instanceof Error ? err.message : 'Google sign-in failed.',
          );
        }
        setStatus('ready');
      }
    }

    return () => {
      cancelled = true;
      teardown?.();
    };
  }, [buttonText, onError, onSuccess, width]);

  return (
    <div className="flex justify-center">
      <div ref={containerRef} aria-busy={status === 'exchanging'} />
    </div>
  );
}
