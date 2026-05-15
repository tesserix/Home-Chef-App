import { Link, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import {
  Search,
  MapPin,
  ChefHat,
  Clock,
  ArrowRight,
  Utensils,
  Heart,
  Truck,
  Shield,
  Star,
  Sparkles,
} from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';
import { cn } from '@/shared/utils/cn';
import { apiClient } from '@/shared/services/api-client';
import { useFavoritesStore } from '@/app/store/favorites-store';
import { useAuth } from '@/app/providers/AuthProvider';
import type { Chef, PaginatedResponse } from '@/shared/types';
import { Button, Card, Input, Badge, Avatar, RatingBadge } from '@/shared/components/ui';

// Animation variants
const fadeInUp = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0 },
};

const staggerContainer = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1,
    },
  },
};

const scaleIn = {
  hidden: { opacity: 0, scale: 0.9 },
  visible: { opacity: 1, scale: 1 },
};

export default function HomePage() {
  const [searchQuery, setSearchQuery] = useState('');

  const { data: featuredChefs } = useQuery({
    queryKey: ['chefs', 'featured'],
    queryFn: () =>
      apiClient.get<PaginatedResponse<Chef>>('/chefs', {
        sort: 'rating',
        limit: 6,
        isOpen: true,
      }),
  });

  const cuisines = [
    { name: 'South Indian', image: 'https://images.unsplash.com/photo-1585937421612-70a008356fbe?w=300&h=200&fit=crop' },
    { name: 'Italian', image: 'https://images.unsplash.com/photo-1498579150354-977475b7ea0b?w=300&h=200&fit=crop' },
    { name: 'Japanese', image: 'https://images.unsplash.com/photo-1579027989536-b7b1f875659b?w=300&h=200&fit=crop' },
    { name: 'North Indian', image: 'https://images.unsplash.com/photo-1505253758473-96b7015fcd40?w=300&h=200&fit=crop' },
    { name: 'Mexican', image: 'https://images.unsplash.com/photo-1565299585323-38d6b0865b47?w=300&h=200&fit=crop' },
    { name: 'Thai', image: 'https://images.unsplash.com/photo-1559314809-0d155014e29e?w=300&h=200&fit=crop' },
  ];

  return (
    <div className="min-h-screen bg-paper">
      {/* Hero Section — photo-forward, the food carries the brand.
          Per the design system: "Food and faces carry the brand. UI chrome
          shrinks." The hero uses an Indian home-cooked spread as the
          backdrop, a three-stop scrim for text legibility, and a subtle
          herb-tinted ambience to tie the photo to the brand accent. */}
      <section className="relative isolate overflow-hidden">
        {/* Photo backdrop */}
        <div aria-hidden="true" className="absolute inset-0 -z-10">
          <img
            src="https://images.unsplash.com/photo-1600565193348-f74bd3c7ccdf?w=2200&h=1400&fit=crop&q=85"
            alt=""
            width={2200}
            height={1400}
            fetchPriority="high"
            decoding="async"
            className="h-full w-full object-cover"
          />
          {/* Theme-invariant 3-stop scrim — keeps text readable on any photo
              in any theme. Mirrors the .scrim-bottom utility but covers the
              whole hero so the headline anchors strongly. */}
          <div
            className="absolute inset-0"
            style={{
              background:
                'linear-gradient(180deg, oklch(0.10 0.012 60 / 0.55) 0%, oklch(0.10 0.012 60 / 0.55) 35%, oklch(0.10 0.012 60 / 0.78) 100%)',
            }}
          />
          {/* Herb-tinted ambience — single-accent brand tie-in. */}
          <div className="absolute inset-0 hero-ambience opacity-90" />
        </div>

        <div className="container-app relative py-24 lg:py-32">
          <motion.div
            initial="hidden"
            animate="visible"
            variants={staggerContainer}
            className="mx-auto max-w-3xl text-center"
          >
            <motion.div variants={fadeInUp}>
              <span className="chip-on-photo-accent mb-6 inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-medium">
                <Sparkles aria-hidden="true" className="h-4 w-4" />
                500+ Home Chefs Near You
              </span>
            </motion.div>

            <motion.h1
              variants={fadeInUp}
              className="text-on-photo mt-6 font-display text-display-lg md:text-display-xl lg:text-display-2xl"
            >
              Homemade Food,{' '}
              <span className="text-on-photo-accent">Delivered Fresh</span>
            </motion.h1>

            <motion.p
              variants={fadeInUp}
              className="text-on-photo-soft mx-auto mt-6 max-w-2xl text-lg"
            >
              Discover talented home chefs in your neighborhood and enjoy authentic,
              homemade meals delivered right to your doorstep.
            </motion.p>

            {/* Search Bar — sits on top of the photo, uses theme tokens so
                inputs keep their normal contrast in either mode. */}
            <motion.div variants={fadeInUp} className="mt-10">
              <Card variant="elevated" padding="sm" className="shadow-3">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center p-2">
                  <div className="relative flex-1">
                    <Input
                      variant="ghost"
                      inputSize="lg"
                      placeholder="Enter your delivery address..."
                      leftIcon={<MapPin className="h-5 w-5"  aria-hidden="true" />}
                      className="border-0"
                    />
                  </div>
                  <div className="hidden sm:block w-px h-10 bg-mist" />
                  <div className="relative flex-1">
                    <Input
                      type="search"
                      aria-label="Search dishes or chefs"
                      variant="ghost"
                      inputSize="lg"
                      placeholder="Search dishes or chefs..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      leftIcon={<Search aria-hidden="true" className="h-5 w-5" />}
                      className="border-0"
                    />
                  </div>
                  <Button
                    asChild
                    variant="primary"
                    size="lg"
                    className="px-8"
                  >
                    <Link to={`/chefs${searchQuery ? `?search=${searchQuery}` : ''}`}>
                      Find Food
                    </Link>
                  </Button>
                </div>
              </Card>
            </motion.div>

            {/* Trust Badges — on-photo chips so they read in any theme. */}
            <motion.div
              variants={fadeInUp}
              className="mt-12 flex flex-wrap items-center justify-center gap-3 sm:gap-4"
            >
              {[
                { icon: ChefHat, label: '500+ Home Chefs' },
                { icon: Star, label: '4.8 Average Rating' },
                { icon: Clock, label: '30-45 min Delivery' },
              ].map(({ icon: Icon, label }) => (
                <div key={label} className="chip-on-photo flex items-center gap-2 rounded-full px-4 py-2">
                  <Icon aria-hidden="true" className="h-4 w-4" />
                  <span className="text-sm font-medium">{label}</span>
                </div>
              ))}
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* How It Works */}
      <section className="py-20 bg-bone">
        <div className="container-app">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, margin: '-100px' }}
            variants={staggerContainer}
            className="text-center"
          >
            <motion.div variants={fadeInUp}>
              <Badge variant="brand" className="mb-4">How It Works</Badge>
              <h2 className="font-display text-display-md text-ink">
                Get Delicious Food in 3 Steps
              </h2>
              <p className="mt-3 text-ink-soft max-w-xl mx-auto">
                From discovery to delivery, we make it simple to enjoy homemade food
              </p>
            </motion.div>

            <motion.div
              variants={staggerContainer}
              className="mt-16 grid gap-8 md:grid-cols-3"
            >
              {[
                {
                  icon: Search,
                  title: 'Discover',
                  description: 'Browse home chefs near you and explore their authentic menus',
                },
                {
                  icon: Utensils,
                  title: 'Order',
                  description: 'Select your favorite dishes and place your order securely',
                },
                {
                  icon: Truck,
                  title: 'Enjoy',
                  description: 'Get fresh homemade food delivered to your doorstep',
                },
              ].map((step, index) => (
                <motion.div key={step.title} variants={scaleIn}>
                  <Card variant="ghost" padding="lg" className="text-center relative">
                    <div className="absolute -top-3 left-1/2 -translate-x-1/2">
                      <span className="inline-flex h-8 w-8 items-center justify-center rounded-full bg-herb text-sm font-medium text-paper">
                        {index + 1}
                      </span>
                    </div>
                    <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-3xl bg-herb shadow-2">
                      <step.icon aria-hidden="true" className="h-10 w-10 text-paper" />
                    </div>
                    <h3 className="mt-6 text-xl font-semibold text-ink">{step.title}</h3>
                    <p className="mt-3 text-ink-soft">{step.description}</p>
                  </Card>
                </motion.div>
              ))}
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Cuisines */}
      <section className="py-20 bg-paper">
        <div className="container-app">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, margin: '-100px' }}
            variants={staggerContainer}
          >
            <motion.div variants={fadeInUp} className="flex items-center justify-between">
              <div>
                <Badge variant="brand" className="mb-3">Cuisines</Badge>
                <h2 className="font-display text-display-md text-ink">Explore Flavors</h2>
                <p className="mt-2 text-ink-soft">Discover authentic dishes from around the world</p>
              </div>
              <Button asChild variant="ghost" className="hidden sm:flex">
                <Link to="/chefs">
                  View All <ArrowRight className="ml-2 h-4 w-4"  aria-hidden="true" />
                </Link>
              </Button>
            </motion.div>

            <motion.div
              variants={staggerContainer}
              className="mt-10 grid gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-6"
            >
              {cuisines.map((cuisine) => (
                <motion.div key={cuisine.name} variants={scaleIn}>
                  <Link
                    to={`/chefs?cuisine=${cuisine.name}`}
                    className="group relative block overflow-hidden rounded-2xl shadow-1 hover:shadow-2 transition-all duration-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
                  >
                    <div className="aspect-[4/3]">
                      <img
                        src={cuisine.image}
                        alt={cuisine.name}
                        width={300}
                        height={200}
                        loading="lazy"
                        decoding="async"
                        className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.04]"
                      />
                    </div>
                    <div aria-hidden="true" className="absolute inset-0 scrim-bottom" />
                    <div className="absolute bottom-0 left-0 right-0 p-4">
                      <h3 className="text-on-photo font-display text-lg font-semibold tracking-tight">{cuisine.name}</h3>
                    </div>
                  </Link>
                </motion.div>
              ))}
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Featured Chefs */}
      <section className="py-20 bg-bone">
        <div className="container-app">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, margin: '-100px' }}
            variants={staggerContainer}
          >
            <motion.div variants={fadeInUp} className="flex items-center justify-between">
              <div>
                <Badge variant="premium" className="mb-3">
                  <Star aria-hidden="true" className="h-3 w-3 mr-1 fill-amber text-amber" />
                  Top Rated
                </Badge>
                <h2 className="font-display text-display-md text-ink">Featured Chefs</h2>
                <p className="mt-2 text-ink-soft">Our community's favorite home chefs</p>
              </div>
              <Button asChild variant="ghost" className="hidden sm:flex">
                <Link to="/chefs">
                  View All Chefs <ArrowRight className="ml-2 h-4 w-4"  aria-hidden="true" />
                </Link>
              </Button>
            </motion.div>

            <motion.div
              variants={staggerContainer}
              className="mt-10 grid gap-6 sm:grid-cols-2 lg:grid-cols-3"
            >
              {(featuredChefs?.data ?? []).map((chef) => (
                <motion.div key={chef.id} variants={scaleIn}>
                  <FeaturedChefCard chef={chef} />
                </motion.div>
              ))}
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Catering CTA */}
      <section className="py-16">
        <div className="container-app">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={fadeInUp}
          >
            <Card
              variant="ghost"
              padding="none"
              className="overflow-hidden bg-herb"
            >
              <div className="flex flex-col items-center gap-8 p-8 text-center lg:flex-row lg:p-12 lg:text-left">
                <div className="flex-1">
                  <Badge variant="default" className="bg-bone/20 text-paper border-0 mb-4">
                    Catering Services
                  </Badge>
                  <h2 className="font-display text-3xl font-semibold tabular-nums text-paper">
                    Planning an Event?
                  </h2>
                  <p className="mt-3 text-lg text-herb-tint max-w-xl">
                    Get catering quotes from multiple home chefs. Perfect for parties,
                    corporate events, and special occasions.
                  </p>
                </div>
                <Button
                  asChild
                  variant="secondary"
                  size="xl"
                  className="bg-bone text-herb hover:bg-herb-tint"
                >
                  <Link to="/catering">Request Catering Quote</Link>
                </Button>
              </div>
            </Card>
          </motion.div>
        </div>
      </section>

      {/* Why Choose Us */}
      <section className="py-20 bg-paper">
        <div className="container-app">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, margin: '-100px' }}
            variants={staggerContainer}
          >
            <motion.div variants={fadeInUp} className="text-center">
              <Badge variant="brand" className="mb-4">Why Choose Us</Badge>
              <h2 className="font-display text-display-md text-ink">
                The Fe3dr Difference
              </h2>
              <p className="mt-3 text-ink-soft max-w-xl mx-auto">
                Join thousands of happy customers enjoying homemade food
              </p>
            </motion.div>

            <motion.div
              variants={staggerContainer}
              className="mt-16 grid gap-6 md:grid-cols-2 lg:grid-cols-4"
            >
              {([
                {
                  // Honest accuracy over marketing claim — we can only assert
                  // chefs upload FSSAI licences before publishing, not that every
                  // chef on the platform has been independently verified.
                  icon: ChefHat,
                  title: 'Licensed home chefs',
                  description:
                    'Home chefs upload their FSSAI food-safety licence before publishing menu items, in line with Indian law.',
                  tone: 'herb',
                },
                {
                  icon: Heart,
                  title: 'Made with Love',
                  description: 'Every meal is prepared fresh with authentic family recipes',
                  tone: 'paprika',
                },
                {
                  icon: Shield,
                  title: 'Secure Payments',
                  description: 'Safe and secure payment processing for every order',
                  tone: 'herb',
                },
                {
                  icon: Truck,
                  title: 'Fast Delivery',
                  description: 'Reliable delivery to your doorstep within 30-45 minutes',
                  tone: 'amber',
                },
              ] as const).map((feature) => {
                // Static class pairs — Tailwind JIT cannot scan template literals,
                // so every combination must appear verbatim somewhere in the source.
                const tone = {
                  herb: 'bg-herb-tint text-herb',
                  paprika: 'bg-paprika-tint text-paprika',
                  amber: 'bg-amber-tint text-amber',
                }[feature.tone];
                return (
                  <motion.div key={feature.title} variants={scaleIn}>
                    <Card variant="default" padding="lg" hover="lift" className="text-center h-full">
                      <div className={cn('mx-auto flex h-16 w-16 items-center justify-center rounded-2xl', tone)}>
                        <feature.icon aria-hidden="true" className="h-8 w-8" />
                      </div>
                      <h3 className="mt-5 text-lg font-semibold text-ink">{feature.title}</h3>
                      <p className="mt-2 text-sm text-ink-soft">{feature.description}</p>
                    </Card>
                  </motion.div>
                );
              })}
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Become a Chef CTA */}
      <section className="py-20 bg-bone">
        <div className="container-app">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={fadeInUp}
          >
            <Card variant="ghost" padding="none" className="overflow-hidden bg-ink rounded-3xl">
              <div className="flex flex-col md:flex-row">
                <div className="flex-1 p-8 md:p-12">
                  <Badge variant="brand" className="mb-4">Join Our Community</Badge>
                  <h2 className="font-display text-3xl font-semibold tabular-nums text-paper">
                    Love Cooking? Share Your Talent
                  </h2>
                  <p className="mt-4 text-ink-muted max-w-lg">
                    Turn your passion into income. Join our community of home chefs
                    and start earning by sharing your delicious homemade food.
                  </p>
                  <div className="mt-8 flex flex-wrap gap-4">
                    <Button asChild variant="primary" size="lg">
                      <Link to="/become-chef">Become a Chef</Link>
                    </Button>
                    <Button asChild variant="outline" size="lg" className="border-ink-soft text-paper hover:bg-ink">
                      <Link to="/chef-resources">Learn More</Link>
                    </Button>
                  </div>
                </div>
                <div className="hidden md:block md:w-2/5 relative aspect-[6/5]">
                  <img
                    src="https://images.unsplash.com/photo-1556910103-1c02745aae4d?w=600&h=500&fit=crop"
                    alt="Home chef cooking"
                    width={600}
                    height={500}
                    fetchPriority="high"
                    decoding="async"
                    className="absolute inset-0 h-full w-full object-cover"
                  />
                  <div aria-hidden="true" className="absolute inset-0 scrim-bottom" />
                </div>
              </div>
            </Card>
          </motion.div>
        </div>
      </section>
    </div>
  );
}

