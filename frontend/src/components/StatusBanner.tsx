import type { MetricPoint, SystemStatus } from '../types';
import { classNames, formatMillis } from '../utils/format';

interface BannerCopy {
  title: string;
  subtitle: string;
  stripe: string;
  label: string;
}

const copy: Record<SystemStatus, BannerCopy> = {
  idle: {
    title: 'System idle',
    subtitle: 'No traffic yet. Start a load test to see how this MongoDB configuration behaves.',
    stripe: 'border-l-slate-500',
    label: 'text-slate-200',
  },
  healthy: {
    title: 'System healthy',
    subtitle: 'Latency is stable, requests are succeeding and the database is keeping up.',
    stripe: 'border-l-status-good',
    label: 'text-status-good',
  },
  degraded: {
    title: 'Performance degraded',
    subtitle: 'Latency or error rate is climbing. The database is starting to struggle.',
    stripe: 'border-l-status-warn',
    label: 'text-status-warn',
  },
  critical: {
    title: 'System under stress',
    subtitle: 'Queries are timing out and requests are failing. This configuration cannot carry the load.',
    stripe: 'border-l-status-danger',
    label: 'text-status-danger',
  },
};

interface Props {
  status: SystemStatus;
  current: MetricPoint | undefined;
}

export function StatusBanner({ status, current }: Props) {
  const content = copy[status];
  return (
      <div role="status" aria-live="polite" className={classNames('border border-line border-l-4 bg-ink-900 px-4 py-4 sm:px-5', content.stripe)}>
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="min-w-0 flex-1 basis-64">
            <div className={classNames('text-base font-semibold', content.label)}>{content.title}</div>
            <p className="mt-1 text-sm text-slate-300">{content.subtitle}</p>
          </div>
          {current && status !== 'idle' ? (
            <div className="flex gap-5 sm:gap-8">
              <Stat label="p95 latency" value={formatMillis(current.p95)} />
              <Stat label="Error rate" value={`${current.errorRate.toFixed(1)}%`} />
              <Stat label="Requests / s" value={Math.round(current.rps).toLocaleString('en-IN')} />
            </div>
          ) : null}
        </div>
      </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-right">
      <div className="text-[11px] text-slate-400">{label}</div>
      <div className="mt-1 font-mono text-sm font-medium text-slate-100">{value}</div>
    </div>
  );
}

export function RequestStream({ history, error }: { history: MetricPoint[]; error: string | null }) {
  const tiles = history.slice(-60);
  const failed = tiles.reduce((sum, point) => sum + point.errors, 0);
  const hasTraffic = tiles.some((point) => point.rps > 0 || point.errors > 0);
  return (
    <div className="panel p-4">
      <div className="flex items-center justify-between">
        <span className="panel-title">Request stream (last 60 s)</span>
        {tiles.length > 0 ? <span className={classNames('text-xs tabular-nums', failed > 0 ? 'text-danger' : 'text-slate-400')}>
          {failed > 0 ? `${failed.toLocaleString('en-IN')} failed requests` : 'No failures'}
        </span> : null}
      </div>
      {error ? <div role="alert" className="mt-2 text-xs text-warn">Refresh failed; showing the last received request data.</div> : null}
      {tiles.length > 0 && hasTraffic ? <div className="mt-3 grid grid-cols-[repeat(30,minmax(0,1fr))] gap-1">
        {tiles.map((point) => {
          const critical = point.errorRate >= 10 || point.p95 >= 2000;
          const degraded = point.errorRate >= 2 || point.p95 >= 500;
          const idle = point.rps < 1;
          const color = idle ? '#1c2740' : critical ? '#ff4d6a' : degraded ? '#fbbf24' : '#22d3a5';
          return (
            <div
              key={point.timestamp}
              className="aspect-square rounded-sm"
              style={{ backgroundColor: color }}
              title={`${Math.round(point.rps)} req/s, p95 ${point.p95.toFixed(0)} ms, errors ${point.errorRate.toFixed(1)}%`}
            />
          );
        })}
      </div> : <div role={error ? 'alert' : 'status'} className="py-5 text-center text-sm text-slate-400">{error ? 'Request data is unavailable.' : tiles.length > 0 ? 'No traffic recorded in this window.' : 'Waiting for live request data.'}</div>}
    </div>
  );
}
