import { Link } from 'react-router-dom';
import { Bar, BarChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { motion } from 'framer-motion';
import { usePolling } from '../hooks/usePolling';
import { api } from '../services/api';
import type { LoadTestResult, OptimizationMode } from '../types';
import { classNames, formatMillis, formatNumber } from '../utils/format';

const modes: { mode: OptimizationMode; label: string }[] = [
  { mode: 'unoptimized', label: 'Unoptimized' },
  { mode: 'indexed', label: 'Indexed' },
  { mode: 'replica-set', label: 'Replica set' },
  { mode: 'sharded', label: 'Sharded' },
  { mode: 'fully-optimized', label: 'Fully optimized' },
];

const loadLevels = [1000, 10000];

type Metric = {
  key: 'p50' | 'p95' | 'p99' | 'rps' | 'errorRate' | 'failedRequests';
  label: string;
  lowerIsBetter: boolean;
  format: (value: number) => string;
};

const metricRows: Metric[] = [
  { key: 'p50', label: 'Latency p50', lowerIsBetter: true, format: formatMillis },
  { key: 'p95', label: 'Latency p95', lowerIsBetter: true, format: formatMillis },
  { key: 'p99', label: 'Latency p99', lowerIsBetter: true, format: formatMillis },
  { key: 'rps', label: 'Throughput (req/s)', lowerIsBetter: false, format: formatNumber },
  { key: 'errorRate', label: 'Error rate', lowerIsBetter: true, format: (value) => `${value.toFixed(2)}%` },
  { key: 'failedRequests', label: 'Failed requests', lowerIsBetter: true, format: formatNumber },
];

function latestFor(results: LoadTestResult[], mode: OptimizationMode, users: number): LoadTestResult | undefined {
  return results.filter((result) => result.mode === mode && result.targetUsers === users).sort((a, b) => b.finishedAt.localeCompare(a.finishedAt))[0];
}

function LevelTable({ users, results }: { users: number; results: LoadTestResult[] }) {
  const columns = modes.map((entry) => ({ ...entry, result: latestFor(results, entry.mode, users) }));
  return (
    <div className="panel overflow-x-auto p-5">
      <div className="panel-title">{formatNumber(users)} concurrent users</div>
      <table className="mt-4 w-full min-w-[720px] text-sm">
        <thead>
          <tr className="text-left text-[11px] uppercase tracking-widest text-slate-500">
            <th className="pb-3 font-medium">Metric</th>
            {columns.map((column) => (
              <th key={column.mode} className="pb-3 text-right font-medium">
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {metricRows.map((row) => {
            const values = columns.map((column) => column.result?.[row.key]);
            const present = values.filter((value): value is number => value !== undefined);
            const best = present.length > 1 ? (row.lowerIsBetter ? Math.min(...present) : Math.max(...present)) : undefined;
            return (
              <tr key={row.key} className="border-t border-line">
                <td className="py-2.5 text-slate-400">{row.label}</td>
                {values.map((value, index) => (
                  <td
                    key={columns[index].mode}
                    className={classNames(
                      'py-2.5 text-right tabular-nums',
                      value === undefined ? 'text-slate-600' : value === best ? 'font-bold text-brand' : 'text-slate-200',
                    )}
                  >
                    {value === undefined ? 'not run' : row.format(value)}
                  </td>
                ))}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function ratio(before: number | undefined, after: number | undefined): string | null {
  if (before === undefined || after === undefined || after <= 0 || before <= 0) return null;
  return `${(before / after).toFixed(1)}×`;
}

export default function ComparisonPage() {
  const { data, error } = usePolling((signal) => api.get<LoadTestResult[]>('/system/load-test/results', signal), 5000);
  const results = data ?? [];

  const before = latestFor(results, 'unoptimized', 1000);
  const after = latestFor(results, 'fully-optimized', 10000);

  const chartData = modes.map((entry) => ({
    name: entry.label,
    '1,000 users': latestFor(results, entry.mode, 1000)?.p95 ?? 0,
    '10,000 users': latestFor(results, entry.mode, 10000)?.p95 ?? 0,
  }));

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-extrabold text-slate-100">Before / after comparison</h1>
        <p className="mt-1 text-sm text-slate-500">Every completed built-in load test is stored. The latest run per mode and load level is shown.</p>
      </div>

      {error ? <div className="rounded-xl border border-danger/50 bg-danger/10 px-4 py-3 text-sm text-danger">{error}</div> : null}

      {results.length === 0 ? (
        <div className="panel p-10 text-center">
          <div className="text-xl font-bold text-slate-100">No completed load tests yet</div>
          <p className="mt-2 text-sm text-slate-500">Run 1,000 users on the Unoptimized mode, then 10,000 users on Fully optimized.</p>
          <Link to="/dashboard" className="btn-primary mt-6">
            Open the dashboard
          </Link>
        </div>
      ) : null}

      {before && after ? (
        <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} className="grid gap-4 md:grid-cols-2">
          <div className="panel border-danger/60 bg-danger/10 p-6 shadow-glowRed">
            <div className="text-xs font-bold uppercase tracking-widest text-danger">Before · unoptimized · 1,000 users</div>
            <div className="mt-3 text-4xl font-black text-danger">{formatMillis(before.p95)}</div>
            <div className="text-sm text-slate-400">p95 latency · {before.errorRate.toFixed(1)}% errors · {formatNumber(before.rps)} req/s</div>
          </div>
          <div className="panel border-brand/60 bg-brand/10 p-6 shadow-glowGreen">
            <div className="text-xs font-bold uppercase tracking-widest text-brand">After · fully optimized · 10,000 users</div>
            <div className="mt-3 text-4xl font-black text-brand">{formatMillis(after.p95)}</div>
            <div className="text-sm text-slate-400">
              p95 latency · {after.errorRate.toFixed(1)}% errors · {formatNumber(after.rps)} req/s
              {ratio(before.p95, after.p95) ? ` · ${ratio(before.p95, after.p95)} faster at 10× the load` : ''}
            </div>
          </div>
        </motion.div>
      ) : null}

      {loadLevels.map((users) => (
        <LevelTable key={users} users={users} results={results} />
      ))}

      <div className="panel p-5">
        <div className="panel-title mb-3">p95 latency by mode (ms)</div>
        <div className="h-72">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={chartData} margin={{ left: -8, right: 8 }}>
              <CartesianGrid stroke="#1c2740" strokeDasharray="3 3" />
              <XAxis dataKey="name" tick={{ fill: '#64748b', fontSize: 12 }} />
              <YAxis tick={{ fill: '#64748b', fontSize: 12 }} />
              <Tooltip contentStyle={{ background: '#0d1320', border: '1px solid #1c2740', borderRadius: 12, fontSize: 12 }} />
              <Legend wrapperStyle={{ fontSize: 12 }} />
              <Bar dataKey="1,000 users" fill="#38bdf8" radius={[6, 6, 0, 0]} />
              <Bar dataKey="10,000 users" fill="#22d3a5" radius={[6, 6, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
        <p className="mt-3 text-xs text-slate-500">Modes with no recorded run are drawn as zero. Results depend on the machine running Docker.</p>
      </div>
    </div>
  );
}
