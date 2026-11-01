import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { AnimatePresence, motion } from 'framer-motion';
import { api, ApiError } from '../services/api';
import { useAuthStore } from '../store/auth';
import { cartTotal, useCartStore } from '../store/cart';
import type { Order, ShippingAddress } from '../types';
import { formatCurrency } from '../utils/format';

const emptyAddress: ShippingAddress = {
  fullName: '',
  line1: '',
  city: '',
  state: '',
  postalCode: '',
  country: 'India',
};

export default function CartPage() {
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const { lines, setQuantity, remove, clear } = useCartStore();
  const [address, setAddress] = useState<ShippingAddress>({ ...emptyAddress, fullName: user?.name ?? '' });
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const place = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await api.post<Order>('/orders', {
        items: lines.map((line) => ({ productId: line.productId, quantity: line.quantity })),
        shippingAddress: address,
      });
      clear();
      navigate('/orders');
    } catch (caught) {
      setError(caught instanceof ApiError ? caught.message : 'Could not place the order');
    } finally {
      setBusy(false);
    }
  };

  if (lines.length === 0) {
    return (
      <div className="py-24 text-center">
        <div className="text-2xl font-bold text-slate-100">Your cart is empty</div>
        <Link to="/" className="btn-primary mt-6">
          Browse products
        </Link>
      </div>
    );
  }

  const field = (key: keyof ShippingAddress, label: string, autoComplete: string) => (
    <label className="block">
      <span className="label">{label}</span>
      <input className="input" required autoComplete={autoComplete} value={address[key]} onChange={(event) => setAddress({ ...address, [key]: event.target.value })} />
    </label>
  );

  return (
    <div className="grid gap-8 lg:grid-cols-[1fr_420px]">
      <div>
        <h1 className="text-3xl font-extrabold text-slate-100">Your cart</h1>
        <div className="mt-6 space-y-3">
          <AnimatePresence>
            {lines.map((line) => (
              <motion.div key={line.productId} layout initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0, x: -40 }} className="panel flex items-center gap-4 p-4">
                <div className="min-w-0 flex-1">
                  <div className="truncate font-semibold text-slate-100">{line.name}</div>
                  <div className="text-sm text-slate-500">{formatCurrency(line.price)} each</div>
                </div>
                <input
                  className="input !w-20"
                  type="number"
                  min={1}
                  max={20}
                  value={line.quantity}
                  onChange={(event) => setQuantity(line.productId, Number(event.target.value) || 1)}
                />
                <div className="w-28 text-right font-bold text-slate-100">{formatCurrency(line.price * line.quantity)}</div>
                <button type="button" className="text-slate-500 hover:text-danger" onClick={() => remove(line.productId)} aria-label={`Remove ${line.name}`}>
                  ✕
                </button>
              </motion.div>
            ))}
          </AnimatePresence>
        </div>
      </div>

      <form onSubmit={place} className="panel h-fit space-y-4 p-6">
        <h2 className="text-xl font-bold text-slate-100">Checkout</h2>
        {field('fullName', 'Full name', 'name')}
        {field('line1', 'Address', 'address-line1')}
        <div className="grid grid-cols-2 gap-3">
          {field('city', 'City', 'address-level2')}
          {field('state', 'State', 'address-level1')}
          {field('postalCode', 'Postal code', 'postal-code')}
          {field('country', 'Country', 'country-name')}
        </div>
        <div className="flex items-center justify-between border-t border-line pt-4 text-lg font-bold text-slate-100">
          <span>Total</span>
          <span className="text-brand">{formatCurrency(cartTotal(lines))}</span>
        </div>
        {error ? <div className="rounded-xl border border-danger/50 bg-danger/10 px-3 py-2 text-sm text-danger">{error}</div> : null}
        {user ? (
          <button type="submit" className="btn-primary w-full" disabled={busy}>
            {busy ? 'Placing order…' : 'Place order'}
          </button>
        ) : (
          <Link to="/login" state={{ from: '/cart' }} className="btn-primary w-full">
            Sign in to order
          </Link>
        )}
      </form>
    </div>
  );
}
