import { Link } from 'react-router-dom';
import { Clock, Plus, Heart, Flame } from 'lucide-react';
import { cn } from '@/shared/utils/cn';
import { hashStringToIndex } from '@/shared/utils/hash';
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

function resolveFoodFallback(id: string): string {
  return FOOD_PLACEHOLDERS[hashStringToIndex(id, FOOD_PLACEHOLDERS.length)]!;
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
  const fallbackUrl = resolveFoodFallback(id);
  const imageUrl = image || fallbackUrl;
  const fp = useFormatPrice();
  const showStrikethrough = originalPrice !== undefined && originalPrice > price;

  return (
    <Card
      variant="default"
      padding="none"
      hover="lift"
      className={cn('overflow-hidden group', className)}
    >
      {/* Image */}
      <Link to={`/menu/${id}`} className="relative block aspect-square overflow-hidden">
        <img
          src={imageUrl}
          alt={name}
          loading="lazy"
          decoding="async"
          className="h-full w-full object-cover transition-transform duration-500 group-hover:opacity-95"
          onError={(e) => {
            const img = e.currentTarget;
            if (img.src !== fallbackUrl) img.src = fallbackUrl;
          }}
        />
        <div className="absolute inset-0 scrim-bottom opacity-0 group-hover:opacity-100 transition-opacity duration-300" />

        {/* Badges */}
        <div className="absolute left-3 top-3 flex flex-col gap-2">
          {discount !== undefined && discount > 0 && (
            <Badge variant="error" size="sm" aria-label={`${discount} percent off`}>
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
          <button
            type="button"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onFavorite();
            }}
            aria-label={isFavorite ? `Remove ${name} from favorites` : `Save ${name} to favorites`}
            aria-pressed={isFavorite}
            className="absolute right-3 top-3 flex h-9 w-9 items-center justify-center rounded-full bg-bone/90 shadow-soft-md backdrop-blur-sm transition-all hover:bg-bone hover:shadow-elevated focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2 active:scale-95"
          >
            <Heart
              aria-hidden="true"
              className={cn(
                'h-4 w-4 transition-colors',
                isFavorite ? 'fill-spice-500 text-herb' : 'text-ink-soft'
              )}
            />
          </button>
        )}
      </Link>

      {/* Content */}
      <div className="p-4">
        <Link to={`/menu/${id}`}>
          <h3
            title={name}
            className="font-semibold text-ink line-clamp-1 group-hover:text-herb transition-colors"
          >
            {name}
          </h3>
        </Link>

        {description && (
          <p title={description} className="mt-1 text-sm text-ink-muted line-clamp-2">
            {description}
          </p>
        )}

        {/* Chef link */}
        {chefName && chefId && (
          <Link
            to={`/chefs/${chefId}`}
            title={chefName}
            className="mt-2 inline-block max-w-full truncate text-sm text-herb hover:underline align-bottom"
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
            <div className="flex items-center gap-1 text-sm text-ink-muted min-w-0">
              <Clock aria-hidden="true" className="h-4 w-4 shrink-0" />
              <span className="truncate">{prepTime}</span>
            </div>
          )}
          {spicyLevel && (
            <div
              role="img"
              aria-label={`Spicy level ${spicyLevel} of 3`}
              className="flex items-center gap-0.5"
            >
              {[1, 2, 3].map((level) => (
                <Flame
                  key={level}
                  aria-hidden="true"
                  className={cn(
                    'h-4 w-4',
                    level <= spicyLevel ? 'text-herb' : 'text-mist-strong'
                  )}
                />
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="mt-4 flex items-center justify-between gap-3">
          <div className="flex items-baseline gap-2 min-w-0">
            <span className="text-xl font-semibold text-herb tabular-nums truncate">
              {fp(price)}
            </span>
            {showStrikethrough && (
              <span
                aria-label={`Original price ${fp(originalPrice!)}`}
                className="text-sm text-ink-muted line-through tabular-nums truncate"
              >
                {fp(originalPrice!)}
              </span>
            )}
          </div>

          {onAddToCart && (
            <Button
              variant="primary"
              size="icon"
              onClick={onAddToCart}
              aria-label={`Add ${name} to cart`}
              className="rounded-xl shrink-0"
            >
              <Plus aria-hidden="true" className="h-5 w-5" />
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
  const fallbackUrl = resolveFoodFallback(id);
  const imageUrl = image || fallbackUrl;
  const fp = useFormatPrice();

  return (
    <Card variant="outlined" padding="sm" hover="lift" className="flex items-center gap-4">
      <img
        src={imageUrl}
        alt={name}
        loading="lazy"
        decoding="async"
        className="h-20 w-20 rounded-xl object-cover shrink-0"
        onError={(e) => {
          const img = e.currentTarget;
          if (img.src !== fallbackUrl) img.src = fallbackUrl;
        }}
      />
      <div className="flex-1 min-w-0">
        <h4 title={name} className="font-medium text-ink truncate">
          {name}
        </h4>
        {chefName && (
          <p title={chefName} className="text-sm text-ink-muted truncate">
            by {chefName}
          </p>
        )}
        <p className="mt-1 font-semibold text-herb tabular-nums">{fp(price)}</p>
      </div>
      {onAddToCart && (
        <Button
          variant="secondary"
          size="icon"
          onClick={onAddToCart}
          aria-label={`Add ${name} to cart`}
          className="shrink-0"
        >
          <Plus aria-hidden="true" className="h-5 w-5" />
        </Button>
      )}
    </Card>
  );
}
