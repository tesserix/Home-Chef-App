// Browser-only wrapper around Google Identity Services (gsi/client).
// Ported from mark8ly/apps/onboarding/lib/gip/google-gsi.ts.
//
// We do NOT use the Firebase JS SDK for Google sign-in. GSI returns a
// Google-issued credential JWT directly; we exchange it via Identity
// Toolkit signInWithIdp for a GIP id_token in the per-tenant pool.

const SCRIPT_URL = 'https://accounts.google.com/gsi/client';

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID as
  | string
  | undefined;

interface GsiCredentialResponse {
  credential: string;
}

interface GsiNotification {
  isNotDisplayed(): boolean;
  isSkippedMoment(): boolean;
  isDismissedMoment(): boolean;
  getNotDisplayedReason(): string;
  getSkippedReason(): string;
  getDismissedReason(): string;
}

interface GsiAccountsId {
  initialize(opts: {
    client_id: string;
    callback: (resp: GsiCredentialResponse) => void;
    auto_select?: boolean;
    cancel_on_tap_outside?: boolean;
    use_fedcm_for_prompt?: boolean;
  }): void;
  prompt(callback?: (n: GsiNotification) => void): void;
  cancel(): void;
  renderButton(
    parent: HTMLElement,
    options: {
      type?: 'standard' | 'icon';
      theme?: 'outline' | 'filled_blue' | 'filled_black';
      size?: 'large' | 'medium' | 'small';
      text?: 'signin_with' | 'signup_with' | 'continue_with' | 'signin';
      shape?: 'rectangular' | 'pill' | 'circle' | 'square';
      logo_alignment?: 'left' | 'center';
      width?: number | string;
      locale?: string;
    },
  ): void;
}

declare global {
  interface Window {
    google?: {
      accounts: {
        id: GsiAccountsId;
      };
    };
  }
}

let scriptPromise: Promise<void> | null = null;

function loadScript(): Promise<void> {
  if (typeof window === 'undefined') {
    return Promise.reject(
      new Error('google sign-in only available in the browser'),
    );
  }
  if (window.google?.accounts?.id) return Promise.resolve();
  if (scriptPromise) return scriptPromise;

  scriptPromise = new Promise((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>(
      `script[src="${SCRIPT_URL}"]`,
    );
    if (existing) {
      existing.addEventListener('load', () => resolve());
      existing.addEventListener('error', () =>
        reject(new Error('gsi load failed')),
      );
      return;
    }
    const s = document.createElement('script');
    s.src = SCRIPT_URL;
    s.async = true;
    s.defer = true;
    s.onload = () => resolve();
    s.onerror = () => reject(new Error('gsi load failed'));
    document.head.appendChild(s);
  });
  return scriptPromise;
}

export interface MountGoogleButtonOptions {
  buttonContainer: HTMLElement;
  onCredential: (credential: string) => void;
  onError?: (err: Error) => void;
  buttonText?: 'signin_with' | 'signup_with' | 'continue_with' | 'signin';
  /** When true, also fire One-Tap as a passive enhancement; failures are silent. */
  tryOneTap?: boolean;
  /** Pixel width for the rendered button. Default 280. */
  width?: number;
}

/**
 * Render the official "Sign in with Google" button into `buttonContainer`
 * and invoke `onCredential` with the Google id_token JWT once the user
 * completes the FedCM modal. Use this on pages where the user is expected
 * to take a deliberate action — One-Tap alone is unreliable because
 * FedCM-era browsers may silently return "unknown_reason".
 *
 * Pass `tryOneTap: true` to also fire One-Tap as a convenience.
 * Returns a teardown function that cancels both flows.
 */
export async function mountGoogleButton(
  opts: MountGoogleButtonOptions,
): Promise<() => void> {
  if (!GOOGLE_CLIENT_ID) {
    throw new Error(
      'Google sign-in is not configured (VITE_GOOGLE_CLIENT_ID missing)',
    );
  }
  await loadScript();
  const gsi = window.google!.accounts.id;

  let cancelled = false;

  gsi.initialize({
    client_id: GOOGLE_CLIENT_ID,
    callback: (resp) => {
      if (cancelled) return;
      if (resp?.credential) {
        opts.onCredential(resp.credential);
      } else {
        opts.onError?.(new Error('google: empty credential'));
      }
    },
    auto_select: false,
    cancel_on_tap_outside: true,
    use_fedcm_for_prompt: true,
  });

  gsi.renderButton(opts.buttonContainer, {
    type: 'standard',
    theme: 'outline',
    size: 'large',
    text: opts.buttonText ?? 'continue_with',
    shape: 'pill',
    logo_alignment: 'left',
    width: opts.width ?? 280,
  });

  if (opts.tryOneTap) {
    try {
      gsi.prompt();
    } catch {
      // ignore — button is the primary path
    }
  }

  return () => {
    cancelled = true;
    try {
      gsi.cancel();
    } catch {
      // ignore
    }
  };
}
