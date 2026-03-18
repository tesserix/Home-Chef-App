import { useState } from 'react';
import { cn } from '@/shared/utils/cn';
import {
  FOOD_PLACEHOLDERS,
  CHEF_PLACEHOLDERS,
  AVATAR_PLACEHOLDERS,
  FOOD_CATEGORIES,
  getRandomFoodImage,
  getRandomChefImage,
  getRandomAvatar,
} from '@/shared/constants/images';

interface PlaceholderImageProps {
  type: 'food' | 'chef' | 'avatar' | 'category';
  category?: keyof typeof FOOD_CATEGORIES;
  index?: number;
  alt: string;
  className?: string;
  aspectRatio?: 'square' | 'video' | 'portrait' | 'auto';
}

const aspectRatioClasses = {
  square: 'aspect-square',
  video: 'aspect-video',
  portrait: 'aspect-[3/4]',
  auto: '',
};

export function PlaceholderImage({
  type,
  category,
  index,
  alt,
  className,
  aspectRatio = 'auto',
}: PlaceholderImageProps) {
  const [isLoading, setIsLoading] = useState(true);
  const [hasError, setHasError] = useState(false);

  const getImageUrl = () => {
    if (type === 'category' && category) {
      return FOOD_CATEGORIES[category] || getRandomFoodImage();
    }

    if (index !== undefined) {
      switch (type) {
        case 'food':
          return FOOD_PLACEHOLDERS[index % FOOD_PLACEHOLDERS.length];
        case 'chef':
          return CHEF_PLACEHOLDERS[index % CHEF_PLACEHOLDERS.length];
        case 'avatar':
          return AVATAR_PLACEHOLDERS[index % AVATAR_PLACEHOLDERS.length];
        default:
          return getRandomFoodImage();
      }
    }

    switch (type) {
      case 'food':
        return getRandomFoodImage();
      case 'chef':
        return getRandomChefImage();
      case 'avatar':
        return getRandomAvatar();
      default:
        return getRandomFoodImage();
    }
  };

  const imageUrl = getImageUrl();

  if (hasError) {
    return (
      <div
        className={cn(
          'flex items-center justify-center bg-muted',
          aspectRatioClasses[aspectRatio],
          className
        )}
      >
        <span className="text-4xl">🍽️</span>
      </div>
    );
  }

  return (
    <div className={cn('relative overflow-hidden', aspectRatioClasses[aspectRatio], className)}>
      {isLoading && (
        <div className="absolute inset-0 skeleton-shimmer" />
      )}
      <img
        src={imageUrl}
        alt={alt}
        className={cn(
          'h-full w-full object-cover transition-opacity duration-300',
          isLoading ? 'opacity-0' : 'opacity-100'
        )}
        onLoad={() => setIsLoading(false)}
        onError={() => {
          setIsLoading(false);
          setHasError(true);
        }}
        loading="lazy"
      />
    </div>
  );
}

export function FoodImage({
  index,
  alt,
  className,
}: {
  index?: number;
  alt: string;
  className?: string;
}) {
  return <PlaceholderImage type="food" index={index} alt={alt} className={className} />;
}

export function ChefAvatar({
  index,
  alt,
  className,
}: {
  index?: number;
  alt: string;
  className?: string;
}) {
  return (
    <PlaceholderImage
      type="chef"
      index={index}
      alt={alt}
      className={cn('rounded-full', className)}
      aspectRatio="square"
    />
  );
}

export function UserAvatar({
  index,
  alt,
  className,
}: {
  index?: number;
  alt: string;
  className?: string;
}) {
  return (
    <PlaceholderImage
      type="avatar"
      index={index}
      alt={alt}
      className={cn('rounded-full', className)}
      aspectRatio="square"
    />
  );
}

export function CategoryImage({
  category,
  alt,
  className,
}: {
  category: keyof typeof FOOD_CATEGORIES;
  alt: string;
  className?: string;
}) {
  return (
    <PlaceholderImage
      type="category"
      category={category}
      alt={alt}
      className={className}
    />
  );
}
