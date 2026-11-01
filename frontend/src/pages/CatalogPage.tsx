import { useEffect, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import { ProductCard } from '../components/ProductCard';
import { api, ApiError, query } from '../services/api';
import type { Page, Product } from '../types';
import { formatNumber } from '../utils/format';

const PAGE_SIZE = 12;

const sorts = [
  { value: 'price_asc', label: 'Price: low to high' },
  { value: 'price_desc', label: 'Price: high to low' },
  { value: 'rating', label: 'Top rated' },
  { value: 'newest', label: 'Newest' },
];

export default function CatalogPage() {
  const [search, setSearch] = useState('');
  const [term, setTerm] = useState('');
  const [category, setCategory] = useState('');
  const [sort, setSort] = useState('price_asc');
  const [categories, setCategories] = useState<string[]>([]);
  const [cursors, setCursors] = useState<string[]>(['']);
  const [pageIndex, setPageIndex] = useState(0);
  const [data, setData] = useState<Page<Product> | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [latency, setLatency] = useState<number | null>(null);

  useEffect(() => {
    const timer = setTimeout(() => setTerm(search.trim()), 350);
    return () => clearTimeout(timer);
  }, [search]);

  useEffect(() => {
    api
      .get<string[]>('/products/categories')
      .then((values) => setCategories([...values].sort()))
      .catch(() => setCategories([]));
  }, []);

  useEffect(() => {
    setCursors(['']);
    setPageIndex(0);
  }, [term, category, sort]);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    const started = performance.now();
    api
      .get<Page<Product>>(
        `/products${query({ q: term, category, sort, limit: PAGE_SIZE, page: pageIndex + 1, cursor: cursors[pageIndex] })}`,
        controller.signal,
      )
      .then((page) => {
        setData(page);
        setError(null);
        setLatency(performance.now() - started);
      })
      .catch((caught) => {
        if (caught instanceof DOMException && caught.name === 'AbortError') return;
        setError(caught instanceof ApiError ? caught.message : 'Could not load products');
      })
      .finally(() => setLoading(false));
    return () => controller.abort();
  }, [term, category, sort, pageIndex, cursors]);

  const goNext = () => {
    if (!data) return;
    if (data.nextCursor) {
      setCursors((previous) => {
        const next = previous.slice(0, pageIndex + 1);
        next.push(data.nextCursor ?? '');
        return next;
      });
    }
    setPageIndex((value) => value + 1);
  };

  return (
    <div>
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-extrabold text-slate-100">Catalog</h1>
          <p className="mt-1 text-sm text-slate-500">
            {data?.total !== undefined ? `${formatNumber(data.total)} products` : 'Browse the MongoScale store'}
            {latency !== null ? ` · served in ${latency.toFixed(0)} ms (${data?.pagination ?? 'offset'} pagination)` : ''}
          </p>
        </div>
      </div>

      <div className="panel mt-6 grid gap-3 p-4 md:grid-cols-[1fr_220px_220px]">
        <input className="input" placeholder="Search products, e.g. wireless" value={search} onChange={(event) => setSearch(event.target.value)} />
        <select className="input" value={category} onChange={(event) => setCategory(event.target.value)}>
          <option value="">All categories</option>
          {categories.map((entry) => (
            <option key={entry} value={entry}>
              {entry}
            </option>
          ))}
        </select>
        <select className="input" value={sort} onChange={(event) => setSort(event.target.value)}>
          {sorts.map((entry) => (
            <option key={entry.value} value={entry.value}>
              {entry.label}
            </option>
          ))}
        </select>
      </div>

      {error ? <div className="mt-6 rounded-xl border border-danger/50 bg-danger/10 px-4 py-3 text-sm text-danger">{error}</div> : null}

      <div className={`mt-6 grid gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 ${loading ? 'opacity-60' : ''}`}>
        <AnimatePresence mode="popLayout">
          {data?.items.map((product) => <ProductCard key={product.id} product={product} />)}
        </AnimatePresence>
      </div>

      {data && data.items.length === 0 && !loading ? (
        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="py-20 text-center text-slate-500">
          No products match these filters.
        </motion.div>
      ) : null}

      <div className="mt-8 flex items-center justify-center gap-4">
        <button type="button" className="btn-ghost" disabled={pageIndex === 0 || loading} onClick={() => setPageIndex((value) => value - 1)}>
          Previous
        </button>
        <span className="text-sm text-slate-500">Page {pageIndex + 1}</span>
        <button type="button" className="btn-ghost" disabled={!data?.hasMore || loading} onClick={goNext}>
          Next
        </button>
      </div>
    </div>
  );
}
