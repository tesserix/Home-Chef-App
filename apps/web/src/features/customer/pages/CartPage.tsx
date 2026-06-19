import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import {
  Minus,
  Plus,
  Trash2,
  ShoppingCart,
  MapPin,
  Clock,
  AlertCircle,
  ArrowRight,
  ChevronRight,
} from 'lucide-react';
import { useCartStore } from '@/app/store/cart-store';
import { useAuth } from '@/app/providers/AuthProvider';
import { useFormatPrice } from '@/shared/utils/format-price';
import { Button } from '@/shared/components/ui';
import { apiClient } from '@/shared/services/api-client';

// Pull the API's promo error out of either the string or {message,...} form (#39).
function promoErrorMessage(e: unknown): string {
  const err = (e as { error?: unknown })?.error;
  if (typeof err === 'string') return err;
  if (err && typeof err === 'object' && 'message' in err) {
    return String((err as { message?: unknown }).message ?? 'Invalid promo code');
  }
  return e instanceof Error ? e.message : 'Invalid promo code';
}

export default function CartPage() {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();
  const cart = useCartStore();
  const fp = useFormatPrice();
  const [promoInput, setPromoInput] = useState('');
  const [promoError, setPromoError] = useState<string | null>(null);
  const [applyingPromo, setApplyingPromo] = useState(false);

  const subtotal = cart.getSubtotal();
  const deliveryFee = cart.chef?.deliveryFee || 0;
  const serviceFee = subtotal * 0.05; // 5% service fee
  // Server-validated promo discount (#39), clamped to the subtotal. The server
  // re-validates + recomputes at order time, so this is a preview for display.
  const discount = cart.promoCode ? Math.min(cart.promoDiscount, subtotal) : 0;
  const total = Math.max(0, subtotal + deliveryFee + serviceFee - discount);
  const minimumOrder = cart.chef?.minimumOrder || 0;
  const belowMinimum = subtotal < minimumOrder;

  const handleApplyPromo = async () => {
    const code = promoInput.trim();
    if (!code || !cart.chefId) return;
    setApplyingPromo(true);
    setPromoError(null);
    try {
      const res = await apiClient.post<{ code: string; discount: number }>('/promo/validate', {
        code,
        orderTotal: subtotal,
        chefId: cart.chefId,
      });
      cart.setPromo(res.code, res.discount);
      setPromoInput('');
    } catch (e: unknown) {
      cart.clearPromo();
      setPromoError(promoErrorMessage(e));
    } finally {
      setApplyingPromo(false);
    }
  };

  const handleRemovePromo = () => {
    cart.clearPromo();
    setPromoError(null);
  };

  const handleCheckout = () => {
    if (!isAuthenticated) {
      navigate('/login', { state: { from: '/checkout' } });
      return;
    }
    navigate('/checkout');
  };

  if (cart.items.length === 0) {
    return (
      <div className="min-h-screen bg-paper py-12">
        <div className="container-app max-w-2xl text-center">
          <div className="rounded-xl bg-bone p-12 shadow-1">
            <div className="mx-auto h-24 w-24 rounded-full bg-mist flex items-center justify-center">
              <ShoppingCart className="h-12 w-12 text-ink-muted"  aria-hidden="true" />
            </div>
            <h2 className="mt-6 font-display text-2xl font-semibold text-ink">
              Your cart is empty
            </h2>
            <p className="mt-3 text-ink-soft">
              Looks like you haven't added any items yet. Browse our home chefs
              and discover delicious homemade food!
            </p>
            <Button asChild variant="primary" size="lg" className="mt-8">
              <Link to="/chefs">Browse Chefs</Link>
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-paper py-8">
      <div className="container-app">
        <h1 className="font-display text-2xl font-semibold text-ink md:text-3xl">Your Cart</h1>

        <div className="mt-8 flex flex-col gap-8 lg:flex-row">
          {/* Cart Items */}
          <div className="flex-1">
            {/* Chef Info */}
            {cart.chef && (
              <Link
                to={`/chefs/${cart.chef.id}`}
                className="mb-6 flex items-center gap-4 rounded-xl bg-bone p-4 shadow-1 hover:shadow-2 transition-shadow"
              >
                {cart.chef.profileImage && (
                  <img
                    src={cart.chef.profileImage}
                    alt=""
                    width={56}
                    height={56}
                    loading="lazy"
                    decoding="async"
                    className="h-14 w-14 rounded-xl object-cover shrink-0"
                    onError={(e) => { e.currentTarget.style.display = 'none'; }}
                  />
                )}
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-ink-muted">Order from</p>
                  <h3 className="font-semibold text-ink truncate">
                    {cart.chef.businessName}
                  </h3>
                </div>
                <ChevronRight className="h-5 w-5 text-ink-muted"  aria-hidden="true" />
              </Link>
            )}

            {/* Items */}
            <div className="rounded-xl bg-bone shadow-1 divide-y">
              {cart.items.map((item) => (
                <div key={item.id} className="flex gap-4 p-4">
                  {item.imageUrl && (
                    <img
                      src={item.imageUrl}
                      alt=""
                      width={80}
                      height={80}
                      loading="lazy"
                      decoding="async"
                      className="h-20 w-20 rounded-lg object-cover shrink-0"
                      onError={(e) => { e.currentTarget.style.display = 'none'; }}
                    />
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <h4 className="font-medium text-ink">{item.name}</h4>
                        <p className="text-sm text-ink-muted line-clamp-1">
                          {item.description}
                        </p>
                      </div>
                      <button type="button"
                        onClick={() => cart.removeItem(item.id)}
                        className="p-1 text-ink-muted hover:text-paprika"
                      >
                        <Trash2 className="h-5 w-5"  aria-hidden="true" />
                      </button>
                    </div>

                    {item.notes && (
                      <p className="mt-1 text-sm text-ink-muted italic">
                        Note: {item.notes}
                      </p>
                    )}

                    <div className="mt-3 flex items-center justify-between">
                      <div className="flex items-center rounded-lg border">
                        <button type="button"
                          onClick={() => cart.updateQuantity(item.id, item.quantity - 1)}
                          className="p-2 hover:bg-mist"
                        >
                          <Minus className="h-4 w-4"  aria-hidden="true" />
                        </button>
                        <span className="w-8 text-center">{item.quantity}</span>
                        <button type="button"
                          onClick={() => cart.updateQuantity(item.id, item.quantity + 1)}
                          className="p-2 hover:bg-mist"
                        >
                          <Plus className="h-4 w-4"  aria-hidden="true" />
                        </button>
                      </div>
                      <span className="font-semibold text-ink">
                        {fp(item.price * item.quantity)}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {/* Add More */}
            <Link
              to={`/chefs/${cart.chefId}`}
              className="mt-4 flex items-center justify-center gap-2 text-herb hover:text-herb"
            >
              <Plus className="h-4 w-4"  aria-hidden="true" />
              Add more items
            </Link>
          </div>

          {/* Order Summary */}
          <div className="lg:w-96">
            <div className="rounded-xl bg-bone p-6 shadow-1 lg:sticky lg:top-24">
              <h3 className="text-lg font-semibold text-ink">Order Summary</h3>

              {/* Promo Code (#39) */}
              <div className="mt-6">
                {cart.promoCode ? (
                  <div className="flex items-center justify-between rounded-lg border border-herb/30 bg-herb-tint px-3 py-2">
                    <div className="flex items-center gap-2">
                      <span className="rounded bg-herb/15 px-2 py-0.5 text-xs font-semibold text-herb">
                        {cart.promoCode}
                      </span>
                      <span className="text-sm text-herb">−{fp(discount)} applied</span>
                    </div>
                    <button
                      type="button"
                      onClick={handleRemovePromo}
                      className="text-xs text-ink-soft underline hover:text-ink"
                    >
                      Remove
                    </button>
                  </div>
                ) : (
                  <>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={promoInput}
                        onChange={(e) => setPromoInput(e.target.value.toUpperCase())}
                        placeholder="Promo code"
                        className="input-base flex-1 uppercase"
                        autoCapitalize="characters"
                      />
                      <Button
                        variant="outline"
                        onClick={handleApplyPromo}
                        isLoading={applyingPromo}
                        disabled={applyingPromo || !promoInput.trim()}
                      >
                        Apply
                      </Button>
                    </div>
                    {promoError && <p className="mt-2 text-sm text-paprika">{promoError}</p>}
                  </>
                )}
              </div>

              {/* Breakdown */}
              <div className="mt-6 space-y-3 border-t pt-6">
                <div className="flex justify-between text-ink-soft">
                  <span>Subtotal</span>
                  <span>{fp(subtotal)}</span>
                </div>
                <div className="flex justify-between text-ink-soft">
                  <span>Delivery fee</span>
                  <span>{fp(deliveryFee)}</span>
                </div>
                <div className="flex justify-between text-ink-soft">
                  <span>Service fee</span>
                  <span>{fp(serviceFee)}</span>
                </div>
                {discount > 0 && (
                  <div className="flex justify-between text-herb">
                    <span>Discount</span>
                    <span>-{fp(discount)}</span>
                  </div>
                )}
              </div>

              <div className="mt-4 flex justify-between border-t pt-4 text-lg font-semibold">
                <span>Total</span>
                <span>{fp(total)}</span>
              </div>

              {/* Minimum Order Warning */}
              {belowMinimum && (
                <div className="mt-4 flex items-start gap-2 rounded-lg bg-amber-tint p-3 text-sm text-amber">
                  <AlertCircle className="h-5 w-5 flex-shrink-0"  aria-hidden="true" />
                  <div>
                    <p className="font-medium">Minimum order not met</p>
                    <p>Add {fp(minimumOrder - subtotal)} more to proceed</p>
                  </div>
                </div>
              )}

              {/* Checkout Button */}
              <Button
                variant="primary"
                size="lg"
                fullWidth
                onClick={handleCheckout}
                disabled={belowMinimum}
                rightIcon={<ArrowRight aria-hidden="true" className="h-5 w-5" />}
                className="mt-6"
              >
                {!isAuthenticated ? 'Sign in to Checkout' : 'Proceed to Checkout'}
              </Button>

              {/* Delivery Info */}
              <div className="mt-6 space-y-3 border-t pt-6">
                <div className="flex items-center gap-3 text-sm text-ink-soft">
                  <MapPin className="h-5 w-5 text-ink-muted"  aria-hidden="true" />
                  <span>Delivery within {cart.chef?.minimumOrder} km radius</span>
                </div>
                <div className="flex items-center gap-3 text-sm text-ink-soft">
                  <Clock className="h-5 w-5 text-ink-muted"  aria-hidden="true" />
                  <span>Estimated prep time: 30-45 mins</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
