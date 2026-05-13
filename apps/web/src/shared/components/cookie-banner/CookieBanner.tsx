import { useCallback, useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { Cookie, X } from 'lucide-react';

const STORAGE_KEY = 'cookie-consent';

interface CookieConsent {
  necessary: true;
  functional: boolean;
  analytics: boolean;
  marketing: boolean;
  decidedAt: string;
}

interface CookieCategoryProps {
  id: string;
  title: string;
  description: string;
  enabled: boolean;
  onChange?: (value: boolean) => void;
  disabled?: boolean;
}

function CookieCategory({
  id,
  title,
  description,
  enabled,
  onChange,
  disabled,
}: CookieCategoryProps) {
  const labelId = `${id}-label`;
  const descId = `${id}-desc`;
  return (
    <label
      htmlFor={id}
      className={`flex items-start justify-between gap-4 rounded-md border border-mist bg-bone p-3 ${
        disabled ? 'opacity-60' : 'cursor-pointer'
      }`}
    >
      <div className="space-y-1 text-sm">
        <div id={labelId} className="font-medium text-ink">
          {title}
        </div>
        <div id={descId} className="text-ink-soft">
          {description}
        </div>
      </div>
      <input
        id={id}
        type="checkbox"
        checked={enabled}
        disabled={disabled}
        onChange={(e) => onChange?.(e.target.checked)}
        aria-labelledby={labelId}
        aria-describedby={descId}
        className="mt-1 h-4 w-4 rounded border-mist text-herb focus:ring-2 focus:ring-herb/40 focus:ring-offset-2 focus:ring-offset-paper"
      />
    </label>
  );
}

export function CookieBanner() {
  const [visible, setVisible] = useState(false);
  const [showCustomize, setShowCustomize] = useState(false);
  const [functional, setFunctional] = useState(false);
  const [analytics, setAnalytics] = useState(false);
  const [marketing, setMarketing] = useState(false);
  const customizeBtnRef = useRef<HTMLButtonElement>(null);
  const dialogRef = useRef<HTMLDivElement>(null);
  const closeBtnRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    // Reading storage at mount only; we never trust external data without
    // a try/catch since private mode / disabled storage will throw.
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (!stored) {
        setVisible(true);
      }
    } catch {
      setVisible(true);
    }
  }, []);

  // Esc closes the customize panel (returning to the compact view), or
  // returns focus to the trigger if already in compact view.
  useEffect(() => {
    if (!visible) return;
    const onKey = (e: KeyboardEvent): void => {
      if (e.key !== 'Escape') return;
      if (showCustomize) {
        setShowCustomize(false);
        customizeBtnRef.current?.focus();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [visible, showCustomize]);

  // When the customize panel opens, send focus into it for screen readers.
  useEffect(() => {
    if (showCustomize) {
      closeBtnRef.current?.focus();
    }
  }, [showCustomize]);

  const persist = useCallback((consent: CookieConsent): void => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(consent));
    } catch {
      // Storage unavailable — decision will not persist across sessions.
      // We still hide the banner for the current session.
    }
    setVisible(false);
  }, []);

  const acceptAll = useCallback((): void => {
    persist({
      necessary: true,
      functional: true,
      analytics: true,
      marketing: true,
      decidedAt: new Date().toISOString(),
    });
  }, [persist]);

  const rejectNonEssential = useCallback((): void => {
    persist({
      necessary: true,
      functional: false,
      analytics: false,
      marketing: false,
      decidedAt: new Date().toISOString(),
    });
  }, [persist]);

  const savePreferences = useCallback((): void => {
    persist({
      necessary: true,
      functional,
      analytics,
      marketing,
      decidedAt: new Date().toISOString(),
    });
  }, [persist, functional, analytics, marketing]);

  if (!visible) return null;

  return (
    <div
      ref={dialogRef}
      role="dialog"
      aria-modal="false"
      aria-labelledby="cookie-banner-title"
      aria-describedby="cookie-banner-body"
      className="fixed inset-x-0 bottom-0 z-50 border-t border-mist bg-paper shadow-3"
    >
      <div className="container-app mx-auto px-4 py-4 sm:py-5">
        {!showCustomize ? (
          // Compact view
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-start gap-3">
              <Cookie aria-hidden="true" className="mt-0.5 h-5 w-5 flex-shrink-0 text-herb" />
              <div className="space-y-1 text-sm">
                <h2 id="cookie-banner-title" className="font-medium text-ink">
                  Cookies on Home Chef
                </h2>
                <p id="cookie-banner-body" className="text-ink-soft">
                  We use strictly necessary cookies to keep you signed in and your cart
                  working. With your permission, we can also use functional cookies for
                  preferences. We don't currently use analytics or marketing cookies. See
                  our{' '}
                  <Link
                    to="/cookies"
                    className="text-herb underline-offset-2 hover:underline focus-visible:outline-none focus-visible:underline"
                  >
                    Cookie Policy
                  </Link>
                  .
                </p>
              </div>
            </div>
            <div className="flex flex-col gap-2 sm:flex-row sm:flex-shrink-0 sm:items-center">
              <button
                ref={customizeBtnRef}
                type="button"
                onClick={() => setShowCustomize(true)}
                className="order-3 rounded-md px-2 py-2 text-sm text-herb hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40 sm:order-1"
              >
                Customize
              </button>
              <button
                type="button"
                onClick={rejectNonEssential}
                className="order-2 rounded-md border border-ink/20 px-4 py-2 text-sm font-medium text-ink hover:bg-bone focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40"
              >
                Reject non-essential
              </button>
              <button
                type="button"
                onClick={acceptAll}
                className="order-1 rounded-md bg-herb px-4 py-2 text-sm font-medium text-paper hover:bg-herb-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40 sm:order-3"
              >
                Accept all
              </button>
            </div>
          </div>
        ) : (
          // Customize panel
          <div className="space-y-4">
            <div className="flex items-start justify-between gap-3">
              <div>
                <h2 id="cookie-banner-title" className="text-base font-medium text-ink">
                  Cookie preferences
                </h2>
                <p id="cookie-banner-body" className="text-sm text-ink-soft">
                  Choose which cookies we can use.
                </p>
              </div>
              <button
                ref={closeBtnRef}
                type="button"
                onClick={() => setShowCustomize(false)}
                aria-label="Close customize panel"
                className="rounded-md p-1 text-ink-muted hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40"
              >
                <X aria-hidden="true" className="h-5 w-5" />
              </button>
            </div>

            <div className="space-y-3">
              <CookieCategory
                id="cookie-cat-necessary"
                title="Strictly necessary"
                description="Keep you signed in, secure your session, and remember your cart. These can't be disabled."
                enabled
                disabled
              />
              <CookieCategory
                id="cookie-cat-functional"
                title="Functional"
                description="Remember preferences like language and currency."
                enabled={functional}
                onChange={setFunctional}
              />
              <CookieCategory
                id="cookie-cat-analytics"
                title="Analytics"
                description="Help us understand how the site is used. We don't currently use analytics cookies."
                enabled={analytics}
                onChange={setAnalytics}
              />
              <CookieCategory
                id="cookie-cat-marketing"
                title="Marketing"
                description="Personalize ads. We don't currently use marketing cookies."
                enabled={marketing}
                onChange={setMarketing}
              />
            </div>

            <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
              <button
                type="button"
                onClick={() => setShowCustomize(false)}
                className="rounded-md px-4 py-2 text-sm text-ink-soft hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={savePreferences}
                className="rounded-md bg-herb px-4 py-2 text-sm font-medium text-paper hover:bg-herb-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/40"
              >
                Save preferences
              </button>
            </div>

            <p className="text-xs text-ink-muted">
              You can change these any time via the{' '}
              <Link
                to="/cookies"
                className="text-herb underline-offset-2 hover:underline focus-visible:outline-none focus-visible:underline"
              >
                Cookie Policy
              </Link>{' '}
              page.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
