import { useState, useEffect } from 'react';
import { useOnboardingStore } from '@/app/store/onboarding-store';
import { Card } from '@/shared/components/ui/Card';
import { Input } from '@/shared/components/ui/Input';
import { MapPin } from 'lucide-react';
import { cn } from '@/shared/utils/cn';
import { apiClient } from '@/shared/services/api-client';

interface LocationOption {
  code?: string;
  name: string;
}

const ADDRESS_LABELS = ['Home', 'Work', 'Other'];

export function StepAddress() {
  const { data, updateData } = useOnboardingStore();
  const [countries, setCountries] = useState<LocationOption[]>([]);
  const [states, setStates] = useState<LocationOption[]>([]);
  const [cities, setCities] = useState<LocationOption[]>([]);
  // Load countries on mount
  useEffect(() => {
    apiClient.get<LocationOption[]>('/locations/countries')
      .then(setCountries)
      .catch(() => {});
  }, []);

  // Load states when country changes
  useEffect(() => {
    if (!data.addressCountry) return;
    setStates([]);
    setCities([]);
    apiClient.get<LocationOption[]>(`/locations/countries/${data.addressCountry}/states`)
      .then(setStates)
      .catch(() => {});
  }, [data.addressCountry]);

  // Load cities when state changes
  useEffect(() => {
    if (!data.addressState) return;
    setCities([]);
    apiClient.get<LocationOption[]>(`/locations/states/${data.addressState}/cities`)
      .then(setCities)
      .catch(() => {});
  }, [data.addressState]);

  return (
    <Card>
      <div className="flex items-center gap-3 mb-6">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
          <MapPin className="h-5 w-5 text-primary" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-foreground">Delivery Address</h2>
          <p className="text-sm text-muted-foreground">Where should we deliver your food?</p>
        </div>
      </div>

      <div className="space-y-4">
        {/* Address Label */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-foreground">Address type</label>
          <div className="flex gap-2">
            {ADDRESS_LABELS.map((label) => (
              <button
                key={label}
                type="button"
                onClick={() => updateData({ addressLabel: label })}
                className={cn(
                  'rounded-lg border-2 px-4 py-2 text-sm font-medium transition-all',
                  data.addressLabel === label
                    ? 'border-primary bg-primary/5 text-primary'
                    : 'border-border hover:border-primary/30'
                )}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        {/* Country */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-foreground">Country</label>
          <select
            value={data.addressCountry}
            onChange={(e) => {
              updateData({ addressCountry: e.target.value, addressState: '', addressCity: '', addressPostalCode: '' });
            }}
            className="w-full h-10 px-4 text-sm rounded-lg border-2 border-input bg-background shadow-sm hover:border-primary/30 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-ring/20 focus-visible:border-ring"
          >
            <option value="">Select country</option>
            {countries.map((c) => (
              <option key={c.code} value={c.code}>{c.name}</option>
            ))}
          </select>
        </div>

        {/* State */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-foreground">State</label>
          <select
            value={data.addressState}
            onChange={(e) => {
              updateData({ addressState: e.target.value, addressCity: '', addressPostalCode: '' });
            }}
            disabled={states.length === 0}
            className="w-full h-10 px-4 text-sm rounded-lg border-2 border-input bg-background shadow-sm hover:border-primary/30 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-ring/20 focus-visible:border-ring disabled:opacity-50"
          >
            <option value="">Select state</option>
            {states.map((s) => (
              <option key={s.code} value={s.code}>{s.name}</option>
            ))}
          </select>
        </div>

        {/* City */}
        <div>
          <label className="mb-1.5 block text-sm font-medium text-foreground">City</label>
          <select
            value={data.addressCity}
            onChange={(e) => {
              updateData({ addressCity: e.target.value, addressPostalCode: '' });
            }}
            disabled={cities.length === 0}
            className="w-full h-10 px-4 text-sm rounded-lg border-2 border-input bg-background shadow-sm hover:border-primary/30 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-ring/20 focus-visible:border-ring disabled:opacity-50"
          >
            <option value="">Select city</option>
            {cities.map((city) => (
              <option key={city.name} value={city.name}>{city.name}</option>
            ))}
          </select>
        </div>

        {/* PIN code */}
        <Input
          label="PIN code"
          value={data.addressPostalCode}
          onChange={(e) => updateData({ addressPostalCode: e.target.value })}
          placeholder="Enter PIN code"
        />

        <Input
          label="Address line 1"
          value={data.addressLine1}
          onChange={(e) => updateData({ addressLine1: e.target.value })}
          placeholder="House / flat / building number, street"
        />

        <Input
          label="Address line 2"
          value={data.addressLine2}
          onChange={(e) => updateData({ addressLine2: e.target.value })}
          placeholder="Landmark, area (optional)"
        />
      </div>
    </Card>
  );
}
