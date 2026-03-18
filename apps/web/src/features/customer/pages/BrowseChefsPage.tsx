import { useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Search,
  MapPin,
  Clock,
  Filter,
  Heart,
  X,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { useFavoritesStore } from '@/app/store/favorites-store';
import { useAuth } from '@/app/providers/AuthProvider';
import type { Chef, PaginatedResponse, ChefFilters } from '@/shared/types';
import {
  Button,
  Card,
  Input,
  Badge,
  Avatar,
  RatingBadge,
  SimpleSelect,
  SkeletonChefCard,
} from '@/shared/components/ui';

const CUISINES = [
  'South Indian',
  'North Indian',
  'Italian',
  'Japanese',
  'Mexican',
  'Thai',
  'Chinese',
  'Mediterranean',
  'American',
  'Continental',
];

const DIETARY_OPTIONS = [
  'Vegetarian',
  'Vegan',
  'Gluten-Free',
  'Dairy-Free',
  'Keto',
  'Halal',
];

const SORT_OPTIONS = [
  { value: 'rating', label: 'Top Rated' },
  { value: 'distance', label: 'Nearest' },
  { value: 'orders', label: 'Most Popular' },
  { value: 'price', label: 'Price' },
];

const RATING_OPTIONS = [
  { value: '', label: 'Any Rating' },
  { value: '4.5', label: '4.5+ Stars' },
  { value: '4', label: '4+ Stars' },
  { value: '3.5', label: '3.5+ Stars' },
];

const staggerContainer = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.05 },
  },
};

const fadeInUp = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0 },
};

