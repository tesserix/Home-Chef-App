import { it, expect, jest } from '@jest/globals';
import { api } from '../lib/api';
import { useOrder, useOrders } from './useOrderHistory';
import type { Order } from '../types/customer';

jest.mock('@tanstack/react-query', () => ({ useQuery: (options: unknown) => options }));
jest.mock('../lib/api', () => ({ api: { get: jest.fn() } }));

it.each(['AUD', 'NZD', 'INR'])('preserves %s on order detail and history', async (currency) => {
  const raw = { id: 'order-1', orderNumber: 'TEST-1', status: 'pending', total: 50, currency };
  jest.mocked(api.get).mockResolvedValueOnce({ data: raw } as never);
  const detail = useOrder('order-1') as unknown as { queryFn: () => Promise<{ data: Order }> };
  expect((await detail.queryFn()).data).toMatchObject({ currency, totalAmount: 50 });
  jest.mocked(api.get).mockResolvedValueOnce({ data: { data: [raw] } } as never);
  const list = useOrders() as unknown as { queryFn: () => Promise<{ data: Order[] }> };
  expect((await list.queryFn()).data[0]).toMatchObject({ currency, totalAmount: 50 });
});