function chefInitials(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return '?';
  if (words.length === 1) return (words[0]!.slice(0, 2)).toUpperCase();
  return ((words[0]![0] ?? '') + (words[1]![0] ?? '')).toUpperCase();
}

function FeaturedChefCard({ chef }: { chef: Chef }) {
  const { isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const { isFavorite, toggle } = useFavoritesStore();
  const favorited = isFavorite(chef.id);
  const bannerSrc = chef.bannerImage || chef.profileImage;
  const initials = chefInitials(chef.businessName);

  const handleFavorite = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    if (!isAuthenticated) {
      toast.error('Please log in to save favorites');
      navigate('/login');
      return;
    }

    const result = await toggle(chef.id);
    if (result === 'max_limit') {
      toast.error('You can save up to 7 favorite chefs. Remove one first.');
    } else if (result === 'unauthorized') {
      toast.error('Please log in to save favorites');
      navigate('/login');
    } else if (result === 'error') {
      toast.error('Something went wrong. Please try again.');
    }
  };

  return (
    <Link
      to={`/chefs/${chef.id}`}
      className="block rounded-2xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
    >
      <article className="card-hive group h-full overflow-hidden">
        {/* Banner — uses .banner-fallback as the underlay so a missing image
            still reads as a brand-tinted cell, never flat black. */}
        <div className="banner-fallback relative h-40 overflow-hidden">
          {bannerSrc ? (
            <img
              src={bannerSrc}
              alt=""
              loading="lazy"
              decoding="async"
              className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.04]"
              onError={(e) => {
                e.currentTarget.style.display = 'none';
              }}
            />
          ) : (
            <span
              aria-hidden="true"
              className="absolute inset-0 flex items-center justify-center font-display text-5xl font-semibold text-on-photo-accent"
            >
              {initials}
            </span>
          )}
          <div aria-hidden="true" className="absolute inset-0 scrim-bottom" />

          {chef.verified && (
            <span className="chip-on-photo-accent absolute top-3 left-3 inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium">
              Verified
            </span>
          )}

          {/* Favorite button */}
          <button
            type="button"
            onClick={handleFavorite}
            aria-label={favorited ? `Remove ${chef.businessName} from favorites` : `Save ${chef.businessName} to favorites`}
            aria-pressed={favorited}
            className="chip-on-photo absolute top-3 right-3 flex h-9 w-9 items-center justify-center rounded-full transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
          >
            <Heart
              aria-hidden="true"
              className={cn(
                'h-4 w-4 transition-colors',
                favorited ? 'fill-paprika text-paprika' : 'text-on-photo'
              )}
            />
          </button>

          <div className="absolute -bottom-8 left-4">
            <Avatar
              src={chef.profileImage}
              alt={chef.businessName}
              size="xl"
              className="border-4 border-[var(--bone)] shadow-2"
            />
          </div>
        </div>

        {/* Content */}
        <div className="p-5 pt-10">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0 flex-1">
              <h3 className="font-display text-lg font-semibold text-ink truncate group-hover:text-herb transition-colors">
                {chef.businessName}
              </h3>
              <p className="mt-1 text-sm text-ink-muted truncate">
                {chef.cuisines.slice(0, 2).join(' • ')}
              </p>
            </div>
            <RatingBadge value={chef.rating} />
          </div>

          <p className="mt-3 line-clamp-2 text-sm text-ink-soft">{chef.description}</p>

          <div className="mt-4 flex items-center gap-4 text-sm text-ink-muted">
            <div className="flex items-center gap-1">
              <Clock className="h-4 w-4"  aria-hidden="true" />
              {chef.prepTime}
            </div>
            <div>{chef.priceRange}</div>
            {chef.acceptingOrders ? (
              <Badge variant="success" size="sm" dot>
                Open
              </Badge>
            ) : (
              <Badge variant="default" size="sm">
                Closed
              </Badge>
            )}
          </div>
        </div>
      </article>
    </Link>
  );
}
