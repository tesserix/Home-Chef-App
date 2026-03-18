import { Link } from 'react-router-dom';
import { MapPin, Clock, Heart, Award } from 'lucide-react';
import { motion } from 'framer-motion';
import { cn } from '@/shared/utils/cn';
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
  const avatarUrl = avatar || CHEF_PLACEHOLDERS[parseInt(id) % CHEF_PLACEHOLDERS.length];
  const coverUrl = coverImage || FOOD_PLACEHOLDERS[parseInt(id) % FOOD_PLACEHOLDERS.length];

  return (
    <Card
      variant="default"
      padding="none"
      hover="lift"
      className={cn('overflow-hidden group', className)}
    >
      {/* Cover Image */}
      <div className="relative h-32 overflow-hidden">
        <motion.img
          src={coverUrl}
          alt={`${name}'s kitchen`}
          className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
          loading="lazy"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-black/20 to-transparent" />

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

        {/* Verified badge */}
        {isVerified && (
          <Badge variant="success" size="sm" className="absolute left-3 top-3">
            <Award className="h-3 w-3 mr-1" />
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
            className="border-4 border-white shadow-elevated"
          />
        </div>
      </div>

      {/* Info */}
      <div className="p-4 pt-12 text-center">
        <Link to={`/chefs/${id}`}>
          <h3 className="font-semibold text-gray-900 text-lg group-hover:text-brand-600 transition-colors">
            {name}
          </h3>
        </Link>

        {specialty && (
          <p className="mt-1 text-sm text-brand-600">{specialty}</p>
        )}

        {/* Cuisines */}
        {cuisines && cuisines.length > 0 && (
          <div className="mt-3 flex flex-wrap justify-center gap-1.5">
            {cuisines.slice(0, 3).map((cuisine, index) => (
              <Badge key={index} variant="default" size="sm">
                {cuisine}
              </Badge>
            ))}
          </div>
        )}

        {/* Rating */}
        {rating !== undefined && (
          <div className="mt-3 flex justify-center">
            <RatingBadge value={rating} showCount={!!reviewCount} count={reviewCount} />
          </div>
        )}

        {/* Meta */}
        <div className="mt-3 flex items-center justify-center gap-4 text-sm text-gray-500">
          {deliveryTime && (
            <div className="flex items-center gap-1">
              <Clock className="h-4 w-4" />
              <span>{deliveryTime}</span>
            </div>
          )}
          {distance && (
            <div className="flex items-center gap-1">
              <MapPin className="h-4 w-4" />
              <span>{distance}</span>
            </div>
          )}
        </div>

        {/* CTA */}
        <Button
          asChild
          variant="primary"
          fullWidth
          className="mt-4"
        >
          <Link to={`/chefs/${id}`}>View Menu</Link>
        </Button>
      </div>
    </Card>
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
  const avatarUrl = avatar || CHEF_PLACEHOLDERS[parseInt(id) % CHEF_PLACEHOLDERS.length];

  return (
    <Link to={`/chefs/${id}`}>
      <Card variant="outlined" padding="md" hover="lift" className="flex items-center gap-4">
        <div className="relative shrink-0">
          <Avatar src={avatarUrl} alt={name} size="lg" />
          {isVerified && (
            <div className="absolute -bottom-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-fresh-500 text-white ring-2 ring-white">
              <Award className="h-3 w-3" />
            </div>
          )}
        </div>
        <div className="flex-1 min-w-0">
          <h4 className="font-semibold text-gray-900 truncate">{name}</h4>
          {specialty && (
            <p className="text-sm text-brand-600 truncate">{specialty}</p>
          )}
          <div className="mt-1 flex items-center gap-3 text-sm text-gray-500">
            {rating !== undefined && (
              <RatingBadge value={rating} size="sm" />
            )}
            {deliveryTime && (
              <div className="flex items-center gap-1">
                <Clock className="h-3 w-3" />
                <span>{deliveryTime}</span>
              </div>
            )}
          </div>
        </div>
      </Card>
    </Link>
  );
}

