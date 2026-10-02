import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { multipartConfig } from '@homechef/mobile-shared/api';
import { api } from '../lib/api';

export type ExpenseCategory =
  | 'ingredients'
  | 'gas'
  | 'utensils'
  | 'packaging'
  | 'transport'
  | 'equipment'
  | 'utilities'
  | 'other';

export const EXPENSE_CATEGORIES: { value: ExpenseCategory; label: string }[] = [
  { value: 'ingredients', label: 'Ingredients' },
  { value: 'gas', label: 'Gas / fuel' },
  { value: 'utensils', label: 'Utensils' },
  { value: 'packaging', label: 'Packaging' },
  { value: 'transport', label: 'Transport' },
  { value: 'equipment', label: 'Equipment' },
  { value: 'utilities', label: 'Utilities' },
  { value: 'other', label: 'Other' },
];

export function expenseCategoryLabel(c: string): string {
  return EXPENSE_CATEGORIES.find((e) => e.value === c)?.label ?? c;
}

export interface ChefExpense {
  id: string;
  category: ExpenseCategory;
  amount: number;
  currency: string;
  note?: string;
  expenseDate: string;
  orderId?: string;
  orderNumber?: string;
  receiptPath?: string;
  receiptUrl?: string;
  createdAt: string;
}

export interface ExpenseInput {
  category: ExpenseCategory;
  amount: number;
  note?: string;
  expenseDate: string;
  orderId?: string;
  receiptPath?: string;
}

export interface ExpenseSummary {
  fyStartYear: number;
  fyLabel: string;
  currency: string;
  total: number;
  count: number;
  byCategory: { category: ExpenseCategory; amount: number; count: number }[];
  byMonth: { month: string; amount: number }[];
}

export interface FYStatement {
  fyStartYear: number;
  fyLabel: string;
  currency: string;
  ordersCount: number;
  foodRevenue: number;
  gstCollected: number;
  tips: number;
  grossReceipts: number;
  platformCommission: number;
  commissionCgst: number;
  commissionSgst: number;
  commissionIgst: number;
  tdsWithheld: number;
  netEarnings: number;
  quarters: { label: string; ordersCount: number; gross: number; tds: number; netPayout: number }[];
  expenses: ExpenseSummary;
  totalExpenses: number;
  netIncome: number;
}

export function useChefExpenses(options?: { orderId?: string; limit?: number }) {
  const limit = options?.limit ?? 100;
  const orderId = options?.orderId;
  return useQuery<{ expenses: ChefExpense[]; total: number }>({
    queryKey: ['chef', 'expenses', { limit, orderId: orderId ?? null }],
    queryFn: () => {
      const params = new URLSearchParams({ limit: String(limit) });
      if (orderId) params.set('orderId', orderId);
      return api
        .get<{ expenses: ChefExpense[]; total: number }>(`/chef/expenses?${params}`)
        .then((r) => ({ expenses: r.data?.expenses ?? [], total: r.data?.total ?? 0 }));
    },
  });
}

export function useFYStatement(fyStartYear: number | undefined) {
  return useQuery<FYStatement>({
    queryKey: ['chef', 'fy-statement', fyStartYear],
    queryFn: () =>
      api.get<FYStatement>(`/chef/tax/fy-statement?year=${fyStartYear}`).then((r) => r.data),
    enabled: fyStartYear !== undefined,
  });
}

export function useUploadExpenseReceipt() {
  return useMutation({
    mutationFn: async (uri: string) => {
      const formData = new FormData();
      const filename = uri.split('/').pop() ?? 'receipt.jpg';
      const ext = filename.toLowerCase().split('.').pop() ?? '';
      const type =
        ext === 'png'
          ? 'image/png'
          : ext === 'webp'
            ? 'image/webp'
            : ext === 'heic' || ext === 'heif'
              ? `image/${ext}`
              : 'image/jpeg';
      formData.append('file', { uri, name: filename, type } as unknown as Blob);
      const res = await api.post<{ path: string }>(
        '/chef/expenses/receipt',
        formData,
        multipartConfig(),
      );
      return res.data.path;
    },
  });
}

export function useExpenseMutations() {
  const queryClient = useQueryClient();
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['chef', 'expenses'] });
    void queryClient.invalidateQueries({ queryKey: ['chef', 'fy-statement'] });
  };

  const create = useMutation({
    mutationFn: (input: ExpenseInput) =>
      api.post<{ expense: ChefExpense }>('/chef/expenses', input).then((r) => r.data),
    onSuccess: invalidate,
  });
  const remove = useMutation({
    mutationFn: (id: string) => api.delete(`/chef/expenses/${id}`).then(() => undefined),
    onSuccess: invalidate,
  });
  return { create, remove };
}
