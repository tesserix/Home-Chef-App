// The bakery section (#1065) — cakes, breads, pastries and patties from every
// kitchen that bakes, whether that's its whole trade or a shelf beside its
// meals. The filter vocabulary comes from the server so the chips can never
// drift from what bakers can actually set.

import { useState } from "react";
import { Link } from "react-router";
import { useQuery } from "@tanstack/react-query";
import { CakeSlice, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { apiClient } from "@/shared/services/api-client";
import { useCartStore, type BakeryLineConfig } from "@/app/store/cart-store";
import { useFormatPrice } from "@/shared/utils/format-price";
import { Button } from "@/shared/components/ui";
import type { MenuItem } from "@/shared/types";
import { BakeryConfigurator } from "../components/BakeryConfigurator";

interface VocabItem {
  value: string;
  label: string;
}

interface BakeryOptionsResponse {
  productTypes: VocabItem[];
  occasions: VocabItem[];
  diets: VocabItem[];
  allergens: VocabItem[];
}

interface BakeryProduct extends MenuItem {
  chefName?: string;
  chefCity?: string;
  chefRating?: number;
}

interface BakeryProductsResponse {
  products: BakeryProduct[];
  total: number;
}

interface Filters {
  productType: string;
  occasion: string;
  dietary: string;
}

export default function BakeryPage() {
  const [filters, setFilters] = useState<Filters>({
    productType: "",
    occasion: "",
    dietary: "",
  });

  const { data: options } = useQuery({
    queryKey: ["bakery", "options"],
    queryFn: () => apiClient.get<BakeryOptionsResponse>("/bakery/options"),
    staleTime: 60 * 60 * 1000,
  });

  const { data, isLoading } = useQuery({
    queryKey: ["bakery", "products", filters],
    queryFn: () => {
      const qs = new URLSearchParams();
      if (filters.productType) qs.set("productType", filters.productType);
      if (filters.occasion) qs.set("occasion", filters.occasion);
      if (filters.dietary) qs.set("dietary", filters.dietary);
      const q = qs.toString();
      return apiClient.get<BakeryProductsResponse>(
        `/bakery/products${q ? `?${q}` : ""}`,
      );
    },
  });

  const products = data?.products ?? [];

  return (
    <div className="min-h-screen bg-paper">
      <div className="container-app py-8">
        <header className="mb-6">
          <h1 className="font-display text-2xl font-semibold text-ink md:text-3xl">
            Bakery
          </h1>
          <p className="mt-1 text-ink-soft">
            Cakes, breads and pastries baked to order. Pick a size, a flavour and
            what it's for.
          </p>
        </header>

        <div className="space-y-3">
          <FilterRow
            label="What are you after?"
            allLabel="All bakes"
            options={options?.productTypes ?? []}
            value={filters.productType}
            onChange={(v) => setFilters((f) => ({ ...f, productType: v }))}
          />
          <FilterRow
            label="Occasion"
            allLabel="Any occasion"
            options={options?.occasions ?? []}
            value={filters.occasion}
            onChange={(v) => setFilters((f) => ({ ...f, occasion: v }))}
          />
          <FilterRow
            label="Diet"
            allLabel="Any diet"
            options={options?.diets ?? []}
            value={filters.dietary}
            onChange={(v) => setFilters((f) => ({ ...f, dietary: v }))}
          />
        </div>

        <div className="mt-8">
          {isLoading ? (
            <div className="flex justify-center py-16">
              <Loader2
                className="h-8 w-8 animate-spin text-herb"
                aria-hidden="true"
              />
            </div>
          ) : products.length === 0 ? (
            <div className="flex flex-col items-center py-16 text-center">
              <CakeSlice
                className="h-12 w-12 text-ink-muted"
                aria-hidden="true"
              />
              <p className="mt-3 font-medium text-ink">
                Nothing matches those filters
              </p>
              <p className="mt-1 text-sm text-ink-muted">
                Try a different bake or occasion.
              </p>
            </div>
          ) : (
            <ul className="grid gap-4 md:grid-cols-2">
              {products.map((p) => (
                <li key={p.id}>
                  <ProductCard product={p} />
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}

function FilterRow({
  label,
  allLabel,
  options,
  value,
  onChange,
}: {
  label: string;
  allLabel: string;
  options: VocabItem[];
  value: string;
  onChange: (value: string) => void;
}) {
  if (options.length === 0) return null;
  return (
    <div>
      <span className="mb-1.5 block text-xs font-medium tracking-wide text-ink-muted uppercase">
        {label}
      </span>
      <div className="flex flex-wrap gap-2">
        {[{ value: "", label: allLabel }, ...options].map((o) => {
          const on = value === o.value;
          return (
            <button
              key={o.value || "all"}
              type="button"
              aria-pressed={on}
              onClick={() => onChange(o.value)}
              className={`min-h-[40px] rounded-full border px-4 text-sm text-ink ${
                on ? "border-herb bg-herb-tint" : "border-mist"
              }`}
            >
              {o.label}
            </button>
          );
        })}
      </div>
    </div>
  );
}

function ProductCard({ product }: { product: BakeryProduct }) {
  const fp = useFormatPrice();
  const cart = useCartStore();
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState<{
    config: BakeryLineConfig;
    quantity: number;
  } | null>(null);

  const spec = product.bakery;
  const from =
    spec && spec.pricePerKg > 0
      ? spec.pricePerKg * spec.minWeightKg
      : product.price;

  const chefInfo = {
    id: product.chefId,
    businessName: product.chefName ?? "This kitchen",
    deliveryFee: 0,
    minimumOrder: 0,
  };

  const add = (config: BakeryLineConfig, quantity: number) => {
    try {
      if (cart.items.length === 0) cart.setChef(chefInfo);
      cart.addItem(product, quantity, undefined, undefined, config);
      toast.success(`Added ${product.name} to cart`);
    } catch (error) {
      // Each order goes to one kitchen, so adding from a second one is a
      // choice between the two — offer it rather than naming the obstacle.
      if (error instanceof Error && error.message === "DIFFERENT_CHEF") {
        setPending({ config, quantity });
      }
    }
  };

  return (
    <div className="card p-4">
      <div className="flex gap-4">
        {product.imageUrl && (
          <img
            src={product.imageUrl}
            alt=""
            width={96}
            height={96}
            loading="lazy"
            decoding="async"
            className="h-24 w-24 shrink-0 rounded-lg object-cover"
          />
        )}
        <div className="min-w-0 flex-1">
          <h2 className="font-semibold text-ink">{product.name}</h2>
          <Link
            to={`/chefs/${product.chefId}`}
            className="text-sm text-ink-muted hover:text-ink"
          >
            {product.chefName ?? "View kitchen"}
            {product.chefCity ? ` · ${product.chefCity}` : ""}
          </Link>
          {product.description && (
            <p className="mt-1 line-clamp-2 text-sm text-ink-muted">
              {product.description}
            </p>
          )}
          <div className="mt-3 flex items-center justify-between gap-3">
            <span className="font-semibold text-ink tabular-nums">
              {spec && spec.pricePerKg > 0 ? `from ${fp(from)}` : fp(from)}
            </span>
            <Button variant="primary" size="sm" onClick={() => setOpen(true)}>
              Customise
            </Button>
          </div>
          {spec && spec.leadTimeHours > 0 && (
            <p className="mt-1 text-xs text-ink-muted">
              Needs {spec.leadTimeHours}h notice
            </p>
          )}
        </div>
      </div>

      <BakeryConfigurator
        item={product}
        open={open}
        onOpenChange={setOpen}
        onConfirm={(config, summary, unitPrice, qty) => {
          setOpen(false);
          add(
            {
              bakery: config,
              unitPrice,
              summary,
              leadTimeHours: spec?.leadTimeHours ?? 0,
            },
            qty,
          );
        }}
      />

      {pending && (
        <ReplaceCartDialog
          itemName={product.name}
          currentChef={cart.chef?.businessName}
          onCancel={() => setPending(null)}
          onReplace={() => {
            cart.clearCart();
            cart.setChef(chefInfo);
            cart.addItem(
              product,
              pending.quantity,
              undefined,
              undefined,
              pending.config,
            );
            toast.success(`Cart replaced with ${product.name}`);
            setPending(null);
          }}
        />
      )}
    </div>
  );
}

function ReplaceCartDialog({
  itemName,
  currentChef,
  onCancel,
  onReplace,
}: {
  itemName: string;
  currentChef?: string;
  onCancel: () => void;
  onReplace: () => void;
}) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Start a new cart?"
      className="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-4"
    >
      <div className="w-full max-w-sm rounded-2xl bg-paper p-5 shadow-3">
        <h3 className="font-semibold text-ink">Start a new cart?</h3>
        <p className="mt-2 text-sm text-ink-soft">
          Your cart has food from{" "}
          <span className="font-medium text-ink">
            {currentChef ?? "another kitchen"}
          </span>
          . Each order goes to one kitchen, so adding {itemName} will empty your
          current cart and start a new one.
        </p>
        <div className="mt-4 flex justify-end gap-2">
          <Button variant="outline" onClick={onCancel}>
            Keep my cart
          </Button>
          <Button onClick={onReplace}>Replace cart</Button>
        </div>
      </div>
    </div>
  );
}
