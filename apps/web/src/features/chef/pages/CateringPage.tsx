import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Calendar,
  Users,
  MapPin,
  DollarSign,
  Clock,
  Send,
  CheckCircle,
  XCircle,
  Loader2,
  ChefHat,
} from 'lucide-react';
import { toast } from 'sonner';
import { useFormatPrice } from '@/shared/utils/format-price';
import { apiClient } from '@/shared/services/api-client';
import type { CateringRequest, CateringQuote, PaginatedResponse } from '@/shared/types';

const TABS = [
  { value: 'requests', label: 'Open Requests' },
  { value: 'my-quotes', label: 'My Quotes' },
  { value: 'booked', label: 'Booked Events' },
];

export default function ChefCateringPage() {
  const [activeTab, setActiveTab] = useState('requests');
  const [selectedRequest, setSelectedRequest] = useState<CateringRequest | null>(null);
  const [showQuoteForm, setShowQuoteForm] = useState(false);
  const queryClient = useQueryClient();

  const { data: requests, isLoading: requestsLoading } = useQuery({
    queryKey: ['chef-catering-requests'],
    queryFn: () => apiClient.get<PaginatedResponse<CateringRequest>>('/chef/catering/requests'),
    enabled: activeTab === 'requests',
  });

  const { data: myQuotes, isLoading: quotesLoading } = useQuery({
    queryKey: ['chef-catering-quotes'],
    queryFn: () => apiClient.get<PaginatedResponse<CateringQuote>>('/chef/catering/quotes'),
    enabled: activeTab === 'my-quotes' || activeTab === 'booked',
  });

  const submitQuoteMutation = useMutation({
    mutationFn: (data: { requestId: string; quote: QuoteFormData }) =>
      apiClient.post(`/chef/catering/requests/${data.requestId}/quote`, data.quote),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-catering'] });
      toast.success('Quote submitted successfully');
      setShowQuoteForm(false);
      setSelectedRequest(null);
    },
    onError: () => {
      toast.error('Failed to submit quote');
    },
  });

  const openRequests = requests?.data ?? [];
  const pendingQuotes = (myQuotes?.data ?? []).filter((q) => q.status === 'pending');
  const bookedEvents = (myQuotes?.data ?? []).filter((q) => q.status === 'accepted');

  const isLoading = activeTab === 'requests' ? requestsLoading : quotesLoading;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Catering</h1>
        <p className="mt-1 text-gray-600">
          Browse catering requests and submit quotes
        </p>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 border-b">
        {TABS.map((tab) => (
          <button
            key={tab.value}
            onClick={() => setActiveTab(tab.value)}
            className={`relative px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === tab.value
                ? 'text-brand-600'
                : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            {tab.label}
            {activeTab === tab.value && (
              <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-brand-500" />
            )}
            {tab.value === 'requests' && openRequests.length > 0 && (
              <span className="ml-2 rounded-full bg-brand-100 px-2 py-0.5 text-xs text-brand-700">
                {openRequests.length}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
        </div>
      ) : activeTab === 'requests' ? (
        <RequestsList
          requests={openRequests}
          onSelectRequest={(request) => {
            setSelectedRequest(request);
            setShowQuoteForm(true);
          }}
        />
      ) : activeTab === 'my-quotes' ? (
        <QuotesList quotes={pendingQuotes} />
      ) : (
        <BookedEventsList events={bookedEvents} />
      )}

      {/* Quote Form Modal */}
      {showQuoteForm && selectedRequest && (
        <QuoteFormModal
          request={selectedRequest}
          onClose={() => {
            setShowQuoteForm(false);
            setSelectedRequest(null);
          }}
          onSubmit={(quote) => {
            submitQuoteMutation.mutate({
              requestId: selectedRequest.id,
              quote,
            });
          }}
          isSubmitting={submitQuoteMutation.isPending}
        />
      )}
    </div>
  );
}

function RequestsList({
  requests,
  onSelectRequest,
}: {
  requests: CateringRequest[];
  onSelectRequest: (request: CateringRequest) => void;
}) {
  const fp = useFormatPrice();

  if (requests.length === 0) {
    return (
      <div className="rounded-xl bg-white p-12 text-center shadow-sm">
        <ChefHat className="mx-auto h-12 w-12 text-gray-400" />
        <h3 className="mt-4 font-medium text-gray-900">No open requests</h3>
        <p className="mt-2 text-gray-600">
          New catering requests will appear here
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {requests.map((request) => (
        <div key={request.id} className="rounded-xl bg-white p-6 shadow-sm">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <div className="flex items-center gap-2">
                <Calendar className="h-5 w-5 text-brand-500" />
                <span className="font-semibold text-gray-900">
                  {new Date(request.eventDate).toLocaleDateString('en-US', {
                    weekday: 'long',
                    month: 'long',
                    day: 'numeric',
                    year: 'numeric',
                  })}
                </span>
              </div>
              <p className="mt-1 text-sm text-gray-500">at {request.eventTime}</p>
            </div>
            <span className="rounded-full bg-yellow-100 px-3 py-1 text-sm font-medium text-yellow-800">
              {request.quotesCount} quotes received
            </span>
          </div>

          <div className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div className="flex items-center gap-2 text-sm text-gray-600">
              <Users className="h-4 w-4" />
              {request.guestCount} guests
            </div>
            <div className="flex items-center gap-2 text-sm text-gray-600">
              <MapPin className="h-4 w-4" />
              {request.eventLocation.city}, {request.eventLocation.state}
            </div>
            {request.budgetMin && request.budgetMax && (
              <div className="flex items-center gap-2 text-sm text-gray-600">
                <DollarSign className="h-4 w-4" />
                {fp(request.budgetMin)} - {fp(request.budgetMax)}
              </div>
            )}
            <div className="flex items-center gap-2 text-sm text-gray-600">
              <Clock className="h-4 w-4" />
              {request.serviceType.replace('_', ' ')}
            </div>
          </div>

          <div className="mt-4 flex flex-wrap gap-2">
            {request.cuisinePreferences.map((cuisine) => (
              <span
                key={cuisine}
                className="rounded-full bg-brand-100 px-3 py-1 text-sm text-brand-700"
              >
                {cuisine}
              </span>
            ))}
            {request.dietaryRequirements.map((diet) => (
              <span
                key={diet}
                className="rounded-full bg-green-100 px-3 py-1 text-sm text-green-700"
              >
                {diet}
              </span>
            ))}
          </div>

          {request.description && (
            <p className="mt-4 text-sm text-gray-600">{request.description}</p>
          )}

          <div className="mt-6 flex gap-3">
            <button
              onClick={() => onSelectRequest(request)}
              className="btn-primary"
            >
              <Send className="h-4 w-4" />
              Submit Quote
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}

function QuotesList({ quotes }: { quotes: CateringQuote[] }) {
  const fp = useFormatPrice();

  if (quotes.length === 0) {
    return (
      <div className="rounded-xl bg-white p-12 text-center shadow-sm">
        <Send className="mx-auto h-12 w-12 text-gray-400" />
        <h3 className="mt-4 font-medium text-gray-900">No pending quotes</h3>
        <p className="mt-2 text-gray-600">
          Your submitted quotes will appear here
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {quotes.map((quote) => (
        <div key={quote.id} className="rounded-xl bg-white p-6 shadow-sm">
          <div className="flex items-start justify-between">
            <div>
              <p className="font-semibold text-gray-900">
                Quote for {quote.menuItems.length} items
              </p>
              <p className="text-sm text-gray-500">
                Submitted {new Date(quote.createdAt).toLocaleDateString()}
              </p>
            </div>
            <div className="text-right">
              <p className="text-xl font-bold text-gray-900">
                {fp(quote.totalPrice)}
              </p>
              <p className="text-sm text-gray-500">
                {fp(quote.pricePerPerson)}/person
              </p>
            </div>
          </div>

          <div className="mt-4 rounded-lg bg-gray-50 p-3">
            <p className="text-sm font-medium text-gray-700">Menu Items:</p>
            <ul className="mt-2 space-y-1 text-sm text-gray-600">
              {quote.menuItems.map((item, i) => (
                <li key={i}>
                  {item.quantity}x {item.name} - {fp(item.pricePerUnit)} each
                </li>
              ))}
            </ul>
          </div>

          <div className="mt-4 flex items-center justify-between">
            <span className="rounded-full bg-yellow-100 px-3 py-1 text-sm font-medium text-yellow-800">
              Awaiting Response
            </span>
            <p className="text-sm text-gray-500">
              Valid until {new Date(quote.validUntil).toLocaleDateString()}
            </p>
          </div>
        </div>
      ))}
    </div>
  );
}

function BookedEventsList({ events }: { events: CateringQuote[] }) {
  const fp = useFormatPrice();

  if (events.length === 0) {
    return (
      <div className="rounded-xl bg-white p-12 text-center shadow-sm">
        <CheckCircle className="mx-auto h-12 w-12 text-gray-400" />
        <h3 className="mt-4 font-medium text-gray-900">No booked events</h3>
        <p className="mt-2 text-gray-600">
          Your confirmed catering events will appear here
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {events.map((event) => (
        <div key={event.id} className="rounded-xl bg-white p-6 shadow-sm ring-2 ring-green-200">
          <div className="flex items-center gap-2 text-green-600 mb-4">
            <CheckCircle className="h-5 w-5" />
            <span className="font-medium">Confirmed Booking</span>
          </div>

          <div className="flex items-start justify-between">
            <div>
              <p className="font-semibold text-gray-900">
                {event.menuItems.length} items for event
              </p>
              <p className="text-sm text-gray-500">
                Booked {new Date(event.createdAt).toLocaleDateString()}
              </p>
            </div>
            <div className="text-right">
              <p className="text-xl font-bold text-gray-900">
                {fp(event.totalPrice)}
              </p>
            </div>
          </div>

          <div className="mt-4 rounded-lg bg-green-50 p-3">
            <p className="text-sm font-medium text-green-800">
              Customer has accepted your quote. Contact details will be provided.
            </p>
          </div>
        </div>
      ))}
    </div>
  );
}

interface QuoteFormData {
  menuItems: Array<{ name: string; description?: string; quantity: number; pricePerUnit: number }>;
  notes?: string;
  validDays: number;
}

function QuoteFormModal({
  request,
  onClose,
  onSubmit,
  isSubmitting,
}: {
  request: CateringRequest;
  onClose: () => void;
  onSubmit: (data: QuoteFormData) => void;
  isSubmitting: boolean;
}) {
  const fp = useFormatPrice();
  const [menuItems, setMenuItems] = useState([
    { name: '', description: '', quantity: request.guestCount, pricePerUnit: 0 },
  ]);
  const [notes, setNotes] = useState('');
  const [validDays, setValidDays] = useState(7);

  const addMenuItem = () => {
    setMenuItems([
      ...menuItems,
      { name: '', description: '', quantity: request.guestCount, pricePerUnit: 0 },
    ]);
  };

  const updateMenuItem = (index: number, field: string, value: string | number) => {
    const updated = [...menuItems];
    (updated[index] as Record<string, string | number>)[field] = value;
    setMenuItems(updated);
  };

  const removeMenuItem = (index: number) => {
    setMenuItems(menuItems.filter((_, i) => i !== index));
  };

  const totalPrice = menuItems.reduce(
    (sum, item) => sum + item.quantity * item.pricePerUnit,
    0
  );
  const pricePerPerson = totalPrice / request.guestCount;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      menuItems: menuItems.filter((item) => item.name && item.pricePerUnit > 0),
      notes: notes || undefined,
      validDays,
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/50 p-4 pt-10">
      <div className="w-full max-w-2xl rounded-xl bg-white shadow-xl">
        <div className="border-b p-6">
          <h2 className="text-xl font-semibold text-gray-900">Submit Quote</h2>
          <p className="mt-1 text-sm text-gray-500">
            For {request.guestCount} guests on{' '}
            {new Date(request.eventDate).toLocaleDateString()}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-6">
          {/* Menu Items */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-3">
              Menu Items
            </label>
            <div className="space-y-3">
              {menuItems.map((item, index) => (
                <div key={index} className="flex gap-3 items-start">
                  <div className="flex-1 grid gap-2 sm:grid-cols-3">
                    <input
                      type="text"
                      value={item.name}
                      onChange={(e) => updateMenuItem(index, 'name', e.target.value)}
                      placeholder="Item name"
                      className="input-base"
                      required
                    />
                    <input
                      type="number"
                      value={item.quantity}
                      onChange={(e) => updateMenuItem(index, 'quantity', Number(e.target.value))}
                      placeholder="Qty"
                      className="input-base"
                      required
                      min={1}
                    />
                    <div className="relative">
                      <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400">
                        $
                      </span>
                      <input
                        type="number"
                        value={item.pricePerUnit || ''}
                        onChange={(e) =>
                          updateMenuItem(index, 'pricePerUnit', Number(e.target.value))
                        }
                        placeholder="Price/unit"
                        className="input-base pl-7"
                        required
                        min={0}
                        step={0.01}
                      />
                    </div>
                  </div>
                  {menuItems.length > 1 && (
                    <button
                      type="button"
                      onClick={() => removeMenuItem(index)}
                      className="p-2 text-red-500 hover:bg-red-50 rounded"
                    >
                      <XCircle className="h-5 w-5" />
                    </button>
                  )}
                </div>
              ))}
            </div>
            <button
              type="button"
              onClick={addMenuItem}
              className="mt-3 text-sm text-brand-600 hover:text-brand-700"
            >
              + Add another item
            </button>
          </div>

          {/* Summary */}
          <div className="rounded-lg bg-gray-50 p-4">
            <div className="flex justify-between text-sm">
              <span className="text-gray-600">Total Price</span>
              <span className="font-semibold text-gray-900">{fp(totalPrice)}</span>
            </div>
            <div className="flex justify-between text-sm mt-1">
              <span className="text-gray-600">Price per person</span>
              <span className="font-medium text-gray-900">{fp(pricePerPerson)}</span>
            </div>
          </div>

          {/* Notes */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Notes for Customer (Optional)
            </label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
              placeholder="Any special notes about your quote..."
              className="input-base"
            />
          </div>

          {/* Valid Days */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Quote Valid For
            </label>
            <select
              value={validDays}
              onChange={(e) => setValidDays(Number(e.target.value))}
              className="input-base"
            >
              <option value={3}>3 days</option>
              <option value={7}>7 days</option>
              <option value={14}>14 days</option>
            </select>
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-3 pt-4 border-t">
            <button type="button" onClick={onClose} className="btn-outline">
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting || menuItems.every((i) => !i.name)}
              className="btn-primary"
            >
              {isSubmitting ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <>
                  <Send className="h-4 w-4" />
                  Submit Quote
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
