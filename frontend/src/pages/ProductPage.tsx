import { useCallback, useEffect, useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { motion } from 'framer-motion';
import { ProductImage, Stars } from '../components/ProductCard';
import { api, ApiError, query } from '../services/api';
import { useAuthStore } from '../store/auth';
import { useCartStore } from '../store/cart';
import type { Page, Product, Review, ReviewSummary } from '../types';
import { formatCurrency, formatDate, formatNumber } from '../utils/format';

export default function ProductPage() {
  const { id = '' } = useParams();
  const user = useAuthStore((state) => state.user);
  const add = useCartStore((state) => state.add);
  const [product, setProduct] = useState<Product | null>(null);
  const [summary, setSummary] = useState<ReviewSummary | null>(null);
  const [reviews, setReviews] = useState<Page<Review> | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [rating, setRating] = useState(5);
  const [comment, setComment] = useState('');
  const [posting, setPosting] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);

  const loadReviews = useCallback(async () => {
    const [nextSummary, page] = await Promise.all([
      api.get<ReviewSummary>(`/reviews/product/${id}/summary`),
      api.get<Page<Review>>(`/reviews/product/${id}${query({ limit: 8 })}`),
    ]);
    setSummary(nextSummary);
    setReviews(page);
  }, [id]);

  useEffect(() => {
    setProduct(null);
    setError(null);
    api
      .get<Product>(`/products/${id}`)
      .then(setProduct)
      .catch((caught) => setError(caught instanceof ApiError ? caught.message : 'Could not load the product'));
    loadReviews().catch(() => undefined);
  }, [id, loadReviews]);

  const loadMore = async () => {
    if (!reviews) return;
    const page = await api.get<Page<Review>>(
      `/reviews/product/${id}${query({ limit: 8, cursor: reviews.nextCursor, page: (reviews.page ?? 1) + 1 })}`,
    );
    setReviews({ ...page, items: [...reviews.items, ...page.items] });
  };

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setPosting(true);
    setNotice(null);
    try {
      await api.post('/reviews', { productId: id, rating, comment });
      setComment('');
      setNotice('Thanks, your review was posted.');
      await loadReviews();
    } catch (caught) {
      setNotice(caught instanceof ApiError ? caught.message : 'Could not post the review');
    } finally {
      setPosting(false);
    }
  };

  if (error) {
    return <div className="rounded-xl border border-danger/50 bg-danger/10 px-4 py-3 text-danger">{error}</div>;
  }
  if (!product) {
    return <div className="py-24 text-center text-slate-500">Loading product…</div>;
  }

  const distribution = summary?.distribution ?? {};
  const total = summary?.totalReviews ?? 0;

  return (
    <div>
      <Link to="/" className="text-sm text-slate-500 hover:text-brand">
        ← Back to catalog
      </Link>
      <div className="mt-4 grid gap-8 lg:grid-cols-2">
        <motion.div initial={{ opacity: 0, scale: 0.97 }} animate={{ opacity: 1, scale: 1 }} className="panel overflow-hidden">
          <ProductImage product={product} className="h-96 w-full" />
        </motion.div>
        <div className="space-y-5">
          <div className="text-xs uppercase tracking-widest text-slate-500">{product.category}</div>
          <h1 className="text-3xl font-extrabold text-slate-100">{product.name}</h1>
          <div className="flex items-center gap-3">
            <Stars value={summary?.averageRating ?? product.rating} />
            <span className="text-sm text-slate-500">
              {(summary?.averageRating ?? product.rating).toFixed(1)} · {formatNumber(total || product.ratingCount)} reviews
            </span>
          </div>
          <div className="text-4xl font-black text-brand">{formatCurrency(product.price)}</div>
          <p className="text-slate-400">{product.description}</p>
          <div className="text-sm text-slate-500">{formatNumber(product.stock)} in stock</div>
          <button type="button" className="btn-primary" onClick={() => add(product)}>
            Add to cart
          </button>

          <div className="panel space-y-2 p-4">
            <div className="panel-title">Rating breakdown (aggregation pipeline)</div>
            {[5, 4, 3, 2, 1].map((star) => {
              const count = distribution[String(star)] ?? 0;
              return (
                <div key={star} className="flex items-center gap-3 text-xs">
                  <span className="w-6 text-slate-400">{star}★</span>
                  <div className="h-2 flex-1 overflow-hidden rounded-full bg-ink-700">
                    <motion.div className="h-full rounded-full bg-warn" animate={{ width: total > 0 ? `${(count / total) * 100}%` : '0%' }} />
                  </div>
                  <span className="w-12 text-right tabular-nums text-slate-500">{formatNumber(count)}</span>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <section className="mt-12 grid gap-8 lg:grid-cols-[1fr_380px]">
        <div>
          <h2 className="text-xl font-bold text-slate-100">Reviews</h2>
          <div className="mt-4 space-y-3">
            {reviews?.items.map((review) => (
              <motion.div key={review.id} initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} className="panel p-4">
                <div className="flex items-center justify-between">
                  <Stars value={review.rating} />
                  <span className="text-xs text-slate-500">{formatDate(review.createdAt)}</span>
                </div>
                <p className="mt-2 text-sm text-slate-300">{review.comment}</p>
              </motion.div>
            ))}
            {reviews && reviews.items.length === 0 ? <div className="text-sm text-slate-500">No reviews yet.</div> : null}
          </div>
          {reviews?.hasMore ? (
            <button type="button" className="btn-ghost mt-4" onClick={loadMore}>
              Load more reviews
            </button>
          ) : null}
        </div>

        <form onSubmit={submit} className="panel h-fit space-y-4 p-5">
          <h3 className="font-bold text-slate-100">Write a review</h3>
          {user ? (
            <>
              <div>
                <span className="label">Rating</span>
                <div className="flex gap-1">
                  {[1, 2, 3, 4, 5].map((star) => (
                    <button key={star} type="button" onClick={() => setRating(star)} className={`text-2xl ${star <= rating ? 'text-warn' : 'text-slate-600'}`}>
                      ★
                    </button>
                  ))}
                </div>
              </div>
              <label className="block">
                <span className="label">Comment</span>
                <textarea className="input min-h-[96px]" value={comment} onChange={(event) => setComment(event.target.value)} maxLength={1000} />
              </label>
              <button type="submit" className="btn-primary w-full" disabled={posting}>
                {posting ? 'Posting…' : 'Post review'}
              </button>
              {notice ? <div className="text-sm text-slate-400">{notice}</div> : null}
            </>
          ) : (
            <p className="text-sm text-slate-500">
              <Link to="/login" className="text-brand hover:underline">
                Sign in
              </Link>{' '}
              to write a review.
            </p>
          )}
        </form>
      </section>
    </div>
  );
}
