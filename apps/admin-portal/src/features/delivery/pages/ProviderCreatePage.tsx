import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import {
  ArrowLeft,
  Loader2,
  Plus,
  Trash2,
  Save,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';

// ---------- Types ----------

interface CreateProviderResponse {
  id: string;
  name: string;
  code: string;
}

interface StatusMappingEntry {
  providerStatus: string;
  fe3drStatus: string;
}

// ---------- Constants ----------

const FE3DR_STATUSES = [
  'pending',
  'assigned',
  'at_pickup',
  'picked_up',
  'in_transit',
  'at_dropoff',
  'delivered',
  'failed',
  'cancelled',
];

const PRICING_MODELS = [
  { value: 'per_delivery', label: 'Per Delivery' },
  { value: 'per_km', label: 'Per KM' },
  { value: 'flat_rate', label: 'Flat Rate' },
];

const CURRENCIES = ['INR', 'USD', 'EUR', 'GBP', 'AED'];

// ---------- Helpers ----------

function toCode(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, '')
    .replace(/\s+/g, '_')
    .replace(/-+/g, '_')
    .replace(/_{2,}/g, '_')
    .replace(/^_|_$/g, '');
}

// ---------- Main Component ----------

export default function ProviderCreatePage() {
  const navigate = useNavigate();

  // Basic Info
  const [name, setName] = useState('');
  const [code, setCode] = useState('');
  const [codeManuallySet, setCodeManuallySet] = useState(false);
  const [description, setDescription] = useState('');
  const [logoUrl, setLogoUrl] = useState('');

  // API Configuration
  const [apiBaseUrl, setApiBaseUrl] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [apiSecret, setApiSecret] = useState('');
  const [webhookSecret, setWebhookSecret] = useState('');

  // Status Mapping
  const [statusMappings, setStatusMappings] = useState<StatusMappingEntry[]>([]);

  // Coverage
  const [countriesInput, setCountriesInput] = useState('');
  const [citiesInput, setCitiesInput] = useState('');
  const [countries, setCountries] = useState<string[]>([]);
  const [cities, setCities] = useState<string[]>([]);
  const [maxDistance, setMaxDistance] = useState<number>(50);

  // Pricing
  const [pricingModel, setPricingModel] = useState('per_delivery');
  const [baseCost, setBaseCost] = useState<number>(0);
  const [perKmCost, setPerKmCost] = useState<number>(0);
  const [currency, setCurrency] = useState('INR');

  // Rate Limits
  const [maxConcurrentDeliveries, setMaxConcurrentDeliveries] = useState<number>(100);
  const [dailyLimit, setDailyLimit] = useState<number>(0);
  const [avgPickupTime, setAvgPickupTime] = useState<number>(15);

  // Contact
  const [contactName, setContactName] = useState('');
  const [contactEmail, setContactEmail] = useState('');
  const [contactPhone, setContactPhone] = useState('');
  const [notes, setNotes] = useState('');

  const createMutation = useMutation({
    mutationFn: (body: Record<string, unknown>) =>
      apiClient.post<CreateProviderResponse>('/admin/delivery/providers', body),
    onSuccess: (data) => {
      const resp = data as unknown as CreateProviderResponse;
      toast.success('Provider created successfully');
      navigate(`/delivery/providers/${resp.id}`);
    },
    onError: () => toast.error('Failed to create provider'),
  });

  const handleNameChange = (val: string) => {
    setName(val);
    if (!codeManuallySet) {
      setCode(toCode(val));
    }
  };

  const handleCodeChange = (val: string) => {
    setCodeManuallySet(true);
    setCode(val);
  };

  // Tag management
  const addCountry = () => {
    const trimmed = countriesInput.trim();
    if (trimmed && !countries.includes(trimmed)) {
      setCountries([...countries, trimmed]);
    }
    setCountriesInput('');
  };

  const removeCountry = (c: string) => {
    setCountries(countries.filter((x) => x !== c));
  };

  const addCity = () => {
    const trimmed = citiesInput.trim();
    if (trimmed && !cities.includes(trimmed)) {
      setCities([...cities, trimmed]);
    }
    setCitiesInput('');
  };

  const removeCity = (c: string) => {
    setCities(cities.filter((x) => x !== c));
  };

  // Status mapping management
  const addStatusMapping = () => {
    setStatusMappings([...statusMappings, { providerStatus: '', fe3drStatus: 'pending' }]);
  };

  const updateStatusMapping = (index: number, field: keyof StatusMappingEntry, value: string) => {
    setStatusMappings((prev) =>
      prev.map((m, i) =>
        i === index
          ? { providerStatus: field === 'providerStatus' ? value : m.providerStatus, fe3drStatus: field === 'fe3drStatus' ? value : m.fe3drStatus }
          : m
      )
    );
  };

  const removeStatusMapping = (index: number) => {
    setStatusMappings(statusMappings.filter((_, i) => i !== index));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!name || !code || !apiBaseUrl) {
      toast.error('Please fill in required fields');
      return;
    }

    const statusMapping: Record<string, string> = {};
    statusMappings.forEach((m) => {
      if (m.providerStatus.trim()) {
        statusMapping[m.providerStatus.trim()] = m.fe3drStatus;
      }
    });

    createMutation.mutate({
      name,
      code,
      description,
      logoUrl,
      apiBaseUrl,
      apiKey,
      apiSecret,
      webhookSecret,
      statusMapping,
      supportedCountries: countries,
      supportedCities: cities,
      maxDistance,
      pricingModel,
      baseCost,
      perKmCost,
      currency,
      maxConcurrentDeliveries,
      dailyLimit,
      avgPickupTime,
      contactName,
      contactEmail,
      contactPhone,
      notes,
    });
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button
          onClick={() => navigate('/delivery/providers')}
          className="rounded-lg p-2 hover:bg-secondary transition-colors"
        >
          <ArrowLeft className="h-5 w-5 text-muted-foreground" />
        </button>
        <div>
          <h1 className="text-2xl font-bold text-foreground">Add Delivery Provider</h1>
          <p className="text-sm text-muted-foreground">Configure a new delivery provider integration</p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic Info */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-card">
          <h2 className="text-lg font-semibold text-foreground mb-4">Basic Info</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">
                Name <span className="text-destructive">*</span>
              </label>
              <input
                type="text"
                required
                value={name}
                onChange={(e) => handleNameChange(e.target.value)}
                placeholder="e.g. Dunzo"
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">
                Code <span className="text-destructive">*</span>
              </label>
              <input
                type="text"
                required
                value={code}
                onChange={(e) => handleCodeChange(e.target.value)}
                placeholder="e.g. dunzo"
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground font-mono placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
              <p className="text-xs text-muted-foreground mt-1">Auto-generated from name. Edit to customize.</p>
            </div>
            <div className="sm:col-span-2">
              <label className="block text-sm font-medium text-foreground mb-1.5">Description</label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Brief description of this delivery provider..."
                rows={2}
                className="w-full rounded-lg border border-input bg-card px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring resize-none"
              />
            </div>
            <div className="sm:col-span-2">
              <label className="block text-sm font-medium text-foreground mb-1.5">Logo URL</label>
              <input
                type="url"
                value={logoUrl}
                onChange={(e) => setLogoUrl(e.target.value)}
                placeholder="https://example.com/logo.png"
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
          </div>
        </div>

        {/* API Configuration */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-card">
          <h2 className="text-lg font-semibold text-foreground mb-4">API Configuration</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="sm:col-span-2">
              <label className="block text-sm font-medium text-foreground mb-1.5">
                API Base URL <span className="text-destructive">*</span>
              </label>
              <input
                type="url"
                required
                value={apiBaseUrl}
                onChange={(e) => setApiBaseUrl(e.target.value)}
                placeholder="https://api.provider.com/v1"
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground font-mono placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">API Key</label>
              <input
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder="Enter API key"
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">API Secret</label>
              <input
                type="password"
                value={apiSecret}
                onChange={(e) => setApiSecret(e.target.value)}
                placeholder="Enter API secret"
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div className="sm:col-span-2">
              <label className="block text-sm font-medium text-foreground mb-1.5">Webhook Secret</label>
              <input
                type="password"
                value={webhookSecret}
                onChange={(e) => setWebhookSecret(e.target.value)}
                placeholder="Enter webhook secret"
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
          </div>
        </div>

        {/* Status Mapping */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-card">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-foreground">Status Mapping</h2>
            <button
              type="button"
              onClick={addStatusMapping}
              className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm font-medium hover:bg-secondary transition-colors"
            >
              <Plus className="h-3.5 w-3.5" />
              Add Mapping
            </button>
          </div>
          {statusMappings.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No status mappings configured. Add mappings to translate provider statuses to Fe3dr statuses.
            </p>
          ) : (
            <div className="space-y-3">
              {statusMappings.map((mapping, index) => (
                <div key={index} className="flex items-center gap-3">
                  <input
                    type="text"
                    value={mapping.providerStatus}
                    onChange={(e) => updateStatusMapping(index, 'providerStatus', e.target.value)}
                    placeholder="Provider status"
                    className="h-10 flex-1 rounded-lg border border-input bg-card px-3 text-sm text-foreground font-mono placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
                  />
                  <span className="text-sm text-muted-foreground shrink-0">maps to</span>
                  <select
                    value={mapping.fe3drStatus}
                    onChange={(e) => updateStatusMapping(index, 'fe3drStatus', e.target.value)}
                    className="h-10 flex-1 rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
                  >
                    {FE3DR_STATUSES.map((s) => (
                      <option key={s} value={s}>
                        {s.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())}
                      </option>
                    ))}
                  </select>
                  <button
                    type="button"
                    onClick={() => removeStatusMapping(index)}
                    className="rounded-lg p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors shrink-0"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Coverage */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-card">
          <h2 className="text-lg font-semibold text-foreground mb-4">Coverage</h2>
          <div className="space-y-4">
            {/* Countries */}
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Supported Countries</label>
              <div className="flex items-center gap-2 mb-2">
                <input
                  type="text"
                  value={countriesInput}
                  onChange={(e) => setCountriesInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') { e.preventDefault(); addCountry(); }
                  }}
                  placeholder="Type country and press Enter"
                  className="h-10 flex-1 rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
                />
                <button
                  type="button"
                  onClick={addCountry}
                  className="rounded-lg border border-border px-3 py-2 text-sm font-medium hover:bg-secondary transition-colors"
                >
                  Add
                </button>
              </div>
              {countries.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {countries.map((c) => (
                    <span
                      key={c}
                      className="inline-flex items-center gap-1.5 rounded-full border border-border bg-secondary/30 px-3 py-1 text-xs font-medium text-foreground"
                    >
                      {c}
                      <button type="button" onClick={() => removeCountry(c)} className="hover:text-destructive">
                        <Trash2 className="h-3 w-3" />
                      </button>
                    </span>
                  ))}
                </div>
              )}
            </div>

            {/* Cities */}
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Supported Cities</label>
              <div className="flex items-center gap-2 mb-2">
                <input
                  type="text"
                  value={citiesInput}
                  onChange={(e) => setCitiesInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') { e.preventDefault(); addCity(); }
                  }}
                  placeholder="Type city and press Enter"
                  className="h-10 flex-1 rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
                />
                <button
                  type="button"
                  onClick={addCity}
                  className="rounded-lg border border-border px-3 py-2 text-sm font-medium hover:bg-secondary transition-colors"
                >
                  Add
                </button>
              </div>
              {cities.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {cities.map((c) => (
                    <span
                      key={c}
                      className="inline-flex items-center gap-1.5 rounded-full border border-border bg-secondary/30 px-3 py-1 text-xs font-medium text-foreground"
                    >
                      {c}
                      <button type="button" onClick={() => removeCity(c)} className="hover:text-destructive">
                        <Trash2 className="h-3 w-3" />
                      </button>
                    </span>
                  ))}
                </div>
              )}
            </div>

            {/* Max Distance */}
            <div className="max-w-xs">
              <label className="block text-sm font-medium text-foreground mb-1.5">Max Distance (km)</label>
              <input
                type="number"
                value={maxDistance}
                onChange={(e) => setMaxDistance(Number(e.target.value))}
                min={0}
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
          </div>
        </div>

        {/* Pricing */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-card">
          <h2 className="text-lg font-semibold text-foreground mb-4">Pricing</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Pricing Model</label>
              <select
                value={pricingModel}
                onChange={(e) => setPricingModel(e.target.value)}
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              >
                {PRICING_MODELS.map((m) => (
                  <option key={m.value} value={m.value}>{m.label}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Currency</label>
              <select
                value={currency}
                onChange={(e) => setCurrency(e.target.value)}
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              >
                {CURRENCIES.map((c) => (
                  <option key={c} value={c}>{c}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Base Cost</label>
              <input
                type="number"
                value={baseCost}
                onChange={(e) => setBaseCost(Number(e.target.value))}
                min={0}
                step={0.01}
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Per KM Cost</label>
              <input
                type="number"
                value={perKmCost}
                onChange={(e) => setPerKmCost(Number(e.target.value))}
                min={0}
                step={0.01}
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
          </div>
        </div>

        {/* Rate Limits */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-card">
          <h2 className="text-lg font-semibold text-foreground mb-4">Rate Limits</h2>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Max Concurrent Deliveries</label>
              <input
                type="number"
                value={maxConcurrentDeliveries}
                onChange={(e) => setMaxConcurrentDeliveries(Number(e.target.value))}
                min={0}
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Daily Limit</label>
              <input
                type="number"
                value={dailyLimit}
                onChange={(e) => setDailyLimit(Number(e.target.value))}
                min={0}
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
              <p className="text-xs text-muted-foreground mt-1">0 = unlimited</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Avg Pickup Time (min)</label>
              <input
                type="number"
                value={avgPickupTime}
                onChange={(e) => setAvgPickupTime(Number(e.target.value))}
                min={0}
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
          </div>
        </div>

        {/* Contact */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-card">
          <h2 className="text-lg font-semibold text-foreground mb-4">Contact</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Contact Name</label>
              <input
                type="text"
                value={contactName}
                onChange={(e) => setContactName(e.target.value)}
                placeholder="John Doe"
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Contact Email</label>
              <input
                type="email"
                value={contactEmail}
                onChange={(e) => setContactEmail(e.target.value)}
                placeholder="contact@provider.com"
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">Contact Phone</label>
              <input
                type="tel"
                value={contactPhone}
                onChange={(e) => setContactPhone(e.target.value)}
                placeholder="+91 98765 43210"
                className="h-10 w-full rounded-lg border border-input bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div className="sm:col-span-2">
              <label className="block text-sm font-medium text-foreground mb-1.5">Notes</label>
              <textarea
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="Any additional notes about this provider..."
                rows={3}
                className="w-full rounded-lg border border-input bg-card px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring resize-none"
              />
            </div>
          </div>
        </div>

        {/* Submit */}
        <div className="flex items-center justify-end gap-3">
          <button
            type="button"
            onClick={() => navigate('/delivery/providers')}
            className="rounded-lg border border-border px-4 py-2.5 text-sm font-medium hover:bg-secondary transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={createMutation.isPending || !name || !code || !apiBaseUrl}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-6 py-2.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {createMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Save className="h-4 w-4" />
            )}
            Create Provider
          </button>
        </div>
      </form>
    </div>
  );
}
