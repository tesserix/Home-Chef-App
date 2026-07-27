import { useId, useState } from 'react';
import { Search, Loader2, MapPin } from 'lucide-react';
import {
  useAddressAutocomplete,
  type AddressSuggestion,
} from '../hooks/useAddressAutocomplete';

// AddressSearch — geocoder-backed address lookup, the web counterpart of the
// mobile address picker.
//
// Its real job is coordinates. A saved address without a lat/lng cannot be
// range-checked, and CreateOrder refuses to place a delivery order it cannot
// range-check — so an address typed straight into the form fields is an address
// that can never be ordered to.

export interface AddressSearchProps {
  label: string;
  placeholder?: string;
  hint?: string;
  onPick: (suggestion: AddressSuggestion) => void;
}

export function AddressSearch({ label, placeholder, hint, onPick }: AddressSearchProps) {
  const id = useId();
  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const { data: suggestions = [], isFetching, isError, refetch } = useAddressAutocomplete(query);

  const showList = open && query.trim().length >= 3;

  return (
    <div className="relative">
      <label htmlFor={id} className="block text-sm font-medium text-ink-soft">
        {label}
      </label>
      <div className="relative mt-1">
        <Search
          className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-muted"
          aria-hidden="true"
        />
        <input
          id={id}
          type="text"
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          // The list closes only on a real focus change to something OUTSIDE this
          // control. The previous 150ms blur timer raced anything that touched
          // focus mid-typing (devtools, a re-render, the browser chrome) and the
          // list would vanish with no way to tell that from "found nothing".
          // Suggestion clicks are handled by onMouseDown-preventDefault below, so
          // focus never leaves the input for them at all.
          onBlur={(e) => {
            if (!e.currentTarget.parentElement?.parentElement?.contains(e.relatedTarget)) {
              setOpen(false);
            }
          }}
          placeholder={placeholder ?? 'Start typing your street, area or landmark'}
          autoComplete="off"
          role="combobox"
          aria-expanded={showList}
          aria-controls={`${id}-list`}
          className="input-base pl-9"
        />
        {isFetching && (
          <Loader2
            className="absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 animate-spin text-ink-muted"
            aria-hidden="true"
          />
        )}
      </div>
      {hint && <p className="mt-1 text-xs text-ink-muted">{hint}</p>}

      {showList && (
        <ul
          id={`${id}-list`}
          role="listbox"
          className="absolute z-20 mt-1 max-h-64 w-full overflow-auto rounded-lg border border-mist bg-bone shadow-2"
        >
          {suggestions.map((s, i) => (
            <li key={`${s.description}-${i}`} role="option" aria-selected={false}>
              <button
                type="button"
                // Keeps focus in the input so the click always lands — the old
                // deferred-blur approach could unmount the row mid-click.
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => {
                  onPick(s);
                  setQuery('');
                  setOpen(false);
                }}
                className="flex w-full items-start gap-2 px-3 py-2.5 text-left hover:bg-paper"
              >
                <MapPin className="mt-0.5 h-4 w-4 flex-shrink-0 text-herb" aria-hidden="true" />
                <span className="text-sm text-ink">{s.description}</span>
              </button>
            </li>
          ))}
          {/* A pending first lookup gets its own row. Without it the list showed
              "no matches" while the request was still in flight, which reads as
              a definitive answer to a question that hasn't been answered yet. */}
          {isFetching && suggestions.length === 0 && (
            <li className="px-3 py-2.5 text-sm text-ink-muted">Searching…</li>
          )}
          {/* An outage and an empty result used to look identical here: both
              showed "no matches", telling the customer their address doesn't
              exist when the geocoder was simply down. The API now says which
              it is (503 `geocoder_unavailable`), so this can offer a retry
              instead of a dead end. */}
          {!isFetching && isError && (
            <li className="px-3 py-2.5 text-sm text-ink-soft">
              Address lookup is unavailable right now.{' '}
              <button
                type="button"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => refetch()}
                className="font-medium text-herb underline underline-offset-2"
              >
                Try again
              </button>
              , or type your address into the fields below.
            </li>
          )}
          {!isFetching && !isError && suggestions.length === 0 && (
            <li className="px-3 py-2.5 text-sm text-ink-muted">
              No matches. Try a nearby landmark or just the area name.
            </li>
          )}
        </ul>
      )}
    </div>
  );
}
