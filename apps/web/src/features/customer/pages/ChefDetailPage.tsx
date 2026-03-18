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
        <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
      </div>
    );
  }

  if (!chef) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center">
        <AlertCircle className="h-16 w-16 text-gray-400" />
        <h2 className="mt-4 text-xl font-semibold text-gray-900">Chef not found</h2>
        <Link to="/chefs" className="btn-primary mt-4">
          Browse Chefs
        </Link>
      </div>
    );
  }

  const categories = menu?.categories || [];
  const menuItems = menu?.items || [];
  const filteredItems = selectedCategory
    ? menuItems.filter((item) => item.categoryId === selectedCategory)
    : menuItems;

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Hero Banner */}
      <div className="relative h-64 md:h-80">
        <img
          src={chef.bannerImage || chef.profileImage}
          alt={chef.businessName}
          className="h-full w-full object-cover"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-black/60 to-transparent" />
      </div>

      {/* Chef Info */}
      <div className="container-app relative -mt-20">
        <div className="rounded-xl bg-white p-6 shadow-sm md:p-8">
          <div className="flex flex-col gap-6 md:flex-row md:items-start">
            {/* Profile Image */}
            <img
              src={chef.profileImage}
              alt={chef.businessName}
              className="h-24 w-24 rounded-xl border-4 border-white object-cover shadow-lg md:h-32 md:w-32"
            />

            {/* Info */}
            <div className="flex-1">
              <div className="flex flex-wrap items-start gap-4">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <h1 className="text-2xl font-bold text-gray-900 md:text-3xl">
                      {chef.businessName}
                    </h1>
                    {chef.verified && (
                      <span className="inline-flex items-center gap-1 rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700">
                        <Check className="h-3 w-3" />
                        Verified
                      </span>
                    )}
                  </div>
                  <p className="mt-1 text-gray-600">{chef.cuisines.join(' • ')}</p>
                </div>

                {/* Actions */}
                <div className="flex gap-2">
                  <button onClick={handleFavorite} className="btn-outline p-2">
                    <Heart className={`h-5 w-5 transition-colors ${favorited ? 'fill-red-500 text-red-500' : ''}`} />
                  </button>
                  <button className="btn-outline p-2">
                    <Share2 className="h-5 w-5" />
                  </button>
                </div>
              </div>

              <p className="mt-4 text-gray-600">{chef.description}</p>

              {/* Stats */}
              <div className="mt-6 flex flex-wrap gap-6">
                <div className="flex items-center gap-2">
                  <div className="flex items-center gap-1 rounded-lg bg-green-50 px-2 py-1">
                    <Star className="h-4 w-4 fill-green-500 text-green-500" />
                    <span className="font-semibold text-green-700">{chef.rating}</span>
                  </div>
                  <span className="text-sm text-gray-500">
                    ({chef.totalReviews} reviews)
                  </span>
                </div>

                <div className="flex items-center gap-2 text-sm text-gray-600">
                  <Clock className="h-4 w-4" />
                  {chef.prepTime} prep time
                </div>

                <div className="flex items-center gap-2 text-sm text-gray-600">
                  <MapPin className="h-4 w-4" />
                  {chef.serviceRadius} km delivery radius
                </div>

                <div className="flex items-center gap-2 text-sm text-gray-600">
                  <ChefHat className="h-4 w-4" />
                  {chef.totalOrders}+ orders
                </div>
              </div>

              {/* Status & Fees */}
              <div className="mt-6 flex flex-wrap items-center gap-4 rounded-lg bg-gray-50 p-4">
                <div>
                  {chef.acceptingOrders ? (
                    <span className="inline-flex items-center gap-1 text-green-600">
                      <span className="h-2 w-2 rounded-full bg-green-500" />
                      Open for orders
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1 text-gray-500">
                      <span className="h-2 w-2 rounded-full bg-gray-400" />
                      Currently closed
                    </span>
                  )}
                </div>
                <div className="text-sm text-gray-600">
                  Min. order: <span className="font-medium">{fp(chef.minimumOrder)}</span>
                </div>
                <div className="text-sm text-gray-600">
                  Delivery: <span className="font-medium">{fp(chef.deliveryFee)}</span>
                </div>
                <div className="text-sm text-gray-600">
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
            <div className="rounded-xl bg-white p-4 shadow-sm lg:sticky lg:top-24">
              <h3 className="font-semibold text-gray-900 mb-3">Categories</h3>
              <div className="flex flex-wrap gap-2 lg:flex-col lg:gap-1">
                <button
                  onClick={() => setSelectedCategory(null)}
                  className={`rounded-lg px-3 py-2 text-left text-sm transition-colors ${
                    selectedCategory === null
                      ? 'bg-brand-100 text-brand-700 font-medium'
                      : 'text-gray-600 hover:bg-gray-100'
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
                          ? 'bg-brand-100 text-brand-700 font-medium'
                          : 'text-gray-600 hover:bg-gray-100'
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
                  className="flex items-center justify-between w-full text-left text-sm text-gray-600 hover:text-gray-900"
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
                <Loader2 className="h-8 w-8 animate-spin text-brand-500" />
              </div>
            ) : showReviews ? (
              <ReviewsList reviews={reviews?.data || []} />
            ) : (
              <div className="space-y-4">
                {filteredItems.length === 0 ? (
                  <div className="rounded-xl bg-white p-8 text-center shadow-sm">
                    <p className="text-gray-500">No items in this category</p>
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
          <Link
            to="/cart"
            className="btn-primary flex items-center justify-between gap-4 py-4 px-6 shadow-lg md:justify-center"
          >
            <div className="flex items-center gap-3">
              <ShoppingCart className="h-5 w-5" />
              <span>View Cart</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="rounded-full bg-white/20 px-2 py-0.5 text-sm">
                {cartItemCount} items
              </span>
              <span className="font-semibold">{fp(cart.getSubtotal())}</span>
            </div>
          </Link>
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
              alt={item.name}
              className="h-full w-full object-cover"
            />
            {item.isFeatured && (
              <div className="absolute top-1 left-1 rounded bg-brand-500 px-1.5 py-0.5 text-xs font-medium text-white">
                Popular
              </div>
            )}
          </div>
        )}

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-2">
            <div>
              <h3 className="font-semibold text-gray-900">{item.name}</h3>
              <p className="mt-1 text-sm text-gray-500 line-clamp-2">
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
                  className="rounded bg-green-50 px-2 py-0.5 text-xs text-green-700"
                >
                  {tag}
                </span>
              ))}
            </div>
          )}

          {/* Price and Actions */}
          <div className="mt-4 flex items-center justify-between">
            <div className="flex items-baseline gap-2">
              <span className="text-lg font-semibold text-gray-900">
                {fp(item.price)}
              </span>
              {item.comparePrice && (
                <span className="text-sm text-gray-400 line-through">
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
                    className="p-2 hover:bg-gray-100"
                  >
                    <Minus className="h-4 w-4" />
                  </button>
                  <span className="w-8 text-center">{quantity}</span>
                  <button
                    onClick={() => setQuantity(quantity + 1)}
                    className="p-2 hover:bg-gray-100"
                  >
                    <Plus className="h-4 w-4" />
                  </button>
                </div>

                <button onClick={handleAddToCart} className="btn-primary py-2 px-4">
                  {cartItem ? 'Add More' : 'Add'}
                </button>
              </div>
            ) : (
              <span className="text-sm text-gray-400">Unavailable</span>
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
      <div className="rounded-xl bg-white p-8 text-center shadow-sm">
        <p className="text-gray-500">No reviews yet</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h3 className="text-xl font-semibold text-gray-900">Customer Reviews</h3>
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
                          ? 'fill-yellow-400 text-yellow-400'
                          : 'text-gray-300'
                      }`}
                    />
                  ))}
                </div>
                <span className="text-sm text-gray-500">
                  {new Date(review.createdAt).toLocaleDateString()}
                </span>
              </div>
              {review.title && (
                <h4 className="mt-2 font-medium text-gray-900">{review.title}</h4>
              )}
              {review.comment && (
                <p className="mt-2 text-gray-600">{review.comment}</p>
              )}
              {review.chefResponse && (
                <div className="mt-3 rounded-lg bg-gray-50 p-3">
                  <p className="text-sm font-medium text-gray-700">Chef's Response:</p>
                  <p className="mt-1 text-sm text-gray-600">{review.chefResponse}</p>
                </div>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
