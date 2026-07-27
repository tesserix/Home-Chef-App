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

  it('hands the picked suggestion — coordinates and all — to the caller', async () => {
    vi.spyOn(apiClient, 'get').mockResolvedValue({ data: [suggestion()] });
    const user = userEvent.setup();
    const onPick = renderSearch();

    await user.type(screen.getByLabelText('Find your address'), 'kalarahanga');

    await user.click(await screen.findByText('Kalarahanga, Bhubaneswar, Odisha'));
    expect(onPick).toHaveBeenCalledWith(expect.objectContaining({ lat: 20.3, lon: 85.82 }));
  });

  it('does not query the geocoder below its 3-character minimum', async () => {
    const get = vi.spyOn(apiClient, 'get').mockResolvedValue({ data: [] });
    const user = userEvent.setup();
    renderSearch();

    await user.type(screen.getByLabelText('Find your address'), 'ka');

    expect(get).not.toHaveBeenCalled();
  });
});
