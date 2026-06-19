import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Heart, Clock, MapPin, Trash2, Loader2, Star } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { useFavoritesStore } from '@/app/store/favorites-store';
import { useFormatPrice } from '@/shared/utils/format-price';
import { useAuth } from '@/app/providers/AuthProvider';
import type { FavoriteChef, FavoriteDish } from '@/shared/types';
import { Card, Badge, Avatar, RatingBadge, Button } from '@/shared/components/ui';

const MAX_FAVORITES = 7;

type FavTab = 'chefs' | 'dishes';

export default function FavoritesPage() {
  const { isAuthenticated } = useAuth();
  const { toggle, toggleDish } = useFavoritesStore();
  const fp = useFormatPrice();
  const [tab, setTab] = useState<FavTab>('chefs');

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['favorites', 'chefs'],
    queryFn: () => apiClient.get<FavoriteChef[]>('/favorites/chefs'),
    enabled: isAuthenticated,
  });

  const {
    data: dishData,
    isLoading: dishLoading,
    refetch: refetchDishes,
  } = useQuery({
    queryKey: ['favorites', 'dishes'],
    queryFn: () => apiClient.get<FavoriteDish[]>('/favorites/dishes'),
    enabled: isAuthenticated,
  });

  // apiClient auto-unwraps { data: [...] } to just the array
  const favorites = data ?? [];
  const count = favorites.length;
  const dishes = dishData ?? [];

  const handleRemove = async (chefId: string, name: string) => {
    await toggle(chefId);
    toast.success(`Removed ${name} from favorites`);
    refetch();
  };

  const handleRemoveDish = async (menuItemId: string, name: string) => {
    await toggleDish(menuItemId);
    toast.success(`Removed ${name} from saved`);
    refetchDishes();
  };

  if (!isAuthenticated) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-paper px-4">
        <Heart className="h-16 w-16 text-ink-muted"  aria-hidden="true" />
        <h2 className="mt-4 text-xl font-semibold text-ink">Log in to see your favorites</h2>
        <p className="mt-2 text-ink-soft text-center">
          Save your favorite home chefs to quickly find them later
        </p>
        <Button asChild variant="primary" className="mt-6">
          <Link to="/login">Log In</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-paper py-8">
      <div className="container-app">
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          className="mb-6"
        >
          <div className="flex items-center gap-3">
            <Heart className="h-6 w-6 text-paprika fill-paprika"  aria-hidden="true" />
            <h1 className="font-display text-display-sm text-ink">My Favorites</h1>
          </div>
          {tab === 'chefs' ? (
            <>
              <p className="mt-2 text-ink-soft">
                {count} of {MAX_FAVORITES} chef slots used
              </p>
              {/* Progress bar */}
              <div className="mt-3 h-2 w-48 rounded-full bg-mist">
                <div
                  className="h-full rounded-full bg-paprika transition-all"
                  style={{ width: `${(count / MAX_FAVORITES) * 100}%` }}
                />
              </div>
            </>
          ) : (
            <p className="mt-2 text-ink-soft">{dishes.length} saved dishes</p>
          )}
        </motion.div>

        {/* Chefs / Dishes tabs (#237) */}
        <div className="mb-6 flex gap-2" role="tablist">
          {([
            { key: 'chefs', label: 'Chefs', n: count },
            { key: 'dishes', label: 'Dishes', n: dishes.length },
          ] as const).map((t) => (
            <button
              key={t.key}
              type="button"
              role="tab"
              aria-selected={tab === t.key}
              onClick={() => setTab(t.key)}
              className={`rounded-full px-4 py-1.5 text-sm font-medium transition-colors ${
                tab === t.key ? 'bg-paprika text-paper' : 'bg-mist text-ink-soft hover:bg-bone'
              }`}
            >
              {t.label} {t.n}
            </button>
          ))}
        </div>

        {tab === 'dishes' ? (
          dishLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="h-8 w-8 animate-spin text-herb" aria-hidden="true" />
            </div>
          ) : dishes.length === 0 ? (
            <Card variant="filled" padding="lg" className="text-center">
              <Heart className="mx-auto h-12 w-12 text-ink-muted" aria-hidden="true" />
              <h3 className="mt-4 text-lg font-semibold text-ink">No saved dishes yet</h3>
              <p className="mt-2 text-ink-soft">
                Open a chef and tap the heart on a dish to save it here for quick reordering.
              </p>
              <Button asChild variant="primary" className="mt-6">
                <Link to="/chefs">Browse Chefs</Link>
              </Button>
            </Card>
          ) : (
            <AnimatePresence mode="popLayout">
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {dishes.map((fav) => (
                  <motion.div
                    key={fav.id}
                    layout
                    initial={{ opacity: 0, scale: 0.95 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0, scale: 0.95 }}
                  >
                    <Card variant="default" padding="none" hover="lift" className="overflow-hidden">
                      <Link to={`/chefs/${fav.chef.id}`}>
                        {fav.menuItem.imageUrl && (
                          <div className="h-32 overflow-hidden">
                            <img
                              src={fav.menuItem.imageUrl}
                              alt=""
                              width={400}
                              height={128}
                              loading="lazy"
                              decoding="async"
                              className="h-full w-full object-cover"
                              onError={(e) => { e.currentTarget.style.display = 'none'; }}
                            />
                          </div>
                        )}
                        <div className="p-4">
                          <div className="flex items-start justify-between gap-2">
                            <h3 className="min-w-0 flex-1 truncate font-semibold text-ink">
                              {fav.menuItem.name}
                            </h3>
                            <span className="shrink-0 font-semibold text-ink tabular-nums">
                              {fp(fav.menuItem.price)}
                            </span>
                          </div>
                          <p className="mt-1 truncate text-sm text-ink-muted">
                            from {fav.chef.businessName}
                          </p>
                          {fav.menuItem.rating != null && fav.menuItem.rating > 0 && (
                            <div className="mt-1 flex items-center gap-1 text-sm">
                              <Star className="h-3.5 w-3.5 fill-herb text-herb" aria-hidden="true" />
                              <span className="font-medium text-ink">{fav.menuItem.rating.toFixed(1)}</span>
                            </div>
                          )}
                        </div>
                      </Link>
                      <div className="border-t px-4 py-2">
                        <button
                          type="button"
                          onClick={() => handleRemoveDish(fav.menuItemId, fav.menuItem.name)}
                          className="flex items-center gap-1.5 text-sm text-paprika transition-colors hover:text-paprika"
                        >
                          <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                          Remove
                        </button>
                      </div>
                    </Card>
                  </motion.div>
                ))}
              </div>
            </AnimatePresence>
          )
        ) : isLoading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="h-8 w-8 animate-spin text-herb"  aria-hidden="true" />
          </div>
        ) : favorites.length === 0 ? (
          <Card variant="filled" padding="lg" className="text-center">
            <Heart className="mx-auto h-12 w-12 text-ink-muted"  aria-hidden="true" />
            <h3 className="mt-4 text-lg font-semibold text-ink">No favorites yet</h3>
            <p className="mt-2 text-ink-soft">
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
                          alt=""
                          width={400}
                          height={112}
                          loading="lazy"
                          decoding="async"
                          className="h-full w-full object-cover"
                          onError={(e) => { e.currentTarget.style.display = 'none'; }}
                        />
                        <div className="absolute -bottom-6 left-4">
                          <Avatar
                            src={fav.chef.profileImage}
                            alt={fav.chef.businessName}
                            size="lg"
                            shape="square"
                            className="border-3 border-bone shadow-2 rounded-lg"
                          />
                        </div>
                      </div>

                      <div className="p-4 pt-8">
                        <div className="flex items-start justify-between">
                          <div className="min-w-0 flex-1">
                            <h3 className="font-semibold text-ink truncate">
                              {fav.chef.businessName}
                            </h3>
                            <p className="mt-1 text-sm text-ink-muted truncate">
                              {fav.chef.cuisines?.slice(0, 2).join(' • ')}
                            </p>
                          </div>
                          <RatingBadge value={fav.chef.rating} className="ml-2" />
                        </div>

                        <div className="mt-3 flex items-center gap-4 text-sm text-ink-muted">
                          <div className="flex items-center gap-1">
                            <Clock className="h-3.5 w-3.5"  aria-hidden="true" />
                            {fav.chef.prepTime}
                          </div>
                          <div className="flex items-center gap-1">
                            <MapPin className="h-3.5 w-3.5"  aria-hidden="true" />
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
                      <button type="button"
                        onClick={() => handleRemove(fav.chef.id, fav.chef.businessName)}
                        className="flex items-center gap-1.5 text-sm text-paprika hover:text-paprika transition-colors"
                      >
                        <Trash2 className="h-3.5 w-3.5"  aria-hidden="true" />
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
