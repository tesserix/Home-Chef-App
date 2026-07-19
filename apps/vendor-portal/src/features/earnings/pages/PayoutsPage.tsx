import { Link } from 'react-router-dom';
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

interface Payout {
  id: string;
  amount: number;
  date: string;
  status: 'completed' | 'pending' | 'failed';
  method: string;
}


function getPayoutStatusVariant(status: string) {
  switch (status) {
    case 'completed':
      return 'success' as const;
    case 'pending':
      return 'warning' as const;
    case 'failed':
      return 'error' as const;
    default:
      return 'default' as const;
  }
}

function PayoutsLoadingSkeleton() {
  return (
    <Card>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b border-mist">
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                Date
              </th>
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                Amount
              </th>
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                Status
              </th>
              <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                Method
              </th>
            </tr>
          </thead>
          <tbody>
            {Array.from({ length: 8 }).map((_, i) => (
              <tr key={i} className="border-b border-mist">
                <td className="px-4 py-3">
                  <Skeleton className="h-4 w-24" />
                </td>
                <td className="px-4 py-3">
                  <Skeleton className="h-4 w-20" />
                </td>
                <td className="px-4 py-3">
                  <Skeleton className="h-5 w-16 rounded-full" />
                </td>
                <td className="px-4 py-3">
                  <Skeleton className="h-4 w-28" />
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
  const { data: payouts, isLoading, isError } = useQuery<Payout[]>({
    queryKey: ['chef', 'earnings', 'payouts'],
    queryFn: () => apiClient.get<Payout[]>('/chef/earnings/payouts'),
  });

  return (
    <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
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
            View all your past and pending payouts
          </p>
        </motion.div>

        {/* Content */}
        <motion.div variants={fadeInUp}>
          {isLoading ? (
            <PayoutsLoadingSkeleton />
          ) : isError || !payouts ? (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <DollarSign className="mb-4 h-12 w-12 text-ink-muted" />
              <h3 className="text-lg font-semibold text-ink">
                Unable to load payouts
              </h3>
              <p className="mt-1 text-sm text-ink-muted">
                Please try again later.
              </p>
            </div>
          ) : payouts.length === 0 ? (
            <Card>
              <div className="flex flex-col items-center justify-center py-16 text-center">
                <CreditCard className="mb-4 h-12 w-12 text-ink-muted" />
                <h3 className="text-lg font-semibold text-ink">
                  No payouts yet
                </h3>
                <p className="mt-1 max-w-sm text-sm text-ink-muted">
                  Your payout history will appear here once you receive your
                  first payout.
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
                        Date
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                        Amount
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                        Status
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-wider text-ink-muted">
                        Method
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-mist">
                    {payouts.map((payout) => (
                      <tr
                        key={payout.id}
                        className="transition-colors hover:bg-paper/50"
                      >
                        <td className="whitespace-nowrap px-6 py-4 text-sm text-ink-soft">
                          {format(new Date(payout.date), 'dd MMM yyyy')}
                        </td>
                        <td className="whitespace-nowrap px-6 py-4 text-sm font-semibold text-ink">
                          {formatCurrency(payout.amount)}
                        </td>
                        <td className="whitespace-nowrap px-6 py-4">
                          <Badge
                            variant={getPayoutStatusVariant(payout.status)}
                            size="sm"
                          >
                            {payout.status}
                          </Badge>
                        </td>
                        <td className="whitespace-nowrap px-6 py-4 text-sm text-ink-muted">
                          {payout.method}
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
