import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Calendar,
  Clock,
  Users,
  MapPin,
  Star,
  ChefHat,
  Check,
  X,
  MessageCircle,
  Loader2,
  Plus,
  FileText,
} from 'lucide-react';
import { toast } from 'sonner';
import { useFormatPrice } from '@/shared/utils/format-price';
import { formatDate } from '@/shared/utils/format-date';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui';
import type { CateringRequest, CateringQuote, PaginatedResponse } from '@/shared/types';

const STATUS_CONFIG = {
  pending: { label: 'Awaiting Quotes', color: 'bg-amber-tint text-amber' },
  quotes_received: { label: 'Quotes Received', color: 'bg-info/10 text-info' },
  booked: { label: 'Booked', color: 'bg-herb-tint text-herb' },
  in_progress: { label: 'In Progress', color: 'bg-info/10 text-info' },
  completed: { label: 'Completed', color: 'bg-mist text-ink-soft' },
  cancelled: { label: 'Cancelled', color: 'bg-paprika-tint text-paprika' },
};

export default function CateringQuotesPage() {
  const [selectedRequest, setSelectedRequest] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const { data: requests, isLoading } = useQuery({
    queryKey: ['catering-requests'],
    queryFn: () => apiClient.get<PaginatedResponse<CateringRequest>>('/catering/requests'),
  });

  const { data: quotes, isLoading: quotesLoading } = useQuery({
    queryKey: ['catering-quotes', selectedRequest],
    queryFn: () =>
      apiClient.get<PaginatedResponse<CateringQuote>>(
        `/catering/requests/${selectedRequest}/quotes`
      ),
    enabled: !!selectedRequest,
  });

  const acceptQuoteMutation = useMutation({
    mutationFn: (quoteId: string) =>
      apiClient.post(`/catering/quotes/${quoteId}/accept`),
    onSuccess: () => {
      toast.success('Quote accepted! The chef has been notified.');
      queryClient.invalidateQueries({ queryKey: ['catering-requests'] });
      queryClient.invalidateQueries({ queryKey: ['catering-quotes'] });
    },
    onError: () => {
      toast.error('Failed to accept quote. Please try again.');
    },
  });

  const declineQuoteMutation = useMutation({
    mutationFn: (quoteId: string) =>
      apiClient.post(`/catering/quotes/${quoteId}/decline`),
    onSuccess: () => {
      toast.success('Quote declined.');
      queryClient.invalidateQueries({ queryKey: ['catering-quotes'] });
    },
  });

  const activeRequest = (requests?.data ?? []).find((r) => r.id === selectedRequest);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-paper py-8">
      <div className="container-app">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="font-display text-2xl font-semibold text-ink md:text-3xl">My Catering Requests</h1>
            <p className="mt-1 text-ink-soft">Manage your catering requests and quotes</p>
          </div>
          <Button asChild variant="primary" leftIcon={<Plus aria-hidden="true" className="h-4 w-4" />}>
            <Link to="/catering">New Request</Link>
          </Button>
        </div>

        <div className="mt-8 flex flex-col gap-8 lg:flex-row">
          {/* Requests List */}
          <div className="lg:w-96">
            <div className="rounded-xl bg-bone shadow-1">
              <div className="border-b p-4">
                <h2 className="font-semibold text-ink">Your Requests</h2>
              </div>

              {(requests?.data ?? []).length === 0 ? (
                <div className="p-8 text-center">
                  <FileText className="mx-auto h-12 w-12 text-ink-muted" />
                  <h3 className="mt-4 font-medium text-ink">No requests yet</h3>
                  <p className="mt-2 text-sm text-ink-soft">
                    Create a catering request to receive quotes from our chefs
                  </p>
                  <Button asChild variant="primary" className="mt-4">
                    <Link to="/catering">Create Request</Link>
                  </Button>
                </div>
              ) : (
                <div className="divide-y">
                  {(requests?.data ?? []).map((request) => {
                    const status = STATUS_CONFIG[request.status];
                    return (
                      <button
                        key={request.id}
                        onClick={() => setSelectedRequest(request.id)}
                        className={`w-full p-4 text-left transition-colors ${
                          selectedRequest === request.id
                            ? 'bg-herb-tint'
                            : 'hover:bg-paper'
                        }`}
                      >
                        <div className="flex items-start justify-between">
                          <div>
                            <div className="flex items-center gap-2">
                              <Calendar className="h-4 w-4 text-ink-muted" />
                              <span className="font-medium text-ink">
                                {formatDate(request.eventDate)}
                              </span>
                            </div>
                            <div className="mt-1 flex items-center gap-2 text-sm text-ink-muted">
                              <Users className="h-4 w-4" />
                              {request.guestCount} guests
                            </div>
                          </div>
                          <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${status.color}`}>
                            {status.label}
                          </span>
                        </div>
                        <div className="mt-2 flex items-center gap-2 text-sm text-ink-muted">
                          <span>{request.cuisinePreferences.slice(0, 2).join(', ')}</span>
                          {request.cuisinePreferences.length > 2 && (
                            <span>+{request.cuisinePreferences.length - 2}</span>
                          )}
                        </div>
                        {request.quotesCount > 0 && (
                          <div className="mt-2 text-sm text-herb">
                            {request.quotesCount} quote{request.quotesCount > 1 ? 's' : ''} received
                          </div>
                        )}
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
          </div>

          {/* Request Details & Quotes */}
          <div className="flex-1">
            {!selectedRequest ? (
              <div className="rounded-xl bg-bone p-12 text-center shadow-1">
                <ChefHat className="mx-auto h-12 w-12 text-ink-muted" />
                <h3 className="mt-4 font-medium text-ink">Select a request</h3>
                <p className="mt-2 text-sm text-ink-soft">
                  Choose a request from the list to view details and quotes
                </p>
              </div>
            ) : (
              <>
                {/* Request Details */}
                {activeRequest && (
                  <div className="rounded-xl bg-bone p-6 shadow-1">
                    <div className="flex items-start justify-between">
                      <h2 className="text-lg font-semibold text-ink">Request Details</h2>
                      <span className={`rounded-full px-3 py-1 text-sm font-medium ${STATUS_CONFIG[activeRequest.status].color}`}>
                        {STATUS_CONFIG[activeRequest.status].label}
                      </span>
                    </div>

                    <div className="mt-6 grid gap-6 md:grid-cols-2">
                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-herb-tint">
                          <Calendar className="h-5 w-5 text-herb" />
                        </div>
                        <div>
                          <p className="text-sm text-ink-muted">Event Date</p>
                          <p className="font-medium text-ink">
                            {formatDate(activeRequest.eventDate, {
                              weekday: 'long',
                              month: 'long',
                              day: 'numeric',
                              year: 'numeric',
                            })}
                          </p>
                        </div>
                      </div>

                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-herb-tint">
                          <Clock className="h-5 w-5 text-herb" />
                        </div>
                        <div>
                          <p className="text-sm text-ink-muted">Event Time</p>
                          <p className="font-medium text-ink">{activeRequest.eventTime}</p>
                        </div>
                      </div>

                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-herb-tint">
                          <Users className="h-5 w-5 text-herb" />
                        </div>
                        <div>
                          <p className="text-sm text-ink-muted">Guest Count</p>
                          <p className="font-medium text-ink">{activeRequest.guestCount} guests</p>
                        </div>
                      </div>

                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-herb-tint">
                          <MapPin className="h-5 w-5 text-herb" />
                        </div>
                        <div>
                          <p className="text-sm text-ink-muted">Location</p>
                          <p className="font-medium text-ink">
                            {activeRequest.eventLocation.city}, {activeRequest.eventLocation.state}
                          </p>
                        </div>
                      </div>
                    </div>

                    <div className="mt-6 flex flex-wrap gap-2">
                      {activeRequest.cuisinePreferences.map((cuisine) => (
                        <span
                          key={cuisine}
                          className="rounded-full bg-herb-tint px-3 py-1 text-sm text-herb"
                        >
                          {cuisine}
                        </span>
                      ))}
                      {activeRequest.dietaryRequirements.map((diet) => (
                        <span
                          key={diet}
                          className="rounded-full bg-herb-tint px-3 py-1 text-sm text-herb"
                        >
                          {diet}
                        </span>
                      ))}
                    </div>

                    {activeRequest.description && (
                      <p className="mt-4 text-ink-soft">{activeRequest.description}</p>
                    )}
                  </div>
                )}

                {/* Quotes */}
                <div className="mt-6">
                  <h2 className="text-lg font-semibold text-ink">
                    Quotes ({quotes?.data?.length ?? 0})
                  </h2>

                  {quotesLoading ? (
                    <div className="mt-4 flex items-center justify-center py-12">
                      <Loader2 className="h-8 w-8 animate-spin text-herb" />
                    </div>
                  ) : (quotes?.data ?? []).length === 0 ? (
                    <div className="mt-4 rounded-xl bg-bone p-8 text-center shadow-1">
                      <ChefHat className="mx-auto h-12 w-12 text-ink-muted" />
                      <h3 className="mt-4 font-medium text-ink">No quotes yet</h3>
                      <p className="mt-2 text-sm text-ink-soft">
                        Our chefs are reviewing your request. You'll receive quotes soon!
                      </p>
                    </div>
                  ) : (
                    <div className="mt-4 space-y-4">
                      {(quotes?.data ?? []).map((quote) => (
                        <QuoteCard
                          key={quote.id}
                          quote={quote}
                          onAccept={() => acceptQuoteMutation.mutate(quote.id)}
                          onDecline={() => declineQuoteMutation.mutate(quote.id)}
                          isAccepting={acceptQuoteMutation.isPending}
                        />
                      ))}
                    </div>
                  )}
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function QuoteCard({
  quote,
  onAccept,
  onDecline,
  isAccepting,
}: {
  quote: CateringQuote;
  onAccept: () => void;
  onDecline: () => void;
  isAccepting: boolean;
}) {
  const [showDetails, setShowDetails] = useState(false);
  const fp = useFormatPrice();

  return (
    <div className="rounded-xl bg-bone shadow-1 overflow-hidden">
      <div className="p-6">
        <div className="flex items-start gap-4">
          {/* Chef Info */}
          <img
            src={quote.chef?.profileImage || '/placeholder-chef.png'}
            alt={quote.chef?.businessName}
            className="h-14 w-14 rounded-xl object-cover"
          />
          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between">
              <div>
                <h3 className="font-semibold text-ink">{quote.chef?.businessName}</h3>
                <div className="mt-1 flex items-center gap-2 text-sm text-ink-muted">
                  <Star className="h-4 w-4 fill-amber text-amber" />
                  <span>{quote.chef?.rating}</span>
                  <span>({quote.chef?.totalReviews} reviews)</span>
                </div>
              </div>
              <div className="text-right">
                <p className="font-display text-2xl font-semibold text-ink">
                  {fp(quote.totalPrice)}
                </p>
                <p className="text-sm text-ink-muted">
                  {fp(quote.pricePerPerson)} per person
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Menu Preview */}
        <div className="mt-4">
          <button
            onClick={() => setShowDetails(!showDetails)}
            className="text-sm text-herb hover:text-herb"
          >
            {showDetails ? 'Hide menu details' : 'View menu details'}
          </button>

          {showDetails && (
            <div className="mt-4 rounded-lg bg-paper p-4">
              <h4 className="font-medium text-ink">Proposed Menu</h4>
              <ul className="mt-3 space-y-2">
                {quote.menuItems.map((item, index) => (
                  <li key={index} className="flex items-start justify-between text-sm">
                    <div>
                      <span className="text-ink">{item.name}</span>
                      {item.description && (
                        <p className="text-ink-muted">{item.description}</p>
                      )}
                    </div>
                    <span className="text-ink-soft">
                      {item.quantity}x {fp(item.pricePerUnit)}
                    </span>
                  </li>
                ))}
              </ul>
              {quote.serviceCharge && quote.serviceCharge > 0 && (
                <div className="mt-3 pt-3 border-t flex justify-between text-sm">
                  <span className="text-ink-soft">Service Charge</span>
                  <span className="text-ink">{fp(quote.serviceCharge)}</span>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Notes */}
        {quote.notes && (
          <div className="mt-4 rounded-lg bg-info/10 p-3">
            <p className="text-sm text-info">
              <span className="font-medium">Chef's Note:</span> {quote.notes}
            </p>
          </div>
        )}

        {/* Valid Until */}
        <p className="mt-4 text-sm text-ink-muted">
          Valid until {formatDate(quote.validUntil)}
        </p>

        {/* Actions */}
        {quote.status === 'pending' && (
          <div className="mt-6 flex gap-3">
            <Button
              variant="primary"
              fullWidth
              isLoading={isAccepting}
              disabled={isAccepting}
              onClick={onAccept}
              leftIcon={!isAccepting ? <Check aria-hidden="true" className="h-5 w-5" /> : undefined}
              className="flex-1"
            >
              {isAccepting ? 'Accepting…' : 'Accept Quote'}
            </Button>
            <Button
              variant="outline"
              fullWidth
              onClick={onDecline}
              leftIcon={<X aria-hidden="true" className="h-5 w-5" />}
              className="flex-1"
            >
              Decline
            </Button>
            <Button variant="outline" size="icon" aria-label="Message chef">
              <MessageCircle aria-hidden="true" className="h-5 w-5" />
            </Button>
          </div>
        )}

        {quote.status === 'accepted' && (
          <div className="mt-6 rounded-lg bg-herb-tint p-4 text-center">
            <Check className="mx-auto h-8 w-8 text-herb" />
            <p className="mt-2 font-medium text-herb">Quote Accepted!</p>
            <p className="text-sm text-herb">The chef will contact you soon to finalize details.</p>
          </div>
        )}

        {quote.status === 'declined' && (
          <div className="mt-6 rounded-lg bg-mist p-4 text-center">
            <p className="text-sm text-ink-soft">You declined this quote</p>
          </div>
        )}
      </div>
    </div>
  );
}
