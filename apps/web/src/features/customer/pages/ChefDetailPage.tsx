import { useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Star,
  Clock,
  MapPin,
  Heart,
  Share2,
  ChefHat,
  Minus,
  Plus,
  ShoppingCart,
  Check,
  AlertCircle,
  Loader2,
  ChevronRight,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiClient } from '@/shared/services/api-client';
import { useCartStore } from '@/app/store/cart-store';
import { useFavoritesStore } from '@/app/store/favorites-store';
import { useAuth } from '@/app/providers/AuthProvider';
import { useFormatPrice } from '@/shared/utils/format-price';
import { formatDate } from '@/shared/utils/format-date';
import { Button } from '@/shared/components/ui';
import type { Chef, MenuItem, MenuCategory, Review, PaginatedResponse } from '@/shared/types';

export default function ChefDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [showReviews, setShowReviews] = useState(false);

  const { data: chef, isLoading: chefLoading } = useQuery({
    queryKey: ['chef', id],
    queryFn: () => apiClient.get<Chef>(`/chefs/${id}`),
    enabled: !!id,
  });

  const { data: menu, isLoading: menuLoading } = useQuery({
    queryKey: ['chef', id, 'menu'],
    queryFn: () => apiClient.get<{ categories: MenuCategory[]; items: MenuItem[] }>(`/chefs/${id}/menu`),
    enabled: !!id,
  });

  const { data: reviews } = useQuery({
    queryKey: ['chef', id, 'reviews'],
    queryFn: () => apiClient.get<PaginatedResponse<Review>>(`/chefs/${id}/reviews`),
    enabled: !!id && showReviews,
  });

  const fp = useFormatPrice();
  const cart = useCartStore();
  const cartItemCount = cart.getItemCount();
  const { isAuthenticated, login } = useAuth();
  const { isFavorite, toggle } = useFavoritesStore();
  const favorited = id ? isFavorite(id) : false;

  const handleFavorite = async () => {
    if (!isAuthenticated) {
      toast.error('Please log in to save favorites');
      login();
      return;
    }
    if (!id) return;
    const result = await toggle(id);
    if (result === 'max_limit') {
      toast.error('You can save up to 7 favorite chefs. Remove one first.');
    } else if (result === 'unauthorized') {
      toast.error('Please log in to save favorites');
      login();
    } else if (result === 'error') {
      toast.error('Something went wrong. Please try again.');
    }
  };

  if (chefLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-herb" />
      </div>
    );
  }

  if (!chef) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center">
        <AlertCircle className="h-16 w-16 text-ink-muted" />
        <h2 className="mt-4 text-xl font-semibold text-ink">Chef not found</h2>
        <Button asChild variant="primary" className="mt-4">
          <Link to="/chefs">Browse Chefs</Link>
        </Button>
      </div>
    );
  }

  const categories = menu?.categories || [];
  const menuItems = menu?.items || [];
  const filteredItems = selectedCategory
    ? menuItems.filter((item) => item.categoryId === selectedCategory)
    : menuItems;

  return (
    <div className="min-h-screen bg-paper">
      {/* Hero Banner */}
      <div className="relative h-64 md:h-80">
        <img
          src={chef.bannerImage || chef.profileImage}
          alt=""
          width={1600}
          height={640}
          loading="eager"
          decoding="async"
          className="h-full w-full object-cover"
        />
        <div aria-hidden="true" className="absolute inset-0 scrim-bottom" />
      </div>

      {/* Chef Info */}
      <div className="container-app relative -mt-20">
        <div className="rounded-xl bg-bone p-6 shadow-1 md:p-8">
          <div className="flex flex-col gap-6 md:flex-row md:items-start">
            {/* Profile Image */}
            <img
              src={chef.profileImage}
              alt=""
              width={128}
              height={128}
              loading="lazy"
              decoding="async"
              className="h-24 w-24 rounded-xl border-4 border-bone object-cover shadow-3 md:h-32 md:w-32 shrink-0"
              onError={(e) => { e.currentTarget.style.display = 'none'; }}
            />

            {/* Info */}
            <div className="flex-1">
              <div className="flex flex-wrap items-start gap-4">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <h1 className="font-display text-2xl font-semibold text-ink md:text-3xl">
                      {chef.businessName}
                    </h1>
                    {chef.verified && (
                      <span className="inline-flex items-center gap-1 rounded-full bg-herb-tint px-2 py-0.5 text-xs font-medium text-herb">
                        <Check className="h-3 w-3" />
                        Verified
                      </span>
                    )}
                  </div>
                  <p className="mt-1 text-ink-soft">{chef.cuisines.join(' • ')}</p>
                </div>

                {/* Actions */}
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={handleFavorite}
                    aria-label={favorited ? `Remove ${chef.businessName} from favorites` : `Save ${chef.businessName} to favorites`}
                    aria-pressed={favorited}
                  >
                    <Heart aria-hidden="true" className={`h-5 w-5 transition-colors ${favorited ? 'fill-paprika text-paprika' : ''}`} />
                  </Button>
                  <Button variant="outline" size="icon" aria-label="Share chef">
                    <Share2 aria-hidden="true" className="h-5 w-5" />
                  </Button>
                </div>
              </div>

              <p className="mt-4 text-ink-soft">{chef.description}</p>

              {/* Stats */}
              <div className="mt-6 flex flex-wrap gap-6">
                <div className="flex items-center gap-2">
                  <div className="flex items-center gap-1 rounded-lg bg-herb-tint px-2 py-1">
                    <Star className="h-4 w-4 fill-herb text-herb" />
                    <span className="font-semibold text-herb">{chef.rating}</span>
                  </div>
                  <span className="text-sm text-ink-muted">
                    ({chef.totalReviews} reviews)
                  </span>
                </div>

                <div className="flex items-center gap-2 text-sm text-ink-soft">
                  <Clock className="h-4 w-4" />
                  {chef.prepTime} prep time
                </div>

                <div className="flex items-center gap-2 text-sm text-ink-soft">
                  <MapPin className="h-4 w-4" />
                  {chef.serviceRadius} km delivery radius
                </div>

                <div className="flex items-center gap-2 text-sm text-ink-soft">
                  <ChefHat className="h-4 w-4" />
                  {chef.totalOrders}+ orders
                </div>
              </div>

              {/* Status & Fees */}
              <div className="mt-6 flex flex-wrap items-center gap-4 rounded-lg bg-paper p-4">
                <div>
                  {chef.acceptingOrders ? (
                    <span className="inline-flex items-center gap-1 text-herb">
                      <span className="h-2 w-2 rounded-full bg-herb" />
                      Open for orders
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1 text-ink-muted">
                      <span className="h-2 w-2 rounded-full bg-ink-muted" />
                      Currently closed
                    </span>
                  )}
                </div>
                <div className="text-sm text-ink-soft">
                  Min. order: <span className="font-medium">{fp(chef.minimumOrder)}</span>
                </div>
                <div className="text-sm text-ink-soft">
                  Delivery: <span className="font-medium">{fp(chef.deliveryFee)}</span>
                </div>
                <div className="text-sm text-ink-soft">
                  Price range: <span className="font-medium">{chef.priceRange}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Menu Section */}
      <div className="container-app mt-8 pb-32">
        <div className="flex flex-col lg:flex-row lg:gap-8">
          {/* Categories Sidebar */}
          <div className="mb-6 lg:mb-0 lg:w-64 lg:flex-shrink-0">
            <div className="rounded-xl bg-bone p-4 shadow-1 lg:sticky lg:top-24">
              <h3 className="font-semibold text-ink mb-3">Categories</h3>
              <div className="flex flex-wrap gap-2 lg:flex-col lg:gap-1">
                <button
                  onClick={() => setSelectedCategory(null)}
                  className={`rounded-lg px-3 py-2 text-left text-sm transition-colors ${
                    selectedCategory === null
                      ? 'bg-herb-tint text-herb font-medium'
                      : 'text-ink-soft hover:bg-mist'
                  }`}
                >
                  All Items ({menuItems.length})
                </button>
                {categories.map((category) => {
                  const count = menuItems.filter((i) => i.categoryId === category.id).length;
                  return (
                    <button
                      key={category.id}
                      onClick={() => setSelectedCategory(category.id)}
                      className={`rounded-lg px-3 py-2 text-left text-sm transition-colors ${
                        selectedCategory === category.id
                          ? 'bg-herb-tint text-herb font-medium'
                          : 'text-ink-soft hover:bg-mist'
                      }`}
                    >
                      {category.name} ({count})
                    </button>
                  );
                })}
              </div>

              {/* Reviews Link */}
              <div className="mt-6 pt-4 border-t">
                <button
                  onClick={() => setShowReviews(!showReviews)}
                  className="flex items-center justify-between w-full text-left text-sm text-ink-soft hover:text-ink"
                >
                  <span>Reviews ({chef.totalReviews})</span>
                  <ChevronRight className={`h-4 w-4 transition-transform ${showReviews ? 'rotate-90' : ''}`} />
                </button>
              </div>
            </div>
          </div>

          {/* Menu Items */}
          <div className="flex-1">
            {menuLoading ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 className="h-8 w-8 animate-spin text-herb" />
              </div>
            ) : showReviews ? (
              <ReviewsList reviews={reviews?.data || []} />
            ) : (
              <div className="space-y-4">
                {filteredItems.length === 0 ? (
                  <div className="rounded-xl bg-bone p-8 text-center shadow-1">
                    <p className="text-ink-muted">No items in this category</p>
                  </div>
                ) : (
                  filteredItems.map((item) => (
                    <MenuItemCard
                      key={item.id}
                      item={item}
                      chefId={chef.id}
                      chefInfo={{
                        id: chef.id,
                        businessName: chef.businessName,
                        profileImage: chef.profileImage,
                        deliveryFee: chef.deliveryFee,
                        minimumOrder: chef.minimumOrder,
                      }}
                    />
                  ))
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Floating Cart Button */}
      {cartItemCount > 0 && (
        <div className="fixed bottom-6 left-4 right-4 md:left-auto md:right-6 md:w-auto z-40">
          <Button asChild variant="primary" size="lg" className="w-full justify-between gap-4 shadow-3 md:justify-center">
            <Link to="/cart" aria-label={`View cart, ${cartItemCount} items, ${fp(cart.getSubtotal())}`}>
              <div className="flex items-center gap-3">
                <ShoppingCart aria-hidden="true" className="h-5 w-5" />
                <span>View Cart</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="rounded-full bg-bone/20 px-2 py-0.5 text-sm">
                  {cartItemCount} items
                </span>
                <span className="font-semibold">{fp(cart.getSubtotal())}</span>
              </div>
            </Link>
          </Button>
        </div>
      )}
    </div>
  );
}

