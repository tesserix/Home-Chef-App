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

export default function CartPage() {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();
  const cart = useCartStore();
  const fp = useFormatPrice();
  const [promoCode, setPromoCode] = useState('');
  const [promoApplied, setPromoApplied] = useState(false);

  const subtotal = cart.getSubtotal();
  const deliveryFee = cart.chef?.deliveryFee || 0;
  const serviceFee = subtotal * 0.05; // 5% service fee
  const discount = promoApplied ? subtotal * 0.1 : 0; // 10% discount
  const total = subtotal + deliveryFee + serviceFee - discount;
  const minimumOrder = cart.chef?.minimumOrder || 0;
  const belowMinimum = subtotal < minimumOrder;

  const handleApplyPromo = () => {
    if (promoCode.toLowerCase() === 'fe3dr10') {
      setPromoApplied(true);
    }
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
          <div className="rounded-xl bg-bone p-12 shadow-sm">
            <div className="mx-auto h-24 w-24 rounded-full bg-mist flex items-center justify-center">
              <ShoppingCart className="h-12 w-12 text-ink-muted" />
            </div>
            <h2 className="mt-6 font-display text-2xl font-semibold text-ink">
              Your cart is empty
            </h2>
            <p className="mt-3 text-ink-soft">
              Looks like you haven't added any items yet. Browse our home chefs
              and discover delicious homemade food!
            </p>
            <Link to="/chefs" className="btn-primary mt-8 inline-flex">
              Browse Chefs
            </Link>
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
                className="mb-6 flex items-center gap-4 rounded-xl bg-bone p-4 shadow-sm hover:shadow-md transition-shadow"
              >
                {cart.chef.profileImage && (
                  <img
                    src={cart.chef.profileImage}
                    alt={cart.chef.businessName}
                    className="h-14 w-14 rounded-xl object-cover"
                  />
                )}
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-ink-muted">Order from</p>
                  <h3 className="font-semibold text-ink truncate">
                    {cart.chef.businessName}
                  </h3>
                </div>
                <ChevronRight className="h-5 w-5 text-ink-muted" />
              </Link>
            )}

            {/* Items */}
            <div className="rounded-xl bg-bone shadow-sm divide-y">
              {cart.items.map((item) => (
                <div key={item.id} className="flex gap-4 p-4">
                  {item.imageUrl && (
                    <img
                      src={item.imageUrl}
                      alt={item.name}
                      className="h-20 w-20 rounded-lg object-cover"
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
                      <button
                        onClick={() => cart.removeItem(item.id)}
                        className="p-1 text-ink-muted hover:text-paprika"
                      >
                        <Trash2 className="h-5 w-5" />
                      </button>
                    </div>

                    {item.notes && (
                      <p className="mt-1 text-sm text-ink-muted italic">
                        Note: {item.notes}
                      </p>
                    )}

                    <div className="mt-3 flex items-center justify-between">
                      <div className="flex items-center rounded-lg border">
                        <button
                          onClick={() => cart.updateQuantity(item.id, item.quantity - 1)}
                          className="p-2 hover:bg-mist"
                        >
                          <Minus className="h-4 w-4" />
                        </button>
                        <span className="w-8 text-center">{item.quantity}</span>
                        <button
                          onClick={() => cart.updateQuantity(item.id, item.quantity + 1)}
                          className="p-2 hover:bg-mist"
                        >
                          <Plus className="h-4 w-4" />
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
              <Plus className="h-4 w-4" />
              Add more items
            </Link>
          </div>

          {/* Order Summary */}
          <div className="lg:w-96">
            <div className="rounded-xl bg-bone p-6 shadow-sm lg:sticky lg:top-24">
              <h3 className="text-lg font-semibold text-ink">Order Summary</h3>

              {/* Promo Code */}
              <div className="mt-6">
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={promoCode}
                    onChange={(e) => setPromoCode(e.target.value)}
                    placeholder="Promo code"
                    className="input-base flex-1"
                    disabled={promoApplied}
                  />
                  <button
                    onClick={handleApplyPromo}
                    disabled={promoApplied || !promoCode}
                    className="btn-outline"
                  >
                    {promoApplied ? 'Applied' : 'Apply'}
                  </button>
                </div>
                {promoApplied && (
                  <p className="mt-2 text-sm text-herb">
                    Promo code applied! 10% off
                  </p>
                )}
                <p className="mt-2 text-xs text-ink-muted">
                  Try: FE3DR10 for 10% off
                </p>
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
                  <AlertCircle className="h-5 w-5 flex-shrink-0" />
                  <div>
                    <p className="font-medium">Minimum order not met</p>
                    <p>Add {fp(minimumOrder - subtotal)} more to proceed</p>
                  </div>
                </div>
              )}

              {/* Checkout Button */}
              <button
                onClick={handleCheckout}
                disabled={belowMinimum}
                className="btn-primary mt-6 w-full py-4 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {!isAuthenticated ? 'Sign in to Checkout' : 'Proceed to Checkout'}
                <ArrowRight className="ml-2 h-5 w-5" />
              </button>

              {/* Delivery Info */}
              <div className="mt-6 space-y-3 border-t pt-6">
                <div className="flex items-center gap-3 text-sm text-ink-soft">
                  <MapPin className="h-5 w-5 text-ink-muted" />
                  <span>Delivery within {cart.chef?.minimumOrder} km radius</span>
                </div>
                <div className="flex items-center gap-3 text-sm text-ink-soft">
                  <Clock className="h-5 w-5 text-ink-muted" />
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
