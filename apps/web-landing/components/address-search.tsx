'use client';

import { useEffect, useId, useRef, useState } from 'react';
import { BROWSE_PATH, APP_LOGIN_PATH } from '@/lib/site';

/**
 * AddressSearch — the landing's primary call to action.
 *
 * The hero used to lead with two "coming soon" app-store badges and bury web
 * ordering in a text link. So the single loudest thing on the page was a button
 * that cannot be pressed, while the one path a visitor CAN take today was the
 * quietest element in the block. This is the fix: type where you are, get real
 * suggestions, land in the kitchens that actually deliver to you.
 *
 * The suggestion API (GET /api/v1/locations/autocomplete) is genuinely public —
 * it sits outside every auth group in routes.go — and returns lat/lon per
 * result. That matters: the browse page filters on ?lat&lng, so a picked
 * suggestion hands off precise coordinates rather than a string the SPA would
 * have to re-resolve.
 *
 * Same-origin by design: fe3dr.com serves this landing and proxies /api to the
 * Go API (see the Istio VirtualService), so there is no CORS surface and no
 * base-URL config to drift.
 */

interface Suggestion {
  description: string;
  line1: string;
  city: string;
  region: string;
  postal: string;
  country: string;
  lat: number;
  lon: number;
}

/** Where a resolved location sends the visitor. */
function browseUrl(lat: number, lng: number): string {
  // `sort=distance` so the first thing they see is the kitchen closest to them,
  // which is the whole promise of "in your neighbourhood".
  return `${BROWSE_PATH}?lat=${lat}&lng=${lng}&sort=distance`;
}

