import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { motion } from 'framer-motion';
import {
  Plus,
  Search,
  Edit2,
  Trash2,
  Eye,
  EyeOff,
  Star,
  Loader2,
  Upload,
  ChefHat,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { Button, Card, Input, Badge, SimpleDialog } from '@/shared/components/ui';
import { fadeInUp, staggerContainer } from '@/shared/utils/animations';
import { useFormatPrice } from '@/shared/utils/format-price';
import type { MenuItem, MenuCategory } from '@/shared/types';

const menuItemSchema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters'),
  description: z.string().optional(),
  price: z.number().min(0.01, 'Price must be greater than 0'),
  comparePrice: z.number().optional(),
  categoryId: z.string().optional(),
  prepTime: z.number().min(1, 'Prep time is required'),
  portionSize: z.string().optional(),
  serves: z.number().min(1).default(1),
  dietaryTags: z.array(z.string()).default([]),
  allergens: z.array(z.string()).default([]),
  isAvailable: z.boolean().default(true),
  isFeatured: z.boolean().default(false),
});

type MenuItemFormData = z.infer<typeof menuItemSchema>;

const DIETARY_TAGS = ['Vegetarian', 'Vegan', 'Gluten-Free', 'Dairy-Free', 'Keto', 'Low-Carb'];
const ALLERGENS = ['Nuts', 'Dairy', 'Gluten', 'Soy', 'Eggs', 'Shellfish', 'Fish'];

