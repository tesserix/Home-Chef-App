import { useState, useEffect, useMemo, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import {
  User,
  MapPin,
  CreditCard,
  Bell,
  Shield,
  ShieldCheck,
  LogOut,
  Plus,
  Edit2,
  Trash2,
  Camera,
  Check,
  UtensilsCrossed,
  Lock,
  Eye,
  EyeOff,
  Info,
  Copy,
  Download,
  AlertTriangle,
} from 'lucide-react';
import { QRCodeSVG } from 'qrcode.react';
import { toast } from 'sonner';
import { useAuth } from '@/app/providers/AuthProvider';
import { useAuthStore } from '@/app/store/auth-store';
import { apiClient } from '@/shared/services/api-client';
import { usePreferences } from '@/shared/hooks/usePreferences';
import { Badge } from '@/shared/components/ui/Badge';
import { Button } from '@/shared/components/ui/Button';
import { Input } from '@/shared/components/ui/Input';
import { cn } from '@/shared/utils/cn';
import type { Address, CustomerProfile, TotpStatusResponse, TotpSetupResponse } from '@/shared/types';

const profileSchema = z.object({
  firstName: z.string().min(2, 'First name must be at least 2 characters'),
  lastName: z.string().min(2, 'Last name must be at least 2 characters'),
  email: z.string().email('Invalid email address'),
  phone: z.string().optional(),
  dateOfBirth: z.string().optional(),
});

type ProfileFormData = z.infer<typeof profileSchema>;

const TABS = [
  { id: 'profile', label: 'Profile', icon: User },
  { id: 'preferences', label: 'Preferences', icon: UtensilsCrossed },
  { id: 'addresses', label: 'Addresses', icon: MapPin },
  { id: 'payments', label: 'Payment Methods', icon: CreditCard },
  { id: 'notifications', label: 'Notifications', icon: Bell },
  { id: 'security', label: 'Security', icon: Shield },
];

export default function ProfilePage() {
  const { user, logout } = useAuth();
  const [searchParams] = useSearchParams();
  const initialTab = useMemo(() => searchParams.get('tab') || 'profile', [searchParams]);
  const [activeTab, setActiveTab] = useState(initialTab);

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="container-app max-w-5xl">
        <h1 className="text-2xl font-bold text-gray-900 md:text-3xl">Account Settings</h1>

        <div className="mt-8 flex flex-col gap-8 lg:flex-row">
          {/* Sidebar */}
          <div className="lg:w-64">
            <div className="rounded-xl bg-white p-4 shadow-sm">
              {/* User Info */}
              <div className="flex items-center gap-3 border-b pb-4">
                <div className="relative">
                  <div className="h-12 w-12 rounded-full bg-brand-100 flex items-center justify-center">
                    {user?.avatar ? (
                      <img
                        src={user.avatar}
                        alt={user.firstName}
                        className="h-full w-full rounded-full object-cover"
                      />
                    ) : (
                      <span className="text-lg font-semibold text-brand-600">
                        {user?.firstName?.charAt(0)}
                        {user?.lastName?.charAt(0)}
                      </span>
                    )}
                  </div>
                </div>
                <div className="min-w-0">
                  <p className="font-medium text-gray-900 truncate">
                    {user?.firstName} {user?.lastName}
                  </p>
                  <p className="text-sm text-gray-500 truncate">{user?.email}</p>
                </div>
              </div>

              {/* Navigation */}
              <nav className="mt-4 space-y-1">
                {TABS.map((tab) => (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-colors ${
                      activeTab === tab.id
                        ? 'bg-brand-50 text-brand-600'
                        : 'text-gray-600 hover:bg-gray-100'
                    }`}
                  >
                    <tab.icon className="h-5 w-5" />
                    <span className="font-medium">{tab.label}</span>
                  </button>
                ))}
              </nav>

              {/* Logout */}
              <div className="mt-4 pt-4 border-t">
                <button
                  onClick={logout}
                  className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-red-600 hover:bg-red-50 transition-colors"
                >
                  <LogOut className="h-5 w-5" />
                  <span className="font-medium">Log Out</span>
                </button>
              </div>
            </div>
          </div>

          {/* Content */}
          <div className="flex-1">
            {activeTab === 'profile' && <ProfileTab />}
            {activeTab === 'preferences' && <PreferencesTab />}
            {activeTab === 'addresses' && <AddressesTab />}
            {activeTab === 'payments' && <PaymentsTab />}
            {activeTab === 'notifications' && <NotificationsTab />}
            {activeTab === 'security' && <SecurityTab />}
          </div>
        </div>
      </div>
    </div>
  );
}

function ProfileTab() {
  const { user } = useAuth();
  const [isEditing, setIsEditing] = useState(false);
  const [profile, setProfile] = useState<CustomerProfile | null>(null);
  const [avatarUrl, setAvatarUrl] = useState(user?.avatar || '');
  const [isUploading, setIsUploading] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
    setValue,
  } = useForm<ProfileFormData>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      firstName: user?.firstName || '',
      lastName: user?.lastName || '',
      email: user?.email || '',
      phone: user?.phone || '',
      dateOfBirth: '',
    },
  });

  useEffect(() => {
    apiClient.get<CustomerProfile>('/customer/profile').then((p) => {
      setProfile(p);
      if (p.avatar) setAvatarUrl(p.avatar);
      setValue('firstName', p.firstName);
      setValue('lastName', p.lastName);
      setValue('email', p.email);
      setValue('phone', p.phone ?? '');
      setValue('dateOfBirth', p.dateOfBirth?.split('T')[0] ?? '');
    }).catch(() => {});
  }, [setValue]);

  const onSubmit = async (formData: ProfileFormData) => {
    try {
      const updated = await apiClient.put<CustomerProfile>('/customer/profile', formData);
      setProfile(updated);
      toast.success('Profile updated successfully');
      setIsEditing(false);
    } catch {
      toast.error('Failed to update profile');
    }
  };

  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (file.size > 5 * 1024 * 1024) {
      toast.error('File too large. Maximum 5 MB.');
      return;
    }
    if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
      toast.error('Invalid file type. Use JPEG, PNG or WebP.');
      return;
    }

    setIsUploading(true);
    try {
      const formData = new FormData();
      formData.append('file', file);
      const result = await apiClient.upload<{ url: string }>('/customer/avatar', formData);
      setAvatarUrl(result.url);
      // Update the session user so header avatar refreshes
      const { useAuthStore } = await import('@/app/store/auth-store');
      const state = useAuthStore.getState();
      if (state.user) {
        useAuthStore.getState().setSession({ ...state.user, avatar: result.url });
      }
      toast.success('Profile photo updated');
    } catch {
      toast.error('Failed to upload photo');
    } finally {
      setIsUploading(false);
      // Reset the input so re-selecting the same file triggers onChange
      e.target.value = '';
    }
  };

  return (
    <div className="rounded-xl bg-white p-6 shadow-sm">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">Personal Information</h2>
        {!isEditing && (
          <button
            onClick={() => setIsEditing(true)}
            className="text-sm text-brand-600 hover:text-brand-700"
          >
            Edit
          </button>
        )}
      </div>

      {/* Avatar */}
      <div className="mt-6 flex items-center gap-4">
        <div className="relative">
          <div className="h-20 w-20 rounded-full bg-brand-100 flex items-center justify-center overflow-hidden">
            {avatarUrl ? (
              <img
                src={avatarUrl}
                alt={user?.firstName}
                className="h-full w-full rounded-full object-cover"
                draggable={false}
                onContextMenu={(e) => e.preventDefault()}
              />
            ) : (
              <span className="text-2xl font-semibold text-brand-600">
                {user?.firstName?.charAt(0)}
                {user?.lastName?.charAt(0)}
              </span>
            )}
          </div>
          <label className="absolute bottom-0 right-0 cursor-pointer rounded-full bg-white p-1.5 shadow-md hover:bg-gray-50">
            {isUploading ? (
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-gray-400 border-t-transparent" />
            ) : (
              <Camera className="h-4 w-4 text-gray-600" />
            )}
            <input
              type="file"
              accept="image/jpeg,image/png,image/webp"
              className="hidden"
              onChange={handleAvatarUpload}
              disabled={isUploading}
            />
          </label>
        </div>
        <div>
          <p className="font-medium text-gray-900">Profile Photo</p>
          <p className="text-sm text-gray-500">JPG, PNG or WebP. Max size 5MB.</p>
        </div>
      </div>

      {/* Form */}
      <form onSubmit={handleSubmit(onSubmit)} className="mt-6 space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="block text-sm font-medium text-gray-700">First name</label>
            <input
              {...register('firstName')}
              disabled={!isEditing}
              className="input-base mt-1 disabled:bg-gray-50"
            />
            {errors.firstName && (
              <p className="mt-1 text-xs text-red-600">{errors.firstName.message}</p>
            )}
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">Last name</label>
            <input
              {...register('lastName')}
              disabled={!isEditing}
              className="input-base mt-1 disabled:bg-gray-50"
            />
            {errors.lastName && (
              <p className="mt-1 text-xs text-red-600">{errors.lastName.message}</p>
            )}
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700">Email</label>
          <input
            {...register('email')}
            type="email"
            disabled={!isEditing}
            className="input-base mt-1 disabled:bg-gray-50"
          />
          {errors.email && (
            <p className="mt-1 text-xs text-red-600">{errors.email.message}</p>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700">Phone</label>
          <input
            {...register('phone')}
            type="tel"
            disabled={!isEditing}
            className="input-base mt-1 disabled:bg-gray-50"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700">Date of birth</label>
          <input
            {...register('dateOfBirth')}
            type="date"
            disabled={!isEditing}
            className="input-base mt-1 disabled:bg-gray-50"
          />
        </div>

        {isEditing && (
          <div className="flex gap-3 pt-4">
            <button type="submit" className="btn-primary">
              Save Changes
            </button>
            <button
              type="button"
              onClick={() => {
                if (profile) {
                  setValue('firstName', profile.firstName);
                  setValue('lastName', profile.lastName);
                  setValue('email', profile.email);
                  setValue('phone', profile.phone ?? '');
                  setValue('dateOfBirth', profile.dateOfBirth?.split('T')[0] ?? '');
                } else {
                  reset();
                }
                setIsEditing(false);
              }}
              className="btn-outline"
            >
              Cancel
            </button>
          </div>
        )}
      </form>
    </div>
  );
}

function PreferencesTab() {
  const { dietary, allergy, cuisine, spiceLevel, householdSize } = usePreferences();
  const [profile, setProfile] = useState<CustomerProfile | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  // Local draft state for editing
  const [dietaryPref, setDietaryPref] = useState<string[]>([]);
  const [allergies, setAllergies] = useState<string[]>([]);
  const [cuisinePref, setCuisinePref] = useState<string[]>([]);
  const [spice, setSpice] = useState('');
  const [household, setHousehold] = useState('');

  useEffect(() => {
    apiClient.get<CustomerProfile>('/customer/profile').then((p) => {
      setProfile(p);
      setDietaryPref(p.dietaryPreferences);
      setAllergies(p.foodAllergies);
      setCuisinePref(p.cuisinePreferences);
      setSpice(p.spiceTolerance);
      setHousehold(p.householdSize);
    }).catch(() => {});
  }, []);

  const toggleItem = (
    current: string[],
    setter: (v: string[]) => void,
    value: string
  ) => {
    setter(
      current.includes(value)
        ? current.filter((v) => v !== value)
        : [...current, value]
    );
  };

  const handleSave = async () => {
    setIsSaving(true);
    try {
      const updated = await apiClient.put<CustomerProfile>('/customer/profile', {
        dietaryPreferences: dietaryPref,
        foodAllergies: allergies,
        cuisinePreferences: cuisinePref,
        spiceTolerance: spice,
        householdSize: household,
      });
      setProfile(updated);
      toast.success('Preferences updated');
      setIsEditing(false);
    } catch {
      toast.error('Failed to save preferences');
    } finally {
      setIsSaving(false);
    }
  };

  const handleCancel = () => {
    if (profile) {
      setDietaryPref(profile.dietaryPreferences);
      setAllergies(profile.foodAllergies);
      setCuisinePref(profile.cuisinePreferences);
      setSpice(profile.spiceTolerance);
      setHousehold(profile.householdSize);
    }
    setIsEditing(false);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">Food Preferences</h2>
        {!isEditing ? (
          <Button variant="outline" size="sm" onClick={() => setIsEditing(true)}>
            Edit
          </Button>
        ) : (
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={handleCancel}>Cancel</Button>
            <Button variant="primary" size="sm" onClick={handleSave} isLoading={isSaving}>Save</Button>
          </div>
        )}
      </div>

      {/* Dietary */}
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <h3 className="font-medium text-gray-900 mb-3">Dietary Preferences</h3>
        <div className="flex flex-wrap gap-2">
          {dietary.map((opt) => {
            const selected = dietaryPref.includes(opt.value);
            return (
              <button
                key={opt.value}
                type="button"
                disabled={!isEditing}
                onClick={() => toggleItem(dietaryPref, setDietaryPref, opt.value)}
              >
                <Badge variant={selected ? 'default' : 'outline'} size="lg" className={isEditing ? 'cursor-pointer' : ''}>
                  {opt.label}
                </Badge>
              </button>
            );
          })}
        </div>
      </div>

      {/* Allergies */}
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <h3 className="font-medium text-gray-900 mb-3">Food Allergies</h3>
        <div className="flex flex-wrap gap-2">
          {allergy.map((opt) => {
            const selected = allergies.includes(opt.value);
            return (
              <button
                key={opt.value}
                type="button"
                disabled={!isEditing}
                onClick={() => toggleItem(allergies, setAllergies, opt.value)}
              >
                <Badge variant={selected ? 'error' : 'outline'} size="lg" className={isEditing ? 'cursor-pointer' : ''}>
                  {opt.label}
                </Badge>
              </button>
            );
          })}
        </div>
      </div>

      {/* Cuisines */}
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <h3 className="font-medium text-gray-900 mb-3">Favourite Cuisines</h3>
        <div className="flex flex-wrap gap-2">
          {cuisine.map((opt) => {
            const selected = cuisinePref.includes(opt.value);
            return (
              <button
                key={opt.value}
                type="button"
                disabled={!isEditing}
                onClick={() => toggleItem(cuisinePref, setCuisinePref, opt.value)}
              >
                <Badge variant={selected ? 'brand' : 'outline'} size="lg" className={isEditing ? 'cursor-pointer' : ''}>
                  {opt.label}
                </Badge>
              </button>
            );
          })}
        </div>
      </div>

      {/* Spice */}
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <h3 className="font-medium text-gray-900 mb-3">Spice Tolerance</h3>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {spiceLevel.map((level) => {
            const selected = spice === level.value;
            return (
              <button
                key={level.value}
                type="button"
                disabled={!isEditing}
                onClick={() => setSpice(level.value)}
                className={cn(
                  'rounded-xl border-2 p-3 text-center transition-all',
                  selected
                    ? 'border-primary bg-primary/5 text-primary'
                    : 'border-border hover:border-primary/30',
                  !isEditing && 'opacity-70'
                )}
              >
                <p className="font-medium">{level.label}</p>
                {level.description && (
                  <p className="text-xs text-muted-foreground mt-0.5">{level.description}</p>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* Household */}
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <h3 className="font-medium text-gray-900 mb-3">Household Size</h3>
        <div className="grid grid-cols-3 gap-3 sm:grid-cols-5">
          {householdSize.map((size) => {
            const selected = household === size.value;
            return (
              <button
                key={size.value}
                type="button"
                disabled={!isEditing}
                onClick={() => setHousehold(size.value)}
                className={cn(
                  'rounded-xl border-2 p-3 text-center transition-all',
                  selected
                    ? 'border-primary bg-primary/5 text-primary'
                    : 'border-border hover:border-primary/30',
                  !isEditing && 'opacity-70'
                )}
              >
                <p className="text-sm font-medium">{size.label}</p>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}

interface AddressFormData {
  label: string;
  line1: string;
  line2: string;
  city: string;
  state: string;
  postalCode: string;
  country: string;
  isDefault: boolean;
}

const emptyAddressForm: AddressFormData = {
  label: '',
  line1: '',
  line2: '',
  city: '',
  state: '',
  postalCode: '',
  country: 'IN',
  isDefault: false,
};

interface LocationOption {
  code?: string;
  name: string;
}

const ADDRESS_LABELS = ['Home', 'Work', 'Other'];

const selectClass = 'w-full h-10 px-4 text-sm rounded-lg border-2 border-input bg-background shadow-sm hover:border-brand-300 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-brand-500/20 focus-visible:border-brand-500 disabled:opacity-50';

function AddressesTab() {
  const [addresses, setAddresses] = useState<Address[]>([]);
  const [loading, setLoading] = useState(true);
  const [formMode, setFormMode] = useState<'hidden' | 'add' | 'edit'>('hidden');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formData, setFormData] = useState<AddressFormData>(emptyAddressForm);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  // Cascading location state
  const [countries, setCountries] = useState<LocationOption[]>([]);
  const [states, setStates] = useState<LocationOption[]>([]);
  const [cities, setCities] = useState<LocationOption[]>([]);

  const fetchAddresses = useCallback(() => {
    setLoading(true);
    apiClient.get<Address[]>('/addresses').then((data) => {
      setAddresses(data);
    }).catch(() => {}).finally(() => setLoading(false));
  }, []);

  useEffect(() => { fetchAddresses(); }, [fetchAddresses]);

  // Load countries on mount
  useEffect(() => {
    apiClient.get<LocationOption[]>('/locations/countries')
      .then(setCountries)
      .catch(() => {});
  }, []);

  // Counter to force location effects to re-fire when form opens
  const [formKey, setFormKey] = useState(0);

  // Load states when country changes or form opens
  useEffect(() => {
    if (formMode === 'hidden' || !formData.country) { setStates([]); setCities([]); return; }
    apiClient.get<LocationOption[]>(`/locations/countries/${formData.country}/states`)
      .then(setStates)
      .catch(() => setStates([]));
  }, [formData.country, formKey]); // eslint-disable-line react-hooks/exhaustive-deps

  // Load cities when state changes
  useEffect(() => {
    if (formMode === 'hidden' || !formData.state) { setCities([]); return; }
    apiClient.get<LocationOption[]>(`/locations/states/${formData.state}/cities`)
      .then(setCities)
      .catch(() => setCities([]));
  }, [formData.state, formKey]); // eslint-disable-line react-hooks/exhaustive-deps


  const openAdd = () => {
    setFormData(emptyAddressForm);
    setEditingId(null);
    setFormMode('add');
    setFormKey((k) => k + 1);
  };

  const openEdit = (address: Address) => {
    setFormData({
      label: address.label,
      line1: address.line1,
      line2: address.line2 || '',
      city: address.city,
      state: address.state,
      postalCode: address.postalCode,
      country: address.country || 'IN',
      isDefault: address.isDefault,
    });
    setEditingId(address.id);
    setFormMode('edit');
    setFormKey((k) => k + 1);
  };

  const cancelForm = () => {
    setFormMode('hidden');
    setEditingId(null);
  };

  const handleSave = async () => {
    if (!formData.label || !formData.line1 || !formData.city || !formData.state || !formData.postalCode) {
      toast.error('Please fill in all required fields');
      return;
    }
    setSaving(true);
    try {
      if (formMode === 'add') {
        await apiClient.post('/addresses', formData);
        toast.success('Address added');
      } else if (formMode === 'edit' && editingId) {
        await apiClient.put(`/addresses/${editingId}`, formData);
        toast.success('Address updated');
      }
      cancelForm();
      fetchAddresses();
    } catch {
      toast.error('Failed to save address');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: string) => {
    setDeletingId(id);
    try {
      await apiClient.delete(`/addresses/${id}`);
      toast.success('Address deleted');
      fetchAddresses();
    } catch {
      toast.error('Failed to delete address');
    } finally {
      setDeletingId(null);
    }
  };

  const updateField = (field: keyof AddressFormData, value: string | boolean) => {
    setFormData((prev) => {
      const next = { ...prev, [field]: value };
      // Cascade clear dependent fields
      if (field === 'country') {
        next.state = '';
        next.city = '';
        next.postalCode = '';
      } else if (field === 'state') {
        next.city = '';
        next.postalCode = '';
      } else if (field === 'city') {
        next.postalCode = '';
      }
      return next;
    });
  };

  return (
    <div className="rounded-xl bg-white p-6 shadow-sm">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">Saved Addresses</h2>
        {formMode === 'hidden' && (
          <Button variant="outline" size="sm" onClick={openAdd}>
            <Plus className="mr-1.5 h-4 w-4" />
            Add New
          </Button>
        )}
      </div>

      {/* Address Form */}
      {formMode !== 'hidden' && (
        <div className="mt-6 rounded-lg border border-brand-200 bg-brand-50/30 p-4">
          <h3 className="mb-4 font-medium text-gray-900">
            {formMode === 'add' ? 'Add New Address' : 'Edit Address'}
          </h3>
          <div className="space-y-4">
            {/* Address Label toggle buttons */}
            <div>
              <label className="mb-1.5 block text-sm font-medium text-gray-700">Address type *</label>
              <div className="flex gap-2">
                {ADDRESS_LABELS.map((label) => (
                  <button
                    key={label}
                    type="button"
                    onClick={() => updateField('label', label)}
                    className={cn(
                      'rounded-lg border-2 px-4 py-2 text-sm font-medium transition-all',
                      formData.label === label
                        ? 'border-brand-500 bg-brand-50 text-brand-700'
                        : 'border-gray-200 hover:border-brand-300 text-gray-600'
                    )}
                  >
                    {label}
                  </button>
                ))}
              </div>
            </div>

            {/* Country dropdown */}
            <div>
              <label className="mb-1.5 block text-sm font-medium text-gray-700">Country *</label>
              <select
                value={formData.country}
                onChange={(e) => updateField('country', e.target.value)}
                className={selectClass}
              >
                <option value="">Select country</option>
                {countries.map((c) => (
                  <option key={c.code} value={c.code}>{c.name}</option>
                ))}
              </select>
            </div>

            {/* State dropdown */}
            <div>
              <label className="mb-1.5 block text-sm font-medium text-gray-700">State *</label>
              <select
                value={formData.state}
                onChange={(e) => updateField('state', e.target.value)}
                disabled={states.length === 0}
                className={selectClass}
              >
                <option value="">Select state</option>
                {states.map((s) => (
                  <option key={s.code} value={s.code}>{s.name}</option>
                ))}
              </select>
            </div>

            {/* City dropdown */}
            <div>
              <label className="mb-1.5 block text-sm font-medium text-gray-700">City *</label>
              <select
                value={formData.city}
                onChange={(e) => updateField('city', e.target.value)}
                disabled={cities.length === 0}
                className={selectClass}
              >
                <option value="">Select city</option>
                {cities.map((city) => (
                  <option key={city.name} value={city.name}>{city.name}</option>
                ))}
              </select>
            </div>

            <Input
              label="PIN code *"
              placeholder="Enter PIN code"
              value={formData.postalCode}
              onChange={(e) => updateField('postalCode', e.target.value)}
            />

            <Input
              label="Address line 1 *"
              placeholder="House / flat / building number, street"
              value={formData.line1}
              onChange={(e) => updateField('line1', e.target.value)}
            />

            <Input
              label="Address line 2"
              placeholder="Landmark, area (optional)"
              value={formData.line2}
              onChange={(e) => updateField('line2', e.target.value)}
            />

            <div>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={formData.isDefault}
                  onChange={(e) => updateField('isDefault', e.target.checked)}
                  className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
                />
                <span className="text-gray-700">Set as default address</span>
              </label>
            </div>
          </div>
          <div className="mt-4 flex gap-3">
            <Button size="sm" onClick={handleSave} disabled={saving}>
              {saving ? 'Saving...' : formMode === 'add' ? 'Add Address' : 'Save Changes'}
            </Button>
            <Button variant="outline" size="sm" onClick={cancelForm} disabled={saving}>
              Cancel
            </Button>
          </div>
        </div>
      )}

      {loading ? (
        <div className="mt-6 flex justify-center py-8">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-gray-300 border-t-brand-600" />
        </div>
      ) : addresses.length === 0 && formMode === 'hidden' ? (
        <div className="mt-6 text-center py-8">
          <MapPin className="mx-auto h-10 w-10 text-gray-300" />
          <p className="mt-2 text-sm text-gray-500">No saved addresses yet</p>
          <Button variant="outline" size="sm" className="mt-3" onClick={openAdd}>
            <Plus className="mr-1.5 h-4 w-4" />
            Add your first address
          </Button>
        </div>
      ) : (
        <div className="mt-6 space-y-4">
          {addresses.map((address) => (
            <div
              key={address.id}
              className={cn(
                'flex items-start justify-between rounded-lg border p-4',
                editingId === address.id && formMode === 'edit' && 'border-brand-300 bg-brand-50/20'
              )}
            >
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-medium text-gray-900">{address.label}</span>
                  {address.isDefault && (
                    <Badge variant="brand" size="sm">Default</Badge>
                  )}
                </div>
                <p className="mt-1 text-sm text-gray-600">
                  {address.line1}
                  {address.line2 && `, ${address.line2}`}
                </p>
                <p className="text-sm text-gray-600">
                  {address.city}, {address.state} {address.postalCode}
                </p>
              </div>
              <div className="flex gap-1">
                <button
                  className="rounded-md p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                  onClick={() => openEdit(address)}
                  title="Edit address"
                >
                  <Edit2 className="h-4 w-4" />
                </button>
                <button
                  className="rounded-md p-2 text-gray-400 hover:bg-red-50 hover:text-red-600"
                  onClick={() => handleDelete(address.id)}
                  disabled={deletingId === address.id}
                  title="Delete address"
                >
                  {deletingId === address.id ? (
                    <div className="h-4 w-4 animate-spin rounded-full border-2 border-gray-300 border-t-red-600" />
                  ) : (
                    <Trash2 className="h-4 w-4" />
                  )}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function PaymentsTab() {
  const paymentMethods = [
    { id: '1', type: 'visa', last4: '4242', expiry: '12/25', isDefault: true },
    { id: '2', type: 'mastercard', last4: '8888', expiry: '06/26', isDefault: false },
  ];

  return (
    <div className="rounded-xl bg-white p-6 shadow-sm">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">Payment Methods</h2>
        <button className="btn-outline">
          <Plus className="h-4 w-4" />
          Add New
        </button>
      </div>

      <div className="mt-6 space-y-4">
        {paymentMethods.map((method) => (
          <div
            key={method.id}
            className="flex items-center justify-between rounded-lg border p-4"
          >
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-16 items-center justify-center rounded bg-gray-100">
                <CreditCard className="h-6 w-6 text-gray-400" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-medium text-gray-900 capitalize">
                    {method.type} •••• {method.last4}
                  </span>
                  {method.isDefault && (
                    <span className="rounded bg-brand-100 px-2 py-0.5 text-xs font-medium text-brand-700">
                      Default
                    </span>
                  )}
                </div>
                <p className="text-sm text-gray-500">Expires {method.expiry}</p>
              </div>
            </div>
            <div className="flex gap-2">
              <button className="p-2 text-gray-400 hover:text-gray-600">
                <Edit2 className="h-4 w-4" />
              </button>
              <button className="p-2 text-gray-400 hover:text-red-600">
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function NotificationsTab() {
  const [settings, setSettings] = useState({
    orderUpdates: true,
    promotions: true,
    newChefs: false,
    reviews: true,
    email: true,
    push: true,
    sms: false,
  });

  const toggleSetting = (key: keyof typeof settings) => {
    setSettings((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  return (
    <div className="rounded-xl bg-white p-6 shadow-sm">
      <h2 className="text-lg font-semibold text-gray-900">Notification Preferences</h2>

      <div className="mt-6 space-y-6">
        <div>
          <h3 className="font-medium text-gray-900">What to notify about</h3>
          <div className="mt-4 space-y-4">
            {[
              { key: 'orderUpdates', label: 'Order updates', desc: 'Get notified about order status changes' },
              { key: 'promotions', label: 'Promotions & offers', desc: 'Receive special deals and discounts' },
              { key: 'newChefs', label: 'New chefs nearby', desc: 'When new chefs join in your area' },
              { key: 'reviews', label: 'Review reminders', desc: 'Reminders to review past orders' },
            ].map((item) => (
              <label key={item.key} className="flex items-start gap-4">
                <input
                  type="checkbox"
                  checked={settings[item.key as keyof typeof settings]}
                  onChange={() => toggleSetting(item.key as keyof typeof settings)}
                  className="mt-1 h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
                />
                <div>
                  <p className="font-medium text-gray-900">{item.label}</p>
                  <p className="text-sm text-gray-500">{item.desc}</p>
                </div>
              </label>
            ))}
          </div>
        </div>

        <div className="border-t pt-6">
          <h3 className="font-medium text-gray-900">How to notify</h3>
          <div className="mt-4 space-y-4">
            {[
              { key: 'email', label: 'Email notifications' },
              { key: 'push', label: 'Push notifications' },
              { key: 'sms', label: 'SMS notifications' },
            ].map((item) => (
              <label key={item.key} className="flex items-center justify-between">
                <span className="text-gray-700">{item.label}</span>
                <button
                  onClick={() => toggleSetting(item.key as keyof typeof settings)}
                  className={`relative h-6 w-11 rounded-full transition-colors ${
                    settings[item.key as keyof typeof settings] ? 'bg-brand-500' : 'bg-gray-300'
                  }`}
                >
                  <span
                    className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform ${
                      settings[item.key as keyof typeof settings] ? 'translate-x-5' : ''
                    }`}
                  />
                </button>
              </label>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

const BFF_URL = import.meta.env.VITE_BFF_URL || 'https://identity.fe3dr.com';
// Same-origin BFF proxy to avoid CORS — Istio rewrites /bff/* → / on BFF
const _isLocalDev = typeof window !== 'undefined' && window.location.hostname === 'localhost';
const BFF_BASE = _isLocalDev ? BFF_URL : '/bff';

type TwoFactorState =
  | 'loading'
  | 'disabled'
  | 'setup'
  | 'show-recovery-codes'
  | 'enabled'
  | 'disabling'
  | 'regenerating';

function bffHeaders(method: 'GET' | 'POST' = 'GET'): HeadersInit {
  const headers: HeadersInit = { 'Content-Type': 'application/json' };
  if (method === 'POST') {
    const csrfToken = useAuthStore.getState().csrfToken;
    if (csrfToken) headers['X-CSRF-Token'] = csrfToken;
  }
  return headers;
}

function TwoFactorSection() {
  const [state, setState] = useState<TwoFactorState>('loading');
  const [backupCodesRemaining, setBackupCodesRemaining] = useState(0);
  const [setupSession, setSetupSession] = useState('');
  const [totpUri, setTotpUri] = useState('');
  const [manualKey, setManualKey] = useState('');
  const [backupCodes, setBackupCodes] = useState<string[]>([]);
  const [code, setCode] = useState('');
  const [verifying, setVerifying] = useState(false);
  const [codesSaved, setCodesSaved] = useState(false);
  const [returnToState, setReturnToState] = useState<'enabled' | 'disabled'>('enabled');

  const fetchStatus = useCallback(async () => {
    try {
      const res = await fetch(`${BFF_BASE}/auth/totp/status`, {
        credentials: 'include',
        headers: bffHeaders(),
      });
      if (!res.ok) throw new Error();
      const data: TotpStatusResponse = await res.json();
      setBackupCodesRemaining(data.backup_codes_remaining);
      setState(data.totp_enabled ? 'enabled' : 'disabled');
    } catch {
      setState('disabled');
    }
  }, []);

  useEffect(() => { fetchStatus(); }, [fetchStatus]);

  const handleInitiateSetup = async () => {
    try {
      const res = await fetch(`${BFF_BASE}/auth/totp/setup/initiate`, {
        method: 'POST',
        credentials: 'include',
        headers: bffHeaders('POST'),
        body: '{}',
      });
      if (!res.ok) throw new Error();
      const data: TotpSetupResponse = await res.json();
      setSetupSession(data.setup_session);
      setTotpUri(data.totp_uri);
      setManualKey(data.manual_entry_key);
      setBackupCodes(data.backup_codes);
      setCode('');
      setState('setup');
    } catch {
      toast.error('Failed to initiate 2FA setup');
    }
  };

  const handleConfirmSetup = async () => {
    if (code.length !== 6) return;
    setVerifying(true);
    try {
      const res = await fetch(`${BFF_BASE}/auth/totp/setup/confirm`, {
        method: 'POST',
        credentials: 'include',
        headers: bffHeaders('POST'),
        body: JSON.stringify({ setup_session: setupSession, code }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => null);
        throw new Error(err?.error || 'Invalid code');
      }
      toast.success('Two-factor authentication enabled');
      setReturnToState('enabled');
      setCodesSaved(false);
      setState('show-recovery-codes');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to verify code');
    } finally {
      setVerifying(false);
      setCode('');
    }
  };

  const handleDisable = async () => {
    if (code.length !== 6) return;
    setVerifying(true);
    try {
      const res = await fetch(`${BFF_BASE}/auth/totp/disable`, {
        method: 'POST',
        credentials: 'include',
        headers: bffHeaders('POST'),
        body: JSON.stringify({ code }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => null);
        throw new Error(err?.error || 'Invalid code');
      }
      toast.success('Two-factor authentication disabled');
      setState('disabled');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to disable 2FA');
    } finally {
      setVerifying(false);
      setCode('');
    }
  };

  const handleRegenerateBackupCodes = async () => {
    if (code.length !== 6) return;
    setVerifying(true);
    try {
      const res = await fetch(`${BFF_BASE}/auth/totp/backup-codes/regenerate`, {
        method: 'POST',
        credentials: 'include',
        headers: bffHeaders('POST'),
        body: JSON.stringify({ code }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => null);
        throw new Error(err?.error || 'Invalid code');
      }
      const data = await res.json();
      setBackupCodes(data.backup_codes);
      setReturnToState('enabled');
      setCodesSaved(false);
      setState('show-recovery-codes');
      toast.success('Backup codes regenerated');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to regenerate codes');
    } finally {
      setVerifying(false);
      setCode('');
    }
  };

  const copyAllCodes = () => {
    navigator.clipboard.writeText(backupCodes.join('\n'));
    toast.success('Backup codes copied to clipboard');
  };

  const downloadCodes = () => {
    const text = `HomeChef 2FA Backup Codes\n${'='.repeat(30)}\n\n${backupCodes.map((c, i) => `${i + 1}. ${c}`).join('\n')}\n\nKeep these codes safe. Each code can only be used once.`;
    const blob = new Blob([text], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'homechef-backup-codes.txt';
    a.click();
    URL.revokeObjectURL(url);
  };

  if (state === 'loading') {
    return (
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <div className="flex items-center gap-3">
          <div className="h-5 w-5 animate-spin rounded-full border-2 border-gray-300 border-t-brand-600" />
          <span className="text-sm text-gray-500">Loading 2FA status...</span>
        </div>
      </div>
    );
  }

  if (state === 'disabled') {
    return (
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900">Two-Factor Authentication</h2>
        <div className="mt-4 flex items-start gap-3">
          <Shield className="mt-0.5 h-5 w-5 shrink-0 text-gray-400" />
          <div>
            <p className="text-sm text-gray-700">
              Add an extra layer of security to your account by requiring a verification code from your authenticator app when signing in.
            </p>
          </div>
        </div>
        <Button variant="primary" className="mt-4" onClick={handleInitiateSetup}>
          Enable 2FA
        </Button>
      </div>
    );
  }

  if (state === 'setup') {
    return (
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900">Set Up Two-Factor Authentication</h2>
        <p className="mt-1 text-sm text-gray-500">
          Scan the QR code with your authenticator app (Google Authenticator, Authy, etc.)
        </p>

        <div className="mt-6 flex flex-col items-center gap-4">
          <div className="rounded-lg border bg-white p-4">
            <QRCodeSVG value={totpUri} size={200} level="M" />
          </div>

          <div className="w-full max-w-sm">
            <p className="text-xs font-medium text-gray-500 uppercase tracking-wide">Manual entry key</p>
            <div className="mt-1 flex items-center gap-2 rounded-lg border bg-gray-50 px-3 py-2">
              <code className="flex-1 text-sm font-mono text-gray-800 break-all select-all">
                {manualKey}
              </code>
              <button
                type="button"
                onClick={() => {
                  navigator.clipboard.writeText(manualKey);
                  toast.success('Key copied');
                }}
                className="shrink-0 text-gray-400 hover:text-gray-600"
              >
                <Copy className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>

        <div className="mt-6 max-w-sm">
          <Input
            label="Enter 6-digit code from your app"
            value={code}
            onChange={(e) => {
              const val = e.target.value.replace(/\D/g, '').slice(0, 6);
              setCode(val);
            }}
            placeholder="000000"
            leftIcon={<Lock className="h-4 w-4" />}
            className="font-mono text-center tracking-widest"
            autoComplete="one-time-code"
            inputMode="numeric"
          />
          <div className="mt-4 flex gap-3">
            <Button
              variant="primary"
              onClick={handleConfirmSetup}
              isLoading={verifying}
              disabled={code.length !== 6}
            >
              Verify & Enable
            </Button>
            <Button variant="outline" onClick={() => setState('disabled')}>
              Cancel
            </Button>
          </div>
        </div>
      </div>
    );
  }

  if (state === 'show-recovery-codes') {
    return (
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900">Save Your Backup Codes</h2>

        <div className="mt-4 flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4">
          <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
          <p className="text-sm text-amber-800">
            Save these codes in a safe place. They won't be shown again. Each code can only be used once to sign in if you lose access to your authenticator app.
          </p>
        </div>

        <div className="mt-4 grid grid-cols-2 gap-2 sm:grid-cols-5">
          {backupCodes.map((bc, i) => (
            <div
              key={i}
              className="rounded-lg border bg-gray-50 px-3 py-2 text-center font-mono text-sm text-gray-800 select-all"
            >
              {bc}
            </div>
          ))}
        </div>

        <div className="mt-4 flex flex-wrap gap-3">
          <Button variant="outline" size="sm" onClick={copyAllCodes}>
            <Copy className="mr-1.5 h-4 w-4" />
            Copy All
          </Button>
          <Button variant="outline" size="sm" onClick={downloadCodes}>
            <Download className="mr-1.5 h-4 w-4" />
            Download
          </Button>
        </div>

        <label className="mt-6 flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={codesSaved}
            onChange={(e) => setCodesSaved(e.target.checked)}
            className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
          />
          <span className="text-sm text-gray-700">I have saved these backup codes</span>
        </label>

        <Button
          variant="primary"
          className="mt-4"
          disabled={!codesSaved}
          onClick={() => {
            fetchStatus();
            setState(returnToState);
          }}
        >
          Done
        </Button>
      </div>
    );
  }

  if (state === 'enabled') {
    return (
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900">Two-Factor Authentication</h2>
          <Badge variant="success">
            <ShieldCheck className="mr-1 h-3.5 w-3.5" />
            Enabled
          </Badge>
        </div>
        <p className="mt-2 text-sm text-gray-500">
          Your account is protected with two-factor authentication.
        </p>

        {backupCodesRemaining > 0 && (
          <p className="mt-2 text-sm text-gray-600">
            <span className="font-medium">{backupCodesRemaining}</span> backup code{backupCodesRemaining !== 1 ? 's' : ''} remaining
          </p>
        )}
        {backupCodesRemaining === 0 && (
          <div className="mt-2 flex items-start gap-2 text-sm text-amber-700">
            <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
            <span>No backup codes remaining. Regenerate them now.</span>
          </div>
        )}

        <div className="mt-4 flex flex-wrap gap-3">
          <Button
            variant="outline"
            onClick={() => {
              setCode('');
              setState('regenerating');
            }}
          >
            Regenerate Backup Codes
          </Button>
          <Button
            variant="destructive"
            onClick={() => {
              setCode('');
              setState('disabling');
            }}
          >
            Disable 2FA
          </Button>
        </div>
      </div>
    );
  }

  // disabling or regenerating — both need TOTP code confirmation
  const isDisabling = state === 'disabling';
  return (
    <div className="rounded-xl bg-white p-6 shadow-sm">
      <h2 className="text-lg font-semibold text-gray-900">
        {isDisabling ? 'Disable Two-Factor Authentication' : 'Regenerate Backup Codes'}
      </h2>
      <p className="mt-1 text-sm text-gray-500">
        Enter a code from your authenticator app to confirm.
      </p>

      <div className="mt-4 max-w-sm">
        <Input
          label="6-digit verification code"
          value={code}
          onChange={(e) => {
            const val = e.target.value.replace(/\D/g, '').slice(0, 6);
            setCode(val);
          }}
          placeholder="000000"
          leftIcon={<Lock className="h-4 w-4" />}
          className="font-mono text-center tracking-widest"
          autoComplete="one-time-code"
          inputMode="numeric"
        />
        <div className="mt-4 flex gap-3">
          <Button
            variant={isDisabling ? 'destructive' : 'primary'}
            onClick={isDisabling ? handleDisable : handleRegenerateBackupCodes}
            isLoading={verifying}
            disabled={code.length !== 6}
          >
            {isDisabling ? 'Disable 2FA' : 'Regenerate'}
          </Button>
          <Button variant="outline" onClick={() => setState('enabled')}>
            Cancel
          </Button>
        </div>
      </div>
    </div>
  );
}

function SecurityTab() {
  const [showPasswordForm, setShowPasswordForm] = useState(false);
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showCurrent, setShowCurrent] = useState(false);
  const [showNew, setShowNew] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [saving, setSaving] = useState(false);
  const [authProvider, setAuthProvider] = useState<string>('email');

  useEffect(() => {
    apiClient.get<CustomerProfile>('/customer/profile').then((profile) => {
      setAuthProvider(profile.authProvider || 'email');
    }).catch(() => {});
  }, []);

  const isSocialLogin = authProvider !== 'email';

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      toast.error('Passwords do not match');
      return;
    }
    if (newPassword.length < 8) {
      toast.error('Password must be at least 8 characters');
      return;
    }
    setSaving(true);
    try {
      await apiClient.put('/profile/password', {
        currentPassword,
        newPassword,
      });
      toast.success('Password updated successfully');
      setShowPasswordForm(false);
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch {
      toast.error('Failed to update password');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Password Section */}
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900">Password</h2>
        <p className="mt-1 text-sm text-gray-500">
          Change your password to keep your account secure
        </p>

        {isSocialLogin ? (
          <div className="mt-4 flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4">
            <Info className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
            <div>
              <p className="text-sm font-medium text-amber-800">
                Password change is not available
              </p>
              <p className="mt-1 text-sm text-amber-700">
                Your account uses {authProvider.charAt(0).toUpperCase() + authProvider.slice(1)} sign-in.
                Password management is handled by your {authProvider.charAt(0).toUpperCase() + authProvider.slice(1)} account.
              </p>
            </div>
          </div>
        ) : showPasswordForm ? (
          <form onSubmit={handleChangePassword} className="mt-6 space-y-4">
            <Input
              label="Current password"
              type={showCurrent ? 'text' : 'password'}
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              leftIcon={<Lock className="h-4 w-4" />}
              rightIcon={
                <button type="button" onClick={() => setShowCurrent(!showCurrent)} className="cursor-pointer">
                  {showCurrent ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              }
              required
            />
            <Input
              label="New password"
              type={showNew ? 'text' : 'password'}
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              leftIcon={<Lock className="h-4 w-4" />}
              rightIcon={
                <button type="button" onClick={() => setShowNew(!showNew)} className="cursor-pointer">
                  {showNew ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              }
              hint="Must be at least 8 characters"
              required
            />
            <Input
              label="Confirm new password"
              type={showConfirm ? 'text' : 'password'}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              leftIcon={<Lock className="h-4 w-4" />}
              rightIcon={
                <button type="button" onClick={() => setShowConfirm(!showConfirm)} className="cursor-pointer">
                  {showConfirm ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              }
              error={confirmPassword && newPassword !== confirmPassword ? 'Passwords do not match' : undefined}
              required
            />
            <div className="flex gap-3 pt-2">
              <Button type="submit" variant="primary" isLoading={saving}>
                Update Password
              </Button>
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setShowPasswordForm(false);
                  setCurrentPassword('');
                  setNewPassword('');
                  setConfirmPassword('');
                }}
              >
                Cancel
              </Button>
            </div>
          </form>
        ) : (
          <Button variant="outline" className="mt-4" onClick={() => setShowPasswordForm(true)}>
            Change Password
          </Button>
        )}
      </div>

      {/* 2FA Section */}
      <TwoFactorSection />

      {/* Connected Accounts Section */}
      <div className="rounded-xl bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900">Connected Accounts</h2>
        <p className="mt-1 text-sm text-gray-500">
          Manage your connected social accounts
        </p>

        <div className="mt-4 space-y-3">
          {[
            { name: 'Google', provider: 'google' },
            { name: 'Facebook', provider: 'facebook' },
            { name: 'Apple', provider: 'apple' },
          ].map((account) => (
            <div
              key={account.name}
              className="flex items-center justify-between rounded-lg border border-gray-200 p-3"
            >
              <span className="font-medium text-gray-900">{account.name}</span>
              {authProvider === account.provider ? (
                <span className="flex items-center gap-1 text-sm text-green-600">
                  <Check className="h-4 w-4" />
                  Connected
                </span>
              ) : (
                <Button variant="ghost" size="sm">
                  Connect
                </Button>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Delete Account Section */}
      <div className="rounded-xl bg-red-50 p-6">
        <h2 className="text-lg font-semibold text-red-800">Delete Account</h2>
        <p className="mt-1 text-sm text-red-600">
          Once you delete your account, there is no going back. Please be certain.
        </p>
        <Button variant="destructive" className="mt-4">
          Delete Account
        </Button>
      </div>
    </div>
  );
}
