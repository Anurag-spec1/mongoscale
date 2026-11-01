import { useCallback, useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { StatusPill, type Tone } from '../components/primitives';
import { api, ApiError, query } from '../services/api';
import type { Order, Page } from '../types';
import { formatCurrency, formatDate } from '../utils/format';

const tones: Record<Order['status'], Tone> = {
  pending: 'warn',
  paid: 'info',
  shipped: 'info',
  delivered: 'good',
  cancelled: 'bad',
};

export default function OrdersPage() {
  const [page, setPage] = useState<Page<Order> | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [working, setWorking] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      setPage(await api.get<Page<Order>>(`/orders${query({ limit: 10 })}`));
      setError(null);
    } catch (caught) {
      setError(caught instanceof ApiError ? caught.message : 'Could not load orders');
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const loadMore = async () => {
    if (!page) return;
    const next = await api.get<Page<Order>>(`/orders${query({ limit: 10, cursor: page.nextCursor, page: (page.page ?? 1) + 1 })}`);
    setPage({ ...next, items: [...page.items, ...next.items] });
  };

  const transition = async (order: Order, status: Order['status']) => {
    setWorking(order.id);
    try {
      await api.patch<Order>(`/orders/${order.id}/status`, { status });
      await load();
    } catch (caught) {
      setError(caught instanceof ApiError ? caught.message : 'Could not update the order');
    } finally {
      setWorking(null);
    }
  };

  return (
    <div>
      <h1 className="text-3xl font-extrabold text-slate-100">Order history</h1>
      {error ? <div className="mt-4 rounded-xl border border-danger/50 bg-danger/10 px-4 py-3 text-sm text-danger">{error}</div> : null}
      <div className="mt-6 space-y-4">
        {page?.items.map((order) => (
          <motion.div key={order.id} initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} className="panel p-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="font-mono text-xs text-slate-500">#{order.id}</div>
                <div className="text-sm text-slate-400">{formatDate(order.createdAt)}</div>
              </div>
              <StatusPill tone={tones[order.status]}>{order.status}</StatusPill>
              <div className="text-xl font-bold text-slate-100">{formatCurrency(order.totalAmount)}</div>
            </div>
            <ul className="mt-3 space-y-1 text-sm text-slate-400">
              {order.items.map((item) => (
                <li key={item.productId} className="flex justify-between">
                  <span>
                    {item.quantity} × {item.name}
                  </span>
                  <span>{formatCurrency(item.price * item.quantity)}</span>
                </li>
              ))}
            </ul>
            <div className="mt-4 flex gap-2">
              {order.status === 'pending' ? (
                <button type="button" className="btn-primary !py-1.5" disabled={working === order.id} onClick={() => transition(order, 'paid')}>
                  Pay now
                </button>
              ) : null}
              {order.status === 'pending' || order.status === 'paid' ? (
                <button type="button" className="btn-ghost !py-1.5" disabled={working === order.id} onClick={() => transition(order, 'cancelled')}>
                  Cancel
                </button>
              ) : null}
            </div>
          </motion.div>
        ))}
        {page && page.items.length === 0 ? <div className="py-16 text-center text-slate-500">No orders yet.</div> : null}
      </div>
      {page?.hasMore ? (
        <button type="button" className="btn-ghost mt-6" onClick={loadMore}>
          Load more
        </button>
      ) : null}
    </div>
  );
}
