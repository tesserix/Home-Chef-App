import { Link } from 'react-router-dom';
import { Search, MapPin, ChefHat, Clock, Star } from 'lucide-react';
import { cn } from '@/shared/utils/cn';
import { HERO_IMAGES } from '@/shared/constants/images';

interface HeroSectionProps {
  variant?: 'home' | 'chefs' | 'catering' | 'feed';
  className?: string;
}

const heroContent = {
  home: {
    title: 'Homemade Food,',
    titleHighlight: 'Delivered Fresh',
    description:
      'Connect with talented home chefs in your area. Enjoy authentic, homemade meals prepared with love and delivered to your doorstep.',
    cta: 'Browse Local Chefs',
    ctaLink: '/chefs',
    image: HERO_IMAGES.home,
  },
  chefs: {
    title: 'Discover',
    titleHighlight: 'Local Chefs',
    description:
      'Explore a diverse community of passionate home chefs offering everything from family recipes to gourmet experiences.',
    cta: 'Start Exploring',
    ctaLink: '#chefs-list',
    image: HERO_IMAGES.chefs,
  },
  catering: {
    title: 'Catering for',
    titleHighlight: 'Every Occasion',
    description:
      'From intimate gatherings to large celebrations, our home chefs bring authentic flavors to your special events.',
    cta: 'Plan Your Event',
    ctaLink: '#catering-form',
    image: HERO_IMAGES.catering,
  },
  feed: {
    title: 'Food',
    titleHighlight: 'Inspiration',
    description:
      'See what your favorite chefs are cooking today. Get inspired by beautiful dishes and discover new flavors.',
    cta: 'Explore Feed',
    ctaLink: '#feed',
    image: HERO_IMAGES.feed,
  },
};

export function HeroSection({ variant = 'home', className }: HeroSectionProps) {
  const content = heroContent[variant];

  return (
    <section
      className={cn(
        'relative overflow-hidden bg-gradient-cream',
        className
      )}
    >
      {/* Background Image with Overlay */}
      <div className="absolute inset-0">
        <img
          src={content.image}
          alt=""
          className="h-full w-full object-cover"
        />
        <div className="absolute inset-0 bg-gradient-to-r from-white via-white/95 to-white/70" />
      </div>

      {/* Content */}
      <div className="container-app relative py-16 md:py-24 lg:py-32">
        <div className="max-w-2xl">
          <h1 className="hero-title">
            {content.title}
            <br />
            <span className="text-brand-600">{content.titleHighlight}</span>
          </h1>
          <p className="hero-description">
            {content.description}
          </p>

          {variant === 'home' && <SearchBar />}

          <div className="mt-8 flex flex-wrap items-center gap-4">
            <Link to={content.ctaLink} className="btn-primary">
              {content.cta}
            </Link>
            <Link to="/how-it-works" className="btn-outline">
              How It Works
            </Link>
          </div>

          {variant === 'home' && <Stats />}
        </div>
      </div>
    </section>
  );
}

function SearchBar() {
  return (
    <div className="mt-8 flex flex-col gap-3 sm:flex-row">
      <div className="flex flex-1 items-center gap-3 rounded-xl border border-gray-200 bg-white px-4 py-3 shadow-sm focus-within:border-brand-500 focus-within:ring-2 focus-within:ring-brand-500/20">
        <MapPin className="h-5 w-5 text-gray-400" />
        <input
          type="text"
          placeholder="Enter your delivery address"
          className="flex-1 bg-transparent text-gray-900 placeholder:text-gray-400 focus:outline-none"
        />
      </div>
      <button className="btn-primary flex items-center gap-2 whitespace-nowrap">
        <Search className="h-5 w-5" />
        Find Chefs
      </button>
    </div>
  );
}

function Stats() {
  const stats = [
    { icon: ChefHat, value: '500+', label: 'Home Chefs' },
    { icon: Star, value: '4.8', label: 'Avg Rating' },
    { icon: Clock, value: '30min', label: 'Avg Delivery' },
  ];

  return (
    <div className="mt-12 flex flex-wrap gap-8">
      {stats.map((stat, index) => {
        const Icon = stat.icon;
        return (
          <div key={index} className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-brand-100">
              <Icon className="h-6 w-6 text-brand-600" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900">{stat.value}</p>
              <p className="text-sm text-gray-500">{stat.label}</p>
            </div>
          </div>
        );
      })}
    </div>
  );
}

// Smaller hero for inner pages
export function PageHero({
  title,
  description,
  backgroundImage,
  className,
}: {
  title: string;
  description?: string;
  backgroundImage?: string;
  className?: string;
}) {
  return (
    <section
      className={cn(
        'relative overflow-hidden bg-gradient-brand py-12 md:py-16',
        className
      )}
    >
      {backgroundImage && (
        <div className="absolute inset-0">
          <img
            src={backgroundImage}
            alt=""
            className="h-full w-full object-cover opacity-20"
          />
        </div>
      )}
      <div className="container-app relative text-center text-white">
        <h1 className="text-3xl font-bold md:text-4xl">{title}</h1>
        {description && (
          <p className="mx-auto mt-3 max-w-2xl text-lg text-white/80">
            {description}
          </p>
        )}
      </div>
    </section>
  );
}
