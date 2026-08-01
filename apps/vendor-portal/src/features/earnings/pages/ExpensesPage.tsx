import { useState } from 'react';
import { motion } from 'framer-motion';
import { format } from 'date-fns';
import { Pencil, Plus, Receipt, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { Card } from '@/shared/components/ui/Card';
import { Button } from '@/shared/components/ui/Button';
import { Input } from '@/shared/components/ui/Input';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue,
} from '@/shared/components/ui/Select';
import { formatCurrency } from '@/shared/utils/format';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';
import { downloadPdf } from '../hooks/useEarningsExtras';
import {
  EXPENSE_CATEGORIES,
  currentFyStartYear,
  expenseCategoryLabel,
  fyOptions,
  useExpenseMutations,
  useExpenses,
  useFYStatement,
  type ChefExpense,
  type ExpenseCategory,
  type ExpenseInput,
} from '../hooks/useExpenses';

// Expenses & annual tax statement — the chef's own books. Expenses recorded
// here feed the FY statement (income from delivered orders − these expenses)
// that backs GST/ITR filing.

interface FormState {
  category: ExpenseCategory;
  amount: string;
  note: string;
  expenseDate: string;
}

const emptyForm = (): FormState => ({
  category: 'ingredients',
  amount: '',
  note: '',
  expenseDate: format(new Date(), 'yyyy-MM-dd'),
});

