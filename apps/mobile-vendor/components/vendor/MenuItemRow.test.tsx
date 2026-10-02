import React from 'react';
import { afterEach, expect, it, jest } from '@jest/globals';
import { act, create } from 'react-test-renderer';
import type { MenuItem } from '../../hooks/useVendorMenu';
jest.mock('react-native', () => ({ Platform: { OS: 'ios' }, Switch: 'Switch', Pressable: ({ children, ...props }: any) => require('react').createElement('Pressable', props, typeof children === 'function' ? children({ pressed: false }) : children), View: 'View', Text: 'Text', StyleSheet: { create: (v: unknown) => v } }));
jest.mock('lucide-react-native', () => ({ UtensilsCrossed: 'UtensilsCrossed' }));
jest.mock('expo-image', () => ({ Image: 'Image' }));
jest.mock('./DietIcon', () => ({ DietIcon: 'DietIcon' }));
jest.mock('../../hooks/useVendorMenu', () => ({ useToggleAvailability: () => ({ mutate: jest.fn() }) }));
jest.mock('@homechef/mobile-shared/ui', () => ({ useToast: () => ({ show: jest.fn() }) }));
import { MenuItemRow } from './MenuItemRow';
(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
let screen: ReturnType<typeof create>;
afterEach(async () => { if (screen) await act(async () => screen.unmount()); });
it.each([['NZD', '$19.50'], ['AUD', '$19.50'], ['INR', '₹19.50']])('renders menu prices in %s', async (currency, expected) => {
  const item: MenuItem = {
    id: 'test-item', name: 'Test meal', description: 'A test meal', price: 19.5,
    categoryId: null, images: [], isVeg: false, isAvailable: true,
    dietaryTags: [], allergens: [], availableDays: [], preparationTime: 15,
  };
  await act(async () => { screen = create(<MenuItemRow item={item} currency={currency} onPress={() => {}} />); });
  expect(JSON.stringify(screen.toJSON())).toContain(expected);
});
