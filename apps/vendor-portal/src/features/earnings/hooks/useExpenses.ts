import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Expense bookkeeping + FY statement wire types. Must match
// apps/api/handlers/chef_expenses.go and services/fy_statement.go exactly.

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
  { value: 'ingredients', label: 'Ingredients & groceries' },
  { value: 'gas', label: 'Cooking gas / fuel' },
  { value: 'utensils', label: 'Utensils & cookware' },
  { value: 'packaging', label: 'Packaging & disposables' },
  { value: 'transport', label: 'Transport & delivery' },
  { value: 'equipment', label: 'Kitchen equipment' },
  { value: 'utilities', label: 'Electricity & utilities' },
  { value: 'other', label: 'Other expenses' },
];

export function expenseCategoryLabel(c: ExpenseCategory | string): string {
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
  /** Short-lived signed URL, present only when a receipt is attached. */
  receiptUrl?: string;
  createdAt: string;
}

export interface ExpenseInput {
  category: ExpenseCategory;
  amount: number;
  note?: string;
  expenseDate: string; // YYYY-MM-DD
  orderId?: string;
  receiptPath?: string;
}

export interface ExpenseCategoryTotal {
  category: ExpenseCategory;
  amount: number;
  count: number;
}

export interface ExpenseSummary {
  fyStartYear: number;
  fyLabel: string;
  currency: string;
  total: number;
  count: number;
  byCategory: ExpenseCategoryTotal[];
  byMonth: { month: string; amount: number }[];
}

export interface FYQuarter {
  label: string;
  ordersCount: number;
  gross: number;
  tds: number;
  netPayout: number;
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
  quarters: FYQuarter[];
  expenses: ExpenseSummary;
  totalExpenses: number;
  netIncome: number;
}

/** FY start-year for today (Jan–Mar belong to the previous April's FY). */
export function currentFyStartYear(d: Date = new Date()): number {
  return d.getMonth() >= 3 ? d.getFullYear() : d.getFullYear() - 1;
}

/** Selectable FYs, newest first, back to the platform's first year. */
export function fyOptions(earliest = 2024): { value: number; label: string }[] {
  const current = currentFyStartYear();
  const out: { value: number; label: string }[] = [];
  for (let y = current; y >= Math.min(earliest, current); y--) {
    out.push({ value: y, label: `FY ${y}-${String((y + 1) % 100).padStart(2, '0')}` });
  }
  return out;
}

export function useExpenses(limit = 100) {
  return useQuery<{ expenses: ChefExpense[]; total: number }>({
    queryKey: ['chef', 'expenses', limit],
    queryFn: () =>
      apiClient
        .get<{ expenses: ChefExpense[]; total: number }>(`/chef/expenses?limit=${limit}`)
        .then((r) => ({ expenses: r?.expenses ?? [], total: r?.total ?? 0 })),
  });
}

export function useExpenseSummary(fyStartYear: number) {
  return useQuery<ExpenseSummary>({
    queryKey: ['chef', 'expenses', 'summary', fyStartYear],
    queryFn: () => apiClient.get<ExpenseSummary>(`/chef/expenses/summary?year=${fyStartYear}`),
  });
}

export function useFYStatement(fyStartYear: number) {
  return useQuery<FYStatement>({
    queryKey: ['chef', 'fy-statement', fyStartYear],
    queryFn: () => apiClient.get<FYStatement>(`/chef/tax/fy-statement?year=${fyStartYear}`),
  });
}

export function useUploadReceipt() {
  return useMutation({
    mutationFn: async (file: File) => {
      const form = new FormData();
      form.append('file', file);
      const res = await apiClient.postForm<{ path: string }>('/chef/expenses/receipt', form);
      return res.path;
    },
  });
}

/** Create / update / delete an expense; all three invalidate every expense view. */
export function useExpenseMutations() {
  const queryClient = useQueryClient();
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['chef', 'expenses'] });
    void queryClient.invalidateQueries({ queryKey: ['chef', 'fy-statement'] });
  };

  const create = useMutation({
    mutationFn: (input: ExpenseInput) =>
      apiClient.post<{ expense: ChefExpense }>('/chef/expenses', input),
    onSuccess: invalidate,
  });
  const update = useMutation({
    mutationFn: ({ id, ...input }: ExpenseInput & { id: string }) =>
      apiClient.put<{ expense: ChefExpense }>(`/chef/expenses/${id}`, input),
    onSuccess: invalidate,
  });
  const remove = useMutation({
    mutationFn: (id: string) => apiClient.delete<{ deleted: boolean }>(`/chef/expenses/${id}`),
    onSuccess: invalidate,
  });
  return { create, update, remove };
}
