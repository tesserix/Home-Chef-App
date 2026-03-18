import { useOnboardingStore } from '@/app/store/onboarding-store';
import { Input } from '@/shared/components/ui/Input';
import { Card } from '@/shared/components/ui/Card';
import { Clock, MapPin, IndianRupee, Truck } from 'lucide-react';
import type { OperatingHours } from '@/shared/types';

interface Props {
  errors: Record<string, string>;
}

const DAYS: { key: keyof OperatingHours; label: string }[] = [
  { key: 'monday', label: 'Monday' },
  { key: 'tuesday', label: 'Tuesday' },
  { key: 'wednesday', label: 'Wednesday' },
  { key: 'thursday', label: 'Thursday' },
  { key: 'friday', label: 'Friday' },
  { key: 'saturday', label: 'Saturday' },
  { key: 'sunday', label: 'Sunday' },
];

const PREP_OPTIONS = [
  { value: '15-30 min', label: '15-30 minutes' },
  { value: '30-45 min', label: '30-45 minutes' },
  { value: '45-60 min', label: '45-60 minutes' },
  { value: '1-2 hours', label: '1-2 hours' },
  { value: '2+ hours', label: '2+ hours (pre-order only)' },
];

export function StepOperations({ errors }: Props) {
  const { data, updateData, updateHours } = useOnboardingStore();

  const toggleDay = (day: keyof OperatingHours) => {
    if (data.operatingHours[day]) {
      updateHours(day, undefined);
    } else {
      updateHours(day, { open: '09:00', close: '21:00' });
    }
  };

  return (
    <div className="space-y-6">
      {/* Business Settings */}
      <Card>
        <h3 className="text-lg font-semibold text-foreground">Delivery & Pricing</h3>
        <p className="mt-1 text-sm text-muted-foreground">
          Set your preparation time, delivery range, and pricing. You can change these anytime.
        </p>

        <div className="mt-6 grid gap-6 sm:grid-cols-2">
          <div>
            <label className="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-foreground">
              <Clock className="h-4 w-4 text-muted-foreground" />
              Average Preparation Time
            </label>
            <select
              value={data.prepTime}
              onChange={(e) => updateData({ prepTime: e.target.value })}
              className="w-full rounded-lg border-2 border-input bg-background px-4 py-2.5 text-sm shadow-sm transition-all hover:border-primary/30 focus:border-ring focus:outline-none focus:ring-4 focus:ring-ring/20"
            >
              {PREP_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-foreground">
              <MapPin className="h-4 w-4 text-muted-foreground" />
              Delivery Radius (km)
            </label>
            <Input
              type="number"
              value={data.serviceRadius}
              onChange={(e) => updateData({ serviceRadius: Number(e.target.value) })}
              error={errors.serviceRadius}
              hint="How far can you deliver or allow pickup from"
            />
          </div>

          <div>
            <label className="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-foreground">
              <IndianRupee className="h-4 w-4 text-muted-foreground" />
              Minimum Order Value (Optional)
            </label>
            <div className="relative">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-sm text-muted-foreground">
                &#8377;
              </span>
              <input
                type="number"
                value={data.minimumOrder}
                onChange={(e) => updateData({ minimumOrder: Number(e.target.value) })}
                className="w-full rounded-lg border-2 border-input bg-background py-2.5 pl-8 pr-4 text-sm shadow-sm transition-all hover:border-primary/30 focus:border-ring focus:outline-none focus:ring-4 focus:ring-ring/20"
                placeholder="0"
              />
            </div>
            <p className="mt-1.5 text-sm text-muted-foreground">Set to 0 for no minimum</p>
          </div>

          <div>
            <label className="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-foreground">
              <Truck className="h-4 w-4 text-muted-foreground" />
              Delivery Fee
            </label>
            <div className="relative">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-sm text-muted-foreground">
                &#8377;
              </span>
              <input
                type="number"
                value={data.deliveryFee}
                onChange={(e) => updateData({ deliveryFee: Number(e.target.value) })}
                className="w-full rounded-lg border-2 border-input bg-background py-2.5 pl-8 pr-4 text-sm shadow-sm transition-all hover:border-primary/30 focus:border-ring focus:outline-none focus:ring-4 focus:ring-ring/20"
                placeholder="30"
              />
            </div>
            <p className="mt-1.5 text-sm text-muted-foreground">Per order delivery charge</p>
          </div>
        </div>
      </Card>

      {/* Operating Hours */}
      <Card>
        <h3 className="text-lg font-semibold text-foreground">Operating Hours</h3>
        <p className="mt-1 text-sm text-muted-foreground">
          Set your weekly availability. Toggle off days you're not available.
        </p>

        <div className="mt-6 space-y-3">
          {DAYS.map(({ key, label }) => {
            const hours = data.operatingHours[key];
            const isOpen = !!hours;
            return (
              <div
                key={key}
                className={`flex items-center gap-4 rounded-lg border p-3 transition-colors ${
                  isOpen ? 'border-border bg-card' : 'border-border/50 bg-muted/30'
                }`}
              >
                <button
                  type="button"
                  onClick={() => toggleDay(key)}
                  className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors ${
                    isOpen ? 'bg-primary' : 'bg-muted-foreground/30'
                  }`}
                >
                  <span
                    className={`inline-block h-5 w-5 rounded-full bg-white shadow-sm transition-transform ${
                      isOpen ? 'translate-x-[22px]' : 'translate-x-[2px]'
                    }`}
                  />
                </button>

                <span className={`w-28 text-sm font-medium ${isOpen ? 'text-foreground' : 'text-muted-foreground'}`}>
                  {label}
                </span>

                {isOpen ? (
                  <div className="flex items-center gap-2">
                    <input
                      type="time"
                      value={hours.open}
                      onChange={(e) => updateHours(key, { ...hours, open: e.target.value })}
                      className="rounded-md border border-input bg-background px-2 py-1.5 text-sm focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/20"
                    />
                    <span className="text-sm text-muted-foreground">to</span>
                    <input
                      type="time"
                      value={hours.close}
                      onChange={(e) => updateHours(key, { ...hours, close: e.target.value })}
                      className="rounded-md border border-input bg-background px-2 py-1.5 text-sm focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/20"
                    />
                  </div>
                ) : (
                  <span className="text-sm text-muted-foreground">Closed</span>
                )}
              </div>
            );
          })}
        </div>
      </Card>
    </div>
  );
}
