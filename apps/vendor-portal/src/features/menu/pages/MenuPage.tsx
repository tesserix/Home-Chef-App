import { useState, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Search,
  Plus,
  Eye,
  Pencil,
  Trash2,
  UtensilsCrossed,
  Clock,
  Users,
  Filter,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { staggerContainer, fadeInUp } from '@/shared/utils/animations';
import { Button } from '@/shared/components/ui/Button';
import { Badge } from '@/shared/components/ui/Badge';
import { Card } from '@/shared/components/ui/Card';
import { Input } from '@/shared/components/ui/Input';
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue,
} from '@/shared/components/ui/Select';
import { SkeletonFoodCard } from '@/shared/components/ui/Skeleton';
import type { MenuItem, MenuCategory } from '@/shared/types';

export default function MenuPage() {
  const queryClient = useQueryClient();
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());

  // Fetch menu items
  const {
    data: menuItems = [],
    isLoading: isLoadingItems,
    isError: isItemsError,
  } = useQuery<MenuItem[]>({
    queryKey: ['chef-menu'],
    queryFn: () => apiClient.get<MenuItem[]>('/chef/menu'),
  });

  // Fetch categories
  const { data: categories = [] } = useQuery<MenuCategory[]>({
    queryKey: ['chef-menu-categories'],
    queryFn: () => apiClient.get<MenuCategory[]>('/chef/menu/categories'),
  });

  // Toggle availability mutation
  const toggleAvailability = useMutation({
    mutationFn: (item: MenuItem) =>
      apiClient.put(`/chef/menu/items/${item.id}`, {
        isAvailable: !item.isAvailable,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-menu'] });
      toast.success('Availability updated');
    },
    onError: () => {
      toast.error('Failed to update availability');
    },
  });

  // Delete item mutation
  const deleteItem = useMutation({
    mutationFn: (itemId: string) =>
      apiClient.delete(`/chef/menu/items/${itemId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['chef-menu'] });
      toast.success('Menu item deleted');
    },
    onError: () => {
      toast.error('Failed to delete item');
    },
  });

  // Bulk actions
  const bulkToggleAvailability = async (available: boolean) => {
    const ids = Array.from(selectedItems);
    try {
      await Promise.all(ids.map(id => apiClient.put(`/chef/menu/items/${id}`, { isAvailable: available })));
      queryClient.invalidateQueries({ queryKey: ['chef-menu'] });
      toast.success(`${ids.length} item${ids.length > 1 ? 's' : ''} marked ${available ? 'available' : 'unavailable'}`);
      clearSelection();
    } catch {
      toast.error('Failed to update some items');
    }
  };

  const bulkDelete = async () => {
    const ids = Array.from(selectedItems);
    if (!confirm(`Delete ${ids.length} item${ids.length > 1 ? 's' : ''}? This cannot be undone.`)) return;
    try {
      await Promise.all(ids.map(id => apiClient.delete(`/chef/menu/items/${id}`)));
      queryClient.invalidateQueries({ queryKey: ['chef-menu'] });
      toast.success(`${ids.length} item${ids.length > 1 ? 's' : ''} deleted`);
      clearSelection();
    } catch {
      toast.error('Failed to delete some items');
    }
  };

  // Filter items by search and category
  const filteredItems = useMemo(() => {
    return menuItems.filter((item) => {
      const matchesSearch =
        !searchQuery ||
        item.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        item.description?.toLowerCase().includes(searchQuery.toLowerCase());
      const matchesCategory =
        selectedCategory === 'all' || item.categoryId === selectedCategory;
      return matchesSearch && matchesCategory;
    });
  }, [menuItems, searchQuery, selectedCategory]);

  // Bulk selection helpers
  const toggleItemSelection = (itemId: string) => {
    setSelectedItems((prev) => {
      const next = new Set(prev);
      if (next.has(itemId)) {
        next.delete(itemId);
      } else {
        next.add(itemId);
      }
      return next;
    });
  };

  const clearSelection = () => setSelectedItems(new Set());

  const handleDelete = (itemId: string) => {
    if (window.confirm('Are you sure you want to delete this item?')) {
      deleteItem.mutate(itemId);
    }
  };

  // Get category name by id
  const getCategoryName = (categoryId?: string) => {
    if (!categoryId) return 'Uncategorized';
    const category = categories.find((c) => c.id === categoryId);
    return category?.name || 'Uncategorized';
  };

  return (
    <motion.div
      variants={staggerContainer}
      initial="hidden"
      animate="visible"
      className="space-y-6"
    >
      {/* Page header */}
      <motion.div
        variants={fadeInUp}
        className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"
      >
        <div>
          <h1 className="text-2xl font-bold text-foreground">Menu Management</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {menuItems.length} item{menuItems.length !== 1 ? 's' : ''} in your
            menu
          </p>
        </div>
        <Link to="/menu/new">
          <Button leftIcon={<Plus className="h-4 w-4" />}>
            Add Item
          </Button>
        </Link>
      </motion.div>

      {/* Search and filter bar */}
      <motion.div
        variants={fadeInUp}
        className="flex flex-col gap-3 sm:flex-row sm:items-center"
      >
        <div className="flex-1">
          <Input
            placeholder="Search menu items..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            leftIcon={<Search className="h-4 w-4" />}
          />
        </div>
        <div className="w-full sm:w-56">
          <Select value={selectedCategory} onValueChange={setSelectedCategory}>
            <SelectTrigger>
              <div className="flex items-center gap-2">
                <Filter className="h-4 w-4 text-muted-foreground" />
                <SelectValue placeholder="All Categories" />
              </div>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Categories</SelectItem>
              {categories.map((category) => (
                <SelectItem key={category.id} value={category.id}>
                  {category.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </motion.div>

      {/* Bulk actions bar */}
      {selectedItems.size > 0 && (
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -10 }}
          className="flex items-center justify-between rounded-lg border border-primary/20 bg-primary/5 px-4 py-3"
        >
          <span className="text-sm font-medium text-foreground">
            {selectedItems.size} item{selectedItems.size !== 1 ? 's' : ''}{' '}
            selected
          </span>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" onClick={clearSelection}>
              Clear
            </Button>
            <Button variant="success" size="sm" onClick={() => bulkToggleAvailability(true)}>
              Mark Available
            </Button>
            <Button variant="secondary" size="sm" onClick={() => bulkToggleAvailability(false)}>
              Mark Unavailable
            </Button>
            <Button variant="danger" size="sm" onClick={bulkDelete}>
              Delete Selected
            </Button>
          </div>
        </motion.div>
      )}

      {/* Loading state */}
      {isLoadingItems && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <SkeletonFoodCard key={i} />
          ))}
        </div>
      )}

      {/* Error state */}
      {isItemsError && (
        <motion.div variants={fadeInUp}>
          <Card padding="lg" className="text-center">
            <UtensilsCrossed className="mx-auto h-12 w-12 text-muted-foreground/40" />
            <h3 className="mt-4 text-lg font-semibold text-foreground">
              Failed to load menu
            </h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Something went wrong while fetching your menu items.
            </p>
            <Button
              variant="outline"
              size="sm"
              className="mt-4"
              onClick={() =>
                queryClient.invalidateQueries({ queryKey: ['chef-menu'] })
              }
            >
              Try Again
            </Button>
          </Card>
        </motion.div>
      )}

      {/* Empty state */}
      {!isLoadingItems && !isItemsError && filteredItems.length === 0 && (
        <motion.div variants={fadeInUp}>
          <Card padding="lg" className="text-center">
            <UtensilsCrossed className="mx-auto h-12 w-12 text-muted-foreground/40" />
            <h3 className="mt-4 text-lg font-semibold text-foreground">
              {menuItems.length === 0
                ? 'No menu items yet'
                : 'No items match your search'}
            </h3>
            <p className="mt-1 text-sm text-muted-foreground">
              {menuItems.length === 0
                ? 'Start building your menu by adding your first dish.'
                : 'Try adjusting your search or filter criteria.'}
            </p>
            {menuItems.length === 0 && (
              <Link to="/menu/new" className="mt-4 inline-block">
                <Button leftIcon={<Plus className="h-4 w-4" />}>
                  Add Your First Item
                </Button>
              </Link>
            )}
          </Card>
        </motion.div>
      )}

      {/* Menu items grid */}
      {!isLoadingItems && !isItemsError && filteredItems.length > 0 && (
        <motion.div
          variants={staggerContainer}
          className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3"
        >
          {filteredItems.map((item) => (
            <motion.div key={item.id} variants={fadeInUp}>
              <MenuItemCard
                item={item}
                categoryName={getCategoryName(item.categoryId)}
                isSelected={selectedItems.has(item.id)}
                onToggleSelect={() => toggleItemSelection(item.id)}
                onToggleAvailability={() => toggleAvailability.mutate(item)}
                onDelete={() => handleDelete(item.id)}
              />
            </motion.div>
          ))}
        </motion.div>
      )}
    </motion.div>
  );
}

// --- MenuItemCard component ---

interface MenuItemCardProps {
  item: MenuItem;
  categoryName: string;
  isSelected: boolean;
  onToggleSelect: () => void;
  onToggleAvailability: () => void;
  onDelete: () => void;
}

function MenuItemCard({
  item,
  categoryName,
  isSelected,
  onToggleSelect,
  onToggleAvailability,
  onDelete,
}: MenuItemCardProps) {
  return (
    <Card
      padding="none"
      hover="lift"
      className={`overflow-hidden ${isSelected ? 'ring-2 ring-primary' : ''}`}
    >
      {/* Image */}
      <div className="relative aspect-video overflow-hidden bg-muted">
        {item.imageUrl ? (
          <img
            src={item.imageUrl}
            alt={item.name}
            className="h-full w-full object-cover"
            loading="lazy"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center bg-secondary">
            <UtensilsCrossed className="h-10 w-10 text-muted-foreground/30" />
          </div>
        )}

        {/* Category badge */}
        <div className="absolute left-2 top-2">
          <Badge variant="secondary" size="sm">
            {categoryName}
          </Badge>
        </div>

        {/* Select checkbox */}
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onToggleSelect();
          }}
          className={`absolute right-2 top-2 flex h-6 w-6 items-center justify-center rounded-md border-2 transition-colors ${
            isSelected
              ? 'border-primary bg-primary text-primary-foreground'
              : 'border-white/80 bg-white/60 backdrop-blur-sm hover:bg-white/80'
          }`}
        >
          {isSelected && (
            <svg
              className="h-3.5 w-3.5"
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
        </button>

        {/* Status / Featured badges */}
        <div className="absolute bottom-2 left-2 flex gap-1">
          {item.isApproved ? (
            <Badge variant="success" size="sm">Approved</Badge>
          ) : (
            <Badge variant="warning" size="sm">Under Review</Badge>
          )}
          {item.isFeatured && (
            <Badge variant="brand" size="sm">Featured</Badge>
          )}
        </div>
      </div>

      {/* Content */}
      <div className="p-4">
        {/* Name and price */}
        <div className="flex items-start justify-between gap-2">
          <Link to={`/menu/${item.id}`} className="hover:text-primary transition-colors">
            <h3 className="font-semibold text-foreground line-clamp-1">
              {item.name}
            </h3>
          </Link>
          <div className="flex items-baseline gap-1 shrink-0">
            {item.comparePrice && item.comparePrice > item.price && (
              <span className="text-xs text-muted-foreground line-through">
                ₹{item.comparePrice.toFixed(2)}
              </span>
            )}
            <span className="font-bold text-primary">
              ₹{item.price.toFixed(2)}
            </span>
          </div>
        </div>

        {/* Description */}
        {item.description && (
          <p className="mt-1 text-sm text-muted-foreground line-clamp-2">
            {item.description}
          </p>
        )}

        {/* Meta info */}
        <div className="mt-3 flex items-center gap-3 text-xs text-muted-foreground">
          <span className="flex items-center gap-1">
            <Clock className="h-3.5 w-3.5" />
            {item.prepTime} min
          </span>
          <span className="flex items-center gap-1">
            <Users className="h-3.5 w-3.5" />
            Serves {item.serves}
          </span>
          {item.portionSize && (
            <span className="text-muted-foreground/70">
              {item.portionSize}
            </span>
          )}
        </div>

        {/* Dietary tags & allergens */}
        {(() => {
          const tags = item.dietaryTags ?? [];
          const allergens = Array.isArray(item.allergens) ? item.allergens : [];
          if (tags.length === 0 && allergens.length === 0) return null;
          return (
            <div className="mt-2 flex flex-wrap gap-1">
              {tags.map((tag) => (
                <Badge key={`diet-${tag}`} variant="outline" size="sm">
                  {tag}
                </Badge>
              ))}
              {allergens.map((allergen) => (
                <Badge key={`allergen-${allergen}`} variant="destructive" size="sm">
                  {allergen}
                </Badge>
              ))}
            </div>
          );
        })()}

        {/* Actions row */}
        <div className="mt-4 flex items-center justify-between border-t border-border pt-3">
          {/* Availability toggle */}
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onToggleAvailability();
            }}
            className="flex items-center gap-2"
            aria-label={
              item.isAvailable ? 'Mark as unavailable' : 'Mark as available'
            }
          >
            <div
              className={`relative h-5 w-9 rounded-full transition-colors ${
                item.isAvailable ? 'bg-success' : 'bg-muted-foreground/30'
              }`}
            >
              <div
                className={`absolute top-0.5 h-4 w-4 rounded-full bg-white shadow-sm transition-transform ${
                  item.isAvailable ? 'translate-x-4' : 'translate-x-0.5'
                }`}
              />
            </div>
            <span
              className={`text-xs font-medium ${
                item.isAvailable ? 'text-success' : 'text-muted-foreground'
              }`}
            >
              {item.isAvailable ? 'Available' : 'Unavailable'}
            </span>
          </button>

          {/* View / Edit / Delete buttons */}
          <div className="flex items-center gap-1">
            <Button variant="ghost" size="icon-sm" asChild>
              <Link to={`/menu/${item.id}`} aria-label="View item">
                <Eye className="h-4 w-4" />
              </Link>
            </Button>
            <Button variant="ghost" size="icon-sm" asChild>
              <Link to={`/menu/${item.id}/edit`} aria-label="Edit item">
                <Pencil className="h-4 w-4" />
              </Link>
            </Button>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={(e) => {
                e.stopPropagation();
                onDelete();
              }}
              aria-label="Delete item"
              className="text-destructive hover:text-destructive"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>
    </Card>
  );
}
