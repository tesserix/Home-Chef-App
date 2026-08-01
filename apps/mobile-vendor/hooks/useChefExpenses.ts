import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// ---- API contract types -------------------------------------------------------
// Must match handlers/chef_expenses.go and services/fy_statement.go exactly.

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
  receiptUrl?: string;
  createdAt: string;
}

export interface ExpenseInput {
  category: ExpenseCategory;
  amount: number;
  note?: string;
  expenseDate: string; // YYYY-MM-DD
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

/** FY start-year for today (Jan–Mar belong to the previous April's FY). */
export function currentFyStartYear(d: Date = new Date()): number {
  return d.getMonth() >= 3 ? d.getFullYear() : d.getFullYear() - 1;
}

export function useChefExpenses(limit = 100) {
  return useQuery<{ expenses: ChefExpense[]; total: number }>({
    queryKey: ['chef', 'expenses', limit],
    queryFn: () =>
      api
        .get<{ expenses: ChefExpense[]; total: number }>(`/chef/expenses?limit=${limit}`)
        .then((r) => ({
          expenses: r.data?.expenses ?? [],
          total: r.data?.total ?? 0,
        })),
  });
}

export function useFYStatement(fyStartYear: number) {
  return useQuery<FYStatement>({
    queryKey: ['chef', 'fy-statement', fyStartYear],
    queryFn: () =>
      api.get<FYStatement>(`/chef/tax/fy-statement?year=${fyStartYear}`).then((r) => r.data),
  });
}

/** Create / delete an expense; both refresh the list and the FY statement. */
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