export function AddressSearch() {
  const [query, setQuery] = useState('');
  const [items, setItems] = useState<Suggestion[]>([]);
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const [locating, setLocating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const listId = useId();
  const rootRef = useRef<HTMLDivElement>(null);

  // Debounced lookup. 250ms is the point where a typist stops out-running the
  // list without it feeling laggy; below ~3 characters the results are noise.
  useEffect(() => {
    const q = query.trim();
    if (q.length < 3) {
      setItems([]);
      return;
    }
    const ctl = new AbortController();
    const t = setTimeout(() => {
      fetch(`/api/v1/locations/autocomplete?q=${encodeURIComponent(q)}`, {
        signal: ctl.signal,
      })
        .then((r) => (r.ok ? r.json() : Promise.reject(new Error(String(r.status)))))
        .then((body: { data?: Suggestion[] }) => {
          // Only results carrying coordinates are useful — a suggestion we
          // cannot turn into lat/lng would dead-end at the browse page.
          const withCoords = (body.data ?? []).filter((s) => s.lat && s.lon);
          setItems(withCoords.slice(0, 6));
          setOpen(withCoords.length > 0);
          setActive(-1);
        })
        .catch((e) => {
          // An aborted request is the normal consequence of typing, not a fault.
          if ((e as Error).name !== 'AbortError') setItems([]);
        });
    }, 250);
    return () => {
      clearTimeout(t);
      ctl.abort();
    };
  }, [query]);

  // Close on outside click — a listbox that survives a click elsewhere on the
  // page reads as stuck.
  useEffect(() => {
    function onDown(e: MouseEvent) {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener('mousedown', onDown);
    return () => document.removeEventListener('mousedown', onDown);
  }, []);

  function choose(s: Suggestion) {
    window.location.href = browseUrl(s.lat, s.lon);
  }

  function submit() {
    // Enter with a highlighted row takes it; otherwise take the first result.
    // Falling back to "the best match" beats refusing to move: the visitor has
    // told us where they are, and making them click again is friction for
    // nothing.
    const pick = items[active] ?? items[0];
    if (pick) {
      choose(pick);
      return;
    }
    setError('Pick a suggestion so we can find kitchens near you.');
  }

  function useMyLocation() {
    if (!navigator.geolocation) {
      setError('Your browser cannot share a location — type your area instead.');
      return;
    }
    setLocating(true);
    setError(null);
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        window.location.href = browseUrl(pos.coords.latitude, pos.coords.longitude);
      },
      () => {
        // Denied or timed out. Never a dead end — the typed path still works.
        setLocating(false);
        setError('We could not get your location. Type your area instead.');
      },
      { enableHighAccuracy: false, timeout: 8000, maximumAge: 300_000 },
    );
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'ArrowDown' && items.length) {
      e.preventDefault();
      setOpen(true);
      setActive((i) => (i + 1) % items.length);
    } else if (e.key === 'ArrowUp' && items.length) {
      e.preventDefault();
      setActive((i) => (i <= 0 ? items.length - 1 : i - 1));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      submit();
    } else if (e.key === 'Escape') {
      setOpen(false);
    }
  }

  return (
    <div ref={rootRef} className="relative">
      {/* One field, one button. Uber's hero splits address / time / submit
          across three controls; a home kitchen cooks to its own schedule, so a
          "Deliver now" selector would promise a choice we do not offer. */}
      <div className="flex flex-col gap-2.5 sm:flex-row">
        <div className="relative flex-1">
          <PinIcon className="pointer-events-none absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-charcoal-soft" />
          <input
            type="text"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setError(null);
            }}
            onKeyDown={onKeyDown}
            onFocus={() => items.length && setOpen(true)}
            placeholder="Enter your area or address"
            aria-label="Your delivery area or address"
            role="combobox"
            aria-expanded={open}
            aria-controls={listId}
            aria-autocomplete="list"
            aria-activedescendant={active >= 0 ? `${listId}-${active}` : undefined}
            className="h-14 w-full rounded-xl border border-hairline bg-canvas pl-12 pr-4 text-base text-charcoal shadow-1 outline-none transition-colors placeholder:text-charcoal-soft focus:border-coral focus:ring-4 focus:ring-coral/15"
          />
        </div>
        <button
          type="button"
          onClick={submit}
          className="h-14 shrink-0 rounded-xl bg-coral px-7 text-base font-semibold text-white transition-colors hover:bg-coral-pressed focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-coral/30"
        >
          Find food
        </button>
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-sm">
        <button
          type="button"
          onClick={useMyLocation}
          disabled={locating}
          className="inline-flex items-center gap-1.5 font-medium text-charcoal transition-colors hover:text-coral disabled:opacity-60"
        >
          <CrosshairIcon className="h-4 w-4" />
          {locating ? 'Finding you…' : 'Use my current location'}
        </button>
        <span className="text-charcoal-soft">
          Already with us?{' '}
          <a
            href={APP_LOGIN_PATH}
            className="font-medium text-coral underline-offset-4 hover:underline"
          >
            Sign in
          </a>
        </span>
      </div>

      {error && (
        <p role="status" className="mt-2 text-sm text-coral">
          {error}
        </p>
      )}

      {open && items.length > 0 && (
        <ul
          id={listId}
          role="listbox"
          aria-label="Address suggestions"
          // max-h + scroll: Mappls returns up to ten matches, and an unbounded
          // list ran past the fold on a laptop — the results furthest from what
          // you typed were the ones covering the rest of the hero.
          className="absolute left-0 right-0 top-[3.75rem] z-50 max-h-[19rem] overflow-y-auto overscroll-contain rounded-xl border border-hairline bg-canvas py-1 shadow-3"
        >
          {items.map((s, i) => (
            <li
              key={`${s.description}-${i}`}
              id={`${listId}-${i}`}
              role="option"
              aria-selected={i === active}
            >
              <button
                type="button"
                // onMouseDown, not onClick: the input's blur would otherwise
                // close the list before the click ever lands.
                onMouseDown={(e) => {
                  e.preventDefault();
                  choose(s);
                }}
                onMouseEnter={() => setActive(i)}
                className={`flex w-full items-start gap-3 px-4 py-2.5 text-left transition-colors ${
                  i === active ? 'bg-surface' : ''
                }`}
              >
                <PinIcon className="mt-0.5 h-4 w-4 shrink-0 text-charcoal-soft" />
                <span className="min-w-0">
                  <span className="block truncate text-sm font-medium text-charcoal">
                    {s.line1 || s.description}
                  </span>
                  <span className="block truncate text-xs text-charcoal-soft">
                    {[s.city, s.region, s.postal].filter(Boolean).join(', ')}
                  </span>
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function PinIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M12 21s7-5.6 7-11a7 7 0 1 0-14 0c0 5.4 7 11 7 11Z"
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
      <circle cx="12" cy="10" r="2.5" stroke="currentColor" strokeWidth="1.6" />
    </svg>
  );
}

function CrosshairIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <circle cx="12" cy="12" r="7" stroke="currentColor" strokeWidth="1.6" />
      <circle cx="12" cy="12" r="2" fill="currentColor" />
      <path d="M12 2v3M12 19v3M2 12h3M19 12h3" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    </svg>
  );
}
