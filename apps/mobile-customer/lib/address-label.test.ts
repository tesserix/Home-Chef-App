import { describe, it, expect } from '@jest/globals';

import { fullAddress, savedAddressLabel, placeLabel } from './address-label';

const office = {
  label: 'Office',
  addressLine1: '12 Queen St',
  addressLine2: 'Level 4',
  city: 'Auckland',
  state: 'Auckland',
  pincode: '1010',
};

describe('fullAddress', () => {
  it('includes street, unit, city, region and postcode', () => {
    expect(fullAddress(office)).toBe('12 Queen St, Level 4, Auckland, Auckland 1010');
  });

  it('skips blank parts without leaving stray commas', () => {
    expect(fullAddress({ addressLine1: '5 Lygon St', addressLine2: ' ', city: 'Carlton', state: 'VIC', pincode: '' })).toBe(
      '5 Lygon St, Carlton, VIC',
    );
  });
});

describe('savedAddressLabel', () => {
  it('shows the address name when the customer set one', () => {
    expect(savedAddressLabel(office)).toBe('Office');
  });

  it('shows the full address when there is no name', () => {
    expect(savedAddressLabel({ ...office, label: ' ' })).toBe('12 Queen St, Level 4, Auckland, Auckland 1010');
  });
});

describe('placeLabel', () => {
  const queenSt = {
    streetNumber: '12',
    street: 'Queen St',
    name: '12 Queen St',
    city: 'Auckland',
    region: 'Auckland',
    postalCode: '1010',
  };

  it('builds the full street address from the reverse-geocoded place', () => {
    expect(placeLabel(queenSt)).toBe('12 Queen St, Auckland, Auckland 1010');
  });

  it('uses the place name when there is no street', () => {
    expect(placeLabel({ ...queenSt, streetNumber: null, street: null, name: 'Britomart' })).toBe(
      'Britomart, Auckland, Auckland 1010',
    );
  });

  it('does not repeat the city as the street when the name is just the city', () => {
    expect(placeLabel({ ...queenSt, streetNumber: null, street: null, name: 'Auckland' })).toBe('Auckland, Auckland 1010');
  });

  it('is empty when nothing is known', () => {
    expect(
      placeLabel({ streetNumber: null, street: null, name: null, city: null, region: null, postalCode: null }),
    ).toBe('');
  });
});
