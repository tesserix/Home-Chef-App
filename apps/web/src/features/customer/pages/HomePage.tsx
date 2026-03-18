import { Link } from 'react-router-dom';
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
    <div className="min-h-screen bg-cream-50">
      {/* Hero Section */}
      <section className="relative overflow-hidden bg-gradient-to-br from-brand-50 via-white to-golden-50 py-20 lg:py-32">
        {/* Decorative background elements */}
        <div className="absolute inset-0 overflow-hidden">
          <div className="absolute -left-32 -top-32 h-64 w-64 rounded-full bg-brand-100/50 blur-3xl" />
          <div className="absolute -right-32 top-1/2 h-96 w-96 rounded-full bg-golden-100/50 blur-3xl" />
        </div>

        <div className="container-app relative">
          <motion.div
            initial="hidden"
            animate="visible"
            variants={staggerContainer}
            className="mx-auto max-w-3xl text-center"
          >
            <motion.div variants={fadeInUp}>
              <Badge variant="premium" size="lg" className="mb-6">
                <Sparkles className="h-4 w-4 mr-1" />
                500+ Home Chefs Near You
              </Badge>
            </motion.div>

            <motion.h1
              variants={fadeInUp}
              className="font-display text-display-lg md:text-display-xl lg:text-display-2xl text-gray-900"
            >
              Homemade Food,{' '}
              <span className="text-brand-500">Delivered Fresh</span>
            </motion.h1>

            <motion.p
              variants={fadeInUp}
              className="mt-6 text-lg text-gray-600 max-w-2xl mx-auto"
            >
              Discover talented home chefs in your neighborhood and enjoy authentic,
              homemade meals delivered right to your doorstep.
            </motion.p>

            {/* Search Bar */}
            <motion.div variants={fadeInUp} className="mt-10">
              <Card variant="elevated" padding="sm" className="shadow-soft-xl">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center p-2">
                  <div className="relative flex-1">
                    <Input
                      variant="ghost"
                      inputSize="lg"
                      placeholder="Enter your delivery address..."
                      leftIcon={<MapPin className="h-5 w-5" />}
                      className="border-0"
                    />
                  </div>
                  <div className="hidden sm:block w-px h-10 bg-gray-200" />
                  <div className="relative flex-1">
                    <Input
                      variant="ghost"
                      inputSize="lg"
                      placeholder="Search dishes or chefs..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      leftIcon={<Search className="h-5 w-5" />}
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

            {/* Trust Badges */}
            <motion.div
              variants={fadeInUp}
              className="mt-12 flex flex-wrap items-center justify-center gap-8"
            >
              {[
                { icon: ChefHat, label: '500+ Home Chefs' },
                { icon: Star, label: '4.8 Average Rating' },
                { icon: Clock, label: '30-45 min Delivery' },
              ].map(({ icon: Icon, label }) => (
                <div key={label} className="flex items-center gap-2 text-gray-600">
                  <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand-100">
                    <Icon className="h-5 w-5 text-brand-600" />
                  </div>
                  <span className="font-medium">{label}</span>
                </div>
              ))}
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* How It Works */}
      <section className="py-20 bg-white">
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
              <h2 className="font-display text-display-md text-gray-900">
                Get Delicious Food in 3 Steps
              </h2>
              <p className="mt-3 text-gray-600 max-w-xl mx-auto">
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
                      <span className="inline-flex h-8 w-8 items-center justify-center rounded-full bg-brand-500 text-sm font-bold text-white">
                        {index + 1}
                      </span>
                    </div>
                    <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-3xl bg-gradient-to-br from-brand-100 to-brand-50 shadow-soft-md">
                      <step.icon className="h-10 w-10 text-brand-600" />
                    </div>
                    <h3 className="mt-6 text-xl font-semibold text-gray-900">{step.title}</h3>
                    <p className="mt-3 text-gray-600">{step.description}</p>
                  </Card>
                </motion.div>
              ))}
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Cuisines */}
      <section className="py-20 bg-gray-50">
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
                <h2 className="font-display text-display-md text-gray-900">Explore Flavors</h2>
                <p className="mt-2 text-gray-600">Discover authentic dishes from around the world</p>
              </div>
              <Button asChild variant="ghost" className="hidden sm:flex">
                <Link to="/chefs">
                  View All <ArrowRight className="ml-2 h-4 w-4" />
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
                    className="group relative block overflow-hidden rounded-2xl shadow-card hover:shadow-card-hover transition-all duration-300"
                  >
                    <div className="aspect-[4/3]">
                      <img
                        src={cuisine.image}
                        alt={cuisine.name}
                        className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
                      />
                    </div>
                    <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-black/20 to-transparent" />
                    <div className="absolute bottom-0 left-0 right-0 p-4">
                      <h3 className="font-semibold text-white">{cuisine.name}</h3>
                    </div>
                  </Link>
                </motion.div>
              ))}
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Featured Chefs */}
      <section className="py-20 bg-white">
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
                  <Star className="h-3 w-3 mr-1 fill-golden-500" />
                  Top Rated
                </Badge>
                <h2 className="font-display text-display-md text-gray-900">Featured Chefs</h2>
                <p className="mt-2 text-gray-600">Our community's favorite home chefs</p>
              </div>
              <Button asChild variant="ghost" className="hidden sm:flex">
                <Link to="/chefs">
                  View All Chefs <ArrowRight className="ml-2 h-4 w-4" />
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
              className="overflow-hidden bg-gradient-to-br from-brand-500 to-brand-600"
            >
              <div className="flex flex-col items-center gap-8 p-8 text-center lg:flex-row lg:p-12 lg:text-left">
                <div className="flex-1">
                  <Badge variant="default" className="bg-white/20 text-white border-0 mb-4">
                    Catering Services
                  </Badge>
                  <h2 className="text-3xl font-bold text-white">
                    Planning an Event?
                  </h2>
                  <p className="mt-3 text-lg text-brand-100 max-w-xl">
                    Get catering quotes from multiple home chefs. Perfect for parties,
                    corporate events, and special occasions.
                  </p>
                </div>
                <Button
                  asChild
                  variant="secondary"
                  size="xl"
                  className="bg-white text-brand-600 hover:bg-brand-50"
                >
                  <Link to="/catering">Request Catering Quote</Link>
                </Button>
              </div>
            </Card>
          </motion.div>
        </div>
      </section>

      {/* Why Choose Us */}
      <section className="py-20 bg-gray-50">
        <div className="container-app">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true, margin: '-100px' }}
            variants={staggerContainer}
          >
            <motion.div variants={fadeInUp} className="text-center">
              <Badge variant="brand" className="mb-4">Why Choose Us</Badge>
              <h2 className="font-display text-display-md text-gray-900">
                The Fe3dr Difference
              </h2>
              <p className="mt-3 text-gray-600 max-w-xl mx-auto">
                Join thousands of happy customers enjoying homemade food
              </p>
            </motion.div>

            <motion.div
              variants={staggerContainer}
              className="mt-16 grid gap-6 md:grid-cols-2 lg:grid-cols-4"
            >
              {[
                {
                  icon: ChefHat,
                  title: 'Verified Chefs',
                  description: 'All our home chefs are verified for food safety and quality',
                  color: 'brand',
                },
                {
                  icon: Heart,
                  title: 'Made with Love',
                  description: 'Every meal is prepared fresh with authentic family recipes',
                  color: 'spice',
                },
                {
                  icon: Shield,
                  title: 'Secure Payments',
                  description: 'Safe and secure payment processing for every order',
                  color: 'fresh',
                },
                {
                  icon: Truck,
                  title: 'Fast Delivery',
                  description: 'Reliable delivery to your doorstep within 30-45 minutes',
                  color: 'golden',
                },
              ].map((feature) => (
                <motion.div key={feature.title} variants={scaleIn}>
                  <Card variant="default" padding="lg" hover="lift" className="text-center h-full">
                    <div className={`mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-${feature.color}-100`}>
                      <feature.icon className={`h-8 w-8 text-${feature.color}-600`} />
                    </div>
                    <h3 className="mt-5 text-lg font-semibold text-gray-900">{feature.title}</h3>
                    <p className="mt-2 text-sm text-gray-600">{feature.description}</p>
                  </Card>
                </motion.div>
              ))}
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* Become a Chef CTA */}
      <section className="py-20 bg-white">
        <div className="container-app">
          <motion.div
            initial="hidden"
            whileInView="visible"
            viewport={{ once: true }}
            variants={fadeInUp}
          >
            <Card variant="ghost" padding="none" className="overflow-hidden bg-gray-900 rounded-3xl">
              <div className="flex flex-col md:flex-row">
                <div className="flex-1 p-8 md:p-12">
                  <Badge variant="brand" className="mb-4">Join Our Community</Badge>
                  <h2 className="text-3xl font-bold text-white">
                    Love Cooking? Share Your Talent
                  </h2>
                  <p className="mt-4 text-gray-300 max-w-lg">
                    Turn your passion into income. Join our community of home chefs
                    and start earning by sharing your delicious homemade food.
                  </p>
                  <div className="mt-8 flex flex-wrap gap-4">
                    <Button asChild variant="primary" size="lg">
                      <Link to="/become-chef">Become a Chef</Link>
                    </Button>
                    <Button asChild variant="outline" size="lg" className="border-gray-600 text-white hover:bg-gray-800">
                      <Link to="/chef-resources">Learn More</Link>
                    </Button>
                  </div>
                </div>
                <div className="hidden md:block md:w-2/5 relative">
                  <img
                    src="https://images.unsplash.com/photo-1556910103-1c02745aae4d?w=600&h=500&fit=crop"
                    alt="Home chef cooking"
                    className="h-full w-full object-cover"
                  />
                  <div className="absolute inset-0 bg-gradient-to-r from-gray-900 to-transparent" />
                </div>
              </div>
            </Card>
          </motion.div>
        </div>
      </section>
    </div>
  );
}

