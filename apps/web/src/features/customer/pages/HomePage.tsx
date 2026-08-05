import { Link, useNavigate } from 'react-router';
import { useQuery } from '@tanstack/react-query';
import { motion } from 'framer-motion';
import {
  Search,
  MapPin,
  ChefHat,
  Clock,
  Utensils,
  Heart,
  Truck,
  Shield,
  Star,
  Sparkles,
  ExternalLink,
} from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';
import { cn } from '@/shared/utils/cn';
import {
  VENDOR_COST_POINTS,
  VENDOR_CTA_BLURB,
  VENDOR_CTA_LABEL,
  VENDOR_PORTAL_URL,
} from '@/shared/config/partner-sites';
import { apiClient } from '@/shared/services/api-client';
import { useFavoritesStore } from '@/app/store/favorites-store';
import { useAuth } from '@/app/providers/AuthProvider';
import type { Chef, PaginatedResponse } from '@/shared/types';
import { Button, Card, Input, Badge } from '@/shared/components/ui';
import { WinbackBanner } from '@/features/customer/components/WinbackBanner';

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
      {/* Win-back offer (#42) — shows when the customer has an active offer. */}
      <WinbackBanner />

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
      <section className="py-10 bg-paper">
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

      {/* Cuisines — a category rail, not a hero grid.
          Round tiles that scroll sideways: the same destinations in a fraction
          of the vertical budget, and they read as navigation instead of
          competing with the hero above them. */}
      <section className="border-b border-mist py-6">
        <div className="container-app">
          <div
            className="-mx-4 flex snap-x snap-mandatory gap-2 overflow-x-auto px-4 pb-1 sm:mx-0 sm:px-0 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
            role="list"
            aria-label="Browse by cuisine"
          >
            {cuisines.map((cuisine) => (
              <Link
                key={cuisine.name}
                to={`/chefs?cuisine=${cuisine.name}`}
                role="listitem"
                className="group flex w-[84px] shrink-0 snap-start flex-col items-center gap-2 rounded-lg py-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
              >
                <img
                  src={cuisine.image}
                  alt=""
                  width={112}
                  height={112}
                  loading="lazy"
                  decoding="async"
                  className="h-14 w-14 rounded-full object-cover ring-1 ring-mist transition-transform duration-200 group-hover:scale-105"
                />
                <span className="text-center text-xs font-medium leading-tight text-ink-soft group-hover:text-ink">
                  {cuisine.name}
                </span>
              </Link>
            ))}
          </div>

          {/* Filter chips. Uber Eats puts these directly under the category
              rail; each is a pre-filtered entry into the chef list rather than
              local state, so this stays a landing page and not a half-built
              search UI. */}
          <div className="mt-4 flex flex-wrap gap-2">
            {[
              { label: 'Open now', to: '/chefs?isOpen=true' },
              { label: 'Highest rated', to: '/chefs?sort=rating' },
              { label: 'Fastest delivery', to: '/chefs?sort=prepTime' },
              { label: 'Food safety verified', to: '/chefs?foodSafety=true' },
            ].map((chip) => (
              <Link
                key={chip.label}
                to={chip.to}
                className="inline-flex min-h-[36px] items-center rounded-full border border-mist px-3.5 text-sm font-medium text-ink transition-colors hover:bg-bone focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
              >
                {chip.label}
              </Link>
            ))}
          </div>
        </div>
      </section>

      {/* Featured Chefs */}
      <section className="py-10 bg-paper">
        <div className="container-app">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, margin: '-100px' }}
            variants={staggerContainer}
          >
            <motion.div variants={fadeInUp} className="flex items-baseline justify-between gap-4">
              <h2 className="font-display text-2xl font-bold tracking-tight text-ink">
                Top rated near you
              </h2>
              <Link
                to="/chefs"
                className="shrink-0 text-sm font-medium text-ink underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
              >
                See all
              </Link>
            </motion.div>

            <motion.div
              variants={staggerContainer}
              className="mt-6 grid gap-x-5 gap-y-8 sm:grid-cols-2 lg:grid-cols-3"
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
      <section className="py-10 bg-paper">
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
                  <Badge variant="brand" className="mb-4">For home chefs</Badge>
                  <h2 className="font-display text-3xl font-semibold tabular-nums text-paper">
                    Your kitchen is already open.
                    <br />
                    Start taking orders from it.
                  </h2>
                  <p className="mt-4 max-w-lg text-ink-muted">
                    {VENDOR_CTA_BLURB} No storefront, no staff, no stock to buy —
                    cook what you already cook, for people a few streets away.
                  </p>

                  {/* The costs, up front. A chef's first question is what this
                      will run them, and burying it is how you lose the good ones. */}
                  <dl className="mt-8 grid gap-5 sm:grid-cols-3">
                    {VENDOR_COST_POINTS.map((point) => (
                      <div key={point.label}>
                        <dt className="font-display text-2xl font-semibold tabular-nums text-paper">
                          {point.figure}
                        </dt>
                        <dd className="mt-1 text-sm font-medium text-paper/80">{point.label}</dd>
                        <dd className="mt-1 text-xs leading-relaxed text-ink-muted">
                          {point.detail}
                        </dd>
                      </div>
                    ))}
                  </dl>

                  {/* Chef signup lives on the vendor portal — the old /become-chef
                      and /chef-resources links were routes this app never had. */}
                  <div className="mt-8 flex flex-wrap items-center gap-4">
                    <Button asChild variant="primary" size="lg">
                      <a href={VENDOR_PORTAL_URL}>
                        {VENDOR_CTA_LABEL}
                        <ExternalLink aria-hidden="true" className="ml-2 h-4 w-4" />
                        <span className="sr-only">(opens the chef portal)</span>
                      </a>
                    </Button>
                    <p className="text-xs text-ink-muted">
                      Takes you to vendors.fe3dr.com — about 10 minutes to set up.
                    </p>
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
      className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb focus-visible:ring-offset-2"
    >
      <article className="group h-full">
        {/* Banner — uses .banner-fallback as the underlay so a missing image
            still reads as a brand-tinted cell, never flat black.
            16:9 and rounded in its own right: the card has no chrome of its
            own, so the image is the card. */}
        <div className="banner-fallback relative aspect-[16/9] overflow-hidden rounded-lg">
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

        </div>

        {/* Content — sits on the page, not in a card. */}
        <div className="pt-3">
          <div className="flex items-baseline justify-between gap-3">
            <h3 className="truncate font-display text-base font-semibold text-ink">
              {chef.businessName}
            </h3>
            {chef.acceptingOrders ? null : (
              <span className="shrink-0 text-xs font-medium text-ink-muted">Closed</span>
            )}
          </div>

          {/* One metadata line, in Uber Eats' order: rating first, because it
              is what people scan for. The description paragraph is gone — at
              three cards across it was the only thing making the row ragged. */}
          <p className="mt-1 flex flex-wrap items-center gap-x-1.5 text-sm text-ink-muted">
            <span className="inline-flex items-center gap-1 font-medium text-ink">
              <Star aria-hidden="true" className="h-3.5 w-3.5 fill-ink text-ink" />
              <span className="tabular-nums">{chef.rating.toFixed(1)}</span>
            </span>
            <span aria-hidden="true">·</span>
            <span className="inline-flex items-center gap-1">
              <Clock aria-hidden="true" className="h-3.5 w-3.5" />
              {chef.prepTime}
            </span>
            {chef.priceRange ? (
              <>
                <span aria-hidden="true">·</span>
                <span>{chef.priceRange}</span>
              </>
            ) : null}
          </p>

          <p className="mt-0.5 truncate text-sm text-ink-muted">
            {chef.cuisines.slice(0, 2).join(' · ')}
          </p>
        </div>
      </article>
    </Link>
  );
}
