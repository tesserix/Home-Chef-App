import { useState, useEffect } from 'react';
import { toast } from 'sonner';
import { useOnboardingStore } from '@/app/store/onboarding-store';
import { Input } from '@/shared/components/ui/Input';
import { Card } from '@/shared/components/ui/Card';
import { FileUpload } from '@tesserix/web';
import { User, Phone, Mail, MapPin, Camera, Loader2 } from 'lucide-react';
import { uploadProfileImage } from '@/shared/services/upload-service';
import { apiClient } from '@/shared/services/api-client';

interface Props {
  errors: Record<string, string>;
}

interface Country {
  id: string;
  code: string;
  name: string;
  phoneCode: string;
}

interface StateItem {
  id: string;
  code: string;
  name: string;
}

interface City {
  id: string;
  name: string;
  isMajor: boolean;
}

const selectClass =
  'w-full rounded-lg border-2 border-input bg-background px-4 py-2.5 text-sm shadow-sm transition-all hover:border-primary/30 focus:border-ring focus:outline-none focus:ring-4 focus:ring-ring/20 disabled:cursor-not-allowed disabled:opacity-50';

export function StepPersonalInfo({ errors }: Props) {
  const { data, updateData, updateAddress } = useOnboardingStore();
  const [avatarFiles, setAvatarFiles] = useState<File[]>([]);
  const [isUploadingAvatar, setIsUploadingAvatar] = useState(false);

  const [countries, setCountries] = useState<Country[]>([]);
  const [states, setStates] = useState<StateItem[]>([]);
  const [cities, setCities] = useState<City[]>([]);
  const [loadingStates, setLoadingStates] = useState(false);
  const [loadingCities, setLoadingCities] = useState(false);

  // Fetch countries on mount
  useEffect(() => {
    apiClient.get<Country[]>('/locations/countries').then(setCountries).catch(() => {});
  }, []);

  // Fetch states when country changes
  useEffect(() => {
    if (!data.kitchenAddress.country) {
      setStates([]);
      return;
    }
    setLoadingStates(true);
    apiClient
      .get<StateItem[]>(`/locations/countries/${data.kitchenAddress.country}/states`)
      .then(setStates)
      .catch(() => setStates([]))
      .finally(() => setLoadingStates(false));
  }, [data.kitchenAddress.country]);

  // Fetch cities when state changes
  useEffect(() => {
    if (!data.kitchenAddress.state) {
      setCities([]);
      return;
    }
    const stateObj = states.find((s) => s.name === data.kitchenAddress.state);
    if (!stateObj) return;
    setLoadingCities(true);
    apiClient
      .get<City[]>(`/locations/states/${stateObj.code}/cities`, { country: data.kitchenAddress.country })
      .then(setCities)
      .catch(() => setCities([]))
      .finally(() => setLoadingCities(false));
  }, [data.kitchenAddress.state, states, data.kitchenAddress.country]);

  const handleCountryChange = (code: string) => {
    updateAddress({ country: code, state: '', city: '', postalCode: '' });
    setStates([]);
    setCities([]);
  };

  const handleStateChange = (name: string) => {
    updateAddress({ state: name, city: '', postalCode: '' });
    setCities([]);
  };

  const handleCityChange = (name: string) => {
    updateAddress({ city: name, postalCode: '' });
  };

  const handleAvatarChange = async (newFiles: File[]) => {
    if (newFiles.length > 0) {
      const file = newFiles[0]!;
      setAvatarFiles(newFiles);
      setIsUploadingAvatar(true);
      try {
        const url = await uploadProfileImage(file);
        updateData({ profileImage: url });
        toast.success('Profile photo uploaded');
      } catch (err) {
        toast.error(err instanceof Error ? err.message : 'Upload failed');
        setAvatarFiles([]);
      } finally {
        setIsUploadingAvatar(false);
      }
    } else {
      setAvatarFiles([]);
      updateData({ profileImage: undefined });
    }
  };

  const selectedCountry = countries.find((c) => c.code === data.kitchenAddress.country);

  return (
    <div className="space-y-6">
      <Card>
        <h3 className="text-lg font-semibold text-foreground">Personal Details</h3>
        <p className="mt-1 text-sm text-muted-foreground">
          Tell us about yourself. This information helps verify your identity.
        </p>

        <div className="mt-6 flex items-center gap-5">
          <div className="relative h-20 w-20 shrink-0 overflow-hidden rounded-full border-2 border-dashed border-border bg-secondary">
            {data.profileImage ? (
              <img src={data.profileImage} alt="Profile" className="h-full w-full object-cover" />
            ) : (
              <div className="flex h-full w-full items-center justify-center">
                <Camera className="h-6 w-6 text-muted-foreground" />
              </div>
            )}
            {isUploadingAvatar && (
              <div className="absolute inset-0 flex items-center justify-center rounded-full bg-black/40">
                <Loader2 className="h-5 w-5 animate-spin text-white" />
              </div>
            )}
          </div>
          <div className="flex-1">
            <p className="text-sm font-medium text-foreground">
              Profile Photo <span className="font-normal text-muted-foreground">(Optional)</span>
            </p>
            <p className="mt-0.5 text-xs text-muted-foreground">
              Shown on your kitchen page. JPEG, PNG, or WebP. Max 5 MB.
            </p>
            <div className="mt-2">
              <FileUpload
                value={avatarFiles}
                onValueChange={handleAvatarChange}
                accept=".jpg,.jpeg,.png,.webp"
                multiple={false}
                maxFiles={1}
                maxSizeBytes={5 * 1024 * 1024}
                helperText=""
              />
            </div>
          </div>
        </div>

        <div className="mt-6 space-y-4">
          <Input
            label="Full Name"
            placeholder="e.g. Meena Sharma"
            value={data.fullName}
            onChange={(e) => updateData({ fullName: e.target.value })}
            leftIcon={<User className="h-4 w-4" />}
            error={errors.fullName}
          />

          <div className="grid gap-4 sm:grid-cols-2">
            <Input
              label="Phone Number"
              type="tel"
              placeholder={selectedCountry ? `${selectedCountry.phoneCode} ...` : '+91 98765 43210'}
              value={data.phone}
              onChange={(e) => updateData({ phone: e.target.value })}
              leftIcon={<Phone className="h-4 w-4" />}
              error={errors.phone}
            />
            <Input
              label="Email Address"
              type="email"
              placeholder="meena@example.com"
              value={data.email}
              onChange={(e) => updateData({ email: e.target.value })}
              leftIcon={<Mail className="h-4 w-4" />}
              error={errors.email}
              hint="Pre-filled from your login"
            />
          </div>
        </div>
      </Card>

      <Card>
        <div className="flex items-center gap-2">
          <MapPin className="h-5 w-5 text-muted-foreground" />
          <h3 className="text-lg font-semibold text-foreground">Kitchen Address</h3>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">
          Where your kitchen is located. This is used for delivery radius and customer visibility.
        </p>

        <div className="mt-6 space-y-4">
          <Input
            label="Address Line 1"
            placeholder="House/Flat No., Building Name, Street"
            value={data.kitchenAddress.line1}
            onChange={(e) => updateAddress({ line1: e.target.value })}
            error={errors['kitchenAddress.line1']}
          />
          <Input
            label="Address Line 2 (Optional)"
            placeholder="Area, Colony, Landmark"
            value={data.kitchenAddress.line2 || ''}
            onChange={(e) => updateAddress({ line2: e.target.value })}
          />
          <Input
            label="Landmark (Optional)"
            placeholder="Near or opposite..."
            value={data.kitchenAddress.landmark || ''}
            onChange={(e) => updateAddress({ landmark: e.target.value })}
          />

          {/* Country */}
          <div className="w-full">
            <label className="mb-1.5 block text-sm font-medium text-foreground">Country</label>
            <select
              value={data.kitchenAddress.country}
              onChange={(e) => handleCountryChange(e.target.value)}
              className={selectClass}
            >
              <option value="">Select country</option>
              {countries.map((c) => (
                <option key={c.code} value={c.code}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>

          <div className="grid gap-4 sm:grid-cols-3">
            {/* State */}
            <div className="w-full">
              <label className="mb-1.5 block text-sm font-medium text-foreground">State</label>
              <select
                value={data.kitchenAddress.state}
                onChange={(e) => handleStateChange(e.target.value)}
                className={selectClass}
                disabled={!data.kitchenAddress.country || loadingStates}
              >
                <option value="">{loadingStates ? 'Loading...' : 'Select state'}</option>
                {states.map((s) => (
                  <option key={s.id} value={s.name}>
                    {s.name}
                  </option>
                ))}
              </select>
              {errors['kitchenAddress.state'] && (
                <p className="mt-1.5 text-sm text-destructive">{errors['kitchenAddress.state']}</p>
              )}
            </div>

            {/* City */}
            <div className="w-full">
              <label className="mb-1.5 block text-sm font-medium text-foreground">City</label>
              <select
                value={data.kitchenAddress.city}
                onChange={(e) => handleCityChange(e.target.value)}
                className={selectClass}
                disabled={!data.kitchenAddress.state || loadingCities}
              >
                <option value="">{loadingCities ? 'Loading...' : 'Select city'}</option>
                {cities.map((c) => (
                  <option key={c.id} value={c.name}>
                    {c.name} {c.isMajor ? '(Major)' : ''}
                  </option>
                ))}
              </select>
              {errors['kitchenAddress.city'] && (
                <p className="mt-1.5 text-sm text-destructive">{errors['kitchenAddress.city']}</p>
              )}
            </div>

            {/* Postcode / PIN Code */}
            <div className="w-full">
              <label className="mb-1.5 block text-sm font-medium text-foreground">PIN Code <span className="text-destructive">*</span></label>
              <input
                type="text"
                inputMode="numeric"
                maxLength={10}
                placeholder="Enter PIN code"
                value={data.kitchenAddress.postalCode}
                onChange={(e) => updateAddress({ postalCode: e.target.value.replace(/[^0-9]/g, '') })}
                className={`h-10 w-full rounded-lg border bg-card px-3 text-sm text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring ${
                  errors['kitchenAddress.postalCode'] ? 'border-destructive' : 'border-input'
                }`}
              />
              {errors['kitchenAddress.postalCode'] && (
                <p className="mt-1.5 text-sm text-destructive">{errors['kitchenAddress.postalCode']}</p>
              )}
            </div>
          </div>
        </div>
      </Card>
    </div>
  );
}
