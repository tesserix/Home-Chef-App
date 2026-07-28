import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/shared/services/api-client';
import { suggestionCoords, type AddressSuggestion } from '../hooks/useAddressAutocomplete';
import { AddressSearch } from './AddressSearch';

// Coordinates are the whole point of this control. An address saved without them
// is rejected by CreateOrder (422 `delivery_location_required`), which is exactly
// how every web order was failing before the geocoder was wired up here.

function suggestion(over: Partial<AddressSuggestion> = {}): AddressSuggestion {
  return {
    description: 'Kalarahanga, Bhubaneswar, Odisha',
    line1: 'Kalarahanga',
    city: 'Bhubaneswar',
    region: 'Odisha',
    postal: '751024',
    country: 'IN',
    lat: 20.3,
    lon: 85.82,
    ...over,
  };
}

function renderSearch(onPick = vi.fn()) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <AddressSearch label="Find your address" onPick={onPick} />
    </QueryClientProvider>
  );
  return onPick;
}

describe('suggestionCoords', () => {
  it('keeps real coordinates', () => {
    expect(suggestionCoords(suggestion())).toEqual({ lat: 20.3, lon: 85.82 });
  });

  it('rejects a geocoder miss rather than saving null island', () => {
    // 0,0 is in the Gulf of Guinea. Persisting it would pass the "has
    // coordinates" check and then fail the range check with a nonsense distance.
    expect(suggestionCoords(suggestion({ lat: 0, lon: 0 }))).toBeNull();
    expect(suggestionCoords(suggestion({ lat: undefined, lon: undefined }))).toBeNull();
  });
});

describe('AddressSearch', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  // apiClient UNWRAPS a `{ data }` envelope that carries no `pagination`, so for
  // this endpoint it resolves to the bare array — NOT `{ data: [...] }`.
  // Mocking the envelope is what let a hook that read `.data` off an array pass
  // its tests while returning nothing at all in production: the picker reported
  // "No matches" for every address. The mock must match what the client really
  // returns, or the test is only checking itself.
  it('hands the picked suggestion — coordinates and all — to the caller', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValue([suggestion()]);
    const user = userEvent.setup();
    const onPick = renderSearch();

    await user.type(screen.getByLabelText('Find your address'), 'kalarahanga');

    await user.click(await screen.findByText('Kalarahanga, Bhubaneswar, Odisha'));
    expect(onPick).toHaveBeenCalledWith(expect.objectContaining({ lat: 20.3, lon: 85.82 }));
  });

  it('does not query the geocoder below its 3-character minimum', async () => {
    const get = vi.spyOn(apiClient, 'get').mockResolvedValue([]);
    const user = userEvent.setup();
    renderSearch();

    await user.type(screen.getByLabelText('Find your address'), 'ka');

    expect(get).not.toHaveBeenCalled();
  });
});

describe('AddressSearch when the geocoder is down', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  // An outage and a genuine miss used to render identically — both said "no
  // matches", which tells the customer their address doesn't exist when the
  // truth is the lookup is broken. These two cases must stay distinguishable.
  it('offers a retry instead of claiming the address does not exist', async () => {
    vi.spyOn(apiClient, 'get').mockRejectedValue(
      Object.assign(new Error('geocoder down'), { status: 503, code: 'geocoder_unavailable' }),
    );
    const user = userEvent.setup();
    renderSearch();

    await user.type(screen.getByLabelText('Find your address'), 'kalarahanga');

    expect(await screen.findByText(/unavailable right now/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument();
    expect(screen.queryByText(/no matches/i)).not.toBeInTheDocument();
  });

  it('still says "no matches" when the geocoder simply found nothing', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValue({ data: [] });
    const user = userEvent.setup();
    renderSearch();

    await user.type(screen.getByLabelText('Find your address'), 'zzzzzzz');

    expect(await screen.findByText(/no matches/i)).toBeInTheDocument();
    expect(screen.queryByText(/unavailable right now/i)).not.toBeInTheDocument();
  });
});
describe('AddressSearch response-shape tolerance', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  // apiClient UNWRAPS a `{ data }` envelope that carries no `pagination`, so for
  // this endpoint it resolves to the BARE ARRAY. Mocking the envelope instead is
  // what let a hook reading `.data` off an array pass its tests while returning
  // nothing in production — the picker said "No matches" for every address.
  //
  // Both shapes are asserted so this cannot rot again if the server ever starts
  // (or stops) paginating and apiClient's unwrapping flips with it.
  it.each([
    ['unwrapped array (what this endpoint returns today)', [suggestion()]],
    ['envelope (if it ever becomes paginated)', { data: [suggestion()] }],
  ])('reads suggestions from %s', async (_label, shape) => {
    vi.spyOn(apiClient, 'get').mockResolvedValue(shape);
    const user = userEvent.setup();
    renderSearch();

    await user.type(screen.getByLabelText('Find your address'), 'kalarahanga');

    expect(await screen.findByText('Kalarahanga, Bhubaneswar, Odisha')).toBeInTheDocument();
    expect(screen.queryByText(/no matches/i)).not.toBeInTheDocument();
  });
});