function MenuItemCard({
  item,
  chefInfo,
}: {
  item: MenuItem;
  chefId?: string;
  chefInfo: {
    id: string;
    businessName: string;
    profileImage?: string;
    deliveryFee: number;
    minimumOrder: number;
  };
}) {
  const [quantity, setQuantity] = useState(1);
  const fp = useFormatPrice();
  const cart = useCartStore();

  const cartItem = cart.items.find((i) => i.menuItemId === item.id);

  const handleAddToCart = () => {
    try {
      // Set chef if this is first item
      if (cart.items.length === 0) {
        cart.setChef(chefInfo);
      }
      cart.addItem(item, quantity);
      toast.success(`Added ${item.name} to cart`);
      setQuantity(1);
    } catch (error) {
      if (error instanceof Error && error.message === 'DIFFERENT_CHEF') {
        toast.error('Your cart has items from another chef. Clear cart first.');
      }
    }
  };

  return (
    <div className="card overflow-hidden">
      <div className="flex gap-4 p-4">
        {/* Image */}
        {item.imageUrl && (
          <div className="relative h-28 w-28 flex-shrink-0 overflow-hidden rounded-lg">
            <img
              src={item.imageUrl}
              alt=""
              width={112}
              height={112}
              loading="lazy"
              decoding="async"
              className="h-full w-full object-cover"
              onError={(e) => { e.currentTarget.style.display = 'none'; }}
            />
            {item.isFeatured && (
              <div className="absolute top-1 left-1 rounded bg-herb px-1.5 py-0.5 text-xs font-medium text-paper">
                Popular
              </div>
            )}
          </div>
        )}

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-2">
            <div>
              <h3 className="font-semibold text-ink">{item.name}</h3>
              <p className="mt-1 text-sm text-ink-muted line-clamp-2">
                {item.description}
              </p>
            </div>
          </div>

          {/* Tags */}
          {item.dietaryTags.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-1">
              {item.dietaryTags.map((tag) => (
                <span
                  key={tag}
                  className="rounded bg-herb-tint px-2 py-0.5 text-xs text-herb"
                >
                  {tag}
                </span>
              ))}
            </div>
          )}

          {/* Price and Actions */}
          <div className="mt-4 flex items-center justify-between">
            <div className="flex items-baseline gap-2">
              <span className="text-lg font-semibold text-ink">
                {fp(item.price)}
              </span>
              {item.comparePrice && (
                <span className="text-sm text-ink-muted line-through">
                  {fp(item.comparePrice)}
                </span>
              )}
            </div>

            {item.isAvailable ? (
              <div className="flex items-center gap-2">
                {/* Quantity selector */}
                <div className="flex items-center rounded-lg border">
                  <button
                    onClick={() => setQuantity(Math.max(1, quantity - 1))}
                    className="p-2 hover:bg-mist"
                  >
                    <Minus className="h-4 w-4" />
                  </button>
                  <span className="w-8 text-center">{quantity}</span>
                  <button
                    onClick={() => setQuantity(quantity + 1)}
                    className="p-2 hover:bg-mist"
                  >
                    <Plus className="h-4 w-4" />
                  </button>
                </div>

                <Button variant="primary" size="sm" onClick={handleAddToCart}>
                  {cartItem ? 'Add More' : 'Add'}
                </Button>
              </div>
            ) : (
              <span className="text-sm text-ink-muted">Unavailable</span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function ReviewsList({ reviews }: { reviews: Review[] }) {
  if (reviews.length === 0) {
    return (
      <div className="rounded-xl bg-bone p-8 text-center shadow-1">
        <p className="text-ink-muted">No reviews yet</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h3 className="text-xl font-semibold text-ink">Customer Reviews</h3>
      {reviews.map((review) => (
        <div key={review.id} className="card p-4">
          <div className="flex items-start gap-4">
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <div className="flex items-center">
                  {[...Array(5)].map((_, i) => (
                    <Star
                      key={i}
                      className={`h-4 w-4 ${
                        i < review.overallRating
                          ? 'fill-amber text-amber'
                          : 'text-ink-muted'
                      }`}
                    />
                  ))}
                </div>
                <span className="text-sm text-ink-muted">
                  {formatDate(review.createdAt)}
                </span>
              </div>
              {review.title && (
                <h4 className="mt-2 font-medium text-ink">{review.title}</h4>
              )}
              {review.comment && (
                <p className="mt-2 text-ink-soft">{review.comment}</p>
              )}
              {review.chefResponse && (
                <div className="mt-3 rounded-lg bg-paper p-3">
                  <p className="text-sm font-medium text-ink-soft">Chef's Response:</p>
                  <p className="mt-1 text-sm text-ink-soft">{review.chefResponse}</p>
                </div>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
