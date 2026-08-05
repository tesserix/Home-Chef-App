import { Link } from 'react-router';
import { motion } from 'framer-motion';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, DollarSign, CreditCard } from 'lucide-react';
import { format } from 'date-fns';
import { apiClient } from '@/shared/services/api-client';
import { formatCurrency } from '@/shared/utils/format';
import { Card } from '@/shared/components/ui/Card';
import { Badge } from '@/shared/components/ui/Badge';
import { Button } from '@/shared/components/ui/Button';
import { Skeleton } from '@/shared/components/ui/Skeleton';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';

// ---- API contract types --------------------------------------------------
// GET /chef/statements/weekly
// Mirrors the wire shape returned by GetWeeklyStatements's c.JSON exactly
// (apps/api/handlers/chef_statements.go:88-106): { statements: [...] }.
// There is no "payout method" on a settlement statement — the field the old
// local Payout interface expected does not exist on the server, so the
// interface below is the real statement shape instead of a fabricated one.

type StatementStatus = 'pending' | 'paid'; // models.PayoutStatus — only these two values exist

interface Statement {
  id: string;
  weekStart: string; // YYYY-MM-DD (Monday, IST)
  weekEnd: string; // YYYY-MM-DD (following Monday, exclusive)
  currency: string;
  ordersCount: number;
  grossRevenue: number;
  platformCommission: number;
  cgst: number;
  sgst: number;
  igst: number;
  tds: number;
  netPayout: number;
  status: StatementStatus;
  paidAt?: string;
  payoutRef?: string;
}

interface WeeklyStatementsResponse {
  statements: Statement[];
}

function getStatusVariant(status: StatementStatus) {
  switch (status) {
    case 'paid':
      return 'success' as const;
    case 'pending':
      return 'warning' as const;
    default:
      return 'default' as const;
  }
}

function getStatusLabel(status: StatementStatus): string {
  return status === 'paid' ? 'Paid' : 'Pending';
}

/** Formats a [weekStart, weekEnd) range as "1 Jun – 7 Jun 2026" (weekEnd is exclusive). */
function formatWeekRange(weekStart: string, weekEnd: string): string {
  const start = new Date(`${weekStart}T00:00:00`);
  const end = new Date(`${weekEnd}T00:00:00`);
  end.setDate(end.getDate() - 1);
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) {
    return weekStart;
  }
  return `${format(start, 'd MMM')} – ${format(end, 'd MMM yyyy')}`;
}

function PayoutsLoadingSkeleton() {
  return (
    <Card>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b border-mist">
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                Period
              </th>
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                Orders
              </th>
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                Net Payout
              </th>
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                Status
              </th>
            </tr>
          </thead>
          <tbody>
            {Array.from({ length: 8 }).map((_, i) => (
              <tr key={i} className="border-b border-mist">
                <td className="px-4 py-3">
                  <Skeleton className="h-4 w-28" />
                </td>
                <td className="px-4 py-3">
                  <Skeleton className="h-4 w-12" />
                </td>
                <td className="px-4 py-3">
                  <Skeleton className="h-4 w-20" />
                </td>
                <td className="px-4 py-3">
                  <Skeleton className="h-5 w-16 rounded-full" />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  );
}

export default function PayoutsPage() {
  const { data: statements, isLoading, isError } = useQuery<Statement[]>({
    queryKey: ['chef', 'statements', 'weekly'],
    // GET /chef/statements/weekly answers { statements: [...] } (handlers/
    // chef_statements.go:106 GetWeeklyStatements's c.JSON) — apiClient only
    // auto-unwraps a {data, pagination} envelope, so this shape passes
    // through unchanged. Select .statements explicitly, same class of bug
    // fixed for /chef/menu and friends in fe60fdf1.
    queryFn: () =>
      apiClient
        .get<WeeklyStatementsResponse>('/chef/statements/weekly')
        .then((res) => res.statements ?? []),
  });

  return (
    <div className="space-y-6">
      <motion.div
        variants={staggerContainer}
        initial="hidden"
        animate="visible"
        className="space-y-6"
      >
        {/* Page Header */}
        <motion.div variants={fadeInUp}>
          <Link to="/earnings">
            <Button variant="ghost" size="sm" className="mb-4">
              <ArrowLeft className="mr-1 h-4 w-4" />
              Back to Earnings
            </Button>
          </Link>
          <h1 className="font-display text-2xl font-semibold text-ink">Payout History</h1>
          <p className="mt-1 text-sm text-ink-muted">
            Weekly settlement statements issued by the platform
          </p>
        </motion.div>

        {/* Content */}
        <motion.div variants={fadeInUp}>
          {isLoading ? (
            <PayoutsLoadingSkeleton />
          ) : isError || !statements ? (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <DollarSign className="mb-4 h-12 w-12 text-ink-muted" />
              <h3 className="text-lg font-semibold text-ink">Unable to load payouts</h3>
              <p className="mt-1 text-sm text-ink-muted">Please try again later.</p>
            </div>
          ) : statements.length === 0 ? (
            <Card>
              <div className="flex flex-col items-center justify-center py-16 text-center">
                <CreditCard className="mb-4 h-12 w-12 text-ink-muted" />
                <h3 className="text-lg font-semibold text-ink">No payouts yet</h3>
                <p className="mt-1 max-w-sm text-sm text-ink-muted">
                  Your weekly settlement statements will appear here once the
                  platform issues your first one.
                </p>
                <Link to="/earnings" className="mt-4">
                  <Button variant="outline" size="sm">
                    <ArrowLeft className="mr-1 h-4 w-4" />
                    Back to Earnings
                  </Button>
                </Link>
              </div>
            </Card>
          ) : (
            <Card padding="none">
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-mist bg-paper/50">
                      <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                        Period
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                        Orders
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                        Net Payout
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                        Status
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-mist">
                    {statements.map((statement) => (
                      <tr
                        key={statement.id}
                        className="transition-colors hover:bg-paper/50"
                      >
                        <td className="whitespace-nowrap px-6 py-4 text-sm text-ink-soft">
                          {formatWeekRange(statement.weekStart, statement.weekEnd)}
                        </td>
                        <td className="whitespace-nowrap px-6 py-4 text-sm text-ink-soft">
                          {statement.ordersCount}
                        </td>
                        <td className="whitespace-nowrap px-6 py-4 text-sm font-semibold text-ink">
                          {formatCurrency(statement.netPayout)}
                        </td>
                        <td className="whitespace-nowrap px-6 py-4">
                          <Badge variant={getStatusVariant(statement.status)} size="sm">
                            {getStatusLabel(statement.status)}
                          </Badge>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </Card>
          )}
        </motion.div>
      </motion.div>
    </div>
  );
}
