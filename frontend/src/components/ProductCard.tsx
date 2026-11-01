import { useState } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import type { Product } from '../types';
import { useCartStore } from '../store/cart';
import { formatCurrency } from '../utils/format';

export function ProductImage({ product, className }: { product: Product; className?: string }) {
  const [failed, setFailed] = useState(false);
  if (failed || !product.imageUrl) {
    return (
      <div className={`flex items-center justify-center bg-gradient-to-br from-ink-700 to-ink-900 text-3xl font-black text-brand/70 ${className ?? ''}`}>
        {product.name.slice(0, 1)}
      </div>
    );
  }
  return <img src={product.imageUrl} alt={product.name} loading="lazy" onError={() => setFailed(true)} className={`object-cover ${className ?? ''}`} />;
}

export function Stars({ value }: { value: number }) {
  const full = Math.round(value);
  return (
    <span className="text-sm tracking-tight text-warn" aria-label={`${value.toFixed(1)} out of 5`}>
      {'★'.repeat(full)}
      <span className="text-slate-600">{'★'.repeat(5 - full)}</span>
    </span>
  );
}

export function ProductCard({ product }: { product: Product }) {
  const add = useCartStore((state) => state.add);
  return (
    <motion.div layout initial={{ opacity: 0, y: 14 }} animate={{ opacity: 1, y: 0 }} whileHover={{ y: -4 }} className="panel group overflow-hidden">
      <Link to={`/products/${product.id}`}>
        <ProductImage product={product} className="h-40 w-full transition duration-500 group-hover:scale-105" />
      </Link>
      <div className="space-y-2 p-4">
        <div className="text-[11px] uppercase tracking-widest text-slate-500">{product.category}</div>
        <Link to={`/products/${product.id}`} className="line-clamp-2 min-h-[2.5rem] text-sm font-semibold text-slate-100 hover:text-brand">
          {product.name}
        </Link>
        <div className="flex items-center gap-2 text-xs text-slate-500">
          <Stars value={product.rating} />
          <span>({product.ratingCount})</span>
        </div>
        <div className="flex items-center justify-between pt-1">
          <span className="text-lg font-bold text-slate-100">{formatCurrency(product.price)}</span>
          <button type="button" className="btn-primary !px-3 !py-1.5 text-xs" onClick={() => add(product)}>
            Add
          </button>
        </div>
      </div>
    </motion.div>
  );
}
