import { Link } from 'react-router-dom';
import { Clock, Plus, Heart, Flame } from 'lucide-react';
import { motion } from 'framer-motion';
import { cn } from '@/shared/utils/cn';
import { useFormatPrice } from '@/shared/utils/format-price';
import { FOOD_PLACEHOLDERS } from '@/shared/constants/images';
import { Card, Badge, RatingBadge, Button } from '@/shared/components/ui';

interface FoodCardProps {
  id: string;
  name: string;
  description?: string;
  price: number;
  originalPrice?: number;
  image?: string;
  rating?: number;
  reviewCount?: number;
  prepTime?: string;
  chefName?: string;
  chefId?: string;
  isVegetarian?: boolean;
  spicyLevel?: 1 | 2 | 3;
  discount?: number;
  onAddToCart?: () => void;
  onFavorite?: () => void;
  isFavorite?: boolean;
  className?: string;
}

export function FoodCard({
  id,
  name,
  description,
  price,
  originalPrice,
  image,
  rating,
  reviewCount,
  prepTime,
  chefName,
  chefId,
  isVegetarian,
  spicyLevel,
  discount,
  onAddToCart,
  onFavorite,
  isFavorite = false,
  className,
}: FoodCardProps) {
  const imageUrl = image || FOOD_PLACEHOLDERS[parseInt(id) % FOOD_PLACEHOLDERS.length];
  const fp = useFormatPrice();

  return (
    <Card
      variant="default"
      padding="none"
      hover="lift"
      className={cn('overflow-hidden group', className)}
    >
      {/* Image */}
      <Link to={`/menu/${id}`} className="relative block aspect-square overflow-hidden">
        <motion.img
          src={imageUrl}
          alt={name}
          loading="lazy"
          className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-black/20 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300" />

        {/* Badges */}
        <div className="absolute left-3 top-3 flex flex-col gap-2">
          {discount && (
            <Badge variant="error" size="sm">
              -{discount}%
            </Badge>
          )}
          {isVegetarian && (
            <Badge variant="success" size="sm">
              Veg
            </Badge>
          )}
        </div>

        {/* Favorite button */}
        {onFavorite && (
          <motion.button
            onClick={(e) => {
              e.preventDefault();
              onFavorite();
            }}
            whileTap={{ scale: 0.9 }}
            className="absolute right-3 top-3 flex h-9 w-9 items-center justify-center rounded-full bg-white/90 shadow-soft-md backdrop-blur-sm transition-all hover:bg-white hover:shadow-elevated"
          >
            <Heart
              className={cn(
                'h-4 w-4 transition-colors',
                isFavorite ? 'fill-spice-500 text-spice-500' : 'text-gray-600'
              )}
            />
          </motion.button>
        )}
      </Link>

      {/* Content */}
      <div className="p-4">
        <Link to={`/menu/${id}`}>
          <h3 className="font-semibold text-gray-900 line-clamp-1 group-hover:text-brand-600 transition-colors">
            {name}
          </h3>
        </Link>

        {description && (
          <p className="mt-1 text-sm text-gray-500 line-clamp-2">{description}</p>
        )}

        {/* Chef link */}
        {chefName && chefId && (
          <Link
            to={`/chefs/${chefId}`}
            className="mt-2 inline-block text-sm text-brand-600 hover:underline"
          >
            by {chefName}
          </Link>
        )}

        {/* Meta info */}
        <div className="mt-3 flex flex-wrap items-center gap-3">
          {rating !== undefined && (
            <RatingBadge value={rating} showCount={!!reviewCount} count={reviewCount} size="sm" />
          )}
          {prepTime && (
            <div className="flex items-center gap-1 text-sm text-gray-500">
              <Clock className="h-4 w-4" />
              <span>{prepTime}</span>
            </div>
          )}
          {spicyLevel && (
            <div className="flex items-center gap-0.5">
              {[1, 2, 3].map((level) => (
                <Flame
                  key={level}
                  className={cn(
                    'h-4 w-4',
                    level <= spicyLevel ? 'text-spice-500' : 'text-gray-200'
                  )}
                />
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="mt-4 flex items-center justify-between">
          <div className="flex items-baseline gap-2">
            <span className="text-xl font-bold text-brand-600">
              {fp(price)}
            </span>
            {originalPrice && originalPrice > price && (
              <span className="text-sm text-gray-400 line-through">
                {fp(originalPrice)}
              </span>
            )}
          </div>

          {onAddToCart && (
            <Button
              variant="primary"
              size="icon"
              onClick={onAddToCart}
              className="rounded-xl"
            >
              <Plus className="h-5 w-5" />
            </Button>
          )}
        </div>
      </div>
    </Card>
  );
}

// Compact version for lists
export function FoodCardCompact({
  id,
  name,
  price,
  image,
  chefName,
  onAddToCart,
}: Pick<FoodCardProps, 'id' | 'name' | 'price' | 'image' | 'chefName' | 'onAddToCart'>) {
  const imageUrl = image || FOOD_PLACEHOLDERS[parseInt(id) % FOOD_PLACEHOLDERS.length];
  const fp = useFormatPrice();

  return (
    <Card variant="outlined" padding="sm" hover="lift" className="flex items-center gap-4">
      <img
        src={imageUrl}
        alt={name}
        className="h-20 w-20 rounded-xl object-cover"
        loading="lazy"
      />
      <div className="flex-1 min-w-0">
        <h4 className="font-medium text-gray-900 truncate">{name}</h4>
        {chefName && (
          <p className="text-sm text-gray-500">by {chefName}</p>
        )}
        <p className="mt-1 font-bold text-brand-600">{fp(price)}</p>
      </div>
      {onAddToCart && (
        <Button
          variant="secondary"
          size="icon"
          onClick={onAddToCart}
          className="shrink-0"
        >
          <Plus className="h-5 w-5" />
        </Button>
      )}
    </Card>
  );
}