export default function BrowseChefsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [showFilters, setShowFilters] = useState(false);
  const [searchQuery, setSearchQuery] = useState(searchParams.get('search') || '');

  const filters: ChefFilters = {
    search: searchParams.get('search') || undefined,
    cuisine: searchParams.get('cuisine') || undefined,
    dietary: searchParams.get('dietary') || undefined,
    rating: searchParams.get('rating') ? Number(searchParams.get('rating')) : undefined,
    isOpen: searchParams.get('isOpen') === 'true' ? true : undefined,
    sort: (searchParams.get('sort') as ChefFilters['sort']) || 'rating',
    page: Number(searchParams.get('page')) || 1,
    limit: 12,
  };

  const { data, isLoading, error } = useQuery({
    queryKey: ['chefs', filters],
    queryFn: () => apiClient.get<PaginatedResponse<Chef>>('/chefs', filters),
  });

  const updateFilters = (newFilters: Partial<ChefFilters>) => {
    const params = new URLSearchParams(searchParams);
    Object.entries(newFilters).forEach(([key, value]) => {
      if (value !== undefined && value !== '') {
        params.set(key, String(value));
      } else {
        params.delete(key);
      }
    });
    if (!newFilters.page) {
      params.set('page', '1');
    }
    setSearchParams(params);
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    updateFilters({ search: searchQuery });
  };

  const clearAllFilters = () => {
    setSearchParams(new URLSearchParams());
    setSearchQuery('');
  };

  const activeFilterCount = [
    filters.cuisine,
    filters.dietary,
    filters.rating,
    filters.isOpen,
  ].filter(Boolean).length;

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="container-app">
        {/* Header */}
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          className="mb-8"
        >
          <h1 className="font-display text-display-sm text-gray-900">Explore Home Chefs</h1>
          <p className="mt-2 text-gray-600">
            Discover talented home chefs serving authentic homemade food
          </p>
        </motion.div>

        {/* Search and Filters */}
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="mb-6 flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between"
        >
          {/* Search */}
          <form onSubmit={handleSearch} className="flex-1 max-w-xl">
            <Input
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search chefs, dishes, cuisines..."
              leftIcon={<Search className="h-5 w-5" />}
              inputSize="lg"
            />
          </form>

          {/* Sort and Filter buttons */}
          <div className="flex items-center gap-3">
            <SimpleSelect
              options={SORT_OPTIONS}
              value={filters.sort || 'rating'}
              onValueChange={(value) => updateFilters({ sort: value as ChefFilters['sort'] })}
            />

            <Button
              variant={activeFilterCount > 0 ? 'brand-outline' : 'outline'}
              onClick={() => setShowFilters(!showFilters)}
              leftIcon={<Filter className="h-4 w-4" />}
            >
              Filters
              {activeFilterCount > 0 && (
                <Badge variant="solid-brand" size="sm" className="ml-2">
                  {activeFilterCount}
                </Badge>
              )}
            </Button>
          </div>
        </motion.div>

        {/* Filter Panel */}
        <AnimatePresence>
          {showFilters && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: 'auto' }}
              exit={{ opacity: 0, height: 0 }}
              transition={{ duration: 0.2 }}
            >
              <Card variant="default" padding="md" className="mb-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="font-semibold text-gray-900">Filters</h3>
                  {activeFilterCount > 0 && (
                    <Button variant="link" onClick={clearAllFilters}>
                      Clear all
                    </Button>
                  )}
                </div>

                <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
                  <SimpleSelect
                    label="Cuisine"
                    options={[{ value: '', label: 'All Cuisines' }, ...CUISINES.map(c => ({ value: c, label: c }))]}
                    value={filters.cuisine || ''}
                    onValueChange={(value) => updateFilters({ cuisine: value || undefined })}
                  />

                  <SimpleSelect
                    label="Dietary"
                    options={[{ value: '', label: 'All Options' }, ...DIETARY_OPTIONS.map(d => ({ value: d, label: d }))]}
                    value={filters.dietary || ''}
                    onValueChange={(value) => updateFilters({ dietary: value || undefined })}
                  />

                  <SimpleSelect
                    label="Minimum Rating"
                    options={RATING_OPTIONS}
                    value={String(filters.rating || '')}
                    onValueChange={(value) => updateFilters({ rating: value ? Number(value) : undefined })}
                  />

                  <div>
                    <label className="mb-1.5 block text-sm font-medium text-gray-700">
                      Availability
                    </label>
                    <label className="flex items-center gap-2 cursor-pointer mt-3">
                      <input
                        type="checkbox"
                        checked={filters.isOpen || false}
                        onChange={(e) =>
                          updateFilters({ isOpen: e.target.checked ? true : undefined })
                        }
                        className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
                      />
                      <span className="text-gray-700">Open Now</span>
                    </label>
                  </div>
                </div>
              </Card>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Active filters */}
        <AnimatePresence>
          {(filters.search || filters.cuisine || filters.dietary) && (
            <motion.div
              initial={{ opacity: 0, y: -10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              className="mb-6 flex flex-wrap items-center gap-2"
            >
              <span className="text-sm text-gray-500">Active filters:</span>
              {filters.search && (
                <Badge variant="brand" className="gap-1">
                  Search: {filters.search}
                  <button
                    onClick={() => {
                      setSearchQuery('');
                      updateFilters({ search: undefined });
                    }}
                    className="hover:text-brand-900 ml-1"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </Badge>
              )}
              {filters.cuisine && (
                <Badge variant="brand" className="gap-1">
                  {filters.cuisine}
                  <button
                    onClick={() => updateFilters({ cuisine: undefined })}
                    className="hover:text-brand-900 ml-1"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </Badge>
              )}
              {filters.dietary && (
                <Badge variant="brand" className="gap-1">
                  {filters.dietary}
                  <button
                    onClick={() => updateFilters({ dietary: undefined })}
                    className="hover:text-brand-900 ml-1"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </Badge>
              )}
            </motion.div>
          )}
        </AnimatePresence>

        {/* Results */}
        {isLoading ? (
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {Array.from({ length: 8 }).map((_, i) => (
              <SkeletonChefCard key={i} />
            ))}
          </div>
        ) : error ? (
          <Card variant="filled" padding="lg" className="text-center bg-red-50">
            <p className="text-red-600">Failed to load chefs. Please try again.</p>
            <Button variant="primary" className="mt-4" onClick={() => window.location.reload()}>
              Retry
            </Button>
          </Card>
        ) : (data?.data ?? []).length === 0 ? (
          <Card variant="filled" padding="lg" className="text-center">
            <div className="mx-auto h-16 w-16 rounded-full bg-gray-200 flex items-center justify-center mb-4">
              <Search className="h-8 w-8 text-gray-400" />
            </div>
            <h3 className="text-lg font-semibold text-gray-900">No chefs found</h3>
            <p className="mt-2 text-gray-600">
              Try adjusting your filters or search query
            </p>
            <Button variant="primary" className="mt-4" onClick={clearAllFilters}>
              Clear filters
            </Button>
          </Card>
        ) : (
          <>
            {/* Results count */}
            <p className="mb-4 text-sm text-gray-500">
              Showing {data?.data?.length ?? 0} of {data?.pagination?.total ?? 0} chefs
            </p>

            {/* Chef Grid */}
            <motion.div
              variants={staggerContainer}
              initial="hidden"
              animate="visible"
              className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
            >
              {(data?.data ?? []).map((chef) => (
                <motion.div key={chef.id} variants={fadeInUp}>
                  <ChefCardItem chef={chef} />
                </motion.div>
              ))}
            </motion.div>

            {/* Pagination */}
            {data && data.pagination.totalPages > 1 && (
              <div className="mt-8 flex items-center justify-center gap-2">
                <Button
                  variant="outline"
                  onClick={() => updateFilters({ page: filters.page! - 1 })}
                  disabled={!data.pagination.hasPrev}
                >
                  Previous
                </Button>
                <span className="px-4 text-sm text-gray-600">
                  Page {data.pagination.page} of {data.pagination.totalPages}
                </span>
                <Button
                  variant="outline"
                  onClick={() => updateFilters({ page: filters.page! + 1 })}
                  disabled={!data.pagination.hasNext}
                >
                  Next
                </Button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

function ChefCardItem({ chef }: { chef: Chef }) {
  const { isAuthenticated, login } = useAuth();
  const { isFavorite, toggle } = useFavoritesStore();
  const favorited = isFavorite(chef.id);

  const handleFavorite = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    if (!isAuthenticated) {
      toast.error('Please log in to save favorites');
      login();
      return;
    }

    const result = await toggle(chef.id);
    if (result === 'max_limit') {
      toast.error('You can save up to 7 favorite chefs. Remove one first.');
    } else if (result === 'unauthorized') {
      toast.error('Please log in to save favorites');
      login();
    } else if (result === 'error') {
      toast.error('Something went wrong. Please try again.');
    }
  };

  return (
    <Link to={`/chefs/${chef.id}`}>
      <Card variant="default" padding="none" hover="lift" className="overflow-hidden group">
        {/* Banner */}
        <div className="relative h-32 overflow-hidden">
          <img
            src={chef.bannerImage || chef.profileImage}
            alt={chef.businessName}
            className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
          />
          {chef.verified && (
            <Badge variant="success" size="sm" className="absolute top-2 left-2">
              Verified
            </Badge>
          )}
          <button
            onClick={handleFavorite}
            className="absolute top-2 right-2 flex h-8 w-8 items-center justify-center rounded-full bg-white/90 shadow-sm backdrop-blur-sm transition-all hover:bg-white hover:shadow-md"
          >
            <Heart
              className={`h-4 w-4 transition-colors ${
                favorited ? 'fill-red-500 text-red-500' : 'text-gray-600'
              }`}
            />
          </button>
          <div className="absolute -bottom-8 left-4">
            <Avatar
              src={chef.profileImage}
              alt={chef.businessName}
              size="xl"
              shape="square"
              className="border-4 border-white shadow-elevated rounded-xl"
            />
          </div>
        </div>

        {/* Content */}
        <div className="p-4 pt-10">
          <div className="flex items-start justify-between">
            <div className="min-w-0 flex-1">
              <h3 className="font-semibold text-gray-900 truncate group-hover:text-brand-600 transition-colors">
                {chef.businessName}
              </h3>
              <p className="mt-1 text-sm text-gray-500 truncate">
                {chef.cuisines.slice(0, 2).join(' • ')}
              </p>
            </div>
            <RatingBadge value={chef.rating} className="ml-2" />
          </div>

          <p className="mt-3 line-clamp-2 text-sm text-gray-600">{chef.description}</p>

          <div className="mt-4 flex items-center gap-4 text-sm text-gray-500">
            <div className="flex items-center gap-1">
              <Clock className="h-4 w-4" />
              {chef.prepTime}
            </div>
            <div>{chef.priceRange}</div>
          </div>

          <div className="mt-3 flex items-center justify-between text-sm">
            <div className="flex items-center gap-1 text-gray-500">
              <MapPin className="h-4 w-4" />
              {chef.serviceRadius} km radius
            </div>
            {chef.acceptingOrders ? (
              <Badge variant="success" size="sm" dot>
                Open
              </Badge>
            ) : (
              <Badge variant="default" size="sm">
                Closed
              </Badge>
            )}
          </div>
        </div>
      </Card>
    </Link>
  );
}
