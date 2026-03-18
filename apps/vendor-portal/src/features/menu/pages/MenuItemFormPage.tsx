import { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { motion } from 'framer-motion';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  ArrowLeft,
  Save,
  Clock,
  IndianRupee,
  Users,
  UtensilsCrossed,
  Loader2,
  Plus,
  X,
  Upload,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { uploadMenuItemImage, deleteMenuItemImage } from '@/shared/services/upload-service';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';
import { Button } from '@/shared/components/ui/Button';
import { Input, Textarea } from '@/shared/components/ui/Input';
import { Card } from '@/shared/components/ui/Card';
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue,
} from '@/shared/components/ui/Select';
import {
  SimpleDialog,
  DialogFooter,
} from '@/shared/components/ui/Dialog';
import type { MenuItem, MenuCategory, MenuItemImage } from '@/shared/types';
import { useDraftForm } from '@/shared/hooks/useDraftForm';

// --- Zod validation schema ---

const menuItemSchema = z.object({
  name: z
    .string()
    .min(2, 'Name must be at least 2 characters')
    .max(100, 'Name must be under 100 characters'),
  description: z
    .string()
    .min(10, 'Description must be at least 10 characters')
    .max(500, 'Description must be under 500 characters'),
  price: z
    .number({ invalid_type_error: 'Price is required' })
    .min(20, 'Minimum price is ₹20')
    .max(9999.99, 'Price must be under ₹10,000'),
  categoryId: z.string().min(1, 'Category is required'),
  dietaryTags: z.array(z.string()).min(1, 'Select at least one dietary tag'),
  allergens: z.array(z.string()).default([]),
  prepTime: z
    .number({ invalid_type_error: 'Prep time is required' })
    .min(1, 'Prep time must be at least 1 minute')
    .max(480, 'Prep time must be under 8 hours'),
  portionSize: z.string().optional().or(z.literal('')),
  serves: z
    .number({ invalid_type_error: 'Serves count is required' })
    .min(1, 'Must serve at least 1')
    .max(100, 'Must serve under 100'),
});

type MenuItemFormValues = z.infer<typeof menuItemSchema>;

const DIETARY_TAG_OPTIONS = [
  { value: 'vegetarian', label: 'Vegetarian' },
  { value: 'non-vegetarian', label: 'Non-Veg' },
  { value: 'eggetarian', label: 'Eggetarian' },
  { value: 'vegan', label: 'Vegan' },
  { value: 'jain', label: 'Jain' },
  { value: 'gluten-free', label: 'Gluten-Free' },
  { value: 'halal', label: 'Halal' },
  { value: 'kosher', label: 'Kosher' },
  { value: 'nut-free', label: 'Nut-Free' },
  { value: 'dairy-free', label: 'Dairy-Free' },
  { value: 'sugar-free', label: 'Sugar-Free' },
];

const NEW_CATEGORY_VALUE = '__new__';

function AllergenTagInput({
  value: rawValue,
  onChange,
  error,
}: {
  value: string[];
  onChange: (v: string[]) => void;
  error?: string;
}) {
  // Defensive: handle old string format from stale drafts
  const value = Array.isArray(rawValue) ? rawValue : (typeof rawValue === 'string' && rawValue ? (rawValue as string).split(',').map((s) => s.trim().toLowerCase()).filter(Boolean) : []);
  const [input, setInput] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  const addTag = (tag: string) => {
    const trimmed = tag.trim().toLowerCase();
    if (trimmed && !value.includes(trimmed)) {
      onChange([...value, trimmed]);
    }
    setInput('');
  };

  const removeTag = (tag: string) => {
    onChange(value.filter((t) => t !== tag));
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault();
      addTag(input);
    } else if (e.key === 'Backspace' && !input && value.length > 0) {
      removeTag(value[value.length - 1]!);
    }
  };

  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-foreground">Allergens</label>
      <div
        className={`flex min-h-[42px] flex-wrap items-center gap-1.5 rounded-lg border-2 px-3 py-2 transition-all focus-within:border-ring focus-within:ring-4 focus-within:ring-ring/20 ${
          error ? 'border-destructive' : 'border-input hover:border-primary/30'
        }`}
        onClick={() => inputRef.current?.focus()}
      >
        {value.map((tag) => (
          <span
            key={tag}
            className="inline-flex items-center gap-1 rounded-md bg-destructive/10 px-2 py-0.5 text-xs font-medium text-destructive"
          >
            {tag}
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                removeTag(tag);
              }}
              className="rounded-full p-0.5 hover:bg-destructive/20"
            >
              <X className="h-3 w-3" />
            </button>
          </span>
        ))}
        <input
          ref={inputRef}
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          onBlur={() => { if (input.trim()) addTag(input); }}
          placeholder={value.length === 0 ? 'Type an allergen and press Enter' : ''}
          className="min-w-[120px] flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground/60"
        />
      </div>
      <p className="mt-1 text-xs text-muted-foreground">
        Type each allergen and press Enter to add
      </p>
      {error && <p className="mt-1.5 text-sm text-destructive">{error}</p>}
    </div>
  );
}