function FeaturedChefCard({ chef }: { chef: Chef }) {
  const { isAuthenticated, login } = useAuth();
  const { isFavorite, toggle } = useFavoritesStore();
  const favorited = isFavorite(chef.id);

  const handleFavorite = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    if (!isAuthenticated) {
      toast.error('Please log in to save favorites');
      login();
      return;
    }

    const result = await toggle(chef.id);
    if (result === 'max_limit') {
      toast.error('You can save up to 7 favorite chefs. Remove one first.');
    } else if (result === 'unauthorized') {
      toast.error('Please log in to save favorites');
      login();
    } else if (result === 'error') {
      toast.error('Something went wrong. Please try again.');
    }
  };

  return (
    <Link to={`/chefs/${chef.id}`}>
      <Card
        variant="default"
        padding="none"
        hover="lift"
        className="overflow-hidden group"
      >
        {/* Banner */}
        <div className="relative h-32 overflow-hidden">
          <img
            src={chef.bannerImage || chef.profileImage}
            alt={chef.businessName}
            className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-black/50 to-transparent" />

          {chef.verified && (
            <Badge variant="success" size="sm" className="absolute top-3 left-3">
              Verified
            </Badge>
          )}

          {/* Favorite button */}
          <button
            onClick={handleFavorite}
            className="absolute top-3 right-3 flex h-8 w-8 items-center justify-center rounded-full bg-white/90 shadow-sm backdrop-blur-sm transition-all hover:bg-white hover:shadow-md"
          >
            <Heart
              className={`h-4 w-4 transition-colors ${
                favorited ? 'fill-red-500 text-red-500' : 'text-gray-600'
              }`}
            />
          </button>

          <div className="absolute -bottom-8 left-4">
            <Avatar
              src={chef.profileImage}
              alt={chef.businessName}
              size="xl"
              className="border-4 border-white shadow-elevated"
            />
          </div>
        </div>

        {/* Content */}
        <div className="p-4 pt-12">
          <div className="flex items-start justify-between">
            <div className="min-w-0 flex-1">
              <h3 className="font-semibold text-gray-900 truncate group-hover:text-brand-600 transition-colors">
                {chef.businessName}
              </h3>
              <p className="mt-1 text-sm text-gray-500 truncate">
                {chef.cuisines.slice(0, 2).join(' • ')}
              </p>
            </div>
            <RatingBadge value={chef.rating} />
          </div>

          <p className="mt-3 line-clamp-2 text-sm text-gray-600">{chef.description}</p>

          <div className="mt-4 flex items-center gap-4 text-sm text-gray-500">
            <div className="flex items-center gap-1">
              <Clock className="h-4 w-4" />
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
      </Card>
    </Link>
  );
}
