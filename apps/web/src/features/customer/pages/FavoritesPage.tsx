import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Heart, Clock, MapPin, Trash2, Loader2 } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { useFavoritesStore } from '@/app/store/favorites-store';
import { useAuth } from '@/app/providers/AuthProvider';
import type { FavoriteChef } from '@/shared/types';
import { Card, Badge, Avatar, RatingBadge, Button } from '@/shared/components/ui';

const MAX_FAVORITES = 7;

export default function FavoritesPage() {
  const { isAuthenticated } = useAuth();
  const { toggle } = useFavoritesStore();

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['favorites', 'chefs'],
    queryFn: () => apiClient.get<FavoriteChef[]>('/favorites/chefs'),
    enabled: isAuthenticated,
  });

  // apiClient auto-unwraps { data: [...] } to just the array
  const favorites = data ?? [];
  const count = favorites.length;

  const handleRemove = async (chefId: string, name: string) => {
    await toggle(chefId);
    toast.success(`Removed ${name} from favorites`);
    refetch();
  };

  if (!isAuthenticated) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-gray-50 px-4">
        <Heart className="h-16 w-16 text-gray-300" />
        <h2 className="mt-4 text-xl font-semibold text-gray-900">Log in to see your favorites</h2>
        <p className="mt-2 text-gray-600 text-center">
          Save your favorite home chefs to quickly find them later
        </p>
        <Link to="/login" className="btn-primary mt-6">
          Log In
        </Link>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="container-app">
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          className="mb-8"
        >
          <div className="flex items-center gap-3">
            <Heart className="h-6 w-6 text-red-500 fill-red-500" />
            <h1 className="font-display text-display-sm text-gray-900">My Favorite Chefs</h1>
          </div>
          <p className="mt-2 text-gray-600">
            {count} of {MAX_FAVORITES} slots used
          </p>
          {/* Progress bar */}
          <div className="mt-3 h-2 w-48 rounded-full bg-gray-200">
            <div
              className="h-full rounded-full bg-red-400 transition-all"
              style={{ width: `${(count / MAX_FAVORITES) * 100}%` }}
            />
          </div>
        </motion.div>

        {isLoading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
          </div>
        ) : favorites.length === 0 ? (
          <Card variant="filled" padding="lg" className="text-center">
            <Heart className="mx-auto h-12 w-12 text-gray-300" />
            <h3 className="mt-4 text-lg font-semibold text-gray-900">No favorites yet</h3>
            <p className="mt-2 text-gray-600">
              Browse chefs and tap the heart icon to save up to {MAX_FAVORITES}
            </p>
            <Button asChild variant="primary" className="mt-6">
              <Link to="/chefs">Browse Chefs</Link>
            </Button>
          </Card>
        ) : (
          <AnimatePresence mode="popLayout">
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {favorites.map((fav) => (
                <motion.div
                  key={fav.id}
                  layout
                  initial={{ opacity: 0, scale: 0.95 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.95 }}
                >
                  <Card variant="default" padding="none" hover="lift" className="overflow-hidden">
                    <Link to={`/chefs/${fav.chef.id}`}>
                      <div className="relative h-28 overflow-hidden">
                        <img
                          src={fav.chef.bannerImage || fav.chef.profileImage}
                          alt={fav.chef.businessName}
                          className="h-full w-full object-cover"
                        />
                        <div className="absolute -bottom-6 left-4">
                          <Avatar
                            src={fav.chef.profileImage}
                            alt={fav.chef.businessName}
                            size="lg"
                            shape="square"
                            className="border-3 border-white shadow-md rounded-lg"
                          />
                        </div>
                      </div>

                      <div className="p-4 pt-8">
                        <div className="flex items-start justify-between">
                          <div className="min-w-0 flex-1">
                            <h3 className="font-semibold text-gray-900 truncate">
                              {fav.chef.businessName}
                            </h3>
                            <p className="mt-1 text-sm text-gray-500 truncate">
                              {fav.chef.cuisines?.slice(0, 2).join(' • ')}
                            </p>
                          </div>
                          <RatingBadge value={fav.chef.rating} className="ml-2" />
                        </div>

                        <div className="mt-3 flex items-center gap-4 text-sm text-gray-500">
                          <div className="flex items-center gap-1">
                            <Clock className="h-3.5 w-3.5" />
                            {fav.chef.prepTime}
                          </div>
                          <div className="flex items-center gap-1">
                            <MapPin className="h-3.5 w-3.5" />
                            {fav.chef.serviceRadius} km
                          </div>
                          {fav.chef.acceptingOrders ? (
                            <Badge variant="success" size="sm" dot>Open</Badge>
                          ) : (
                            <Badge variant="default" size="sm">Closed</Badge>
                          )}
                        </div>
                      </div>
                    </Link>

                    <div className="border-t px-4 py-2">
                      <button
                        onClick={() => handleRemove(fav.chef.id, fav.chef.businessName)}
                        className="flex items-center gap-1.5 text-sm text-red-500 hover:text-red-700 transition-colors"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                        Remove
                      </button>
                    </div>
                  </Card>
                </motion.div>
              ))}
            </div>
          </AnimatePresence>
        )}
      </div>
    </div>
  );
}