function ExpenseForm({
  initial,
  busy,
  onSubmit,
  onCancel,
}: {
  initial: FormState;
  busy: boolean;
  onSubmit: (input: ExpenseInput) => void;
  onCancel?: () => void;
}) {
  const [form, setForm] = useState<FormState>(initial);

  function submit(e: React.FormEvent) {
    e.preventDefault();
    const amount = Number(form.amount);
    if (!Number.isFinite(amount) || amount <= 0) {
      toast.error('Enter a valid amount');
      return;
    }
    onSubmit({
      category: form.category,
      amount,
      note: form.note.trim() || undefined,
      expenseDate: form.expenseDate,
    });
  }

  return (
    <form onSubmit={submit} className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5">
      <Select
        value={form.category}
        onValueChange={(v) => setForm((f) => ({ ...f, category: v as ExpenseCategory }))}
      >
        <SelectTrigger aria-label="Category">
          <SelectValue placeholder="Category" />
        </SelectTrigger>
        <SelectContent>
          {EXPENSE_CATEGORIES.map((c) => (
            <SelectItem key={c.value} value={c.value}>
              {c.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Input
        type="number"
        inputMode="decimal"
        min="0.01"
        step="0.01"
        placeholder="Amount (₹)"
        aria-label="Amount"
        value={form.amount}
        onChange={(e) => setForm((f) => ({ ...f, amount: e.target.value }))}
        required
      />
      <Input
        type="date"
        aria-label="Date"
        value={form.expenseDate}
        max={format(new Date(), 'yyyy-MM-dd')}
        onChange={(e) => setForm((f) => ({ ...f, expenseDate: e.target.value }))}
        required
      />
      <Input
        type="text"
        placeholder="Note (optional)"
        aria-label="Note"
        maxLength={500}
        value={form.note}
        onChange={(e) => setForm((f) => ({ ...f, note: e.target.value }))}
      />
      <div className="flex gap-2">
        <Button type="submit" disabled={busy} className="flex-1">
          {busy ? 'Saving…' : onCancel ? 'Save' : 'Add expense'}
        </Button>
        {onCancel && (
          <Button type="button" variant="ghost" onClick={onCancel}>
            Cancel
          </Button>
        )}
      </div>
    </form>
  );
}

/** One FY-statement money line; `strong` for the totals rows. */
function Line({ label, amount, strong, negative }: {
  label: string;
  amount: number;
  strong?: boolean;
  negative?: boolean;
}) {
  return (
    <div className="flex items-center justify-between py-2">
      <span className={`text-sm ${strong ? 'font-semibold text-ink' : 'text-ink-soft'}`}>
        {label}
      </span>
      <span
        className={`text-sm tabular-nums ${
          strong ? 'font-semibold text-ink' : negative ? 'text-paprika' : 'text-ink'
        }`}
      >
        {negative ? '− ' : ''}
        {formatCurrency(Math.abs(amount))}
      </span>
    </div>
  );
}

export default function ExpensesPage() {
  const [fy, setFy] = useState<number>(currentFyStartYear());
  const [editing, setEditing] = useState<ChefExpense | null>(null);
  const [downloading, setDownloading] = useState(false);

  const { data: expenseData, isLoading: expensesLoading } = useExpenses();
  const { data: stmt, isLoading: stmtLoading } = useFYStatement(fy);
  const { create, update, remove } = useExpenseMutations();

  const expenses = expenseData?.expenses ?? [];
  const busy = create.isPending || update.isPending;

  function handleCreate(input: ExpenseInput) {
    create.mutate(input, {
      onSuccess: () => toast.success('Expense added'),
      onError: () => toast.error('Could not save the expense. Please try again.'),
    });
  }

  function handleUpdate(input: ExpenseInput) {
    if (!editing) return;
    update.mutate(
      { id: editing.id, ...input },
      {
        onSuccess: () => {
          toast.success('Expense updated');
          setEditing(null);
        },
        onError: () => toast.error('Could not update the expense. Please try again.'),
      },
    );
  }

  function handleDelete(id: string) {
    remove.mutate(id, {
      onSuccess: () => toast.success('Expense deleted'),
      onError: () => toast.error('Could not delete the expense. Please try again.'),
    });
  }

  async function downloadStatement() {
    setDownloading(true);
    try {
      await downloadPdf(`/chef/tax/fy-statement.pdf?year=${fy}`, `fy-statement-${fy}.pdf`);
    } catch {
      toast.error('Could not download the statement. Please try again.');
    } finally {
      setDownloading(false);
    }
  }

  const gstOnCommission = stmt
    ? stmt.commissionCgst + stmt.commissionSgst + stmt.commissionIgst
    : 0;

  return (
    <motion.div variants={staggerContainer} initial="hidden" animate="visible" className="space-y-6">
      <motion.div
        variants={fadeInUp}
        className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"
      >
        <div>
          <h1 className="font-display text-2xl font-semibold text-ink">Expenses & Tax</h1>
          <p className="mt-1 text-sm text-ink-muted">
            Record kitchen expenses and generate your annual statement for GST / income-tax filing
          </p>
        </div>
        <div className="w-40">
          <Select value={String(fy)} onValueChange={(v) => setFy(Number(v))}>
            <SelectTrigger aria-label="Financial year">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {fyOptions().map((o) => (
                <SelectItem key={o.value} value={String(o.value)}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </motion.div>

      {/* Add expense */}
      <motion.div variants={fadeInUp}>
        <Card className="p-5">
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-ink-muted">
            <Plus className="h-4 w-4" /> Add expense
          </h2>
          <ExpenseForm initial={emptyForm()} busy={busy && !editing} onSubmit={handleCreate} />
        </Card>
      </motion.div>

      {/* FY statement summary */}
      <motion.div variants={fadeInUp}>
        <Card className="p-5">
          <div className="mb-2 flex items-center justify-between gap-3">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-ink-muted">
              Annual statement · {stmt?.fyLabel ?? `FY ${fy}-${String((fy + 1) % 100).padStart(2, '0')}`}
            </h2>
            <Button size="sm" onClick={() => void downloadStatement()} disabled={downloading || stmtLoading}>
              {downloading ? 'Preparing…' : 'Download PDF'}
            </Button>
          </div>
          {stmtLoading || !stmt ? (
            <Skeleton className="mt-3 h-40 w-full" />
          ) : (
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
              <div className="divide-y divide-mist">
                <p className="pb-2 text-xs font-medium uppercase tracking-wide text-ink-muted">
                  Income · {stmt.ordersCount} delivered orders
                </p>
                <Line label="Food revenue" amount={stmt.foodRevenue} />
                <Line label="GST collected from customers" amount={stmt.gstCollected} />
                <Line label="Customer tips" amount={stmt.tips} />
                <Line label="Gross receipts" amount={stmt.grossReceipts} strong />
                <Line label="Platform commission" amount={stmt.platformCommission} negative />
                <Line label="GST on commission (ITC eligible)" amount={gstOnCommission} />
                <Line label="TDS withheld (194-O)" amount={stmt.tdsWithheld} negative />
                <Line label="Net earnings from platform" amount={stmt.netEarnings} strong />
              </div>
              <div className="divide-y divide-mist">
                <p className="pb-2 text-xs font-medium uppercase tracking-wide text-ink-muted">
                  Expenses · self-declared
                </p>
                {stmt.expenses.byCategory.length === 0 ? (
                  <p className="py-3 text-sm text-ink-soft">
                    No expenses recorded in this financial year yet.
                  </p>
                ) : (
                  stmt.expenses.byCategory.map((c) => (
                    <Line
                      key={c.category}
                      label={`${expenseCategoryLabel(c.category)} (${c.count})`}
                      amount={c.amount}
                      negative
                    />
                  ))
                )}
                <Line label="Total expenses" amount={stmt.totalExpenses} strong />
                <div className="flex items-center justify-between py-3">
                  <span className="text-sm font-semibold text-ink">Net income (pre-tax)</span>
                  <span className="text-lg font-semibold text-herb tabular-nums">
                    {formatCurrency(stmt.netIncome)}
                  </span>
                </div>
              </div>
            </div>
          )}
        </Card>
      </motion.div>

      {/* Expense list */}
      <motion.div variants={fadeInUp}>
        <Card className="p-5">
          <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-ink-muted">
            <Receipt className="h-4 w-4" /> Recorded expenses
          </h2>
          {expensesLoading ? (
            <Skeleton className="mt-3 h-24 w-full" />
          ) : expenses.length === 0 ? (
            <p className="mt-3 text-sm text-ink-soft">
              Nothing recorded yet. Add gas refills, ingredient runs, utensils and other kitchen
              costs above — they roll into your annual statement automatically.
            </p>
          ) : (
            <div className="divide-y divide-mist">
              {expenses.map((e) =>
                editing?.id === e.id ? (
                  <div key={e.id} className="py-3">
                    <ExpenseForm
                      initial={{
                        category: e.category,
                        amount: String(e.amount),
                        note: e.note ?? '',
                        expenseDate: format(new Date(e.expenseDate), 'yyyy-MM-dd'),
                      }}
                      busy={update.isPending}
                      onSubmit={handleUpdate}
                      onCancel={() => setEditing(null)}
                    />
                  </div>
                ) : (
                  <div key={e.id} className="flex items-center gap-3 py-3">
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-medium text-ink">
                        {expenseCategoryLabel(e.category)}
                      </p>
                      <p className="truncate text-xs text-ink-muted tabular-nums">
                        {format(new Date(e.expenseDate), 'dd MMM yyyy')}
                        {e.note ? ` · ${e.note}` : ''}
                      </p>
                    </div>
                    <p className="shrink-0 text-sm font-semibold text-ink tabular-nums">
                      {formatCurrency(e.amount)}
                    </p>
                    <div className="flex shrink-0 gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label="Edit expense"
                        onClick={() => setEditing(e)}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label="Delete expense"
                        onClick={() => handleDelete(e.id)}
                        disabled={remove.isPending}
                      >
                        <Trash2 className="h-4 w-4 text-paprika" />
                      </Button>
                    </div>
                  </div>
                ),
              )}
            </div>
          )}
        </Card>
      </motion.div>
    </motion.div>
  );
}