export default function ChefMenuPage() {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editingItem, setEditingItem] = useState<MenuItem | null>(null);
  const queryClient = useQueryClient();

  const { data: menuData, isLoading } = useQuery({
    queryKey: ['chef-menu'],
    queryFn: () =>
      apiClient.get<{ categories: MenuCategory[]; items: MenuItem[] }>('/chef/menu'),
  });

  const createMutation = useMutation({
    mutationFn: (data: MenuItemFormData) => apiClient.post<MenuItem>('/chef/menu/items', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-menu'] });
      toast.success('Menu item created successfully');
      setShowForm(false);
    },
    onError: () => {
      toast.error('Failed to create menu item');
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<MenuItemFormData> }) =>
      apiClient.put<MenuItem>(`/chef/menu/items/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-menu'] });
      toast.success('Menu item updated successfully');
      setShowForm(false);
      setEditingItem(null);
    },
    onError: () => {
      toast.error('Failed to update menu item');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => apiClient.delete(`/chef/menu/items/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-menu'] });
      toast.success('Menu item deleted');
    },
  });

  const toggleAvailability = useMutation({
    mutationFn: ({ id, isAvailable }: { id: string; isAvailable: boolean }) =>
      apiClient.put(`/chef/menu/items/${id}`, { isAvailable }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-menu'] });
    },
  });

  const categories = menuData?.categories || [];
  const allItems = menuData?.items || [];

  const filteredItems = allItems.filter((item) => {
    const matchesSearch =
      !searchQuery ||
      item.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      item.description?.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory = !selectedCategory || item.categoryId === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  const handleEdit = (item: MenuItem) => {
    setEditingItem(item);
    setShowForm(true);
  };

  const handleDelete = (item: MenuItem) => {
    if (confirm(`Are you sure you want to delete "${item.name}"?`)) {
      deleteMutation.mutate(item.id);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
      </div>
    );
  }

  return (
    <motion.div
      initial="hidden"
      animate="visible"
      variants={staggerContainer}
      className="space-y-6"
    >
      {/* Header */}
      <motion.div variants={fadeInUp} className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="font-display text-display-xs text-gray-900">Menu Management</h1>
          <p className="mt-1 text-gray-600">
            {allItems.length} item{allItems.length !== 1 ? 's' : ''} in your menu
          </p>
        </div>
        <Button variant="primary" leftIcon={<Plus className="h-5 w-5" />} onClick={() => setShowForm(true)}>
          Add Item
        </Button>
      </motion.div>

      {/* Search and Filter */}
      <motion.div variants={fadeInUp} className="flex flex-col gap-4 sm:flex-row sm:items-center">
        <div className="flex-1 max-w-md">
          <Input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search menu items..."
            leftIcon={<Search className="h-5 w-5" />}
          />
        </div>
        <div className="flex gap-2 overflow-x-auto pb-2 sm:pb-0">
          <Button
            variant={selectedCategory === null ? 'primary' : 'outline'}
            size="sm"
            onClick={() => setSelectedCategory(null)}
          >
            All
          </Button>
          {categories.map((category) => (
            <Button
              key={category.id}
              variant={selectedCategory === category.id ? 'primary' : 'outline'}
              size="sm"
              onClick={() => setSelectedCategory(category.id)}
            >
              {category.name}
            </Button>
          ))}
        </div>
      </motion.div>

      {/* Menu Items Grid */}
      {filteredItems.length === 0 ? (
        <motion.div variants={fadeInUp}>
          <Card variant="filled" padding="lg" className="text-center">
            <ChefHat className="mx-auto h-12 w-12 text-gray-400" />
            <h3 className="mt-4 font-medium text-gray-900">No menu items</h3>
            <p className="mt-2 text-gray-600">
              {searchQuery
                ? 'No items match your search'
                : 'Add your first menu item to get started'}
            </p>
            {!searchQuery && (
              <Button variant="primary" className="mt-4" onClick={() => setShowForm(true)}>
                Add Menu Item
              </Button>
            )}
          </Card>
        </motion.div>
      ) : (
        <motion.div
          variants={staggerContainer}
          className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3"
        >
          {filteredItems.map((item) => (
            <motion.div key={item.id} variants={fadeInUp}>
              <MenuItemCard
                item={item}
                onEdit={() => handleEdit(item)}
                onDelete={() => handleDelete(item)}
                onToggleAvailability={() =>
                  toggleAvailability.mutate({ id: item.id, isAvailable: !item.isAvailable })
                }
              />
            </motion.div>
          ))}
        </motion.div>
      )}

      {/* Add/Edit Form Dialog */}
      <SimpleDialog
        open={showForm}
        onOpenChange={(open) => {
          setShowForm(open);
          if (!open) setEditingItem(null);
        }}
        title={editingItem ? 'Edit Menu Item' : 'Add Menu Item'}
        size="lg"
      >
        <MenuItemForm
          item={editingItem}
          categories={categories}
          onSubmit={(data) => {
            if (editingItem) {
              updateMutation.mutate({ id: editingItem.id, data });
            } else {
              createMutation.mutate(data);
            }
          }}
          onClose={() => {
            setShowForm(false);
            setEditingItem(null);
          }}
          isLoading={createMutation.isPending || updateMutation.isPending}
        />
      </SimpleDialog>
    </motion.div>
  );
}

function MenuItemCard({
  item,
  onEdit,
  onDelete,
  onToggleAvailability,
}: {
  item: MenuItem;
  onEdit: () => void;
  onDelete: () => void;
  onToggleAvailability: () => void;
}) {
  const fp = useFormatPrice();

  return (
    <Card
      variant="default"
      padding="none"
      className={`overflow-hidden ${!item.isAvailable ? 'opacity-60' : ''}`}
    >
      {/* Image */}
      <div className="relative aspect-video">
        {item.imageUrl ? (
          <img
            src={item.imageUrl}
            alt={item.name}
            className="h-full w-full object-cover"
          />
        ) : (
          <div className="h-full w-full bg-gray-100 flex items-center justify-center">
            <ChefHat className="h-12 w-12 text-gray-300" />
          </div>
        )}
        {item.isFeatured && (
          <Badge variant="solid-brand" size="sm" className="absolute top-2 left-2">
            <Star className="mr-1 h-3 w-3" />
            Featured
          </Badge>
        )}
        {!item.isAvailable && (
          <div className="absolute inset-0 flex items-center justify-center bg-black/50">
            <Badge variant="default" size="md">Unavailable</Badge>
          </div>
        )}
      </div>

      {/* Content */}
      <div className="p-4">
        <div className="flex items-start justify-between">
          <div>
            <h3 className="font-semibold text-gray-900">{item.name}</h3>
            <p className="mt-1 text-sm text-gray-500 line-clamp-2">{item.description}</p>
          </div>
        </div>

        {/* Tags */}
        {item.dietaryTags.length > 0 && (
          <div className="mt-3 flex flex-wrap gap-1">
            {item.dietaryTags.slice(0, 3).map((tag) => (
              <Badge key={tag} variant="success" size="sm">
                {tag}
              </Badge>
            ))}
          </div>
        )}

        {/* Price & Prep Time */}
        <div className="mt-4 flex items-center justify-between">
          <div className="flex items-baseline gap-2">
            <span className="text-lg font-bold text-gray-900">{fp(item.price)}</span>
            {item.comparePrice && (
              <span className="text-sm text-gray-400 line-through">
                {fp(item.comparePrice)}
              </span>
            )}
          </div>
          <span className="text-sm text-gray-500">{item.prepTime} min</span>
        </div>

        {/* Actions */}
        <div className="mt-4 flex items-center gap-2 border-t pt-4">
          <Button
            variant={item.isAvailable ? 'outline' : 'success'}
            size="sm"
            className="flex-1"
            onClick={onToggleAvailability}
            leftIcon={item.isAvailable ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          >
            {item.isAvailable ? 'Hide' : 'Show'}
          </Button>
          <Button
            variant="outline"
            size="icon-sm"
            onClick={onEdit}
            className="text-brand-600"
          >
            <Edit2 className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="icon-sm"
            onClick={onDelete}
            className="text-red-600"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </Card>
  );
}

function MenuItemForm({
  item,
  categories,
  onSubmit,
  onClose,
  isLoading,
}: {
  item: MenuItem | null;
  categories: MenuCategory[];
  onSubmit: (data: MenuItemFormData) => void;
  onClose: () => void;
  isLoading: boolean;
}) {
  const {
    register,
    handleSubmit,
    formState: { errors },
    watch,
    setValue,
  } = useForm<MenuItemFormData>({
    resolver: zodResolver(menuItemSchema),
    defaultValues: item
      ? {
          name: item.name,
          description: item.description || '',
          price: item.price,
          comparePrice: item.comparePrice,
          categoryId: item.categoryId,
          prepTime: item.prepTime,
          portionSize: item.portionSize,
          serves: item.serves,
          dietaryTags: item.dietaryTags,
          allergens: item.allergens,
          isAvailable: item.isAvailable,
          isFeatured: item.isFeatured,
        }
      : {
          serves: 1,
          isAvailable: true,
          isFeatured: false,
          dietaryTags: [],
          allergens: [],
        },
  });

  const dietaryTags = watch('dietaryTags') || [];
  const allergens = watch('allergens') || [];

  const toggleTag = (tag: string, field: 'dietaryTags' | 'allergens') => {
    const current = field === 'dietaryTags' ? dietaryTags : allergens;
    if (current.includes(tag)) {
      setValue(field, current.filter((t) => t !== tag));
    } else {
      setValue(field, [...current, tag]);
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      {/* Image Upload */}
      <div>
        <label className="mb-2 block text-sm font-medium text-gray-700">
          Photo
        </label>
        <div className="flex items-center justify-center rounded-xl border-2 border-dashed border-gray-300 p-8 hover:border-brand-400 transition-colors cursor-pointer">
          <div className="text-center">
            <Upload className="mx-auto h-8 w-8 text-gray-400" />
            <p className="mt-2 text-sm text-gray-600">
              Click to upload or drag and drop
            </p>
            <p className="text-xs text-gray-400">PNG, JPG up to 5MB</p>
          </div>
        </div>
      </div>

      {/* Basic Info */}
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <Input
            label="Name *"
            {...register('name')}
            error={errors.name?.message}
          />
        </div>

        <div className="sm:col-span-2">
          <label className="mb-1.5 block text-sm font-medium text-gray-700">Description</label>
          <textarea
            {...register('description')}
            rows={3}
            className="w-full rounded-xl border border-gray-300 px-4 py-3 text-gray-900 placeholder:text-gray-400 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20 transition-all"
          />
        </div>

        <Input
          label="Price *"
          type="number"
          step="0.01"
          {...register('price', { valueAsNumber: true })}
          error={errors.price?.message}
          leftIcon={<span className="text-gray-400">$</span>}
        />

        <Input
          label="Compare Price (optional)"
          type="number"
          step="0.01"
          {...register('comparePrice', { valueAsNumber: true })}
          leftIcon={<span className="text-gray-400">$</span>}
        />

        <div>
          <label className="mb-1.5 block text-sm font-medium text-gray-700">Category</label>
          <select
            {...register('categoryId')}
            className="w-full rounded-xl border border-gray-300 px-4 py-3 text-gray-900 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20 transition-all"
          >
            <option value="">No category</option>
            {categories.map((cat) => (
              <option key={cat.id} value={cat.id}>
                {cat.name}
              </option>
            ))}
          </select>
        </div>

        <Input
          label="Prep Time (min) *"
          type="number"
          {...register('prepTime', { valueAsNumber: true })}
          error={errors.prepTime?.message}
        />

        <Input
          label="Portion Size"
          {...register('portionSize')}
          placeholder="e.g., 500g"
        />

        <Input
          label="Serves"
          type="number"
          {...register('serves', { valueAsNumber: true })}
        />
      </div>

      {/* Dietary Tags */}
      <div>
        <label className="mb-2 block text-sm font-medium text-gray-700">Dietary Tags</label>
        <div className="flex flex-wrap gap-2">
          {DIETARY_TAGS.map((tag) => (
            <Button
              key={tag}
              type="button"
              variant={dietaryTags.includes(tag) ? 'success' : 'outline'}
              size="sm"
              onClick={() => toggleTag(tag, 'dietaryTags')}
            >
              {tag}
            </Button>
          ))}
        </div>
      </div>

      {/* Allergens */}
      <div>
        <label className="mb-2 block text-sm font-medium text-gray-700">Contains Allergens</label>
        <div className="flex flex-wrap gap-2">
          {ALLERGENS.map((allergen) => (
            <Button
              key={allergen}
              type="button"
              variant={allergens.includes(allergen) ? 'danger' : 'outline'}
              size="sm"
              onClick={() => toggleTag(allergen, 'allergens')}
            >
              {allergen}
            </Button>
          ))}
        </div>
      </div>

      {/* Toggles */}
      <div className="flex gap-6">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            {...register('isAvailable')}
            className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
          />
          <span className="text-sm text-gray-700">Available for order</span>
        </label>
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            {...register('isFeatured')}
            className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
          />
          <span className="text-sm text-gray-700">Featured item</span>
        </label>
      </div>

      {/* Actions */}
      <div className="flex justify-end gap-3 border-t pt-6">
        <Button type="button" variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit" variant="primary" isLoading={isLoading}>
          {item ? 'Save Changes' : 'Add Item'}
        </Button>
      </div>
    </form>
  );
}
