import { CookWithUs } from '@/components/cook-with-us';
import { DishShowcase } from '@/components/dish-showcase';
import { FinalCta } from '@/components/final-cta';
import { Hero } from '@/components/hero';
import { HowItWorks } from '@/components/how-it-works';
import { Marquee } from '@/components/marquee';
import { SiteFooter } from '@/components/site-footer';
import { SiteNav } from '@/components/site-nav';
import { TrustStrip } from '@/components/trust-strip';
import { WhyHomeChef } from '@/components/why-home-chef';

export default function HomePage() {
  return (
    <>
      <SiteNav />
      <main id="main">
        <Hero />
        <Marquee />
        <HowItWorks />
        <DishShowcase />
        <WhyHomeChef />
        <TrustStrip />
        <CookWithUs />
        <FinalCta />
      </main>
      <SiteFooter />
    </>
  );
}
