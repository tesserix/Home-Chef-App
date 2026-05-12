import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Camera,
  MapPin,
  Clock,
  DollarSign,
  Save,
  Loader2,
  Plus,
  X,
  ChefHat,
  Shield,
  Star,
  Check,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import type { Chef } from '@/shared/types';

const profileSchema = z.object({
  businessName: z.string().min(2, 'Business name must be at least 2 characters'),
  description: z.string().min(20, 'Description must be at least 20 characters'),
  cuisines: z.array(z.string()).min(1, 'Select at least one cuisine'),
  specialties: z.array(z.string()),
  prepTime: z.string(),
  minimumOrder: z.number().min(0),
  deliveryFee: z.number().min(0),
  serviceRadius: z.number().min(1).max(50),
  acceptingOrders: z.boolean(),
});

type ProfileFormData = z.infer<typeof profileSchema>;

const CUISINES = [
  'South Indian', 'North Indian', 'Italian', 'Japanese', 'Mexican',
  'Thai', 'Chinese', 'Mediterranean', 'Continental', 'American',
];

export default function ChefProfilePage() {
  const [isEditing, setIsEditing] = useState(false);
  const queryClient = useQueryClient();

  const { data: profile, isLoading } = useQuery({
    queryKey: ['chef-profile'],
    queryFn: () => apiClient.get<Chef>('/chef/profile'),
  });

  const {
    register,
    handleSubmit,
    formState: { errors },
    watch,
    setValue,
    reset,
  } = useForm<ProfileFormData>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      businessName: profile?.businessName || '',
      description: profile?.description || '',
      cuisines: profile?.cuisines || [],
      specialties: profile?.specialties || [],
      prepTime: profile?.prepTime || '30-45 min',
      minimumOrder: profile?.minimumOrder || 0,
      deliveryFee: profile?.deliveryFee || 0,
      serviceRadius: profile?.serviceRadius || 5,
      acceptingOrders: profile?.acceptingOrders ?? true,
    },
  });

  const updateMutation = useMutation({
    mutationFn: (data: ProfileFormData) => apiClient.put('/chef/profile', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-profile'] });
      toast.success('Profile updated successfully');
      setIsEditing(false);
    },
    onError: () => {
      toast.error('Failed to update profile');
    },
  });

  const cuisines = watch('cuisines') || [];
  const specialties = watch('specialties') || [];
  const [newSpecialty, setNewSpecialty] = useState('');

  const toggleCuisine = (cuisine: string) => {
    if (cuisines.includes(cuisine)) {
      setValue('cuisines', cuisines.filter((c) => c !== cuisine));
    } else {
      setValue('cuisines', [...cuisines, cuisine]);
    }
  };

  const addSpecialty = () => {
    if (newSpecialty && !specialties.includes(newSpecialty)) {
      setValue('specialties', [...specialties, newSpecialty]);
      setNewSpecialty('');
    }
  };

  const removeSpecialty = (specialty: string) => {
    setValue('specialties', specialties.filter((s) => s !== specialty));
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-herb" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="font-display text-2xl font-semibold text-ink">Kitchen Profile</h1>
          <p className="mt-1 text-ink-soft">Manage your public profile and settings</p>
        </div>
        {!isEditing ? (
          <button onClick={() => setIsEditing(true)} className="btn-primary">
            Edit Profile
          </button>
        ) : (
          <div className="flex gap-3">
            <button
              onClick={() => {
                reset();
                setIsEditing(false);
              }}
              className="btn-outline"
            >
              Cancel
            </button>
            <button
              onClick={handleSubmit((data) => updateMutation.mutate(data))}
              disabled={updateMutation.isPending}
              className="btn-primary"
            >
              {updateMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <>
                  <Save className="h-4 w-4" />
                  Save Changes
                </>
              )}
            </button>
          </div>
        )}
      </div>

      {/* Verification Status */}
      <div className={`rounded-xl p-4 ${profile?.verified ? 'bg-herb-tint' : 'bg-amber-tint'}`}>
        <div className="flex items-center gap-3">
          <div className={`rounded-full p-2 ${profile?.verified ? 'bg-herb-tint' : 'bg-amber-tint'}`}>
            {profile?.verified ? (
              <Shield className="h-5 w-5 text-herb" />
            ) : (
              <Shield className="h-5 w-5 text-amber" />
            )}
          </div>
          <div>
            <p className={`font-medium ${profile?.verified ? 'text-herb' : 'text-amber'}`}>
              {profile?.verified ? 'Verified Chef' : 'Verification Pending'}
            </p>
            <p className={`text-sm ${profile?.verified ? 'text-herb' : 'text-amber'}`}>
              {profile?.verified
                ? 'Your kitchen has been verified and approved'
                : 'Complete your profile to get verified'}
            </p>
          </div>
        </div>
      </div>

      <form className="space-y-6">
        {/* Profile Images */}
        <div className="rounded-xl bg-bone p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-ink">Profile Images</h2>

          <div className="mt-6 grid gap-6 md:grid-cols-2">
            {/* Profile Picture */}
            <div>
              <label className="block text-sm font-medium text-ink-soft mb-2">
                Profile Picture
              </label>
              <div className="relative inline-block">
                <div className="h-32 w-32 rounded-xl bg-mist overflow-hidden">
                  {profile?.profileImage ? (
                    <img
                      src={profile.profileImage}
                      alt="Profile"
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    <div className="flex h-full w-full items-center justify-center">
                      <ChefHat className="h-12 w-12 text-ink-muted" />
                    </div>
                  )}
                </div>
                {isEditing && (
                  <button
                    type="button"
                    className="absolute bottom-0 right-0 rounded-full bg-herb p-2 text-paper shadow-lg hover:bg-herb"
                  >
                    <Camera className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>

            {/* Banner Image */}
            <div>
              <label className="block text-sm font-medium text-ink-soft mb-2">
                Banner Image
              </label>
              <div className="relative">
                <div className="h-32 w-full rounded-xl bg-mist overflow-hidden">
                  {profile?.bannerImage ? (
                    <img
                      src={profile.bannerImage}
                      alt="Banner"
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    <div className="flex h-full w-full items-center justify-center">
                      <Camera className="h-12 w-12 text-ink-muted" />
                    </div>
                  )}
                </div>
                {isEditing && (
                  <button
                    type="button"
                    className="absolute bottom-2 right-2 rounded-full bg-herb p-2 text-paper shadow-lg hover:bg-herb"
                  >
                    <Camera className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Basic Info */}
        <div className="rounded-xl bg-bone p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-ink">Basic Information</h2>

          <div className="mt-6 space-y-4">
            <div>
              <label className="block text-sm font-medium text-ink-soft">Business Name</label>
              <input
                {...register('businessName')}
                disabled={!isEditing}
                className="input-base mt-1 disabled:bg-paper"
              />
              {errors.businessName && (
                <p className="mt-1 text-xs text-paprika">{errors.businessName.message}</p>
              )}
            </div>

            <div>
              <label className="block text-sm font-medium text-ink-soft">Description</label>
              <textarea
                {...register('description')}
                disabled={!isEditing}
                rows={4}
                className="input-base mt-1 disabled:bg-paper"
                placeholder="Tell customers about your kitchen, cooking style, and what makes your food special..."
              />
              {errors.description && (
                <p className="mt-1 text-xs text-paprika">{errors.description.message}</p>
              )}
            </div>
          </div>
        </div>

        {/* Cuisines */}
        <div className="rounded-xl bg-bone p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-ink">Cuisines</h2>
          <p className="mt-1 text-sm text-ink-muted">Select the cuisines you specialize in</p>

          <div className="mt-4 flex flex-wrap gap-2">
            {CUISINES.map((cuisine) => (
              <button
                key={cuisine}
                type="button"
                onClick={() => isEditing && toggleCuisine(cuisine)}
                disabled={!isEditing}
                className={`rounded-full px-4 py-2 text-sm font-medium transition-colors ${
                  cuisines.includes(cuisine)
                    ? 'bg-herb text-paper'
                    : 'bg-mist text-ink-soft'
                } ${isEditing ? 'hover:opacity-80' : ''}`}
              >
                {cuisines.includes(cuisine) && <Check className="mr-1 inline h-4 w-4" />}
                {cuisine}
              </button>
            ))}
          </div>
          {errors.cuisines && (
            <p className="mt-2 text-xs text-paprika">{errors.cuisines.message}</p>
          )}
        </div>

        {/* Specialties */}
        <div className="rounded-xl bg-bone p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-ink">Specialties</h2>
          <p className="mt-1 text-sm text-ink-muted">Add your signature dishes or specialties</p>

          <div className="mt-4 flex flex-wrap gap-2">
            {specialties.map((specialty) => (
              <span
                key={specialty}
                className="inline-flex items-center gap-1 rounded-full bg-herb-tint px-3 py-1 text-sm text-herb"
              >
                {specialty}
                {isEditing && (
                  <button
                    type="button"
                    onClick={() => removeSpecialty(specialty)}
                    className="hover:text-herb"
                  >
                    <X className="h-3 w-3" />
                  </button>
                )}
              </span>
            ))}
          </div>

          {isEditing && (
            <div className="mt-4 flex gap-2">
              <input
                type="text"
                value={newSpecialty}
                onChange={(e) => setNewSpecialty(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && (e.preventDefault(), addSpecialty())}
                placeholder="Add a specialty..."
                className="input-base flex-1"
              />
              <button type="button" onClick={addSpecialty} className="btn-outline">
                <Plus className="h-4 w-4" />
                Add
              </button>
            </div>
          )}
        </div>

        {/* Business Settings */}
        <div className="rounded-xl bg-bone p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-ink">Business Settings</h2>

          <div className="mt-6 grid gap-6 sm:grid-cols-2">
            <div>
              <label className="block text-sm font-medium text-ink-soft">
                <Clock className="mr-1 inline h-4 w-4" />
                Average Prep Time
              </label>
              <select
                {...register('prepTime')}
                disabled={!isEditing}
                className="input-base mt-1 disabled:bg-paper"
              >
                <option value="15-30 min">15-30 minutes</option>
                <option value="30-45 min">30-45 minutes</option>
                <option value="45-60 min">45-60 minutes</option>
                <option value="1-2 hours">1-2 hours</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-ink-soft">
                <MapPin className="mr-1 inline h-4 w-4" />
                Delivery Radius (km)
              </label>
              <input
                type="number"
                {...register('serviceRadius', { valueAsNumber: true })}
                disabled={!isEditing}
                className="input-base mt-1 disabled:bg-paper"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-ink-soft">
                <DollarSign className="mr-1 inline h-4 w-4" />
                Minimum Order
              </label>
              <div className="relative mt-1">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-muted">$</span>
                <input
                  type="number"
                  {...register('minimumOrder', { valueAsNumber: true })}
                  disabled={!isEditing}
                  className="input-base pl-7 disabled:bg-paper"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-ink-soft">
                <DollarSign className="mr-1 inline h-4 w-4" />
                Delivery Fee
              </label>
              <div className="relative mt-1">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-muted">$</span>
                <input
                  type="number"
                  {...register('deliveryFee', { valueAsNumber: true })}
                  disabled={!isEditing}
                  className="input-base pl-7 disabled:bg-paper"
                />
              </div>
            </div>
          </div>

          <div className="mt-6 flex items-center justify-between rounded-lg bg-paper p-4">
            <div>
              <p className="font-medium text-ink">Accepting Orders</p>
              <p className="text-sm text-ink-muted">Toggle to pause or resume taking orders</p>
            </div>
            <label className="relative inline-flex cursor-pointer items-center">
              <input
                type="checkbox"
                {...register('acceptingOrders')}
                disabled={!isEditing}
                className="peer sr-only"
              />
              <div className="h-6 w-11 rounded-full bg-mist-strong peer-checked:bg-herb peer-disabled:opacity-50 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:bg-bone after:transition-all peer-checked:after:translate-x-5" />
            </label>
          </div>
        </div>

        {/* Stats */}
        <div className="rounded-xl bg-bone p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-ink">Your Stats</h2>

          <div className="mt-6 grid gap-4 sm:grid-cols-3">
            <div className="text-center">
              <div className="flex items-center justify-center gap-1 font-display text-3xl font-semibold tabular-nums text-ink">
                <Star className="h-6 w-6 fill-amber text-amber" />
                {profile?.rating || 0}
              </div>
              <p className="mt-1 text-sm text-ink-muted">Rating</p>
            </div>
            <div className="text-center">
              <p className="font-display text-3xl font-semibold tabular-nums text-ink">{profile?.totalReviews || 0}</p>
              <p className="mt-1 text-sm text-ink-muted">Reviews</p>
            </div>
            <div className="text-center">
              <p className="font-display text-3xl font-semibold tabular-nums text-ink">{profile?.totalOrders || 0}</p>
              <p className="mt-1 text-sm text-ink-muted">Total Orders</p>
            </div>
          </div>
        </div>
      </form>
    </div>
  );
}
