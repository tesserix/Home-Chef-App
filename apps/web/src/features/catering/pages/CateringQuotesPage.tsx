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
import { apiClient } from '@/shared/services/api-client';
import type { CateringRequest, CateringQuote, PaginatedResponse } from '@/shared/types';

const STATUS_CONFIG = {
  pending: { label: 'Awaiting Quotes', color: 'bg-yellow-100 text-yellow-800' },
  quotes_received: { label: 'Quotes Received', color: 'bg-blue-100 text-blue-800' },
  booked: { label: 'Booked', color: 'bg-green-100 text-green-800' },
  in_progress: { label: 'In Progress', color: 'bg-purple-100 text-purple-800' },
  completed: { label: 'Completed', color: 'bg-gray-100 text-gray-800' },
  cancelled: { label: 'Cancelled', color: 'bg-red-100 text-red-800' },
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
        <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="container-app">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 md:text-3xl">My Catering Requests</h1>
            <p className="mt-1 text-gray-600">Manage your catering requests and quotes</p>
          </div>
          <Link to="/catering" className="btn-primary">
            <Plus className="h-4 w-4" />
            New Request
          </Link>
        </div>

        <div className="mt-8 flex flex-col gap-8 lg:flex-row">
          {/* Requests List */}
          <div className="lg:w-96">
            <div className="rounded-xl bg-white shadow-sm">
              <div className="border-b p-4">
                <h2 className="font-semibold text-gray-900">Your Requests</h2>
              </div>

              {(requests?.data ?? []).length === 0 ? (
                <div className="p-8 text-center">
                  <FileText className="mx-auto h-12 w-12 text-gray-400" />
                  <h3 className="mt-4 font-medium text-gray-900">No requests yet</h3>
                  <p className="mt-2 text-sm text-gray-600">
                    Create a catering request to receive quotes from our chefs
                  </p>
                  <Link to="/catering" className="btn-primary mt-4 inline-flex">
                    Create Request
                  </Link>
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
                            ? 'bg-brand-50'
                            : 'hover:bg-gray-50'
                        }`}
                      >
                        <div className="flex items-start justify-between">
                          <div>
                            <div className="flex items-center gap-2">
                              <Calendar className="h-4 w-4 text-gray-400" />
                              <span className="font-medium text-gray-900">
                                {new Date(request.eventDate).toLocaleDateString('en-US', {
                                  month: 'short',
                                  day: 'numeric',
                                  year: 'numeric',
                                })}
                              </span>
                            </div>
                            <div className="mt-1 flex items-center gap-2 text-sm text-gray-500">
                              <Users className="h-4 w-4" />
                              {request.guestCount} guests
                            </div>
                          </div>
                          <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${status.color}`}>
                            {status.label}
                          </span>
                        </div>
                        <div className="mt-2 flex items-center gap-2 text-sm text-gray-500">
                          <span>{request.cuisinePreferences.slice(0, 2).join(', ')}</span>
                          {request.cuisinePreferences.length > 2 && (
                            <span>+{request.cuisinePreferences.length - 2}</span>
                          )}
                        </div>
                        {request.quotesCount > 0 && (
                          <div className="mt-2 text-sm text-brand-600">
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
              <div className="rounded-xl bg-white p-12 text-center shadow-sm">
                <ChefHat className="mx-auto h-12 w-12 text-gray-400" />
                <h3 className="mt-4 font-medium text-gray-900">Select a request</h3>
                <p className="mt-2 text-sm text-gray-600">
                  Choose a request from the list to view details and quotes
                </p>
              </div>
            ) : (
              <>
                {/* Request Details */}
                {activeRequest && (
                  <div className="rounded-xl bg-white p-6 shadow-sm">
                    <div className="flex items-start justify-between">
                      <h2 className="text-lg font-semibold text-gray-900">Request Details</h2>
                      <span className={`rounded-full px-3 py-1 text-sm font-medium ${STATUS_CONFIG[activeRequest.status].color}`}>
                        {STATUS_CONFIG[activeRequest.status].label}
                      </span>
                    </div>

                    <div className="mt-6 grid gap-6 md:grid-cols-2">
                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-100">
                          <Calendar className="h-5 w-5 text-brand-600" />
                        </div>
                        <div>
                          <p className="text-sm text-gray-500">Event Date</p>
                          <p className="font-medium text-gray-900">
                            {new Date(activeRequest.eventDate).toLocaleDateString('en-US', {
                              weekday: 'long',
                              month: 'long',
                              day: 'numeric',
                              year: 'numeric',
                            })}
                          </p>
                        </div>
                      </div>

                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-100">
                          <Clock className="h-5 w-5 text-brand-600" />
                        </div>
                        <div>
                          <p className="text-sm text-gray-500">Event Time</p>
                          <p className="font-medium text-gray-900">{activeRequest.eventTime}</p>
                        </div>
                      </div>

                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-100">
                          <Users className="h-5 w-5 text-brand-600" />
                        </div>
                        <div>
                          <p className="text-sm text-gray-500">Guest Count</p>
                          <p className="font-medium text-gray-900">{activeRequest.guestCount} guests</p>
                        </div>
                      </div>

                      <div className="flex items-center gap-3">
                        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-100">
                          <MapPin className="h-5 w-5 text-brand-600" />
                        </div>
                        <div>
                          <p className="text-sm text-gray-500">Location</p>
                          <p className="font-medium text-gray-900">
                            {activeRequest.eventLocation.city}, {activeRequest.eventLocation.state}
                          </p>
                        </div>
                      </div>
                    </div>

                    <div className="mt-6 flex flex-wrap gap-2">
                      {activeRequest.cuisinePreferences.map((cuisine) => (
                        <span
                          key={cuisine}
                          className="rounded-full bg-brand-100 px-3 py-1 text-sm text-brand-700"
                        >
                          {cuisine}
                        </span>
                      ))}
                      {activeRequest.dietaryRequirements.map((diet) => (
                        <span
                          key={diet}
                          className="rounded-full bg-green-100 px-3 py-1 text-sm text-green-700"
                        >
                          {diet}
                        </span>
                      ))}
                    </div>

                    {activeRequest.description && (
                      <p className="mt-4 text-gray-600">{activeRequest.description}</p>
                    )}
                  </div>
                )}

                {/* Quotes */}
                <div className="mt-6">
                  <h2 className="text-lg font-semibold text-gray-900">
                    Quotes ({quotes?.data?.length ?? 0})
                  </h2>

                  {quotesLoading ? (
                    <div className="mt-4 flex items-center justify-center py-12">
                      <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
                    </div>
                  ) : (quotes?.data ?? []).length === 0 ? (
                    <div className="mt-4 rounded-xl bg-white p-8 text-center shadow-sm">
                      <ChefHat className="mx-auto h-12 w-12 text-gray-400" />
                      <h3 className="mt-4 font-medium text-gray-900">No quotes yet</h3>
                      <p className="mt-2 text-sm text-gray-600">
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
    <div className="rounded-xl bg-white shadow-sm overflow-hidden">
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
                <h3 className="font-semibold text-gray-900">{quote.chef?.businessName}</h3>
                <div className="mt-1 flex items-center gap-2 text-sm text-gray-500">
                  <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
                  <span>{quote.chef?.rating}</span>
                  <span>({quote.chef?.totalReviews} reviews)</span>
                </div>
              </div>
              <div className="text-right">
                <p className="text-2xl font-bold text-gray-900">
                  {fp(quote.totalPrice)}
                </p>
                <p className="text-sm text-gray-500">
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
            className="text-sm text-brand-600 hover:text-brand-700"
          >
            {showDetails ? 'Hide menu details' : 'View menu details'}
          </button>

          {showDetails && (
            <div className="mt-4 rounded-lg bg-gray-50 p-4">
              <h4 className="font-medium text-gray-900">Proposed Menu</h4>
              <ul className="mt-3 space-y-2">
                {quote.menuItems.map((item, index) => (
                  <li key={index} className="flex items-start justify-between text-sm">
                    <div>
                      <span className="text-gray-900">{item.name}</span>
                      {item.description && (
                        <p className="text-gray-500">{item.description}</p>
                      )}
                    </div>
                    <span className="text-gray-600">
                      {item.quantity}x {fp(item.pricePerUnit)}
                    </span>
                  </li>
                ))}
              </ul>
              {quote.serviceCharge && quote.serviceCharge > 0 && (
                <div className="mt-3 pt-3 border-t flex justify-between text-sm">
                  <span className="text-gray-600">Service Charge</span>
                  <span className="text-gray-900">{fp(quote.serviceCharge)}</span>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Notes */}
        {quote.notes && (
          <div className="mt-4 rounded-lg bg-blue-50 p-3">
            <p className="text-sm text-blue-800">
              <span className="font-medium">Chef's Note:</span> {quote.notes}
            </p>
          </div>
        )}

        {/* Valid Until */}
        <p className="mt-4 text-sm text-gray-500">
          Valid until {new Date(quote.validUntil).toLocaleDateString()}
        </p>

        {/* Actions */}
        {quote.status === 'pending' && (
          <div className="mt-6 flex gap-3">
            <button
              onClick={onAccept}
              disabled={isAccepting}
              className="btn-primary flex-1"
            >
              {isAccepting ? (
                <Loader2 className="h-5 w-5 animate-spin" />
              ) : (
                <>
                  <Check className="h-5 w-5" />
                  Accept Quote
                </>
              )}
            </button>
            <button
              onClick={onDecline}
              className="btn-outline flex-1"
            >
              <X className="h-5 w-5" />
              Decline
            </button>
            <button className="btn-outline">
              <MessageCircle className="h-5 w-5" />
            </button>
          </div>
        )}

        {quote.status === 'accepted' && (
          <div className="mt-6 rounded-lg bg-green-50 p-4 text-center">
            <Check className="mx-auto h-8 w-8 text-green-600" />
            <p className="mt-2 font-medium text-green-800">Quote Accepted!</p>
            <p className="text-sm text-green-600">The chef will contact you soon to finalize details.</p>
          </div>
        )}

        {quote.status === 'declined' && (
          <div className="mt-6 rounded-lg bg-gray-100 p-4 text-center">
            <p className="text-sm text-gray-600">You declined this quote</p>
          </div>
        )}
      </div>
    </div>
  );
}
