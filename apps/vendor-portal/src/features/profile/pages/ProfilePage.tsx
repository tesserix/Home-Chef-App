import { useState, useEffect, useRef } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import {
  Save,
  ChefHat,
  Star,
  ShoppingBag,
  MessageSquare,
  Plus,
  X,
  Check,
  Clock,
  MapPin,
  IndianRupee,
  Truck,
  Settings2,
  Shield,
  Camera,
  ArrowRight,
  Loader2,
  FileText,
  Upload,
  Download,
  CheckCircle,
  AlertCircle,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';
import { Button } from '@/shared/components/ui/Button';
import { Badge } from '@/shared/components/ui/Badge';
import { Card } from '@/shared/components/ui/Card';
import { Input, Textarea } from '@/shared/components/ui/Input';
import { Avatar } from '@/shared/components/ui/Avatar';
import type { Chef } from '@/shared/types';

const BFF_URL = import.meta.env.VITE_BFF_URL || 'https://identity.fe3dr.com';

const profileSchema = z.object({
  businessName: z.string().min(2, 'Business name must be at least 2 characters'),
  description: z.string().min(20, 'Description must be at least 20 characters'),
  cuisines: z.array(z.string()).min(1, 'Select at least one cuisine'),
  specialties: z.array(z.string()),
  prepTime: z.string(),
  serviceRadius: z.number().min(1, 'Minimum 1 km').max(50, 'Maximum 50 km'),
  minimumOrder: z.number().min(0, 'Cannot be negative'),
  deliveryFee: z.number().min(0, 'Cannot be negative'),
  acceptingOrders: z.boolean(),
});

type ProfileFormData = z.infer<typeof profileSchema>;

const CUISINES = [
  'South Indian',
  'North Indian',
  'Bengali',
  'Gujarati',
  'Rajasthani',
  'Punjabi',
  'Mughlai',
  'Kerala',
  'Hyderabadi',
  'Street Food',
  'Chinese',
  'Continental',
];

async function uploadImage(endpoint: string, file: File): Promise<string> {
  const formData = new FormData();
  formData.append('file', file);

  const res = await fetch(`${BFF_URL}/api/v1${endpoint}`, {
    method: 'POST',
    credentials: 'include',
    body: formData,
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Upload failed' }));
    throw new Error(err.error || 'Upload failed');
  }

  const data = await res.json();
  return data.url;
}

export default function ProfilePage() {
  const queryClient = useQueryClient();
  const [newSpecialty, setNewSpecialty] = useState('');
  const avatarInputRef = useRef<HTMLInputElement>(null);
  const bannerInputRef = useRef<HTMLInputElement>(null);

  const { data: profile, isLoading } = useQuery({
    queryKey: ['chef-profile'],
    queryFn: () => apiClient.get<Chef>('/chef/profile'),
  });

  const {
    register,
    handleSubmit,
    formState: { errors, isDirty },
    watch,
    setValue,
    reset,
  } = useForm<ProfileFormData>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      businessName: '',
      description: '',
      cuisines: [],
      specialties: [],
      prepTime: '30-45 min',
      serviceRadius: 5,
      minimumOrder: 0,
      deliveryFee: 0,
      acceptingOrders: true,
    },
  });

  useEffect(() => {
    if (profile) {
      reset({
        businessName: profile.businessName || '',
        description: profile.description || '',
        cuisines: profile.cuisines || [],
        specialties: profile.specialties || [],
        prepTime: profile.prepTime || '30-45 min',
        serviceRadius: profile.serviceRadius || 5,
        minimumOrder: profile.minimumOrder || 0,
        deliveryFee: profile.deliveryFee || 0,
        acceptingOrders: profile.acceptingOrders ?? true,
      });
    }
  }, [profile, reset]);

  const updateMutation = useMutation({
    mutationFn: (data: ProfileFormData) => apiClient.put('/chef/profile', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-profile'] });
      toast.success('Profile updated successfully');
    },
    onError: () => {
      toast.error('Failed to update profile');
    },
  });

  const avatarMutation = useMutation({
    mutationFn: (file: File) => uploadImage('/chef/profile-image', file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-profile'] });
      toast.success('Profile photo updated');
    },
    onError: (err: Error) => {
      toast.error(err.message || 'Failed to upload profile photo');
    },
  });

  const bannerMutation = useMutation({
    mutationFn: (file: File) => uploadImage('/chef/banner-image', file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-profile'] });
      toast.success('Cover photo updated');
    },
    onError: (err: Error) => {
      toast.error(err.message || 'Failed to upload cover photo');
    },
  });

  const cuisines = watch('cuisines') || [];
  const specialties = watch('specialties') || [];
  const acceptingOrders = watch('acceptingOrders');

  const handleFileSelect = (
    e: React.ChangeEvent<HTMLInputElement>,
    mutation: typeof avatarMutation,
  ) => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (file.size > 5 * 1024 * 1024) {
      toast.error('File too large. Maximum 5 MB.');
      return;
    }

    if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
      toast.error('Invalid file type. Allowed: JPEG, PNG, WebP.');
      return;
    }

    mutation.mutate(file);
    e.target.value = '';
  };

  const toggleCuisine = (cuisine: string) => {
    if (cuisines.includes(cuisine)) {
      setValue('cuisines', cuisines.filter((c) => c !== cuisine), { shouldDirty: true });
    } else {
      setValue('cuisines', [...cuisines, cuisine], { shouldDirty: true });
    }
  };

  const addSpecialty = () => {
    const trimmed = newSpecialty.trim();
    if (trimmed && !specialties.includes(trimmed)) {
      setValue('specialties', [...specialties, trimmed], { shouldDirty: true });
      setNewSpecialty('');
    }
  };

  const removeSpecialty = (specialty: string) => {
    setValue(
      'specialties',
      specialties.filter((s) => s !== specialty),
      { shouldDirty: true }
    );
  };

  const toggleAcceptingOrders = () => {
    setValue('acceptingOrders', !acceptingOrders, { shouldDirty: true });
  };

  const onSubmit = (data: ProfileFormData) => {
    updateMutation.mutate(data);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-500 border-t-transparent" />
      </div>
    );
  }

  return (
    <motion.div
      variants={staggerContainer}
      initial="hidden"
      animate="visible"
      className="space-y-6 pb-8"
    >
      {/* Hidden file inputs */}
      <input
        ref={avatarInputRef}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        className="hidden"
        onChange={(e) => handleFileSelect(e, avatarMutation)}
      />
      <input
        ref={bannerInputRef}
        type="file"
        accept="image/jpeg,image/png,image/webp"
        className="hidden"
        onChange={(e) => handleFileSelect(e, bannerMutation)}
      />

      {/* Page Header */}
      <motion.div variants={fadeInUp} className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Kitchen Profile</h1>
          <p className="mt-1 text-sm text-gray-500">Manage your public profile and business settings</p>
        </div>
        <Button
          type="button"
          onClick={handleSubmit(onSubmit)}
          isLoading={updateMutation.isPending}
          disabled={!isDirty}
          leftIcon={<Save className="h-4 w-4" />}
        >
          Save Changes
        </Button>
      </motion.div>

      {/* Profile Header Card */}
      <motion.div variants={fadeInUp}>
        <Card padding="none" className="overflow-hidden">
          {/* Banner */}
          <div className="relative h-32 bg-gradient-to-r from-brand-500 to-brand-700 sm:h-40">
            {profile?.bannerImage && (
              <img
                src={profile.bannerImage}
                alt="Banner"
                className="h-full w-full object-cover"
              />
            )}
            <button
              type="button"
              onClick={() => bannerInputRef.current?.click()}
              disabled={bannerMutation.isPending}
              className="absolute bottom-3 right-3 rounded-full bg-black/40 p-2 text-white backdrop-blur-sm transition-colors hover:bg-black/60 disabled:opacity-50"
            >
              {bannerMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Camera className="h-4 w-4" />
              )}
            </button>
          </div>

          {/* Profile Info */}
          <div className="relative px-6 pb-6">
            <div className="flex flex-col items-start gap-4 sm:flex-row sm:items-end">
              {/* Avatar */}
              <div className="relative -mt-12">
                <Avatar
                  src={profile?.profileImage}
                  alt={profile?.businessName}
                  fallback={profile?.businessName?.charAt(0)}
                  size="2xl"
                  ring="brand"
                  className="border-4 border-white"
                />
                <button
                  type="button"
                  onClick={() => avatarInputRef.current?.click()}
                  disabled={avatarMutation.isPending}
                  className="absolute bottom-0 right-0 rounded-full bg-brand-500 p-1.5 text-white shadow-lg transition-colors hover:bg-brand-600 disabled:opacity-50"
                >
                  {avatarMutation.isPending ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <Camera className="h-3.5 w-3.5" />
                  )}
                </button>
              </div>

              {/* Name & Stats */}
              <div className="flex-1 pt-2 sm:pt-0">
                <div className="flex items-center gap-2">
                  <h2 className="text-xl font-bold text-gray-900">
                    {profile?.businessName || 'Your Kitchen'}
                  </h2>
                  {profile?.verified && (
                    <Shield className="h-5 w-5 text-brand-500" />
                  )}
                </div>
                <div className="mt-2 flex flex-wrap items-center gap-3">
                  <div className="flex items-center gap-1 text-sm text-gray-600">
                    <Star className="h-4 w-4 fill-warning text-warning" />
                    <span className="font-semibold text-gray-900">
                      {profile?.rating?.toFixed(1) || '0.0'}
                    </span>
                  </div>
                  <div className="flex items-center gap-1 text-sm text-gray-600">
                    <ShoppingBag className="h-4 w-4" />
                    <span>{profile?.totalOrders || 0} orders</span>
                  </div>
                  <div className="flex items-center gap-1 text-sm text-gray-600">
                    <MessageSquare className="h-4 w-4" />
                    <span>{profile?.totalReviews || 0} reviews</span>
                  </div>
                </div>
              </div>

              {/* Accepting Orders Toggle */}
              <div className="flex items-center gap-3">
                <span className="text-sm font-medium text-gray-700">
                  {acceptingOrders ? 'Accepting Orders' : 'Orders Paused'}
                </span>
                <button
                  type="button"
                  onClick={toggleAcceptingOrders}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                    acceptingOrders ? 'bg-brand-500' : 'bg-gray-300'
                  }`}
                >
                  <span
                    className={`inline-block h-5 w-5 rounded-full bg-white shadow-sm transition-transform ${
                      acceptingOrders ? 'translate-x-[22px]' : 'translate-x-[2px]'
                    }`}
                  />
                </button>
              </div>
            </div>
          </div>
        </Card>
      </motion.div>

      {/* Business Info */}
      <motion.div variants={fadeInUp}>
        <Card>
          <h3 className="text-lg font-semibold text-gray-900">Business Information</h3>
          <p className="mt-1 text-sm text-gray-500">
            This is displayed publicly on your kitchen profile
          </p>

          <div className="mt-6 space-y-5">
            <Input
              label="Business Name"
              placeholder="e.g. Meena's Kitchen"
              error={errors.businessName?.message}
              {...register('businessName')}
            />

            <Textarea
              label="Description"
              placeholder="Tell customers about your kitchen, cooking style, and what makes your food special..."
              rows={4}
              error={errors.description?.message}
              {...register('description')}
            />
          </div>
        </Card>
      </motion.div>

      {/* Cuisines */}
      <motion.div variants={fadeInUp}>
        <Card>
          <h3 className="text-lg font-semibold text-gray-900">Cuisines</h3>
          <p className="mt-1 text-sm text-gray-500">Select the cuisines you specialize in</p>

          <div className="mt-4 flex flex-wrap gap-2">
            {CUISINES.map((cuisine) => {
              const isSelected = cuisines.includes(cuisine);
              return (
                <button
                  key={cuisine}
                  type="button"
                  onClick={() => toggleCuisine(cuisine)}
                  className={`inline-flex items-center gap-1.5 rounded-full px-4 py-2 text-sm font-medium transition-all ${
                    isSelected
                      ? 'bg-brand-500 text-white shadow-sm'
                      : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                  }`}
                >
                  {isSelected && <Check className="h-3.5 w-3.5" />}
                  {cuisine}
                </button>
              );
            })}
          </div>
          {errors.cuisines && (
            <p className="mt-2 text-sm text-destructive">{errors.cuisines.message}</p>
          )}
        </Card>
      </motion.div>

      {/* Specialties */}
      <motion.div variants={fadeInUp}>
        <Card>
          <h3 className="text-lg font-semibold text-gray-900">Specialties</h3>
          <p className="mt-1 text-sm text-gray-500">Add your signature dishes or specialties</p>

          <div className="mt-4 flex flex-wrap gap-2">
            {specialties.map((specialty) => (
              <Badge key={specialty} variant="brand" size="lg" className="gap-1.5">
                {specialty}
                <button
                  type="button"
                  onClick={() => removeSpecialty(specialty)}
                  className="ml-0.5 rounded-full p-0.5 hover:bg-primary/20"
                >
                  <X className="h-3 w-3" />
                </button>
              </Badge>
            ))}
            {specialties.length === 0 && (
              <p className="text-sm text-gray-400">No specialties added yet</p>
            )}
          </div>

          <div className="mt-4 flex gap-2">
            <Input
              placeholder="Add a specialty..."
              value={newSpecialty}
              onChange={(e) => setNewSpecialty(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault();
                  addSpecialty();
                }
              }}
              className="flex-1"
            />
            <Button
              type="button"
              variant="outline"
              onClick={addSpecialty}
              disabled={!newSpecialty.trim()}
              leftIcon={<Plus className="h-4 w-4" />}
            >
              Add
            </Button>
          </div>
        </Card>
      </motion.div>

      {/* Business Settings */}
      <motion.div variants={fadeInUp}>
        <Card>
          <div className="flex items-center gap-2">
            <Settings2 className="h-5 w-5 text-gray-400" />
            <h3 className="text-lg font-semibold text-gray-900">Business Settings</h3>
          </div>

          <div className="mt-6 grid gap-6 sm:grid-cols-2">
            <div>
              <label className="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-gray-700">
                <Clock className="h-4 w-4 text-gray-400" />
                Average Prep Time
              </label>
              <select
                {...register('prepTime')}
                className="w-full rounded-lg border-2 border-input bg-background px-4 py-2.5 text-sm shadow-sm transition-all hover:border-primary/30 focus:border-ring focus:outline-none focus:ring-4 focus:ring-ring/20"
              >
                <option value="15-30 min">15-30 minutes</option>
                <option value="30-45 min">30-45 minutes</option>
                <option value="45-60 min">45-60 minutes</option>
                <option value="1-2 hours">1-2 hours</option>
              </select>
            </div>

            <div>
              <label className="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-gray-700">
                <MapPin className="h-4 w-4 text-gray-400" />
                Delivery Radius (km)
              </label>
              <input
                type="number"
                {...register('serviceRadius', { valueAsNumber: true })}
                className="w-full rounded-lg border-2 border-input bg-background px-4 py-2.5 text-sm shadow-sm transition-all hover:border-primary/30 focus:border-ring focus:outline-none focus:ring-4 focus:ring-ring/20"
              />
              {errors.serviceRadius && (
                <p className="mt-1 text-sm text-destructive">{errors.serviceRadius.message}</p>
              )}
            </div>

            <div>
              <label className="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-gray-700">
                <IndianRupee className="h-4 w-4 text-gray-400" />
                Minimum Order
              </label>
              <div className="relative">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-sm text-gray-400">
                  INR
                </span>
                <input
                  type="number"
                  {...register('minimumOrder', { valueAsNumber: true })}
                  className="w-full rounded-lg border-2 border-input bg-background py-2.5 pl-12 pr-4 text-sm shadow-sm transition-all hover:border-primary/30 focus:border-ring focus:outline-none focus:ring-4 focus:ring-ring/20"
                />
              </div>
              {errors.minimumOrder && (
                <p className="mt-1 text-sm text-destructive">{errors.minimumOrder.message}</p>
              )}
            </div>

            <div>
              <label className="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-gray-700">
                <Truck className="h-4 w-4 text-gray-400" />
                Delivery Fee
              </label>
              <div className="relative">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-sm text-gray-400">
                  INR
                </span>
                <input
                  type="number"
                  {...register('deliveryFee', { valueAsNumber: true })}
                  className="w-full rounded-lg border-2 border-input bg-background py-2.5 pl-12 pr-4 text-sm shadow-sm transition-all hover:border-primary/30 focus:border-ring focus:outline-none focus:ring-4 focus:ring-ring/20"
                />
              </div>
              {errors.deliveryFee && (
                <p className="mt-1 text-sm text-destructive">{errors.deliveryFee.message}</p>
              )}
            </div>
          </div>
        </Card>
      </motion.div>

      {/* Documents & Certificates */}
      <motion.div variants={fadeInUp}>
        <Card>
          <div className="flex items-center gap-3 mb-6">
            <FileText className="h-5 w-5 text-brand-500" />
            <div>
              <h3 className="text-lg font-semibold text-gray-900">Documents & Certificates</h3>
              <p className="text-sm text-gray-500">Upload required documents for verification</p>
            </div>
          </div>
          <DocumentsSection chefId={profile?.id} />
        </Card>
      </motion.div>

      {/* Kitchen Setup Link */}
      <motion.div variants={fadeInUp}>
        <Link to="/profile/kitchen">
          <Card hover="border" className="group">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-brand-50">
                  <ChefHat className="h-6 w-6 text-brand-500" />
                </div>
                <div>
                  <h3 className="font-semibold text-gray-900">Kitchen Setup</h3>
                  <p className="text-sm text-gray-500">
                    Operating hours, kitchen photos, and payout details
                  </p>
                </div>
              </div>
              <ArrowRight className="h-5 w-5 text-gray-400 transition-transform group-hover:translate-x-1" />
            </div>
          </Card>
        </Link>
      </motion.div>
    </motion.div>
  );
}

// ============================================================================
// Documents & Certificates Section
// ============================================================================

interface ChefDocument {
  id: string;
  type: string;
  fileName: string;
  fileUrl?: string;
  contentType: string;
  fileSize: number;
  status: string;
  rejectionReason?: string;
  createdAt: string;
}

const DOCUMENT_TYPES = [
  { type: 'fssai_license', label: 'FSSAI License', description: 'Food Safety and Standards Authority certificate', required: true },
  { type: 'pan_card', label: 'PAN Card', description: 'Permanent Account Number card', required: true },
  { type: 'aadhaar_card', label: 'Aadhaar Card', description: 'Aadhaar identity card', required: true },
  { type: 'food_safety_cert', label: 'Food Safety Certificate', description: 'Food safety training certificate', required: false },
  { type: 'cancelled_cheque', label: 'Cancelled Cheque', description: 'For bank account verification', required: false },
];

function DocumentsSection({ chefId }: { chefId?: string }) {
  const queryClient = useQueryClient();
  const [uploadingType, setUploadingType] = useState<string | null>(null);

  const { data: docsData } = useQuery({
    queryKey: ['chef-documents'],
    queryFn: () => apiClient.get<ChefDocument[]>('/chef/documents'),
    enabled: !!chefId,
  });

  const docs: ChefDocument[] = Array.isArray(docsData) ? docsData : [];

  const getDocByType = (type: string) => docs.find((d) => d.type === type);

  const handleUpload = async (file: File, docType: string) => {
    setUploadingType(docType);
    try {
      const formData = new FormData();
      formData.append('file', file);
      formData.append('type', docType);

      const BFF_URL = typeof window !== 'undefined' && window.location.hostname !== 'localhost'
        ? `${window.location.origin}/bff`
        : '/bff';

      const res = await fetch(`${BFF_URL}/api/v1/chef/documents`, {
        method: 'POST',
        credentials: 'include',
        body: formData,
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: 'Upload failed' }));
        throw new Error(err.error || 'Upload failed');
      }

      toast.success(`${DOCUMENT_TYPES.find(d => d.type === docType)?.label || docType} uploaded`);
      queryClient.invalidateQueries({ queryKey: ['chef-documents'] });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setUploadingType(null);
    }
  };

  const statusBadge = (status: string) => {
    switch (status) {
      case 'verified':
        return <span className="inline-flex items-center gap-1 rounded-full bg-green-50 px-2 py-0.5 text-xs font-medium text-green-600"><CheckCircle className="h-3 w-3" />Verified</span>;
      case 'rejected':
        return <span className="inline-flex items-center gap-1 rounded-full bg-red-50 px-2 py-0.5 text-xs font-medium text-red-600"><AlertCircle className="h-3 w-3" />Rejected</span>;
      default:
        return <span className="inline-flex items-center gap-1 rounded-full bg-yellow-50 px-2 py-0.5 text-xs font-medium text-yellow-600"><Clock className="h-3 w-3" />Pending</span>;
    }
  };

  return (
    <div className="space-y-3">
      {DOCUMENT_TYPES.map((docDef) => {
        const existingDoc = getDocByType(docDef.type);
        const isUploading = uploadingType === docDef.type;

        return (
          <div key={docDef.type} className="flex items-center gap-4 rounded-lg border border-gray-200 bg-gray-50/50 p-4">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-brand-50">
              <FileText className="h-5 w-5 text-brand-500" />
            </div>

            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <p className="text-sm font-medium text-gray-900">{docDef.label}</p>
                {docDef.required && <span className="text-xs text-red-500">Required</span>}
              </div>
              <p className="text-xs text-gray-500">{docDef.description}</p>

              {existingDoc && (
                <div className="mt-1 flex items-center gap-2">
                  <p className="text-xs text-gray-400 truncate max-w-[200px]">{existingDoc.fileName}</p>
                  <span className="text-xs text-gray-300">|</span>
                  <p className="text-xs text-gray-400">{formatBytes(existingDoc.fileSize)}</p>
                  {statusBadge(existingDoc.status)}
                </div>
              )}

              {existingDoc?.rejectionReason && (
                <p className="mt-1 text-xs text-red-500">Reason: {existingDoc.rejectionReason}</p>
              )}
            </div>

            <div className="flex items-center gap-2 shrink-0">
              {existingDoc?.fileUrl && (
                <a href={existingDoc.fileUrl} target="_blank" rel="noopener noreferrer"
                  className="rounded-lg p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors" title="Download">
                  <Download className="h-4 w-4" />
                </a>
              )}

              {/* Upload button - can re-upload if not verified */}
              {(!existingDoc || existingDoc.status !== 'verified') && (
                <label className="cursor-pointer">
                  <input
                    type="file"
                    className="hidden"
                    accept=".jpg,.jpeg,.png,.pdf,.webp"
                    onChange={(e) => {
                      const file = e.target.files?.[0];
                      if (file) handleUpload(file, docDef.type);
                      e.target.value = '';
                    }}
                    disabled={isUploading}
                  />
                  <span className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                    existingDoc
                      ? 'border border-gray-200 text-gray-600 hover:bg-gray-100'
                      : 'bg-brand-500 text-white hover:bg-brand-600'
                  } ${isUploading ? 'opacity-50 pointer-events-none' : ''}`}>
                    {isUploading ? (
                      <><Loader2 className="h-3.5 w-3.5 animate-spin" />Uploading...</>
                    ) : existingDoc ? (
                      <><Upload className="h-3.5 w-3.5" />Re-upload</>
                    ) : (
                      <><Upload className="h-3.5 w-3.5" />Upload</>
                    )}
                  </span>
                </label>
              )}

              {existingDoc?.status === 'verified' && (
                <span className="text-xs text-green-600 font-medium">Approved</span>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (!bytes) return '0 B';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
