import { Link } from 'react-router-dom';
import { MapPin, Clock, Heart, Award } from 'lucide-react';
import { cn } from '@/shared/utils/cn';
import { hashStringToIndex } from '@/shared/utils/hash';
import { CHEF_PLACEHOLDERS, FOOD_PLACEHOLDERS } from '@/shared/constants/images';
import { Card, Badge, Avatar, RatingBadge, Button } from '@/shared/components/ui';

interface ChefCardProps {
  id: string;
  name: string;
  avatar?: string;
  coverImage?: string;
  specialty?: string;
  cuisines?: string[];
  rating?: number;
  reviewCount?: number;
  deliveryTime?: string;
  distance?: string;
  isVerified?: boolean;
  isFavorite?: boolean;
  onFavorite?: () => void;
  className?: string;
}

const MAX_CUISINES_VISIBLE = 3;

function resolveCoverFallback(id: string): string {
  return FOOD_PLACEHOLDERS[hashStringToIndex(id, FOOD_PLACEHOLDERS.length)]!;
}

function resolveAvatarFallback(id: string): string {
  return CHEF_PLACEHOLDERS[hashStringToIndex(id, CHEF_PLACEHOLDERS.length)]!;
}

export function ChefCard({
  id,
  name,
  avatar,
  coverImage,
  specialty,
  cuisines,
  rating,
  reviewCount,
  deliveryTime,
  distance,
  isVerified,
  isFavorite = false,
  onFavorite,
  className,
}: ChefCardProps) {
  const coverUrl = coverImage || resolveCoverFallback(id);
  const avatarUrl = avatar || resolveAvatarFallback(id);
  const coverFallback = resolveCoverFallback(id);

  const visibleCuisines = cuisines?.slice(0, MAX_CUISINES_VISIBLE) ?? [];
  const overflowCount = (cuisines?.length ?? 0) - visibleCuisines.length;

  return (
    <article className={cn('card-hive group h-full overflow-hidden', className)}>
      {/* Cover Image — banner-fallback underlay so a missing photo still
          reads as a brand-tinted cell rather than flat black. */}
      <div className="banner-fallback relative h-32 overflow-hidden">
        <img
          src={coverUrl}
          alt=""
          loading="lazy"
          decoding="async"
          className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.04]"
          onError={(e) => {
            const img = e.currentTarget;
            if (img.src !== coverFallback) img.src = coverFallback;
            else img.style.display = 'none';
          }}
        />
        <div className="absolute inset-0 scrim-bottom" />

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
            className="absolute right-3 top-3 flex h-9 w-9 items-center justify-center rounded-full bg-bone shadow-2 transition-all hover:bg-bone hover:shadow-3 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2 active:scale-95"
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

        {/* Verified badge */}
        {isVerified && (
          <Badge variant="success" size="sm" className="absolute left-3 top-3">
            <Award aria-hidden="true" className="h-3 w-3 mr-1" />
            Verified
          </Badge>
        )}

        {/* Avatar - overlapping the cover */}
        <div className="absolute -bottom-8 left-4">
          <Avatar
            src={avatarUrl}
            alt={name}
            size="xl"
            ring="default"
            className="border-4 border-bone shadow-elevated"
          />
        </div>
      </div>

      {/* Info */}
      <div className="min-w-0 p-4 pt-12 text-center">
        <Link to={`/chefs/${id}`}>
          <h3
            title={name}
            className="truncate font-semibold text-ink text-lg group-hover:text-herb transition-colors"
          >
            {name}
          </h3>
        </Link>

        {specialty && (
          <p title={specialty} className="mt-1 truncate text-sm text-herb">
            {specialty}
          </p>
        )}

        {/* Cuisines */}
        {visibleCuisines.length > 0 && (
          <div className="mt-3 flex flex-wrap justify-center gap-1.5">
            {visibleCuisines.map((cuisine) => (
              <Badge key={cuisine} variant="default" size="sm">
                {cuisine}
              </Badge>
            ))}
            {overflowCount > 0 && (
              <Badge
                variant="default"
                size="sm"
                aria-label={`${overflowCount} more cuisines`}
              >
                +{overflowCount}
              </Badge>
            )}
          </div>
        )}

        {/* Rating */}
        {rating !== undefined && (
          <div className="mt-3 flex justify-center">
            <RatingBadge value={rating} showCount={!!reviewCount} count={reviewCount} />
          </div>
        )}

        {/* Meta */}
        <div className="mt-3 flex flex-wrap items-center justify-center gap-x-4 gap-y-1 text-sm text-ink-muted">
          {deliveryTime && (
            <div className="flex items-center gap-1 min-w-0">
              <Clock aria-hidden="true" className="h-4 w-4 shrink-0" />
              <span className="truncate">{deliveryTime}</span>
            </div>
          )}
          {distance && (
            <div className="flex items-center gap-1 min-w-0">
              <MapPin aria-hidden="true" className="h-4 w-4 shrink-0" />
              <span className="truncate">{distance}</span>
            </div>
          )}
        </div>

        {/* CTA */}
        <Button asChild variant="primary" fullWidth className="mt-4">
          <Link to={`/chefs/${id}`}>View Menu</Link>
        </Button>
      </div>
    </article>
  );
}

// Compact horizontal version
export function ChefCardHorizontal({
  id,
  name,
  avatar,
  specialty,
  rating,
  deliveryTime,
  isVerified,
}: Pick<
  ChefCardProps,
  'id' | 'name' | 'avatar' | 'specialty' | 'rating' | 'deliveryTime' | 'isVerified'
>) {
  const avatarUrl = avatar || resolveAvatarFallback(id);

  return (
    <Link to={`/chefs/${id}`}>
      <Card variant="outlined" padding="md" hover="lift" className="flex items-center gap-4">
        <div className="relative shrink-0">
          <Avatar src={avatarUrl} alt={name} size="lg" />
          {isVerified && (
            <div
              aria-label="Verified chef"
              title="Verified chef"
              className="absolute -bottom-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-fresh-500 text-paper ring-2 ring-white"
            >
              <Award aria-hidden="true" className="h-3 w-3" />
            </div>
          )}
        </div>
        <div className="flex-1 min-w-0">
          <h4 title={name} className="font-semibold text-ink truncate">
            {name}
          </h4>
          {specialty && (
            <p title={specialty} className="text-sm text-herb truncate">
              {specialty}
            </p>
          )}
          <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-ink-muted">
            {rating !== undefined && <RatingBadge value={rating} size="sm" />}
            {deliveryTime && (
              <div className="flex items-center gap-1 min-w-0">
                <Clock aria-hidden="true" className="h-3 w-3 shrink-0" />
                <span className="truncate">{deliveryTime}</span>
              </div>
            )}
          </div>
        </div>
      </Card>
    </Link>
  );
}
