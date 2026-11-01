import { motion } from 'framer-motion';
import type { MetricsResponse, MongoStatus, QueryPlan } from '../types';
import { classNames, formatMillis, formatNumber } from '../utils/format';
import { ProgressRing, StatusPill, type Tone } from './primitives';

export function QueryPlanTable({ plans }: { plans: QueryPlan[] }) {
  return (
    <div className="panel p-5">
      <div className="flex items-center justify-between">
        <span className="panel-title">Query plans (explain executionStats)</span>
        <span className="text-xs text-slate-500">refreshed every 15 s</span>
      </div>
      <div className="mt-4 overflow-x-auto">
        <table className="w-full min-w-[640px] text-left text-sm">
          <thead className="text-[11px] uppercase tracking-widest text-slate-500">
            <tr>
              <th className="pb-2 font-medium">Query</th>
              <th className="pb-2 font-medium">Plan</th>
              <th className="pb-2 text-right font-medium">Docs examined</th>
              <th className="pb-2 text-right font-medium">Returned</th>
              <th className="pb-2 text-right font-medium">Time</th>
            </tr>
          </thead>
          <tbody>
            {plans.length === 0 ? (
              <tr>
                <td colSpan={5} className="py-6 text-center text-slate-500">
                  Waiting for the first explain run…
                </td>
              </tr>
            ) : null}
            {plans.map((plan) => {
              const scan = plan.stage === 'COLLSCAN';
              return (
                <motion.tr key={plan.name} layout className="border-t border-line">
                  <td className="py-2.5 pr-3">
                    <div className="font-medium text-slate-200">{plan.name}</div>
                    <div className="font-mono text-[11px] text-slate-500">{plan.collection}: {plan.filter}</div>
                  </td>
                  <td className="py-2.5 pr-3">
                    <StatusPill tone={scan ? 'bad' : 'good'}>{scan ? 'COLLSCAN' : 'IXSCAN'}</StatusPill>
                    {plan.indexName ? <div className="mt-1 font-mono text-[11px] text-slate-500">{plan.indexName}</div> : null}
                  </td>
                  <td className={classNames('py-2.5 text-right tabular-nums', scan ? 'font-semibold text-danger' : 'text-slate-300')}>
                    {formatNumber(plan.docsExamined)}
                  </td>
                  <td className="py-2.5 text-right tabular-nums text-slate-300">{formatNumber(plan.returned)}</td>
                  <td className="py-2.5 text-right tabular-nums text-slate-300">{plan.executionMillis} ms</td>
                </motion.tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export function IndexUsageList({ indexes }: { indexes: MongoStatus['indexes'] }) {
  const max = Math.max(1, ...indexes.map((index) => index.accesses));
  return (
    <div className="panel p-5">
      <span className="panel-title">Index usage (accesses since start)</span>
      <div className="mt-4 space-y-2.5">
        {indexes.length === 0 ? <div className="text-sm text-slate-500">No index statistics yet.</div> : null}
        {indexes.map((index) => (
          <div key={`${index.collection}.${index.name}`}>
            <div className="flex justify-between text-xs">
              <span className="font-mono text-slate-300">
                {index.collection}.{index.name}
              </span>
              <span className="tabular-nums text-slate-500">{formatNumber(index.accesses)}</span>
            </div>
            <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-ink-700">
              <motion.div
                className="h-full rounded-full bg-info"
                animate={{ width: `${(index.accesses / max) * 100}%` }}
                transition={{ duration: 0.6 }}
              />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function poolTone(percent: number): Tone {
  if (percent >= 90) return 'bad';
  if (percent >= 65) return 'warn';
  return 'good';
}

function cpuTone(percent: number): Tone {
  if (percent >= 90) return 'bad';
  if (percent >= 70) return 'warn';
  return 'info';
}

export function HealthGauges({ metrics, mongo }: { metrics: MetricsResponse | null; mongo: MongoStatus | null }) {
  const pool = metrics?.pool;
  const current = metrics?.snapshot.current;
  const poolPercent = pool && pool.max > 0 ? (pool.inUse / pool.max) * 100 : 0;
  const cacheTotal = (metrics?.cache.hits ?? 0) + (metrics?.cache.misses ?? 0);
  const cacheRate = cacheTotal > 0 ? ((metrics?.cache.hits ?? 0) / cacheTotal) * 100 : 0;
  const connectionLimit = (mongo?.connections.current ?? 0) + (mongo?.connections.available ?? 0);

  return (
    <div className="panel p-5">
      <span className="panel-title">Database health</span>
      <div className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-4">
        <ProgressRing value={poolPercent} max={100} label="Pool in use" caption={pool ? `${pool.inUse}/${pool.max}` : undefined} tone={poolTone(poolPercent)} />
        <ProgressRing value={current?.cpuPercent ?? 0} max={100} label="API CPU" tone={cpuTone(current?.cpuPercent ?? 0)} />
        <ProgressRing
          value={mongo?.connections.current ?? 0}
          max={Math.max(connectionLimit, 1)}
          label="Mongo connections"
          caption={mongo ? String(mongo.connections.current) : undefined}
          tone="info"
        />
        <ProgressRing value={cacheRate} max={100} label="Cache hit rate" caption={metrics?.cache.enabled ? undefined : 'off'} tone={metrics?.cache.enabled ? 'good' : 'neutral'} />
      </div>
      <dl className="mt-5 grid grid-cols-2 gap-x-6 gap-y-2 text-sm sm:grid-cols-3">
        <Fact label="Pool waiting" value={formatNumber(pool?.waiting ?? 0)} bad={(pool?.waiting ?? 0) > 0} />
        <Fact label="Checkout failures" value={formatNumber(pool?.failed ?? 0)} bad={(pool?.failed ?? 0) > 0} />
        <Fact label="Slow queries" value={formatNumber(metrics?.slowQueries ?? 0)} bad={(metrics?.slowQueries ?? 0) > 100} />
        <Fact label="API memory" value={`${(current?.memoryMB ?? 0).toFixed(0)} MB`} />
        <Fact label="mongod resident" value={`${mongo?.residentMB ?? 0} MB`} />
        <Fact label="Latency p99" value={formatMillis(current?.p99 ?? 0)} bad={(current?.p99 ?? 0) > 2000} />
      </dl>
    </div>
  );
}

function Fact({ label, value, bad = false }: { label: string; value: string; bad?: boolean }) {
  return (
    <div>
      <dt className="text-[11px] uppercase tracking-widest text-slate-500">{label}</dt>
      <dd className={classNames('font-semibold tabular-nums', bad ? 'text-danger' : 'text-slate-100')}>{value}</dd>
    </div>
  );
}
