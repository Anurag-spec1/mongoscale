import { Link } from 'react-router-dom';
import { Bar, BarChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
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
    <div className="panel p-5">
      <div className="panel-title">{formatNumber(users)} concurrent users</div>
      <div className="mt-4 max-h-96 overflow-auto">
        <table className="w-full min-w-[720px] text-sm">
        <thead className="sticky top-0 z-10 bg-ink-900">
          <tr className="text-left text-[11px] text-slate-400">
            <th scope="col" className="pb-3 font-medium">Metric</th>
            {columns.map((column) => (
              <th key={column.mode} scope="col" className="pb-3 text-right font-medium">
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
                      value === undefined ? 'text-slate-500' : value === best ? 'font-semibold text-brand' : 'text-slate-200',
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
    </div>
  );
}

function ratio(before: number | undefined, after: number | undefined): string | null {
  if (before === undefined || after === undefined || after <= 0 || before <= 0) return null;
  return `${(before / after).toFixed(1)}x`;
}

export default function ComparisonPage() {
  const { data, error, loading } = usePolling((signal) => api.get<LoadTestResult[]>('/system/load-test/results', signal), 5000);
  const results = data ?? [];

  const before = latestFor(results, 'unoptimized', 1000);
  const after = latestFor(results, 'fully-optimized', 10000);

  const chartData = modes.map((entry) => ({
    name: entry.label,
    '1,000 users': latestFor(results, entry.mode, 1000)?.p95,
    '10,000 users': latestFor(results, entry.mode, 10000)?.p95,
  }));

  return (
    <div className="space-y-6">
      <p className="text-sm text-slate-400">Every completed built-in load test is stored. The latest run per mode and load level is shown.</p>

      {error ? <div role="alert" className="border-l-2 border-danger bg-status-danger/10 px-4 py-3 text-sm text-danger">{error}</div> : null}

      {loading && !data ? <div role="status" className="panel px-5 py-8 text-center text-sm text-slate-400">Loading comparison results…</div> : null}

      {data && results.length === 0 ? (
        <div className="panel p-8 text-center">
          <div className="text-lg font-semibold text-slate-100">No completed load tests yet</div>
          <p className="mt-2 text-sm text-slate-500">Run 1,000 users on the Unoptimized mode, then 10,000 users on Fully optimized.</p>
          <Link to="/dashboard" className="btn-primary mt-6">
            Open the dashboard
          </Link>
        </div>
      ) : null}

      {before && after ? (
        <div className="grid gap-4 md:grid-cols-2">
          <div className="panel border-l-4 border-l-status-danger p-5">
            <div className="text-xs font-semibold text-status-danger">Before | Unoptimized | 1,000 users</div>
            <div className="mt-3 font-mono text-3xl font-semibold text-slate-100">{formatMillis(before.p95)}</div>
            <div className="text-sm text-slate-400">p95 latency | {before.errorRate.toFixed(1)}% errors | {formatNumber(before.rps)} req/s</div>
          </div>
          <div className="panel border-l-4 border-l-status-good p-5">
            <div className="text-xs font-semibold text-status-good">After | Fully optimized | 10,000 users</div>
            <div className="mt-3 font-mono text-3xl font-semibold text-slate-100">{formatMillis(after.p95)}</div>
            <div className="text-sm text-slate-400">
              p95 latency | {after.errorRate.toFixed(1)}% errors | {formatNumber(after.rps)} req/s
              {ratio(before.p95, after.p95) ? ` | ${ratio(before.p95, after.p95)} faster at 10x the load` : ''}
            </div>
          </div>
        </div>
      ) : null}

      {loadLevels.map((users) => (
        <LevelTable key={users} users={users} results={results} />
      ))}

      <div className="panel p-5">
        <div className="panel-title mb-3">p95 latency by mode (ms)</div>
        {results.some((result) => loadLevels.includes(result.targetUsers)) ? <div className="h-72">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={chartData} margin={{ left: -8, right: 8 }}>
              <CartesianGrid stroke="#283039" />
              <XAxis dataKey="name" tick={{ fill: '#9aa5b1', fontSize: 11 }} />
              <YAxis tick={{ fill: '#9aa5b1', fontSize: 11 }} />
              <Tooltip contentStyle={{ background: '#171c22', border: '1px solid #2a313a', borderRadius: 4, fontSize: 12 }} />
              <Legend wrapperStyle={{ fontSize: 11 }} />
              <Bar dataKey="1,000 users" fill="#73a7ff" radius={[2, 2, 0, 0]} />
              <Bar dataKey="10,000 users" fill="#a4adb8" radius={[2, 2, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div> : <p className="py-10 text-center text-sm text-slate-400">No runs at these load levels are available to chart.</p>}
        <p className="mt-3 text-xs text-slate-500">Results depend on the machine running Docker.</p>
      </div>
    </div>
  );
}