export default function MenuItemFormPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [showNewCategoryDialog, setShowNewCategoryDialog] = useState(false);
  const [newCategoryName, setNewCategoryName] = useState('');

  const isEditMode = Boolean(id);
  const draftKey = isEditMode ? `menu-item-edit-${id}` : 'menu-item-new';
  const { loadDraft, saveDraft, clearDraft } = useDraftForm<MenuItemFormValues>(draftKey);
  const draftLoadedRef = useRef(false);

  // Fetch existing item for edit mode
  const {
    data: existingItem,
    isLoading: isLoadingItem,
  } = useQuery<MenuItem>({
    queryKey: ['chef-menu-item', id],
    queryFn: () => apiClient.get<MenuItem>(`/chef/menu/items/${id}`),
    enabled: isEditMode,
  });

  // Fetch categories
  const { data: categories = [] } = useQuery<MenuCategory[]>({
    queryKey: ['chef-menu-categories'],
    queryFn: () => apiClient.get<MenuCategory[]>('/chef/menu/categories'),
  });

  // Create category mutation
  const createCategory = useMutation({
    mutationFn: (name: string) =>
      apiClient.post<MenuCategory>('/chef/menu/categories', { name }),
    onSuccess: (newCategory) => {
      queryClient.invalidateQueries({ queryKey: ['chef-menu-categories'] });
      toast.success(`Category "${newCategory.name}" created`);
      // Auto-select the newly created category
      setValue('categoryId', newCategory.id);
      setShowNewCategoryDialog(false);
      setNewCategoryName('');
    },
    onError: (err: unknown) => {
      const message = (err as { message?: string })?.message ?? '';
      if (message.toLowerCase().includes('already exists')) {
        toast.error('A category with this name already exists');
      } else {
        toast.error('Failed to create category');
      }
    },
  });

  // Case-insensitive duplicate check for new category name
  const isDuplicateCategory = categories.some(
    (c) => c.name.toLowerCase() === newCategoryName.trim().toLowerCase()
  );

  // Form setup
  const {
    register,
    handleSubmit,
    control,
    reset,
    watch,
    setValue,
    formState: { errors, isDirty },
  } = useForm<MenuItemFormValues>({
    resolver: zodResolver(menuItemSchema),
    defaultValues: {
      name: '',
      description: '',
      price: undefined as unknown as number,
      categoryId: '',
      dietaryTags: [],
      allergens: [],
      prepTime: undefined as unknown as number,
      portionSize: '',
      serves: 1,
    },
  });

  // Normalize allergens from old string format or API response to string[]
  const normalizeAllergens = (val: unknown): string[] => {
    if (Array.isArray(val)) return val;
    if (typeof val === 'string' && val.trim()) {
      return val.split(',').map((a) => a.trim().toLowerCase()).filter(Boolean);
    }
    return [];
  };

  // Restore draft on mount (new items) or populate from existing item (edit mode)
  useEffect(() => {
    if (draftLoadedRef.current) return;
    draftLoadedRef.current = true;

    if (existingItem) {
      // Edit mode: load from server, then overlay any draft edits
      const serverValues: MenuItemFormValues = {
        name: existingItem.name,
        description: existingItem.description || '',
        price: existingItem.price,
        categoryId: existingItem.categoryId || '',
        dietaryTags: existingItem.dietaryTags || [],
        allergens: normalizeAllergens(existingItem.allergens),
        prepTime: existingItem.prepTime,
        portionSize: existingItem.portionSize || '',
        serves: existingItem.serves,
      };
      // In edit mode, always use server data (not stale drafts from other items)
      clearDraft();
      reset(serverValues);
    } else {
      // New mode: restore draft if available
      const draft = loadDraft();
      if (draft) {
        draft.allergens = normalizeAllergens(draft.allergens);
        reset(draft);
        toast.info('Restored your unsaved draft');
      }
    }
  }, [existingItem, reset, loadDraft, clearDraft]);

  // Auto-save draft on form changes
  const formValues = watch();
  useEffect(() => {
    // Only save if user has started filling something
    if (formValues.name || formValues.description || formValues.price) {
      saveDraft(formValues);
    }
  }, [formValues, saveDraft]);

  // Create mutation
  const createItem = useMutation({
    mutationFn: (data: MenuItemFormValues) => {
      return apiClient.post<MenuItem>('/chef/menu/items', data);
    },
    onSuccess: async (newItem) => {
      // Upload pending images for the newly created item
      if (pendingFiles.length > 0) {
        await uploadPendingImages(newItem.id);
      }
      clearDraft();
      queryClient.invalidateQueries({ queryKey: ['chef-menu'] });
      toast.success('Menu item created successfully');
      navigate('/menu');
    },
    onError: () => {
      toast.error('Failed to create menu item');
    },
  });

  // Update mutation
  const updateItem = useMutation({
    mutationFn: (data: MenuItemFormValues) => {
      return apiClient.put<MenuItem>(`/chef/menu/items/${id}`, data);
    },
    onSuccess: async () => {
      // Upload any new pending images
      if (pendingFiles.length > 0 && id) {
        await uploadPendingImages(id);
      }
      clearDraft();
      queryClient.invalidateQueries({ queryKey: ['chef-menu'] });
      queryClient.invalidateQueries({ queryKey: ['chef-menu-item', id] });
      toast.success('Menu item updated successfully');
      navigate('/menu');
    },
    onError: () => {
      toast.error('Failed to update menu item');
    },
  });

  const onSubmit = (data: MenuItemFormValues) => {
    if (itemImages.length === 0 && pendingFiles.length === 0) {
      toast.error('At least one image is required');
      return;
    }
    if (isEditMode) {
      updateItem.mutate(data);
    } else {
      createItem.mutate(data);
    }
  };

  const isSubmitting = createItem.isPending || updateItem.isPending;

  // Image management state
  const [itemImages, setItemImages] = useState<MenuItemImage[]>([]);
  const [pendingFiles, setPendingFiles] = useState<File[]>([]);
  const [isUploading, setIsUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Sync images from existing item (edit mode)
  useEffect(() => {
    if (existingItem?.images) {
      setItemImages(existingItem.images);
    }
  }, [existingItem]);

  const totalImageCount = itemImages.length + pendingFiles.length;
  const canAddMore = totalImageCount < 5;

  const handleFileSelect = useCallback((files: FileList | null) => {
    if (!files) return;
    const allowed = ['image/jpeg', 'image/png', 'image/webp'];
    const maxSize = 5 * 1024 * 1024;
    const remaining = 5 - totalImageCount;
    const newFiles: File[] = [];

    for (let i = 0; i < Math.min(files.length, remaining); i++) {
      const f = files.item(i);
      if (!f) continue;
      if (!allowed.includes(f.type)) {
        toast.error(`${f.name}: Invalid type. Allowed: JPEG, PNG, WebP.`);
        continue;
      }
      if (f.size > maxSize) {
        toast.error(`${f.name}: Too large. Max 5 MB.`);
        continue;
      }
      newFiles.push(f);
    }
    if (newFiles.length > 0) {
      setPendingFiles((prev) => [...prev, ...newFiles]);
    }
  }, [totalImageCount]);

  const removePendingFile = (index: number) => {
    setPendingFiles((prev) => prev.filter((_, i) => i !== index));
  };

  const handleDeleteImage = async (image: MenuItemImage) => {
    if (!existingItem) return;
    try {
      await deleteMenuItemImage(existingItem.id, image.id);
      setItemImages((prev) => prev.filter((img) => img.id !== image.id));
      queryClient.invalidateQueries({ queryKey: ['chef-menu-item', id] });
      toast.success('Image deleted');
    } catch {
      toast.error('Failed to delete image');
    }
  };

  // Upload pending files to the API (called after item creation or directly in edit mode)
  const uploadPendingImages = async (itemId: string) => {
    if (pendingFiles.length === 0) return;
    setIsUploading(true);
    try {
      for (const file of pendingFiles) {
        const img = await uploadMenuItemImage(itemId, file);
        setItemImages((prev) => [...prev, img]);
      }
      setPendingFiles([]);
      queryClient.invalidateQueries({ queryKey: ['chef-menu-item', itemId] });
    } catch {
      toast.error('Some images failed to upload');
    } finally {
      setIsUploading(false);
    }
  };

  const watchedDietaryTags = watch('dietaryTags') ?? [];

  // Loading state for edit mode
  if (isEditMode && isLoadingItem) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <motion.div
      variants={staggerContainer}
      initial="hidden"
      animate="visible"
      className="mx-auto max-w-3xl space-y-6"
    >
      {/* Page header */}
      <motion.div variants={fadeInUp} className="flex items-center gap-4">
        <Button variant="ghost" size="icon" asChild>
          <Link to="/menu" aria-label="Back to menu">
            <ArrowLeft className="h-5 w-5" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-bold text-foreground">
            {isEditMode ? 'Edit Menu Item' : 'Add Menu Item'}
          </h1>
          <p className="mt-0.5 text-sm text-muted-foreground">
            {isEditMode
              ? 'Update the details of your dish'
              : 'Add a new dish to your menu'}
          </p>
        </div>
      </motion.div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {/* Basic info */}
        <motion.div variants={fadeInUp}>
          <Card>
            <h2 className="mb-4 text-lg font-semibold text-foreground">
              Basic Information
            </h2>
            <div className="space-y-4">
              <Input
                label="Item Name"
                placeholder="e.g. Paneer Butter Masala"
                error={errors.name?.message}
                {...register('name')}
              />

              <Textarea
                label="Description"
                placeholder="Describe your dish - ingredients, taste, what makes it special..."
                rows={3}
                error={errors.description?.message}
                {...register('description')}
              />

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <Input
                  label="Price (₹)"
                  type="number"
                  step="0.01"
                  min="0"
                  placeholder="0.00"
                  error={errors.price?.message}
                  leftIcon={<IndianRupee className="h-4 w-4" />}
                  {...register('price', { valueAsNumber: true })}
                />

                <Controller
                  name="categoryId"
                  control={control}
                  render={({ field }) => (
                    <div className="w-full">
                      <label className="mb-1.5 block text-sm font-medium text-foreground">
                        Category
                      </label>
                      <Select
                        value={field.value || ''}
                        onValueChange={(value) => {
                          if (value === NEW_CATEGORY_VALUE) {
                            setShowNewCategoryDialog(true);
                          } else {
                            field.onChange(value);
                          }
                        }}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select category" />
                        </SelectTrigger>
                        <SelectContent>
                          {categories.map((category) => (
                            <SelectItem key={category.id} value={category.id}>
                              {category.name}
                            </SelectItem>
                          ))}
                          <SelectItem value={NEW_CATEGORY_VALUE}>
                            <span className="flex items-center gap-1.5 text-primary font-medium">
                              <Plus className="h-3.5 w-3.5" />
                              New Category
                            </span>
                          </SelectItem>
                        </SelectContent>
                      </Select>
                      {errors.categoryId?.message && (
                        <p className="mt-1.5 text-sm text-destructive">
                          {errors.categoryId.message}
                        </p>
                      )}
                    </div>
                  )}
                />
              </div>
            </div>
          </Card>
        </motion.div>

        {/* Dietary tags and allergens */}
        <motion.div variants={fadeInUp}>
          <Card>
            <h2 className="mb-4 text-lg font-semibold text-foreground">
              Dietary Information
            </h2>
            <div className="space-y-4">
              {/* Dietary tags checkboxes */}
              <div>
                <label className="mb-2 block text-sm font-medium text-foreground">
                  Dietary Tags
                </label>
                <Controller
                  name="dietaryTags"
                  control={control}
                  render={({ field }) => (
                    <div className="flex flex-wrap gap-2">
                      {DIETARY_TAG_OPTIONS.map((option) => {
                        const tags = field.value ?? [];
                        const isChecked = tags.includes(option.value);
                        return (
                          <label
                            key={option.value}
                            className={`flex cursor-pointer items-center gap-2 rounded-lg border-2 px-3 py-2 text-sm font-medium transition-colors ${
                              isChecked
                                ? 'border-primary bg-primary/5 text-primary'
                                : 'border-border bg-background text-muted-foreground hover:border-primary/30'
                            }`}
                          >
                            <input
                              type="checkbox"
                              className="sr-only"
                              checked={isChecked}
                              onChange={() => {
                                if (isChecked) {
                                  field.onChange(
                                    tags.filter(
                                      (v) => v !== option.value
                                    )
                                  );
                                } else {
                                  field.onChange([
                                    ...tags,
                                    option.value,
                                  ]);
                                }
                              }}
                            />
                            <div
                              className={`flex h-4 w-4 items-center justify-center rounded border-2 transition-colors ${
                                isChecked
                                  ? 'border-primary bg-primary'
                                  : 'border-muted-foreground/30'
                              }`}
                            >
                              {isChecked && (
                                <svg
                                  className="h-2.5 w-2.5 text-primary-foreground"
                                  fill="none"
                                  viewBox="0 0 24 24"
                                  stroke="currentColor"
                                  strokeWidth={3}
                                >
                                  <path
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    d="M5 13l4 4L19 7"
                                  />
                                </svg>
                              )}
                            </div>
                            {option.label}
                          </label>
                        );
                      })}
                    </div>
                  )}
                />
                {/* Custom dietary tag input */}
                <Controller
                  name="dietaryTags"
                  control={control}
                  render={({ field }) => (
                    <div className="mt-2">
                      <input
                        type="text"
                        placeholder="Add custom tag (press Enter)"
                        className="h-9 w-full max-w-xs rounded-lg border border-input bg-background px-3 text-sm placeholder:text-muted-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-ring"
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            e.preventDefault();
                            const val = (e.target as HTMLInputElement).value.trim().toLowerCase();
                            if (val && !(field.value ?? []).includes(val)) {
                              field.onChange([...(field.value ?? []), val]);
                              (e.target as HTMLInputElement).value = '';
                            }
                          }
                        }}
                      />
                    </div>
                  )}
                />
                {watchedDietaryTags.length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-1">
                    {watchedDietaryTags.map((tag) => (
                      <span key={tag} className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2.5 py-0.5 text-xs font-medium text-primary">
                        {DIETARY_TAG_OPTIONS.find((o) => o.value === tag)?.label || tag}
                        <button type="button" onClick={() => {
                          const current = watchedDietaryTags.filter(t => t !== tag);
                          setValue('dietaryTags', current);
                        }} className="ml-0.5 hover:text-destructive">
                          <X className="h-3 w-3" />
                        </button>
                      </span>
                    ))}
                  </div>
                )}
              </div>

              <Controller
                name="allergens"
                control={control}
                render={({ field }) => (
                  <AllergenTagInput
                    value={field.value}
                    onChange={field.onChange}
                    error={errors.allergens?.message}
                  />
                )}
              />
            </div>
          </Card>
        </motion.div>

        {/* Preparation details */}
        <motion.div variants={fadeInUp}>
          <Card>
            <h2 className="mb-4 text-lg font-semibold text-foreground">
              Preparation Details
            </h2>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <Input
                label="Prep Time (minutes)"
                type="number"
                min="1"
                placeholder="30"
                error={errors.prepTime?.message}
                leftIcon={<Clock className="h-4 w-4" />}
                {...register('prepTime', { valueAsNumber: true })}
              />

              <Input
                label="Portion Size"
                placeholder="e.g. 250g, 1 plate"
                error={errors.portionSize?.message}
                leftIcon={<UtensilsCrossed className="h-4 w-4" />}
                {...register('portionSize')}
              />

              <Input
                label="Serves"
                type="number"
                min="1"
                placeholder="1"
                error={errors.serves?.message}
                leftIcon={<Users className="h-4 w-4" />}
                {...register('serves', { valueAsNumber: true })}
              />
            </div>
          </Card>
        </motion.div>

        {/* Images */}
        <motion.div variants={fadeInUp}>
          <Card>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-foreground">
                Images
              </h2>
              <span className="text-sm text-muted-foreground">
                {totalImageCount}/5 (min 1)
              </span>
            </div>

            {/* Image grid */}
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
              {/* Existing uploaded images */}
              {itemImages.map((img) => (
                <div key={img.id} className="group relative aspect-square overflow-hidden rounded-lg border border-border bg-muted">
                  <img
                    src={img.url}
                    alt="Menu item"
                    className="h-full w-full object-cover"
                  />
                  {img.isPrimary && (
                    <span className="absolute left-1 top-1 rounded bg-primary/90 px-1.5 py-0.5 text-[10px] font-medium text-primary-foreground">
                      Primary
                    </span>
                  )}
                  <button
                    type="button"
                    onClick={() => handleDeleteImage(img)}
                    className="absolute right-1 top-1 flex h-6 w-6 items-center justify-center rounded-full bg-red-500/90 text-white opacity-0 transition-opacity group-hover:opacity-100"
                  >
                    <X className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}

              {/* Pending files (not yet uploaded) */}
              {pendingFiles.map((file, i) => (
                <div key={`pending-${i}`} className="group relative aspect-square overflow-hidden rounded-lg border border-dashed border-primary/50 bg-primary/5">
                  <img
                    src={URL.createObjectURL(file)}
                    alt={file.name}
                    className="h-full w-full object-cover opacity-70"
                  />
                  <span className="absolute left-1 top-1 rounded bg-amber-500/90 px-1.5 py-0.5 text-[10px] font-medium text-white">
                    Pending
                  </span>
                  <button
                    type="button"
                    onClick={() => removePendingFile(i)}
                    className="absolute right-1 top-1 flex h-6 w-6 items-center justify-center rounded-full bg-red-500/90 text-white opacity-0 transition-opacity group-hover:opacity-100"
                  >
                    <X className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}

              {/* Add image button */}
              {canAddMore && (
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  className="flex aspect-square flex-col items-center justify-center rounded-lg border-2 border-dashed border-border bg-muted/50 text-muted-foreground transition-colors hover:border-primary hover:bg-primary/5 hover:text-primary"
                >
                  <Upload className="mb-1 h-6 w-6" />
                  <span className="text-xs font-medium">Add Image</span>
                </button>
              )}
            </div>

            <input
              ref={fileInputRef}
              type="file"
              accept="image/jpeg,image/png,image/webp"
              multiple
              className="hidden"
              onChange={(e) => {
                handleFileSelect(e.target.files);
                e.target.value = '';
              }}
            />

            {/* Upload button for edit mode (immediate upload) */}
            {isEditMode && pendingFiles.length > 0 && (
              <div className="mt-3">
                <Button
                  type="button"
                  size="sm"
                  onClick={() => id && uploadPendingImages(id)}
                  isLoading={isUploading}
                  disabled={isUploading}
                  leftIcon={<Upload className="h-4 w-4" />}
                >
                  Upload {pendingFiles.length} image{pendingFiles.length > 1 ? 's' : ''}
                </Button>
              </div>
            )}

            <p className="mt-2 text-xs text-muted-foreground">
              JPEG, PNG, or WebP. Max 5 MB each. First image is the primary display image.
            </p>
          </Card>
        </motion.div>

        {/* Actions */}
        <motion.div
          variants={fadeInUp}
          className="flex items-center justify-between border-t border-border pt-6"
        >
          <Button variant="outline" asChild>
            <Link to="/menu">Cancel</Link>
          </Button>
          <Button
            type="submit"
            isLoading={isSubmitting}
            disabled={isSubmitting || (isEditMode && !isDirty)}
            leftIcon={<Save className="h-4 w-4" />}
          >
            {isEditMode ? 'Save Changes' : 'Create Item'}
          </Button>
        </motion.div>
      </form>

      {/* New Category Dialog */}
      <SimpleDialog
        open={showNewCategoryDialog}
        onOpenChange={setShowNewCategoryDialog}
        title="Create New Category"
        description="Add a category to organize your menu items."
        size="sm"
      >
        <div className="mt-4 space-y-4">
          <div>
            <Input
              label="Category Name"
              placeholder="e.g. Starters, Main Course, Desserts"
              value={newCategoryName}
              onChange={(e) => setNewCategoryName(e.target.value)}
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter' && newCategoryName.trim() && !isDuplicateCategory) {
                  e.preventDefault();
                  createCategory.mutate(newCategoryName.trim());
                }
              }}
            />
            {isDuplicateCategory && (
              <p className="mt-1 text-sm text-red-500">
                A category with this name already exists
              </p>
            )}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowNewCategoryDialog(false);
                setNewCategoryName('');
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={() => createCategory.mutate(newCategoryName.trim())}
              disabled={!newCategoryName.trim() || isDuplicateCategory || createCategory.isPending}
              isLoading={createCategory.isPending}
            >
              Create
            </Button>
          </DialogFooter>
        </div>
      </SimpleDialog>
    </motion.div>
  );
}
